package rclone

import (
	"errors"
	"os"
	"os/exec"
	"path/filepath"
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

func TestUpdatePreservesUnchangedSensitiveValue(t *testing.T) {
	if _, err := exec.LookPath("rclone"); err != nil {
		t.Skip("rclone is not installed")
	}
	configPath := filepath.Join(t.TempDir(), "rclone.conf")
	t.Setenv("RCLONE_CONFIG", configPath)
	create := exec.Command("rclone", "config", "create", "mdc-editor-secret-check", "crypt", "remote=local:/tmp", "password=not-a-real-secret", "--non-interactive")
	create.Stderr = nil
	if output, err := create.Output(); err != nil {
		t.Fatalf("create synthetic crypt fixture: status=%v output=%s", err, output)
	}

	showPassword := func() string {
		t.Helper()
		cmd := exec.Command("rclone", "config", "show", "mdc-editor-secret-check")
		output, err := cmd.Output()
		if err != nil {
			t.Fatalf("read synthetic config: %v", err)
		}
		for _, line := range strings.Split(string(output), "\n") {
			if strings.HasPrefix(strings.TrimSpace(line), "password =") {
				return strings.TrimSpace(strings.TrimPrefix(strings.TrimSpace(line), "password ="))
			}
		}
		t.Fatal("password option missing from synthetic config")
		return ""
	}
	before := showPassword()
	question, err := rclonefs.StartConfigUpdate("mdc-editor-secret-check")
	if err != nil {
		t.Fatal(err)
	}
	creating := false
	sensitiveSeen := false
	for i := 0; i < 100 && question.State != ""; i++ {
		if question.Option == nil {
			t.Fatalf("missing option in state %q", question.State)
		}
		if question.Option.IsPassword || question.Option.Sensitive {
			sensitiveSeen = true
		}
		answer := initialAnswer(*question.Option)
		question, err = rclonefs.ContinueConfig("mdc-editor-secret-check", "", creating, question.State, answer)
		if err != nil {
			t.Fatal(err)
		}
	}
	if question.State != "" || !sensitiveSeen {
		t.Fatalf("wizard did not complete or ask a sensitive field: state=%q sensitive=%v", question.State, sensitiveSeen)
	}
	if after := showPassword(); after != before {
		t.Fatal("keeping the masked password changed the stored value")
	}
	_ = os.Remove(configPath)
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

func themeForTest() theme.Theme { return theme.Theme{} }
