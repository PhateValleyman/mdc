// Package rclone provides the remote picker overlay for rclone-backed filesystems.
package rclone

import (
	"fmt"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"github.com/kooler/MiddayCommander/internal/ui/overlay"
	"github.com/kooler/MiddayCommander/internal/ui/theme"
	"github.com/kooler/MiddayCommander/internal/vfs/rclonefs"
)

type DismissMsg struct{}

type SelectMsg struct{ Remote string }

type CreateConfigRequestedMsg struct{}

type EditConfigRequestedMsg struct{ Remote string }

type RemotesLoadedMsg struct {
	Remotes []string
	Err     error
}

// LoadRemotesCmd asks rclone for remote names only. Configuration values are
// never read into this process or included in the returned message.
func LoadRemotesCmd() tea.Cmd {
	return func() tea.Msg {
		remotes, err := rclonefs.ListRemotes()
		return RemotesLoadedMsg{Remotes: remotes, Err: err}
	}
}

// Model is a searchable list of configured rclone remotes.
type Model struct {
	remotes   []string
	items     []string
	cursor    int
	offset    int
	filter    string
	filtering bool
	loading   bool
	err       error
	width     int
	height    int
}

func New(width, height int) Model {
	return Model{loading: true, width: width, height: height}
}

func (m *Model) HandleRemotes(msg RemotesLoadedMsg) {
	m.loading = false
	m.err = msg.Err
	if msg.Err == nil {
		m.remotes = append([]string(nil), msg.Remotes...)
	}
	m.refilter()
}

func (m *Model) BeginRefresh() {
	m.loading = true
	m.err = nil
}

func (m *Model) SetSize(width, height int) {
	m.width = width
	m.height = height
	m.clampOffset()
}

func (m Model) Update(msg tea.KeyMsg) (Model, tea.Cmd) {
	if m.filtering {
		return m.updateFilter(msg)
	}
	switch msg.String() {
	case "esc":
		return m, func() tea.Msg { return DismissMsg{} }
	case "enter":
		if !m.loading && m.cursor >= 0 && m.cursor < len(m.items) {
			remote := m.items[m.cursor]
			return m, func() tea.Msg { return SelectMsg{Remote: remote} }
		}
	case "up", "k":
		if m.cursor > 0 {
			m.cursor--
			m.clampOffset()
		}
	case "down", "j":
		if m.cursor < len(m.items)-1 {
			m.cursor++
			m.clampOffset()
		}
	case "f":
		m.filtering = true
		m.filter = ""
		m.refilter()
	case "a":
		return m, func() tea.Msg { return CreateConfigRequestedMsg{} }
	case "e":
		if !m.loading && m.cursor >= 0 && m.cursor < len(m.items) {
			remote := m.items[m.cursor]
			return m, func() tea.Msg { return EditConfigRequestedMsg{Remote: remote} }
		}
	default:
		if len(msg.String()) == 1 && msg.String()[0] >= '0' && msg.String()[0] <= '9' {
			idx := int(msg.String()[0] - '0')
			if !m.loading && idx < len(m.items) {
				remote := m.items[idx]
				return m, func() tea.Msg { return SelectMsg{Remote: remote} }
			}
		}
	}
	return m, nil
}

func (m Model) updateFilter(msg tea.KeyMsg) (Model, tea.Cmd) {
	switch msg.String() {
	case "esc":
		m.filtering = false
		m.filter = ""
		m.refilter()
	case "enter":
		m.filtering = false
		if m.cursor >= 0 && m.cursor < len(m.items) {
			remote := m.items[m.cursor]
			return m, func() tea.Msg { return SelectMsg{Remote: remote} }
		}
	case "backspace":
		if len(m.filter) > 0 {
			m.filter = m.filter[:len(m.filter)-1]
			m.refilter()
		} else {
			m.filtering = false
		}
	case "up":
		if m.cursor > 0 {
			m.cursor--
			m.clampOffset()
		}
	case "down":
		if m.cursor < len(m.items)-1 {
			m.cursor++
			m.clampOffset()
		}
	default:
		if !msg.Alt && msg.Type == tea.KeyRunes {
			m.filter += string(msg.Runes)
			m.refilter()
		} else if msg.Type == tea.KeySpace {
			m.filter += " "
			m.refilter()
		}
	}
	return m, nil
}

