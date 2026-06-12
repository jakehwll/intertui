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
			name: "connect progress drains status quietly",
			msg: connectProgressMsg{
				statusCh: closedStringCh(),
				doneCh:   make(chan clientReadyMsg),
			},
			wantState: stateConnecting,
			wantCmd:   true,
		},
		{
			name:      "connect success clears log and focuses input",
			msg:       clientReadyMsg{user: "alice"},
			wantState: stateConnected,
			wantUser:  "alice",
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
		},
		{
			name:      "append unknown event summary",
			line:      "server → clink, ok",
			wantInLog: []string{"server → clink, ok"},
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
		wantInLog string
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
			name: "submit echoes command with input prompt",
			actions: func(t *testing.T, m *Model) {
				submit(t, m, "ls")
			},
			wantInput: "",
			wantHist:  []string{"ls"},
			wantInLog: "> ",
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
			name: "up and down walk command history",
			actions: func(t *testing.T, m *Model) {
				submit(t, m, "help")
				submit(t, m, "scan")
				pressKey(t, m, tea.KeyPressMsg{Code: tea.KeyUp})
				pressKey(t, m, tea.KeyPressMsg{Code: tea.KeyUp})
				pressKey(t, m, tea.KeyPressMsg{Code: tea.KeyDown})
			},
			wantInput: "scan",
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
			name: "up filters history by typed prefix",
			actions: func(t *testing.T, m *Model) {
				submit(t, m, "help")
				submit(t, m, "cmds filesystem")
				submit(t, m, "scan")
				submit(t, m, "cmds client")
				m.input.SetValue("cmds ")
				pressKey(t, m, tea.KeyPressMsg{Code: tea.KeyUp})
				pressKey(t, m, tea.KeyPressMsg{Code: tea.KeyUp})
			},
			wantInput: "cmds filesystem",
			wantHist:  []string{"help", "cmds filesystem", "scan", "cmds client"},
		},
		{
			name: "down walks prefix-filtered history forward",
			actions: func(t *testing.T, m *Model) {
				submit(t, m, "help")
				submit(t, m, "cmds filesystem")
				submit(t, m, "scan")
				submit(t, m, "cmds client")
				m.input.SetValue("cmds ")
				pressKey(t, m, tea.KeyPressMsg{Code: tea.KeyUp})
				pressKey(t, m, tea.KeyPressMsg{Code: tea.KeyUp})
				pressKey(t, m, tea.KeyPressMsg{Code: tea.KeyDown})
			},
			wantInput: "cmds client",
			wantHist:  []string{"help", "cmds filesystem", "scan", "cmds client"},
		},
		{
			name: "up at oldest prefix match stays put",
			actions: func(t *testing.T, m *Model) {
				submit(t, m, "cmds filesystem")
				submit(t, m, "cmds client")
				m.input.SetValue("cmds ")
				pressKey(t, m, tea.KeyPressMsg{Code: tea.KeyUp})
				pressKey(t, m, tea.KeyPressMsg{Code: tea.KeyUp})
				pressKey(t, m, tea.KeyPressMsg{Code: tea.KeyUp})
			},
			wantInput: "cmds filesystem",
			wantHist:  []string{"cmds filesystem", "cmds client"},
		},
		{
			name: "down works after extra up at oldest prefix match",
			actions: func(t *testing.T, m *Model) {
				submit(t, m, "cmds filesystem")
				submit(t, m, "cmds client")
				m.input.SetValue("cmds ")
				pressKey(t, m, tea.KeyPressMsg{Code: tea.KeyUp})
				pressKey(t, m, tea.KeyPressMsg{Code: tea.KeyUp})
				pressKey(t, m, tea.KeyPressMsg{Code: tea.KeyUp})
				pressKey(t, m, tea.KeyPressMsg{Code: tea.KeyDown})
			},
			wantInput: "cmds client",
			wantHist:  []string{"cmds filesystem", "cmds client"},
		},
		{
			name: "down restores prefix draft after newest match",
			actions: func(t *testing.T, m *Model) {
				submit(t, m, "cmds filesystem")
				submit(t, m, "cmds client")
				m.input.SetValue("cmds ")
				pressKey(t, m, tea.KeyPressMsg{Code: tea.KeyUp})
				pressKey(t, m, tea.KeyPressMsg{Code: tea.KeyDown})
			},
			wantInput: "cmds ",
			wantHist:  []string{"cmds filesystem", "cmds client"},
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
			if tt.wantInLog != "" && !hasMessage(m.messages, tt.wantInLog) {
				t.Fatalf("messages missing %q: %#v", tt.wantInLog, m.messages)
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
			wantInLog: []string{"game output", "Disconnected."},
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
			wantInLog: []string{"game output", "Disconnected."},
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
			wantInLog: []string{"game output", "Connection failed: dial refused"},
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

func TestMouseDragSelect(t *testing.T) {
	t.Parallel()

	m := connectedModel(t)
	updated, _ := m.Update(intercept.GameLineMsg{Line: "alpha beta"})
	m = updated.(Model)

	updated, _ = m.Update(tea.MouseClickMsg{X: 0, Y: 0, Button: tea.MouseLeft})
	m = updated.(Model)
	updated, _ = m.Update(tea.MouseMotionMsg{X: 4, Y: 0, Button: tea.MouseLeft})
	m = updated.(Model)
	if got := m.selectionText(); got != "alpha" {
		t.Fatalf("selectionText() = %q, want %q", got, "alpha")
	}

	updated, cmd := m.Update(tea.MouseReleaseMsg{X: 4, Y: 0, Button: tea.MouseLeft})
	if cmd == nil {
		t.Fatal("expected copy cmd on drag release")
	}
	m = updated.(Model)
	if m.selActive || m.selecting {
		t.Fatal("expected selection cleared after release")
	}

	// Click without drag does not copy.
	updated, _ = m.Update(tea.MouseClickMsg{X: 9, Y: 0, Button: tea.MouseLeft})
	m = updated.(Model)
	updated, cmd = m.Update(tea.MouseReleaseMsg{X: 9, Y: 0, Button: tea.MouseLeft})
	if cmd != nil {
		t.Fatal("expected no copy cmd on click without drag")
	}
}

func TestCtrlCCancelsInput(t *testing.T) {
	t.Parallel()

	m := connectedModel(t)
	m.input.SetValue("rm logs/foo")
	updated, cmd := m.Update(ctrlKey('c'))
	model := updated.(Model)

	if !model.detachHint {
		t.Fatal("expected detachHint armed after cancelling input")
	}
	if model.input.Value() != "" {
		t.Fatalf("input = %q, want empty", model.input.Value())
	}
	if !hasMessage(model.messages, "> ") || !hasMessage(model.messages, "rm logs/foo ^C") {
		t.Fatalf("cancel not logged: %#v", model.messages)
	}
	if cmd == nil {
		t.Fatal("expected timeout cmd after first ctrl+c")
	}
}

func TestDetachHint(t *testing.T) {
	t.Parallel()

	m := connectedModel(t)
	updated, cmd := m.Update(ctrlKey('c'))
	model := updated.(Model)
	if !model.detachHint {
		t.Fatal("expected detachHint after first ctrl+c")
	}
	if cmd == nil {
		t.Fatal("expected timeout cmd on first ctrl+c")
	}

	updated, _ = m.Update(tea.KeyPressMsg{Code: 'a'})
	model = updated.(Model)
	if model.detachHint {
		t.Fatal("expected detachHint cleared after typing")
	}
}

func TestDoubleCtrlCDoesNotQuit(t *testing.T) {
	t.Parallel()

	m := connectedModel(t)
	updated, _ := m.Update(ctrlKey('c'))
	m = updated.(Model)

	updated, cmd := m.Update(ctrlKey('c'))
	if isQuitCmd(cmd) {
		t.Fatal("second ctrl+c should not quit")
	}
	model := updated.(Model)
	if !model.detachHint {
		t.Fatal("expected detachHint still armed after second ctrl+c")
	}
}

func TestCtrlADetachQuits(t *testing.T) {
	t.Parallel()

	m := connectedModel(t)
	updated, cmd := m.Update(ctrlKey('a'))
	if cmd != nil {
		t.Fatal("expected no cmd after ctrl+a")
	}
	model := updated.(Model)
	if !model.prefixArmed {
		t.Fatal("expected prefix armed after ctrl+a")
	}

	updated, cmd = model.Update(tea.KeyPressMsg{Code: 'd'})
	if !isQuitCmd(cmd) {
		t.Fatal("expected ctrl+a, d to quit")
	}
}

func TestCtrlADCtrlDDetachQuits(t *testing.T) {
	t.Parallel()

	m := connectedModel(t)
	updated, _ := m.Update(ctrlKey('a'))
	m = updated.(Model)

	updated, cmd := m.Update(ctrlKey('d'))
	if !isQuitCmd(cmd) {
		t.Fatal("expected ctrl+a, ctrl+d to quit")
	}
}

func TestDetachHintTimeout(t *testing.T) {
	t.Parallel()

	m := connectedModel(t)
	updated, _ := m.Update(ctrlKey('c'))
	m = updated.(Model)

	// A stale timer (older seq) must not clear a re-armed hint.
	updated, _ = m.Update(detachHintTimeoutMsg{seq: m.detachHintSeq - 1})
	m = updated.(Model)
	if !m.detachHint {
		t.Fatal("stale timeout should not clear detachHint")
	}

	// The matching timer resets the hint.
	updated, _ = m.Update(detachHintTimeoutMsg{seq: m.detachHintSeq})
	m = updated.(Model)
	if m.detachHint {
		t.Fatal("expected detachHint cleared after timeout")
	}

	// After the reset, ctrl+c arms again.
	updated, _ = m.Update(ctrlKey('c'))
	m = updated.(Model)
	if !m.detachHint {
		t.Fatal("expected detachHint re-armed after timeout reset")
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

func isQuitCmd(cmd tea.Cmd) bool {
	if cmd == nil {
		return false
	}
	_, ok := cmd().(tea.QuitMsg)
	return ok
}

func closedStringCh() <-chan string {
	ch := make(chan string, 1)
	close(ch)
	return ch
}

type errTest string

func (e errTest) Error() string { return string(e) }
