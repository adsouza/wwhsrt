package ui

import (
	"context"
	"slices"
	"testing"

	"wwhsrt/pkg/gameapi"
)

func TestProspectsRankByMinimumPlayerCash(t *testing.T) {
	f := gameapi.Frame{Corridors: []gameapi.Corridor{
		{ID: "other", Alignments: []gameapi.Alignment{{Cost: 90}}},
		{ID: "cheapest", FromCity: "a", ToCity: "b", Alignments: []gameapi.Alignment{{ID: "direct", Cost: 100}, {ID: "subsidized", Cost: 400, Grant: 370}}},
		{ID: "public", Existing: true}, {ID: "started", Alignments: []gameapi.Alignment{{Cost: 2}}},
	}, Projects: []gameapi.Project{{CorridorID: "started"}}}
	rows := RankedProspects(f)
	if len(rows) != 2 || rows[0].ID != "cheapest" || rows[0].Cash != 30 || rows[1].Cash != 90 {
		t.Fatal(rows)
	}
	s := State{}
	if !SelectProspect(&s, f, rows[0].ID) {
		t.Fatal("row should open a draft")
	}
	q, ok := FundingBreakdown(f, s)
	if !ok || q.AlignmentID != rows[0].Alignment || q.Contribution != rows[0].Cash {
		t.Fatal(q, rows)
	}
}

func TestOperatingSectionsIncludeSharedThroughTrack(t *testing.T) {
	f := gameapi.Frame{Corridors: []gameapi.Corridor{{ID: "bc", SharedSections: []string{"bx", "xc"}}}, Routes: []gameapi.Route{
		{ID: "through", Open: true, Sections: []string{"ab", "bc"}},
		{ID: "ab", Open: true, Sections: []string{"ab"}},
		{ID: "cd", Open: true, Sections: []string{"cd"}},
	}, Services: []gameapi.Service{{CorridorID: "through", Frequency: 2}, {CorridorID: "ab", Frequency: 0}}, Fleet: []gameapi.FleetLease{{CorridorID: "cd"}}}
	running := OperatingSections(f)
	direct := ServiceCandidates(f)[2]
	if len(running) != 4 || !running["bx"] || !running["xc"] || !running["ab"] || !running["bc"] || len(direct) != 1 || direct[0].ID != "cd" {
		t.Fatal(running, direct)
	}
	f.Services[0].Frequency = 0
	if len(OperatingSections(f)) > 0 {
		t.Fatal("suspension should clear both sections")
	}
}

func TestBestFrequencyIncludesIdleCapacityAndLeaseCosts(t *testing.T) {
	f := gameapi.Frame{}
	q := gameapi.ForecastRequest{CorridorID: "through", Fare: 5000, ReplaceServiceID: "starter"}
	previews := 0
	forecast := func(draft gameapi.ForecastRequest) (gameapi.Forecast, error) {
		previews++
		if draft.Fare != q.Fare || draft.ReplaceServiceID != q.ReplaceServiceID || draft.CorridorID != q.CorridorID {
			t.Fatal(draft)
		}
		n := draft.Frequency
		if n > 5 {
			return gameapi.Forecast{}, &gameapi.Error{Code: "track_capacity"}
		}
		// Frequencies 4 and 5 tie for best company result. The fifth trip
		// brings more route revenue at the expense of another service.
		return gameapi.Forecast{Open: true, FleetNeeded: n, FleetAvailable: 2, CompanyChange: gameapi.Money(min(n, 4)), Contribution: gameapi.Money(n)}, nil
	}
	if n, ok := BestFrequency(f, q, forecast); !ok || n != 4 || previews != 17 {
		t.Fatal(n, ok, previews)
	}
	f.Player.Cash = -100
	if n, ok := BestFrequency(f, q, forecast); !ok || n != 2 {
		t.Fatal(n, ok)
	}
	q.ReplaceServiceID = ""
	if n, ok := BestFrequency(f, q, func(q gameapi.ForecastRequest) (gameapi.Forecast, error) {
		return gameapi.Forecast{Open: true, CompanyChange: -gameapi.Money(q.Frequency)}, nil
	}); !ok || n != 0 {
		t.Fatal(n, ok)
	}
}

