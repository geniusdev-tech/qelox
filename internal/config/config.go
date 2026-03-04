// Package config — loads and validates QELO-X config.toml.
// All configuration is typed and validated before use.
package config

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"

	"github.com/BurntSushi/toml"
)

// Config groups all daemon configuration.
type Config struct {
	Node    NodeConfig    `toml:"node"`
	Daemon  DaemonConfig  `toml:"daemon"`
	Monitor MonitorConfig `toml:"monitor"`
	Web     WebConfig     `toml:"web"`
}

// NodeConfig — go-quai binary configuration.
type NodeConfig struct {
	BinaryPath string   `toml:"binary_path"`
	BaseDir    string   `toml:"base_dir"`
	DataDir    string   `toml:"data_dir"`
	MaxPeers   int      `toml:"max_peers"`
	AutoStart  bool     `toml:"auto_start"`
	ExtraArgs  []string `toml:"extra_args"`
}

// DaemonConfig — qeloxd daemon behavior.
type DaemonConfig struct {
	SocketPath   string `toml:"socket_path"`
	LockFile     string `toml:"lock_file"`
	RestartDelay int    `toml:"restart_delay_sec"`
	MaxRestarts  int    `toml:"max_restarts"`
}

// MonitorConfig — metrics collection intervals.
type MonitorConfig struct {
	IntervalSec      int    `toml:"interval_sec"`
	FreezeTimeoutMin int    `toml:"freeze_timeout_min"`
	RPCURL           string `toml:"rpc_url"`
	MinPeers         int    `toml:"min_peers"`
}

// WebConfig — embedded web dashboard.
type WebConfig struct {
	Enabled  bool   `toml:"enabled"`
	Port     int    `toml:"port"`
	Bind     string `toml:"bind"`
	Username string `toml:"username"`
	Password string `toml:"password"`
}

// Load searches for config.toml in ~/qelox/config.toml.
func Load() (*Config, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return nil, err
	}
	path := filepath.Join(home, "qelox", "config.toml")

	cfg := defaults()
	if _, statErr := os.Stat(path); os.IsNotExist(statErr) {
		// No configuration file — use defaults.
		return cfg, nil
	}
	if _, err := toml.DecodeFile(path, cfg); err != nil {
		return nil, err
	}
	return cfg, validate(cfg)
}

// Save writes current configuration back to disk.
func Save(cfg *Config) error {
	if err := validate(cfg); err != nil {
		return err
	}
	home, err := os.UserHomeDir()
	if err != nil {
		return err
	}
	path := filepath.Join(home, "qelox", "config.toml")
	f, err := os.Create(path)
	if err != nil {
		return err
	}
	defer f.Close()
	return toml.NewEncoder(f).Encode(cfg)
}

func defaults() *Config {
	home, _ := os.UserHomeDir()
	base := filepath.Join(home, "qelox")
	return &Config{
		Node: NodeConfig{
			BinaryPath: "/usr/local/bin/go-quai",
			DataDir:    filepath.Join(home, ".go-quai"),
			MaxPeers:   25,
			AutoStart:  true,
		},
		Daemon: DaemonConfig{
			SocketPath:   filepath.Join(base, "runtime", "qelox.sock"),
			LockFile:     filepath.Join(base, "runtime", "qeloxd.lock"),
			RestartDelay: 5,
			MaxRestarts:  0,
		},
		Monitor: MonitorConfig{
			IntervalSec:      2,
			FreezeTimeoutMin: 10,
			RPCURL:           "http://localhost:9200",
			MinPeers:         10,
		},
		Web: WebConfig{
			Enabled:  true,
			Port:     9201,
			Bind:     "127.0.0.1",
			Username: "",
			Password: "",
		},
	}
}

func validate(cfg *Config) error {
	if cfg.Node.BinaryPath == "" {
		return errors.New("node.binary_path cannot be empty")
	}
	if cfg.Daemon.SocketPath == "" {
		return errors.New("daemon.socket_path cannot be empty")
	}
	if cfg.Web.Enabled && (cfg.Web.Port < 1 || cfg.Web.Port > 65535) {
		return fmt.Errorf("invalid web.port: %d", cfg.Web.Port)
	}
	return nil
}

// LogFile returns the daemons log file path.
func (c *Config) LogFile() string {
	home, _ := os.UserHomeDir()
	return filepath.Join(home, "qelox", "logs", "qeloxd.log")
}

// NodeLogFile returns the go-quai log file path.
func (c *Config) NodeLogFile() string {
	home, _ := os.UserHomeDir()
	return filepath.Join(home, "qelox", "logs", "go-quai.log")
}
