//go:build js && wasm

package log

import (
	"fmt"
	"sync"
	"syscall/js"
	"time"
)

var (
	mu      sync.Mutex
	enabled bool
)

// Open enables browser console logging.
func Open() error {
	mu.Lock()
	defer mu.Unlock()
	enabled = true
	return nil
}

// Close disables browser console logging.
func Close() error {
	mu.Lock()
	defer mu.Unlock()
	enabled = false
	return nil
}

// Path reports the browser console as the log sink.
func Path() (string, error) {
	return "console", nil
}

// Enabled reports whether console logging is active.
func Enabled() bool {
	mu.Lock()
	defer mu.Unlock()
	return enabled
}

func write(kind, line string) {
	mu.Lock()
	on := enabled
	mu.Unlock()
	if !on {
		return
	}

	ts := time.Now().UTC().Format(time.RFC3339Nano)
	msg := fmt.Sprintf("%s %s %s", ts, kind, line)

	console := js.Global().Get("console")
	switch kind {
	case "read", "write":
		console.Call("debug", msg)
	default:
		console.Call("log", msg)
	}
}
