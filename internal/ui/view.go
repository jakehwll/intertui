package ui

import (
	"strings"

	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
	"github.com/charmbracelet/x/ansi"
)

var (
	dim      = lipgloss.NewStyle().Foreground(lipgloss.Color("240"))
	selStyle = lipgloss.NewStyle().Reverse(true)
)

// View implements tea.Model.
func (m Model) View() tea.View {
	v := tea.NewView("")
	v.AltScreen = true
	// Wheel scrolls the log; without this, many terminals map scroll to ↑/↓ (history).
	v.MouseMode = tea.MouseModeCellMotion
	if !m.ready {
		return v
	}

	w := max(1, m.width)
	v.SetContent(lipgloss.JoinVertical(lipgloss.Left, m.logView(), m.chrome(w)))
	return v
}

// logView renders the log area, overlaying the in-app mouse selection.
func (m Model) logView() string {
	view := m.viewport.View()
	if !m.selActive && !m.selecting {
		return view
	}

	x0, y0, x1, y1 := m.selBounds()
	rows := strings.Split(view, "\n")
	off := m.viewport.YOffset()
	for i, row := range rows {
		abs := off + i
		if abs < y0 || abs > y1 {
			continue
		}
		width := ansi.StringWidth(row)
		from, to := 0, width
		if abs == y0 {
			from = x0
		}
		if abs == y1 {
			to = min(to, x1+1)
		}
		if from >= to {
			continue
		}
		rows[i] = ansi.Cut(row, 0, from) +
			selStyle.Render(ansi.Strip(ansi.Cut(row, from, to))) +
			ansi.Cut(row, to, width)
	}
	return strings.Join(rows, "\n")
}

func (m Model) chrome(w int) string {
	row := lipgloss.NewStyle().Width(w)

	var input string
	switch m.state {
	case stateConnected:
		input = m.input.View()
	case stateError:
		input = dim.Render("press r to reconnect")
	default:
		input = dim.Render("connecting…")
	}

	return lipgloss.JoinVertical(lipgloss.Left,
		row.Render(""),
		row.Render(input),
		row.Render(m.footer(w)),
	)
}

func (m Model) footer(w int) string {
	if m.quitConfirm {
		return lipgloss.NewStyle().Width(w).Render(dim.Render("Press Ctrl+C again to quit!"))
	}

	status := m.statusStyle().Render("● " + m.connectionStatus())

	var left string
	switch {
	case m.copied:
		left = dim.Render("copied selection")
	default:
		if user := m.displayUser(); user != "" {
			left = dim.Render(user)
		}
	}

	if left == "" {
		return lipgloss.NewStyle().Width(w).Align(lipgloss.Right).Render(status)
	}
	return lipgloss.NewStyle().Width(w).Render(
		left + strings.Repeat(" ", max(1,
			w-lipgloss.Width(left)-lipgloss.Width(status),
		)) + status,
	)
}

func (m Model) connectionStatus() string {
	switch m.state {
	case stateConnecting:
		return "Connecting"
	case stateConnected:
		return "Connected"
	default:
		return "Offline"
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

func (m Model) chromeHeight() int {
	return lipgloss.Height(m.chrome(max(1, m.width)))
}

func (m *Model) layout() {
	w := max(1, m.width)
	chromeH := m.chromeHeight()
	m.viewport.SetWidth(w)
	m.viewport.SetHeight(max(0, m.height-chromeH))
	m.input.SetWidth(max(0, w-lipgloss.Width(m.input.Prompt)))
}
