package ui

import (
	"strings"
	"testing"
	"wwhsrt/pkg/gameapi"
)

func TestSuggestedPlanPreservesAcceptedServicesAndUsesRouteAdvice(t *testing.T) {
	f := gameapi.Frame{Routes: []gameapi.Route{{ID: "regional", SuggestedFare: 4446, SuggestedFrequency: 1, Sections: []string{"regional"}, Cities: []gameapi.CityID{"a", "b"}, Open: true}}, Services: []gameapi.Service{{CorridorID: "regional", Frequency: 3, Fare: 6000}}}
	s := State{}
	DraftService(&s, f, "regional")
	if s.Frequency != 3 || s.Fare != 6000 {
		t.Fatal("advice silently replaced accepted service")
	}
	f.Services = nil
	DraftService(&s, f, "regional")
	if s.Frequency != 1 || s.Fare != 4446 {
		t.Fatal("new route retained blanket defaults")
	}
	frequency, fare := ProspectServicePlan(f, "regional")
	if frequency != 1 || fare != 4446 {
		t.Fatal("construction preview used different assumptions")
	}
}
func TestEconomicsDetailsAreLocalizedAndExposeActualBreakEven(t *testing.T) {
	f := gameapi.Forecast{BreakEvenAvailable: true, BreakEvenPassengers: 6399, Passengers: 8825, RunningCosts: 11646700, AccessCosts: 4479500, LeaseCosts: 12000000, HandlingCosts: 441250}
	for _, locale := range []string{"en", "zh-Hans"} {
		rows := EconomicsDetails(f, locale)
		if len(rows) != 7 || rows[1].Value != "6399" {
			t.Fatal(rows)
		}
		for _, row := range rows {
			if row.Label == "" || strings.Contains(row.Label, "forecastPassengers") {
				t.Fatal("untranslated economics row")
			}
		}
	}
	f.BreakEvenAvailable = false
	if BreakEvenTarget(f) != "—" {
		t.Fatal("unavailable break-even looked like zero riders")
	}
}
