//go:build js && wasm

package browser

import (
	"fmt"
	"net"
	"strconv"
	"strings"

	"intertui/internal/config"
)

const defaultSIOPort = 13370

// Config holds browser runtime options.
type Config struct {
	config.Config

	Proxy    bool
	SocketIO bool
}

func (c *Config) finalize() {
	switch {
	case c.URL != "" && (strings.HasPrefix(c.URL, "ws://") || strings.HasPrefix(c.URL, "wss://")):
		c.WS = true
		c.SocketIO = false
	case c.URL != "" && (strings.HasPrefix(c.URL, "http://") || strings.HasPrefix(c.URL, "https://")):
		c.SocketIO = true
	}
}

func (c Config) resolveSocketIOURL() string {
	if c.URL != "" {
		return c.URL
	}
	port := c.Port
	if port == 0 {
		port = defaultSIOPort
	}
	scheme := "http"
	if c.TLS {
		scheme = "https"
	}
	return fmt.Sprintf("%s://%s", scheme, net.JoinHostPort(c.Server, strconv.Itoa(port)))
}

func (c Config) remoteSocketIOURL() string {
	if c.URL != "" && (strings.HasPrefix(c.URL, "http://") || strings.HasPrefix(c.URL, "https://")) {
		return c.URL
	}
	return c.resolveSocketIOURL()
}

// DialDescription returns a human-readable connection target for logs.
func (c Config) DialDescription() string {
	if c.Proxy {
		return "socket.io via proxy → " + c.remoteSocketIOURL()
	}
	if c.SocketIO || strings.HasPrefix(c.URL, "http") {
		return c.resolveSocketIOURL()
	}
	if c.WS || strings.HasPrefix(c.URL, "ws") {
		return c.ResolveURL()
	}
	return "tcp://" + c.ResolveAddr()
}
