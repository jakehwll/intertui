package intercept

import (
	"bufio"
	"context"
	"encoding/json"
	"net"
	"testing"
	"time"
)

func TestClientTCPLoginConnectedEvent(t *testing.T) {
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	defer ln.Close()

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
				_, _ = conn.Write([]byte(`{"event":"auth","success":true,"token":"tok","player":"alice"}` + "\n"))
			case "connect":
				_, _ = conn.Write([]byte(`{"event":"connected","msg":"welcome to the game"}` + "\n"))
			}
		}
	}()

	c := NewTCP(ln.Addr().String(), Credentials{User: "alice", Pass: "secret"})
	if err := c.Start(context.Background()); err != nil {
		t.Fatalf("Start: %v", err)
	}
	defer c.Close()
}

func TestClientTCPLogin(t *testing.T) {
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	defer ln.Close()

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
				_, _ = conn.Write([]byte(`{"event":"auth","success":true,"token":"tok","player":"alice"}` + "\n"))
			case "connect":
				_, _ = conn.Write([]byte(`{"event":"connect","success":true}` + "\n"))
			case "command":
				_, _ = conn.Write([]byte(`{"event":"command","success":true,"cmd":"help","msg":"mock help"}` + "\n"))
			}
		}
	}()

	c := NewTCP(ln.Addr().String(), Credentials{User: "alice", Pass: "secret"})
	if err := c.Start(context.Background()); err != nil {
		t.Fatalf("Start: %v", err)
	}
	defer c.Close()

	if c.User() != "alice" {
		t.Fatalf("user = %q", c.User())
	}

	c.SendCommand("help")
	select {
	case msg := <-c.Messages():
		line, ok := msg.(GameLineMsg)
		if !ok || line.Line == "" {
			t.Fatalf("unexpected msg: %#v", msg)
		}
	case <-time.After(5 * time.Second):
		t.Fatal("timeout")
	}
}
