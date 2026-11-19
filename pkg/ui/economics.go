package ui

import (
	"fmt"
	"wwhsrt/pkg/gameapi"
)

func BreakEvenTarget(f gameapi.Forecast) string {
	if !f.BreakEvenAvailable {
		return "—"
	}
	return fmt.Sprint(f.BreakEvenPassengers)
}
func EconomicsDetails(f gameapi.Forecast, locale string) []Detail {
	t := func(k string) string { return Text(locale, k) }
	return []Detail{{t("passengers"), fmt.Sprint(f.Passengers)},
		{t("breakEvenMonthly"), BreakEvenTarget(f)},
		{t("runningCosts"), Cash(f.RunningCosts)},
		{t("accessCosts"), Cash(f.AccessCosts)},
		{t("stationCosts"), Cash(f.HandlingCosts)},
		{t("leaseCosts"), Cash(f.LeaseCosts)},
		{t("maintenance"), Cash(f.MaintenanceCosts)}}
}
func ProspectServicePlan(f gameapi.Frame, id string) (int, gameapi.Money) {
	if r, ok := ServiceRoute(f, ProjectService(f, id)); ok && r.SuggestedFrequency > 0 {
		return r.SuggestedFrequency, r.SuggestedFare
	}
	return 6, 9000
}
