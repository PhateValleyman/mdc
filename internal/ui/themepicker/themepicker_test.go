package themepicker

import (
	"strings"
	"testing"

	uitext "github.com/kooler/MiddayCommander/internal/ui/text"
	"github.com/kooler/MiddayCommander/internal/ui/theme"
)

func TestViewSanitizesThemeNamesWithoutChangingEntries(t *testing.T) {
	const hostileName = "evil\x1b]8;;https://evil.example\x07\u202e"
	const themeKey = "stable-theme-key"

	for _, source := range []string{theme.SourceLocal, theme.SourceRemote} {
		t.Run(source, func(t *testing.T) {
			entry := theme.AvailableTheme{
				Key:    themeKey,
				Name:   hostileName,
				Source: source,
				Theme:  theme.Default(),
			}
			m := New([]theme.AvailableTheme{entry}, themeKey, 80, 24)
			view := m.View(theme.Default(), 80, 24)
			safeName := uitext.SafeTerminalText(hostileName)

			if !strings.Contains(view, safeName) {
				t.Fatalf("rendered picker should contain sanitized name %q: %q", safeName, view)
			}
			if strings.Contains(view, hostileName) {
				t.Fatalf("rendered picker exposed the raw theme name: %q", view)
			}
			if m.entries[0].Name != hostileName || m.entries[0].Key != themeKey {
				t.Fatalf("rendering changed the stored entry: %#v", m.entries[0])
			}
		})
	}
}
