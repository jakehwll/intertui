package ui

import (
	"strings"

	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
)

// View implements tea.Model.
func (m Model) View() tea.View {
	v := tea.NewView("\n  loading...")
	v.AltScreen = true

	if !m.ready {
		return v
	}

	border := lipgloss.NormalBorder()
	innerW, _ := m.innerSize()

	var inputPanel string
	if m.state == stateConnected {
		inputPanel = lipgloss.NewStyle().
			Width(innerW).
			Border(border).
			BorderLeft(false).
			BorderRight(false).
			BorderBottom(false).
			Render(m.input.View())
	} else {
		hint := "…"
		switch m.state {
		case stateError:
			hint = "press r to reconnect"
		case stateConnecting:
			hint = "connecting…"
		}
		inputPanel = lipgloss.NewStyle().
			Width(innerW).
			Border(border).
			BorderLeft(false).
			BorderRight(false).
			BorderBottom(false).
			Render(clientLine(hint))
	}

	inner := lipgloss.JoinVertical(
		lipgloss.Left,
		m.headerPanel(innerW),
		m.viewport.View(),
		inputPanel,
	)

	v.SetContent(m.boxStyle().Render(inner))
	v.MouseMode = tea.MouseModeCellMotion

	return v
}

func (m Model) boxStyle() lipgloss.Style {
	return lipgloss.NewStyle().
		Width(m.width).
		Height(m.height).
		Border(lipgloss.NormalBorder())
}

func (m Model) innerSize() (width, height int) {
	style := m.boxStyle()
	return max(0, m.width-style.GetHorizontalFrameSize()),
		max(0, m.height-style.GetVerticalFrameSize())
}

func (m *Model) layout() {
	innerW, innerH := m.innerSize()
	messageInnerHeight := max(0, innerH-headerRows-inputRows-2)

	m.viewport.SetWidth(innerW)
	m.viewport.SetHeight(messageInnerHeight)
	m.input.SetWidth(max(0, innerW-lipgloss.Width(m.input.Prompt)))
}

func (m *Model) updateViewport() {
	atBottom := m.viewport.AtBottom()
	m.viewport.SetContent(strings.Join(m.messages, "\n"))
	if atBottom {
		m.viewport.GotoBottom()
	}
}
