package ui

import (
	"charm.land/bubbles/v2/key"
	"charm.land/bubbles/v2/viewport"
	tea "charm.land/bubbletea/v2"
)

// scrollKeyMap binds scroll keys that don't overlap with normal typing.
func scrollKeyMap() viewport.KeyMap {
	return viewport.KeyMap{
		PageDown:     key.NewBinding(key.WithKeys("pgdown")),
		PageUp:       key.NewBinding(key.WithKeys("pgup")),
		HalfPageUp:   key.NewBinding(key.WithKeys("ctrl+u")),
		HalfPageDown: key.NewBinding(key.WithKeys("ctrl+d")),
		Up:           key.NewBinding(key.WithKeys("up")),
		Down:         key.NewBinding(key.WithKeys("down")),
		Left:         key.NewBinding(key.WithKeys("left")),
		Right:        key.NewBinding(key.WithKeys("right")),
	}
}

// isScrollMsg reports whether msg should scroll the log, not the input.
func isScrollMsg(msg tea.Msg) bool {
	switch msg := msg.(type) {
	case tea.MouseWheelMsg:
		return true
	case tea.KeyPressMsg:
		km := scrollKeyMap()
		return key.Matches(msg, km.Up) ||
			key.Matches(msg, km.Down) ||
			key.Matches(msg, km.PageUp) ||
			key.Matches(msg, km.PageDown) ||
			key.Matches(msg, km.HalfPageUp) ||
			key.Matches(msg, km.HalfPageDown) ||
			key.Matches(msg, km.Left) ||
			key.Matches(msg, km.Right)
	default:
		return false
	}
}
