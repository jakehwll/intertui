package ui

import (
	"strings"
	"time"
	"unicode"
	"unicode/utf8"

	"charm.land/bubbles/v2/key"
	"charm.land/bubbles/v2/textinput"
	"charm.land/bubbles/v2/viewport"
	tea "charm.land/bubbletea/v2"
	"github.com/atotto/clipboard"
	"github.com/charmbracelet/x/ansi"

	"intertui/internal/config"
	"intertui/internal/intercept"
	filelog "intertui/internal/log"
)

type connState int

const (
	stateConnecting connState = iota
	stateConnected
	stateError
)

// quitConfirmWindow is how long the Ctrl+C confirmation stays armed.
const quitConfirmWindow = 2 * time.Second

// quitConfirmTimeoutMsg resets the quit confirmation if no second Ctrl+C
// arrived. seq guards against stale timers clearing a re-armed confirm.
type quitConfirmTimeoutMsg struct{ seq int }

// Model is the Bubble Tea model for the terminal UI.
type Model struct {
	cfg    config.Config
	client *intercept.Client

	messages []string
	history  []string

	// displayLines is messages hard-wrapped to the terminal width so each
	// entry is exactly one screen row. This makes mouse-selection mapping
	// (screen cell -> text) exact.
	displayLines []string

	viewport viewport.Model
	input    textinput.Model

	state         connState
	connectedUser string
	reconnecting  bool

	width  int
	height int
	ready  bool

	historyIndex   int
	historyDraft   string
	quitConfirm    bool
	quitConfirmSeq int

	completion completionState

	// In-app mouse selection (Claude Code-style): we capture the mouse, draw
	// the highlight ourselves, and copy to the clipboard on release.
	selecting            bool // left button held, dragging
	selActive            bool // selection exists (visible highlight)
	selStartX, selStartY int  // anchor: X = cell column, Y = displayLines index
	selEndX, selEndY     int
	copied               bool
}

// New returns the initial UI model.
func New(cfg config.Config) Model {
	ti := textinput.New()
	ti.Placeholder = "type a command..."
	ti.CharLimit = 280
	styles := textinput.DefaultDarkStyles()
	styles.Cursor.Blink = false
	ti.SetStyles(styles)

	return Model{
		cfg:        cfg,
		input:      ti,
		state:      stateConnecting,
		completion: newCompletionState(),
	}
}

func (m Model) Init() tea.Cmd {
	if m.state == stateConnecting {
		return startClient(m.cfg)
	}
	return nil
}

