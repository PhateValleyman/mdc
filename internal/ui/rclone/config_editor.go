package rclone

import (
	"errors"
	"fmt"
	"strings"
	"unicode/utf8"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"github.com/kooler/MiddayCommander/internal/ui/overlay"
	uitext "github.com/kooler/MiddayCommander/internal/ui/text"
	"github.com/kooler/MiddayCommander/internal/ui/theme"
	"github.com/kooler/MiddayCommander/internal/vfs/rclonefs"
)

type ConfigProvidersLoadedMsg struct {
	Providers []rclonefs.Provider
	Err       error
}

type ConfigQuestionMsg struct {
	Remote   string
	Provider string
	Creating bool
	Created  bool
	Question rclonefs.ConfigQuestion
	Done     bool
	Err      error
}

type ConfigCancelMsg struct {
	Remote       string
	DeleteRemote bool
}

type ConfigEditorDismissMsg struct{}

type ConfigCancelledMsg struct{ Err error }

func LoadConfigProvidersCmd() tea.Cmd {
	return func() tea.Msg {
		providers, err := rclonefs.ListProviders()
		return ConfigProvidersLoadedMsg{Providers: providers, Err: err}
	}
}

func StartConfigUpdateCmd(remote string) tea.Cmd {
	return func() tea.Msg {
		question, err := rclonefs.StartConfigUpdate(remote)
		return ConfigQuestionMsg{Remote: remote, Question: question, Done: err == nil && question.State == "" && question.Error == "", Err: err}
	}
}

func StartConfigCreateCmd(remote, provider string) tea.Cmd {
	return func() tea.Msg {
		question, err := rclonefs.StartConfigCreate(remote, provider)
		return ConfigQuestionMsg{Remote: remote, Provider: provider, Creating: true, Created: err == nil, Question: question, Done: err == nil && question.State == "" && question.Error == "", Err: err}
	}
}

func RejectConfigAnswerCmd(remote string) tea.Cmd {
	return func() tea.Msg {
		return ConfigQuestionMsg{
			Remote: remote,
			Err:    errors.New("configuration answers must be entered through native `rclone config`"),
		}
	}
}

func CancelConfigCmd(remote string, deleteRemote bool) tea.Cmd {
	return func() tea.Msg {
		var err error
		if deleteRemote && remote != "" {
			err = rclonefs.DeleteConfig(remote)
		}
		return ConfigCancelledMsg{Err: err}
	}
}

const (
	configModeProviders = iota
	configModeName
	configModeLoading
	configModeQuestion
)

// ConfigEditorModel is a legacy renderer retained for configuration-option
// display tests. The root app does not mount it; real create/edit sessions run
// entirely through rclone's native terminal wizard so answers never enter argv.
type ConfigEditorModel struct {
	mode          int
	providers     []rclonefs.Provider
	items         []rclonefs.Provider
	cursor        int
	offset        int
	filter        string
	filtering     bool
	loading       bool
	remote        string
	provider      string
	creating      bool
	remoteCreated bool
	state         string
	question      rclonefs.ConfigQuestion
	answer        string
	err           string
	width         int
	height        int
}

func NewConfigCreate(width, height int) ConfigEditorModel {
	return ConfigEditorModel{mode: configModeProviders, creating: true, loading: true, width: width, height: height}
}

func NewConfigEdit(remote string, width, height int) ConfigEditorModel {
	return ConfigEditorModel{mode: configModeLoading, remote: remote, width: width, height: height}
}

func (m *ConfigEditorModel) HandleProviders(msg ConfigProvidersLoadedMsg) {
	m.loading = false
	if msg.Err != nil {
		m.err = msg.Err.Error()
		return
	}
	m.providers = append([]rclonefs.Provider(nil), msg.Providers...)
	m.refilter()
}

// HandleQuestion applies the next question or a completed configuration result.
// It returns true only when the entire create/update operation succeeded.
func (m *ConfigEditorModel) HandleQuestion(msg ConfigQuestionMsg) bool {
	m.remote = msg.Remote
	m.provider = msg.Provider
	m.creating = msg.Creating
	if msg.Created {
		m.remoteCreated = true
	}
	if msg.Err != nil {
		m.err = msg.Err.Error()
		m.question = rclonefs.ConfigQuestion{}
		m.mode = configModeQuestion
		return false
	}
	m.question = msg.Question
	m.state = msg.Question.State
	m.err = msg.Question.Error
	if m.question.Option != nil && (m.question.Option.IsPassword || m.question.Option.Sensitive) && m.err != "" {
		m.err = "The password was rejected; check the value and try again."
	}
	if msg.Done {
		return true
	}
	if msg.Question.Option == nil {
		if m.err == "" {
			m.err = "rclone returned no configuration question"
		}
		m.mode = configModeQuestion
		return false
	}
	m.answer = initialAnswer(*msg.Question.Option)
	m.mode = configModeQuestion
	return false
}

