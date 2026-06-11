//go:build js && wasm

package ui

import (
	"time"

	"charm.land/bubbles/v2/textinput"
	tea "charm.land/bubbletea/v2"
)

type deferredStartMsg struct{}

func (m Model) programInit() tea.Cmd {
	return tea.Batch(
		textinput.Blink,
		tea.Tick(200*time.Millisecond, func(time.Time) tea.Msg { return deferredStartMsg{} }),
	)
}

func (m *Model) maybeStartClient() tea.Cmd {
	if m.state == stateConnecting && !m.clientStarted && m.ready && m.width > 0 && m.height > 0 {
		m.clientStarted = true
		return startClient(m.cfg, m.newClient)
	}
	return nil
}