func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmds []tea.Cmd

	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		if !m.ready {
			m.viewport = viewport.New()
			m.viewport.SoftWrap = false
			m.viewport.FillHeight = true
			m.viewport.KeyMap = logScrollKeys()
			m.ready = true
		}
		m.layout()
		m.rewrap()
		m.clearSelection()

	case connectProgressMsg:
		return m, pollConnect(msg.statusCh, msg.doneCh)

	case clientReadyMsg:
		if msg.err != nil {
			m.reconnecting = false
			m.state = stateError
			filelog.Info("connect failed err=%v", msg.err)
			m.log(dim.Render("Connection failed: " + msg.err.Error()))
			break
		}
		filelog.Info("connect ok user=%s", msg.user)
		m.client = msg.client
		m.connectedUser = msg.user
		m.state = stateConnected
		if m.reconnecting {
			m.reconnecting = false
		} else {
			m.messages = nil
			m.rewrap()
			m.clearSelection()
		}
		m.input.Focus()
		m.viewport.GotoBottom()
		cmds = append(cmds, waitClientMsg(m.client))

	case intercept.GameLineMsg:
		m.log(msg.Line)
		if m.client != nil {
			cmds = append(cmds, waitClientMsg(m.client))
		}

	case intercept.DisconnectedMsg:
		if m.state == stateConnected {
			m.state = stateError
			line := "Disconnected."
			if msg.Err != nil {
				line = "Disconnected: " + msg.Err.Error()
				filelog.Info("disconnect err=%v", msg.Err)
			} else {
				filelog.Info("disconnect")
			}
			m.log(dim.Render(line))
		}

	case probeResultMsg:
		m.applyProbeResult(msg)
		return m, nil

	case vocabResultMsg:
		m.applyVocabResult(msg)
		return m, nil

	case subcommandResultMsg:
		m.applySubcommandResult(msg)
		return m, nil

	case indexedListResultMsg:
		m.applyIndexedListResult(msg)
		return m, nil

	case quitConfirmTimeoutMsg:
		if msg.seq == m.quitConfirmSeq {
			m.quitConfirm = false
		}
		return m, nil

	case tea.MouseWheelMsg:
		var wheelCmd tea.Cmd
		m.viewport, wheelCmd = m.viewport.Update(msg)
		return m, wheelCmd

	case tea.MouseClickMsg:
		m.copied = false
		if msg.Button == tea.MouseLeft && msg.Y < m.viewport.Height() && len(m.displayLines) > 0 {
			x, y := m.clampToLog(msg.X, msg.Y)
			m.selecting = true
			m.selActive = false
			m.selStartX, m.selStartY = x, y
			m.selEndX, m.selEndY = x, y
		} else {
			m.clearSelection()
		}
		return m, nil

	case tea.MouseMotionMsg:
		if m.selecting {
			m.selEndX, m.selEndY = m.clampToLog(msg.X, msg.Y)
			m.selActive = m.selStartX != m.selEndX || m.selStartY != m.selEndY
		}
		return m, nil

	case tea.MouseReleaseMsg:
		if m.selecting {
			m.selecting = false
			if !m.selActive {
				m.expandSelectionToWord()
			}
			if m.selActive {
				if text := m.selectionText(); text != "" {
					m.copied = true
					return m, copyText(text)
				}
			}
			m.clearSelection()
		}
		return m, nil

	case tea.KeyPressMsg:
		m.copied = false
		if m.quitConfirm && msg.String() != "ctrl+c" {
			m.quitConfirm = false
		}
		switch msg.String() {
		case "ctrl+c":
			if m.quitConfirm {
				if m.client != nil {
					m.client.Close()
				}
				return m, tea.Quit
			}
			if v := m.input.Value(); v != "" {
				m.cancelInput(v)
			}
			m.quitConfirm = true
			m.quitConfirmSeq++
			seq := m.quitConfirmSeq
			return m, tea.Tick(quitConfirmWindow, func(time.Time) tea.Msg {
				return quitConfirmTimeoutMsg{seq: seq}
			})
		case "esc":
			if m.selecting || m.selActive {
				m.clearSelection()
				return m, nil
			}
			if m.client != nil {
				m.client.Close()
			}
			return m, tea.Quit
		case "ctrl+shift+c":
			cmds = append(cmds, copyLog(m.messages))
		case "up", "ctrl+p", "ctrl+up":
			if m.state == stateConnected {
				m.historyUp()
				return m, tea.Batch(cmds...)
			}
		case "down", "ctrl+n", "ctrl+down":
			if m.state == stateConnected {
				m.historyDown()
				return m, tea.Batch(cmds...)
			}
		case "tab":
			if m.state == stateConnected {
				if cmd := m.completeInput(true); cmd != nil {
					cmds = append(cmds, cmd)
				}
				return m, tea.Batch(cmds...)
			}
		case "enter":
			if m.state == stateConnected {
				if v := strings.TrimSpace(m.input.Value()); v != "" {
					m.submit(v)
				}
			}
		case "r":
			if m.state == stateError {
				cmds = append(cmds, m.reconnect()...)
				return m, tea.Batch(cmds...)
			}
		}
	}

	var cmd tea.Cmd
	m.viewport, cmd = m.viewport.Update(msg)
	cmds = append(cmds, cmd)

	if m.state == stateConnected && !logScrollKey(msg) {
		m.input, cmd = m.input.Update(msg)
		cmds = append(cmds, cmd)
	}

	return m, tea.Batch(cmds...)
}

func (m *Model) log(line string) {
	m.messages = append(m.messages, line)
	m.displayLines = append(m.displayLines, wrapLine(line, m.width)...)
	m.syncLog()
	// Don't auto-follow while the user is dragging a selection, or the text
	// would slide out from under the cursor.
	if !m.selecting {
		m.viewport.GotoBottom()
	}
}

func (m *Model) rewrap() {
	m.displayLines = m.displayLines[:0]
	for _, msg := range m.messages {
		m.displayLines = append(m.displayLines, wrapLine(msg, m.width)...)
	}
	m.syncLog()
}

func (m *Model) syncLog() {
	m.viewport.SetContent(strings.Join(m.displayLines, "\n"))
}

