package domain

import "wwhsrt/pkg/gameapi"

// RepresentativeDay derives an entire repeating day from the resolved plan and
// aggregate directed-leg loads. It does not dispatch agents or change business time.
func (w World) RepresentativeDay(network gameapi.NetworkReport) []gameapi.DayTrip {
	out := []gameapi.DayTrip{}
	if !network.Available {
		return out
	}
	// Resolve the representative plan with the same historical topology as its
	// report, including services retired by a later catalog correction.
	start := Calendar(0)
	month := (network.Month.Year-start.Year)*12 + network.Month.Month - start.Month
	if month >= 0 && month < CampaignMonths {
		w = w.atMonth(month)
	}
	for _, service := range network.Services {
		route, ok := w.route(service.ID)
		if !ok || len(route.Cities) < 2 {
			continue
		}
		for _, direction := range []int{1, -1} {
			cities := append([]gameapi.CityID{}, route.Cities...)
			sections := append([]string{}, route.Sections...)
			if direction < 0 {
				for i, j := 0, len(cities)-1; i < j; i, j = i+1, j-1 {
					cities[i], cities[j] = cities[j], cities[i]
				}
				for i, j := 0, len(sections)-1; i < j; i, j = i+1, j-1 {
					sections[i], sections[j] = sections[j], sections[i]
				}
			}
			for departure := 0; departure < service.Delivered; departure++ {
				at := departure * trainDayMinutes / service.Delivered
				trip := gameapi.DayTrip{
					ID:        departureID(service.ID, direction, departure),
					ServiceID: service.ID,
					Departure: at,
					Legs:      []gameapi.DayLeg{},
					Calls:     []gameapi.DayCall{{CityID: cities[0], Arrival: at, Departure: at}},
				}
				for i, id := range sections {
					co, _ := w.catalog.find(id)
					a, _ := w.alignment(co, "")
					load := 0
					for _, leg := range service.Legs {
						if leg.From == cities[i] && leg.To == cities[i+1] && leg.Capacity > 0 {
							load = leg.Passengers * 1000 / leg.Capacity
						}
					}
					trip.Legs = append(
						trip.Legs,
						gameapi.DayLeg{
							CorridorID:   id,
							AlignmentID:  a.ID,
							From:         cities[i],
							To:           cities[i+1],
							Start:        at,
							End:          at + a.Minutes,
							LoadPermille: load,
						},
					)
					at += a.Minutes
					if containsCity(route.Stops, cities[i+1]) {
						arrival := at
						if i < len(sections)-1 {
							at += w.catalog.network.DwellMinutes
						}
						trip.Calls = append(
							trip.Calls,
							gameapi.DayCall{CityID: cities[i+1], Arrival: arrival, Departure: at},
						)
					}
				}
				trip.Arrival = at
				out = append(out, trip)
			}
		}
	}
	return out
}
