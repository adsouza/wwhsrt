//go:build !js

package app

import (
	"fmt"
	"image"
	"image/color"
	"math"

	uimage "github.com/ebitenui/ebitenui/image"
	"github.com/ebitenui/ebitenui/widget"
	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/text/v2"
	"github.com/hajimehoshi/ebiten/v2/vector"
	"wwhsrt/pkg/ui"
)

type comparingServicesButton struct {
	*widget.Button
	game *Game
}

func (b *comparingServicesButton) Render(screen *ebiten.Image) {
	b.Button.Render(screen)
	r := b.GetWidget().Rect
	position := 0.5
	if b.game.State.Motion {
		position = (1 - math.Cos(b.game.tick*math.Pi/48)) / 2
	}
	x := float64(r.Min.X) + float64(r.Dx())*(0.05+0.6*position)
	height := b.game.host.px(3)
	vector.FillRect(screen, float32(x), float32(r.Max.Y-2*height), float32(r.Dx())*0.3, float32(height), ui.PaletteFor(b.game.State).Accent, false)
}

func (h *host) addProspectBoard(g *Game, root *widget.Container) {
	r := h.workspace.Prospects
	if r.Empty() {
		return
	}
	p := ui.PaletteFor(g.State)
	t := func(k string) string { return ui.Text(g.State.Locale, k) }
	rows := ui.RankedProspects(g.Frame)
	choose := func(id string) {
		if ui.SelectProspect(&g.State, g.Frame, id) {
			if h.scroll != nil {
				h.setSidebarScroll(0)
			}
			h.workspaceChanged(g)
			g.focusSelected()
		}
	}

	head := widget.NewContainer(widget.ContainerOpts.Layout(placed{}),
		widget.ContainerOpts.BackgroundImage(uimage.NewNineSliceColor(p.Surface)),
		widget.ContainerOpts.WidgetOpts(widget.WidgetOpts.LayoutData(r)))
	root.AddChild(head)
	face := h.face(g, 13)
	actionWidth := min(r.Dx()/2, h.px(220))
	w := r.Dx() - actionWidth - h.px(22)
	title := ui.WrapText(t("rankedProspects"), float64(w), func(s string) float64 { return text.Advance(s, face) })
	headingRect := image.Rect(h.px(8), h.px(3), w+h.px(8), h.px(46))
	head.AddChild(widget.NewText(
		widget.TextOpts.Text(title, &face, p.Text),
		widget.TextOpts.WidgetOpts(widget.WidgetOpts.LayoutData(headingRect)),
	))
	next := "earnWithService"
	canBuild := len(rows) > 0 && rows[0].Cash <= g.Frame.Player.Cash
	canSuggest := g.canSuggestService()
	if !canBuild && !canSuggest {
		next = "finance"
	}
	if canBuild {
		next = "reviewCheapest"
	}
	if h.pendingSearch != nil {
		next = "comparingServices"
	}
	if ui.CampaignFinished(g.Frame) {
		next = "campaignEndTitle"
	}
	b := h.fixedButton(g, t(next), image.Rect(r.Dx()-actionWidth, h.px(1), r.Dx()-h.px(4), h.px(45)), func() {
		if canBuild {
			choose(rows[0].ID)
			return
		}
		if canSuggest {
			h.pendingSearch = g.searchServices()
			g.dirty = true
		} else {
			g.State.View, g.State.CompanyPane = "company", "finance"
			h.workspaceChanged(g)
		}
	})
	b.GetWidget().Disabled = h.busy || h.pendingSearch != nil || ui.CampaignFinished(g.Frame)
	b.SetImage(&widget.ButtonImage{
		Idle:    uimage.NewBorderedNineSliceColor(p.Selected, p.Accent, 1),
		Hover:   uimage.NewNineSliceColor(p.Hover),
		Pressed: uimage.NewNineSliceColor(p.Pressed),
	})
	if h.pendingSearch != nil {
		head.AddChild(&comparingServicesButton{Button: b, game: g})
	} else {
		head.AddChild(b)
	}
	h.actions["next-investment"] = b

	layout := widget.NewRowLayout(
		widget.RowLayoutOpts.Direction(widget.DirectionVertical),
		widget.RowLayoutOpts.Spacing(h.px(2)),
	)
	list := widget.NewContainer(widget.ContainerOpts.Layout(layout))
	width := r.Dx() - h.px(26)
	if len(rows) == 0 {
		list.AddChild(widget.NewText(widget.TextOpts.Text(t("noProspects"), &face, p.Text), widget.TextOpts.MaxWidth(float64(width))))
	}
	normal := &widget.ButtonImage{
		Idle:    uimage.NewNineSliceColor(p.Control),
		Hover:   uimage.NewNineSliceColor(p.Hover),
		Pressed: uimage.NewNineSliceColor(p.Pressed),
	}
	selected, _ := ui.Pair(g.Frame, g.State.FromCity, g.State.ToCity)
	for i, row := range rows {
		title := ui.ProspectName(g.State.Locale, row.ID, row.From, row.To) + "  ·  " + ui.Cash(row.Cash)
		if row.Cash <= g.Frame.Player.Cash {
			title += "  ·  " + t("withinBudget")
		} else if i == 0 {
			title += "\n" + fmt.Sprintf(t("saveForLine"), ui.Cash(row.Cash-g.Frame.Player.Cash))
		}
		label := ui.WrapText(title, float64(width-h.px(18)), func(s string) float64 { return text.Advance(s, face) })
		b := widget.NewButton(
			widget.ButtonOpts.WidgetOpts(
				widget.WidgetOpts.MinSize(h.px(44), h.px(44)),
				widget.WidgetOpts.LayoutData(widget.RowLayoutData{Stretch: true, MaxWidth: width}),
			),
			widget.ButtonOpts.Image(normal),
			widget.ButtonOpts.TextPosition(widget.TextPositionStart, widget.TextPositionCenter),
			widget.ButtonOpts.Text(label, &face, &widget.ButtonTextColor{Idle: p.Text}),
			widget.ButtonOpts.TextPadding(widget.NewInsetsSimple(h.px(4))),
			widget.ButtonOpts.ClickedHandler(func(*widget.ButtonClickedEventArgs) { choose(row.ID) }))
		if g.State.View == "build" && row.ID == selected.ID {
			b.SetImage(&widget.ButtonImage{Idle: uimage.NewNineSliceColor(p.Selected), Hover: normal.Hover, Pressed: normal.Pressed})
		}
		list.AddChild(b)
		h.actions["prospect-"+row.ID] = b
	}
	rect := image.Rect(r.Min.X+h.px(4), r.Min.Y+h.px(47), r.Max.X-h.px(20), r.Max.Y)
	scroll := widget.NewScrollContainer(widget.ScrollContainerOpts.Content(list), widget.ScrollContainerOpts.StretchContentWidth(),
		widget.ScrollContainerOpts.Image(&widget.ScrollContainerImage{Idle: uimage.NewNineSliceColor(p.Background), Mask: uimage.NewNineSliceColor(color.White)}),
		widget.ScrollContainerOpts.WidgetOpts(widget.WidgetOpts.LayoutData(rect)))
	shareMaskBuffer(scroll, &h.scrollMask)
	scroll.SetLocation(rect)
	if h.prospectScroll != nil {
		scroll.ScrollTop = h.prospectScroll.ScrollTop
	}
	h.prospectScroll = scroll
	root.AddChild(scroll)
	h.addScrollBar(g, root, scroll, func() bool { return h.prospectScroll == scroll })
}

func (h *host) finishServiceSearch(g *Game) {
	job := h.pendingSearch
	if job == nil {
		return
	}
	select {
	case <-job.done:
		h.pendingSearch = nil
		g.dirty = true
	default:
		return
	}
	if h.busy || h.confirmNewGame || job.checkpoint != g.Frame.Checkpoint {
		return
	}
	plan := job.plan
	if plan.CorridorID == "" {
		g.State.View, g.State.CompanyPane = "company", "finance"
		h.message = "noImprovingService"
	} else {
		h.message = "frequencyOptimized"
		ui.DraftService(&g.State, g.Frame, plan.CorridorID)
		g.State.Frequency, g.State.Fare, g.State.ReplaceServiceID = plan.Frequency, plan.Fare, plan.ReplaceServiceID
		g.State.View, g.State.ServicePane, g.State.ServiceQuery = "services", "plan", ""
		if h.scroll != nil {
			h.setSidebarScroll(0)
		}
	}
	h.workspaceChanged(g)
	if plan.CorridorID != "" {
		g.focusSelected()
	}
}
