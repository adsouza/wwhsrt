package application

import (
	"encoding/json"
	"reflect"
	"slices"
	"testing"
	"wwhsrt/internal/domain"
	"wwhsrt/pkg/gameapi"
)

func TestOccitanieServicesShareTheCoastalContract(t *testing.T) {
	store := &memoryStore{}
	g := openPublic(t, store)
	route := func(id string) gameapi.Route {
		for _, r := range g.Snapshot().Routes {
			if r.ID == id {
				return r
			}
		}
		t.Fatalf("missing route %s", id)
		return gameapi.Route{}
	}
	const fast = "occitanie_toulouse__narbonne"
	const coast = "launch_25_beziers__perpignan"
	const starter = "occitanie_toulouse__beziers"
	if r := route(starter); !r.Open || r.Minutes != 93 || !slices.Equal(r.Stops, []gameapi.CityID{"toulouse", "carcassonne", "narbonne", "beziers"}) {
		t.Fatal(r)
	}
	for _, id := range []string{fast, "occitanie_toulouse__perpignan", "occitanie_barcelona__toulouse"} {
		if route(id).Open {
			t.Fatal("unbuilt HSR is open", id)
		}
	}
	publicCommand(t, g, gameapi.Command{Kind: gameapi.LeaseTrain, CorridorID: starter})
	publicCommand(t, g, gameapi.Command{Kind: gameapi.SetService, CorridorID: starter, Frequency: 1, Fare: 9000})
	first := publicCommand(t, g, gameapi.Command{Kind: gameapi.AdvanceMonth})
	if !slices.ContainsFunc(first.Reports[0].Services, func(r gameapi.ServiceRun) bool { return r.CorridorID == starter && r.Revenue > 0 }) {
		t.Fatal("starting service earns no revenue")
	}
	publicCommand(t, g, gameapi.Command{Kind: gameapi.TakeLoan, Amount: domain.DebtLimit})
	for _, id := range []string{fast, coast} {
		publicCommand(t, g, gameapi.Command{Kind: gameapi.BuildLine, CorridorID: id, AlignmentID: "launch"})
	}
	if _, e := g.Apply(gameapi.Command{Kind: gameapi.BuildLine, CorridorID: "occitanie_narbonne__perpignan", AlignmentID: "launch", Revision: g.w.Revision()}); e == nil {
		t.Fatal("coastal section was sold twice")
	}
	for i := 0; i < 4; i++ {
		publicCommand(t, g, gameapi.Command{Kind: gameapi.AdvanceYear})
	}
	g = openPublic(t, store)
	if len(g.w.Projects()) != 2 || !route("occitanie_toulouse__perpignan").Open || !route("occitanie_barcelona__toulouse").Open || route("occitanie_toulouse__beziers_fast").Minutes >= route(starter).Minutes {
		t.Fatal("shared lines did not open after construction")
	}
}

func TestOccitanieMigrationPreservesWithdrawnServices(t *testing.T) {
	c, e := publicHistoryCatalog()
	if e != nil {
		t.Fatal(e)
	}
	old := domain.New(priorOccitanieCatalog)
	const line = "public_berlin__dresden"
	for _, cmd := range []gameapi.Command{{Kind: gameapi.LeaseTrain, CorridorID: line}, {Kind: gameapi.SetService, CorridorID: line, Frequency: 1, Fare: 9000}, {Kind: gameapi.AdvanceMonth}} {
		cmd.Revision = old.Revision()
		old, e = old.Change(cmd)
		if e != nil {
			t.Fatal(e)
		}
	}
	p := stateData(old).(serbiaPayload).germanPayload
	p.Rules = "network-v12"
	p.PublicSHA = occitaniePriorSHA
	b, _ := json.Marshal(p)
	raw, _ := json.Marshal(envelope{"wwhsrt-first-business", 12, hash(b), b})
	migrated, e := decodeExpanded(raw, c)
	if e != nil {
		t.Fatal(e)
	}
	if !reflect.DeepEqual(migrated.Reports(), old.Reports()) || !reflect.DeepEqual(migrated.Fleet(), old.Fleet()) {
		t.Fatal("lost settled history or assigned trains")
	}
	store := &memoryStore{b: raw}
	g := openPublic(t, store)
	f := publicCommand(t, g, gameapi.Command{Kind: gameapi.AdvanceMonth})
	if openPublic(t, store).Snapshot().Checkpoint != f.Checkpoint {
		t.Fatal("grandfathered route cannot reload")
	}
	fresh := openPublic(t, nil)
	if _, e = fresh.Apply(gameapi.Command{Kind: gameapi.LeaseTrain, CorridorID: line, Revision: fresh.w.Revision()}); e == nil {
		t.Fatal("new company can book retired railway")
	}
}
