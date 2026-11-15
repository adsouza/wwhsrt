package render

import (
	"image"
	"image/color"

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
// at 3737x2029, against 1 GB with the coast cut into runs like this.
const coastRun = 32

// landPixel is an opaque white pixel for drawing plain triangles, cut from the
// middle of a larger image so that sampling never reaches its edge.
var landPixel = func() *ebiten.Image {
	img := ebiten.NewImage(3, 3)
	img.Fill(color.White)
	return img.SubImage(image.Rect(1, 1, 2, 2)).(*ebiten.Image)
}()

// paintLand fills the pieces of land and strokes the coast of the outlines
// they were cut from (see basemap.fill for the pieces).
//
// The fill is a fan of triangles per piece, each flipping the pixels it covers
// between land and clear, so a pixel ends up land when an odd number of
// triangles cover it. That is the even-odd rule: a lake inside land is clear
// and an island inside the lake is land again, whichever way they wind. It is
// not Ebitengine's own fill, which counts the triangles of each facing over a
// pixel in four bits of a stencil. With one land outline a continent is a
// single shape, pixels deep inland sat under 20 to 40 triangles, the count
// wrapped, and the map showed water-coloured streaks from Spain to the Baltic.
// Flipping has no count to overflow. It is not antialiased; the antialiased
// coast drawn over its edge hides that.
//
// The coast is one antialiased path. Ebitengine packs the bounding box of
// every path in an antialiased batch, at twice its width, into stencil images
// that it keeps at their largest size, so one path means one map-sized box.
func paintLand(dst *ebiten.Image, pieces, outlines [][]mapview.Point, pal ui.Palette) {
	if vertices, indices := landFan(pieces, pal.Land); len(indices) > 0 {
		dst.DrawTriangles32(vertices, indices, landPixel, &ebiten.DrawTrianglesOptions{Blend: ebiten.BlendXor})
	}
	var shore vector.Path
	for _, run := range coastRuns(outlines) {
		for j, q := range run {
			if j == 0 {
				shore.MoveTo(float32(q.X), float32(q.Y))
			} else {
				shore.LineTo(float32(q.X), float32(q.Y))
			}
		}
	}
	op := &vector.DrawPathOptions{AntiAlias: true}
	op.ColorScale.ScaleWithColor(pal.Coast)
	vector.StrokePath(dst, &shore, &vector.StrokeOptions{Width: 1}, op)
}

// landFan returns each piece as a fan of triangles from its first point, in
// the colour given.
func landFan(pieces [][]mapview.Point, c color.NRGBA) ([]ebiten.Vertex, []uint32) {
	points := 0
	for _, piece := range pieces {
		points += len(piece)
	}
	vertices := make([]ebiten.Vertex, 0, points)
	indices := make([]uint32, 0, 3*points)
	r, g, b, a := float32(c.R)/255, float32(c.G)/255, float32(c.B)/255, float32(c.A)/255
	for _, piece := range pieces {
		if len(piece) < 3 {
			continue
		}
		first := uint32(len(vertices))
		for _, p := range piece {
			vertices = append(vertices, ebiten.Vertex{
				DstX: float32(p.X), DstY: float32(p.Y), SrcX: 1, SrcY: 1, ColorR: r, ColorG: g, ColorB: b, ColorA: a,
			})
		}
		for i := uint32(1); i+1 < uint32(len(piece)); i++ {
			indices = append(indices, first, first+i, first+i+1)
		}
	}
	return vertices, indices
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
