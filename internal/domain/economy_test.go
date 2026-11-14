package domain

import (
	"encoding/json"
	"testing"
	"wwhsrt/assets"
	"wwhsrt/pkg/gameapi"
)

func balancedWorld(t *testing.T) World {
	t.Helper()
	var source struct {
		Corridors []gameapi.Corridor
		Network   gameapi.NetworkRules
	}
	var rules gameapi.EconomyRules
	if e := json.Unmarshal(assets.PublicNetwork, &source); e != nil {
		t.Fatal(e)
	}
	if e := json.Unmarshal(assets.Economy, &rules); e != nil {
		t.Fatal(e)
	}
	c, e := NewCatalog(source.Corridors)
	if e != nil {
		t.Fatal(e)
	}
	c, e = c.WithNetwork(source.Network)
	if e != nil {
		t.Fatal(e)
	}
	c, e = c.WithEconomy(rules)
	if e != nil {
		t.Fatal(e)
	}
	return New(c)
}

func TestEconomyDemandIsBoundedMonotonicAndConserved(t *testing.T) {
	w := balancedWorld(t)
	for base, want := range map[int]int{500: 3000, 1195: 7170, 50000: 156000, 1000000: 400000} {
		if got := w.catalog.economyTrips(base); got != want {
			t.Fatal(base, got, want)
		}
	}
	last := 0
	for base := 1; base <= 100000; base++ {
		n := w.catalog.economyTrips(base)
		if n < last || n > 400000 {
			t.Fatal("unbounded/discontinuous trip budget")
		}
		last = n
	}
	for _, month := range []int{0, 1, 12} {
		n := w.simulate(w.services, month)
		want := 0
		for _, city := range w.catalog.network.Cities {
			want += w.dailyTrips(city) * days(Calendar(month))
		}
		got := n.UnservedDemand
		for _, m := range n.Markets {
			got += m.Demand
			if m.Demand != m.Carried+m.Road+m.Air+m.Conventional+m.NoTrip {
				t.Fatal("OD demand not conserved", m)
			}
		}
		if got != want || n.Demand != want {
			t.Fatal("network budget not conserved", got, want, n.Demand)
		}
		for _, s := range n.Services {
			for _, leg := range s.Legs {
				if leg.Passengers > leg.Capacity {
					t.Fatal("demand rebalance oversold seats")
				}
			}
		}
	}
}

func TestBreakEvenUsesActualFareMixAndRoundsUp(t *testing.T) {
	for _, test := range []struct {
		name   string
		f      gameapi.Forecast
		direct bool
		want   int
	}{
		{"empty direct", gameapi.Forecast{Costs: 100000}, true, 106},
		{"direct", gameapi.Forecast{Passengers: 10, Revenue: 10000, Costs: 100500, HandlingCosts: 500}, true, 106},
		{
			"short rides on through",
			gameapi.Forecast{Passengers: 100, Revenue: 50000, Costs: 105000, HandlingCosts: 5000},
			false,
			223,
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			f := test.f
			breakEven(&f, 1000, 1, test.direct)
			if !f.BreakEvenAvailable || f.BreakEvenPassengers != test.want {
				t.Fatal(f)
			}
		})
	}
	for _, f := range []gameapi.Forecast{
		{Passengers: 100, Revenue: 5000, HandlingCosts: 5000, Costs: 105000},
		{Costs: 100000},
	} {
		breakEven(&f, 1000, 1, false)
		if f.BreakEvenAvailable {
			t.Fatal("invented profitable margin or through passenger mix")
		}
	}
	f := gameapi.Forecast{Costs: 12000000}
	breakEven(&f, 9000, 0, true)
	if f.BreakEvenAvailable {
		t.Fatal("suspended service has a ridership target")
	}
}

func TestEconomyDefaultsFollowDistanceDemandAndRemainBounded(t *testing.T) {
	w := balancedWorld(t)
	routes := w.Routes()
	found := map[string]gameapi.Route{}
	for _, r := range routes {
		if r.SuggestedFrequency < 1 || r.SuggestedFrequency > 8 || r.SuggestedFare < 1000 || r.SuggestedFare > 30000 {
			t.Fatal("invalid suggested plan", r)
		}
		found[r.ID] = r
	}
	if found["public_kaohsiung__tainan"].SuggestedFare != 1000 ||
		found["public_lyon__marseille"].SuggestedFare != 4446 {
		t.Fatal("distance fare regression")
	}
	if found["public_lyon__marseille"].SuggestedFrequency >= found["public_beijing__tianjin"].SuggestedFrequency {
		t.Fatal("defaults do not respond to corridor demand")
	}
	for f := 1; f < 16; f++ {
		if w.serviceWait(f+1) >= w.serviceWait(f) {
			t.Fatal("more frequent trains did not improve scheduling")
		}
	}
	legacy := networkWorld(t)
	if legacy.TrainFamily().LeaseMonthly != 40000000 || w.TrainFamily().LeaseMonthly != 12000000 {
		t.Fatal("versioned rates leaked into historical economy")
	}
}
