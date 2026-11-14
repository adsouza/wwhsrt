package ui

import (
	"reflect"
	"testing"
	"wwhsrt/pkg/gameapi"
)

func TestNewGameClearsDraftsAndRetainsDevicePreferences(t *testing.T) {
	s := State{
		Locale:          "zh-Hans",
		Portrait:        true,
		Motion:          false,
		HideTerrain:     true,
		Appearance:      "dark",
		Dark:            true,
		View:            "services",
		Expanded:        true,
		Selected:        "sapporo",
		Query:           "old",
		DraftName:       "Old",
		HintDismissed:   true,
		FromCity:        "osaka",
		ToCity:          "sapporo",
		AlignmentID:     "valley",
		ServiceCorridor: "old",
		Frequency:       0,
		Fare:            12000,
		LoanAmount:      "987654321",
	}
	f := gameapi.Frame{
		Player:   gameapi.Company{Name: "Sunrise Rail"},
		Services: []gameapi.Service{{CorridorID: "tokyo-nagoya", Frequency: 6, Fare: 9000}},
	}
	want := State{
		Locale:          "zh-Hans",
		Portrait:        true,
		Motion:          false,
		HideTerrain:     true,
		Appearance:      "dark",
		Dark:            true,
		View:            "map",
		DraftName:       "Sunrise Rail",
		FromCity:        "tsuruga",
		ToCity:          "osaka",
		Choosing:        "from",
		ServiceCorridor: "tokyo-nagoya",
		Frequency:       6,
		Fare:            9000,
		LoanAmount:      "1000000.00",
	}
	if got := NewGameState(s, f); !reflect.DeepEqual(got, want) {
		t.Fatal("new game retained company drafts or lost preferences", got)
	}
}

func TestChineseCatalogAndAliases(t *testing.T) {
	for key := range catalogs()["en"] {
		if catalogs()["zh-Hans"][key] == "" {
			t.Errorf("missing Chinese message %s", key)
		}
	}
	for key := range catalogs()["zh-Hans"] {
		if catalogs()["en"][key] == "" {
			t.Errorf("missing English message %s", key)
		}
	}
	for _, s := range []string{"zh", "zh-CN", "zh-SG", "zh-Hans-CN"} {
		if Locale(s) != "zh-Hans" {
			t.Fatal(s)
		}
	}
	if Locale("zh-Hant") != "en" {
		t.Fatal("unsupported locale did not fall back")
	}
	if !Matches(gameapi.City{ID: "tokyo", Name: "Tokyo"}, "东京") {
		t.Fatal("Chinese city search failed")
	}
}

func TestLoanAmountParsingIsExact(t *testing.T) {
	for text, want := range map[string]gameapi.Money{
		"1063340000":       106_334_000_000,
		"1,063,340,000.11": 106_334_000_011,
		"0.01":             1,
		" 123.45 ":         12345,
		"123.":             12300,
	} {
		got, ok := ParseDollars(text)
		if !ok || got != want {
			t.Errorf("%q: got %d %v", text, got, ok)
		}
	}
	for _, text := range []string{
		"",
		"-1",
		"+1",
		"1e9",
		"1,2",
		"12,34,567",
		"1.234",
		"NaN",
		"92233720368547758.08",
		"123abc",
		".12",
	} {
		if _, ok := ParseDollars(text); ok {
			t.Errorf("accepted invalid amount %q", text)
		}
	}
}
func TestFundingShortfallUsesChosenAlignmentAndCurrentCash(t *testing.T) {
	f := gameapi.Frame{
		Player: gameapi.Company{Cash: 10000},
		Corridors: []gameapi.Corridor{
			{
				ID:       "a-b",
				FromCity: "a",
				ToCity:   "b",
				Alignments: []gameapi.Alignment{
					{ID: "fast", Cost: 100000, Grant: 60000},
					{ID: "cheap", Cost: 80000, Grant: 60000},
				},
			},
		},
	}
	s := State{FromCity: "b", ToCity: "a", AlignmentID: "cheap"}
	if n := FundingShortfall(f, s); n != 10000 {
		t.Fatal(n)
	}
	f.Player.Cash = 20000
	if n := FundingShortfall(f, s); n != 0 {
		t.Fatal(n)
	}
	s.AlignmentID = "fast"
	if n := FundingShortfall(f, s); n != 20000 {
		t.Fatal(n)
	}
	f.Projects = []gameapi.Project{{CorridorID: "a-b"}}
	if n := FundingShortfall(f, s); n != 0 {
		t.Fatal("borrowed again for committed project", n)
	}
}

func TestConstructionOpportunitiesExcludeCommittedPairs(t *testing.T) {
	f := gameapi.Frame{Corridors: []gameapi.Corridor{
		{ID: "public", FromCity: "p", ToCity: "q", Existing: true},
		{ID: "started", FromCity: "a", ToCity: "b", Alignments: []gameapi.Alignment{{ID: "tunnel"}, {ID: "valley"}}},
		{ID: "unstarted", FromCity: "c", ToCity: "d"},
	}}
	if got := ConstructionOpportunities(f); len(got) != 2 || got[0].ID != "started" || got[1].ID != "unstarted" {
		t.Fatal("fresh construction opportunities missing", got)
	}
	f.Projects = []gameapi.Project{{CorridorID: "started", AlignmentID: "tunnel", Started: 1, Ready: 10}}
	for _, month := range []int{1, 5, 10, 50} {
		f.MonthsResolved = month
		got := ConstructionOpportunities(f)
		if len(got) != 1 || got[0].ID != "unstarted" {
			t.Fatalf("committed pair reappeared at month %d: %+v", month, got)
		}
		if pair, ok := Pair(f, "b", "a"); !ok || pair.ID != "started" {
			t.Fatal("committed pair must remain available for progress inspection")
		}
	}
	f.Projects = append(f.Projects, gameapi.Project{CorridorID: "unstarted"})
	if got := ConstructionOpportunities(f); len(got) != 0 {
		t.Fatal("committed catalog should have no new opportunities", got)
	}
}
