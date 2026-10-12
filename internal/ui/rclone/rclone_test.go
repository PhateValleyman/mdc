package rclone

import (
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/kooler/MiddayCommander/internal/ui/theme"
)

func TestSelectRemote(t *testing.T) {
	m := New(100, 30)
	m.HandleRemotes(RemotesLoadedMsg{Remotes: []string{"backup", "drive"}})
	_, cmd := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'d'}})
	if cmd != nil {
		t.Fatal("typing in the remote list should not directly select")
	}
	_, cmd = m.Update(tea.KeyMsg{Type: tea.KeyEnter})
	if cmd == nil {
		t.Fatal("Enter should select the current remote")
	}
	msg, ok := cmd().(SelectMsg)
	if !ok || msg.Remote != "backup" {
		t.Fatalf("selection = %#v, want backup", msg)
	}
}

func TestAddConfigUsesDedicatedRequest(t *testing.T) {
	m := New(100, 30)
	_, cmd := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'a'}})
	if cmd == nil {
		t.Fatal("a should request the native rclone config command")
	}
	if _, ok := cmd().(CreateConfigRequestedMsg); !ok {
		t.Fatal("a should emit CreateConfigRequestedMsg")
	}
}

func TestEditRequestsCurrentRemote(t *testing.T) {
	m := New(100, 30)
	m.HandleRemotes(RemotesLoadedMsg{Remotes: []string{"backup"}})
	_, cmd := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'e'}})
	if cmd == nil {
		t.Fatal("e should request editing the selected remote")
	}
	msg, ok := cmd().(EditConfigRequestedMsg)
	if !ok || msg.Remote != "backup" {
		t.Fatalf("edit request = %#v, want backup", msg)
	}
}

func TestOAuthReconnectRequestsCurrentRemote(t *testing.T) {
	m := New(100, 30)
	m.HandleRemotes(RemotesLoadedMsg{Remotes: []string{"backup", "drive"}})
	m, cmd := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'o'}})
	if cmd == nil {
		t.Fatal("o should request browser-based OAuth reconnect")
	}
	msg, ok := cmd().(ReconnectRequestedMsg)
	if !ok || msg.Remote != "backup" {
		t.Fatalf("reconnect request = %#v, want remote backup", msg)
	}
}

func TestFilterRemotes(t *testing.T) {
	m := New(100, 30)
	m.HandleRemotes(RemotesLoadedMsg{Remotes: []string{"backup", "drive", "drive-archive"}})
	m, _ = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'f'}})
	if !m.filtering {
		t.Fatal("f should enter filter mode")
	}
	m, _ = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'d', 'r'}})
	if len(m.items) != 2 || m.items[0] != "drive" || m.items[1] != "drive-archive" {
		t.Fatalf("filtered remotes = %#v", m.items)
	}
}

func TestFilterBackspaceRemovesWholeUnicodeRune(t *testing.T) {
	m := New(100, 30)
	m.HandleRemotes(RemotesLoadedMsg{Remotes: []string{"café"}})
	m.filtering = true
	m.filter = "café"
	m.refilter()

	m, _ = m.Update(tea.KeyMsg{Type: tea.KeyBackspace})
	if m.filter != "caf" {
		t.Fatalf("filter after backspace = %q, want %q", m.filter, "caf")
	}
	if len(m.items) != 1 || m.items[0] != "café" {
		t.Fatalf("filtered remotes = %#v, want café", m.items)
	}
}

func TestRemoteNameCannotEmitTerminalControlSequence(t *testing.T) {
	const maliciousRemote = "backup\x1b]52;c;payload\a"
	m := New(100, 30)
	m.HandleRemotes(RemotesLoadedMsg{Remotes: []string{"first", maliciousRemote}})
	view := m.View(theme.Theme{}, 100, 30)
	if strings.Contains(view, "\x1b]52;") || strings.ContainsRune(view, '\a') {
		t.Fatal("rclone remote metadata emitted a terminal control sequence")
	}
}
