//go:build !js

package app

import "wwhsrt/pkg/ui"

func (h *host) reviewIdleLine(g *Game, id string) {
	if !ui.ReviewIdle(&g.State, g.Frame, id) {
		return
	}
	if h.scroll != nil {
		h.scroll.ScrollTop = 0
	}
	h.layout(g)
	g.focusIdle(id)
	g.dirty = true
	h.persist(g)
}
