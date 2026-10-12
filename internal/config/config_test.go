package config

import (
	"os"
	"path/filepath"
	"testing"
)

func TestRcloneKeyDefaultAndOverride(t *testing.T) {
	defaults := Default()
	if len(defaults.Keys.Rclone) != 1 || defaults.Keys.Rclone[0] != "alt+r" {
		t.Fatalf("default rclone key = %#v, want alt+r", defaults.Keys.Rclone)
	}

	xdg := t.TempDir()
	t.Setenv("XDG_CONFIG_HOME", xdg)
	configDir := filepath.Join(xdg, "mdc")
	if err := os.MkdirAll(configDir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(configDir, "config.toml"), []byte("[keys]\nrclone = \"ctrl+alt+r\"\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	loaded := Load()
	if len(loaded.Keys.Rclone) != 1 || loaded.Keys.Rclone[0] != "ctrl+alt+r" {
		t.Fatalf("loaded rclone key = %#v, want configured override", loaded.Keys.Rclone)
	}
}
