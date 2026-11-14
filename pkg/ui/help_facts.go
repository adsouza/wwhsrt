package ui

import (
	"fmt"
	"wwhsrt/pkg/gameapi"
)

// Facts explain authoritative outputs; this layer does not calculate demand,
// fleet allocation, construction prices, grants or settlement again.
func HelpFacts(f gameapi.Frame, s State, concept string, forecast *gameapi.Forecast) []Detail {
	t := func(key string) string { return Text(s.Locale, key) }
	out := []Detail{}
	add := func(key, value string) { out = append(out, Detail{t(key), value}) }
	switch concept {
	case "getting_started":
		active := 0
		for _, plan := range f.Services {
			if plan.Frequency > 0 {
				active++
			}
		}
		add("learnActiveServices", fmt.Sprint(active))
		add("fleet", fmt.Sprint(len(f.Fleet)))
	case "city_demand":
		if s.Selected == "" {
			add("selected", t("selectCity"))
			break
		}
		add("selected", t(string(s.Selected)))
		station, ok := StationAt(f, s.Selected)
		status := "stationAbsent"
		if ok {
			status = "stationBuilding"
			if station.Open {
				status = "stationOpen"
			}
		}
		add("station", t(status))
		if f.Network.Available {
			potential, carried := 0, 0
			for _, market := range f.Network.Markets {
				if market.From == s.Selected {
					potential += market.Demand
					carried += market.Carried
				}
			}
			add("learnOriginDemand", fmt.Sprint(potential))
			add("learnOriginCarried", fmt.Sprint(carried))
		} else {
			add("passengers", t("noNetworkReport"))
		}
	case "connections":
		add("fromCity", t(string(s.FromCity)))
		add("toCity", t(string(s.ToCity)))
		if _, ok := Pair(f, s.FromCity, s.ToCity); !ok {
			add("plans", t("chooseProspect"))
		}
	case "terrain":
		if c, ok := Pair(f, s.FromCity, s.ToCity); ok {
			for _, a := range c.Alignments {
				if a.ID == s.AlignmentID || s.AlignmentID == "" && a.ID == c.Alignments[0].ID {
					add("totalCost", Cash(a.Cost))
					add("tunnelKM", fmt.Sprint(a.TunnelKM))
					add("viaductKM", fmt.Sprint(a.ViaductKM))
					add("duration", fmt.Sprint(a.Months))
					break
				}
			}
		}
	case "funding":
		if q, ok := FundingBreakdown(f, s); ok {
			add("fundingPlan", t(string(q.FromCity))+" – "+t(string(q.ToCity)))
			add("totalCost", Cash(q.Cost))
			add("grant", "− "+Cash(q.Grant))
			add("contribution", "= "+Cash(q.Contribution))
			add("fundingCash", "− "+Cash(q.CashAvailable))
			if q.CashDeficit > 0 {
				add("fundingDeficit", "+ "+Cash(q.CashDeficit))
			}
			add("fundingGap", "= "+Cash(q.Shortfall))
		}
		add("creditAvailable", Cash(f.DebtLimit-f.Debt))
	case "construction":
		if c, ok := Pair(f, s.FromCity, s.ToCity); ok {
			for _, p := range f.Projects {
				if p.CorridorID == c.ID {
					add("opens", fmt.Sprintf("%04d-%02d", 2029+p.Ready/12, 1+p.Ready%12))
					add("monthsLeft", fmt.Sprint(max(0, p.Ready-f.MonthsResolved)))
				}
			}
		}
	case "fleet":
		if forecast != nil {
			add("fleet", fmt.Sprint(forecast.FleetAvailable))
			add("needFleet", fmt.Sprint(forecast.FleetNeeded))
			add("frequency", fmt.Sprint(s.Frequency))
			add("learnRunningMinutes", fmt.Sprint(forecast.Minutes))
			add("trainTurnaround", fmt.Sprint(f.TrainFamily.RoundTripTurnaround))
			add("learnFleetDayMinutes", fmt.Sprint(f.TrainFamily.DailyMinutes))
		}
		add("trainLease", Cash(f.TrainFamily.LeaseMonthly))
	case "service_plans", "through_services":
		if r, ok := ServiceRoute(f, s.ServiceCorridor); ok {
			add("calledStops", StopNames(r.Stops, s.Locale))
			add("learnRunningMinutes", fmt.Sprint(r.Minutes))
		}
		if s.ReplaceServiceID != "" {
			if r, ok := ServiceRoute(f, s.ReplaceServiceID); ok {
				add("learnReplacedService", RouteName(r, s.Locale))
			}
		}
		if forecast != nil {
			add("additionalLeases", fmt.Sprint(forecast.AdditionalLeases))
			add("companyChange", Cash(forecast.CompanyChange))
		}
	case "fares_demand":
		add("fare", Cash(s.Fare))
		if forecast != nil {
			add("passengers", fmt.Sprint(forecast.Passengers))
			add("breakEvenMonthly", BreakEvenTarget(*forecast))
			add("revenue", Cash(forecast.Revenue))
			add("forecastCosts", Cash(forecast.Costs))
			add("operatingContribution", Cash(forecast.Contribution))
			add("companyChange", Cash(forecast.CompanyChange))
			add("feederRevenue", Cash(forecast.FeederRevenue))
		}
	case "capacity":
		if forecast != nil && len(forecast.CapacityUsage) > 0 {
			out = append(out, CapacityDetails(f, s.Locale, forecast.CapacityUsage)...)
			break
		}
		if forecast != nil {
			add("deliveredFrequency", fmt.Sprintf("%d / %d", forecast.DeliveredFrequency, s.Frequency))
		}
		for _, section := range f.Network.Sections {
			if section.Delivered < section.Requested {
				name := section.CorridorID
				for _, c := range f.Corridors {
					if c.ID == section.CorridorID {
						name = t(string(c.FromCity)) + " – " + t(string(c.ToCity))
					}
				}
				add("pathsDelivered", fmt.Sprintf("%s: %d/%d", name, section.Delivered, section.Capacity))
				add("pathsRequested", fmt.Sprint(section.Requested))
			}
		}
	case "monthly_report":
		if len(f.Reports) > 0 {
			r := f.Reports[len(f.Reports)-1]
			add("passengers", fmt.Sprint(r.Passengers))
			add("revenue", Cash(r.Revenue))
			add("forecastCosts", Cash(r.Costs))
			add("learnOperatingCash", Cash(r.Revenue-r.Costs))
			add("borrowed", Cash(r.Borrowed))
			add("construction", Cash(r.Construction))
			add("closing", Cash(r.ClosingCash))
		} else {
			add("report", t("noReport"))
		}
	case "idle_assets":
		add("idleCosts", Cash(IdleCosts(f))+t("monthlyUnit"))
	case "operating_day":
		if len(f.Reports) > 0 {
			r := f.Reports[len(f.Reports)-1]
			add("month", fmt.Sprintf("%04d-%02d", r.Month.Year, r.Month.Month))
		} else {
			add("report", t("noReport"))
		}
		add("learnRepresentativeDepartures", fmt.Sprint(len(f.Day)))
	}
	return out
}
