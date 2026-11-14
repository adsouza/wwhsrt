package ui

import (
	"fmt"
	"wwhsrt/pkg/gameapi"
)

// CapacityDetails formats authoritative preview/validation output. It does not
// reconstruct plans, capacity limits, replacement rules or maximum frequency.
func CapacityDetails(f gameapi.Frame, locale string, usage []gameapi.CapacityUsage) []Detail {
	t := func(k string) string { return Text(locale, k) }
	out := []Detail{}
	for _, c := range usage {
		name := t(string(c.CityID))
		unit := "capacityCalls"
		if c.Kind == "track" {
			unit = "capacityTrips"
			name = c.CorridorID
			for _, co := range f.Corridors {
				if co.ID == c.CorridorID {
					name = t(string(co.FromCity)) + " – " + t(string(co.ToCity))
				}
			}
		}
		out = append(out, Detail{name, fmt.Sprintf(t("capacityUsage"), c.Requested, c.Capacity, t(unit))})
		if c.Excess <= 0 {
			continue
		}
		out = append(out, Detail{t("capacityOverBy"), fmt.Sprintf("%d %s", c.Excess, t(unit))})
		out = append(out, Detail{t("capacityOtherServices"), fmt.Sprintf("%d %s", c.OtherRequested, t(unit))})
		for _, s := range c.Services {
			name := s.ServiceID
			if r, ok := ServiceRoute(f, s.ServiceID); ok {
				name = RouteName(r, locale)
			}
			if s.Draft {
				name = t("capacityThisDraft") + " · " + name
			}
			out = append(out, Detail{name, fmt.Sprintf("%d %s", s.Usage, t(unit))})
		}
		out = append(out, Detail{t("capacityDraftLimit"), fmt.Sprintf(t("capacityMaxDraft"), c.MaxFrequency)})
		key := "capacityTrackWhy"
		if c.Kind == "station" {
			key = "capacityStationWhy"
		}
		out = append(out, Detail{t("capacityWhy"), t(key)})
	}
	return out
}

// Keep the tightest resource visible; the complete explanation is on demand.
func CapacitySummary(f gameapi.Frame, locale string, usage []gameapi.CapacityUsage) []Detail {
	if len(usage) == 0 {
		return nil
	}
	best := usage[0]
	for _, c := range usage[1:] {
		if c.Requested*best.Capacity > best.Requested*c.Capacity {
			best = c
		}
	}
	rows := CapacityDetails(f, locale, []gameapi.CapacityUsage{best})
	if len(rows) > 0 {
		return rows[:1]
	}
	return nil
}
