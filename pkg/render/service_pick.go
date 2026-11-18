package render

import (
	"math"
	"sort"

	"wwhsrt/pkg/gameapi"
	"wwhsrt/pkg/mapview"
	"wwhsrt/pkg/ui"
)

func pathDistance(c mapview.Camera, path []gameapi.GeoPoint, at mapview.Point) float64 {
	best := math.Inf(1)
	for i := 1; i < len(path); i++ {
		a := c.Screen(mapview.Project(path[i-1].Lat, path[i-1].Lon))
		b := c.Screen(mapview.Project(path[i].Lat, path[i].Lon))
		dx, dy := b.X-a.X, b.Y-a.Y
		u := 0.0
		if d := dx*dx + dy*dy; d > 0 {
			u = math.Max(0, math.Min(1, ((at.X-a.X)*dx+(at.Y-a.Y)*dy)/d))
		}
		best = math.Min(best, math.Hypot(at.X-a.X-u*dx, at.Y-a.Y-u*dy))
	}
	return best
}

// Match the actual displayed, open alignment, rather than endpoint chords.
// Nearby overlapping sections are retained for named route disambiguation.
func PickServiceSections(c mapview.Camera, f gameapi.Frame, at mapview.Point) []string {
	if at.X < 0 || at.Y < 0 || at.X > c.Width || at.Y > c.Height {
		return nil
	}
	type hit struct {
		id       string
		distance float64
	}
	var hits []hit
	running := ui.OperatingSections(f)
	for _, co := range f.Corridors {
		if (co.Retired && !running[co.ID]) || (!running[co.ID] && !ui.Open(f, co.ID)) || len(co.Alignments) == 0 {
			continue
		}
		a := co.Alignments[0]
		for _, p := range f.Projects {
			if p.CorridorID == co.ProjectID() {
				for _, candidate := range co.Alignments {
					if candidate.ID == p.AlignmentID {
						a = candidate
					}
				}
			}
		}
		if d := pathDistance(c, a.Path, at); d <= 18 {
			hits = append(hits, hit{co.ID, d})
		}
	}
	sort.SliceStable(hits, func(i, j int) bool { return hits[i].distance < hits[j].distance })
	var ids []string
	for _, h := range hits {
		if h.distance <= hits[0].distance+4 {
			ids = append(ids, h.id)
		}
	}
	return ids
}
