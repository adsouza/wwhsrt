package render

import (
	"slices"
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
		if len(paths) != 101 || !got["tsuruga-osaka"] || !got["hokkaido-extension"] {
			t.Fatal(got)
		}
	}
	if len(prospectPaths(f, ui.State{View: "services"})) != 0 {
		t.Fatal("construction cluttered service planning")
	}
}

func TestViennaGrazProspectFollowsBothApproaches(t *testing.T) {
	game, err := application.Open(nil, nil)
	if err != nil {
		t.Fatal(err)
	}
	frame := game.Snapshot()
	const tunnel = "semmering_base_tunnel"
	state := ui.State{View: "map"}
	for _, view := range []string{"map", "build"} {
		if view == "build" && !ui.SelectProspect(&state, frame, tunnel) {
			t.Fatal("Semmering contract missing from the construction list")
		}
		found := false
		for _, line := range prospectPaths(frame, state) {
			if line.id != tunnel {
				continue
			}
			found = true
			if line.selected != (view == "build") || len(line.Paths) != 3 ||
				!slices.Equal(line.Stops, []gameapi.CityID{"vienna", "gloggnitz", "muerzzuschlag", "graz"}) {
				t.Fatalf("%s has incorrect sections or stops", view)
			}
			at := slices.IndexFunc(frame.Cities, func(c gameapi.City) bool { return c.ID == "vienna" })
			if at < 0 || frame.Cities[at].StationName != "Wien Hauptbahnhof" || len(line.AlternateStations) != 0 {
				t.Fatal("Vienna must be marked once at Hauptbahnhof")
			}
			start := line.Paths[0][0]
			station := frame.Cities[at]
			if start.Lat != station.StationLat || start.Lon != station.StationLon {
				t.Fatal("Vienna marker is detached from the route")
			}
			for i, id := range []string{"public_vienna__gloggnitz", tunnel, "public_muerzzuschlag__graz"} {
				at := slices.IndexFunc(frame.Corridors, func(c gameapi.Corridor) bool { return c.ID == id })
				path := frame.Corridors[at].Alignments[0].Path
				if !slices.Equal(line.Paths[i], path) {
					t.Fatal("wrong railway alignment for", id)
				}
				mid := path[len(path)/2]
				camera := mapview.Fit(900, 600, []mapview.Point{mapview.Project(mid.Lat-.3, mid.Lon-.3), mapview.Project(mid.Lat+.3, mid.Lon+.3)})
				if pick := PickProspect(camera, frame, state, camera.Screen(mapview.Project(mid.Lat, mid.Lon))); pick != tunnel {
					t.Fatalf("picked %q on %s in %s view", pick, id, view)
				}
			}
		}
		if !found {
			t.Fatalf("Vienna–Graz missing in %s view", view)
		}
	}
}

func TestWarsawCentralnaAndJaktorowAreDistinctMapEndpoints(t *testing.T) {
	port, err := application.Open(nil, nil)
	if err != nil {
		t.Fatal(err)
	}
	f := port.Snapshot()
	const project = "launch_28_warsaw__lodz"
	var state ui.State
	if !ui.SelectProspect(&state, f, project) {
		t.Fatal("Łódź contract missing")
	}
	co, _ := ui.Pair(f, state.FromCity, state.ToCity)
	works := ui.MapForProspect(f, co, co.Alignments[0])
	if len(works.AlternateStations) != 1 {
		t.Fatal("expected the named construction terminal", works.AlternateStations)
	}
	origin := works.AlternateStations[0]
	if origin.StationName != "Jaktorów" || origin.ID != "warsaw" || works.Paths[0][0] != (gameapi.GeoPoint{Lat: origin.StationLat, Lon: origin.StationLon}) {
		t.Fatal("Łódź works detached from Jaktorów", origin)
	}
	central := mapview.Project(52.22882, 21.00316)
	junction := mapview.Project(52.08667, 20.55188)
	camera := mapview.Fit(900, 600, []mapview.Point{central, junction})
	track := slices.IndexFunc(f.Corridors, func(c gameapi.Corridor) bool { return c.ID == "public_warsaw__wloszczowa" })
	if track < 0 {
		t.Fatal("Warsaw public line missing")
	}
	end := f.Corridors[track].Alignments[0].Path[0]
	i := slices.IndexFunc(f.Cities, func(c gameapi.City) bool { return c.ID == "warsaw" })
	if i < 0 || f.Cities[i].StationName != "Warszawa Centralna" || mapview.Project(end.Lat, end.Lon) != central || CityPoint(camera, f.Cities[i]) != camera.Screen(central) || Pick(camera, f.Cities, camera.Screen(central)) != "warsaw" {
		t.Fatal("Warsaw marker detached from public line")
	}
	if CityPoint(camera, origin) != camera.Screen(junction) || PickProspect(camera, f, state, camera.Screen(junction)) != project || PickProspect(camera, f, state, camera.Screen(central)) != "" {
		t.Fatal("incorrect construction endpoints")
	}
	f.Projects = append(f.Projects, gameapi.Project{CorridorID: project})
	if station := ui.ConstructionTerminals(f)["warsaw"]; station != origin {
		t.Fatal("Jaktorów disappeared once construction started", station)
	}
}
