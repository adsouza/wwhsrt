package mapview

import (
	"math"
	"testing"
)

func TestMapResizeKeepsClickedRailwayCentered(t *testing.T) {
	c := New(390, 281)
	points := []Point{Project(39.9, 116.4), Project(39.1, 117.2)}
	c.Focus(points)
	before := c.World(Point{X: c.Width / 2, Y: c.Height / 2})
	c.Resize(390, 240)
	after := c.World(Point{X: c.Width / 2, Y: c.Height / 2})
	if math.Hypot(before.X-after.X, before.Y-after.Y) > 1e-10 {
		t.Fatal("opening service inspector lost the selected railway", before, after)
	}
	center := Point{X: (points[0].X + points[1].X) / 2, Y: (points[0].Y + points[1].Y) / 2}
	p := c.Screen(center)
	if p.X < 0 || p.X > c.Width || p.Y < 0 || p.Y > c.Height {
		t.Fatal("railway left the map", p)
	}
	c.Resize(850, 700)
	after = c.World(Point{X: c.Width / 2, Y: c.Height / 2})
	if math.Hypot(before.X-after.X, before.Y-after.Y) > 1e-10 {
		t.Fatal("desktop resize lost geographic center")
	}
}
