package render

import (
	"math"
	"testing"

	"wwhsrt/internal/application"
	"wwhsrt/pkg/gameapi"
	"wwhsrt/pkg/mapview"
	"wwhsrt/pkg/ui"
)

func TestProspectPickingFollowsAlignmentAndCurrentAvailability(t *testing.T) {
	c := mapview.Fit(1000, 700, []mapview.Point{mapview.Project(30, 100), mapview.Project(40, 110)})
	f := gameapi.Frame{Corridors: []gameapi.Corridor{
		{ID: "bend", FromCity: "a", ToCity: "b", Alignments: []gameapi.Alignment{
			{ID: "valley", Path: []gameapi.GeoPoint{{Lat: 31, Lon: 101}, {Lat: 36, Lon: 102}, {Lat: 39, Lon: 109}}},
			{ID: "tunnel", Path: []gameapi.GeoPoint{{Lat: 31, Lon: 101}, {Lat: 39, Lon: 109}}},
		}},
		{
			ID:         "existing",
			Existing:   true,
			Alignments: []gameapi.Alignment{{Path: []gameapi.GeoPoint{{Lat: 32, Lon: 109}, {Lat: 38, Lon: 109}}}},
		},
	}}
	s := ui.State{View: "map"}
	mid := func(a, b gameapi.GeoPoint) mapview.Point {
		x, y := c.Screen(mapview.Project(a.Lat, a.Lon)), c.Screen(mapview.Project(b.Lat, b.Lon))
		return mapview.Point{X: (x.X + y.X) / 2, Y: (x.Y + y.Y) / 2}
	}
	valley := f.Corridors[0].Alignments[0].Path
	at := mid(valley[0], valley[1])
	if PickProspect(c, f, s, at) != "bend" {
		t.Fatal("missed curved alignment body")
	}
	if PickProspect(c, f, s, mapview.Point{X: at.X + 14, Y: at.Y}) != "bend" {
		t.Fatal("touch target too narrow")
	}
	if PickProspect(c, f, s, mapview.Point{X: at.X + 50, Y: at.Y}) != "" {
		t.Fatal("selected distant prospect")
	}
	straight := mid(valley[0], valley[2])
	if PickProspect(c, f, s, straight) != "" {
		t.Fatal("picked endpoint chord instead of actual detour")
	}
	s = ui.State{View: "build", FromCity: "b", ToCity: "a", AlignmentID: "tunnel"}
	if PickProspect(c, f, s, straight) != "bend" {
		t.Fatal("selected alternative is not pickable")
	}
	if len(prospectPaths(f, s)) != 1 {
		t.Fatal("public line included")
	}
	f.Projects = []gameapi.Project{{CorridorID: "bend", Ready: 5}}
	for _, month := range []int{0, 5, 20} {
		f.MonthsResolved = month
		if len(prospectPaths(f, s)) != 0 || PickProspect(c, f, s, straight) != "" {
			t.Fatal("committed line remains a prospect")
		}
	}
}

func TestProductionProspectsCoverTheSeventyFourBuildableCorridors(t *testing.T) {
	p, err := application.Open(nil, nil)
	if err != nil {
		t.Fatal(err)
	}
	f := p.Snapshot()
	for _, view := range []string{"map", "build"} {
		paths := prospectPaths(f, ui.State{View: view})
		got := map[string]bool{}
		for _, path := range paths {
			got[path.id] = true
		}
		if len(paths) != 74 || !got["tsuruga-osaka"] || !got["hokkaido-extension"] {
			t.Fatal(got)
		}
	}
	if len(prospectPaths(f, ui.State{View: "services"})) != 0 {
		t.Fatal("construction cluttered service planning")
	}
}

// A prospect's dots sit under the station marks drawn after them, so a short corridor at a region-wide zoom
// used to vanish: Béziers–Perpignan is 13–55 px long in Frangistan and lost all or some of its dots under the two
// station discs. Measured at every region camera, in both orientations, at phone and desktop map sizes.
func TestProspectDotsStayClearOfTheEndStations(t *testing.T) {
	p, err := application.Open(nil, nil)
	if err != nil {
		t.Fatal(err)
	}
	f := p.Snapshot()
	city := map[gameapi.CityID]gameapi.City{}
	for _, c := range f.Cities {
		city[c.ID] = c
	}
	// The widest station mark is the 10 px construction square; a dot's outer radius is 3.5 px.
	clear := math.Sqrt(50) + 3.5
	sizes := map[bool][][2]float64{false: {{700, 600}, {1068, 1000}, {2937, 1965}}, true: {{390, 330}, {768, 600}}}
	checked, short := 0, 0
	for _, portrait := range []bool{false, true} {
		for _, r := range mapview.VisibleRegions(portrait) {
			for _, size := range sizes[portrait] {
				c := mapview.RegionCamera(size[0], size[1], r.ID, portrait)
				for _, co := range ui.ConstructionOpportunities(f) {
					path := co.Alignments[0].Path
					ends := []mapview.Point{CityPoint(c, city[co.FromCity]), CityPoint(c, city[co.ToCity])}
					length := 0.0
					for i := 1; i < len(path); i++ {
						a, b := c.Screen(mapview.Project(path[i-1].Lat, path[i-1].Lon)), c.Screen(mapview.Project(path[i].Lat, path[i].Lon))
						length += math.Hypot(b.X-a.X, b.Y-a.Y)
					}
					onScreen := func(q mapview.Point) bool { return q.X >= 0 && q.Y >= 0 && q.X <= c.Width && q.Y <= c.Height }
					if !onScreen(ends[0]) || !onScreen(ends[1]) {
						continue
					}
					checked++
					dots := prospectDots(c, path)
					if len(dots) == 0 {
						t.Errorf("%s %v %s: %.0f px prospect has no dots", r.ID, size, co.ID, length)
						continue
					}
					if apart := math.Hypot(ends[1].X-ends[0].X, ends[1].Y-ends[0].Y); len(dots) < int((apart-2*clear)/16)+1 {
						t.Errorf("%s %v %s: %d dots between stations %.0f px apart", r.ID, size, co.ID, len(dots), apart)
					}
					if len(dots) == 1 {
						// Where no dot can clear both marks, the one dot is at least half the stations' distance from each.
						short++
						half := math.Hypot(ends[1].X-ends[0].X, ends[1].Y-ends[0].Y) / 2
						if near := math.Min(math.Hypot(dots[0].X-ends[0].X, dots[0].Y-ends[0].Y), math.Hypot(dots[0].X-ends[1].X, dots[0].Y-ends[1].Y)); near < math.Min(clear, half)-0.5 {
							t.Errorf("%s %v %s: lone dot %.1f px from a station, %.1f px between them", r.ID, size, co.ID, near, 2*half)
						}
						continue
					}
					for _, d := range dots {
						for _, e := range ends {
							if gap := math.Hypot(d.X-e.X, d.Y-e.Y); gap < clear-0.5 {
								t.Errorf("%s %v %s: dot %.1f px from a station, under its mark", r.ID, size, co.ID, gap)
							}
						}
					}
					for i := 1; i < len(dots); i++ {
						if gap := math.Hypot(dots[i].X-dots[i-1].X, dots[i].Y-dots[i-1].Y); gap > 16 {
							t.Errorf("%s %v %s: dots %.1f px apart", r.ID, size, co.ID, gap)
						}
					}
				}
			}
		}
	}
	if checked == 0 || short == 0 {
		t.Fatalf("checked %d on-screen prospects, %d with a lone dot", checked, short)
	}
	t.Logf("checked %d on-screen prospects, %d with a lone dot", checked, short)
}
