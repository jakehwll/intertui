package ui

import "charm.land/lipgloss/v2"

var clientStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("240"))

func clientLine(s string) string {
	return clientStyle.Render(s)
}
