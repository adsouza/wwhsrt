// Package render illustrates frames; it cannot advance business time.
package render

import (
	"bytes"
	"encoding/json"
	"image/color"
	"math"
	"sort"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/text/v2"
	"github.com/hajimehoshi/ebiten/v2/vector"
	"wwhsrt/assets"
	"wwhsrt/pkg/gameapi"
	"wwhsrt/pkg/mapview"
	"wwhsrt/pkg/ui"
)

type Map struct {
	Font *text.GoTextFaceSource
	// Text is the host's text scale. Its zero value draws the sizes written in this package, which is
	// what the browser client uses.
	Text  ui.TextScale
	rings [][][2]float64
	// ringCoast reports which rings get a coast line; see coastRings.
	ringCoast []bool
	anchors   map[string][][2]float64
	terrain   *relief
	public    *publicLayer
	scene     *ebiten.Image
	sceneKey  sceneKey
	marks     markers
}
type labelRect struct{ x, y, w, h float64 }

func (a labelRect) overlaps(b labelRect) bool {
	return a.x < b.x+b.w+4 && a.x+a.w+4 > b.x && a.y < b.y+b.h+4 && a.y+a.h+4 > b.y
}

func New() (*Map, error) {
	font, e := text.NewGoTextFaceSource(bytes.NewReader(assets.Font))
	if e != nil {
		return nil, e
	}
	m := &Map{Font: font}
	e = json.Unmarshal(assets.Land, &m.rings)
	if e != nil {
		return nil, e
	}
	m.ringCoast = coastRings(m.rings)
	e = json.Unmarshal(assets.Camera, &m.anchors)
	if e != nil {
		return nil, e
	}
	m.public, e = newPublicLayer()
	if e != nil {
		return nil, e
	}
	m.public.marks = &m.marks
	m.terrain, e = newRelief()
	return m, e
}
func (m *Map) Camera(w, h float64, portrait bool) mapview.Camera {
	key := "landscape"
	if portrait {
		key = "portrait"
	}
	points := []mapview.Point{}
	for _, p := range m.anchors[key] {
		points = append(points, mapview.Project(p[0], p[1]))
	}
	if len(points) == 0 {
		// No scenario anchors loaded: frame the built-in default region.
		return mapview.New(w, h)
	}
	// Metros the region keeps out of view (Dhaka for East Asia) are panned past an edge.
	avoid := []mapview.Point{}
	for _, p := range m.anchors["exclude"] {
		avoid = append(avoid, mapview.Project(p[0], p[1]))
	}
	c := mapview.FitAvoiding(w, h, points, avoid)
	if !portrait {
		mapview.LimitToWorld(&c)
	}
	return c
}

// face is the only place the map makes a font, so drawn and measured sizes cannot differ.
func (m *Map) face(base float64) *text.GoTextFace {
	return &text.GoTextFace{Source: m.Font, Size: m.Text.Size(base)}
}

// by scales a length written beside text of size base by however much that text has grown.
func (m *Map) by(base, v float64) float64 { return v * (m.Text.Size(base) / base) }

func (m *Map) Label(dst *ebiten.Image, s string, x, y, size float64, c color.Color) {
	op := &text.DrawOptions{}
	op.GeoM.Translate(x, y)
	op.ColorScale.ScaleWithColor(c)
	text.Draw(dst, s, m.face(size), op)
}

// legendRow is one row of the legend under the map's title: the prospect key or an idle-line key.
type legendRow struct {
	key, label  string
	size, textX float64
	box         labelRect
}

// legendTop is where the rows under the title and the station legend begin.
func (m *Map) legendTop() float64 { return m.by(20, 18) + m.by(20, 22) + m.by(11, 16) }

