package intercept

import (
	"net/http"
	"net/http/httptest"
	"strings"

	"github.com/coder/websocket"
	"github.com/coder/websocket/wsjson"
)

// StartMockServer runs a minimal Intercept-compatible WebSocket server for offline dev.
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

			request, _ := req["request"].(string)
			switch request {
			case "auth":
				if login, ok := req["login"].(map[string]any); ok {
					user, _ := login["username"].(string)
					_ = wsjson.Write(r.Context(), conn, map[string]any{
						"event":   "auth",
						"success": true,
						"token":   "mock-token",
						"player":  user,
					})
					continue
				}
				if token, ok := req["token"].(string); ok && token != "" {
					_ = wsjson.Write(r.Context(), conn, map[string]any{
						"event":   "auth",
						"success": true,
						"player":  "offline",
					})
					continue
				}
				_ = wsjson.Write(r.Context(), conn, map[string]any{
					"event":   "auth",
					"success": true,
					"token":   "mock-token",
					"player":  "offline",
				})

			case "systems":
				_ = wsjson.Write(r.Context(), conn, map[string]any{
					"event":   "systems",
					"success": true,
					"systems": []map[string]string{
						{"id": "mock-main", "ip": "127.0.0.1", "hostname": "mock", "type": "main"},
					},
				})

			case "connect":
				_ = wsjson.Write(r.Context(), conn, map[string]any{
					"event":    "connected",
					"hostname": "mock",
					"user":     "offline",
					"access":   true,
				})
				_ = wsjson.Write(r.Context(), conn, map[string]any{
					"event": "broadcast",
					"msg":   "Welcome to offline Intercept.",
				})

			case "command":
				cmd, _ := req["cmd"].(string)
				first := cmd
				if i := strings.IndexByte(cmd, ' '); i >= 0 {
					first = cmd[:i]
				}
				_ = wsjson.Write(r.Context(), conn, map[string]any{
					"event":   "command",
					"success": true,
					"cmd":     first,
					"msg":     "mock response for: " + cmd,
				})
			}
		}
	}))

	wsURL := "ws" + strings.TrimPrefix(srv.URL, "http")
	return srv, wsURL
}
