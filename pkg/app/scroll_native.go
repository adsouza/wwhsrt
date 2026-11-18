//go:build !js

package app

import (
	"image"

	"math"

	uimage "github.com/ebitenui/ebitenui/image"
	"github.com/ebitenui/ebitenui/widget"
	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/inpututil"
	"wwhsrt/pkg/ui"
)

const sidebarScrollSteps = 10000

func (h *host) configureSidebarScroll(g *Game, root *widget.Container) {
	scroll := h.scroll
	h.scrollbar = h.addScrollBar(g, root, scroll, func() bool { return h.scroll == scroll })
}

func (h *host) addScrollBar(g *Game, root *widget.Container, scroll *widget.ScrollContainer, current func() bool) *widget.Slider {
	p := ui.PaletteFor(g.State)
	r := scroll.ViewRect()
	_, height := scroll.PreferredSize()
	barRect := image.Rect(r.Max.X+h.px(3), r.Min.Y+h.px(4), r.Max.X+h.px(17), r.Max.Y-h.px(4))
	var bar *widget.Slider
	bar = widget.NewSlider(
		widget.SliderOpts.WidgetOpts(widget.WidgetOpts.LayoutData(barRect)),
		widget.SliderOpts.Direction(widget.DirectionVertical),
		widget.SliderOpts.MinMax(0, sidebarScrollSteps),
		widget.SliderOpts.FixedHandleSize(max(min(h.px(44), barRect.Dy()/2), r.Dy()*r.Dy()/max(1, height))),
		widget.SliderOpts.PageSizeFunc(func() int {
			_, ch := scroll.PreferredSize()
			return max(1, int(float64(sidebarScrollSteps)*float64(r.Dy())*.9/float64(max(1, ch-r.Dy()))))
		}),
		widget.SliderOpts.Images(
			&widget.SliderTrackImage{Idle: uimage.NewNineSliceColor(p.Control)},
			&widget.ButtonImage{
				Idle:    uimage.NewNineSliceColor(p.Accent),
				Hover:   uimage.NewNineSliceColor(p.Text),
				Pressed: uimage.NewNineSliceColor(p.Pressed),
			},
		),
		widget.SliderOpts.ChangedHandler(func(a *widget.SliderChangedEventArgs) {
			if current() && !g.dirty && !g.State.Expanded {
				setScroll(scroll, bar, float64(a.Current)/sidebarScrollSteps)
			}
		}))
	// ScrollContainer only clips/positions content; it supplies no wheel policy.
	// Keep fractional trackpad deltas and scroll a fixed pixel distance per notch.
	scroll.GetWidget().ScrolledEvent.AddHandler(func(args interface{}) {
		if a, ok := args.(*widget.WidgetScrolledEventArgs); ok && current() && !g.dirty && !g.State.Expanded {
			if overflow := height - r.Dy(); overflow > 0 {
				setScroll(scroll, bar, scroll.ScrollTop-a.Y*float64(h.px(48))/float64(overflow))
			}
		}
	})
	bar.SetLocation(barRect)
	if height <= r.Dy() {
		bar.GetWidget().SetVisibility(widget.Visibility_Hide)
	}
	setScroll(scroll, bar, scroll.ScrollTop)
	root.AddChild(bar)
	return bar
}

func (h *host) setSidebarScroll(value float64) {
	setScroll(h.scroll, h.scrollbar, value)
}
func setScroll(scroll *widget.ScrollContainer, bar *widget.Slider, value float64) {
	_, height := scroll.PreferredSize()
	if height <= scroll.ViewRect().Dy() {
		value = 0
	}
	scroll.ScrollTop = math.Max(0, math.Min(1, value))
	if bar != nil {
		bar.Current = int(math.Round(scroll.ScrollTop * sidebarScrollSteps))
	}
}

func (h *host) scrollSidebar(pixels float64) {
	_, height := h.scroll.PreferredSize()
	if overflow := height - h.scroll.ViewRect().Dy(); overflow > 0 {
		h.setSidebarScroll(h.scroll.ScrollTop + pixels/float64(overflow))
	}
}

func (h *host) sidebarKeyboard(g *Game) {
	editing := h.routeSearch != nil && h.routeSearch.IsFocused() ||
		g.State.Help.Open && h.helpSearchField.IsFocused() ||
		g.State.View == "company" && (h.field.IsFocused() || h.loanInput != nil && h.loanInput.IsFocused())
	if h.scroll == nil || g.State.Expanded || editing {
		return
	}
	x, y := ebiten.CursorPosition()
	if g.State.View != "report" && !image.Pt(x, y).In(h.scroll.ViewRect()) && !h.scrollbar.IsFocused() {
		return
	}
	switch {
	case inpututil.IsKeyJustPressed(ebiten.KeyHome):
		h.setSidebarScroll(0)
	case inpututil.IsKeyJustPressed(ebiten.KeyEnd):
		h.setSidebarScroll(1)
	case inpututil.IsKeyJustPressed(ebiten.KeyPageDown):
		h.scrollSidebar(float64(h.scroll.ViewRect().Dy()) * .9)
	case inpututil.IsKeyJustPressed(ebiten.KeyPageUp):
		h.scrollSidebar(-float64(h.scroll.ViewRect().Dy()) * .9)
	}
}
