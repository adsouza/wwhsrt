//go:build !js

package app

import (
	"fmt"
	"github.com/ebitenui/ebitenui/widget"
	"wwhsrt/pkg/gameapi"
	"wwhsrt/pkg/ui"
)

type nativeLabel func(string, float64)
type nativeButton func(string, func()) *widget.Button

func (h *host) buildPlanner(g *Game, t func(string) string, label nativeLabel, button nativeButton) {
	name := func(id gameapi.CityID) string { return t(string(id)) }
	label(t("build"), 22)
	label(t("fromCity")+": "+name(g.State.FromCity)+"\n"+t("toCity")+": "+name(g.State.ToCity), 16)
	button(t("chooseFrom"), func() { g.State.Choosing = "from"; g.State.Expanded = true; h.layout(g); g.dirty = true })
	button(t("chooseTo"), func() { g.State.Choosing = "to"; g.State.Expanded = true; h.layout(g); g.dirty = true })
	label(t("plans"), 16)
	for _, co := range ui.ConstructionOpportunities(g.Frame) {
		c := co
		button(name(c.FromCity)+" – "+name(c.ToCity), func() {
			g.State.FromCity = c.FromCity
			g.State.ToCity = c.ToCity
			g.State.AlignmentID = c.Alignments[0].ID
			g.dirty = true
			h.persist(g)
		})
	}
	co, ok := ui.Pair(g.Frame, g.State.FromCity, g.State.ToCity)
	if !ok {
		label(t("no_route"), 15)
		return
	}
	for _, p := range g.Frame.Projects {
		if p.CorridorID == co.ID {
			if p.Ready <= g.Frame.MonthsResolved {
				label(t("readyForService"), 20)
				button(t("services"), func() { ui.DraftService(&g.State, g.Frame, co.ID); g.State.View = "services"; g.dirty = true })
			} else {
				m := gameapi.Month{Year: 2029 + p.Ready/12, Month: 1 + p.Ready%12}
				label(fmt.Sprintf("%s\n%s: %04d-%02d\n%s: %d", t("underConstruction"), t("opens"), m.Year, m.Month, t("monthsLeft"), p.Ready-g.Frame.MonthsResolved), 18)
				button(t("advanceYear"), func() { h.send(g, gameapi.AdvanceYear) })
			}
			return
		}
	}
	if co.Existing {
		label(t("publicTrack"), 16)
		return
	}
	if g.State.AlignmentID == "" {
		g.State.AlignmentID = co.Alignments[0].ID
	}
	for _, al := range co.Alignments {
		a := al
		title := fmt.Sprintf("%s · %d km · %d min · %s %d", t(a.ID), a.KM, a.Minutes, t("duration"), a.Months)
		if a.ID == g.State.AlignmentID {
			title = "● " + title
		}
		button(title, func() { g.State.AlignmentID = a.ID; g.dirty = true; h.persist(g) })
	}
	for _, a := range co.Alignments {
		if a.ID != g.State.AlignmentID {
			continue
		}
		label(fmt.Sprintf("%s: %d · %s: %d", t("tunnelKM"), a.TunnelKM, t("viaductKM"), a.ViaductKM), 14)
		why := "kansai"
		if co.ID == "hokkaido-extension" {
			why = "hokkaido"
		}
		if a.ID == "mountain" {
			why += "MountainWhy"
		} else {
			why += "ValleyWhy"
		}
		label(t(why), 14)
		q, _ := ui.FundingBreakdown(g.Frame, g.State)
		fundingLabels(g, q, label)
		if gap := q.Shortfall; gap > 0 {
			label(t("shortfallInterest")+"\n"+ui.Cash((g.Frame.Debt+gap+599)/600-(g.Frame.Debt+599)/600), 14)
			loan := button(t("borrowShortfall")+"\n"+ui.Cash(gap), func() { g.State.LoanAmount = ui.DollarInput(gap); h.send(g, gameapi.TakeLoan) })
			loan.GetWidget().Disabled = h.busy || gap > g.Frame.DebtLimit-g.Frame.Debt
			button(t("finance"), func() {
				g.State.LoanAmount = ui.DollarInput(gap)
				g.State.View = "company"
				g.dirty = true
				h.persist(g)
			})
		}
		f, e := g.Port.Forecast(gameapi.ForecastRequest{CorridorID: co.ID, AlignmentID: a.ID, Frequency: 6, Fare: 9000})
		if e == nil {
			label(t("operatingContribution")+"\n"+ui.Cash(f.Contribution), 16)
		}
		b := button(t("commitBuild"), func() { h.send(g, gameapi.BuildLine) })
		b.GetWidget().Disabled = h.busy || g.Frame.Player.Cash < a.Cost-a.Grant
	}
	label(t("estimateNote"), 14)
	label(t("buildHint"), 14)
}
func (h *host) buildServices(g *Game, t func(string) string, label nativeLabel, button nativeButton) {
	label(t("services"), 22)
	for _, line := range g.Frame.IdleLines {
		if line.CorridorID == g.State.ServiceCorridor {
			reason := "idleNeedsService"
			if line.NeedsTrains {
				reason = "idleNeedsTrains"
			}
			label(t("idleLinesTitle")+"\n"+t(reason)+"\n"+t("idleCosts")+": "+ui.Cash(line.Maintenance+line.Leases)+t("monthlyUnit"), 14)
		}
	}
	if ui.ServiceDirty(g.State, g.Frame) {
		label(t("draft_not_applied"), 14)
		button(t("restorePlan"), func() { ui.DraftService(&g.State, g.Frame, g.State.ServiceCorridor); g.dirty = true; h.persist(g) })
	}
	for _, co := range g.Frame.Corridors {
		if !ui.Open(g.Frame, co.ID) {
			continue
		}
		c := co
		title := t(string(c.FromCity)) + " – " + t(string(c.ToCity))
		for _, line := range g.Frame.IdleLines {
			if line.CorridorID == c.ID {
				title += " · " + t("idleShort")
			}
		}
		if c.ID == g.State.ServiceCorridor {
			title = "● " + title
		}
		button(title, func() { ui.DraftService(&g.State, g.Frame, c.ID); g.dirty = true; h.persist(g) })
	}
	change := func() { g.dirty = true; h.persist(g) }
	label(fmt.Sprintf("%s: %d", t("frequency"), g.State.Frequency), 17)
	button("− 1", func() { g.State.Frequency = max(0, g.State.Frequency-1); change() })
	button("+ 1", func() { g.State.Frequency = min(16, g.State.Frequency+1); change() })
	label(t("fare")+": "+ui.Cash(g.State.Fare), 17)
	button("− USD 10", func() { g.State.Fare = max(1000, g.State.Fare-1000); change() })
	button("+ USD 10", func() { g.State.Fare = min(30000, g.State.Fare+1000); change() })
	f, e := g.Port.Forecast(gameapi.ForecastRequest{CorridorID: g.State.ServiceCorridor, Frequency: g.State.Frequency, Fare: g.State.Fare})
	if e != nil {
		label(t(e.Error()), 15)
		return
	}
	label(fmt.Sprintf("%s: %d\n%s: %d", t("fleet"), f.FleetAvailable, t("needFleet"), f.FleetNeeded), 16)
	button(t("leaseTrain"), func() { h.send(g, gameapi.LeaseTrain) })
	button(t("returnTrain"), func() { h.send(g, gameapi.ReturnTrain) })
	label(fmt.Sprintf("%s: %d", t("passengers"), f.Passengers), 18)
	label(t("monthForecast")+"\n"+ui.Cash(f.Revenue), 16)
	label(t("forecastCosts")+"\n"+ui.Cash(f.Costs), 16)
	label(t("operatingContribution")+"\n"+ui.Cash(f.Contribution), 18)
	if f.FleetNeeded > f.FleetAvailable {
		label(t("fleet_shortage"), 15)
	}
	b := button(t("applyService"), func() { h.send(g, gameapi.SetService) })
	b.GetWidget().Disabled = h.busy || f.FleetNeeded > f.FleetAvailable || !f.Open
	label(t("serviceHint"), 14)
}
