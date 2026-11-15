package render

import (
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
	chains      [][]mapview.Point
	Stops       []publicStop
	trackLayer  layer
	camera      mapview.Camera
	color       color.NRGBA
	marks       *markers
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
	if scene.SourceSHA != assets.PublicNetworkSHA256() || len(scene.Lines) != len(scene.CorridorIDs) {
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
	p.chains = publicTrackChains(p.tracks)
	return p, nil
}
func (p *publicLayer) draw(dst *ebiten.Image, c mapview.Camera, s ui.State) {
	pal := ui.PaletteFor(s)
	if p.trackLayer.view == nil || p.camera != c || p.color != pal.Existing {
		img := p.trackLayer.reset(int(c.Width), int(c.Height))
		p.camera = c
		p.color = pal.Existing
		var path vector.Path
		for _, chain := range p.chains {
			for _, run := range screenRuns(c, chain) {
				for i, q := range simplifyScreen(run, screenTolerance) {
					if i == 0 {
						path.MoveTo(float32(q.X), float32(q.Y))
					} else {
						path.LineTo(float32(q.X), float32(q.Y))
					}
				}
			}
		}
		op := &vector.DrawPathOptions{AntiAlias: true}
		op.ColorScale.ScaleWithColor(pal.Existing)
		vector.StrokePath(img, &path, &vector.StrokeOptions{Width: 2}, op)
		p.drawStops(img, c, pal.Existing)
	}
	dst.DrawImage(p.trackLayer.view, nil)
}

func (p *publicLayer) drawStops(dst *ebiten.Image, c mapview.Camera, ink color.NRGBA) {
	// Hollow stops go on top, as they did when each kind was one path.
	for _, hollow := range []bool{false, true} {
		for _, stop := range p.Stops {
			if stop.MapGap != hollow {
				continue
			}
			q := c.Screen(mapview.Project(stop.Lat, stop.Lon))
			if q.X < 0 || q.Y < 0 || q.X > c.Width || q.Y > c.Height {
				continue
			}
			if c.Zoom < 2 && !stop.Tagged && stop.Operator == "" {
				continue
			}
			if hollow {
				p.marks.ring(dst, float32(q.X), float32(q.Y), 2.5, 1, ink)
			} else {
				p.marks.disc(dst, float32(q.X), float32(q.Y), 2, ink)
			}
		}
	}
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
