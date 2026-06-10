//go:build js && wasm

package config

import (
	"fmt"
	"strconv"
	"strings"
	"syscall/js"
)

// ParseQuery reads runtime options from the browser URL query string.
// Defaults to offline mock when no live-connection params are present.
func ParseQuery() (Config, error) {
	search := js.Global().Get("window").Get("location").Get("search").String()
	params := parseSearch(search)

	cfg := Config{Offline: true}

	if v := params["offline"]; v == "0" || v == "false" {
		cfg.Offline = false
	}
	if v := params["ws"]; v == "1" || v == "true" {
		cfg.WS = true
		cfg.Offline = false
	}
	if v := params["tls"]; v == "1" || v == "true" {
		cfg.TLS = true
	}
	if v := params["user"]; v != "" {
		cfg.User = v
	}
	if v := params["pass"]; v != "" {
		cfg.Pass = v
	}
	if v := params["token"]; v != "" {
		cfg.Token = v
	}
	if v := params["server"]; v != "" {
		cfg.Server = v
	}
	if v := params["port"]; v != "" {
		port, err := strconv.Atoi(v)
		if err != nil {
			return Config{}, fmt.Errorf("invalid port %q", v)
		}
		cfg.Port = port
	}
	if v := params["url"]; v != "" {
		cfg.URL = v
	}
	cfg.finalize()

	if !cfg.Offline {
		if !cfg.HasCreds() {
			return Config{}, fmt.Errorf("credentials required: add ?user=...&pass=... or ?token=...")
		}
		if !cfg.WS && cfg.Server == "" && cfg.URL == "" {
			return Config{}, fmt.Errorf("server required: add ?ws=1&server=HOST (TCP is unavailable in the browser)")
		}
		if !cfg.WS {
			return Config{}, fmt.Errorf("TCP is unavailable in the browser; add ?ws=1")
		}
	}

	return cfg, nil
}

func parseSearch(search string) map[string]string {
	out := make(map[string]string)
	search = strings.TrimPrefix(search, "?")
	if search == "" {
		return out
	}
	for _, part := range strings.Split(search, "&") {
		if part == "" {
			continue
		}
		key, val, ok := strings.Cut(part, "=")
		if !ok {
			out[key] = ""
			continue
		}
		out[key] = val
	}
	return out
}
