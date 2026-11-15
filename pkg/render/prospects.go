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
		// Carry spacing across short source segments, and clip before generating
		// dots to bound work even when most of an alignment is offscreen.
		total := 0.0
		for i := 1; i < len(prospect.path); i++ {
			a := c.Screen(mapview.Project(prospect.path[i-1].Lat, prospect.path[i-1].Lon))
			b := c.Screen(mapview.Project(prospect.path[i].Lat, prospect.path[i].Lon))
			length := math.Hypot(b.X-a.X, b.Y-a.Y)
			left, right, ok := screenSegment(c, a, b)
			if ok {
				visible := math.Hypot(right.X-left.X, right.Y-left.Y)
				start := math.Mod(12-math.Mod(total+math.Hypot(left.X-a.X, left.Y-a.Y), 12), 12)
				for d := start; d < visible; d += 12 {
					u := d / visible
					x, y := float32(left.X+(right.X-left.X)*u), float32(left.Y+(right.Y-left.Y)*u)
					m.marks.disc(dst, x, y, 3.5, p.Surface)
					m.marks.disc(dst, x, y, 2.5, p.Action)
				}
			}
			total += length
		}
	}
}