func (m *Map) legendRows(c mapview.Camera, s ui.State, idleLine, idleFleet bool) []legendRow {
	rows := []legendRow{}
	y := m.legendTop()
	if s.View == "map" || s.View == "build" {
		label := ui.Text(s.Locale, "prospectLegend")
		width := text.Advance(label, m.face(11)) + m.by(11, 48)
		box := labelRect{m.by(11, 12), y, math.Min(width, c.Width-24), m.by(11, 22)}
		rows = append(rows, legendRow{key: "prospect", label: label, size: 11, textX: m.by(11, 54), box: box})
		y += box.h
	}
	first, width := len(rows), m.by(12, 285)
	for _, legend := range []struct {
		show bool
		key  string
	}{{idleLine, "idleHighlightLegend"}, {idleFleet, "idleFleetHighlightLegend"}} {
		if !legend.show {
			continue
		}
		label := ui.Text(s.Locale, legend.key)
		if m.Text != (ui.TextScale{}) {
			// The browser keeps its fixed 285 px box; a scaled box holds its text.
			width = math.Max(width, m.by(12, 52)+text.Advance(label, m.face(12))+m.by(12, 8))
		}
		box := labelRect{m.by(12, 12), y, 0, m.by(12, 24)}
		rows = append(rows, legendRow{key: legend.key, label: label, size: 12, textX: m.by(12, 52), box: box})
		y += box.h
	}
	for i := first; i < len(rows); i++ {
		rows[i].box.w = math.Min(width, c.Width-24)
	}
	return rows
}
func CityPoint(c mapview.Camera, city gameapi.City) mapview.Point {
	lat, lon := city.Lat, city.Lon
	if city.StationName != "" {
		lat, lon = city.StationLat, city.StationLon
	}
	return c.Screen(mapview.Project(lat, lon))
}
func Pick(c mapview.Camera, cities []gameapi.City, at mapview.Point) gameapi.CityID {
	best := 24.0
	var id gameapi.CityID
	for _, city := range cities {
		p := CityPoint(c, city)
		d := math.Hypot(p.X-at.X, p.Y-at.Y)
		if d < best {
			best = d
			id = city.ID
		}
	}
	return id
}
func (m *Map) drawScene(dst *ebiten.Image, c mapview.Camera, f gameapi.Frame, s ui.State) {
	pal := ui.PaletteFor(s)
	dst.Fill(pal.Water)
	if f.PublicNetwork {
		m.public.drawLand(dst, c, s)
	} else {
		drawLand(dst, c, m.rings, m.ringCoast, pal)
	}
	if !s.HideTerrain {
		m.terrain.draw(dst, c, s)
	}
	if f.PublicNetwork {
		m.public.draw(dst, c, s)
	}
	// The static source network is cached; only active selections need a second
	// stroke. Resolve the route once, rather than searching it for every section.
	draftSections := map[string]bool{}
	if s.View == "services" {
		if route, ok := ui.ServiceRoute(f, s.ServiceCorridor); ok {
			for _, id := range route.Sections {
				draftSections[id] = true
			}
		}
	}
	// Track styles reflect authoritative infrastructure and construction state.
	for _, co := range f.Corridors {
		if f.PublicNetwork && co.Existing && !draftSections[co.ID] &&
			!(s.View == "build" && (co.FromCity == s.FromCity && co.ToCity == s.ToCity ||
				co.FromCity == s.ToCity && co.ToCity == s.FromCity)) {
			continue
		}
		var chosen *gameapi.Alignment
		stroke := pal.Existing
		dashed := false
		if co.Existing {
			chosen = &co.Alignments[0]
		} else {
			for _, project := range f.Projects {
				if project.CorridorID != co.ID {
					continue
				}
				for i := range co.Alignments {
					if co.Alignments[i].ID == project.AlignmentID {
						chosen = &co.Alignments[i]
					}
				}
				if project.Ready > f.MonthsResolved {
					stroke = pal.Construction
					dashed = true
				} else {
					stroke = pal.Accent
				}
			}
		}
		if draftSections[co.ID] && chosen != nil {
			stroke = pal.Draft
		}
		if chosen != nil {
			m.route(dst, c, chosen.Path, stroke, pal.Outline, dashed)
		}

	}
	m.drawProspects(dst, c, f, s)
	idle := []ui.IdleHighlight{}
	if s.View == "services" {
		idle = ui.IdleHighlights(f)
	}
	idleCities := map[gameapi.CityID]bool{}
	hasIdleLine, hasIdleFleet := false, false
	// Fleet overlays go first so a physical idle-line warning remains solid if
	// its geometry is also covered by an idle through-service fleet.
	for _, fleetOnly := range []bool{true, false} {
		for _, line := range idle {
			if line.FleetOnly != fleetOnly {
				continue
			}
			m.routeWidth(dst, c, line.Path, pal.WarningText, pal.Warning, line.FleetOnly, 11, 5)
			idleCities[line.FromCity] = true
			idleCities[line.ToCity] = true
			if fleetOnly {
				hasIdleFleet = true
			} else {
				hasIdleLine = true
			}
		}
	}
	// Selected labels take precedence; remaining labels avoid collisions.
	cities := append([]gameapi.City{}, f.Cities...)
	rank := func(id gameapi.CityID) int {
		if id == s.Selected {
			return 0
		}
		if idleCities[id] {
			return 1
		}
		return 2
	}
	sort.SliceStable(cities, func(i, j int) bool { return rank(cities[i].ID) < rank(cities[j].ID) })
	occupied := []labelRect{}
	// City labels keep clear of the legend. Both blocks are reserved from the top, as they always were.
	legend := m.legendRows(c, s, hasIdleLine, hasIdleFleet)
	idleRows, idleWidth := 0, math.Min(m.by(12, 285), c.Width-24)
	for _, row := range legend {
		if row.key == "prospect" {
			occupied = append(occupied, labelRect{row.box.x, m.legendTop(), row.box.w, row.box.h})
		} else {
			idleRows++
			idleWidth = row.box.w
		}
	}
	if len(idle) > 0 {
		occupied = append(occupied, labelRect{m.by(12, 12), m.legendTop(), idleWidth, float64(idleRows) * m.by(12, 24)})
	}
	for _, city := range cities {
		p := CityPoint(c, city)
		if p.X < 0 || p.Y < 0 || p.X > c.Width || p.Y > c.Height {
			continue
		}
		if idleCities[city.ID] {
			m.marks.ring(dst, float32(p.X), float32(p.Y), 10, 3, pal.WarningText)
		}
		if city.ID == s.Selected {
			m.marks.disc(dst, float32(p.X), float32(p.Y), 13, pal.Surface)
			m.marks.ring(dst, float32(p.X), float32(p.Y), 13, 3, pal.Accent)
		}
		station, hasStation := ui.StationAt(f, city.ID)
		if !hasStation {
			m.marks.ring(dst, float32(p.X), float32(p.Y), 4.5, 1.5, pal.Muted)
		} else if !station.Open {
			vector.FillRect(dst, float32(p.X-5), float32(p.Y-5), 10, 10, pal.Construction, true)
		} else {
			ink := pal.Accent
			if station.Public {
				ink = pal.Existing
			}
			m.marks.disc(dst, float32(p.X), float32(p.Y), 6, ink)
		}
		m.marks.disc(dst, float32(p.X), float32(p.Y), 3.5, pal.Surface)
		if s.View == "build" {
			tag := ""
			if city.ID == s.FromCity {
				tag = "A"
			}
			if city.ID == s.ToCity {
				tag = "B"
			}
			if tag != "" {
				m.marks.disc(dst, float32(p.X), float32(p.Y), float32(m.by(13, 12)), pal.Accent)
				tagInk := pal.OnAction
				if ui.IsDark(s) {
					tagInk = pal.Outline
				}
				m.Label(dst, tag, p.X-m.by(13, 4), p.Y-m.by(13, 9), 13, tagInk)
			}
		}
		label := ui.Text(s.Locale, string(city.ID))
		width := text.Advance(label, m.face(15))
		for _, offset := range []mapview.Point{
			{X: m.by(15, 12), Y: m.by(15, -10)},
			{X: -width - m.by(15, 12), Y: m.by(15, -10)},
			{X: -width / 2, Y: m.by(15, 14)},
			{X: -width / 2, Y: m.by(15, -30)},
		} {
			x, y := math.Max(4, math.Min(c.Width-width-4, p.X+offset.X)), math.Max(4, p.Y+offset.Y)
			r := labelRect{x, y, width, m.by(15, 22)}
			collision := false
			for _, o := range occupied {
				if r.x < o.x+o.w+4 && r.x+r.w+4 > o.x && r.y < o.y+o.h+4 && r.y+r.h+4 > o.y {
					collision = true
					break
				}
			}
			if collision {
				continue
			}
			occupied = append(occupied, r)
			m.Label(dst, label, r.x+1, r.y+1, 15, pal.Background)
			m.Label(dst, label, r.x, r.y, 15, pal.Text)
			break
		}
	}
	if !s.HideTerrain {
		m.terrain.labels(m, dst, c, s, occupied)
	}
	title := ui.Text(s.Locale, "east_asia")
	for _, r := range mapview.VisibleRegions(s.Portrait) {
		if r.ID == s.Region {
			title = r.Name
			if s.Locale == "zh-Hans" {
				title = r.Chinese
			}
		}
	}
	m.Label(dst, title, m.by(20, 20), m.by(20, 18), 20, pal.Text)
	if stop := m.PublicStop(s.SourceStation); stop != nil {
		p := c.Screen(mapview.Project(stop.Lat, stop.Lon))
		vector.StrokeCircle(dst, float32(p.X), float32(p.Y), 9, 2, pal.Accent, true)
		m.Label(dst, stop.Name, m.by(14, 12), c.Height-m.by(14, 28), 14, pal.Text)
	}
	m.Label(dst, ui.Text(s.Locale, "stationLegend"), m.by(11, 20), m.by(20, 18)+m.by(20, 22), 11, pal.Muted)
	for _, row := range legend {
		y := float32(row.box.y)
		if row.key == "prospect" {
			for _, x := range []float64{24, 34, 44} {
				vector.FillCircle(dst, float32(m.by(11, x)), y+float32(m.by(11, 9)), float32(m.by(11, 2.5)), pal.Action, true)
			}
			m.Label(dst, row.label, row.textX, row.box.y, row.size, pal.Text)
			continue
		}
		vector.FillRect(dst, float32(row.box.x), y, float32(row.box.w), float32(row.box.h), pal.Warning, true)
		mid, thick := y+float32(m.by(12, 12)), float32(m.by(12, 5))
		if row.key == "idleFleetHighlightLegend" {
			for x := 20.0; x < 43; x += 10 {
				vector.StrokeLine(dst, float32(m.by(12, x)), mid, float32(m.by(12, x+5)), mid, thick, pal.WarningText, true)
			}
		} else {
			vector.StrokeLine(dst, float32(m.by(12, 20)), mid, float32(m.by(12, 43)), mid, thick, pal.WarningText, true)
		}
		m.Label(dst, row.label, row.textX, row.box.y+m.by(12, 4), row.size, pal.WarningText)
	}

}

