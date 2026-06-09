package intercept

import (
	"strings"
	"testing"
)

func TestParseEnvelope(t *testing.T) {
	t.Parallel()

	raw := []byte(`{"event":"chat","msg":"(  ) bob: ¬whello"}`)
	env, err := ParseEnvelope(raw)
	if err != nil {
		t.Fatal(err)
	}
	if env.Event != "chat" {
		t.Fatalf("event = %q", env.Event)
	}

	line, ok := env.DisplayLine()
	if !ok {
		t.Fatal("expected display line")
	}
	if strings.Contains(line, "¬") {
		t.Fatalf("leftover color codes in %q", line)
	}
	if !strings.Contains(line, "hello") {
		t.Fatalf("line = %q", line)
	}
}

func TestPlayerName(t *testing.T) {
	t.Parallel()

	env, err := ParseEnvelope([]byte(`{"event":"auth","player":"alice"}`))
	if err != nil {
		t.Fatal(err)
	}
	if env.PlayerName() != "alice" {
		t.Fatalf("got %q", env.PlayerName())
	}

	env2, err := ParseEnvelope([]byte(`{"event":"connect","player":{"ip":"1.2.3.4","conn":"bob"}}`))
	if err != nil {
		t.Fatal(err)
	}
	if env2.PlayerName() != "bob" {
		t.Fatalf("got %q", env2.PlayerName())
	}
}

func TestIsSilent(t *testing.T) {
	t.Parallel()

	silent := []string{"info", "auth", "systems", "cfg"}
	for _, event := range silent {
		if !(Envelope{Event: event}).IsSilent() {
			t.Fatalf("event %q should be silent", event)
		}
	}

	loud := []string{"chat", "command", "clink", "broadcast"}
	for _, event := range loud {
		if (Envelope{Event: event}).IsSilent() {
			t.Fatalf("event %q should not be silent", event)
		}
	}
}

func TestConnectOK(t *testing.T) {
	t.Parallel()

	if !(Envelope{Event: "connected"}).ConnectOK() {
		t.Fatal("connected should be ok")
	}
	if !(Envelope{Event: "connect", Success: true}).ConnectOK() {
		t.Fatal("connect success should be ok")
	}
	if (Envelope{Event: "connect", Success: false}).ConnectOK() {
		t.Fatal("connect fail should not be ok")
	}
}
