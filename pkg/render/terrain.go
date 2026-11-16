package render

import (
	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/text/v2"
	"github.com/hajimehoshi/ebiten/v2/vector"
	"image"
	"image/color"
	"math"
	"wwhsrt/assets"
	"wwhsrt/pkg/mapview"
	"wwhsrt/pkg/terrain"
	"wwhsrt/pkg/ui"
)

// relief is the terrain layer: shaded tiles, Japan's contours, and a few named features.
//
// Tiles are loaded as a view asks for them (see tileCache). Each visible area of land is drawn once, from
// the finest tile that is ready, so a view shows coarse relief at once and sharpens as tiles arrive.
type relief struct {
	set      *terrain.Set
	cache    *tileCache
	contours []contourSet
	features []terrain.Feature
	// arrivals counts the frames in which a tile arrived. It is part of the scene's key, so a late tile
	// redraws the map without the player moving it.
	arrivals uint64
	// world holds each drawn area's mesh points on the map, which do not change with the camera.
	world     map[meshKey][]mapview.Point
	triangles map[int][]uint16
	vertices  []ebiten.Vertex
}

// meshKey names an area's mesh: the area and how many cells it is cut into along each side.
type meshKey struct {
	area  terrain.Address
	cells int
}

const (
	// meshCells is how many cells a drawn area is cut into along each side when it is of the level being
	// drawn. Tiles are square on the Web Mercator grid and curved on this map; eight cells keep the
	// curve within a fraction of a pixel.
	meshCells = 8
	// meshCellsMax bounds the cells of an area coarser than the level being drawn (see cellsFor).
	meshCellsMax = 64
	// worldLimit bounds the remembered meshes; past it they are forgotten and worked out again.
	worldLimit = 4096
)

// cellsFor is how many cells a side an area is cut into in a view drawn from a level. An area coarser
// than the level spans several of the level's tiles, and is cut as finely as they are: every mesh point
// on a fine piece's edge is then also a point of the coarse piece beside it. Without that, the coarse
// piece's longer straight edges leave slivers of plain land along the join.
func cellsFor(area terrain.Address, level int) int {
	return min(meshCellsMax, meshCells<<max(0, level-area.Z))
}

// indices joins a mesh of the given cells a side into triangles.
func (t *relief) indices(cells int) []uint16 {
	if out := t.triangles[cells]; out != nil {
		return out
	}
	out := make([]uint16, 0, cells*cells*6)
	for y := range cells {
		for x := range cells {
			a := uint16(y*(cells+1) + x)
			b, c := a+1, a+uint16(cells)+1
			out = append(out, a, b, c, b, c+1, c)
		}
	}
	t.triangles[cells] = out
	return out
}

// contourSet is one region's contour lines and the box on the map that holds them.
type contourSet struct {
	lines    [][]mapview.Point
	min, max mapview.Point
}

func (s contourSet) onScreen(c mapview.Camera) bool {
	a, b := c.Screen(s.min), c.Screen(s.max) // the map's y runs up and the screen's down
	return b.X >= 0 && a.X <= c.Width && a.Y >= 0 && b.Y <= c.Height
}

// newRelief reads which tiles exist. It decodes none of them.
func newRelief() (*relief, error) {
	set, e := terrain.Load(assets.Terrain)
	if e != nil {
		return nil, e
	}
	t := &relief{set: set, features: set.Features, world: map[meshKey][]mapview.Point{}, triangles: map[int][]uint16{}}
	for _, region := range set.Regions {
		lines := contourSet{
			min: mapview.Point{X: math.Inf(1), Y: math.Inf(1)},
			max: mapview.Point{X: math.Inf(-1), Y: math.Inf(-1)},
		}
		for _, line := range region.Contours {
			points := make([]mapview.Point, len(line.Points))
			for i, p := range line.Points {
				points[i] = mapview.Project(p[0], p[1])
				lines.min.X, lines.min.Y = math.Min(lines.min.X, points[i].X), math.Min(lines.min.Y, points[i].Y)
				lines.max.X, lines.max.Y = math.Max(lines.max.X, points[i].X), math.Max(lines.max.Y, points[i].Y)
			}
			lines.lines = append(lines.lines, points)
		}
		t.contours = append(t.contours, lines)
	}
	t.cache = newTileCache(tileCap, t.decode, func(img *image.RGBA) texture {
		// Unmanaged: a tile on Ebitengine's shared atlas would leave its page held by small images after
		// it is freed (see layer). An unmanaged image's texture is rounded up to a power of two, which
		// 256 already is.
		tile := ebiten.NewImageWithOptions(img.Bounds(), &ebiten.NewImageOptions{Unmanaged: true})
		tile.WritePixels(img.Pix)
		return tile
	})
	return t, nil
}