// wrapLine splits a message into screen rows: embedded newlines first, then
// hard-wrapping to the terminal width. Each returned entry is exactly one
// rendered row, keeping mouse-selection indices aligned with the viewport.
func wrapLine(line string, w int) []string {
	var out []string
	for _, part := range strings.Split(strings.ReplaceAll(line, "\r\n", "\n"), "\n") {
		if w <= 0 || ansi.StringWidth(part) <= w {
			out = append(out, part)
			continue
		}
		out = append(out, strings.Split(ansi.Hardwrap(part, w, true), "\n")...)
	}
	return out
}

// clampToLog converts screen coordinates to (cell column, displayLines index).
func (m Model) clampToLog(x, y int) (int, int) {
	if x < 0 {
		x = 0
	}
	if w := max(1, m.width); x >= w {
		x = w - 1
	}
	y = max(0, min(y, m.viewport.Height()-1))
	line := m.viewport.YOffset() + y
	line = max(0, min(line, len(m.displayLines)-1))
	return x, line
}

// selBounds returns the selection normalized to top-left -> bottom-right order.
func (m Model) selBounds() (x0, y0, x1, y1 int) {
	x0, y0, x1, y1 = m.selStartX, m.selStartY, m.selEndX, m.selEndY
	if y1 < y0 || (y0 == y1 && x1 < x0) {
		x0, y0, x1, y1 = x1, y1, x0, y0
	}
	return
}

// selectionText extracts the selected text (end cell inclusive), ANSI stripped.
func (m Model) selectionText() string {
	x0, y0, x1, y1 := m.selBounds()
	var out []string
	for ln := y0; ln <= y1 && ln < len(m.displayLines); ln++ {
		line := m.displayLines[ln]
		from, to := 0, ansi.StringWidth(line)
		if ln == y0 {
			from = x0
		}
		if ln == y1 {
			to = min(to, x1+1)
		}
		seg := ""
		if from < to {
			seg = ansi.Strip(ansi.Cut(line, from, to))
		}
		out = append(out, strings.TrimRight(seg, " "))
	}
	return strings.Join(out, "\n")
}

func isWordChar(r rune) bool {
	if unicode.IsLetter(r) || unicode.IsDigit(r) {
		return true
	}
	switch r {
	case '.', ':', '/', '-', '_', '@', '#':
		return true
	default:
		return false
	}
}

func wordCharAt(line string, col int) bool {
	if col < 0 || col >= ansi.StringWidth(line) {
		return false
	}
	ch := ansi.Strip(ansi.Cut(line, col, col+1))
	if ch == "" {
		return false
	}
	r, _ := utf8.DecodeRuneInString(ch)
	return isWordChar(r)
}

// wordSpans returns [from, to) display-column spans for each word on a line.
func wordSpans(line string) [][2]int {
	width := ansi.StringWidth(line)
	var spans [][2]int
	for col := 0; col < width; {
		if !wordCharAt(line, col) {
			col++
			continue
		}
		from := col
		for col < width && wordCharAt(line, col) {
			col++
		}
		spans = append(spans, [2]int{from, col})
	}
	return spans
}

func wordBoundsAt(line string, col int) (from, to int) {
	spans := wordSpans(line)
	if len(spans) == 0 {
		return col, col
	}
	width := ansi.StringWidth(line)
	col = max(0, min(col, width-1))

	for _, sp := range spans {
		if col >= sp[0] && col < sp[1] {
			return sp[0], sp[1] - 1
		}
	}

	best := spans[0]
	bestDist := wordSpanDistance(col, best)
	for _, sp := range spans[1:] {
		if d := wordSpanDistance(col, sp); d < bestDist {
			best, bestDist = sp, d
		}
	}
	return best[0], best[1] - 1
}

func wordSpanDistance(col int, span [2]int) int {
	if col < span[0] {
		return span[0] - col
	}
	if col >= span[1] {
		return col - span[1] + 1
	}
	return 0
}

func (m *Model) expandSelectionToWord() {
	if m.selStartY < 0 || m.selStartY >= len(m.displayLines) {
		return
	}
	from, to := wordBoundsAt(m.displayLines[m.selStartY], m.selStartX)
	if from > to {
		return
	}
	m.selStartX, m.selEndX = from, to
	m.selEndY = m.selStartY
	m.selActive = true
}

