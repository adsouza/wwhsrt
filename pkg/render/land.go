package render

import (
	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/vector"
	"wwhsrt/pkg/mapview"
	"wwhsrt/pkg/ui"
)

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
	var fill vector.Path
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
	vector.StrokePath(dst, &fill, &vector.StrokeOptions{Width: 1}, op)
}
