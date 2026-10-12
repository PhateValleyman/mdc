package text

import (
	"strings"
	"unicode"

	tea "github.com/charmbracelet/bubbletea"
)

// SafeTerminalText replaces control characters before untrusted text is rendered
// in a terminal, preventing filenames and remote metadata from emitting control
// sequences or breaking the layout.
func SafeTerminalText(s string) string {
	return strings.Map(func(r rune) rune {
		if unicode.IsControl(r) {
			return '�'
		}
		return r
	}, s)
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
