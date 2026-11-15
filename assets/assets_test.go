package assets

import (
	"encoding/json"
	"os"
	"testing"
	"wwhsrt/pkg/gameapi"
)

func TestEastAsiaScenarioMatchesSourceAndCameraPolicy(t *testing.T) {
	var scenario struct {
		ID     string         `json:"id"`
		Cities []gameapi.City `json:"cities"`
	}
	if e := json.Unmarshal(Scenario, &scenario); e != nil {
		t.Fatal(e)
	}
	if scenario.ID != "east-asia-first-business-v2" || len(scenario.Cities) != 12 {
		t.Fatal("unexpected first region/roster")
	}
	raw, e := os.ReadFile("../data/prototypes/playable-metros.v1.json")
	if e != nil {
		t.Fatal(e)
	}
	var source []struct {
		ID  string  `json:"id"`
		Lat float64 `json:"lat"`
		Lon float64 `json:"lng"`
	}
	if e = json.Unmarshal(raw, &source); e != nil {
		t.Fatal(e)
	}
	coords := map[string][2]float64{}
	for _, c := range source {
		coords[c.ID] = [2]float64{c.Lat, c.Lon}
	}
	coords["tsuruga"] = [2]float64{35.645, 136.076}
	coords["shin_hakodate_hokuto"] = [2]float64{41.904, 140.648}
	coords["sapporo"] = [2]float64{43.0687, 141.3507}
	seen := map[gameapi.CityID]bool{}
	for _, c := range scenario.Cities {
		if seen[c.ID] || coords[string(c.ID)] != [2]float64{c.Lat, c.Lon} {
			t.Fatal("duplicate or changed source coordinates", c.ID)
		}
		seen[c.ID] = true
	}
	// The launch map applies its framing overrides to the base presets, and
	// those overrides may add anchors outside the metro roster (Harbin).
	raw, e = os.ReadFile("../data/prototypes/launch-map-framing.v1.json")
	if e != nil {
		t.Fatal(e)
	}
	type anchorSet struct {
		IDs []string `json:"metro_ids"`
	}
	type regionPolicy struct {
		ID        string    `json:"id"`
		Exclude   []string  `json:"exclude_metro_ids"`
		Landscape anchorSet `json:"landscape"`
		Portrait  anchorSet `json:"portrait"`
	}
	var launch struct {
		Anchors []struct {
			ID  string  `json:"id"`
			Lat float64 `json:"lat"`
			Lon float64 `json:"lng"`
		} `json:"additional_anchors"`
		Overrides []regionPolicy `json:"region_overrides"`
	}
	if e = json.Unmarshal(raw, &launch); e != nil {
		t.Fatal(e)
	}
	for _, a := range launch.Anchors {
		coords[a.ID] = [2]float64{a.Lat, a.Lon}
	}
	raw, e = os.ReadFile("../data/prototypes/map-region-presets.v1.json")
	if e != nil {
		t.Fatal(e)
	}
	var policy struct {
		Regions []regionPolicy `json:"regions"`
	}
	if e = json.Unmarshal(raw, &policy); e != nil {
		t.Fatal(e)
	}
	var effective *regionPolicy
	for i := range policy.Regions {
		if policy.Regions[i].ID == "east-asia" {
			effective = &policy.Regions[i]
		}
	}
	if effective == nil {
		t.Fatal("East Asia camera policy missing")
	}
	for _, o := range launch.Overrides {
		if o.ID != "east-asia" {
			continue
		}
		if o.Landscape.IDs != nil {
			effective.Landscape = o.Landscape
		}
		if o.Portrait.IDs != nil {
			effective.Portrait = o.Portrait
		}
	}
	var camera map[string][][2]float64
	if e = json.Unmarshal(Camera, &camera); e != nil {
		t.Fatal(e)
	}
	// Metros the preset keeps off screen ride along as camera.json "exclude".
	if len(camera["exclude"]) != len(effective.Exclude) || len(effective.Exclude) == 0 {
		t.Fatal("camera exclusions differ from the preset", camera["exclude"], effective.Exclude)
	}
	for i, id := range effective.Exclude {
		if camera["exclude"][i] != coords[id] {
			t.Fatal("camera exclusion differs from the preset", id)
		}
	}
	var regions []struct {
		ID        string `json:"id"`
		Landscape struct {
			Points [][2]float64 `json:"points"`
		} `json:"landscape"`
		Portrait struct {
			Points [][2]float64 `json:"points"`
		} `json:"portrait"`
	}
	if e = json.Unmarshal(Regions, &regions); e != nil {
		t.Fatal(e)
	}
	menu := map[string][][2]float64{}
	for _, r := range regions {
		if r.ID == "east-asia" {
			menu["landscape"], menu["portrait"] = r.Landscape.Points, r.Portrait.Points
		}
	}
	for orientation, ids := range map[string][]string{"landscape": effective.Landscape.IDs, "portrait": effective.Portrait.IDs} {
		if len(camera[orientation]) != len(ids) || len(menu[orientation]) != len(ids) {
			t.Fatal("camera anchor count changed", orientation)
		}
		for i, id := range ids {
			want, ok := coords[id]
			if !ok {
				t.Fatal("camera anchor has no coordinates", orientation, id)
			}
			if camera[orientation][i] != want {
				t.Fatal("camera differs from owning preset", orientation, id)
			}
			// regions.json stores the same anchors as lon/lat for the region picker.
			if menu[orientation][i] != [2]float64{want[1], want[0]} {
				t.Fatal("regions.json East Asia differs from camera.json", orientation, id)
			}
		}
	}
}

