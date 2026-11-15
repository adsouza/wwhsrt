package render

import (
	"math"
	"testing"

	"wwhsrt/pkg/mapview"
)

func TestSimplifyScreenStaysWithinToleranceAndDropsDenseVertices(t *testing.T) {
	// A wavy coast sampled every 0.05 px: far denser than a screen can show.
	var line []mapview.Point
	for i := 0; i <= 4000; i++ {
		x := float64(i) * 0.05
		line = append(line, mapview.Point{X: x, Y: 20 + 6*math.Sin(x/9) + 0.1*math.Sin(x*3)})
	}
	got := simplifyScreen(line, screenTolerance)
	if got[0] != line[0] || got[len(got)-1] != line[len(line)-1] {
		t.Fatal("simplification moved an end of the line")
	}
	if len(got)*10 > len(line) {
		t.Fatalf("kept %d of %d sub-pixel vertices", len(got), len(line))
	}
	for _, p := range line {
		best := math.Inf(1)
		for i := 1; i < len(got); i++ {
			best = math.Min(best, segmentDistance2(p, got[i-1], got[i]))
		}
		if math.Sqrt(best) > screenTolerance+1e-9 {
			t.Fatalf("source point %v is %.3f px from the drawn line", p, math.Sqrt(best))
		}
	}
	short := line[:2]
	if out := simplifyScreen(short, screenTolerance); len(out) != 2 {
		t.Fatal("a two-point line lost a point")
	}
	speck := []mapview.Point{{X: 5, Y: 5}, {X: 5.1, Y: 5}, {X: 5.1, Y: 5.1}}
	if out := simplifyRing(speck); len(out) != 3 {
		t.Fatal("a ring too small to simplify was dropped instead of drawn as it is")
	}
}

func TestPublicTrackChainsDrawEveryEdgeExactlyOnce(t *testing.T) {
	// A junction, a branch and a separate loop.
	a, b, c, d := mapview.Point{X: 0}, mapview.Point{X: 1}, mapview.Point{X: 2}, mapview.Point{X: 1, Y: 1}
	l1, l2, l3 := mapview.Point{X: 5}, mapview.Point{X: 6}, mapview.Point{X: 5, Y: 1}
	segments := []railSegment{{a, b}, {b, c}, {b, d}, {l1, l2}, {l2, l3}, {l3, l1}}
	checkChains(t, segments, publicTrackChains(segments))

	layer, err := newPublicLayer()
	if err != nil {
		t.Fatal(err)
	}
	checkChains(t, layer.tracks, layer.chains)
	if len(layer.chains)*5 > len(layer.tracks) {
		t.Fatalf("%d chains for %d edges: edges were not joined", len(layer.chains), len(layer.tracks))
	}
}

func checkChains(t *testing.T, segments []railSegment, chains [][]mapview.Point) {
	t.Helper()
	want := map[railSegment]bool{}
	for _, s := range segments {
		want[s] = true
	}
	seen := map[railSegment]bool{}
	for _, chain := range chains {
		for i := 1; i < len(chain); i++ {
			e := railSegment{chain[i-1], chain[i]}
			if !want[e] {
				e = railSegment{chain[i], chain[i-1]}
			}
			if !want[e] || seen[e] {
				t.Fatalf("chain edge %v is not a source edge or is drawn twice", e)
			}
			seen[e] = true
		}
	}
	if len(seen) != len(want) {
		t.Fatalf("chains draw %d of %d edges", len(seen), len(want))
	}
}

func TestScreenRunsBreakWhereALineLeavesTheMap(t *testing.T) {
	c := mapview.Fit(200, 200, []mapview.Point{{X: 0, Y: 0}, {X: 10, Y: 10}})
	inside := func(x, y float64) mapview.Point { return c.World(mapview.Point{X: x, Y: y}) }
	line := []mapview.Point{inside(20, 100), inside(60, 100), inside(-300, 100), inside(-300, 50), inside(40, 50), inside(80, 50)}
	runs := screenRuns(c, line)
	if len(runs) != 2 {
		t.Fatalf("got %d visible stretches, want 2: %v", len(runs), runs)
	}
	if first := runs[0]; first[0].X < 19 || first[0].X > 21 || first[len(first)-1].X > -3 {
		t.Fatal("first stretch did not run from its start to the map edge", first)
	}
	if second := runs[1]; second[0].X > -3 || second[len(second)-1].X < 79 {
		t.Fatal("second stretch did not resume at the map edge", second)
	}
}

func TestCoastRingsHideOnlyOutlinesALaterRingPaintedOver(t *testing.T) {
	square := func(x0, y0, size float64) [][2]float64 {
		return [][2]float64{{x0, y0}, {x0 + size, y0}, {x0 + size, y0 + size}, {x0, y0 + size}, {x0, y0}}
	}
	rings := [][][2]float64{
		square(2, 2, 1),   // inside the later ring 1: painted over before
		square(0, 0, 10),  // a continent
		square(5, 5, 1),   // inside the earlier ring 1: stayed drawn
		square(9, 9, 3),   // overlaps ring 1 without lying inside it
		square(20, 0, 10), // apart from everything
	}
	got := coastRings(rings)
	want := []bool{false, true, true, true, true}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("ring %d: coast %v, want %v", i, got[i], want[i])
		}
	}
}

func TestMarkerPlacementLandsWithinAnEighthOfAPixel(t *testing.T) {
	for v := float32(-3); v < 3; v += 0.01 {
		pixel, phase := markerPlace(v)
		if phase < 0 || phase >= markerPhases {
			t.Fatalf("v=%v: phase %d", v, phase)
		}
		at := float32(pixel) + float32(phase)/markerPhases
		if d := at - v; d > 0.125+1e-5 || d < -0.125-1e-5 {
			t.Fatalf("v=%v drawn at %v", v, at)
		}
	}
}

// The World view at the size the native app gives the map on a 3737x2029
// window. Every vertex here becomes sixteen or more in Ebitengine's
// antialiased fill, and its command queue keeps the largest frame's buffer.
func TestWorldViewGeometryStaysNearScreenResolution(t *testing.T) {
	layer, err := newPublicLayer()
	if err != nil {
		t.Fatal(err)
	}
	c := mapview.RegionCamera(3032, 1863, "world", false)
	source, land := 0, 0
	for _, ring := range layer.land {
		source += len(ring)
		if points := screenRing(c, ring); len(points) >= 3 {
			land += len(simplifyRing(points))
		}
	}
	track := 0
	for _, chain := range layer.chains {
		for _, run := range screenRuns(c, chain) {
			track += len(simplifyScreen(run, screenTolerance))
		}
	}
	t.Logf("land %d of %d vertices; track %d vertices for %d edges", land, source, track, len(layer.tracks))
	if land > 22000 || track > 9000 {
		t.Fatalf("World view draws %d land and %d track vertices; simplify or cull before drawing more", land, track)
	}
}
