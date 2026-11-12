//go:build !js

package app

import (
	"bytes"
	"fmt"
	"image"
	"path/filepath"
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

func TestNativeIdleWarningsFitAndOpenServicePlanning(t *testing.T) {
	font, err := text.NewGoTextFaceSource(bytes.NewReader(assets.Font))
	if err != nil {
		t.Fatal(err)
	}
	port, err := application.Open(nil, nil)
	if err != nil {
		t.Fatal(err)
	}
	act := func(c gameapi.Command) {
		c.Revision = port.Snapshot().Revision
		if _, e := port.Apply(c); e != nil {
			t.Fatal(e)
		}
	}
	act(gameapi.Command{Kind: gameapi.TakeLoan, Amount: 500000000000})
	act(gameapi.Command{Kind: gameapi.BuildLine, CorridorID: "tsuruga-osaka", AlignmentID: "mountain"})
	act(gameapi.Command{Kind: gameapi.BuildLine, CorridorID: "hokkaido-extension", AlignmentID: "mountain"})
	for i := 0; i < 4; i++ {
		act(gameapi.Command{Kind: gameapi.AdvanceYear})
	}
	frame := port.Snapshot()
	if len(frame.IdleLines) != 2 {
		t.Fatal("fixture needs two idle owned lines")
	}
	before := frame.Checkpoint
	oldPath := preferencesPath
	preferencesPath = filepath.Join(t.TempDir(), "ui.json")
	t.Cleanup(func() { preferencesPath = oldPath })
	for _, locale := range []string{"en", "zh-Hans"} {
		for _, size := range [][2]int{{1200, 800}, {800, 600}, {390, 844}, {320, 480}} {
			t.Run(fmt.Sprintf("%s/%dx%d", locale, size[0], size[1]), func(t *testing.T) {
				mapW, mapH := size[0]-360, size[1]
				if size[0] < 800 {
					mapW, mapH = size[0], max(180, size[1]/3)
				}
				g := &Game{Port: port, Frame: frame, Map: &render.Map{Font: font}, width: size[0], height: size[1], Camera: mapview.Camera{Width: float64(mapW), Height: float64(mapH), Zoom: 1}, State: ui.State{Locale: locale, View: "report", Portrait: size[1] > size[0]}}
				g.host.build(g)
				event.ExecuteDeferred()
				footer := g.host.ui.Container.Children()[1].(*widget.Container)
				bounds := footer.GetWidget().LayoutData.(image.Rectangle)
				w, h := footer.PreferredSize()
				if w > bounds.Dx() || h > bounds.Dy() || g.host.scroll.ViewRect().Max.Y > bounds.Min.Y {
					t.Fatal("idle warning/footer clipped", w, h, bounds)
				}
				if w, _ := g.host.scroll.PreferredSize(); w > g.host.scroll.ViewRect().Dx() {
					t.Fatal("warning widens sidebar")
				}
				review := footer.Children()[0].(*widget.Button)
				review.ClickedEvent.Fire(&widget.ButtonClickedEventArgs{Button: review})
				event.ExecuteDeferred()
				if g.State.View != "services" || g.State.ServiceCorridor != frame.IdleLines[0].CorridorID || g.State.Expanded {
					t.Fatal("review did not open affected line")
				}
				if port.Snapshot().Checkpoint != before {
					t.Fatal("review changed finances or accepted service")
				}
			})
		}
	}
}
