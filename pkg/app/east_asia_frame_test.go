//go:build !js

package app

import (
	"encoding/json"
	"testing"

	"wwhsrt/assets"
	"wwhsrt/pkg/mapview"
	"wwhsrt/pkg/render"
	"wwhsrt/pkg/ui"
)

// East Asia keeps Dhaka off screen in both orientations by panning within the frame's spare room,
// while every anchor keeps its marker clearance. Measured at the Map panels real windows produce.
func TestEastAsiaFrameKeepsDhakaOffScreen(t *testing.T) {
	m, e := render.New()
	if e != nil {
		t.Fatal(e)
	}
	var anchors map[string][][2]float64
	if e = json.Unmarshal(assets.Camera, &anchors); e != nil {
		t.Fatal(e)
	}
	dhaka := mapview.Project(23.8103, 90.4125)
	for _, win := range [][2]int{{1920, 1080}, {1440, 900}, {1280, 800}, {1024, 768}, {800, 600}, {844, 390}, {768, 1024}, {390, 844}, {360, 780}, {430, 932}} {
		portrait := win[1] > win[0]
		panel := ui.Workspace(win[0], win[1], ui.State{View: "map"}).Map
		g := &Game{Map: m, Camera: mapview.New(float64(panel.Dx()), float64(panel.Dy()))}
		c := g.cameraForRegion("east-asia", portrait)
		if d := c.Screen(dhaka); !c.Clear(d) {
			t.Errorf("%dx%d: Dhaka at %.1f,%.1f is within %v px of the %.0fx%.0f map", win[0], win[1], d.X, d.Y, mapview.MarkerClearance, c.Width, c.Height)
		}
		key := "landscape"
		if portrait {
			key = "portrait"
		}
		for _, a := range anchors[key] {
			q := c.Screen(mapview.Project(a[0], a[1]))
			if q.X < mapview.MarkerClearance-1e-9 || q.X > c.Width-mapview.MarkerClearance+1e-9 ||
				q.Y < mapview.MarkerClearance-1e-9 || q.Y > c.Height-mapview.MarkerClearance+1e-9 {
				t.Errorf("%dx%d: anchor %v at %.1f,%.1f left its marker clearance", win[0], win[1], a, q.X, q.Y)
			}
		}
	}
}
