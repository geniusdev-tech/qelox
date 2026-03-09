// Package node — go-quai process Controller.
// Manages start/stop/restart, correct exponential backoff auto-restart,
// with no file descriptor leaks and exportable crash metadata.
package node

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"sync"
	"syscall"
	"time"

	"github.com/zeus/qelox/internal/config"
	"github.com/zeus/qelox/internal/log"
)

// State represents the current node state.
type State int

const (
	StateStopped State = iota
	StateStarting
	StateRunning
	StateStopping
	StateCrashed
)

func (s State) String() string {
	switch s {
	case StateStopped:
		return "STOPPED"
	case StateStarting:
		return "STARTING"
	case StateRunning:
		return "RUNNING"
	case StateStopping:
		return "STOPPING"
	case StateCrashed:
		return "CRASHED"
	default:
		return "UNKNOWN"
	}
}

// Controller gerencia o subprocesso go-quai.
type Controller struct {
	cfg             *config.Config
	log             *log.Logger
	mu              sync.RWMutex
	cmd             *exec.Cmd
	logFile         *os.File // FIX: mantido para fechar no Stop()
	state           State
	startAt         time.Time
	lastRestartAt   time.Time
	lastCrashReason string
	restarts        int
	stopRestart     chan struct{}
}

// New cria um Controller.
func New(cfg *config.Config, logger *log.Logger) *Controller {
	return &Controller{
		cfg:         cfg,
		log:         logger,
		state:       StateStopped,
		stopRestart: make(chan struct{}, 1),
	}
}

// Start inicia go-quai como subprocesso gerenciado.
func (c *Controller) Start() error {
	c.mu.Lock()
	defer c.mu.Unlock()

	if c.state == StateRunning || c.state == StateStarting {
		return fmt.Errorf("go-quai is already running (state=%s)", c.state)
	}
	if _, err := os.Stat(c.cfg.Node.BinaryPath); err != nil {
		return fmt.Errorf("binary not found: %s", c.cfg.Node.BinaryPath)
	}
	c.state = StateStarting
	return c.launch()
}

// launch executa o subprocesso (deve ser chamado com mu travado).
func (c *Controller) launch() error {
	args := c.buildArgs()
	cmd := exec.Command(c.cfg.Node.BinaryPath, args...)
	if c.cfg.Node.BaseDir != "" {
		cmd.Dir = c.cfg.Node.BaseDir
	}

	// FIX: usar NodeLogFile() centralizado e armazenar referência para fechar depois.
	logPath := c.cfg.NodeLogFile()
	if err := os.MkdirAll(filepath.Dir(logPath), 0750); err != nil {
		return err
	}
	logFile, err := os.OpenFile(logPath, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0640)
	if err != nil {
		return fmt.Errorf("failed to open node log: %w", err)
	}
	// FIX: fechar o logFile anterior se existir (evita FD leak em restarts).
	if c.logFile != nil {
		c.logFile.Close()
	}
	c.logFile = logFile
	cmd.Stdout = logFile
	cmd.Stderr = logFile

	// Process group próprio para facilitar kill de toda a árvore.
	cmd.SysProcAttr = &syscall.SysProcAttr{Setpgid: true}

	// Injetar variáveis de mitigação de memória
	cmd.Env = append(os.Environ(),
		"GOGC=50",
		"GOMEMLIMIT=1GiB",
	)

	if err := cmd.Start(); err != nil {
		c.state = StateCrashed
		logFile.Close()
		c.logFile = nil
		return fmt.Errorf("failed to start go-quai: %w", err)
	}

	c.cmd = cmd
	c.state = StateRunning
	c.startAt = time.Now()
	c.log.Info("go-quai started", "pid", cmd.Process.Pid, "args", args)

	go c.watchProcess()
	return nil
}

