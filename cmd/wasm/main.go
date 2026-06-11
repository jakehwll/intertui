//go:build js && wasm

package main

import (
	"syscall/js"

	"intertui/internal/browser"
)

func main() {
	cfg, err := browser.ParseQuery()
	if err != nil {
		js.Global().Get("console").Call("error", err.Error())
		return
	}

	if err := browser.Run(cfg); err != nil {
		js.Global().Get("console").Call("error", "intertui:", err.Error())
	}
}