func (t *relief) decode(a terrain.Address) (*image.RGBA, error) {
	return t.set.Decode(assets.Terrain, a)
}

// tick lets decoded tiles in. Call it once per frame. While terrain is hidden nothing is taken in and
// nothing more is decoded.
func (t *relief) tick(hidden bool) {
	if hidden {
		t.cache.want(nil)
		return
	}
	if t.cache.frame() {
		t.arrivals++
	}
}

// level is the tile level to draw for a camera, from how wide a degree of longitude is at its centre.
func (t *relief) level(c mapview.Camera) int {
	lat, lon := mapview.Unproject(c.World(mapview.Point{X: c.Width / 2, Y: c.Height / 2}))
	lat = math.Max(-80, math.Min(80, lat))
	a, b := c.Screen(mapview.Project(lat, lon-.5)), c.Screen(mapview.Project(lat, lon+.5))
	return terrain.LevelFor(math.Abs(b.X - a.X))
}

// points are an area's mesh points on the map: a grid of cells a side, row by row from the north-west.
func (t *relief) points(a terrain.Address, cells int) []mapview.Point {
	key := meshKey{a, cells}
	if p := t.world[key]; p != nil {
		return p
	}
	if len(t.world) >= worldLimit {
		clear(t.world)
	}
	p := make([]mapview.Point, (cells+1)*(cells+1))
	for i := range p {
		lat, lon := terrain.Corner(a, float64(i%(cells+1))/float64(cells), float64(i/(cells+1))/float64(cells))
		p[i] = mapview.Project(lat, lon)
	}
	t.world[key] = p
	return p
}

// visible reports whether any part of a tile may be on screen, from its corners, edge middles and centre.
func (t *relief) visible(c mapview.Camera, a terrain.Address) bool {
	p := t.points(a, meshCells)
	x0, y0, x1, y1 := math.Inf(1), math.Inf(1), math.Inf(-1), math.Inf(-1)
	for row := 0; row <= meshCells; row += meshCells / 2 {
		for col := 0; col <= meshCells; col += meshCells / 2 {
			s := c.Screen(p[row*(meshCells+1)+col])
			x0, y0, x1, y1 = math.Min(x0, s.X), math.Min(y0, s.Y), math.Max(x1, s.X), math.Max(y1, s.Y)
		}
	}
	return x1 >= 0 && y1 >= 0 && x0 <= c.Width && y0 <= c.Height
}

// cover is what to draw for a camera and which tiles it wants loaded.
func (t *relief) cover(c mapview.Camera) ([]terrain.Piece, []terrain.Address) {
	return t.set.Cover(t.level(c), func(a terrain.Address) bool { return t.visible(c, a) }, t.cache.ready)
}

// reliefOpacity fades relief at wide views, where it would otherwise bury the lines.
func reliefOpacity(c mapview.Camera) float32 { return float32(math.Min(.98, .42+.075*c.Zoom)) }

// mesh places one piece on screen for a view drawn from a level. It reads the part of the source
// tile's image that covers the piece's area (see terrain.Piece.Rect).
func (t *relief) mesh(c mapview.Camera, p terrain.Piece, level int, opacity float32, dark bool) []ebiten.Vertex {
	cells := cellsFor(p.Area, level)
	left, top, size := p.Rect()
	r, g, b := float32(1), float32(1), float32(1)
	if dark {
		r, g, b = .30, .38, .40
	}
	t.vertices = t.vertices[:0]
	for i, point := range t.points(p.Area, cells) {
		s := c.Screen(point)
		t.vertices = append(t.vertices, ebiten.Vertex{
			DstX:   float32(s.X),
			DstY:   float32(s.Y),
			SrcX:   float32(left + size*float64(i%(cells+1))/float64(cells)),
			SrcY:   float32(top + size*float64(i/(cells+1))/float64(cells)),
			ColorR: r,
			ColorG: g,
			ColorB: b,
			ColorA: opacity,
		})
	}
	return t.vertices
}

