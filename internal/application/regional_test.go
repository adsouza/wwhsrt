package application

import (
	"bytes"
	"encoding/json"
	"os"
	"reflect"
	"testing"

	"wwhsrt/assets"
	"wwhsrt/pkg/gameapi"
)

func TestRegionalV8UpgradePreservesHistoricalAccountsAndAssets(t *testing.T) {
	for _, fixture := range []string{
		"regional-v8.json",
		"regional-cutover-v8.json",
		"regional-assets-v8.json",
		"regional-texas-v8.json",
		"regional-korea-v8.json",
	} {
		t.Run(fixture, func(t *testing.T) {
			raw, err := os.ReadFile("testdata/" + fixture)
			if err != nil {
				t.Fatal(err)
			}
			var env envelope
			var old constructionPayload
			if json.Unmarshal(raw, &env) != nil || json.Unmarshal(env.Payload, &old) != nil || env.Version != 8 ||
				old.PublicSHA != regionalPriorSHA {
				t.Fatal("fixture is not genuine v8")
			}
			store := &memoryStore{b: raw}
			g := openPublic(t, store)
			f := g.Snapshot()
			if !bytes.Equal(raw, store.b) || !reflect.DeepEqual(old.Reports, f.Reports) || old.Company != f.Player ||
				!reflect.DeepEqual(old.Projects, f.Projects) ||
				!reflect.DeepEqual(old.Services, f.Services) ||
				!reflect.DeepEqual(old.Fleet, f.Fleet) ||
				old.Debt != f.Debt ||
				old.Pending != f.Pending ||
				g.w.RegionalSince() != old.Months {
				t.Fatal("read upgrade changed saved company state")
			}
			if fixture == "regional-texas-v8.json" {
				var oldRoute, current gameapi.Route
				for _, r := range f.Routes {
					if r.ID == "launch_route_houston__san_antonio" {
						oldRoute = r
					}
					if r.ID == "launch_route_houston__san_antonio_regional" {
						current = r
					}
				}
				if !oldRoute.Open || current.Open ||
					!reflect.DeepEqual(
						oldRoute.Cities,
						[]gameapi.CityID{"houston", "dallas", "austin", "san_antonio"},
					) {
					t.Fatal("existing Texas service was redirected over unbuilt new track", oldRoute, current)
				}
			}
			store.fail = true
			if _, err := g.Apply(
				gameapi.Command{Kind: gameapi.AdvanceMonth, Revision: f.Revision},
			); err == nil || g.Snapshot().Checkpoint != f.Checkpoint ||
				!bytes.Equal(raw, store.b) {
				t.Fatal("failed migration commit changed memory/save")
			}
			store.fail = false
			f = publicCommand(t, g, gameapi.Command{Kind: gameapi.AdvanceMonth})
			if !reflect.DeepEqual(f.Reports[:old.Months], old.Reports) ||
				openPublic(t, store).Snapshot().Checkpoint != f.Checkpoint {
				t.Fatal("historical/new demand epochs cannot reload")
			}
			var next viennaPayload
			json.Unmarshal(store.b, &env)
			json.Unmarshal(env.Payload, &next)
			if env.Version != 18 || next.Rules != "network-v18" || next.RegionalSince != old.Months ||
				next.RegionalPriorSHA != regionalPriorSHA ||
				next.PublicSHA != hash(assets.PublicNetwork) ||
				next.ConstructionSince != old.ConstructionSince {
				t.Fatal("migration catalog identity/epochs incorrect")
			}
			for _, mutate := range []func(*viennaPayload){
				func(p *viennaPayload) { p.RegionalPriorSHA = "unknown" },
				func(p *viennaPayload) { p.RegionalSince = -1 },
				func(p *viennaPayload) { p.RegionalSince = p.Months + 1 },
				func(p *viennaPayload) { p.RegionalSince = 0; p.ConstructionSince = p.Months },
			} {
				bad := next
				mutate(&bad)
				env.Payload, _ = json.Marshal(bad)
				env.SHA = hash(env.Payload)
				b, _ := json.Marshal(env)
				if _, err := Open(&memoryStore{b: b}, nil); err == nil {
					t.Fatal("invalid regional metadata accepted")
				}
			}
		})
	}
}

