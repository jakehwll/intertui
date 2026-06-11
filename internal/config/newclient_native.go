//go:build !js || !wasm

package config

import "intertui/internal/intercept"

// NewClient builds an intercept client from config.
func (c Config) NewClient() *intercept.Client {
	if c.Offline {
		return intercept.NewMock(c.Credentials())
	}
	if c.WS {
		return intercept.NewWebSocket(c.ResolveURL(), c.Credentials())
	}
	return intercept.NewTCP(c.ResolveAddr(), c.Credentials())
}
