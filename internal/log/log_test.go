//go:build !js || !wasm

package log

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestRotateLatest(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)

	logsDir := filepath.Join(home, ".intertui", "logs")
	if err := os.MkdirAll(logsDir, 0o700); err != nil {
		t.Fatal(err)
	}

	latest := filepath.Join(logsDir, "latest.log")
	mod := time.Date(2025, 6, 10, 12, 34, 56, 0, time.UTC)
	if err := os.WriteFile(latest, []byte("old session\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.Chtimes(latest, mod, mod); err != nil {
		t.Fatal(err)
	}

	if err := Open(); err != nil {
		t.Fatal(err)
	}
	defer Close()

	archive := filepath.Join(logsDir, "2025-06-10T12-34-56.log")
	if _, err := os.Stat(archive); err != nil {
		t.Fatalf("archive missing: %v", err)
	}
	got, err := os.ReadFile(archive)
	if err != nil {
		t.Fatal(err)
	}
	if string(got) != "old session\n" {
		t.Fatalf("archive = %q", got)
	}

	Info("hello")
	if err := Close(); err != nil {
		t.Fatal(err)
	}

	current, err := os.ReadFile(latest)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(current), "hello") {
		t.Fatalf("latest = %q", current)
	}
}

func TestRedactWire(t *testing.T) {
	t.Parallel()

	raw := []byte(`{"request":"auth","login":{"username":"alice","password":"secret"}}`)
	got := redactWire(raw)
	if strings.Contains(got, "secret") {
		t.Fatalf("password not redacted: %s", got)
	}
	if !strings.Contains(got, "alice") {
		t.Fatalf("username removed: %s", got)
	}
}
