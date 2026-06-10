//go:build js && wasm

package main

import (
	"syscall/js"

	"intertui/internal/app"
	"intertui/internal/config"
	"intertui/internal/wasmio"
)

func main() {
	wasmio.InitTerminal()

	cfg, err := config.ParseQuery()
	if err != nil {
		js.Global().Get("console").Call("error", err.Error())
		return
	}

	if err := app.Run(cfg); err != nil {
		js.Global().Get("console").Call("error", "intertui:", err.Error())
	}
}
