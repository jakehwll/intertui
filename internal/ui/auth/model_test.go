package auth

import (
	"os"
	"strings"
	"testing"

	tea "charm.land/bubbletea/v2"

	"intertui/internal/config"
)

func TestNeeds(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		cfg  config.Config
		want bool
	}{
		{
			name: "empty",
			cfg:  config.Config{},
			want: true,
		},
		{
			name: "user only",
			cfg:  config.Config{User: "alice", Pass: "secret"},
			want: true,
		},
		{
			name: "server only",
			cfg:  config.Config{Server: "game.example"},
			want: true,
		},
		{
			name: "complete",
			cfg:  config.Config{Server: "game.example", User: "alice", Pass: "secret"},
			want: false,
		},
		{
			name: "url only",
			cfg:  config.Config{URL: "ws://localhost:13373/ws", User: "alice"},
			want: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			if got := Needs(tt.cfg); got != tt.want {
				t.Fatalf("Needs() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestMenuNavigation(t *testing.T) {
	t.Parallel()

	model := New(config.Config{})
	model.ready = true
	model.width = 80
	model.height = 24

	next, _ := model.Update(downKey())
	model = next.(Model)
	if model.menuCursor != choiceRegister {
		t.Fatalf("menuCursor = %v, want register", model.menuCursor)
	}

	next, _ = model.Update(enterKey())
	model = next.(Model)
	if model.screen != screenRegister {
		t.Fatalf("screen = %v, want register form", model.screen)
	}

	view := model.View().Content
	if !strings.Contains(view, "Register") {
		t.Fatalf("view missing register title: %q", view)
	}
}

func TestLoginSubmitWritesConfig(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("HOME", dir)

	m := New(config.Config{})
	m.ready = true
	m.screen = screenLogin
	m.server.SetValue("game.example")
	m.user.SetValue("alice")
	m.pass.SetValue("secret")

	final, cmd := m.submitForm()
	m = final.(Model)
	if cmd == nil {
		t.Fatal("expected quit command")
	}
	if !m.done {
		t.Fatal("expected done")
	}
	if m.result.Server != "game.example" || m.result.User != "alice" {
		t.Fatalf("result = %+v", m.result)
	}
}

func TestLoginWithoutSaveSkipsConfig(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("HOME", dir)

	m := New(config.Config{})
	m.ready = true
	m.screen = screenLogin
	m.saveCreds = false
	m.server.SetValue("game.example")
	m.user.SetValue("alice")
	m.pass.SetValue("secret")

	final, cmd := m.submitForm()
	m = final.(Model)
	if cmd == nil {
		t.Fatal("expected quit command")
	}
	if !m.done {
		t.Fatal("expected done")
	}

	path, err := config.ConfigPath()
	if err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(path); !os.IsNotExist(err) {
		t.Fatalf("config should not exist, stat err=%v", err)
	}
}

func TestSaveCredsToggle(t *testing.T) {
	t.Parallel()

	m := New(config.Config{})
	m.ready = true
	m.screen = screenLogin
	m.focusForm(3)

	next, _ := m.Update(tea.KeyPressMsg{Code: tea.KeySpace})
	m = next.(Model)
	if m.saveCreds {
		t.Fatal("expected saveCreds false after space")
	}

	view := m.View().Content
	if !strings.Contains(view, "Save credentials") || strings.Contains(view, "[x]") {
		t.Fatalf("view should show unchecked save creds: %q", view)
	}
}

func TestLoginSubmitWritesConfigFile(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("HOME", dir)

	m := New(config.Config{})
	m.ready = true
	m.screen = screenLogin
	m.server.SetValue("game.example")
	m.user.SetValue("alice")
	m.pass.SetValue("secret")

	final, _ := m.submitForm()
	m = final.(Model)
	if !m.done {
		t.Fatal("expected done")
	}

	path, err := config.ConfigPath()
	if err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(path); err != nil {
		t.Fatalf("config should exist: %v", err)
	}
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(data), "game.example") || !strings.Contains(string(data), "alice") {
		t.Fatalf("config contents unexpected: %s", data)
	}
}

func downKey() tea.KeyPressMsg {
	return tea.KeyPressMsg{Code: tea.KeyDown, Text: ""}
}

func enterKey() tea.KeyPressMsg {
	return tea.KeyPressMsg{Code: tea.KeyEnter}
}

func TestFormClickFocusesField(t *testing.T) {
	t.Parallel()

	model := New(config.Config{})
	model.ready = true
	model.width = 80
	model.height = 24
	model.screen = screenLogin

	body := model.formBody()
	ox, oy := bodyOffset(body, model.width, model.height)
	fieldY := oy + formHeaderLines + 1 // server input row

	next, _ := model.Update(tea.MouseClickMsg{X: ox + 4, Y: fieldY, Button: tea.MouseLeft})
	model = next.(Model)
	if model.focus != 0 {
		t.Fatalf("focus = %d, want server field", model.focus)
	}
}

func TestFormClickTogglesSaveCreds(t *testing.T) {
	t.Parallel()

	model := New(config.Config{})
	model.ready = true
	model.width = 80
	model.height = 24
	model.screen = screenLogin
	if !model.saveCreds {
		t.Fatal("expected saveCreds true by default")
	}

	body := model.formBody()
	ox, oy := bodyOffset(body, model.width, model.height)

	next, _ := model.Update(tea.MouseClickMsg{
		X: ox + 2, Y: oy + formCheckboxLine, Button: tea.MouseLeft,
	})
	model = next.(Model)
	if model.saveCreds {
		t.Fatal("expected saveCreds toggled off")
	}
	if model.focus != 3 {
		t.Fatalf("focus = %d, want checkbox", model.focus)
	}
}

func TestMenuClickSelectsItem(t *testing.T) {
	t.Parallel()

	model := New(config.Config{})
	model.ready = true
	model.width = 80
	model.height = 24

	body := model.viewMenu()
	_, bodyH := bodyContentSize(body)
	ox, oy := bodyOffset(body, model.width, model.height)
	start, _ := menuItemLineRange(int(choiceRegister))

	clickY := oy + start
	clickX := ox + 4

	next, _ := model.Update(tea.MouseClickMsg{X: clickX, Y: clickY, Button: tea.MouseLeft})
	model = next.(Model)
	if model.screen != screenRegister {
		t.Fatalf("screen = %v, want register form after click", model.screen)
	}
	if model.menuCursor != choiceRegister {
		t.Fatalf("menuCursor = %v, want register", model.menuCursor)
	}
	if bodyH < 10 {
		t.Fatalf("unexpected body height %d", bodyH)
	}
}
