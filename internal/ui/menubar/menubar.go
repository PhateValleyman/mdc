package menubar

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/charmbracelet/lipgloss"

	"github.com/kooler/MiddayCommander/internal/config"
	"github.com/kooler/MiddayCommander/internal/ui/overlay"
	"github.com/kooler/MiddayCommander/internal/ui/theme"
)

// Item represents a single menu bar button.
type Item struct {
	Key    string // display label, e.g. "F5"
	Label  string // action label, e.g. "Copy"
	RawKey string // actual key string for matching clicks, e.g. "f5"
	Pinned bool   // fixed-width trailing menu item, independent of F-key slots
}

// DefaultItems returns the default menu bar items.
func DefaultItems(cfg config.Config) []Item {
	items := []Item{
		itemFromCfg(cfg.Keys.Help, "Help"),
		itemFromCfg(cfg.Keys.Bookmarks, "Bookm"),
		itemFromCfg(cfg.Keys.View, "View"),
		itemFromCfg(cfg.Keys.Edit, "Edit"),
		itemFromCfg(cfg.Keys.Copy, "Copy"),
		itemFromCfg(cfg.Keys.Move, "Move"),
		itemFromCfg(cfg.Keys.Mkdir, "Mkdir"),
		itemFromCfg(cfg.Keys.Delete, "Delete"),
		itemFromCfg(cfg.Keys.FuzzyFind, "FZF"),
		itemFromCfg(cfg.Keys.Quit, "Quit"),
	}
	settings := itemFromCfg(cfg.Keys.Settings, "Settings")
	settings.Pinned = true
	return append(items, settings)
}

// ShiftItems returns menu bar items for the Shift+F1..F10 state.
// Only positions with a shift+F binding get a label; the rest are blank.
func ShiftItems(cfg config.Config) []Item {
	items := make([]Item, 10)
	// Always show F1-F10 key hints; label is empty unless a shift binding exists.
	for i := range items {
		items[i] = Item{Key: fmt.Sprintf("F%d", i+1)}
	}

	type entry struct {
		keys  config.StringOrList
		label string
	}
	bindings := []entry{
		{cfg.Keys.Help, "Help"},
		{cfg.Keys.Bookmarks, "Bookm"},
		{cfg.Keys.View, "View"},
		{cfg.Keys.Edit, "Edit"},
		{cfg.Keys.Copy, "Copy"},
		{cfg.Keys.Move, "Move"},
		{cfg.Keys.Mkdir, "Mkdir"},
		{cfg.Keys.Delete, "Delete"},
		{cfg.Keys.FuzzyFind, "FZF"},
		{cfg.Keys.Quit, "Quit"},
		{cfg.Keys.Rename, "Rename"},
		{cfg.Keys.CopyPath, "CpPath"},
		{cfg.Keys.Servers, "SSH"},
		{cfg.Keys.GoTo, "GoTo"},
		{cfg.Keys.TogglePanel, "Panel"},
		{cfg.Keys.SwapPanels, "Swap"},
		{cfg.Keys.ThemePicker, "Theme"},
	}

	for _, b := range bindings {
		for _, k := range b.keys {
			if pos := shiftFKeyPos(k); pos >= 0 {
				items[pos].Label = b.label
				items[pos].RawKey = k
			}
		}
	}
	settings := itemFromCfg(cfg.Keys.Settings, "Settings")
	settings.Pinned = true
	return append(items, settings)
}

// shiftFKeyPos returns the 0-based menu position for a shift F-key string,
// or -1 if the key is not a shift F-key. "f13" -> 0, "f18" -> 5, etc.
func shiftFKeyPos(k string) int {
	if len(k) < 3 || k[0] != 'f' {
		return -1
	}
	n, err := strconv.Atoi(k[1:])
	if err != nil || n < 13 || n > 20 {
		return -1
	}
	return n - 13
}

func itemFromCfg(keys config.StringOrList, label string) Item {
	if len(keys) == 0 {
		return Item{Label: label}
	}
	raw := keys[0]
	return Item{
		Key:    formatKeyDisplay(raw),
		Label:  label,
		RawKey: raw,
	}
}

