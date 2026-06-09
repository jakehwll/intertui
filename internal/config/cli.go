package config

import (
	"os"

	"github.com/coder/serpent"
)

// RunCLI parses flags/env/config and runs the default command or a subcommand.
func RunCLI(run func(Config) error) error {
	cmd := RootCmd(run)

	inv := cmd.Invoke()
	inv.Args = cliArgs()
	inv.Stdout = os.Stdout
	inv.Stderr = os.Stderr
	inv.Stdin = os.Stdin
	inv.Environ = serpent.ParseEnviron(os.Environ(), "")

	return inv.Run()
}

func cliArgs() []string {
	args := os.Args[1:]
	if hasConfigFlag(args) || os.Getenv("INTERTUI_CONFIG") != "" {
		return args
	}
	if path := defaultConfigPath(); path != "" {
		return append([]string{"--config", path}, args...)
	}
	return args
}

func hasConfigFlag(args []string) bool {
	for _, arg := range args {
		if arg == "--config" || arg == "-config" {
			return true
		}
		if len(arg) > 9 && arg[:9] == "--config=" {
			return true
		}
	}
	return false
}