func TestRegionalTaipeiThroughServiceCarriesPassengersAndReloads(t *testing.T) {
	store := &memoryStore{}
	g := openPublic(t, store)
	f := g.Snapshot()
	found := false
	for _, c := range f.Cities {
		if c.ID == "taipei" {
			found = true
			if c.StationName != "Taipei" || c.StationLat != 25.04793 || c.StationLon != 121.517 {
				t.Fatal("wrong Taipei terminal", c)
			}
		}
	}
	if !found {
		t.Fatal("Taipei excluded")
	}
	id := "network_taiwan_hsr_all_stops"
	fc, err := g.Forecast(gameapi.ForecastRequest{CorridorID: id, Frequency: 1, Fare: 4500})
	if err != nil || !fc.Open {
		t.Fatal("Taiwan through service unavailable", err)
	}
	for i := 0; i < fc.AdditionalLeases; i++ {
		publicCommand(t, g, gameapi.Command{Kind: gameapi.LeaseTrain, CorridorID: id})
	}
	publicCommand(t, g, gameapi.Command{Kind: gameapi.SetService, CorridorID: id, Frequency: 1, Fare: 4500})
	f = publicCommand(t, g, gameapi.Command{Kind: gameapi.AdvanceMonth})
	carried := false
	for _, j := range f.Network.Journeys {
		if (j.From == "taipei" && j.To == "kaohsiung" || j.To == "taipei" && j.From == "kaohsiung") &&
			j.Passengers > 0 {
			carried = true
		}
	}
	if !carried || openPublic(t, store).Snapshot().Checkpoint != f.Checkpoint {
		t.Fatal("Taipei end-to-end service/reload failed")
	}
}

func TestRegionalDaeguUlsanHasNoYeongcheonTurnaround(t *testing.T) {
	g := openPublic(t, nil)
	before, _ := g.Export()
	f := g.Snapshot()
	found := false
	for _, c := range f.Corridors {
		if c.ID != "public_daegu__ulsan" {
			continue
		}
		found = true
		for _, p := range c.Alignments[0].Path {
			if p.Lat > 35.9 {
				t.Fatal("Yeongcheon out-and-back retained", p)
			}
		}
		if c.Alignments[0].KM != 79 {
			t.Fatal("service length does not match the repaired HSR alignment")
		}
	}
	if !found {
		t.Fatal("Daegu–Ulsan removed instead of repaired")
	}
	for _, c := range f.Cities {
		if c.ID == "ulsan" && (c.StationLon != 129.13793 || c.StationLat != 35.55131) {
			t.Fatal("Ulsan is still pinned to Bugulsan")
		}
	}
	after, _ := g.Export()
	if !bytes.Equal(before, after) {
		t.Fatal("map inspection changed save")
	}
}

func TestRegionalHoustonAustinEnablesNewSanAntonioThroughService(t *testing.T) {
	store := &memoryStore{}
	g := openPublic(t, store)
	for _, id := range []string{"launch_71_houston__austin", "launch_07_austin__san_antonio"} {
		f := g.Snapshot()
		var co gameapi.Corridor
		for _, c := range f.Corridors {
			if c.ID == id {
				co = c
			}
		}
		if co.ID == "" {
			t.Fatal("missing Texas prospect", id)
		}
		a := co.Alignments[0]
		shortfall := max(gameapi.Money(0), a.Cost-a.Grant-f.Player.Cash)
		if shortfall > 0 {
			publicCommand(t, g, gameapi.Command{Kind: gameapi.TakeLoan, Amount: shortfall})
		}
		publicCommand(t, g, gameapi.Command{Kind: gameapi.BuildLine, CorridorID: id, AlignmentID: a.ID})
	}
	id := "launch_route_houston__san_antonio_regional"
	fc, err := g.Forecast(gameapi.ForecastRequest{CorridorID: id, Frequency: 1, Fare: 4500})
	if err == nil || err.Error() != "line_not_open" || fc.Open {
		t.Fatal("new through service opens before its works", err)
	}
	for i := 0; i < 3; i++ {
		publicCommand(t, g, gameapi.Command{Kind: gameapi.AdvanceYear})
	}
	fc, err = g.Forecast(gameapi.ForecastRequest{CorridorID: id, Frequency: 1, Fare: 4500})
	if err != nil || !fc.Open {
		t.Fatal("completed Texas works do not enable through service", err)
	}
	for i := 0; i < fc.AdditionalLeases; i++ {
		publicCommand(t, g, gameapi.Command{Kind: gameapi.LeaseTrain, CorridorID: id})
	}
	publicCommand(t, g, gameapi.Command{Kind: gameapi.SetService, CorridorID: id, Frequency: 1, Fare: 4500})
	f := publicCommand(t, g, gameapi.Command{Kind: gameapi.AdvanceMonth})
	carried := false
	for _, j := range f.Network.Journeys {
		if (j.From == "houston" && j.To == "san_antonio" || j.To == "houston" && j.From == "san_antonio") &&
			j.Passengers > 0 {
			carried = true
		}
	}
	if !carried || openPublic(t, store).Snapshot().Checkpoint != f.Checkpoint {
		t.Fatal("new Texas through passengers/reload failed")
	}
}
