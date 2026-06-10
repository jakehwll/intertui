//go:build !js || !wasm

package app

import (
	tea "charm.land/bubbletea/v2"

	"intertui/internal/config"
	"intertui/internal/ui"
)

func newProgram(cfg config.Config) *tea.Program {
	return tea.NewProgram(ui.New(cfg), tea.WithFPS(120))
}
