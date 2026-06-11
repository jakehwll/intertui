package app

import (
	"fmt"

	"intertui/internal/config"
	filelog "intertui/internal/log"
)

// Run starts the TUI with the given configuration.
func Run(cfg config.Config) error {
	if err := filelog.Open(); err != nil {
		fmt.Printf("warning: file logging disabled: %v\n", err)
	} else {
		defer filelog.Close()
	}

	if !cfg.HasCreds() {
		return fmt.Errorf("credentials required: run `intertui init`, or use --user and --pass")
	}
	if cfg.Server == "" && cfg.URL == "" {
		return fmt.Errorf("server required: run `intertui init --server HOST`, or use --server / --url")
	}

	logPath, _ := filelog.Path()
	filelog.Info("start target=%s ws=%v user=%s log=%s",
		cfg.DialDescription(), cfg.WS, cfg.User, logPath)

	p := newProgram(cfg)
	if err := runProgram(p); err != nil {
		return err
	}
	return nil
}
