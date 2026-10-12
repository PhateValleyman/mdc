package menubar

import (
	"strings"
	"testing"

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
		if got := ansi.StringWidth(view); got != width {
			t.Errorf("menu width at terminal width %d = %d", width, got)
		}
	}
}