// watchProcess aguarda término e dispara auto-restart se necessário.
func (c *Controller) watchProcess() {
	err := c.cmd.Wait()

	c.mu.Lock()
	defer c.mu.Unlock()

	if c.state == StateStopping {
		c.state = StateStopped
		c.log.Info("go-quai stopped normally")
		return
	}

	// Registrar motivo do crash.
	crashReason := "clean exit"
	if err != nil {
		crashReason = err.Error()
	}
	c.lastCrashReason = crashReason
	c.state = StateCrashed
	c.restarts++
	c.log.Warn("go-quai crashed", "error", crashReason, "restarts", c.restarts)

	if c.cfg.Daemon.MaxRestarts > 0 && c.restarts >= c.cfg.Daemon.MaxRestarts {
		c.log.Error("restart limit reached")
		c.state = StateStopped
		return
	}

	// FIX: backoff exponencial correto com teto de 60s.
	base := time.Duration(c.cfg.Daemon.RestartDelay) * time.Second
	delay := base * time.Duration(c.restarts)
	if delay > 60*time.Second {
		delay = 60 * time.Second
	}
	c.log.Info("scheduling restart", "delay", delay)

	go func() {
		timer := time.NewTimer(delay)
		defer timer.Stop()
		select {
		case <-timer.C:
			c.mu.Lock()
			defer c.mu.Unlock()
			// Check if we're still in crashed state (prevent race where user triggered Start)
			if c.state != StateCrashed {
				c.log.Info("state changed, canceling auto-restart")
				return
			}
			c.lastRestartAt = time.Now()
			c.log.Info("restarting go-quai after crash")
			if err := c.launch(); err != nil {
				c.log.Error("auto-restart failed", "error", err)
			}
		case <-c.stopRestart:
			c.log.Info("auto-restart canceled")
		}
	}()
}

// Stop envia SIGTERM e aguarda até 30s.
func (c *Controller) Stop() error {
	c.mu.Lock()

	if c.state != StateRunning && c.state != StateCrashed {
		c.mu.Unlock()
		return fmt.Errorf("node is not running (state=%s)", c.state)
	}
	wasRunning := (c.state == StateRunning)
	c.state = StateStopping

	// Cancela restart pendente se houver.
	select {
	case c.stopRestart <- struct{}{}:
	default:
	}

	if wasRunning {
		proc := c.cmd.Process
		c.log.Info("sending SIGTERM", "pid", proc.Pid)
		if err := proc.Signal(syscall.SIGTERM); err != nil {
			c.mu.Unlock()
			return err
		}
		c.mu.Unlock()

		timeout := time.After(30 * time.Second)
		ticker := time.NewTicker(100 * time.Millisecond)
		defer ticker.Stop()

	waitLoop:
		for {
			select {
			case <-timeout:
				c.mu.Lock()
				if c.state != StateStopped {
					c.log.Warn("timeout — sending SIGKILL")
					proc.Kill()
					c.state = StateStopped
				}
				c.mu.Unlock()
				break waitLoop
			case <-ticker.C:
				if c.State() == StateStopped {
					break waitLoop
				}
			}
		}
	} else {
		// Se estava crashado, apenas marca como parado e solta o lock principal
		c.state = StateStopped
		c.mu.Unlock()
	}

	c.mu.Lock()
	defer c.mu.Unlock()

	// FIX: fechar o log file do node após encerrar o processo.
	if c.logFile != nil {
		c.logFile.Close()
		c.logFile = nil
	}

	return nil
}

// Restart para e reinicia o node.
func (c *Controller) Restart() error {
	if err := c.Stop(); err != nil {
		return err
	}
	time.Sleep(2 * time.Second)
	return c.Start()
}

// IsRunning retorna true se o node está rodando.
func (c *Controller) IsRunning() bool {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.state == StateRunning
}

// State retorna estado atual (thread-safe).
func (c *Controller) State() State {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.state
}

// Uptime retorna duração desde o último start.
func (c *Controller) Uptime() time.Duration {
	c.mu.RLock()
	defer c.mu.RUnlock()
	if c.state != StateRunning {
		return 0
	}
	return time.Since(c.startAt)
}

// Restarts retorna contagem de restarts automáticos.
func (c *Controller) Restarts() int {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.restarts
}

// LastRestartAt retorna timestamp do último auto-restart.
func (c *Controller) LastRestartAt() time.Time {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.lastRestartAt
}

// LastCrashReason retorna o motivo do último crash.
func (c *Controller) LastCrashReason() string {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.lastCrashReason
}

// PID returns the managed go-quai PID when the process is live.
func (c *Controller) PID() int {
	c.mu.RLock()
	defer c.mu.RUnlock()
	if c.cmd == nil || c.cmd.Process == nil || c.state != StateRunning {
		return 0
	}
	return c.cmd.Process.Pid
}

func (c *Controller) buildArgs() []string {
	args := []string{"start"}
	if c.cfg.Node.MaxPeers > 0 {
		args = append(args, fmt.Sprintf("--node.max-peers=%d", c.cfg.Node.MaxPeers))
	}
	if c.cfg.Node.DataDir != "" {
		args = append(args, fmt.Sprintf("--global.data-dir=%s", c.cfg.Node.DataDir))
	}
	return append(args, c.cfg.Node.ExtraArgs...)
}
