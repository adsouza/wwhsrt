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
