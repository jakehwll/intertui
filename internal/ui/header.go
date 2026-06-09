package ui

import "charm.land/lipgloss/v2"

const headerRows = 1

func (m Model) connectionStatus() string {
	switch m.state {
	case stateConnecting:
		return "Connecting"
	case stateConnected:
		return "Connected"
	default:
		return "Not connected"
	}
}

func (m Model) statusStyle() lipgloss.Style {
	switch m.state {
	case stateConnecting:
		return lipgloss.NewStyle().Foreground(lipgloss.Color("214"))
	case stateConnected:
		return lipgloss.NewStyle().Foreground(lipgloss.Color("42"))
	default:
		return lipgloss.NewStyle().Foreground(lipgloss.Color("203"))
	}
}

func (m Model) headerPanel(innerW int) string {
	border := lipgloss.NormalBorder()
	line := m.statusStyle().Render("● " + m.connectionStatus())

	return lipgloss.NewStyle().
		Width(innerW).
		Align(lipgloss.Right).
		Border(border).
		BorderLeft(false).
		BorderRight(false).
		BorderTop(false).
		BorderBottom(true).
		Render(line)
}
