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
	for _, change := range []func(*ui.State){func(s *ui.State) { s.Selected = "beijing" }, func(s *ui.State) { s.SourceStation = "node/1" }, func(s *ui.State) { s.Region = "china" }, func(s *ui.State) { s.Locale = "zh-Hans" }, func(s *ui.State) { s.Dark = true }, func(s *ui.State) { s.HideTerrain = true }, func(s *ui.State) { s.View = "services" }, func(s *ui.State) { s.ServiceCorridor = "route_beijing__shanghai" }, func(s *ui.State) { s.AlignmentID = "valley" }, func(s *ui.State) { s.FromCity = "beijing" }, func(s *ui.State) { s.ToCity = "shanghai" }, func(s *ui.State) { s.Portrait = true }} {
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
