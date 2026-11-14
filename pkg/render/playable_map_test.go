package render

import (
	"math"
	"testing"
	"wwhsrt/internal/application"
	"wwhsrt/pkg/gameapi"
	"wwhsrt/pkg/mapview"
)

func TestMapShowsEveryPlayablePublicCorridorAndNoExtraSourceFragments(t *testing.T) {
	layer, e := newPublicLayer()
	if e != nil {
		t.Fatal(e)
	}
	port, e := application.Open(nil, nil)
	if e != nil {
		t.Fatal(e)
	}
	f := port.Snapshot()
	byID := map[string][]gameapi.GeoPoint{}
	for _, co := range f.Corridors {
		if co.Existing {
			byID[co.ID] = co.Alignments[0].Path
		}
	}
	if len(layer.Lines) != len(byID) || len(layer.CorridorIDs) != len(byID) {
		t.Fatal("playable paths lost or source decoration retained")
	}
	seen := map[string]bool{}
	for i, id := range layer.CorridorIDs {
		path, ok := byID[id]
		if !ok || seen[id] {
			t.Fatalf("unplayable or duplicate map corridor: %s", id)
		}
		seen[id] = true
		if len(path) != len(layer.Lines[i]) {
			t.Fatal("map path differs from service geometry")
		}
		for j, p := range path {
			if layer.Lines[i][j] != mapview.Project(p.Lat, p.Lon) {
				t.Fatalf("%s geometry mismatch", id)
			}
		}
	}
	if !seen["public_harbin__shenyang"] || !seen["public_changchun__jilin"] {
		t.Fatal("legitimate northeast routes were hidden")
	}
	// These points lie on the formerly drawn western dead-end fragment. It is
	// several km from the retained Harbin–Shenyang through corridor.
	for _, p := range []gameapi.GeoPoint{{Lon: 124.271, Lat: 43.0289}, {Lon: 124.4899, Lat: 43.257}, {Lon: 125.1136, Lat: 43.746}} {
		closest := math.Inf(1)
		for _, path := range byID {
			closest = min(closest, distanceToRail(p, path))
		}
		if closest < 1000 {
			t.Fatalf("dead fragment unexpectedly retained: %v metres", closest)
		}
	}
	// Dropping the unused rails must not leave unrelated floating station dots.
	for _, stop := range layer.Stops {
		closest := math.Inf(1)
		p := gameapi.GeoPoint{Lon: stop.Lon, Lat: stop.Lat}
		for _, path := range byID {
			closest = min(closest, distanceToRail(p, path))
		}
		if closest > 405 {
			t.Fatalf("unassociated map station %s (%.1fm)", stop.Name, closest)
		}
	}
	if len(layer.Stops) < 1000 || len(layer.Stops) >= 4323 {
		t.Fatal("source station filtering was not applied")
	}
	if f.Checkpoint != port.Snapshot().Checkpoint {
		t.Fatal("map loading changed the world")
	}
}

// Independent local metre projection for the 400m map association test.
func distanceToRail(p gameapi.GeoPoint, path []gameapi.GeoPoint) float64 {
	scale := 111195.0
	cos := math.Cos(p.Lat * math.Pi / 180)
	best := math.Inf(1)
	for i := 1; i < len(path); i++ {
		a, b := path[i-1], path[i]
		ax, ay := (a.Lon-p.Lon)*scale*cos, (a.Lat-p.Lat)*scale
		bx, by := (b.Lon-p.Lon)*scale*cos, (b.Lat-p.Lat)*scale
		dx, dy := bx-ax, by-ay
		u := 0.0
		if d := dx*dx + dy*dy; d > 0 {
			u = max(0, min(1, -(ax*dx+ay*dy)/d))
		}
		best = min(best, math.Hypot(ax+u*dx, ay+u*dy))
	}
	return best
}
