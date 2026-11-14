// balancecheck compares each direct opportunity independently against the same
// fresh-company baseline. It does not read or write a player's save.
package main

import (
	"crypto/sha256"
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"wwhsrt/assets"
	"wwhsrt/internal/domain"
	"wwhsrt/pkg/gameapi"
)

type plan struct {
	Frequency int              `json:"frequency"`
	Fare      gameapi.Money    `json:"fare_cents"`
	Forecast  gameapi.Forecast `json:"forecast"`
}
type row struct {
	ID                                 string `json:"id"`
	From, To                           gameapi.CityID
	Legacy, RebalancedFixed, Suggested plan
}

func main() {
	original := flag.Bool("original-catalog", false, "reproduce the 146-route pre-repair calibration")
	output := flag.String("out", "", "optional JSON artifact path")
	flag.Parse()
	var raw struct {
		Corridors []gameapi.Corridor
		Network   gameapi.NetworkRules
	}
	source := assets.PublicNetwork
	if *original {
		source = assets.PublicNetworkV1
	}
	var rules gameapi.EconomyRules
	if e := json.Unmarshal(source, &raw); e != nil {
		panic(e)
	}
	if e := json.Unmarshal(assets.Economy, &rules); e != nil {
		panic(e)
	}
	c, e := domain.NewCatalog(raw.Corridors)
	if e != nil {
		panic(e)
	}
	c, e = c.WithNetwork(raw.Network)
	if e != nil {
		panic(e)
	}
	old := domain.New(c)
	modern, e := c.WithEconomy(rules)
	if e != nil {
		panic(e)
	}
	next := domain.New(modern)
	routes := map[string]gameapi.Route{}
	for _, r := range next.Routes() {
		routes[r.ID] = r
	}
	run := func(w domain.World, id string, n int, fare gameapi.Money) plan {
		f, e := w.Forecast(gameapi.ForecastRequest{CorridorID: id, Frequency: n, Fare: fare})
		if e != nil {
			panic(e)
		}
		return plan{n, fare, f}
	}
	rows := []row{}
	before, fixed, after := 0, 0, 0
	for _, co := range raw.Corridors {
		if !co.Existing {
			continue
		}
		r := routes[co.ID]
		a, b, d := run(old, co.ID, 4, 9000), run(next, co.ID, 4, 9000), run(next, co.ID, r.SuggestedFrequency, r.SuggestedFare)
		if a.Forecast.Contribution > 0 {
			before++
		}
		if b.Forecast.Contribution > 0 {
			fixed++
		}
		if d.Forecast.Contribution > 0 {
			after++
		}
		if d.Forecast.Costs != d.Forecast.RunningCosts+d.Forecast.AccessCosts+d.Forecast.HandlingCosts+d.Forecast.LeaseCosts+d.Forecast.MaintenanceCosts {
			panic("cost reconciliation")
		}
		rows = append(rows, row{co.ID, co.FromCity, co.ToCity, a, b, d})
	}
	if len(rows) < 146 || after < 90 || (*original && (len(rows) != 146 || before != 14 || after > 125)) {
		panic(fmt.Sprintf("balance guard: %d routes, before %d, after %d", len(rows), before, after))
	}
	result := struct {
		CatalogSHA, EconomySHA                                                           string
		Rules                                                                            gameapi.EconomyRules
		ExistingRoutes, LegacyProfitable, RebalancedFixedProfitable, SuggestedProfitable int
		Scope                                                                            string
		Rows                                                                             []row
	}{fmt.Sprintf("%x", sha256.Sum256(source)), fmt.Sprintf("%x", sha256.Sum256(assets.Economy)), rules, len(rows), before, fixed, after, "Each candidate is forecast independently alongside the starter service in January 2029; not a combined portfolio or empirical ridership certification.", rows}
	b, e := json.MarshalIndent(result, "", "  ")
	if e != nil {
		panic(e)
	}
	b = append(b, '\n')
	if *output != "" {
		if e = os.WriteFile(*output, b, 0644); e != nil {
			panic(e)
		}
	} else {
		os.Stdout.Write(b)
	}
	fmt.Fprintf(os.Stderr, "Existing routes %d; profitable legacy %d, new rules with old defaults %d, suggested plans %d\n", len(rows), before, fixed, after)
}
