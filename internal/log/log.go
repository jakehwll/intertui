package log

import (
	"encoding/json"
	"fmt"
)

// Info writes a session-level log line.
func Info(format string, args ...any) {
	write("info", fmt.Sprintf(format, args...))
}

// Status writes a connection progress line.
func Status(line string) {
	write("status", line)
}

// Event writes a parsed wire event summary.
func Event(name, kind string) {
	write("event", fmt.Sprintf("%s %s", name, kind))
}

// WireRead logs an inbound frame.
func WireRead(raw []byte) {
	write("read", string(raw))
}

// WireWrite logs an outbound frame, redacting passwords.
func WireWrite(raw []byte, writeErr error) {
	line := redactWire(raw)
	if writeErr != nil {
		write("write", fmt.Sprintf("%s err=%v", line, writeErr))
		return
	}
	write("write", line)
}

func redactWire(raw []byte) string {
	var v map[string]any
	if err := json.Unmarshal(raw, &v); err != nil {
		return string(raw)
	}

	if login, ok := v["login"].(map[string]any); ok {
		if _, ok := login["password"]; ok {
			login["password"] = "…"
		}
	}
	if _, ok := v["pass"]; ok {
		v["pass"] = "…"
	}

	byt, err := json.Marshal(v)
	if err != nil {
		return string(raw)
	}
	return string(byt)
}