func (m *Model) refilter() {
	m.items = nil
	query := strings.ToLower(m.filter)
	for _, remote := range m.remotes {
		if query == "" || strings.Contains(strings.ToLower(remote), query) {
			m.items = append(m.items, remote)
		}
	}
	if m.cursor >= len(m.items) {
		m.cursor = max(0, len(m.items)-1)
	}
	m.offset = 0
}

func (m *Model) clampOffset() {
	visible := m.resultHeight()
	if m.cursor < m.offset {
		m.offset = m.cursor
	}
	if m.cursor >= m.offset+visible {
		m.offset = m.cursor - visible + 1
	}
}

func (m Model) BoxSize(screenWidth, screenHeight int) (int, int) {
	w := screenWidth * 2 / 3
	if w < 48 {
		w = min(48, screenWidth)
	}
	h := len(m.items) + 5
	if h < 9 {
		h = 9
	}
	maxH := screenHeight * 3 / 4
	if maxH < 1 {
		maxH = screenHeight
	}
	if h > maxH {
		h = maxH
	}
	return w, h
}

func (m Model) resultHeight() int {
	_, h := m.BoxSize(m.width, m.height)
	visible := h - 5
	if visible < 1 {
		return 1
	}
	return visible
}

func (m Model) View(_ theme.Theme, screenWidth, screenHeight int) string {
	boxW, boxH := m.BoxSize(screenWidth, screenHeight)
	innerW := boxW - 2
	bg := lipgloss.Color("#1e1e2e")
	fg := lipgloss.Color("#cdd6f4")
	subtle := lipgloss.Color("#a6adc8")
	accent := lipgloss.Color("#89b4fa")
	highlight := lipgloss.Color("#f9e2af")
	cursorBg := lipgloss.Color("#45475a")
	base := lipgloss.NewStyle().Background(bg).Foreground(fg)
	dim := lipgloss.NewStyle().Background(bg).Foreground(subtle)
	key := lipgloss.NewStyle().Background(bg).Foreground(accent).Bold(true)
	selected := lipgloss.NewStyle().Background(cursorBg).Foreground(fg)

	var lines []string
	if m.filtering {
		lines = append(lines, key.Render(" Filter: ")+base.Render(m.filter+"_"))
	}
	if m.loading {
		lines = append(lines, dim.Render(" Loading rclone remotes…"))
	} else if m.err != nil {
		lines = append(lines, dim.Render(" "+m.err.Error()))
	} else if len(m.items) == 0 {
		lines = append(lines, dim.Render(" No remotes configured. Press 'a' to add one in MDC."))
	} else {
		visible := m.resultHeight()
		end := min(m.offset+visible, len(m.items))
		for i := m.offset; i < end; i++ {
			prefix := "  "
			if i < 10 {
				prefix = fmt.Sprintf("%d ", i)
			}
			text := prefix + m.items[i]
			if i == m.cursor {
				lines = append(lines, selected.Render(pad(text, innerW)))
			} else {
				lines = append(lines, key.Render(prefix)+base.Render(pad(m.items[i], innerW-len(prefix))))
			}
		}
	}

	footer := key.Render(" Enter") + dim.Render(":Open  ") + key.Render("a") + dim.Render(":Add  ") + key.Render("e") + dim.Render(":Edit  ") + key.Render("f") + dim.Render(":Filter  ") + key.Render("Esc") + dim.Render(":Close")
	if m.filtering {
		footer = key.Render(" Enter") + dim.Render(":Open  ") + key.Render("Esc") + dim.Render(":Clear filter")
	}
	return overlay.RenderBox("rclone remotes", lines, footer, boxW, boxH, accent, bg, highlight)
}

func pad(s string, width int) string {
	if width <= 0 {
		return ""
	}
	if lipgloss.Width(s) > width {
		return lipgloss.NewStyle().MaxWidth(width).Render(s)
	}
	return s + strings.Repeat(" ", width-lipgloss.Width(s))
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

func max(a, b int) int {
	if a > b {
		return a
	}
	return b
}
