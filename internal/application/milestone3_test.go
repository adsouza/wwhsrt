package application

import (
	"encoding/json"
	"os"
	"reflect"
	"testing"
	"wwhsrt/pkg/gameapi"
)

func TestM3ExtendStarterThroughTokyoNagoyaOsaka(t *testing.T) {
	store := &memoryStore{}
	g := openTest(t, store)
	before := g.Snapshot()
	q := gameapi.ForecastRequest{CorridorID: "tokyo-osaka", Frequency: 6, Fare: 13800, ReplaceServiceID: "tokyo-nagoya"}
	forecast, err := g.Forecast(q)
	if err != nil || !forecast.Open || forecast.KM != 552 || forecast.Minutes != 156 || forecast.FleetNeeded != 3 || forecast.FleetAvailable != 2 || forecast.AdditionalLeases != 1 {
		t.Fatal(forecast, err)
	}
	if g.Snapshot().Checkpoint != before.Checkpoint {
		t.Fatal("extension preview mutated state")
	}
	rejected(t, g, gameapi.Command{Kind: gameapi.ExtendService, CorridorID: q.CorridorID, ReplaceServiceID: q.ReplaceServiceID, Frequency: q.Frequency, Fare: q.Fare}, "fleet_shortage")
	store.fail = true
	rejected(t, g, gameapi.Command{Kind: gameapi.ExtendService, CorridorID: q.CorridorID, ReplaceServiceID: q.ReplaceServiceID, Frequency: q.Frequency, Fare: q.Fare, LeaseCount: 1}, "storage_failed")
	if g.Snapshot().Checkpoint != before.Checkpoint {
		t.Fatal("failed extension changed service/fleet")
	}
	store.fail = false
	f := act(t, g, gameapi.Command{Kind: gameapi.ExtendService, CorridorID: q.CorridorID, ReplaceServiceID: q.ReplaceServiceID, Frequency: q.Frequency, Fare: q.Fare, LeaseCount: 1})
	if len(f.Fleet) != 3 || len(f.IdleLines) != 0 {
		t.Fatal("starter fleet not transferred exactly once", f.Fleet, f.IdleLines)
	}
	for _, train := range f.Fleet {
		if train.CorridorID != "tokyo-osaka" {
			t.Fatal("train left behind", train)
		}
	}
	f = act(t, g, gameapi.Command{Kind: gameapi.AdvanceMonth})
	n := f.Network
	r := f.Reports[0]
	if n.Carried == 0 || r.Passengers != n.Carried || r.Revenue != n.Revenue || r.LeaseCosts != 120_000_000 || len(f.Day) != 12 {
		t.Fatal("network settlement/day missing", r)
	}
	if r.ClosingCash-before.Player.Cash-(714_132_000-315_425_800) != forecast.CompanyChange {
		t.Fatal("marginal company forecast failed to reconcile", forecast, r)
	}
	through := 0
	for _, flow := range n.Journeys {
		if flow.From == "tokyo" && flow.To == "osaka" && len(flow.Services) == 1 && flow.Services[0] == "tokyo-osaka" {
			through += flow.Passengers
		}
	}
	if through == 0 {
		t.Fatal("no actual Tokyo to Osaka through passenger")
	}
	if !reflect.DeepEqual(openTest(t, store).Snapshot(), f) {
		t.Fatal("through network did not resume exactly")
	}
	// Detailed read models cannot alias saved state or future resolution inputs.
	f.Routes[0].Cities[0] = "hacked"
	f.Network.Journeys[0].Services[0] = "hacked"
	f.Day[0].Legs[0].LoadPermille = 9999
	if g.Snapshot().Network.Journeys[0].Services[0] == "hacked" || g.Snapshot().Day[0].Legs[0].LoadPermille == 9999 {
		t.Fatal("projection aliases world")
	}
}
func TestM3LegacyV2MigrationPreservesHistoryAndFailureAtomicity(t *testing.T) {
	fixture, err := os.ReadFile("testdata/milestone2-v2.json")
	if err != nil {
		t.Fatal(err)
	}
	s := &memoryStore{b: fixture}
	g := openTest(t, s)
	f := g.Snapshot()
	if string(s.b) != string(fixture) || f.MonthsResolved != 2 || f.Reports[0].Revenue != 1_150_596_000 || len(f.Fleet) != 2 || f.Network.Available {
		t.Fatal("v2 history was rewritten")
	}
	s.fail = true
	rejected(t, g, gameapi.Command{Kind: gameapi.AdvanceMonth}, "storage_failed")
	if string(s.b) != string(fixture) {
		t.Fatal("failed migration overwrote original v2 save")
	}
	s.fail = false
	f = act(t, g, gameapi.Command{Kind: gameapi.AdvanceMonth})
	raw, _ := g.Export()
	var env envelope
	_ = json.Unmarshal(raw, &env)
	if env.Version != 3 || !f.Network.Available || f.Reports[0].Revenue != 1_150_596_000 {
		t.Fatal("v3 upgrade did not preserve historical accounting")
	}
	if !reflect.DeepEqual(openTest(t, s).Snapshot(), f) {
		t.Fatal("upgraded save did not resume")
	}
}
func TestM3InvalidPathsCapacityAndReportTampering(t *testing.T) {
	g := openTest(t, nil)
	rejected(t, g, gameapi.Command{Kind: gameapi.SetService, CorridorID: "tokyo>osaka>tokyo", Frequency: 1, Fare: 9000}, "no_route")
	rejected(t, g, gameapi.Command{Kind: gameapi.ExtendService, CorridorID: "tokyo-osaka-express", ReplaceServiceID: "tokyo-nagoya", Frequency: 6, Fare: 13800, LeaseCount: 1}, "invalid_extension")
	rejected(t, g, gameapi.Command{Kind: gameapi.SetService, CorridorID: "tokyo-tsuruga", Frequency: 1, Fare: 9000}, "line_not_open")
	act(t, g, gameapi.Command{Kind: gameapi.AdvanceMonth})
	raw, _ := g.Export()
	var env envelope
	_ = json.Unmarshal(raw, &env)
	var p payload
	_ = json.Unmarshal(env.Payload, &p)
	p.Reports[0].Services[0].Passengers++ // Sums alone could miss incorrect seat allocation.
	env.Payload, _ = json.Marshal(p)
	env.SHA = hash(env.Payload)
	bad, _ := json.Marshal(env)
	s := &memoryStore{b: bad}
	if _, err := Open(s, nil); err == nil {
		t.Fatal("re-signed corrupted network accepted")
	}
	if string(s.b) != string(bad) {
		t.Fatal("invalid network overwritten")
	}
}

