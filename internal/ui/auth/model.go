package auth

import (
	"context"
	"strings"

	"charm.land/bubbles/v2/spinner"
	"charm.land/bubbles/v2/textinput"
	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"

	"intertui/internal/config"
	"intertui/internal/constants"
	"intertui/internal/intercept"
)

const formFields = 4 // server, user, pass, save checkbox

type screen int

const (
	screenMenu screen = iota
	screenLogin
	screenRegister
	screenBusy
)

type menuChoice int

const (
	choiceLogin menuChoice = iota
	choiceRegister
	choiceQuit
)

// Model is the Bubble Tea model for the auth flow.
type Model struct {
	screen screen
	seed   config.Config

	width  int
	height int
	ready  bool

	menuCursor menuChoice
	server     textinput.Model
	user       textinput.Model
	pass       textinput.Model
	focus      int
	saveCreds  bool

	errMsg  string
	busyMsg string
	spinner spinner.Model

	done   bool
	quit   bool
	result config.Config
}

// New returns the initial auth model, pre-filling fields from seed config.
func New(seed config.Config) Model {
	server := newField("server host", seed.Server)
	user := newField("username", seed.User)
	pass := newField("password", seed.Pass)
	pass.EchoMode = textinput.EchoPassword
	pass.EchoCharacter = '•'

	s := spinner.New()
	s.Spinner = spinner.Dot
	s.Style = dim

	m := Model{
		screen:    screenMenu,
		seed:      seed,
		server:    server,
		user:      user,
		pass:      pass,
		saveCreds: true,
		spinner: s,
	}
	m.layoutFormFields()
	return m
}

func (m *Model) layoutFormFields() {
	w := fieldInputWidth()
	m.server.SetWidth(w)
	m.user.SetWidth(w)
	m.pass.SetWidth(w)
}

func fieldInputWidth() int {
	styles := invertedFieldStyles()
	promptW := lipgloss.Width(styles.Focused.Prompt.Render("> "))
	return max(1, formColumnWidth()-promptW)
}

func newField(placeholder, value string) textinput.Model {
	ti := textinput.New()
	ti.Placeholder = placeholder
	ti.CharLimit = 120
	ti.SetValue(value)
	ti.SetStyles(invertedFieldStyles())
	return ti
}

func invertedFieldStyles() textinput.Styles {
	styles := textinput.DefaultDarkStyles()
	styles.Cursor.Blink = true
	styles.Cursor.Color = lipgloss.Color("0")

	inverted := lipgloss.NewStyle().
		Foreground(lipgloss.Color("0")).
		Background(lipgloss.Color("7"))
	dimInverted := lipgloss.NewStyle().
		Foreground(lipgloss.Color("238")).
		Background(lipgloss.Color("7"))

	styles.Focused.Text = inverted
	styles.Focused.Placeholder = dimInverted
	styles.Focused.Prompt = inverted
	styles.Blurred.Text = inverted
	styles.Blurred.Placeholder = dimInverted
	styles.Blurred.Prompt = inverted
	return styles
}

// Needs reports whether the auth screens should run before connecting.
func Needs(cfg config.Config) bool {
	return !cfg.HasCreds() || (cfg.Server == "" && cfg.URL == "")
}

// Run shows the auth TUI. The second return value is true when auth completed.
func Run(seed config.Config) (config.Config, bool, error) {
	p := tea.NewProgram(New(seed), tea.WithFPS(30))
	final, err := p.Run()
	if err != nil {
		return config.Config{}, false, err
	}
	m := final.(Model)
	if m.quit || !m.done {
		return config.Config{}, false, nil
	}
	return m.result, true, nil
}

func (m Model) Init() tea.Cmd {
	return nil
}

func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		m.ready = true
		m.layoutFormFields()
		return m, nil

	case tea.KeyPressMsg:
		if m.screen == screenBusy {
			if keyStroke(msg) == "ctrl+c" {
				return m, tea.Quit
			}
			return m, nil
		}

		switch keyStroke(msg) {
		case "ctrl+c", "q":
			if m.screen == screenMenu {
				m.quit = true
				return m, tea.Quit
			}
			m.errMsg = ""
			m.screen = screenMenu
			m.blurForm()
			return m, nil
		case "esc":
			if m.screen == screenMenu {
				m.quit = true
				return m, tea.Quit
			}
			m.errMsg = ""
			m.screen = screenMenu
			m.blurForm()
			return m, nil
		}

	case registerDoneMsg:
		if msg.err != nil {
			m.screen = screenRegister
			m.errMsg = msg.err.Error()
			m.focusForm(2)
			return m, nil
		}
		m.result = msg.cfg
		m.done = true
		return m, tea.Quit

	case spinner.TickMsg:
		if m.screen == screenBusy {
			var cmd tea.Cmd
			m.spinner, cmd = m.spinner.Update(msg)
			return m, cmd
		}
	}

	switch m.screen {
	case screenMenu:
		return m.updateMenu(msg)
	case screenLogin, screenRegister:
		return m.updateForm(msg)
	case screenBusy:
		var cmd tea.Cmd
		m.spinner, cmd = m.spinner.Update(msg)
		return m, cmd
	}

	return m, nil
}

