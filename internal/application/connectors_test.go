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

func TestConnectorsV9MigrationPreservesPlansAssetsAndHistory(t *testing.T) {
	for _, name := range []string{"connectors-v9.json", "connectors-cutover-v9.json"} {
		t.Run(name, func(t *testing.T) {
			raw, e := os.ReadFile("testdata/" + name)
			if e != nil {
				t.Fatal(e)
			}
			var env envelope
			var old regionalPayload
			if json.Unmarshal(raw, &env) != nil || json.Unmarshal(env.Payload, &old) != nil || env.Version != 9 ||
				old.PublicSHA != connectorPriorSHA {
				t.Fatal("not a genuine v9 fixture")
			}
			store := &memoryStore{b: raw}
			g := openPublic(t, store)
			f := g.Snapshot()
			if !bytes.Equal(raw, store.b) || old.Company != f.Player || !reflect.DeepEqual(old.Reports, f.Reports) ||
				!reflect.DeepEqual(old.Projects, f.Projects) ||
				!reflect.DeepEqual(old.Services, f.Services) ||
				!reflect.DeepEqual(old.Fleet, f.Fleet) ||
				old.Debt != f.Debt ||
				old.Pending != f.Pending ||
				g.w.ConnectorSince() != old.Months {
				t.Fatal("read upgrade changed existing campaign")
			}
			store.fail = true
			if _, e = g.Apply(
				gameapi.Command{Kind: gameapi.AdvanceMonth, Revision: f.Revision},
			); e == nil || g.Snapshot().Checkpoint != f.Checkpoint ||
				!bytes.Equal(raw, store.b) {
				t.Fatal("failed upgrade lost save/state")
			}
			store.fail = false
			f = publicCommand(t, g, gameapi.Command{Kind: gameapi.AdvanceMonth})
			if !reflect.DeepEqual(f.Reports[:old.Months], old.Reports) ||
				openPublic(t, store).Snapshot().Checkpoint != f.Checkpoint {
				t.Fatal("nested historical epochs do not reload")
			}
			var next serbiaPayload
			json.Unmarshal(store.b, &env)
			json.Unmarshal(env.Payload, &next)
			if env.Version != 14 || next.Rules != "network-v14" || next.ConnectorSince != old.Months ||
				next.ConnectorPriorSHA != connectorPriorSHA ||
				next.PublicSHA != hash(assets.PublicNetwork) ||
				next.RegionalSince != old.RegionalSince {
				t.Fatal("wrong connector save metadata")
			}
			for _, mutate := range []func(*serbiaPayload){
				func(p *serbiaPayload) { p.ConnectorPriorSHA = "unknown" },
				func(p *serbiaPayload) { p.ConnectorSince = -1 },
				func(p *serbiaPayload) { p.ConnectorSince = p.Months + 1 },
				func(p *serbiaPayload) { p.ConnectorSince = 0; p.RegionalSince = p.Months },
			} {
				bad := next
				mutate(&bad)
				env.Payload, _ = json.Marshal(bad)
				env.SHA = hash(env.Payload)
				b, _ := json.Marshal(env)
				if _, e = Open(&memoryStore{b: b}, nil); e == nil {
					t.Fatal("invalid connector metadata accepted")
				}
			}
			publicCommand(t, g, gameapi.Command{Kind: gameapi.NewGame})
			if g.w.ConnectorSince() != 0 || g.w.RegionalSince() != 0 ||
				openPublic(t, store).Snapshot().MonthsResolved != 0 {
				t.Fatal("new game retains an old epoch")
			}
		})
	}
}

func TestConnectorsPublicRepairsOfferOrderedServicesWithoutConstruction(t *testing.T) {
	for _, id := range []string{
		"network_italian_hsr_all_stops",
		"network_catalan_hsr_all_stops",
		"network_tokaido_junction_all_stops",
	} {
		t.Run(id, func(t *testing.T) {
			store := &memoryStore{}
			g := openPublic(t, store)
			fc, e := g.Forecast(gameapi.ForecastRequest{CorridorID: id, Frequency: 1, Fare: 4500})
			if e != nil || !fc.Open {
				t.Fatal("existing railway requires construction", e)
			}
			for i := 0; i < fc.AdditionalLeases; i++ {
				publicCommand(t, g, gameapi.Command{Kind: gameapi.LeaseTrain, CorridorID: id})
			}
			publicCommand(t, g, gameapi.Command{Kind: gameapi.SetService, CorridorID: id, Frequency: 1, Fare: 4500})
			f := publicCommand(t, g, gameapi.Command{Kind: gameapi.AdvanceMonth})
			if len(f.Projects) != 0 {
				t.Fatal("public repairs became private projects")
			}
			var route gameapi.Route
			for _, r := range f.Routes {
				if r.ID == id {
					route = r
				}
			}
			if !route.Open || len(route.Cities) < 3 {
				t.Fatal("missing repaired service")
			}
			carried := false
			for _, j := range f.Network.Journeys {
				if (j.From == route.Cities[0] && j.To == route.Cities[len(
					route.Cities,
				)-1] || j.To == route.Cities[0] && j.From == route.Cities[len(route.Cities)-1]) &&
					j.Passengers > 0 {
					carried = true
				}
			}
			if !carried || openPublic(t, store).Snapshot().Checkpoint != f.Checkpoint {
				t.Fatal("repaired end-to-end journey/reload failed")
			}
		})
	}
}