func TestRecommendServiceRanksSchedulesWithinPriority(t *testing.T) {
	f := gameapi.Frame{
		Routes: []gameapi.Route{
			{ID: "direct", Open: true, Sections: []string{"direct"}},
			{ID: "fleet", Open: true, Sections: []string{"fleet"}},
			{ID: "owned-a", Open: true, Sections: []string{"a"}},
			{ID: "owned-b", Open: true, Sections: []string{"b", "c"}, SuggestedFare: 7500, SuggestedFrequency: 2},
		},
		Corridors: []gameapi.Corridor{{ID: "project-b", OpeningService: "owned-b"}},
		IdleLines: []gameapi.IdleLine{{CorridorID: "fleet", Leases: 1}, {CorridorID: "owned-a", Maintenance: 1}, {CorridorID: "project-b", Maintenance: 1}},
	}
	gains := map[string]gameapi.Money{"direct": 100, "fleet": 80, "owned-a": 20, "owned-b": 35}
	forecast := func(q gameapi.ForecastRequest) (gameapi.Forecast, error) {
		r := gameapi.Forecast{Open: true, DeliveredFrequency: q.Frequency, CompanyChange: gains[q.CorridorID]}
		if q.CorridorID == "owned-b" && q.Frequency != 9 {
			r.CompanyChange = 0
		}
		return r, nil
	}
	pick := func(id string, frequency int) {
		t.Helper()
		got := RecommendService(context.Background(), f, forecast)
		if got.CorridorID != id || got.Frequency != frequency {
			t.Fatalf("got %+v, want %s at %d", got, id, frequency)
		}
		if id == "owned-b" && got.Fare != 7500 {
			t.Fatal("changed planner fare", got)
		}
	}
	pick("owned-b", 9)
	slices.Reverse(f.Routes)
	pick("owned-b", 9)
	gains["owned-a"], gains["owned-b"] = 0, -1
	pick("fleet", 1)
	gains["fleet"] = 0
	pick("direct", 1)
	gains["direct"] = 0
	pick("", 0)
}

func TestRecommendServiceRequiresFeasibleGainAndBreaksTies(t *testing.T) {
	f := gameapi.Frame{Services: []gameapi.Service{{CorridorID: "a", Frequency: 2, Fare: 6300}}}
	f.Player.Cash = -1
	for _, id := range []string{"z", "b", "a", "unleased"} {
		f.Routes = append(f.Routes, gameapi.Route{ID: id, Open: true, Sections: []string{id}})
		if id != "unleased" {
			f.Fleet = append(f.Fleet, gameapi.FleetLease{CorridorID: id})
		}
	}
	forecast := func(q gameapi.ForecastRequest) (gameapi.Forecast, error) {
		if q.CorridorID == "unleased" {
			t.Fatal("compared route without a leased train while in deficit")
		}
		r := gameapi.Forecast{Open: true, DeliveredFrequency: q.Frequency, FleetAvailable: 1, FleetNeeded: 1, CompanyChange: 50}
		switch q.Frequency {
		case 1:
			r.CompanyChange = 400
			r.DeliveredFrequency = 0
		case 2:
			r.CompanyChange = 800
			return r, &gameapi.Error{Code: "track_capacity"}
		case 3:
			r.CompanyChange = 900
			r.FleetNeeded = 2
			r.AdditionalLeases = 1
		}
		return r, nil
	}
	got := RecommendService(context.Background(), f, forecast)
	if got.CorridorID != "a" || got.Frequency != 4 || got.Fare != 6300 {
		t.Fatal(got)
	}
	f.Player.Cash = 0
	f.Routes = f.Routes[:3]
	got = RecommendService(context.Background(), f, func(q gameapi.ForecastRequest) (gameapi.Forecast, error) {
		r := gameapi.Forecast{Open: true, DeliveredFrequency: q.Frequency, CompanyChange: 50, AdditionalLeases: 1}
		if q.CorridorID == "z" && q.Frequency >= 7 {
			r.AdditionalLeases = 0
		}
		return r, nil
	})
	if got.CorridorID != "z" || got.Frequency != 7 {
		t.Fatal(got)
	}
}
