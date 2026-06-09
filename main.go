package main

import (
	"errors"
	"fmt"
	"os"

	tea "charm.land/bubbletea/v2"
	"github.com/spf13/pflag"

	"intertui/internal/config"
	"intertui/internal/intercept"
	"intertui/internal/ui"
)

func main() {
	cfg, err := config.Parse()
	if err != nil {
		if !errors.Is(err, pflag.ErrHelp) {
			fmt.Fprintln(os.Stderr, err)
		}
		os.Exit(1)
	}

	if cfg.Offline {
		srv, url := intercept.StartMockServer()
		defer srv.Close()
		cfg.URL = url
		cfg.WS = true
	} else {
		if !cfg.HasCreds() {
			fmt.Fprintln(os.Stderr, "credentials required: use --user and --pass, or set INTERCEPT_USER and INTERCEPT_PASS")
			os.Exit(1)
		}
		if cfg.Server == "" && cfg.URL == "" {
			fmt.Fprintln(os.Stderr, "server required: use --server, --url, or set INTERCEPT_SERVER")
			os.Exit(1)
		}
	}

	p := tea.NewProgram(ui.New(cfg), tea.WithFPS(120))

	if _, err := p.Run(); err != nil {
		fmt.Println("error:", err)
		os.Exit(1)
	}
}
