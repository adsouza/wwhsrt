//go:build !js

package app

import (
	"testing"

	"wwhsrt/pkg/mapview"
	"wwhsrt/pkg/render"
)

// The World view exists to show every city at once. It must do so without dragging, and without
// spending the map on ocean beyond the outermost cities.
func TestWorldViewFramesEveryCity(t *testing.T) {
	cities := newChromeWorld(t).frame.Cities
	if len(cities) < 200 {
		t.Fatal("want the whole catalog, got", len(cities))
	}
	for _, size := range [][2]float64{{2929, 1861}, {832, 682}, {1840, 900}} {
		c := mapview.RegionCamera(size[0], size[1], "world", false)
		left, right := c.Width, 0.0
		for _, city := range cities {
			p := render.CityPoint(c, city)
			if p.X < 0 || p.X > c.Width || p.Y < 0 || p.Y > c.Height {
				t.Errorf("%vx%v: %s is off screen at %.0f,%.0f", size[0], size[1], city.ID, p.X, p.Y)
			}
			left, right = min(left, p.X), max(right, p.X)
		}
		// Where the map's width sets the scale, the outermost cities sit near its edges.
		if c.Width/c.Height < 2 && (left > c.Width/10 || c.Width-right > c.Width/10) {
			t.Errorf("%vx%v: %.0f px of empty map west of the westernmost city and %.0f px east of the easternmost",
				size[0], size[1], left, c.Width-right)
		}
	}
}
