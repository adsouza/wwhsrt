// This isolated feasibility spike uses real ebitenui widgets. Its small economic
// model is illustrative and is not the game's simulation or application layer.
package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"image"
	"image/color"
	"log"
	"math"
	"strconv"

	"github.com/ebitenui/ebitenui"
	uimage "github.com/ebitenui/ebitenui/image"
	"github.com/ebitenui/ebitenui/utilities/mobile"
	"github.com/ebitenui/ebitenui/widget"
	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/inpututil"
	"github.com/hajimehoshi/ebiten/v2/text/v2"
	"golang.org/x/image/font/gofont/goregular"
)

var (
	bg         = color.NRGBA{246, 247, 244, 255}
	panel      = color.NRGBA{255, 255, 255, 255}
	ink        = color.NRGBA{24, 51, 52, 255}
	muted      = color.NRGBA{82, 107, 107, 255}
	border     = color.NRGBA{214, 223, 217, 255}
	accent     = color.NRGBA{0, 105, 82, 255}
	soft       = color.NRGBA{233, 243, 236, 255}
	warning    = color.NRGBA{251, 242, 224, 255}
	warningInk = color.NRGBA{117, 83, 32, 255}
	red        = color.NRGBA{152, 47, 40, 255}
)

type placedLayout struct{}

func (placedLayout) PreferredSize([]widget.PreferredSizeLocateableWidget) (int, int) { return 0, 0 }
func (placedLayout) Layout(ws []widget.PreferredSizeLocateableWidget, r image.Rectangle) {
	for _, w := range ws {
		w.SetLocation(w.GetWidget().LayoutData.(image.Rectangle).Add(r.Min))
	}
}

type game struct {
	ui                                                                   *ebitenui.UI
	w, h                                                                 int
	scale                                                                float64
	dirty                                                                bool
	view                                                                 string
	hideHint                                                             bool
	applied                                                              bool
	rawFrequency, rawFare                                                string
	frequency, fare, fleet                                               int
	valid                                                                bool
	operating                                                            int
	frames                                                               int
	font                                                                 *text.GoTextFaceSource
	faces                                                                map[float64]*text.Face
	images                                                               map[string]*uimage.NineSlice
	tracked                                                              map[string]widget.PreferredSizeLocateableWidget
	scroll                                                               *widget.ScrollContainer
	freqInput, fareInput                                                 *widget.TextInput
	note, forecast, passengers, occupancy, requirement, ack, actionValue *widget.Text
	apply                                                                *widget.Button
	touchID                                                              ebiten.TouchID
	touchY                                                               int
	touching                                                             bool
	touchMoved                                                           bool
}