func (m Model) updateMenu(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.MouseClickMsg:
		if msg.Button != tea.MouseLeft {
			return m, nil
		}
		if choice, ok := m.menuChoiceAt(msg.X, msg.Y); ok {
			return m.activateMenuChoice(choice)
		}
		return m, nil

	case tea.KeyPressMsg:
		switch keyStroke(msg) {
		case "up", "k":
			if m.menuCursor > choiceLogin {
				m.menuCursor--
			}
		case "down", "j":
			if m.menuCursor < choiceQuit {
				m.menuCursor++
			}
		case "enter":
			return m.activateMenuChoice(m.menuCursor)
		}
	}
	return m, nil
}

func (m Model) activateMenuChoice(choice menuChoice) (tea.Model, tea.Cmd) {
	m.menuCursor = choice
	switch choice {
	case choiceLogin:
		m.screen = screenLogin
		m.errMsg = ""
		m.focusForm(0)
	case choiceRegister:
		m.screen = screenRegister
		m.errMsg = ""
		m.focusForm(0)
	case choiceQuit:
		m.quit = true
		return m, tea.Quit
	}
	return m, nil
}

func (m Model) formBody() string {
	switch m.screen {
	case screenLogin:
		return m.viewForm("Login", "Save credentials and connect.")
	case screenRegister:
		return m.viewForm("Register", "Create an Intercept account on the server.")
	default:
		return ""
	}
}

func (m Model) updateForm(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.MouseClickMsg:
		if msg.Button != tea.MouseLeft {
			return m, nil
		}
		body := m.formBody()
		field, ok := formFieldAt(body, m.width, m.height, msg.X, msg.Y)
		if !ok {
			return m, nil
		}
		if field == 3 {
			m.saveCreds = !m.saveCreds
		}
		m.focusForm(field)
		return m, nil
	}

	key, ok := msg.(tea.KeyPressMsg)
	if ok {
		switch keyStroke(key) {
		case "tab", "down":
			m.focusForm((m.focus + 1) % formFields)
			return m, nil
		case "shift+tab", "up":
			m.focusForm((m.focus + formFields - 1) % formFields)
			return m, nil
		case "space":
			if m.focus == 3 {
				m.saveCreds = !m.saveCreds
				return m, nil
			}
		case "enter":
			if m.focus < formFields-1 {
				m.focusForm(m.focus + 1)
				return m, nil
			}
			return m.submitForm()
		}
	}

	if m.focus > 2 {
		return m, nil
	}

	var cmd tea.Cmd
	switch m.focus {
	case 0:
		m.server, cmd = m.server.Update(msg)
	case 1:
		m.user, cmd = m.user.Update(msg)
	case 2:
		m.pass, cmd = m.pass.Update(msg)
	}
	return m, cmd
}

func (m Model) submitForm() (tea.Model, tea.Cmd) {
	server := trim(m.server.Value())
	user := trim(m.user.Value())
	pass := m.pass.Value()

	if server == "" {
		m.errMsg = "server is required"
		m.focusForm(0)
		return m, nil
	}
	if user == "" || pass == "" {
		m.errMsg = "username and password are required"
		m.focusForm(1)
		return m, nil
	}

	m.errMsg = ""

	switch m.screen {
	case screenLogin:
		cfg, err := saveLogin(server, user, pass, m.saveCreds)
		if err != nil {
			m.errMsg = err.Error()
			return m, nil
		}
		m.result = cfg
		m.done = true
		return m, tea.Quit

	case screenRegister:
		m.screen = screenBusy
		m.busyMsg = "Creating account…"
		m.blurForm()
		return m, tea.Batch(m.spinner.Tick, doRegister(server, user, pass, m.saveCreds))
	}

	return m, nil
}

func saveLogin(server, user, pass string, persist bool) (config.Config, error) {
	cfg := resultConfig(server, user, pass)
	if !persist {
		return cfg, nil
	}
	_, err := config.RunInit(config.InitOptions{
		Server: server,
		Port:   constants.DEFAULT_PORT,
		User:   user,
		Pass:   pass,
		Force:  true,
	})
	if err != nil {
		return config.Config{}, err
	}
	return cfg, nil
}

func resultConfig(server, user, pass string) config.Config {
	return config.Config{
		Server: server,
		Port:   constants.DEFAULT_PORT,
		User:   user,
		Pass:   pass,
	}
}

type registerDoneMsg struct {
	cfg config.Config
	err error
}

func doRegister(server, user, pass string, persist bool) tea.Cmd {
	return func() tea.Msg {
		cfg := resultConfig(server, user, pass)
		if persist {
			_, _, err := config.RunRegister(context.Background(), config.RegisterOptions{
				Server: server,
				Port:   constants.DEFAULT_PORT,
				User:   user,
				Pass:   pass,
				Force:  true,
			})
			if err != nil {
				return registerDoneMsg{err: err}
			}
			return registerDoneMsg{cfg: cfg}
		}

		_, err := intercept.Register(context.Background(), cfg.ResolveAddr(), user, pass)
		if err != nil {
			return registerDoneMsg{err: err}
		}
		return registerDoneMsg{cfg: cfg}
	}
}

func (m *Model) focusForm(i int) {
	m.focus = i
	m.server.Blur()
	m.user.Blur()
	m.pass.Blur()
	switch i {
	case 0:
		m.server.Focus()
	case 1:
		m.user.Focus()
	case 2:
		m.pass.Focus()
	}
}

func (m *Model) blurForm() {
	m.server.Blur()
	m.user.Blur()
	m.pass.Blur()
}

func trim(s string) string {
	return strings.TrimSpace(s)
}

func keyStroke(msg tea.KeyPressMsg) string {
	return msg.Keystroke()
}
