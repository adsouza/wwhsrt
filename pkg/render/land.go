package render

import (
	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/vector"
	"wwhsrt/pkg/mapview"
	"wwhsrt/pkg/ui"
)

// coastRun is the most segments of coast stroked as one sub-path.
//
// Ebitengine fills a sub-path as a fan of triangles from its first point, and
// a stroke is filled the same way. A continent's outline stroked as one closed
// sub-path made thousands of triangles that each crossed the map, and Apple's
// tile-based GPUs list a triangle in every tile it covers: 5 GB of GPU memory
// at 3737x2029, against 1 GB with the coast cut into runs like this. The land
// fill has the same fan, but one triangle per coast segment, not the several a
// stroked segment needs, and did not show the cost.
const coastRun = 32

// paintLand fills every outline as one path and strokes the coast as one more.
//
// Ebitengine packs the bounding box of every path in an antialiased batch, at
// twice its width, into stencil images that it keeps at their largest size.
// Filling and stroking each of 729 outlines in turn made that one batch of
// 1,458 boxes, and the stencil grew into a 537 MB GPU texture at 3737x2029.
// One path per pass needs one box the size of the map.
//
// The fill is nonzero. Land and the islands inside lakes wind one way and
// lakes the other (see basemap), so the joined path fills land and leaves
// lakes clear without a pass of their own.
func paintLand(dst *ebiten.Image, outlines [][]mapview.Point, pal ui.Palette) {
	var fill, shore vector.Path
	for _, points := range outlines {
		for j, q := range points {
			if j == 0 {
				fill.MoveTo(float32(q.X), float32(q.Y))
			} else {
				fill.LineTo(float32(q.X), float32(q.Y))
			}
		}
		fill.Close()
	}
	for _, run := range coastRuns(outlines) {
		for j, q := range run {
			if j == 0 {
				shore.MoveTo(float32(q.X), float32(q.Y))
			} else {
				shore.LineTo(float32(q.X), float32(q.Y))
			}
		}
	}
	// The fill names source-over and the coast leaves Blend at its zero
	// value, which Ebitengine also draws as source-over. FillPath compares the
	// two as different options and flushes the fill before batching the
	// coast, so each batch holds one map-sized box. Batched together, the two
	// boxes stack into a stencil twice as tall: an 8192x8192 GPU page (268 MB)
	// at 3737x2029 instead of 8192x4096.
	op := &vector.DrawPathOptions{AntiAlias: true, Blend: ebiten.BlendSourceOver}
	op.ColorScale.ScaleWithColor(pal.Land)
	vector.FillPath(dst, &fill, nil, op)
	op = &vector.DrawPathOptions{AntiAlias: true}
	op.ColorScale.ScaleWithColor(pal.Coast)
	vector.StrokePath(dst, &shore, &vector.StrokeOptions{Width: 1}, op)
}

// coastRuns cuts each closed outline into open runs of at most coastRun
// segments. Each run starts on the last point of the one before, and the last
// run of an outline returns to its first point.
func coastRuns(outlines [][]mapview.Point) [][]mapview.Point {
	runs := [][]mapview.Point{}
	for _, points := range outlines {
		if len(points) < 2 {
			continue
		}
		if points[0] != points[len(points)-1] { // clipping can leave an outline without its closing point
			points = append(points[:len(points):len(points)], points[0])
		}
		for start := 0; start < len(points)-1; start += coastRun {
			runs = append(runs, points[start:min(start+coastRun, len(points)-1)+1])
		}
	}
	return runs
}
