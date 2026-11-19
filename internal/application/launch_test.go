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

func TestLaunchV7UpgradeAndConventionalMoroccoWithdrawal(t *testing.T) {
	for _, fixture := range []string{"launch-v7.json", "morocco-v7.json"} {
		t.Run(fixture, func(t *testing.T) {
			raw, e := os.ReadFile("testdata/" + fixture)
			if e != nil {
				t.Fatal(e)
			}
			var env envelope
			var old connectivityPayload
			json.Unmarshal(raw, &env)
			json.Unmarshal(env.Payload, &old)
			if env.Version != 7 || old.PublicSHA != launchPriorSHA {
				t.Fatal("not a genuine v7 fixture")
			}
			store := &memoryStore{b: raw}
			g := openPublic(t, store)
			f := g.Snapshot()
			if !bytes.Equal(raw, store.b) || !reflect.DeepEqual(old.Reports, f.Reports) || old.Company != f.Player ||
				g.w.ConstructionSince() != old.Months {
				t.Fatal("opening changed historical accounts")
			}
			if fixture == "launch-v7.json" &&
				(!reflect.DeepEqual(old.Services, f.Services) || !reflect.DeepEqual(old.Fleet, f.Fleet)) {
				t.Fatal("unrelated fleet/plan changed")
			}
			if fixture == "morocco-v7.json" {
				for _, s := range f.Services {
					if s.CorridorID == "network_al_boraq_all_stops" || s.CorridorID == "public_casablanca__rabat" {
						t.Fatal("conventional service retained as HSR")
					}
				}
				for _, trip := range f.Day {
					if trip.ServiceID == "network_al_boraq_all_stops" || trip.ServiceID == "public_casablanca__rabat" {
						t.Fatal("retired conventional train remains on the HSR map")
					}
				}
				if len(f.Fleet) != 2 {
					t.Fatal("withdrawn trains not returned")
				}
			}
			before := f.Checkpoint
			store.fail = true
			if _, e = g.Apply(
				gameapi.Command{Kind: gameapi.AdvanceMonth, Revision: f.Revision},
			); e == nil || g.Snapshot().Checkpoint != before ||
				!bytes.Equal(store.b, raw) {
				t.Fatal("failed upgrade changed state")
			}
			store.fail = false
			f = publicCommand(t, g, gameapi.Command{Kind: gameapi.AdvanceMonth})
			if !reflect.DeepEqual(f.Reports[:old.Months], old.Reports) ||
				openPublic(t, store).Snapshot().Checkpoint != f.Checkpoint {
				t.Fatal("historical/new interval cannot reload")
			}
			var next viennaPayload
			json.Unmarshal(store.b, &env)
			json.Unmarshal(env.Payload, &next)
			if env.Version != 18 || next.LaunchPriorSHA != launchPriorSHA || next.ConstructionSince != old.Months ||
				next.PublicSHA != hash(assets.PublicNetwork) {
				t.Fatal("catalog identity not pinned")
			}
			for _, mutate := range []func(*viennaPayload){
				func(p *viennaPayload) { p.LaunchPriorSHA = "unknown" },
				func(p *viennaPayload) { p.ConstructionSince = -1 },
				func(p *viennaPayload) { p.ConstructionSince = p.Months + 1 },
			} {
				bad := next
				mutate(&bad)
				env.Payload, _ = json.Marshal(bad)
				env.SHA = hash(env.Payload)
				b, _ := json.Marshal(env)
				if _, e = Open(&memoryStore{b: b}, nil); e == nil {
					t.Fatal("invalid catalog metadata accepted")
				}
			}
		})
	}
}
func TestLaunchIndependentRegionalProjectsOpenCarryRevenueAndReload(t *testing.T) {
	for _, prefix := range []string{
		"launch_02_",
		"launch_17_",
		"launch_33_",
		"launch_39_",
		"launch_57_",
		"launch_71_",
		"hokkaido-extension",
	} {
		t.Run(prefix, func(t *testing.T) {
			store := &memoryStore{}
			g := openPublic(t, store)
			f := g.Snapshot()
			var co gameapi.Corridor
			for _, c := range f.Corridors {
				if len(c.ID) >= len(prefix) && c.ID[:len(prefix)] == prefix {
					co = c
				}
			}
			if co.ID == "" || co.Existing {
				t.Fatal("missing buildable project")
			}
			a := co.Alignments[0]
			need := max(gameapi.Money(0), a.Cost-a.Grant-f.Player.Cash)
			if need > f.DebtLimit {
				t.Fatal("regional opening requires unavailable finance", co.ID, need)
			}
			if need > 0 {
				publicCommand(t, g, gameapi.Command{Kind: gameapi.TakeLoan, Amount: need})
			}
			f = publicCommand(t, g, gameapi.Command{Kind: gameapi.BuildLine, CorridorID: co.ID, AlignmentID: a.ID})
			for _, r := range f.Routes {
				if r.ID == co.ID && r.Open {
					t.Fatal("project opens immediately")
				}
			}
			for f.MonthsResolved+12 <= a.Months {
				f = publicCommand(t, g, gameapi.Command{Kind: gameapi.AdvanceYear})
			}
			for f.MonthsResolved < a.Months {
				f = publicCommand(t, g, gameapi.Command{Kind: gameapi.AdvanceMonth})
			}
			serviceID := co.ID
			if co.ID == "hokkaido-extension" {
				serviceID = "launch_route_sapporo__tokyo"
			}
			fc, e := g.Forecast(gameapi.ForecastRequest{CorridorID: serviceID, Frequency: 1, Fare: 1000})
			if e != nil || !fc.Open {
				t.Fatal("completed project not usable", e)
			}
			for i := 0; i < fc.AdditionalLeases; i++ {
				publicCommand(t, g, gameapi.Command{Kind: gameapi.LeaseTrain, CorridorID: serviceID})
			}
			publicCommand(
				t,
				g,
				gameapi.Command{Kind: gameapi.SetService, CorridorID: serviceID, Frequency: 1, Fare: 1000},
			)
			f = publicCommand(t, g, gameapi.Command{Kind: gameapi.AdvanceMonth})
			carried := false
			for _, j := range f.Network.Journeys {
				if (j.From == co.FromCity && j.To == co.ToCity || j.To == co.FromCity && j.From == co.ToCity) &&
					j.Passengers > 0 {
					carried = true
				}
			}
			if !carried {
				t.Fatal("opened project has no passengers", co.ID)
			}
			if co.ID == "hokkaido-extension" {
				end := false
				for _, j := range f.Network.Journeys {
					if (j.From == "tokyo" && j.To == "sapporo" || j.To == "tokyo" && j.From == "sapporo") &&
						j.Passengers > 0 {
						end = true
					}
				}
				if !end {
					t.Fatal("new extension has no Tokyo–Sapporo through riders")
				}
			}
			r := f.Reports[len(f.Reports)-1]
			if r.ClosingCash != r.OpeningCash+r.Revenue-r.Costs+r.Grants+r.Borrowed-r.Repaid-r.Construction {
				t.Fatal("ledger does not reconcile")
			}
			if openPublic(t, store).Snapshot().Checkpoint != f.Checkpoint {
				t.Fatal("constructed service cannot reload")
			}
		})
	}
}
func TestLaunchSeventyFourProspectsAndMoroccoHSRScope(t *testing.T) {
	f := openPublic(t, nil).Snapshot()
	n := 0
	for _, c := range f.Corridors {
		if !c.Existing && c.PartOf == "" {
			n++
		}
		if c.ID == "public_casablanca__rabat" || c.ID == "public_kenitra__rabat" {
			t.Fatal("conventional track offered as public HSR")
		}
	}
	if n != 101 {
		t.Fatal("missing construction opportunities", n)
	}
	for _, r := range f.Routes {
		if r.Open {
			for _, id := range r.Cities {
				if id == "casablanca" || id == "rabat" {
					t.Fatal("southern Morocco already open", r.ID)
				}
			}
		}
	}
}