func (g *game) px(n int) int { return int(math.Round(float64(n) * g.scale)) }
func (g *game) face(size float64) *text.Face {
	key := size * g.scale
	if f := g.faces[key]; f != nil {
		return f
	}
	var f text.Face = &text.GoTextFace{Source: g.font, Size: key}
	g.faces[key] = &f
	return &f
}
func (g *game) rounded(body, edge color.NRGBA) *uimage.NineSlice {
	key := fmt.Sprintf("%v/%v/%v", body, edge, g.scale)
	if im := g.images[key]; im != nil {
		return im
	}
	r := g.px(8)
	n := r*2 + 1
	rgba := image.NewNRGBA(image.Rect(0, 0, n, n))
	for y := 0; y < n; y++ {
		for x := 0; x < n; x++ {
			dx := math.Max(float64(r-x), math.Max(0, float64(x-r)))
			dy := math.Max(float64(r-y), math.Max(0, float64(y-r)))
			d := math.Hypot(dx, dy)
			if d <= float64(r) {
				c := body
				if d > float64(r)-g.scale {
					c = edge
				}
				rgba.SetNRGBA(x, y, c)
			}
		}
	}
	im := uimage.NewNineSlice(ebiten.NewImageFromImage(rgba), [3]int{r, 1, r}, [3]int{r, 1, r})
	g.images[key] = im
	return im
}
func (g *game) label(value string, size float64, c color.Color, width int) *widget.Text {
	return widget.NewText(
		widget.TextOpts.Text(value, g.face(size), c),
		widget.TextOpts.MaxWidth(float64(g.px(width))),
		widget.TextOpts.WidgetOpts(widget.WidgetOpts.LayoutData(widget.RowLayoutData{Stretch: true})),
	)
}
func (g *game) column(padding, spacing int, background color.Color) *widget.Container {
	opts := []widget.ContainerOpt{
		widget.ContainerOpts.Layout(
			widget.NewRowLayout(
				widget.RowLayoutOpts.Direction(widget.DirectionVertical),
				widget.RowLayoutOpts.Spacing(g.px(spacing)),
				widget.RowLayoutOpts.Padding(widget.NewInsetsSimple(g.px(padding))),
			),
		),
		widget.ContainerOpts.WidgetOpts(widget.WidgetOpts.LayoutData(widget.RowLayoutData{Stretch: true})),
	}
	if background != nil {
		opts = append(opts, widget.ContainerOpts.BackgroundImage(uimage.NewNineSliceColor(background)))
	}
	return widget.NewContainer(opts...)
}
func (g *game) button(id, label string, primary bool, fn func()) *widget.Button {
	idle, hover, fg := panel, soft, ink
	if primary {
		idle, hover, fg = accent, accent, panel
	}
	b := widget.NewButton(
		widget.ButtonOpts.WidgetOpts(widget.WidgetOpts.MinSize(g.px(44), g.px(44))),
		widget.ButtonOpts.Image(
			&widget.ButtonImage{
				Idle:     g.rounded(idle, border),
				Hover:    g.rounded(hover, accent),
				Pressed:  g.rounded(soft, accent),
				Disabled: g.rounded(bg, border),
			},
		),
		widget.ButtonOpts.Text(label, g.face(14), &widget.ButtonTextColor{Idle: fg, Disabled: muted}),
		widget.ButtonOpts.TextPadding(widget.NewInsetsSimple(g.px(10))),
		widget.ButtonOpts.ClickedHandler(func(*widget.ButtonClickedEventArgs) { fn() }),
	)
	g.tracked[id] = b
	return b
}
func (g *game) grid(cols int, stretch []bool) *widget.Container {
	return widget.NewContainer(
		widget.ContainerOpts.WidgetOpts(widget.WidgetOpts.LayoutData(widget.RowLayoutData{Stretch: true})),
		widget.ContainerOpts.Layout(
			widget.NewGridLayout(
				widget.GridLayoutOpts.Columns(cols),
				widget.GridLayoutOpts.Stretch(stretch, nil),
				widget.GridLayoutOpts.Spacing(g.px(8), g.px(8)),
			),
		),
	)
}

