# ebitenui rendering feasibility spike

An isolated Go module demonstrating the proposed service planner and monthly report with **real ebitenui widgets**. It does not implement the game, save format, server, or authoritative economic model.

Pinned dependencies: ebitenui **0.7.3**, Ebitengine **2.10.5**, x/image **0.47.0**. Module sums are retained. Validated using Go **1.27.1** on macOS arm64; the module declares Go 1.26.4, matching the reference application's baseline.

## Build and inspect

From this directory:

```sh
GOMEMLIMIT=768MiB GOMAXPROCS=2 GOOS=js GOARCH=wasm go build -p 1 -o web/main.wasm .
cp "$(go env GOROOT)/lib/wasm/wasm_exec.js" web/wasm_exec.js
python3 -m http.server 8765 --bind 127.0.0.1 --directory web
```

Open `http://127.0.0.1:8765/`. Resize, edit fields, enter an infeasible frequency, dismiss the hint, scroll, and switch to Monthly report. Rendered controls are Go/ebitenui widgets; the page supplies the WebAssembly bootstrap. ebitenui itself creates a hidden browser input for its mobile keyboard bridge.

`go run .` opens a native development window using the same widget construction. The native macOS arm64 binary was compiled during this investigation; its window was not exercised. Generated executables, WASM, and the runtime shim are ignored.

## Reproduce focused checks

With the server running and Playwright/Chromium available:

```sh
node check-browser.cjs http://127.0.0.1:8765/
```

If Playwright is installed elsewhere, set `NODE_PATH` to its `node_modules` directory. The checks operate actual widgets using pointer, keyboard, and CDP touch events. `wwhsrtSpikeSnapshot()` is a read-only probe for geometry/state; it cannot perform actions. These are focused renderer checks, not the game's full test suite.

Evidence: [validation report](../../docs/validation/README.md), [machine-readable results](../../docs/validation/ebitenui-checks.json).

## Application responsibilities

- Responsive compositions, explicit grid stretching, and width-aware text wrapping.
- Theme images, including rounded nine-slices, colors, and typography.
- State preservation across layout rebuilds and in-place field/forecast updates during editing.
- Clipping and separate footer/navigation rectangles, wheel handling, and one-finger drag policy. Scroll gestures do not start on buttons or text fields.
- Platform select-all adaptation. The spike restores the pre-chord value and uses `TextInput.SelectAll()` for Ctrl/Cmd+A, preventing insertion of the shortcut's printable letter.
- Accessible semantics and actual phone lifecycle/keyboard qualification.

The drag policy has no momentum. Physical touch, gestures begun on controls, international input, clipboard shortcuts beyond select-all, software-keyboard viewport changes, background/resume, and screen readers need further qualification. The semantic HTML HUD fallback remains in the product design.
