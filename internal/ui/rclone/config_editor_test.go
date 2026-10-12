package rclone

import (
	"errors"
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/kooler/MiddayCommander/internal/ui/theme"
	"github.com/kooler/MiddayCommander/internal/vfs/rclonefs"
)

func TestPasswordAnswerIsNeverRendered(t *testing.T) {
	const secret = "not-a-real-password-123"
	m := ConfigEditorModel{mode: configModeQuestion, width: 100, height: 30}
	m.HandleQuestion(ConfigQuestionMsg{
		Remote: "cloud",
		Question: rclonefs.ConfigQuestion{
			State:  "state",
			Option: &rclonefs.ConfigOption{Name: "client_secret", Help: "Client secret", ValueStr: secret, IsPassword: true},
		},
	})
	view := m.View(themeForTest(), 100, 30)
	if strings.Contains(view, secret) {
		t.Fatal("password was rendered in the config dialog")
	}
	if !strings.Contains(view, "••••") {
		t.Fatal("password mask is missing")
	}
	if !strings.Contains(view, "Ctrl+U") || !strings.Contains(view, "keeps the stored secret") {
		t.Fatal("password keep/replace instructions are missing")
	}
	_, cmd := m.Update(tea.KeyMsg{Type: tea.KeyEnter})
	if cmd == nil {
		t.Fatal("Enter should continue the config wizard")
	}
	msg, ok := cmd().(ConfigQuestionMsg)
	if !ok || msg.Err == nil || !strings.Contains(msg.Err.Error(), "native `rclone config`") {
		t.Fatalf("continuation = %#v; secret answers must use native rclone config", msg)
	}
}

func TestCreateConfigFlowSelectsProviderAndRequestsName(t *testing.T) {
	m := NewConfigCreate(100, 30)
	m.HandleProviders(ConfigProvidersLoadedMsg{Providers: []rclonefs.Provider{{Name: "local", Description: "Local disk"}}})
	m, _ = m.Update(tea.KeyMsg{Type: tea.KeyEnter})
	if m.mode != configModeName || m.provider != "local" {
		t.Fatalf("after provider selection: mode=%d provider=%q", m.mode, m.provider)
	}
	for _, r := range "work" {
		m, _ = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{r}})
	}
	if m.remote != "work" {
		t.Fatalf("remote name = %q", m.remote)
	}
}

func TestFailedCreateDoesNotDeleteExistingRemote(t *testing.T) {
	m := NewConfigCreate(100, 30)
	m.HandleQuestion(ConfigQuestionMsg{Remote: "existing", Provider: "local", Creating: true, Err: errors.New("already exists")})
	_, cmd := m.Update(tea.KeyMsg{Type: tea.KeyEsc})
	if cmd == nil {
		t.Fatal("Esc should close the failed create dialog")
	}
	msg, ok := cmd().(ConfigCancelMsg)
	if !ok || msg.DeleteRemote {
		t.Fatalf("cancel = %#v; must not delete an existing remote", msg)
	}
}

func TestExistingRemoteNameCannotEmitTerminalControlSequence(t *testing.T) {
	const maliciousRemote = "backup\x1b]52;c;payload\a"
	m := NewConfigEdit(maliciousRemote, 100, 30)
	m.HandleQuestion(ConfigQuestionMsg{
		Remote: maliciousRemote,
		Question: rclonefs.ConfigQuestion{
			State:  "state",
			Option: &rclonefs.ConfigOption{Name: "token", Help: "Token value"},
		},
	})
	view := m.View(themeForTest(), 100, 30)
	if strings.Contains(view, "\x1b]52;") || strings.ContainsRune(view, '\a') {
		t.Fatal("rclone remote name emitted a terminal control sequence")
	}
}

func TestSensitiveExclusiveExamplesNeverRenderValueOrHelp(t *testing.T) {
	const exampleValue = "synthetic-secret-example-value"
	const exampleHelp = "synthetic-secret-example-help"
	const optionHelp = "synthetic-secret-option-help"
	const currentSecret = "synthetic-current-secret-value"

	for _, test := range []struct {
		name       string
		isPassword bool
		sensitive  bool
	}{
		{name: "password", isPassword: true},
		{name: "sensitive", sensitive: true},
	} {
		t.Run(test.name, func(t *testing.T) {
			m := ConfigEditorModel{
				mode:   configModeQuestion,
				remote: "test",
				answer: currentSecret,
				width:  100,
				height: 30,
				question: rclonefs.ConfigQuestion{Option: &rclonefs.ConfigOption{
					Name:       "credential",
					Help:       optionHelp,
					IsPassword: test.isPassword,
					Sensitive:  test.sensitive,
					Exclusive:  true,
					Examples:   []rclonefs.ConfigExample{{Value: exampleValue, Help: exampleHelp}},
				}},
			}
			view := m.View(themeForTest(), 100, 30)
			for _, secretText := range []string{exampleValue, exampleHelp, optionHelp, currentSecret} {
				if strings.Contains(view, secretText) {
					t.Fatalf("sensitive example text %q was rendered", secretText)
				}
			}
		})
	}
}

func themeForTest() theme.Theme { return theme.Theme{} }
