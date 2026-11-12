//go:build !js

package app

import (
	"bytes"
	"errors"
	"image"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
	"unicode/utf8"

	"github.com/ebitenui/ebitenui/event"
	"github.com/ebitenui/ebitenui/widget"
	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/exp/textinput"
	"github.com/hajimehoshi/ebiten/v2/text/v2"
	"wwhsrt/assets"
	"wwhsrt/internal/application"
	"wwhsrt/pkg/mapview"
	"wwhsrt/pkg/render"
	"wwhsrt/pkg/ui"
)

func nameFixture(t *testing.T, value string) (*nameInput, *string) {
	t.Helper()
	source, err := text.NewGoTextFaceSource(bytes.NewReader(assets.Font))
	if err != nil {
		t.Fatal(err)
	}
	e := newNameInput(&textinput.Field{}, value, &text.GoTextFace{Source: source, Size: 16}, ui.PaletteFor(ui.State{}))
	e.SetLocation(image.Rect(0, 0, 300, 44))
	clip := ""
	e.readClipboard = func() (string, error) { return clip, nil }
	e.writeClipboard = func(s string) error { clip = s; return nil }
	t.Cleanup(func() { e.Focus(false); event.ExecuteDeferred() })
	return e, &clip
}
func nameCommand(t *testing.T, e *nameInput, key string, shift bool) {
	t.Helper()
	if err := e.command(key, shift, false, false); err != nil {
		t.Fatal(err)
	}
}

func TestNativeNameUnicodeClipboardAndUndo(t *testing.T) {
	e, clip := nameFixture(t, "晨曦晨曦 Railway")
	// The repeated substring must be removed at the selected position, not the
	// first match; byte offsets must not split either Chinese character.
	e.selectTo(len("晨曦"), false)
	e.selectTo(len("晨曦晨曦"), true)
	nameCommand(t, e, "copy", false)
	if *clip != "晨曦" {
		t.Fatal(*clip)
	}
	nameCommand(t, e, "cut", false)
	if e.field.Text() != "晨曦 Railway" {
		t.Fatal(e.field.Text())
	}
	nameCommand(t, e, "undo", false)
	if e.field.Text() != "晨曦晨曦 Railway" {
		t.Fatal(e.field.Text())
	}
	nameCommand(t, e, "redo", false)
	if e.field.Text() != "晨曦 Railway" {
		t.Fatal(e.field.Text())
	}
	*clip = "国际高速铁路"
	nameCommand(t, e, "paste", false)
	if e.field.Text() != "晨曦国际高速铁路 Railway" || !utf8.ValidString(e.field.Text()) {
		t.Fatal(e.field.Text())
	}
	nameCommand(t, e, "all", false)
	*clip = "Sunrise Rail"
	nameCommand(t, e, "paste", false)
	if e.field.Text() != "Sunrise Rail" {
		t.Fatal(e.field.Text())
	}
	nameCommand(t, e, "undo", false)
	if e.field.Text() != "晨曦国际高速铁路 Railway" {
		t.Fatal(e.field.Text())
	}
	nameCommand(t, e, "copy", false)
	if *clip != "晨曦国际高速铁路 Railway" {
		t.Fatal(*clip)
	}
}

func TestNativeNameSelectionNavigationAndGraphemes(t *testing.T) {
	e, _ := nameFixture(t, "A晨e\u0301👩‍🚆Z")
	nameCommand(t, e, "left", true)
	if a, b := e.selection(); e.field.Text()[a:b] != "Z" {
		t.Fatal(a, b)
	}
	nameCommand(t, e, "left", true)
	if a, b := e.selection(); e.field.Text()[a:b] != "👩‍🚆Z" {
		t.Fatal(a, b)
	}
	nameCommand(t, e, "right", false) // Collapse to selection's right edge.
	if a, b := e.selection(); a != b || b != len(e.field.Text()) {
		t.Fatal(a, b)
	}
	nameCommand(t, e, "backspace", false)
	nameCommand(t, e, "backspace", false)
	if e.field.Text() != "A晨e\u0301" {
		t.Fatal(e.field.Text())
	}
	nameCommand(t, e, "backspace", false)
	if e.field.Text() != "A晨" {
		t.Fatal(e.field.Text())
	}
	nameCommand(t, e, "home", false)
	nameCommand(t, e, "right", true)
	nameCommand(t, e, "delete", false)
	if e.field.Text() != "晨" {
		t.Fatal(e.field.Text())
	}
	nameCommand(t, e, "home", false)
	nameCommand(t, e, "delete", false)
	if e.field.Text() != "" {
		t.Fatal(e.field.Text())
	}
	nameCommand(t, e, "backspace", false)
	nameCommand(t, e, "right", true)
	if e.field.Text() != "" {
		t.Fatal("empty-field boundary")
	}
}

