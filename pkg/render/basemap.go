package render

import (
	"encoding/json"
	"image/color"
	"math"

	"github.com/hajimehoshi/ebiten/v2"
	"wwhsrt/assets"
	"wwhsrt/pkg/mapview"
	"wwhsrt/pkg/ui"
)

// landTolerance is the farthest, in screen pixels, that a drawn outline may
// stray from the basemap.
//
// The basemap follows the coast to about 300 m near lines, far finer than a
// wide view can show. At the World view on a 3737x2029 window a quarter-pixel
// tolerance drew 45,800 land vertices, more than twice what the coarser
// basemap before it did, and every vertex of the coast becomes sixteen or more
// in Ebitengine's antialiased stroke (see screenTolerance). One pixel draws
// about 22,000. It is no coarser than that earlier basemap was at the World
// view, and at close zooms it is finer than the data.
const landTolerance = 1.0

// landSpeck is the length in screen pixels below which an outline with no
// shape at this zoom is not drawn. Longer ones are kept as slivers (see
// chosen). Keeping every such outline from one pixel up added 1,980 vertices
// at the World view and took it past its limit of 22,000; from three pixels up
// adds 796, for islands and lakes that would otherwise be a dot of coast line.
const landSpeck = 3.0

// basemap is the one worldwide land outline: land pieces, the lakes cut out of
// them, and the islands inside those lakes. tools/prepare-basemap.py writes it,
// detailed near lines and coarse elsewhere. There are no lines between
// countries.
//
// The land is filled by the even-odd rule (see paintLand), so a lake inside
// land is left clear for the scene's water to show through, and an island
// inside the lake is land again. That needs no two land outlines to overlap,
// which the tool guarantees. The file also winds land and lake islands
// counter-clockwise and lakes clockwise, for readers that fill by winding.
type basemap struct {
	rings                [][][2]float64 // land, then lakes, then lake islands
	land, lakes, islands int            // how many of each
	outlines             []outline      // one per ring
	layer                layer
	camera               mapview.Camera
	color                color.NRGBA
}

// outline is a ring projected once, with what is needed to draw it at any zoom.
type outline struct {
	points   []mapview.Point
	rank     []float64 // see rankOutline
	min, max mapview.Point
	// corners are the two highest-ranked points other than the ends, in order along the outline:
	// with the ends, the least that still draws the outline as a shape.
	corners [2]int
}

func newBasemap() (*basemap, error) {
	data, err := assets.Terrain.ReadFile("terrain/world-land.json")
	if err != nil {
		return nil, err
	}
	var file struct {
		Land    [][][2]float64 `json:"land"`
		Lakes   [][][2]float64 `json:"lakes"`
		Islands [][][2]float64 `json:"lake_islands"`
	}
	if err = json.Unmarshal(data, &file); err != nil {
		return nil, err
	}
	b := &basemap{land: len(file.Land), lakes: len(file.Lakes), islands: len(file.Islands)}
	b.rings = append(append(append(b.rings, file.Land...), file.Lakes...), file.Islands...)
	b.outlines = make([]outline, len(b.rings))
	for i, ring := range b.rings {
		o := outline{
			points: make([]mapview.Point, len(ring)),
			min:    mapview.Point{X: math.Inf(1), Y: math.Inf(1)},
			max:    mapview.Point{X: math.Inf(-1), Y: math.Inf(-1)},
		}
		for j, p := range ring {
			q := mapview.Project(p[1], p[0])
			o.points[j] = q
			o.min = mapview.Point{X: min(o.min.X, q.X), Y: min(o.min.Y, q.Y)}
			o.max = mapview.Point{X: max(o.max.X, q.X), Y: max(o.max.Y, q.Y)}
		}
		o.rank = rankOutline(o.points)
		for j := 1; j < len(o.points)-1; j++ {
			switch {
			case o.corners[0] == 0 || o.rank[j] > o.rank[o.corners[0]]:
				o.corners = [2]int{j, o.corners[0]}
			case o.corners[1] == 0 || o.rank[j] > o.rank[o.corners[1]]:
				o.corners[1] = j
			}
		}
		if o.corners[0] > o.corners[1] {
			o.corners[0], o.corners[1] = o.corners[1], o.corners[0]
		}
		b.outlines[i] = o
	}
	return b, nil
}

