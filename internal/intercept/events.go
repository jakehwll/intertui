package intercept

import (
	"encoding/json"
	"fmt"
	"strings"
)

// Envelope is the top-level JSON shape for inbound messages.
type Envelope struct {
	Event string `json:"event"`

	Success bool            `json:"success"`
	Token   string          `json:"token"`
	Player  json.RawMessage `json:"player"`
	Cmd     string `json:"cmd"`
	Msg     string `json:"msg"`
	Error   string `json:"error"`

	Hostname string `json:"hostname"`
	User     string `json:"user"`
	Access   bool   `json:"access"`

	Systems []SystemInfo `json:"systems"`
}

// SystemInfo describes a game system from the systems event.
type SystemInfo struct {
	ID       string `json:"id"`
	IP       string `json:"ip"`
	Hostname string `json:"hostname"`
	Type     string `json:"type"`
}

// ParseEnvelope unmarshals raw JSON into an Envelope.
func ParseEnvelope(raw []byte) (Envelope, error) {
	var env Envelope
	err := json.Unmarshal(raw, &env)
	return env, err
}

// PlayerName returns a username from auth (string) or connect (object) payloads.
func (e Envelope) PlayerName() string {
	if len(e.Player) == 0 || string(e.Player) == "null" {
		return ""
	}
	var name string
	if json.Unmarshal(e.Player, &name) == nil && name != "" {
		return name
	}
	var info struct {
		Conn string `json:"conn"`
		IP   string `json:"ip"`
	}
	if json.Unmarshal(e.Player, &info) == nil {
		if info.Conn != "" {
			return info.Conn
		}
		return info.IP
	}
	return ""
}

// DisplayLine returns a user-visible line for events that carry msg or error text.
func (e Envelope) DisplayLine() (string, bool) {
	switch e.Event {
	case "broadcast", "chat", "command", "traceStart", "traceComplete":
		if e.Msg != "" {
			return ANSI(e.Msg), true
		}
	case "error":
		if e.Error != "" {
			return ANSI(e.Error), true
		}
	case "connected", "connect":
		if e.Msg != "" {
			return ANSI(e.Msg), true
		}
		if e.User != "" || e.Hostname != "" {
			return "connected to " + e.Hostname + " as " + e.User, true
		}
	}
	return "", false
}

// Summarize returns a short debug line for unexpected server events.
func (e Envelope) Summarize() string {
	parts := []string{e.Event}
	if e.Success {
		parts = append(parts, "ok")
	}
	if e.Error != "" {
		parts = append(parts, "error="+e.Error)
	}
	if e.Msg != "" {
		parts = append(parts, truncate(e.Msg, 60))
	}
	if e.User != "" {
		parts = append(parts, "user="+e.User)
	}
	if e.Token != "" {
		parts = append(parts, "token=…")
	}
	return "server → " + strings.Join(parts, ", ")
}

func truncate(s string, n int) string {
	if len(s) <= n {
		return s
	}
	return s[:n] + "…"
}

// ConnectOK reports whether an envelope completes the login connect handshake.
func (e Envelope) ConnectOK() bool {
	switch e.Event {
	case "connected":
		return true
	case "connect":
		return e.Success
	case "error":
		return false
	default:
		return false
	}
}

// ConnectErr formats a connect-handshake failure.
func (e Envelope) ConnectErr() error {
	if e.Event == "error" && e.Error != "" {
		return fmt.Errorf("connect error: %s", e.Error)
	}
	return fmt.Errorf("connect failed (%s)", e.Event)
}
