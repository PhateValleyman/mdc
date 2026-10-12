package app

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/charmbracelet/x/ansi"
)

func TestStatusBarShowsCurrentItemAndSanitizesFilename(t *testing.T) {
	isolate(t)
	dir := t.TempDir()
	name := "report\x1b[31m.txt"
	if err := os.WriteFile(filepath.Join(dir, name), make([]byte, 2048), 0o644); err != nil {
		t.Fatal(err)
	}

	m := newModel(t)
	m.leftPanel.SetPath(dir)
	m = loadPanel(t, m)
	m = selectEntry(t, m, name)

	got := m.statusBarView()
	if !strings.Contains(got, "LEFT") {
		t.Errorf("status bar should identify the active pane: %q", got)
	}
	if !strings.Contains(got, "report�[31m.txt") {
		t.Errorf("status bar should show a sanitized filename: %q", got)
	}
	if strings.Contains(got, name) {
		t.Errorf("status bar rendered a raw control character: %q", got)
	}
	if !strings.Contains(got, "2.0K") {
		t.Errorf("status bar should show the selected file size: %q", got)
	}
}

func TestStatusBarFitsNarrowTerminalWidths(t *testing.T) {
	m := newModel(t)
	for _, width := range []int{1, 4, 20, 80} {
		m.width = width
		if got := ansi.StringWidth(m.statusBarView()); got != width {
			t.Errorf("status bar width at terminal width %d = %d, want %d", width, got, width)
		}
	}
}

func TestStatusBarReservesOneRowOnlyWhenTerminalCanUseIt(t *testing.T) {
	for _, height := range []int{30, 6, 5, 4} {
		t.Run(itoa(height), func(t *testing.T) {
			m := newModel(t)
			m.width = 80
			m.height = height
			m.recalcLayout()

			view := m.View()
			got := strings.Count(view, "\n") + 1
			if got != height {
				t.Errorf("rendered height = %d lines, want %d", got, height)
			}
			if height >= 6 && !strings.Contains(view, "LEFT") {
				t.Error("expected the contextual status bar at usable heights")
			}
		})
	}
}
