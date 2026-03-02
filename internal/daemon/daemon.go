// Package daemon — núcleo do qeloxd.
// Orquestra o socket server, node controller, monitor e web dashboard,
// com graceful shutdown e tratamento de sinais do SO.
package daemon

import (
	"fmt"
	"os"
	"os/signal"
	"path/filepath"
	"syscall"

	"github.com/zeus/qelox/internal/config"
	"github.com/zeus/qelox/internal/log"
	"github.com/zeus/qelox/internal/monitor"
	"github.com/zeus/qelox/internal/node"
	"github.com/zeus/qelox/internal/socket"
	"github.com/zeus/qelox/internal/web"
)

// Daemon é o objeto principal do processo qeloxd.
type Daemon struct {
	cfg    *config.Config
	log    *log.Logger
	node   *node.Controller
	mon    *monitor.Monitor
	server *socket.Server
	web    *web.Server
	lock   *os.File
}

// New constrói um Daemon com todas as dependências injetadas.
func New(cfg *config.Config, logger *log.Logger) *Daemon {
	nc := node.New(cfg, logger)
	mon := monitor.New(cfg, nc, logger)
	srv := socket.NewServer(cfg, nc, mon, logger)
	var webSrv *web.Server
	if cfg.Web.Enabled {
		webSrv = web.New(cfg, nc, mon, logger)
	}
	return &Daemon{cfg: cfg, log: logger, node: nc, mon: mon, server: srv, web: webSrv}
}

// Run bloqueia até receber sinal de shutdown.
func (d *Daemon) Run() error {
	// Criar diretório de runtime se não existir.
	rdir := filepath.Dir(d.cfg.Daemon.SocketPath)
	if err := os.MkdirAll(rdir, 0750); err != nil {
		return fmt.Errorf("falha ao criar runtime dir: %w", err)
	}

	// Adquire lock de instância única.
	if err := d.acquireLock(); err != nil {
		return err
	}
	defer d.releaseLock()

	// Limpa processos go-quai órfãos.
	node.KillOrphans(d.log)

	// Inicia socket server.
	if err := d.server.Start(); err != nil {
		return fmt.Errorf("falha ao iniciar socket: %w", err)
	}
	defer d.server.Stop()

	// Inicia monitor.
	d.mon.Start()
	defer d.mon.Stop()

	// Inicia web dashboard (se habilitado).
	if d.web != nil {
		if err := d.web.Start(); err != nil {
			// Não fatal — apenas loga e continua.
			d.log.Error("falha ao iniciar web dashboard", "error", err)
		} else {
			defer d.web.Stop()
		}
	}

	// Auto-start do node se configurado.
	if d.cfg.Node.AutoStart {
		if err := d.node.Start(); err != nil {
			d.log.Error("falha no auto-start do node", "error", err)
		}
	}

	d.log.Info("daemon pronto", "socket", d.cfg.Daemon.SocketPath)

	// Aguarda sinal de shutdown.
	ch := make(chan os.Signal, 1)
	signal.Notify(ch, syscall.SIGTERM, syscall.SIGINT)
	sig := <-ch
	d.log.Info("sinal recebido — graceful shutdown", "signal", sig.String())

	// Para o node se estiver rodando.
	if d.node.IsRunning() {
		d.log.Info("parando go-quai...")
		if err := d.node.Stop(); err != nil {
			d.log.Error("erro ao parar node", "error", err)
		}
	}

	return nil
}

// acquireLock cria e trava o arquivo de lock (single-instance).
func (d *Daemon) acquireLock() error {
	f, err := os.OpenFile(d.cfg.Daemon.LockFile, os.O_CREATE|os.O_RDWR, 0640)
	if err != nil {
		return fmt.Errorf("falha ao abrir lock file: %w", err)
	}
	if err := syscall.Flock(int(f.Fd()), syscall.LOCK_EX|syscall.LOCK_NB); err != nil {
		f.Close()
		return fmt.Errorf("outra instância do qeloxd já está rodando")
	}
	fmt.Fprintf(f, "%d\n", os.Getpid())
	d.lock = f
	return nil
}

func (d *Daemon) releaseLock() {
	if d.lock != nil {
		syscall.Flock(int(d.lock.Fd()), syscall.LOCK_UN)
		d.lock.Close()
		os.Remove(d.cfg.Daemon.LockFile)
	}
}
