//go:build !js

package app

import (
	"bytes"
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

func TestNativeProspectsMapListDragAndCommittedAvailability(t *testing.T) {
	old := preferencesPath
	preferencesPath = filepath.Join(t.TempDir(), "ui.json")
	t.Cleanup(func() { preferencesPath = old })
	p, err := application.Open(nil, nil)
	if err != nil {
		t.Fatal(err)
	}
	m, err := render.New()
	if err != nil {
		t.Fatal(err)
	}
	before, _ := p.Export()
	f := p.Snapshot()
	for _, locale := range []string{"en", "zh-Hans"} {
		for _, size := range [][2]int{{390, 844}, {1280, 850}} {
			g := &Game{Port: p, Frame: f, Map: m, width: size[0], height: size[1], Camera: mapview.New(800, 700), State: ui.State{Locale: locale, View: "map", FromCity: "arbitrary", ToCity: "other"}}
			g.host.layout(g)
			co := ui.ConstructionOpportunities(f)[0]
			path := co.Alignments[0].Path
			points := []mapview.Point{}
			for _, q := range path {
				points = append(points, mapview.Project(q.Lat, q.Lon))
			}
			g.Camera.Focus(points)
			at := g.Camera.Screen(points[len(points)/2])
			g.moved = true
			g.finish(at)
			if g.State.View != "map" || g.State.FromCity != "arbitrary" {
				t.Fatal("drag selected prospect")
			}
			g.moved = false
			g.finish(at)
			if g.State.View != "build" || g.State.FromCity != co.FromCity || g.State.ToCity != co.ToCity {
				t.Fatal("map did not select whole prospect", g.State)
			}
			g.host.build(g)
			if w, _ := g.host.scroll.PreferredSize(); w > g.host.scroll.ViewRect().Dx() {
				t.Fatal("prospect list overflow")
			}
			other := ui.QuickProspects(f, g.State)[1]
			b := g.host.actions["prospect-"+other.ID]
			if b == nil {
				t.Fatal("missing named prospect")
			}
			b.ClickedEvent.Fire(&widget.ButtonClickedEventArgs{Button: b})
			event.ExecuteDeferred()
			if g.State.FromCity != other.FromCity || g.State.ToCity != other.ToCity {
				t.Fatal("list did not select prospect")
			}
			browse := g.host.actions["browse-prospects"]
			browse.ClickedEvent.Fire(&widget.ButtonClickedEventArgs{Button: browse})
			event.ExecuteDeferred()
			if !g.State.Expanded {
				t.Fatal("cannot browse full map")
			}
			g.State.Expanded = false
			g.Frame.Projects = append(append([]gameapi.Project{}, f.Projects...), gameapi.Project{CorridorID: other.ID, Ready: 100, AlignmentID: other.Alignments[0].ID})
			g.host.build(g)
			if g.host.actions["prospect-"+other.ID] != nil {
				t.Fatal("started line stayed in prospects")
			}
		}
	}
	after, _ := p.Export()
	if !bytes.Equal(before, after) {
		t.Fatal("map/list selection changed campaign")
	}
}

func TestNativeWorldwideProspectPickerFitsAndSelectsDedicatedHSR(t *testing.T) {
	old := preferencesPath
	preferencesPath = filepath.Join(t.TempDir(), "ui.json")
	t.Cleanup(func() { preferencesPath = old })
	port, e := application.Open(nil, nil)
	if e != nil {
		t.Fatal(e)
	}
	m, e := render.New()
	if e != nil {
		t.Fatal(e)
	}
	f := port.Snapshot()
	if len(ui.ConstructionOpportunities(f)) != 74 {
		t.Fatal("catalog incomplete")
	}
	for _, target := range []string{"launch_26_lisbon__porto", "launch_71_houston__austin", "launch_72_tsuruga__maibara", "launch_73_valencia__camp_de_tarragona", "launch_74_porto__vigo"} {
		for _, locale := range []string{"en", "zh-Hans"} {
			for _, size := range [][2]int{{320, 480}, {1440, 900}} {
				g := &Game{Port: port, Frame: f, Map: m, width: size[0], height: size[1], Camera: mapview.New(800, 700), State: ui.State{Locale: locale, View: "build", FromCity: "tsuruga", ToCity: "osaka", AlignmentID: "mountain"}}
				g.host.layout(g)
				g.host.build(g)
				if len(g.host.combos) == 0 {
					t.Fatal("missing complete-catalog picker")
				}
				// Initial control events run at the end of a normal UI update,
				// before the player can open the dropdown.
				event.ExecuteDeferred()
				stale := g.host.combos[0]
				stale.SetSelectedEntry(choice{ID: target, Name: target})
				event.ExecuteDeferred()
				stale.SetSelectedEntry(choice{ID: "launch_31_cairo__alexandria", Name: "stale event"})
				event.ExecuteDeferred()
				co, ok := ui.Pair(g.Frame, g.State.FromCity, g.State.ToCity)
				if !ok || co.ID != target || g.State.AlignmentID != "launch" {
					t.Fatal("picker selected wrong project", g.State)
				}
				if w, _ := g.host.scroll.PreferredSize(); w > g.host.scroll.ViewRect().Dx() {
					t.Fatal("worldwide picker overflow")
				}
				if len(ui.QuickProspects(g.Frame, g.State)) > 2 {
					t.Fatal("full catalog crowds the ordinary plan")
				}
			}
		}
	}
}
