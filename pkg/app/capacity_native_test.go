//go:build !js

package app

import (
	"bytes"
	"fmt"
	"github.com/ebitenui/ebitenui/event"
	"github.com/ebitenui/ebitenui/widget"
	"github.com/hajimehoshi/ebiten/v2/text/v2"
	"path/filepath"
	"strings"
	"testing"
	"wwhsrt/assets"
	"wwhsrt/internal/application"
	"wwhsrt/pkg/gameapi"
	"wwhsrt/pkg/mapview"
	"wwhsrt/pkg/render"
	"wwhsrt/pkg/ui"
)

func TestNativeCapacityExplanationFitsAndNamesConflictingServices(t *testing.T) {
	oldPath := preferencesPath
	preferencesPath = filepath.Join(t.TempDir(), "ui.json")
	t.Cleanup(func() { preferencesPath = oldPath })
	port, err := application.Open(nil, nil)
	if err != nil {
		t.Fatal(err)
	}
	act := func(c gameapi.Command) {
		t.Helper()
		c.Revision = port.Snapshot().Revision
		if _, e := port.Apply(c); e != nil {
			t.Fatal(e)
		}
	}
	act(gameapi.Command{Kind: gameapi.TakeLoan, Amount: 300_000_000_000})
	act(gameapi.Command{Kind: gameapi.BuildLine, CorridorID: "tsuruga-osaka", AlignmentID: "mountain"})
	for i := 0; i < 3; i++ {
		act(gameapi.Command{Kind: gameapi.AdvanceYear})
	}
	for _, id := range []string{"tsuruga-osaka", "nagoya-tsuruga-express", "tokyo-tsuruga"} {
		f, e := port.Forecast(gameapi.ForecastRequest{CorridorID: id, Frequency: 4, Fare: 9000})
		if e != nil {
			t.Fatal(e)
		}
		for i := 0; i < f.FleetNeeded; i++ {
			act(gameapi.Command{Kind: gameapi.LeaseTrain, CorridorID: id})
		}
		act(gameapi.Command{Kind: gameapi.SetService, CorridorID: id, Frequency: 4, Fare: 9000})
	}
	before, _ := port.Export()
	font, err := text.NewGoTextFaceSource(bytes.NewReader(assets.Font))
	if err != nil {
		t.Fatal(err)
	}
	for _, locale := range []string{"en", "zh-Hans"} {
		for _, dark := range []bool{false, true} {
			for _, width := range []int{320, 390, 800, 1200, 1920} {
				t.Run(fmt.Sprintf("%s/%v/%d", locale, dark, width), func(t *testing.T) {
					mw := width - 360
					if width < 800 {
						mw = width
					}
					g := &Game{
						Port:   port,
						Frame:  port.Snapshot(),
						width:  width,
						height: 844,
						Map:    &render.Map{Font: font},
						Camera: mapview.New(float64(mw), 280),
						State: ui.State{
							Locale:          locale,
							Dark:            dark,
							View:            "services",
							ServiceCorridor: "tokyo-tsuruga-express",
							Frequency:       1,
							Fare:            18925,
						},
					}
					ui.SetDisclosure(&g.State, "capacity", true)
					g.host.build(g)
					if w, _ := g.host.scroll.PreferredSize(); w > g.host.scroll.ViewRect().Dx() {
						t.Fatal("capacity explanation overflows", w, g.host.scroll.ViewRect())
					}
					copy := ""
					buttons := map[string]func(){}
					g.host.buildServices(
						g,
						func(k string) string { return ui.Text(locale, k) },
						func(s string, _ float64) { copy += s + "\n" },
						func(s string, fn func()) *widget.Button { buttons[s] = fn; return widget.NewButton() },
					)
					for _, want := range []string{
						"13",
						"12",
						ui.Text(locale, "capacityOtherServices"),
						ui.Text(locale, "tsuruga"),
						ui.Text(locale, "nagoya"),
						ui.Text(locale, "capacityDraftLimit"),
					} {
						if !strings.Contains(copy, want) {
							t.Fatal("missing native capacity reason", want, copy)
						}
					}
					if g.host.helpModel(g).Cue.Concept != "" {
						t.Fatal("unsolicited cue competes with blocking capacity error")
					}
					g.host.message = "set_service"
					g.host.actions["frequency-minus"].ClickedEvent.Fire(
						&widget.ButtonClickedEventArgs{Button: g.host.actions["frequency-minus"]},
					)
					event.ExecuteDeferred()
					if g.host.message != "draft_not_applied" {
						t.Fatal("draft edit left a misleading saved receipt", g.host.message)
					}
				})
			}
		}
	}
	after, _ := port.Export()
	if !bytes.Equal(before, after) {
		t.Fatal("rendering diagnostics changed save")
	}
}
