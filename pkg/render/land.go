package render

import (
	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/vector"
	"wwhsrt/pkg/mapview"
	"wwhsrt/pkg/ui"
)

// drawLand fills every ring as one path and strokes the coast as one more.
//
// Ebitengine packs the bounding box of every path in an antialiased batch, at
// twice its width, into stencil images that it keeps at their largest size.
// Filling and stroking each of 729 rings in turn made that one batch of 1,458
// boxes, and the stencil grew into a 537 MB GPU texture at 3737x2029. One path
// per pass needs one box the size of the map.
//
// All rings wind the same way, so the nonzero fill of the joined path is the
// union the separate fills painted. The fill stays antialiased: neighbouring
// rings (refined Chile beside coarse Argentina) leave sub-pixel slivers that a
// one-pass fill shows as dotted water.
func drawLand(dst *ebiten.Image, c mapview.Camera, rings [][][2]float64, coast []bool, pal ui.Palette) {
	var fill, shore vector.Path
	for i, ring := range rings {
		points := screenRing(c, ring)
		if len(points) < 3 {
			continue
		}
		points = simplifyRing(points)
		paths := []*vector.Path{&fill}
		if coast[i] {
			paths = append(paths, &shore)
		}
		for _, path := range paths {
			for j, q := range points {
				if j == 0 {
					path.MoveTo(float32(q.X), float32(q.Y))
				} else {
					path.LineTo(float32(q.X), float32(q.Y))
				}
			}
			path.Close()
		}
	}
	op := &vector.DrawPathOptions{AntiAlias: true}
	op.ColorScale.ScaleWithColor(pal.Land)
	vector.FillPath(dst, &fill, nil, op)
	op = &vector.DrawPathOptions{AntiAlias: true}
	op.ColorScale.ScaleWithColor(pal.Coast)
	vector.StrokePath(dst, &shore, &vector.StrokeOptions{Width: 1}, op)
}

// coastRings reports, for each ring, whether its outline is drawn. Rings used
// to be filled and outlined one at a time, so a ring lying wholly inside a
// later one (Lesotho inside South Africa, Mongolia inside Asia) had its outline
// painted over. Joining the passes would bring those lines back; this keeps
// the map as it was. Outlines inside an earlier ring (Bolivia) stay drawn.
func coastRings(rings [][][2]float64) []bool {
	type box struct{ minX, minY, maxX, maxY float64 }
	boxes := make([]box, len(rings))
	for i, ring := range rings {
		b := box{1e9, 1e9, -1e9, -1e9}
		for _, p := range ring {
			b = box{min(b.minX, p[0]), min(b.minY, p[1]), max(b.maxX, p[0]), max(b.maxY, p[1])}
		}
		boxes[i] = b
	}
	coast := make([]bool, len(rings))
	for i, ring := range rings {
		coast[i] = true
		a := boxes[i]
		for j := i + 1; j < len(rings); j++ {
			outer, b := rings[j], boxes[j]
			if a.minX < b.minX || a.minY < b.minY || a.maxX > b.maxX || a.maxY > b.maxY {
				continue
			}
			inside := len(ring) > 0
			for _, p := range ring {
				if !ringContains(outer, p) {
					inside = false
					break
				}
			}
			if inside {
				coast[i] = false
				break
			}
		}
	}
	return coast
}

// ringContains reports whether p is strictly inside ring (even-odd rule).
func ringContains(ring [][2]float64, p [2]float64) bool {
	in := false
	for i, j := 0, len(ring)-1; i < len(ring); j, i = i, i+1 {
		a, b := ring[i], ring[j]
		if (a[1] > p[1]) != (b[1] > p[1]) && p[0] < (b[0]-a[0])*(p[1]-a[1])/(b[1]-a[1])+a[0] {
			in = !in
		}
	}
	return in
}