func (t *relief) draw(dst *ebiten.Image, c mapview.Camera, s ui.State) {
	pal := ui.PaletteFor(s)
	level := t.level(c)
	pieces, wanted := t.set.Cover(level, func(a terrain.Address) bool { return t.visible(c, a) }, t.cache.ready)
	t.cache.want(wanted)
	t.cache.beginScene()
	opacity, dark := reliefOpacity(c), ui.IsDark(s)
	op := &ebiten.DrawTrianglesOptions{Filter: ebiten.FilterLinear}
	for _, p := range pieces {
		if tile, ok := t.cache.get(p.Source).(*ebiten.Image); ok {
			dst.DrawTriangles(t.mesh(c, p, level, opacity, dark), t.indices(cellsFor(p.Area, level)), tile, op)
		}
	}
	if c.Zoom >= 4 {
		for _, region := range t.contours {
			if !region.onScreen(c) {
				continue
			}
			for _, line := range region.lines {
				var path vector.Path
				for j, p := range line {
					q := c.Screen(p)
					if j == 0 {
						path.MoveTo(float32(q.X), float32(q.Y))
					} else {
						path.LineTo(float32(q.X), float32(q.Y))
					}
				}
				op := &vector.DrawPathOptions{AntiAlias: true}
				op.ColorScale.ScaleWithColor(color.NRGBA{pal.Contour.R, pal.Contour.G, pal.Contour.B, 75})
				vector.StrokePath(dst, &path, &vector.StrokeOptions{Width: .8}, op)
			}
		}
	}
	t.marks(dst, c, pal)
}

// marks draws the peak and lake symbols of the named features.
func (t *relief) marks(dst *ebiten.Image, c mapview.Camera, pal ui.Palette) {
	if c.Zoom < 3 {
		return
	}
	for _, f := range t.features {
		p := c.Screen(mapview.Project(f.Lat, f.Lon))
		if p.X < 0 || p.Y < 0 || p.X > c.Width || p.Y > c.Height {
			continue
		}
		if f.Kind == "peak" {
			var path vector.Path
			path.MoveTo(float32(p.X), float32(p.Y-8))
			path.LineTo(float32(p.X-7), float32(p.Y+5))
			path.LineTo(float32(p.X+7), float32(p.Y+5))
			path.Close()
			op := &vector.DrawPathOptions{AntiAlias: true}
			op.ColorScale.ScaleWithColor(pal.TerrainText)
			vector.FillPath(dst, &path, nil, op)
			vector.StrokeLine(dst, float32(p.X-2), float32(p.Y-2), float32(p.X+2), float32(p.Y-2), 2, color.White, true)
		} else if f.Kind == "lake" {
			vector.FillCircle(dst, float32(p.X), float32(p.Y), 4.5, pal.Water, true)
			vector.StrokeCircle(dst, float32(p.X), float32(p.Y), 4.5, 1, pal.Coast, true)
		}
	}
}

// terrainLegendColours are the legend's swatches: the relief colours at 0, 400, 1,200, 2,600 and 4,800 m.
var terrainLegendColours = []color.NRGBA{
	{231, 244, 204, 255},
	{199, 211, 145, 255},
	{183, 168, 133, 255},
	{166, 141, 128, 255},
	{244, 243, 240, 255},
}

// terrainLegendLayout places the terrain key: a title, five swatches and three captions under them.
type terrainLegendLayout struct {
	box                      labelRect // y is chosen by the caller
	swatchY, swatchW, labelY float64
	labels                   [3]struct {
		x    float64 // from the box's left edge
		text string
	}
}

