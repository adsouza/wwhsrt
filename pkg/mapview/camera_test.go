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
	points := []Point{
		Project(41.904, 140.648),
		Project(42.514, 140.38),
		Project(42.63, 141.6),
		Project(43.0687, 141.3507),
	}
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

// Dragging at zoom 1 can show half a screen beyond the fitted frame on each side. Zooming in must not
// put any of that out of reach: the world position of each screen edge, with the map dragged as far as
// it goes, is the same at every zoom.
func TestPanLimitDoesNotShrinkWithZoom(t *testing.T) {
	// Landscape cameras are limited by the World view; portrait ones by their own frame.
	for _, tc := range []struct {
		id       string
		portrait bool
	}{{"world", false}, {"east-asia", false}, {"japan", false}, {"south-america", true}, {"japan", true}} {
		c := RegionCamera(2929, 1861, tc.id, tc.portrait)
		base := edgeReach(c, 1)
		for _, zoom := range []float64{2, 8, 32} {
			for i, edge := range edgeReach(c, zoom) {
				if math.Abs(edge-base[i]) > 1e-9 {
					t.Errorf("%s portrait=%v at zoom %v: dragged fully, edge %d reaches %.4f but zoom 1 reaches %.4f", tc.id, tc.portrait, zoom, i, edge, base[i])
				}
			}
		}
	}
}
func TestTokyoStaysReachableInTheWorldViewAtMaximumZoom(t *testing.T) {
	tokyo := Project(35.685, 139.7514)
	for _, zoom := range []float64{1, 4, 16, 32} {
		c := RegionCamera(2929, 1861, "world", false)
		c.ZoomAt(zoom, Point{X: c.Width / 2, Y: c.Height / 2})
		// Drag west as far as the map goes, then level Tokyo's latitude with the middle of the screen.
		c.PanX = -1e12
		c.PanY += c.Height/2 - c.Screen(tokyo).Y
		c.Clamp()
		if at := c.Screen(tokyo); at.X > c.Width || at.Y < 0 || at.Y > c.Height {
			t.Errorf("zoom %v: Tokyo is at %.0f,%.0f, off a %.0fx%.0f map", zoom, at.X, at.Y, c.Width, c.Height)
		}
	}
}

// An avoided point inside the fitted frame is panned past the nearest edge it can reach while every
// fitted point keeps its marker clearance; a point already outside, or one no pan can clear, changes nothing.
func TestFitAvoidingPansAPointOutWithinTheGutters(t *testing.T) {
	anchors := []Point{Project(45.75, 126.65), Project(30.5728, 104.0668), Project(22.3193, 114.1694), Project(35.685, 139.7514), Project(16.8661, 96.1951)}
	dhaka := Project(23.8103, 90.4125)
	inside := func(c Camera, p Point) bool { return !c.Clear(c.Screen(p)) }
	plain := Fit(390, 281, anchors)
	if !inside(plain, dhaka) {
		t.Fatal("fixture: the plain fit should show Dhaka")
	}
	c := FitAvoiding(390, 281, anchors, []Point{dhaka})
	if inside(c, dhaka) {
		t.Fatal("Dhaka is still within the margin", c.Screen(dhaka))
	}
	for _, p := range anchors {
		q := c.Screen(p)
		if q.X < AnchorClearance-1e-9 || q.X > c.Width-AnchorClearance+1e-9 || q.Y < AnchorClearance-1e-9 || q.Y > c.Height-AnchorClearance+1e-9 {
			t.Fatal("an anchor left its marker clearance", q)
		}
	}
	if c.Zoom != plain.Zoom || c.Min != plain.Min || c.Max != plain.Max {
		t.Fatal("avoiding a point must pan, not refit")
	}
	if got := c.Screen(dhaka).X; math.Abs(got+ExcludeClearance) > 1e-6 {
		t.Fatal("the pan should be the smallest that clears the margin", got)
	}
	far := Project(-33.8688, 151.2093)
	if FitAvoiding(390, 281, anchors, []Point{dhaka, dhaka}) != c {
		t.Fatal("clearing an already-cleared point must not pan again")
	}
	if FitAvoiding(390, 281, anchors, []Point{far}) != plain {
		t.Fatal("a point already outside must not move the frame")
	}
	// Squeezed to the clearance on both axes there is no spare room to pan into.
	tight := FitAvoiding(2*AnchorClearance+1, 2*AnchorClearance+1, anchors, []Point{Project(30, 115)})
	if tight != Fit(2*AnchorClearance+1, 2*AnchorClearance+1, anchors) {
		t.Fatal("an unclearable point must leave the fit unchanged")
	}
}

// edgeReach is how far each screen edge reaches with the map dragged fully west, east, south and north
// at the given zoom factor from c: world X of the right, left edges, world Y of the bottom, top edges.
func edgeReach(c Camera, zoom float64) [4]float64 {
	var out [4]float64
	for i, pan := range [][2]float64{{-1e12, 0}, {1e12, 0}, {0, -1e12}, {0, 1e12}} {
		d := c
		d.ZoomAt(zoom, Point{X: d.Width / 2, Y: d.Height / 2})
		d.PanX, d.PanY = pan[0], pan[1]
		d.Clamp()
		switch i {
		case 0:
			out[i] = d.World(Point{X: d.Width, Y: d.Height / 2}).X
		case 1:
			out[i] = d.World(Point{X: 0, Y: d.Height / 2}).X
		case 2:
			out[i] = d.World(Point{X: d.Width / 2, Y: d.Height}).Y
		default:
			out[i] = d.World(Point{X: d.Width / 2, Y: 0}).Y
		}
	}
	return out
}

