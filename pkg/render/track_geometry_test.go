package render

import (
	"encoding/json"
	"testing"

	"wwhsrt/assets"
	"wwhsrt/pkg/gameapi"
	"wwhsrt/pkg/mapview"
)

func TestPublicTracksShareOneStrokeAndRetainNearbyParallelTracks(t *testing.T) {
	a, b, c := mapview.Point{X: 1, Y: 2}, mapview.Point{X: 3, Y: 4}, mapview.Point{X: 5, Y: 6}
	parallelA, parallelB := a, b
	parallelA.Y += 0.000001
	parallelB.Y += 0.000001
	lines := [][]mapview.Point{{a, b, c}, {c, b, a}, {a, b}, {b, a}, {a, a}, {parallelA, parallelB}, nil}
	got := publicTrackSegments(lines)
	want := map[railSegment]bool{{a, b}: true, {b, c}: true, {parallelA, parallelB}: true}
	if len(got) != len(want) {
		t.Fatal("shared strokes retained or nearby physical tracks merged", got)
	}
	for _, edge := range got {
		if !want[edge] {
			t.Fatal("unexpected or repeated physical stroke", edge)
		}
		delete(want, edge)
	}
	if len(publicTrackSegments(nil)) != 0 {
		t.Fatal("empty network gained track")
	}
}

func TestShijiazhuangJinanSharedTrackDrawnOnceAndAllRoutesRemainPickable(t *testing.T) {
	layer, err := newPublicLayer()
	if err != nil {
		t.Fatal(err)
	}
	// Independently count every input edge in both directions. Rendering must
	// preserve their exact union, without altering the source corridor paths.
	input := map[railSegment]int{}
	for _, line := range layer.Lines {
		for i := 1; i < len(line); i++ {
			a, b := line[i-1], line[i]
			if a == b {
				continue
			}
			forward, reverse := railSegment{a, b}, railSegment{b, a}
			if _, ok := input[reverse]; ok {
				input[reverse]++
			} else {
				input[forward]++
			}
		}
	}
	if len(layer.tracks) != len(input) {
		t.Fatal("public edge union changed")
	}
	for _, edge := range layer.tracks {
		_, forward := input[edge]
		_, reverse := input[railSegment{edge[1], edge[0]}]
		if !forward && !reverse {
			t.Fatal("missing, repeated or invented track", edge)
		}
		delete(input, edge)
		delete(input, railSegment{edge[1], edge[0]})
	}
	if len(input) != 0 {
		t.Fatal("original physical track disappeared")
	}

	a := mapview.Project(37.265741, 116.5401442)
	b := mapview.Project(37.2907421, 116.5173897)
	n := 0
	for _, edge := range layer.tracks {
		if edge == (railSegment{a, b}) || edge == (railSegment{b, a}) {
			n++
		}
	}
	if n != 1 {
		t.Fatal("regression: shared Jinan approach needs exactly one stroke", n)
	}
	var raw struct{ Corridors []gameapi.Corridor }
	if err := json.Unmarshal(assets.PublicNetwork, &raw); err != nil {
		t.Fatal(err)
	}
	camera := mapview.Fit(1200, 800, []mapview.Point{mapview.Project(36.7, 116.4), mapview.Project(37.5, 117.3)})
	x, y := camera.Screen(a), camera.Screen(b)
	hits := PickServiceSections(
		camera,
		gameapi.Frame{Corridors: raw.Corridors},
		mapview.Point{X: (x.X + y.X) / 2, Y: (x.Y + y.Y) / 2},
	)
	seen := map[string]bool{}
	for _, id := range hits {
		seen[id] = true
	}
	for _, id := range []string{
		"public_jinan__shijiazhuang",
		"public_jinan__tianjin",
		"public_jinan__shangqiu",
		"public_jinan__zhengzhou",
		"public_beijing__shangqiu",
		"public_shijiazhuang__tianjin",
	} {
		if !seen[id] {
			t.Fatal("deduplicating drawn track hid a selectable route", id, hits)
		}
	}
}
