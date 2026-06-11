//go:build !js || !wasm

package log

import (
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

// Path returns the active session log path.
func Path() (string, error) {
	return latestPath()
}

// Enabled reports whether file logging is active.
func Enabled() bool {
	mu.Lock()
	defer mu.Unlock()
	return file != nil
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
