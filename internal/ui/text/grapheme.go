// Package text provides grapheme-aware helpers shared by the TUI overlays:
// cluster boundary navigation for text editing and key-message-to-text
// mapping for input fields.
package text

import (
	"github.com/rivo/uniseg"
)

// FirstGraphemeCluster returns the first grapheme cluster in s and its display width.
func FirstGraphemeCluster(s string) (string, int) {
	cluster, _, width, _ := uniseg.FirstGraphemeClusterInString(s, -1)
	return cluster, width
}

// PreviousGraphemeBoundary returns the byte index before the grapheme cluster
// at pos, or -1 when pos is already at the start.
func PreviousGraphemeBoundary(s string, pos int) int {
	if pos <= 0 {
		return -1
	}
	if pos > len(s) {
		pos = len(s)
	}
	previous, offset := 0, 0
	for offset < pos {
		cluster, _ := FirstGraphemeCluster(s[offset:])
		next := offset + len(cluster)
		if next >= pos {
			break
		}
		previous = next
		offset = next
	}
	return previous
}

// NextGraphemeBoundary returns the byte index after the grapheme cluster at
// pos, or pos when already at the end.
func NextGraphemeBoundary(s string, pos int) int {
	if pos >= len(s) {
		return pos
	}
	cluster, _ := FirstGraphemeCluster(s[pos:])
	return pos + len(cluster)
}
