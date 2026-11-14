//go:build !js

package app

import (
	"github.com/ebitenui/ebitenui/widget"
	"image"
	"wwhsrt/pkg/gameapi"
	"wwhsrt/pkg/ui"
)

func (h *host) helpEvent(g *Game, e ui.HelpEvent) {
	if e.Kind == "open" && h.scroll != nil {
		e.Scroll = h.scroll.ScrollTop
	}
	h.blurName()
	if h.helpSearch != nil {
		h.helpSearch.Focus(false)
	}
	ui.ApplyHelpEvent(&g.State, e)
	if e.Kind == "close" {
		h.helpRestore = true
		h.helpFocus = g.State.Help.ReturnAnchor
	} else if e.Kind == "open" || e.Kind == "task" {
		h.scroll = nil
		if e.Kind == "task" {
			h.helpFocus = g.State.Help.ReturnAnchor
			h.helpFocusScroll = true
		}
	}
	h.layout(g)
	g.dirty = true
	h.persist(g)
}
func (h *host) helpModel(g *Game) ui.HelpModel {
	var forecast *gameapi.Forecast
	issue := h.message
	if g.State.View == "services" || g.State.Help.Open {
		f, e := g.Port.Forecast(
			gameapi.ForecastRequest{
				CorridorID:       g.State.ServiceCorridor,
				Frequency:        g.State.Frequency,
				Fare:             g.State.Fare,
				ReplaceServiceID: g.State.ReplaceServiceID,
			},
		)
		if e == nil || g.State.Help.Topic == "capacity" {
			forecast = &f
		}
		if e != nil {
			issue = e.Error()
		}
	}
	return ui.BuildHelpModel(g.Frame, g.State, forecast, issue)
}
func (h *host) helpSlot(g *Game, anchor string, label nativeLabel, button nativeButton) {
	m := h.helpModel(g)
	fallback := "city"
	switch g.State.View {
	case "services":
		fallback = "service-plan"
		if ui.ServicePane(g.State) == "fleet" {
			fallback = "fleet"
		}
		if ui.ServicePane(g.State) == "analysis" {
			fallback = "capacity"
		}
	case "build":
		fallback = "endpoints"
		if ui.BuildPane(g.State) == "costs" {
			fallback = "engineering"
		}
	case "company":
		fallback = "finance"
	case "report":
		fallback = "accounts"
	}
	active := m.Cue.Anchor
	if active == "" {
		active = fallback
	}
	if anchor != active && anchor != g.State.Help.Expanded && anchor != "expanded-endpoint" {
		return
	}
	if h.helpAnchors == nil {
		h.helpAnchors = map[string]*widget.Button{}
	}
	for _, slot := range m.Slots {
		if slot.Anchor != anchor {
			continue
		}
		slot := slot
		t := func(k string) string { return ui.Text(g.State.Locale, k) }
		if anchor == "expanded-endpoint" {
			if m.Cue.Concept == slot.Topic.ID {
				label(t(slot.Topic.SummaryKey), 14)
				button(
					t("dismissConcept"),
					func() { h.helpEvent(g, ui.HelpEvent{Kind: "dismiss", Concept: slot.Topic.ID}) },
				)
				h.helpCue = slot.Topic.ID
				h.helpCueAnchor = anchor
			}
			b := button(
				t("fieldNotes"),
				func() { h.helpEvent(g, ui.HelpEvent{Kind: "open", Concept: slot.Topic.ID, Anchor: anchor}) },
			)
			h.helpAnchors[anchor] = b
			continue
		}
		toggle := func() {
			kind := "expand"
			if g.State.Help.Expanded == anchor {
				kind = "collapse"
			}
			h.helpEvent(g, ui.HelpEvent{Kind: kind, Concept: slot.Topic.ID, Anchor: anchor})
		}
		var b *widget.Button
		if m.Cue.Anchor == anchor && m.Cue.Concept == slot.Topic.ID {
			label(t(slot.Topic.SummaryKey), 13)
			h.row(2, func() {
				b = button(t("explainShort"), toggle)
				button(
					t("dismissShort"),
					func() { h.helpEvent(g, ui.HelpEvent{Kind: "dismiss", Concept: slot.Topic.ID}) },
				)
			})
			h.helpCue = slot.Topic.ID
			h.helpCueAnchor = anchor
		} else {
			b = button(t("explainShort")+" "+t(slot.Topic.TitleKey), toggle)
		}
		h.helpAnchors[anchor] = b

		if g.State.Help.Expanded == anchor {
			label(t(slot.Topic.DetailKey), 14)
			for _, row := range slot.Facts {
				label(row.Label+"\n"+row.Value, 14)
			}
			button(
				t("fieldNotes"),
				func() { h.helpEvent(g, ui.HelpEvent{Kind: "open", Concept: slot.Topic.ID, Anchor: anchor}) },
			)
		}
	}
}
func (h *host) buildReference(g *Game, label nativeLabel, button nativeButton, content *widget.Container) {
	t := func(k string) string { return ui.Text(g.State.Locale, k) }
	button(t("returnTask"), func() { h.helpEvent(g, ui.HelpEvent{Kind: "close"}) })
	label(t("fieldNotes"), 22)
	label(t("searchHelp"), 16)
	face := h.face(g, 16)
	h.helpSearch = newNameInput(&h.helpSearchField, g.State.Help.Query, face, ui.PaletteFor(g.State))
	h.helpSearch.fit(h.text)
	content.AddChild(h.helpSearch)
	m := h.helpModel(g)
	if len(m.Results) == 0 {
		label(t("noHelpResults"), 14)
	}
	if m.Article.ID != "" {
		label(t(m.Article.TitleKey), 20)
		label(t(m.Article.SummaryKey), 15)
		label(t(m.Article.DetailKey), 14)
		for _, row := range m.ArticleFacts {
			label(row.Label+"\n"+row.Value, 14)
		}
		progress := g.State.Help.Concepts[m.Article.ID]
		if progress.Seen {
			label(t("helpProgressSeen"), 12)
		}
		if progress.Dismissed {
			label(t("helpProgressDismissed"), 12)
		}
		if progress.Completed {
			label(t("helpProgressCompleted"), 12)
		}
		id := m.Article.ID
		button(t("goToTask"), func() { h.helpEvent(g, ui.HelpEvent{Kind: "task", Concept: id}) })
	}
	for _, topic := range m.Results {
		topic := topic
		button(t(topic.TitleKey), func() { h.helpEvent(g, ui.HelpEvent{Kind: "topic", Concept: topic.ID}) })
	}
}

// Mark only a cue whose actual widget intersects the rendered scroll viewport.
func (h *host) observeHelp(g *Game) {
	if h.helpCue == "" || g.State.Help.Open {
		return
	}
	w := h.helpAnchors[h.helpCueAnchor]
	if w == nil {
		return
	}
	viewport := image.Rect(0, 0, g.width, g.height)
	if !g.State.Expanded && h.scroll != nil {
		viewport = h.scroll.ViewRect()
	}
	if !w.GetWidget().Rect.Intersect(viewport).Empty() && !g.State.Help.Concepts[h.helpCue].Seen {
		ui.ApplyHelpEvent(&g.State, ui.HelpEvent{Kind: "seen", Concept: h.helpCue})
		h.persist(g)
	}
}
