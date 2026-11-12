package application

import (
	"encoding/json"
	"testing"
	"wwhsrt/pkg/gameapi"
)

func act(t *testing.T, g *Game, c gameapi.Command) gameapi.Frame {
	t.Helper()
	c.Revision = g.Snapshot().Revision
	f, e := g.Apply(c)
	if e != nil {
		t.Fatalf("%s: %v", c.Kind, e)
	}
	return f
}
func rejected(t *testing.T, g *Game, c gameapi.Command, code string) {
	t.Helper()
	before := g.Snapshot().Checkpoint
	c.Revision = g.Snapshot().Revision
	if _, e := g.Apply(c); e == nil || e.Error() != code {
		t.Fatalf("%s: wanted %s, got %v", c.Kind, code, e)
	}
	if g.Snapshot().Checkpoint != before {
		t.Fatal("rejection mutated world")
	}
}
func financedBuild(t *testing.T, g *Game, id, alignment string) {
	t.Helper()
	for i := 0; i < 5; i++ {
		act(t, g, gameapi.Command{Kind: gameapi.TakeLoan, Amount: 50_000_000_000})
	}
	act(t, g, gameapi.Command{Kind: gameapi.BuildLine, CorridorID: id, AlignmentID: alignment})
}
func TestStarterRevenueForecastAndCapacity(t *testing.T) {
	g := openTest(t, nil)
	before := g.Snapshot()
	q := gameapi.ForecastRequest{CorridorID: "tokyo-nagoya", Frequency: 6, Fare: 9000}
	f, e := g.Forecast(q)
	if e != nil || f.Revenue <= 0 || f.Contribution <= 0 || f.Passengers > f.Capacity || f.FleetNeeded != 2 {
		t.Fatal(f, e)
	}
	q.Fare = 30000
	high, _ := g.Forecast(q)
	if high.Passengers >= f.Passengers {
		t.Fatal("fare has no demand effect")
	}
	q.Fare = 9000
	q.Frequency = 1
	low, _ := g.Forecast(q)
	if low.Passengers > 24800 {
		t.Fatal("seat capacity exceeded")
	}
	if g.Snapshot().Checkpoint != before.Checkpoint {
		t.Fatal("forecast mutated world")
	}
	after := act(t, g, gameapi.Command{Kind: gameapi.AdvanceMonth})
	r := after.Reports[0]
	if r.Revenue != f.Revenue || r.Passengers != f.Passengers || r.ClosingCash != before.Player.Cash+r.Revenue-r.Costs || r.ClosingCash <= before.Player.Cash {
		t.Fatal("tickets not collected or cash did not reconcile", r)
	}
}
func TestConstructionOpeningAndDisconnectedOperation(t *testing.T) {
	s := &memoryStore{}
	g := openTest(t, s)
	rejected(t, g, gameapi.Command{Kind: gameapi.BuildLine, CorridorID: "hokkaido-extension", AlignmentID: "mountain"}, "insufficient_cash")
	financedBuild(t, g, "hokkaido-extension", "mountain")
	rejected(t, g, gameapi.Command{Kind: gameapi.BuildLine, CorridorID: "hokkaido-extension", AlignmentID: "valley"}, "already_built")
	rejected(t, g, gameapi.Command{Kind: gameapi.LeaseTrain, CorridorID: "hokkaido-extension"}, "line_not_open")
	for i := 0; i < 4; i++ {
		act(t, g, gameapi.Command{Kind: gameapi.AdvanceYear})
	}
	f := g.Snapshot()
	if f.MonthsResolved != 48 || f.Projects[0].Ready != 48 {
		t.Fatal("opening month drifted")
	}
	for _, r := range f.Reports {
		for _, run := range r.Services {
			if run.CorridorID == "hokkaido-extension" {
				t.Fatal("new line operated before completion")
			}
		}
	}
	q := gameapi.Command{Kind: gameapi.SetService, CorridorID: "hokkaido-extension", Frequency: 4, Fare: 11000}
	rejected(t, g, q, "fleet_shortage") // Japanese mainland fleet cannot teleport.
	act(t, g, gameapi.Command{Kind: gameapi.LeaseTrain, CorridorID: "hokkaido-extension"})
	act(t, g, q)
	g = openTest(t, s)
	after := act(t, g, gameapi.Command{Kind: gameapi.AdvanceMonth})
	r := after.Reports[48]
	found := false
	for _, run := range r.Services {
		if run.CorridorID == "hokkaido-extension" {
			found = run.Passengers > 0 && run.Revenue > 0
		}
	}
	if !found || r.Maintenance <= 0 {
		t.Fatal("opened independent line did not operate and maintain", r)
	}
	capital := after.Reports[0]
	if capital.Construction <= 0 || capital.Grants*10 != capital.Construction*9 || capital.Borrowed != 250_000_000_000 {
		t.Fatal("grant/funding ledger missing", capital)
	}
	for _, r := range after.Reports {
		if r.ClosingCash != r.OpeningCash+r.Revenue-r.Costs+r.Grants+r.Borrowed-r.Repaid-r.Construction {
			t.Fatal("cash conservation violated")
		}
	}
}
func TestFleetPlanLifecycleAndInterest(t *testing.T) {
	g := openTest(t, nil)
	q := gameapi.Command{Kind: gameapi.SetService, CorridorID: "tokyo-nagoya", Frequency: 16, Fare: 9000}
	rejected(t, g, q, "fleet_shortage")
	for i := 0; i < 2; i++ {
		act(t, g, gameapi.Command{Kind: gameapi.LeaseTrain, CorridorID: "tokyo-nagoya"})
	}
	act(t, g, q)
	rejected(t, g, gameapi.Command{Kind: gameapi.ReturnTrain, CorridorID: "tokyo-nagoya"}, "fleet_in_use")
	act(t, g, gameapi.Command{Kind: gameapi.TakeLoan, Amount: 50_000_000_000})
	act(t, g, gameapi.Command{Kind: gameapi.AdvanceMonth})
	r := g.Snapshot().Reports[0]
	if r.Interest != 83_333_334 || r.LeaseCosts != 160_000_000 {
		t.Fatal("interest/lease units or rounding wrong", r.Interest, r.LeaseCosts)
	}
	q.Frequency = 0
	act(t, g, q)
	act(t, g, gameapi.Command{Kind: gameapi.ReturnTrain, CorridorID: "tokyo-nagoya"})
	after := act(t, g, gameapi.Command{Kind: gameapi.AdvanceMonth})
	r = after.Reports[1]
	if r.Revenue != 0 || r.LeaseCosts != 120_000_000 {
		t.Fatal("suspension or idle fleet costs wrong", r)
	}
}
func TestBuildSaveFailureAndFinancialFrameIsolation(t *testing.T) {
	s := &memoryStore{}
	g := openTest(t, s)
	for i := 0; i < 4; i++ {
		act(t, g, gameapi.Command{Kind: gameapi.TakeLoan, Amount: 50_000_000_000})
	}
	before := g.Snapshot()
	raw := string(s.b)
	s.fail = true
	rejected(t, g, gameapi.Command{Kind: gameapi.BuildLine, CorridorID: "tsuruga-osaka", AlignmentID: "valley"}, "storage_failed")
	if string(s.b) != raw || g.Snapshot().Checkpoint != before.Checkpoint {
		t.Fatal("failed build spent cash")
	}
	s.fail = false
	f := act(t, g, gameapi.Command{Kind: gameapi.BuildLine, CorridorID: "tsuruga-osaka", AlignmentID: "valley"})
	checkpoint := f.Checkpoint
	f.Projects[0].Ready = 0
	f.Services[0].Frequency = 0
	f.Fleet[0].CorridorID = "bad"
	f.Corridors[0].Alignments[0].Path[0].Lat = 0
	if g.Snapshot().Checkpoint != checkpoint || g.Snapshot().Corridors[0].Alignments[0].Path[0].Lat == 0 {
		t.Fatal("business frame leaked state")
	}
}
func TestRestoreRejectsFinancialAndOwnershipCorruption(t *testing.T) {
	g := openTest(t, nil)
	financedBuild(t, g, "tsuruga-osaka", "valley")
	act(t, g, gameapi.Command{Kind: gameapi.AdvanceMonth})
	raw, _ := g.Export()
	var original envelope
	_ = json.Unmarshal(raw, &original)
	for name, mutate := range map[string]func(*payload){"debt": func(p *payload) { p.Debt = 0 }, "owner": func(p *payload) { p.Projects[0].Owner = "rival" }, "ready": func(p *payload) { p.Projects[0].Ready = 0 }, "duplicate-fleet": func(p *payload) { p.Fleet[1].ID = p.Fleet[0].ID }, "phantom-riders": func(p *payload) { p.Reports[0].Services[0].Passengers++ }} {
		t.Run(name, func(t *testing.T) {
			var p payload
			_ = json.Unmarshal(original.Payload, &p)
			mutate(&p)
			env := original
			env.Payload, _ = json.Marshal(p)
			env.SHA = hash(env.Payload)
			b, _ := json.Marshal(env)
			if _, e := Open(&memoryStore{b: b}, nil); e == nil {
				t.Fatal("corrupt financial state accepted")
			}
		})
	}
}