func (m *ConfigEditorModel) SetSize(width, height int) { m.width, m.height = width, height }

func (m ConfigEditorModel) Update(msg tea.KeyMsg) (ConfigEditorModel, tea.Cmd) {
	switch m.mode {
	case configModeProviders:
		return m.updateProviders(msg)
	case configModeName:
		return m.updateName(msg)
	case configModeQuestion:
		return m.updateQuestion(msg)
	default:
		return m, nil
	}
}

func (m ConfigEditorModel) updateProviders(msg tea.KeyMsg) (ConfigEditorModel, tea.Cmd) {
	if m.filtering {
		return m.updateProviderFilter(msg)
	}
	switch msg.String() {
	case "esc":
		return m, func() tea.Msg { return ConfigEditorDismissMsg{} }
	case "enter":
		if m.cursor >= 0 && m.cursor < len(m.items) {
			m.provider = m.items[m.cursor].Name
			m.mode = configModeName
			m.remote = ""
			m.err = ""
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
	default:
		if len(msg.String()) == 1 && msg.String()[0] >= '0' && msg.String()[0] <= '9' {
			idx := int(msg.String()[0] - '0')
			if idx < len(m.items) {
				m.cursor = idx
				m.clampOffset()
			}
		}
	}
	return m, nil
}

func (m ConfigEditorModel) updateProviderFilter(msg tea.KeyMsg) (ConfigEditorModel, tea.Cmd) {
	switch msg.String() {
	case "esc":
		m.filtering = false
		m.filter = ""
		m.refilter()
	case "enter":
		m.filtering = false
		if m.cursor >= 0 && m.cursor < len(m.items) {
			m.provider = m.items[m.cursor].Name
			m.mode = configModeName
			m.remote = ""
			m.err = ""
		}
	case "backspace":
		m.filter = dropLastRune(m.filter)
		m.refilter()
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
		if text := insertableText(msg); text != "" {
			m.filter += text
			m.refilter()
		}
	}
	return m, nil
}

func (m ConfigEditorModel) updateName(msg tea.KeyMsg) (ConfigEditorModel, tea.Cmd) {
	switch msg.String() {
	case "esc":
		m.mode = configModeProviders
		m.remote = ""
		m.err = ""
	case "enter":
		name := strings.TrimSpace(m.remote)
		if !validConfigRemoteName(name) {
			m.err = "Enter a valid remote name (without : / or \\)."
			return m, nil
		}
		m.remote = name
		m.mode = configModeLoading
		m.err = ""
		return m, StartConfigCreateCmd(m.remote, m.provider)
	case "backspace":
		m.remote = dropLastRune(m.remote)
	case "ctrl+u":
		m.remote = ""
	default:
		if text := insertableText(msg); text != "" {
			m.remote += text
		}
	}
	return m, nil
}

func (m ConfigEditorModel) updateQuestion(msg tea.KeyMsg) (ConfigEditorModel, tea.Cmd) {
	if msg.String() == "esc" {
		return m, func() tea.Msg { return ConfigCancelMsg{Remote: m.remote, DeleteRemote: m.remoteCreated} }
	}
	option := m.question.Option
	if option == nil {
		return m, nil
	}
	if option.Exclusive && len(option.Examples) > 0 {
		switch msg.String() {
		case "up", "k":
			m.answer = exampleValue(option.Examples, -1, m.answer)
			return m, nil
		case "down", "j":
			m.answer = exampleValue(option.Examples, 1, m.answer)
			return m, nil
		}
	}
	switch msg.String() {
	case "enter":
		if option.Required && strings.TrimSpace(m.answer) == "" {
			m.err = "A value is required."
			return m, nil
		}
		m.mode = configModeLoading
		m.err = ""
		return m, RejectConfigAnswerCmd(m.remote)
	case "backspace":
		if !(option.Exclusive && len(option.Examples) > 0) {
			m.answer = dropLastRune(m.answer)
		}
	case "ctrl+u":
		if !(option.Exclusive && len(option.Examples) > 0) {
			m.answer = ""
		}
	default:
		if !(option.Exclusive && len(option.Examples) > 0) {
			if text := insertableText(msg); text != "" {
				m.answer += text
			}
		}
	}
	return m, nil
}

func (m *ConfigEditorModel) refilter() {
	m.items = nil
	query := strings.ToLower(m.filter)
	for _, provider := range m.providers {
		if query == "" || strings.Contains(strings.ToLower(provider.Name+" "+provider.Description), query) {
			m.items = append(m.items, provider)
		}
	}
	if m.cursor >= len(m.items) {
		m.cursor = max(0, len(m.items)-1)
	}
	m.offset = 0
}

func (m *ConfigEditorModel) clampOffset() {
	visible := m.resultHeight()
	if m.cursor < m.offset {
		m.offset = m.cursor
	}
	if m.cursor >= m.offset+visible {
		m.offset = m.cursor - visible + 1
	}
}

func (m ConfigEditorModel) BoxSize(screenWidth, screenHeight int) (int, int) {
	w := screenWidth * 4 / 5
	if w < 56 {
		w = min(56, screenWidth)
	}
	content := 11
	if m.mode == configModeProviders {
		content = len(m.items) + 5
	}
	if m.mode == configModeQuestion && m.question.Option != nil {
		content = 8 + len(wrapText(m.question.Option.Help, max(20, w-8)))
		if m.question.Option.Exclusive {
			content += len(m.question.Option.Examples)
		}
		if m.question.Option.IsPassword || m.question.Option.Sensitive {
			content++
		}
	}
	if content < 9 {
		content = 9
	}
	maxH := screenHeight * 4 / 5
	if maxH < 1 {
		maxH = screenHeight
	}
	if content > maxH {
		content = maxH
	}
	return w, content
}

func (m ConfigEditorModel) resultHeight() int {
	_, h := m.BoxSize(m.width, m.height)
	if h-5 < 1 {
		return 1
	}
	return h - 5
}

func (m ConfigEditorModel) View(th theme.Theme, screenWidth, screenHeight int) string {
	boxW, boxH := m.BoxSize(screenWidth, screenHeight)
	innerW := max(1, boxW-2)
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

	switch m.mode {
	case configModeProviders:
		if m.filtering {
			lines = append(lines, key.Render(" Filter: ")+base.Render(uitext.SafeTerminalText(m.filter)+"_"))
		}
		if m.loading {
			lines = append(lines, dim.Render(" Loading rclone providers…"))
		} else if m.err != "" {
			lines = append(lines, dim.Render(" "+uitext.SafeTerminalText(m.err)))
		} else if len(m.items) == 0 {
			lines = append(lines, dim.Render(" No rclone providers found."))
		} else {
			visible := m.resultHeight()
			end := min(m.offset+visible, len(m.items))
			for i := m.offset; i < end; i++ {
				p := m.items[i]
				prefix := "  "
				if i < 10 {
					prefix = fmt.Sprintf("%d ", i)
				}
				label := uitext.SafeTerminalText(p.Name)
				if p.Description != "" {
					label += " — " + uitext.SafeTerminalText(p.Description)
				}
				if i == m.cursor {
					lines = append(lines, selected.Render(pad(prefix+label, innerW)))
				} else {
					lines = append(lines, key.Render(prefix)+base.Render(pad(label, innerW-len(prefix))))
				}
			}
		}
	case configModeName:
		lines = append(lines, key.Render(" Provider: ")+base.Render(uitext.SafeTerminalText(m.provider)))
		lines = append(lines, dim.Render(" Enter a name for the new rclone remote."))
		lines = append(lines, base.Render(" Name: "+uitext.SafeTerminalText(m.remote)+"_"))
		if m.err != "" {
			lines = append(lines, dim.Render(" "+uitext.SafeTerminalText(m.err)))
		}
	case configModeLoading:
		lines = append(lines, dim.Render(" Working with rclone configuration…"))
	case configModeQuestion:
		if option := m.question.Option; option != nil {
			lines = append(lines, key.Render(" "+uitext.SafeTerminalText(m.remote)+" · "+uitext.SafeTerminalText(option.Name)))
			sensitive := option.IsPassword || option.Sensitive
			if !sensitive {
				lines = append(lines, wrapText(uitext.SafeTerminalText(option.Help), innerW-2)...)
			}
			if option.Required {
				lines = append(lines, dim.Render(" Required"))
			}
			if sensitive {
				shown := strings.Repeat("•", utf8.RuneCountInString(m.answer))
				lines = append(lines, key.Render(" Value: ")+base.Render(shown+"_"))
				if m.answer != "" {
					lines = append(lines, dim.Render(" Enter keeps the stored secret; Ctrl+U clears it for replacement."))
				} else {
					lines = append(lines, dim.Render(" Enter the new secret; use Ctrl+U to clear the field."))
				}
			} else if option.Exclusive && len(option.Examples) > 0 {
				for _, example := range option.Examples {
					value := uitext.SafeTerminalText(valueString(example.Value))
					prefix := "  "
					if value == m.answer {
						prefix = "> "
					}
					label := value
					if example.Help != "" {
						label += " — " + uitext.SafeTerminalText(example.Help)
					}
					lines = append(lines, key.Render(prefix)+base.Render(label))
				}
			} else {
				shown := uitext.SafeTerminalText(m.answer)
				lines = append(lines, key.Render(" Value: ")+base.Render(shown+"_"))
			}
			if m.err != "" {
				lines = append(lines, dim.Render(" "+uitext.SafeTerminalText(m.err)))
			}
		} else if m.err != "" {
			lines = append(lines, dim.Render(" "+uitext.SafeTerminalText(m.err)))
		}
	}

	footer := key.Render(" Enter") + dim.Render(":Continue  ") + key.Render("Esc") + dim.Render(":Cancel")
	if m.mode == configModeProviders {
		footer = key.Render(" Enter") + dim.Render(":Select  ") + key.Render("f") + dim.Render(":Filter  ") + key.Render("Esc") + dim.Render(":Cancel")
	}
	if m.mode == configModeName {
		footer = key.Render(" Enter") + dim.Render(":Create  ") + key.Render("Esc") + dim.Render(":Back")
	}
	if m.mode == configModeQuestion && m.question.Option != nil && m.question.Option.Exclusive && len(m.question.Option.Examples) > 0 {
		footer = key.Render(" ↑/↓") + dim.Render(":Choose  ") + key.Render("Enter") + dim.Render(":Continue  ") + key.Render("Esc") + dim.Render(":Cancel")
	}
	if m.mode == configModeQuestion && m.question.Option != nil && (m.question.Option.IsPassword || m.question.Option.Sensitive) {
		footer = key.Render(" Enter") + dim.Render(":Keep/continue  ") + key.Render("Ctrl+U") + dim.Render(":Replace  ") + key.Render("Esc") + dim.Render(":Cancel")
	}
	if m.mode == configModeLoading {
		footer = dim.Render(" Please wait…")
	}
	return overlay.RenderBox("rclone configuration", lines, footer, boxW, boxH, accent, bg, highlight)
}

func initialAnswer(option rclonefs.ConfigOption) string {
	if option.ValueStr != "" {
		return option.ValueStr
	}
	if option.Value != nil {
		return valueString(option.Value)
	}
	if option.DefaultStr != "" {
		return option.DefaultStr
	}
	return valueString(option.Default)
}

func exampleValue(examples []rclonefs.ConfigExample, delta int, current string) string {
	if len(examples) == 0 {
		return current
	}
	idx := 0
	for i, ex := range examples {
		if valueString(ex.Value) == current {
			idx = i
			break
		}
	}
	idx = (idx + delta + len(examples)) % len(examples)
	return valueString(examples[idx].Value)
}

func valueString(value any) string {
	if value == nil {
		return ""
	}
	return fmt.Sprint(value)
}

func validConfigRemoteName(name string) bool {
	return name != "" && strings.TrimSpace(name) == name && !strings.ContainsAny(name, ":/\\\x00\r\n")
}

func insertableText(msg tea.KeyMsg) string {
	if msg.Alt {
		return ""
	}
	if msg.Type == tea.KeyRunes {
		return string(msg.Runes)
	}
	if msg.Type == tea.KeySpace {
		return " "
	}
	return ""
}

func dropLastRune(value string) string {
	if value == "" {
		return ""
	}
	_, size := utf8.DecodeLastRuneInString(value)
	return value[:len(value)-size]
}

func wrapText(text string, width int) []string {
	if width < 1 {
		width = 1
	}
	var out []string
	for _, paragraph := range strings.Split(text, "\n") {
		words := strings.Fields(paragraph)
		if len(words) == 0 {
			out = append(out, " ")
			continue
		}
		line := " "
		for _, word := range words {
			if len(strings.TrimRight(line, " "))+1+len(word) > width && strings.TrimSpace(line) != "" {
				out = append(out, line)
				line = " "
			}
			if line != " " {
				line += " "
			}
			line += word
		}
		out = append(out, line)
	}
	return out
}
