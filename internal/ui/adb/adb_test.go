package adb

import (
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/kooler/MiddayCommander/internal/ui/theme"
	"github.com/kooler/MiddayCommander/internal/vfs/adbfs"
)

func TestOnlyAuthorizedDeviceCanBeSelected(t *testing.T) {
	m := New(100, 30)
	m.HandleDevices(DevicesLoadedMsg{Devices: []adbfs.Device{
		{Serial: "offline-1", State: "offline"},
		{Serial: "phone-1", State: "device", Model: "Pixel"},
	}})
	_, cmd := m.Update(tea.KeyMsg{Type: tea.KeyEnter})
	if cmd != nil {
		t.Fatal("an offline device must not connect")
	}
	m, _ = m.Update(tea.KeyMsg{Type: tea.KeyDown})
	_, cmd = m.Update(tea.KeyMsg{Type: tea.KeyEnter})
	if cmd == nil {
		t.Fatal("authorized device should be selectable")
	}
	msg, ok := cmd().(SelectMsg)
	if !ok || msg.Device.Serial != "phone-1" {
		t.Fatalf("selection = %#v, want phone-1", msg)
	}
}

func TestConnectModeStartsAsyncConnection(t *testing.T) {
	m := New(100, 30)
	m.HandleDevices(DevicesLoadedMsg{})
	m, _ = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'c'}})
	if !m.connectMode {
		t.Fatal("c should open network connect form")
	}
	m, _ = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("phone.local")})
	if m.connectTarget != "phone.local" {
		t.Fatalf("connect target = %q", m.connectTarget)
	}
	m, cmd := m.Update(tea.KeyMsg{Type: tea.KeyEnter})
	if cmd == nil || !m.connectLoading {
		t.Fatal("Enter should start connection asynchronously")
	}
}

func TestConnectAndFilterInputsCannotReorderDisplayedText(t *testing.T) {
	m := New(100, 30)
	m.connectMode = true
	m.connectTarget = "phone\u202e.example"
	view := m.View(theme.Theme{}, 100, 30)
	if strings.Contains(view, "\u202e") {
		t.Fatal("ADB target input retained a bidi control character")
	}

	m.connectMode = false
	m.filtering = true
	m.filter = "phone\u202e"
	view = m.View(theme.Theme{}, 100, 30)
	if strings.Contains(view, "\u202e") {
		t.Fatal("ADB filter retained a bidi control character")
	}
}
