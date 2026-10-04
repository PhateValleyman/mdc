package cli

import (
	"strings"
	"testing"

	"github.com/kooler/MiddayCommander/internal/ui/theme"
)

func TestParseHelpAndVersion(t *testing.T) {
	for _, arg := range []string{"--help", "-h", "-?"} {
		opts, err := Parse([]string{arg})
		if err != nil {
			t.Fatalf("Parse(%q) failed: %v", arg, err)
		}
		if !opts.ShowHelp {
			t.Errorf("Parse(%q) ShowHelp = false, want true", arg)
		}
	}

	for _, arg := range []string{"--version", "-v"} {
		opts, err := Parse([]string{arg})
		if err != nil {
			t.Fatalf("Parse(%q) failed: %v", arg, err)
		}
		if !opts.ShowVersion {
			t.Errorf("Parse(%q) ShowVersion = false, want true", arg)
		}
	}
}

func TestParseReturnPath(t *testing.T) {
	opts, err := Parse([]string{"-r"})
	if err != nil {
		t.Fatalf("Parse(-r) failed: %v", err)
	}
	if !opts.ReturnPath {
		t.Errorf("Parse(-r) ReturnPath = false, want true")
	}

	opts, err = Parse([]string{"--return-path"})
	if err != nil {
		t.Fatalf("Parse(--return-path) failed: %v", err)
	}
	if !opts.ReturnPath {
		t.Errorf("Parse(--return-path) ReturnPath = false, want true")
	}
}

func TestParseColorFlags(t *testing.T) {
	tests := []struct {
		args      []string
		wantMode  theme.ColorMode
		wantProf  string
		wantError bool
	}{
		{[]string{"-c"}, theme.ColorAlways, "", false},
		{[]string{"--color"}, theme.ColorAlways, "", false},
		{[]string{"--color=always"}, theme.ColorAlways, "", false},
		{[]string{"--color=auto"}, theme.ColorAuto, "", false},
		{[]string{"--color=never"}, theme.ColorNever, "", false},
		{[]string{"--color=invalid"}, theme.ColorAuto, "", true},
		{[]string{"-b"}, theme.ColorNever, "", false},
		{[]string{"--no-color"}, theme.ColorNever, "", false},
		{[]string{"--color-profile", "truecolor"}, theme.ColorAuto, "truecolor", false},
		{[]string{"--color-profile=256"}, theme.ColorAuto, "256", false},
		{[]string{"--color-profile=invalid"}, theme.ColorAuto, "", true},
		{[]string{"--color-profile"}, theme.ColorAuto, "", true},
	}

	for _, tt := range tests {
		opts, err := Parse(tt.args)
		if tt.wantError {
			if err == nil {
				t.Errorf("Parse(%v) expected error, got nil", tt.args)
			}
			continue
		}
		if err != nil {
			t.Errorf("Parse(%v) unexpected error: %v", tt.args, err)
			continue
		}
		if opts.ColorMode != tt.wantMode {
			t.Errorf("Parse(%v) ColorMode = %v, want %v", tt.args, opts.ColorMode, tt.wantMode)
		}
		if opts.ColorProfile != tt.wantProf {
			t.Errorf("Parse(%v) ColorProfile = %q, want %q", tt.args, opts.ColorProfile, tt.wantProf)
		}
	}
}

func TestParseThemeFlag(t *testing.T) {
	tests := []struct {
		args      []string
		wantTheme string
		wantError bool
	}{
		{[]string{"-t", "mc-classic"}, "mc-classic", false},
		{[]string{"-tmc-classic"}, "mc-classic", false},
		{[]string{"-t=mc-classic"}, "mc-classic", false},
		{[]string{"--theme", "tokyo-night"}, "tokyo-night", false},
		{[]string{"--theme=tokyo-night"}, "tokyo-night", false},
		{[]string{"-t"}, "", true},
		{[]string{"--theme"}, "", true},
	}

	for _, tt := range tests {
		opts, err := Parse(tt.args)
		if tt.wantError {
			if err == nil {
				t.Errorf("Parse(%v) expected error, got nil", tt.args)
			}
			continue
		}
		if err != nil {
			t.Errorf("Parse(%v) unexpected error: %v", tt.args, err)
			continue
		}
		if opts.Theme != tt.wantTheme {
			t.Errorf("Parse(%v) Theme = %q, want %q", tt.args, opts.Theme, tt.wantTheme)
		}
	}
}

func TestParsePositionalPaths(t *testing.T) {
	// Single path
	opts, err := Parse([]string{"/var/log"})
	if err != nil {
		t.Fatalf("Parse(/var/log) failed: %v", err)
	}
	if opts.LeftPath != "/var/log" || opts.RightPath != "" {
		t.Errorf("Parse(/var/log) = (%q, %q), want (/var/log, '')", opts.LeftPath, opts.RightPath)
	}

	// Two paths
	opts, err = Parse([]string{"/etc", "/var"})
	if err != nil {
		t.Fatalf("Parse(/etc, /var) failed: %v", err)
	}
	if opts.LeftPath != "/etc" || opts.RightPath != "/var" {
		t.Errorf("Parse(/etc, /var) = (%q, %q), want (/etc, /var)", opts.LeftPath, opts.RightPath)
	}

	// Flags combined with paths
	opts, err = Parse([]string{"-c", "-t", "mc-classic", "/tmp", "/home"})
	if err != nil {
		t.Fatalf("Parse flags+paths failed: %v", err)
	}
	if opts.ColorMode != theme.ColorAlways || opts.Theme != "mc-classic" || opts.LeftPath != "/tmp" || opts.RightPath != "/home" {
		t.Errorf("Parse flags+paths got %+v", opts)
	}

	// Too many paths
	_, err = Parse([]string{"/a", "/b", "/c"})
	if err == nil {
		t.Errorf("Parse 3 paths expected error, got nil")
	}

	// Double dash delimiter
	opts, err = Parse([]string{"--", "-not-a-flag"})
	if err != nil {
		t.Fatalf("Parse with -- failed: %v", err)
	}
	if opts.LeftPath != "-not-a-flag" {
		t.Errorf("Parse with -- LeftPath = %q, want '-not-a-flag'", opts.LeftPath)
	}
}

func TestParseUnknownFlag(t *testing.T) {
	_, err := Parse([]string{"--unknown-flag"})
	if err == nil {
		t.Errorf("Parse(--unknown-flag) expected error, got nil")
	}

	_, err = Parse([]string{"-z"})
	if err == nil {
		t.Errorf("Parse(-z) expected error, got nil")
	}
}

func TestHelpText(t *testing.T) {
	txt := HelpText()
	if !strings.Contains(txt, "Usage:") {
		t.Errorf("HelpText missing 'Usage:'")
	}
	if !strings.Contains(txt, "--help") {
		t.Errorf("HelpText missing '--help'")
	}
	if !strings.Contains(txt, "--color") {
		t.Errorf("HelpText missing '--color'")
	}
}
