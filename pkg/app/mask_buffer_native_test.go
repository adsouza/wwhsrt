//go:build !js

package app

import (
	"reflect"
	"testing"
	"unsafe"

	uimage "github.com/ebitenui/ebitenui/image"
	"github.com/ebitenui/ebitenui/widget"
	"wwhsrt/pkg/ui"
)

// maskBufferOf reads the mask buffer a scroll area or text field draws through. Once drawn, each
// buffer holds two images the size of the whole window.
func maskBufferOf(t *testing.T, w any) *uimage.MaskedRenderBuffer {
	t.Helper()
	f := reflect.ValueOf(w).Elem().FieldByName("renderBuf")
	if !f.IsValid() {
		t.Fatalf("%T no longer has a renderBuf field", w)
	}
	return *(**uimage.MaskedRenderBuffer)(unsafe.Pointer(f.UnsafeAddr()))
}

func TestNativeRebuildKeepsItsMaskBuffers(t *testing.T) {
	g := chromeGame(t, 1280, 850, ui.State{View: "company", CompanyPane: "finance"})
	if g.host.scroll == nil || g.host.loanInput == nil {
		t.Fatal("the finance pane should hold a scroll area and the loan field")
	}
	scroll, input := maskBufferOf(t, g.host.scroll), maskBufferOf(t, g.host.loanInput)
	if scroll == input {
		t.Fatal("the loan field draws while the scroll area around it is mid-draw, so they cannot share a buffer")
	}
	for range 3 {
		previous := g.host.scroll
		g.host.build(g)
		if g.host.scroll == previous {
			t.Fatal("the rebuild did not make a new scroll area")
		}
		if maskBufferOf(t, g.host.scroll) != scroll || maskBufferOf(t, g.host.loanInput) != input {
			t.Fatal("a rebuild made new window-sized mask buffers; the old ones stay on the GPU until a collection")
		}
	}
	g.State.View = "services"
	g.host.build(g)
	if maskBufferOf(t, g.host.scroll) != scroll {
		t.Fatal("changing view made a new scroll mask buffer")
	}
	g.host.confirmNewGame = true
	g.host.scroll = nil
	g.host.build(g)
	if g.host.scroll == nil || maskBufferOf(t, g.host.scroll) != scroll {
		t.Fatal("the New game confirmation made a new scroll mask buffer")
	}
}

func TestShareMaskBufferReportsWhatItCouldNotShare(t *testing.T) {
	var shared *uimage.MaskedRenderBuffer
	scroll := widget.NewScrollContainer()
	if !shareMaskBuffer(scroll, &shared) || shared == nil || maskBufferOf(t, scroll) != shared {
		t.Fatal("a scroll area did not take the shared buffer")
	}
	first := shared
	input := widget.NewTextInput()
	if !shareMaskBuffer(input, &shared) || shared != first || maskBufferOf(t, input) != first {
		t.Fatal("a second widget should take the buffer that already exists")
	}
	if shareMaskBuffer(widget.NewContainer(), &shared) || shareMaskBuffer(nil, &shared) || shareMaskBuffer(42, &shared) {
		t.Fatal("a widget without a mask buffer was reported as sharing one")
	}
}
