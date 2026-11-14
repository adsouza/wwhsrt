//go:build !js

package app

import (
	"fmt"
	"path/filepath"
	"strings"
	"testing"

	"github.com/ebitenui/ebitenui/widget"
	"wwhsrt/internal/application"
	"wwhsrt/pkg/gameapi"
	"wwhsrt/pkg/mapview"
	"wwhsrt/pkg/render"
	"wwhsrt/pkg/ui"
)

func routeChoiceFrame(t *testing.T) (gameapi.Game, gameapi.Frame) {
	t.Helper()
	p, e := application.Open(nil, nil)
	if e != nil {
		t.Fatal(e)
	}
	return p, p.Snapshot()
}

func TestRouteBoxesCoverEveryOpenRouteAndItsSections(t *testing.T) {
	_, f := routeChoiceFrame(t)
	boxes := routeBoxes(f)
	open, through := 0, 0
	for _, r := range f.Routes {
		if !r.Open {
			continue
		}
		open++
		box, ok := boxes[r.ID]
		if !ok || box.Span() <= 0 {
			t.Fatal("open route has no usable box", r.ID, box)
		}
		if len(r.Sections) < 2 {
			continue
		}
		through++
		for _, section := range r.Sections {
			part, ok := boxes[section]
			if !ok || !box.Contains(part) {
				t.Fatal("route box must cover each of its sections", r.ID, section)
			}
		}
	}
	if open < 1000 || through < 1000 {
		t.Fatal("expected the public catalog's routes", open, through)
	}
}

func TestViewBoxMatchesCamera(t *testing.T) {
	c := mapview.New(1000, 600)
	centre := mapview.Point{X: 500, Y: 300}
	wide := viewBox(c)
	at := c.World(centre)
	if !wide.Contains(ui.Box{MinX: at.X, MinY: at.Y, MaxX: at.X, MaxY: at.Y}) {
		t.Fatal("view box must contain the point at the screen centre", wide, at)
	}
	for _, corner := range []mapview.Point{{X: 0, Y: 0}, {X: 1000, Y: 600}} {
		w := c.World(corner)
		if !wide.Contains(ui.Box{MinX: w.X, MinY: w.Y, MaxX: w.X, MaxY: w.Y}) {
			t.Fatal("view box must reach the screen corners", wide, w)
		}
	}
	c.ZoomAt(4, centre)
	tight := viewBox(c)
	if !wide.Contains(tight) || tight.Span() >= wide.Span()/3 {
		t.Fatal("zooming in must shrink the view box inside the old one", wide, tight)
	}
}

func TestRegionFrameIsWhatTheMapWouldShow(t *testing.T) {
	_, f := routeChoiceFrame(t)
	m, e := render.New()
	if e != nil {
		t.Fatal(e)
	}
	boxes := routeBoxes(f)
	small := &Game{Map: m, Camera: mapview.New(390, 260)}
	large := &Game{Map: m, Camera: mapview.New(1920, 1080)}
	for _, portrait := range []bool{true, false} {
		frame := small.regionFrame("japan", portrait)
		if frame != large.regionFrame("japan", portrait) {
			t.Fatal("a region's frame must not depend on the window", portrait)
		}
		c := mapview.RegionCamera(100, 100, "japan", portrait)
		if !frame.Contains(ui.Box{MinX: c.Min.X, MinY: c.Min.Y, MaxX: c.Max.X, MaxY: c.Max.Y}) {
			t.Fatal("the frame must cover the region's defining points", portrait)
		}
		// Japan is defined by two corner points; the projection narrows their
		// bounding box, so the frame has to be what the map actually shows.
		if !frame.Contains(boxes["tokyo-nagoya"]) {
			t.Fatal("Japan's frame must cover Tokyo–Nagoya", portrait, frame, boxes["tokyo-nagoya"])
		}
		if small.regionFrame("china", portrait) == frame {
			t.Fatal("different regions need different frames")
		}
		if small.regionFrame("", portrait) != small.regionFrame("east-asia", portrait) ||
			!small.regionFrame("", portrait).Contains(boxes["tokyo-nagoya"]) {
			t.Fatal("the default region is East Asia and includes Japan")
		}
	}
}