func (g *game) cashRow(label, value string, width int) *widget.Container {
	row := g.grid(2, []bool{true, false})
	row.AddChild(g.label(label, 14, muted, width-108))
	row.AddChild(g.label(value, 14, ink, 100))
	return row
}
func (g *game) field(id, title string, value string, delta int, width int) (*widget.Container, *widget.TextInput) {
	c := g.column(0, 8, nil)
	c.AddChild(g.label(title, 16, ink, width))
	line := g.grid(3, []bool{false, true, false})
	var input *widget.TextInput
	change := func(n int) {
		v, _ := strconv.Atoi(input.GetText())
		input.SetText(strconv.Itoa(v + n))
		g.applied = false
	}
	line.AddChild(g.button(id+"Minus", "−", false, func() { change(-delta) }))
	input = widget.NewTextInput(
		widget.TextInputOpts.WidgetOpts(widget.WidgetOpts.MinSize(g.px(60), g.px(44))),
		widget.TextInputOpts.Image(
			&widget.TextInputImage{Idle: g.rounded(panel, border), Disabled: g.rounded(bg, border)},
		),
		widget.TextInputOpts.Face(g.face(16)),
		widget.TextInputOpts.Color(&widget.TextInputColor{Idle: ink, Caret: accent}),
		widget.TextInputOpts.Padding(widget.NewInsetsSimple(g.px(10))),
		widget.TextInputOpts.MobileInputMode(mobile.NUMERIC),
		widget.TextInputOpts.ChangedHandler(func(a *widget.TextInputChangedEventArgs) {
			if id == "frequency" {
				g.rawFrequency = a.InputText
			} else {
				g.rawFare = a.InputText
			}
			g.applied = false
		}),
	)
	input.SetText(value)
	g.tracked[id] = input
	line.AddChild(input)
	line.AddChild(g.button(id+"Plus", "+", false, func() { change(delta) }))
	c.AddChild(line)
	return c, input
}
func (g *game) build() {
	oldScroll := 0.0
	if g.scroll != nil {
		oldScroll = g.scroll.ScrollTop
	}
	g.tracked = map[string]widget.PreferredSizeLocateableWidget{}
	g.faces = map[float64]*text.Face{}
	g.freqInput = nil
	g.fareInput = nil
	root := widget.NewContainer(
		widget.ContainerOpts.Layout(placedLayout{}),
		widget.ContainerOpts.BackgroundImage(uimage.NewNineSliceColor(bg)),
	)
	place := func(id string, c widget.PreferredSizeLocateableWidget, x, y, w, h int) {
		c.GetWidget().LayoutData = image.Rect(g.px(x), g.px(y), g.px(x+w), g.px(y+h))
		root.AddChild(c)
		g.tracked[id] = c
	}
	header := g.grid(2, []bool{true, false})
	header.AddChild(g.label("Feb 2029\nOffline · business month", 14, ink, g.w/2))
	header.AddChild(g.label("$32.06M\nUnrestricted cash", 14, ink, g.w/2))
	place("header", header, 16, 12, g.w-32, 52)
	width := g.w - 64
	columns := 1
	if g.w >= 900 {
		columns = 2
		width = (g.w - 88) / 2
	}
	content := g.column(16, 16, panel)
	if g.view == "Services" {
		content.AddChild(g.label("SERVICES · PLAN FOR FEB 2029", 12, muted, g.w-64))
		content.AddChild(g.label("Lyon–Turin Express", 24, ink, g.w-64))
		g.ack = g.label("Current operating plan", 13, muted, g.w-64)
		content.AddChild(g.ack)
		panes := g.grid(columns, []bool{true, true})
		settings := g.column(0, 14, nil)
		settings.AddChild(g.label("Lyon  →  Turin", 18, ink, width))
		settings.AddChild(
			g.label("Nonstop · 1 h 08 min · 220 km\nPublic track · three 400-seat trainsets", 14, muted, width),
		)
		freq, in := g.field("frequency", "Return trips per day", g.rawFrequency, 1, width)
		g.freqInput = in
		settings.AddChild(freq)
		g.requirement = g.label("", 14, muted, width)
		settings.AddChild(g.requirement)
		if !g.hideHint {
			hint := g.column(12, 8, warning)
			hint.AddChild(
				g.label(
					"One return trip means one departure each way. More trips can attract passengers and need more trains.",
					14,
					warningInk,
					width-24,
				),
			)
			hint.AddChild(g.button("dismissHint", "Got it", false, func() { g.hideHint = true; g.dirty = true }))
			settings.AddChild(hint)
		}
		fare, fin := g.field("fare", "One-way fare · USD", g.rawFare, 5, width)
		g.fareInput = fin
		settings.AddChild(fare)
		settings.AddChild(g.label("No train purchase or loan is included.", 14, muted, width))
		g.note = g.label("", 14, red, width)
		settings.AddChild(g.note)
		panes.AddChild(settings)
		preview := g.column(12, 16, soft)
		preview.AddChild(g.label("Forecast · February 2029", 18, ink, width-24))
		preview.AddChild(g.label("28 days · illustrative demand model", 13, muted, width-24))
		g.forecast = g.label("", 26, ink, width-24)
		preview.AddChild(g.forecast)
		g.passengers = g.label("", 18, ink, width-24)
		preview.AddChild(g.passengers)
		g.occupancy = g.label("", 14, muted, width-24)
		preview.AddChild(g.occupancy)
		preview.AddChild(
			g.label("Operating cash excludes construction, interest, and principal repayments.", 14, muted, width-24),
		)
		panes.AddChild(preview)
		content.AddChild(panes)
	} else {
		content.AddChild(g.label("MONTH CLOSED", 12, muted, g.w-64))
		content.AddChild(g.label("January 2029", 24, ink, g.w-64))
		panes := g.grid(columns, []bool{true, true})
		cash := g.column(0, 14, nil)
		cash.AddChild(g.label("Operating cash", 14, muted, width))
		cash.AddChild(g.label("+$3.06M", 28, ink, width))
		cash.AddChild(g.label("Total cash change", 14, muted, width))
		cash.AddChild(g.label("−$9.94M", 28, ink, width))
		if !g.hideHint {
			hint := g.column(12, 8, warning)
			hint.AddChild(
				g.label(
					"Your trains earned cash, but the station upgrade used more. Investment is separate from operating costs.",
					14,
					warningInk,
					width-24,
				),
			)
			hint.AddChild(g.button("dismissHint", "Got it", false, func() { g.hideHint = true; g.dirty = true }))
			cash.AddChild(hint)
		}
		cash.AddChild(g.label("Cash reconciliation", 18, ink, width))
		for _, entry := range [][2]string{
			{"Opening cash", "$42.00M"},
			{"Operating cash", "+$3.06M"},
			{"Station upgrade", "−$12.00M"},
			{"Interest + principal", "−$1.00M"},
			{"Closing cash", "$32.06M"},
		} {
			cash.AddChild(g.cashRow(entry[0], entry[1], width))
		}
		panes.AddChild(cash)
		service := g.column(12, 16, soft)
		service.AddChild(g.label("Lyon–Turin Express", 18, ink, width-24))
		service.AddChild(g.label("95,239 journeys", 26, ink, width-24))
		service.AddChild(
			g.label("64% seat occupancy\n6 daily return trips\n$65 one-way fare\n31 operating days", 16, ink, width-24),
		)
		service.AddChild(
			g.label(
				"Empty seats are a market signal. Compare fare and frequency in the service editor.",
				14,
				muted,
				width-24,
			),
		)
		panes.AddChild(service)
		content.AddChild(panes)
	}
	g.scroll = widget.NewScrollContainer(
		widget.ScrollContainerOpts.Content(content),
		widget.ScrollContainerOpts.StretchContentWidth(),
		widget.ScrollContainerOpts.Image(
			&widget.ScrollContainerImage{Idle: uimage.NewNineSliceColor(panel), Mask: uimage.NewNineSliceColor(panel)},
		),
	)
	g.scroll.ScrollTop = oldScroll
	viewH := max(96, g.h-64-86-64-24)
	place("scroll", g.scroll, 16, 76, g.w-32, viewH)
	footer := g.grid(2, []bool{true, false})
	g.actionValue = g.label("Monthly operating cash", 14, ink, g.w-190)
	footer.AddChild(g.actionValue)
	g.apply = g.button("apply", "Apply plan", true, func() {
		if g.view == "Reports" {
			g.view = "Services"
			g.scroll = nil
			g.dirty = true
		} else if g.valid {
			g.applied = true
		}
	})
	if g.view == "Reports" {
		g.apply.SetText("Adjust service")
	}
	footer.AddChild(g.apply)
	place("footer", footer, 16, g.h-138, g.w-32, 64)
	nav := g.grid(2, []bool{true, true})
	nav.AddChild(
		g.button("servicesTab", "Services", false, func() { g.view = "Services"; g.scroll = nil; g.dirty = true }),
	)
	nav.AddChild(
		g.button("reportsTab", "Monthly report", false, func() { g.view = "Reports"; g.scroll = nil; g.dirty = true }),
	)
	place("navigation", nav, 16, g.h-62, g.w-32, 46)
	g.ui = &ebitenui.UI{Container: root}
	g.dirty = false
	g.refresh()
}
func (g *game) refresh() {
	if g.view != "Services" {
		g.actionValue.Label = "Plan the next month\nFeb 2029"
		return
	}
	f, err1 := strconv.Atoi(g.rawFrequency)
	fare, err2 := strconv.Atoi(g.rawFare)
	g.frequency = f
	g.fare = fare
	g.fleet = int(math.Ceil(float64(f)*2.9/12)) + 1
	g.valid = err1 == nil && err2 == nil && f >= 1 && f <= 12 && fare >= 20 && fare <= 150 && g.fleet <= 3
	note := ""
	if err1 != nil || f < 1 || f > 12 {
		note = "Enter 1–12 return trips per day."
	} else if err2 != nil || fare < 20 || fare > 150 {
		note = "Enter a fare from $20 to $150."
	} else if g.fleet > 3 {
		note = fmt.Sprintf("%d return trips need %d trainsets; only 3 are available.", f, g.fleet)
	}
	g.note.Label = note
	g.requirement.Label = fmt.Sprintf("%d departures each way · %d trainsets incl. spare", f, g.fleet)
	g.apply.GetWidget().Disabled = !g.valid
	if g.valid {
		p := int(math.Round(137000 * (1 - math.Exp(-float64(f)/4)) * math.Exp(-float64(fare-55)/90) * 28 / 31))
		seats := 2 * f * 400 * 28
		p = min(p, seats)
		g.operating = p*fare - 2*f*220*28*26 - 1000000
		g.forecast.Label = fmt.Sprintf("+$%.2fM operating cash", float64(g.operating)/1e6)
		g.passengers.Label = fmt.Sprintf("%s passenger journeys", comma(p))
		g.occupancy.Label = fmt.Sprintf("%d%% seat occupancy · %d of 3 trainsets", p*100/seats, g.fleet)
		g.actionValue.Label = fmt.Sprintf("Monthly operating cash\n+$%.2fM", float64(g.operating)/1e6)
	} else {
		g.forecast.Label = "More fleet or valid inputs needed"
		g.passengers.Label = "Preview requires an operable plan"
		g.occupancy.Label = ""
		g.actionValue.Label = "Review the constraint"
	}
	if g.applied {
		g.ack.Label = "Plan applied for February · month unchanged"
		g.apply.SetText("Plan applied")
	} else {
		g.ack.Label = "Draft · no money spent"
		g.apply.SetText("Apply plan")
	}
}
func comma(n int) string {
	s := strconv.Itoa(n)
	for i := len(s) - 3; i > 0; i -= 3 {
		s = s[:i] + "," + s[i:]
	}
	return s
}
func (g *game) Layout(w, h int) (int, int) {
	s := ebiten.DeviceScaleFactor()
	if w != g.w || h != g.h || s != g.scale {
		g.w = w
		g.h = h
		g.scale = s
		g.dirty = true
	}
	return int(float64(w) * s), int(float64(h) * s)
}
func (g *game) Update() error {
	if g.dirty {
		g.build()
	}
	// ebitenui 0.7.3's canvas text input needs an application-level adapter
	// for platform select-all shortcuts. Preserve the pre-update value so
	// the chord's printable "A" is not inserted into the number field.
	var selectField *widget.TextInput
	before := ""
	if (ebiten.IsKeyPressed(ebiten.KeyMeta) || ebiten.IsKeyPressed(ebiten.KeyControl)) &&
		inpututil.IsKeyJustPressed(ebiten.KeyA) {
		for _, field := range []*widget.TextInput{g.freqInput, g.fareInput} {
			if field != nil && field.IsFocused() {
				selectField = field
				before = field.GetText()
			}
		}
	}
	g.ui.Update()
	if selectField != nil {
		selectField.SetText(before)
		selectField.SelectAll()
	}
	// Navigation handlers replace the widget tree at the next frame boundary.
	// Do not read the old scroll container after a handler invalidates it.
	if g.dirty {
		g.frames++
		return nil
	}
	g.refresh()
	g.frames++
	// ScrollContainer masks/clips widgets, but application code supplies wheel
	// and drag policy. Dragging on a text field/button is left to that widget.
	x, y := ebiten.CursorPosition()
	r := g.scroll.ViewRect()
	if image.Pt(x, y).In(r) {
		_, wheel := ebiten.Wheel()
		if wheel != 0 {
			g.scroll.ScrollTop = math.Max(0, math.Min(1, g.scroll.ScrollTop-wheel*.08))
		}
	}
	ids := inpututil.AppendJustPressedTouchIDs(nil)
	if len(ids) > 0 {
		tx, ty := ebiten.TouchPosition(ids[0])
		point := image.Pt(tx, ty)
		owned := false
		for _, control := range g.tracked {
			switch control.(type) {
			case *widget.Button, *widget.TextInput:
				owned = owned || point.In(control.GetWidget().Rect)
			}
		}
		if point.In(r) && !owned {
			g.touchID = ids[0]
			g.touchY = ty
			g.touching = true
			g.touchMoved = false
		}
	}
	if g.touching {
		if inpututil.IsTouchJustReleased(g.touchID) {
			g.touching = false
		} else {
			_, ty := ebiten.TouchPosition(g.touchID)
			delta := g.touchY - ty
			if math.Abs(float64(delta)) > g.scale*3 {
				g.touchMoved = true
			}
			if g.touchMoved {
				ch := g.scroll.ContentRect().Dy() - r.Dy()
				if ch > 0 {
					g.scroll.ScrollTop = math.Max(0, math.Min(1, g.scroll.ScrollTop+float64(delta)/float64(ch)))
				}
				g.touchY = ty
			}
		}
	}
	return nil
}
func (g *game) Draw(s *ebiten.Image) { g.ui.Draw(s) }
func (g *game) snapshot() string {
	items := map[string]any{}
	for id, w := range g.tracked {
		r := w.GetWidget().Rect
		items[id] = map[string]any{
			"x":        float64(r.Min.X) / g.scale,
			"y":        float64(r.Min.Y) / g.scale,
			"w":        float64(r.Dx()) / g.scale,
			"h":        float64(r.Dy()) / g.scale,
			"disabled": w.GetWidget().Disabled,
		}
	}
	focused := ""
	if g.freqInput != nil && g.freqInput.IsFocused() {
		focused = "frequency"
	}
	if g.fareInput != nil && g.fareInput.IsFocused() {
		focused = "fare"
	}
	scroll := 0.0
	if g.scroll != nil {
		scroll = g.scroll.ScrollTop
	}
	b, _ := json.Marshal(
		map[string]any{
			"view":          g.view,
			"width":         g.w,
			"height":        g.h,
			"scale":         g.scale,
			"frequency":     g.rawFrequency,
			"fare":          g.rawFare,
			"valid":         g.valid,
			"fleet":         g.fleet,
			"applied":       g.applied,
			"hintDismissed": g.hideHint,
			"operating":     g.operating,
			"scroll":        scroll,
			"focused":       focused,
			"frames":        g.frames,
			"widgets":       items,
		},
	)
	return string(b)
}
func main() {
	source, err := text.NewGoTextFaceSource(bytes.NewReader(goregular.TTF))
	if err != nil {
		log.Fatal(err)
	}
	g := &game{
		w:            1280,
		h:            800,
		scale:        1,
		dirty:        true,
		view:         "Services",
		rawFrequency: "6",
		rawFare:      "65",
		font:         source,
		faces:        map[float64]*text.Face{},
		images:       map[string]*uimage.NineSlice{},
	}
	installProbe(g)
	ebiten.SetWindowSize(1280, 800)
	ebiten.SetWindowResizingMode(ebiten.WindowResizingModeEnabled)
	ebiten.SetWindowTitle("WWH SRT · actual ebitenui feasibility spike")
	if err := ebiten.RunGame(g); err != nil {
		log.Fatal(err)
	}
}
