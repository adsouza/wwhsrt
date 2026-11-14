package domain

import (
	"errors"
	"testing"
	"wwhsrt/pkg/gameapi"
)

func TestCapacityStationCallsAndPublicProvisionalPolicy(t *testing.T) {
	w := networkWorld(t)
	w.projects = []gameapi.Project{{CorridorID: "tsuruga-osaka", AlignmentID: "mountain", Owner: w.company.ID, Ready: 0}}
	// A scenario with more track paths makes station calls the actual constraint.
	w.catalog.network.PrivatePaths = 64
	w.services = []gameapi.Service{{CorridorID: "tsuruga-osaka", Frequency: 16, Fare: 9000}, {CorridorID: "nagoya-tsuruga-express", Frequency: 16, Fare: 9000}}
	f, err := w.Forecast(gameapi.ForecastRequest{CorridorID: "tokyo-tsuruga-express", Frequency: 1, Fare: 9000})
	var e *gameapi.Error
	if !errors.As(err, &e) || e.Code != "station_capacity" || len(e.CapacityUsage) != 1 {
		t.Fatal(f, err)
	}
	c := e.CapacityUsage[0]
	if c.CityID != "tsuruga" || c.Requested != 66 || c.Capacity != 64 || c.Excess != 2 || c.DraftUse != 2 || c.MaxFrequency != 0 {
		t.Fatal(c)
	}
	for _, s := range c.Services {
		if s.Usage != s.Frequency*2 {
			t.Fatal("station count uses wrong unit", s)
		}
	}
	w = networkWorld(t)
	f, err = w.Forecast(gameapi.ForecastRequest{CorridorID: "tokyo-osaka", Frequency: 16, Fare: 9000})
	if err != nil || f.DeliveredFrequency >= 16 || len(f.CapacityUsage) != 0 {
		t.Fatal("public paths should remain provisional", f, err)
	}
}
func TestCapacityReplacementRemovesSourceServiceFromUsage(t *testing.T) {
	w := networkWorld(t)
	w.projects = []gameapi.Project{{CorridorID: "tsuruga-osaka", AlignmentID: "mountain", Owner: w.company.ID, Ready: 0}}
	w.services = []gameapi.Service{{CorridorID: "tsuruga-osaka", Frequency: 10, Fare: 9000}}
	f, err := w.Forecast(gameapi.ForecastRequest{CorridorID: "tokyo-tsuruga", ReplaceServiceID: "tsuruga-osaka", Frequency: 10, Fare: 9000})
	if err != nil || f.CapacityUsage[0].Requested != 10 || f.CapacityUsage[0].DraftUse != 10 || len(f.CapacityUsage[0].Services) != 1 {
		t.Fatal("replacement source counted twice", f, err)
	}
}
