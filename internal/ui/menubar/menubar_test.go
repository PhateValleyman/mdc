package menubar

import (
	"strings"
	"testing"

	"github.com/charmbracelet/lipgloss"
	"github.com/charmbracelet/x/ansi"

	"github.com/kooler/MiddayCommander/internal/config"
	"github.com/kooler/MiddayCommander/internal/ui/theme"
)

// The bar is built from a hardcoded table, so a new binding can silently miss
// it and leave the action invisible.
func TestEveryShiftFKeyBindingIsLabelled(t *testing.T) {
	cfg := config.Default()
	items := ShiftItems(cfg)

	bindings := map[string]config.StringOrList{
		"rename":    cfg.Keys.Rename,
		"copy_path": cfg.Keys.CopyPath,
		"servers":   cfg.Keys.Servers,
	}

	for name, keys := range bindings {
		for _, k := range keys {
			pos := shiftFKeyPos(k)
			if pos < 0 {
				continue // not a shift F-key, nothing to show
			}
			if items[pos].Label == "" {
				t.Errorf("%s is bound to %s (slot F%d) but the shift bar shows no label",
					name, k, pos+1)
			}
		}
	}
}

func TestShiftBarLabelsServersOnF2(t *testing.T) {
	items := ShiftItems(config.Default())

	if got := items[1].Label; got == "" {
		t.Fatal("want a label on Shift+F2 for the SSH server list")
	}
	if items[1].RawKey != "f14" {
		t.Errorf("want Shift+F2 to map to f14, got %q", items[1].RawKey)
	}
}

func TestShiftBarKeepsUnboundSlotsBlank(t *testing.T) {
	items := ShiftItems(config.Default())

	if len(items) != 11 {
		t.Fatalf("want 10 F-key slots plus Settings, got %d", len(items))
	}
	for i, itm := range items[:10] {
		if itm.Key != "F"+itoa(i+1) {
			t.Errorf("slot %d: want key F%d, got %q", i, i+1, itm.Key)
		}
	}
	if !items[10].Pinned || items[10].Label != "Settings" {
		t.Fatalf("trailing menu item = %+v, want pinned Settings", items[10])
	}
}

func TestDefaultBarLabelsEverySlot(t *testing.T) {
	for _, itm := range DefaultItems(config.Default()) {
		if strings.TrimSpace(itm.Label) == "" {
			t.Errorf("slot %q has no label", itm.Key)
		}
	}
}

func itoa(n int) string {
	if n < 10 {
		return string(rune('0' + n))
	}
	return string(rune('0'+n/10)) + string(rune('0'+n%10))
}

func TestHandleClickNarrowTerminal(t *testing.T) {
	items := []Item{
		{Label: "One", RawKey: "one"},
		{Label: "Two", RawKey: "two"},
		{Label: "Three", RawKey: "three"},
	}
	if got := HandleClick(0, 2, items); got != "one" {
		t.Fatalf("first narrow slot = %q, want one", got)
	}
	if got := HandleClick(1, 2, items); got != "two" {
		t.Fatalf("second narrow slot = %q, want two", got)
	}
	if got := HandleClick(0, 0, items); got != "" {
		t.Fatalf("zero-width click = %q, want empty", got)
	}
}

func TestSettingsMenuItemDoesNotShiftFKeySlots(t *testing.T) {
	items := DefaultItems(config.Default())
	if len(items) != 11 || items[0].Label != "Help" || items[9].Label != "Quit" {
		t.Fatalf("F-key slots changed: %+v", items)
	}
	if got := HandleClick(0, 80, items); got != "f1" {
		t.Errorf("first function-key slot = %q, want f1", got)
	}
	if got := HandleClick(79, 80, items); got != "alt+o" {
		t.Errorf("settings menu click = %q, want alt+o", got)
	}
	if got := HandleClick(39, 40, items); got == "alt+o" {
		t.Errorf("settings should not overlap F-key slots on a narrow terminal; got %q", got)
	}
}

func TestSettingsMenuViewFitsTerminalWidth(t *testing.T) {
	for _, width := range []int{24, 54, 80} {
		view := View(theme.Default(), width, DefaultItems(config.Default()))
		if got := lipgloss.Width(view); got != width {
			t.Errorf("menu width at terminal width %d = %d", width, got)
		}
	}
}

func TestMenuChoicesUseResponsiveFramesWithoutChangingFKeySlots(t *testing.T) {
	items := DefaultItems(config.Default())
	for width := 43; width <= 80; width++ {
		view := View(theme.Default(), width, items)
		wantFrames := 10
		if width >= 64 {
			wantFrames++ // all ten F-key boxes plus Settings
		}
		if got := strings.Count(view, "╞"); got != wantFrames {
			t.Errorf("%d-column menu has %d framed choices, want %d", width, got, wantFrames)
		}
		if got := ansi.StringWidth(view); got != width {
			t.Errorf("menu width at %d columns = %d", width, got)
		}
		if got := Height(width, items); got != 1 || strings.Contains(view, "\n") {
			t.Errorf("%d-column menu should remain one row, got height %d: %q", width, got, view)
		}
	}
}

