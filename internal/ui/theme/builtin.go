package theme

import (
	_ "embed"
	"sync"

	"github.com/BurntSushi/toml"
)

// DefaultKey is the config key of the theme applied when none is configured.
const DefaultKey = "gotar"

//go:embed gotar.toml
var gotarTOML []byte

//go:embed catppuccin-mocha.toml
var mochaTOML []byte

var (
	gotarOnce     sync.Once
	gotarThemeVal Theme
	mochaOnce     sync.Once
	mochaThemeVal Theme
)

func gotarTheme() Theme {
	gotarOnce.Do(func() {
		var tf ThemeFile
		if err := toml.Unmarshal(gotarTOML, &tf); err != nil {
			gotarThemeVal = Default()
			return
		}
		gotarThemeVal = buildTheme(tf)
	})
	return gotarThemeVal
}

func mochaTheme() Theme {
	mochaOnce.Do(func() {
		var tf ThemeFile
		if err := toml.Unmarshal(mochaTOML, &tf); err != nil {
			mochaThemeVal = Default()
			return
		}
		mochaThemeVal = buildTheme(tf)
	})
	return mochaThemeVal
}

// builtinThemes returns themes compiled into the binary, in display order.
// The first entry is the default.
func builtinThemes() []AvailableTheme {
	return []AvailableTheme{
		{Key: DefaultKey, Name: "Gotar", Source: SourceDefault, Theme: gotarTheme()},
		{Key: "catppuccin-mocha", Name: "Catppuccin Mocha", Source: SourceDefault, Theme: mochaTheme()},
		{Key: "mc-classic", Name: "MC Classic", Source: SourceDefault, Theme: Default()},
	}
}

// Resolve returns the theme for the given config key, honoring built-ins before
// disk lookup. An empty key selects Gotar.
func Resolve(key string) Theme {
	if key == "" {
		key = DefaultKey
	}
	for _, b := range builtinThemes() {
		if b.Key == key {
			return b.Theme
		}
	}
	if loaded, err := LoadByName(key); err == nil {
		return loaded
	}
	return gotarTheme()
}
