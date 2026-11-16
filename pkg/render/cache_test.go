package render

import (
	"testing"
	"wwhsrt/pkg/gameapi"
	"wwhsrt/pkg/mapview"
	"wwhsrt/pkg/ui"
)

func TestSceneCacheTracksVisibleStateAndExcludesReplayClock(t *testing.T) {
	c := mapview.New(800, 600)
	f := gameapi.Frame{Checkpoint: "accepted-world"}
	s := ui.State{Locale: "en", View: "map"}
	base := mapSceneKey(c, f, s)
	for _, change := range []func(*ui.State){
		func(s *ui.State) { s.Selected = "beijing" },
		func(s *ui.State) { s.SourceStation = "node/1" },
		func(s *ui.State) { s.Region = "europe" },
		func(s *ui.State) { s.Locale = "zh-Hans" },
		func(s *ui.State) { s.Dark = true },
		func(s *ui.State) { s.HideTerrain = true },
		func(s *ui.State) { s.View = "services" },
		func(s *ui.State) { s.ServiceCorridor = "route_beijing__shanghai" },
		func(s *ui.State) { s.AlignmentID = "valley" },
		func(s *ui.State) { s.FromCity = "beijing" },
		func(s *ui.State) { s.ToCity = "shanghai" },
		func(s *ui.State) { s.Portrait = true },
	} {
		next := s
		change(&next)
		if mapSceneKey(c, f, next) == base {
			t.Fatal("visible change left stale scene")
		}
	}
	s.Motion = true
	s.Watching = true
	s.ReplayMinute = 123
	s.SelectedTrain = "train/1"
	if mapSceneKey(c, f, s) != base {
		t.Fatal("animation invalidates static scene")
	}
	f.Checkpoint = "new-month"
	if mapSceneKey(c, f, s) == base {
		t.Fatal("new company state kept old geometry")
	}
}

// Ebitengine rounds an antialiased path's box up to 16 px and caps its stencil
// buffer by the destination's whole image. A layer's image must leave room for
// that rounding, or a path spanning the map reallocates the buffer every redraw
// and loses its last few columns.
func TestLayerLeavesRoomForStencilRounding(t *testing.T) {
	var l layer
	for _, size := range [][2]int{{3032, 1863}, {3040, 1872}, {1, 1}, {801, 4100}} {
		img := l.reset(size[0], size[1])
		if b := img.Bounds(); b.Dx() != size[0] || b.Dy() != size[1] {
			t.Fatalf("asked for %v, drew into %v", size, b)
		}
		if b := l.backing.Bounds(); b.Dx() < roundUp16(size[0]) || b.Dy() < roundUp16(size[1]) {
			t.Fatalf("%v layer is backed by %v: no room for 16 px rounding", size, b)
		}
		if l.view != img {
			t.Fatal("the layer composites a different image from the one it drew into")
		}
	}
	backing := l.backing
	if l.reset(801, 4100); l.backing != backing {
		t.Fatal("a redraw at the same size allocated a new image")
	}
}
