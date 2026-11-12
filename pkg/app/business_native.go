//go:build !js

package app

import (
	"wwhsrt/pkg/gameapi"
	"wwhsrt/pkg/ui"
)

func (h *host) followGuidance(g *Game, cue ui.Guidance) {
	switch cue.Action {
	case gameapi.AdvanceMonth, gameapi.AdvanceYear:
		h.send(g, cue.Action)
		return
	case "services":
		if cue.CorridorID != "" {
			ui.DraftService(&g.State, g.Frame, cue.CorridorID)
		}
		g.State.View = "services"
	case "company":
		if gap := ui.FundingShortfall(g.Frame, g.State); gap > 0 {
			g.State.LoanAmount = ui.DollarInput(gap)
		}
		g.State.View = "company"
	case "build":
		g.State.View = "build"
	}
	h.blurName()
	g.dirty = true
	h.persist(g)
}

func (h *host) guide(g *Game, label nativeLabel, button nativeButton) {
	if cue := ui.HelpFor(g.Frame, g.State); cue.Key != "" {
		label(ui.Text(g.State.Locale, "nextStep"), 17)
		label(ui.Text(g.State.Locale, cue.Key), 14)
		if cue.Action != "" {
			c := cue
			button(ui.Text(g.State.Locale, "helpAction_"+cue.Action), func() { h.followGuidance(g, c) })
		}
		button(ui.Text(g.State.Locale, "dismiss"), func() { g.State.HintDismissed = true; g.dirty = true; h.persist(g) })
	}
}
