//go:build js && wasm

package app

// Browser window placement belongs to the browser, not the game iframe.
func (g *Game) ConfigureWindow()          {}
func (g *Game) FlushWindowPreferences()   {}
func (h *host) windowUpdate(g *Game) bool { return false }