func TestConnectorsBuildOpenAndCarryThroughPassengers(t *testing.T) {
	cases := []struct {
		project, service string
		from, to         gameapi.CityID
	}{
		{"launch_72_tsuruga__maibara", "connector_kanazawa__osaka_via_maibara", "kanazawa", "osaka"},
		{"launch_73_valencia__camp_de_tarragona", "connector_valencia__barcelona", "valencia", "barcelona"},
		{"launch_74_porto__vigo", "connector_lisbon__pontevedra", "lisbon", "stop_node_453651815"},
	}
	for _, c := range cases {
		t.Run(c.project, func(t *testing.T) {
			store := &memoryStore{}
			g := openPublic(t, store)
			f := g.Snapshot()
			var co gameapi.Corridor
			for _, row := range f.Corridors {
				if row.ID == c.project {
					co = row
				}
			}
			if co.ID == "" || co.Existing {
				t.Fatal("missing prospect")
			}
			a := co.Alignments[0]
			fc, e := g.Forecast(gameapi.ForecastRequest{CorridorID: c.service, Frequency: 1, Fare: 4500})
			if e == nil || fc.Open {
				t.Fatal("unbuilt connector is usable")
			}
			gap := max(gameapi.Money(0), a.Cost-a.Grant-f.Player.Cash)
			if gap > 0 {
				publicCommand(t, g, gameapi.Command{Kind: gameapi.TakeLoan, Amount: gap})
			}
			f = publicCommand(t, g, gameapi.Command{Kind: gameapi.BuildLine, CorridorID: co.ID, AlignmentID: a.ID})
			if f.Projects[0].Ready != a.Months {
				t.Fatal("wrong opening date")
			}
			for f.MonthsResolved+12 < a.Months {
				f = publicCommand(t, g, gameapi.Command{Kind: gameapi.AdvanceYear})
			}
			for f.MonthsResolved < a.Months-1 {
				f = publicCommand(t, g, gameapi.Command{Kind: gameapi.AdvanceMonth})
			}
			if _, e = g.Forecast(gameapi.ForecastRequest{CorridorID: c.service, Frequency: 1, Fare: 4500}); e == nil {
				t.Fatal("connector opens a month early")
			}
			publicCommand(t, g, gameapi.Command{Kind: gameapi.AdvanceMonth})
			fc, e = g.Forecast(gameapi.ForecastRequest{CorridorID: c.service, Frequency: 1, Fare: 4500})
			if e != nil || !fc.Open {
				t.Fatal("completed connector through route is unavailable", e)
			}
			for i := 0; i < fc.AdditionalLeases; i++ {
				publicCommand(t, g, gameapi.Command{Kind: gameapi.LeaseTrain, CorridorID: c.service})
			}
			publicCommand(
				t,
				g,
				gameapi.Command{Kind: gameapi.SetService, CorridorID: c.service, Frequency: 1, Fare: 4500},
			)
			f = publicCommand(t, g, gameapi.Command{Kind: gameapi.AdvanceMonth})
			carried := false
			for _, j := range f.Network.Journeys {
				if (j.From == c.from && j.To == c.to || j.To == c.from && j.From == c.to) && j.Passengers > 0 {
					carried = true
				}
			}
			if !carried || openPublic(t, store).Snapshot().Checkpoint != f.Checkpoint {
				t.Fatal("new through route carries no end-to-end passengers or cannot reload")
			}
			r := f.Reports[len(f.Reports)-1]
			if r.ClosingCash != r.OpeningCash+r.Revenue-r.Costs+r.Grants+r.Borrowed-r.Repaid-r.Construction {
				t.Fatal("ledger does not reconcile")
			}
		})
	}
}
