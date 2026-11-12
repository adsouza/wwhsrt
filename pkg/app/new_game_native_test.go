//go:build !js

package app

import (
	"bytes"
	"fmt"
	"image"
	"testing"

	"github.com/ebitenui/ebitenui/event"
	"github.com/ebitenui/ebitenui/widget"
	"github.com/hajimehoshi/ebiten/v2/text/v2"
	"wwhsrt/assets"
	"wwhsrt/internal/application"
	"wwhsrt/pkg/mapview"
	"wwhsrt/pkg/render"
	"wwhsrt/pkg/ui"
)

func TestNativeNewGameConfirmationFitsAndCancels(t *testing.T) {
	font, err := text.NewGoTextFaceSource(bytes.NewReader(assets.Font))
	if err != nil {
		t.Fatal(err)
	}
	port, err := application.Open(nil, nil)
	if err != nil {
		t.Fatal(err)
	}
	before := port.Snapshot().Checkpoint
	for _, locale := range []string{"en", "zh-Hans"} {
		for _, size := range [][2]int{{320, 480}, {390, 844}, {800, 600}, {1200, 800}} {
			t.Run(fmt.Sprintf("%s/%v", locale, size), func(t *testing.T) {
				g := &Game{Port: port, Frame: port.Snapshot(), Map: &render.Map{Font: font}, width: size[0], height: size[1], Camera: mapview.Camera{Width: float64(size[0]), Height: 200}, State: ui.State{Locale: locale, View: "company", DraftName: "Unapplied company name", LoanAmount: "12345"}}
				g.host.openNewGame(g)
				g.host.build(g)
				event.ExecuteDeferred()
				if !g.host.blocksMap() {
					t.Fatal("confirmation permits background map input")
				}
				if w, _ := g.host.scroll.PreferredSize(); w > g.host.scroll.ViewRect().Dx() {
					t.Fatal("confirmation text overflows")
				}
				children := g.host.ui.Container.Children()
				actions := children[len(children)-1].(*widget.Container)
				r := actions.GetWidget().LayoutData.(image.Rectangle)
				w, h := actions.PreferredSize()
				if w > r.Dx() || h > r.Dy() || r.Max.Y > size[1] || g.host.scroll.ViewRect().Max.Y > r.Min.Y {
					t.Fatal("confirmation actions clipped", w, h, r)
				}
				// Cancel through the actual button event; drafts and accepted game survive.
				cancel := actions.Children()[0].(*widget.Button)
				cancel.ClickedEvent.Fire(&widget.ButtonClickedEventArgs{Button: cancel})
				event.ExecuteDeferred()
				if g.host.confirmNewGame || g.State.DraftName != "Unapplied company name" || g.State.LoanAmount != "12345" || port.Snapshot().Checkpoint != before {
					t.Fatal("cancel changed game or drafts")
				}
			})
		}
	}
}
