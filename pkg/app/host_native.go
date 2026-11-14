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

	"github.com/ebitenui/ebitenui"
	uimage "github.com/ebitenui/ebitenui/image"
	"github.com/ebitenui/ebitenui/widget"
	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/exp/textinput"
	"github.com/hajimehoshi/ebiten/v2/inpututil"
	"github.com/hajimehoshi/ebiten/v2/text/v2"
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
	text          ui.TextScale // the ladder and step for this window; every font and chrome size asks it
	nameCache     *nameInput
	mapControls   []image.Rectangle
	combos        []*widget.ListComboButton
	comboChoices  [][]choice // what each dropdown in combos was given, in order
	menuInput     bool
	actions       map[string]*widget.Button
	primaryAction *widget.Button
	workspace     ui.WorkspaceLayout
	content       *widget.Container
	contentWidth  int
	mapImage      *ebiten.Image
	resizing      bool
	idleAction    *widget.Button

	routeBoxCache    map[string]ui.Box
	routeBoxRevision uint64
	routeBoxReady    bool
	routeChoices     ui.RouteChoices
	routeView        ui.Box
	routeViewUsed    bool
	routePendingView ui.Box
	routeStableTicks int

	helpFocus              string
	helpFocusScroll        bool
	helpOpen               bool
	routeSearchField       textinput.Field
	routeSearch            *nameInput
	helpSearchField        textinput.Field
	helpSearch             *nameInput
	helpAnchors            map[string]*widget.Button
	helpCue, helpCueAnchor string
	helpRestore            bool
	ui                     *ebitenui.UI
	message                string
	busy                   bool
	result                 chan commandResult
	field                  textinput.Field
	nameInput              *nameInput
	statusLabel            *widget.Text
	trainDescription       *widget.Text
	replaySlider           *widget.Slider
	scroll                 *widget.ScrollContainer
	scrollbar              *widget.Slider
	lastView               string
	loanInput              *widget.TextInput
	confirmNewGame         bool
	window                 *windowPlacement
	windowConfigured       bool
	windowClosing          bool
	windowDirty            bool
	windowPoll             time.Time
	windowChanged          time.Time
	windowAttempt          time.Time
	appearanceMode         string
	appearancePoll         time.Time
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
	ui.NormalizeHelp(&g.State)
	g.State.Locale = ui.Locale(g.State.Locale)
	g.State.ReplayMinute = max(0, min(1439, g.State.ReplayMinute))
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
	if g.Map != nil {
		g.Camera = g.regionCamera(g.State.Portrait)
	}
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

// face is the only place the native panels make a font: base is the size at the call site, and the
// window's ladder and text step decide what is drawn.
// px scales a layout length with the text step, so chrome keeps its proportions around larger text.
func (h *host) px(n int) int { return h.text.Px(n) }

// logicalWidth is the window's width in 100% units: the width whose layout this window is showing.
func (h *host) logicalWidth(g *Game) int { return g.width * 100 / max(100, h.workspace.Step) }

func (h *host) face(g *Game, base float64) text.Face {
	return &text.GoTextFace{Source: g.Map.Font, Size: h.text.Size(base)}
}
func (h *host) layout(g *Game) {
	h.workspace = ui.Workspace(g.width, g.height, g.State)
	h.text = ui.NativeText(g.width, g.height, g.State)
	r := h.workspace.Map
	g.mapOrigin = mapview.Point{X: float64(r.Min.X), Y: float64(r.Min.Y)}
	g.Camera.Resize(float64(max(1, r.Dx())), float64(max(1, r.Dy())))
	portrait := g.height > g.width
	if g.Camera.Zoom == 0 || g.State.Portrait != portrait {
		g.Camera = g.regionCamera(portrait)
	}
	g.State.Portrait = portrait
}

