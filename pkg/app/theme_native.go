//go:build !js

package app

import (
	"github.com/hajimehoshi/ebiten/v2"
	"time"
	"wwhsrt/pkg/ui"
)

func (h *host) refreshAppearance(g *Game) {
	mode := ui.AppearanceMode(g.State.Appearance)
	if mode == h.appearanceMode && time.Since(h.appearancePoll) < time.Second {
		return
	}
	h.appearancePoll = time.Now()
	dark := mode == "dark" || mode == "system" && ebiten.SystemColorMode() == ebiten.ColorModeDark
	if g.State.Dark != dark || mode != h.appearanceMode {
		g.State.Dark = dark
		g.dirty = true
	}
	if mode != h.appearanceMode {
		preferred := ebiten.ColorModeUnknown
		if mode == "light" {
			preferred = ebiten.ColorModeLight
		} else if mode == "dark" {
			preferred = ebiten.ColorModeDark
		}
		ebiten.SetPreferredColorMode(preferred)
		h.appearanceMode = mode
	}
}
