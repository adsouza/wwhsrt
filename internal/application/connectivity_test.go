package application

import (
	"bytes"
	"compress/gzip"
	"encoding/json"
	"io"
	"os"
	"reflect"
	"testing"
	"wwhsrt/assets"
	"wwhsrt/pkg/gameapi"
)

func TestConnectivityV6MigrationPreservesAccountsPlansFleetAndAtomicity(t *testing.T) {
	raw, e := os.ReadFile("testdata/connectivity-v6.json")
	if e != nil {
		t.Fatal(e)
	}
	var env envelope
	var prior topologyPayload
	if json.Unmarshal(raw, &env) != nil || json.Unmarshal(env.Payload, &prior) != nil || env.Version != 6 ||
		len(prior.Fleet) == 0 ||
		len(prior.Services) < 2 {
		t.Fatal("not a populated genuine v6 fixture")
	}
	store := &memoryStore{b: raw}
	g := openPublic(t, store)
	f := g.Snapshot()
	if !bytes.Equal(raw, store.b) || !reflect.DeepEqual(f.Reports, prior.Reports) ||
		!reflect.DeepEqual(f.Fleet, prior.Fleet) ||
		!reflect.DeepEqual(f.Services, prior.Services) ||
		f.Player != prior.Company ||
		g.w.ConnectivitySince() != prior.Months {
		t.Fatal("opening changed historical accounts or accepted plans")
	}
	store.fail = true
	if _, e = g.Apply(
		gameapi.Command{Kind: gameapi.AdvanceMonth, Revision: f.Revision},
	); e == nil || g.Snapshot().Checkpoint != f.Checkpoint ||
		!bytes.Equal(raw, store.b) {
		t.Fatal("failed migration changed state")
	}
	store.fail = false
	f = publicCommand(t, g, gameapi.Command{Kind: gameapi.AdvanceMonth})
	if !reflect.DeepEqual(f.Reports[:prior.Months], prior.Reports) {
		t.Fatal("past reports rewritten")
	}
	var next viennaPayload
	json.Unmarshal(store.b, &env)
	json.Unmarshal(env.Payload, &next)
	if env.Version != 18 || next.Rules != "network-v18" || next.ConnectivitySince != prior.Months ||
		next.PreviousRepairSHA != previousRepairSHA ||
		next.PublicSHA != hash(assets.PublicNetwork) {
		t.Fatal("new topology identity/cutover missing")
	}
	restored := openPublic(t, store)
	if restored.Snapshot().Checkpoint != f.Checkpoint {
		t.Fatal("mixed history cannot restore")
	}
	a := publicCommand(t, g, gameapi.Command{Kind: gameapi.AdvanceMonth})
	b := publicCommand(t, restored, gameapi.Command{Kind: gameapi.AdvanceMonth})
	if a.Checkpoint != b.Checkpoint || !reflect.DeepEqual(a.Reports, b.Reports) {
		t.Fatal("reloaded continuation diverged")
	}
	for _, change := range []func(*viennaPayload){
		func(p *viennaPayload) { p.PreviousRepairSHA = "unknown" },
		func(p *viennaPayload) { p.ConnectivitySince = -1 },
		func(p *viennaPayload) { p.ConnectivitySince = p.Months + 1 },
		func(p *viennaPayload) { p.ConnectivitySince = 0 },
		func(p *viennaPayload) { p.PublicSHA = previousRepairSHA },
	} {
		bad := next
		change(&bad)
		env.Payload, _ = json.Marshal(bad)
		env.SHA = hash(env.Payload)
		data, _ := json.Marshal(env)
		if _, e = Open(&memoryStore{b: data}, nil); e == nil {
			t.Fatal("invalid cutover/hash accepted")
		}
	}
}

func TestConnectivityRetainsV6ServiceAndFleetRequirements(t *testing.T) {
	r, e := gzip.NewReader(bytes.NewReader(assets.PublicNetworkV2Compressed))
	if e != nil {
		t.Fatal(e)
	}
	defer r.Close()
	raw, e := io.ReadAll(r)
	if e != nil || hash(raw) != previousRepairSHA {
		t.Fatal("frozen v6 catalog changed")
	}
	var old importedNetwork
	json.Unmarshal(raw, &old)
	currentCatalog, _ := publicData()
	corridors := map[string]gameapi.Corridor{}
	routes := map[string]gameapi.Route{}
	for _, co := range currentCatalog.Corridors {
		corridors[co.ID] = co
	}
	for _, route := range currentCatalog.Network.Routes {
		routes[route.ID] = route
	}
	for _, co := range old.Corridors {
		now, ok := corridors[co.ID]
		if !ok || now.FromCity != co.FromCity || now.ToCity != co.ToCity || len(now.Alignments) != len(co.Alignments) {
			t.Fatal("accepted asset identity removed", co.ID)
		}
		for i, a := range co.Alignments {
			b := now.Alignments[i]
			// The v9 Ulsan gateway correction shortens these services; it must
			// never require more trains from an existing company.
			correctedUlsan := (co.FromCity == "ulsan" || co.ToCity == "ulsan") && b.KM <= a.KM && b.Minutes <= a.Minutes
			correctedMontabaur := co.ToCity == "stop_node_2453992382" && b.Minutes <= a.Minutes
			if a.ID != b.ID || (!correctedUlsan && !correctedMontabaur && !(co.ID == "public_hamburg__stop_node_2146217850" && b.Minutes <= a.Minutes) && (a.KM != b.KM || a.Minutes != b.Minutes)) {
				t.Fatal("accepted fleet timing changed", co.ID, a.ID)
			}
		}
	}
	for _, r := range old.Network.Routes {
		n, ok := routes[r.ID]
		if !ok || !reflect.DeepEqual(n.Cities, r.Cities) || !reflect.DeepEqual(n.Sections, r.Sections) ||
			!reflect.DeepEqual(n.Stops, r.Stops) {
			t.Fatal("accepted service changed", r.ID)
		}
	}
}

