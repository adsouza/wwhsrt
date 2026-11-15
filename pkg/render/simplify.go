package render

import "wwhsrt/pkg/mapview"

// screenTolerance is the farthest, in screen pixels, that a simplified outline
// may stray from its source geometry.
//
// Ebitengine's antialiased vector fill emits eight passes of two vertices per
// line segment, and its command queue keeps its largest vertex buffer for the
// life of the process. At the World view most coastline and track vertices
// are far closer together than a pixel. Drawing them all cost about 340 MB of
// queued vertices per scene rebuild, and the queue retained twice that.
const screenTolerance = 0.25

// simplifyScreen removes screen points that lie within tol of the line through
// their kept neighbours (Douglas-Peucker). It keeps the first and last points,
// so a ring whose source closes on itself stays closed.
func simplifyScreen(points []mapview.Point, tol float64) []mapview.Point {
	if len(points) < 3 {
		return points
	}
	keep := make([]bool, len(points))
	keep[0], keep[len(points)-1] = true, true
	tol2 := tol * tol
	stack := [][2]int{{0, len(points) - 1}}
	for len(stack) > 0 {
		span := stack[len(stack)-1]
		stack = stack[:len(stack)-1]
		a, b := points[span[0]], points[span[1]]
		far, best := -1, tol2
		for i := span[0] + 1; i < span[1]; i++ {
			if d := segmentDistance2(points[i], a, b); d > best {
				far, best = i, d
			}
		}
		if far >= 0 {
			keep[far] = true
			stack = append(stack, [2]int{span[0], far}, [2]int{far, span[1]})
		}
	}
	out := make([]mapview.Point, 0, len(points)/4+2)
	for i, k := range keep {
		if k {
			out = append(out, points[i])
		}
	}
	return out
}

func segmentDistance2(p, a, b mapview.Point) float64 {
	dx, dy := b.X-a.X, b.Y-a.Y
	px, py := p.X-a.X, p.Y-a.Y
	if l2 := dx*dx + dy*dy; l2 > 0 {
		u := min(1, max(0, (px*dx+py*dy)/l2))
		px, py = px-u*dx, py-u*dy
	}
	return px*px + py*py
}

// screenRuns projects a world polyline and clips it to the camera, returning
// the visible stretches in screen space. A stretch breaks where the line leaves
// the screen and resumes where it re-enters.
func screenRuns(c mapview.Camera, line []mapview.Point) [][]mapview.Point {
	var runs [][]mapview.Point
	var run []mapview.Point
	prev := c.Screen(line[0])
	for _, w := range line[1:] {
		next := c.Screen(w)
		a, b, ok := screenSegment(c, prev, next)
		prev = next
		if !ok {
			continue
		}
		if len(run) == 0 || run[len(run)-1] != a {
			if len(run) > 1 {
				runs = append(runs, run)
			}
			run = []mapview.Point{a}
		}
		run = append(run, b)
	}
	if len(run) > 1 {
		runs = append(runs, run)
	}
	return runs
}
