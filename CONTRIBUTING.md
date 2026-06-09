# Contributing to intertui

Thanks for your interest in contributing. This is a small Go project; keeping changes focused and easy to review helps everyone.

## Getting started

1. Fork and clone the repository.
2. Install [Go 1.25+](https://go.dev/dl/).
3. Build and run:

   ```bash
   go build -o intertui .
   ./intertui init --server HOST --user YOU --pass SECRET   # or --offline
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

### Package layout

| Package | Role |
|---------|------|
| `internal/constants` | Default server host and port |
| `internal/config` | CLI flags and environment variables |
| `internal/intercept` | Network client, protocol parsing, mock server |
| `internal/ui` | Bubble Tea model, view, input handling |
| `cmd/probe` | Manual protocol debugging (not required for normal use) |

### Defaults and configuration

Put shared default values in `internal/constants/constants.go` (`DEFAULT_PORT`, etc.). Do not duplicate magic strings in config or client code.

### Offline development

Use `--offline` or the mock server in `internal/intercept/mock.go` when you do not have access to a live game server:

```bash
go run . --offline
```

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
