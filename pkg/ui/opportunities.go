package ui

import (
	"sort"

	"wwhsrt/pkg/gameapi"
)

type Prospect struct {
	ID        string         `json:"id"`
	From      gameapi.CityID `json:"from"`
	To        gameapi.CityID `json:"to"`
	Alignment string         `json:"alignment"`
	Cash      gameapi.Money  `json:"cash"`
}

func cheapestAlignment(c gameapi.Corridor) gameapi.Alignment {
	best := c.Alignments[0]
	for _, a := range c.Alignments[1:] {
		if a.Cost-a.Grant < best.Cost-best.Grant {
			best = a
		}
	}
	return best
}

func RankedProspects(f gameapi.Frame) []Prospect {
	rows := []Prospect{}
	for _, c := range ConstructionOpportunities(f) {
		if len(c.Alignments) > 0 {
			a := cheapestAlignment(c)
			rows = append(rows, Prospect{c.ID, c.FromCity, c.ToCity, a.ID, a.Cost - a.Grant})
		}
	}
	sort.SliceStable(rows, func(i, j int) bool { return rows[i].Cash < rows[j].Cash })
	return rows
}

// An accepted through service operates every section along its route.
func OperatingSections(f gameapi.Frame) map[string]bool {
	sections := map[string]bool{}
	for _, s := range f.Services {
		if s.Frequency > 0 {
			if r, ok := ServiceRoute(f, s.CorridorID); ok {
				for _, id := range r.Sections {
					sections[id] = true
				}
			}
		}
	}
	return sections
}

// Start with an idle asset already costing the player money, then an unserved
// direct line. The player can switch to any open pattern in the planner.
func NextService(f gameapi.Frame) string {
	if f.Player.Cash < 0 {
		// A company in deficit can run its leased trains but cannot add leases.
		for _, train := range f.Fleet {
			if Open(f, train.CorridorID) {
				return train.CorridorID
			}
		}
		return ""
	}
	for _, line := range f.IdleLines {
		if Open(f, line.CorridorID) {
			return line.CorridorID
		}
	}
	running := OperatingSections(f)
	for _, r := range f.Routes {
		if r.Open && len(r.Sections) == 1 && !running[r.Sections[0]] {
			return r.ID
		}
	}
	for _, r := range f.Routes {
		if r.Open {
			return r.ID
		}
	}
	return ""
}

// Compare the same company's monthly operating result at a fixed fare. The
// forecast includes new leases, capacity and revenue displaced on other lines.
func BestFrequency(f gameapi.Frame, q gameapi.ForecastRequest, forecast func(gameapi.ForecastRequest) (gameapi.Forecast, error)) (int, bool) {
	best, found := 0, false
	var profit gameapi.Money
	for n := 0; n <= gameapi.MaxFrequency; n++ {
		q.Frequency = n
		result, err := forecast(q)
		if err != nil || !result.Open || (result.FleetNeeded > result.FleetAvailable && f.Player.Cash < 0) {
			continue
		}
		if !found || result.CompanyChange > profit {
			best, profit, found = n, result.CompanyChange, true
		}
	}
	return best, found
}