func (m *Map) terrainLegend(c mapview.Camera, locale string) terrainLegendLayout {
	l := terrainLegendLayout{swatchY: m.by(11, 24), swatchW: m.by(9, 24)}
	l.labelY = l.swatchY + m.by(9, 9)
	left, width := m.by(9, 8), m.by(9, 245)
	l.labels[0].x, l.labels[0].text = left, "0 m"
	l.labels[1].x, l.labels[1].text = m.by(9, 93), "4800+ m"
	l.labels[2].x, l.labels[2].text = m.by(9, 138), "Mapzen / USGS"
	if m.Text != (ui.TextScale{}) {
		// The browser keeps its fixed positions; scaled captions are placed by their measured widths.
		face := m.face(9)
		low, high := text.Advance(l.labels[0].text, face), text.Advance(l.labels[1].text, face)
		l.labels[1].x = math.Max(left+low+m.by(9, 10), left+5*l.swatchW-high)
		l.labels[2].x = l.labels[1].x + high + m.by(9, 10)
		title := m.by(11, 8) + text.Advance(ui.Text(locale, "terrainLegend"), m.face(11))
		width = math.Max(width, math.Max(title, l.labels[2].x+text.Advance(l.labels[2].text, face))+left)
	}
	l.box = labelRect{m.by(11, 12), 0, math.Min(width, c.Width-24), l.labelY + m.by(9, 19)}
	return l
}

func (t *relief) labels(m *Map, dst *ebiten.Image, c mapview.Camera, s ui.State, occupied []labelRect) {
	pal := ui.PaletteFor(s)
	dark := ui.IsDark(s)
	locale := s.Locale
	// City labels and endpoint picking always have priority over terrain text.
	layout := m.terrainLegend(c, locale)
	legend, top := layout.box, m.legendTop()
	legendVisible := false
	for _, y := range []float64{top, c.Height - m.by(11, 173) - legend.h, top + m.by(11, 60)} {
		legend.y = y
		if y < top || y+legend.h > c.Height-m.by(11, 150) {
			continue
		}
		collision := false
		for _, o := range occupied {
			if legend.overlaps(o) {
				collision = true
				break
			}
		}
		if !collision {
			legendVisible = true
			occupied = append(occupied, legend)
			break
		}
	}
	if c.Zoom >= 3 {
		for _, f := range t.features {
			p := c.Screen(mapview.Project(f.Lat, f.Lon))
			if p.X < 8 || p.Y < 8 || p.X > c.Width-8 || p.Y > c.Height-52 {
				continue
			}
			value := ui.Text(locale, f.Key)
			width := text.Advance(value, m.face(12))
			for _, off := range []mapview.Point{
				{X: m.by(12, 10), Y: m.by(12, -8)},
				{X: -width - m.by(12, 10), Y: m.by(12, -8)},
				{X: -width / 2, Y: m.by(12, 10)},
				{X: -width / 2, Y: m.by(12, -25)},
			} {
				r := labelRect{p.X + off.X, p.Y + off.Y, width, m.by(12, 18)}
				if r.x < 4 || r.x+r.w > c.Width-4 || r.y < 4 {
					continue
				}
				collision := false
				for _, o := range occupied {
					if r.overlaps(o) {
						collision = true
						break
					}
				}
				if collision {
					continue
				}
				occupied = append(occupied, r)
				vector.FillRect(dst, float32(r.x-2), float32(r.y-1), float32(r.w+4), float32(r.h), pal.TerrainFill, true)
				m.Label(dst, value, r.x, r.y, 12, pal.TerrainText)
				break
			}
		}
	}
	if !legendVisible {
		return
	}
	vector.FillRect(
		dst,
		float32(legend.x),
		float32(legend.y),
		float32(legend.w),
		float32(legend.h),
		pal.TerrainFill,
		true,
	)
	m.Label(dst, ui.Text(locale, "terrainLegend"), legend.x+m.by(11, 8), legend.y+m.by(11, 4), 11, pal.Text)
	for i, col := range terrainLegendColours {
		vector.FillRect(
			dst,
			float32(legend.x+m.by(9, 8))+float32(i)*float32(layout.swatchW),
			float32(legend.y+layout.swatchY),
			float32(layout.swatchW),
			float32(m.by(9, 7)),
			terrainSwatch(col, dark),
			true,
		)
	}
	for _, label := range layout.labels {
		m.Label(dst, label.text, legend.x+label.x, legend.y+layout.labelY, 9, pal.Text)
	}
}

func terrainSwatch(c color.NRGBA, dark bool) color.NRGBA {
	if dark {
		c.R = uint8(float64(c.R) * .30)
		c.G = uint8(float64(c.G) * .38)
		c.B = uint8(float64(c.B) * .40)
	}
	return c
}