func routePoint(c mapview.Camera, path []gameapi.GeoPoint, u float64) mapview.Point {
	count := len(path) - 1
	at := u * float64(count)
	i := min(count-1, int(at))
	v := at - float64(i)
	a, b := c.Screen(mapview.Project(path[i].Lat, path[i].Lon)), c.Screen(mapview.Project(path[i+1].Lat, path[i+1].Lon))
	return mapview.Point{X: a.X + (b.X-a.X)*v, Y: a.Y + (b.Y-a.Y)*v}
}

func (m *Map) route(
	dst *ebiten.Image,
	c mapview.Camera,
	path []gameapi.GeoPoint,
	stroke, outline color.Color,
	dashed bool,
) {
	m.routeWidth(dst, c, path, stroke, outline, dashed, 5, 3)
}

func (m *Map) routeWidth(
	dst *ebiten.Image,
	c mapview.Camera,
	points []gameapi.GeoPoint,
	stroke, outline color.Color,
	dashed bool,
	outerWidth, innerWidth float32,
) {
	var path vector.Path
	for i := 0; i < len(points)-1; i++ {
		a, b, ok := screenSegment(
			c,
			c.Screen(mapview.Project(points[i].Lat, points[i].Lon)),
			c.Screen(mapview.Project(points[i+1].Lat, points[i+1].Lon)),
		)
		if !ok {
			continue
		}
		n := 1
		if dashed {
			n = max(1, int(math.Hypot(b.X-a.X, b.Y-a.Y)/6))
		}
		for j := 0; j < n; j++ {
			if dashed && j%2 == 1 {
				continue
			}
			u, v := float64(j)/float64(n), float64(j+1)/float64(n)
			path.MoveTo(float32(a.X+(b.X-a.X)*u), float32(a.Y+(b.Y-a.Y)*u))
			path.LineTo(float32(a.X+(b.X-a.X)*v), float32(a.Y+(b.Y-a.Y)*v))
		}
	}
	for _, style := range []struct {
		width float32
		ink   color.Color
	}{{outerWidth, outline}, {innerWidth, stroke}} {
		op := &vector.DrawPathOptions{AntiAlias: true}
		op.ColorScale.ScaleWithColor(style.ink)
		vector.StrokePath(dst, &path, &vector.StrokeOptions{Width: style.width}, op)
	}
}
