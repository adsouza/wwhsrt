package render

import (
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/vector"
	"image/color"
	"math"
	"wwhsrt/assets"
	"wwhsrt/pkg/mapview"
	"wwhsrt/pkg/ui"
)

type publicStop struct {
	ID, Name string
	Lon, Lat float64
	Tagged   bool
	Operator string
	MapGap   bool `json:"map_gap"`
}
type publicLayer struct {
	CorridorIDs []string
	Lines       [][]mapview.Point
	tracks      []railSegment
	Stops       []publicStop
	land        [][][2]float64
	image       *ebiten.Image
	camera      mapview.Camera
	landImage   *ebiten.Image
	landCamera  mapview.Camera
	landColor   color.NRGBA
	color       color.NRGBA
}

func newPublicLayer() (*publicLayer, error) {
	var raw struct {
		Stations []publicStop
	}
	if err := json.Unmarshal(assets.PublicNetwork, &raw); err != nil {
		return nil, err
	}
	var scene struct {
		SourceSHA   string         `json:"source_catalog_sha256"`
		CorridorIDs []string       `json:"corridor_ids"`
		Lines       [][][2]float64 `json:"lines"`
		StationIDs  []string       `json:"station_ids"`
	}
	if err := json.Unmarshal(assets.PlayableRailMap, &scene); err != nil {
		return nil, err
	}
	if scene.SourceSHA != fmt.Sprintf("%x", sha256.Sum256(assets.PublicNetwork)) || len(scene.Lines) != len(scene.CorridorIDs) {
		return nil, fmt.Errorf("playable rail map does not match its catalog")
	}
	allowed := map[string]bool{}
	for _, id := range scene.StationIDs {
		allowed[id] = true
	}
	p := &publicLayer{CorridorIDs: scene.CorridorIDs}
	for _, stop := range raw.Stations {
		if allowed[stop.ID] {
			p.Stops = append(p.Stops, stop)
		}
	}
	if len(p.Stops) != len(scene.StationIDs) {
		return nil, fmt.Errorf("playable map references unknown or duplicate stations")
	}
	for _, line := range scene.Lines {
		points := []mapview.Point{}
		for _, v := range line {
			points = append(points, mapview.Project(v[1], v[0]))
		}
		p.Lines = append(p.Lines, points)
	}
	p.tracks = publicTrackSegments(p.Lines)
	data, _ := assets.Terrain.ReadFile("terrain/world-land.json")
	if err := json.Unmarshal(data, &p.land); err != nil {
		return nil, err
	}
	return p, nil
}
func (p *publicLayer) drawLand(dst *ebiten.Image, c mapview.Camera, s ui.State) {
	pal := ui.PaletteFor(s)
	if p.landImage != nil && p.landCamera == c && p.landColor == pal.Land {
		dst.DrawImage(p.landImage, nil)
		return
	}
	if p.landImage != nil {
		p.landImage.Dispose()
	}
	p.landImage = ebiten.NewImage(max(1, int(c.Width)), max(1, int(c.Height)))
	p.landCamera = c
	p.landColor = pal.Land
	target := p.landImage
	for _, ring := range p.land {
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
		vector.FillPath(target, &path, nil, op)
		op = &vector.DrawPathOptions{AntiAlias: true}
		op.ColorScale.ScaleWithColor(pal.Coast)
		vector.StrokePath(target, &path, &vector.StrokeOptions{Width: 1}, op)
	}
	dst.DrawImage(p.landImage, nil)
}
func (p *publicLayer) draw(dst *ebiten.Image, c mapview.Camera, s ui.State) {
	pal := ui.PaletteFor(s)
	if p.image == nil || p.camera != c || p.color != pal.Existing {
		if p.image != nil {
			p.image.Dispose()
		}
		p.image = ebiten.NewImage(max(1, int(c.Width)), max(1, int(c.Height)))
		p.camera = c
		p.color = pal.Existing
		var path vector.Path
		for _, edge := range p.tracks {
			a, b, ok := screenSegment(c, c.Screen(edge[0]), c.Screen(edge[1]))
			if !ok {
				continue
			}
			path.MoveTo(float32(a.X), float32(a.Y))
			path.LineTo(float32(b.X), float32(b.Y))
		}
		op := &vector.DrawPathOptions{AntiAlias: true}
		op.ColorScale.ScaleWithColor(pal.Existing)
		vector.StrokePath(p.image, &path, &vector.StrokeOptions{Width: 2}, op)
		p.drawStops(p.image, c, pal.Existing)
	}
	dst.DrawImage(p.image, nil)
}

func (p *publicLayer) drawStops(dst *ebiten.Image, c mapview.Camera, ink color.NRGBA) {
	var filled, hollow vector.Path
	for _, stop := range p.Stops {
		q := c.Screen(mapview.Project(stop.Lat, stop.Lon))
		if q.X < 0 || q.Y < 0 || q.X > c.Width || q.Y > c.Height {
			continue
		}
		if c.Zoom < 2 && !stop.Tagged && stop.Operator == "" {
			continue
		}
		radius := float32(2)
		path := &filled
		if stop.MapGap {
			radius = 2.5
			path = &hollow
		}
		path.MoveTo(float32(q.X)+radius, float32(q.Y))
		path.Arc(float32(q.X), float32(q.Y), radius, 0, 2*math.Pi, vector.Clockwise)
		path.Close()
	}
	op := &vector.DrawPathOptions{AntiAlias: true}
	op.ColorScale.ScaleWithColor(ink)
	vector.FillPath(dst, &filled, nil, op)
	vector.StrokePath(dst, &hollow, &vector.StrokeOptions{Width: 1}, op)
}

// Source stops are inspectable cartographic features. Strategic city stations
// remain the endpoints used by service and demand rules.
func (m *Map) PickPublicStop(c mapview.Camera, at mapview.Point) string {
	best := 12.0
	id := ""
	for _, s := range m.public.Stops {
		if c.Zoom < 2 && !s.Tagged && s.Operator == "" {
			continue
		}
		q := c.Screen(mapview.Project(s.Lat, s.Lon))
		if q.X < 0 || q.Y < 0 || q.X > c.Width || q.Y > c.Height {
			continue
		}
		d := math.Hypot(q.X-at.X, q.Y-at.Y)
		if d < best {
			best = d
			id = s.ID
		}
	}
	return id
}
func (m *Map) PublicStop(id string) *publicStop {
	if id == "" || m.public == nil {
		return nil
	}
	for _, s := range m.public.Stops {
		if s.ID == id {
			return &s
		}
	}
	return nil
}
