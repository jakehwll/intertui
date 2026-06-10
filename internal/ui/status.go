package ui

import tea "charm.land/bubbletea/v2"

func (m *Model) maybeStartClient() tea.Cmd {
	if m.state == stateConnecting && !m.clientStarted && m.ready && m.width > 0 && m.height > 0 {
		m.clientStarted = true
		return startClient(m.cfg)
	}
	return nil
}

func (m *Model) appendStatus(line string) {
	m.messages = append(m.messages, clientLine("› "+line))
	m.updateViewport()
}
