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
	"wwhsrt/pkg/gameapi"
	"wwhsrt/pkg/mapview"
	"wwhsrt/pkg/render"
	"wwhsrt/pkg/ui"
)

func TestNativeWorkspacePrimaryPlanFitsAndPanesPreserveDrafts(t *testing.T) {
	old := preferencesPath
	preferencesPath = filepath.Join(t.TempDir(), "ui.json")
	t.Cleanup(func() { preferencesPath = old })
	p, e := application.Open(nil, nil)
	if e != nil {
		t.Fatal(e)
	}
	m, e := render.New()
	if e != nil {
		t.Fatal(e)
	}
	before, _ := p.Export()
	for _, locale := range []string{"en", "zh-Hans"} {
		for _, dock := range []string{"top", "left"} {
			for _, size := range [][2]int{{390, 844}, {1280, 850}, {1920, 1080}} {
				t.Run(fmt.Sprintf("%s/%s/%v", locale, dock, size), func(t *testing.T) {
					g := &Game{Port: p, Frame: p.Snapshot(), Map: m, width: size[0], height: size[1], State: ui.NewGameState(ui.State{Locale: locale, CommandDock: dock, PanelWidth: 520}, p.Snapshot())}
					g.State.View = "services"
					g.State.Fare = 9500
					g.host.layout(g)
					g.host.build(g)
					_, ch := g.host.scroll.PreferredSize()
					if ch > g.host.scroll.ViewRect().Dy() {
						t.Fatal("ordinary plan still needs scrolling", ch, g.host.scroll.ViewRect())
					}
					if g.host.primaryAction == nil {
						t.Fatal("missing primary service action")
					}
					originalFrequency, originalFare := g.State.Frequency, g.State.Fare
					for _, pane := range []string{"fleet", "analysis", "plan"} {
						g.State.ServicePane = pane
						g.host.build(g)
						if g.State.Frequency != originalFrequency || g.State.Fare != originalFare {
							t.Fatal("tab changed service draft")
						}
						if w, _ := g.host.scroll.PreferredSize(); w > g.host.scroll.ViewRect().Dx() {
							t.Fatal("pane overflows")
						}
					}
					g.State.View = "company"
					g.State.CompanyPane = "settings"
					g.host.build(g)
					// Disclosure and pane controls never apply a domain command.
					if g.host.actions["disclosure-build-details"] != nil {
						b := g.host.actions["disclosure-build-details"]
						b.ClickedEvent.Fire(&widget.ButtonClickedEventArgs{Button: b})
						event.ExecuteDeferred()
					}
				})
			}
		}
	}
	after, _ := p.Export()
	if !bytes.Equal(before, after) {
		t.Fatal("workspace edits mutated campaign")
	}
}
func TestNativeMapCoordinatesAccountForDockOrigin(t *testing.T) {
	old := preferencesPath
	preferencesPath = filepath.Join(t.TempDir(), "ui.json")
	t.Cleanup(func() { preferencesPath = old })
	p, _ := application.Open(nil, nil)
	m, e := render.New()
	if e != nil {
		t.Fatal(e)
	}
	for _, dock := range []string{"top", "left"} {
		g := &Game{Port: p, Frame: p.Snapshot(), Map: m, width: 1280, height: 850, Camera: mapview.New(800, 700), State: ui.State{View: "map", CommandDock: dock}}
		g.host.layout(g)
		for _, city := range g.Frame.Cities {
			if city.ID != "tokyo" {
				continue
			}
			local := render.CityPoint(g.Camera, city)
			screen := mapview.Point{X: local.X + g.mapOrigin.X, Y: local.Y + g.mapOrigin.Y}
			g.finish(g.mapLocal(screen.X, screen.Y))
			if g.State.Selected != gameapi.CityID("tokyo") {
				t.Fatal("map click offset by dock", dock, g.mapOrigin)
			}
		}
		if g.mapLocal(10, 10).Y >= 0 {
			t.Fatal("header counted as map")
		}
	}
}
