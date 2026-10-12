package themepicker

import (
	"strings"
	"testing"

	"github.com/charmbracelet/lipgloss"

	uitext "github.com/kooler/MiddayCommander/internal/ui/text"
	"github.com/kooler/MiddayCommander/internal/ui/theme"
)

func TestSelectedFrameBorderUsesExplicitThemeAccent(t *testing.T) {
	th := theme.Default()
	accent := lipgloss.Color("201")
	highlight := lipgloss.Color("196")
	th.Overlay.Accent = accent
	th.Overlay.Highlight = highlight
	originalPalette := th.Overlay

	styles := th.OverlayStyles()
	border := selectedFrameBorderStyle(styles)
	if got := border.GetForeground(); got != accent {
		t.Fatalf("selected outline foreground = %v, want explicit Accent %v", got, accent)
	}
	if got := border.GetForeground(); got == styles.Highlight.GetForeground() {
		t.Fatalf("selected outline must use Accent, not Highlight fill color %v", got)
	}
	if th.Overlay != originalPalette {
		t.Fatalf("resolving overlay styles mutated the stored theme palette: before=%+v after=%+v", originalPalette, th.Overlay)
	}
}

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
