//go:build js && wasm

package browser

import (
	"fmt"
	"net/url"
	"strconv"
	"strings"
	"syscall/js"

	"intertui/internal/config"
)

// ParseQuery reads runtime options from the browser URL query string.
func ParseQuery() (Config, error) {
	search := js.Global().Get("window").Get("location").Get("search").String()
	params := parseSearch(search)

	cfg := Config{
		Offline: true,
		Config: config.Config{
			User: "offline",
			Pass: "offline",
		},
	}

	if v := params["offline"]; v == "0" || v == "false" {
		cfg.Offline = false
	}
	if v := params["ws"]; v == "1" || v == "true" {
		cfg.WS = true
		cfg.SocketIO = false
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

	if cfg.Server != "" || cfg.URL != "" {
		cfg.Offline = false
	}
	if !cfg.Offline && !cfg.WS {
		cfg.SocketIO = true
	}

	cfg.finalize()

	if !cfg.Offline && cfg.SocketIO && params["url"] == "" && proxyEnabled(params) {
		cfg.Proxy = true
		cfg.URL = js.Global().Get("window").Get("location").Get("origin").String()
	}

	if !cfg.Offline {
		if !cfg.HasCreds() {
			return Config{}, fmt.Errorf("credentials required: add ?user=...&pass=... or ?token=...")
		}
		if cfg.Server == "" && cfg.URL == "" {
			return Config{}, fmt.Errorf("server required: add ?server=HOST")
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
		decoded, err := url.QueryUnescape(val)
		if err != nil {
			out[key] = val
			continue
		}
		out[key] = decoded
	}
	return out
}

func proxyEnabled(params map[string]string) bool {
	return params["direct"] != "1" && params["direct"] != "true"
}
