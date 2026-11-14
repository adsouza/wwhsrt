//go:build !js

package app

import (
	"fmt"

	uimage "github.com/ebitenui/ebitenui/image"
	"github.com/ebitenui/ebitenui/widget"
	"github.com/hajimehoshi/ebiten/v2/text/v2"
	"wwhsrt/pkg/ui"
)

func (h *host) reviewIdleLine(g *Game, id string) {
	if !ui.ReviewIdle(&g.State, g.Frame, id) {
		return
	}
	if h.scroll != nil {
		h.scroll.ScrollTop = 0
	}
	h.layout(g)
	g.focusIdle(id)
	g.dirty = true
	h.persist(g)
}

func (h *host) idleReviewButton(g *Game, value string, width int) *widget.Button {
	p := ui.PaletteFor(g.State)
	var face text.Face = &text.GoTextFace{Source: g.Map.Font, Size: 13}
	value = ui.WrapText(value, float64(width-16), func(s string) float64 { return text.Advance(s, face) })
	return widget.NewButton(
		widget.ButtonOpts.WidgetOpts(widget.WidgetOpts.MinSize(44, 44), widget.WidgetOpts.LayoutData(widget.RowLayoutData{Stretch: true, MaxWidth: width})),
		widget.ButtonOpts.Image(&widget.ButtonImage{Idle: uimage.NewBorderedNineSliceColor(p.Warning, p.WarningText, 1), Hover: uimage.NewNineSliceColor(p.WarningHover), Pressed: uimage.NewNineSliceColor(p.Warning)}),
		widget.ButtonOpts.Text(value, &face, &widget.ButtonTextColor{Idle: p.WarningText}), widget.ButtonOpts.TextPadding(widget.NewInsetsSimple(8)),
		widget.ButtonOpts.ClickedHandler(func(*widget.ButtonClickedEventArgs) { h.reviewIdleLine(g, "") }))
}

func (h *host) idleNotice(g *Game, width int) *widget.Container {
	p := ui.PaletteFor(g.State)
	t := func(key string) string { return ui.Text(g.State.Locale, key) }
	notice := widget.NewContainer(widget.ContainerOpts.WidgetOpts(widget.WidgetOpts.LayoutData(widget.RowLayoutData{Stretch: true})),
		widget.ContainerOpts.BackgroundImage(uimage.NewNineSliceColor(p.Warning)),
		widget.ContainerOpts.Layout(widget.NewRowLayout(widget.RowLayoutOpts.Direction(widget.DirectionVertical), widget.RowLayoutOpts.Padding(widget.NewInsetsSimple(12)), widget.RowLayoutOpts.Spacing(8))))
	var face text.Face = &text.GoTextFace{Source: g.Map.Font, Size: 13}
	value := fmt.Sprintf("%s · %d\n%s\n%s%s", t("idleLinesTitle"), len(g.Frame.IdleLines), t("idleCosts"), ui.Cash(ui.IdleCosts(g.Frame)), t("monthlyUnit"))
	value = ui.WrapText(value, float64(width-24), func(s string) float64 { return text.Advance(s, face) })
	notice.AddChild(widget.NewText(widget.TextOpts.Text(value, &face, p.WarningText)))
	notice.AddChild(h.idleReviewButton(g, t("reviewIdleLines"), width-24))
	return notice
}
