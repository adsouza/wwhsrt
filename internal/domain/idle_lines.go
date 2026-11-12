package domain

import "wwhsrt/pkg/gameapi"

// IdleLines reports open, company-managed lines with continuing costs but no
// accepted positive-frequency train service. Drafts cannot clear this warning.
func (w World) IdleLines() []gameapi.IdleLine {
	result := []gameapi.IdleLine{}
	for _, c := range w.catalog.corridors {
		owned := false
		for _, p := range w.projects {
			if p.CorridorID == c.ID && p.Owner == w.company.ID && p.Ready <= w.months {
				owned = true
			}
		}
		fleet := w.FleetCount(c.ID)
		if !owned && (!c.Existing || fleet == 0) {
			continue
		}
		active := false
		for _, s := range w.services {
			if s.CorridorID == c.ID && s.Frequency > 0 {
				active = true
			}
		}
		if active && fleet > 0 {
			continue
		}
		line := gameapi.IdleLine{CorridorID: c.ID, FromCity: c.FromCity, ToCity: c.ToCity, NeedsTrains: fleet == 0, Leases: gameapi.Money(fleet) * LeaseMonthly}
		if owned {
			a, _ := w.alignment(c, "")
			line.Maintenance = a.Cost / maintenanceDivisor
		}
		if line.Maintenance+line.Leases > 0 {
			result = append(result, line)
		}
	}
	return result
}
