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

func TestEconomyV4MigrationPreservesHistoryAndCommitsAtomically(t *testing.T) {
	raw, e := os.ReadFile("testdata/economy-v4.json")
	if e != nil {
		t.Fatal(e)
	}
	var old envelope
	var prior expandedPayload
	json.Unmarshal(raw, &old)
	json.Unmarshal(old.Payload, &prior)
	store := &memoryStore{b: raw}
	g := openPublic(t, store)
	f := g.Snapshot()
	if !bytes.Equal(raw, store.b) || !reflect.DeepEqual(prior.Reports, f.Reports) || prior.Company != f.Player ||
		!reflect.DeepEqual(prior.Fleet, f.Fleet) ||
		!reflect.DeepEqual(prior.Services, f.Services) ||
		g.w.EconomySince() != 2 {
		t.Fatal("read migration changed existing accounts/plans")
	}
	if f.Reports[0].LeaseCosts != 80000000 || f.TrainFamily.LeaseMonthly != 12000000 {
		t.Fatal("historical/current rates mixed")
	}
	before := f.Checkpoint
	store.fail = true
	if _, e = g.Apply(
		gameapi.Command{Kind: gameapi.AdvanceMonth, Revision: f.Revision},
	); e == nil || g.Snapshot().Checkpoint != before ||
		!bytes.Equal(store.b, raw) {
		t.Fatal("failed migration mutated save")
	}
	store.fail = false
	f = publicCommand(t, g, gameapi.Command{Kind: gameapi.AdvanceMonth})
	if !reflect.DeepEqual(f.Reports[:2], prior.Reports) || f.Reports[2].LeaseCosts != 24000000 {
		t.Fatal("cutover rewrote history or failed to apply new rates")
	}
	var env envelope
	var p connectorPayload
	json.Unmarshal(store.b, &env)
	json.Unmarshal(env.Payload, &p)
	if env.Version != 10 || p.Rules != "network-v10" || p.EconomySince != 2 || p.EconomySHA != hash(assets.Economy) {
		t.Fatal("economy identity/cutover missing")
	}
	if openPublic(t, store).Snapshot().Checkpoint != f.Checkpoint {
		t.Fatal("mixed-rule save cannot continue")
	}
	for _, mutate := range []func(*connectorPayload){
		func(p *connectorPayload) { p.EconomySHA = "unknown" },
		func(p *connectorPayload) { p.EconomySince = 0 },
		func(p *connectorPayload) { p.EconomySince = 4 },
	} {
		bad := p
		mutate(&bad)
		env.Payload, _ = json.Marshal(bad)
		env.SHA = hash(env.Payload)
		b, _ := json.Marshal(env)
		if _, e = Open(&memoryStore{b: b}, nil); e == nil {
			t.Fatal("invalid economy identity/cutover accepted")
		}
	}
}

func TestEconomyForecastBreakEvenAndSettlementReconcile(t *testing.T) {
	for _, through := range []bool{false, true} {
		g := openPublic(t, &memoryStore{})
		baseline := openPublic(t, &memoryStore{})
		id := "public_lyon__marseille"
		q := gameapi.ForecastRequest{CorridorID: id, Frequency: 1, Fare: 4446}
		if through {
			id = "tokyo-osaka"
			q = gameapi.ForecastRequest{CorridorID: id, Frequency: 6, Fare: 8128, ReplaceServiceID: "tokyo-nagoya"}
		}
		before := g.Snapshot().Checkpoint
		f, e := g.Forecast(q)
		if e != nil {
			t.Fatal(e)
		}
		if g.Snapshot().Checkpoint != before || !f.BreakEvenAvailable ||
			f.Costs != f.RunningCosts+f.AccessCosts+f.HandlingCosts+f.LeaseCosts+f.MaintenanceCosts {
			t.Fatal("forecast/expense breakdown invalid", f)
		}
		// New station markets and cross-border endpoints redistribute the finite demand budget.
		if !through && (f.BreakEvenPassengers != 6399 || f.Passengers != 8723 || f.LeaseCosts != 12000000) {
			t.Fatal("regional calibration drift", f)
		}
		if through {
			publicCommand(
				t,
				g,
				gameapi.Command{
					Kind:             gameapi.ExtendService,
					CorridorID:       id,
					ReplaceServiceID: q.ReplaceServiceID,
					Frequency:        q.Frequency,
					Fare:             q.Fare,
					LeaseCount:       f.AdditionalLeases,
				},
			)
		} else {
			for i := 0; i < f.AdditionalLeases; i++ {
				publicCommand(t, g, gameapi.Command{Kind: gameapi.LeaseTrain, CorridorID: id})
			}
			publicCommand(
				t,
				g,
				gameapi.Command{Kind: gameapi.SetService, CorridorID: id, Frequency: q.Frequency, Fare: q.Fare},
			)
		}
		base := publicCommand(t, baseline, gameapi.Command{Kind: gameapi.AdvanceMonth})
		next := publicCommand(t, g, gameapi.Command{Kind: gameapi.AdvanceMonth})
		if next.Player.Cash-base.Player.Cash != f.CompanyChange {
			t.Fatal("forecast company change disagrees with settlement", f, next.Reports[0], base.Reports[0])
		}
		if next.Reports[0].Revenue != f.CompanyRevenue {
			t.Fatal("forecast revenue disagrees with settlement")
		}
		b, _ := g.Export()
		if _, e = Open(&memoryStore{b: b}, nil); e != nil {
			t.Fatal("new economic month does not restore", e)
		}
	}
}