// Harbin is the northern extent of East Asia in both orientations.
func TestEastAsiaNorthernExtentIsHarbin(t *testing.T) {
	var camera map[string][][2]float64
	if e := json.Unmarshal(Camera, &camera); e != nil {
		t.Fatal(e)
	}
	harbin := [2]float64{45.75, 126.65}
	for _, orientation := range []string{"landscape", "portrait"} {
		found := false
		for _, p := range camera[orientation] {
			if p == harbin {
				found = true
			} else if p[0] >= harbin[0] {
				t.Fatal("an anchor north of Harbin", orientation, p)
			}
		}
		if !found {
			t.Fatal("Harbin is not an East Asia anchor", orientation)
		}
	}
}

// Every regions.json view keeps off screen exactly the metros its preset excludes, whether the
// preset names them for the whole region (East Asia) or for one orientation (portrait Africa+Arabia).
func TestRegionExclusionsMatchPresets(t *testing.T) {
	raw, e := os.ReadFile("../data/prototypes/playable-metros.v1.json")
	if e != nil {
		t.Fatal(e)
	}
	var metros []struct {
		ID  string  `json:"id"`
		Lat float64 `json:"lat"`
		Lon float64 `json:"lng"`
	}
	if e = json.Unmarshal(raw, &metros); e != nil {
		t.Fatal(e)
	}
	lonLat := map[string][2]float64{}
	for _, m := range metros {
		lonLat[m.ID] = [2]float64{m.Lon, m.Lat}
	}
	raw, e = os.ReadFile("../data/prototypes/map-region-presets.v1.json")
	if e != nil {
		t.Fatal(e)
	}
	type view struct {
		Exclude []string `json:"exclude_metro_ids"`
	}
	var presets struct {
		Regions []struct {
			ID        string   `json:"id"`
			Exclude   []string `json:"exclude_metro_ids"`
			Landscape *view    `json:"landscape"`
			Portrait  *view    `json:"portrait"`
		} `json:"regions"`
	}
	if e = json.Unmarshal(raw, &presets); e != nil {
		t.Fatal(e)
	}
	var regions []struct {
		ID        string `json:"id"`
		Landscape struct {
			Exclude [][2]float64 `json:"exclude"`
		} `json:"landscape"`
		Portrait struct {
			Exclude [][2]float64 `json:"exclude"`
		} `json:"portrait"`
	}
	if e = json.Unmarshal(Regions, &regions); e != nil {
		t.Fatal(e)
	}
	got := map[string][][2]float64{}
	for _, r := range regions {
		got[r.ID+"/landscape"], got[r.ID+"/portrait"] = r.Landscape.Exclude, r.Portrait.Exclude
	}
	checked := 0
	for _, p := range presets.Regions {
		for orientation, v := range map[string]*view{"landscape": p.Landscape, "portrait": p.Portrait} {
			ids := p.Exclude
			if v != nil && v.Exclude != nil {
				ids = v.Exclude
			}
			key := p.ID + "/" + orientation
			if len(got[key]) != len(ids) {
				t.Fatal("regions.json exclusions differ from the preset", key, got[key], ids)
			}
			for i, id := range ids {
				if want, ok := lonLat[id]; !ok || got[key][i] != want {
					t.Fatal("regions.json exclusion differs from the preset", key, id)
				}
				checked++
			}
		}
	}
	if checked != 3 { // Dhaka twice for East Asia, Karachi once for portrait Africa+Arabia
		t.Fatal("expected three preset exclusions, checked", checked)
	}
}