// rankOutline gives each point of a closed outline the tolerance below which
// it must be drawn: simplifying the outline at any tolerance keeps exactly the
// points whose rank exceeds it (Douglas-Peucker, as simplifyScreen does it).
// The first and last points always stay.
//
// Ranking once, in map space, means which points are drawn depends on the zoom
// and not on where the camera is. Simplifying the clipped outline on every
// frame picks different points as the map is dragged, which is invisible at a
// quarter of a pixel but would make the coast shimmer at landTolerance.
func rankOutline(points []mapview.Point) []float64 {
	rank := make([]float64, len(points))
	if len(points) == 0 {
		return rank
	}
	last := len(points) - 1
	rank[0], rank[last] = math.Inf(1), math.Inf(1)
	type span struct {
		a, b  int
		limit float64
	}
	stack := []span{{0, last, math.Inf(1)}}
	for len(stack) > 0 {
		s := stack[len(stack)-1]
		stack = stack[:len(stack)-1]
		far, best := -1, 0.0
		for i := s.a + 1; i < s.b; i++ {
			if d := segmentDistance2(points[i], points[s.a], points[s.b]); d > best {
				far, best = i, d
			}
		}
		if far < 0 {
			continue
		}
		// A point is only reached once its span's ends are kept, so it cannot outrank them.
		rank[far] = min(math.Sqrt(best), s.limit)
		stack = append(stack, span{s.a, far, rank[far]}, span{far, s.b, rank[far]})
	}
	return rank
}

// chosen returns the points of outline i, in map space, that are drawn at the camera's zoom.
func (b *basemap) chosen(i int, c mapview.Camera) []mapview.Point {
	scale := c.Screen(mapview.Point{X: 1}).X - c.Screen(mapview.Point{}).X // pixels per map unit
	if !(scale > 0) || math.IsInf(scale, 0) {
		return nil
	}
	o := &b.outlines[i]
	tolerance := landTolerance / scale
	out := make([]mapview.Point, 0, 64)
	for j, p := range o.points {
		if o.rank[j] > tolerance {
			out = append(out, p)
		}
	}
	if len(out) >= 4 {
		return out
	}
	// Every point of a long thin outline lies within the tolerance of the line through its ends, so
	// rank alone would leave an island many pixels long with nothing to draw. Keep it as a sliver.
	if max(o.max.X-o.min.X, o.max.Y-o.min.Y) < landSpeck/scale || o.corners[0] == 0 {
		return nil
	}
	last := len(o.points) - 1
	return []mapview.Point{o.points[0], o.points[o.corners[0]], o.points[o.corners[1]], o.points[last]}
}

// visible returns the outlines to draw, in order and in screen space, each
// clipped to the screen. An outline wholly off screen is skipped without
// touching its points, and one shorter than landSpeck is not drawn.
func (b *basemap) visible(c mapview.Camera) [][]mapview.Point {
	if c.Width <= 0 || c.Height <= 0 {
		return nil
	}
	out := [][]mapview.Point{}
	for i := range b.outlines {
		o := &b.outlines[i]
		// Screen is a scale and a shift (with y flipped), so the bounds map to the screen box.
		lo, hi := c.Screen(o.min), c.Screen(o.max)
		if hi.X < -4 || lo.X > c.Width+4 || lo.Y < -4 || hi.Y > c.Height+4 {
			continue
		}
		points := b.chosen(i, c)
		if len(points) < 4 { // the closing point repeats the first, so fewer than three corners
			continue
		}
		for j, p := range points {
			points[j] = c.Screen(p)
		}
		if points = clipScreen(c, points); len(points) >= 3 {
			out = append(out, points)
		}
	}
	return out
}

// landCell is the most pixels across a piece of land filled as one fan.
//
// A fan's triangles all start at one point of the piece. For a whole
// continent each would cross the map, and Apple's tile-based GPUs list a
// triangle in every tile it covers (see coastRun for what that cost the
// coast). Cutting the outline first keeps every triangle inside one cell.
const landCell = 256

// cells cuts screen outlines into pieces no larger than landCell for filling.
// Two pieces meet on exactly the same points along a cut, and each pixel is
// flipped by whichever piece holds its centre, so the cuts leave no seam. The
// coast is stroked from the outlines, not from the pieces.
func cells(outlines [][]mapview.Point) [][]mapview.Point {
	pieces := make([][]mapview.Point, 0, len(outlines))
	for _, points := range outlines {
		pieces = cutCells(points, landCell, pieces)
	}
	return pieces
}

// fill returns the pieces of land to fill at the camera, in screen space.
func (b *basemap) fill(c mapview.Camera) [][]mapview.Point { return cells(b.visible(c)) }

// draw paints the land, redrawing its layer only when the camera or palette changed.
func (b *basemap) draw(dst *ebiten.Image, c mapview.Camera, pal ui.Palette) {
	if b.layer.view == nil || b.camera != c || b.color != pal.Land {
		b.camera = c
		b.color = pal.Land
		outlines := b.visible(c)
		paintLand(b.layer.reset(int(c.Width), int(c.Height)), cells(outlines), outlines, pal)
	}
	dst.DrawImage(b.layer.view, nil)
}
