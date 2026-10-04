package theme

import (
	"os"
	"testing"

	"github.com/muesli/termenv"
)

func TestParseColorProfile(t *testing.T) {
	tests := []struct {
		input   string
		want    termenv.Profile
		wantOk  bool
	}{
		{"truecolor", termenv.TrueColor, true},
		{"24bit", termenv.TrueColor, true},
		{"256", termenv.ANSI256, true},
		{"ansi256", termenv.ANSI256, true},
		{"ansi", termenv.ANSI, true},
		{"16", termenv.ANSI, true},
		{"ascii", termenv.Ascii, true},
		{"none", termenv.Ascii, true},
		{"invalid", termenv.Ascii, false},
	}

	for _, tt := range tests {
		got, ok := ParseColorProfile(tt.input)
		if ok != tt.wantOk || got != tt.want {
			t.Errorf("ParseColorProfile(%q) = (%v, %v), want (%v, %v)", tt.input, got, ok, tt.want, tt.wantOk)
		}
	}
}

func TestDetectBestColorProfile(t *testing.T) {
	tests := []struct {
		name      string
		term      string
		colorterm string
		want      termenv.Profile
	}{
		{
			name: "dumb terminal",
			term: "dumb",
			want: termenv.Ascii,
		},
		{
			name:      "COLORTERM=truecolor",
			term:      "xterm",
			colorterm: "truecolor",
			want:      termenv.TrueColor,
		},
		{
			name: "xterm-kitty",
			term: "xterm-kitty",
			want: termenv.TrueColor,
		},
		{
			name: "plain xterm on server without COLORTERM",
			term: "xterm",
			want: termenv.ANSI256,
		},
		{
			name: "xterm-256color",
			term: "xterm-256color",
			want: termenv.ANSI256,
		},
		{
			name: "screen on server",
			term: "screen",
			want: termenv.ANSI256,
		},
		{
			name: "tmux on server",
			term: "tmux",
			want: termenv.ANSI256,
		},
		{
			name: "vt100",
			term: "vt100",
			want: termenv.ANSI,
		},
		{
			name: "linux console",
			term: "linux",
			want: termenv.ANSI,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Setenv("TERM", tt.term)
			t.Setenv("COLORTERM", tt.colorterm)
			got := DetectBestColorProfile()
			if got != tt.want {
				t.Errorf("DetectBestColorProfile() with TERM=%q COLORTERM=%q = %v, want %v",
					tt.term, tt.colorterm, got, tt.want)
			}
		})
	}
}

func TestSetupColor(t *testing.T) {
	// 1. ColorNever should force Ascii
	t.Run("ColorNever", func(t *testing.T) {
		t.Setenv("TERM", "xterm-256color")
		p := SetupColor(ColorNever, "", nil)
		if p != termenv.Ascii {
			t.Errorf("SetupColor(ColorNever) = %v, want Ascii", p)
		}
	})

	// 2. NO_COLOR environment variable should force Ascii in auto mode
	t.Run("NO_COLOR env", func(t *testing.T) {
		t.Setenv("NO_COLOR", "1")
		t.Setenv("TERM", "xterm-256color")
		p := SetupColor(ColorAuto, "", nil)
		if p != termenv.Ascii {
			t.Errorf("SetupColor(ColorAuto) with NO_COLOR=1 = %v, want Ascii", p)
		}
	})

	// 3. Explicit profile flag should take precedence
	t.Run("ExplicitProfile", func(t *testing.T) {
		t.Setenv("TERM", "dumb")
		p := SetupColor(ColorAuto, "truecolor", nil)
		if p != termenv.TrueColor {
			t.Errorf("SetupColor(ColorAuto, truecolor) = %v, want TrueColor", p)
		}
	})

	// 4. ColorAlways should force color even on dumb/xterm
	t.Run("ColorAlways", func(t *testing.T) {
		t.Setenv("NO_COLOR", "")
		os.Unsetenv("NO_COLOR")
		t.Setenv("TERM", "xterm")
		p := SetupColor(ColorAlways, "", nil)
		if p == termenv.Ascii {
			t.Errorf("SetupColor(ColorAlways) = Ascii, want colored profile")
		}
	})
}
