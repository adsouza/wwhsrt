package application

import (
	"encoding/json"
	"reflect"
	"testing"
	"wwhsrt/pkg/gameapi"
)

func stationByCity(f gameapi.Frame, id gameapi.CityID) (gameapi.Station, bool) {
	for _, s := range f.Stations {
		if s.CityID == id {
			return s, true
		}
	}
	return gameapi.Station{}, false
}

func TestM2StationDeliveryAndIndependentOperation(t *testing.T) {
	store := &memoryStore{}
	g := openTest(t, store)
	f := g.Snapshot()
	if len(f.Stations) != 3 || f.TrainFamily.Seats != 400 || f.TrainFamily.LeaseMonthly != 40_000_000 || f.TrainFamily.ID != "swift-400" {
		t.Fatal("starter facilities/family missing", f.Stations, f.TrainFamily)
	}
	for _, id := range []gameapi.CityID{"tokyo", "nagoya", "osaka"} {
		s, ok := stationByCity(f, id)
		if !ok || !s.Open || !s.Public || s.Owner != "" {
			t.Fatal("public station not available", id, s)
		}
	}
	for _, id := range []gameapi.CityID{"shin_hakodate_hokuto", "sapporo"} {
		if _, ok := stationByCity(f, id); ok {
			t.Fatal("unserved city already has a station", id)
		}
	}
	financedBuild(t, g, "hokkaido-extension", "mountain")
	f = g.Snapshot()
	if len(f.Stations) != 5 {
		t.Fatal("independent project did not deliver both station works")
	}
	for _, id := range []gameapi.CityID{"shin_hakodate_hokuto", "sapporo"} {
		s, ok := stationByCity(f, id)
		if !ok || s.Open || s.Public || s.Ready != 48 || s.Owner != f.Player.ID {
			t.Fatal("incorrect construction station state", s)
		}
	}
	rejected(t, g, gameapi.Command{Kind: gameapi.LeaseTrain, CorridorID: "hokkaido-extension"}, "line_not_open")
	for i := 0; i < 3; i++ {
		act(t, g, gameapi.Command{Kind: gameapi.AdvanceYear})
	}
	for i := 0; i < 11; i++ {
		act(t, g, gameapi.Command{Kind: gameapi.AdvanceMonth})
	}
	f = g.Snapshot()
	s, _ := stationByCity(f, "sapporo")
	if s.Open {
		t.Fatal("station opened early")
	}
	act(t, g, gameapi.Command{Kind: gameapi.AdvanceMonth})
	f = g.Snapshot()
	for _, id := range []gameapi.CityID{"shin_hakodate_hokuto", "sapporo"} {
		s, _ := stationByCity(f, id)
		if !s.Open {
			t.Fatal("station did not open with line", s)
		}
	}
	g = openTest(t, store)
	if !reflect.DeepEqual(g.Snapshot(), f) {
		t.Fatal("station lifecycle did not survive restore")
	}
	act(t, g, gameapi.Command{Kind: gameapi.LeaseTrain, CorridorID: "hokkaido-extension"})
	act(t, g, gameapi.Command{Kind: gameapi.SetService, CorridorID: "hokkaido-extension", Frequency: 4, Fare: 11000})
	f = act(t, g, gameapi.Command{Kind: gameapi.AdvanceMonth})
	r := f.Reports[len(f.Reports)-1]
	if r.StationCosts != gameapi.Money(r.Passengers)*f.TrainFamily.StationPerJourney {
		t.Fatal("disclosed station charge differs from settlement")
	}
	found := false
	for _, run := range r.Services {
		if run.CorridorID == "hokkaido-extension" && run.Revenue > 0 {
			found = true
		}
	}
	if !found {
		t.Fatal("independent stations/fleet produced no service")
	}
	raw, _ := g.Export()
	var envelope map[string]any
	if json.Unmarshal(raw, &envelope) != nil || envelope["version"] != float64(3) {
		t.Fatal("network development save version is wrong")
	}
}

func TestM2EstimatesReconcileAndProjectionsAreIsolated(t *testing.T) {
	g := openTest(t, nil)
	f := g.Snapshot()
	before := f.Checkpoint
	want := map[string]gameapi.Money{"tsuruga-osaka/mountain": 1_981_611_000_000, "tsuruga-osaka/valley": 1_163_340_000_000, "hokkaido-extension/mountain": 2_303_565_000_000, "hokkaido-extension/valley": 1_643_235_000_000}
	for _, c := range f.Corridors {
		for _, a := range c.Alignments {
			e := a.Estimate
			if e.Surface+e.Hills+e.Viaducts+e.Tunnels+e.Management+e.Contingency != a.Cost {
				t.Fatal("estimate components do not sum", c.ID, a.ID)
			}
			if cost, ok := want[c.ID+"/"+a.ID]; ok && a.Cost != cost {
				t.Fatal("milestone changed established construction price", c.ID, a.ID)
			}
		}
	}
	f.Stations[0].Owner = "hacked"
	f.TrainFamily.Seats = 1
	f.Corridors[0].Alignments[0].Estimate.Management = 1
	after := g.Snapshot()
	if after.Checkpoint != before || after.Stations[0].Owner == "hacked" || after.TrainFamily.Seats != 400 || after.Corridors[0].Alignments[0].Estimate.Management != 0 {
		t.Fatal("facilities/family/estimate projection aliases world")
	}
	forecast, err := g.Forecast(gameapi.ForecastRequest{CorridorID: "tokyo-nagoya", Frequency: 6, Fare: 9000})
	if err != nil || forecast.Capacity != 148800 || forecast.Passengers != 79348 || forecast.FleetNeeded != 2 {
		t.Fatal("network starter regression", forecast, err)
	}
}

func TestM2ExistingStationIsReusedWithoutOwnershipChange(t *testing.T) {
	g := openTest(t, nil)
	financedBuild(t, g, "tsuruga-osaka", "valley")
	f := g.Snapshot()
	if len(f.Stations) != 4 {
		t.Fatal("public station duplicated", f.Stations)
	}
	osaka, ok := stationByCity(f, "osaka")
	if !ok || !osaka.Public || !osaka.Open || osaka.Owner != "" || osaka.Ready != 0 {
		t.Fatal("new line captured public station", osaka)
	}
	tsuruga, ok := stationByCity(f, "tsuruga")
	if !ok || tsuruga.Public || tsuruga.Open || tsuruga.Owner != f.Player.ID || tsuruga.Ready != 48 {
		t.Fatal("new station works not tracked", tsuruga)
	}
}
