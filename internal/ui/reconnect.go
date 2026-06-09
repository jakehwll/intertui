package ui

import (
	tea "charm.land/bubbletea/v2"

	filelog "intertui/internal/log"
)

func (m *Model) beginReconnect() []tea.Cmd {
	if m.client != nil {
		m.client.Close()
		m.client = nil
	}

	m.reconnecting = true
	m.state = stateConnecting
	m.historyIndex = -1
	m.historyDraft = ""
	m.input.Blur()
	m.input.SetValue("")
	m.appendStatus("Reconnecting…")
	filelog.Info("reconnect target=%s", m.cfg.DialDescription())

	return []tea.Cmd{startClient(m.cfg)}
}
