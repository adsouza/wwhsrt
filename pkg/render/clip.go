package render

import "wwhsrt/pkg/mapview"

// clipScreen clips a closed outline in screen space to the camera's screen, with a 4 px margin.
//
// Clip before passing paths to the antialias renderer. Its temporary images use
// path bounds, so offscreen world geometry must not allocate world-sized images.
func clipScreen(c mapview.Camera, points []mapview.Point) []mapview.Point {
	for _, edge := range []struct {
		axis    int
		bound   float64
		greater bool
	}{{0, -4, true}, {0, c.Width + 4, false}, {1, -4, true}, {1, c.Height + 4, false}} {
		if len(points) == 0 {
			break
		}
		out := []mapview.Point{}
		value := func(p mapview.Point) float64 {
			if edge.axis == 0 {
				return p.X
			}
			return p.Y
		}
		inside := func(p mapview.Point) bool {
			if edge.greater {
				return value(p) >= edge.bound
			}
			return value(p) <= edge.bound
		}
		prev := points[len(points)-1]
		for _, next := range points {
			a, b := inside(prev), inside(next)
			if a != b {
				u := (edge.bound - value(prev)) / (value(next) - value(prev))
				out = append(out, mapview.Point{X: prev.X + u*(next.X-prev.X), Y: prev.Y + u*(next.Y-prev.Y)})
			}
			if b {
				out = append(out, next)
			}
			prev = next
		}
		points = out
	}
	return points
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
