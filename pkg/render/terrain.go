package render

import (
	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/text/v2"
	"github.com/hajimehoshi/ebiten/v2/vector"
	"image/color"
	"math"
	"wwhsrt/assets"
	"wwhsrt/pkg/mapview"
	"wwhsrt/pkg/terrain"
	"wwhsrt/pkg/ui"
)

type reliefTile struct {
	image    *ebiten.Image
	world    []mapview.Point
	vertices []ebiten.Vertex
	indices  []uint16
	contours [][]mapview.Point
}
type relief struct {
	tiles    []reliefTile
	features []terrain.Feature
	lakes    [][]mapview.Point
}

func newRelief() (*relief, error) {
	d, e := terrain.Load(assets.Terrain)
	if e != nil {
		return nil, e
	}
	out := &relief{features: d.Features}
	for _, region := range d.Regions {
		t := reliefTile{image: ebiten.NewImageFromImage(region.Image)}
		cols, rows := region.Width/16, region.Height/16
		for y := 0; y <= rows; y++ {
			for x := 0; x <= cols; x++ {
				px, py := float64(x*16), float64(y*16)
				t.world = append(t.world, terrain.ProjectedPixel(region, px, py))
				t.vertices = append(
					t.vertices,
					ebiten.Vertex{SrcX: float32(px), SrcY: float32(py), ColorR: 1, ColorG: 1, ColorB: 1, ColorA: 1},
				)
			}
		}
		for y := 0; y < rows; y++ {
			for x := 0; x < cols; x++ {
				a := uint16(y*(cols+1) + x)
				b := a + 1
				c := a + uint16(cols+1)
				d := c + 1
				t.indices = append(t.indices, a, b, c, b, d, c)
			}
		}
		for _, line := range region.Contours {
			points := []mapview.Point{}
			for _, p := range line.Points {
				points = append(points, mapview.Project(p[0], p[1]))
			}
			t.contours = append(t.contours, points)
		}
		out.tiles = append(out.tiles, t)
	}
	for _, lake := range d.Lakes {
		points := []mapview.Point{}
		for _, p := range lake.Points {
			points = append(points, mapview.Project(p[0], p[1]))
		}
		out.lakes = append(out.lakes, points)
	}
	return out, nil
}
func (t *relief) draw(dst *ebiten.Image, c mapview.Camera, s ui.State) {
	pal := ui.PaletteFor(s)
	dark := ui.IsDark(s)
	opacity := float32(math.Min(.98, .42+.075*c.Zoom))
	for i := range t.tiles {
		tile := &t.tiles[i]
		x0, y0, x1, y1 := math.Inf(1), math.Inf(1), math.Inf(-1), math.Inf(-1)
		for j, p := range tile.world {
			s := c.Screen(p)
			tile.vertices[j].DstX = float32(s.X)
			tile.vertices[j].DstY = float32(s.Y)
			tile.vertices[j].ColorA = opacity
			tile.vertices[j].ColorR, tile.vertices[j].ColorG, tile.vertices[j].ColorB = 1, 1, 1
			if dark {
				tile.vertices[j].ColorR, tile.vertices[j].ColorG, tile.vertices[j].ColorB = .30, .38, .40
			}
			x0 = math.Min(x0, s.X)
			x1 = math.Max(x1, s.X)
			y0 = math.Min(y0, s.Y)
			y1 = math.Max(y1, s.Y)
		}
		if x1 < 0 || y1 < 0 || x0 > c.Width || y0 > c.Height {
			continue
		}
		dst.DrawTriangles(
			tile.vertices,
			tile.indices,
			tile.image,
			&ebiten.DrawTrianglesOptions{Filter: ebiten.FilterLinear},
		)
		if c.Zoom >= 4 {
			for _, line := range tile.contours {
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
	for _, lake := range t.lakes {
		var path vector.Path
		for j, p := range lake {
			q := c.Screen(p)
			if j == 0 {
				path.MoveTo(float32(q.X), float32(q.Y))
			} else {
				path.LineTo(float32(q.X), float32(q.Y))
			}
		}
		path.Close()
		op := &vector.DrawPathOptions{AntiAlias: true}
		op.ColorScale.ScaleWithColor(pal.Water)
		vector.FillPath(dst, &path, nil, op)
		op = &vector.DrawPathOptions{AntiAlias: true}
		op.ColorScale.ScaleWithColor(pal.Coast)
		vector.StrokePath(dst, &path, &vector.StrokeOptions{Width: 1}, op)
	}
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
	l.labels[1].x, l.labels[1].text = m.by(9, 93), "2000+ m"
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
	for i, col := range []color.NRGBA{
		{231, 244, 204, 255},
		{199, 211, 145, 255},
		{183, 168, 133, 255},
		{177, 157, 139, 255},
		{234, 230, 216, 255},
	} {
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
