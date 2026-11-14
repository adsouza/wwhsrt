package domain

import (
	"encoding/json"
	"reflect"
	"testing"
	"wwhsrt/assets"
	"wwhsrt/pkg/gameapi"
)

func networkWorld(t *testing.T) World {
	t.Helper()
	var corridors []gameapi.Corridor
	var rules gameapi.NetworkRules
	if err := json.Unmarshal(assets.Business, &corridors); err != nil {
		t.Fatal(err)
	}
	if err := json.Unmarshal(assets.Network, &rules); err != nil {
		t.Fatal(err)
	}
	c, err := NewCatalog(corridors)
	if err != nil {
		t.Fatal(err)
	}
	c, err = c.WithNetwork(rules)
	if err != nil {
		t.Fatal(err)
	}
	return New(c)
}
func networkConservation(t *testing.T, n gameapi.NetworkReport) {
	t.Helper()
	carried, requested, demand, alternate, rejected := 0, 0, 0, 0, 0
	for _, m := range n.Markets {
		if m.Carried+m.Road+m.Air+m.Conventional+m.NoTrip != m.Demand || m.Requested != m.Carried+m.Rejected {
			t.Fatalf("market not conserved: %+v", m)
		}
		carried += m.Carried
		requested += m.Requested
		demand += m.Demand
		alternate += m.AlternateServed
		rejected += m.Rejected
	}
	if carried != n.Carried || requested != n.Requested || demand != n.Demand || alternate != n.AlternateServed ||
		rejected != n.Rejected {
		t.Fatal("network totals do not reconcile")
	}
	var revenue gameapi.Money
	var pkm, skm int64
	boards := 0
	for _, s := range n.Services {
		revenue += s.Revenue
		boards += s.Boardings
		if s.Delivered > s.Requested {
			t.Fatal("invented departures")
		}
		for _, leg := range s.Legs {
			if leg.Passengers < 0 || leg.Passengers > leg.Capacity {
				t.Fatal("leg overbooked", leg)
			}
			pkm += int64(leg.Passengers) * int64(leg.KM)
			skm += int64(leg.Capacity) * int64(leg.KM)
		}
	}
	if revenue != n.Revenue || pkm != n.PassengerKM || skm != n.SeatKM {
		t.Fatal("revenue/load attribution mismatch")
	}
	var ticketSales gameapi.Money
	journeys, transfers := 0, 0
	for _, flow := range n.Journeys {
		journeys += flow.Passengers
		transfers += flow.Passengers * len(flow.Transfers)
		ticketSales += gameapi.Money(flow.Passengers) * flow.Fare
	}
	if ticketSales != n.Revenue || journeys != n.Carried || transfers != n.Transfers || boards != journeys+transfers {
		t.Fatal("journeys counted more than once")
	}
	for _, section := range n.Sections {
		if section.Delivered > section.Capacity {
			t.Fatal("track overbooked")
		}
	}
	for _, station := range n.Stations {
		if station.Calls > station.Capacity || station.Transfers > station.Boardings ||
			station.Transfers > station.Alightings {
			t.Fatal("station activity invalid", station)
		}
	}
}
func TestNetworkThroughJourneyAndIntermediateSeats(t *testing.T) {
	w := networkWorld(t)
	plans := []gameapi.Service{{CorridorID: "tokyo-osaka", Frequency: 6, Fare: 13800}}
	n := w.simulate(plans, 0)
	networkConservation(t, n)
	through, locals := 0, 0
	for _, flow := range n.Journeys {
		if flow.From == "tokyo" && flow.To == "osaka" {
			if len(flow.Services) != 1 || len(flow.Transfers) != 0 || flow.Fare != 13800 {
				t.Fatal("through ticket invalid", flow)
			}
			through += flow.Passengers
		}
		if flow.From == "nagoya" {
			locals += flow.Passengers
		}
	}
	if through == 0 || locals == 0 {
		t.Fatal("through service did not carry end-to-end and intermediate markets")
	}
	graph := w.rides(plans, map[string]int{"tokyo-osaka": 6})
	for _, a := range graph["tokyo"] {
		for _, b := range graph[a.to] {
			if b.to == "tokyo" && a.fare != b.fare {
				t.Fatal("reverse fare differs", a.fare, b.fare)
			}
		}
	}
	plans[0].Fare = 13801
	graph = w.rides(plans, map[string]int{"tokyo-osaka": 6})
	for _, a := range graph["tokyo"] {
		for _, b := range graph[a.to] {
			if b.to == "tokyo" && a.fare != b.fare {
				t.Fatal("fractional reverse fare differs", a.fare, b.fare)
			}
		}
	}
	plans[0].Fare = 13800
	day := w.RepresentativeDay(n)
	if len(day) != 12 || len(day[0].Calls) != 3 || day[0].Calls[1].CityID != "nagoya" ||
		day[0].Calls[1].Departure-day[0].Calls[1].Arrival != 4 ||
		day[0].Arrival-day[0].Departure != 156 {
		t.Fatal("representative through day missing Nagoya dwell", day)
	}
	if !reflect.DeepEqual(n, w.simulate(plans, 0)) || !reflect.DeepEqual(day, w.RepresentativeDay(n)) {
		t.Fatal("repeated solve/day not deterministic")
	}
}

