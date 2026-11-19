package application

import (
	"slices"
	"testing"

	"wwhsrt/pkg/gameapi"
	"wwhsrt/pkg/ui"
)

func TestExpansionNetworkServicesAndConstruction(t *testing.T) {
	checkRoute := func(t *testing.T, f gameapi.Frame, id string, open bool, stops ...gameapi.CityID) gameapi.Route {
		t.Helper()
		route, found := ui.ServiceRoute(f, id)
		if !found || route.Open != open || !slices.Equal(route.Stops, stops) {
			t.Fatalf("%s: found=%t, open=%t, stops=%v", id, found, route.Open, route.Stops)
		}
		return route
	}
	start := openPublic(t, nil).Snapshot()
	for id, stops := range map[string][]gameapi.CityID{
		"stockholm__sodertalje": {"stockholm", "flemingsberg", "sodertalje"},
		"stockholm__eskilstuna": {"stockholm", "flemingsberg", "sodertalje", "eskilstuna"},
		"public_malmo__lund":    {"malmo", "lund"},
		"malmo__helsingborg":    {"malmo", "lund", "landskrona", "helsingborg"},
	} {
		checkRoute(t, start, id, true, stops...)
	}
	cases := []struct {
		project, service string
		stops            []gameapi.CityID
	}{
		{"upgrade_bern__frutigen", "bern__visp", []gameapi.CityID{"bern", "frutigen", "visp"}},
		{"upgrade_milan__lugano", "arth_goldau__milan", []gameapi.CityID{"arth_goldau", "bellinzona", "lugano", "milan"}},
		{"upgrade_olten__arth_goldau", "olten__arth_goldau_via_lucerne", []gameapi.CityID{"olten", "lucerne", "arth_goldau"}},
		{"new_kinshasa__luanda", "kinshasa__luanda_via_mbanza", []gameapi.CityID{"kinshasa", "mbanza_kongo", "luanda"}},
		{"new_pune__hyderabad", "new_pune__hyderabad", []gameapi.CityID{"pune", "hyderabad"}},
		{"new_dhaka__kolkata", "dhaka__kolkata_via_jashore", []gameapi.CityID{"dhaka", "jashore", "kolkata"}},
	}
	for _, tc := range cases {
		t.Run(tc.project, func(t *testing.T) {
			store := &memoryStore{}
			g := openPublic(t, store)
			checkRoute(t, start, tc.service, false, tc.stops...)
			var offer ui.Prospect
			for _, p := range ui.RankedProspects(start) {
				if p.ID == tc.project {
					offer = p
					break
				}
			}
			if offer.ID == "" {
				t.Fatal("missing construction offer")
			}
			q, err := g.Forecast(gameapi.ForecastRequest{CorridorID: offer.ID, AlignmentID: offer.Alignment, Frequency: 1, Fare: 3000})
			if err != nil || q.Minutes == 0 || q.Open {
				t.Fatal("unbuilt project preview", q.Minutes, err)
			}
			publicCommand(t, g, gameapi.Command{Kind: gameapi.TakeLoan, Amount: offer.Cash})
			f := publicCommand(t, g, gameapi.Command{Kind: gameapi.BuildLine, CorridorID: tc.project, AlignmentID: offer.Alignment})
			checkRoute(t, f, tc.service, false, tc.stops...)
			if len(f.Projects) != 1 {
				t.Fatal("expected a single construction contract")
			}
			for g.w.Months() < f.Projects[0].Ready {
				publicCommand(t, g, gameapi.Command{Kind: gameapi.AdvanceYear})
			}
			g = openPublic(t, store)
			f = g.Snapshot()
			r := checkRoute(t, f, tc.service, true, tc.stops...)
			for _, stop := range r.Stops {
				if !slices.ContainsFunc(f.Stations, func(s gameapi.Station) bool { return s.CityID == stop && s.Open }) {
					t.Fatalf("missing passenger station %s", stop)
				}
			}
			if tc.project == "new_dhaka__kolkata" {
				co := f.Corridors[slices.IndexFunc(f.Corridors, func(c gameapi.Corridor) bool { return c.ID == offer.ID })]
				if co.Alignments[0].RetainedKM != 7 || !slices.Contains(r.Sections, "public_padma_north__padma_south") {
					t.Fatal("bridge reconstruction charged or deck bypassed")
				}
			}
			publicCommand(t, g, gameapi.Command{Kind: gameapi.LeaseTrain, CorridorID: tc.service})
			publicCommand(t, g, gameapi.Command{Kind: gameapi.SetService, CorridorID: tc.service, Frequency: 1, Fare: 3000})
			f = publicCommand(t, g, gameapi.Command{Kind: gameapi.AdvanceMonth})
			last := f.Reports[len(f.Reports)-1]
			if !slices.ContainsFunc(last.Services, func(r gameapi.ServiceRun) bool { return r.CorridorID == tc.service && r.Revenue > 0 }) {
				t.Fatal("service did not carry paying passengers", last.Services)
			}
			if tc.project == "new_dhaka__kolkata" {
				for _, trip := range f.Day {
					if trip.ServiceID == tc.service {
						if !slices.ContainsFunc(trip.Legs, func(l gameapi.DayLeg) bool {
							return l.CorridorID == "public_padma_north__padma_south" && l.End-l.Start == 5
						}) {
							t.Fatal("bridge has no slower timetable leg")
						}
						return
					}
				}
				t.Fatal("no scheduled bridge crossing")
			}
		})
	}
}

