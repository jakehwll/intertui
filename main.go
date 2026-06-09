package main

import (
	"errors"
	"fmt"
	"os"

	tea "charm.land/bubbletea/v2"
	"github.com/spf13/pflag"

	"intertui/internal/config"
	"intertui/internal/intercept"
	filelog "intertui/internal/log"
	"intertui/internal/ui"
)

func main() {
	err := config.RunCLI(run)
	if err != nil {
		if !errors.Is(err, pflag.ErrHelp) {
			fmt.Fprintln(os.Stderr, err)
		}
		os.Exit(1)
	}
}

func run(cfg config.Config) error {
	if err := filelog.Open(); err != nil {
		fmt.Fprintf(os.Stderr, "warning: file logging disabled: %v\n", err)
	} else {
		defer filelog.Close()
	}

	if cfg.Offline {
		srv, url := intercept.StartMockServer()
		defer srv.Close()
		cfg.URL = url
		cfg.WS = true
	} else {
		if !cfg.HasCreds() {
			return fmt.Errorf("credentials required: run `intertui init`, or use --user and --pass")
		}
		if cfg.Server == "" && cfg.URL == "" {
			return fmt.Errorf("server required: run `intertui init --server HOST`, or use --server / --url")
		}
	}

	logPath, _ := filelog.Path()
	filelog.Info("start target=%s offline=%v ws=%v user=%s log=%s", cfg.DialDescription(), cfg.Offline, cfg.WS, cfg.User, logPath)

	p := tea.NewProgram(ui.New(cfg), tea.WithFPS(120))

	if _, err := p.Run(); err != nil {
		return err
	}
	return nil
}
