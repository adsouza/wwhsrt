package application

import (
	"encoding/json"
	"reflect"
	"slices"
	"testing"

	"wwhsrt/internal/domain"
	"wwhsrt/pkg/gameapi"
	"wwhsrt/pkg/ui"
)

func TestBranchProjectsUnlockPassengerServices(t *testing.T) {
	store := &memoryStore{}
	g := openPublic(t, store)
	const polish = "upgrade_psary__krakow"
	const tunnel = "semmering_base_tunnel"
	const stopping = "vienna__graz_via_both_portals"
	routes := func(f gameapi.Frame, poland, austria bool) {
		t.Helper()
		wantOpen := map[string]bool{
			"warsaw__krakow": poland, "vienna__graz": austria, stopping: austria, "vienna__klagenfurt": austria,
			"public_graz__klagenfurt": true, "public_wloszczowa__zawiercie": true,
		}
		for id, want := range wantOpen {
			r, ok := ui.ServiceRoute(f, id)
			if !ok || r.Open != want {
				t.Fatalf("%s open=%t, want %t (found %t)", id, r.Open, want, ok)
			}
			if slices.Contains(r.Stops, "psary") {
				t.Fatal("passenger call at Psary", r)
			}
		}
	}
	f := g.Snapshot()
	routes(f, false, false)
	for _, id := range []string{polish, tunnel} {
		if !slices.ContainsFunc(ui.RankedProspects(f), func(p ui.Prospect) bool { return p.ID == id }) {
			t.Fatal("missing construction offer", id)
		}
	}
	publicCommand(t, g, gameapi.Command{Kind: gameapi.TakeLoan, Amount: domain.DebtLimit})
	publicCommand(t, g, gameapi.Command{Kind: gameapi.BuildLine, CorridorID: polish, AlignmentID: "launch"})
	routes(g.Snapshot(), false, false)
	for i := 0; i < 3; i++ {
		publicCommand(t, g, gameapi.Command{Kind: gameapi.AdvanceYear})
	}
	f = g.Snapshot()
	routes(f, true, false)
	var draft ui.State
	if !ui.ReviewIdle(&draft, f, polish) || draft.ServiceCorridor != "warsaw__krakow" {
		t.Fatal("idle branch did not lead to Warsaw–Kraków service")
	}
	if _, ok := ui.ServiceRoute(f, polish); ok {
		t.Fatal("Psary junction offered as a passenger terminus")
	}
	publicCommand(t, g, gameapi.Command{Kind: gameapi.BuildLine, CorridorID: tunnel, AlignmentID: "launch"})
	for i := 0; i < 4; i++ {
		publicCommand(t, g, gameapi.Command{Kind: gameapi.AdvanceYear})
	}
	g = openPublic(t, store)
	f = g.Snapshot()
	routes(f, true, true)
	if !ui.ReviewIdle(&draft, f, tunnel) || draft.ServiceCorridor != stopping {
		t.Fatal("idle tunnel did not lead to Vienna–Graz service")
	}
	stoppingRoute, _ := ui.ServiceRoute(f, stopping)
	if !slices.Equal(stoppingRoute.Stops, []gameapi.CityID{"vienna", "gloggnitz", "muerzzuschlag", "graz"}) {
		t.Fatal("incorrect tunnel stops", stoppingRoute.Stops)
	}
	klagenfurt, _ := ui.ServiceRoute(f, "vienna__klagenfurt")
	if !slices.Equal(klagenfurt.Stops, []gameapi.CityID{"vienna", "graz", "klagenfurt"}) || !slices.Contains(klagenfurt.Sections, "public_graz__klagenfurt") {
		t.Fatal("through trains must serve Graz and reuse the Koralmbahn")
	}
	for _, id := range []string{"warsaw__krakow", stopping, "vienna__klagenfurt"} {
		publicCommand(t, g, gameapi.Command{Kind: gameapi.LeaseTrain, CorridorID: id})
		fare := gameapi.Money(5000)
		if id == stopping {
			fare = 3000
		} // The longer train also sells Vienna–Graz tickets.
		publicCommand(t, g, gameapi.Command{Kind: gameapi.SetService, CorridorID: id, Frequency: 1, Fare: fare})
	}
	f = publicCommand(t, g, gameapi.Command{Kind: gameapi.AdvanceMonth})
	for _, id := range []string{"warsaw__krakow", stopping, "vienna__klagenfurt"} {
		if !slices.ContainsFunc(f.Reports[len(f.Reports)-1].Services, func(r gameapi.ServiceRun) bool { return r.CorridorID == id && r.Revenue > 0 }) {
			t.Fatal("no paying passengers", id, f.Reports[len(f.Reports)-1].Services)
		}
	}
	if openPublic(t, store).Snapshot().Checkpoint != f.Checkpoint {
		t.Fatal("operating services failed to resume")
	}
}