func TestMenuActionLabelsStayReadableAcrossResponsiveWidths(t *testing.T) {
	items := DefaultItems(config.Default())
	cases := []struct {
		width  int
		labels []string
	}{
		{64, []string{"H", "B", "V", "E", "C", "M", "K", "D", "F", ""}},
		{80, []string{"Hp", "Bk", "Vw", "Ed", "Cp", "Mv", "Mk", "Dl", "FZ", "Qt"}},
		{120, []string{"Help", "Bkmk", "View", "Edit", "Copy", "Move", "Mkdir", "Del", "Fuzzy", "Quit"}},
	}

	for _, tc := range cases {
		view := View(theme.Default(), tc.width, items)
		widths := itemWidths(tc.width, items)
		start := 0
		for i, item := range items[:10] {
			segment := strings.TrimSpace(ansi.Strip(ansi.Cut(view, start, start+widths[i])))
			segment = strings.Trim(segment, "╞╡")
			if !strings.HasPrefix(segment, item.Key) {
				t.Errorf("width %d slot %d shows %q; want complete key %q", tc.width, i+1, segment, item.Key)
			}
			label := strings.TrimSpace(strings.TrimPrefix(segment, item.Key))
			if label != tc.labels[i] {
				t.Errorf("width %d slot %d action label = %q, want stable alias %q", tc.width, i+1, label, tc.labels[i])
			}
			if got := HandleClick(start, tc.width, items); got != item.RawKey {
				t.Errorf("width %d slot %d start click routed to %q, want %q", tc.width, i+1, got, item.RawKey)
			}
			if got := HandleClick(start+widths[i]-1, tc.width, items); got != item.RawKey {
				t.Errorf("width %d slot %d end click routed to %q, want %q", tc.width, i+1, got, item.RawKey)
			}
			start += widths[i]
		}

		settingsStart := tc.width - widths[10]
		for _, x := range []int{settingsStart, tc.width - 1} {
			if got := HandleClick(x, tc.width, items); got != items[10].RawKey {
				t.Errorf("width %d Settings click at %d routed to %q, want %q", tc.width, x, got, items[10].RawKey)
			}
		}
	}
}

func TestSettingsWaitsForCompleteFunctionKeyLabels(t *testing.T) {
	items := DefaultItems(config.Default())
	for width := 43; width <= 80; width++ {
		widths := itemWidths(width, items)
		settingsShown := width >= 64
		if (widths[10] > 0) != settingsShown {
			t.Errorf("width %d reserved %d columns for Settings, shown=%v", width, widths[10], settingsShown)
		}
		view := View(theme.Default(), width, items)
		if got := strings.Contains(ansi.Strip(view), "Settings"); got != settingsShown {
			t.Errorf("width %d Settings visible=%v, want %v: %q", width, got, settingsShown, ansi.Strip(view))
		}
		if got := ansi.StringWidth(view); got != width {
			t.Errorf("width %d rendered as %d cells", width, got)
		}
		for _, tc := range []struct {
			index int
			key   string
		}{{0, "F1"}, {8, "F9"}, {9, "F10"}} {
			start := 0
			for i := 0; i < tc.index; i++ {
				start += widths[i]
			}
			segment := strings.TrimSpace(ansi.Strip(ansi.Cut(view, start, start+widths[tc.index])))
			segment = strings.Trim(segment, "╞╡")
			if !strings.HasPrefix(segment, tc.key) {
				t.Errorf("width %d slot %d shows %q; want complete %s", width, tc.index+1, segment, tc.key)
			}
			if got := HandleClick(start, width, items); got != items[tc.index].RawKey {
				t.Errorf("width %d click at slot %d start routed to %q, want %q", width, tc.index+1, got, items[tc.index].RawKey)
			}
			if got := HandleClick(start+widths[tc.index]-1, width, items); got != items[tc.index].RawKey {
				t.Errorf("width %d click at slot %d end routed to %q, want %q", width, tc.index+1, got, items[tc.index].RawKey)
			}
		}
	}

	for _, tc := range []struct {
		width         int
		settingsShown bool
	}{{63, false}, {64, true}, {65, true}} {
		view := View(theme.Default(), tc.width, items)
		widths := itemWidths(tc.width, items)
		if got := strings.Contains(ansi.Strip(view), "Settings"); got != tc.settingsShown {
			t.Errorf("width %d Settings visible=%v, want %v", tc.width, got, tc.settingsShown)
		}
		settingsStart := tc.width - widths[10]
		for _, x := range []int{settingsStart, tc.width - 1} {
			if got := HandleClick(x, tc.width, items); (got == items[10].RawKey) != tc.settingsShown {
				t.Errorf("width %d click at column %d routed to %q, Settings shown=%v", tc.width, x, got, tc.settingsShown)
			}
		}
		if widths[9] < lipgloss.Width(items[9].Key)+2 && tc.settingsShown {
			t.Errorf("width %d reserved Settings with a clipped F10 slot width %d", tc.width, widths[9])
		}
	}
}

