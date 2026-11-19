package ui

import (
	"context"
	"testing"

	"wwhsrt/pkg/gameapi"
)

func TestServiceRankingReusesScoresAcrossCompanyChanges(t *testing.T) {
	f := gameapi.Frame{Month: gameapi.Month{Year: 2032, Month: 1}}
	for _, id := range []string{"a", "b", "c"} {
		f.Routes = append(f.Routes, gameapi.Route{ID: id, Open: true, Sections: []string{id}})
	}
	independent := map[string]bool{"a": true, "b": true, "c": true}
	forecasts := map[string]int{}
	gain := map[string]gameapi.Money{"a": 30, "b": 20, "c": 10}
	batch := gameapi.ServiceComparison{Independent: func(r gameapi.Route) bool { return independent[r.ID] }, Forecast: func(q gameapi.ForecastRequest) (gameapi.Forecast, error) {
		forecasts[q.CorridorID]++
		return gameapi.Forecast{Open: true, DeliveredFrequency: q.Frequency, CompanyChange: gain[q.CorridorID]}, nil
	}}
	var ranking ServiceRanking
	check := func(want string, repriced ...string) {
		t.Helper()
		clear(forecasts)
		var plan gameapi.ForecastRequest
		plan, ranking = ranking.Recommend(context.Background(), f, batch)
		if plan.CorridorID != want || len(forecasts) != len(repriced) {
			t.Fatalf("got %s and forecasts %v, want %s and %v", plan.CorridorID, forecasts, want, repriced)
		}
		for _, id := range repriced {
			if forecasts[id] != gameapi.MaxFrequency {
				t.Fatalf("missing fresh frequencies for %s: %v", id, forecasts)
			}
		}
	}
	check("a", "a", "b", "c")
	f.Player.Cash += 100
	check("a")
	// Leasing changes the economics of that route and moves it up a priority group.
	f.Fleet = []gameapi.FleetLease{{CorridorID: "a"}}
	f.IdleLines = []gameapi.IdleLine{{CorridorID: "a", Leases: 1}}
	check("a", "a")
	f.IdleLines = nil
	f.Services = []gameapi.Service{{CorridorID: "a", Frequency: 1, Fare: 9000}}
	independent["a"], independent["c"] = false, false
	gain["c"] = 40 // a connection increases C's projected earnings.
	check("c", "c")
	f.Services[0].Fare = 8000
	gain["c"] = 5
	check("b", "c")
	f.Month.Month = 2
	check("b", "b", "c")
	f.Month.Month = 3
	check("b") // January and March are both 31-day months.
	f.Player.Cash = -100
	check("a", "a")
	f.Player.Cash = 0
	f.Projects = []gameapi.Project{{CorridorID: "new", Ready: 5}}
	f.Stations = []gameapi.Station{{CityID: "new-city", Ready: 5}}
	check("b")
	f.MonthsResolved = 5
	f.Stations[0].Open = true
	check("b", "b", "c")
}