func TestBranchMigrationPreservesCMKService(t *testing.T) {
	if _, err := publicHistoryCatalog(); err != nil {
		t.Fatal(err)
	}
	old := domain.New(priorBranchCatalog)
	const line = "public_wloszczowa__zawiercie"
	for _, cmd := range []gameapi.Command{{Kind: gameapi.LeaseTrain, CorridorID: line}, {Kind: gameapi.SetService, CorridorID: line, Frequency: 1, Fare: 5000}, {Kind: gameapi.AdvanceMonth}} {
		cmd.Revision = old.Revision()
		var err error
		old, err = old.Change(cmd)
		if err != nil {
			t.Fatal(err)
		}
	}
	p := stateData(old).(viennaPayload).operatingPayload
	p.Rules, p.PublicSHA = "network-v15", branchPriorSHA
	b, err := json.Marshal(p)
	if err != nil {
		t.Fatal(err)
	}
	raw, err := json.Marshal(envelope{"wwhsrt-first-business", 15, hash(b), b})
	if err != nil {
		t.Fatal(err)
	}
	store := &memoryStore{b: raw}
	g := openPublic(t, store)
	if g.w.BranchSince() != old.Months() || !reflect.DeepEqual(g.w.Reports(), old.Reports()) || !reflect.DeepEqual(g.w.Services(), old.Services()) {
		t.Fatal("upgrade changed CMK history")
	}
	f := publicCommand(t, g, gameapi.Command{Kind: gameapi.AdvanceMonth})
	if openPublic(t, store).Snapshot().Checkpoint != f.Checkpoint {
		t.Fatal("migrated service failed to resume")
	}
}

func TestViennaWesternApproachIsOpenBeforeSemmering(t *testing.T) {
	g := openPublic(t, &memoryStore{})
	frame := g.Snapshot()
	const line = "public_tullnerfeld__vienna"
	direct, ok := ui.ServiceRoute(frame, line)
	if !ok || !direct.Open || !slices.Equal(direct.StationNames, []string{"Tullnerfeld", "Wien Hauptbahnhof"}) {
		t.Fatal("missing public western approach", direct)
	}
	for _, origin := range []string{"munich", "nuremberg", "dresden"} {
		r, ok := ui.ServiceRoute(frame, origin+"__vienna_via_tullnerfeld")
		if !ok || !r.Open || !slices.Equal(r.Sections, []string{"public_" + origin + "__tullnerfeld", line}) {
			t.Fatalf("%s trains do not reach Hauptbahnhof via the shared track", origin)
		}
	}
	publicCommand(t, g, gameapi.Command{Kind: gameapi.LeaseTrain, CorridorID: line})
	publicCommand(t, g, gameapi.Command{Kind: gameapi.SetService, CorridorID: line, Frequency: 1, Fare: 1000})
	f := publicCommand(t, g, gameapi.Command{Kind: gameapi.AdvanceMonth})
	if len(f.Projects) != 0 || !slices.ContainsFunc(f.Reports[0].Services, func(s gameapi.ServiceRun) bool {
		return s.CorridorID == line && s.Revenue > 0
	}) {
		t.Fatal("western railway did not carry paying passengers at launch")
	}
}

func TestViennaMigrationPreservesOperatingWesternService(t *testing.T) {
	if _, err := publicHistoryCatalog(); err != nil {
		t.Fatal(err)
	}
	g := openPublic(t, nil)
	g.w = domain.New(priorViennaCatalog)
	const line = "public_munich__vienna"
	publicCommand(t, g, gameapi.Command{Kind: gameapi.LeaseTrain, CorridorID: line})
	publicCommand(t, g, gameapi.Command{Kind: gameapi.SetService, CorridorID: line, Frequency: 1, Fare: 3000})
	before := publicCommand(t, g, gameapi.Command{Kind: gameapi.AdvanceMonth})
	p := stateData(g.w).(viennaPayload).expansionPayload
	p.Rules, p.PublicSHA = "network-v17", viennaPriorSHA
	b, err := json.Marshal(p)
	if err != nil {
		t.Fatal(err)
	}
	raw, err := json.Marshal(envelope{"wwhsrt-first-business", 17, hash(b), b})
	if err != nil {
		t.Fatal(err)
	}
	store := &memoryStore{b: raw}
	g = openPublic(t, store)
	got := g.Snapshot()
	if got.Player != before.Player || !reflect.DeepEqual(got.Reports, before.Reports) ||
		!reflect.DeepEqual(got.Services, before.Services) || !reflect.DeepEqual(got.Fleet, before.Fleet) {
		t.Fatal("western station correction changed the saved company")
	}
	next := publicCommand(t, g, gameapi.Command{Kind: gameapi.AdvanceMonth})
	if openPublic(t, store).Snapshot().Checkpoint != next.Checkpoint {
		t.Fatal("migrated western service failed to resume")
	}
}
