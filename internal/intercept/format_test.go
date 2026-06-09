package intercept

import (
	"strings"
	"testing"
)

func TestClean(t *testing.T) {
	t.Parallel()

	tests := []struct {
		in, want string
	}{
		{"hello", "hello"},
		{"¬whello", "hello"},
		{"¬Rred¬* normal", "red normal"},
		{"no codes here", "no codes here"},
	}

	for _, tt := range tests {
		if got := Clean(tt.in); got != tt.want {
			t.Errorf("Clean(%q) = %q, want %q", tt.in, got, tt.want)
		}
	}
}

func TestANSI(t *testing.T) {
	t.Parallel()

	got := ANSI("¬g[¬*¬GSUCCESS¬*¬g]¬* done")
	if strings.Contains(got, "¬") {
		t.Fatalf("leftover codes in %q", got)
	}
	if !strings.Contains(got, "SUCCESS") || !strings.Contains(got, "done") {
		t.Fatalf("got %q", got)
	}
	if !strings.Contains(got, "\033[92m") {
		t.Fatalf("expected green ANSI, got %q", got)
	}
	if !strings.HasSuffix(got, ansiReset) {
		t.Fatalf("expected trailing reset, got %q", got)
	}
}
