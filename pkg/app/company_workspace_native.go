//go:build !js

package app

import (
	"fmt"
	"github.com/ebitenui/ebitenui/widget"
	"github.com/hajimehoshi/ebiten/v2/text/v2"
	"wwhsrt/pkg/gameapi"
	"wwhsrt/pkg/ui"
)

func (h *host) buildCompany(
	g *Game,
	t func(string) string,
	label nativeLabel,
	button nativeButton,
	content *widget.Container,
	width int,
	previousName *nameInput,
	refocus bool,
) {
	label(t("company"), 20)
	h.taskTabs(g, "company", []string{"finance", "identity", "settings"}, button)
	switch ui.CompanyPane(g.State) {
	case "identity":
		label(t("companyName"), 16)
		var face text.Face = &text.GoTextFace{Source: g.Map.Font, Size: 16}
		h.nameInput = newNameInput(&h.field, g.State.DraftName, face, ui.PaletteFor(g.State))
		if previousName != nil && previousName.observed.value == h.field.Text() {
			h.nameInput.undo, h.nameInput.redo = previousName.undo, previousName.redo
			h.nameInput.anchor, h.nameInput.caret = previousName.anchor, previousName.caret
			h.nameInput.offset = previousName.offset
			h.nameInput.observed = h.nameInput.snapshot()
		}
		content.AddChild(h.nameInput)
		button(t("applyName"), func() {
			if h.field.UncommittedTextLengthInBytes() > 0 {
				return
			}
			h.blurName()
			h.syncName(g)
			h.send(g, gameapi.RenameCompany)
		})
		button(t("reset"), func() { h.openNewGame(g) }).GetWidget().Disabled = h.busy
	case "settings":
		label(t("workspace"), 17)
		label(t("workspaceResponsive"), 13)
		h.row(2, func() {
			button(t("dock_top"), func() { g.State.CommandDock = "top"; h.workspaceChanged(g) })
			button(t("dock_left"), func() { g.State.CommandDock = "left"; h.workspaceChanged(g) })
		})
		label(fmt.Sprintf("%s: %d px", t("panelWidth"), ui.PanelWidth(g.State.PanelWidth)), 14)
		h.row(2, func() {
			button(t("narrower"), func() {
				g.State.PanelWidth = ui.PanelWidth(ui.PanelWidth(g.State.PanelWidth) - 40)
				h.workspaceChanged(g)
			})
			button(t("wider"), func() {
				g.State.PanelWidth = ui.PanelWidth(ui.PanelWidth(g.State.PanelWidth) + 40)
				h.workspaceChanged(g)
			})
		})
		button(t("resetWorkspace"), func() { g.State.PanelWidth = 0; g.State.CommandDock = ""; h.workspaceChanged(g) })
		button(t("appearance")+" · "+t("appearance_"+ui.AppearanceMode(g.State.Appearance)), func() {
			g.State.Appearance = ui.NextAppearance(g.State.Appearance)
			h.refreshAppearance(g)
			h.workspaceChanged(g)
		})
		for _, setting := range []struct {
			key   string
			value bool
			run   func()
		}{
			{"motion", g.State.Motion, func() { g.State.Motion = !g.State.Motion }},
			{"terrain", !g.State.HideTerrain, func() { g.State.HideTerrain = !g.State.HideTerrain }},
			{"guidanceSettings", !g.State.Help.Muted, func() { g.State.Help.Muted = !g.State.Help.Muted }},
		} {
			setting := setting
			state := "motionOff"
			if setting.value {
				state = "motionOn"
			}
			button(t(setting.key)+" · "+t(state), func() { setting.run(); h.workspaceChanged(g) })
		}
		button(t("resetLearning"), func() { h.helpEvent(g, ui.HelpEvent{Kind: "reset"}) })
		label(t("soundless"), 13)
		label(t("terrainCredits"), 12)
	default:
		label(t("debt")+" · "+ui.Cash(g.Frame.Debt), 16)
		h.financeForm(g, t, label, button, content, width)
		button(t("repay"), func() { h.send(g, gameapi.RepayLoan) })
		h.financeProject(g, t, label, button)
		h.helpSlot(g, "finance", label, button)
	}
}
func (h *host) buildReport(g *Game, t func(string) string, label nativeLabel, button nativeButton) {
	label(t("report"), 20)
	h.taskTabs(g, "report", []string{"overview", "accounts", "network", "markets"}, button)
	if len(g.Frame.Reports) == 0 {
		label(t("noReport"), 15)
		return
	}
	r := g.Frame.Reports[len(g.Frame.Reports)-1]
	label(fmt.Sprintf("%04d-%02d", r.Month.Year, r.Month.Month), 14)
	switch ui.ReportPane(g.State) {
	case "accounts":
		for _, row := range []struct {
			k string
			v gameapi.Money
		}{
			{"opening", r.OpeningCash},
			{"revenue", r.Revenue},
			{"runningCosts", r.RunningCosts},
			{"accessCosts", r.AccessCosts},
			{"stationCosts", r.StationCosts},
			{"leaseCosts", r.LeaseCosts},
			{"maintenance", r.Maintenance},
			{"interest", r.Interest},
			{"construction", r.Construction},
			{"grants", r.Grants},
			{"borrowed", r.Borrowed},
			{"repaid", r.Repaid},
			{"closing", r.ClosingCash},
		} {
			label(t(row.k)+"\n"+ui.Cash(row.v), 15)
		}
		h.helpSlot(g, "accounts", label, button)
	case "network":
		for _, row := range ui.NetworkDetails(g.Frame, g.State.Locale) {
			label(row.Label+"\n"+row.Value, 14)
		}
		h.helpSlot(g, "capacity", label, button)
	case "markets":
		h.marketControls(g, label, button)
	default:
		h.row(2, func() {
			label(t("revenue")+"\n"+ui.Cash(r.Revenue), 17)
			label(t("forecastCosts")+"\n"+ui.Cash(r.Costs), 17)
			label(t("closing")+"\n"+ui.Cash(r.ClosingCash), 17)
			label(t("passengers")+"\n"+fmt.Sprint(r.Passengers), 17)
		})
		h.helpSlot(g, "accounts", label, button)
	}
}