func TestEndOfCampaignCannotSpendOrBorrow(t *testing.T) {
	g := openTest(t, nil)
	for i := 0; i < 30; i++ {
		act(t, g, gameapi.Command{Kind: gameapi.AdvanceYear})
	}
	rejected(t, g, gameapi.Command{Kind: gameapi.TakeLoan, Amount: 50_000_000_000}, "campaign_complete")
	rejected(t, g, gameapi.Command{Kind: gameapi.LeaseTrain, CorridorID: "tokyo-nagoya"}, "campaign_complete")
}

func TestOneLoanFundsExactProjectShortfall(t *testing.T) {
	s := &memoryStore{}
	g := openTest(t, s)
	before := g.Snapshot()
	var net gameapi.Money
	for _, c := range before.Corridors {
		if c.ID == "tsuruga-osaka" {
			for _, a := range c.Alignments {
				if a.ID == "valley" {
					net = a.Cost - a.Grant
				}
			}
		}
	}
	gap := net - before.Player.Cash
	f := act(t, g, gameapi.Command{Kind: gameapi.TakeLoan, Amount: gap})
	if f.Revision != 1 || f.Debt != gap || f.Player.Cash != net {
		t.Fatal("single loan did not cover the exact shortfall", f.Debt, gap)
	}
	f = act(t, g, gameapi.Command{Kind: gameapi.BuildLine, CorridorID: "tsuruga-osaka", AlignmentID: "valley"})
	if f.Player.Cash != 0 || f.Debt != gap {
		t.Fatal("funding was overdrawn or rounded")
	}
	loaded := openTest(t, s)
	if loaded.Snapshot().Checkpoint != f.Checkpoint {
		t.Fatal("custom loan did not survive restore")
	}
}
func TestCustomLoanCentsLimitAndFailureAtomicity(t *testing.T) {
	s := &memoryStore{}
	g := openTest(t, s)
	for _, amount := range []gameapi.Money{0, -1} {
		rejected(t, g, gameapi.Command{Kind: gameapi.TakeLoan, Amount: amount}, "invalid_loan_amount")
	}
	for _, amount := range []gameapi.Money{500_000_000_001, 1<<63 - 1} {
		rejected(t, g, gameapi.Command{Kind: gameapi.TakeLoan, Amount: amount}, "loan_limit")
	}
	s.fail = true
	rejected(t, g, gameapi.Command{Kind: gameapi.TakeLoan, Amount: 12345}, "storage_failed")
	s.fail = false
	act(t, g, gameapi.Command{Kind: gameapi.TakeLoan, Amount: 12345})
	f := act(t, g, gameapi.Command{Kind: gameapi.AdvanceMonth})
	if f.Debt != 12345 || f.Reports[0].Borrowed != 12345 || f.Reports[0].Interest != 21 {
		t.Fatal("custom cents or interest rounding incorrect")
	}
	f = act(t, g, gameapi.Command{Kind: gameapi.TakeLoan, Amount: 500_000_000_000 - 12345})
	if f.Debt != f.DebtLimit {
		t.Fatal("could not use exact remaining credit")
	}
	rejected(t, g, gameapi.Command{Kind: gameapi.TakeLoan, Amount: 1}, "loan_limit")
}
