// bridgecheck settles actual shipped connector projects against a public-rail
// feeder baseline. It never reads or writes a player save.
package main

import (
	"crypto/sha256"
	"encoding/json"
	"flag"
	"fmt"
	"math"
	"os"
	"wwhsrt/assets"
	"wwhsrt/internal/domain"
	"wwhsrt/pkg/gameapi"
)

type catalog struct {
	Cities    []gameapi.City
	Corridors []gameapi.Corridor
	Network   gameapi.NetworkRules
}

type metrics struct {
	Replaces                   string `json:"replaces,omitempty"`
	BaselinePlans              []gameapi.Service
	BaselineFleet              []gameapi.FleetLease
	NewPlans                   []gameapi.Service
	NewFleet                   []gameapi.FleetLease
	Route                      string        `json:"route"`
	Frequency                  int           `json:"frequency"`
	Fare                       gameapi.Money `json:"fare"`
	Cost, Grant, Debt          gameapi.Money
	MonthlyIncrement           []gameapi.Money
	MonthlyNewServicePortfolio []gameapi.Money
	MonthlyFeederPortfolio     []gameapi.Money
	AverageIncrement           gameapi.Money
	MinimumIncrement           gameapi.Money
	Forecast                   gameapi.Forecast
	Error                      string `json:"error,omitempty"`
}

func must(e error) {
	if e != nil {
		panic(e)
	}
}

func clone() catalog { var c catalog; must(json.Unmarshal(assets.PublicNetwork, &c)); return c }

func change(w domain.World, c gameapi.Command) domain.World {
	c.Revision = w.Revision()
	n, e := w.Change(c)
	must(e)
	return n
}

func publicRoute(c catalog, a, b, id string) gameapi.Route {
	dist := map[string]int{a: 0}
	done := map[string]bool{}
	prev := map[string]string{}
	edge := map[string]string{}
	for {
		n := ""
		cost := math.MaxInt
		for v, d := range dist {
			if !done[v] && (d < cost || (d == cost && v < n)) {
				n = v
				cost = d
			}
		}
		if n == "" {
			return gameapi.Route{}
		}
		if n == b {
			break
		}
		done[n] = true
		for _, co := range c.Corridors {
			if !co.Existing {
				continue
			}
			other := ""
			if string(co.FromCity) == n {
				other = string(co.ToCity)
			} else if string(co.ToCity) == n {
				other = string(co.FromCity)
			}
			if other == "" {
				continue
			}
			v := cost + co.Alignments[0].Minutes
			old, ok := dist[other]
			if !ok || v < old {
				dist[other] = v
				prev[other] = n
				edge[other] = co.ID
			}
		}
	}
	nodes := []string{b}
	sections := []string{}
	for n := b; n != a; n = prev[n] {
		sections = append(sections, edge[n])
		nodes = append(nodes, prev[n])
	}
	r := gameapi.Route{ID: id}
	for i := len(nodes) - 1; i >= 0; i-- {
		r.Cities = append(r.Cities, gameapi.CityID(nodes[i]))
		r.Stops = append(r.Stops, gameapi.CityID(nodes[i]))
	}
	for i := len(sections) - 1; i >= 0; i-- {
		r.Sections = append(r.Sections, sections[i])
	}
	if len(r.Sections) == 1 {
		r.ID = r.Sections[0]
	}
	return r
}

func best(w domain.World, id string) (gameapi.ForecastRequest, gameapi.Forecast) {
	routes := map[string]gameapi.Route{}
	for _, r := range w.Routes() {
		routes[r.ID] = r
	}
	r := routes[id]
	if r.ID == "" {
		panic("no route")
	}
	sources := []string{""}
	contains := func(xs []string, x string) bool {
		for _, v := range xs {
			if v == x {
				return true
			}
		}
		return false
	}
	for _, plan := range w.Services() {
		source := routes[plan.CorridorID]
		if plan.Frequency == 0 || source.ID == r.ID || len(source.Sections) >= len(r.Sections) {
			continue
		}
		valid := true
		for _, v := range source.Sections {
			valid = valid && contains(r.Sections, v)
		}
		for _, v := range source.Stops {
			found := false
			for _, a := range r.Stops {
				found = found || a == v
			}
			valid = valid && found
		}
		if valid {
			sources = append(sources, source.ID)
		}
	}
	q := gameapi.ForecastRequest{CorridorID: id, Frequency: 1, Fare: 1000}
	var fc gameapi.Forecast
	score := gameapi.Money(math.MinInt64)
	fares := map[gameapi.Money]bool{}
	for _, v := range []gameapi.Money{1000, 2000, 3000, 4500, 6500, 9000, 12000, 16000, 22000, 30000} {
		fares[v] = true
	}
	for _, pct := range []int{80, 110, 150, 200} {
		fares[max(
			gameapi.Money(1000),
			min(gameapi.Money(30000), gameapi.Money(400+r.KM*14)*gameapi.Money(pct)/100),
		)] = true
	}
	for _, source := range sources {
		for _, n := range []int{1, 2, 4, 6} {
			for fare := range fares {
				p := gameapi.ForecastRequest{CorridorID: id, Frequency: n, Fare: fare, ReplaceServiceID: source}
				f, e := w.Forecast(p)
				if e != nil {
					continue
				}
				if f.CompanyChange > score ||
					(f.CompanyChange == score && (fare < q.Fare || fare == q.Fare && n < q.Frequency)) {
					score = f.CompanyChange
					q = p
					fc = f
				}
			}
		}
	}
	if score == gameapi.Money(math.MinInt64) {
		panic("no feasible draft")
	}
	return q, fc
}

