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

// Regions that keep a metro out of view do so by panning within the frame's spare room, while every
// anchor keeps its marker clearance: East Asia keeps Dhaka off screen in both orientations, portrait
// Africa+Arabia keeps Karachi off. Measured at the Map panels real windows produce.
func TestRegionFramesKeepExcludedMetrosOffScreen(t *testing.T) {
	m, e := render.New()
	if e != nil {
		t.Fatal(e)
	}
	var eastAsia map[string][][2]float64 // lat, lon
	if e = json.Unmarshal(assets.Camera, &eastAsia); e != nil {
		t.Fatal(e)
	}
	anchors := func(id string, portrait bool) [][2]float64 {
		key := "landscape"
		if portrait {
			key = "portrait"
		}
		if id == "east-asia" {
			return eastAsia[key]
		}
		for _, r := range mapview.Regions {
			if r.ID == id {
				v := r.Landscape
				if portrait {
					v = r.Portrait
				}
				out := [][2]float64{}
				for _, p := range v.Points {
					out = append(out, [2]float64{p[1], p[0]})
				}
				return out
			}
		}
		t.Fatal("no region", id)
		return nil
	}
	windows := [][2]int{{1920, 1080}, {1440, 900}, {1280, 800}, {1024, 768}, {800, 600}, {844, 390}, {768, 1024}, {390, 844}, {360, 780}, {430, 932}}
	for _, tc := range []struct {
		region       string
		portraitOnly bool
		metro        string
		lat, lon     float64
	}{
		{"east-asia", false, "Dhaka", 23.8103, 90.4125},
		{"africa", true, "Karachi", 24.8607, 67.0011},
	} {
		excluded := mapview.Project(tc.lat, tc.lon)
		for _, win := range windows {
			portrait := win[1] > win[0]
			if tc.portraitOnly && !portrait {
				continue
			}
			panel := ui.Workspace(win[0], win[1], ui.State{View: "map"}).Map
			g := &Game{Map: m, Camera: mapview.New(float64(panel.Dx()), float64(panel.Dy()))}
			c := g.cameraForRegion(tc.region, portrait)
			if d := c.Screen(excluded); !c.Clear(d) {
				t.Errorf("%s %dx%d: %s at %.1f,%.1f is within %v px of the %.0fx%.0f map", tc.region, win[0], win[1], tc.metro, d.X, d.Y, mapview.ExcludeClearance, c.Width, c.Height)
			}
			for _, a := range anchors(tc.region, portrait) {
				q := c.Screen(mapview.Project(a[0], a[1]))
				if q.X < mapview.AnchorClearance-1e-9 || q.X > c.Width-mapview.AnchorClearance+1e-9 ||
					q.Y < mapview.AnchorClearance-1e-9 || q.Y > c.Height-mapview.AnchorClearance+1e-9 {
					t.Errorf("%s %dx%d: anchor %v at %.1f,%.1f left its marker clearance", tc.region, win[0], win[1], a, q.X, q.Y)
				}
			}
		}
	}
}