func TestExpansionConnectionsNeedBothContracts(t *testing.T) {
	cases := []struct{ route, first, second string }{
		{"olten__milan", "upgrade_milan__lugano", "upgrade_olten__arth_goldau"},
		{"mumbai__hyderabad", "new_pune__hyderabad", "launch_41_mumbai__pune"},
		{"kolkata__chattogram", "new_dhaka__kolkata", "launch_44_dhaka__chattogram"},
	}
	for _, tc := range cases {
		t.Run(tc.route, func(t *testing.T) {
			g := openPublic(t, &memoryStore{})
			for i, id := range []string{tc.first, tc.second} {
				f := g.Snapshot()
				r, ok := ui.ServiceRoute(f, tc.route)
				if !ok || r.Open {
					t.Fatal("connection available too early", tc.route)
				}
				offers := ui.RankedProspects(f)
				at := slices.IndexFunc(offers, func(q ui.Prospect) bool { return q.ID == id })
				if at < 0 {
					t.Fatal("missing offer", id)
				}
				q := offers[at]
				if gap := q.Cash - f.Player.Cash; gap > 0 {
					publicCommand(t, g, gameapi.Command{Kind: gameapi.TakeLoan, Amount: gap})
				}
				f = publicCommand(t, g, gameapi.Command{Kind: gameapi.BuildLine, CorridorID: id, AlignmentID: q.Alignment})
				for g.w.Months() < f.Projects[i].Ready {
					publicCommand(t, g, gameapi.Command{Kind: gameapi.AdvanceYear})
				}
			}
			r, ok := ui.ServiceRoute(g.Snapshot(), tc.route)
			if !ok || !r.Open {
				t.Fatal("completed connection unavailable", tc.route)
			}
		})
	}
}

func TestGloggnitzCallUsesTheSameTunnel(t *testing.T) {
	f := openPublic(t, nil).Snapshot()
	express, _ := ui.ServiceRoute(f, "vienna__graz")
	stopping, ok := ui.ServiceRoute(f, "vienna__graz_via_gloggnitz")
	if !ok || !slices.Equal(stopping.Stops, []gameapi.CityID{"vienna", "gloggnitz", "graz"}) ||
		!slices.Equal(express.Sections, stopping.Sections) || stopping.Minutes != express.Minutes+4 ||
		ui.RouteName(express, "en") == ui.RouteName(stopping, "en") {
		t.Fatal("invalid Gloggnitz service option", stopping)
	}
}
