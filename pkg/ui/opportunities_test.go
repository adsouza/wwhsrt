package ui

import (
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
	f := gameapi.Frame{Routes: []gameapi.Route{
		{ID: "through", Open: true, Sections: []string{"ab", "bc"}},
		{ID: "ab", Open: true, Sections: []string{"ab"}},
		{ID: "cd", Open: true, Sections: []string{"cd"}},
	}, Services: []gameapi.Service{{CorridorID: "through", Frequency: 2}, {CorridorID: "ab", Frequency: 0}}, Fleet: []gameapi.FleetLease{{CorridorID: "cd"}}}
	running := OperatingSections(f)
	if len(running) != 2 || !running["ab"] || !running["bc"] || NextService(f) != "cd" {
		t.Fatal(running, NextService(f))
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
