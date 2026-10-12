package copypath

import (
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	uitext "github.com/kooler/MiddayCommander/internal/ui/text"
	"github.com/kooler/MiddayCommander/internal/ui/theme"
)

func TestBuildPaths(t *testing.T) {
	sep := string(filepath.Separator)
	join := func(parts ...string) string { return filepath.Join(parts...) }

	tests := []struct {
		name string
		in   string
		want []string
	}{
		{
			name: "nested file",
			in:   join(sep, "users", "bob", "data", "1.txt"),
			want: []string{
				"1.txt",
				join("data", "1.txt"),
				join("bob", "data", "1.txt"),
				join(sep, "users", "bob", "data", "1.txt"),
			},
		},
		{
			name: "single parent",
			in:   join(sep, "home", "1.txt"),
			want: []string{
				"1.txt",
				join(sep, "home", "1.txt"),
			},
		},
		{
			name: "file at root",
			in:   join(sep, "1.txt"),
			want: []string{
				"1.txt",
				join(sep, "1.txt"),
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := buildPaths(tt.in)
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("buildPaths(%q)\n got: %#v\nwant: %#v", tt.in, got, tt.want)
			}
		})
	}
}

func TestViewSanitizesPathsOnlyForRender(t *testing.T) {
	const hostileName = "f\x1b]x\x07\u202e"
	fullPath := filepath.Join(string(filepath.Separator), hostileName)
	m := New(fullPath, 80, 24)

	view := m.View(theme.Default(), 80, 24)
	safeName := uitext.SafeTerminalText(hostileName)
	if !strings.Contains(view, safeName) {
		t.Fatalf("copy-path view should contain sanitized path %q: %q", safeName, view)
	}
	if strings.Contains(view, hostileName) {
		t.Fatalf("copy-path view exposed the raw path: %q", view)
	}
	if m.paths[0] != hostileName || m.paths[len(m.paths)-1] != fullPath {
		t.Fatalf("rendering changed copy candidates: %#v", m.paths)
	}
}
