package config

import (
	"bufio"
	"context"
	"encoding/json"
	"errors"
	"net"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"testing"
)

func TestRunRegisterValidation(t *testing.T) {
	_, _, err := RunRegister(context.Background(), RegisterOptions{})
	if err == nil {
		t.Fatal("expected error without server")
	}

	_, _, err = RunRegister(context.Background(), RegisterOptions{Server: "example.com"})
	if err == nil {
		t.Fatal("expected error without user/pass")
	}
}

func TestRunRegisterWritesConfig(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("HOME", dir)

	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { ln.Close() })

	host, portStr, err := net.SplitHostPort(ln.Addr().String())
	if err != nil {
		t.Fatal(err)
	}

	go func() {
		conn, err := ln.Accept()
		if err != nil {
			return
		}
		defer conn.Close()

		sc := bufio.NewScanner(conn)
		for sc.Scan() {
			var req map[string]any
			if json.Unmarshal(sc.Bytes(), &req) != nil {
				continue
			}
			if req["request"] == "auth" {
				_, _ = conn.Write([]byte(`{"event":"auth","success":true,"token":"tok","player":"newbie"}` + "\n"))
			}
		}
	}()

	port, err := strconv.Atoi(portStr)
	if err != nil {
		t.Fatal(err)
	}

	player, path, err := RunRegister(context.Background(), RegisterOptions{
		Server: host,
		Port:   int64(port),
		User:   "newbie",
		Pass:   "hunter2",
	})
	if err != nil {
		t.Fatal(err)
	}
	if player != "newbie" {
		t.Fatalf("player = %q", player)
	}

	want := filepath.Join(dir, ".intertui", "config.yaml")
	if path != want {
		t.Fatalf("path = %q, want %q", path, want)
	}

	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	content := string(data)
	for _, want := range []string{"newbie", "hunter2", host} {
		if !strings.Contains(content, want) {
			t.Fatalf("config missing %q:\n%s", want, content)
		}
	}
}

func TestRunRegisterConfigConflictIncludesYAML(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("HOME", dir)

	if _, err := RunInit(InitOptions{
		Server: "old.example.com",
		User:   "old",
		Pass:   "old",
	}); err != nil {
		t.Fatal(err)
	}

	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { ln.Close() })

	host, portStr, err := net.SplitHostPort(ln.Addr().String())
	if err != nil {
		t.Fatal(err)
	}

	go func() {
		conn, err := ln.Accept()
		if err != nil {
			return
		}
		defer conn.Close()

		sc := bufio.NewScanner(conn)
		for sc.Scan() {
			var req map[string]any
			if json.Unmarshal(sc.Bytes(), &req) != nil {
				continue
			}
			if req["request"] == "auth" {
				_, _ = conn.Write([]byte(`{"event":"auth","success":true,"token":"tok","player":"newbie"}` + "\n"))
			}
		}
	}()

	port, err := strconv.Atoi(portStr)
	if err != nil {
		t.Fatal(err)
	}

	_, _, err = RunRegister(context.Background(), RegisterOptions{
		Server: host,
		Port:   int64(port),
		User:   "newbie",
		Pass:   "hunter2",
	})
	if err == nil {
		t.Fatal("expected config conflict error")
	}

	var conflict *RegisterConfigConflict
	if !errors.As(err, &conflict) {
		t.Fatalf("expected RegisterConfigConflict, got %T: %v", err, err)
	}
	if conflict.Player != "newbie" {
		t.Fatalf("player = %q", conflict.Player)
	}
	if !strings.Contains(conflict.Exist.YAML, "newbie") {
		t.Fatalf("YAML missing newbie:\n%s", conflict.Exist.YAML)
	}
	if !strings.Contains(conflict.Exist.YAML, host) {
		t.Fatalf("YAML missing host:\n%s", conflict.Exist.YAML)
	}
}
