package domain

import "wwhsrt/pkg/gameapi"

// Physical upkeep is charged once per line, and idle train leases once per
// service pattern. Through-running on a line clears its no-service warning.
func (w World) IdleLines() []gameapi.IdleLine { return w.IdleLinesAmong(w.Routes()) }

// IdleLinesAmong takes the already resolved Routes() of this world, so a
// caller that needs both builds the route list once.
func (w World) IdleLinesAmong(routes []gameapi.Route) []gameapi.IdleLine {
	result := []gameapi.IdleLine{}
	for _, co := range w.catalog.corridors {
		owned := false
		for _, p := range w.projects {
			if p.CorridorID == co.ID && p.Owner == w.company.ID && p.Ready <= w.months {
				owned = true
			}
		}
		if !owned {
			continue
		}
		active, fleet := false, 0
		for _, route := range routes {
			covers := false
			for _, id := range route.Sections {
				if id == co.ID {
					covers = true
				}
			}
			if !covers {
				continue
			}
			count := w.FleetCount(route.ID)
			fleet += count
			for _, s := range w.services {
				if s.CorridorID == route.ID && s.Frequency > 0 && count > 0 {
					active = true
				}
			}
		}
		if active {
			continue
		}
		a, _ := w.alignment(co, "")
		result = append(
			result,
			gameapi.IdleLine{
				CorridorID:  co.ID,
				FromCity:    co.FromCity,
				ToCity:      co.ToCity,
				NeedsTrains: fleet == 0,
				Maintenance: a.Cost / maintenanceDivisor,
			},
		)
	}
	for _, route := range routes {
		count := w.FleetCount(route.ID)
		if count == 0 {
			continue
		}
		active := false
		for _, s := range w.services {
			if s.CorridorID == route.ID && s.Frequency > 0 {
				active = true
			}
		}
		if active {
			continue
		}
		merged := false
		for i := range result {
			if result[i].CorridorID == route.ID {
				result[i].Leases = gameapi.Money(count) * w.leaseMonthly()
				result[i].NeedsTrains = false
				merged = true
			}
		}
		if !merged {
			result = append(
				result,
				gameapi.IdleLine{
					CorridorID: route.ID,
					FromCity:   route.Cities[0],
					ToCity:     route.Cities[len(route.Cities)-1],
					Leases:     gameapi.Money(count) * w.leaseMonthly(),
				},
			)
		}
	}
	return result
}
