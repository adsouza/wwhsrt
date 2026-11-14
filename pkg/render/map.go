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
	Font     *text.GoTextFaceSource
	rings    [][][2]float64
	anchors  map[string][][2]float64
	terrain  *relief
	public   *publicLayer
	scene    *ebiten.Image
	sceneKey sceneKey
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
	e = json.Unmarshal(assets.Camera, &m.anchors)
	if e != nil {
		return nil, e
	}
	m.public, e = newPublicLayer()
	if e != nil {
		return nil, e
	}
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
	return mapview.Fit(w, h, points)
}
func (m *Map) Label(dst *ebiten.Image, s string, x, y, size float64, c color.Color) {
	op := &text.DrawOptions{}
	op.GeoM.Translate(x, y)
	op.ColorScale.ScaleWithColor(c)
	text.Draw(dst, s, &text.GoTextFace{Source: m.Font, Size: size}, op)
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
		for _, ring := range m.rings {
			var path vector.Path
			points := screenRing(c, ring)
			if len(points) < 3 {
				continue
			}
			for i, q := range points {
				if i == 0 {
					path.MoveTo(float32(q.X), float32(q.Y))
				} else {
					path.LineTo(float32(q.X), float32(q.Y))
				}
			}
			path.Close()
			op := &vector.DrawPathOptions{AntiAlias: true}
			op.ColorScale.ScaleWithColor(pal.Land)
			vector.FillPath(dst, &path, nil, op)
			op = &vector.DrawPathOptions{AntiAlias: true}
			op.ColorScale.ScaleWithColor(pal.Coast)
			vector.StrokePath(dst, &path, &vector.StrokeOptions{Width: 1}, op)
		}
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
	if s.View == "map" || s.View == "build" {
		width := text.Advance(ui.Text(s.Locale, "prospectLegend"), &text.GoTextFace{Source: m.Font, Size: 11}) + 48
		occupied = append(occupied, labelRect{12, 56, math.Min(width, c.Width-24), 22})
	}
	if len(idle) > 0 {
		rows := 0
		if hasIdleLine {
			rows++
		}
		if hasIdleFleet {
			rows++
		}
		occupied = append(occupied, labelRect{12, 56, math.Min(285, c.Width-24), float64(rows * 24)})
	}
	for _, city := range cities {
		p := CityPoint(c, city)
		if p.X < 0 || p.Y < 0 || p.X > c.Width || p.Y > c.Height {
			continue
		}
		if idleCities[city.ID] {
			vector.StrokeCircle(dst, float32(p.X), float32(p.Y), 10, 3, pal.WarningText, true)
		}
		if city.ID == s.Selected {
			vector.FillCircle(dst, float32(p.X), float32(p.Y), 13, pal.Surface, true)
			vector.StrokeCircle(dst, float32(p.X), float32(p.Y), 13, 3, pal.Accent, true)
		}
		station, hasStation := ui.StationAt(f, city.ID)
		if !hasStation {
			vector.StrokeCircle(dst, float32(p.X), float32(p.Y), 4.5, 1.5, pal.Muted, true)
		} else if !station.Open {
			vector.FillRect(dst, float32(p.X-5), float32(p.Y-5), 10, 10, pal.Construction, true)
		} else {
			ink := pal.Accent
			if station.Public {
				ink = pal.Existing
			}
			vector.FillCircle(dst, float32(p.X), float32(p.Y), 6, ink, true)
		}
		vector.FillCircle(dst, float32(p.X), float32(p.Y), 3.5, pal.Surface, true)
		if s.View == "build" {
			tag := ""
			if city.ID == s.FromCity {
				tag = "A"
			}
			if city.ID == s.ToCity {
				tag = "B"
			}
			if tag != "" {
				vector.FillCircle(dst, float32(p.X), float32(p.Y), 12, pal.Accent, true)
				tagInk := pal.OnAction
				if ui.IsDark(s) {
					tagInk = pal.Outline
				}
				m.Label(dst, tag, p.X-4, p.Y-9, 13, tagInk)
			}
		}
		label := ui.Text(s.Locale, string(city.ID))
		width := text.Advance(label, &text.GoTextFace{Source: m.Font, Size: 15})
		for _, offset := range []mapview.Point{
			{X: 12, Y: -10},
			{X: -width - 12, Y: -10},
			{X: -width / 2, Y: 14},
			{X: -width / 2, Y: -30},
		} {
			r := labelRect{math.Max(4, math.Min(c.Width-width-4, p.X+offset.X)), math.Max(4, p.Y+offset.Y), width, 22}
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
	m.Label(dst, title, 20, 18, 20, pal.Text)
	if stop := m.PublicStop(s.SourceStation); stop != nil {
		p := c.Screen(mapview.Project(stop.Lat, stop.Lon))
		vector.StrokeCircle(dst, float32(p.X), float32(p.Y), 9, 2, pal.Accent, true)
		m.Label(dst, stop.Name, 12, c.Height-28, 14, pal.Text)
	}
	m.Label(dst, ui.Text(s.Locale, "stationLegend"), 20, 40, 11, pal.Muted)
	y := float32(56)
	if s.View == "map" || s.View == "build" {
		vector.FillCircle(dst, 24, y+9, 2.5, pal.Action, true)
		vector.FillCircle(dst, 34, y+9, 2.5, pal.Action, true)
		vector.FillCircle(dst, 44, y+9, 2.5, pal.Action, true)
		m.Label(dst, ui.Text(s.Locale, "prospectLegend"), 54, float64(y), 11, pal.Text)
		y += 22
	}
	for _, legend := range []struct {
		show, fleet bool
		key         string
	}{{hasIdleLine, false, "idleHighlightLegend"}, {hasIdleFleet, true, "idleFleetHighlightLegend"}} {
		if !legend.show {
			continue
		}
		vector.FillRect(dst, 12, y, float32(math.Min(285, c.Width-24)), 24, pal.Warning, true)
		if legend.fleet {
			for x := float32(20); x < 43; x += 10 {
				vector.StrokeLine(dst, x, y+12, x+5, y+12, 5, pal.WarningText, true)
			}
		} else {
			vector.StrokeLine(dst, 20, y+12, 43, y+12, 5, pal.WarningText, true)
		}
		m.Label(dst, ui.Text(s.Locale, legend.key), 52, float64(y+4), 12, pal.WarningText)
		y += 24
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