func apply(w domain.World, q gameapi.ForecastRequest, fc gameapi.Forecast) domain.World {
	if q.ReplaceServiceID != "" {
		return change(
			w,
			gameapi.Command{
				Kind:             gameapi.ExtendService,
				CorridorID:       q.CorridorID,
				ReplaceServiceID: q.ReplaceServiceID,
				Frequency:        q.Frequency,
				Fare:             q.Fare,
				LeaseCount:       fc.AdditionalLeases,
			},
		)
	}
	for i := 0; i < fc.AdditionalLeases; i++ {
		w = change(w, gameapi.Command{Kind: gameapi.LeaseTrain, CorridorID: q.CorridorID})
	}
	return change(
		w,
		gameapi.Command{Kind: gameapi.SetService, CorridorID: q.CorridorID, Frequency: q.Frequency, Fare: q.Fare},
	)
}

func mirror(w, plans domain.World) domain.World {
	for _, s := range w.Services() {
		w = change(w, gameapi.Command{Kind: gameapi.SetService, CorridorID: s.CorridorID, Frequency: 0, Fare: s.Fare})
		for w.FleetCount(s.CorridorID) > 0 {
			w = change(w, gameapi.Command{Kind: gameapi.ReturnTrain, CorridorID: s.CorridorID})
		}
	}
	for _, s := range plans.Services() {
		for i := 0; i < plans.FleetCount(s.CorridorID); i++ {
			w = change(w, gameapi.Command{Kind: gameapi.LeaseTrain, CorridorID: s.CorridorID})
		}
		w = change(
			w,
			gameapi.Command{Kind: gameapi.SetService, CorridorID: s.CorridorID, Frequency: s.Frequency, Fare: s.Fare},
		)
	}
	return w
}

func cash(w domain.World) gameapi.Money {
	rs := w.Reports()
	r := rs[len(rs)-1]
	return r.Revenue - r.Costs
}

