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

const (
	envConfigPath = "QELOX_CONFIG"
	envHomePath   = "QELOX_HOME"
)

// Config groups all daemon configuration.
type Config struct {
	Node    NodeConfig    `toml:"node" json:"node"`
	Daemon  DaemonConfig  `toml:"daemon" json:"daemon"`
	Monitor MonitorConfig `toml:"monitor" json:"monitor"`
	Web     WebConfig     `toml:"web" json:"web"`
}

// NodeConfig — go-quai binary configuration.
type NodeConfig struct {
	BinaryPath string   `toml:"binary_path" json:"binary_path"`
	BaseDir    string   `toml:"base_dir" json:"base_dir"`
	DataDir    string   `toml:"data_dir" json:"data_dir"`
	MaxPeers   int      `toml:"max_peers" json:"max_peers"`
	AutoStart  bool     `toml:"auto_start" json:"auto_start"`
	ExtraArgs  []string `toml:"extra_args" json:"extra_args"`
}

// DaemonConfig — qeloxd daemon behavior.
type DaemonConfig struct {
	SocketPath   string `toml:"socket_path" json:"socket_path"`
	LockFile     string `toml:"lock_file" json:"lock_file"`
	RestartDelay int    `toml:"restart_delay_sec" json:"restart_delay_sec"`
	MaxRestarts  int    `toml:"max_restarts" json:"max_restarts"`
}

// MonitorConfig — metrics collection intervals.
type MonitorConfig struct {
	IntervalSec      int    `toml:"interval_sec" json:"interval_sec"`
	FreezeTimeoutMin int    `toml:"freeze_timeout_min" json:"freeze_timeout_min"`
	RPCURL           string `toml:"rpc_url" json:"rpc_url"`
	MinPeers         int    `toml:"min_peers" json:"min_peers"`
}

// WebConfig — embedded web dashboard.
type WebConfig struct {
	Enabled  bool   `toml:"enabled" json:"enabled"`
	Port     int    `toml:"port" json:"port"`
	Bind     string `toml:"bind" json:"bind"`
	Username string `toml:"username" json:"username"`
	Password string `toml:"password" json:"password"`
}

// Load searches for config.toml in QELOX_CONFIG or QELOX_HOME/config.toml.
func Load() (*Config, error) {
	path := configPath()
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
	path := configPath()
	if err := os.MkdirAll(filepath.Dir(path), 0750); err != nil {
		return err
	}
	f, err := os.Create(path)
	if err != nil {
		return err
	}
	defer f.Close()
	return toml.NewEncoder(f).Encode(cfg)
}

func defaults() *Config {
	home := userHomeDir()
	base := basePath()
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
	return filepath.Join(basePath(), "logs", "qeloxd.log")
}

// NodeLogFile returns the go-quai log file path.
func (c *Config) NodeLogFile() string {
	return filepath.Join(basePath(), "logs", "go-quai.log")
}

func configPath() string {
	if path := os.Getenv(envConfigPath); path != "" {
		return path
	}
	return filepath.Join(basePath(), "config.toml")
}

func basePath() string {
	if path := os.Getenv(envHomePath); path != "" {
		return path
	}
	return filepath.Join(userHomeDir(), "qelox")
}

func userHomeDir() string {
	home, err := os.UserHomeDir()
	if err == nil && home != "" {
		return home
	}
	return os.Getenv("HOME")
}
