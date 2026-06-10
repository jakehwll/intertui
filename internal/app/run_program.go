//go:build !js || !wasm

package app

import tea "charm.land/bubbletea/v2"

func runProgram(p *tea.Program) error {
	_, err := p.Run()
	return err
}
