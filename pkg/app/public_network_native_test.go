//go:build !js

package app

import (
	"bytes"
	"github.com/ebitenui/ebitenui/event"
	"github.com/ebitenui/ebitenui/widget"
	"path/filepath"
	"testing"
	"wwhsrt/internal/application"
	"wwhsrt/pkg/gameapi"
	"wwhsrt/pkg/mapview"
	"wwhsrt/pkg/render"
	"wwhsrt/pkg/ui"
)

func TestNativePublicNetworkSearchAndRegionRotation(t *testing.T) {
	old := preferencesPath
	preferencesPath = filepath.Join(t.TempDir(), "ui.json")
	t.Cleanup(func() { preferencesPath = old })
	port, e := application.Open(nil, nil)
	if e != nil {
		t.Fatal(e)
	}
	m, e := render.New()
	if e != nil {
		t.Fatal(e)
	}
	before := port.Snapshot().Checkpoint
	for _, locale := range []string{"en", "zh-Hans"} {
		query := "Beijing Shanghai"
		if locale == "zh-Hans" {
			query = "北京 上海"
		}
		g := &Game{Port: port, Frame: port.Snapshot(), Map: m, width: 390, height: 844, Camera: mapview.New(390, 260), State: ui.State{Locale: locale, View: "services", Region: "china", ServiceCorridor: "route_beijing__shanghai", Frequency: 2, Fare: 16000, ServiceQuery: query, Disclosures: map[string]bool{"route-search": true}}}
		g.host.build(g)
		if g.host.routeSearch == nil || g.host.routeSearchField.Text() != query {
			t.Fatal("no native editable route search")
		}
		if w, _ := g.host.scroll.PreferredSize(); w > g.host.scroll.ViewRect().Dx() {
			t.Fatal("worldwide service header overflows phone")
		}
		g.Camera = g.regionCamera(false)
		g.State.Portrait = false
		g.host.layout(g)
		expected := mapview.RegionCamera(g.Camera.Width, g.Camera.Height, "china", true)
		if g.Camera.Min != expected.Min || g.Camera.Max != expected.Max {
			t.Fatal("rotation lost chosen region")
		}
	}
	if port.Snapshot().Checkpoint != before {
		t.Fatal("search/camera changed economy")
	}
}

func TestNativeExistingPublicConnectionOpensServicesWithoutConstruction(t *testing.T) {
	old := preferencesPath
	preferencesPath = filepath.Join(t.TempDir(), "ui.json")
	t.Cleanup(func() { preferencesPath = old })
	port, e := application.Open(nil, nil)
	if e != nil {
		t.Fatal(e)
	}
	m, e := render.New()
	if e != nil {
		t.Fatal(e)
	}
	f := port.Snapshot()
	before, _ := port.Export()
	for _, co := range ui.ConstructionOpportunities(f) {
		if co.Existing {
			t.Fatal("public line offered for construction")
		}
	}
	for _, pair := range []struct{ from, to, id string }{{"beijing", "tianjin", "public_beijing__tianjin"}, {"chengdu", "chongqing", "public_chengdu__chongqing"}, {"chengdu", "xi_an", "public_chengdu__xi_an"}} {
		for _, lang := range []string{"en", "zh-Hans"} {
			for _, width := range []int{320, 1280} {
				g := &Game{Port: port, Frame: f, Map: m, width: width, height: 844, State: ui.State{Locale: lang, View: "build", FromCity: gameapi.CityID(pair.from), ToCity: gameapi.CityID(pair.to), Disclosures: map[string]bool{"opportunities": true}}}
				g.host.layout(g)
				g.host.build(g)
				b := g.host.actions["plan-existing-service"]
				if b == nil {
					t.Fatal("missing existing-rail service shortcut")
				}
				if g.host.primaryAction != nil {
					t.Fatal("existing rail has a construction commit action")
				}
				if w, _ := g.host.scroll.PreferredSize(); w > g.host.scroll.ViewRect().Dx() {
					t.Fatal("public connection text overflows")
				}
				b.ClickedEvent.Fire(&widget.ButtonClickedEventArgs{Button: b})
				event.ExecuteDeferred()
				if g.State.View != "services" || g.State.ServiceCorridor != pair.id || g.State.ServicePane != "plan" {
					t.Fatal("did not open existing service")
				}
			}
		}
	}
	after, _ := port.Export()
	if !bytes.Equal(before, after) {
		t.Fatal("inspection/shortcut changed campaign")
	}
}
