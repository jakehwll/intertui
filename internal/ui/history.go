package ui

// historyUp moves to the previous submitted message.
func (m *Model) historyUp() {
	if len(m.history) == 0 {
		return
	}

	if m.historyIndex == -1 {
		m.historyDraft = m.input.Value()
		m.historyIndex = len(m.history) - 1
	} else if m.historyIndex > 0 {
		m.historyIndex--
	}

	m.input.SetValue(m.history[m.historyIndex])
	m.input.CursorEnd()
}

// historyDown moves to the next submitted message, or back to the draft.
func (m *Model) historyDown() {
	if m.historyIndex == -1 {
		return
	}

	if m.historyIndex < len(m.history)-1 {
		m.historyIndex++
		m.input.SetValue(m.history[m.historyIndex])
	} else {
		m.historyIndex = -1
		m.input.SetValue(m.historyDraft)
	}

	m.input.CursorEnd()
}

func (m *Model) submitMessage(value string) {
	m.messages = append(m.messages, clientLine("> "+value))

	if m.client != nil {
		m.client.SendCommand(value)
	}

	if len(m.history) == 0 || m.history[len(m.history)-1] != value {
		m.history = append(m.history, value)
	}

	m.historyIndex = -1
	m.historyDraft = ""
	m.input.SetValue("")
	m.input.CursorEnd()
	m.updateViewport()
	m.viewport.GotoBottom()
}
