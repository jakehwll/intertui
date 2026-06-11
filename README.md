# intertui

A terminal client for [Intercept](https://bubmet.itch.io/intercept), the hacking MUD by [bubmet](https://github.com/bubmet). Built with [Bubble Tea](https://github.com/charmbracelet/bubbletea).

This project is unofficial and not affiliated with the game or its authors.

![intertui preview](.github/assets/preview.png)

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
# New account (registers on server + writes ~/.intertui/config.yaml)
intertui register --server HOST --user YOU --pass SECRET

# Or one-time setup if you already have an account
intertui init --server HOST --user YOU --pass SECRET

# Start the intertui UI
intertui
```

### Flags

| Flag | Description |
|------|-------------|
| `--user` | Intercept username |
| `--pass` | Intercept password |

### Environment variables

| Variable | Flag |
|----------|------|
| `INTERCEPT_USER` | `--user` |
| `INTERCEPT_PASS` | `--pass` |


## Configuration

Default settings live in `~/.intertui/config.yaml` (create with `intertui register` or `intertui init`). `--user` and `--pass` override the file for a single session.

Session logs are written to `~/.intertui/logs/latest.log`.

## Web client (self-hosted)

The same TUI runs in the browser via WebAssembly and [xterm.js](https://xtermjs.org/). Live play uses **Socket.IO** (port **13370**) proxied through your web server so the browser never needs CORS access to the game host.

```bash
./scripts/build-wasm.sh
go run ./web/serve.go
# open http://localhost:8080/?server=GAME_HOST&user=YOU&pass=SECRET
```

| Query param | Description |
|-------------|-------------|
| *(none)* | Offline mock (default) |
| `server` | Game host (required for live play) |
| `user`, `pass` | Credentials (or `token`) |
| `port` | Socket.IO port (default `13370`) |
| `tls=1` | Use `https://` to reach the game server |
| `direct=1` | Skip proxy; browser connects to game host directly (needs CORS on the server) |
| `ws=1` | Plain WebSocket instead of Socket.IO |
| `url` | Full Socket.IO or WebSocket URL override |

Pin the upstream without URL params: `INTERTUI_PROXY=http://game:13370 go run ./web/serve.go`

Put Caddy or nginx in front of `serve.go` for TLS in production. Avoid putting passwords in query strings on shared links — use a config file or auth in front of the reverse proxy.

Pushes to `main` can also deploy `web/` to GitHub Pages via [`.github/workflows/pages.yml`](.github/workflows/pages.yml).

## Development

See [CONTRIBUTING.md](CONTRIBUTING.md).

## License

GNU General Public License v3.0 or later. See [LICENSE](LICENSE).

## Acknowledgements

- [Intercept](https://bubmet.itch.io/intercept) by bubmet
- [intercept.py](https://github.com/Martmists-GH/intercept.py) for protocol reference
- [Charm](https://charm.sh/) — Bubble Tea, Bubbles, Lip Gloss
