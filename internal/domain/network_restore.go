package domain

import "wwhsrt/pkg/gameapi"

func (w World) validNetworkReport(report gameapi.Report, month int) bool {
	w = w.atMonth(month)
	plans := []gameapi.Service{}
	needed := 0
	for _, run := range report.Services {
		route, ok := w.route(run.CorridorID)
		if !ok || !route.Open {
			return false
		}
		plans = append(plans, gameapi.Service{CorridorID: run.CorridorID, Frequency: run.RequestedFrequency, Fare: run.Fare})
		needed += required(run.RequestedFrequency, route.Minutes)
	}
	if w.validateOwnedCapacity(plans) != nil || report.LeaseCosts%w.leaseMonthly() != 0 || report.LeaseCosts < gameapi.Money(needed)*w.leaseMonthly() {
		return false
	}
	network := w.simulate(plans, month)
	if network.Revenue != report.Revenue || network.Carried != report.Passengers || len(network.Services) != len(report.Services) {
		return false
	}
	var running, access, handling gameapi.Money
	for _, s := range network.Services {
		running += s.Running
		access += s.Access
		handling += s.Handling
		found := false
		for _, run := range report.Services {
			if run.CorridorID == s.ID {
				found = true
				if run.Frequency != s.Delivered || run.Passengers != s.Boardings || run.Revenue != s.Revenue {
					return false
				}
			}
		}
		if !found {
			return false
		}
	}
	return running == report.RunningCosts && access == report.AccessCosts && handling == report.StationCosts
}

// Extend is one atomic replacement: source trains move once, only the explicitly
// priced additional leases are ordered, and failure preserves both services.
func (w *World) extendService(c gameapi.Command) error {
	if c.Frequency < 0 || c.Frequency > MaxFrequency || c.Fare < MinFare || c.Fare > MaxFare || c.LeaseCount < 0 {
		return errCode("invalid_service")
	}
	if !w.validExtension(c.CorridorID, c.ReplaceServiceID) {
		return errCode("invalid_extension")
	}
	target, _ := w.route(c.CorridorID)
	available := w.FleetCount(c.ReplaceServiceID) + w.FleetCount(target.ID)
	if c.LeaseCount != max(0, required(c.Frequency, target.Minutes)-available) {
		return errCode("fleet_shortage")
	}
	if c.LeaseCount > 0 && w.company.Cash < 0 {
		return errCode("insufficient_cash")
	}
	for i := range w.services {
		if w.services[i].CorridorID == c.ReplaceServiceID {
			w.services[i].Frequency = 0
		}
	}
	for i := range w.fleet {
		if w.fleet[i].CorridorID == c.ReplaceServiceID {
			w.fleet[i].CorridorID = target.ID
		}
	}
	for i := 0; i < c.LeaseCount; i++ {
		if err := w.change(gameapi.Command{Kind: gameapi.LeaseTrain, CorridorID: target.ID}); err != nil {
			return err
		}
	}
	return w.change(gameapi.Command{Kind: gameapi.SetService, CorridorID: target.ID, Frequency: c.Frequency, Fare: c.Fare})
}

func (w World) validExtension(targetID, sourceID string) bool {
	target, ok := w.route(targetID)
	source, found := w.route(sourceID)
	if !ok || !found || !source.Open || !target.Open || target.ID == source.ID || len(target.Sections) <= len(source.Sections) {
		return false
	}
	for _, section := range source.Sections {
		matched := false
		for _, id := range target.Sections {
			if id == section {
				matched = true
			}
		}
		if !matched {
			return false
		}
	}
	for _, stop := range source.Stops {
		if !containsCity(target.Stops, stop) {
			return false
		}
	}
	active := false
	for _, s := range w.services {
		if s.CorridorID == source.ID && s.Frequency > 0 {
			active = true
		}
		if s.CorridorID == target.ID && s.Frequency > 0 {
			return false
		}
	}
	return active
}
