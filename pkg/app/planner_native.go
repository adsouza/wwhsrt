//go:build !js

package app

import (
	"fmt"
	uimage "github.com/ebitenui/ebitenui/image"
	"github.com/ebitenui/ebitenui/widget"
	"github.com/hajimehoshi/ebiten/v2/text/v2"
	"strings"
	"wwhsrt/pkg/gameapi"
	"wwhsrt/pkg/ui"
)

type nativeLabel func(string, float64)
type nativeButton func(string, func()) *widget.Button

func (h *host) buildPlanner(g *Game, t func(string) string, label nativeLabel, button nativeButton) {
	label(t("build"), 20)
	h.taskTabs(g, "build", []string{"plan", "costs", "projects"}, button)
	pane := ui.BuildPane(g.State)
	if pane == "projects" {
		if len(g.Frame.Projects) == 0 {
			label(t("noProjects"), 14)
		}
		for _, project := range g.Frame.Projects {
			project := project
			for _, co := range g.Frame.Corridors {
				if co.ID != project.CorridorID {
					continue
				}
				co := co
				status := "underConstruction"
				if project.Ready <= g.Frame.MonthsResolved {
					status = "readyForService"
				}
				button(t(string(co.FromCity))+" – "+t(string(co.ToCity))+" · "+t(status), func() {
					g.State.FromCity = co.FromCity
					g.State.ToCity = co.ToCity
					g.State.AlignmentID = project.AlignmentID
					g.State.BuildPane = "plan"
					h.workspaceChanged(g)
				})
			}
		}
		return
	}
	if pane == "plan" {
		label(t("prospectHint"), 14)
		b := button(t("browseProspects"), func() { g.State.Expanded = true; h.layout(g); g.dirty = true })
		h.actions["browse-prospects"] = b
		opportunities := ui.ConstructionOpportunities(g.Frame)
		if len(opportunities) == 0 {
			label(t("noProspects"), 14)
		}
		options := []choice{}
		selected := ""
		if co, ok := ui.Pair(g.Frame, g.State.FromCity, g.State.ToCity); ok {
			for _, candidate := range opportunities {
				if candidate.ID == co.ID {
					selected = co.ID
				}
			}
		}
		for _, co := range opportunities {
			options = append(options, choice{co.ID, t(string(co.FromCity)) + " – " + t(string(co.ToCity))})
		}
		label(fmt.Sprintf(t("allProspectsCount"), len(opportunities)), 14)
		h.selectChoice(g, selected, options, func(id string) {
			if ui.SelectProspect(&g.State, g.Frame, id) {
				h.workspaceChanged(g)
			}
		}, t("chooseProspect"))
		for _, co := range ui.QuickProspects(g.Frame, g.State) {
			co := co
			b := button(t(string(co.FromCity))+" – "+t(string(co.ToCity)), func() {
				if ui.SelectProspect(&g.State, g.Frame, co.ID) {
					h.workspaceChanged(g)
				}
			})
			h.actions["prospect-"+co.ID] = b
			if selected, ok := ui.Pair(g.Frame, g.State.FromCity, g.State.ToCity); ok && selected.ID == co.ID {
				p := ui.PaletteFor(g.State)
				b.SetImage(
					&widget.ButtonImage{
						Idle:    uimage.NewBorderedNineSliceColor(p.Selected, p.Accent, 1),
						Hover:   uimage.NewNineSliceColor(p.Hover),
						Pressed: uimage.NewNineSliceColor(p.Pressed),
					},
				)
			}
		}
	}
	co, ok := ui.Pair(g.Frame, g.State.FromCity, g.State.ToCity)
	if !ok {
		label(t("chooseProspect"), 14)
		h.helpSlot(g, "endpoints", label, button)
		return
	}
	for _, project := range g.Frame.Projects {
		if project.CorridorID != co.ID {
			continue
		}
		if project.Ready <= g.Frame.MonthsResolved {
			label(t("readyForService"), 18)
			button(t("services"), func() {
				ui.DraftService(&g.State, g.Frame, co.ID)
				g.State.ServicePane = "plan"
				g.State.View = "services"
				h.workspaceChanged(g)
			})
		} else {
			label(
				fmt.Sprintf(
					"%s · %04d-%02d\n%s: %d",
					t("opens"),
					2029+project.Ready/12,
					1+project.Ready%12,
					t("monthsLeft"),
					project.Ready-g.Frame.MonthsResolved,
				),
				16,
			)
		}
		h.helpSlot(g, "project", label, button)
		return
	}
	if co.Existing {
		label(t("existingPublicConnection"), 15)
		b := button(t("planExistingService"), func() {
			ui.DraftService(&g.State, g.Frame, co.ID)
			g.State.ServicePane = "plan"
			g.State.View = "services"
			h.workspaceChanged(g)
		})
		h.actions["plan-existing-service"] = b
		return
	}
	label(t(string(co.FromCity))+" – "+t(string(co.ToCity)), 18)
	if co.ConstructionKind != "" {
		label(t("constructionScope_"+co.ConstructionKind), 14)
	}
	if g.State.AlignmentID == "" {
		g.State.AlignmentID = co.Alignments[0].ID
	}
	alignments := []choice{}
	for _, a := range co.Alignments {
		alignments = append(
			alignments,
			choice{a.ID, fmt.Sprintf("%s · %d km · %d min · %s %d", t(a.ID), a.KM, a.Minutes, t("duration"), a.Months)},
		)
	}
	h.selectChoice(
		g,
		g.State.AlignmentID,
		alignments,
		func(id string) { g.State.AlignmentID = id; h.workspaceChanged(g) },
	)
	for _, a := range co.Alignments {
		if a.ID != g.State.AlignmentID {
			continue
		}
		q, _ := ui.FundingBreakdown(g.Frame, g.State)
		if pane == "costs" {
			if co.ConstructionNote != "" {
				label(co.ConstructionNote+" · "+t("authoredAlignmentNote"), 13)
			}
			for _, row := range ui.EstimateDetails(a, g.State.Locale) {
				label(row.Label+"\n"+row.Value, 14)
			}
			label(fmt.Sprintf("%s %d · %s %d", t("tunnelKM"), a.TunnelKM, t("viaductKM"), a.ViaductKM), 14)
			label(t("stationPackage"), 14)
			label(t("estimateNote"), 13)
			h.helpSlot(g, "engineering", label, button)
			return
		}
		label(t("contribution")+" · "+ui.Cash(q.Contribution), 17)
		if q.Shortfall > 0 {
			label(t("fundingGap")+" · "+ui.Cash(q.Shortfall), 14)
			b := button(
				t("borrowShortfall")+" · "+ui.Cash(q.Shortfall),
				func() { g.State.LoanAmount = ui.DollarInput(q.Shortfall); h.send(g, gameapi.TakeLoan) },
			)
			b.GetWidget().Disabled = h.busy || q.Shortfall > g.Frame.DebtLimit-g.Frame.Debt
		}
		if h.disclosure(g, "funding", t("fundingPlan"), button) {
			fundingLabels(g, q, label)
			button(
				t("finance"),
				func() { g.State.CompanyPane = "finance"; g.State.View = "company"; h.workspaceChanged(g) },
			)
		}
		frequency, fare := ui.ProspectServicePlan(g.Frame, co.ID)
		if forecast, e := g.Port.Forecast(
			gameapi.ForecastRequest{CorridorID: co.ID, AlignmentID: a.ID, Frequency: frequency, Fare: fare},
		); e == nil {
			label(t("operatingContribution")+"\n"+ui.Cash(forecast.Contribution), 14)
			label(fmt.Sprintf(t("previewServicePlan"), frequency, ui.Cash(fare)), 12)
		}
		b := button(t("commitBuild"), func() { h.send(g, gameapi.BuildLine) })
		h.primaryAction = b
		b.GetWidget().Disabled = h.busy || q.Shortfall > 0
		h.helpSlot(g, "endpoints", label, button)
		h.helpSlot(g, "alignments", label, button)
		h.helpSlot(g, "funding", label, button)
	}
}
func (h *host) buildServices(g *Game, t func(string) string, label nativeLabel, button nativeButton) {
	label(t("services"), 20)
	h.taskTabs(g, "services", []string{"plan", "fleet", "analysis"}, button)
	if h.disclosure(g, "route-search", t("searchRoutes"), button) {
		label(t("routeSearch"), 13)
		h.routeSearch = newNameInput(
			&h.routeSearchField,
			g.State.ServiceQuery,
			&text.GoTextFace{Source: g.Map.Font, Size: 14},
			ui.PaletteFor(g.State),
		)
		h.content.AddChild(h.routeSearch)
	}
	if len(g.State.ServiceMapSections) > 0 {
		label(t("mapServiceChoices"), 13)
		b := button(t("allServiceRoutes"), func() { g.State.ServiceMapSections = nil; h.workspaceChanged(g) })
		h.actions["all-service-routes"] = b
	}
	label(t("chooseRoute"), 13)
	options := []choice{}
	for _, r := range ui.MapServiceRoutes(g.Frame, g.State) {
		matches := true
		for _, word := range strings.Fields(strings.ToLower(g.State.ServiceQuery)) {
			if !strings.Contains(strings.ToLower(ui.RouteName(r, g.State.Locale)), word) {
				matches = false
			}
		}
		if r.Open && (matches || r.ID == g.State.ServiceCorridor) {
			options = append(options, choice{r.ID, ui.RouteName(r, g.State.Locale)})
		}
	}
	h.selectChoice(
		g,
		g.State.ServiceCorridor,
		options,
		func(id string) { ui.DraftService(&g.State, g.Frame, id); h.workspaceChanged(g) },
	)
	r, routeOK := ui.ServiceRoute(g.Frame, g.State.ServiceCorridor)
	if routeOK {
		label(fmt.Sprintf("%d km · %d min", r.KM, r.Minutes), 13)
	}
	f, e := g.Port.Forecast(
		gameapi.ForecastRequest{
			CorridorID:       g.State.ServiceCorridor,
			Frequency:        g.State.Frequency,
			Fare:             g.State.Fare,
			ReplaceServiceID: g.State.ReplaceServiceID,
		},
	)
	pane := ui.ServicePane(g.State)
	if pane == "fleet" {
		if e != nil {
			label(t(e.Error()), 14)
			button(t("pane_plan"), func() { g.State.ServicePane = "plan"; h.workspaceChanged(g) })
		}
		label(fmt.Sprintf("%s: %d / %d", t("fleet"), f.FleetAvailable, f.FleetNeeded), 16)
		h.row(2, func() {
			button(t("leaseTrain"), func() { h.send(g, gameapi.LeaseTrain) })
			button(t("returnTrain"), func() { h.send(g, gameapi.ReturnTrain) })
		})
		label(t(g.Frame.TrainFamily.NameKey)+" · "+fmt.Sprint(g.Frame.TrainFamily.Seats)+" "+t("seatUnit"), 15)
		if h.disclosure(g, "build-details", t("fleetDetails"), button) {
			for _, row := range ui.TrainDetails(g.Frame, g.State.Locale) {
				label(row.Label+"\n"+row.Value, 14)
			}
		}
		h.helpSlot(g, "fleet", label, button)
		return
	}
	if pane == "analysis" {
		if r.ConnectorKM > 0 {
			label(fmt.Sprintf("%s: %d km. %s", t("conventionalConnectors"), r.ConnectorKM, t("connectorHint")), 13)
		}
		if e != nil {
			label(t(e.Error()), 14)
			h.helpSlot(g, "capacity", label, button)
			return
		} else {
			for _, row := range []struct {
				k string
				v gameapi.Money
			}{
				{"revenue", f.Revenue},
				{"forecastCosts", f.Costs},
				{"operatingContribution", f.Contribution},
				{"companyChange", f.CompanyChange},
				{"feederRevenue", f.FeederRevenue},
			} {
				label(t(row.k)+"\n"+ui.Cash(row.v), 16)
			}
			label(
				fmt.Sprintf(
					"%s: %d · %s %d/%d",
					t("passengers"),
					f.Passengers,
					t("deliveredFrequency"),
					f.DeliveredFrequency,
					g.State.Frequency,
				),
				14,
			)
		}
		for _, row := range ui.CapacityDetails(g.Frame, g.State.Locale, f.CapacityUsage) {
			label(row.Label+"\n"+row.Value, 14)
		}
		for _, row := range ui.EconomicsDetails(f, g.State.Locale) {
			label(row.Label+"\n"+row.Value, 13)
		}
		label(t("breakEvenAssumptions"), 12)
		if !f.BreakEvenAvailable {
			label(t("breakEvenUnavailable"), 12)
		}
		h.helpSlot(g, "capacity", label, button)
		h.helpSlot(g, "fares", label, button)
		return
	}
	if len(g.Frame.IdleLines) > 0 &&
		h.disclosure(g, "idle", fmt.Sprintf("%s · %d", t("idleReview"), len(g.Frame.IdleLines)), button) {
		for _, line := range g.Frame.IdleLines {
			line := line
			button(ui.IdleName(g.Frame, g.State.Locale, line), func() { h.reviewIdleLine(g, line.CorridorID) })
		}
	}
	h.helpSlot(g, "idle", label, button)
	if routeOK && len(r.Stops) > 2 && h.disclosure(g, "service-path", t("calledStops"), button) {
		label(ui.StopNames(r.Stops, g.State.Locale), 14)
	}
	h.helpSlot(g, "through", label, button)
	h.helpSlot(g, "service-plan", label, button)
	if g.State.ServiceCorridor == "tokyo-osaka" {
		for _, plan := range g.Frame.Services {
			if plan.CorridorID == "tokyo-nagoya" && plan.Frequency > 0 {
				title := t("extendStarter")
				if g.State.ReplaceServiceID != "" {
					title = "● " + title
				}
				button(title, func() {
					if g.State.ReplaceServiceID == "" {
						g.State.ReplaceServiceID = "tokyo-nagoya"
					} else {
						g.State.ReplaceServiceID = ""
					}
					h.workspaceChanged(g)
				})
			}
		}
	}
	change := func() {
		if h.message == "track_capacity" || h.message == "station_capacity" {
			h.message = ""
		}
		if !ui.BlockingHelpIssue(h.message) {
			if ui.ServiceDirty(g.State, g.Frame) {
				h.message = "draft_not_applied"
			} else if h.message == "draft_not_applied" {
				h.message = ""
			}
		}
		g.dirty = true
		h.persist(g)
	}
	h.row(2, func() {
		h.stepper(
			g,
			"frequency",
			t("frequency"),
			fmt.Sprint(g.State.Frequency),
			func() { g.State.Frequency = max(0, g.State.Frequency-1); change() },
			func() { g.State.Frequency = min(16, g.State.Frequency+1); change() },
		)
		h.stepper(
			g,
			"fare",
			t("fareShort"),
			ui.DollarInput(g.State.Fare),
			func() { g.State.Fare = max(1000, g.State.Fare-1000); change() },
			func() { g.State.Fare = min(30000, g.State.Fare+1000); change() },
		)
	})
	h.helpSlot(g, "fares", label, button)
	if e != nil && len(f.CapacityUsage) == 0 {
		label(t(e.Error()), 14)
	}
	for _, summary := range ui.CapacitySummary(g.Frame, g.State.Locale, f.CapacityUsage) {
		title := summary.Label + " · " + summary.Value
		colored := func(title string, fn func()) *widget.Button {
			b := button(title, fn)
			if e != nil {
				p := ui.PaletteFor(g.State)
				b.SetImage(
					&widget.ButtonImage{
						Idle:    uimage.NewNineSliceColor(p.Warning),
						Hover:   uimage.NewNineSliceColor(p.WarningHover),
						Pressed: uimage.NewNineSliceColor(p.Warning),
					},
				)
			}
			return b
		}
		if h.disclosure(g, "capacity", title, colored) {
			for _, row := range ui.CapacityDetails(g.Frame, g.State.Locale, f.CapacityUsage) {
				label(row.Label+"\n"+row.Value, 14)
			}
		}
	}

	h.helpSlot(g, "capacity", label, button)
	label(
		fmt.Sprintf(
			"%s: %d / %d · %s%s",
			t("fleetShort"),
			f.FleetAvailable,
			f.FleetNeeded,
			ui.Cash(g.Frame.TrainFamily.LeaseMonthly),
			t("monthlyUnit"),
		),
		13,
	)
	h.row(2, func() {
		b := button(t("leaseShort"), func() { h.send(g, gameapi.LeaseTrain) })
		b.GetWidget().Disabled = h.busy
		b = button(t("returnShort"), func() { h.send(g, gameapi.ReturnTrain) })
		b.GetWidget().Disabled = h.busy
	})
	h.helpSlot(g, "fleet-plan", label, button)
	if e == nil {
		h.row(3, func() {
			label(t("forecastRevenue")+"\n"+ui.Cash(f.Revenue), 13)
			label(t("forecastContribution")+"\n"+ui.Cash(f.Contribution), 13)
			label(t("ridersBreakEven")+"\n"+ui.BreakEvenTarget(f), 13)
		})
	}
	if ui.ServiceDirty(g.State, g.Frame) {
		label(t("draftChanges"), 13)
	}
	title := t("applyService")
	kind := gameapi.SetService
	if g.State.ReplaceServiceID != "" {
		title = t("applyExtension")
		kind = gameapi.ExtendService
		label(
			fmt.Sprintf("%s: %d · %s: %d", t("moveFleet"), f.FleetAvailable, t("additionalLeases"), f.AdditionalLeases),
			13,
		)
	}
	apply := func() {
		b := button(title, func() { h.send(g, kind) })
		h.primaryAction = b
		b.GetWidget().Disabled = h.busy || e != nil || !f.Open ||
			g.State.ReplaceServiceID == "" && f.FleetNeeded > f.FleetAvailable ||
			g.State.ReplaceServiceID != "" && f.AdditionalLeases > 0 && g.Frame.Player.Cash < 0
	}
	if ui.ServiceDirty(g.State, g.Frame) {
		h.row(2, func() {
			button(t("restorePlan"), func() {
				ui.DraftService(&g.State, g.Frame, g.State.ServiceCorridor)
				h.message = ""
				h.workspaceChanged(g)
			})
			apply()
		})
	} else {
		apply()
	}
}
