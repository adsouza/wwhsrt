package application

import (
	"bytes"
	"encoding/json"
	"reflect"
	"testing"
	"wwhsrt/assets"
	"wwhsrt/internal/domain"
	"wwhsrt/pkg/gameapi"
)

func openPublic(t *testing.T, store Store) *Game {
	t.Helper()
	g, e := Open(store, nil)
	if e != nil {
		t.Fatal(e)
	}
	return g
}

// revision reads the current revision without building a whole frame.
func revision(g *Game) uint64 {
	g.mu.Lock()
	defer g.mu.Unlock()
	return g.w.Revision()
}
func publicCommand(t *testing.T, g *Game, c gameapi.Command) gameapi.Frame {
	t.Helper()
	c.Revision = revision(g)
	f, e := g.Apply(c)
	if e != nil {
		t.Fatalf("%+v: %v", c, e)
	}
	return f
}
func TestOsloStationCorrectionResumesV16AirportService(t *testing.T) {
	store := &memoryStore{}
	if _, err := publicHistoryCatalog(); err != nil {
		t.Fatal(err)
	}
	g := openPublic(t, nil)
	g.w = domain.New(priorExpansionCatalog)
	const line = "public_oslo__gardermoen"
	publicCommand(t, g, gameapi.Command{Kind: gameapi.LeaseTrain, CorridorID: line})
	publicCommand(t, g, gameapi.Command{Kind: gameapi.SetService, CorridorID: line, Frequency: 1, Fare: 5000})
	before := publicCommand(t, g, gameapi.Command{Kind: gameapi.AdvanceMonth})

	p := stateData(g.w).(viennaPayload).branchPayload
	p.Rules = "network-v16"
	p.PublicSHA = osloAirportPinSHA
	b, err := json.Marshal(p)
	if err != nil {
		t.Fatal(err)
	}
	store.b, err = json.Marshal(envelope{"wwhsrt-first-business", 16, hash(b), b})
	if err != nil {
		t.Fatal(err)
	}
	resumed := openPublic(t, store)
	if got := resumed.Snapshot(); got.Player != before.Player || !reflect.DeepEqual(got.Reports, before.Reports) || !reflect.DeepEqual(got.Fleet, before.Fleet) {
		t.Fatal("station correction changed the saved company")
	}
	next := publicCommand(t, resumed, gameapi.Command{Kind: gameapi.AdvanceMonth})
	if openPublic(t, store).Snapshot().Checkpoint != next.Checkpoint {
		t.Fatal("continued airport service failed to reload")
	}
}

