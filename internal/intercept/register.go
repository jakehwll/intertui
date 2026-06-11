package intercept

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"net"
	"time"
)

// authUserPassPayload builds an auth request for login or register.
func authUserPassPayload(mode, user, pass string) map[string]any {
	return map[string]any{
		"request": "auth",
		mode: map[string]string{
			"username": user,
			"password": pass,
		},
	}
}

// Register creates a new account over TCP and returns the player name.
// It does not connect to the game after auth.
func Register(ctx context.Context, addr, user, pass string) (string, error) {
	if user == "" || pass == "" {
		return "", fmt.Errorf("username and password required")
	}

	dialer := net.Dialer{Timeout: 15 * time.Second}
	conn, err := dialer.DialContext(ctx, "tcp", addr)
	if err != nil {
		return "", fmt.Errorf("dial %s: %w", addr, err)
	}
	defer conn.Close()

	var waiter Waiter
	readDone := make(chan struct{})
	readErr := make(chan error, 1)

	go func() {
		defer close(readDone)
		scanner := bufio.NewScanner(conn)
		scanner.Buffer(make([]byte, 0, 64*1024), 1<<20)
		for scanner.Scan() {
			env, err := ParseEnvelope(scanner.Bytes())
			if err != nil {
				continue
			}
			waiter.Deliver(env)
		}
		if err := scanner.Err(); err != nil {
			readErr <- err
		}
	}()

	match := func(e Envelope) bool { return e.Event == "auth" || e.Event == "error" }
	ch := waiter.Wait(match)

	payload, err := json.Marshal(authUserPassPayload("register", user, pass))
	if err != nil {
		return "", err
	}
	if _, err := conn.Write(append(payload, '\n')); err != nil {
		return "", fmt.Errorf("send auth: %w", err)
	}

	var auth Envelope
	select {
	case auth = <-ch:
	case <-time.After(30 * time.Second):
		return "", fmt.Errorf("timed out waiting for auth response")
	case <-ctx.Done():
		return "", ctx.Err()
	case err := <-readErr:
		if err != nil {
			return "", fmt.Errorf("read: %w", err)
		}
	case <-readDone:
		return "", fmt.Errorf("connection closed before auth response")
	}

	if auth.Event == "error" {
		return "", fmt.Errorf("registration failed: %s", auth.Error)
	}
	if !auth.Success || auth.Token == "" {
		return "", fmt.Errorf("registration failed")
	}

	player := auth.PlayerName()
	if player == "" {
		player = user
	}
	return player, nil
}
