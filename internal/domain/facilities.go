package domain

import (
	"sort"
	"wwhsrt/pkg/gameapi"
)

const trainSeats = 400
const trainDayMinutes = 960
const roundTripTurnaround = 40
const runningPerKM gameapi.Money = 1200
const publicAccessPerKM gameapi.Money = 500
const stationPerJourney gameapi.Money = 50

// Service limits shared by command validation, forecasts and save restoration.
// Change them here only. Fleet size is deliberately unlimited: a lease needs
// only nonnegative cash, and each train then costs its monthly lease.
const MaxFrequency = gameapi.MaxFrequency
const MinFare gameapi.Money = 1000
const MaxFare gameapi.Money = 30000

// This slice has one train family, so every persisted lease has this family.
func TrainFamily() gameapi.TrainFamily {
	return gameapi.TrainFamily{
		ID:                  "swift-400",
		NameKey:             "swift400",
		Seats:               trainSeats,
		LeaseMonthly:        LeaseMonthly,
		RunningPerKM:        runningPerKM,
		PublicAccessPerKM:   publicAccessPerKM,
		StationPerJourney:   stationPerJourney,
		DailyMinutes:        trainDayMinutes,
		RoundTripTurnaround: roundTripTurnaround,
	}
}

// The station lifecycle is completely determined by the pinned starter network
// and committed projects. Deriving it keeps existing saves and balances intact.
func (w World) Stations() []gameapi.Station {
	byCity := map[gameapi.CityID]gameapi.Station{}
	for _, c := range w.catalog.corridors {
		if !c.Existing || (c.Retired && !c.ThroughOnly && !w.offered(c.ID)) {
			continue
		}
		for _, id := range []gameapi.CityID{c.FromCity, c.ToCity} {
			byCity[id] = gameapi.Station{CityID: id, Public: true, Open: true}
		}
	}
	contracts := make(map[string]gameapi.Project, len(w.projects))
	for _, p := range w.projects {
		contracts[p.CorridorID] = p
	}
	for _, c := range w.catalog.corridors {
		p, owned := contracts[c.ProjectID()]
		if !owned {
			continue
		}
		for _, id := range []gameapi.CityID{c.FromCity, c.ToCity} {
			old, exists := byCity[id]
			if exists && (old.Public || old.Ready <= p.Ready) {
				continue
			}
			byCity[id] = gameapi.Station{CityID: id, Owner: p.Owner, Ready: p.Ready, Open: p.Ready <= w.months}
		}
	}
	result := make([]gameapi.Station, 0, len(byCity))
	for _, station := range byCity {
		result = append(result, station)
	}
	sort.Slice(result, func(i, j int) bool { return result[i].CityID < result[j].CityID })
	return result
}
