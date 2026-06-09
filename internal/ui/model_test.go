package ui

import (
	"strings"
	"testing"

	tea "charm.land/bubbletea/v2"

	"intertui/internal/config"
	"intertui/internal/intercept"
)

func sizedModel(t *testing.T) Model {
	t.Helper()

	m := New(config.Config{})
	updated, _ := m.Update(tea.WindowSizeMsg{Width: 80, Height: 24})
	model, ok := updated.(Model)
	if !ok {
		t.Fatal("expected Model")
	}
	if !model.ready {
		t.Fatal("expected ready model after WindowSizeMsg")
	}
	return model
}

func connectedModel(t *testing.T) Model {
	t.Helper()

	m := sizedModel(t)
	updated, _ := m.Update(clientReadyMsg{user: "alice"})
	model, ok := updated.(Model)
	if !ok {
		t.Fatal("expected Model")
	}
	return model
}

func hasMessage(msgs []string, want string) bool {
	for _, msg := range msgs {
		if strings.Contains(msg, want) {
			return true
		}
	}
	return false
}

func TestModelUpdateConnect(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name       string
		msg        tea.Msg
		wantState  connState
		wantUser   string
		wantInLog  []string
		wantNoLog  []string
		wantCmd    bool
	}{
		{
			name: "connect progress appends status",
			msg: connectProgressMsg{
				line:     "Logging in…",
				statusCh: closedStringCh(),
				doneCh:   make(chan clientReadyMsg),
			},
			wantState: stateConnecting,
			wantInLog: []string{"Logging in"},
			wantCmd:   true,
		},
		{
			name:      "connect success clears log and focuses input",
			msg:       clientReadyMsg{user: "alice"},
			wantState: stateConnected,
			wantUser:  "alice",
			wantNoLog: []string{"Intercept terminal", "Target:"},
			wantCmd:   true,
		},
		{
			name:      "connect failure shows error",
			msg:       clientReadyMsg{err: errTest("dial refused")},
			wantState: stateError,
			wantInLog: []string{"Connection failed: dial refused"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			m := sizedModel(t)
			updated, cmd := m.Update(tt.msg)
			model := updated.(Model)

			if model.state != tt.wantState {
				t.Fatalf("state = %v, want %v", model.state, tt.wantState)
			}
			if tt.wantUser != "" && model.connectedUser != tt.wantUser {
				t.Fatalf("connectedUser = %q, want %q", model.connectedUser, tt.wantUser)
			}
			for _, want := range tt.wantInLog {
				if !hasMessage(model.messages, want) {
					t.Fatalf("messages missing %q: %#v", want, model.messages)
				}
			}
			for _, omit := range tt.wantNoLog {
				if hasMessage(model.messages, omit) {
					t.Fatalf("messages still contain %q: %#v", omit, model.messages)
				}
			}
			if tt.wantCmd && cmd == nil {
				t.Fatal("expected non-nil cmd")
			}
			if !tt.wantCmd && cmd != nil {
				t.Fatalf("expected nil cmd, got %#v", cmd)
			}
		})
	}
}

func TestModelUpdateGameLine(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name      string
		line      string
		wantInLog []string
		wantCmd   bool
	}{
		{
			name:      "append game output",
			line:      "welcome to intercept",
			wantInLog: []string{"welcome to intercept"},
			wantCmd:   false,
		},
		{
			name:      "append unknown event summary",
			line:      "server → clink, ok",
			wantInLog: []string{"server → clink, ok"},
			wantCmd:   false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			m := connectedModel(t)
			updated, cmd := m.Update(intercept.GameLineMsg{Line: tt.line})
			model := updated.(Model)

			for _, want := range tt.wantInLog {
				if !hasMessage(model.messages, want) {
					t.Fatalf("messages missing %q: %#v", want, model.messages)
				}
			}
			if tt.wantCmd && cmd == nil {
				t.Fatal("expected non-nil cmd")
			}
			if !tt.wantCmd && cmd != nil {
				t.Fatalf("expected nil cmd, got %#v", cmd)
			}
		})
	}
}

func TestModelUpdateDisconnect(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name      string
		setup     func(t *testing.T) Model
		msg       intercept.DisconnectedMsg
		wantState connState
		wantInLog []string
	}{
		{
			name:      "connected session ends in error state",
			setup:     connectedModel,
			msg:       intercept.DisconnectedMsg{},
			wantState: stateError,
			wantInLog: []string{"Disconnected."},
		},
		{
			name:      "disconnect reason is shown",
			setup:     connectedModel,
			msg:       intercept.DisconnectedMsg{Err: errTest("connection reset")},
			wantState: stateError,
			wantInLog: []string{"Disconnected: connection reset"},
		},
		{
			name: "ignored while connecting",
			setup: func(t *testing.T) Model {
				return sizedModel(t)
			},
			msg:       intercept.DisconnectedMsg{Err: errTest("early close")},
			wantState: stateConnecting,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			m := tt.setup(t)
			updated, _ := m.Update(tt.msg)
			model := updated.(Model)

			if model.state != tt.wantState {
				t.Fatalf("state = %v, want %v", model.state, tt.wantState)
			}
			for _, want := range tt.wantInLog {
				if !hasMessage(model.messages, want) {
					t.Fatalf("messages missing %q: %#v", want, model.messages)
				}
			}
		})
	}
}

