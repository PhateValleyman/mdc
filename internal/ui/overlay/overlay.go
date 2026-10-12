package overlay

import (
	"strings"

	"github.com/charmbracelet/lipgloss"
	"github.com/charmbracelet/x/ansi"

	uitext "github.com/kooler/MiddayCommander/internal/ui/text"
	"github.com/kooler/MiddayCommander/internal/ui/theme"
)

// Place composites a box on top of a background string, centered.
// The background remains visible around the box.
func Place(bg string, boxContent string, bgWidth, bgHeight, boxWidth, boxHeight int) string {
	bgLines := strings.Split(bg, "\n")
	for len(bgLines) < bgHeight {
		bgLines = append(bgLines, strings.Repeat(" ", bgWidth))
	}

	fgLines := strings.Split(boxContent, "\n")

	xOff := (bgWidth - boxWidth) / 2
	yOff := (bgHeight - boxHeight) / 2
	if xOff < 0 {
		xOff = 0
	}
	if yOff < 0 {
		yOff = 0
	}

	for i, fgLine := range fgLines {
		bgIdx := yOff + i
		if bgIdx >= len(bgLines) {
			break
		}

		bgLine := bgLines[bgIdx]

		// Cell- and grapheme-aware slicing: keep the left and right portions of
		// the background around the foreground box without breaking ANSI escapes
		left := ansi.Truncate(bgLine, xOff, "")
		right := ansi.Cut(bgLine, xOff+boxWidth, bgWidth)

		bgLines[bgIdx] = left + fgLine + right
	}

	return strings.Join(bgLines[:bgHeight], "\n")
}

// RenderBox draws a bordered box with title, content lines, and optional footer.
func RenderBox(title string, contentLines []string, footer string, width, height int, styles theme.OverlayStyles) string {
	if width < 2 || height < 2 {
		return PadOrTrunc(title, width)
	}
	if height == 2 {
		footer = ""
	}

	borderStyle := styles.Border
	titleStyle := styles.Title
	fillStyle := styles.Base

	innerWidth := width - 2

	var lines []string

	// Top border with title (width-aware so multibyte titles align correctly)
	titleStr := " " + title + " "
	if lipgloss.Width(titleStr) > innerWidth {
		titleStr = ansi.Truncate(titleStr, innerWidth, "")
	}
	padLen := innerWidth - lipgloss.Width(titleStr)
	if padLen < 0 {
		padLen = 0
	}
	top := borderStyle.Render("┌") + titleStyle.Render(titleStr) + borderStyle.Render(strings.Repeat("─", padLen)+"┐")
	lines = append(lines, top)

	// Content rows
	contentHeight := height - 2
	if footer != "" {
		contentHeight--
	}

	for i := 0; i < contentHeight; i++ {
		var row string
		if i < len(contentLines) {
			row = fitLine(contentLines[i], innerWidth, fillStyle)
		} else {
			row = fillStyle.Render(strings.Repeat(" ", innerWidth))
		}
		lines = append(lines, borderStyle.Render("│")+row+borderStyle.Render("│"))
	}

	// Footer
	if footer != "" {
		footer = fitLine(footer, innerWidth, fillStyle)
		lines = append(lines, borderStyle.Render("│")+footer+borderStyle.Render("│"))
	}

	// Bottom border
	bottom := borderStyle.Render("└" + strings.Repeat("─", innerWidth) + "┘")
	lines = append(lines, bottom)

	return strings.Join(lines, "\n")
}

func fitLine(s string, width int, fill lipgloss.Style) string {
	if width < 1 {
		return ""
	}
	w := lipgloss.Width(s)
	if w > width {
		s = ansi.Truncate(s, width, "")
		w = lipgloss.Width(s)
	}
	if w < width {
		s += fill.Render(strings.Repeat(" ", width-w))
	}
	return s
}

// TruncateLeftEllipsis keeps the right-most cells of s, prefixing with an
// ellipsis if clipped. It never returns wider than width cells.
func TruncateLeftEllipsis(s string, width int) string {
	const ellipsis = "…"
	if width < 1 {
		return ""
	}
	w := ansi.StringWidth(s)
	if w <= width {
		return s
	}
	if width == 1 {
		return ellipsis
	}

	needToStrip := w - (width - 1)
	strippedWidth := 0
	for i := 0; i < len(s); {
		cluster, gw := uitext.FirstGraphemeCluster(s[i:])
		strippedWidth += gw
		if strippedWidth >= needToStrip {
			break
		}
		i += len(cluster)
	}
	tail := ansi.TruncateLeft(s, strippedWidth, "")
	if ansi.StringWidth(tail) > width-1 {
		tail = ""
	}
	return ellipsis + strings.Repeat(" ", width-1-ansi.StringWidth(tail)) + tail
}

// PadOrTrunc pads s with trailing spaces to exactly width cells, truncating
// from the right when s is wider. The result is exactly width cells when
// width >= 1 and empty otherwise
func PadOrTrunc(s string, width int) string {
	if width < 1 {
		return ""
	}
	w := ansi.StringWidth(s)
	if w > width {
		out := ansi.Truncate(s, width, "")
		return out + strings.Repeat(" ", width-ansi.StringWidth(out))
	}
	return s + strings.Repeat(" ", width-w)
}

// PadOrTruncDots is PadOrTrunc, but clipped values end with "..." so the
// cut stays visible. The result is exactly width cells when width >= 1
func PadOrTruncDots(s string, width int) string {
	if width < 1 {
		return ""
	}
	w := ansi.StringWidth(s)
	if w > width {
		if width > 3 {
			out := ansi.Truncate(s, width-3, "") + "..."
			return out + strings.Repeat(" ", width-ansi.StringWidth(out))
		}
		out := ansi.Truncate(s, width, "")
		return out + strings.Repeat(" ", width-ansi.StringWidth(out))
	}
	return s + strings.Repeat(" ", width-w)
}

// PadLeft right-aligns s in width cells, clipping on the left edge when s is
// wider. The result is exactly width cells when width >= 1
func PadLeft(s string, width int) string {
	if width < 1 {
		return ""
	}
	w := ansi.StringWidth(s)
	if w >= width {
		out := ansi.Truncate(s, width, "")
		return strings.Repeat(" ", width-ansi.StringWidth(out)) + out
	}
	return strings.Repeat(" ", width-w) + s
}
