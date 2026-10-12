// Package settings provides the application settings overlay.
package settings

import (
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"github.com/kooler/MiddayCommander/internal/ui/overlay"
	uitext "github.com/kooler/MiddayCommander/internal/ui/text"
	"github.com/kooler/MiddayCommander/internal/ui/theme"
)

// DismissMsg closes the settings overlay.
type DismissMsg struct{}

// ThemeRequestedMsg asks the application to open the existing live theme picker.
type ThemeRequestedMsg struct{}

// ServersRequestedMsg opens the existing saved SSH server manager.
type ServersRequestedMsg struct{}

// RcloneRequestedMsg opens the existing rclone remote manager.
type RcloneRequestedMsg struct{}

// ShowHiddenChangedMsg reports the requested visibility for dot files.
type ShowHiddenChangedMsg struct{ Show bool }

// Model is the settings overlay.
type Model struct {
	themeName  string
	showHidden bool
	cursor     int
	offset     int
	width      int
	height     int
}

// New creates a settings overlay for the current preferences.
func New(themeName string, showHidden bool, width, height int) Model {
	if themeName == "" {
		themeName = theme.DefaultKey
	}
	return Model{themeName: themeName, showHidden: showHidden, width: width, height: height}
}

// SetSize updates the overlay's latest terminal dimensions.
func (m *Model) SetSize(width, height int) {
	m.width, m.height = width, height
	m.clampOffset()
}

// SetShowHidden updates the visible value after the application applies it.
func (m *Model) SetShowHidden(show bool) { m.showHidden = show }

// SetThemeName updates the displayed theme after a live theme selection.
func (m *Model) SetThemeName(name string) {
	if name == "" {
		name = theme.DefaultKey
	}
	m.themeName = name
}

// Update handles settings navigation and selection.
func (m Model) Update(msg tea.KeyMsg) (Model, tea.Cmd) {
	switch msg.String() {
	case "esc", "q":
		return m, func() tea.Msg { return DismissMsg{} }
	case "up", "k":
		if m.cursor > 0 {
			m.cursor--
			m.clampOffset()
		}
	case "down", "j":
		if m.cursor < 3 {
			m.cursor++
			m.clampOffset()
		}
	case "enter", " ":
		switch m.cursor {
		case 0:
			return m, func() tea.Msg { return ThemeRequestedMsg{} }
		case 1:
			show := !m.showHidden
			m.showHidden = show
			return m, func() tea.Msg { return ShowHiddenChangedMsg{Show: show} }
		case 2:
			return m, func() tea.Msg { return ServersRequestedMsg{} }
		case 3:
			return m, func() tea.Msg { return RcloneRequestedMsg{} }
		}
	}
	return m, nil
}

// BoxSize returns the desired settings box dimensions.
func (m Model) BoxSize(screenWidth, screenHeight int) (int, int) {
	w := 56
	if w > screenWidth-4 {
		w = screenWidth - 4
	}
	if w < 1 {
		w = screenWidth
	}
	h := 10
	if h > screenHeight {
		h = screenHeight
	}
	return w, h
}

func (m Model) visibleRows() (int, bool) {
	_, boxH := m.BoxSize(m.width, m.height)
	available := boxH - 3 // title, footer, and bottom border
	showHint := available > 4
	if showHint {
		available--
	}
	if available < 1 {
		available = 1
	}
	if available > 4 {
		available = 4
	}
	return available, showHint
}

func (m *Model) clampOffset() {
	visible, _ := m.visibleRows()
	if m.cursor < m.offset {
		m.offset = m.cursor
	}
	if m.cursor >= m.offset+visible {
		m.offset = m.cursor - visible + 1
	}
}

// View renders the settings overlay using the active theme's shared styles.
func (m Model) View(th theme.Theme, screenWidth, screenHeight int) string {
	boxW, boxH := m.BoxSize(screenWidth, screenHeight)
	if boxW < 2 || boxH < 4 {
		return overlay.PadOrTrunc("Settings", boxW)
	}
	innerW := boxW - 2
	styles := th.OverlayStyles()

	themeLine := " Theme         " + uitext.SafeTerminalText(m.themeName)
	showValue := "Off"
	if m.showHidden {
		showValue = "On"
	}
	showLine := " Show hidden   [" + showValue + "]"
	rows := []string{themeLine, showLine, " SSH / Servers", " rclone Remotes / Config"}
	visible, showHint := m.visibleRows()
	end := m.offset + visible
	if end > len(rows) {
		end = len(rows)
	}
	content := make([]string, 0, visible+1)
	for i := m.offset; i < end; i++ {
		row := rows[i]
		line := overlay.PadOrTrunc(row, innerW-2)
		if i == m.cursor {
			content = append(content, overlay.FrameRow(styles.Selected.Render(line), innerW, styles.Highlight))
		} else {
			content = append(content, overlay.FrameRow(styles.Base.Render(line), innerW, styles.Border))
		}
	}
	if showHint {
		content = append(content, styles.Dim.Render(overlay.PadOrTrunc(" Select a row and press Enter", innerW)))
	}

	key := styles.Prompt
	dim := styles.Dim
	fullFooter := key.Render(" ↑/↓") + dim.Render(":Navigate  ") +
		key.Render("Enter") + dim.Render(":Choose / Toggle  ") + key.Render("Esc") + dim.Render(":Close")
	compactFooter := key.Render(" ↑↓") + dim.Render(":Move ") + key.Render("Enter") + dim.Render(":Select ") + key.Render("Esc") + dim.Render(":Back")
	shortFooter := key.Render("Enter") + dim.Render(":OK ") + key.Render("Esc") + dim.Render(":Back")
	footer := fullFooter
	if innerW < lipgloss.Width(compactFooter) {
		footer = shortFooter
	} else if innerW < lipgloss.Width(fullFooter) {
		footer = compactFooter
	}
	footer = overlay.PadOrTrunc(footer, innerW)
	return overlay.RenderBox("Settings", content, footer, boxW, boxH, styles)
}
