package ui

import (
	"strings"

	"charm.land/bubbles/v2/textinput"
	"charm.land/bubbles/v2/viewport"
	tea "charm.land/bubbletea/v2"

	"intertui/internal/config"
	"intertui/internal/intercept"
)

const inputRows = 1

type connState int

const (
	stateConnecting connState = iota
	stateConnected
	stateError
)

// Model is the Bubble Tea model for the terminal UI.
type Model struct {
	cfg    config.Config
	client *intercept.Client

	messages []string
	history  []string

	viewport viewport.Model
	input    textinput.Model

	state          connState
	connectedUser  string

	width        int
	height       int
	ready        bool
	historyIndex int
	historyDraft string
}

// New returns the initial UI model.
func New(cfg config.Config) Model {
	ti := textinput.New()
	ti.Placeholder = "type a command..."
	ti.CharLimit = 280

	m := Model{
		cfg:      cfg,
		messages: []string{clientLine("Intercept terminal")},
		input:    ti,
		state:    stateConnecting,
	}

	if cfg.Offline && !cfg.HasCreds() {
		m.cfg.User = "offline"
		m.cfg.Pass = "offline"
	}
	m.appendStatus("Target: " + cfg.DialDescription())

	return m
}

// Init implements tea.Model.
func (m Model) Init() tea.Cmd {
	cmds := []tea.Cmd{textinput.Blink}

	if m.state == stateConnecting {
		cmds = append(cmds, startClient(m.cfg))
	}

	return tea.Batch(cmds...)
}

// Update implements tea.Model.
func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmds []tea.Cmd

	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height

		if !m.ready {
			m.viewport = viewport.New()
			m.viewport.KeyMap = scrollKeyMap()
			m.viewport.SoftWrap = true
			m.viewport.MouseWheelEnabled = true
			m.viewport.MouseWheelDelta = 1
			m.ready = true
		}

		m.layout()
		m.updateViewport()

	case connectProgressMsg:
		m.appendStatus(msg.line)
		return m, pollConnect(msg.statusCh, msg.doneCh)

	case clientReadyMsg:
		if msg.err != nil {
			m.state = stateError
			m.messages = append(m.messages, clientLine("Connection failed: "+msg.err.Error()))
			m.updateViewport()
			break
		}
		m.client = msg.client
		m.connectedUser = msg.user
		m.state = stateConnected
		m.messages = nil
		m.input.Focus()
		m.updateViewport()
		m.viewport.GotoBottom()
		cmds = append(cmds, waitClientMsg(m.client))

	case intercept.GameLineMsg:
		m.messages = append(m.messages, msg.Line)
		m.updateViewport()
		if m.client != nil {
			cmds = append(cmds, waitClientMsg(m.client))
		}

	case intercept.DisconnectedMsg:
		if m.state == stateConnected {
			m.state = stateError
			line := "Disconnected."
			if msg.Err != nil {
				line = "Disconnected: " + msg.Err.Error()
			}
			m.messages = append(m.messages, clientLine(line))
			m.updateViewport()
		}

	case tea.KeyPressMsg:
		switch msg.String() {
		case "ctrl+c", "esc":
			if m.client != nil {
				m.client.Close()
			}
			return m, tea.Quit

		case "ctrl+p", "ctrl+up":
			if m.state == stateConnected {
				m.historyUp()
				return m, nil
			}

		case "ctrl+n", "ctrl+down":
			if m.state == stateConnected {
				m.historyDown()
				return m, nil
			}

		case "enter":
			if m.state == stateConnected {
				if value := strings.TrimSpace(m.input.Value()); value != "" {
					m.submitMessage(value)
				}
			}
		}
	}

	var cmd tea.Cmd
	m.viewport, cmd = m.viewport.Update(msg)
	cmds = append(cmds, cmd)

	if m.state == stateConnected && !isScrollMsg(msg) {
		var cmd tea.Cmd
		m.input, cmd = m.input.Update(msg)
		cmds = append(cmds, cmd)
	}

	return m, tea.Batch(cmds...)
}
