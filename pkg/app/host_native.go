//go:build !js

package app

import (
	"encoding/json"
	"fmt"
	"image"
	"image/color"
	"log/slog"
	"os"
	"path/filepath"
	"time"
	"unicode/utf8"

	"github.com/ebitenui/ebitenui"
	uimage "github.com/ebitenui/ebitenui/image"
	"github.com/ebitenui/ebitenui/widget"
	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/exp/textinput"
	"github.com/hajimehoshi/ebiten/v2/inpututil"
	"github.com/hajimehoshi/ebiten/v2/text/v2"
	"github.com/hajimehoshi/ebiten/v2/vector"
	"wwhsrt/internal/adapters/filestore"
	"wwhsrt/internal/application"
	"wwhsrt/pkg/gameapi"
	"wwhsrt/pkg/mapview"
	"wwhsrt/pkg/ui"
)

var preferencesPath string

func OpenHost() (gameapi.Game, func(), error) {
	dir, e := os.UserConfigDir()
	if e != nil {
		return nil, nil, e
	}
	dir = filepath.Join(dir, "wwhsrt", "first-business")
	if custom := os.Getenv("WWHSRT_DATA_DIR"); custom != "" {
		dir = custom
	}
	path := filepath.Join(dir, "campaign.json")
	preferencesPath = filepath.Join(dir, "ui.json")
	unlock, e := filestore.Lock(path)
	if e != nil {
		return nil, nil, fmt.Errorf("game already open or storage unavailable: %w", e)
	}
	port, e := application.Open(filestore.Store{Path: path}, slog.New(slog.NewJSONHandler(os.Stderr, nil)))
	if e != nil {
		unlock()
		return nil, nil, e
	}
	return port, unlock, nil
}
func StartupError(e error) {
	slog.Error("startup.failed", "error", e)
	fmt.Fprintln(os.Stderr, "Saved data preserved. Check storage or close the other instance.")
}

type host struct {
	ui               *ebitenui.UI
	message          string
	busy             bool
	result           chan commandResult
	field            textinput.Field
	nameButton       *widget.Button
	scroll           *widget.ScrollContainer
	scrollbar        *widget.Slider
	lastView         string
	loanInput        *widget.TextInput
	confirmNewGame   bool
	window           *windowPlacement
	windowConfigured bool
	windowClosing    bool
	windowDirty      bool
	windowPoll       time.Time
	windowChanged    time.Time
	windowAttempt    time.Time
	appearanceMode   string
	appearancePoll   time.Time
}
type commandResult struct {
	frame gameapi.Frame
	err   error
	kind  string
}
type placed struct{}

