//go:build !js

package app

import (
	"image"
	"runtime"
	"strings"
	"time"
	"unicode"
	"unicode/utf8"

	"github.com/atotto/clipboard"
	"github.com/ebitenui/ebitenui/input"
	"github.com/ebitenui/ebitenui/widget"
	"github.com/go-text/typesetting/segmenter"
	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/exp/textinput"
	"github.com/hajimehoshi/ebiten/v2/inpututil"
	"github.com/hajimehoshi/ebiten/v2/text/v2"
	"github.com/hajimehoshi/ebiten/v2/vector"
	"wwhsrt/pkg/ui"
)

// nameInput participates in Ebiten UI layout/focus/input clipping, while keeping
// the IME-aware Field as the source of truth. Ebiten UI 0.7.3's TextInput lacks
// native IME/clipboard integration and slices rune selection offsets as bytes.
type nameInput struct {
	w                 *widget.Widget
	field             *textinput.Field
	face              text.Face
	palette           ui.Palette
	focused, dragging bool
	anchor, caret     int // UTF-8 byte offsets, always on grapheme boundaries
	offset            float64
	clicked           time.Time
	clickX            int
	readClipboard     func() (string, error)
	writeClipboard    func(string) error
	undo, redo        []nameEdit
	observed          nameEdit
	focusMap          map[widget.FocusDirection]widget.Focuser
	onFocus           func()
}

type nameEdit struct {
	value         string
	anchor, caret int
}

func newNameInput(field *textinput.Field, value string, face text.Face, palette ui.Palette) *nameInput {
	if field.Text() != value {
		field.SetTextAndSelection(value, len(value), len(value))
	}
	e := &nameInput{
		field:          field,
		face:           face,
		palette:        palette,
		readClipboard:  clipboard.ReadAll,
		writeClipboard: clipboard.WriteAll,
		focusMap:       make(map[widget.FocusDirection]widget.Focuser),
	}
	e.anchor, e.caret = field.Selection()
	e.observed = e.snapshot()
	e.w = widget.NewWidget(
		widget.WidgetOpts.MinSize(60, 44),
		widget.WidgetOpts.LayoutData(widget.RowLayoutData{Stretch: true}),
		widget.WidgetOpts.TrackHover(true),
		widget.WidgetOpts.CursorHovered(input.CURSOR_TEXT),
		widget.WidgetOpts.CursorPressed(input.CURSOR_TEXT),
		widget.WidgetOpts.MouseButtonPressedHandler(func(a *widget.WidgetMouseButtonPressedEventArgs) {
			if a.Button != ebiten.MouseButtonLeft || e.w.Disabled {
				return
			}
			e.Focus(true)
			e.pointerDown(a.OffsetX, ebiten.IsKeyPressed(ebiten.KeyShift), time.Now())
		}),
	)
	return e
}

func (e *nameInput) GetWidget() *widget.Widget                          { return e.w }
func (e *nameInput) Validate()                                          {}
func (e *nameInput) PreferredSize() (int, int)                          { return 60, 44 }
func (e *nameInput) SetLocation(r image.Rectangle)                      { e.w.SetLocation(r) }
func (e *nameInput) TabOrder() int                                      { return 0 }
func (e *nameInput) GetFocus(d widget.FocusDirection) widget.Focuser    { return e.focusMap[d] }
func (e *nameInput) AddFocus(d widget.FocusDirection, f widget.Focuser) { e.focusMap[d] = f }
func (e *nameInput) IsFocused() bool                                    { return e.focused }
func (e *nameInput) Focus(focused bool) {
	if focused == e.focused {
		return
	}
	e.focused = focused
	if focused {
		if e.onFocus != nil {
			e.onFocus()
		}
		e.field.Focus()
	} else {
		e.field.Blur()
		e.observeField()
		e.dragging = false
	}
	e.w.FireFocusEvent(e, focused, image.Pt(-1, -1))
}
func (e *nameInput) Update(u *widget.UpdateObject) { e.w.Update(u) }

func (e *nameInput) observeField() {
	if e.field.Text() != e.observed.value {
		e.undo = nameRemember(e.undo, e.observed)
		e.redo = nil
		e.anchor, e.caret = e.field.Selection()
		e.observed = e.snapshot()
	}
}