func (h *host) send(g *Game, kind string) {
	if h.busy || h.windowClosing || h.confirmNewGame && kind != gameapi.NewGame {
		return
	}
	if (kind == gameapi.AdvanceMonth || kind == gameapi.AdvanceYear) && g.State.View == "services" &&
		ui.ServiceDirty(g.State, g.Frame) {
		h.message = "draft_not_applied"
		g.dirty = true
		return
	}
	h.busy = true
	g.dirty = true
	c := gameapi.Command{
		Kind:       kind,
		Revision:   g.Frame.Revision,
		Name:       g.State.DraftName,
		CorridorID: g.State.ServiceCorridor,
		Frequency:  g.State.Frequency,
		Fare:       g.State.Fare,
	}
	if kind == gameapi.ExtendService {
		forecast, err := g.Port.Forecast(
			gameapi.ForecastRequest{
				CorridorID:       g.State.ServiceCorridor,
				Frequency:        g.State.Frequency,
				Fare:             g.State.Fare,
				ReplaceServiceID: g.State.ReplaceServiceID,
			},
		)
		if err != nil {
			h.busy = false
			h.message = err.Error()
			return
		}
		c.ReplaceServiceID = g.State.ReplaceServiceID
		c.LeaseCount = forecast.AdditionalLeases
	}
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
	if h.trainDescription != nil && g.State.Watching {
		h.updateTrainDescription(g)
	}
	h.menuInput = false
	for _, combo := range h.combos {
		if combo.ContentVisible() {
			h.menuInput = true
		}
	}
	h.refreshAppearance(g)
	if h.routeListStale(g) {
		g.dirty = true
	}
	if h.resizeWorkspace(g) {
		if g.dirty && g.width > 0 {
			h.build(g)
			g.dirty = false
		}
		return
	}
	loanBefore := g.State.LoanAmount
	select {
	case r := <-h.result:
		h.busy = false
		g.Frame = r.frame
		if g.State.ReviewMap && len(g.Frame.IdleLines) == 0 && len(g.State.ServiceMapSections) == 0 {
			g.State.ReviewMap = false
			h.layout(g)
		}
		if r.err == nil && r.kind != gameapi.NewGame {
			ui.RecordHelpAction(&g.State, g.Frame, r.kind)
			h.persist(g)
		}
		if r.err == nil && r.kind == gameapi.ExtendService {
			g.State.ReplaceServiceID = ""
			h.persist(g)
		}
		if r.err == nil && r.kind == gameapi.BuildLine {
			h.scroll = nil
		}
		if r.err == nil && r.kind == gameapi.NewGame {
			h.confirmNewGame = false
			h.blurName()
			h.nameInput = nil
			g.newGamePresentation()
			h.nameCache = nil
			h.field.SetTextAndSelection(g.State.DraftName, len(g.State.DraftName), len(g.State.DraftName))
			g.Camera = g.regionCamera(g.State.Portrait)
			h.scroll = nil
			h.layout(g)
			h.persist(g)
		}
		if r.err == nil && r.kind == gameapi.AdvanceMonth {
			g.State.View = "report"
			g.State.ReviewMap = false
			h.layout(g)
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
	if inpututil.IsKeyJustPressed(ebiten.KeyEscape) && g.State.Help.Open &&
		h.helpSearchField.UncommittedTextLengthInBytes() == 0 {
		h.helpEvent(g, ui.HelpEvent{Kind: "close"})
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
		h.blurName()
	}
	if g.State.View == "company" && h.loanInput != nil && h.loanInput.IsFocused() &&
		(ebiten.IsKeyPressed(ebiten.KeyMeta) || ebiten.IsKeyPressed(ebiten.KeyControl)) &&
		inpututil.IsKeyJustPressed(ebiten.KeyA) {
		h.loanInput.SetText(loanBefore)
		h.loanInput.SelectAll()
	}
	if !g.State.Help.Open && g.State.View == "services" && h.routeSearch != nil {
		_ = h.routeSearch.updateInput()
		if value := h.routeSearchField.Text(); value != g.State.ServiceQuery {
			g.State.ServiceQuery = value
			g.dirty = true
			h.persist(g)
		}
	}
	if g.State.Help.Open && h.helpSearch != nil {
		before := g.State.Help.Query
		_ = h.helpSearch.updateInput()
		value := h.helpSearchField.Text()
		if value != before {
			ui.ApplyHelpEvent(&g.State, ui.HelpEvent{Kind: "query", Value: value})
			h.persist(g)
			g.dirty = true
		}
	}
	if g.State.View == "company" && !g.State.Help.Open && h.nameInput != nil {
		if err := h.nameInput.updateInput(); err != nil {
			h.message = "text_edit_failed"
			if h.statusLabel != nil {
				h.statusLabel.Label = ui.Text(g.State.Locale, h.message)
			}
		}
		h.syncName(g)
	}
}
func (h *host) draw(g *Game, screen *ebiten.Image) {
	p := ui.PaletteFor(g.State)
	screen.Fill(p.Background)
	r := h.workspace.Map
	if !r.Empty() {
		if h.mapImage == nil || h.mapImage.Bounds().Dx() != r.Dx() || h.mapImage.Bounds().Dy() != r.Dy() {
			if h.mapImage != nil {
				h.mapImage.Deallocate()
			}
			h.mapImage = ebiten.NewImage(r.Dx(), r.Dy())
		}
		g.Map.Draw(h.mapImage, g.Camera, g.Frame, g.State, g.replayTime())
		op := &ebiten.DrawImageOptions{}
		op.GeoM.Translate(float64(r.Min.X), float64(r.Min.Y))
		screen.DrawImage(h.mapImage, op)
	}

	if h.ui != nil {
		h.ui.Draw(screen)
		if h.helpFocus != "" {
			if w := h.helpAnchors[h.helpFocus]; w != nil {
				if h.helpFocusScroll && h.scroll != nil {
					h.scrollSidebar(float64(w.GetWidget().Rect.Min.Y - h.scroll.ViewRect().Min.Y))
				}
				h.ui.SetFocusedWidget(w)
			}
			h.helpFocus = ""
			h.helpFocusScroll = false
		}

		h.observeHelp(g)
	}
}
func (h *host) build(g *Game) {
	previousRoute := h.routeSearch
	refocusRoute := previousRoute != nil && previousRoute.IsFocused() && !g.State.Help.Open
	if previousRoute != nil {
		previousRoute.Focus(false)
	}
	h.routeSearch = nil

	previousSearch := h.helpSearch
	refocusSearch := previousSearch != nil && previousSearch.IsFocused() && g.State.Help.Open
	if previousSearch != nil {
		previousSearch.Focus(false)
	}
	h.helpSearch = nil
	h.helpOpen = g.State.Help.Open
	h.helpAnchors = map[string]*widget.Button{}
	h.helpCue = ""
	h.helpCueAnchor = ""
	previousName := h.nameInput
	if previousName == nil {
		previousName = h.nameCache
	}
	previousLoan := h.loanInput
	refocusLoan := previousLoan != nil && previousLoan.IsFocused() && g.State.View == "company" &&
		ui.CompanyPane(g.State) == "finance" &&
		!g.State.Help.Open
	loanAllSelected := refocusLoan && previousLoan.SelectedText() == previousLoan.GetText()
	if previousLoan != nil {
		previousLoan.Focus(false)
	}
	h.loanInput = nil
	refocusName := h.nameInput != nil && h.nameInput.IsFocused() && g.State.View == "company" && !g.State.Expanded &&
		!h.confirmNewGame &&
		!g.State.Help.Open
	h.blurName()
	if h.nameInput != nil {
		h.syncName(g)
		h.nameCache = h.nameInput
	}
	h.nameInput = nil
	h.actions = map[string]*widget.Button{}
	h.mapControls = nil
	h.combos = nil
	h.comboChoices = nil
	h.primaryAction = nil
	h.trainDescription = nil
	h.replaySlider = nil
	h.text = ui.NativeText(g.width, g.height, g.State)
	p := ui.PaletteFor(g.State)
	if h.confirmNewGame {
		h.buildNewGame(g)
		return
	}
	t := func(k string) string { return ui.Text(g.State.Locale, k) }
	face := func(n float64) *text.Face { f := h.face(g, n); return &f }
	root := widget.NewContainer(widget.ContainerOpts.Layout(placed{}))
	h.workspace = ui.Workspace(g.width, g.height, g.State)
	r := h.workspace.Inspector
	x, y, w, height := r.Min.X, r.Min.Y, r.Dx(), r.Dy()
	if g.State.Expanded {
		x = 0
		y = 0
		w = g.width
		height = g.height
	}
	contentWidth := max(h.px(60), w-h.px(20))
	spacing := h.px(8)
	if g.State.View == "services" && ui.ServicePane(g.State) == "plan" {
		spacing = h.px(6)
	}
	content := widget.NewContainer(
		widget.ContainerOpts.Layout(
			widget.NewRowLayout(
				widget.RowLayoutOpts.Direction(widget.DirectionVertical),
				widget.RowLayoutOpts.Padding(&widget.Insets{Left: h.px(16), Right: h.px(16), Top: h.px(12), Bottom: h.px(12)}),
				widget.RowLayoutOpts.Spacing(spacing),
			),
		),
	)
	h.content = content
	h.contentWidth = contentWidth
	wrap := func(value string, size float64, width int) string {
		return ui.WrapText(value, float64(width), func(s string) float64 { return text.Advance(s, *face(size)) })
	}
	label := func(value string, size float64) {
		h.content.AddChild(
			widget.NewText(
				widget.TextOpts.Text(wrap(value, size, h.contentWidth-h.px(32)), face(size), p.Text),
				widget.TextOpts.MaxWidth(float64(h.contentWidth-h.px(32))),
			),
		)
	}
	button := func(title string, fn func()) *widget.Button {
		b := widget.NewButton(
			widget.ButtonOpts.WidgetOpts(
				widget.WidgetOpts.MinSize(h.px(44), h.px(44)),
				widget.WidgetOpts.LayoutData(widget.RowLayoutData{Stretch: true, MaxWidth: h.contentWidth - h.px(32)}),
			),
			widget.ButtonOpts.Image(
				&widget.ButtonImage{
					Idle:     uimage.NewNineSliceColor(p.Control),
					Hover:    uimage.NewNineSliceColor(p.Selected),
					Pressed:  uimage.NewNineSliceColor(p.Pressed),
					Disabled: uimage.NewNineSliceColor(p.Disabled),
				},
			),
			widget.ButtonOpts.Text(
				wrap(title, 14, h.contentWidth-h.px(56)),
				face(14),
				&widget.ButtonTextColor{Idle: p.Text, Disabled: p.Muted},
			),
			widget.ButtonOpts.TextPadding(widget.NewInsetsSimple(h.px(10))),
			widget.ButtonOpts.ClickedHandler(func(*widget.ButtonClickedEventArgs) { fn() }),
		)
		h.content.AddChild(b)
		return b
	}

	if g.State.Help.Open {
		h.buildReference(g, label, button, content)
	} else if g.State.View == "map" {
		h.helpSlot(g, "start", label, button)
		h.helpSlot(g, "idle", label, button)
		h.regionPicker(g, label, func(string) bool { return true })
		if stop := g.Map.PublicStop(g.State.SourceStation); stop != nil {
			label(stop.Name, 18)
			label(t("sourceStationInfo"), 13)
			if stop.MapGap {
				label(t("sourceStationGap"), 13)
			}
		}
		label(t("cities"), 20)
		choices := []choice{}
		for _, city := range g.Frame.Cities {
			choices = append(choices, choice{string(city.ID), t(string(city.ID))})
		}
		h.selectChoice(g, string(g.State.Selected), choices, func(id string) {
			g.State.SourceStation = ""
			g.State.Selected = gameapi.CityID(id)
			g.dirty = true
			h.persist(g)
		})
		if g.State.Selected != "" {
			for _, row := range ui.StationDetails(g.Frame, g.State.Locale, g.State.Selected) {
				if row.Label == t("station") {
					label(row.Value, 15)
				}
			}
			if h.disclosure(g, "station", t("moreDetails"), button) {
				for _, row := range ui.StationDetails(g.Frame, g.State.Locale, g.State.Selected) {
					label(row.Label+"\n"+row.Value, 14)
				}
			}
		}
		h.helpSlot(g, "city", label, button)
		if h.disclosure(g, "build-details", t("representativeDay"), button) || g.State.Watching {
			h.replayControls(g, label, button, content)
			h.helpSlot(g, "replay", label, button)
		}

	} else if g.State.View == "build" {
		h.buildPlanner(g, t, label, button)
	} else if g.State.View == "services" {
		h.buildServices(g, t, label, button)
	} else if g.State.View == "report" {
		h.buildReport(g, t, label, button)
	} else {
		h.buildCompany(g, t, label, button, content, contentWidth-h.px(32), previousName, refocusName)
	}
	oldScroll := 0.0
	if h.scroll != nil && h.lastView == g.State.View {
		oldScroll = h.scroll.ScrollTop
	}
	scrollRect := image.Rect(x, y, x+contentWidth, y+max(1, height))
	scroll := widget.NewScrollContainer(
		widget.ScrollContainerOpts.Content(content),
		widget.ScrollContainerOpts.StretchContentWidth(),
		widget.ScrollContainerOpts.Image(
			&widget.ScrollContainerImage{
				Idle: uimage.NewNineSliceColor(p.Background),
				Mask: uimage.NewNineSliceColor(color.White),
			},
		),
		widget.ScrollContainerOpts.WidgetOpts(widget.WidgetOpts.LayoutData(scrollRect)),
	)
	scroll.SetLocation(scrollRect)
	scroll.ScrollTop = oldScroll
	h.scroll = scroll
	h.lastView = g.State.View
	root.AddChild(scroll)
	footer := widget.NewContainer(
		widget.ContainerOpts.Layout(placed{}),
		widget.ContainerOpts.BackgroundImage(uimage.NewNineSliceColor(p.Surface)),
		widget.ContainerOpts.WidgetOpts(widget.WidgetOpts.LayoutData(h.workspace.Footer)),
	)
	status := t("saved")
	if h.message != "" {
		status = t(h.message)
	}
	fw, fh := h.workspace.Footer.Dx(), h.workspace.Footer.Dy()
	statusWidth := max(h.px(40), fw-h.px(290))
	if ui.CampaignFinished(g.Frame) && !h.workspace.Compact {
		statusWidth = fw - h.px(354)
	}
	statusRect := image.Rect(h.px(12), h.px(12), statusWidth, fh-h.px(4))
	if h.workspace.Compact {
		statusWidth = fw - h.px(24)
		statusRect = image.Rect(h.px(12), 0, fw-h.px(12), h.px(32))
	}
	h.statusLabel = widget.NewText(
		widget.TextOpts.Text(wrap(status, 12, statusWidth-h.px(12)), face(12), p.Text),
		widget.TextOpts.MaxWidth(float64(statusWidth-h.px(12))),
		widget.TextOpts.WidgetOpts(widget.WidgetOpts.LayoutData(statusRect)),
	)
	footer.AddChild(h.statusLabel)
	start, bw, by := fw-h.px(276), h.px(132), h.px(8)
	if h.workspace.Compact {
		start = h.px(8)
		bw = (fw - h.px(24)) / 2
		by = h.px(36)
	}
	if ui.CampaignFinished(g.Frame) {
		if !h.workspace.Compact {
			start = fw - h.px(340)
		}
		endWidth := fw - h.px(132) - start
		notice := t("campaignEndTitle") + "\n" + ui.CampaignEndDetail(g.State.Locale, g.Frame)
		footer.AddChild(
			widget.NewText(
				widget.TextOpts.Text(wrap(notice, 12, endWidth), face(12), p.Text),
				widget.TextOpts.MaxWidth(float64(endWidth)),
				widget.TextOpts.WidgetOpts(
					widget.WidgetOpts.LayoutData(image.Rect(start, by, start+endWidth, by+h.px(44))),
				),
			),
		)
		restart := image.Rect(fw-h.px(124), by, fw-h.px(8), by+h.px(44))
		b := h.fixedButton(g, t("campaignRestart"), restart, func() { h.openNewGame(g) })
		b.GetWidget().Disabled = h.busy
		h.actions["campaign-new-game"] = b
		footer.AddChild(b)
	} else {
		for i, key := range []string{"advance", "advanceYear"} {
			key := key
			b := h.fixedButton(g, t(key), image.Rect(start+i*(bw+h.px(8)), by, start+i*(bw+h.px(8))+bw, by+h.px(44)), func() {
				kind := gameapi.AdvanceMonth
				if key == "advanceYear" {
					kind = gameapi.AdvanceYear
				}
				h.send(g, kind)
			})
			b.GetWidget().Disabled = h.busy || g.State.View == "services" && ui.ServiceDirty(g.State, g.Frame)
			h.actions[key] = b
			footer.AddChild(b)
		}
	}
	root.AddChild(footer)

	h.configureSidebarScroll(g, root)
	if !g.State.Expanded {
		h.addCommandDock(g, root)
	}
	if g.State.Expanded {
		root = widget.NewContainer(widget.ContainerOpts.Layout(placed{}))
		b := widget.NewButton(
			widget.ButtonOpts.WidgetOpts(
				widget.WidgetOpts.LayoutData(image.Rect(max(0, g.width-h.px(280)), h.px(12), g.width-h.px(12), h.px(60))),
			),
			widget.ButtonOpts.Image(
				&widget.ButtonImage{
					Idle:    uimage.NewNineSliceColor(p.Control),
					Hover:   uimage.NewNineSliceColor(p.Control),
					Pressed: uimage.NewNineSliceColor(p.Control),
				},
			),
			widget.ButtonOpts.Text(
				t("backPlanner"),
				face(15),
				&widget.ButtonTextColor{Idle: p.Text, Disabled: p.Muted},
			),
			widget.ButtonOpts.ClickedHandler(
				func(*widget.ButtonClickedEventArgs) { g.State.Expanded = false; h.layout(g); g.dirty = true },
			),
		)
		root.AddChild(b)
		h.mapControls = append(h.mapControls, image.Rect(max(0, g.width-h.px(280)), h.px(12), g.width-h.px(12), h.px(60)))
		if g.State.View == "build" {
			contentWidth = g.width - h.px(20)
			content = widget.NewContainer(
				widget.ContainerOpts.Layout(
					widget.NewRowLayout(
						widget.RowLayoutOpts.Direction(widget.DirectionVertical),
						widget.RowLayoutOpts.Padding(widget.NewInsetsSimple(h.px(10))),
						widget.RowLayoutOpts.Spacing(h.px(4)),
					),
				),
				widget.ContainerOpts.BackgroundImage(uimage.NewNineSliceColor(p.Surface)),
				widget.ContainerOpts.WidgetOpts(
					widget.WidgetOpts.LayoutData(image.Rect(0, int(g.Camera.Height), g.width, g.height)),
				),
			)
			h.content = content
			h.contentWidth = contentWidth
			label(t("prospectHint"), 14)
			if co, ok := ui.Pair(g.Frame, g.State.FromCity, g.State.ToCity); ok {
				label(t(string(co.FromCity))+" – "+t(string(co.ToCity)), 14)
			}
			h.helpSlot(g, "expanded-endpoint", label, button)
			root.AddChild(content)
		}
		h.addMapTools(g, root)

	}
	h.ui = &ebitenui.UI{Container: root}
	if h.helpRestore && h.scroll != nil {
		h.setSidebarScroll(g.State.Help.ReturnScroll)
		h.helpRestore = false
	}
	if previousSearch != nil && h.helpSearch != nil && previousSearch.observed.value == h.helpSearchField.Text() {
		h.helpSearch.undo, h.helpSearch.redo = previousSearch.undo, previousSearch.redo
		h.helpSearch.anchor, h.helpSearch.caret = previousSearch.anchor, previousSearch.caret
		h.helpSearch.offset = previousSearch.offset
		h.helpSearch.observed = h.helpSearch.snapshot()
	}
	if previousRoute != nil && h.routeSearch != nil && previousRoute.observed.value == h.routeSearchField.Text() {
		h.routeSearch.undo, h.routeSearch.redo = previousRoute.undo, previousRoute.redo
		h.routeSearch.anchor, h.routeSearch.caret = previousRoute.anchor, previousRoute.caret
		h.routeSearch.offset = previousRoute.offset
		h.routeSearch.observed = h.routeSearch.snapshot()
	}
	if refocusRoute && h.routeSearch != nil {
		h.ui.SetFocusedWidget(h.routeSearch)
	}
	if refocusSearch && h.helpSearch != nil {
		h.ui.SetFocusedWidget(h.helpSearch)
	}
	if refocusLoan && h.loanInput != nil {
		h.ui.SetFocusedWidget(h.loanInput)
		if loanAllSelected {
			h.loanInput.SelectAll()
		}
	}
	if refocusName && h.nameInput != nil {
		h.ui.SetFocusedWidget(h.nameInput)
	}
}
