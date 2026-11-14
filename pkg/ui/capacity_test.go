package ui

import (
	"strings"
	"testing"
	"wwhsrt/pkg/gameapi"
)

func TestCapacityDetailsNameResourcesAndContributors(t *testing.T) {
	f := gameapi.Frame{
		Corridors: []gameapi.Corridor{{ID: "line", FromCity: "tsuruga", ToCity: "osaka"}},
		Routes: []gameapi.Route{
			{
				ID:     "draft",
				Cities: []gameapi.CityID{"tokyo", "nagoya", "osaka", "tsuruga"},
				Stops:  []gameapi.CityID{"tokyo", "tsuruga"},
			},
		},
	}
	usage := []gameapi.CapacityUsage{
		{
			Kind:           "track",
			CorridorID:     "line",
			Requested:      13,
			Capacity:       12,
			Excess:         1,
			DraftUse:       1,
			OtherRequested: 12,
			MaxFrequency:   0,
			Services: []gameapi.CapacityService{
				{ServiceID: "line", Frequency: 12, Usage: 12},
				{ServiceID: "draft", Frequency: 1, Usage: 1, Draft: true},
			},
		},
	}
	for _, locale := range []string{"en", "zh-Hans"} {
		rows := CapacityDetails(f, locale, usage)
		all := ""
		for _, r := range rows {
			all += r.Label + " " + r.Value + "\n"
		}
		for _, want := range []string{
			Text(locale, "tsuruga"),
			Text(locale, "osaka"),
			Text(locale, "capacityThisDraft"),
			Text(locale, "express"),
			"13",
			"12",
			"0",
		} {
			if !strings.Contains(all, want) {
				t.Fatal("missing explanation", want, all)
			}
		}
	}
}
