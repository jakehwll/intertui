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
	User  string
	Pass  string
	Token string

	Server   string
	Port     int
	WS       bool
	SocketIO bool
	TLS      bool
	URL      string
	Offline  bool // WASM mock mode; set via URL query, not CLI
	Proxy    bool // WASM: Socket.IO via same-origin reverse proxy
}

// RootCmd returns the CLI root with the default TUI command and subcommands.
func RootCmd(run func(Config) error) *serpent.Command {
	var cfg Config
	var port int64
	var configPath serpent.YAMLConfigPath

	opts := serpent.OptionSet{
		{
			Name:        "config",
			Flag:        "config",
			Env:         "INTERTUI_CONFIG",
			Value:       &configPath,
			Description: "Path to YAML config file (default: ~/.intertui/config.yaml).",
		},
		{
			Name:        "user",
			Flag:        "user",
			Env:         "INTERCEPT_USER",
			YAML:        "user",
			Value:       serpent.StringOf(&cfg.User),
			Description: "Intercept username.",
		},
		{
			Name:        "pass",
			Flag:        "pass",
			Env:         "INTERCEPT_PASS",
			YAML:        "pass",
			Value:       serpent.StringOf(&cfg.Pass),
			Description: "Intercept password.",
		},
		{
			Name:   "server",
			YAML:   "server",
			Hidden: true,
			Value:  serpent.StringOf(&cfg.Server),
		},
		{
			Name:    "port",
			YAML:    "port",
			Hidden:  true,
			Default: "0",
			Value:   serpent.Int64Of(&port),
		},
		{
			Name:   "token",
			YAML:   "token",
			Hidden: true,
			Value:  serpent.StringOf(&cfg.Token),
		},
		{
			Name:   "ws",
			YAML:   "ws",
			Hidden: true,
			Value:  serpent.BoolOf(&cfg.WS),
		},
		{
			Name:   "tls",
			YAML:   "tls",
			Hidden: true,
			Value:  serpent.BoolOf(&cfg.TLS),
		},
		{
			Name:   "url",
			YAML:   "url",
			Hidden: true,
			Value:  serpent.StringOf(&cfg.URL),
		},
	}

	root := &serpent.Command{
		Use:     "intertui",
		Short:   "Terminal client for Intercept.",
		Options: opts,
		Handler: func(inv *serpent.Invocation) error {
			cfg.Port = int(port)
			cfg.finalize()
			return run(cfg)
		},
	}

	root.AddSubcommands(InitCmd(), RegisterCmd())
	return root
}

// Parse reads flags, environment variables, and ~/.intertui/config.yaml.
func Parse() (Config, error) {
	var cfg Config
	err := RunCLI(func(c Config) error {
		cfg = c
		return nil
	})
	return cfg, err
}

func (c *Config) finalize() {
	switch {
	case c.URL != "" && (strings.HasPrefix(c.URL, "ws://") || strings.HasPrefix(c.URL, "wss://")):
		c.WS = true
	case c.URL != "" && (strings.HasPrefix(c.URL, "http://") || strings.HasPrefix(c.URL, "https://")):
		c.SocketIO = true
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

// ResolveSocketIOURL builds a Socket.IO URL when not overridden.
func (c Config) ResolveSocketIOURL() string {
	if c.URL != "" {
		return c.URL
	}
	port := c.Port
	if port == 0 {
		port = constants.DEFAULT_SIO_PORT
	}
	scheme := "http"
	if c.TLS {
		scheme = "https"
	}
	return fmt.Sprintf("%s://%s", scheme, net.JoinHostPort(c.Server, strconv.Itoa(port)))
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
	if c.Proxy {
		return "socket.io via proxy → " + c.remoteSocketIOURL()
	}
	if c.SocketIO || strings.HasPrefix(c.URL, "http") {
		return c.ResolveSocketIOURL()
	}
	if c.WS || strings.HasPrefix(c.URL, "ws") {
		return c.ResolveURL()
	}
	return "tcp://" + c.ResolveAddr()
}

func (c Config) remoteSocketIOURL() string {
	if c.URL != "" && (strings.HasPrefix(c.URL, "http://") || strings.HasPrefix(c.URL, "https://")) {
		return c.URL
	}
	return c.ResolveSocketIOURL()
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
