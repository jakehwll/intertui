package auth

import (
	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
)

const logo = `▗▄▄▄▖▗▖  ▗▖▗▄▄▄▖▗▄▄▄▖▗▄▄▖ ▗▄▄▄▖▗▖ ▗▖▗▄▄▄▖
  █  ▐▛▚▖▐▌  █  ▐▌   ▐▌ ▐▌  █  ▐▌ ▐▌  █  
  █  ▐▌ ▝▜▌  █  ▐▛▀▀▘▐▛▀▚▖  █  ▐▌ ▐▌  █  
▗▄█▄▖▐▌  ▐▌  █  ▐▙▄▄▖▐▌ ▐▌  █  ▝▚▄▞▘▗▄█▄▖`

var (
	titleStyle   = lipgloss.NewStyle().Bold(true)
	dim          = lipgloss.NewStyle().Foreground(lipgloss.Color("240"))
	errStyle     = lipgloss.NewStyle().Foreground(lipgloss.Color("203"))
	hintStyle    = lipgloss.NewStyle().Foreground(lipgloss.Color("242"))
	activeItem   = lipgloss.NewStyle().Bold(true)
	idleItem     = lipgloss.NewStyle()
	menuSelected = lipgloss.NewStyle().
			Bold(true).
			Foreground(lipgloss.Color("0")).
			Background(lipgloss.Color("7"))
	menuIdle     = lipgloss.NewStyle()
	checkboxMark = lipgloss.NewStyle().
			Foreground(lipgloss.Color("0")).
			Background(lipgloss.Color("7"))
)

// View implements tea.Model.
func (m Model) View() tea.View {
	v := tea.NewView("")
	v.AltScreen = true
	if !m.ready {
		return v
	}

	var body string
	switch m.screen {
	case screenMenu:
		body = m.viewMenu()
	case screenLogin:
		body = m.viewForm("Login", "Save credentials and connect.")
	case screenRegister:
		body = m.viewForm("Register", "Create an Intercept account on the server.")
	case screenBusy:
		body = m.viewBusy()
	}

	v.MouseMode = tea.MouseModeCellMotion
	v.SetContent(renderCentered(body, m.width, m.height))
	return v
}

func (m Model) viewMenu() string {
	labels := []struct {
		title, desc string
	}{
		{"Login", "Use an existing account"},
		{"Register", "Create a new account"},
		{"Quit", "Exit without connecting"},
	}

	items := make([]string, 0, len(labels)*2-1)
	for i, item := range labels {
		cursor := "  "
		titleStyle := menuIdle
		descStyle := dim
		if menuChoice(i) == m.menuCursor {
			cursor = "› "
			titleStyle = menuSelected
			descStyle = menuSelected
		}
		entry := lipgloss.JoinVertical(lipgloss.Left,
			titleStyle.Render(cursor+item.title),
			descStyle.Render("  "+item.desc),
		)
		if i > 0 {
			items = append(items, "")
		}
		items = append(items, entry)
	}

	menu := lipgloss.JoinVertical(lipgloss.Left, items...)
	return menuShell(
		menu,
		"",
		"",
		menuFooterHints(),
	)
}

func (m Model) viewForm(title, subtitle string) string {
	fields := []struct {
		label string
		view  string
	}{
		{"Server", m.server.View()},
		{"Username", m.user.View()},
		{"Password", m.pass.View()},
	}

	items := make([]string, 0, len(fields)*2)
	for i, f := range fields {
		label := idleItem.Render(f.label)
		if i == m.focus {
			label = activeItem.Render(f.label)
		}
		entry := lipgloss.JoinVertical(lipgloss.Left, label, f.view)
		if i > 0 {
			items = append(items, "")
		}
		items = append(items, entry)
	}
	items = append(items, "", m.saveCredsView())

	header := lipgloss.JoinVertical(lipgloss.Left,
		titleStyle.Render(title),
		dim.Render(subtitle),
	)
	form := lipgloss.JoinVertical(lipgloss.Left, items...)

	parts := []string{header, "", form}
	if m.errMsg != "" {
		parts = append(parts, "", errStyle.Render(m.errMsg))
	}
	parts = append(parts, "", "", formFooterHints())
	return subShell(parts...)
}

func menuFooterHints() string {
	return lipgloss.JoinVertical(lipgloss.Left,
		hintStyle.Render("↑/↓ select · click or enter"),
		hintStyle.Render("q quit"),
	)
}

func formFooterHints() string {
	return lipgloss.JoinVertical(lipgloss.Left,
		hintStyle.Render("click to focus · tab next"),
		hintStyle.Render("space toggle · enter submit · esc back"),
	)
}

func (m Model) saveCredsView() string {
	mark := "[ ]"
	if m.saveCreds {
		mark = "[x]"
	}
	label := " Save credentials"
	if m.focus == 3 {
		return checkboxMark.Render(mark) + activeItem.Render(label)
	}
	return checkboxMark.Render(mark) + idleItem.Render(label)
}

func (m Model) viewBusy() string {
	body := lipgloss.JoinVertical(lipgloss.Left,
		titleStyle.Render("Please wait"),
		dim.Render(m.busyMsg),
		"",
		m.spinner.View(),
	)
	return subShell(body)
}

func menuShell(parts ...string) string {
	content := append([]string{logo, "", ""}, parts...)
	return lipgloss.JoinVertical(lipgloss.Left, content...)
}

func subShell(parts ...string) string {
	return lipgloss.JoinVertical(lipgloss.Left, parts...)
}
