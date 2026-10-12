package app

import (
	"testing"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/kooler/MiddayCommander/internal/vfs"
)

func TestADBURLOpensDevicePath(t *testing.T) {
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	m := New("test")
	next, cmd, ok := m.openADBTarget("adb://phone-123/sdcard/Download")
	if !ok {
		t.Fatal("adb URL was not recognized")
	}
	updated, ok := next.(Model)
	if !ok {
		t.Fatalf("model type = %T", next)
	}
	loc := updated.leftPanel.Location()
	if loc.Kind != vfs.KindADB || loc.Path != "/sdcard/Download" || loc.Label != "adb://phone-123" {
		t.Fatalf("unexpected adb location: kind=%v path=%q label=%q", loc.Kind, loc.Path, loc.Label)
	}
	if cmd == nil {
		t.Fatal("opening an adb path should load its directory")
	}
}

func TestShiftAOpensADBDevicePicker(t *testing.T) {
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	m := New("test")
	next, cmd := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'A'}})
	updated, ok := next.(Model)
	if !ok || updated.adbPicker == nil {
		t.Fatal("Shift+A should open the adb device picker")
	}
	if cmd == nil {
		t.Fatal("opening the device picker should scan adb devices")
	}
}
