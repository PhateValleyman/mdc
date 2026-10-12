package text

import (
	"strings"
	"unicode"

	tea "github.com/charmbracelet/bubbletea"
)

// SafeTerminalText replaces terminal control and bidi formatting characters
// before untrusted text is rendered, preventing control sequences, layout breaks,
// and visually reordered filenames or remote metadata.
func SafeTerminalText(s string) string {
	return strings.Map(func(r rune) rune {
		if unicode.IsControl(r) || isBidiControl(r) {
			return '�'
		}
		return r
	}, s)
}

func isBidiControl(r rune) bool {
	switch r {
	case '\u061c', '\u200e', '\u200f':
		return true
	default:
		return r >= '\u202a' && r <= '\u202e' || r >= '\u2066' && r <= '\u2069'
	}
}

// PrintableInput returns the printable text represented by a key message.
// Bubble Tea reports a regular space as KeySpace, while other text uses
// KeyRunes and may contain more than one rune when pasted or entered via IME.
func PrintableInput(msg tea.KeyMsg) (string, bool) {
	// Alt+key is a shortcut, not text.
	if msg.Alt {
		return "", false
	}
	if msg.Type != tea.KeyRunes && msg.Type != tea.KeySpace {
		return "", false
	}
	s := string(msg.Runes)
	if msg.Type == tea.KeySpace {
		s = " "
	}
	if s == "" {
		return "", false
	}
	for _, r := range s {
		if r < 32 {
			return "", false
		}
	}
	return s, true
}
