package theme

import (
	"os"
	"strings"

	"github.com/charmbracelet/lipgloss"
	"github.com/muesli/termenv"
)

// ColorMode defines how terminal color output should be handled.
type ColorMode string

const (
	ColorAuto   ColorMode = "auto"
	ColorAlways ColorMode = "always"
	ColorNever  ColorMode = "never"
)

// ParseColorProfile parses a string name into a termenv.Profile.
func ParseColorProfile(s string) (termenv.Profile, bool) {
	switch strings.ToLower(strings.TrimSpace(s)) {
	case "truecolor", "24bit", "tc", "rgb":
		return termenv.TrueColor, true
	case "256", "ansi256", "256color", "8bit":
		return termenv.ANSI256, true
	case "ansi", "16", "16color", "4bit":
		return termenv.ANSI, true
	case "ascii", "mono", "monochrome", "none", "0bit":
		return termenv.Ascii, true
	default:
		return termenv.Ascii, false
	}
}

// SetupColor configures the global lipgloss color profile based on user settings
// and the execution environment.
func SetupColor(mode ColorMode, explicitProfile string, tty *os.File) termenv.Profile {
	// If a dedicated TTY output file was provided (e.g. mdc -r), configure
	// lipgloss to direct output queries and rendering to that TTY.
	if tty != nil {
		lipgloss.DefaultRenderer().SetOutput(termenv.NewOutput(tty))
	}

	// 1. Explicit profile requested via flag
	if explicitProfile != "" {
		if p, ok := ParseColorProfile(explicitProfile); ok {
			lipgloss.SetColorProfile(p)
			return p
		}
	}

	// 2. Color explicitly disabled: mode=never or NO_COLOR env var (unless mode=always)
	if mode == ColorNever || (mode != ColorAlways && os.Getenv("NO_COLOR") != "") {
		lipgloss.SetColorProfile(termenv.Ascii)
		return termenv.Ascii
	}

	// 3. Forced color mode
	if mode == ColorAlways {
		p := DetectBestColorProfile()
		if p == termenv.Ascii {
			// If auto-detection yielded Ascii, force at least ANSI256 for vibrant UI
			p = termenv.ANSI256
		}
		lipgloss.SetColorProfile(p)
		return p
	}

	// 4. Auto mode: check lipgloss detected profile first
	current := lipgloss.ColorProfile()
	if current != termenv.Ascii {
		return current
	}

	// If termenv detected Ascii, check if terminal environment is capable of color.
	// Many servers (e.g. over SSH) report TERM=xterm, vt100, screen, etc. without COLORTERM,
	// which causes standard termenv heuristics to fall back to monochrome Ascii.
	p := DetectBestColorProfile()
	lipgloss.SetColorProfile(p)
	return p
}

// DetectBestColorProfile evaluates environment variables to choose the most suitable
// color profile, especially on SSH/server environments where termenv's default heuristics
// fall back to monochrome Ascii (e.g. plain TERM=xterm without COLORTERM).
func DetectBestColorProfile() termenv.Profile {
	term := strings.ToLower(os.Getenv("TERM"))
	colorTerm := strings.ToLower(os.Getenv("COLORTERM"))

	if term == "dumb" {
		return termenv.Ascii
	}

	// TrueColor / 24-bit
	if colorTerm == "truecolor" || colorTerm == "24bit" ||
		term == "xterm-kitty" || term == "wezterm" || term == "alacritty" ||
		term == "foot" || strings.Contains(term, "direct") {
		return termenv.TrueColor
	}

	// 256 colors
	if strings.Contains(term, "256") ||
		strings.HasPrefix(term, "xterm") ||
		strings.HasPrefix(term, "rxvt") ||
		strings.HasPrefix(term, "screen") ||
		strings.HasPrefix(term, "tmux") ||
		strings.HasPrefix(term, "st") ||
		strings.HasPrefix(term, "cygwin") ||
		strings.HasPrefix(term, "putty") ||
		strings.HasPrefix(term, "konsole") {
		return termenv.ANSI256
	}

	// 16 colors (ANSI)
	if strings.HasPrefix(term, "vt") ||
		term == "linux" ||
		strings.Contains(term, "color") ||
		strings.Contains(term, "ansi") {
		return termenv.ANSI
	}

	// Fallback for interactive sessions with a set TERM:
	// Default to ANSI256 so themes render nicely on modern terminals.
	if term != "" {
		return termenv.ANSI256
	}

	return termenv.Ascii
}
