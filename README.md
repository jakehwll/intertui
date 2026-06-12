```
▗▄▄▄▖▗▖  ▗▖▗▄▄▄▖▗▄▄▄▖▗▄▄▖▗▄▄▄▖▗▖ ▗▖▗▄▄▄▖
  █  ▐▛▚▖▐▌  █  ▐▌   ▐▌ ▐▌ █  ▐▌ ▐▌  █  
  █  ▐▌ ▝▜▌  █  ▐▛▀▀▘▐▛▀▚▖ █  ▐▌ ▐▌  █  
▗▄█▄▖▐▌  ▐▌  █  ▐▙▄▄▖▐▌ ▐▌ █  ▝▚▄▞▘▗▄█▄▖
```

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

## Development

See [CONTRIBUTING.md](CONTRIBUTING.md).

## License

GNU General Public License v3.0 or later. See [LICENSE](LICENSE).

## Acknowledgements

- [Intercept](https://bubmet.itch.io/intercept) by bubmet
- [intercept.py](https://github.com/Martmists-GH/intercept.py) for protocol reference
- [Charm](https://charm.sh/) — Bubble Tea, Bubbles, Lip Gloss
