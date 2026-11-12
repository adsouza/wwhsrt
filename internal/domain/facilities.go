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

// This slice has one train family, so every persisted lease has this family.
func TrainFamily() gameapi.TrainFamily {
	return gameapi.TrainFamily{ID: "swift-400", NameKey: "swift400", Seats: trainSeats, LeaseMonthly: LeaseMonthly, RunningPerKM: runningPerKM, PublicAccessPerKM: publicAccessPerKM, StationPerJourney: stationPerJourney, DailyMinutes: trainDayMinutes, RoundTripTurnaround: roundTripTurnaround}
}

// The station lifecycle is completely determined by the pinned starter network
// and committed projects. Deriving it keeps existing saves and balances intact.
func (w World) Stations() []gameapi.Station {
	byCity := map[gameapi.CityID]gameapi.Station{}
	for _, c := range w.catalog.corridors {
		if !c.Existing {
			continue
		}
		for _, id := range []gameapi.CityID{c.FromCity, c.ToCity} {
			byCity[id] = gameapi.Station{CityID: id, Public: true, Open: true}
		}
	}
	for _, p := range w.projects {
		c, _ := w.catalog.find(p.CorridorID)
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
