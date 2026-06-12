//go:build js && wasm

package browser

import (
	"intertui/internal/config"
	"intertui/internal/intercept"
)

// NewClient builds an intercept client for the browser runtime.
func NewClient(cfg Config) *intercept.Client {
	cred := cfg.Credentials()
	if cfg.WS {
		return intercept.NewWebSocket(cfg.ResolveURL(), cred)
	}
	return intercept.NewPlugin(cfg.resolveSocketIOURL(), cred, &socketIO{})
}

// ClientFactory returns a client constructor for ui.WithClient.
func ClientFactory(cfg Config) func(config.Config) *intercept.Client {
	return func(config.Config) *intercept.Client {
		return NewClient(cfg)
	}
}
