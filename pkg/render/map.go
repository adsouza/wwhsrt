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
	Font    *text.GoTextFaceSource
	rings   [][][2]float64
	anchors map[string][][2]float64
	terrain *relief
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
	return mapview.Fit(w, h, points)
}
func (m *Map) Label(dst *ebiten.Image, s string, x, y, size float64, c color.Color) {
	op := &text.DrawOptions{}
	op.GeoM.Translate(x, y)
	op.ColorScale.ScaleWithColor(c)
	text.Draw(dst, s, &text.GoTextFace{Source: m.Font, Size: size}, op)
}
func CityPoint(c mapview.Camera, city gameapi.City) mapview.Point {
	return c.Screen(mapview.Project(city.Lat, city.Lon))
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
func (m *Map) Draw(dst *ebiten.Image, c mapview.Camera, f gameapi.Frame, s ui.State, tick float64) {
	pal := ui.PaletteFor(s)
	dst.Fill(pal.Water)
	for _, ring := range m.rings {
		var path vector.Path
		for i, p := range ring {
			q := c.Screen(mapview.Project(p[1], p[0]))
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
	if !s.HideTerrain {
		m.terrain.draw(dst, c, s)
	}
	// Track styles reflect authoritative infrastructure and construction state.
	for _, co := range f.Corridors {
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
		if chosen != nil {
			m.route(dst, c, chosen.Path, stroke, pal.Outline, dashed)
		}
		if s.View == "build" && (co.FromCity == s.FromCity && co.ToCity == s.ToCity || co.FromCity == s.ToCity && co.ToCity == s.FromCity) {
			for _, a := range co.Alignments {
				if a.ID == s.AlignmentID {
					m.route(dst, c, a.Path, pal.Draft, pal.Outline, true)
				}
			}
		}
	}
	if len(f.Reports) > 0 {
		for index, run := range f.Reports[len(f.Reports)-1].Services {
			if run.Passengers == 0 {
				continue
			}
			for _, co := range f.Corridors {
				if co.ID != run.CorridorID {
					continue
				}
				for _, a := range co.Alignments {
					if a.ID != run.AlignmentID {
						continue
					}
					u := .5
					if s.Motion {
						u = (1 - math.Cos(tick/180+float64(index))) / 2
					}
					p := routePoint(c, a.Path, u)
					vector.DrawFilledRect(dst, float32(p.X-13), float32(p.Y-7), 26, 14, pal.Accent, true)
					vector.DrawFilledRect(dst, float32(p.X-8), float32(p.Y-4), 9, 5, pal.Surface, true)
					vector.DrawFilledCircle(dst, float32(p.X+7), float32(p.Y), 2, color.NRGBA{255, 194, 71, 255}, true)
				}
			}
		}
	}
	// Selected labels take precedence; remaining labels avoid collisions.
	cities := append([]gameapi.City{}, f.Cities...)
	sort.SliceStable(cities, func(i, j int) bool { return cities[i].ID == s.Selected && cities[j].ID != s.Selected })
	occupied := []labelRect{}
	for _, city := range cities {
		p := CityPoint(c, city)
		if p.X < 0 || p.Y < 0 || p.X > c.Width || p.Y > c.Height {
			continue
		}
		if city.ID == s.Selected {
			vector.DrawFilledCircle(dst, float32(p.X), float32(p.Y), 13, pal.Surface, true)
			vector.StrokeCircle(dst, float32(p.X), float32(p.Y), 13, 3, pal.Accent, true)
		}
		station, hasStation := ui.StationAt(f, city.ID)
		if !hasStation {
			vector.StrokeCircle(dst, float32(p.X), float32(p.Y), 4.5, 1.5, pal.Muted, true)
		} else if !station.Open {
			vector.DrawFilledRect(dst, float32(p.X-5), float32(p.Y-5), 10, 10, pal.Construction, true)
		} else {
			ink := pal.Accent
			if station.Public {
				ink = pal.Existing
			}
			vector.DrawFilledCircle(dst, float32(p.X), float32(p.Y), 6, ink, true)
		}
		vector.DrawFilledCircle(dst, float32(p.X), float32(p.Y), 3.5, pal.Surface, true)
		if s.View == "build" {
			tag := ""
			if city.ID == s.FromCity {
				tag = "A"
			}
			if city.ID == s.ToCity {
				tag = "B"
			}
			if tag != "" {
				vector.DrawFilledCircle(dst, float32(p.X), float32(p.Y), 12, pal.Accent, true)
				tagInk := pal.OnAction
				if ui.IsDark(s) {
					tagInk = pal.Outline
				}
				m.Label(dst, tag, p.X-4, p.Y-9, 13, tagInk)
			}
		}
		label := ui.Text(s.Locale, string(city.ID))
		width := text.Advance(label, &text.GoTextFace{Source: m.Font, Size: 15})
		for _, offset := range []mapview.Point{{X: 12, Y: -10}, {X: -width - 12, Y: -10}, {X: -width / 2, Y: 14}, {X: -width / 2, Y: -30}} {
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
	m.Label(dst, ui.Text(s.Locale, "east_asia"), 20, 18, 20, pal.Text)
	m.Label(dst, ui.Text(s.Locale, "stationLegend"), 20, 40, 11, pal.Muted)
}

func routePoint(c mapview.Camera, path []gameapi.GeoPoint, u float64) mapview.Point {
	count := len(path) - 1
	at := u * float64(count)
	i := min(count-1, int(at))
	v := at - float64(i)
	a, b := c.Screen(mapview.Project(path[i].Lat, path[i].Lon)), c.Screen(mapview.Project(path[i+1].Lat, path[i+1].Lon))
	return mapview.Point{X: a.X + (b.X-a.X)*v, Y: a.Y + (b.Y-a.Y)*v}
}
func (m *Map) route(dst *ebiten.Image, c mapview.Camera, path []gameapi.GeoPoint, stroke, outline color.Color, dashed bool) {
	for i := 0; i < len(path)-1; i++ {
		a, b := c.Screen(mapview.Project(path[i].Lat, path[i].Lon)), c.Screen(mapview.Project(path[i+1].Lat, path[i+1].Lon))
		n := max(1, int(math.Hypot(b.X-a.X, b.Y-a.Y)/6))
		for j := 0; j < n; j++ {
			if dashed && j%2 == 1 {
				continue
			}
			u, v := float64(j)/float64(n), float64(j+1)/float64(n)
			x1, y1, x2, y2 := float32(a.X+(b.X-a.X)*u), float32(a.Y+(b.Y-a.Y)*u), float32(a.X+(b.X-a.X)*v), float32(a.Y+(b.Y-a.Y)*v)
			vector.StrokeLine(dst, x1, y1, x2, y2, 5, outline, true)
			vector.StrokeLine(dst, x1, y1, x2, y2, 3, stroke, true)
		}
	}
}
