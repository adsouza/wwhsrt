package ui

import (
	"context"
	"maps"
	"slices"
	"time"

	"wwhsrt/pkg/gameapi"
)

// A route earns a different monthly return in a 28-, 29-, 30- or 31-day month.
// Keep at most one score per route and month length within this campaign.
type routeMonth struct {
	id   string
	days int
}

type scoreInputs struct {
	fare        gameapi.Money
	replacement string
	trains      int
	deficit     bool
}

type routeScore struct {
	inputs           scoreInputs
	best             serviceOption
	independent      bool
	timetableVersion uint64
}

// ServiceRanking retains the best feasible schedule (including a negative search
// result) for each route examined so far. Completed rankings are read-only.
type ServiceRanking struct {
	scores           map[routeMonth]routeScore
	openProjects     []gameapi.Project
	openStations     []gameapi.Station
	trainFamily      gameapi.TrainFamily
	timetable        []gameapi.Service
	timetableVersion uint64
}

func (old ServiceRanking) forFrame(f gameapi.Frame) ServiceRanking {
	next := ServiceRanking{
		trainFamily:      f.TrainFamily,
		timetable:        slices.Clone(f.Services),
		timetableVersion: old.timetableVersion,
	}
	if !slices.Equal(old.timetable, next.timetable) {
		next.timetableVersion++
	}
	// Committing future construction affects the budget and the prospect board.
	// Existing trains still use the same stations and alignments until opening.
	for _, p := range f.Projects {
		if p.Ready <= f.MonthsResolved {
			next.openProjects = append(next.openProjects, p)
		}
	}
	for _, s := range f.Stations {
		if s.Open {
			next.openStations = append(next.openStations, s)
		}
	}
	if old.scores != nil && old.trainFamily == next.trainFamily && slices.Equal(old.openProjects, next.openProjects) && slices.Equal(old.openStations, next.openStations) {
		next.scores = maps.Clone(old.scores)
	} else {
		next.scores = make(map[routeMonth]routeScore)
	}
	return next
}

// Recommend reuses valid route scores, then selects the highest gain in the
// first currently eligible priority group. The returned ranking owns its map,
// so an interrupted worker can never overwrite a later search.
func (old ServiceRanking) Recommend(ctx context.Context, f gameapi.Frame, batch gameapi.ServiceComparison) (gameapi.ForecastRequest, ServiceRanking) {
	next := old.forFrame(f)
	leases := make(map[string]int)
	for _, train := range f.Fleet {
		leases[train.CorridorID]++
	}
	days := time.Date(f.Month.Year, time.Month(f.Month.Month)+1, 0, 0, 0, 0, 0, time.UTC).Day()
	for _, group := range ServiceCandidates(f) {
		var best serviceOption
		for _, route := range group {
			if ctx.Err() != nil {
				return gameapi.ForecastRequest{}, old
			}
			var draft State
			draftServiceRoute(&draft, f.Services, route)
			inputs := scoreInputs{
				fare:        draft.Fare,
				replacement: draft.ReplaceServiceID,
				trains:      leases[route.ID] + leases[draft.ReplaceServiceID],
				deficit:     f.Player.Cash < 0,
			}
			key := routeMonth{id: route.ID, days: days}
			score, found := next.scores[key]
			sameInputs := found && score.inputs == inputs
			if !sameInputs || score.timetableVersion != next.timetableVersion {
				independent := draft.ReplaceServiceID == "" && batch.Independent != nil && batch.Independent(route)
				// A changed timetable can keep this score only if both comparisons
				// are equivalent to running the candidate route on its own.
				if !sameInputs || !score.independent || !independent {
					q := gameapi.ForecastRequest{CorridorID: route.ID, Fare: draft.Fare, ReplaceServiceID: draft.ReplaceServiceID}
					score.best = bestServiceSchedule(ctx, q, inputs.deficit, batch.Forecast)
					if ctx.Err() != nil {
						return gameapi.ForecastRequest{}, old
					}
				}
				score.inputs, score.independent, score.timetableVersion = inputs, independent, next.timetableVersion
				next.scores[key] = score
			}
			if score.best.betterThan(best) {
				best = score.best
			}
		}
		if best.plan.CorridorID != "" {
			return best.plan, next
		}
	}
	return gameapi.ForecastRequest{}, next
}

func bestServiceSchedule(ctx context.Context, q gameapi.ForecastRequest, deficit bool, forecast func(gameapi.ForecastRequest) (gameapi.Forecast, error)) serviceOption {
	var best serviceOption
	for q.Frequency = 1; q.Frequency <= gameapi.MaxFrequency; q.Frequency++ {
		if ctx.Err() != nil {
			break
		}
		r, err := forecast(q)
		if err != nil || !r.Open || r.DeliveredFrequency == 0 || r.CompanyChange <= 0 || (deficit && r.FleetNeeded > r.FleetAvailable) {
			continue
		}
		option := serviceOption{plan: q, gain: r.CompanyChange, leases: r.AdditionalLeases}
		if option.betterThan(best) {
			best = option
		}
	}
	return best
}
