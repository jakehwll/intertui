//go:build js && wasm

package app

import (
	tea "charm.land/bubbletea/v2"

	"intertui/internal/config"
	"intertui/internal/ui"
	"intertui/internal/wasmio"
)

func newProgram(cfg config.Config) *tea.Program {
	return wasmio.NewProgram(ui.New(cfg))
}
