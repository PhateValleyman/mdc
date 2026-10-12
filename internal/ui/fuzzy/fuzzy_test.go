package fuzzy

import (
	"path/filepath"
	"strings"
	"testing"

	uitext "github.com/kooler/MiddayCommander/internal/ui/text"
	"github.com/kooler/MiddayCommander/internal/ui/theme"
)

func TestViewSanitizesFilesystemPathsOnlyForRender(t *testing.T) {
	const root = "/safe/root"
	const hostileName = "f\x1b]x\x07\u202e"
	path := filepath.Join(root, hostileName)
	m := New(root, 80, 24)
	m.walking = false
	m.allPaths = []string{path}
	m.matches = []match{{path: path}}

	view := m.View(theme.Default(), 80, 24)
	safeName := uitext.SafeTerminalText(hostileName)
	if !strings.Contains(view, safeName) {
		t.Fatalf("fuzzy view should contain sanitized path label %q: %q", safeName, view)
	}
	if strings.Contains(view, hostileName) {
		t.Fatalf("fuzzy view exposed the raw path label: %q", view)
	}
	if m.matches[0].path != path || m.allPaths[0] != path {
		t.Fatal("rendering changed the selected or indexed filesystem path")
	}
}
