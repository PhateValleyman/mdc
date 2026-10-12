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

func TestRemoteSettingsRowsOpenExistingManagers(t *testing.T) {
	for _, tc := range []struct {
		index int
		want  any
	}{{2, ServersRequestedMsg{}}, {3, RcloneRequestedMsg{}}} {
		m := New("gotar", true, 80, 24)
		for i := 0; i < tc.index; i++ {
			m, _ = m.Update(tea.KeyMsg{Type: tea.KeyDown})
		}
		_, cmd := m.Update(tea.KeyMsg{Type: tea.KeyEnter})
		if cmd == nil || cmd() != tc.want {
			t.Errorf("row %d should request %T", tc.index, tc.want)
		}
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
	for _, size := range [][2]int{{12, 8}, {18, 12}, {24, 24}, {32, 24}, {40, 24}, {52, 24}, {53, 24}, {54, 24}, {80, 24}} {
		width, height := size[0], size[1]
		m := New("gotar", true, width, height)
		boxWidth, boxHeight := m.BoxSize(width, height)
		view := m.View(theme.Default(), width, height)
		if got := strings.Count(view, "╞"); got != 4 {
			t.Errorf("width %d: got %d framed choices, want 4", width, got)
		}
		if got := len(strings.Split(view, "\n")); got > boxHeight {
			t.Errorf("size %dx%d produced %d rows, box height %d", width, height, got, boxHeight)
		}
		for row, line := range strings.Split(view, "\n") {
			if got := ansi.StringWidth(line); got > boxWidth {
				t.Errorf("width %d row %d = %d cells, box width %d", width, row, got, boxWidth)
			}
		}
	}
}

func TestShortTerminalHeightsKeepEverySettingsRowReachable(t *testing.T) {
	for _, height := range []int{5, 6, 7, 8} {
		m := New("gotar", true, 40, height)
		visible, _ := m.visibleRows()
		for index := 0; index < 4; index++ {
			if m.cursor < m.offset || m.cursor >= m.offset+visible {
				t.Fatalf("height %d cursor %d is outside viewport offset=%d visible=%d", height, m.cursor, m.offset, visible)
			}
			view := m.View(theme.Default(), 40, height)
			if got := strings.Count(view, "╞"); got != visible {
				t.Fatalf("height %d cursor %d shows %d framed rows, want %d", height, index, got, visible)
			}
			if index < 3 {
				m, _ = m.Update(tea.KeyMsg{Type: tea.KeyDown})
			}
		}
	}
}
