package application

import (
	"bytes"
	"encoding/json"
	"os"
	"reflect"
	"slices"
	"testing"

	"wwhsrt/pkg/gameapi"
)

func TestRhineStartingServices(t *testing.T) {
	const dusseldorf = "stop_node_1880180223"
	store := &memoryStore{}
	g := openPublic(t, store)
	f := g.Snapshot()
	byID := map[string]gameapi.Route{}
	for _, r := range f.Routes {
		byID[r.ID] = r
	}
	for _, id := range []string{"route_brussels__stop_node_1880180223", "route_brussels__rhine_ruhr"} {
		if _, ok := byID[id]; ok {
			t.Fatalf("former French detour offered in new game: %s", id)
		}
	}
	routes := []struct {
		id    string
		stops []gameapi.CityID
	}{
		{"network_brussels__randstad", []gameapi.CityID{"brussels", "antwerp", "rotterdam", "schiphol", "randstad"}},
		{"network_dusseldorf__randstad", []gameapi.CityID{dusseldorf, "duisburg", "arnhem", "utrecht", "randstad"}},
		{"network_brussels__dusseldorf", []gameapi.CityID{"brussels", "liege", "aachen", "koln", dusseldorf}},
	}
	for _, tc := range routes {
		id := tc.id
		r := byID[id]
		if !r.Open || !slices.Equal(r.Stops, tc.stops) {
			t.Fatalf("%s: unexpected stops or closed line: %v", id, r.Stops)
		}
		fc, err := g.Forecast(gameapi.ForecastRequest{CorridorID: id, Frequency: 1, Fare: 4500})
		if err != nil {
			t.Fatal(err)
		}
		for i := 0; i < fc.AdditionalLeases; i++ {
			publicCommand(t, g, gameapi.Command{Kind: gameapi.LeaseTrain, CorridorID: id})
		}
		publicCommand(t, g, gameapi.Command{Kind: gameapi.SetService, CorridorID: id, Frequency: 1, Fare: 4500})
	}
	f = publicCommand(t, g, gameapi.Command{Kind: gameapi.AdvanceMonth})
	for _, tc := range routes {
		a, b := tc.stops[0], tc.stops[len(tc.stops)-1]
		if !slices.ContainsFunc(f.Network.Journeys, func(j gameapi.JourneyFlow) bool {
			return j.Passengers > 0 && ((j.From == a && j.To == b) || (j.To == a && j.From == b))
		}) {
			t.Fatalf("no end-to-end passengers on %s", tc.id)
		}
	}
	if len(f.Projects) != 0 || openPublic(t, store).Snapshot().Checkpoint != f.Checkpoint {
		t.Fatal("starting services required construction or failed to reload")
	}
}

func TestRhineMigrationPreservesOperatingDetour(t *testing.T) {
	raw, err := os.ReadFile("testdata/rhine-v10.json")
	if err != nil {
		t.Fatal(err)
	}
	var env envelope
	var old connectorPayload
	if json.Unmarshal(raw, &env) != nil || env.Version != 10 || json.Unmarshal(env.Payload, &old) != nil || old.PublicSHA != rhinePriorSHA {
		t.Fatal("invalid v10 fixture")
	}
	store := &memoryStore{b: raw}
	g := openPublic(t, store)
	f := g.Snapshot()
	if !bytes.Equal(raw, store.b) || g.w.RhineSince() != old.Months || f.Player != old.Company ||
		!reflect.DeepEqual(f.Reports, old.Reports) || !reflect.DeepEqual(f.Fleet, old.Fleet) ||
		!reflect.DeepEqual(f.Services, old.Services) {
		t.Fatal("migration changed old company")
	}
	if !slices.ContainsFunc(f.Routes, func(r gameapi.Route) bool {
		return r.ID == "route_brussels__stop_node_1880180223" && slices.Contains(r.Cities, "paris")
	}) {
		t.Fatal("operating detour lost")
	}
	if !slices.ContainsFunc(f.Routes, func(r gameapi.Route) bool {
		return r.ID == "network_brussels__dusseldorf" && r.Open && slices.Contains(r.Stops, "liege")
	}) {
		t.Fatal("corrected service unavailable from existing save")
	}
	f = publicCommand(t, g, gameapi.Command{Kind: gameapi.AdvanceMonth})
	if openPublic(t, store).Snapshot().Checkpoint != f.Checkpoint || !reflect.DeepEqual(f.Reports[:old.Months], old.Reports) {
		t.Fatal("upgraded campaign history failed to round-trip")
	}
}
