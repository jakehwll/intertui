package ui

import (
	"intertui/internal/config"
	"intertui/internal/intercept"
)

// Option configures the UI model.
type Option func(*Model)

// WithClient overrides how the model builds an intercept client.
func WithClient(fn func(config.Config) *intercept.Client) Option {
	return func(m *Model) {
		m.newClient = fn
	}
}
