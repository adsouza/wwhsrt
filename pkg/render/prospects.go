package render

import (
	"math"

	"github.com/hajimehoshi/ebiten/v2"
	"wwhsrt/pkg/gameapi"
	"wwhsrt/pkg/mapview"
	"wwhsrt/pkg/ui"
)

type prospectPath struct {
	id       string
	path     []gameapi.GeoPoint
	selected bool
}

// One path per available pair, shared by drawing and picking. Alternative
// alignments are disclosed only after selecting a prospect.
func prospectPaths(f gameapi.Frame, s ui.State) []prospectPath {
	if s.View != "map" && s.View != "build" {
		return nil
	}
	var out []prospectPath
	for _, co := range ui.ConstructionOpportunities(f) {
		if len(co.Alignments) == 0 {
			continue
		}
		a := co.Alignments[0]
		selected := s.View == "build" &&
			(co.FromCity == s.FromCity && co.ToCity == s.ToCity || co.FromCity == s.ToCity && co.ToCity == s.FromCity)
		if selected {
			for _, candidate := range co.Alignments {
				if candidate.ID == s.AlignmentID {
					a = candidate
				}
			}
		}
		if len(a.Path) >= 2 {
			out = append(out, prospectPath{co.ID, a.Path, selected})
		}
	}
	return out
}

// Pick the continuous alignment, including the gaps between visible dots.
// Distances are screen pixels so touch/click targets survive zoom changes.
func PickProspect(c mapview.Camera, f gameapi.Frame, s ui.State, at mapview.Point) string {
	if at.X < 0 || at.Y < 0 || at.X > c.Width || at.Y > c.Height {
		return ""
	}
	best, id := 18.0, ""
	for _, prospect := range prospectPaths(f, s) {
		if d := pathDistance(c, prospect.path, at); d < best {
			best, id = d, prospect.id
		}
	}
	return id
}

func (m *Map) drawProspects(dst *ebiten.Image, c mapview.Camera, f gameapi.Frame, s ui.State) {
	p := ui.PaletteFor(s)
	for _, prospect := range prospectPaths(f, s) {
		if prospect.selected {
			m.route(dst, c, prospect.path, p.Draft, p.Outline, false)
			continue
		}
		for _, d := range prospectDots(c, prospect.path) {
			m.marks.disc(dst, float32(d.X), float32(d.Y), 3.5, p.Surface)
			m.marks.disc(dst, float32(d.X), float32(d.Y), 2.5, p.Action)
		}
	}
}

// Station marks are drawn over prospects. A dot is wholly clear of the widest
// mark, the 10 px construction square, when its centre is at least the
// square's half-diagonal plus the dot's 3.5 px outer radius from the station.
var prospectEndClearance = math.Sqrt(50) + 3.5

// prospectDots places a prospect's dots in screen pixels. They are spread
// evenly, at most 12 px apart and at least 8 px apart where there is room, over
// the stretch of line clear of both station marks: from where the path last
// leaves the start's clearance circle to where it first enters the end's. A
// line with no such stretch gets one dot where it crosses the perpendicular
// bisector of its stations, at least half their distance from each mark, so a
// short corridor at a region-wide zoom still shows. Spacing is carried across
// short source segments, and segments are clipped before dots are generated to
// bound work when most of an alignment is offscreen.
func prospectDots(c mapview.Camera, path []gameapi.GeoPoint) []mapview.Point {
	if len(path) < 2 {
		return nil
	}
	screen := make([]mapview.Point, len(path))
	for i, g := range path {
		screen[i] = c.Screen(mapview.Project(g.Lat, g.Lon))
	}
	reversed := make([]mapview.Point, len(screen))
	for i, q := range screen {
		reversed[len(screen)-1-i] = q
	}
	total := pathLength(screen)
	from := lastInside(screen, prospectEndClearance)
	to := total - lastInside(reversed, prospectEndClearance)
	first, spacing, count := bisectorCrossing(screen), 0.0, 1
	if room := to - from; room > 0 {
		if count = int(math.Ceil(room/12)) + 1; room/float64(count-1) < 8 {
			count--
		}
		if count == 1 {
			first = (from + to) / 2
		} else {
			first, spacing = from, room/float64(count-1)
		}
	}
	var out []mapview.Point
	at := 0.0
	for i := 1; i < len(screen); i++ {
		a, b := screen[i-1], screen[i]
		length := math.Hypot(b.X-a.X, b.Y-a.Y)
		if left, right, ok := screenSegment(c, a, b); ok && length > 0 {
			lo := at + math.Hypot(left.X-a.X, left.Y-a.Y)
			hi := at + math.Hypot(right.X-a.X, right.Y-a.Y)
			k := 0
			if spacing > 0 {
				k = max(0, int(math.Ceil((lo-first)/spacing)))
			}
			for ; k < count; k++ {
				d := first + float64(k)*spacing
				// A dot on a shared vertex belongs to the segment it starts.
				if d > hi || d == hi && i < len(screen)-1 {
					break
				}
				if d >= lo {
					u := (d - at) / length
					out = append(out, mapview.Point{X: a.X + (b.X-a.X)*u, Y: a.Y + (b.Y-a.Y)*u})
				}
			}
		}
		at += length
	}
	return out
}

func pathLength(path []mapview.Point) float64 {
	total := 0.0
	for i := 1; i < len(path); i++ {
		total += math.Hypot(path[i].X-path[i-1].X, path[i].Y-path[i-1].Y)
	}
	return total
}

// lastInside is how far along the path its last point within r of the path's
// start lies: beyond it, the line never comes back under the start's mark.
func lastInside(path []mapview.Point, r float64) float64 {
	o, last, at := path[0], 0.0, 0.0
	for i := 1; i < len(path); i++ {
		a, b := path[i-1], path[i]
		dx, dy := b.X-a.X, b.Y-a.Y
		length := math.Hypot(dx, dy)
		// |a + t(b-a) - o|² = r² gives the part of the segment inside the circle.
		qa, qb, qc := dx*dx+dy*dy, 2*(dx*(a.X-o.X)+dy*(a.Y-o.Y)), (a.X-o.X)*(a.X-o.X)+(a.Y-o.Y)*(a.Y-o.Y)-r*r
		if disc := qb*qb - 4*qa*qc; length > 0 && disc >= 0 {
			t1, t2 := (-qb-math.Sqrt(disc))/(2*qa), (-qb+math.Sqrt(disc))/(2*qa)
			if t1 <= 1 && t2 >= 0 {
				last = at + min(t2, 1)*length
			}
		}
		at += length
	}
	return last
}

// bisectorCrossing is how far along the path it first comes equally close to
// both ends. Such a point is at least half the ends' distance from each.
func bisectorCrossing(path []mapview.Point) float64 {
	o, e := path[0], path[len(path)-1]
	// The difference of squared distances to the ends is linear along a segment.
	side := func(q mapview.Point) float64 {
		return (q.X-o.X)*(q.X-o.X) + (q.Y-o.Y)*(q.Y-o.Y) - (q.X-e.X)*(q.X-e.X) - (q.Y-e.Y)*(q.Y-e.Y)
	}
	at := 0.0
	for i := 1; i < len(path); i++ {
		a, b := path[i-1], path[i]
		length := math.Hypot(b.X-a.X, b.Y-a.Y)
		if sa, sb := side(a), side(b); sa <= 0 && sb >= 0 && sb > sa {
			return at - sa/(sb-sa)*length
		}
		at += length
	}
	return at / 2
}
