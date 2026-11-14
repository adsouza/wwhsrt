//go:build !js

package app

import (
	"bytes"
	"fmt"
	"github.com/ebitenui/ebitenui/event"
	"github.com/ebitenui/ebitenui/widget"

	"path/filepath"
	"testing"

	"wwhsrt/internal/application"
	"wwhsrt/pkg/render"
	"wwhsrt/pkg/ui"
)

func TestNativeHelpLayersFitAndPreserveTask(t *testing.T) {
	old := preferencesPath
	preferencesPath = filepath.Join(t.TempDir(), "ui.json")
	t.Cleanup(func() { preferencesPath = old })
	m, err := render.New()
	if err != nil {
		t.Fatal(err)
	}
	port, err := application.Open(nil, nil)
	if err != nil {
		t.Fatal(err)
	}
	before, _ := port.Export()
	for _, locale := range []string{"en", "zh-Hans"} {
		for _, dark := range []bool{false, true} {
			for _, size := range [][2]int{{320, 480}, {390, 844}, {800, 600}, {1200, 800}, {1920, 1080}} {
				t.Run(fmt.Sprintf("%s/%v/%v", locale, dark, size), func(t *testing.T) {
					g := &Game{Port: port, Frame: port.Snapshot(), Map: m, width: size[0], height: size[1], State: ui.NewGameState(ui.State{Locale: locale, Dark: dark}, port.Snapshot())}
					g.State.View = "services"
					g.State.Fare = 12300
					g.State.Frequency = 8
					g.State.DraftName = "Draft 草案"
					ui.NormalizeHelp(&g.State)
					for _, view := range []string{"map", "build", "services", "company", "report"} {
						g.State.View = view
						for _, anchor := range []string{"", "alignments", "funding", "fleet", "accounts"} {
							g.State.Help.Expanded = anchor
							g.host.layout(g)
							g.host.build(g)
							if w, _ := g.host.scroll.PreferredSize(); w > g.host.scroll.ViewRect().Dx() {
								t.Fatal("inline help overflows", view, anchor, w, g.host.scroll.ViewRect())
							}
						}
					}
					g.State.View = "services"
					g.State.Help.Expanded = "fares"
					g.host.layout(g)
					g.host.build(g)
					g.host.setSidebarScroll(.5)
					// Activate the actual disclosure widget. It only changes presentation.
					anchor := g.host.helpAnchors["fares"]
					if anchor == nil {
						t.Fatal("missing fare anchor")
					}
					explain := anchor
					explain.ClickedEvent.Fire(&widget.ButtonClickedEventArgs{Button: explain})
					event.ExecuteDeferred()
					g.host.build(g)
					explain = g.host.helpAnchors["fares"]
					if explain == nil {
						ui.ApplyHelpEvent(&g.State, ui.HelpEvent{Kind: "expand", Concept: "fares_demand", Anchor: "fares"})
						g.host.build(g)
					} else {
						explain.ClickedEvent.Fire(&widget.ButtonClickedEventArgs{Button: explain})
						event.ExecuteDeferred()
					}
					if g.State.Help.Expanded != "fares" || !g.State.Help.Concepts["fares_demand"].Explained {
						t.Fatal("not explained")
					}
					g.host.setSidebarScroll(.5)
					g.host.helpEvent(g, ui.HelpEvent{Kind: "open", Concept: "fares_demand", Anchor: "fares"})
					expectedScroll := g.State.Help.ReturnScroll
					g.host.build(g)
					if !g.host.blocksMap() || g.host.helpSearch == nil {
						t.Fatal("reference not active")
					}
					if w, _ := g.host.scroll.PreferredSize(); w > g.host.scroll.ViewRect().Dx() {
						t.Fatal("reference overflows")
					}
					g.host.helpSearch.Focus(true)
					g.host.helpSearch.selectTo(0, false)
					g.host.helpSearch.selectTo(len(g.host.helpSearchField.Text()), true)
					g.host.helpSearch.replace("票价")
					ui.ApplyHelpEvent(&g.State, ui.HelpEvent{Kind: "query", Value: g.host.helpSearchField.Text()})
					g.host.build(g)
					if !g.host.helpSearch.IsFocused() {
						t.Fatal("search rebuild lost focus")
					}
					g.host.helpEvent(g, ui.HelpEvent{Kind: "close"})
					g.host.build(g)
					if g.State.Fare != 12300 || g.State.Frequency != 8 || g.State.DraftName != "Draft 草案" || g.State.View != "services" || g.host.scroll.ScrollTop != expectedScroll {
						t.Fatal("reference lost task", g.State, g.host.scroll.ScrollTop)
					}
					g.State.View = "build"
					g.State.Expanded = true
					g.host.layout(g)
					g.host.build(g)
					strip := g.host.ui.Container.Children()[1]
					sw, sh := strip.PreferredSize()
					if sw > g.width || sh > g.height-int(g.Camera.Height) {
						t.Fatal("expanded strip overflow", sw, sh, g.width, g.height-int(g.Camera.Height))
					}
					if g.Camera.Height >= float64(g.height) || g.host.helpAnchors["expanded-endpoint"] == nil {
						t.Fatal("expanded endpoint strip missing")
					}
				})
			}
		}
	}
	after, _ := port.Export()
	if !bytes.Equal(before, after) {
		t.Fatal("help changed durable simulation")
	}
}
