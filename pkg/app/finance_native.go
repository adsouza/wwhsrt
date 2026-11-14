//go:build !js

package app

import (
	uimage "github.com/ebitenui/ebitenui/image"
	"github.com/ebitenui/ebitenui/widget"

	"wwhsrt/pkg/gameapi"

	"wwhsrt/pkg/ui"
)

func (h *host) financeForm(
	g *Game,
	t func(string) string,
	label nativeLabel,
	button nativeButton,
	content *widget.Container,
	width int,
) {
	p := ui.PaletteFor(g.State)
	remaining := g.Frame.DebtLimit - g.Frame.Debt
	label(t("creditAvailable")+"\n"+ui.Cash(remaining), 15)
	label(t("loanAmount"), 16)
	face := h.face(g, 16)
	note := widget.NewText(widget.TextOpts.Text("", &face, p.Text), widget.TextOpts.MaxWidth(float64(width)))
	var borrow *widget.Button
	update := func() {
		amount, ok := ui.ParseDollars(g.State.LoanAmount)
		valid := ok && amount > 0 && amount <= remaining
		if borrow != nil {
			borrow.GetWidget().Disabled = h.busy || !valid || g.Frame.MonthsResolved >= g.Frame.CampaignMonths
		}
		if !ok || amount <= 0 {
			note.Label = t("invalid_loan_amount")
		} else if amount > remaining {
			note.Label = t("loan_limit")
		} else {
			note.Label = t("loanInterest") + "\n" + ui.Cash((g.Frame.Debt+amount+599)/600-(g.Frame.Debt+599)/600)
		}
	}
	input := widget.NewTextInput(
		widget.TextInputOpts.WidgetOpts(
			widget.WidgetOpts.MinSize(h.px(60), h.px(44)),
			widget.WidgetOpts.LayoutData(widget.RowLayoutData{Stretch: true}),
		),
		widget.TextInputOpts.Image(
			&widget.TextInputImage{
				Idle:     uimage.NewNineSliceColor(p.Control),
				Disabled: uimage.NewNineSliceColor(p.Disabled),
			},
		),
		widget.TextInputOpts.Face(&face),
		widget.TextInputOpts.Color(&widget.TextInputColor{Idle: p.Text, Disabled: p.Muted, Caret: p.Accent}),
		widget.TextInputOpts.Padding(widget.NewInsetsSimple(h.px(10))),
		widget.TextInputOpts.ChangedHandler(
			func(a *widget.TextInputChangedEventArgs) { g.State.LoanAmount = a.InputText; h.persist(g); update() },
		),
	)
	input.SetText(g.State.LoanAmount)
	h.loanInput = input
	content.AddChild(input)
	content.AddChild(note)
	borrow = button(t("borrow"), func() { h.send(g, gameapi.TakeLoan) })
	update()
}

func (h *host) financeProject(g *Game, t func(string) string, label nativeLabel, button nativeButton) {
	if !h.disclosure(g, "company-funding", t("fundingPlan"), button) {
		return
	}
	if q, ok := ui.FundingBreakdown(g.Frame, g.State); ok {
		fundingLabels(g, q, label)
		button(t("reviewFundingPlan"), func() {
			g.State.AlignmentID = q.AlignmentID
			g.State.BuildPane = "plan"
			g.State.View = "build"
			h.workspaceChanged(g)
		})
		if gap := q.Shortfall; gap > 0 {
			label(t("shortfallInterest")+"\n"+ui.Cash((g.Frame.Debt+gap+599)/600-(g.Frame.Debt+599)/600), 14)
			b := button(
				t("borrowShortfall")+"\n"+ui.Cash(gap),
				func() { g.State.LoanAmount = ui.DollarInput(gap); h.send(g, gameapi.TakeLoan) },
			)
			b.GetWidget().Disabled = h.busy || gap > g.Frame.DebtLimit-g.Frame.Debt
		}
		label(t("fundingLoanHelp"), 14)
	} else {
		label(t("fundingChoosePlan"), 14)
		button(t("build"), func() { g.State.BuildPane = "plan"; g.State.View = "build"; h.workspaceChanged(g) })
	}
}
