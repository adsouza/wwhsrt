//go:build !js

package app

import (
	"bytes"
	"github.com/hajimehoshi/ebiten/v2/text/v2"
	"testing"
	"wwhsrt/assets"
	"wwhsrt/internal/application"
	"wwhsrt/pkg/gameapi"
	"wwhsrt/pkg/mapview"
	"wwhsrt/pkg/render"
	"wwhsrt/pkg/ui"
)

func TestNativeExtensionApplyEnabledWithLargeFleet(t *testing.T) {
	font, err := text.NewGoTextFaceSource(bytes.NewReader(assets.Font))
	if err != nil {
		t.Fatal(err)
	}
	port, err := application.Open(nil, nil)
	if err != nil {
		t.Fatal(err)
	}
	parked := ""
	for _, r := range port.Snapshot().Routes {
		if r.Open && r.ID != "tokyo-nagoya" && r.ID != "tokyo-osaka" {
			parked = r.ID
			break
		}
	}
	if parked == "" {
		t.Fatal("no third open route to hold trains")
	}
	// Thirty parked trains plus the two starters reach the old limit, so the
	// extension's one additional lease would have crossed it.
	for i := 0; i < 30; i++ {
		if _, err = port.Apply(
			gameapi.Command{Kind: gameapi.LeaseTrain, CorridorID: parked, Revision: port.Snapshot().Revision},
		); err != nil {
			t.Fatal(err)
		}
	}
	g := &Game{
		Port:   port,
		Frame:  port.Snapshot(),
		width:  1200,
		height: 844,
		Map:    &render.Map{Font: font},
		Camera: mapview.Camera{Width: 840, Height: 280},
		State: ui.State{
			Locale:           "en",
			View:             "services",
			ServiceCorridor:  "tokyo-osaka",
			Frequency:        6,
			Fare:             13800,
			ReplaceServiceID: "tokyo-nagoya",
		},
	}
	g.host.build(g)
	if g.host.primaryAction == nil {
		t.Fatal("no apply-extension action")
	}
	if g.host.primaryAction.GetWidget().Disabled {
		t.Fatal("apply extension disabled although the company may lease the additional train")
	}
}
