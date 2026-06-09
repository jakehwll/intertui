package config

import (
	"os"
	"path/filepath"
	"testing"

	"intertui/internal/constants"
)

func TestRunInit(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("HOME", dir)

	path, err := RunInit(InitOptions{
		Server: "example.com",
		Port:   int64(constants.DEFAULT_PORT),
		User:   "alice",
		Pass:   "secret",
	})
	if err != nil {
		t.Fatal(err)
	}

	want := filepath.Join(dir, ".intertui", "config.yaml")
	if path != want {
		t.Fatalf("path = %q, want %q", path, want)
	}

	info, err := os.Stat(path)
	if err != nil {
		t.Fatal(err)
	}
	if info.Mode().Perm() != 0o600 {
		t.Fatalf("mode = %o, want 0600", info.Mode().Perm())
	}

	_, err = RunInit(InitOptions{Server: "example.com"})
	if err == nil {
		t.Fatal("expected error when config exists")
	}

	_, err = RunInit(InitOptions{Server: "other.com", Force: true})
	if err != nil {
		t.Fatal(err)
	}
}

func TestParseFromConfigFile(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("HOME", dir)

	_, err := RunInit(InitOptions{
		Server: "game.example.com",
		Port:   4242,
		User:   "bob",
		Pass:   "hunter2",
	})
	if err != nil {
		t.Fatal(err)
	}

	path, err := ConfigPath()
	if err != nil {
		t.Fatal(err)
	}

	var cfg Config
	inv := RootCmd(func(c Config) error {
		cfg = c
		return nil
	}).Invoke("--config", path)
	err = inv.Run()
	if err != nil {
		t.Fatal(err)
	}

	if cfg.Server != "game.example.com" {
		t.Fatalf("server = %q", cfg.Server)
	}
	if cfg.Port != 4242 {
		t.Fatalf("port = %d", cfg.Port)
	}
	if cfg.User != "bob" || cfg.Pass != "hunter2" {
		t.Fatalf("creds = %q / %q", cfg.User, cfg.Pass)
	}
}