// A landscape region opens on its own frame but is not a cage: it zooms out as far as the World view
// and pans as far as the World view does, at every zoom. Portrait keeps the region's own limits.
func TestLandscapeRegionsReachWhatTheWorldViewReaches(t *testing.T) {
	const w, h = 1472, 912
	world := RegionCamera(w, h, "world", false)
	want := edgeReach(world, 1)
	for _, id := range []string{"south-america", "japan", "mashreq", "australia", "north-america"} {
		c := RegionCamera(w, h, id, false)
		out := c
		out.ZoomAt(1e-6, Point{X: w / 2, Y: h / 2})
		if math.Abs(out.scale()-world.scale()) > 1e-9*world.scale() {
			t.Errorf("%s: zoomed fully out at scale %v, the World view is %v", id, out.scale(), world.scale())
		}
		for _, zoom := range []float64{1e-6, 0.5, 1, 4, 32} {
			for i, edge := range edgeReach(c, zoom) {
				if math.Abs(edge-want[i]) > 1e-9 {
					t.Errorf("%s at zoom %v: edge %d reaches %.4f, the World view reaches %.4f", id, zoom, i, edge, want[i])
				}
			}
		}
	}
	// The case that was reported: Bogotá from landscape South America on a 1920x1080 window.
	// Dragging to centre it must leave it in view, at zoom 1 and zoomed in.
	bogota := Project(4.711, -74.0721)
	for _, zoom := range []float64{1, 8} {
		c := RegionCamera(w, h, "south-america", false)
		c.ZoomAt(zoom, Point{X: w / 2, Y: h / 2})
		q := c.Screen(bogota)
		c.PanX += w/2 - q.X
		c.PanY += h/2 - q.Y
		c.Clamp()
		if q = c.Screen(bogota); q.X < 0 || q.X > w || q.Y < 0 || q.Y > h {
			t.Errorf("zoom %v: dragged towards Bogotá, it stays at %.0f,%.0f outside the %vx%v map", zoom, q.X, q.Y, w, h)
		}
	}
	p := RegionCamera(w, h, "south-america", true)
	fit := p.scale()
	p.ZoomAt(1e-6, Point{X: w / 2, Y: h / 2})
	if p.scale() != fit {
		t.Errorf("portrait South America zoomed out past its own frame: %v < %v", p.scale(), fit)
	}
}

// In landscape the World view at its own fit is a hard edge: fully zoomed out nothing pans, and at
// any zoom the screen edges reach exactly what that view shows. Portrait keeps half a screen of
// drag beyond its region's frame, its only way to see past the region.
func TestLandscapeWorldEdgeIsHard(t *testing.T) {
	for _, wh := range [][2]float64{{1472, 912}, {396, 222}} {
		w, h := wh[0], wh[1]
		world := RegionCamera(w, h, "world", false)
		shown := [4]float64{
			world.World(Point{X: w, Y: h / 2}).X, world.World(Point{X: 0, Y: h / 2}).X,
			world.World(Point{X: w / 2, Y: h}).Y, world.World(Point{X: w / 2, Y: 0}).Y,
		}
		for _, id := range []string{"world", "south-america", "east-asia", "australia"} {
			c := RegionCamera(w, h, id, false)
			for _, zoom := range []float64{1e-6, 1, 8} {
				for i, edge := range edgeReach(c, zoom) {
					if math.Abs(edge-shown[i]) > 1e-9 {
						t.Errorf("%vx%v %s at zoom %v: edge %d reaches %.4f, the World view shows to %.4f", w, h, id, zoom, i, edge, shown[i])
					}
				}
			}
		}
		for _, pan := range [][2]float64{{-1e12, 0}, {1e12, 0}, {0, -1e12}, {0, 1e12}} {
			c := world
			c.PanX, c.PanY = pan[0], pan[1]
			c.Clamp()
			if math.Abs(c.PanX) > 1e-9 || math.Abs(c.PanY) > 1e-9 {
				t.Errorf("%vx%v: the World view at its fit panned to %v,%v", w, h, c.PanX, c.PanY)
			}
		}
		p := RegionCamera(w, h, "south-america", true)
		p.PanX, p.PanY = 1e12, -1e12
		p.Clamp()
		if math.Abs(p.PanX-w/2) > 1e-9 || math.Abs(p.PanY+h/2) > 1e-9 {
			t.Errorf("%vx%v: portrait South America at zoom 1 pans to %v,%v, want half a screen", w, h, p.PanX, p.PanY)
		}
	}
}

// The World limit changes what can be reached, never where a landscape region opens.
func TestLandscapeRegionsOpenOnTheirOwnFit(t *testing.T) {
	for _, wh := range [][2]float64{{1472, 912}, {992, 732}, {576, 600}, {352, 432}, {396, 222}} {
		for _, r := range VisibleRegions(false) {
			got, fit := RegionCamera(wh[0], wh[1], r.ID, false), regionFit(wh[0], wh[1], r.ID, false)
			if got.Zoom != fit.Zoom || math.Abs(got.PanX-fit.PanX) > 1e-9 || math.Abs(got.PanY-fit.PanY) > 1e-9 {
				t.Errorf("%vx%v %s opens at zoom %v pan %.2f,%.2f, its fit is zoom %v pan %.2f,%.2f", wh[0], wh[1], r.ID, got.Zoom, got.PanX, got.PanY, fit.Zoom, fit.PanX, fit.PanY)
			}
		}
	}
}
