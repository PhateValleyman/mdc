package theme

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/charmbracelet/lipgloss"
)

func TestLoadByName(t *testing.T) {
	// Set up a temp config dir so LoadByName finds the theme file
	// (the repo's themes/ dir isn't in ~/.config/mdc/themes on CI).
	tmp := t.TempDir()
	t.Setenv("XDG_CONFIG_HOME", tmp)
	themesDir := filepath.Join(tmp, "mdc", "themes")
	if err := os.MkdirAll(themesDir, 0o755); err != nil {
		t.Fatal(err)
	}
	data, err := os.ReadFile(filepath.Join("..", "..", "..", "themes", "catppuccin-mocha.toml"))
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(themesDir, "catppuccin-mocha.toml"), data, 0o644); err != nil {
		t.Fatal(err)
	}

	th, err := LoadByName("catppuccin-mocha")
	if err != nil {
		t.Fatalf("LoadByName error: %v", err)
	}

	// Verify non-default colors were applied
	def := Default()

	// FileNormal should differ from default (catppuccin uses #cdd6f4 on #1e1e2e, not ANSI 15 on 4)
	if th.FileNormal.GetForeground() == def.FileNormal.GetForeground() &&
		th.FileNormal.GetBackground() == def.FileNormal.GetBackground() {
		t.Errorf("FileNormal was not overridden by theme")
	}

	t.Logf("FileNormal fg=%v bg=%v", th.FileNormal.GetForeground(), th.FileNormal.GetBackground())
	t.Logf("FileDir fg=%v bg=%v bold=%v", th.FileDir.GetForeground(), th.FileDir.GetBackground(), th.FileDir.GetBold())
	t.Logf("PanelBorder fg=%v bg=%v", th.PanelBorder.GetForeground(), th.PanelBorder.GetBackground())
	t.Logf("StatusBar fg=%v bg=%v", th.StatusBar.GetForeground(), th.StatusBar.GetBackground())
}

func TestOrDefault(t *testing.T) {
	def := lipgloss.NewStyle().Foreground(lipgloss.Color("15"))
	empty := lipgloss.NewStyle()
	colored := lipgloss.NewStyle().Foreground(lipgloss.Color("#ff0000"))

	result := orDefault(empty, def)
	if result.GetForeground() != def.GetForeground() {
		t.Errorf("orDefault should return default for empty style")
	}

	result = orDefault(colored, def)
	if result.GetForeground() == def.GetForeground() {
		t.Errorf("orDefault should return colored style, not default")
	}
}

func TestGotarIsTheEmbeddedDefault(t *testing.T) {
	if DefaultKey != "gotar" {
		t.Fatalf("default key = %q, want gotar", DefaultKey)
	}
	defaultTheme := Resolve("")
	gotarTheme, err := LoadFromFile("gotar.toml")
	if err != nil {
		t.Fatal(err)
	}
	if defaultTheme.PanelBorder.GetForeground() != gotarTheme.PanelBorder.GetForeground() ||
		defaultTheme.FileCursor.GetBackground() != gotarTheme.FileCursor.GetBackground() {
		t.Fatal("empty theme selection did not resolve to Gotar")
	}
}

func TestLegacyThemeDerivesOverlayPalette(t *testing.T) {
	data := []byte(`
[palette]
bg = "#101820"
fg = "#f0f0f0"
subtle = "#8090a0"
accent = "#50c0d0"
selected = "#e0c050"
cursor = "#4080a0"
success = "#60c060"
border = "#303840"
active = "#70b0c0"

[panel]
border_fg = "border"
border_active_fg = "active"

[panel.file]
normal_fg = "fg"
normal_bg = "bg"
exec_fg = "success"
cursor_bg = "cursor"
selected_fg = "selected"

[menubar]
fkey_hint_fg = "accent"
`)
	loaded, err := ParseTOML("legacy", data)
	if err != nil {
		t.Fatal(err)
	}
	got := loaded.Theme.Overlay
	want := OverlayPalette{
		Background: "#101820",
		Foreground: "#f0f0f0",
		Subtle:     "#50c0d0",
		Accent:     "#50c0d0",
		Highlight:  "#e0c050",
		CursorBG:   "#4080a0",
		Success:    "#60c060",
		Danger:     Default().Overlay.Danger,
		Border:     "#303840",
	}
	if got != want {
		t.Fatalf("derived overlay palette = %#v, want %#v", got, want)
	}
}

func TestOverlayPaletteCanBeOverridden(t *testing.T) {
	data := []byte(`
[palette]
bg = "#101820"
fg = "#f0f0f0"
custom_bg = "#202830"
custom_accent = "#f09030"

[panel.file]
normal_fg = "fg"
normal_bg = "bg"

[overlay]
bg = "custom_bg"
accent = "custom_accent"
`)
	loaded, err := ParseTOML("custom", data)
	if err != nil {
		t.Fatal(err)
	}
	if loaded.Theme.Overlay.Background != lipgloss.Color("#202830") || loaded.Theme.Overlay.Accent != lipgloss.Color("#f09030") {
		t.Fatalf("explicit overlay colors not applied: %#v", loaded.Theme.Overlay)
	}
}

func TestZeroThemeOverlayStylesUseMCClassicFallback(t *testing.T) {
	want := Default().Overlay
	got := (Theme{}).OverlayStyles()
	if got.Base.GetBackground() != want.Background || got.Base.GetForeground() != want.Foreground {
		t.Fatalf("zero-value theme did not use fallback colors: fg=%v bg=%v", got.Base.GetForeground(), got.Base.GetBackground())
	}
}
