//go:build !js

package app

import (
	"fmt"
	"image"
	"image/color"

	"github.com/ebitenui/ebitenui"
	uimage "github.com/ebitenui/ebitenui/image"
	"github.com/ebitenui/ebitenui/widget"
	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/text/v2"
	"wwhsrt/pkg/gameapi"

	"wwhsrt/pkg/ui"
)

func (h *host) blocksMap() bool {
	x, y := ebiten.CursorPosition()
	if h.mapControlAt(x, y) {
		return true
	}
	for _, id := range ebiten.AppendTouchIDs(nil) {
		x, y := ebiten.TouchPosition(id)
		if h.mapControlAt(x, y) {
			return true
		}
	}
	for _, combo := range h.combos {
		if combo.ContentVisible() {
			return true
		}
	}
	return h.menuInput || h.confirmNewGame || h.helpOpen || h.resizing || h.workspace.Map.Empty()
}

func (h *host) mapControlAt(x, y int) bool {
	for _, r := range h.mapControls {
		if image.Pt(x, y).In(r) {
			return true
		}
	}
	return false
}

func (h *host) openNewGame(g *Game) {
	if h.busy {
		return
	}
	h.blurName()
	if h.loanInput != nil {
		h.loanInput.Focus(false)
	}
	h.confirmNewGame = true
	h.message = ""
	h.scroll = nil
	g.drag, g.touching = false, false
	g.dirty = true
}

func (h *host) cancelNewGame(g *Game) {
	if h.busy {
		return
	}
	h.confirmNewGame = false
	h.scroll = nil
	g.dirty = true
}

func (h *host) buildNewGame(g *Game) {
	p := ui.PaletteFor(g.State)
	t := func(k string) string { return ui.Text(g.State.Locale, k) }
	face := func(n float64) *text.Face { f := h.face(g, n); return &f }
	w, height := min(440, g.width-24), min(540, g.height-24)
	x, y := (g.width-w)/2, (g.height-height)/2
	bounds := image.Rect(x, y, x+w, y+height)
	root := widget.NewContainer(
		widget.ContainerOpts.Layout(placed{}),
		widget.ContainerOpts.BackgroundImage(uimage.NewNineSliceColor(color.NRGBA{20, 50, 82, 180})),
	)
	panel := widget.NewContainer(
		widget.ContainerOpts.Layout(placed{}),
		widget.ContainerOpts.BackgroundImage(uimage.NewNineSliceColor(p.Control)),
		widget.ContainerOpts.WidgetOpts(widget.WidgetOpts.LayoutData(bounds)),
	)
	root.AddChild(panel)
	content := widget.NewContainer(
		widget.ContainerOpts.Layout(
			widget.NewRowLayout(
				widget.RowLayoutOpts.Direction(widget.DirectionVertical),
				widget.RowLayoutOpts.Padding(widget.NewInsetsSimple(16)),
				widget.RowLayoutOpts.Spacing(12),
			),
		),
	)
	label := func(s string, size float64) {
		wrapped := ui.WrapText(s, float64(w-52), func(v string) float64 { return text.Advance(v, *face(size)) })
		content.AddChild(
			widget.NewText(widget.TextOpts.Text(wrapped, face(size), p.Text), widget.TextOpts.MaxWidth(float64(w-52))),
		)
	}
	label(t("newGameTitle"), 22)
	if h.message != "" {
		label(t(h.message), 14)
	}
	label(fmt.Sprintf("%s · %04d-%02d", g.Frame.Player.Name, g.Frame.Month.Year, g.Frame.Month.Month), 15)
	label(t("newGameWarning"), 15)
	label(t("newGameStart"), 15)
	r := image.Rect(x, y, x+w-20, y+height-124)
	scroll := widget.NewScrollContainer(
		widget.ScrollContainerOpts.Content(content),
		widget.ScrollContainerOpts.StretchContentWidth(),
		widget.ScrollContainerOpts.Image(
			&widget.ScrollContainerImage{
				Idle: uimage.NewNineSliceColor(p.Control),
				Mask: uimage.NewNineSliceColor(color.White),
			},
		),
		widget.ScrollContainerOpts.WidgetOpts(widget.WidgetOpts.LayoutData(r)),
	)
	scroll.SetLocation(r)
	if h.scroll != nil {
		scroll.ScrollTop = h.scroll.ScrollTop
	}
	h.scroll = scroll
	root.AddChild(scroll)
	h.configureSidebarScroll(g, root)
	buttons := widget.NewContainer(
		widget.ContainerOpts.Layout(
			widget.NewRowLayout(
				widget.RowLayoutOpts.Direction(widget.DirectionVertical),
				widget.RowLayoutOpts.Padding(widget.NewInsetsSimple(12)),
				widget.RowLayoutOpts.Spacing(8),
			),
		),
		widget.ContainerOpts.WidgetOpts(widget.WidgetOpts.LayoutData(image.Rect(x, y+height-124, x+w, y+height))),
	)
	for _, action := range []struct {
		key string
		run func()
	}{
		{"cancel", func() { h.cancelNewGame(g) }},
		{"newGameConfirm", func() { h.send(g, gameapi.NewGame) }},
	} {
		fn := action.run
		b := widget.NewButton(
			widget.ButtonOpts.WidgetOpts(
				widget.WidgetOpts.MinSize(44, 46),
				widget.WidgetOpts.LayoutData(widget.RowLayoutData{Stretch: true}),
			),
			widget.ButtonOpts.Image(
				&widget.ButtonImage{
					Idle:    uimage.NewNineSliceColor(p.Selected),
					Hover:   uimage.NewNineSliceColor(p.Hover),
					Pressed: uimage.NewNineSliceColor(p.Pressed),
				},
			),
			widget.ButtonOpts.Text(t(action.key), face(15), &widget.ButtonTextColor{Idle: p.Text}),
			widget.ButtonOpts.ClickedHandler(func(*widget.ButtonClickedEventArgs) { fn() }),
		)
		b.GetWidget().Disabled = h.busy
		buttons.AddChild(b)
	}
	root.AddChild(buttons)
	h.ui = &ebitenui.UI{Container: root}
	if !h.busy {
		h.ui.SetFocusedWidget(buttons.Children()[0].(*widget.Button))
	}
}
