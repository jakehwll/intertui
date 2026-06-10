# intertui

A terminal client for [Intercept](https://bubmet.itch.io/intercept), the hacking MUD by [bubmet](https://github.com/bubmet). Built with [Bubble Tea](https://github.com/charmbracelet/bubbletea).

This project is unofficial and not affiliated with the game or its authors.

![intertui preview](.github/assets/preview.png)

## Features

- Full-screen TUI with scrollable game log and command input
- ANSI colors for in-game `¬` color codes
- TCP transport (default) with optional WebSocket mode
- Username/password login (TCP) or extended WebSocket login flows
- Offline mode with a built-in in-process mock server
- Browser build (WASM + xterm.js) for GitHub Pages

## Requirements

- Go 1.25 or later
- A terminal that supports ANSI colors and alternate screen mode

## Install

```bash
go install github.com/jakehwll/intertui@latest
```

Or build from source:

```bash
git clone https://github.com/jakehwll/intertui.git
cd intertui
go build -o intertui .
```

## Usage

```bash
# One-time setup (writes ~/.intertui/config.yaml)
intertui init --server HOST --user YOU --pass SECRET

# Connect using config file
intertui

# Or pass flags (override config)
intertui --server HOST --user YOU --pass SECRET

# Offline mock server (no network)
./intertui --offline

# WebSocket (alternate API; not what the live server uses by default)
./intertui --ws --user YOU --pass SECRET

# Custom host
./intertui --server example.com --port 13373 --user YOU --pass SECRET
```

### Flags

| Flag | Description |
|------|-------------|
| `--user` | Intercept username |
| `--pass` | Intercept password |
| `--token` | API token (WebSocket mode only) |
| `--server` | Game server host |
| `--port` | Server port (default: `13373`) |
| `--ws` | Use WebSocket instead of TCP |
| `--tls` | Use `wss://` instead of `ws://` (with `--ws`) |
| `--url` | Full WebSocket URL (overrides `--server` / `--port`) |
| `--offline` | Built-in in-process mock server |

### Environment variables

| Variable | Flag |
|----------|------|
| `INTERCEPT_USER` | `--user` |
| `INTERCEPT_PASS` | `--pass` |
| `INTERCEPT_TOKEN` | `--token` |
| `INTERCEPT_SERVER` | `--server` |
| `INTERCEPT_PORT` | `--port` |
| `INTERCEPT_WS` | `--ws` (`1` or `true`) |
| `INTERCEPT_TLS` | `--tls` (`1` or `true`) |
| `INTERCEPT_URL` | `--url` |

### Keyboard shortcuts

| Key | Action |
|-----|--------|
| `Enter` | Send command |
| `↑` / `↓` | Scroll log |
| `PgUp` / `PgDn` | Page scroll |
| `Ctrl+U` / `Ctrl+D` | Half-page scroll |
| `Ctrl+P` / `Ctrl+N` | Command history |
| `r` | Reconnect (after disconnect or failed login) |
| `Ctrl+C` / `Esc` | Quit |
| Mouse wheel | Scroll log |

## Configuration

Default settings live in `~/.intertui/config.yaml` (create with `intertui init`). Flags and environment variables override the file. Default port is in [`internal/constants/constants.go`](internal/constants/constants.go) (`DEFAULT_PORT`). WebSocket URLs are derived as `ws://host:port/ws` unless you pass `--url`.

Session logs are written to `~/.intertui/logs/latest.log`. On each launch, the previous `latest.log` is renamed to a timestamped file in the same directory (for example `2025-06-10T12-34-56.log`).

## Web (GitHub Pages)

The browser build runs the same Bubble Tea TUI in WASM with [xterm.js](https://xtermjs.org/) as the terminal.

```bash
./scripts/build-wasm.sh
go run ./web/serve.go
# open http://localhost:8080
```

Pushes to `main` deploy `web/` to GitHub Pages via [`.github/workflows/pages.yml`](.github/workflows/pages.yml). Enable **Pages → Source: GitHub Actions** in the repo settings once.

### URL parameters

| Param | Description |
|-------|-------------|
| *(none)* | Offline mock (default) |
| `offline=0` | Require live connection params |
| `ws=1` | Use WebSocket transport |
| `server` | Game server host |
| `port` | Server port |
| `user` / `pass` | Login credentials |
| `token` | API token (WebSocket) |
| `tls=1` | Use `wss://` |
| `url` | Full WebSocket URL |

Example live play: `?ws=1&server=example.com&user=YOU&pass=SECRET`

TCP is not available in the browser. Live WebSocket may fail if the game server blocks cross-origin browser connections.

## Development

See [CONTRIBUTING.md](CONTRIBUTING.md).

## Protocol

JSON request/response protocol, informed by [intercept.py](https://github.com/Martmists-GH/intercept.py):

- **TCP (default):** one JSON object per line on port `13373` — `auth` (login) → `connect` (token)
- **WebSocket:** JSON frames — `auth` → `systems` → `connect` (system)

Inbound events include `chat`, `broadcast`, `command`, `connect`, and others. Commands are sent as `{"request":"command","cmd":"..."}`.

## Project layout

```
intertui/
  main.go
  internal/
    app/           # shared TUI runner
    constants/     # DEFAULT_PORT
    config/        # flags and env
    intercept/     # protocol client
    ui/            # Bubble Tea TUI
    wasmio/        # xterm.js bridge (WASM only)
  cmd/
    probe/         # optional protocol debugger
    wasm/          # browser entrypoint
  web/             # index.html + local dev server
  scripts/         # build-wasm.sh
```

## License

GNU General Public License v3.0 or later. See [LICENSE](LICENSE).

## Acknowledgements

- [Intercept](https://bubmet.itch.io/intercept) by bubmet
- [intercept.py](https://github.com/Martmists-GH/intercept.py) for protocol reference
- [Charm](https://charm.sh/) — Bubble Tea, Bubbles, Lip Gloss
