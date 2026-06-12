package ui

import (
	"strings"

	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
	"github.com/charmbracelet/x/ansi"
)

var (
	dim         = lipgloss.NewStyle().Foreground(lipgloss.Color("240"))
	hintStyle   = lipgloss.NewStyle().Foreground(lipgloss.Color("242"))
	selStyle    = lipgloss.NewStyle().Reverse(true)
	footerStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("0")).
			Background(lipgloss.Color("7"))
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
	v.SetContent(lipgloss.JoinVertical(lipgloss.Left, m.footer(w), "", m.logView(), m.chrome(w)))
	return v
}

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
		m.chromeHint(),
	)
}

func (m Model) chromeHint() string {
	var hint string
	switch m.state {
	case stateConnected:
		hint = "enter send · tab complete · ↑/↓ history · drag select · ctrl+a,d detach · esc quit"
	case stateError:
		hint = "r reconnect · esc quit"
	default:
		hint = "esc quit"
	}
	return hintStyle.Render(hint)
}

func (m Model) footer(w int) string {
	var content string
	switch {
	case m.prefixArmed:
		content = "Press D to detach"
	case m.detachHint:
		content = "Press Ctrl+A, D to detach"
	default:
		status := "● " + m.connectionStatus()
		left := m.displayUser()
		if left == "" {
			return footerStyle.Width(w).Align(lipgloss.Right).Render(status)
		}
		content = left + strings.Repeat(" ", max(1,
			w-ansi.StringWidth(left)-ansi.StringWidth(status),
		)) + status
	}
	return footerStyle.Width(w).Render(content)
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

func (m Model) chromeHeight() int {
	w := max(1, m.width)
	return lipgloss.Height(m.footer(w)) + 1 + lipgloss.Height(m.chrome(w))
}

func (m *Model) layout() {
	w := max(1, m.width)
	chromeH := m.chromeHeight()
	m.viewport.SetWidth(w)
	m.viewport.SetHeight(max(0, m.height-chromeH))
	m.input.SetWidth(max(0, w-lipgloss.Width(m.input.Prompt)))
}
