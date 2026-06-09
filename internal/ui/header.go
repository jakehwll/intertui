package ui

import (
	"strings"

	"charm.land/lipgloss/v2"
)

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

func (m Model) displayUser() string {
	if m.connectedUser != "" {
		return m.connectedUser
	}
	return m.cfg.User
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
	status := m.statusStyle().Render("● " + m.connectionStatus())

	var line string
	if user := m.displayUser(); user != "" {
		userStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("245"))
		gap := innerW - lipgloss.Width(userStyle.Render(user)) - lipgloss.Width(status)
		if gap < 1 {
			gap = 1
		}
		line = userStyle.Render(user) + strings.Repeat(" ", gap) + status
	} else {
		line = status
	}

	return lipgloss.NewStyle().
		Width(innerW).
		Border(border).
		BorderLeft(false).
		BorderRight(false).
		BorderTop(false).
		BorderBottom(true).
		Render(line)
}
