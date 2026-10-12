package overlay

import (
	"strings"
	"testing"

	"github.com/charmbracelet/x/ansi"

	"github.com/kooler/MiddayCommander/internal/ui/theme"
)

// TestPadOrTrunc checks padding and right-truncation to exact cell widths.
func TestPadOrTrunc(t *testing.T) {
	tests := []struct {
		name  string
		s     string
		width int
		want  string
	}{
		{"pad short", "abc", 6, "abc   "},
		{"no change", "abc", 3, "abc"},
		{"pad empty", "", 3, "   "},
		{"truncate ascii", "abcdef", 4, "abcd"},
		{"truncate ascii exact", "abcdef", 3, "abc"},
		{"pad cjk", "日本", 6, "日本  "},
		{"truncate cjk drops straddler", "日本語のファイル", 5, "日本 "},
		{"truncate cjk pads back", "日本語のファイル名", 6, "日本語"},
		{"zero width", "abc", 0, ""},
		{"negative width", "abc", -1, ""},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := PadOrTrunc(tt.s, tt.width)
			if got != tt.want {
				t.Errorf("PadOrTrunc(%q, %d) = %q, want %q", tt.s, tt.width, got, tt.want)
			}
			if w := ansi.StringWidth(got); tt.width >= 1 && w != tt.width {
				t.Errorf("PadOrTrunc(%q, %d) width = %d, want %d", tt.s, tt.width, w, tt.width)
			}
		})
	}
}

// TestPadOrTruncDots ensures clipped values end with ASCII "..." and the
// result is exactly width cells, even when a wide grapheme straddles the cut.
func TestPadOrTruncDots(t *testing.T) {
	tests := []struct {
		name  string
		s     string
		width int
	}{
		{"ascii fits", "report.pdf", 12},
		{"ascii truncate", "report.pdf", 8},
		{"cjk truncate", "日本語のファイル名前です.txt", 10},
		{"emoji truncate", "🎉🎉🎉🎉🎉report.pdf", 10},
		{"cjk at 3-cell width", "日本語", 3},
		{"single wide char", "日", 1},
		{"pad short", "ab", 6},
		{"empty", "", 4},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := PadOrTruncDots(tt.s, tt.width)
			if w := ansi.StringWidth(got); tt.width >= 1 && w != tt.width {
				t.Errorf("PadOrTruncDots(%q, %d) = %q width %d, want %d", tt.s, tt.width, got, w, tt.width)
			}
		})
	}
}

// TestPadLeft checks right-alignment to exact cell widths.
func TestPadLeft(t *testing.T) {
	tests := []struct {
		name  string
		s     string
		width int
		want  string
	}{
		{"pad", "100", 6, "   100"},
		{"no change", "100", 3, "100"},
		{"cjk pad", "日本", 8, "    日本"},
		{"truncate left", "日本語の", 5, " 日本"},
		{"empty", "", 3, "   "},
		{"zero width", "a", 0, ""},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := PadLeft(tt.s, tt.width)
			if got != tt.want {
				t.Errorf("PadLeft(%q, %d) = %q, want %q", tt.s, tt.width, got, tt.want)
			}
			if w := ansi.StringWidth(got); tt.width >= 1 && w != tt.width {
				t.Errorf("PadLeft(%q, %d) width = %d, want %d", tt.s, tt.width, w, tt.width)
			}
		})
	}
}

// TestTruncateLeftEllipsis checks left-truncation with the legacy U+2026
// glyph ("..." in sources) and exact result width.
func TestTruncateLeftEllipsis(t *testing.T) {
	const ellipsis = "…" // U+2026, one cell wide
	tests := []struct {
		name  string
		s     string
		width int
		want  string
	}{
		{"fits", "/usr/bin", 10, "/usr/bin"},
		{"left cut", "/home/user/documents/report.pdf", 12, ellipsis + "/report.pdf"},
		{"width 1", "/long/path", 1, ellipsis},
		{"zero width", "/path", 0, ""},
		{"cjk tail padded to exact width", "日本語のとても長いファイル名.txt", 8, ellipsis + " 名.txt"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := TruncateLeftEllipsis(tt.s, tt.width)
			if got != tt.want {
				t.Errorf("TruncateLeftEllipsis(%q, %d) = %q, want %q", tt.s, tt.width, got, tt.want)
			}
			if w := ansi.StringWidth(got); tt.width >= 1 && w > tt.width {
				t.Errorf("TruncateLeftEllipsis(%q, %d) = %q width %d, want <= %d", tt.s, tt.width, got, w, tt.width)
			}
		})
	}
}

func TestTruncateLeftEllipsisNarrowWideTail(t *testing.T) {
	tests := []struct {
		name  string
		s     string
		width int
		want  string
	}{
		{"cjk tail does not loop", "日本", 2, "… "},
		{"emoji tail does not loop", "report🎉", 2, "… "},
		{"wide tail fits with two cells", "日本", 3, "…本"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := TruncateLeftEllipsis(tt.s, tt.width)
			if got != tt.want {
				t.Errorf("TruncateLeftEllipsis(%q, %d) = %q, want %q", tt.s, tt.width, got, tt.want)
			}
			if w := ansi.StringWidth(got); w != tt.width {
				t.Errorf("TruncateLeftEllipsis(%q, %d) width = %d, want %d", tt.s, tt.width, w, tt.width)
			}
		})
	}
}

func TestRenderBoxClipsContentAndFooterToWidth(t *testing.T) {
	styles := theme.Default().OverlayStyles()
	view := RenderBox(
		"A title longer than the box",
		[]string{styles.Base.Render("日本語のファイル名 is too long"), styles.Selected.Render("another long row")},
		styles.Prompt.Render("Enter:確認  Esc:閉じる  extra help"),
		14,
		5,
		styles,
	)
	lines := strings.Split(view, "\n")
	if len(lines) != 5 {
		t.Fatalf("rendered %d rows, want height 5: %q", len(lines), view)
	}
	for row, line := range lines {
		if got := ansi.StringWidth(line); got != 14 {
			t.Errorf("row %d width = %d, want 14: %q", row, got, line)
		}
	}
}

func TestRenderBoxFillsContentAndFooterWithThemeBackground(t *testing.T) {
	styles := theme.Default().OverlayStyles()
	content := styles.Selected.Render("ok")
	footer := styles.Prompt.Render("Esc:Close")
	view := RenderBox("Settings", []string{content}, footer, 16, 4, styles)
	lines := strings.Split(view, "\n")
	contentFill := styles.Base.Render(strings.Repeat(" ", 12))
	footerFill := styles.Base.Render(strings.Repeat(" ", 5))
	if !strings.Contains(lines[1], content+contentFill) {
		t.Errorf("short content row did not receive the theme background fill: %q", lines[1])
	}
	if !strings.Contains(lines[2], footer+footerFill) {
		t.Errorf("short footer did not receive the theme background fill: %q", lines[2])
	}
}

func TestRenderBoxHandlesTinyDimensions(t *testing.T) {
	styles := theme.Default().OverlayStyles()
	for _, dimensions := range [][2]int{{0, 0}, {1, 1}, {2, 1}, {2, 2}} {
		view := RenderBox("Settings", []string{"content"}, "footer", dimensions[0], dimensions[1], styles)
		for row, line := range strings.Split(view, "\n") {
			if got := ansi.StringWidth(line); got > dimensions[0] {
				t.Errorf("dimensions %v row %d returned width %d: %q", dimensions, row, got, line)
			}
		}
	}
}
