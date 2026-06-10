//go:build js && wasm

package app

import (
	tea "charm.land/bubbletea/v2"

	"intertui/internal/wasmio"
)

func runProgram(p *tea.Program) error {
	return wasmio.Run(p)
}