func nameBoundaries(s string) []int {
	var seg segmenter.Segmenter
	_ = seg.InitWithString(s)
	b := []int{0}
	iter := seg.GraphemeIterator()
	for iter.Next() {
		g := iter.Grapheme()
		b = append(b, g.OffsetInBytes+g.LengthInBytes)
	}
	return b
}
func nameStep(s string, pos, direction int) int {
	b := nameBoundaries(s)
	if direction < 0 {
		for i := len(b) - 1; i >= 0; i-- {
			if b[i] < pos {
				return b[i]
			}
		}
		return 0
	}
	for _, p := range b {
		if p > pos {
			return p
		}
	}
	return len(s)
}
func nameWord(s string, pos, direction int) int {
	space := func(p int) bool { r, _ := utf8.DecodeRuneInString(s[p:]); return unicode.IsSpace(r) }
	if direction < 0 {
		for pos > 0 && space(nameStep(s, pos, -1)) {
			pos = nameStep(s, pos, -1)
		}
		for pos > 0 && !space(nameStep(s, pos, -1)) {
			pos = nameStep(s, pos, -1)
		}
	} else {
		for pos < len(s) && !space(pos) {
			pos = nameStep(s, pos, 1)
		}
		for pos < len(s) && space(pos) {
			pos = nameStep(s, pos, 1)
		}
	}
	return pos
}
func (e *nameInput) selection() (int, int) { return min(e.anchor, e.caret), max(e.anchor, e.caret) }
func (e *nameInput) selectTo(pos int, extend bool) {
	e.caret = pos
	if !extend {
		e.anchor = pos
	}
	a, b := e.selection()
	e.field.SetSelection(a, b)
	e.observed = e.snapshot()
}
func (e *nameInput) snapshot() nameEdit { return nameEdit{e.field.Text(), e.anchor, e.caret} }
func nameRemember(stack []nameEdit, edit nameEdit) []nameEdit {
	if len(stack) >= 100 {
		stack = append([]nameEdit(nil), stack[len(stack)-99:]...)
	}
	return append(stack, edit)
}
func (e *nameInput) replace(value string) {
	before := e.snapshot()
	a, b := e.selection()
	inserted := len(value)
	value = before.value[:a] + value + before.value[b:]
	if value == before.value {
		return
	}
	e.undo = nameRemember(e.undo, before)
	e.redo = nil
	e.anchor = a + inserted
	e.caret = e.anchor
	e.field.SetTextAndSelection(value, e.caret, e.caret)
	e.observed = e.snapshot()
}
func (e *nameInput) restore(redo bool) {
	source, target := &e.undo, &e.redo
	if redo {
		source, target = &e.redo, &e.undo
	}
	if len(*source) == 0 {
		return
	}
	*target = nameRemember(*target, e.snapshot())
	s := (*source)[len(*source)-1]
	*source = (*source)[:len(*source)-1]
	e.anchor, e.caret = s.anchor, s.caret
	a, b := e.selection()
	e.field.SetTextAndSelection(s.value, a, b)
	e.observed = s
}

// command is also used by the regression tests, with an in-memory clipboard.
// Clipboard failures leave the draft and selection intact (especially Cut).
func (e *nameInput) command(key string, extend, word, edge bool) error {
	s := e.field.Text()
	a, b := e.selection()
	switch key {
	case "all":
		e.anchor = 0
		e.selectTo(len(s), true)
	case "copy", "cut":
		if a == b {
			return nil
		}
		if err := e.writeClipboard(s[a:b]); err != nil {
			return err
		}
		if key == "cut" {
			e.replace("")
		}
	case "paste":
		value, err := e.readClipboard()
		if err != nil {
			return err
		}
		if value != "" {
			e.replace(strings.Map(func(r rune) rune {
				if r == '\n' || r == '\r' || r == '\t' {
					return ' '
				}
				if unicode.IsControl(r) || unicode.Is(unicode.Cf, r) && r != '\u200d' {
					return -1
				}
				return r
			}, strings.ToValidUTF8(value, "")))
		}
	case "undo":
		e.restore(false)
	case "redo":
		e.restore(true)
	case "left", "right", "home", "end":
		pos := e.caret
		direction := 1
		if key == "left" {
			direction = -1
		}
		switch {
		case key == "home" || edge && key == "left":
			pos = 0
		case key == "end" || edge && key == "right":
			pos = len(s)
		case !extend && a != b:
			if direction < 0 {
				pos = a
			} else {
				pos = b
			}
		case word:
			pos = nameWord(s, pos, direction)
		default:
			pos = nameStep(s, pos, direction)
		}
		e.selectTo(pos, extend)
	case "backspace", "delete":
		if a == b {
			direction := 1
			if key == "backspace" {
				direction = -1
			}
			pos := nameStep(s, e.caret, direction)
			if word {
				pos = nameWord(s, e.caret, direction)
			}
			if edge {
				if direction < 0 {
					pos = 0
				} else {
					pos = len(s)
				}
			}
			e.selectTo(pos, true)
		}
		e.replace("")
	case "blur":
		e.Focus(false)
	}
	return nil
}

