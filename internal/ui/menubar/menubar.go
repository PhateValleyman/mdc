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
		itemFromCfg(cfg.Keys.Bookmarks, "Bookmarks"),
		itemFromCfg(cfg.Keys.View, "View"),
		itemFromCfg(cfg.Keys.Edit, "Edit"),
		itemFromCfg(cfg.Keys.Copy, "Copy"),
		itemFromCfg(cfg.Keys.Move, "Move"),
		itemFromCfg(cfg.Keys.Mkdir, "Mkdir"),
		itemFromCfg(cfg.Keys.Delete, "Delete"),
		itemFromCfg(cfg.Keys.FuzzyFind, "Fuzzy"),
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
		{cfg.Keys.Bookmarks, "Bookmarks"},
		{cfg.Keys.View, "View"},
		{cfg.Keys.Edit, "Edit"},
		{cfg.Keys.Copy, "Copy"},
		{cfg.Keys.Move, "Move"},
		{cfg.Keys.Mkdir, "Mkdir"},
		{cfg.Keys.Delete, "Delete"},
		{cfg.Keys.FuzzyFind, "Fuzzy"},
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

// HandleClickAt resolves a click within the menu block, where row is zero-based
// from the top. It falls back to the legacy one-row hit regions when applicable.
func HandleClickAt(x, row, width int, items []Item) string {
	rows := layoutRows(width, items)
	if row < 0 || row >= len(rows) {
		return ""
	}
	if len(rows) == 1 {
		return HandleClick(x, width, items)
	}
	position := 0
	for j, index := range rows[row].indices {
		itemWidth := rows[row].widths[j]
		if x >= position && x < position+itemWidth {
			return items[index].RawKey
		}
		position += itemWidth
	}
	return ""
}

// Height returns the number of terminal rows occupied by the menu.
func Height(width int, items []Item) int {
	rows := layoutRows(width, items)
	if len(rows) == 0 {
		return 1
	}
	return len(rows)
}

// View renders the F-key hints bar at the bottom, evenly spaced across width.
func View(th theme.Theme, width int, items []Item) string {
	n := len(items)
	if n == 0 {
		return th.MenuBar.Render(strings.Repeat(" ", width))
	}

	rows := layoutRows(width, items)
	widths := itemWidths(width, items)
	wideLabels := minRegularItemWidth(widths, items) >= 10
	viewRows := make([]string, 0, len(rows))
	for _, row := range rows {
		var b strings.Builder
		for j, index := range row.indices {
			itm := items[index]
			w := row.widths[j]
			contentWidth := w - 2
			keyWidth := lipgloss.Width(itm.Key)
			labelWidth := contentWidth - keyWidth
			if labelWidth < 0 {
				labelWidth = 0
			}
			label := menuActionLabel(itm.Label, labelWidth, wideLabels)
			content := th.FKeyHint.Render(itm.Key) + th.FKeyLabel.Render(overlay.PadOrTrunc(label, labelWidth))
			b.WriteString(overlay.FrameRow(content, w, th.FKeyHint))
		}
		viewRows = append(viewRows, b.String())
	}
	return strings.Join(viewRows, "\n")
}

const pinnedMenuWidth = 14

type labelAliases struct {
	short   string
	compact string
	wide    string
}

var actionLabelAliases = map[string]labelAliases{
	"Help":      {short: "H", compact: "Hp", wide: "Help"},
	"Bookmarks": {short: "B", compact: "Bk", wide: "Bkmk"},
	"View":      {short: "V", compact: "Vw", wide: "View"},
	"Edit":      {short: "E", compact: "Ed", wide: "Edit"},
	"Copy":      {short: "C", compact: "Cp", wide: "Copy"},
	"Move":      {short: "M", compact: "Mv", wide: "Move"},
	"Mkdir":     {short: "K", compact: "Mk", wide: "Mkdir"},
	"Delete":    {short: "D", compact: "Dl", wide: "Del"},
	"Fuzzy":     {short: "F", compact: "FZ", wide: "Fuzzy"},
	"Quit":      {short: "Q", compact: "Qt", wide: "Quit"},
	"Rename":    {short: "R", compact: "Rn", wide: "Rename"},
	"CpPath":    {short: "P", compact: "CP", wide: "CpPath"},
	"SSH":       {short: "S", compact: "SH", wide: "SSH"},
	"GoTo":      {short: "G", compact: "GT", wide: "GoTo"},
	"Panel":     {short: "P", compact: "Pn", wide: "Panel"},
	"Swap":      {short: "S", compact: "Sw", wide: "Swap"},
	"Theme":     {short: "T", compact: "Th", wide: "Theme"},
}

