//go:build !js

package app

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
	"wwhsrt/pkg/mapview"
	"wwhsrt/pkg/render"
	"wwhsrt/pkg/ui"
)

// China, Japan and Korea had regions of their own until East Asia took them over. A preferences
// file that still names one of them opens on East Asia, the default, framed as East Asia.
func TestNativePreferencesDropARemovedRegion(t *testing.T) {
	old := preferencesPath
	preferencesPath = filepath.Join(t.TempDir(), "ui.json")
	t.Cleanup(func() { preferencesPath = old })
	port, f := routeChoiceFrame(t)
	m, e := render.New()
	if e != nil {
		t.Fatal(e)
	}
	for stored, want := range map[string]string{"japan": "", "china": "", "korea": "", "europe": "europe", "": ""} {
		raw, _ := json.Marshal(nativePreferences{State: ui.State{Locale: "en", Region: stored}})
		if err := os.WriteFile(preferencesPath, raw, 0600); err != nil {
			t.Fatal(err)
		}
		g := &Game{Port: port, Frame: f, Map: m, Camera: mapview.New(1280, 850)}
		g.host.init(g)
		if g.State.Region != want {
			t.Fatalf("stored region %q must load as %q, got %q", stored, want, g.State.Region)
		}
		if g.Camera != g.cameraForRegion(want, false) {
			t.Fatalf("stored region %q must open framed as %q", stored, want)
		}
	}
}
