// Package daemon — qeloxd core.
// Orchestrates the socket server, node controller, monitor, and web dashboard,
// with graceful shutdown and OS signal handling.
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

// Daemon is the main object of the qeloxd process.
type Daemon struct {
	cfg    *config.Config
	log    *log.Logger
	node   *node.Controller
	mon    *monitor.Monitor
	server *socket.Server
	web    *web.Server
	lock   *os.File
}

// New constructs a Daemon with all dependencies injected.
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

// Run blocks until a shutdown signal is received.
func (d *Daemon) Run() error {
	// Criar diretório de runtime se não existir.
	rdir := filepath.Dir(d.cfg.Daemon.SocketPath)
	if err := os.MkdirAll(rdir, 0750); err != nil {
		return fmt.Errorf("failed to create runtime dir: %w", err)
	}

	// Adquire lock de instância única.
	if err := d.acquireLock(); err != nil {
		return err
	}
	defer d.releaseLock()

	// Inicia socket server.
	if err := d.server.Start(); err != nil {
		return fmt.Errorf("failed to start socket: %w", err)
	}
	defer d.server.Stop()

	// Inicia monitor.
	d.mon.Start()
	defer d.mon.Stop()

	// Inicia web dashboard (se habilitado).
	if d.web != nil {
		if err := d.web.Start(); err != nil {
			// Não fatal — apenas loga e continua.
			d.log.Error("failed to start web dashboard", "error", err)
		} else {
			defer d.web.Stop()
		}
	}

	// Auto-start do node se configurado.
	if d.cfg.Node.AutoStart {
		if err := d.node.Start(); err != nil {
			d.log.Error("failed node auto-start", "error", err)
		}
	}

	d.log.Info("daemon ready", "socket", d.cfg.Daemon.SocketPath)

	// Aguarda sinal de shutdown.
	ch := make(chan os.Signal, 1)
	signal.Notify(ch, syscall.SIGTERM, syscall.SIGINT)
	sig := <-ch
	d.log.Info("signal received — graceful shutdown", "signal", sig.String())

	// Para o node se estiver rodando.
	if d.node.IsRunning() {
		d.log.Info("stopping go-quai...")
		if err := d.node.Stop(); err != nil {
			d.log.Error("error stopping node", "error", err)
		}
	}

	return nil
}

// acquireLock cria e trava o arquivo de lock (single-instance).
func (d *Daemon) acquireLock() error {
	f, err := os.OpenFile(d.cfg.Daemon.LockFile, os.O_CREATE|os.O_RDWR, 0640)
	if err != nil {
		return fmt.Errorf("failed to open lock file: %w", err)
	}
	if err := syscall.Flock(int(f.Fd()), syscall.LOCK_EX|syscall.LOCK_NB); err != nil {
		f.Close()
		return fmt.Errorf("another instance of qeloxd is already running")
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