// formatKeyDisplay turns "f5" into "F5", "ctrl+g" into "C-g", etc.
// F13-F20 (BubbleTea's representation of Shift+F1..F8) display as "S-F1".."S-F8".
func formatKeyDisplay(k string) string {
	if len(k) > 1 && k[0] == 'f' && k[1] >= '0' && k[1] <= '9' {
		if n, err := strconv.Atoi(k[1:]); err == nil && n >= 13 && n <= 20 {
			return fmt.Sprintf("S-F%d", n-12)
		}
		return "F" + k[1:]
	}
	if strings.HasPrefix(k, "ctrl+") {
		return "C-" + k[5:]
	}
	if strings.HasPrefix(k, "alt+") {
		return "A-" + k[4:]
	}
	if strings.HasPrefix(k, "shift+") {
		return "S-" + k[6:]
	}
	return k
}

// HandleClick checks if a mouse click at column x hits a menu item.
// Returns the raw key string, or "" if no hit.
func HandleClick(x, width int, items []Item) string {
	n := len(items)
	if n == 0 {
		return ""
	}
	// Very narrow terminals can be smaller than the number of menu slots.
	// Avoid a zero-width division and treat any visible click as the first slot.
	if width <= 0 {
		return ""
	}
	widths := itemWidths(width, items)
	if width < n && pinnedIndex(items) < 0 {
		idx := x
		if idx >= n {
			idx = n - 1
		}
		return items[idx].RawKey
	}
	position := 0
	for i, itemWidth := range widths {
		if itemWidth > 0 && x >= position && x < position+itemWidth {
			return items[i].RawKey
		}
		position += itemWidth
	}
	for i := n - 1; i >= 0; i-- {
		if widths[i] > 0 {
			return items[i].RawKey
		}
	}
	return ""
}

// View renders the F-key hints bar at the bottom, evenly spaced across width.
func View(th theme.Theme, width int, items []Item) string {
	n := len(items)
	if n == 0 {
		return th.MenuBar.Render(strings.Repeat(" ", width))
	}

	widths := itemWidths(width, items)

	var b strings.Builder
	for i, itm := range items {
		w := widths[i]
		if w <= 0 {
			continue
		}

		keyWidth := lipgloss.Width(itm.Key)
		if keyWidth > w {
			keyWidth = w
		}
		keyStr := th.FKeyHint.Render(overlay.PadOrTrunc(itm.Key, keyWidth))
		labelWidth := w - keyWidth
		labelStr := th.FKeyLabel.Render(overlay.PadOrTrunc(itm.Label, labelWidth))

		b.WriteString(keyStr)
		b.WriteString(labelStr)
	}

	return b.String()
}

const pinnedMenuWidth = 14

func pinnedIndex(items []Item) int {
	for i := len(items) - 1; i >= 0; i-- {
		if items[i].Pinned {
			return i
		}
	}
	return -1
}

func itemWidths(width int, items []Item) []int {
	widths := make([]int, len(items))
	if width <= 0 || len(items) == 0 {
		return widths
	}

	pinned := pinnedIndex(items)
	regularCount := len(items)
	regularWidth := width
	if pinned >= 0 {
		regularCount--
		if regularCount > 0 && width >= pinnedMenuWidth+regularCount*3 {
			regularWidth = width - pinnedMenuWidth
			widths[pinned] = pinnedMenuWidth
		}
	}
	if regularCount <= 0 {
		return widths
	}

	colWidth := regularWidth / regularCount
	if colWidth == 0 {
		for i := 0; i < regularCount && i < regularWidth; i++ {
			widths[i] = 1
		}
		return widths
	}

	for i := 0; i < len(items); i++ {
		if i == pinned {
			continue
		}
		w := colWidth
		if i == regularCount-1 {
			w = regularWidth - colWidth*(regularCount-1)
		}
		widths[i] = w
	}
	return widths
}
