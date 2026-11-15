package render

import "wwhsrt/pkg/mapview"

// clipScreen clips a closed outline in screen space to the camera's screen, with a 4 px margin.
//
// Clip before passing paths to the antialias renderer. Its temporary images use
// path bounds, so offscreen world geometry must not allocate world-sized images.
func clipScreen(c mapview.Camera, points []mapview.Point) []mapview.Point {
	points = clipEdge(points, 0, -4, true)
	points = clipEdge(points, 0, c.Width+4, false)
	points = clipEdge(points, 1, -4, true)
	return clipEdge(points, 1, c.Height+4, false)
}

// clipEdge keeps the part of a closed outline on one side of a vertical (axis 0)
// or horizontal (axis 1) line: at or beyond bound when greater, at or before it
// otherwise. Points made where the outline crosses the line lie exactly on it,
// so the two halves of an outline cut by the same line share them.
func clipEdge(points []mapview.Point, axis int, bound float64, greater bool) []mapview.Point {
	if len(points) == 0 {
		return points
	}
	value := func(p mapview.Point) float64 {
		if axis == 0 {
			return p.X
		}
		return p.Y
	}
	inside := func(p mapview.Point) bool {
		if greater {
			return value(p) >= bound
		}
		return value(p) <= bound
	}
	out := make([]mapview.Point, 0, len(points)+4)
	prev := points[len(points)-1]
	for _, next := range points {
		a, b := inside(prev), inside(next)
		if a != b {
			u := (bound - value(prev)) / (value(next) - value(prev))
			cut := mapview.Point{X: prev.X + u*(next.X-prev.X), Y: prev.Y + u*(next.Y-prev.Y)}
			if axis == 0 {
				cut.X = bound
			} else {
				cut.Y = bound
			}
			out = append(out, cut)
		}
		if b {
			out = append(out, next)
		}
		prev = next
	}
	return out
}

// cutCells appends to out the pieces of a closed outline, none wider or taller
// than size, halving it until they fit. The pieces cover exactly what the
// outline did and wind the same way.
func cutCells(points []mapview.Point, size float64, out [][]mapview.Point) [][]mapview.Point {
	if len(points) < 3 {
		return out
	}
	lo, hi := points[0], points[0]
	for _, p := range points {
		lo = mapview.Point{X: min(lo.X, p.X), Y: min(lo.Y, p.Y)}
		hi = mapview.Point{X: max(hi.X, p.X), Y: max(hi.Y, p.Y)}
	}
	for axis, span := range [2][2]float64{{lo.X, hi.X}, {lo.Y, hi.Y}} {
		if span[1]-span[0] > size {
			middle := (span[0] + span[1]) / 2
			out = cutCells(clipEdge(points, axis, middle, false), size, out)
			return cutCells(clipEdge(points, axis, middle, true), size, out)
		}
	}
	return append(out, points)
}
func screenSegment(c mapview.Camera, a, b mapview.Point) (mapview.Point, mapview.Point, bool) {
	// Liang-Barsky clipping retains segments even when both vertices are outside.
	dx, dy := b.X-a.X, b.Y-a.Y
	lo, hi := 0.0, 1.0
	for _, test := range [][2]float64{{-dx, a.X + 4}, {dx, c.Width + 4 - a.X}, {-dy, a.Y + 4}, {dy, c.Height + 4 - a.Y}} {
		p, q := test[0], test[1]
		if p == 0 {
			if q < 0 {
				return a, b, false
			}
			continue
		}
		u := q / p
		if p < 0 {
			lo = max(lo, u)
		} else {
			hi = min(hi, u)
		}
		if lo > hi {
			return a, b, false
		}
	}
	return mapview.Point{X: a.X + lo*dx, Y: a.Y + lo*dy}, mapview.Point{X: a.X + hi*dx, Y: a.Y + hi*dy}, true
}
