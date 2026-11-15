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
// basemap before it did, and every vertex becomes sixteen or more in
// Ebitengine's antialiased fill (see screenTolerance). One pixel draws about
// 22,000. It is no coarser than that earlier basemap was at the World view,
// and at close zooms it is finer than the data.
const landTolerance = 1.0

// basemap is the one worldwide land outline: land pieces, the lakes cut out of
// them, and the islands inside those lakes. tools/prepare-basemap.py writes it,
// detailed near lines and coarse elsewhere. There are no lines between
// countries.
//
// Land and lake islands wind counter-clockwise and lakes clockwise, so the
// single nonzero fill in paintLand leaves lakes clear and the scene's water
// shows through them. Lakes need no fill of their own, which would be a third
// antialiased pass and another map-sized box in Ebitengine's stencil.
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
	return out
}

// visible returns the outlines to draw, in order and in screen space, each
// clipped to the screen. An outline wholly off screen is skipped without
// touching its points, and one smaller than landTolerance is not drawn.
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

// draw paints the land, redrawing its layer only when the camera or palette changed.
func (b *basemap) draw(dst *ebiten.Image, c mapview.Camera, pal ui.Palette) {
	if b.layer.view == nil || b.camera != c || b.color != pal.Land {
		b.camera = c
		b.color = pal.Land
		paintLand(b.layer.reset(int(c.Width), int(c.Height)), b.visible(c), pal)
	}
	dst.DrawImage(b.layer.view, nil)
}
