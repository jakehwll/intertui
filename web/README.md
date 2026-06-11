# Browser client

Same intertui TUI in the browser (WASM + [xterm.js](https://xtermjs.org/)). This extends the native client — it is not a separate app.

Live play uses **Socket.IO** on port **13370**, proxied through this server so the browser never needs CORS access to the game host.

## Quick start

```bash
./web/build.sh
go run ./web/serve.go
# http://localhost:8080/?server=GAME_HOST&user=YOU&pass=SECRET
```

## URL parameters

| Param | Description |
|-------|-------------|
| *(none)* | Offline mock (default) |
| `server` | Game host (required for live play) |
| `user`, `pass` | Credentials (or `token`) |
| `port` | Socket.IO port (default `13370`) |
| `tls=1` | Use `https://` to reach the game server |
| `direct=1` | Skip proxy; browser hits game host directly (needs CORS) |
| `ws=1` | Plain WebSocket instead of Socket.IO |
| `url` | Full Socket.IO or WebSocket URL override |

## Proxy

By default the WASM client dials `window.location.origin` and `serve.go` reverse-proxies `/socket.io/` to the game server (upstream read from the page `Referer` and `?server=`).

Pin upstream without URL params:

```bash
INTERTUI_PROXY=http://game:13370 go run ./web/serve.go
```

For production, put Caddy or nginx in front for TLS. Avoid passwords in shared query strings.

## Layout

```
web/
  index.html    # shell + xterm.js
  socketio.js   # Socket.IO bridge for WASM
  serve.go      # static files + Socket.IO proxy
  build.sh      # WASM build (patches Bubble Tea for xterm.js)
cmd/wasm/       # WASM main
internal/browser/  # browser-only runtime (config, Socket.IO, xterm bridge)
```

Pushes to `main` deploy `web/` to GitHub Pages via [`.github/workflows/pages.yml`](../.github/workflows/pages.yml).
