package servers

import (
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/kooler/MiddayCommander/internal/remote"
	uitext "github.com/kooler/MiddayCommander/internal/ui/text"
	"github.com/kooler/MiddayCommander/internal/ui/theme"
)

func newStore(t *testing.T, srvs ...remote.Server) *remote.Store {
	t.Helper()
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	s := remote.LoadStore()
	for _, srv := range srvs {
		s.Add(srv)
	}
	return s
}

func key(s string) tea.KeyMsg {
	if len(s) == 1 {
		return tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune(s)}
	}
	switch s {
	case "enter":
		return tea.KeyMsg{Type: tea.KeyEnter}
	case "esc":
		return tea.KeyMsg{Type: tea.KeyEsc}
	case "tab":
		return tea.KeyMsg{Type: tea.KeyTab}
	case "down":
		return tea.KeyMsg{Type: tea.KeyDown}
	case "backspace":
		return tea.KeyMsg{Type: tea.KeyBackspace}
	}
	return tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune(s)}
}

func typeIn(m Model, text string) Model {
	for _, r := range text {
		m, _ = m.Update(key(string(r)))
	}
	return m
}

// Narrow sizes are where the padding maths can slice past a string's end.
func TestViewRendersAtAnySize(t *testing.T) {
	store := newStore(t,
		remote.Server{Name: "prod", Host: "prod.example.com", User: "deploy", Dir: "/srv/app"},
		remote.Server{Name: "a-very-long-server-name-that-overflows", Host: "long.example.com"},
	)

	sizes := [][2]int{{100, 30}, {60, 20}, {40, 12}, {24, 9}}
	for _, size := range sizes {
		m := New(store, size[0], size[1])
		if out := m.View(theme.Theme{}, size[0], size[1]); out == "" {
			t.Errorf("%dx%d: empty render", size[0], size[1])
		}

		m.startAdd()
		if out := m.View(theme.Theme{}, size[0], size[1]); out == "" {
			t.Errorf("%dx%d: empty form render", size[0], size[1])
		}
	}
}

func TestEmptyListRenders(t *testing.T) {
	m := New(newStore(t), 80, 24)
	out := m.View(theme.Theme{}, 80, 24)
	if !strings.Contains(out, "No servers yet") {
		t.Error("want the empty-list hint")
	}
}

func TestAddServerThroughTheForm(t *testing.T) {
	store := newStore(t)
	m := New(store, 80, 24)

	m, _ = m.Update(key("a"))
	m = typeIn(m, "staging")    // Name
	m, _ = m.Update(key("tab")) // -> Host
	m = typeIn(m, "staging.example.com")
	m, _ = m.Update(key("tab")) // -> Port
	m = typeIn(m, "2222")
	m, _ = m.Update(key("enter")) // save

	saved, ok := store.Find("staging")
	if !ok {
		t.Fatal("want the server saved")
	}
	if saved.Host != "staging.example.com" || saved.Port != 2222 {
		t.Errorf("server did not save correctly: %+v", saved)
	}
	if m.mode != modeList {
		t.Error("want the form closed after saving")
	}
}

func TestFormRejectsAMissingHost(t *testing.T) {
	store := newStore(t)
	m := New(store, 80, 24)

	m, _ = m.Update(key("a"))
	m = typeIn(m, "nameonly")
	m, _ = m.Update(key("enter"))

	if m.mode != modeForm {
		t.Error("want the form to stay open when the host is missing")
	}
	if m.formErr == "" {
		t.Error("want an error message explaining what is missing")
	}
	if len(store.Servers) != 0 {
		t.Error("want nothing saved")
	}
}

func TestFormRejectsABadPort(t *testing.T) {
	store := newStore(t)
	m := New(store, 80, 24)

	m, _ = m.Update(key("a"))
	m, _ = m.Update(key("tab")) // -> Host
	m = typeIn(m, "example.com")
	m, _ = m.Update(key("tab")) // -> Port
	m = typeIn(m, "99999")
	m, _ = m.Update(key("enter"))

	if m.formErr == "" {
		t.Error("want a port validation error")
	}
	if len(store.Servers) != 0 {
		t.Error("want nothing saved with an invalid port")
	}
}

func TestNameDefaultsToHost(t *testing.T) {
	store := newStore(t)
	m := New(store, 80, 24)

	m, _ = m.Update(key("a"))
	m, _ = m.Update(key("tab"))
	m = typeIn(m, "box.example.com")
	_, _ = m.Update(key("enter"))

	if _, ok := store.Find("box.example.com"); !ok {
		t.Error("want the host used as the name when none is given")
	}
}

func TestEnterConnects(t *testing.T) {
	store := newStore(t, remote.Server{Name: "prod", Host: "prod.example.com"})
	m := New(store, 80, 24)

	_, cmd := m.Update(key("enter"))
	if cmd == nil {
		t.Fatal("want a connect command")
	}
	msg, ok := cmd().(ConnectMsg)
	if !ok {
		t.Fatalf("want ConnectMsg, got %T", cmd())
	}
	if msg.Server.Host != "prod.example.com" {
		t.Errorf("want the selected server, got %+v", msg.Server)
	}
}

