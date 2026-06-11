//go:build js && wasm

package ui

import (
	"intertui/internal/config"
	"intertui/internal/intercept"
)

func defaultClientFactory(cfg config.Config) func(config.Config) *intercept.Client {
	_ = cfg
	return func(config.Config) *intercept.Client {
		panic("browser builds must use ui.WithClient")
	}
}
