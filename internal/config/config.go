package config

import (
	"fmt"
	"net"
	"strconv"
	"strings"

	"github.com/coder/serpent"

	"intertui/internal/constants"
	"intertui/internal/intercept"
)

// Config holds runtime options for intertui.
type Config struct {
	User    string
	Pass    string
	Token   string
	Offline bool

	Server string
	Port   int
	WS     bool
	TLS    bool
	URL    string
}

// Parse reads flags and environment variables.
func Parse() (Config, error) {
	var cfg Config
	var port int64

	cmd := &serpent.Command{
		Use:   "intertui",
		Short: "Terminal client for Intercept.",
		Options: serpent.OptionSet{
			{
				Name:        "user",
				Flag:        "user",
				Env:         "INTERCEPT_USER",
				Value:       serpent.StringOf(&cfg.User),
				Description: "Intercept username.",
			},
			{
				Name:        "pass",
				Flag:        "pass",
				Env:         "INTERCEPT_PASS",
				Value:       serpent.StringOf(&cfg.Pass),
				Description: "Intercept password.",
			},
			{
				Name:        "token",
				Flag:        "token",
				Env:         "INTERCEPT_TOKEN",
				Value:       serpent.StringOf(&cfg.Token),
				Description: "Intercept API token (WebSocket mode only).",
			},
			{
				Name:        "server",
				Flag:        "server",
				Env:         "INTERCEPT_SERVER",
				Value:       serpent.StringOf(&cfg.Server),
				Description: "Game server host or IP.",
			},
			{
				Name:        "port",
				Flag:        "port",
				Env:         "INTERCEPT_PORT",
				Default:     "0",
				Value:       serpent.Int64Of(&port),
				Description: fmt.Sprintf("Server port (default %d).", constants.DEFAULT_PORT),
			},
			{
				Name:        "ws",
				Flag:        "ws",
				Env:         "INTERCEPT_WS",
				Value:       serpent.BoolOf(&cfg.WS),
				Description: fmt.Sprintf("Use WebSocket transport (default: raw TCP on port %d).", constants.DEFAULT_PORT),
			},
			{
				Name:        "tls",
				Flag:        "tls",
				Env:         "INTERCEPT_TLS",
				Value:       serpent.BoolOf(&cfg.TLS),
				Description: "Use wss:// instead of ws:// (with --ws).",
			},
			{
				Name:        "url",
				Flag:        "url",
				Env:         "INTERCEPT_URL",
				Value:       serpent.StringOf(&cfg.URL),
				Description: "Full endpoint URL (overrides --server; ws:// or wss:// enables WebSocket).",
			},
			{
				Name:        "offline",
				Flag:        "offline",
				Value:       serpent.BoolOf(&cfg.Offline),
				Description: "Use built-in mock server.",
			},
		},
		Handler: func(inv *serpent.Invocation) error {
			cfg.Port = int(port)
			cfg.finalize()
			return nil
		},
	}

	if err := cmd.Invoke().WithOS().Run(); err != nil {
		return Config{}, err
	}
	return cfg, nil
}

func (c *Config) finalize() {
	if c.URL != "" && (strings.HasPrefix(c.URL, "ws://") || strings.HasPrefix(c.URL, "wss://")) {
		c.WS = true
	}
}

// ResolveAddr returns host:port for TCP dial.
func (c Config) ResolveAddr() string {
	port := c.Port
	if port == 0 {
		port = constants.DEFAULT_PORT
	}
	return net.JoinHostPort(c.Server, strconv.Itoa(port))
}

// ResolveURL builds a WebSocket URL when not overridden.
func (c Config) ResolveURL() string {
	if c.URL != "" {
		return c.URL
	}
	server := c.Server
	port := c.Port
	if port == 0 {
		port = constants.DEFAULT_PORT
	}
	scheme := "ws"
	if c.TLS {
		scheme = "wss"
	}
	return fmt.Sprintf("%s://%s:%d/ws", scheme, server, port)
}

// DialDescription returns a human-readable target for status output.
func (c Config) DialDescription() string {
	if c.Offline {
		return "offline mock server"
	}
	if c.WS || strings.HasPrefix(c.URL, "ws") {
		return c.ResolveURL()
	}
	return "tcp://" + c.ResolveAddr()
}

// NewClient builds an intercept client from config.
func (c Config) NewClient() *intercept.Client {
	if c.WS {
		return intercept.NewWebSocket(c.ResolveURL(), c.Credentials())
	}
	return intercept.NewTCP(c.ResolveAddr(), c.Credentials())
}

// HasCreds reports whether flags/env provide login details.
func (c Config) HasCreds() bool {
	return c.Token != "" || c.User != ""
}

// Credentials builds intercept credentials from config.
func (c Config) Credentials() intercept.Credentials {
	return intercept.Credentials{
		User:  c.User,
		Pass:  c.Pass,
		Token: c.Token,
	}
}