func TestDeleteRemovesServer(t *testing.T) {
	store := newStore(t, remote.Server{Name: "gone", Host: "gone.example.com"})
	m := New(store, 80, 24)

	m, _ = m.Update(key("d"))

	if _, ok := store.Find("gone"); ok {
		t.Error("want the server removed")
	}
	if len(m.items) != 0 {
		t.Error("want the list refreshed after deleting")
	}
}

func TestEscDismisses(t *testing.T) {
	m := New(newStore(t), 80, 24)
	_, cmd := m.Update(key("esc"))
	if cmd == nil {
		t.Fatal("want a dismiss command")
	}
	if _, ok := cmd().(DismissMsg); !ok {
		t.Errorf("want DismissMsg, got %T", cmd())
	}
}

func TestFilterNarrowsTheList(t *testing.T) {
	store := newStore(t,
		remote.Server{Name: "prod", Host: "prod.example.com"},
		remote.Server{Name: "staging", Host: "staging.example.com"},
	)
	m := New(store, 80, 24)

	m, _ = m.Update(key("f"))
	m = typeIn(m, "stag")

	if len(m.items) != 1 || m.items[0].Name != "staging" {
		t.Errorf("want only the staging server, got %v", m.items)
	}
}

func TestEditPrefillsAndReplaces(t *testing.T) {
	store := newStore(t, remote.Server{Name: "box", Host: "old.example.com", Port: 2222})
	m := New(store, 80, 24)

	m, _ = m.Update(key("e"))
	if m.values[fieldName] != "box" || m.values[fieldHost] != "old.example.com" {
		t.Fatalf("want the form prefilled, got %v", m.values)
	}
	if m.values[fieldPort] != "2222" {
		t.Errorf("want the port prefilled, got %q", m.values[fieldPort])
	}

	// Clear the host and type a new one.
	m, _ = m.Update(key("tab"))
	for range "old.example.com" {
		m, _ = m.Update(key("backspace"))
	}
	m = typeIn(m, "new.example.com")
	m, _ = m.Update(key("enter"))

	saved, ok := store.Find("box")
	if !ok {
		t.Fatal("want the server still saved under its name")
	}
	if saved.Host != "new.example.com" {
		t.Errorf("want the host updated, got %q", saved.Host)
	}
	if len(store.Servers) != 1 {
		t.Errorf("want one server, not a duplicate: %d", len(store.Servers))
	}
}

func TestServerLabelsAreSanitizedOnlyForRender(t *testing.T) {
	const hostileName = "p\u202e"
	const hostileHost = "h\x1b]x\x07"
	const hostileDir = "/\u202d"
	server := remote.Server{Name: hostileName, Host: hostileHost, User: "deploy", Dir: hostileDir}
	store := newStore(t, server)
	m := New(store, 80, 24)

	view := m.View(theme.Default(), 80, 24)
	want := uitext.SafeTerminalText(hostileName + "  ssh://deploy@" + hostileHost + " " + hostileDir)
	if !strings.Contains(view, want) {
		t.Fatalf("server view should contain sanitized details %q: %q", want, view)
	}
	if strings.Contains(view, hostileName) || strings.Contains(view, hostileHost) || strings.Contains(view, hostileDir) {
		t.Fatalf("server view exposed raw details: %q", view)
	}
	got := m.items[0]
	if got.Name != hostileName || got.Host != hostileHost || got.Dir != hostileDir {
		t.Fatalf("rendering changed the saved server: %#v", got)
	}
}

func TestSavedServerEditFormSanitizesFieldsOnlyForRender(t *testing.T) {
	const hostileName = "n\x1b]x\x07\u202e"
	const hostileHost = "h\x1b]x\x07"
	const hostileDir = "/d\u202d"
	store := newStore(t, remote.Server{Name: hostileName, Host: hostileHost, Dir: hostileDir})
	m := New(store, 120, 24)
	m.startEdit()

	view := m.View(theme.Default(), 120, 24)
	for _, raw := range []string{hostileName, hostileHost, hostileDir} {
		if strings.Contains(view, raw) {
			t.Errorf("edit form exposed raw saved value %q: %q", raw, view)
		}
		if safe := uitext.SafeTerminalText(raw); !strings.Contains(view, safe) {
			t.Errorf("edit form omitted sanitized saved value %q: %q", safe, view)
		}
	}
	if m.values[fieldName] != hostileName || m.values[fieldHost] != hostileHost || m.values[fieldDir] != hostileDir {
		t.Fatalf("rendering changed saved form values: %#v", m.values)
	}
}

func TestServerFilterSanitizesOnlyForRender(t *testing.T) {
	const hostileFilter = "host\u202e"
	m := New(newStore(t), 80, 24)
	m.mode = modeFilter
	m.filter = hostileFilter

	view := m.View(theme.Default(), 80, 24)
	if !strings.Contains(view, uitext.SafeTerminalText(hostileFilter)) || strings.Contains(view, hostileFilter) {
		t.Fatalf("server filter was not safely rendered: %q", view)
	}
	if m.filter != hostileFilter {
		t.Fatalf("rendering changed server filter: %q", m.filter)
	}
}
