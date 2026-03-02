// Package config — carrega e valida config.toml do QELO-X.
// Toda configuração é tipada e validada antes de ser usada.
package config

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"

	"github.com/BurntSushi/toml"
)

// Config agrupa toda a configuração do daemon.
type Config struct {
	Node    NodeConfig    `toml:"node"`
	Daemon  DaemonConfig  `toml:"daemon"`
	Monitor MonitorConfig `toml:"monitor"`
	Web     WebConfig     `toml:"web"`
}

// NodeConfig — configuração do binário go-quai.
type NodeConfig struct {
	BinaryPath string   `toml:"binary_path"`
	BaseDir    string   `toml:"base_dir"`
	DataDir    string   `toml:"data_dir"`
	MaxPeers   int      `toml:"max_peers"`
	AutoStart  bool     `toml:"auto_start"`
	ExtraArgs  []string `toml:"extra_args"`
}

// DaemonConfig — comportamento do daemon qeloxd.
type DaemonConfig struct {
	SocketPath   string `toml:"socket_path"`
	LockFile     string `toml:"lock_file"`
	RestartDelay int    `toml:"restart_delay_sec"`
	MaxRestarts  int    `toml:"max_restarts"`
}

// MonitorConfig — intervalos de coleta de métricas.
type MonitorConfig struct {
	IntervalSec      int    `toml:"interval_sec"`
	FreezeTimeoutMin int    `toml:"freeze_timeout_min"`
	RPCURL           string `toml:"rpc_url"`
}

// WebConfig — dashboard web embutido.
type WebConfig struct {
	Enabled bool   `toml:"enabled"`
	Port    int    `toml:"port"`
	Bind    string `toml:"bind"`
}

// Load procura config.toml em ~/qelox/config.toml.
func Load() (*Config, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return nil, err
	}
	path := filepath.Join(home, "qelox", "config.toml")

	cfg := defaults()
	if _, statErr := os.Stat(path); os.IsNotExist(statErr) {
		// Sem arquivo de config — usa defaults.
		return cfg, nil
	}
	if _, err := toml.DecodeFile(path, cfg); err != nil {
		return nil, err
	}
	return cfg, validate(cfg)
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
		},
		Web: WebConfig{
			Enabled: true,
			Port:    9201,
			Bind:    "127.0.0.1",
		},
	}
}

func validate(cfg *Config) error {
	if cfg.Node.BinaryPath == "" {
		return errors.New("node.binary_path não pode ser vazio")
	}
	if cfg.Daemon.SocketPath == "" {
		return errors.New("daemon.socket_path não pode ser vazio")
	}
	if cfg.Web.Enabled && (cfg.Web.Port < 1 || cfg.Web.Port > 65535) {
		return fmt.Errorf("web.port inválido: %d", cfg.Web.Port)
	}
	return nil
}

// LogFile retorna caminho do arquivo de log do daemon.
func (c *Config) LogFile() string {
	home, _ := os.UserHomeDir()
	return filepath.Join(home, "qelox", "logs", "qeloxd.log")
}

// NodeLogFile retorna caminho do arquivo de log do go-quai.
func (c *Config) NodeLogFile() string {
	home, _ := os.UserHomeDir()
	return filepath.Join(home, "qelox", "logs", "go-quai.log")
}
