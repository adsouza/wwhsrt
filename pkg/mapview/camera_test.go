package mapview

import (
	"math"
	"testing"
)

func TestEqualEarthAndPickingTransform(t *testing.T) {
	// Published Equal Earth origin and antimeridian scale, independent constants.
	if p := Project(0, 0); p != (Point{}) {
		t.Fatal(p)
	}
	if p := Project(0, 180); math.Abs(p.X-2.7066299837) > 1e-8 || p.Y != 0 {
		t.Fatal(p)
	}
	for _, size := range [][2]float64{{320, 300}, {1280, 800}, {1920, 1080}} {
		c := New(size[0], size[1])
		p := Project(31.2304, 121.4737)
		for i := 0; i < 4; i++ {
			screen := c.Screen(p)
			back := c.World(screen)
			if math.Hypot(back.X-p.X, back.Y-p.Y) > 1e-12 {
				t.Fatal("draw/pick mismatch")
			}
			c.ZoomAt(1.8, screen)
			after := c.Screen(p)
			if math.Hypot(screen.X-after.X, screen.Y-after.Y) > 1e-9 {
				t.Fatal("zoom moved anchor")
			}
		}
	}
}

func TestPhoneCanInspectHokkaidoCorridor(t *testing.T) {
	c := New(390, 680)
	points := []Point{Project(41.904, 140.648), Project(42.514, 140.38), Project(42.63, 141.6), Project(43.0687, 141.3507)}
	c.Focus(points)
	if c.Zoom <= 8 {
		t.Fatal("regional zoom leaves the short corridor too small on a phone")
	}
	for _, p := range points {
		q := c.Screen(p)
		if q.X < 35 || q.X > c.Width-35 || q.Y < 39 || q.Y > c.Height-169 {
			t.Fatal("focused corridor clipped", q, c)
		}
	}
}
func FuzzProjectionIsFinite(f *testing.F) {
	f.Add(45.0, 9.0)
	f.Add(90.0, 180.0)
	f.Fuzz(func(t *testing.T, lat, lon float64) {
		if math.IsNaN(lat) || math.IsNaN(lon) || math.IsInf(lat, 0) || math.IsInf(lon, 0) {
			return
		}
		lat = math.Mod(lat, 90)
		lon = math.Mod(lon, 180)
		p := Project(lat, lon)
		if math.IsNaN(p.X) || math.IsNaN(p.Y) || math.IsInf(p.X, 0) || math.IsInf(p.Y, 0) {
			t.Fatal(p)
		}
	})
}
