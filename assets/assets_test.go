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
	raw, e = os.ReadFile("../data/prototypes/map-region-presets.v1.json")
	if e != nil {
		t.Fatal(e)
	}
	var policy struct {
		Regions []struct {
			ID        string `json:"id"`
			Landscape struct {
				IDs []string `json:"metro_ids"`
			} `json:"landscape"`
			Portrait struct {
				IDs []string `json:"metro_ids"`
			} `json:"portrait"`
		} `json:"regions"`
	}
	if e = json.Unmarshal(raw, &policy); e != nil {
		t.Fatal(e)
	}
	var camera map[string][][2]float64
	if e = json.Unmarshal(Camera, &camera); e != nil {
		t.Fatal(e)
	}
	for _, r := range policy.Regions {
		if r.ID != "east-asia" {
			continue
		}
		for orientation, ids := range map[string][]string{"landscape": r.Landscape.IDs, "portrait": r.Portrait.IDs} {
			if len(camera[orientation]) != len(ids) {
				t.Fatal("camera anchor count changed", orientation)
			}
			for i, id := range ids {
				if camera[orientation][i] != coords[id] {
					t.Fatal("camera differs from owning preset", orientation, id)
				}
			}
		}
		return
	}
	t.Fatal("East Asia camera policy missing")
}
