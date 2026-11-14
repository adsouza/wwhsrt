package render

import (
	"encoding/json"
	"os"
	"testing"
	"wwhsrt/assets"
)

func TestWorldCoastReplacesRegionalOutlinesExactlyOnce(t *testing.T) {
	layer, e := newPublicLayer()
	if e != nil {
		t.Fatal(e)
	}
	var base, refined [][][2]float64
	b, e := os.ReadFile("../../assets/scenario/land.json")
	if e != nil {
		t.Fatal(e)
	}
	if e = json.Unmarshal(b, &base); e != nil {
		t.Fatal(e)
	}
	if e = json.Unmarshal(assets.Land, &refined); e != nil {
		t.Fatal(e)
	}
	key := func(r [][2]float64) string { b, _ := json.Marshal(r); return string(b) }
	worldCount := map[string]int{}
	for _, r := range layer.land {
		worldCount[key(r)]++
		if worldCount[key(r)] != 1 {
			t.Fatal("duplicate basemap ring")
		}
	}
	regional := map[string]bool{}
	for _, r := range refined {
		regional[key(r)] = true
		if worldCount[key(r)] != 1 {
			t.Fatal("refined shoreline missing or duplicated")
		}
	}
	removed := 0
	for _, r := range base {
		if !regional[key(r)] {
			removed++
			if worldCount[key(r)] != 0 {
				t.Fatal("coarse shoreline still painted under refinement")
			}
		}
	}
	if removed != 6 {
		t.Fatalf("expected replacement of Japan, China and Taiwan coarse rings, got %d", removed)
	}
	var metadata struct {
		Refinements map[string]struct{ Rings, Vertices int }
	}
	b, e = assets.Terrain.ReadFile("terrain/land.metadata.json")
	if e != nil {
		t.Fatal(e)
	}
	if e = json.Unmarshal(b, &metadata); e != nil {
		t.Fatal(e)
	}
	if metadata.Refinements["Japan"].Vertices < 4000 || metadata.Refinements["China"].Vertices < 9000 || metadata.Refinements["Taiwan"].Vertices < 300 {
		t.Fatal("detailed country shorelines missing")
	}
}
