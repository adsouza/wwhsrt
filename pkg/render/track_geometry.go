package render

import "wwhsrt/pkg/mapview"

type railSegment [2]mapview.Point

// A public corridor is a service/capacity identity, not an extra physical stroke.
// Reversed duplicates can cancel in the vector rasterizer's nonzero winding
// fill, leaving pale gaps in shared track. Keep exact source edges once, without
// snapping nearby parallel railways or changing corridor geometry for picking.
func publicTrackSegments(lines [][]mapview.Point) []railSegment {
	seen := map[railSegment]bool{}
	var segments []railSegment
	for _, line := range lines {
		for i := 1; i < len(line); i++ {
			a, b := line[i-1], line[i]
			if a == b {
				continue
			}
			if b.X < a.X || b.X == a.X && b.Y < a.Y {
				a, b = b, a
			}
			edge := railSegment{a, b}
			if !seen[edge] {
				seen[edge] = true
				segments = append(segments, edge)
			}
		}
	}
	return segments
}
