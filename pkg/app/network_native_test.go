//go:build !js

package app

import (
	"bytes"
	"fmt"
	"github.com/hajimehoshi/ebiten/v2/text/v2"
	"testing"
	"wwhsrt/assets"
	"wwhsrt/internal/application"
	"wwhsrt/pkg/gameapi"
	"wwhsrt/pkg/mapview"
	"wwhsrt/pkg/render"
	"wwhsrt/pkg/ui"
)

func TestNativeNetworkViewsFit(t *testing.T) {
	font, err := text.NewGoTextFaceSource(bytes.NewReader(assets.Font))
	if err != nil {
		t.Fatal(err)
	}
	port, err := application.Open(nil, nil)
	if err != nil {
		t.Fatal(err)
	}
	for _, command := range []gameapi.Command{
		{
			Kind:             gameapi.ExtendService,
			CorridorID:       "tokyo-osaka",
			ReplaceServiceID: "tokyo-nagoya",
			Frequency:        6,
			Fare:             13800,
			LeaseCount:       1,
		},
		{Kind: gameapi.AdvanceMonth},
	} {
		command.Revision = port.Snapshot().Revision
		if _, err = port.Apply(command); err != nil {
			t.Fatal(err)
		}
	}
	f := port.Snapshot()
	for _, locale := range []string{"en", "zh-Hans"} {
		for _, dark := range []bool{false, true} {
			for _, width := range []int{320, 390, 800, 1200, 1920} {
				for _, view := range []string{"services", "report", "map"} {
					t.Run(fmt.Sprintf("%s/%v/%d/%s", locale, dark, width, view), func(t *testing.T) {
						mapWidth := width - 360
						if width < 800 {
							mapWidth = width
						}
						g := &Game{
							Port:   port,
							Frame:  f,
							width:  width,
							height: 844,
							Map:    &render.Map{Font: font},
							Camera: mapview.Camera{Width: float64(mapWidth), Height: 280},
							State: ui.State{
								Locale:          locale,
								Dark:            dark,
								View:            view,
								ServiceCorridor: "tokyo-osaka",
								Frequency:       6,
								Fare:            13800,
								NetworkDetails:  true,
								FleetDetails:    true,
								Watching:        true,
								SelectedTrain:   f.Day[0].ID,
								ReplayMinute:    50,
							},
						}
						g.host.build(g)
						if w, _ := g.host.scroll.PreferredSize(); w > g.host.scroll.ViewRect().Dx() {
							t.Fatalf("network content %d wider than available %d", w, g.host.scroll.ViewRect().Dx())
						}
						if view == "map" && g.host.trainDescription == nil {
							t.Fatal("no live train inspector")
						}
					})
				}
			}
		}
	}
}