func TestNativeNamePointerSelectionAndWordMovement(t *testing.T) {
	e, _ := nameFixture(t, "Sunrise 晨曦 Rail")
	x := func(s string) int { return 10 + int(text.Advance(s, e.face)+.5) }
	now := time.Now()
	e.GetWidget().MouseButtonPressedEvent.Fire(&widget.WidgetMouseButtonPressedEventArgs{Widget: e.GetWidget(), Button: ebiten.MouseButtonLeft, OffsetX: x("Sunrise ")})
	event.ExecuteDeferred()
	if !e.IsFocused() || !e.field.IsFocused() || e.caret != len("Sunrise ") {
		t.Fatal("widget press did not place caret/focus", e.caret)
	}
	e.selectTo(e.indexAt(float64(x("Sunrise 晨曦"))), true)
	if a, b := e.selection(); e.field.Text()[a:b] != "晨曦" {
		t.Fatal(a, b)
	}
	e.pointerDown(x("Sun"), false, now.Add(time.Second))
	e.pointerDown(x("Sun"), false, now.Add(time.Second+100*time.Millisecond))
	if a, b := e.selection(); e.field.Text()[a:b] != "Sunrise" {
		t.Fatal(a, b)
	}
	nameCommand(t, e, "end", false)
	if err := e.command("left", true, true, false); err != nil {
		t.Fatal(err)
	}
	if a, b := e.selection(); e.field.Text()[a:b] != "Rail" {
		t.Fatal(a, b)
	}
	if err := e.command("left", true, true, false); err != nil {
		t.Fatal(err)
	}
	if a, b := e.selection(); e.field.Text()[a:b] != "晨曦 Rail" {
		t.Fatal(a, b)
	}
	// Horizontal scrolling must preserve hit testing and keep a caret visible.
	e.offset = text.Advance("Sunrise ", e.face)
	if pos := e.indexAt(10); pos != len("Sunrise ") {
		t.Fatal("scrolled hit test", pos)
	}
}

func TestNativeNameClipboardFailuresAndSingleLine(t *testing.T) {
	e, clip := nameFixture(t, "晨曦 Rail")
	nameCommand(t, e, "all", false)
	before := e.snapshot()
	unavailable := errors.New("clipboard unavailable")
	e.writeClipboard = func(string) error { return unavailable }
	if err := e.command("cut", false, false, false); !errors.Is(err, unavailable) || e.snapshot() != before {
		t.Fatal("failed Cut changed selection/draft")
	}
	e.readClipboard = func() (string, error) { return "", unavailable }
	if err := e.command("paste", false, false, false); !errors.Is(err, unavailable) || e.snapshot() != before {
		t.Fatal("failed Paste changed selection/draft")
	}
	e.readClipboard = func() (string, error) { return *clip, nil }
	*clip = ""
	nameCommand(t, e, "paste", false)
	if e.snapshot() != before {
		t.Fatal("empty clipboard erased selection")
	}
	*clip = "New\nRail\t晨曦\x00"
	nameCommand(t, e, "paste", false)
	if e.field.Text() != "New Rail 晨曦" {
		t.Fatal(e.field.Text())
	}
}

func TestNativeNameDraftFocusAndCampaignIsolation(t *testing.T) {
	e, _ := nameFixture(t, "")
	port, err := application.Open(nil, nil)
	if err != nil {
		t.Fatal(err)
	}
	g := &Game{Port: port, Frame: port.Snapshot(), Map: &render.Map{Font: e.face.(*text.GoTextFace).Source}, width: 1200, height: 800, Camera: mapview.Camera{Width: 840, Height: 800}, State: ui.State{Locale: "en", View: "company", DraftName: "晨曦 Rail", LoanAmount: "1000000"}}
	old := preferencesPath
	preferencesPath = filepath.Join(t.TempDir(), "ui.json")
	t.Cleanup(func() { preferencesPath = old })
	g.host.build(g)
	event.ExecuteDeferred()
	input := g.host.nameInput
	g.host.ui.SetFocusedWidget(input)
	event.ExecuteDeferred()
	if g.host.ui.GetFocusedWidget() != input || !input.field.IsFocused() {
		t.Fatal("editor not integrated with UI focus")
	}
	g.host.ui.ChangeFocus(widget.FOCUS_NEXT)
	event.ExecuteDeferred()
	if input.IsFocused() || input.field.IsFocused() {
		t.Fatal("Tab focus left editor active")
	}
	g.host.ui.ChangeFocus(widget.FOCUS_PREVIOUS)
	event.ExecuteDeferred()
	if !input.IsFocused() || !input.field.IsFocused() {
		t.Fatal("Shift-Tab did not return to editor")
	}
	before := port.Snapshot().Checkpoint
	nameCommand(t, input, "all", false)
	input.replace("新公司")
	g.host.syncName(g)
	if g.State.DraftName != "新公司" || port.Snapshot().Checkpoint != before || port.Snapshot().Player.Name == "新公司" {
		t.Fatal("editing applied business command")
	}
	saved, err := os.ReadFile(preferencesPath)
	if err != nil || !strings.Contains(string(saved), "新公司") {
		t.Fatal("draft not saved", err)
	}
	g.host.ui.SetFocusedWidget(g.host.loanInput)
	event.ExecuteDeferred()
	if input.field.IsFocused() || input.IsFocused() {
		t.Fatal("loan field did not release name input")
	}
	// A pointer press must also transfer focus away from the loan widget.
	input.GetWidget().MouseButtonPressedEvent.Fire(&widget.WidgetMouseButtonPressedEventArgs{Widget: input.GetWidget(), Button: ebiten.MouseButtonLeft, OffsetX: 10})
	event.ExecuteDeferred()
	if !input.IsFocused() || g.host.loanInput.IsFocused() {
		t.Fatal("click did not transfer focus from loan to name")
	}
	nameCommand(t, input, "all", false)
	g.host.build(g)
	event.ExecuteDeferred()
	input = g.host.nameInput
	if !input.IsFocused() {
		t.Fatal("resize/rebuild lost editor focus")
	}
	if a, b := input.selection(); a != 0 || b != len("新公司") {
		t.Fatal("resize/rebuild lost selection", a, b)
	}
	nameCommand(t, input, "undo", false)
	if input.field.Text() != "晨曦 Rail" {
		t.Fatal("resize/rebuild lost undo history", input.field.Text())
	}
	g.host.openNewGame(g)
	event.ExecuteDeferred()
	if input.field.IsFocused() || input.IsFocused() {
		t.Fatal("modal left invisible editor active")
	}
}
