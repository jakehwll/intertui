package ui

func (m *Model) appendStatus(line string) {
	m.messages = append(m.messages, clientLine("› "+line))
	m.updateViewport()
}
