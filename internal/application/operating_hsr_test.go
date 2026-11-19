package application

import (
	"encoding/json"
	"reflect"
	"slices"
	"testing"

	"wwhsrt/internal/domain"
	"wwhsrt/pkg/gameapi"
)

func TestOperatingHSRLinesArePublicAtLaunch(t *testing.T) {
	g := openPublic(t, nil)
	frame := g.Snapshot()
	corridors := map[string]gameapi.Corridor{}
	for _, c := range frame.Corridors {
		corridors[c.ID] = c
	}
	routes := map[string]gameapi.Route{}
	for _, r := range frame.Routes {
		routes[r.ID] = r
	}
	for _, id := range []string{
		"public_munich_hbf__augsburg_hbf", "network_munich__ingolstadt__nuremberg", "network_berlin__wolfsburg__hannover",
		"network_belgrade__subotica", "network_beijing__shijiazhuang", "network_guangzhou__hong_kong",
		"network_haikou__sanya_east", "network_haikou__sanya_west", "network_fukuoka__kagoshima", "public_takeo__nagasaki",
		"network_madrid__valladolid", "network_madrid__malaga", "network_madrid__alicante", "network_paris__rennes",
		"public_milan__brescia", "network_ankara__sivas", "public_seoul__gangneung", "public_graz__klagenfurt",
		"network_copenhagen__ringsted", "public_helsinki__lahti", "network_oslo__eidsvoll", "network_warsaw__zawiercie",
		"network_lund__helsingborg", "public_sodertalje__eskilstuna", "public_bern__olten", "public_frutigen__visp",
		"network_arth_goldau__lugano", "public_naples__salerno", "network_ankara__karaman", "public_paris__strasbourg",
	} {
		t.Run(id, func(t *testing.T) {
			r := routes[id]
			if !r.Open || r.InfrastructureKind != "hsr" || len(r.StationNames) != len(r.Stops) {
				t.Fatalf("missing starting HSR service or terminal: %+v", r)
			}
			var previous gameapi.GeoPoint
			for i, s := range r.Sections {
				c := corridors[s]
				if !c.Existing || c.Retired {
					t.Fatal("requires construction", s)
				}
				path := c.Alignments[0].Path
				first, last := path[0], path[len(path)-1]
				if c.FromCity != r.Cities[i] {
					first, last = last, first
				}
				if i > 0 && first != previous {
					t.Fatal("sections meet at different terminals", s, previous, first)
				}
				previous = last
			}
			w := g.w
			for _, cmd := range []gameapi.Command{{Kind: gameapi.LeaseTrain, CorridorID: id}, {Kind: gameapi.SetService, CorridorID: id, Frequency: 1, Fare: 3500}, {Kind: gameapi.AdvanceMonth}} {
				cmd.Revision = w.Revision()
				var err error
				if w, err = w.Change(cmd); err != nil {
					t.Fatal(err)
				}
			}
			if !slices.ContainsFunc(w.Reports()[0].Services, func(s gameapi.ServiceRun) bool { return s.CorridorID == id && s.Revenue > 0 }) {
				t.Fatal("scheduled train earned no revenue")
			}
		})
	}
	for _, pair := range [][2]string{{"public_madrid__sevilla", "network_madrid__malaga"}, {"public_madrid__valencia", "network_madrid__alicante"}, {"public_hengyang__shenzhen", "network_guangzhou__hong_kong"}} {
		if !slices.Contains(corridors[pair[0]].SharedSections, routes[pair[1]].Sections[0]) {
			t.Fatal("overlapping services lack a common capacity limit", pair)
		}
	}
	for _, id := range []string{"upgrade_de_munich__stop_node_1058668074", "upgrade_de_munich__nuremberg", "upgrade_de_berlin__hannover"} {
		if _, exists := corridors[id]; exists {
			t.Fatal("obsolete offer advertised", id)
		}
		if _, err := g.w.Change(gameapi.Command{Kind: gameapi.BuildLine, CorridorID: id, AlignmentID: "launch", Revision: g.w.Revision()}); err == nil {
			t.Fatal("obsolete offer purchasable", id)
		}
	}
}

func TestOperatingHSRMigrationKeepsPurchasedUpgrade(t *testing.T) {
	if _, err := publicHistoryCatalog(); err != nil {
		t.Fatal(err)
	}
	old := domain.New(priorOperatingCatalog)
	const id = "upgrade_de_munich__stop_node_1058668074"
	for _, cmd := range []gameapi.Command{{Kind: gameapi.TakeLoan, Amount: domain.DebtLimit}, {Kind: gameapi.BuildLine, CorridorID: id, AlignmentID: "launch"}, {Kind: gameapi.AdvanceMonth}} {
		cmd.Revision = old.Revision()
		var err error
		if old, err = old.Change(cmd); err != nil {
			t.Fatal(err)
		}
	}
	p := stateData(old).(viennaPayload).serbiaPayload
	p.Rules, p.PublicSHA = "network-v14", operatingPriorSHA
	b, err := json.Marshal(p)
	if err != nil {
		t.Fatal(err)
	}
	raw, err := json.Marshal(envelope{"wwhsrt-first-business", 14, hash(b), b})
	if err != nil {
		t.Fatal(err)
	}
	store := &memoryStore{b: raw}
	g := openPublic(t, store)
	if g.w.OperatingSince() != old.Months() || g.w.Company() != old.Company() || !reflect.DeepEqual(g.w.Projects(), old.Projects()) || !reflect.DeepEqual(g.w.Reports(), old.Reports()) {
		t.Fatal("purchased contract or company history changed")
	}
	for g.w.Months() < old.Projects()[0].Ready {
		publicCommand(t, g, gameapi.Command{Kind: gameapi.AdvanceMonth})
	}
	publicCommand(t, g, gameapi.Command{Kind: gameapi.LeaseTrain, CorridorID: id})
	publicCommand(t, g, gameapi.Command{Kind: gameapi.SetService, CorridorID: id, Frequency: 1, Fare: 5000})
	f := publicCommand(t, g, gameapi.Command{Kind: gameapi.AdvanceMonth})
	if openPublic(t, store).Snapshot().Checkpoint != f.Checkpoint {
		t.Fatal("upgraded company could not resume")
	}
}
