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

func TestMouseSelection(t *testing.T) {
	t.Parallel()

	m := connectedModel(t)
	for _, line := range []string{"alpha beta", "gamma delta"} {
		updated, _ := m.Update(intercept.GameLineMsg{Line: line})
		m = updated.(Model)
	}

	updated, _ := m.Update(tea.MouseClickMsg{X: 0, Y: 0, Button: tea.MouseLeft})
	m = updated.(Model)
	if !m.selecting {
		t.Fatal("expected selecting after left click in log")
	}

	updated, _ = m.Update(tea.MouseMotionMsg{X: 4, Y: 0, Button: tea.MouseLeft})
	m = updated.(Model)
	if !m.selActive {
		t.Fatal("expected active selection after drag")
	}
	if got := m.selectionText(); got != "alpha" {
		t.Fatalf("selectionText() = %q, want %q", got, "alpha")
	}

	updated, cmd := m.Update(tea.MouseReleaseMsg{X: 4, Y: 0, Button: tea.MouseLeft})
	m = updated.(Model)
	if cmd == nil {
		t.Fatal("expected copy cmd on release")
	}
	if m.selecting {
		t.Fatal("expected selecting=false after release")
	}
	if !m.selActive {
		t.Fatal("expected selection to stay visible after release")
	}
	if !m.copied {
		t.Fatal("expected copied flag after release")
	}

	// Esc clears the selection instead of quitting.
	updated, cmd = m.Update(tea.KeyPressMsg{Code: tea.KeyEscape})
	m = updated.(Model)
	if m.selActive {
		t.Fatal("expected esc to clear selection")
	}
	if cmd != nil {
		t.Fatal("expected esc with selection to not quit")
	}
}

func TestMouseSelectionBottomLine(t *testing.T) {
	t.Parallel()

	m := connectedModel(t)
	// 24-row terminal, 3 chrome rows -> 21 log rows. Fill past one screen so
	// the viewport scrolls and the last line sits on the bottom rows.
	for i := 0; i < 30; i++ {
		updated, _ := m.Update(intercept.GameLineMsg{Line: strings.Repeat("x", 5) + " line"})
		m = updated.(Model)
	}

	lastRow := m.viewport.Height() - 1
	updated, _ := m.Update(tea.MouseClickMsg{X: 0, Y: lastRow - 1, Button: tea.MouseLeft})
	m = updated.(Model)
	// Drag below the viewport into the chrome; must clamp to the last line.
	updated, _ = m.Update(tea.MouseMotionMsg{X: 9, Y: lastRow + 2, Button: tea.MouseLeft})
	m = updated.(Model)

	want := "xxxxx line\nxxxxx line"
	if got := m.selectionText(); got != want {
		t.Fatalf("selectionText() = %q, want %q", got, want)
	}
}

func TestNewLinesDoNotScrollDuringSelection(t *testing.T) {
	t.Parallel()

	m := connectedModel(t)
	for i := 0; i < 30; i++ {
		updated, _ := m.Update(intercept.GameLineMsg{Line: "old line"})
		m = updated.(Model)
	}
	offBefore := m.viewport.YOffset()

	updated, _ := m.Update(tea.MouseClickMsg{X: 0, Y: 0, Button: tea.MouseLeft})
	m = updated.(Model)
	updated, _ = m.Update(tea.MouseMotionMsg{X: 3, Y: 1, Button: tea.MouseLeft})
	m = updated.(Model)

	// New output mid-drag must not yank the viewport to the bottom.
	updated, _ = m.Update(intercept.GameLineMsg{Line: "new line"})
	m = updated.(Model)
	if got := m.viewport.YOffset(); got != offBefore {
		t.Fatalf("YOffset = %d during drag, want %d (no auto-scroll)", got, offBefore)
	}

	// After release, auto-follow resumes on the next line.
	updated, _ = m.Update(tea.MouseReleaseMsg{X: 3, Y: 1, Button: tea.MouseLeft})
	m = updated.(Model)
	updated, _ = m.Update(intercept.GameLineMsg{Line: "after release"})
	m = updated.(Model)
	if got := m.viewport.YOffset(); got == offBefore {
		t.Fatal("expected auto-follow to resume after release")
	}
}

func TestMouseSelectionMultiline(t *testing.T) {
	t.Parallel()

	m := connectedModel(t)
	for _, line := range []string{"alpha beta", "gamma delta"} {
		updated, _ := m.Update(intercept.GameLineMsg{Line: line})
		m = updated.(Model)
	}

	updated, _ := m.Update(tea.MouseClickMsg{X: 6, Y: 0, Button: tea.MouseLeft})
	m = updated.(Model)
	updated, _ = m.Update(tea.MouseMotionMsg{X: 4, Y: 1, Button: tea.MouseLeft})
	m = updated.(Model)

	if got, want := m.selectionText(), "beta\ngamma"; got != want {
		t.Fatalf("selectionText() = %q, want %q", got, want)
	}
}

func TestQuitConfirm(t *testing.T) {
	t.Parallel()

	m := connectedModel(t)
	updated, cmd := m.Update(ctrlKey('c'))
	model := updated.(Model)
	if !model.quitConfirm {
		t.Fatal("expected quitConfirm after first ctrl+c")
	}
	if cmd == nil {
		t.Fatal("expected timeout cmd on first ctrl+c")
	}

	updated, _ = m.Update(tea.KeyPressMsg{Code: 'a'})
	model = updated.(Model)
	if model.quitConfirm {
		t.Fatal("expected quitConfirm cleared after typing")
	}
}

func TestQuitConfirmTimeout(t *testing.T) {
	t.Parallel()

	m := connectedModel(t)
	updated, _ := m.Update(ctrlKey('c'))
	m = updated.(Model)

	// A stale timer (older seq) must not clear a re-armed confirm.
	updated, _ = m.Update(quitConfirmTimeoutMsg{seq: m.quitConfirmSeq - 1})
	m = updated.(Model)
	if !m.quitConfirm {
		t.Fatal("stale timeout should not clear quitConfirm")
	}

	// The matching timer resets the confirmation.
	updated, _ = m.Update(quitConfirmTimeoutMsg{seq: m.quitConfirmSeq})
	m = updated.(Model)
	if m.quitConfirm {
		t.Fatal("expected quitConfirm cleared after timeout")
	}

	// After the reset, ctrl+c arms again instead of quitting.
	updated, _ = m.Update(ctrlKey('c'))
	m = updated.(Model)
	if !m.quitConfirm {
		t.Fatal("expected quitConfirm re-armed after timeout reset")
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
