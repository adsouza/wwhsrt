package application

import (
	"encoding/json"
	"reflect"
	"slices"
	"testing"

	"wwhsrt/internal/domain"
	"wwhsrt/pkg/gameapi"
)

func TestSerbiaStartingService(t *testing.T) {
	const routeID = "network_belgrade__vrbas_nova"
	const vrbas = "stop_node_9767449803"
	store := &memoryStore{}
	g := openPublic(t, store)
	f := g.Snapshot()
	i := slices.IndexFunc(f.Routes, func(r gameapi.Route) bool { return r.ID == routeID })
	if i < 0 {
		t.Fatal("Belgrade–Vrbas Nova missing")
	}
	r := f.Routes[i]
	if !r.Open || r.InfrastructureKind != "hsr" || r.Minutes != 69 ||
		!slices.Equal(r.Stops, []gameapi.CityID{"belgrade", "petrovaradin", vrbas}) {
		t.Fatalf("unexpected starting service: %+v", r)
	}
	pins := map[gameapi.CityID]gameapi.GeoPoint{}
	for _, c := range f.Cities {
		pins[c.ID] = gameapi.GeoPoint{Lat: c.StationLat, Lon: c.StationLon}
		if c.ID == "belgrade" && (c.StationName != "Beograd Centar" || c.StationLat != 44.79358 || c.StationLon != 20.45304) {
			t.Fatal("wrong Belgrade terminal", c)
		}
	}
	corridors := map[string]gameapi.Corridor{}
	for _, c := range f.Corridors {
		corridors[c.ID] = c
	}
	for i, id := range r.Sections {
		c := corridors[id]
		if !c.Existing || c.FromCity != r.Stops[i] || c.ToCity != r.Stops[i+1] {
			t.Fatal("disconnected section", id)
		}
		path := c.Alignments[0].Path
		if path[0] != pins[c.FromCity] || path[len(path)-1] != pins[c.ToCity] {
			t.Fatal("station gap", id)
		}
	}
	nis := corridors["launch_30_belgrade__nis"]
	if nis.Existing || nis.Alignments[0].Path[0] != pins["belgrade"] {
		t.Fatal("Niš prospect disconnected")
	}
	publicCommand(t, g, gameapi.Command{Kind: gameapi.LeaseTrain, CorridorID: routeID})
	publicCommand(t, g, gameapi.Command{Kind: gameapi.SetService, CorridorID: routeID, Frequency: 1, Fare: 9000})
	f = publicCommand(t, g, gameapi.Command{Kind: gameapi.AdvanceMonth})
	if !slices.ContainsFunc(f.Reports[0].Services, func(s gameapi.ServiceRun) bool { return s.CorridorID == routeID && s.Revenue > 0 }) {
		t.Fatal("through service earned no revenue")
	}
	if openPublic(t, store).Snapshot().Checkpoint != f.Checkpoint {
		t.Fatal("service did not survive reload")
	}
}

func TestSerbiaMigrationPreservesOperatingCompany(t *testing.T) {
	if _, err := publicHistoryCatalog(); err != nil {
		t.Fatal(err)
	}
	old := domain.New(priorSerbiaCatalog)
	const north = "public_stop_node_13860287537__stop_node_9767449803"
	for _, cmd := range []gameapi.Command{{Kind: gameapi.LeaseTrain, CorridorID: north}, {Kind: gameapi.SetService, CorridorID: north, Frequency: 1, Fare: 9000}, {Kind: gameapi.AdvanceMonth}} {
		cmd.Revision = old.Revision()
		var err error
		old, err = old.Change(cmd)
		if err != nil {
			t.Fatal(err)
		}
	}
	p := stateData(old).(viennaPayload).occitaniePayload
	p.Rules, p.PublicSHA = "network-v13", serbiaPriorSHA
	b, err := json.Marshal(p)
	if err != nil {
		t.Fatal(err)
	}
	raw, err := json.Marshal(envelope{"wwhsrt-first-business", 13, hash(b), b})
	if err != nil {
		t.Fatal(err)
	}
	store := &memoryStore{b: raw}
	g := openPublic(t, store)
	if g.w.SerbiaSince() != old.Months() || !reflect.DeepEqual(g.w.Reports(), old.Reports()) || !reflect.DeepEqual(g.w.Fleet(), old.Fleet()) {
		t.Fatal("previous company history changed")
	}
	publicCommand(t, g, gameapi.Command{Kind: gameapi.LeaseTrain, CorridorID: "network_belgrade__vrbas_nova"})
	f := publicCommand(t, g, gameapi.Command{Kind: gameapi.AdvanceMonth})
	if openPublic(t, store).Snapshot().Checkpoint != f.Checkpoint {
		t.Fatal("migrated company could not resume")
	}
}
