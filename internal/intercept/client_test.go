package intercept

import (
	"context"
	"testing"
	"time"
)

func TestClientOfflineLogin(t *testing.T) {
	srv, url := StartMockServer()
	defer srv.Close()

	c := NewWebSocket(url, Credentials{User: "test", Pass: "test"})
	if err := c.Start(context.Background()); err != nil {
		t.Fatalf("Start: %v", err)
	}
	defer c.Close()

	if c.User() != "test" {
		t.Fatalf("user = %q", c.User())
	}

	c.SendCommand("help")

	select {
	case msg := <-c.Messages():
		line, ok := msg.(GameLineMsg)
		if !ok {
			t.Fatalf("expected GameLineMsg, got %T", msg)
		}
		if line.Line == "" {
			t.Fatal("empty line")
		}
	case <-time.After(5 * time.Second):
		t.Fatal("timed out waiting for command response")
	}
}