func TestPublicNetworkChinaServiceAndContinuation(t *testing.T) {
	s := &memoryStore{}
	g := openPublic(t, s)
	f := g.Snapshot()
	if !f.PublicNetwork || len(f.Cities) < 250 || len(f.Corridors) < 100 {
		t.Fatal("public catalog missing")
	}
	cities := map[gameapi.CityID]gameapi.City{}
	for _, c := range f.Cities {
		cities[c.ID] = c
	}
	for _, id := range []gameapi.CityID{"beijing", "shanghai", "chengdu", "shenyang", "hong_kong", "taipei"} {
		if cities[id].StationName == "" {
			t.Fatalf("%s missing mapped station", id)
		}
	}
	for _, id := range []gameapi.CityID{"manila", "brasilia"} {
		if _, ok := cities[id]; ok {
			t.Fatalf("excluded metro %s returned", id)
		}
	}
	route := "route_beijing__shanghai"
	found := false
	for _, r := range f.Routes {
		if r.ID == route {
			found = r.Open && len(r.Sections) > 1 && r.Cities[0] == "beijing" && r.Cities[len(r.Cities)-1] == "shanghai"
		}
	}
	if !found {
		t.Fatal("China through route missing")
	}
	req := gameapi.ForecastRequest{CorridorID: route, Frequency: 2, Fare: 16000}
	quote, e := g.Forecast(req)
	if e != nil {
		t.Fatal(e)
	}
	for i := 0; i < quote.FleetNeeded; i++ {
		publicCommand(t, g, gameapi.Command{Kind: gameapi.LeaseTrain, CorridorID: route})
	}
	publicCommand(t, g, gameapi.Command{Kind: gameapi.SetService, CorridorID: route, Frequency: 2, Fare: 16000})
	f = publicCommand(t, g, gameapi.Command{Kind: gameapi.AdvanceMonth})
	carried := false
	for _, flow := range f.Network.Journeys {
		if flow.From == "beijing" && flow.To == "shanghai" && flow.Passengers > 0 {
			carried = true
		}
	}
	if !carried {
		t.Fatal("no end-to-end Chinese passengers")
	}
	for _, section := range f.Network.Sections {
		if section.Delivered > section.Capacity {
			t.Fatal("overbooked public track")
		}
	}
	total := f.Network.UnservedDemand
	for _, m := range f.Network.Markets {
		total += m.Demand
		if m.Demand != m.Carried+m.Road+m.Air+m.Conventional+m.NoTrip {
			t.Fatal("market conservation")
		}
	}
	source, _ := publicData()
	expectedDemand := 0
	for _, city := range source.Network.Cities {
		expectedDemand += min(400000, city.DailyTrips*3+min(6000, city.DailyTrips*3)) * 31
	}
	if f.Network.Demand != expectedDemand {
		t.Fatal("world origin budgets not conserved")
	}
	if total != f.Network.Demand {
		t.Fatal("sparse demand dropped")
	}
	r := f.Reports[0]
	if r.Revenue <= 0 || r.AccessCosts <= 0 ||
		r.ClosingCash != r.OpeningCash+r.Revenue-r.Costs+r.Grants+r.Borrowed-r.Repaid-r.Construction {
		t.Fatal("monthly ledger")
	}
	before := f.Checkpoint
	outside, e := g.InspectMarket("lagos", "kinshasa", "business")
	if e != nil || outside.Demand <= 0 || outside.Carried != 0 ||
		outside.Demand != outside.Road+outside.Air+outside.Conventional+outside.NoTrip {
		t.Fatalf("outside market: %+v %v", outside, e)
	}
	if g.Snapshot().Checkpoint != before {
		t.Fatal("inspection mutated world")
	}
	reopened := openPublic(t, s)
	if reopened.Snapshot().Checkpoint != before || !reflect.DeepEqual(reopened.Snapshot().Network, f.Network) {
		t.Fatal("public save continuation")
	}
	publicCommand(t, g, gameapi.Command{Kind: gameapi.AdvanceMonth})
	publicCommand(t, reopened, gameapi.Command{Kind: gameapi.AdvanceMonth})
	if g.Snapshot().Checkpoint != reopened.Snapshot().Checkpoint {
		t.Fatal("continued replay differs")
	}
}
func TestPublicNetworkLegacyUpgradeAtomicAndPinned(t *testing.T) {
	s := &memoryStore{}
	old := openTest(t, s)
	applyTest(t, old, gameapi.RenameCompany, "Preserved company")
	applyTest(t, old, gameapi.AdvanceMonth, "")
	before := old.Snapshot()
	raw := append([]byte{}, s.b...)
	g := openPublic(t, s)
	f := g.Snapshot()
	if !bytes.Equal(s.b, raw) || f.MonthsResolved != before.MonthsResolved || f.Player != before.Player ||
		!reflect.DeepEqual(f.Reports, before.Reports) ||
		!reflect.DeepEqual(f.Fleet, before.Fleet) ||
		!reflect.DeepEqual(f.Network, before.Network) {
		t.Fatal("read upgrade rewrote historical state")
	}
	s.fail = true
	_, e := g.Apply(gameapi.Command{Kind: gameapi.AdvanceMonth, Revision: f.Revision})
	if e == nil || !bytes.Equal(s.b, raw) || g.Snapshot().Checkpoint != f.Checkpoint {
		t.Fatal("migration write not atomic")
	}
	s.fail = false
	f = publicCommand(t, g, gameapi.Command{Kind: gameapi.AdvanceMonth})
	if !reflect.DeepEqual(f.Reports[0], before.Reports[0]) {
		t.Fatal("old report changed")
	}
	var env envelope
	json.Unmarshal(s.b, &env)
	if env.Version != 18 {
		t.Fatal("wrong save version")
	}
	if openPublic(t, s).Snapshot().Checkpoint != f.Checkpoint {
		t.Fatal("upgraded reload")
	}
	var p viennaPayload
	json.Unmarshal(env.Payload, &p)
	p.PublicSHA = "different catalog"
	env.Payload, _ = json.Marshal(p)
	env.SHA = hash(env.Payload)
	bad, _ := json.Marshal(env)
	if _, e := Open(&memoryStore{b: bad}, nil); e == nil {
		t.Fatal("foreign catalog accepted")
	}
}
func TestPublicNetworkSourceAndPhysicalCapacity(t *testing.T) {
	var raw struct {
		Stations []struct {
			ID       string
			Name     string
			Lon, Lat float64
			Tagged   bool
			Operator string
		}
		Lines     []json.RawMessage
		SourceSHA string `json:"source_sha256"`
	}
	if e := json.Unmarshal(assets.PublicNetwork, &raw); e != nil {
		t.Fatal(e)
	}
	if len(raw.Stations) != 4380 || len(raw.Lines) != 2397 || len(raw.SourceSHA) != 64 {
		t.Fatal("prototype inventory missing")
	}
	g := openPublic(t, nil)
	f := g.Snapshot()
	for _, city := range f.Cities {
		if len(city.ID) < 5 || string(city.ID[:5]) != "stop_" {
			continue
		}
		admitted := false
		for _, s := range raw.Stations {
			if s.Lon == city.StationLon && s.Lat == city.StationLat {
				admitted = s.Tagged || s.Operator != ""
				break
			}
		}
		if !admitted {
			t.Fatalf("unconfirmed source stop promoted to operating hub: %s", city.Name)
		}
	}
	co := gameapi.Corridor{}
	for _, c := range f.Corridors {
		if c.ID == "nagoya-osaka" {
			co = c
		}
	}
	if !reflect.DeepEqual(
		co.SharedSections,
		[]string{"public_maibara__nagoya", "public_kyoto__maibara", "public_kyoto__osaka"},
	) {
		t.Fatal("legacy express bypasses shared Maibara/Kyoto track capacity", co.SharedSections)
	}
	// Existing express and newly imported locals must consume the same resources.
	for _, id := range append([]string{"nagoya-osaka"}, co.SharedSections...) {
		quote, e := g.Forecast(gameapi.ForecastRequest{CorridorID: id, Frequency: 12, Fare: 5000})
		if e != nil {
			t.Fatal(e)
		}
		for i := 0; i < quote.FleetNeeded; i++ {
			publicCommand(t, g, gameapi.Command{Kind: gameapi.LeaseTrain, CorridorID: id})
		}
		publicCommand(t, g, gameapi.Command{Kind: gameapi.SetService, CorridorID: id, Frequency: 12, Fare: 5000})
	}
	f = publicCommand(t, g, gameapi.Command{Kind: gameapi.AdvanceMonth})
	for _, id := range co.SharedSections {
		found := false
		for _, s := range f.Network.Sections {
			if s.CorridorID == id {
				found = true
				if s.Requested != 24 || s.Delivered > 16 {
					t.Fatalf("shared capacity: %+v", s)
				}
			}
		}
		if !found {
			t.Fatal("missing shared section")
		}
	}
}

