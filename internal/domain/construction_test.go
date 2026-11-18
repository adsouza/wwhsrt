package domain

import (
	"encoding/json"
	"testing"
	"wwhsrt/assets"
	"wwhsrt/pkg/gameapi"
)

func TestRegionalConstructionCostsAndInvalidIntervals(t *testing.T) {
	row := gameapi.Corridor{
		ID:          "new",
		FromCity:    "a",
		ToCity:      "b",
		DailyMarket: 500,
		OutsideCost: 1000,
		Alignments: []gameapi.Alignment{
			{
				ID:        "new",
				KM:        4,
				Minutes:   5,
				Months:    12,
				SurfaceKM: 1,
				HillKM:    1,
				ViaductKM: 1,
				TunnelKM:  1,
				Path:      []gameapi.GeoPoint{{}, {Lat: 1, Lon: 1}},
			},
		},
	}
	for _, base := range []gameapi.Money{1_000_000_000, 3_000_000_000} {
		row.Alignments[0].CostIntervals = []gameapi.ConstructionInterval{
			{
				Country:           "Fixture",
				ProfileID:         "regional",
				BasePerKM:         base,
				SurfaceKM:         1,
				HillKM:            1,
				ViaductKM:         1,
				TunnelKM:          1,
				HillFactorPercent: 150,
				TunnelFactor:      5,
			},
		}
		c, e := NewCatalog([]gameapi.Corridor{row})
		if e != nil {
			t.Fatal(e)
		}
		a := c.Corridors()[0].Alignments[0]
		civil := base - 300_000_000
		direct := base + (300_000_000 + civil*150/100) + (300_000_000 + civil*250/100) + (300_000_000 + civil*5)
		if a.Cost != (direct+direct*15/100)*120/100 || a.Grant != a.Cost*90/100 {
			t.Fatal("wrong regional cost/rounding")
		}
		copy := c.Corridors()
		copy[0].Alignments[0].CostIntervals[0].BasePerKM = 1
		if c.Corridors()[0].Alignments[0].CostIntervals[0].BasePerKM != base {
			t.Fatal("frame can mutate cost rules")
		}
	}
	row.Alignments[0].CostIntervals[0].SurfaceKM = 0
	if _, e := NewCatalog([]gameapi.Corridor{row}); e == nil {
		t.Fatal("unpriced interval accepted")
	}
}
func TestEveryLaunchProspectBuildsAndOpensAtItsBoundary(t *testing.T) {
	var raw struct {
		Corridors []gameapi.Corridor
		Network   gameapi.NetworkRules
	}
	if json.Unmarshal(assets.PublicNetwork, &raw) != nil {
		t.Fatal("catalog")
	}
	c, e := NewCatalog(raw.Corridors)
	if e != nil {
		t.Fatal(e)
	}
	c, e = c.WithNetwork(raw.Network)
	if e != nil {
		t.Fatal(e)
	}
	n := 0
	for _, co := range c.Corridors() {
		if co.Existing || co.PartOf != "" {
			continue
		}
		n++
		w := New(c)
		w.company.Cash = 100_000_000_000_000 // Explicitly funded command/boundary fixture, not an earned campaign.
		a := co.Alignments[0]
		cash := w.company.Cash
		w, e = w.Change(
			gameapi.Command{Kind: gameapi.BuildLine, CorridorID: co.ID, AlignmentID: a.ID, Revision: w.revision},
		)
		if e != nil {
			t.Fatal(co.ID, e)
		}
		if w.company.Cash != cash-a.Cost+a.Grant || w.projects[0].Ready != a.Months {
			t.Fatal("construction accounting", co.ID)
		}
		if _, e = w.Change(
			gameapi.Command{Kind: gameapi.BuildLine, CorridorID: co.ID, AlignmentID: a.ID, Revision: w.revision},
		); e == nil {
			t.Fatal("duplicate construction", co.ID)
		}
		w.months = a.Months - 1
		r, _ := w.route(co.ID)
		if r.Open {
			t.Fatal("opened early", co.ID)
		}
		w.months++
		r, _ = w.route(co.ID)
		if !r.Open {
			t.Fatal("failed to open", co.ID)
		}
	}
	if n != 96 {
		t.Fatal("missing spec prospects", n)
	}
}

func TestParallelConstructionRequiresAnExplicitDistinctProject(t *testing.T) {
	a := gameapi.Alignment{ID: "a", KM: 10, Minutes: 5, SurfaceKM: 10, Path: []gameapi.GeoPoint{{}, {Lat: 1, Lon: 1}}}
	public := gameapi.Corridor{
		ID:          "public",
		FromCity:    "a",
		ToCity:      "b",
		Existing:    true,
		DailyMarket: 1000,
		OutsideCost: 1000,
		Alignments:  []gameapi.Alignment{a},
	}
	a.Months = 12
	newLine := gameapi.Corridor{
		ID:          "new",
		FromCity:    "a",
		ToCity:      "b",
		DailyMarket: 1000,
		OutsideCost: 1000,
		Alignments:  []gameapi.Alignment{a},
	}
	if _, e := NewCatalog([]gameapi.Corridor{public, newLine}); e == nil {
		t.Fatal("ordinary duplicate accepted")
	}
	newLine.ConstructionKind = "dedicated_hsr"
	newLine.ParallelTo = []string{"public"}
	c, e := NewCatalog([]gameapi.Corridor{public, newLine})
	if e != nil {
		t.Fatal(e)
	}
	if c.resources("public")[0] == c.resources("new")[0] {
		t.Fatal("separate physical lines share one capacity resource")
	}
	newLine.ParallelTo = []string{"missing"}
	if _, e := NewCatalog([]gameapi.Corridor{public, newLine}); e == nil {
		t.Fatal("orphan parallel project accepted")
	}
}
