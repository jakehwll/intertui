package intercept

import (
	"bufio"
	"context"
	"encoding/json"
	"net"
	"strings"
	"testing"
)

func startRegisterMock(t *testing.T, authReply string) (addr string, sawRegister *bool, sawConnect *bool) {
	t.Helper()

	registered := false
	connected := false

	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { ln.Close() })

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
			request, _ := req["request"].(string)
			switch request {
			case "auth":
				if _, ok := req["register"]; ok {
					registered = true
				}
				if _, ok := req["login"]; ok {
					t.Error("expected register auth, got login")
				}
				_, _ = conn.Write([]byte(authReply + "\n"))
			case "connect":
				connected = true
			}
		}
	}()

	return ln.Addr().String(), &registered, &connected
}

func TestRegisterSuccess(t *testing.T) {
	addr, sawRegister, sawConnect := startRegisterMock(t,
		`{"event":"auth","success":true,"token":"tok","player":"alice"}`)

	player, err := Register(context.Background(), addr, "alice", "secret")
	if err != nil {
		t.Fatalf("Register: %v", err)
	}
	if player != "alice" {
		t.Fatalf("player = %q, want alice", player)
	}
	if !*sawRegister {
		t.Fatal("server never received register auth")
	}
	if *sawConnect {
		t.Fatal("register should not send connect")
	}
}

func TestRegisterError(t *testing.T) {
	addr, _, _ := startRegisterMock(t,
		`{"event":"error","error":"username taken"}`)

	_, err := Register(context.Background(), addr, "alice", "secret")
	if err == nil {
		t.Fatal("expected error")
	}
	if !strings.Contains(err.Error(), "username taken") {
		t.Fatalf("err = %v", err)
	}
}

func TestRegisterMissingCreds(t *testing.T) {
	_, err := Register(context.Background(), "127.0.0.1:1", "", "secret")
	if err == nil {
		t.Fatal("expected error for empty user")
	}
}

func TestAuthUserPassPayload(t *testing.T) {
	login := authUserPassPayload("login", "u", "p")
	if login["request"] != "auth" {
		t.Fatalf("request = %v", login["request"])
	}
	if _, ok := login["login"]; !ok {
		t.Fatal("expected login key")
	}
	if _, ok := login["register"]; ok {
		t.Fatal("unexpected register key")
	}

	reg := authUserPassPayload("register", "u", "p")
	if _, ok := reg["register"]; !ok {
		t.Fatal("expected register key")
	}
}