func TestExistingChineseConnectionsRejectDuplicateConstruction(t *testing.T) {
	store := &memoryStore{}
	g := openPublic(t, store)
	before := g.Snapshot()
	saved := append([]byte{}, store.b...)
	for _, id := range []string{"public_beijing__tianjin", "public_chengdu__chongqing", "public_chengdu__xi_an"} {
		found := false
		for _, c := range before.Corridors {
			if c.ID == id {
				found = c.Existing
			}
		}
		if !found {
			t.Fatalf("existing public link missing: %s", id)
		}
		_, err := g.Apply(
			gameapi.Command{
				Kind:        gameapi.BuildLine,
				CorridorID:  id,
				AlignmentID: "prototype",
				Revision:    before.Revision,
			},
		)
		if err == nil || err.Error() != "already_built" {
			t.Fatalf("%s duplicate build: %v", id, err)
		}
		if f := g.Snapshot(); f.Checkpoint != before.Checkpoint || len(f.Projects) != 0 ||
			!bytes.Equal(saved, store.b) {
			t.Fatal("rejected duplicate build changed company")
		}
		quote, e := g.Forecast(gameapi.ForecastRequest{CorridorID: id, Frequency: 2, Fare: 9000})
		if e != nil || !quote.Open {
			t.Fatalf("existing line unavailable for services: %s %v", id, e)
		}
	}
}

