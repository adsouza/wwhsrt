//go:build !js

package app

import (
	"bytes"
	"math"
	"path/filepath"
	"testing"

	"github.com/ebitenui/ebitenui/event"
	"github.com/ebitenui/ebitenui/widget"
	"wwhsrt/internal/application"
	"wwhsrt/pkg/gameapi"
	"wwhsrt/pkg/mapview"
	"wwhsrt/pkg/render"
	"wwhsrt/pkg/ui"
)

func TestNativeMapServiceClickDragThroughChoicesAndClear(t *testing.T) {
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
	f := p.Snapshot()
	before, _ := p.Export()
	for _, locale := range []string{"en", "zh-Hans"} {
		for _, size := range [][2]int{{390, 844}, {1280, 850}} {
			g := &Game{
				Port:   p,
				Frame:  f,
				Map:    m,
				width:  size[0],
				height: size[1],
				Camera: mapview.New(800, 700),
				State: ui.State{
					Locale:          locale,
					View:            "map",
					ServiceCorridor: "unrelated",
					FromCity:        "tsuruga",
					ToCity:          "osaka",
				},
			}
			g.host.layout(g)
			var points []mapview.Point
			for _, co := range f.Corridors {
				if co.ID == "tokyo-nagoya" {
					for _, q := range co.Alignments[0].Path {
						points = append(points, mapview.Project(q.Lat, q.Lon))
					}
				}
			}
			g.Camera.Focus(points)
			var at mapview.Point
			found := false
			for i := 1; i < len(points); i++ {
				a, b := g.Camera.Screen(points[i-1]), g.Camera.Screen(points[i])
				q := mapview.Point{X: (a.X + b.X) / 2, Y: (a.Y + b.Y) / 2}
				near := false
				for _, city := range f.Cities {
					z := render.CityPoint(g.Camera, city)
					if math.Hypot(q.X-z.X, q.Y-z.Y) <= 10 {
						near = true
					}
				}
				if id := m.PickPublicStop(g.Camera, q); id != "" {
					stop := m.PublicStop(id)
					z := g.Camera.Screen(mapview.Project(stop.Lat, stop.Lon))
					near = near || math.Hypot(q.X-z.X, q.Y-z.Y) <= 8
				}
				if !near && len(render.PickServiceSections(g.Camera, f, q)) > 0 &&
					render.PickProspect(g.Camera, f, g.State, q) == "" {
					at, found = q, true
					break
				}
			}
			if !found {
				t.Fatal("no usable route body sample")
			}
			g.moved = true
			g.finish(at)
			if g.State.View != "map" {
				t.Fatal("drag selected service")
			}
			g.moved = false
			g.finish(at)
			if g.State.View != "services" || g.State.ServicePane != "plan" || len(g.State.ServiceMapSections) == 0 ||
				!g.State.ReviewMap {
				t.Fatal(g.State)
			}
			if g.State.FromCity != "tsuruga" || g.State.ToCity != "osaka" {
				t.Fatal("service click lost project draft")
			}
			// An accepted ordinary service action must not erase map context
			// merely because there are no idle-line warnings in this company.
			g.host.result = make(chan commandResult, 1)
			g.host.result <- commandResult{frame: f, kind: gameapi.LeaseTrain}
			g.host.update(g)
			if !g.State.ReviewMap {
				t.Fatal("service result hid selected track map")
			}
			g.host.build(g)
			event.ExecuteDeferred()
			if w, _ := g.host.scroll.PreferredSize(); w > g.host.scroll.ViewRect().Dx() {
				t.Fatal("map service chooser overflowed")
			}
			if len(ui.MapServiceRoutes(f, g.State)) >= len(f.Routes) {
				t.Fatal("selector was not scoped to track")
			}
			b := g.host.actions["all-service-routes"]
			if b == nil {
				t.Fatal("cannot clear map scope")
			}
			b.ClickedEvent.Fire(&widget.ButtonClickedEventArgs{Button: b})
			event.ExecuteDeferred()
			if len(g.State.ServiceMapSections) != 0 {
				t.Fatal("filter not cleared")
			}
		}
	}
	after, _ := p.Export()
	if !bytes.Equal(before, after) {
		t.Fatal("map service selection changed campaign")
	}
}
