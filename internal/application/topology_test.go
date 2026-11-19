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

func TestRepairedConnectionsCarryPassengersAndReload(t *testing.T) {
	pairs := [][2]gameapi.CityID{
		{"sendai", "shin_hakodate_hokuto"},
		{"london", "paris"},
		{"paris", "lyon"},
		{"milan", "rome"},
		{"berlin", "munich"},
		{"new_york", "boston"},
		{"changchun", "shenyang"},
	}
	for _, pair := range pairs {
		t.Run(string(pair[0])+"_"+string(pair[1]), func(t *testing.T) {
			store := &memoryStore{}
			g := openPublic(t, store)
			f := g.Snapshot()
			var route gameapi.Route
			for _, r := range f.Routes {
				a, b := r.Cities[0], r.Cities[len(r.Cities)-1]
				if a == pair[0] && b == pair[1] || a == pair[1] && b == pair[0] {
					if route.ID == "" || r.Minutes < route.Minutes {
						route = r
					}
				}
			}
			if !route.Open {
				t.Fatal("missing operational route")
			}
			q := gameapi.ForecastRequest{CorridorID: route.ID, Frequency: 1, Fare: route.SuggestedFare}
			fc, e := g.Forecast(q)
			if e != nil {
				t.Fatal(e)
			}
			for i := 0; i < fc.FleetNeeded; i++ {
				publicCommand(t, g, gameapi.Command{Kind: gameapi.LeaseTrain, CorridorID: route.ID})
			}
			publicCommand(
				t,
				g,
				gameapi.Command{Kind: gameapi.SetService, CorridorID: route.ID, Frequency: 1, Fare: q.Fare},
			)
			f = publicCommand(t, g, gameapi.Command{Kind: gameapi.AdvanceMonth})
			carried := false
			for _, j := range f.Network.Journeys {
				if (j.From == pair[0] && j.To == pair[1] || j.From == pair[1] && j.To == pair[0]) && j.Passengers > 0 {
					carried = true
				}
			}
			if !carried {
				t.Fatal("connectivity produces no end-to-end riders", route.ID, fc)
			}
			for _, s := range f.Network.Sections {
				if s.Delivered > s.Capacity {
					t.Fatal("section overbooked")
				}
			}
			if openPublic(t, store).Snapshot().Checkpoint != f.Checkpoint {
				t.Fatal("repaired service cannot restore")
			}
		})
	}
}

func TestTopologyV5UpgradePreservesHistoricalEconomyAndFleet(t *testing.T) {
	raw, e := os.ReadFile("testdata/topology-v5.json")
	if e != nil {
		t.Fatal(e)
	}
	var env envelope
	var prior balancedPayload
	json.Unmarshal(raw, &env)
	json.Unmarshal(env.Payload, &prior)
	store := &memoryStore{b: raw}
	g := openPublic(t, store)
	f := g.Snapshot()
	if !bytes.Equal(raw, store.b) || !reflect.DeepEqual(f.Reports, prior.Reports) ||
		!reflect.DeepEqual(f.Fleet, prior.Fleet) ||
		!reflect.DeepEqual(f.Services, prior.Services) ||
		f.Player != prior.Company ||
		g.w.TopologySince() != 2 ||
		g.w.EconomySince() != 0 {
		t.Fatal("opening rewrote historical state")
	}
	checkpoint := f.Checkpoint
	store.fail = true
	if _, e := g.Apply(
		gameapi.Command{Kind: gameapi.AdvanceMonth, Revision: f.Revision},
	); e == nil || g.Snapshot().Checkpoint != checkpoint ||
		!bytes.Equal(raw, store.b) {
		t.Fatal("failed upgrade was not atomic")
	}
	store.fail = false
	f = publicCommand(t, g, gameapi.Command{Kind: gameapi.AdvanceMonth})
	if !reflect.DeepEqual(f.Reports[:2], prior.Reports) {
		t.Fatal("historical reports rewritten")
	}
	var next viennaPayload
	json.Unmarshal(store.b, &env)
	json.Unmarshal(env.Payload, &next)
	if env.Version != 18 || next.TopologySince != 2 || next.PriorPublicSHA != hash(assets.PublicNetworkV1) ||
		next.PublicSHA != hash(assets.PublicNetwork) ||
		next.Rules != "network-v18" {
		t.Fatal("topology identity/cutover not pinned")
	}
	if openPublic(t, store).Snapshot().Checkpoint != f.Checkpoint {
		t.Fatal("mixed topology history does not restore")
	}
	for _, change := range []func(*viennaPayload){
		func(p *viennaPayload) { p.PriorPublicSHA = "unknown" },
		func(p *viennaPayload) { p.TopologySince = 4 },
	} {
		bad := next
		change(&bad)
		env.Payload, _ = json.Marshal(bad)
		env.SHA = hash(env.Payload)
		data, _ := json.Marshal(env)
		if _, e := Open(&memoryStore{b: data}, nil); e == nil {
			t.Fatal("invalid topology metadata accepted")
		}
	}
}

func TestTopologyRetainsPublishedRouteIdentitiesAndFleetRequirements(t *testing.T) {
	var old importedNetwork
	json.Unmarshal(assets.PublicNetworkV1, &old)
	currentCatalog, _ := publicData()
	current := map[string]gameapi.Corridor{}
	for _, c := range currentCatalog.Corridors {
		current[c.ID] = c
	}
	for _, c := range old.Corridors {
		n, ok := current[c.ID]
		correctedUlsan := ok && len(n.Alignments) > 0 && (c.FromCity == "ulsan" || c.ToCity == "ulsan") &&
			n.Alignments[0].KM <= c.Alignments[0].KM &&
			n.Alignments[0].Minutes <= c.Alignments[0].Minutes
		correctedMontabaur := ok && len(n.Alignments) > 0 && c.ToCity == "stop_node_2453992382" && n.Alignments[0].Minutes <= c.Alignments[0].Minutes
		if !ok || n.FromCity != c.FromCity || n.ToCity != c.ToCity ||
			(!correctedUlsan && !correctedMontabaur && !(c.ID == "public_hamburg__stop_node_2146217850" && n.Alignments[0].Minutes <= c.Alignments[0].Minutes) && (n.Alignments[0].KM != c.Alignments[0].KM ||
				n.Alignments[0].Minutes != c.Alignments[0].Minutes)) {
			t.Fatal("published corridor identity/timing changed", c.ID)
		}
	}
	routes := map[string]gameapi.Route{}
	for _, r := range currentCatalog.Network.Routes {
		routes[r.ID] = r
	}
	for _, r := range old.Network.Routes {
		n, ok := routes[r.ID]
		if !ok || !reflect.DeepEqual(n.Cities, r.Cities) || !reflect.DeepEqual(n.Sections, r.Sections) ||
			!reflect.DeepEqual(n.Stops, r.Stops) {
			t.Fatal("accepted service pattern changed", r.ID)
		}
	}
}