func TestServiceForecastsMatchPlannerAndRemainPinned(t *testing.T) {
	g := openPublic(t, nil)
	q := gameapi.ForecastRequest{CorridorID: "route_beijing__shanghai", Frequency: 2, Fare: 16000}
	before := g.ServiceForecasts()
	first, err := before(q)
	if err != nil {
		t.Fatal(err)
	}
	for i := 0; i < first.FleetNeeded; i++ {
		publicCommand(t, g, gameapi.Command{Kind: gameapi.LeaseTrain, CorridorID: q.CorridorID})
	}
	publicCommand(t, g, gameapi.Command{Kind: gameapi.SetService, CorridorID: q.CorridorID, Frequency: q.Frequency, Fare: q.Fare})
	again, err := before(q)
	if err != nil || !reflect.DeepEqual(first, again) {
		t.Fatal("forecast followed a subsequent company change", err)
	}
	batch := g.ServiceForecasts()
	for _, id := range []string{q.CorridorID, "public_chengdu__chongqing", "tsuruga-osaka"} {
		for _, n := range []int{0, 1, 4, 16} {
			q.CorridorID, q.Frequency = id, n
			got, err := batch(q)
			want, plannerErr := g.Forecast(q)
			if !reflect.DeepEqual(err, plannerErr) || !reflect.DeepEqual(got, want) {
				t.Fatalf("batch and planner diverged for %+v: %v, %v", q, err, plannerErr)
			}
		}
	}
}

func TestWarsawMarkerCorrectionResumesV18Service(t *testing.T) {
	g := openPublic(t, nil)
	const line = "public_warsaw__wloszczowa"
	publicCommand(t, g, gameapi.Command{Kind: gameapi.LeaseTrain, CorridorID: line})
	publicCommand(t, g, gameapi.Command{Kind: gameapi.SetService, CorridorID: line, Frequency: 1, Fare: 5000})
	before := publicCommand(t, g, gameapi.Command{Kind: gameapi.AdvanceMonth})
	p := stateData(g.w).(viennaPayload)
	p.PublicSHA = beforeWarsawCentralnaSHA
	b, err := json.Marshal(p)
	if err != nil {
		t.Fatal(err)
	}
	store := &memoryStore{}
	store.b, err = json.Marshal(envelope{"wwhsrt-first-business", 18, hash(b), b})
	if err != nil {
		t.Fatal(err)
	}
	resumed := openPublic(t, store)
	if got := resumed.Snapshot(); got.Checkpoint != before.Checkpoint {
		t.Fatal("saved company changed during station correction")
	}
	next := publicCommand(t, resumed, gameapi.Command{Kind: gameapi.AdvanceMonth})
	if openPublic(t, store).Snapshot().Checkpoint != next.Checkpoint {
		t.Fatal("continued Warsaw service failed to reload")
	}
}
