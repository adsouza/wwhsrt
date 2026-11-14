package ui

import (
	"testing"
	"wwhsrt/pkg/gameapi"
)

func TestFundingBreakdownExplainsCashAndDeficits(t *testing.T) {
	f := gameapi.Frame{
		Corridors: []gameapi.Corridor{
			{
				ID:         "a-b",
				FromCity:   "a",
				ToCity:     "b",
				Alignments: []gameapi.Alignment{{ID: "tunnel", Cost: 10_000_003, Grant: 9_000_002}},
			},
		},
	}
	s := State{FromCity: "b", ToCity: "a"} // Empty choice uses the named default alignment.
	for _, row := range []struct{ cash, available, deficit, gap gameapi.Money }{
		{300_001, 300_001, 0, 700_000},
		{0, 0, 0, 1_000_001},
		{-250_001, 0, 250_001, 1_250_002},
		{1_000_001, 1_000_001, 0, 0},
		{2_000_002, 2_000_002, 0, 0},
	} {
		f.Player.Cash = row.cash
		q, ok := FundingBreakdown(f, s)
		if !ok || q.CorridorID != "a-b" || q.FromCity != "a" || q.ToCity != "b" || q.AlignmentID != "tunnel" ||
			q.Cost != 10_000_003 ||
			q.Grant != 9_000_002 ||
			q.Contribution != 1_000_001 ||
			q.CashAvailable != row.available ||
			q.CashDeficit != row.deficit ||
			q.Shortfall != row.gap ||
			FundingShortfall(f, s) != row.gap {
			t.Fatal("funding explanation disagrees with quote", row, q)
		}
		if f.Player.Cash != row.cash || s.AlignmentID != "" {
			t.Fatal("preview changed state")
		}
	}
	f.Projects = []gameapi.Project{{CorridorID: "a-b"}}
	if _, ok := FundingBreakdown(f, s); ok {
		t.Fatal("committed project offered funding again")
	}
	f.Projects = nil
	f.Corridors[0].Existing = true
	if _, ok := FundingBreakdown(f, s); ok {
		t.Fatal("public track offered construction funding")
	}
}