func main() {
	outputPath := flag.String("out", "", "optional JSON result path")
	flag.Parse()
	type project struct {
		ID     string
		Routes []string
	}
	projects := []project{
		{
			"launch_72_tsuruga__maibara",
			[]string{"connector_kanazawa__osaka_via_maibara", "connector_kanazawa__nagoya_via_maibara"},
		},
		{
			"launch_73_valencia__camp_de_tarragona",
			[]string{"connector_valencia__barcelona", "connector_valencia__perpignan"},
		},
		{"launch_74_porto__vigo", []string{"connector_lisbon__pontevedra"}},
	}
	results := []map[string]any{}
	for _, project := range projects {
		fmt.Fprintln(os.Stderr, "checking shipped project", project.ID)
		c := clone()
		feeds := [][]string{}
		for i, id := range project.Routes {
			var r gameapi.Route
			for _, v := range c.Network.Routes {
				if v.ID == id {
					r = v
				}
			}
			if r.ID == "" {
				panic("missing service")
			}
			ids := []string{}
			seen := map[string]bool{}
			for j := 0; j < len(r.Cities); j++ {
				for k := j + 1; k < len(r.Cities); k++ {
					f := publicRoute(
						c,
						string(r.Cities[j]),
						string(r.Cities[k]),
						fmt.Sprintf("validation_base_%d_%d_%d", i, j, k),
					)
					if f.ID == "" || seen[f.ID] {
						continue
					}
					seen[f.ID] = true
					if len(f.Sections) > 1 {
						c.Network.Routes = append(c.Network.Routes, f)
					}
					ids = append(ids, f.ID)
				}
			}
			feeds = append(feeds, ids)
		}
		cat, e := domain.NewCatalog(c.Corridors)
		must(e)
		cat, e = cat.WithNetwork(c.Network)
		must(e)
		var econ gameapi.EconomyRules
		must(json.Unmarshal(assets.Economy, &econ))
		cat, e = cat.WithEconomy(econ)
		must(e)
		var co gameapi.Corridor
		for _, r := range cat.Corridors() {
			if r.ID == project.ID {
				co = r
			}
		}
		a := co.Alignments[0]
		base, built := domain.New(cat), domain.New(cat)
		debt := max(gameapi.Money(0), a.Cost-a.Grant-built.Company().Cash)
		if debt > 0 {
			built = change(built, gameapi.Command{Kind: gameapi.TakeLoan, Amount: debt})
		}
		built = change(built, gameapi.Command{Kind: gameapi.BuildLine, CorridorID: co.ID, AlignmentID: a.ID})
		for month := 0; month < a.Months; month++ {
			base = change(base, gameapi.Command{Kind: gameapi.AdvanceMonth})
			built = change(built, gameapi.Command{Kind: gameapi.AdvanceMonth})
		}
		trials := []metrics{}
		bestScore := gameapi.Money(math.MinInt64)
		var winner metrics
		for i, id := range project.Routes {
			b, n := base, built
			for _, f := range feeds[i] {
				q, fc := best(b, f)
				if fc.CompanyChange > 0 {
					b = apply(b, q, fc)
				}
			}
			n = mirror(n, b)
			q, fc := best(n, id)
			n = apply(n, q, fc)
			m := metrics{
				Replaces:         q.ReplaceServiceID,
				BaselinePlans:    b.Services(),
				BaselineFleet:    b.Fleet(),
				NewPlans:         n.Services(),
				NewFleet:         n.Fleet(),
				Route:            id,
				Frequency:        q.Frequency,
				Fare:             q.Fare,
				Cost:             a.Cost,
				Grant:            a.Grant,
				Debt:             n.Debt(),
				Forecast:         fc,
				MinimumIncrement: gameapi.Money(math.MaxInt64),
			}
			var total gameapi.Money
			for month := 0; month < 12; month++ {
				b = change(b, gameapi.Command{Kind: gameapi.AdvanceMonth})
				n = change(n, gameapi.Command{Kind: gameapi.AdvanceMonth})
				delta := cash(n) - cash(b)
				m.MonthlyIncrement = append(m.MonthlyIncrement, delta)
				m.MonthlyNewServicePortfolio = append(m.MonthlyNewServicePortfolio, cash(n))
				m.MonthlyFeederPortfolio = append(m.MonthlyFeederPortfolio, cash(b))
				total += delta
				m.MinimumIncrement = min(m.MinimumIncrement, delta)
			}
			m.AverageIncrement = total / 12
			trials = append(trials, m)
			if m.AverageIncrement > bestScore {
				bestScore = m.AverageIncrement
				winner = m
			}
		}
		results = append(
			results,
			map[string]any{"project": project.ID, "km": a.KM, "months": a.Months, "best": winner, "trials": trials},
		)
		fmt.Fprintln(
			os.Stderr,
			project.ID,
			"mean incremental cents",
			winner.AverageIncrement,
			"worst",
			winner.MinimumIncrement,
		)
	}
	output := map[string]any{
		"catalog_sha256": fmt.Sprintf("%x", sha256.Sum256(assets.PublicNetwork)),
		"economy_sha256": fmt.Sprintf("%x", sha256.Sum256(assets.Economy)),
		"method": "Actual shipped costs, durations, grant, financing and service patterns. " +
			"Default starter operated during real construction months. " +
			"Baseline existing public services optimized first among all service-stop pairs, " +
			"mirrored into project world; add or extend one connector through service. " +
			"Fixed first-post-opening fare/frequency chosen from bounded grid, then settle 12 months. " +
			"Monthly cash deltas include maintenance and actual loan interest, " +
			"exclude upfront capital/principal repayment and competition. A greedy portfolio screen, " +
			"not an optimum or campaign NPV.",
		"projects": results,
	}
	b, e := json.MarshalIndent(output, "", "  ")
	must(e)
	if *outputPath != "" {
		must(os.WriteFile(*outputPath, append(b, '\n'), 0644))
	} else {
		_, e = os.Stdout.Write(append(b, '\n'))
		must(e)
	}
	for _, r := range results {
		m := r["best"].(metrics)
		if m.MinimumIncrement <= 0 {
			panic("connector has a loss-making month in this qualified portfolio: " + r["project"].(string))
		}
	}
}
