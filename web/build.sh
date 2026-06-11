#!/bin/bash
set -euo pipefail

ROOT="$(cd "$(dirname "$0")/.." && pwd)"
WEB="$(cd "$(dirname "$0")" && pwd)"
cd "$ROOT"

echo "Vendoring dependencies for WASM patches..."
go mod vendor

PATCH_FILE="vendor/charm.land/bubbletea/v2/tea_js.go"
echo "Patching Bubble Tea v2 for WASM..."
cat << 'EOF' > "$PATCH_FILE"
//go:build js || wasm

package tea

const suspendSupported = false

func suspendProcess() {}

func (p *Program) initInput() error { return nil }

func (p *Program) listenForResize(done chan struct{}) { close(done) }
EOF

TEA_GO="vendor/charm.land/bubbletea/v2/tea.go"
CURSED_GO="vendor/charm.land/bubbletea/v2/cursed_renderer.go"
UV_GO="vendor/github.com/charmbracelet/ultraviolet/terminal_renderer.go"

echo "Patching Bubble Tea renderer for xterm.js..."
perl -pi -e 's/mapNl := runtime\.GOOS != "windows" && p\.ttyInput == nil/mapNl := false; _ = runtime.GOOS \/\/ xterm.js: use CSI cursor moves/' "$TEA_GO"
perl -0777 -pi -e 's/func shouldQuerySynchronizedOutput\(environ uv\.Environ\) bool \{.*?\n\}/func shouldQuerySynchronizedOutput(environ uv.Environ) bool {\n\treturn false\n}/s' "$TEA_GO"
perl -pi -e 's/\/\/ Clear our screen buffer before copying/s.scr.Erase() \/\/ xterm.js: avoid ghost cells\n\t\/\/ Clear our screen buffer before copying/' "$CURSED_GO"
perl -0777 -pi -e 's/\t\t\tif !s\.flags\.Contains\(tFullscreen\) \|\| n < len\(yseq\) \{.*?\n\t\t\t\}//s' "$UV_GO"

GOROOT_WASM="$(go env GOROOT)/lib/wasm/wasm_exec.js"
if [[ ! -f "$GOROOT_WASM" ]]; then
	GOROOT_WASM="$(go env GOROOT)/misc/wasm/wasm_exec.js"
fi

echo "Copying wasm_exec.js..."
cp "$GOROOT_WASM" "$WEB/wasm_exec.js"

echo "Building intertui.wasm..."
GOOS=js GOARCH=wasm go build -mod=vendor -o "$WEB/intertui.wasm" ./cmd/wasm

echo "Cleaning up vendor..."
rm -rf vendor

echo "Done. Run: go run ./web/serve.go"
