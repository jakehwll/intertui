package config

import (
	"fmt"
	"testing"

	"intertui/internal/constants"
)

func TestResolveAddr(t *testing.T) {
	t.Parallel()

	cfg := Config{Server: "example.com"}
	want := "example.com:13373"
	if got := cfg.ResolveAddr(); got != want {
		t.Fatalf("ResolveAddr() = %q, want %q", got, want)
	}

	cfg.Port = 9999
	if got := cfg.ResolveAddr(); got != "example.com:9999" {
		t.Fatalf("ResolveAddr() = %q", got)
	}
}

func TestResolveURL(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		cfg  Config
		want string
	}{
		{
			name: "explicit url",
			cfg:  Config{URL: "wss://example.com/custom"},
			want: "wss://example.com/custom",
		},
		{
			name: "ws custom port",
			cfg:  Config{Server: "example.com", WS: true, Port: constants.DEFAULT_PORT},
			want: fmt.Sprintf("ws://example.com:%d/ws", constants.DEFAULT_PORT),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.cfg.ResolveURL(); got != tt.want {
				t.Fatalf("ResolveURL() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestNewClientTransport(t *testing.T) {
	t.Parallel()

	tcp := Config{Server: "example.com"}.NewClient()
	ws := Config{Server: "example.com", WS: true}.NewClient()
	mock := Config{Offline: true}.NewClient()

	if tcp == nil || ws == nil || mock == nil {
		t.Fatal("expected clients")
	}
}
