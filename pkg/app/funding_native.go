//go:build !js

package app

import (
	"wwhsrt/pkg/gameapi"
	"wwhsrt/pkg/ui"
)

func fundingLabels(g *Game, q ui.ConstructionFunding, label nativeLabel) {
	t := func(k string) string { return ui.Text(g.State.Locale, k) }
	label(t("fundingPlan"), 18)
	label(t(string(q.FromCity))+" – "+t(string(q.ToCity))+" · "+t(q.AlignmentID), 15)
	for _, row := range []struct {
		key, sign string
		value     gameapi.Money
	}{
		{"totalCost", "", q.Cost},
		{"grant", "− ", q.Grant},
		{"contribution", "= ", q.Contribution},
		{"fundingCash", "− ", q.CashAvailable},
		{"fundingDeficit", "+ ", q.CashDeficit},
		{"fundingGap", "= ", q.Shortfall},
	} {
		if row.key == "fundingDeficit" && row.value == 0 {
			continue
		}
		label(t(row.key)+"\n"+row.sign+ui.Cash(row.value), 15)
	}
	if q.CashDeficit > 0 {
		label(t("fundingNegativeCash"), 14)
	}
	label(t("fundingFormula"), 14)
}
