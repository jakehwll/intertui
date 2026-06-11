// Dev entrypoint: built-in mock server for offline UI and protocol work.
// Not installed with `go install`; run with `go run ./cmd/dev`.
package main

import (
	"fmt"
	"os"

	tea "charm.land/bubbletea/v2"

	"intertui/internal/config"
	"intertui/internal/intercept"
	filelog "intertui/internal/log"
	"intertui/internal/ui"
)

func main() {
	srv, url := intercept.StartMockServer()
	defer srv.Close()

	cfg := config.Config{
		URL:  url,
		WS:   true,
		User: "offline",
		Pass: "offline",
	}

	if err := run(cfg); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func run(cfg config.Config) error {
	if err := filelog.Open(); err != nil {
		fmt.Fprintf(os.Stderr, "warning: file logging disabled: %v\n", err)
	} else {
		defer filelog.Close()
	}

	logPath, _ := filelog.Path()
	filelog.Info("start target=%s ws=%v user=%s log=%s", cfg.DialDescription(), cfg.WS, cfg.User, logPath)

	p := tea.NewProgram(ui.New(cfg), tea.WithFPS(30))
	if _, err := p.Run(); err != nil {
		return err
	}
	return nil
}
