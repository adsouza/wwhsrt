package mapview

import "testing"

// A region's points are fitted after projection, where a northward step also moves a point towards the
// central meridian. Japan's two corner points alone therefore span a strip that ends west of Tokyo and
// east of Fukuoka.
func TestJapanRegionPointsBoundItsEndCities(t *testing.T) {
	cities := map[string]Point{"Tokyo": Project(35.685, 139.7514), "Fukuoka": Project(33.6, 130.41667)}
	for _, portrait := range []bool{false, true} {
		c := RegionCamera(390, 281, "japan", portrait)
		for name, p := range cities {
			if p.X < c.Min.X || p.X > c.Max.X || p.Y < c.Min.Y || p.Y > c.Max.Y {
				t.Fatalf("portrait=%v: %s %+v lies outside Japan's bounds %+v..%+v", portrait, name, p, c.Min, c.Max)
			}
		}
	}
}

// The World view is framed by the outermost cities, not by a longitude/latitude rectangle: on this
// projection a rectangle's two corners span far less than the rectangle, which left Japan off screen,
// and there is nothing to show west of North America's west coast.
func TestWorldViewShowsTheOutermostCities(t *testing.T) {
	c := RegionCamera(2929, 1861, "world", false)
	for name, at := range map[string][2]float64{
		"Tokyo": {35.685, 139.7514}, "Sapporo": {43.06, 141.35}, "Gold Coast": {-28.00029, 153.43088},
		"San Francisco": {37.7749, -122.4194}, "Melbourne": {-37.8136, 144.9631}, "Saint Petersburg": {59.9311, 30.3609},
		"Buenos Aires": {-34.6037, -58.3816},
	} {
		p := Project(at[0], at[1])
		if p.X < c.Min.X || p.X > c.Max.X || p.Y < c.Min.Y || p.Y > c.Max.Y {
			t.Errorf("%s lies outside the World view's bounds %+v..%+v", name, c.Min, c.Max)
		}
		// Within bounds is also on screen before any dragging.
		if at := c.Screen(p); at.X < 0 || at.X > c.Width || at.Y < 0 || at.Y > c.Height {
			t.Errorf("%s is off screen in the World view at %.0f,%.0f", name, at.X, at.Y)
		}
	}
}
