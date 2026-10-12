package bookmarks

import (
	"strings"
	"testing"
	"unicode/utf8"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/kooler/MiddayCommander/internal/bookmark"
	uitext "github.com/kooler/MiddayCommander/internal/ui/text"
	"github.com/kooler/MiddayCommander/internal/ui/theme"
)

func bookmarkKey(m Model, key tea.KeyType, runes ...rune) Model {
	var msg tea.KeyMsg
	if runes != nil {
		msg = tea.KeyMsg{Type: tea.KeyRunes, Runes: runes}
	} else {
		msg = tea.KeyMsg{Type: key}
	}
	updated, _ := m.Update(msg)
	return updated
}

func TestFilterUnicodeEditing(t *testing.T) {
	m := New(&bookmark.Store{}, ".", 80, 24)
	m.filtering = true
	m = bookmarkKey(m, tea.KeyRunes, '日', '本', '🎉')
	m = bookmarkKey(m, tea.KeyBackspace)
	if m.filter != "日本" || !utf8.ValidString(m.filter) {
		t.Errorf("filter after backspace = %q, want 日本", m.filter)
	}
	m = bookmarkKey(m, tea.KeySpace)
	if m.filter != "日本 " {
		t.Errorf("filter after space = %q, want 日本 ", m.filter)
	}
}

func TestAddNameUnicodeEditing(t *testing.T) {
	m := New(&bookmark.Store{}, ".", 80, 24)
	m.adding = true
	m = bookmarkKey(m, tea.KeyRunes, '👨', '\u200d', '👩')
	m = bookmarkKey(m, tea.KeyBackspace)
	if m.addName != "" {
		t.Errorf("name after backspace = %q, want empty", m.addName)
	}
}

func TestViewSanitizesBookmarkLabelsOnlyForRender(t *testing.T) {
	const hostileName = "n\x1b]x\x07\u202e"
	const hostilePath = "/p\x1b]y\x07\u202e"
	m := Model{items: []bookmark.Bookmark{{Name: hostileName, Path: hostilePath}}, width: 80, height: 24}

	view := m.View(theme.Default(), 80, 24)
	want := uitext.SafeTerminalText(hostileName + " → " + hostilePath)
	if !strings.Contains(view, want) {
		t.Fatalf("bookmark view should contain sanitized label %q: %q", want, view)
	}
	if strings.Contains(view, hostileName) || strings.Contains(view, hostilePath) {
		t.Fatalf("bookmark view exposed a raw label: %q", view)
	}
	if m.items[0].Name != hostileName || m.items[0].Path != hostilePath {
		t.Fatalf("rendering changed bookmark data: %#v", m.items[0])
	}
}

func TestInteractiveBookmarkInputsSanitizedOnlyForRender(t *testing.T) {
	const hostile = "search\u202e"
	m := New(&bookmark.Store{}, ".", 80, 24)
	m.filtering = true
	m.filter = hostile

	view := m.View(theme.Default(), 80, 24)
	if !strings.Contains(view, uitext.SafeTerminalText(hostile)) || strings.Contains(view, hostile) {
		t.Fatalf("bookmark filter was not safely rendered: %q", view)
	}
	if m.filter != hostile {
		t.Fatalf("rendering changed bookmark filter: %q", m.filter)
	}

	m.filtering = false
	m.adding = true
	m.addName = hostile
	view = m.View(theme.Default(), 80, 24)
	if !strings.Contains(view, uitext.SafeTerminalText(hostile)) || strings.Contains(view, hostile) {
		t.Fatalf("bookmark add-name input was not safely rendered: %q", view)
	}
	if m.addName != hostile {
		t.Fatalf("rendering changed bookmark name input: %q", m.addName)
	}
}
