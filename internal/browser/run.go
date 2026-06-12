//go:build js && wasm

package browser

import (
	"fmt"

	filelog "intertui/internal/log"
	"intertui/internal/ui"
)

// Run starts the browser TUI.
func Run(cfg Config) error {
	if err := filelog.Open(); err != nil {
		fmt.Printf("warning: logging disabled: %v\n", err)
	} else {
		defer filelog.Close()
	}

	if !cfg.HasCreds() {
		return fmt.Errorf("credentials required")
	}
	if cfg.Server == "" && cfg.URL == "" {
		return fmt.Errorf("server required")
	}

	logPath, _ := filelog.Path()
	filelog.Info("start target=%s ws=%v sio=%v user=%s log=%s",
		cfg.DialDescription(), cfg.WS, cfg.SocketIO, cfg.User, logPath)

	initTerminal()
	p := newProgram(ui.New(cfg.Config, ui.WithClient(ClientFactory(cfg))))
	return runProgram(p)
}
