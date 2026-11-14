//go:build !js

package app

import (
	"bytes"
	"fmt"
	"image"
	"path/filepath"
	"strings"
	"testing"

	"github.com/ebitenui/ebitenui/event"
	"github.com/ebitenui/ebitenui/widget"
	"github.com/hajimehoshi/ebiten/v2/text/v2"
	"wwhsrt/assets"
	"wwhsrt/internal/application"
	"wwhsrt/pkg/render"
	"wwhsrt/pkg/ui"
)

func TestNativeCampaignCompletionExplainsStoppedTimeAndOffersConfirmedRestart(t *testing.T) {
	old := preferencesPath
	preferencesPath = filepath.Join(t.TempDir(), "ui.json")
	t.Cleanup(func() { preferencesPath = old })
	port, err := application.Open(nil, nil)
	if err != nil {
		t.Fatal(err)
	}
	before, _ := port.Export()
	font, err := text.NewGoTextFaceSource(bytes.NewReader(assets.Font))
	if err != nil {
		t.Fatal(err)
	}
	for _, locale := range []string{"en", "zh-Hans"} {
		for _, size := range [][2]int{{320, 480}, {390, 844}, {800, 600}, {1440, 900}} {
			t.Run(fmt.Sprintf("%s/%v", locale, size), func(t *testing.T) {
				f := port.Snapshot()
				f.CampaignMonths = 360
				f.MonthsResolved = 359
				g := &Game{Port: port, Frame: f, Map: &render.Map{Font: font}, width: size[0], height: size[1], State: ui.NewGameState(ui.State{Locale: locale}, f)}
				g.State.Help.Muted = true
				g.host.build(g)
				event.ExecuteDeferred()
				if g.host.actions["advance"] == nil || g.host.actions["advance"].GetWidget().Disabled || g.host.actions["campaign-new-game"] != nil {
					t.Fatal("campaign ended before its final month")
				}
				g.Frame.MonthsResolved = 360
				for _, view := range []string{"map", "report", "company"} {
					g.State.View = view
					g.host.message = "saved" // Also models reopening a completed save.
					g.host.build(g)
					event.ExecuteDeferred()
					if g.host.actions["advance"] != nil || g.host.actions["advanceYear"] != nil {
						t.Fatal("completed campaign still shows time actions")
					}
					footer := g.host.ui.Container.Children()[1].(*widget.Container)
					footer.Validate()
					found := false
					for _, child := range footer.Children() {
						if label, ok := child.(*widget.Text); ok && strings.Contains(label.Label, ui.Text(locale, "campaignEndTitle")) {
							found = true
							if !strings.Contains(label.Label, "30") {
								t.Fatal("elapsed years missing")
							}
							r := label.GetWidget().LayoutData.(image.Rectangle)
							w, h := label.PreferredSize()
							if w > r.Dx() || h > r.Dy() {
								t.Fatal("completion explanation clipped", w, h, r)
							}
						}
					}
					if !found {
						t.Fatal("persistent completion explanation missing", view)
					}
				}
				b := g.host.actions["campaign-new-game"]
				if b == nil || b.GetWidget().Disabled {
					t.Fatal("restart unavailable")
				}
				b.ClickedEvent.Fire(&widget.ButtonClickedEventArgs{Button: b})
				event.ExecuteDeferred()
				if !g.host.confirmNewGame {
					t.Fatal("restart bypassed confirmation")
				}
				g.host.cancelNewGame(g)
			})
		}
	}
	after, _ := port.Export()
	if !bytes.Equal(before, after) {
		t.Fatal("completion/restart prompt changed saved campaign")
	}
}
