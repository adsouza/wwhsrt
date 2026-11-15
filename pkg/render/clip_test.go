package render

import (
	"math"
	"testing"
	"wwhsrt/pkg/mapview"
)

func TestClipWorldGeometryBeforeAntialiasing(t *testing.T) {
	c := mapview.Fit(390, 280, []mapview.Point{mapview.Project(25, 100), mapview.Project(40, 125)})
	onScreen := func(ring [][2]float64) []mapview.Point {
		points := []mapview.Point{}
		for _, p := range ring {
			points = append(points, c.Screen(mapview.Project(p[1], p[0])))
		}
		return points
	}
	clipped := clipScreen(c, onScreen([][2]float64{{-170, -80}, {170, -80}, {170, 80}, {-170, 80}}))
	if len(clipped) < 3 {
		t.Fatal("visible world polygon lost")
	}
	for _, p := range clipped {
		if p.X < -4.001 || p.X > c.Width+4.001 || p.Y < -4.001 || p.Y > c.Height+4.001 {
			t.Fatalf("unbounded offscreen path %+v", p)
		}
	}
	if points := clipScreen(c, onScreen([][2]float64{{-100, 20}, {-90, 20}, {-90, 25}})); len(points) != 0 {
		t.Fatal("offscreen polygon retained")
	}
	a, b, ok := screenSegment(c, mapview.Point{X: -10000, Y: 140}, mapview.Point{X: 10000, Y: 140})
	if !ok || math.Abs(a.X+4) > 0.001 || b.X > 394.001 {
		t.Fatalf("crossing segment lost or not clipped: %+v %+v %v", a, b, ok)
	}
	if _, _, ok := screenSegment(c, mapview.Point{X: -100, Y: -100}, mapview.Point{X: -50, Y: -50}); ok {
		t.Fatal("outside segment retained")
	}
}