func (m *Model) clearSelection() {
	m.selecting = false
	m.selActive = false
}

// copyText writes to the system clipboard directly and via OSC52 so copy
// works both locally and over SSH.
func copyText(text string) tea.Cmd {
	return tea.Batch(
		tea.SetClipboard(text),
		func() tea.Msg {
			_ = clipboard.WriteAll(text)
			return nil
		},
	)
}

func logScrollKeys() viewport.KeyMap {
	return viewport.KeyMap{
		PageDown:     key.NewBinding(key.WithKeys("pgdown")),
		PageUp:       key.NewBinding(key.WithKeys("pgup")),
		HalfPageUp:   key.NewBinding(key.WithKeys("ctrl+u")),
		HalfPageDown: key.NewBinding(key.WithKeys("ctrl+d")),
	}
}

func logScrollKey(msg tea.Msg) bool {
	k, ok := msg.(tea.KeyPressMsg)
	if !ok {
		return false
	}
	km := logScrollKeys()
	return key.Matches(k, km.PageUp) || key.Matches(k, km.PageDown) ||
		key.Matches(k, km.HalfPageUp) || key.Matches(k, km.HalfPageDown)
}

func copyLog(messages []string) tea.Cmd {
	lines := make([]string, len(messages))
	for i, line := range messages {
		lines[i] = intercept.Clean(line)
	}
	return copyText(strings.Join(lines, "\n"))
}

func (m *Model) historyMatchBefore(prefix string, before int) int {
	for i := before; i >= 0; i-- {
		if strings.HasPrefix(m.history[i], prefix) {
			return i
		}
	}
	return -1
}

func (m *Model) historyMatchAfter(prefix string, after int) int {
	for i := after; i < len(m.history); i++ {
		if strings.HasPrefix(m.history[i], prefix) {
			return i
		}
	}
	return -1
}

func (m *Model) historyUp() {
	if len(m.history) == 0 {
		return
	}
	if m.historyIndex == -1 {
		m.historyDraft = m.input.Value()
		m.historyIndex = m.historyMatchBefore(m.historyDraft, len(m.history)-1)
		if m.historyIndex == -1 {
			return
		}
	} else if prev := m.historyMatchBefore(m.historyDraft, m.historyIndex-1); prev >= 0 {
		m.historyIndex = prev
	} else {
		return // oldest prefix match; keep index so down still works
	}
	m.input.SetValue(m.history[m.historyIndex])
	m.input.CursorEnd()
}

func (m *Model) historyDown() {
	if m.historyIndex == -1 {
		return
	}
	if next := m.historyMatchAfter(m.historyDraft, m.historyIndex+1); next >= 0 {
		m.historyIndex = next
		m.input.SetValue(m.history[m.historyIndex])
	} else {
		m.historyIndex = -1
		m.input.SetValue(m.historyDraft)
	}
	m.input.CursorEnd()
}

// localEcho renders a client-side log line with the same "> " prompt as the
// input, so local commands are visually distinct from server output.
func (m Model) localEcho(text string) string {
	st := m.input.Styles().Focused
	return st.Prompt.Render(m.input.Prompt) + st.Text.Render(text)
}

func (m *Model) cancelInput(value string) {
	m.log(m.localEcho(value + " ^C"))
	m.historyIndex = -1
	m.historyDraft = ""
	m.input.SetValue("")
	m.input.CursorEnd()
}

func (m *Model) submit(value string) {
	if m.client != nil {
		m.client.SendCommand(value)
	}
	m.invalidateCompletions(value)
	if len(m.history) == 0 || m.history[len(m.history)-1] != value {
		m.history = append(m.history, value)
	}
	m.historyIndex = -1
	m.historyDraft = ""
	m.input.SetValue("")
	m.input.CursorEnd()
	m.log(m.localEcho(value))
}

func (m *Model) reconnect() []tea.Cmd {
	if m.client != nil {
		m.client.Close()
		m.client = nil
	}
	m.reconnecting = true
	m.state = stateConnecting
	m.historyIndex = -1
	m.historyDraft = ""
	m.completion.onReconnect()
	m.input.Blur()
	m.input.SetValue("")
	filelog.Info("reconnect target=%s", m.cfg.DialDescription())
	return []tea.Cmd{startClient(m.cfg)}
}