func TestBoxesForRecomputesOnNewRevision(t *testing.T) {
	_, f := routeChoiceFrame(t)
	h := &host{}
	first := h.boxesFor(f)
	dropped := f.Routes[0].ID
	if _, ok := first[dropped]; !ok {
		t.Fatal("missing box", dropped)
	}
	first[dropped] = ui.Box{MinX: 1, MaxX: 2}
	if again := h.boxesFor(f); again[dropped] != (ui.Box{MinX: 1, MaxX: 2}) {
		t.Fatal("the same revision must reuse the cached boxes")
	}
	next := f
	next.Revision++
	next.Routes = f.Routes[1:]
	if _, ok := h.boxesFor(next)[dropped]; ok {
		t.Fatal("a new revision must rebuild the boxes")
	}
}

// servicesGame builds the native Services panel for the starter route.
func servicesGame(t *testing.T, width, height int, region string) *Game {
	t.Helper()
	old := preferencesPath
	preferencesPath = filepath.Join(t.TempDir(), "ui.json")
	t.Cleanup(func() { preferencesPath = old })
	p, f := routeChoiceFrame(t)
	m, e := render.New()
	if e != nil {
		t.Fatal(e)
	}
	state := ui.State{Locale: "en", View: "services", Motion: true, Region: region}
	ui.DraftService(&state, f, "tokyo-nagoya")
	g := &Game{Port: p, Frame: f, Map: m, width: width, height: height, State: state}
	g.host.layout(g)
	g.host.build(g)
	return g
}

func listedRouteIDs(g *Game) []string {
	ids := []string{}
	for _, r := range g.host.routeChoices.Routes {
		ids = append(ids, r.ID)
	}
	return ids
}

func panelShows(g *Game, want string) bool {
	for _, child := range g.host.content.Children() {
		if text, ok := child.(*widget.Text); ok &&
			strings.Join(strings.Fields(text.Label), " ") == strings.Join(strings.Fields(want), " ") {
			return true
		}
	}
	return false
}

func TestNativeServiceRoutesFollowTheMapView(t *testing.T) {
	g := servicesGame(t, 1280, 850, "")
	if g.host.workspace.Map.Empty() || len(g.host.combos) != 1 {
		t.Fatal("desktop Services shows the map and one route dropdown", len(g.host.combos))
	}
	wide := g.host.routeChoices
	ids := listedRouteIDs(g)
	if !wide.InView || len(ids) < 2 || len(ids) > ui.RouteChoiceLimit+1 || !strings.Contains(strings.Join(ids, " "), "tokyo-nagoya") {
		t.Fatal("want a short in-view list that keeps the selection", wide.InView, len(ids), ids)
	}
	boxes, view := g.host.boxesFor(g.Frame), viewBox(g.Camera)
	for _, id := range ids {
		if id != "tokyo-nagoya" && !view.Contains(boxes[id]) {
			t.Fatal("listed route does not fit in the view", id)
		}
	}
	if !panelShows(g, ui.RouteChoiceHint("en", wide, true)) || wide.Matching <= ui.RouteChoiceLimit ||
		!panelShows(g, ui.Text("en", "chooseRoute")) {
		t.Fatal("a capped desktop list must say so under the dropdown", wide.Matching)
	}
	g.Camera.ZoomAt(32, mapview.Point{X: g.Camera.Width / 2, Y: g.Camera.Height / 2})
	g.host.build(g)
	tight := listedRouteIDs(g)
	if strings.Join(tight, " ") == strings.Join(ids, " ") || !strings.Contains(strings.Join(tight, " "), "tokyo-nagoya") {
		t.Fatal("zooming in must change the list and keep the selection", tight)
	}
	g.State.ServiceQuery = "shanghai"
	g.host.build(g)
	found := g.host.routeChoices
	if found.InView || len(found.Routes) < 2 || len(found.Routes) > ui.RouteChoiceLimit+1 {
		t.Fatal("search must look beyond the view", found.InView, len(found.Routes))
	}
	for _, r := range found.Routes {
		if r.ID != "tokyo-nagoya" && !strings.Contains(strings.ToLower(ui.RouteName(r, "en")), "shanghai") {
			t.Fatal("search listed a route that does not match", r.ID)
		}
	}
}

