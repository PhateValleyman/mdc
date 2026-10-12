package config

import (
	"os"
	"path/filepath"
	"testing"
)

func TestRcloneKeyDefaultAndOverride(t *testing.T) {
	defaults := Default()
	if len(defaults.Keys.Rclone) != 1 || defaults.Keys.Rclone[0] != "R" {
		t.Fatalf("default rclone key = %#v, want R (Shift+R)", defaults.Keys.Rclone)
	}
	if got := normalizeKey("shift+r"); got != "R" {
		t.Fatalf("normalizeKey(shift+r) = %q, want R", got)
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

func TestADBKeyDefaultAndOverride(t *testing.T) {
	defaults := Default()
	if len(defaults.Keys.ADB) != 1 || defaults.Keys.ADB[0] != "A" {
		t.Fatalf("default adb key = %#v, want A (Shift+A)", defaults.Keys.ADB)
	}
	if got := normalizeKey("shift+a"); got != "A" {
		t.Fatalf("normalizeKey(shift+a) = %q, want A", got)
	}

	xdg := t.TempDir()
	t.Setenv("XDG_CONFIG_HOME", xdg)
	configDir := filepath.Join(xdg, "mdc")
	if err := os.MkdirAll(configDir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(configDir, "config.toml"), []byte("[keys]\nadb = \"ctrl+alt+a\"\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	loaded := Load()
	if len(loaded.Keys.ADB) != 1 || loaded.Keys.ADB[0] != "ctrl+alt+a" {
		t.Fatalf("loaded adb key = %#v, want configured override", loaded.Keys.ADB)
	}
}

func TestSettingsKeyDefaultAndOverride(t *testing.T) {
	defaults := Default()
	if len(defaults.Keys.Settings) != 1 || defaults.Keys.Settings[0] != "alt+o" {
		t.Fatalf("default settings key = %#v, want alt+o", defaults.Keys.Settings)
	}

	xdg := t.TempDir()
	t.Setenv("XDG_CONFIG_HOME", xdg)
	configDir := filepath.Join(xdg, "mdc")
	if err := os.MkdirAll(configDir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(configDir, "config.toml"), []byte("[keys]\nsettings = \"alt+s\"\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	loaded := Load()
	if len(loaded.Keys.Settings) != 1 || loaded.Keys.Settings[0] != "alt+s" {
		t.Fatalf("loaded settings key = %#v, want configured override", loaded.Keys.Settings)
	}
}
