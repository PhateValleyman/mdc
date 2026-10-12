package settings

import (
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/x/ansi"

	uitext "github.com/kooler/MiddayCommander/internal/ui/text"
	"github.com/kooler/MiddayCommander/internal/ui/theme"
)

func TestThemeRowRequestsExistingPicker(t *testing.T) {
	m := New("gotar", true, 80, 24)
	_, cmd := m.Update(tea.KeyMsg{Type: tea.KeyEnter})
	if cmd == nil {
		t.Fatal("theme row should return a command")
	}
	msg := cmd()
	if _, ok := msg.(ThemeRequestedMsg); !ok {
		t.Fatalf("theme row command returned %T, want ThemeRequestedMsg", msg)
	}
}

func TestShowHiddenRowTogglesAndReportsValue(t *testing.T) {
	m := New("gotar", true, 80, 24)
	m, _ = m.Update(tea.KeyMsg{Type: tea.KeyDown})
	m, cmd := m.Update(tea.KeyMsg{Type: tea.KeyEnter})
	if m.showHidden {
		t.Fatal("show hidden should toggle off")
	}
	msg, ok := cmd().(ShowHiddenChangedMsg)
	if !ok || msg.Show {
		t.Fatalf("toggle command = %#v, want ShowHiddenChangedMsg{Show:false}", msg)
	}
}

func TestDismissAndThemedRender(t *testing.T) {
	m := New("mc-classic", false, 80, 24)
	_, cmd := m.Update(tea.KeyMsg{Type: tea.KeyEsc})
	if cmd == nil {
		t.Fatal("Esc should dismiss settings")
	}
	if _, ok := cmd().(DismissMsg); !ok {
		t.Fatalf("Esc command returned %T, want DismissMsg", cmd())
	}

	view := m.View(theme.Default(), 80, 24)
	if !strings.Contains(view, "Settings") || !strings.Contains(view, "mc-classic") || !strings.Contains(view, "Show hidden") {
		t.Fatalf("settings view is missing expected labels: %q", view)
	}
}

func TestThemeNameIsSanitizedOnlyForRender(t *testing.T) {
	const hostileName = "evil\x1b]8;;https://evil.example\x07\u202e"
	m := New(hostileName, false, 80, 24)
	view := m.View(theme.Default(), 80, 24)
	safeName := uitext.SafeTerminalText(hostileName)

	if !strings.Contains(view, safeName) {
		t.Fatalf("rendered settings should contain sanitized theme name %q: %q", safeName, view)
	}
	if strings.Contains(view, hostileName) {
		t.Fatalf("rendered settings exposed the raw theme name: %q", view)
	}
	if m.themeName != hostileName {
		t.Fatalf("rendering changed the stored theme name: %q", m.themeName)
	}
}

func TestViewFitsNarrowTerminalWidths(t *testing.T) {
	for _, width := range []int{18, 24, 32, 40, 52, 53, 54, 80} {
		m := New("gotar", true, width, 24)
		boxWidth, _ := m.BoxSize(width, 24)
		for row, line := range strings.Split(m.View(theme.Default(), width, 24), "\n") {
			if got := ansi.StringWidth(line); got > boxWidth {
				t.Errorf("width %d row %d = %d cells, box width %d", width, row, got, boxWidth)
			}
		}
	}
}