func TestModelUpdateHistory(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name      string
		actions   func(t *testing.T, m *Model)
		wantInput string
		wantHist  []string
	}{
		{
			name: "submit stores command history",
			actions: func(t *testing.T, m *Model) {
				submit(t, m, "help")
				submit(t, m, "scan")
			},
			wantInput: "",
			wantHist:  []string{"help", "scan"},
		},
		{
			name: "ctrl+p walks backward through history",
			actions: func(t *testing.T, m *Model) {
				submit(t, m, "help")
				submit(t, m, "scan")
				pressKey(t, m, ctrlKey('p'))
				pressKey(t, m, ctrlKey('p'))
			},
			wantInput: "help",
			wantHist:  []string{"help", "scan"},
		},
		{
			name: "ctrl+n restores draft after history",
			actions: func(t *testing.T, m *Model) {
				submit(t, m, "help")
				m.input.SetValue("draft")
				pressKey(t, m, ctrlKey('p'))
				pressKey(t, m, ctrlKey('n'))
			},
			wantInput: "draft",
			wantHist:  []string{"help"},
		},
		{
			name: "duplicate submits are not stored twice",
			actions: func(t *testing.T, m *Model) {
				submit(t, m, "help")
				submit(t, m, "help")
			},
			wantInput: "",
			wantHist:  []string{"help"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			m := connectedModel(t)
			tt.actions(t, &m)

			if got := m.input.Value(); got != tt.wantInput {
				t.Fatalf("input = %q, want %q", got, tt.wantInput)
			}
			if len(m.history) != len(tt.wantHist) {
				t.Fatalf("history = %#v, want %#v", m.history, tt.wantHist)
			}
			for i, want := range tt.wantHist {
				if m.history[i] != want {
					t.Fatalf("history[%d] = %q, want %q", i, m.history[i], want)
				}
			}
		})
	}
}

func disconnectedModel(t *testing.T) Model {
	t.Helper()

	m := connectedModel(t)
	updated, _ := m.Update(intercept.GameLineMsg{Line: "game output"})
	m = updated.(Model)
	updated, _ = m.Update(intercept.DisconnectedMsg{})
	return updated.(Model)
}

func TestModelUpdateReconnect(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name      string
		setup     func(t *testing.T) Model
		actions   func(t *testing.T, m *Model)
		wantState connState
		wantInLog []string
	}{
		{
			name:  "r starts reconnect from error state",
			setup: disconnectedModel,
			actions: func(t *testing.T, m *Model) {
				pressKey(t, m, tea.KeyPressMsg{Code: 'r'})
			},
			wantState: stateConnecting,
			wantInLog: []string{"game output", "Disconnected.", "Reconnecting"},
		},
		{
			name:  "r is ignored while connected",
			setup: connectedModel,
			actions: func(t *testing.T, m *Model) {
				pressKey(t, m, tea.KeyPressMsg{Code: 'r'})
			},
			wantState: stateConnected,
		},
		{
			name:  "reconnect success preserves log",
			setup: disconnectedModel,
			actions: func(t *testing.T, m *Model) {
				pressKey(t, m, tea.KeyPressMsg{Code: 'r'})
				updated, _ := m.Update(clientReadyMsg{user: "alice"})
				*m = updated.(Model)
			},
			wantState: stateConnected,
			wantInLog: []string{"game output", "Disconnected.", "Reconnecting", "Reconnected."},
		},
		{
			name:  "reconnect failure returns to error state",
			setup: disconnectedModel,
			actions: func(t *testing.T, m *Model) {
				pressKey(t, m, tea.KeyPressMsg{Code: 'r'})
				updated, _ := m.Update(clientReadyMsg{err: errTest("dial refused")})
				*m = updated.(Model)
			},
			wantState: stateError,
			wantInLog: []string{"game output", "Reconnecting", "Connection failed: dial refused"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			m := tt.setup(t)
			tt.actions(t, &m)

			if m.state != tt.wantState {
				t.Fatalf("state = %v, want %v", m.state, tt.wantState)
			}
			for _, want := range tt.wantInLog {
				if !hasMessage(m.messages, want) {
					t.Fatalf("messages missing %q: %#v", want, m.messages)
				}
			}
		})
	}
}

func TestModelUpdateReconnectStartsClient(t *testing.T) {
	t.Parallel()

	m := disconnectedModel(t)
	updated, cmd := m.Update(tea.KeyPressMsg{Code: 'r'})
	model := updated.(Model)

	if model.state != stateConnecting {
		t.Fatalf("state = %v, want connecting", model.state)
	}
	if !model.reconnecting {
		t.Fatal("expected reconnecting=true")
	}
	if cmd == nil {
		t.Fatal("expected reconnect cmd")
	}
}

func submit(t *testing.T, m *Model, value string) {
	t.Helper()

	m.input.SetValue(value)
	pressKey(t, m, tea.KeyPressMsg{Code: tea.KeyEnter})
}

func pressKey(t *testing.T, m *Model, key tea.KeyPressMsg) {
	t.Helper()

	updated, _ := m.Update(key)
	*m = updated.(Model)
}

func ctrlKey(ch rune) tea.KeyPressMsg {
	return tea.KeyPressMsg{Code: ch, Mod: tea.ModCtrl}
}

func closedStringCh() <-chan string {
	ch := make(chan string, 1)
	close(ch)
	return ch
}

type errTest string

func (e errTest) Error() string { return string(e) }