func TestEveryFKeyIsBoxedAndClickableAcrossWidths43To80(t *testing.T) {
	items := DefaultItems(config.Default())
	for width := 43; width <= 80; width++ {
		view := ansi.Strip(View(theme.Default(), width, items))
		widths := itemWidths(width, items)
		position := 0
		for i, item := range items[:10] {
			segment := ansi.Strip(ansi.Cut(view, position, position+widths[i]))
			if !strings.HasPrefix(segment, "╞") || !strings.HasSuffix(segment, "╡") {
				t.Errorf("width %d F%d is not individually boxed: %q", width, i+1, segment)
			}
			inside := strings.TrimSuffix(strings.TrimPrefix(segment, "╞"), "╡")
			if !strings.HasPrefix(inside, item.Key) {
				t.Errorf("width %d F%d token is clipped in %q; want complete %s", width, i+1, inside, item.Key)
			}
			for _, x := range []int{position, position + widths[i] - 1} {
				if got := HandleClickAt(x, 0, width, items); got != item.RawKey {
					t.Errorf("width %d F%d click at %d routed to %q, want %q", width, i+1, x, got, item.RawKey)
				}
			}
			position += widths[i]
		}

		wantSettings := width >= 64
		if (widths[10] > 0) != wantSettings {
			t.Errorf("width %d Settings slot width=%d, shown=%v", width, widths[10], wantSettings)
		}
		wantFrames := 10
		if wantSettings {
			wantFrames++
			settingsStart := width - widths[10]
			settings := ansi.Strip(ansi.Cut(view, settingsStart, width))
			if !strings.HasPrefix(settings, "╞") || !strings.HasSuffix(settings, "╡") || !strings.Contains(settings, "Settings") {
				t.Errorf("width %d Settings is not visibly boxed: %q", width, settings)
			}
			for _, x := range []int{settingsStart, width - 1} {
				if got := HandleClickAt(x, 0, width, items); got != items[10].RawKey {
					t.Errorf("width %d Settings click at %d routed to %q", width, x, got)
				}
			}
		} else if got := HandleClickAt(width-1, 0, width, items); got != items[9].RawKey {
			t.Errorf("width %d trailing F10 click routed to %q", width, got)
		}
		if got := strings.Count(view, "╞"); got != wantFrames {
			t.Errorf("width %d has %d boxes, want %d", width, got, wantFrames)
		}
	}
}

func TestVeryNarrowFallbackKeepsFKeysBoxedAndClickable(t *testing.T) {
	items := DefaultItems(config.Default())
	for _, width := range []int{24, 40} {
		view := strings.Split(ansi.Strip(View(theme.Default(), width, items)), "\n")
		rows := layoutRows(width, items)
		if len(view) != len(rows) || Height(width, items) != len(rows) {
			t.Fatalf("width %d layout has %d rendered rows and height %d; want %d", width, len(view), Height(width, items), len(rows))
		}
		for rowIndex, row := range rows {
			if got := ansi.StringWidth(view[rowIndex]); got != width {
				t.Errorf("width %d row %d spans %d cells", width, rowIndex, got)
			}
			if got := strings.Count(view[rowIndex], "╞"); got != len(row.indices) {
				t.Errorf("width %d row %d has %d boxes, want %d", width, rowIndex, got, len(row.indices))
			}
			position := 0
			for j, index := range row.indices {
				segment := ansi.Cut(view[rowIndex], position, position+row.widths[j])
				inside := strings.TrimSuffix(strings.TrimPrefix(segment, "╞"), "╡")
				if !strings.HasPrefix(segment, "╞") || !strings.HasSuffix(segment, "╡") || !strings.HasPrefix(inside, items[index].Key) {
					t.Errorf("width %d row %d item %s is not fully boxed: %q", width, rowIndex, items[index].Key, segment)
				}
				for _, x := range []int{position, position + row.widths[j] - 1} {
					if got := HandleClickAt(x, rowIndex, width, items); got != items[index].RawKey {
						t.Errorf("width %d row %d click at %d routed to %q; want %q", width, rowIndex, x, got, items[index].RawKey)
					}
				}
				position += row.widths[j]
			}
		}
	}
}
