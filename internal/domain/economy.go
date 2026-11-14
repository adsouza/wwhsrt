package domain

import (
	"math/big"
	"wwhsrt/pkg/gameapi"
)

// Economy v1 is a bounded gameplay calibration, not an empirical transport model.
// The saved cutover preserves replay of all months resolved under the old rules.
func (w World) rebalanced() bool  { return w.catalog.economy.ID != "" && w.months >= w.economySince }
func (w World) EconomySince() int { return w.economySince }
func (w World) leaseMonthly() gameapi.Money {
	if w.rebalanced() {
		return w.catalog.economy.LeaseMonthly
	} // USD 120,000 per 400-seat train/month.
	return LeaseMonthly
}
func (w World) runningRate() gameapi.Money {
	if w.rebalanced() {
		return w.catalog.economy.RunningPerKM
	}
	return runningPerKM
}
func (w World) accessRate() gameapi.Money {
	if w.rebalanced() {
		return w.catalog.economy.PublicAccessPerKM
	}
	return publicAccessPerKM
}
func (w World) TrainFamily() gameapi.TrainFamily {
	f := TrainFamily()
	f.LeaseMonthly = w.leaseMonthly()
	f.RunningPerKM = w.runningRate()
	f.PublicAccessPerKM = w.accessRate()
	return f
}
func (c Catalog) economyTrips(base int) int {
	e := c.economy
	return min(e.DailyTripCap, base*e.TripScale+min(e.CatchmentBonusCap, base*e.TripScale))
}
func (w World) dailyTrips(c gameapi.MarketCity) int {
	if w.rebalanced() {
		return w.catalog.economyTrips(c.DailyTrips)
	}
	return c.DailyTrips
}
func country(region string) string {
	if region == "honshu" || region == "hokkaido" {
		return "japan"
	}
	return region
}
func (c Catalog) economyWeight(from, to gameapi.MarketCity) int64 {
	d := int64(c.economy.GravityPaddingKM + c.distance(from.ID, to.ID))
	weight := max(1, int64(to.Attraction)*1_000_000/(d*d))
	if country(from.RoadRegion) == country(to.RoadRegion) {
		weight *= int64(c.economy.DomesticWeight)
	}
	return weight
}
func (w World) destinationWeight(from, to gameapi.MarketCity) int64 {
	if w.rebalanced() {
		return w.catalog.economyWeight(from, to)
	}
	d := int64(300 + w.catalog.distance(from.ID, to.ID))
	return max(1, int64(to.Attraction)*1_000_000/(d*d))
}
func (w World) serviceWait(frequency int) int {
	if w.rebalanced() {
		return w.catalog.economy.WaitNumerator / (frequency + 1)
	}
	return min(120, 480/frequency)
}

// Defaults are deliberately conservative and read-only. Estimate peak two-way
// leg demand, so through routes count intermediate markets without double-counting
// every boarding as an end-to-end occupied seat. Forecast remains authoritative.
func (w World) suggest(r *gameapi.Route) {
	if !w.rebalanced() {
		return
	}
	r.SuggestedFare = gameapi.Money(min(30000, max(1000, int(w.catalog.economy.FareBase)+r.KM*int(w.catalog.economy.FarePerKM))))
	diff := make([]int64, len(r.Cities)+1)
	for i, a := range r.Cities {
		if !containsCity(r.Stops, a) {
			continue
		}
		ia, ok := w.catalog.cityIndex[a]
		if !ok {
			continue
		}
		ca := w.catalog.network.Cities[ia]
		for j := i + 1; j < len(r.Cities); j++ {
			b := r.Cities[j]
			if !containsCity(r.Stops, b) {
				continue
			}
			ib := w.catalog.cityIndex[b]
			cb := w.catalog.network.Cities[ib]
			totalA, totalB := w.catalog.economyWeightTotals[ia], w.catalog.economyWeightTotals[ib]
			if totalA == 0 || totalB == 0 {
				continue
			}
			trips := (int64(w.catalog.economyTrips(ca.DailyTrips))*w.catalog.economyWeight(ca, cb)/totalA + int64(w.catalog.economyTrips(cb.DailyTrips))*w.catalog.economyWeight(cb, ca)/totalB) * 2 / 5
			diff[i] += trips
			diff[j] -= trips
		}
	}
	peak, load := int64(0), int64(0)
	for _, d := range diff {
		load += d
		peak = max(peak, load)
	}
	r.SuggestedFrequency = min(8, max(1, int((peak+519)/520)))
}

func breakEven(f *gameapi.Forecast, fare gameapi.Money, frequency int, direct bool) {
	if frequency == 0 {
		return
	}
	fixed := f.Costs - f.HandlingCosts
	var riders, net int64
	if f.Passengers > 0 {
		riders, net = int64(f.Passengers), int64(f.Revenue-f.HandlingCosts)
		f.BreakEvenUsesObservedMix = true
	} else if direct {
		riders, net = 1, int64(fare-stationPerJourney)
	}
	if riders <= 0 || net <= 0 {
		return
	}
	n := new(big.Int).Mul(big.NewInt(int64(fixed)), big.NewInt(riders))
	n.Add(n, big.NewInt(net-1))
	n.Quo(n, big.NewInt(net))
	// Keep JSON/native/WASM integer behavior identical, including extreme drafts.
	if !n.IsInt64() || n.Int64() > 2_000_000_000 {
		return
	}
	f.BreakEvenPassengers = int(n.Int64())
	f.BreakEvenAvailable = true
}

func (c Catalog) WithEconomy(e gameapi.EconomyRules) (Catalog, error) {
	if !c.network.Sparse || e.ID != "regional-balance-v1" || e.LeaseMonthly < 1 || e.LeaseMonthly > LeaseMonthly || e.RunningPerKM < 1 || e.RunningPerKM > 10000 || e.PublicAccessPerKM < 0 || e.PublicAccessPerKM > 10000 || e.TripScale < 1 || e.TripScale > 64 || e.CatchmentBonusCap < 0 || e.CatchmentBonusCap > 100000 || e.DailyTripCap < 1000 || e.DailyTripCap > 1000000 || e.GravityPaddingKM < 1 || e.GravityPaddingKM > 1000 || e.DomesticWeight < 1 || e.DomesticWeight > 16 || e.WaitNumerator < 1 || e.WaitNumerator > 960 || e.FareBase < 0 || e.FarePerKM < 1 || e.FarePerKM > 100 {
		return c, errCode("invalid_scenario")
	}
	c.economy = e
	c.economyWeightTotals = make([]int64, len(c.network.Cities))
	total := 0
	for i, a := range c.network.Cities {
		total += c.economyTrips(a.DailyTrips)
		for _, b := range c.network.Cities {
			if a.ID != b.ID {
				c.economyWeightTotals[i] += c.economyWeight(a, b)
			}
		}
	}
	if total > 30_000_000 {
		return c, errCode("invalid_scenario")
	}
	return c, nil
}
