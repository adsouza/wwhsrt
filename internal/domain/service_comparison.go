package domain

import "wwhsrt/pkg/gameapi"

// A disconnected, two-stop draft has one fare and one seat pool in each
// direction. Rationing between business and leisure cannot change its total
// ticket sales. Its addition must also leave the accepted network's earnings
// alone: check physical resources and transfers, including shared track aliases.
//
// The allocator rounds shares across the entire company. An unrelated binding
// constraint can affect rounding on competing itineraries, so only reuse an
// isolated score if the existing services are separate two-stop lines, or if
// their entire timetable and first-choice passenger demand can be met.
func (w World) independentServiceCheck(allDemandMet func() bool) func(gameapi.Route) bool {
	stops := map[gameapi.CityID]bool{}
	tracks := map[string]bool{}
	touches := func(r gameapi.Route) bool {
		for _, id := range r.Stops {
			if stops[id] {
				return true
			}
		}
		for _, section := range r.Sections {
			for _, id := range w.catalog.resources(section) {
				if tracks[id] {
					return true
				}
			}
		}
		return false
	}
	simple := true
	for _, s := range w.services {
		if s.Frequency <= 0 {
			continue
		}
		r, ok := w.route(s.CorridorID)
		if !ok {
			return nil
		}
		if len(r.Stops) != 2 || touches(r) {
			simple = false
		}
		for _, id := range r.Stops {
			stops[id] = true
		}
		for _, section := range r.Sections {
			for _, id := range w.catalog.resources(section) {
				tracks[id] = true
			}
		}
	}
	// An active route always touches itself. Evaluate the more expensive demand
	// check only when a detached draft could otherwise reuse a score.
	var checked, unconstrained bool
	return func(r gameapi.Route) bool {
		if len(r.Stops) != 2 || touches(r) {
			return false
		}
		if simple {
			return true
		}
		if !checked {
			unconstrained = allDemandMet()
			checked = true
		}
		return unconstrained
	}
}
