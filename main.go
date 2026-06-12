package main

import (
	"errors"
	"fmt"
	"os"

	tea "charm.land/bubbletea/v2"
	"github.com/spf13/pflag"

	"intertui/internal/config"
	filelog "intertui/internal/log"
	"intertui/internal/ui"
	"intertui/internal/ui/auth"
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

	if auth.Needs(cfg) {
		authCfg, ok, err := auth.Run(cfg)
		if err != nil {
			return err
		}
		if !ok {
			return nil
		}
		cfg = authCfg
	}

	logPath, _ := filelog.Path()
	filelog.Info("start target=%s ws=%v user=%s log=%s", cfg.DialDescription(), cfg.WS, cfg.User, logPath)

	p := tea.NewProgram(ui.New(cfg), tea.WithFPS(30))

	if _, err := p.Run(); err != nil {
		return err
	}
	return nil
}
