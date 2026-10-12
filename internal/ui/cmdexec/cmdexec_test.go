package cmdexec

import (
	"strings"
	"testing"
	"unicode/utf8"

	tea "github.com/charmbracelet/bubbletea"
	uitext "github.com/kooler/MiddayCommander/internal/ui/text"
	"github.com/kooler/MiddayCommander/internal/ui/theme"
)

func press(m *Model, key tea.KeyType, runes ...rune) {
	var msg tea.KeyMsg
	if runes != nil {
		msg = tea.KeyMsg{Type: tea.KeyRunes, Runes: runes}
	} else {
		msg = tea.KeyMsg{Type: key}
	}
	updated, _ := m.handleKey(msg)
	*m = updated
}

func TestUnicodeInputEditing(t *testing.T) {
	m := New(".", 80, 24)
	press(&m, tea.KeyRunes, '日', '本')
	press(&m, tea.KeyRunes, '🎉')
	if m.input != "日本🎉" {
		t.Fatalf("input = %q, want 日本🎉", m.input)
	}

	press(&m, tea.KeyBackspace)
	if m.input != "日本" || m.inputPos != len(m.input) {
		t.Errorf("after backspace: input = %q, position = %d", m.input, m.inputPos)
	}
	press(&m, tea.KeyLeft)
	press(&m, tea.KeyDelete)
	if m.input != "日" || !utf8.ValidString(m.input) {
		t.Errorf("after delete: input = %q, want 日", m.input)
	}
}

func TestSpaceInput(t *testing.T) {
	m := New(".", 80, 24)
	press(&m, tea.KeySpace)
	if m.input != " " {
		t.Errorf("input = %q, want a space", m.input)
	}
}

func TestViewSanitizesDirectoryOnlyForRender(t *testing.T) {
	const hostileDir = "/safe\x1b]x\x07\u202e"
	m := New(hostileDir, 80, 24)

	view := m.View(theme.Default(), 80, 24)
	safeDir := uitext.SafeTerminalText(hostileDir)
	if !strings.Contains(view, safeDir) {
		t.Fatalf("command runner view should contain sanitized directory %q: %q", safeDir, view)
	}
	if strings.Contains(view, hostileDir) {
		t.Fatalf("command runner view exposed the raw directory: %q", view)
	}
	if m.dir != hostileDir {
		t.Fatalf("rendering changed the command working directory: %q", m.dir)
	}
}

func TestViewSanitizesInputSuggestionsAndOutputOnlyForRender(t *testing.T) {
	const hostile = "x\x1b]52;c;payload\a\u202e"
	m := New(".", 80, 24)
	m.input = hostile
	m.inputPos = len(hostile)
	m.suggestions = []string{hostile}

	view := m.View(theme.Default(), 80, 24)
	safe := uitext.SafeTerminalText(hostile)
	if !strings.Contains(view, safe) {
		t.Fatalf("command view should contain sanitized input/suggestion %q: %q", safe, view)
	}
	if strings.Contains(view, hostile) {
		t.Fatalf("command view exposed raw input or suggestion: %q", view)
	}
	if m.input != hostile || m.inputPos != len(hostile) || m.suggestions[0] != hostile {
		t.Fatalf("rendering changed command input/completion values: %#v", m)
	}

	m.outputLines = []string{hostile}
	view = m.View(theme.Default(), 80, 24)
	if !strings.Contains(view, safe) || strings.Contains(view, hostile) {
		t.Fatalf("command output was not safely rendered: %q", view)
	}
	if m.outputLines[0] != hostile {
		t.Fatalf("rendering changed captured command output: %q", m.outputLines[0])
	}
}
