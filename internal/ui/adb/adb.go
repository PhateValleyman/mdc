// Package adb provides an overlay for selecting and connecting Android devices.
package adb

import (
	"fmt"
	"strings"
	"unicode/utf8"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"github.com/kooler/MiddayCommander/internal/ui/overlay"
	uitext "github.com/kooler/MiddayCommander/internal/ui/text"
	"github.com/kooler/MiddayCommander/internal/ui/theme"
	"github.com/kooler/MiddayCommander/internal/vfs/adbfs"
)

type DismissMsg struct{}
type SelectMsg struct{ Device adbfs.Device }

type DevicesLoadedMsg struct {
	Devices []adbfs.Device
	Err     error
}

type ConnectDoneMsg struct {
	Target  string
	Devices []adbfs.Device
	Err     error
}

func LoadDevicesCmd() tea.Cmd {
	return func() tea.Msg {
		devices, err := adbfs.ListDevices()
		return DevicesLoadedMsg{Devices: devices, Err: err}
	}
}

func ConnectDeviceCmd(address string) tea.Cmd {
	return func() tea.Msg {
		target, err := adbfs.Connect(address)
		devices, listErr := adbfs.ListDevices()
		if err == nil {
			err = listErr
		}
		if target == "" {
			target = address
		}
		return ConnectDoneMsg{Target: target, Devices: devices, Err: err}
	}
}

// Model is a searchable list of adb targets with an optional TCP/IP connect form.
type Model struct {
	devices        []adbfs.Device
	items          []adbfs.Device
	cursor         int
	offset         int
	filter         string
	filtering      bool
	loading        bool
	connectMode    bool
	connectLoading bool
	connectTarget  string
	connectNotice  string
	err            error
	width          int
	height         int
}

func New(width, height int) Model {
	return Model{loading: true, width: width, height: height}
}

func (m *Model) HandleDevices(msg DevicesLoadedMsg) {
	m.loading = false
	m.err = msg.Err
	if msg.Err == nil {
		m.devices = append([]adbfs.Device(nil), msg.Devices...)
	}
	m.refilter()
}

func (m *Model) HandleConnect(msg ConnectDoneMsg) {
	m.connectMode = false
	m.connectLoading = false
	m.connectTarget = ""
	m.err = msg.Err
	m.devices = append([]adbfs.Device(nil), msg.Devices...)
	if msg.Err == nil {
		m.connectNotice = "Connected to " + msg.Target + "; select it and press Enter."
	}
	m.refilter()
	if msg.Err == nil {
		for i, device := range m.items {
			if device.Serial == msg.Target {
				m.cursor = i
				break
			}
		}
	}
	m.clampOffset()
}

func (m *Model) SetSize(width, height int) {
	m.width, m.height = width, height
	m.clampOffset()
}

