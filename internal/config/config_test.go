package config

import (
	"os"
	"path/filepath"
	"testing"
)

func TestConfigPathUsesEnvOverride(t *testing.T) {
	t.Setenv("QELOX_CONFIG", "/tmp/custom-qelox.toml")
	t.Setenv("QELOX_HOME", "")

	if got := configPath(); got != "/tmp/custom-qelox.toml" {
		t.Fatalf("configPath() = %q, want %q", got, "/tmp/custom-qelox.toml")
	}
}

func TestBasePathUsesEnvOverride(t *testing.T) {
	tmp := t.TempDir()
	t.Setenv("QELOX_HOME", tmp)
	t.Setenv("QELOX_CONFIG", "")

	if got := basePath(); got != tmp {
		t.Fatalf("basePath() = %q, want %q", got, tmp)
	}
}

func TestSaveCreatesParentDirectory(t *testing.T) {
	tmp := t.TempDir()
	cfgPath := filepath.Join(tmp, "nested", "config.toml")
	t.Setenv("QELOX_CONFIG", cfgPath)
	t.Setenv("QELOX_HOME", "")

	cfg := defaults()
	if err := Save(cfg); err != nil {
		t.Fatalf("Save() error = %v", err)
	}

	if _, err := os.Stat(cfgPath); err != nil {
		t.Fatalf("expected config file at %q: %v", cfgPath, err)
	}
}