func FuzzNetworkConservation(f *testing.F) {
	f.Add(uint8(0), uint8(0), uint8(0), uint16(0))
	f.Add(uint8(16), uint8(16), uint8(16), uint16(29000))
	f.Add(uint8(1), uint8(8), uint8(13), uint16(12345))
	f.Fuzz(func(t *testing.T, a, b, c uint8, fare uint16) {
		w := networkWorld(t)
		price := gameapi.Money(1000 + int(fare)%29001)
		plans := []gameapi.Service{
			{CorridorID: "tokyo-nagoya", Frequency: int(a) % 17, Fare: price},
			{CorridorID: "nagoya-osaka", Frequency: int(b) % 17, Fare: price},
			{CorridorID: "tokyo-osaka", Frequency: int(c) % 17, Fare: price},
		}
		n := w.simulate(plans, 1)
		networkConservation(t, n)
		if !reflect.DeepEqual(n, w.simulate([]gameapi.Service{plans[2], plans[0], plans[1]}, 1)) {
			t.Fatal("permuting plans changes result")
		}
	})
}
func TestNetworkTransfersExpressAndSharedBottleneck(t *testing.T) {
	w := networkWorld(t)
	plans := []gameapi.Service{
		{CorridorID: "tokyo-nagoya", Frequency: 8, Fare: 9000},
		{CorridorID: "nagoya-osaka", Frequency: 8, Fare: 5000},
	}
	n := w.simulate(plans, 0)
	networkConservation(t, n)
	transfer := 0
	for _, flow := range n.Journeys {
		if flow.From == "tokyo" && flow.To == "osaka" {
			if len(flow.Services) != 2 || len(flow.Transfers) != 1 || flow.Transfers[0] != "nagoya" ||
				flow.Fare != 14000 {
				t.Fatal(flow)
			}
			transfer += flow.Passengers
		}
	}
	if transfer == 0 {
		t.Fatal("no through journey via transfer")
	}
	plans = append(plans, gameapi.Service{CorridorID: "tokyo-osaka-express", Frequency: 16, Fare: 12000})
	n = w.simulate(plans, 0)
	networkConservation(t, n)
	limited := false
	for _, s := range n.Services {
		if s.Delivered < s.Requested {
			limited = true
		}
		if s.ID == "tokyo-osaka-express" {
			for _, ride := range w.rides(plans, map[string]int{s.ID: s.Delivered})["nagoya"] {
				if ride.service == s.ID {
					t.Fatal("express allows Nagoya boarding")
				}
			}
		}
	}
	if !limited {
		t.Fatal("shared bottleneck did not constrain departures")
	}
	reversed := []gameapi.Service{plans[2], plans[1], plans[0]}
	if !reflect.DeepEqual(n, w.simulate(reversed, 0)) {
		t.Fatal("service insertion order changes allocation")
	}
}
func TestNetworkFeederAndOutsideEligibility(t *testing.T) {
	w := networkWorld(t)
	co, _ := w.catalog.find("tsuruga-osaka")
	a := co.Alignments[0]
	w.projects = []gameapi.Project{{CorridorID: co.ID, AlignmentID: a.ID, Owner: w.company.ID, Ready: 0}}
	w.services = []gameapi.Service{{CorridorID: "tokyo-osaka", Frequency: 12, Fare: 13800}}
	base := w.simulate(w.services, 0)
	with := w.simulate(append(w.Services(), gameapi.Service{CorridorID: "tsuruga-osaka", Frequency: 4, Fare: 5000}), 0)
	networkConservation(t, with)
	feeder := 0
	for _, flow := range with.Journeys {
		if (flow.From == "tsuruga" || flow.To == "tsuruga") && len(flow.Services) > 1 {
			feeder += flow.Passengers
		}
	}
	if feeder == 0 || with.Carried <= base.Carried {
		t.Fatal("branch did not create useful feeder journeys")
	}
	for _, m := range with.Markets {
		if m.From == "tokyo" && m.To == "sapporo" && m.Road != 0 {
			t.Fatal("road crosses an unauthored island connection")
		}
		if m.From == "nagoya" && m.To == "osaka" {
			for _, o := range m.Offers {
				if o.Mode == "air" {
					t.Fatal("short air option is ineligible")
				}
			}
		}
	}
}
func TestNetworkRationCompleteJourneysAndRounding(t *testing.T) {
	r := []capacityRequest{
		{id: "through", need: 100, legs: []string{"east", "west"}},
		{id: "east-local", need: 100, legs: []string{"east"}},
		{id: "west-local", need: 100, legs: []string{"west"}},
	}
	caps := map[string]int{"east": 100, "west": 60}
	ration(r, caps)
	if r[0].served != 30 || r[1].served != 70 || r[2].served != 30 || caps["east"] != 0 || caps["west"] != 0 {
		t.Fatal("proportional bottleneck/residual wrong", r, caps)
	}
	r = []capacityRequest{{id: "z", need: 1, legs: []string{"x"}}, {id: "a", need: 1, legs: []string{"x"}}}
	ration(r, map[string]int{"x": 1})
	if r[0].served != 0 || r[1].served != 1 {
		t.Fatal("largest remainder tie depends on input order")
	}
}
