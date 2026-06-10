package intercept

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"

	"github.com/coder/websocket"
	"github.com/coder/websocket/wsjson"
)

// mockResponses returns JSON frames for a mock Intercept request.
func mockResponses(req map[string]any) [][]byte {
	request, _ := req["request"].(string)
	var out []map[string]any

	switch request {
	case "auth":
		if login, ok := req["login"].(map[string]any); ok {
			user, _ := login["username"].(string)
			out = append(out, map[string]any{
				"event":   "auth",
				"success": true,
				"token":   "mock-token",
				"player":  user,
			})
			break
		}
		if token, ok := req["token"].(string); ok && token != "" {
			out = append(out, map[string]any{
				"event":   "auth",
				"success": true,
				"player":  "offline",
			})
			break
		}
		out = append(out, map[string]any{
			"event":   "auth",
			"success": true,
			"token":   "mock-token",
			"player":  "offline",
		})

	case "systems":
		out = append(out, map[string]any{
			"event":   "systems",
			"success": true,
			"systems": []map[string]string{
				{"id": "mock-main", "ip": "127.0.0.1", "hostname": "mock", "type": "main"},
			},
		})

	case "connect":
		out = append(out, map[string]any{
			"event":    "connected",
			"hostname": "mock",
			"user":     "offline",
			"access":   true,
		})
		out = append(out, map[string]any{
			"event": "broadcast",
			"msg":   "Welcome to offline Intercept.",
		})

	case "command":
		cmd, _ := req["cmd"].(string)
		first := cmd
		if i := strings.IndexByte(cmd, ' '); i >= 0 {
			first = cmd[:i]
		}
		out = append(out, map[string]any{
			"event":   "command",
			"success": true,
			"cmd":     first,
			"msg":     "mock response for: " + cmd,
		})
	}

	frames := make([][]byte, 0, len(out))
	for _, resp := range out {
		raw, err := json.Marshal(resp)
		if err != nil {
			continue
		}
		frames = append(frames, raw)
	}
	return frames
}

// StartMockServer runs a minimal Intercept-compatible WebSocket server for tests.
func StartMockServer() (*httptest.Server, string) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		conn, err := websocket.Accept(w, r, nil)
		if err != nil {
			return
		}
		defer conn.Close(websocket.StatusNormalClosure, "")

		for {
			var req map[string]any
			if err := wsjson.Read(r.Context(), conn, &req); err != nil {
				return
			}

			for _, raw := range mockResponses(req) {
				_ = wsjson.Write(r.Context(), conn, json.RawMessage(raw))
			}
		}
	}))

	wsURL := "ws" + strings.TrimPrefix(srv.URL, "http")
	return srv, wsURL
}
