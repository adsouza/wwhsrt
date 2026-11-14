//go:build !js

package app

import (
	"fmt"
	uimage "github.com/ebitenui/ebitenui/image"
	"github.com/ebitenui/ebitenui/widget"
	"github.com/hajimehoshi/ebiten/v2/text/v2"
	"wwhsrt/pkg/gameapi"
	"wwhsrt/pkg/render"
	"wwhsrt/pkg/ui"
)

func (h *host) replayControls(g *Game, label nativeLabel, button nativeButton, content *widget.Container) {
	t := func(key string) string { return ui.Text(g.State.Locale, key) }
	b := button(t("watchDay"), func() {
		g.State.Watching = !g.State.Watching
		g.State.ReplayPlaying = false
		g.replayTick = 0
		if g.State.SelectedTrain == "" && len(g.Frame.Day) > 0 {
			g.State.SelectedTrain = g.Frame.Day[0].ID
		}
		g.dirty = true
		h.persist(g)
	})
	b.GetWidget().Disabled = len(g.Frame.Day) == 0
	if !g.State.Watching {
		return
	}
	label(t("representativeDay"), 14)
	p := ui.PaletteFor(g.State)
	var play *widget.Button
	slider := widget.NewSlider(
		widget.SliderOpts.WidgetOpts(
			widget.WidgetOpts.MinSize(h.px(60), h.px(44)),
			widget.WidgetOpts.LayoutData(widget.RowLayoutData{Stretch: true}),
		),
		widget.SliderOpts.Direction(widget.DirectionHorizontal),
		widget.SliderOpts.MinMax(0, 1439),
		widget.SliderOpts.InitialCurrent(g.State.ReplayMinute),
		widget.SliderOpts.Images(
			&widget.SliderTrackImage{Idle: uimage.NewNineSliceColor(p.Border)},
			&widget.ButtonImage{Idle: uimage.NewNineSliceColor(p.Action)},
		),
		widget.SliderOpts.ChangedHandler(func(a *widget.SliderChangedEventArgs) {
			// Widget initialization/rebuilds are not player seeks. Deferred events
			// from a replaced panel must not pause the current playback either.
			if h.replaySlider != a.Slider || a.Current == g.State.ReplayMinute {
				return
			}
			g.State.ReplayMinute = a.Current
			g.State.ReplayPlaying = false
			g.replayTick = 0
			if play != nil {
				play.SetText(t("playDay"))
			}
			h.persist(g)
		}),
	)
	// InitialCurrent initializes both the value and Ebiten UI's change baseline.
	// Assigning Current here emits a seek on the first Render and stops Play.
	h.replaySlider = slider
	content.AddChild(slider)
	key := "playDay"
	if g.State.ReplayPlaying {
		key = "pauseDay"
	}
	b = button(t(key), func() {
		g.State.ReplayMinute = int(g.replayTime())
		g.replayTick = 0
		g.State.ReplayPlaying = !g.State.ReplayPlaying
		g.dirty = true
		h.persist(g)
	})
	b.GetWidget().Disabled = !g.State.Motion
	play = b
	button(t("inspectTrain"), func() {
		if len(g.Frame.Day) == 0 {
			return
		}
		index := 0
		for i, trip := range g.Frame.Day {
			if trip.ID == g.State.SelectedTrain {
				index = (i + 1) % len(g.Frame.Day)
				break
			}
		}
		g.State.SelectedTrain = g.Frame.Day[index].ID
		g.State.ReplayMinute = g.Frame.Day[index].Departure
		g.State.ReplayPlaying = false
		g.replayTick = 0
		g.dirty = true
		h.persist(g)
	})
	face := h.face(g, 13)
	h.trainDescription = widget.NewText(
		widget.TextOpts.Text("", &face, p.Text),
		widget.TextOpts.MaxWidth(float64(min(h.px(360), g.width)-h.px(60))),
	)
	content.AddChild(h.trainDescription)
	h.updateTrainDescription(g)
}
func (h *host) updateTrainDescription(g *Game) {
	width := float64(min(h.px(360), g.width) - h.px(60))
	face := h.face(g, 13)
	h.trainDescription.Label = ui.WrapText(
		ui.ReplayClock(g.replayTime())+" · "+render.TrainDescription(g.Camera, g.Frame, g.State, g.replayTime()),
		width,
		func(s string) float64 { return text.Advance(s, face) },
	)
}
func (h *host) networkMarkets(g *Game, label nativeLabel) {
	t := func(key string) string { return ui.Text(g.State.Locale, key) }
	from, to, class := g.State.MarketFrom, g.State.MarketTo, g.State.MarketClass
	if from == "" {
		from = "tokyo"
	}
	if to == "" {
		to = "osaka"
	}
	if class == "" {
		class = "business"
	}
	markets := g.Frame.Network.Markets
	found := false
	for _, m := range markets {
		if m.From == from && m.To == to && m.Class == class {
			found = true
			break
		}
	}
	if !found {
		if port, ok := g.Port.(gameapi.MarketInspector); ok {
			if m, err := port.InspectMarket(from, to, class); err == nil {
				markets = append(append([]gameapi.MarketRun{}, markets...), m)
			}
		}
	}
	for _, m := range markets {
		if m.From != from || m.To != to || m.Class != class {
			continue
		}
		label(t("finiteDemand")+": "+fmt.Sprint(m.Demand)+" · "+t("companyJourneys")+": "+fmt.Sprint(m.Carried), 14)
		label(
			fmt.Sprintf(
				"%s %d · %s %d · %s %d · %s %d",
				t("mode_road"),
				m.Road,
				t("mode_air"),
				m.Air,
				t("mode_conventional"),
				m.Conventional,
				t("mode_no_trip"),
				m.NoTrip,
			),
			14,
		)
		for _, offer := range m.Offers {
			label(
				t(
					"mode_"+offer.Mode,
				)+" · "+ui.Cash(
					offer.Fare,
				)+fmt.Sprintf(
					" · %d min",
					offer.Minutes,
				)+"\n"+t(
					"generalizedCost",
				)+": "+ui.Cash(
					offer.Generalized,
				),
				13,
			)
		}
	}
	for _, flow := range g.Frame.Network.Journeys {
		if flow.From != from || flow.To != to || flow.Class != class {
			continue
		}
		parts := ""
		for i, id := range flow.Services {
			if i > 0 {
				parts += " / "
			}
			r, _ := ui.ServiceRoute(g.Frame, id)
			parts += ui.RouteName(r, g.State.Locale)
		}
		label(
			fmt.Sprintf(
				"%d · %s\n%s · %s %d",
				flow.Passengers,
				parts,
				ui.Cash(flow.Fare),
				t("transferCount"),
				len(flow.Transfers),
			),
			14,
		)
	}
}
func (h *host) marketControls(g *Game, label nativeLabel, button nativeButton) {
	t := func(key string) string { return ui.Text(g.State.Locale, key) }
	if g.State.MarketFrom == "" {
		g.State.MarketFrom = "tokyo"
	}
	if g.State.MarketTo == "" {
		g.State.MarketTo = "osaka"
	}
	if g.State.MarketClass == "" {
		g.State.MarketClass = "business"
	}
	label(t("passengerInspector"), 18)
	choices := []choice{}
	for _, city := range g.Frame.Cities {
		choices = append(choices, choice{string(city.ID), t(string(city.ID))})
	}
	label(t("fromCity"), 13)
	h.selectChoice(
		g,
		string(g.State.MarketFrom),
		choices,
		func(id string) { g.State.MarketFrom = gameapi.CityID(id); g.dirty = true; h.persist(g) },
	)
	label(t("toCity"), 13)
	h.selectChoice(
		g,
		string(g.State.MarketTo),
		choices,
		func(id string) { g.State.MarketTo = gameapi.CityID(id); g.dirty = true; h.persist(g) },
	)

	button(t(g.State.MarketClass+"Travel"), func() {
		if g.State.MarketClass == "business" {
			g.State.MarketClass = "leisure"
		} else {
			g.State.MarketClass = "business"
		}
		g.dirty = true
		h.persist(g)
	})
	h.networkMarkets(g, label)
}
