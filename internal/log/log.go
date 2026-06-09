package log

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"time"
)

const (
	appDirName    = ".intertui"
	logsDirName   = "logs"
	latestLogName = "latest.log"
)

var (
	mu   sync.Mutex
	file *os.File
)

// Open rotates any existing latest.log and opens a new session log.
func Open() error {
	mu.Lock()
	defer mu.Unlock()

	if file != nil {
		return nil
	}

	logsDir, err := logsDir()
	if err != nil {
		return err
	}
	if err := os.MkdirAll(logsDir, 0o700); err != nil {
		return err
	}

	latest, err := latestPath()
	if err != nil {
		return err
	}
	if err := rotateLatest(latest, logsDir); err != nil {
		return err
	}

	f, err := os.OpenFile(latest, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0o600)
	if err != nil {
		return err
	}
	file = f
	return nil
}

func rotateLatest(latest, logsDir string) error {
	st, err := os.Stat(latest)
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return err
	}

	base := st.ModTime().UTC().Format("2006-01-02T15-04-05")
	archive := filepath.Join(logsDir, base+".log")
	for i := 1; ; i++ {
		if _, err := os.Stat(archive); os.IsNotExist(err) {
			break
		}
		archive = filepath.Join(logsDir, fmt.Sprintf("%s-%d.log", base, i))
	}
	return os.Rename(latest, archive)
}

// Close flushes and closes the session log.
func Close() error {
	mu.Lock()
	defer mu.Unlock()

	if file == nil {
		return nil
	}
	err := file.Close()
	file = nil
	return err
}

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

func write(kind, line string) {
	mu.Lock()
	defer mu.Unlock()

	if file == nil {
		return
	}

	ts := time.Now().UTC().Format(time.RFC3339Nano)
	fmt.Fprintf(file, "%s %s %s\n", ts, kind, line)
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

// Path returns the active session log path.
func Path() (string, error) {
	return latestPath()
}

func logsDir() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(home, appDirName, logsDirName), nil
}

func latestPath() (string, error) {
	dir, err := logsDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(dir, latestLogName), nil
}

// Enabled reports whether file logging is active.
func Enabled() bool {
	mu.Lock()
	defer mu.Unlock()
	return file != nil
}
