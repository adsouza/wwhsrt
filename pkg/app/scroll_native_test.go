//go:build !js

package app

import (
	"bytes"
	"fmt"
	"image"
	"math"
	"testing"

	"github.com/ebitenui/ebitenui/event"
	"github.com/ebitenui/ebitenui/widget"
	"github.com/hajimehoshi/ebiten/v2/text/v2"
	"wwhsrt/assets"
	"wwhsrt/internal/application"
	"wwhsrt/pkg/gameapi"
	"wwhsrt/pkg/mapview"
	"wwhsrt/pkg/render"
	"wwhsrt/pkg/ui"
)

func TestNativeReportScrollEventsReachBothEnds(t *testing.T) {
	font, err := text.NewGoTextFaceSource(bytes.NewReader(assets.Font))
	if err != nil {
		t.Fatal(err)
	}
	port, err := application.Open(nil, nil)
	if err != nil {
		t.Fatal(err)
	}
	if _, err = port.Apply(gameapi.Command{Kind: gameapi.AdvanceMonth}); err != nil {
		t.Fatal(err)
	}
	before := port.Snapshot().Checkpoint
	for _, locale := range []string{"en", "zh-Hans"} {
		for _, size := range [][2]int{{1200, 800}, {800, 600}, {390, 844}, {320, 480}} {
			t.Run(fmt.Sprintf("%s/%dx%d", locale, size[0], size[1]), func(t *testing.T) {
				mapW, mapH := size[0]-360, size[1]
				if size[0] < 800 {
					mapW, mapH = size[0], max(180, size[1]/3)
				}
				g := &Game{Port: port, Frame: port.Snapshot(), Map: &render.Map{Font: font}, width: size[0], height: size[1],
					Camera: mapview.Camera{Width: float64(mapW), Height: float64(mapH)}, State: ui.State{Locale: locale, View: "report"}}
				g.host.build(g)
				h := &g.host
				event.ExecuteDeferred()
				viewport := h.scroll.ViewRect()
				_, contentHeight := h.scroll.PreferredSize()
				overflow := contentHeight - viewport.Dy()
				if overflow <= 0 {
					t.Fatal("fixture must contain a populated, overflowing monthly report")
				}
				footer := h.ui.Container.Children()[1].GetWidget().LayoutData.(image.Rectangle)
				if viewport.Max.Y > footer.Min.Y || viewport.Overlaps(h.scrollbar.GetWidget().Rect) {
					t.Fatal("footer or scrollbar overlaps report viewport")
				}
				if h.scrollbar.GetWidget().GetVisibility() != widget.Visibility_Show {
					t.Fatal("overflow has no visible scrollbar")
				}
				wheel := func(delta float64) {
					h.scroll.GetWidget().ScrolledEvent.Fire(&widget.WidgetScrolledEventArgs{Widget: h.scroll.GetWidget(), Y: delta})
					event.ExecuteDeferred()
				}
				wheel(-.25)
				if pixels := h.scroll.ScrollTop * float64(overflow); math.Abs(pixels-12) > 1e-7 {
					t.Fatalf("fractional trackpad input lost: moved %.3f pixels", pixels)
				}
				wheel(-10000)
				if h.scroll.ScrollTop != 1 || h.scrollbar.Current != sidebarScrollSteps {
					t.Fatal("wheel cannot reach report bottom")
				}
				wheel(10000)
				if h.scroll.ScrollTop != 0 || h.scrollbar.Current != 0 {
					t.Fatal("wheel cannot return to top")
				}
				h.scrollbar.Current = sidebarScrollSteps / 2
				h.scrollbar.ChangedEvent.Fire(&widget.SliderChangedEventArgs{Slider: h.scrollbar, Current: h.scrollbar.Current, Dragging: true})
				event.ExecuteDeferred()
				if h.scroll.ScrollTop != .5 {
					t.Fatal("dragging scrollbar does not move report")
				}
				h.scrollSidebar(100000)
				h.build(g)
				event.ExecuteDeferred()
				if h.scroll.ScrollTop != 1 || h.scrollbar.Current != sidebarScrollSteps {
					t.Fatal("widget rebuild lost scroll position")
				}
				stale := h.scroll
				g.State.View = "map"
				h.build(g)
				event.ExecuteDeferred()
				stale.GetWidget().ScrolledEvent.Fire(&widget.WidgetScrolledEventArgs{Widget: stale.GetWidget(), Y: -1000})
				event.ExecuteDeferred()
				if h.scroll.ScrollTop != 0 {
					t.Fatal("old report event scrolled a different view")
				}
				g.State.Expanded = true
				h.scroll.GetWidget().ScrolledEvent.Fire(&widget.WidgetScrolledEventArgs{Widget: h.scroll.GetWidget(), Y: -1000})
				event.ExecuteDeferred()
				if h.scroll.ScrollTop != 0 {
					t.Fatal("expanded map input scrolled the hidden sidebar")
				}
				g.State.Expanded = false
				g.State.View = "report"
				g.height = 5000
				g.Camera.Height = 5000
				if g.width < 800 {
					g.Camera.Height = 5000 / 3
				}
				h.build(g)
				event.ExecuteDeferred()
				h.scrollSidebar(100000)
				if h.scroll.ScrollTop != 0 || h.scrollbar.GetWidget().GetVisibility() != widget.Visibility_Hide {
					t.Fatal("non-overflowing content should not scroll or show a scrollbar")
				}
			})
		}
	}
	if port.Snapshot().Checkpoint != before {
		t.Fatal("scrolling changed simulation state")
	}
}
