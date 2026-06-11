package ui

import (
	"strings"
	"testing"

	"intertui/internal/config"
)

func TestFooter(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name  string
		model Model
		want  []string
		omit  []string
	}{
		{
			name: "connected shows user and status",
			model: Model{
				cfg:           config.Config{User: "bob"},
				connectedUser: "alice",
				state:         stateConnected,
			},
			want: []string{"alice", "Connected"},
		},
		{
			name: "offline shows configured user",
			model: Model{
				cfg:   config.Config{User: "bob"},
				state: stateError,
			},
			want: []string{"bob", "Offline"},
		},
		{
			name: "connecting without user shows status only",
			model: Model{
				state: stateConnecting,
			},
			want: []string{"Connecting"},
		},
		{
			name: "detach hint replaces username and hides status",
			model: Model{
				cfg:        config.Config{User: "bob"},
				state:      stateConnected,
				detachHint: true,
			},
			want: []string{"Press Ctrl+A, D to detach"},
			omit: []string{"Connected", "bob"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			got := tt.model.footer(80)
			for _, want := range tt.want {
				if !strings.Contains(got, want) {
					t.Fatalf("footer() = %q, want substring %q", got, want)
				}
			}
			for _, omit := range tt.omit {
				if strings.Contains(got, omit) {
					t.Fatalf("footer() = %q, must not contain %q", got, omit)
				}
			}
		})
	}
}
