//go:build !js

package app

import (
	"testing"

	"wwhsrt/internal/application"
	"wwhsrt/pkg/gameapi"
	"wwhsrt/pkg/mapview"
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

func TestRegionFrameIgnoresWindowSize(t *testing.T) {
	phone := mapview.RegionCamera(390, 260, "japan", true)
	desk := mapview.RegionCamera(1920, 1080, "japan", true)
	frame := regionFrame(phone)
	if frame != regionFrame(desk) {
		t.Fatal("a region's frame must not depend on the window", frame, regionFrame(desk))
	}
	points := ui.Box{MinX: phone.Min.X, MinY: phone.Min.Y, MaxX: phone.Max.X, MaxY: phone.Max.Y}
	if !frame.Contains(points) || frame.Span() <= points.Span() {
		t.Fatal("the frame must add a margin around the region's points", frame, points)
	}
	if regionFrame(mapview.RegionCamera(390, 260, "china", true)) == frame {
		t.Fatal("different regions need different frames")
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
