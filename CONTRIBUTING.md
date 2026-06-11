# Contributing to intertui

Thanks for your interest in contributing. This is a small Go project; keeping changes focused and easy to review helps everyone.

## Getting started

1. Fork and clone the repository.
2. Install [Go 1.25+](https://go.dev/dl/).
3. Build and run:

   ```bash
   go build -o intertui .
   ./intertui init --server HOST --user YOU --pass SECRET
   ./intertui
   ```

4. Run the test suite:

   ```bash
   go test ./...
   ```

## What to work on

Good first contributions:

- UI polish (layout, accessibility, keyboard UX)
- Protocol edge cases (new event types, better error messages)
- Tests and fixtures for the intercept client
- Documentation fixes

Check open issues before starting large changes. For substantial features, open an issue first so we can align on approach.

## Development notes

### Project layout

```
intertui/
  main.go
  cmd/
    dev/           # offline mock server + TUI (not installed)
    probe/         # low-level protocol debugger
    wasm/          # browser build entrypoint
  internal/browser/  # browser runtime (WASM-only; not compiled into desktop binary)
  web/             # browser shell, proxy, WASM build — see web/README.md
  internal/
    constants/     # DEFAULT_PORT
    config/        # CLI flags and config file
    intercept/     # protocol client and mock server
    ui/            # Bubble Tea TUI
```

| Package | Role |
|---------|------|
| `internal/constants` | Default server host and port |
| `internal/config` | CLI flags and environment variables |
| `internal/intercept` | Network client, protocol parsing, mock server |
| `internal/ui` | Bubble Tea model, view, input handling |
| `cmd/dev` | Offline development entrypoint |
| `cmd/probe` | Manual protocol debugging |

### Defaults and configuration

Put shared default values in `internal/constants/constants.go` (`DEFAULT_PORT`, etc.). Do not duplicate magic strings in config or client code.

Connection settings (`server`, `port`, transport mode) live in `~/.intertui/config.yaml` and are written by `intertui init`. The main binary only exposes `--user` and `--pass` on the command line; everything else is config-file driven.

### Offline development

Run the dev entrypoint when you do not have access to a live game server:

```bash
go run ./cmd/dev
```

This starts the built-in mock WebSocket server in `internal/intercept/mock.go` and launches the TUI against it. It is not part of the installed `intertui` binary.

### Alternate transports and servers

The live game server uses raw TCP on port `13373` by default. WebSocket mode and custom hosts are for development and testing.

Set transport options in config via `intertui init`:

```bash
# Custom host or port
./intertui init --server example.com --port 13373 --user YOU --pass SECRET --force

# WebSocket (alternate API; not what the live server uses by default)
./intertui init --ws --user YOU --pass SECRET --server HOST --force

# Full WebSocket URL
./intertui init --url wss://example.com/ws --user YOU --pass SECRET --force
```

`init` also accepts `--tls` (with WebSocket) and `--token` (API token login for WebSocket mode).

### Protocol

JSON request/response protocol, informed by [intercept.py](https://github.com/Martmists-GH/intercept.py):

- **TCP (default):** one JSON object per line on port `13373` — `auth` (login) → `connect` (token)
- **WebSocket:** JSON frames — `auth` → `systems` → `connect` (system)

Inbound events include `chat`, `broadcast`, `command`, `connect`, and others. Commands are sent as `{"request":"command","cmd":"..."}`.

WebSocket URLs are derived as `ws://host:port/ws` unless `url` is set in config.

### Live server tests

Live tests are behind the `live` build tag and require real credentials:

```bash
INTERCEPT_SERVER=host INTERCEPT_USER=you INTERCEPT_PASS=secret \
  go test -tags live ./internal/intercept/ -run TestLiveTCPLogin -v
```

Do not commit credentials. Use environment variables only.

### Debug wire logging

Set `INTERCEPT_DEBUG=1` to log raw JSON frames during client development.

## Code style

- Match existing code in the file you are editing (naming, imports, error handling).
- Keep diffs small and purposeful — avoid drive-by refactors.
- Run `go test ./...` and `go build ./...` before opening a PR.
- Run `go fmt ./...` on changed files.
