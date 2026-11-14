package application

import (
	"encoding/json"
	"errors"
	"testing"
	"wwhsrt/pkg/gameapi"
)

type memoryStore struct {
	b    []byte
	fail bool
}

func (s *memoryStore) Load() ([]byte, error) { return append([]byte{}, s.b...), nil }
func (s *memoryStore) Commit(b []byte) error {
	if s.fail {
		return errors.New("disk full")
	}
	s.b = append([]byte{}, b...)
	return nil
}
func openTest(t *testing.T, s Store) *Game {
	t.Helper()
	g, e := openMode(s, nil, false)
	if e != nil {
		t.Fatal(e)
	}
	return g
}
func applyTest(t *testing.T, g *Game, k, name string) gameapi.Frame {
	t.Helper()
	f, e := g.Apply(gameapi.Command{Kind: k, Name: name, Revision: g.Snapshot().Revision})
	if e != nil {
		t.Fatal(e)
	}
	return f
}
func TestContinuationAndCalendar(t *testing.T) {
	s := &memoryStore{}
	g := openTest(t, s)
	control := openTest(t, nil)
	for _, p := range []*Game{g, control} {
		applyTest(t, p, gameapi.RenameCompany, "晨曦铁路")
		for i := 0; i < 13; i++ {
			applyTest(t, p, gameapi.AdvanceMonth, "")
		}
	}
	g = openTest(t, s)
	for _, p := range []*Game{g, control} {
		applyTest(t, p, gameapi.AdvanceMonth, "")
	}
	f := g.Snapshot()
	if f.Checkpoint != control.Snapshot().Checkpoint {
		t.Fatal("save/resume diverged")
	}
	if f.Month != (gameapi.Month{Year: 2030, Month: 3}) || f.Player.Name != "晨曦铁路" || len(f.Reports) != 14 {
		t.Fatalf("wrong calendar/state: %+v", f)
	}
	r := f.Reports[13]
	if r.Revenue <= 0 || r.Passengers <= 0 || r.ClosingCash != r.OpeningCash+r.Revenue-r.Costs {
		t.Fatal("operating report did not reconcile")
	}
}
func TestPersistenceAndCommandFailureAreAtomic(t *testing.T) {
	s := &memoryStore{}
	g := openTest(t, s)
	before := g.Snapshot()
	bytesBefore := string(s.b)
	for _, c := range []gameapi.Command{{Kind: gameapi.AdvanceMonth, Revision: 1}, {Kind: "spend_money"}, {Kind: gameapi.RenameCompany, Name: "\n"}, {Kind: gameapi.RenameCompany, Name: "abc\u202edef"}} {
		if _, e := g.Apply(c); e == nil {
			t.Fatal("invalid command accepted")
		}
	}
	s.fail = true
	if _, e := g.Apply(gameapi.Command{Kind: gameapi.AdvanceMonth}); e == nil {
		t.Fatal("failed save acknowledged")
	}
	if g.Snapshot().Checkpoint != before.Checkpoint || string(s.b) != bytesBefore {
		t.Fatal("failed command changed live/durable state")
	}
	s.fail = false
	applyTest(t, g, gameapi.AdvanceMonth, "")
	if g.Snapshot().MonthsResolved != 1 {
		t.Fatal("retry advanced twice")
	}
}
func TestFramesCannotMutateWorld(t *testing.T) {
	g := openTest(t, nil)
	applyTest(t, g, gameapi.AdvanceMonth, "")
	f := g.Snapshot()
	hash := f.Checkpoint
	f.Cities[0].Name = "hacked"
	f.Reports[0].ClosingCash = 0
	f.Player.Name = "hacked"
	if g.Snapshot().Checkpoint != hash || g.Snapshot().Cities[0].Name == "hacked" {
		t.Fatal("projection aliases live state")
	}
}
func TestRestoreRejectsMalformedAndIncompatibleSnapshots(t *testing.T) {
	g := openTest(t, nil)
	applyTest(t, g, gameapi.AdvanceMonth, "")
	b, _ := g.Export()
	var base envelope
	_ = json.Unmarshal(b, &base)
	check := func(name string, raw []byte) {
		t.Helper()
		t.Run(name, func(t *testing.T) {
			s := &memoryStore{b: raw}
			if _, e := Open(s, nil); e == nil {
				t.Fatal("bad save accepted")
			}
			if string(s.b) != string(raw) {
				t.Fatal("bad save overwritten")
			}
		})
	}
	check("checksum", append(append([]byte{}, b[:len(b)-2]...), '}'))
	check("duplicate", []byte(`{"format":"a","format":"b"}`))
	check("trailing", append(b, []byte(` {}`)...))
	for name, mutate := range map[string]func(map[string]any){
		"missing-revision":   func(p map[string]any) { delete(p, "revision") },
		"null-months":        func(p map[string]any) { p["months"] = nil },
		"other-scenario":     func(p map[string]any) { p["scenario_sha256"] = "other" },
		"other-rules":        func(p map[string]any) { p["rules"] = "other" },
		"bad-ledger":         func(p map[string]any) { p["reports"].([]any)[0].(map[string]any)["closing_cash"] = 0 },
		"unknown-field":      func(p map[string]any) { p["surprise"] = 1 },
		"missing-zero-value": func(p map[string]any) { delete(p["reports"].([]any)[0].(map[string]any), "revenue") },
		"null-zero-value":    func(p map[string]any) { p["reports"].([]any)[0].(map[string]any)["revenue"] = nil },
	} {
		var p map[string]any
		_ = json.Unmarshal(base.Payload, &p)
		mutate(p)
		payload, _ := json.Marshal(p)
		env := base
		env.Payload = payload
		env.SHA = hash(payload)
		raw, _ := json.Marshal(env)
		check(name, raw)
	}
	future := base
	future.Version = 4
	raw, _ := json.Marshal(future)
	check("future", raw)
}
func TestCampaignBoundary(t *testing.T) {
	g := openTest(t, nil)
	for i := 0; i < 360; i++ {
		applyTest(t, g, gameapi.AdvanceMonth, "")
	}
	before := g.Snapshot()
	if _, e := g.Apply(gameapi.Command{Kind: gameapi.AdvanceMonth, Revision: before.Revision}); e == nil {
		t.Fatal("campaign exceeded")
	}
	if g.Snapshot().Checkpoint != before.Checkpoint {
		t.Fatal("rejection changed state")
	}
}

func TestDevelopmentRevisionBoundPreservesRestorableState(t *testing.T) {
	g := openTest(t, nil)
	raw, _ := g.Export()
	var env envelope
	_ = json.Unmarshal(raw, &env)
	var p payload
	_ = json.Unmarshal(env.Payload, &p)
	p.Revision = 1_000_000
	env.Payload, _ = json.Marshal(p)
	env.SHA = hash(env.Payload)
	raw, _ = json.Marshal(env)
	g = openTest(t, &memoryStore{b: raw})
	before := g.Snapshot().Checkpoint
	if _, e := g.Apply(gameapi.Command{Kind: gameapi.RenameCompany, Revision: 1_000_000, Name: "Next"}); e == nil || e.Error() != "action_limit" {
		t.Fatal("revision boundary not enforced", e)
	}
	if g.Snapshot().Checkpoint != before {
		t.Fatal("rejection mutated state")
	}
}
func FuzzRestoreNeverPanics(f *testing.F) {
	g, _ := Open(nil, nil)
	b, _ := g.Export()
	f.Add(b)
	f.Add([]byte(`{"payload":null}`))
	f.Fuzz(func(t *testing.T, b []byte) { _, _ = decode(b) })
}
