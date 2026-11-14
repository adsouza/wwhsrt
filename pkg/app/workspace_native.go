//go:build !js

package app

import (
	"fmt"
	uimage "github.com/ebitenui/ebitenui/image"
	"github.com/ebitenui/ebitenui/widget"
	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/inpututil"
	"github.com/hajimehoshi/ebiten/v2/text/v2"
	"image"
	"strings"
	"unicode/utf8"
	"wwhsrt/pkg/mapview"
	"wwhsrt/pkg/ui"
)

func (h *host) workspaceChanged(g *Game) { h.blurName(); h.layout(g); g.dirty = true; h.persist(g) }
func (h *host) row(cols int, fn func()) {
	if h.content == nil {
		fn()
		return
	}
	parent, width := h.content, h.contentWidth
	stretch := make([]bool, cols)
	for i := range stretch {
		stretch[i] = true
	}
	group := widget.NewContainer(
		widget.ContainerOpts.Layout(
			widget.NewGridLayout(
				widget.GridLayoutOpts.Columns(cols),
				widget.GridLayoutOpts.Stretch(stretch, nil),
				widget.GridLayoutOpts.Spacing(h.px(8), h.px(4)),
			),
		),
		widget.ContainerOpts.WidgetOpts(widget.WidgetOpts.LayoutData(widget.RowLayoutData{Stretch: true})),
	)
	parent.AddChild(group)
	h.content = group
	h.contentWidth = (width-32-8*(cols-1))/cols + 32
	fn()
	h.content = parent
	h.contentWidth = width
}
func (h *host) disclosure(g *Game, id, title string, button nativeButton) bool {
	mark := "+ "
	if ui.Disclosure(g.State, id) {
		mark = "− "
	}
	b := button(
		mark+title,
		func() { ui.SetDisclosure(&g.State, id, !ui.Disclosure(g.State, id)); g.dirty = true; h.persist(g) },
	)
	if h.actions != nil {
		h.actions["disclosure-"+id] = b
	}
	return ui.Disclosure(g.State, id)
}
func (h *host) taskTabs(g *Game, task string, keys []string, button nativeButton) {
	current := ui.ServicePane(g.State)
	if task == "company" {
		current = ui.CompanyPane(g.State)
	}
	if task == "build" {
		current = ui.BuildPane(g.State)
	}
	if task == "report" {
		current = ui.ReportPane(g.State)
	}
	h.row(len(keys), func() {
		for _, key := range keys {
			key := key
			b := button(ui.Text(g.State.Locale, "pane_"+key), func() {
				switch task {
				case "services":
					g.State.ServicePane = key
				case "build":
					g.State.BuildPane = key
				case "company":
					g.State.CompanyPane = key
				case "report":
					g.State.ReportPane = key
				}
				h.scroll = nil
				h.workspaceChanged(g)
			})
			if key == current {
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
	})
}

type choice struct{ ID, Name string }

func (h *host) selectChoice(g *Game, selected string, options []choice, changed func(string), prompt ...string) {
	if h.content == nil || len(options) == 0 {
		return
	}
	p := ui.PaletteFor(g.State)
	face := h.face(g, 14)
	width := max(h.px(60), h.contentWidth-h.px(32))
	entries := []any{}
	var chosen any
	if selected == "" {
		title := ui.Text(g.State.Locale, "selectCity")
		if len(prompt) > 0 {
			title = prompt[0]
		}
		placeholder := choice{"", title}
		entries = append(entries, placeholder)
		chosen = placeholder
	}
	for _, c := range options {
		entries = append(entries, c)
		if c.ID == selected {
			chosen = c
		}
	}
	images := &widget.ButtonImage{
		Idle:     uimage.NewNineSliceColor(p.Control),
		Hover:    uimage.NewNineSliceColor(p.Selected),
		Pressed:  uimage.NewNineSliceColor(p.Pressed),
		Disabled: uimage.NewNineSliceColor(p.Disabled),
	}
	handle := h.px(12)
	labels := func(e any) string {
		return ui.WrapText(e.(choice).Name, float64(width-24), func(v string) float64 { return text.Advance(v, face) })
	}
	h.comboListHeight = h.px(240)
	consumed := false
	var combo *widget.ListComboButton
	combo = widget.NewListComboButton(
		widget.ListComboButtonOpts.Entries(entries),
		widget.ListComboButtonOpts.InitialEntry(chosen),
		widget.ListComboButtonOpts.MaxContentHeight(h.comboListHeight),
		widget.ListComboButtonOpts.WidgetOpts(
			widget.WidgetOpts.MinSize(h.px(44), h.px(44)),
			widget.WidgetOpts.LayoutData(widget.RowLayoutData{Stretch: true, MaxWidth: width}),
		),
		widget.ListComboButtonOpts.ButtonParams(
			&widget.ButtonParams{
				Image:       images,
				TextPadding: widget.NewInsetsSimple(h.px(8)),
				TextColor:   &widget.ButtonTextColor{Idle: p.Text, Disabled: p.Muted},
				TextFace:    &face,
				MinSize:     &image.Point{X: 60, Y: 44},
			},
		),
		widget.ListComboButtonOpts.ListParams(
			&widget.ListParams{
				ScrollContainerImage: &widget.ScrollContainerImage{
					Idle: uimage.NewNineSliceColor(p.Surface),
					Mask: uimage.NewNineSliceColor(p.Surface),
				},
				Slider: &widget.SliderParams{
					TrackImage:    &widget.SliderTrackImage{Idle: uimage.NewNineSliceColor(p.Border)},
					HandleImage:   images,
					MinHandleSize: &handle,
					TrackPadding:  widget.NewInsetsSimple(h.px(2)),
				},
				EntryFace: &face,
				EntryColor: &widget.ListEntryColor{
					Selected:                  p.Text,
					Unselected:                p.Text,
					SelectedBackground:        p.Selected,
					SelectedFocusedBackground: p.Hover,
					FocusedBackground:         p.Selected,
				},
				EntryTextPadding: widget.NewInsetsSimple(h.px(8)),
				MinSize:          &image.Point{X: 60, Y: 44},
			},
		),
		widget.ListComboButtonOpts.EntryLabelFunc(labels, labels),
		widget.ListComboButtonOpts.EntrySelectedHandler(func(a *widget.ListComboButtonEntrySelectedEventArgs) {
			if consumed || combo == nil || a.Entry == nil {
				return
			}
			current := combo.SelectedEntry()
			// Initializing a ListComboButton queues its first-entry event before
			// SetSelectedEntry applies our controlled value. Old rebuild events
			// also outlive their widget. Neither may overwrite a current draft.
			if current == nil || current.(choice).ID != a.Entry.(choice).ID {
				return
			}
			active := false
			for _, c := range h.combos {
				if c == combo {
					active = true
					break
				}
			}
			if active && a.Entry.(choice).ID != "" && a.Entry.(choice).ID != selected {
				// All callers rebuild after accepting a controlled draft change.
				// Ignore the old state's remaining events before that next frame.
				consumed = true
				changed(a.Entry.(choice).ID)
			}
		}),
	)
	h.content.AddChild(combo)
	h.combos = append(h.combos, combo)
	h.comboChoices = append(h.comboChoices, options)
}
func (h *host) fixedButton(g *Game, title string, r image.Rectangle, run func()) *widget.Button {
	p := ui.PaletteFor(g.State)
	face := h.face(g, 13)
	// The label wraps to the button's inner width, so a two-line label fits a 44 px button.
	pad := widget.Insets{Left: h.px(4), Right: h.px(4), Top: h.px(3), Bottom: h.px(3)}
	inner := float64(max(h.px(20), r.Dx()-h.px(8)))
	title = ui.WrapText(title, inner, func(v string) float64 { return text.Advance(v, face) })
	return widget.NewButton(
		widget.ButtonOpts.WidgetOpts(widget.WidgetOpts.MinSize(h.px(44), h.px(44)), widget.WidgetOpts.LayoutData(r)),
		widget.ButtonOpts.Image(
			&widget.ButtonImage{
				Idle:     uimage.NewNineSliceColor(p.Control),
				Hover:    uimage.NewNineSliceColor(p.Selected),
				Pressed:  uimage.NewNineSliceColor(p.Pressed),
				Disabled: uimage.NewNineSliceColor(p.Disabled),
			},
		),
		widget.ButtonOpts.Text(title, &face, &widget.ButtonTextColor{Idle: p.Text, Disabled: p.Muted}),
		widget.ButtonOpts.TextPadding(&pad),
		widget.ButtonOpts.ClickedHandler(func(*widget.ButtonClickedEventArgs) { run() }),
	)
}
func (h *host) addCommandDock(g *Game, root *widget.Container) {
	l := h.workspace
	p := ui.PaletteFor(g.State)
	t := func(k string) string { return ui.Text(g.State.Locale, k) }
	header := widget.NewContainer(
		widget.ContainerOpts.Layout(placed{}),
		widget.ContainerOpts.BackgroundImage(uimage.NewNineSliceColor(p.Surface)),
		widget.ContainerOpts.WidgetOpts(widget.WidgetOpts.LayoutData(l.Header)),
	)
	face := h.face(g, 14)
	value := fmt.Sprintf("%04d-%02d  ·  %s", g.Frame.Month.Year, g.Frame.Month.Month, ui.Cash(g.Frame.Player.Cash))
	if h.logicalWidth(g) >= 800 {
		name := g.Frame.Player.Name
		if utf8.RuneCountInString(name) > 22 {
			name = string([]rune(name)[:21]) + "…"
		}
		value = name + "   ·   " + value
	}
	statWidth := max(h.px(40), g.width-h.px(80))
	if len(g.Frame.IdleLines) > 0 {
		statWidth = max(h.px(40), g.width-h.px(154))
	}
	value = ui.WrapText(value, float64(statWidth), func(v string) float64 { return text.Advance(v, face) })
	stat := widget.NewText(
		widget.TextOpts.Text(value, &face, p.Text),
		widget.TextOpts.MaxWidth(float64(statWidth)),
		widget.TextOpts.WidgetOpts(widget.WidgetOpts.LayoutData(image.Rect(h.px(12), h.px(8), h.px(12)+statWidth, h.px(46)))),
	)
	header.AddChild(stat)
	language := "EN"
	if g.State.Locale == "zh-Hans" {
		language = "简"
	}
	header.AddChild(h.fixedButton(g, language, image.Rect(g.width-h.px(52), h.px(2), g.width-h.px(8), h.px(46)), func() {
		if g.State.Locale == "en" {
			g.State.Locale = "zh-Hans"
		} else {
			g.State.Locale = "en"
		}
		h.workspaceChanged(g)
	}))
	if len(g.Frame.IdleLines) > 0 {
		h.idleAction = h.fixedButton(
			g,
			t("idleShort")+" · "+fmt.Sprint(len(g.Frame.IdleLines)),
			image.Rect(g.width-h.px(136), h.px(2), g.width-h.px(58), h.px(46)),
			func() { h.reviewIdleLine(g, "") },
		)
		h.idleAction.SetImage(
			&widget.ButtonImage{
				Idle:    uimage.NewNineSliceColor(p.Warning),
				Hover:   uimage.NewNineSliceColor(p.WarningHover),
				Pressed: uimage.NewNineSliceColor(p.Warning),
			},
		)
		header.AddChild(h.idleAction)
	} else {
		h.idleAction = nil
	}
	root.AddChild(header)
	dock := widget.NewContainer(
		widget.ContainerOpts.Layout(placed{}),
		widget.ContainerOpts.BackgroundImage(uimage.NewNineSliceColor(p.Surface)),
		widget.ContainerOpts.WidgetOpts(widget.WidgetOpts.LayoutData(l.Dock)),
	)
	keys := []string{"map", "build", "services", "report", "company", "fieldNotes"}
	cols := 6
	if l.DockLeft {
		cols = 1
	} else if l.Compact && h.logicalWidth(g) < 390 {
		cols = 3
	}
	cell := (l.Dock.Dx() - h.px(16) - (cols-1)*h.px(4)) / cols
	for i, key := range keys {
		key := key
		x := h.px(8) + (i%cols)*(cell+h.px(4))
		y := h.px(4) + (i/cols)*h.px(46)
		r := image.Rect(x, y, x+cell, y+h.px(44))
		labelKey := key
		if key == "map" {
			labelKey = "explore"
		}
		b := h.fixedButton(g, t(labelKey), r, func() {
			if key == "fieldNotes" {
				h.helpEvent(g, ui.HelpEvent{Kind: "open"})
			} else {
				h.helpEvent(g, ui.HelpEvent{Kind: "leave", Value: key})
				h.layout(g)
			}
		})
		if key == g.State.View && !g.State.Help.Open || key == "fieldNotes" && g.State.Help.Open {
			b.SetImage(
				&widget.ButtonImage{
					Idle:    uimage.NewBorderedNineSliceColor(p.Selected, p.Accent, 1),
					Hover:   uimage.NewNineSliceColor(p.Selected),
					Pressed: uimage.NewNineSliceColor(p.Pressed),
				},
			)
		}
		dock.AddChild(b)
	}
	root.AddChild(dock)
	h.addMapTools(g, root)
	if !l.Divider.Empty() {
		bar := widget.NewContainer(
			widget.ContainerOpts.BackgroundImage(uimage.NewNineSliceColor(p.Border)),
			widget.ContainerOpts.WidgetOpts(widget.WidgetOpts.LayoutData(l.Divider)),
		)
		root.AddChild(bar)
	}
}
func (h *host) addMapTools(g *Game, root *widget.Container) {
	l := h.workspace
	if l.Map.Empty() {
		return
	}
	keys := []string{"fit", "focusRoute", "zoomIn", "zoomOut"}
	if !g.State.Expanded {
		keys = append(keys, "showMap")
	}
	// Fewer columns give wider buttons and so fewer lines per label. Take the arrangement that covers
	// the least of the map, preferring more columns when two cover the same.
	available := l.Map.Dx() - h.px(16)
	face := h.face(g, 13)
	metrics := face.Metrics()
	line := metrics.HAscent + metrics.HDescent
	least, bestCols, bestWidth := -1.0, 1, h.px(44)
	for cols := len(keys); cols >= 1; cols-- {
		bw := min(h.px(132), max(h.px(44), available/cols-h.px(4)))
		if cols > 1 && cols*(bw+h.px(4)) > available+h.px(4) {
			continue
		}
		// Estimate from line counts; the buttons themselves are built once, for the arrangement chosen.
		inner := float64(max(h.px(20), bw-h.px(8)))
		covered, row := 0.0, 0.0
		for i, key := range keys {
			wrapped := ui.WrapText(ui.Text(g.State.Locale, key), inner, func(v string) float64 {
				return text.Advance(v, face)
			})
			row = max(row, float64(h.px(44)), float64(strings.Count(wrapped, "\n")+1)*line+float64(h.px(6)))
			if i%cols == cols-1 || i == len(keys)-1 {
				covered += row + float64(h.px(4))
				row = 0
			}
		}
		if least < 0 || covered < least {
			least, bestCols, bestWidth = covered, cols, bw
		}
	}
	buttons, rects := h.mapToolRows(g, keys, bestCols, bestWidth)
	for i, b := range buttons {
		h.mapControls = append(h.mapControls, rects[i])
		root.AddChild(b)
	}
}

// mapToolRows places the map tools in rows of cols buttons above the map's bottom edge. Each row is as
// tall as its tallest label.
func (h *host) mapToolRows(g *Game, keys []string, cols, bw int) ([]*widget.Button, []image.Rectangle) {
	l := h.workspace
	buttons := make([]*widget.Button, len(keys))
	rects := make([]image.Rectangle, len(keys))
	rows := (len(keys) + cols - 1) / cols
	heights := make([]int, rows)
	for i, key := range keys {
		buttons[i] = h.fixedButton(g, ui.Text(g.State.Locale, key), image.Rect(0, 0, bw, h.px(44)), h.mapTool(g, key))
		heights[i/cols] = max(heights[i/cols], h.px(44), buttonHeight(buttons[i]))
	}
	covered := h.px(4) * (rows - 1)
	for _, height := range heights {
		covered += height
	}
	y := l.Map.Max.Y - h.px(8) - covered
	for i := range keys {
		if i > 0 && i%cols == 0 {
			y += heights[i/cols-1] + h.px(4)
		}
		x := l.Map.Min.X + h.px(8) + (i%cols)*(bw+h.px(4))
		rects[i] = image.Rect(x, y, x+bw, y+heights[i/cols])
		buttons[i].GetWidget().LayoutData = rects[i]
	}
	return buttons, rects
}

// buttonHeight is the height a hand-placed button needs for its wrapped label.
func buttonHeight(b *widget.Button) int {
	b.Validate()
	_, height := b.PreferredSize()
	return height
}

func (h *host) mapTool(g *Game, key string) func() {
	return func() {
		switch key {
		case "fit":
			if g.State.Region != "" && g.State.Region != "east-asia" {
				g.Camera = mapview.RegionCamera(g.Camera.Width, g.Camera.Height, g.State.Region, g.State.Portrait)
			} else {
				g.Camera = g.Map.Camera(g.Camera.Width, g.Camera.Height, g.State.Portrait)
			}
		case "focusRoute":
			g.focusSelected()
		case "zoomIn":
			g.Camera.ZoomAt(1.3, mapview.Point{X: g.Camera.Width / 2, Y: g.Camera.Height / 2})
		case "zoomOut":
			g.Camera.ZoomAt(1/1.3, mapview.Point{X: g.Camera.Width / 2, Y: g.Camera.Height / 2})
		case "showMap":
			g.State.Expanded = true
			h.layout(g)
		}
		// Every map tool moves the view, which the Services route list follows.
		g.dirty = true
		h.persist(g)
	}
}

func (h *host) resizeWorkspace(g *Game) bool {
	if g.State.Expanded {
		return false
	}
	x, y := ebiten.CursorPosition()
	if inpututil.IsMouseButtonJustPressed(ebiten.MouseButtonLeft) && image.Pt(x, y).In(h.workspace.Divider) {
		h.resizing = true
		g.drag = false
	}
	if !h.resizing {
		return false
	}
	if ebiten.IsMouseButtonPressed(ebiten.MouseButtonLeft) {
		h.dragPanelTo(g, x)
	} else {
		h.resizing = false
		h.persist(g)
	}
	return true
}

// dragPanelTo moves the inspector's inner edge to window x. The preference is kept in 100% units so it
// means the same panel at every text step.
func (h *host) dragPanelTo(g *Game, x int) {
	g.State.PanelWidth = ui.PanelWidth((g.width - x) * 100 / max(100, h.workspace.Step))
	h.layout(g)
	g.dirty = true
}

func (h *host) stepper(g *Game, id, title, value string, minus, plus func()) {
	if h.content == nil {
		return
	}
	p := ui.PaletteFor(g.State)
	face := h.face(g, 13)
	width := h.contentWidth - h.px(32)
	col := widget.NewContainer(
		widget.ContainerOpts.Layout(
			widget.NewRowLayout(
				widget.RowLayoutOpts.Direction(widget.DirectionVertical),
				widget.RowLayoutOpts.Spacing(h.px(4)),
			),
		),
		widget.ContainerOpts.WidgetOpts(
			widget.WidgetOpts.LayoutData(widget.RowLayoutData{Stretch: true, MaxWidth: width}),
		),
	)
	caption := ui.WrapText(title, float64(width), func(v string) float64 { return text.Advance(v, face) })
	col.AddChild(
		widget.NewText(widget.TextOpts.Text(caption, &face, p.Muted), widget.TextOpts.MaxWidth(float64(width))),
	)
	row := widget.NewContainer(
		widget.ContainerOpts.Layout(
			widget.NewGridLayout(
				widget.GridLayoutOpts.Columns(3),
				widget.GridLayoutOpts.Stretch([]bool{false, true, false}, nil),
				widget.GridLayoutOpts.Spacing(h.px(2), h.px(0)),
			),
		),
		widget.ContainerOpts.WidgetOpts(widget.WidgetOpts.LayoutData(widget.RowLayoutData{Stretch: true})),
	)
	for i, run := range []func(){minus, plus} {
		key := []string{"−", "+"}[i]
		run := run
		b := widget.NewButton(
			widget.ButtonOpts.WidgetOpts(widget.WidgetOpts.MinSize(h.px(44), h.px(44))),
			widget.ButtonOpts.Image(
				&widget.ButtonImage{
					Idle:    uimage.NewNineSliceColor(p.Control),
					Hover:   uimage.NewNineSliceColor(p.Selected),
					Pressed: uimage.NewNineSliceColor(p.Pressed),
				},
			),
			widget.ButtonOpts.Text(key, &face, &widget.ButtonTextColor{Idle: p.Text}),
			widget.ButtonOpts.ClickedHandler(func(*widget.ButtonClickedEventArgs) { run() }),
		)
		if i == 0 {
			row.AddChild(b)
			// The value shrinks to fit its box, so it alone may go below the floor.
			shrunk := h.face(g, 13).(*text.GoTextFace)
			shrunk.Size *= min(1, float64(max(24, width-92))/max(1, text.Advance(value, face)))
			valueFace := text.Face(shrunk)
			row.AddChild(
				widget.NewText(
					widget.TextOpts.Text(value, &valueFace, p.Text),
					widget.TextOpts.Position(widget.TextPositionCenter, widget.TextPositionCenter),
				),
			)
		} else {
			row.AddChild(b)
		}
		if h.actions != nil {
			h.actions[id+[]string{"-minus", "-plus"}[i]] = b
		}
	}
	col.AddChild(row)
	h.content.AddChild(col)
}