func menuActionLabel(label string, width int, wide bool) string {
	if label == "" || width <= 0 {
		return ""
	}
	aliases, ok := actionLabelAliases[label]
	if !ok {
		aliases = labelAliases{short: label, compact: label, wide: label}
	}
	candidates := []string{aliases.compact, aliases.short}
	if wide {
		candidates = []string{aliases.wide, aliases.compact, aliases.short}
	}
	for _, candidate := range candidates {
		candidateWidth := lipgloss.Width(candidate)
		if candidate == "" || candidateWidth > width {
			continue
		}
		if candidateWidth+1 <= width {
			return " " + candidate
		}
		return candidate
	}
	return ""
}

func minRegularItemWidth(widths []int, items []Item) int {
	minimum := 0
	for i, item := range items {
		if item.Pinned || widths[i] <= 0 {
			continue
		}
		if minimum == 0 || widths[i] < minimum {
			minimum = widths[i]
		}
	}
	return minimum
}

type menuRow struct {
	indices []int
	widths  []int
}

func layoutRows(width int, items []Item) []menuRow {
	if width <= 0 || len(items) == 0 {
		return []menuRow{{}}
	}

	minimumTotal := 0
	visible := make([]int, 0, len(items))
	for i, item := range items {
		if item.Pinned {
			continue
		}
		visible = append(visible, i)
		minimumTotal += lipgloss.Width(item.Key) + 2
	}
	widths := itemWidths(width, items)
	if width >= minimumTotal {
		row := menuRow{}
		for i, w := range widths {
			if w > 0 {
				row.indices = append(row.indices, i)
				row.widths = append(row.widths, w)
			}
		}
		return []menuRow{row}
	}

	// Below the one-row physical minimum, wrap only as many rows as needed to
	// keep every key token and both frame edges intact. Widths 41+ stay one row.
	for rowCount := 2; rowCount <= len(visible); rowCount++ {
		base, remainder := len(visible)/rowCount, len(visible)%rowCount
		rows := make([]menuRow, 0, rowCount)
		cursor := 0
		valid := true
		for r := 0; r < rowCount; r++ {
			count := base
			if r < remainder {
				count++
			}
			row := menuRow{}
			used := 0
			for j := 0; j < count; j++ {
				index := visible[cursor]
				cursor++
				slotWidth := lipgloss.Width(items[index].Key) + 2
				row.indices = append(row.indices, index)
				row.widths = append(row.widths, slotWidth)
				used += slotWidth
			}
			if used > width {
				valid = false
				break
			}
			if len(row.widths) > 0 {
				row.widths[len(row.widths)-1] += width - used
			}
			rows = append(rows, row)
		}
		if valid {
			return rows
		}
	}

	// Terminals narrower than a single boxed F10 token are physically unable to
	// display the bar; retain the complete token in its own minimally sized row.
	rows := make([]menuRow, 0, len(visible))
	for _, index := range visible {
		slotWidth := lipgloss.Width(items[index].Key) + 2
		if slotWidth < width {
			slotWidth = width
		}
		rows = append(rows, menuRow{indices: []int{index}, widths: []int{slotWidth}})
	}
	return rows
}

func minFramedChoiceWidth(items []Item) int {
	maxKeyWidth := 0
	for _, item := range items {
		if item.Pinned {
			continue
		}
		if keyWidth := lipgloss.Width(item.Key); keyWidth > maxKeyWidth {
			maxKeyWidth = keyWidth
		}
	}
	return maxKeyWidth + 2 // preserve every key label plus both frame edges
}

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
		minSlotWidth := minFramedChoiceWidth(items[:pinned])
		if regularCount > 0 && width >= pinnedMenuWidth+regularCount*minSlotWidth {
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