func TestConnectivityCompleteServicesCarryEndAndIntermediateRiders(t *testing.T) {
	for _, id := range []string{
		"network_nec_all_stops",
		"network_uzbek_main_all_stops",
		"network_uzbek_branch_all_stops",
		"public_kenitra__tangier",
		"route_frankfurt_am_main__rhine_ruhr_hsr",
		"public_ankara__istanbul",
		"route_moscow__saint_petersburg",
		"public_lisbon__porto",
		"munich__vienna_via_tullnerfeld",
		"route_beijing__kunming",
	} {
		t.Run(id, func(t *testing.T) {
			store := &memoryStore{}
			g := openPublic(t, store)
			if id == "route_frankfurt_am_main__rhine_ruhr_hsr" {
				publicCommand(t, g, gameapi.Command{Kind: gameapi.TakeLoan, Amount: 3_000_000_000})
				for _, section := range []string{"upgrade_de_koln__stop_node_1880180223", "upgrade_de_rhine_ruhr__stop_node_1880180223"} {
					publicCommand(t, g, gameapi.Command{Kind: gameapi.BuildLine, CorridorID: section, AlignmentID: "launch"})
				}
				publicCommand(t, g, gameapi.Command{Kind: gameapi.AdvanceYear})
			}
			f := g.Snapshot()
			var route gameapi.Route
			for _, r := range f.Routes {
				if r.ID == id {
					route = r
				}
			}
			if !route.Open || len(route.Stops) < 2 {
				t.Fatal("missing complete open route", id)
			}
			// A low positive fare exercises reachability even on long-distance services
			// whose market share at a commercial fare may legitimately be small.
			req := gameapi.ForecastRequest{CorridorID: id, Frequency: 1, Fare: 1000}
			fc, e := g.Forecast(req)
			if e != nil {
				t.Fatal(e)
			}
			for i := 0; i < fc.AdditionalLeases; i++ {
				publicCommand(t, g, gameapi.Command{Kind: gameapi.LeaseTrain, CorridorID: id})
			}
			publicCommand(t, g, gameapi.Command{Kind: gameapi.SetService, CorridorID: id, Frequency: 1, Fare: req.Fare})
			f = publicCommand(t, g, gameapi.Command{Kind: gameapi.AdvanceMonth})
			r := f.Reports[len(f.Reports)-1]
			end, mid := false, false
			a, b := route.Stops[0], route.Stops[len(route.Stops)-1]
			for _, j := range f.Network.Journeys {
				if j.Passengers <= 0 {
					continue
				}
				if j.From == a && j.To == b || j.From == b && j.To == a {
					end = true
				}
				for _, stop := range route.Stops[1 : len(route.Stops)-1] {
					if j.From == stop || j.To == stop {
						mid = true
					}
				}
			}
			if !end || len(route.Stops) > 2 && !mid {
				t.Fatal("no end-to-end/intermediate riders", route.Stops, end, mid)
			}
			for _, s := range f.Network.Sections {
				if s.Delivered > s.Capacity {
					t.Fatal("capacity exceeded")
				}
			}
			if r.Revenue <= 0 ||
				r.ClosingCash != r.OpeningCash+r.Revenue-r.Costs+r.Grants+r.Borrowed-r.Repaid-r.Construction {
				t.Fatal("ledger does not reconcile")
			}
			if openPublic(t, store).Snapshot().Checkpoint != f.Checkpoint {
				t.Fatal("settled route cannot restore")
			}
		})
	}
}

func TestConnectivityMixedTopologyHistoryRemainsIndependent(t *testing.T) {
	raw, e := os.ReadFile("testdata/connectivity-mixed-v6.json")
	if e != nil {
		t.Fatal(e)
	}
	var env envelope
	var prior topologyPayload
	json.Unmarshal(raw, &env)
	json.Unmarshal(env.Payload, &prior)
	if env.Version != 6 || prior.Months != 3 || prior.TopologySince != 2 {
		t.Fatal("fixture does not span both source catalogs")
	}
	store := &memoryStore{b: raw}
	g := openPublic(t, store)
	if g.w.TopologySince() != 2 || g.w.ConnectivitySince() != 3 || !bytes.Equal(raw, store.b) ||
		!reflect.DeepEqual(g.Snapshot().Reports, prior.Reports) {
		t.Fatal("historical boundaries collapsed")
	}
	f := publicCommand(t, g, gameapi.Command{Kind: gameapi.AdvanceMonth})
	if !reflect.DeepEqual(f.Reports[:3], prior.Reports) || openPublic(t, store).Snapshot().Checkpoint != f.Checkpoint {
		t.Fatal("three-catalog history cannot round-trip")
	}
}
