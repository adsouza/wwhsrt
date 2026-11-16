package mapview

import "testing"

// East Asia covers China, Japan and Korea, which no longer have regions of their own.
func TestKnownRegionsAreThePickersOwn(t *testing.T) {
	for _, id := range []string{"china", "japan", "korea", ""} {
		if KnownRegion(id) {
			t.Errorf("%q must not be a region", id)
		}
	}
	for _, r := range Regions {
		if !KnownRegion(r.ID) {
			t.Errorf("%q is in the picker but not known", r.ID)
		}
	}
	if !KnownRegion("east-asia") {
		t.Error("East Asia must stay a region")
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
