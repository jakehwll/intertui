package intercept

import (
	"encoding/json"
	"testing"
)

func TestMockResponsesAuth(t *testing.T) {
	t.Parallel()

	frames := mockResponses(map[string]any{
		"request": "auth",
		"login": map[string]any{
			"username": "alice",
			"password": "secret",
		},
	})
	if len(frames) != 1 {
		t.Fatalf("frames = %d, want 1", len(frames))
	}

	var env Envelope
	if err := json.Unmarshal(frames[0], &env); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if env.Event != "auth" || !env.Success {
		t.Fatalf("auth frame = %+v", env)
	}
	if env.PlayerName() != "alice" {
		t.Fatalf("player = %q", env.PlayerName())
	}
}

func TestMockResponsesCommand(t *testing.T) {
	t.Parallel()

	frames := mockResponses(map[string]any{
		"request": "command",
		"cmd":     "help topics",
	})
	if len(frames) != 1 {
		t.Fatalf("frames = %d, want 1", len(frames))
	}

	var env Envelope
	if err := json.Unmarshal(frames[0], &env); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if env.Event != "command" || env.Cmd != "help" {
		t.Fatalf("command frame = %+v", env)
	}
}
