package render

import (
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
		if len(paths) != 96 || !got["tsuruga-osaka"] || !got["hokkaido-extension"] {
			t.Fatal(got)
		}
	}
	if len(prospectPaths(f, ui.State{View: "services"})) != 0 {
		t.Fatal("construction cluttered service planning")
	}
}
