package domain

import (
	"sort"
	"wwhsrt/pkg/gameapi"
)

// Sparse network reports retain every market for which the company offers a
// rail itinerary. Unserved OD detail is projected on demand from the same finite
// origin budget, rather than serializing hundreds of thousands of empty rows.
func (w World) InspectMarket(from, to gameapi.CityID, class string) (gameapi.MarketRun, error) {
	if class != "business" && class != "leisure" || from == to {
		return gameapi.MarketRun{}, errCode("no_route")
	}
	if len(w.reports) == 0 {
		return gameapi.MarketRun{}, errCode("no_report")
	}
	month := w.months - 1
	w = w.atMonth(month)
	var a, b gameapi.MarketCity
	destinations := []gameapi.MarketCity{}
	weights := []weighted{}
	for _, city := range w.catalog.network.Cities {
		if city.ID == from {
			a = city
		}
		if city.ID == to {
			b = city
		}
	}
	if a.ID == "" || b.ID == "" {
		return gameapi.MarketRun{}, errCode("no_route")
	}
	// Match simulation's sorted destination order and largest-remainder tie breaks.
	cities := append([]gameapi.MarketCity{}, w.catalog.network.Cities...)
	sort.Slice(cities, func(i, j int) bool { return cities[i].ID < cities[j].ID })
	for _, city := range cities {
		if city.ID == from {
			continue
		}
		destinations = append(destinations, city)
		weights = append(weights, weighted{string(city.ID), w.destinationWeight(a, city)})
	}
	d := days(Calendar(month))
	budget := w.dailyTrips(a) * d * 30 / 100
	if class == "leisure" {
		budget = w.dailyTrips(a)*d - budget
	}
	demand := 0
	for i, n := range shares(budget, weights) {
		if destinations[i].ID == to {
			demand = n
			break
		}
	}
	offers := w.outside(a, b, class)
	counts := shares(demand, offerWeights(offers))
	m := gameapi.MarketRun{From: from, To: to, Class: class, Demand: demand, Offers: offers}
	for i, o := range offers {
		switch o.Mode {
		case "road":
			m.Road = counts[i]
		case "air":
			m.Air = counts[i]
		case "conventional":
			m.Conventional = counts[i]
		case "no_trip":
			m.NoTrip = counts[i]
		}
	}
	return m, nil
}
