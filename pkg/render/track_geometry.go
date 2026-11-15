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

// publicTrackChains joins the deduplicated edges into polylines that end at
// junctions and line ends, so a camera can simplify them as lines rather than
// stroking every short edge as its own ribbon. Every edge appears in exactly
// one chain. Stroke ribbons have the same winding whichever way a chain runs,
// so chains that meet or cross still fill as a union.
func publicTrackChains(segments []railSegment) [][]mapview.Point {
	at := map[mapview.Point][]int{}
	for i, s := range segments {
		at[s[0]] = append(at[s[0]], i)
		at[s[1]] = append(at[s[1]], i)
	}
	used := make([]bool, len(segments))
	other := func(i int, p mapview.Point) mapview.Point {
		if segments[i][0] == p {
			return segments[i][1]
		}
		return segments[i][0]
	}
	walk := func(start mapview.Point, first int) []mapview.Point {
		chain := []mapview.Point{start}
		p, i := start, first
		for {
			used[i] = true
			p = other(i, p)
			chain = append(chain, p)
			if len(at[p]) != 2 {
				return chain
			}
			next := at[p][0]
			if next == i {
				next = at[p][1]
			}
			if used[next] {
				return chain
			}
			i = next
		}
	}
	var chains [][]mapview.Point
	// Open chains start at ends and junctions; what remains are closed loops.
	for _, pass := range []bool{true, false} {
		for i, s := range segments {
			if used[i] {
				continue
			}
			for _, p := range s {
				if !used[i] && (len(at[p]) != 2) == pass {
					chains = append(chains, walk(p, i))
				}
			}
		}
	}
	return chains
}
