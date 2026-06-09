//go:build live

package intercept

import (
	"context"
	"net"
	"os"
	"strconv"
	"testing"

	"intertui/internal/constants"
)

func TestLiveTCPLogin(t *testing.T) {
	user := os.Getenv("INTERCEPT_USER")
	pass := os.Getenv("INTERCEPT_PASS")
	server := os.Getenv("INTERCEPT_SERVER")
	if user == "" || pass == "" || server == "" {
		t.Skip("set INTERCEPT_USER, INTERCEPT_PASS, and INTERCEPT_SERVER")
	}

	addr := net.JoinHostPort(server, strconv.Itoa(constants.DEFAULT_PORT))
	c := NewTCP(addr, Credentials{User: user, Pass: pass})
	c.SetStatus(func(s string) { t.Log(s) })
	if err := c.Start(context.Background()); err != nil {
		t.Fatal(err)
	}
	defer c.Close()
}