func (placed) PreferredSize([]widget.PreferredSizeLocateableWidget) (int, int) { return 0, 0 }
func (placed) Layout(ws []widget.PreferredSizeLocateableWidget, r image.Rectangle) {
	for _, w := range ws {
		w.SetLocation(w.GetWidget().LayoutData.(image.Rectangle).Add(r.Min))
	}
}
func (h *host) init(g *Game) {
	h.result = make(chan commandResult, 1)
	if b, e := os.ReadFile(preferencesPath); e == nil {
		var s nativePreferences
		if json.Unmarshal(b, &s) == nil {
			g.State = s.State
			h.window = s.Window
		}
	}
	g.State.Locale = ui.Locale(g.State.Locale)
	if g.State.FromCity == "" {
		g.State.FromCity = "tsuruga"
		g.State.ToCity = "osaka"
		g.State.Choosing = "from"
	}
	if g.State.ServiceCorridor == "" {
		ui.DraftService(&g.State, g.Frame, "tokyo-nagoya")
	}
	if g.State.View != "report" && g.State.View != "company" && g.State.View != "build" && g.State.View != "services" {
		g.State.View = "map"
	}
	if g.State.DraftName == "" {
		g.State.DraftName = g.Frame.Player.Name
	}
	if g.State.LoanAmount == "" {
		g.State.LoanAmount = ui.DollarInput(ui.FundingShortfall(g.Frame, g.State))
		if g.State.LoanAmount == "0.00" {
			g.State.LoanAmount = "1000000.00"
		}
	}
	h.field.SetTextAndSelection(g.State.DraftName, len(g.State.DraftName), len(g.State.DraftName))
}
func (h *host) persist(g *Game) {
	b, _ := json.Marshal(nativePreferences{State: g.State, Window: h.window})
	if e := (filestore.Store{Path: preferencesPath}).Commit(b); e != nil {
		h.message = "storage_failed"
		slog.Warn("preferences.commit_failed")
	} else {
		h.windowDirty = false
	}
}
func (h *host) changed(g *Game) { h.persist(g) }
func (h *host) layout(g *Game) {
	w, hgt := g.width-360, g.height
	if g.width < 800 {
		w = g.width
		hgt = max(180, g.height/3)
	}
	if g.State.Expanded {
		w = g.width
		hgt = g.height
	}
	g.Camera.Width = float64(max(1, w))
	g.Camera.Height = float64(max(1, hgt))
	portrait := g.height > g.width
	if g.Camera.Zoom == 0 || g.State.Portrait != portrait {
		g.Camera = g.Map.Camera(g.Camera.Width, g.Camera.Height, portrait)
	}
	g.State.Portrait = portrait
}
func (h *host) send(g *Game, kind string) {
	if h.busy || h.windowClosing || h.confirmNewGame && kind != gameapi.NewGame {
		return
	}
	if (kind == gameapi.AdvanceMonth || kind == gameapi.AdvanceYear) && g.State.View == "services" && ui.ServiceDirty(g.State, g.Frame) {
		h.message = "draft_not_applied"
		g.dirty = true
		return
	}
	h.busy = true
	g.dirty = true
	c := gameapi.Command{Kind: kind, Revision: g.Frame.Revision, Name: g.State.DraftName, CorridorID: g.State.ServiceCorridor, Frequency: g.State.Frequency, Fare: g.State.Fare}
	if kind == gameapi.TakeLoan {
		amount, ok := ui.ParseDollars(g.State.LoanAmount)
		if !ok || amount <= 0 {
			h.busy = false
			h.message = "invalid_loan_amount"
			return
		}
		c.Amount = amount
	}
	if kind == gameapi.BuildLine {
		co, ok := ui.Pair(g.Frame, g.State.FromCity, g.State.ToCity)
		if !ok {
			h.busy = false
			h.message = "no_route"
			return
		}
		c.CorridorID = co.ID
		c.AlignmentID = g.State.AlignmentID
		if c.AlignmentID == "" {
			c.AlignmentID = co.Alignments[0].ID
		}
	}
	go func() { f, e := g.Port.Apply(c); h.result <- commandResult{f, e, kind} }()
}
func (h *host) update(g *Game) {
	h.refreshAppearance(g)
	loanBefore := g.State.LoanAmount
	select {
	case r := <-h.result:
		h.busy = false
		g.Frame = r.frame
		if r.err == nil && r.kind == gameapi.NewGame {
			h.confirmNewGame = false
			g.newGamePresentation()
			h.field.SetTextAndSelection(g.State.DraftName, len(g.State.DraftName), len(g.State.DraftName))
			h.scroll = nil
			h.layout(g)
			h.persist(g)
		}
		if r.err == nil && r.kind == gameapi.AdvanceMonth {
			g.State.View = "report"
		}
		h.message = r.kind
		if r.err != nil {
			h.message = r.err.Error()
		}
		g.dirty = true
	default:
	}
	if g.dirty && g.width > 0 {
		h.build(g)
		g.dirty = false
	}
	if h.confirmNewGame {
		if inpututil.IsKeyJustPressed(ebiten.KeyEscape) && !h.busy {
			h.cancelNewGame(g)
		}
		if h.ui != nil {
			h.ui.Update()
		}
		if !g.dirty {
			h.sidebarKeyboard(g)
		}
		return
	}
	if inpututil.IsKeyJustPressed(ebiten.KeyEscape) && g.State.Expanded {
		g.State.Expanded = false
		h.layout(g)
		g.dirty = true
	}
	if h.ui != nil {
		h.ui.Update()
	}
	if !g.dirty {
		h.sidebarKeyboard(g)
	}
	if g.State.View == "company" && h.loanInput != nil && h.loanInput.IsFocused() {
		h.field.Blur()
	}
	if g.State.View == "company" && h.loanInput != nil && h.loanInput.IsFocused() && (ebiten.IsKeyPressed(ebiten.KeyMeta) || ebiten.IsKeyPressed(ebiten.KeyControl)) && inpututil.IsKeyJustPressed(ebiten.KeyA) {
		h.loanInput.SetText(loanBefore)
		h.loanInput.SelectAll()
	}
	if h.field.IsFocused() {
		bounds := image.Rect(g.width-310, 180, g.width-290, 202)
		if h.nameButton != nil {
			bounds = h.nameButton.GetWidget().Rect
		}
		handled, e := h.field.HandleInputWithBounds(bounds)
		if e != nil {
			h.message = "invalid_name"
		}
		if !handled && h.field.UncommittedTextLengthInBytes() == 0 {
			value := h.field.Text()
			a, b := h.field.Selection()
			switch {
			case (ebiten.IsKeyPressed(ebiten.KeyMeta) || ebiten.IsKeyPressed(ebiten.KeyControl)) && inpututil.IsKeyJustPressed(ebiten.KeyA):
				h.field.SetSelection(0, len(value))
			case inpututil.IsKeyJustPressed(ebiten.KeyBackspace):
				if a == b && a > 0 {
					_, n := utf8.DecodeLastRuneInString(value[:a])
					a -= n
				}
				value = value[:a] + value[b:]
				h.field.SetTextAndSelection(value, a, a)
			case inpututil.IsKeyJustPressed(ebiten.KeyLeft):
				if a > 0 {
					_, n := utf8.DecodeLastRuneInString(value[:a])
					a -= n
				}
				h.field.SetSelection(a, a)
			case inpututil.IsKeyJustPressed(ebiten.KeyRight):
				if b < len(value) {
					_, n := utf8.DecodeRuneInString(value[b:])
					b += n
				}
				h.field.SetSelection(b, b)
			case inpututil.IsKeyJustPressed(ebiten.KeyEnter) || inpututil.IsKeyJustPressed(ebiten.KeyEscape):
				h.field.Blur()
			}
		}
		if h.field.Text() != g.State.DraftName {
			g.State.DraftName = h.field.Text()
			h.persist(g)
		}
		if h.nameButton != nil {
			h.nameButton.SetText(h.field.TextForRendering() + " │")
		}
	}
}
func (h *host) draw(g *Game, screen *ebiten.Image) {
	p := ui.PaletteFor(g.State)
	screen.Fill(p.Background)
	viewport := screen.SubImage(image.Rect(0, 0, int(g.Camera.Width), int(g.Camera.Height))).(*ebiten.Image)
	g.Map.Draw(viewport, g.Camera, g.Frame, g.State, g.tick)
	caption := ui.Text(g.State.Locale, "illustration")
	vector.DrawFilledRect(viewport, 12, float32(g.Camera.Height-36), float32(g.Camera.Width-24), 25, p.Surface, true)
	g.Map.Label(viewport, caption, 20, g.Camera.Height-32, 11, p.Text)
	if h.ui != nil {
		h.ui.Draw(screen)
	}
}
func (h *host) build(g *Game) {
	p := ui.PaletteFor(g.State)
	if h.confirmNewGame {
		h.buildNewGame(g)
		return
	}
	t := func(k string) string { return ui.Text(g.State.Locale, k) }
	face := func(n float64) *text.Face { var f text.Face = &text.GoTextFace{Source: g.Map.Font, Size: n}; return &f }
	root := widget.NewContainer(widget.ContainerOpts.Layout(placed{}))
	x, y, w, height := int(g.Camera.Width), 0, 360, g.height
	if g.width < 800 {
		x = 0
		y = int(g.Camera.Height)
		w = g.width
		height = g.height - y
	}
	contentWidth := w - 20 // Reserve a separate gutter for the scrollbar.
	footerHeight := 120
	if len(g.Frame.IdleLines) > 0 {
		footerHeight = 184
	}
	content := widget.NewContainer(widget.ContainerOpts.Layout(widget.NewRowLayout(widget.RowLayoutOpts.Direction(widget.DirectionVertical), widget.RowLayoutOpts.Padding(widget.NewInsetsSimple(20)), widget.RowLayoutOpts.Spacing(12))))
	wrap := func(value string, size float64, width int) string {
		return ui.WrapText(value, float64(width), func(s string) float64 { return text.Advance(s, *face(size)) })
	}
	label := func(value string, size float64) {
		content.AddChild(widget.NewText(widget.TextOpts.Text(wrap(value, size, contentWidth-40), face(size), p.Text), widget.TextOpts.MaxWidth(float64(contentWidth-40))))
	}
	button := func(title string, fn func()) *widget.Button {
		b := widget.NewButton(widget.ButtonOpts.WidgetOpts(widget.WidgetOpts.MinSize(44, 44), widget.WidgetOpts.LayoutData(widget.RowLayoutData{Stretch: true, MaxWidth: contentWidth - 40})), widget.ButtonOpts.Image(&widget.ButtonImage{Idle: uimage.NewNineSliceColor(p.Control), Hover: uimage.NewNineSliceColor(p.Selected), Pressed: uimage.NewNineSliceColor(p.Pressed), Disabled: uimage.NewNineSliceColor(p.Disabled)}), widget.ButtonOpts.Text(wrap(title, 15, contentWidth-60), face(15), &widget.ButtonTextColor{Idle: p.Text, Disabled: p.Muted}), widget.ButtonOpts.TextPadding(widget.NewInsetsSimple(10)), widget.ButtonOpts.ClickedHandler(func(*widget.ButtonClickedEventArgs) { fn() }))
		content.AddChild(b)
		return b
	}
	label(fmt.Sprintf("%04d-%02d  ·  %s", g.Frame.Month.Year, g.Frame.Month.Month, t("offline")), 14)
	label(g.Frame.Player.Name, 24)
	label(t("cash")+"\n"+ui.Cash(g.Frame.Player.Cash), 16)
	if len(g.Frame.IdleLines) > 0 {
		content.AddChild(h.idleNotice(g, contentWidth-40))
	}
	nav := widget.NewContainer(widget.ContainerOpts.WidgetOpts(widget.WidgetOpts.LayoutData(widget.RowLayoutData{Stretch: true})), widget.ContainerOpts.Layout(widget.NewGridLayout(widget.GridLayoutOpts.Columns(3), widget.GridLayoutOpts.Stretch([]bool{true, true, true}, nil), widget.GridLayoutOpts.Spacing(6, 6))))
	for _, view := range []string{"map", "build", "services", "report", "company"} {
		v := view
		tabImage := &widget.ButtonImage{Idle: uimage.NewNineSliceColor(p.Control), Hover: uimage.NewNineSliceColor(p.Selected), Pressed: uimage.NewNineSliceColor(p.Control)}
		tabInk := p.Text
		if v == g.State.View {
			tabImage = &widget.ButtonImage{
				Idle:    uimage.NewBorderedNineSliceColor(p.Selected, p.Accent, 1),
				Hover:   uimage.NewBorderedNineSliceColor(p.Hover, p.Accent, 1),
				Pressed: uimage.NewBorderedNineSliceColor(p.Pressed, p.Accent, 1),
			}
			tabInk = p.Accent
		}
		b := widget.NewButton(widget.ButtonOpts.WidgetOpts(widget.WidgetOpts.MinSize(44, 44)), widget.ButtonOpts.Image(tabImage), widget.ButtonOpts.Text(t(v), face(12), &widget.ButtonTextColor{Idle: tabInk}), widget.ButtonOpts.ClickedHandler(func(*widget.ButtonClickedEventArgs) { h.field.Blur(); g.State.View = v; g.dirty = true; h.persist(g) }))
		nav.AddChild(b)
	}
	content.AddChild(nav)

	if g.State.View == "map" {
		if !g.State.HintDismissed {
			label(t("hint"), 15)
			button(t("dismiss"), func() { g.State.HintDismissed = true; g.dirty = true; h.persist(g) })
		}
		label(t("cities"), 18)
		for _, city := range g.Frame.Cities {
			city := city
			name := t(string(city.ID))
			if city.ID == g.State.Selected {
				name = "● " + name
			}
			button(name, func() { g.State.Selected = city.ID; g.dirty = true; h.persist(g) })
		}
		for _, city := range g.Frame.Cities {
			if city.ID == g.State.Selected {
				label(fmt.Sprintf("%s · %.4f°, %.4f°", t(string(city.ID)), city.Lat, city.Lon), 14)
			}
		}
	} else if g.State.View == "build" {
		h.buildPlanner(g, t, label, button)
	} else if g.State.View == "services" {
		h.buildServices(g, t, label, button)
	} else if g.State.View == "report" {
		label(t("report"), 22)
		if len(g.Frame.Reports) == 0 {
			label(t("noReport"), 16)
		} else {
			r := g.Frame.Reports[len(g.Frame.Reports)-1]
			label(fmt.Sprintf("%04d-%02d", r.Month.Year, r.Month.Month), 18)
			for _, row := range []struct {
				k string
				v gameapi.Money
			}{{"opening", r.OpeningCash}, {"revenue", r.Revenue}, {"runningCosts", r.RunningCosts}, {"accessCosts", r.AccessCosts}, {"stationCosts", r.StationCosts}, {"leaseCosts", r.LeaseCosts}, {"maintenance", r.Maintenance}, {"interest", r.Interest}, {"construction", r.Construction}, {"grants", r.Grants}, {"borrowed", r.Borrowed}, {"repaid", r.Repaid}, {"closing", r.ClosingCash}} {
				label(t(row.k)+"\n"+ui.Cash(row.v), 16)
			}
		}
		if len(g.Frame.Reports) > 0 {
			label(fmt.Sprintf("%s: %d", t("passengers"), g.Frame.Reports[len(g.Frame.Reports)-1].Passengers), 16)
		}
		label(t("reportHint"), 15)
	} else {
		button(t("reset"), func() { h.openNewGame(g) }).GetWidget().Disabled = h.busy
		button(t("appearance")+" · "+t("appearance_"+ui.AppearanceMode(g.State.Appearance)), func() {
			g.State.Appearance = ui.NextAppearance(g.State.Appearance)
			h.refreshAppearance(g)
			g.dirty = true
			h.persist(g)
		})
		label(t("finance"), 22)
		label(t("debt")+"\n"+ui.Cash(g.Frame.Debt), 16)
		h.financeForm(g, t, label, button, content, contentWidth-40)
		button(t("repay"), func() { h.send(g, gameapi.RepayLoan) })
		label(t("loanTerms"), 14)
		label(t("terrainCredits"), 12)
		label(t("companyName"), 18)
		h.nameButton = button(g.State.DraftName, func() {
			if h.loanInput != nil {
				h.loanInput.Focus(false)
			}
			h.field.Focus()
			h.field.SetSelection(0, len(h.field.Text()))
		})
		label(t("nameHint"), 14)
		button(t("applyName"), func() {
			if h.field.UncommittedTextLengthInBytes() > 0 {
				return
			}
			h.field.Blur()
			h.send(g, gameapi.RenameCompany)
		})
	}
	button(t("showMap"), func() { g.State.Expanded = true; h.layout(g); g.dirty = true })
	button(t("advanceYear"), func() { h.send(g, gameapi.AdvanceYear) })
	button(t("fit"), func() { g.Camera = g.Map.Camera(g.Camera.Width, g.Camera.Height, g.State.Portrait) })
	button(t("focusRoute"), func() { g.focusSelected() })
	button(t("zoomIn"), func() { g.Camera.ZoomAt(1.3, mapview.Point{X: g.Camera.Width / 2, Y: g.Camera.Height / 2}) })
	button(t("zoomOut"), func() { g.Camera.ZoomAt(1/1.3, mapview.Point{X: g.Camera.Width / 2, Y: g.Camera.Height / 2}) })
	terrainTitle := t("terrain")
	if !g.State.HideTerrain {
		terrainTitle = "● " + terrainTitle
	}
	button(terrainTitle, func() { g.State.HideTerrain = !g.State.HideTerrain; g.dirty = true; h.persist(g) })
	motionState := t("motionOff")
	if g.State.Motion {
		motionState = t("motionOn")
	}
	motion := button(t("motion")+" · "+motionState, func() { g.State.Motion = !g.State.Motion; g.dirty = true; h.persist(g) })
	if g.State.Motion {
		motion.SetImage(&widget.ButtonImage{
			Idle:    uimage.NewBorderedNineSliceColor(p.Selected, p.Accent, 1),
			Hover:   uimage.NewBorderedNineSliceColor(p.Hover, p.Accent, 1),
			Pressed: uimage.NewBorderedNineSliceColor(p.Pressed, p.Accent, 1),
		})
	}
	button("English / 简体中文", func() {
		if g.State.Locale == "en" {
			g.State.Locale = "zh-Hans"
		} else {
			g.State.Locale = "en"
		}
		g.dirty = true
		h.persist(g)
	})
	label(t("illustration"), 14)
	label(t("scope"), 13)
	oldScroll := 0.0
	if h.scroll != nil && h.lastView == g.State.View {
		oldScroll = h.scroll.ScrollTop
	}
	scrollRect := image.Rect(x, y, x+contentWidth, y+max(1, height-footerHeight))
	scroll := widget.NewScrollContainer(widget.ScrollContainerOpts.Content(content), widget.ScrollContainerOpts.StretchContentWidth(), widget.ScrollContainerOpts.Image(&widget.ScrollContainerImage{Idle: uimage.NewNineSliceColor(p.Background), Mask: uimage.NewNineSliceColor(color.White)}), widget.ScrollContainerOpts.WidgetOpts(widget.WidgetOpts.LayoutData(scrollRect)))
	scroll.SetLocation(scrollRect)
	scroll.ScrollTop = oldScroll
	h.scroll = scroll
	h.lastView = g.State.View
	root.AddChild(scroll)
	footer := widget.NewContainer(widget.ContainerOpts.Layout(widget.NewRowLayout(widget.RowLayoutOpts.Direction(widget.DirectionVertical), widget.RowLayoutOpts.Padding(widget.NewInsetsSimple(12)), widget.RowLayoutOpts.Spacing(6))), widget.ContainerOpts.BackgroundImage(uimage.NewNineSliceColor(p.Surface)), widget.ContainerOpts.WidgetOpts(widget.WidgetOpts.LayoutData(image.Rect(x, y+height-footerHeight, x+w, y+height))))
	if len(g.Frame.IdleLines) > 0 {
		footer.AddChild(h.idleReviewButton(g, fmt.Sprintf("%s · %d · %s%s", t("idleShort"), len(g.Frame.IdleLines), ui.Cash(ui.IdleCosts(g.Frame)), t("monthlyUnit")), w-24))
	}
	labelText := t("saved")
	if h.message != "" {
		labelText = t(h.message)
	}
	footer.AddChild(widget.NewText(widget.TextOpts.Text(labelText, face(13), p.Text), widget.TextOpts.MaxWidth(float64(w-24))))
	b := widget.NewButton(widget.ButtonOpts.WidgetOpts(widget.WidgetOpts.MinSize(44, 48), widget.WidgetOpts.LayoutData(widget.RowLayoutData{Stretch: true})), widget.ButtonOpts.Image(&widget.ButtonImage{Idle: uimage.NewNineSliceColor(p.Action), Hover: uimage.NewNineSliceColor(p.ActionHover), Pressed: uimage.NewNineSliceColor(p.ActionHover), Disabled: uimage.NewNineSliceColor(p.Disabled)}), widget.ButtonOpts.Text(t("advance"), face(16), &widget.ButtonTextColor{Idle: p.OnAction, Disabled: p.OnAction}), widget.ButtonOpts.ClickedHandler(func(*widget.ButtonClickedEventArgs) { h.send(g, gameapi.AdvanceMonth) }))
	b.GetWidget().Disabled = h.busy || g.Frame.MonthsResolved >= g.Frame.CampaignMonths || g.State.View == "services" && ui.ServiceDirty(g.State, g.Frame)
	timeButtons := widget.NewContainer(widget.ContainerOpts.Layout(widget.NewGridLayout(widget.GridLayoutOpts.Columns(2), widget.GridLayoutOpts.Stretch([]bool{true, true}, nil), widget.GridLayoutOpts.Spacing(6, 0))), widget.ContainerOpts.WidgetOpts(widget.WidgetOpts.LayoutData(widget.RowLayoutData{Stretch: true})))
	timeButtons.AddChild(b)
	year := widget.NewButton(widget.ButtonOpts.WidgetOpts(widget.WidgetOpts.MinSize(44, 48)), widget.ButtonOpts.Image(&widget.ButtonImage{Idle: uimage.NewNineSliceColor(p.ActionHover), Hover: uimage.NewNineSliceColor(p.Action), Pressed: uimage.NewNineSliceColor(p.ActionHover), Disabled: uimage.NewNineSliceColor(p.Disabled)}), widget.ButtonOpts.Text(t("advanceYear"), face(13), &widget.ButtonTextColor{Idle: p.OnAction, Disabled: p.OnAction}), widget.ButtonOpts.ClickedHandler(func(*widget.ButtonClickedEventArgs) { h.send(g, gameapi.AdvanceYear) }))
	year.GetWidget().Disabled = b.GetWidget().Disabled
	timeButtons.AddChild(year)
	footer.AddChild(timeButtons)
	root.AddChild(footer)
	h.configureSidebarScroll(g, root)
	if g.State.Expanded {
		root = widget.NewContainer(widget.ContainerOpts.Layout(placed{}))
		b := widget.NewButton(widget.ButtonOpts.WidgetOpts(widget.WidgetOpts.LayoutData(image.Rect(max(0, g.width-280), 12, g.width-12, 60))), widget.ButtonOpts.Image(&widget.ButtonImage{Idle: uimage.NewNineSliceColor(p.Control), Hover: uimage.NewNineSliceColor(p.Control), Pressed: uimage.NewNineSliceColor(p.Control)}), widget.ButtonOpts.Text(t("backPlanner"), face(15), &widget.ButtonTextColor{Idle: p.Text, Disabled: p.Muted}), widget.ButtonOpts.ClickedHandler(func(*widget.ButtonClickedEventArgs) { g.State.Expanded = false; h.layout(g); g.dirty = true }))
		root.AddChild(b)
	}
	h.ui = &ebitenui.UI{Container: root}
}