func (m Model) Update(msg tea.KeyMsg) (Model, tea.Cmd) {
	if m.connectMode {
		return m.updateConnect(msg)
	}
	if m.filtering {
		return m.updateFilter(msg)
	}
	switch msg.String() {
	case "esc":
		return m, func() tea.Msg { return DismissMsg{} }
	case "enter":
		if !m.loading && m.cursor >= 0 && m.cursor < len(m.items) && m.items[m.cursor].State == "device" {
			device := m.items[m.cursor]
			return m, func() tea.Msg { return SelectMsg{Device: device} }
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
	case "c":
		m.connectMode = true
		m.connectTarget = ""
		m.connectNotice = ""
		m.err = nil
	case "r":
		m.loading = true
		m.err = nil
		m.connectNotice = ""
		return m, LoadDevicesCmd()
	default:
		if len(msg.String()) == 1 && msg.String()[0] >= '0' && msg.String()[0] <= '9' {
			idx := int(msg.String()[0] - '0')
			if !m.loading && idx < len(m.items) && m.items[idx].State == "device" {
				device := m.items[idx]
				return m, func() tea.Msg { return SelectMsg{Device: device} }
			}
		}
	}
	return m, nil
}

func (m Model) updateConnect(msg tea.KeyMsg) (Model, tea.Cmd) {
	if m.connectLoading {
		return m, nil
	}
	switch msg.String() {
	case "esc":
		m.connectMode = false
		m.connectTarget = ""
		m.err = nil
	case "enter":
		if strings.TrimSpace(m.connectTarget) == "" {
			m.err = fmt.Errorf("enter a host name or IP address")
			return m, nil
		}
		m.connectLoading = true
		m.err = nil
		return m, ConnectDeviceCmd(m.connectTarget)
	case "backspace":
		m.connectTarget = dropLastRune(m.connectTarget)
	case "ctrl+u":
		m.connectTarget = ""
	default:
		if !msg.Alt && msg.Type == tea.KeyRunes {
			m.connectTarget += string(msg.Runes)
		} else if msg.Type == tea.KeySpace {
			m.connectTarget += " "
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
		if m.cursor >= 0 && m.cursor < len(m.items) && m.items[m.cursor].State == "device" {
			device := m.items[m.cursor]
			return m, func() tea.Msg { return SelectMsg{Device: device} }
		}
	case "backspace":
		if len(m.filter) > 0 {
			m.filter = dropLastRune(m.filter)
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
	for _, d := range m.devices {
		label := strings.ToLower(d.Serial + " " + d.Model + " " + d.Product + " " + d.State)
		if query == "" || strings.Contains(label, query) {
			m.items = append(m.items, d)
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
	h := max(9, len(m.items)+5)
	if m.connectMode {
		h = 11
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
	if h-5 < 1 {
		return 1
	}
	return h - 5
}

func (m Model) View(th theme.Theme, screenWidth, screenHeight int) string {
	boxW, boxH := m.BoxSize(screenWidth, screenHeight)
	innerW := max(1, boxW-2)
	styles := th.OverlayStyles()
	base := styles.Base
	dim := styles.Dim
	key := styles.Prompt
	selected := styles.Selected

	var lines []string
	if m.connectMode {
		lines = append(lines, dim.Render(" Connect an ADB TCP/IP device using host[:port]. Port defaults to 5555."))
		lines = append(lines, key.Render(" Device: ")+base.Render(uitext.SafeTerminalText(m.connectTarget)+"_"))
		if m.connectLoading {
			lines = append(lines, dim.Render(" Connecting and refreshing devices…"))
		} else if m.err != nil {
			lines = append(lines, dim.Render(" "+uitext.SafeTerminalText(m.err.Error())))
		}
	} else {
		if m.filtering {
			lines = append(lines, key.Render(" Filter: ")+base.Render(uitext.SafeTerminalText(m.filter)+"_"))
		}
		if m.connectNotice != "" {
			lines = append(lines, dim.Render(" "+uitext.SafeTerminalText(m.connectNotice)))
		}
		if m.loading {
			lines = append(lines, dim.Render(" Scanning adb devices…"))
		} else {
			if m.err != nil {
				lines = append(lines, dim.Render(" "+uitext.SafeTerminalText(m.err.Error())))
			}
			if len(m.items) == 0 && m.err == nil {
				lines = append(lines, dim.Render(" No adb devices found. Press 'c' to connect one or 'r' to refresh."))
			} else {
				visible := m.resultHeight() - len(lines)
				if visible < 1 {
					visible = 1
				}
				end := min(m.offset+visible, len(m.items))
				for i := m.offset; i < end; i++ {
					d := m.items[i]
					prefix := "  "
					if i < 10 {
						prefix = fmt.Sprintf("%d ", i)
					}
					label := d.Serial
					if d.Model != "" {
						label += "  " + d.Model
					}
					if d.Product != "" && d.Model == "" {
						label += "  " + d.Product
					}
					if d.State != "device" {
						label += "  [" + d.State + "]"
					}
					label = uitext.SafeTerminalText(label)
					text := prefix + label
					if i == m.cursor {
						lines = append(lines, overlay.FrameRow(selected.Render(pad(text, innerW-2)), innerW, styles.Highlight))
					} else {
						row := key.Render(prefix) + base.Render(pad(label, innerW-2-len(prefix)))
						lines = append(lines, overlay.FrameRow(row, innerW, styles.Border))
					}
				}
			}
		}
	}

	footer := key.Render(" Enter") + dim.Render(":Browse  ") + key.Render("c") + dim.Render(":Connect  ") + key.Render("f") + dim.Render(":Filter  ") + key.Render("r") + dim.Render(":Refresh  ") + key.Render("Esc") + dim.Render(":Close")
	if m.filtering {
		footer = key.Render(" Enter") + dim.Render(":Browse  ") + key.Render("Esc") + dim.Render(":Clear filter")
	}
	if m.connectMode {
		footer = key.Render(" Enter") + dim.Render(":Connect  ") + key.Render("Esc") + dim.Render(":Cancel")
		if m.connectLoading {
			footer = dim.Render(" Please wait…")
		}
	}
	return overlay.RenderBox("ADB devices", lines, footer, boxW, boxH, styles)
}

func dropLastRune(value string) string {
	if value == "" {
		return ""
	}
	_, size := utf8.DecodeLastRuneInString(value)
	return value[:len(value)-size]
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
