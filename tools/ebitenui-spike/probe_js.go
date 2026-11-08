//go:build js && wasm

package main

import "syscall/js"

var probe js.Func

// Read-only diagnostics allow browser checks to target real canvas widgets.
// All actions still pass through pointer/touch/keyboard events and ebitenui.
func installProbe(g *game) {
	probe = js.FuncOf(func(js.Value, []js.Value) any { return g.snapshot() })
	js.Global().Set("wwhsrtSpikeSnapshot", probe)
}
