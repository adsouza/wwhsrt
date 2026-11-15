//go:build !js

package app

import (
	"math"
	"testing"

	"wwhsrt/pkg/mapview"
	"wwhsrt/pkg/render"
)

// Every landscape region, East Asia's scenario frame included, zooms out to the World view's scale;
// portrait regions stop at their own frame. Scale is read from the screen distance between two points.
func TestLandscapeRegionCamerasZoomOutToTheWorld(t *testing.T) {
	m, e := render.New()
	if e != nil {
		t.Fatal(e)
	}
	const w, h = 1472, 912
	a, b := mapview.Project(0, -60), mapview.Project(0, 60)
	scale := func(c mapview.Camera) float64 { return c.Screen(b).X - c.Screen(a).X }
	g := &Game{Map: m, Camera: mapview.New(w, h)}
	world := scale(g.cameraForRegion("world", false))
	for _, r := range mapview.VisibleRegions(false) {
		c := g.cameraForRegion(r.ID, false)
		c.ZoomAt(1e-6, mapview.Point{X: w / 2, Y: h / 2})
		if got := scale(c); math.Abs(got-world) > 1e-6*world {
			t.Errorf("landscape %s zooms out to %.2f px, the World view shows %.2f px", r.ID, got, world)
		}
	}
	for _, r := range mapview.VisibleRegions(true) {
		c := g.cameraForRegion(r.ID, true)
		fit := scale(c)
		c.ZoomAt(1e-6, mapview.Point{X: w / 2, Y: h / 2})
		if got := scale(c); math.Abs(got-fit) > 1e-9*fit {
			t.Errorf("portrait %s zoomed out past its own frame: %.2f px < %.2f px", r.ID, got, fit)
		}
	}
}