func TestM3RestoreChecksInfrastructureBeforeNetworkReplay(t *testing.T) {
	g := openTest(t, nil)
	act(t, g, gameapi.Command{Kind: gameapi.TakeLoan, Amount: 200_000_000_000})
	act(t, g, gameapi.Command{Kind: gameapi.BuildLine, CorridorID: "tsuruga-osaka", AlignmentID: "valley"})
	for i := 0; i < 4; i++ {
		act(t, g, gameapi.Command{Kind: gameapi.AdvanceYear})
	}
	act(t, g, gameapi.Command{Kind: gameapi.LeaseTrain, CorridorID: "tsuruga-osaka"})
	act(t, g, gameapi.Command{Kind: gameapi.SetService, CorridorID: "tsuruga-osaka", Frequency: 4, Fare: 5000})
	act(t, g, gameapi.Command{Kind: gameapi.AdvanceMonth})
	raw, _ := g.Export()
	var env envelope
	_ = json.Unmarshal(raw, &env)
	var p payload
	_ = json.Unmarshal(env.Payload, &p)
	p.Projects[0].AlignmentID = "missing"
	for i := range p.Reports[len(p.Reports)-1].Services {
		if p.Reports[len(p.Reports)-1].Services[i].CorridorID == "tsuruga-osaka" {
			p.Reports[len(p.Reports)-1].Services[i].AlignmentID = ""
		}
	}
	env.Payload, _ = json.Marshal(p)
	env.SHA = hash(env.Payload)
	bad, _ := json.Marshal(env)
	s := &memoryStore{b: bad}
	if _, err := Open(s, nil); err == nil {
		t.Fatal("invalid engineering reference accepted")
	}
	if string(s.b) != string(bad) {
		t.Fatal("invalid infrastructure save overwritten")
	}
}
