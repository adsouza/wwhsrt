package domain

import "wwhsrt/pkg/gameapi"

// Count requested paths/calls, not passenger seats or last month's allocation.
// This is also the validator's input, so explanations cannot drift from its rule.
func (w World) ownedCapacityUsage(plans []gameapi.Service, draftID string) ([]gameapi.CapacityUsage, error) {
	sections := map[string][]gameapi.CapacityService{}
	stations := map[gameapi.CityID][]gameapi.CapacityService{}
	draftSections := map[string]bool{}
	draftStations := map[gameapi.CityID]bool{}
	for _, s := range sortedServices(plans) {
		r, ok := w.route(s.CorridorID)
		if !ok || !r.Open {
			return nil, errCode("line_not_open")
		}
		used := map[string]bool{}
		for _, section := range r.Sections {
			for _, id := range w.catalog.resources(section) {
				if used[id] {
					continue
				}
				used[id] = true
				if s.CorridorID == draftID {
					draftSections[id] = true
				}
				if s.Frequency > 0 {
					sections[id] = append(
						sections[id],
						gameapi.CapacityService{
							ServiceID: s.CorridorID,
							Frequency: s.Frequency,
							Usage:     s.Frequency,
							Draft:     s.CorridorID == draftID,
						},
					)
				}
			}
		}
		for _, city := range r.Stops {
			if s.CorridorID == draftID {
				draftStations[city] = true
			}
			if s.Frequency > 0 {
				stations[city] = append(
					stations[city],
					gameapi.CapacityService{
						ServiceID: s.CorridorID,
						Frequency: s.Frequency,
						Usage:     s.Frequency * 2,
						Draft:     s.CorridorID == draftID,
					},
				)
			}
		}
	}
	out := []gameapi.CapacityUsage{}
	finish := func(c gameapi.CapacityUsage, units int) {
		for _, s := range c.Services {
			c.Requested += s.Usage
			if s.Draft {
				c.DraftUse += s.Usage
			}
		}
		c.OtherRequested = c.Requested - c.DraftUse
		c.Excess = max(0, c.Requested-c.Capacity)
		c.MaxFrequency = max(0, (c.Capacity-c.OtherRequested)/units)
		out = append(out, c)
	}
	// Preserve validation priority and deterministic order: sections, then stations.
	for _, co := range w.catalog.corridors {
		if !co.Existing && (len(sections[co.ID]) > 0 || draftSections[co.ID]) {
			finish(
				gameapi.CapacityUsage{
					Kind:       "track",
					CorridorID: co.ID,
					Capacity:   w.sectionLimit(co.ID),
					Draft:      draftSections[co.ID],
					Services:   sections[co.ID],
				},
				1,
			)
		}
	}
	for _, station := range w.Stations() {
		if !station.Public && (len(stations[station.CityID]) > 0 || draftStations[station.CityID]) {
			finish(
				gameapi.CapacityUsage{
					Kind:     "station",
					CityID:   station.CityID,
					Capacity: w.catalog.network.StationCalls,
					Draft:    draftStations[station.CityID],
					Services: stations[station.CityID],
				},
				2,
			)
		}
	}
	return out, nil
}
func capacityError(usage []gameapi.CapacityUsage) error {
	failed := []gameapi.CapacityUsage{}
	code := ""
	for _, c := range usage {
		if c.Excess > 0 {
			failed = append(failed, c)
			if code == "" {
				code = c.Kind + "_capacity"
			}
		}
	}
	if code == "" {
		return nil
	}
	return &gameapi.Error{Code: code, CapacityUsage: failed}
}
