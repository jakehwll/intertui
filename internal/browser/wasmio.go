//go:build js && wasm

package browser

import (
	"io"
	"os"
	"strings"
	"syscall/js"
	"time"

	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
	"github.com/charmbracelet/colorprofile"
)

type xtermWriter struct {
	term js.Value
}

func (w *xtermWriter) Write(p []byte) (n int, err error) {
	s := strings.ReplaceAll(string(p), "\r\n", "\n")
	s = strings.ReplaceAll(s, "\n", "\r\n")
	w.term.Call("write", s)
	return len(p), nil
}

func newProgram(model tea.Model, options ...tea.ProgramOption) *tea.Program {
	term := js.Global().Get("term")
	cols := term.Get("cols").Int()
	rows := term.Get("rows").Int()

	inR, inW := io.Pipe()
	outW := &xtermWriter{term: term}

	opts := append([]tea.ProgramOption{
		tea.WithInput(inR),
		tea.WithOutput(outW),
		tea.WithFPS(120),
		tea.WithWindowSize(cols, rows),
		tea.WithColorProfile(colorprofile.TrueColor),
		tea.WithEnvironment(wasmEnviron()),
	}, options...)

	p := tea.NewProgram(model, opts...)

	onData := js.FuncOf(func(_ js.Value, args []js.Value) any {
		_, _ = inW.Write([]byte(args[0].String()))
		return nil
	})
	term.Call("onData", onData)

	sendWindowSize := func() {
		cols := term.Get("cols").Int()
		rows := term.Get("rows").Int()
		if cols <= 0 || rows <= 0 {
			return
		}
		p.Send(tea.WindowSizeMsg{Width: cols, Height: rows})
	}

	onResize := js.FuncOf(func(_ js.Value, args []js.Value) any {
		if len(args) > 0 {
			evt := args[0]
			cols := evt.Get("cols").Int()
			rows := evt.Get("rows").Int()
			if cols > 0 && rows > 0 {
				p.Send(tea.WindowSizeMsg{Width: cols, Height: rows})
			}
		}
		return nil
	})
	term.Call("onResize", onResize)

	js.Global().Set("wasmInitialResize", js.FuncOf(func(_ js.Value, _ []js.Value) any {
		sendWindowSize()
		return nil
	}))

	return p
}

func runProgram(p *tea.Program) error {
	go func() {
		time.Sleep(100 * time.Millisecond)
		js.Global().Call("wasmInitialResize")
	}()

	_, err := p.Run()
	return err
}

func wasmEnviron() []string {
	return []string{
		"TERM=xterm-256color",
		"COLORTERM=truecolor",
		"CLICOLOR_FORCE=1",
	}
}

func initTerminal() {
	for _, entry := range wasmEnviron() {
		key, val, ok := strings.Cut(entry, "=")
		if ok {
			_ = os.Setenv(key, val)
		}
	}

	js.Global().Get("document").Get("body").Set("style", map[string]any{
		"margin":           "0",
		"background-color": "#000",
	})
	lipgloss.Writer.Profile = colorprofile.TrueColor
}
