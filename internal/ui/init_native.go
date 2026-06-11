//go:build !js || !wasm

package ui

import tea "charm.land/bubbletea/v2"

// deferredStartMsg exists for shared Update switch; unused on native.
type deferredStartMsg struct{}

func (m Model) programInit() tea.Cmd {
	if m.state == stateConnecting {
		return startClient(m.cfg)
	}
	return nil
}

func (m *Model) maybeStartClient() tea.Cmd { return nil }