func (e *nameInput) indexAt(x float64) int {
	s := e.field.Text()
	x += e.offset - 10
	previous := 0.0
	for _, p := range nameBoundaries(s)[1:] {
		next := text.Advance(s[:p], e.face)
		if x < (previous+next)/2 {
			return nameStep(s, p, -1)
		}
		previous = next
	}
	return len(s)
}
func (e *nameInput) pointerDown(x int, extend bool, now time.Time) {
	if e.field.UncommittedTextLengthInBytes() > 0 {
		return
	}
	pos := e.indexAt(float64(x))
	if !extend && now.Sub(e.clicked) < 400*time.Millisecond && abs(x-e.clickX) < 5 {
		s := e.field.Text()
		start := pos
		for start > 0 {
			p := nameStep(s, start, -1)
			r, _ := utf8.DecodeRuneInString(s[p:])
			if unicode.IsSpace(r) {
				break
			}
			start = p
		}
		end := pos
		for end < len(s) {
			r, _ := utf8.DecodeRuneInString(s[end:])
			if unicode.IsSpace(r) {
				break
			}
			end = nameStep(s, end, 1)
		}
		e.anchor = start
		e.selectTo(end, true)
		e.dragging = false
		e.clicked = time.Time{}
		return
	}
	e.selectTo(pos, extend)
	e.dragging = true
	e.clicked = now
	e.clickX = x
}
func abs(v int) int {
	if v < 0 {
		return -v
	}
	return v
}

