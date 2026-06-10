package main

import (
	"errors"
	"fmt"
	"os"

	"github.com/spf13/pflag"

	"intertui/internal/app"
	"intertui/internal/config"
)

func main() {
	err := config.RunCLI(app.Run)
	if err != nil {
		if !errors.Is(err, pflag.ErrHelp) {
			fmt.Fprintln(os.Stderr, err)
		}
		os.Exit(1)
	}
}