func TestNativeServiceRoutesFollowTheRegionWithoutAMap(t *testing.T) {
	g := servicesGame(t, 390, 844, "japan")
	if !g.host.workspace.Map.Empty() || len(g.host.combos) != 1 {
		t.Fatal("phone Services hides the map and keeps the ordinary plan to one dropdown", len(g.host.combos))
	}
	japan := listedRouteIDs(g)
	choices := g.host.routeChoices
	if !choices.InView || len(japan) < 2 || len(japan) > ui.RouteChoiceLimit+1 || choices.Matching <= ui.RouteChoiceLimit {
		t.Fatal("want a capped list for the region", choices.InView, len(japan), choices.Matching)
	}
	boxes, frame := g.host.boxesFor(g.Frame), g.regionFrame("japan", true)
	for _, id := range japan {
		if id != "tokyo-nagoya" && !frame.Contains(boxes[id]) {
			t.Fatal("listed route does not fit in the region", id)
		}
	}
	// The ordinary phone plan must still fit on one screen, so the count rides
	// on the route label and the advice waits inside the disclosure.
	capped := fmt.Sprintf(ui.Text("en", "chooseRouteCapped"), ui.RouteChoiceLimit, choices.Matching)
	advice := ui.RouteChoiceHint("en", choices, false)
	if !panelShows(g, capped) || panelShows(g, advice) {
		t.Fatal("closed disclosure shows only the count on the route label")
	}
	if _, height := g.host.scroll.PreferredSize(); height > g.host.scroll.ViewRect().Dy() {
		t.Fatal("ordinary phone plan needs scrolling", height, g.host.scroll.ViewRect())
	}
	ui.SetDisclosure(&g.State, "route-search", true)
	g.host.build(g)
	if len(g.host.combos) != 2 || !panelShows(g, advice) {
		t.Fatal("open disclosure offers the region dropdown and the advice", len(g.host.combos))
	}
	title := g.host.actions["disclosure-route-search"].Text().Label
	if !strings.Contains(strings.Join(strings.Fields(title), " "), ui.Text("en", "narrowRoutes")) {
		t.Fatal("with no map the disclosure is about narrowing the list", title)
	}
	if current := g.host.combos[0].SelectedEntry(); current == nil || current.(choice).ID != "japan" {
		t.Fatal("region dropdown must show the current region", current)
	}
	g.State.Region = "china"
	g.host.build(g)
	if china := listedRouteIDs(g); strings.Join(china, " ") == strings.Join(japan, " ") {
		t.Fatal("another region must list other routes", china)
	}
	g.State.Region = "south-asia"
	g.host.build(g)
	if current := g.host.combos[0].SelectedEntry(); current == nil || current.(choice).ID != "south-asia" {
		t.Fatal("the current region stays in the dropdown even with no routes", current)
	}
	ui.SetDisclosure(&g.State, "route-search", false)
	g.host.build(g)
	empty := g.host.routeChoices
	if ids := listedRouteIDs(g); len(ids) != 1 || ids[0] != "tokyo-nagoya" || empty.Matching != 0 {
		t.Fatal("a region with no fitting route lists only the selection", ids, empty.Matching)
	}
	if !panelShows(g, ui.Text("en", "routeChoicesNoneRegion")) || !panelShows(g, ui.Text("en", "chooseRoute")) {
		t.Fatal("an empty region must explain itself even with the disclosure closed")
	}
}

func TestNativeServiceRoutesExplainAnEmptyView(t *testing.T) {
	g := servicesGame(t, 1280, 850, "")
	g.Camera.ZoomAt(32, mapview.Point{X: g.Camera.Width / 2, Y: g.Camera.Height / 2})
	g.Camera.PanX, g.Camera.PanY = 1e9, 1e9
	g.Camera.Clamp()
	g.host.build(g)
	choices := g.host.routeChoices
	if ids := listedRouteIDs(g); len(ids) != 1 || ids[0] != "tokyo-nagoya" || choices.Matching != 0 || !choices.InView {
		t.Fatal("an empty view lists only the selection", ids, choices.Matching)
	}
	if !panelShows(g, ui.Text("en", "routeChoicesNone")) {
		t.Fatal("an empty view must explain itself")
	}
}