func nameKeyPressed(k ebiten.Key, repeat bool) bool {
	d := inpututil.KeyPressDuration(k)
	return d == 1 || repeat && d >= 24 && (d-24)%3 == 0
}
func (e *nameInput) updateInput() error {
	if !e.focused {
		return nil
	}
	if e.dragging {
		if !ebiten.IsMouseButtonPressed(ebiten.MouseButtonLeft) {
			e.dragging = false
		} else {
			x, _ := ebiten.CursorPosition()
			x -= e.w.Rect.Min.X
			if x < 10 {
				e.offset = max(0, e.offset-8)
			}
			if x > e.w.Rect.Dx()-10 {
				e.offset += 8
			}
			e.selectTo(e.indexAt(float64(x)), true)
		}
	}
	r := e.textRect()
	caret := r.Min.X + int(e.caretAdvance()-e.offset)
	handled, err := e.field.HandleInputWithBounds(image.Rect(caret, r.Min.Y, caret+1, r.Min.Y+24))
	if err != nil {
		return err
	}
	e.observeField()
	if handled || e.field.UncommittedTextLengthInBytes() > 0 {
		return nil
	}
	meta, ctrl, shift, alt := ebiten.IsKeyPressed(
		ebiten.KeyMeta,
	), ebiten.IsKeyPressed(
		ebiten.KeyControl,
	), ebiten.IsKeyPressed(
		ebiten.KeyShift,
	), ebiten.IsKeyPressed(
		ebiten.KeyAlt,
	)
	shortcut, word, edge := ctrl, ctrl, false
	if runtime.GOOS == "darwin" {
		shortcut, word, edge = meta, alt, meta
	}
	if shortcut {
		for _, c := range []struct {
			k      ebiten.Key
			action string
		}{
			{ebiten.KeyA, "all"},
			{ebiten.KeyC, "copy"},
			{ebiten.KeyX, "cut"},
			{ebiten.KeyV, "paste"},
			{ebiten.KeyZ, "undo"},
			{ebiten.KeyY, "redo"},
		} {
			if nameKeyPressed(c.k, false) {
				action := c.action
				if action == "undo" && shift {
					action = "redo"
				}
				return e.command(action, shift, word, edge)
			}
		}
	}
	for _, c := range []struct {
		k      ebiten.Key
		action string
	}{
		{ebiten.KeyLeft, "left"},
		{ebiten.KeyRight, "right"},
		{ebiten.KeyHome, "home"},
		{ebiten.KeyEnd, "end"},
		{ebiten.KeyBackspace, "backspace"},
		{ebiten.KeyDelete, "delete"},
		{ebiten.KeyEnter, "blur"},
		{ebiten.KeyEscape, "blur"},
	} {
		if nameKeyPressed(c.k, true) {
			return e.command(c.action, shift, word, edge)
		}
	}
	return nil
}
func (e *nameInput) textRect() image.Rectangle { r := e.w.Rect.Inset(10); return r }
func (e *nameInput) caretAdvance() float64 {
	s := e.field.TextForRendering()
	a, _ := e.field.Selection()
	if e.field.UncommittedTextLengthInBytes() > 0 {
		_, end, _ := e.field.CompositionSelection()
		return text.Advance(s[:min(len(s), a+end)], e.face)
	}
	return text.Advance(s[:min(len(s), e.caret)], e.face)
}
func (e *nameInput) Render(screen *ebiten.Image) {
	r := e.w.Rect
	p := e.palette
	bg, border := p.Control, p.Border
	if e.focused {
		border = p.Accent
	}
	vector.FillRect(screen, float32(r.Min.X), float32(r.Min.Y), float32(r.Dx()), float32(r.Dy()), bg, false)
	vector.StrokeRect(
		screen,
		float32(r.Min.X)+.5,
		float32(r.Min.Y)+.5,
		float32(r.Dx()-1),
		float32(r.Dy()-1),
		1,
		border,
		false,
	)
	r = e.textRect()
	if r.Empty() {
		return
	}
	canvas := screen.SubImage(r).(*ebiten.Image)
	caret := e.caretAdvance()
	if e.focused {
		e.offset = max(0, max(caret-float64(r.Dx()-2), min(e.offset, caret)))
	}
	x, y := float64(r.Min.X)-e.offset, float64(r.Min.Y)
	a, b := e.field.Selection()
	s := e.field.Text()
	if e.focused && a != b && e.field.UncommittedTextLengthInBytes() == 0 {
		left, right := text.Advance(s[:a], e.face), text.Advance(s[:b], e.face)
		vector.FillRect(canvas, float32(x+left), float32(y), float32(right-left), 24, p.Action, false)
	}
	op := &text.DrawOptions{}
	op.GeoM.Translate(x, y)
	op.ColorScale.ScaleWithColor(p.Text)
	text.Draw(canvas, e.field.TextForRendering(), e.face, op)
	if e.focused && a != b && e.field.UncommittedTextLengthInBytes() == 0 {
		left, right := text.Advance(s[:a], e.face), text.Advance(s[:b], e.face)
		selection := image.Rect(int(x+left), r.Min.Y, int(x+right+1), r.Max.Y).Intersect(r)
		if !selection.Empty() {
			op.ColorScale.Reset()
			op.ColorScale.ScaleWithColor(p.OnAction)
			text.Draw(canvas.SubImage(selection).(*ebiten.Image), s, e.face, op)
		}
	}
	if e.focused && (time.Now().UnixMilli()/500)%2 == 0 {
		vector.FillRect(canvas, float32(x+caret), float32(y), 1, 24, p.Text, false)
	}
	if e.field.UncommittedTextLengthInBytes() > 0 {
		left := text.Advance(s[:a], e.face)
		rendered := e.field.TextForRendering()
		right := text.Advance(rendered[:a+e.field.UncommittedTextLengthInBytes()], e.face)
		vector.FillRect(canvas, float32(x+left), float32(y+23), float32(right-left), 1, p.Accent, false)
	}
}

func (h *host) blurName() {
	if h.nameInput != nil {
		h.nameInput.Focus(false)
	}
	h.field.Blur()
}
func (h *host) syncName(g *Game) {
	if value := h.field.Text(); value != g.State.DraftName {
		g.State.DraftName = value
		h.persist(g)
	}
}
