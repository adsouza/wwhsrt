// Package app is the Ebitengine composition shared by native and browser hosts.
package app

import (
	"math"
	"sync"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/inpututil"
	"wwhsrt/pkg/gameapi"
	"wwhsrt/pkg/mapview"
	"wwhsrt/pkg/render"
	"wwhsrt/pkg/ui"
)

type Game struct {
	mapOrigin      mapview.Point
	mu             sync.Mutex
	Port           gameapi.Game
	Frame          gameapi.Frame
	State          ui.State
	Map            *render.Map
	Camera         mapview.Camera
	width, height  int
	tick           float64
	replayTick     float64
	dirty          bool
	drag           bool
	moved          bool
	start, last    mapview.Point
	touch          ebiten.TouchID
	touching       bool
	pinch          float64
	serviceSearch  *serviceSearch
	serviceRanking ui.ServiceRanking
	host           host
}

func New(port gameapi.Game) (*Game, error) {
	m, e := render.New()
	if e != nil {
		return nil, e
	}
	g := &Game{
		Port:   port,
		Frame:  port.Snapshot(),
		Map:    m,
		Camera: m.Camera(1, 1, false),
		dirty:  true,
		State:  ui.State{Locale: "en", View: "map", Motion: true},
	}
	g.host.init(g)
	return g, nil
}
func (g *Game) Update() error {
	g.mu.Lock()
	defer g.mu.Unlock()
	if g.host.windowUpdate(g) {
		return ebiten.Termination
	}
	g.host.update(g)
	g.tick++
	if g.State.Watching && g.State.ReplayPlaying && g.State.Motion {
		g.replayTick++
	}
	if g.host.blocksMap() {
		g.drag, g.touching = false, false
		g.pinch = 0
		return nil
	}
	x, y := ebiten.CursorPosition()
	at := g.mapLocal(float64(x), float64(y))
	inside := at.X >= 0 && at.Y >= 0 && at.X < g.Camera.Width && at.Y < g.Camera.Height
	if inside {
		_, wheel := ebiten.Wheel()
		if wheel != 0 {
			g.Camera.ZoomAt(math.Exp(wheel*.12), at)
			g.host.persist(g)
		}
	}
	if inpututil.IsMouseButtonJustPressed(ebiten.MouseButtonLeft) && inside {
		g.drag = true
		g.moved = false
		g.start = at
		g.last = at
	}
	if g.drag && ebiten.IsMouseButtonPressed(ebiten.MouseButtonLeft) {
		g.move(at)
	}
	if g.drag && inpututil.IsMouseButtonJustReleased(ebiten.MouseButtonLeft) {
		g.finish(at)
	}
	ids := ebiten.AppendTouchIDs(nil)
	if len(ids) >= 2 {
		x1, y1 := ebiten.TouchPosition(ids[0])
		x2, y2 := ebiten.TouchPosition(ids[1])
		a, b := g.mapLocal(float64(x1), float64(y1)), g.mapLocal(float64(x2), float64(y2))
		insidePoint := func(p mapview.Point) bool {
			return p.X >= 0 && p.Y >= 0 && p.X < g.Camera.Width && p.Y < g.Camera.Height
		}
		if !insidePoint(a) || !insidePoint(b) {
			g.pinch = 0
			g.touching = false
			g.drag = false
			return nil
		}
		dist := math.Hypot(float64(x1-x2), float64(y1-y2))
		if g.pinch > 0 {
			g.Camera.ZoomAt(dist/g.pinch, g.mapLocal(float64(x1+x2)/2, float64(y1+y2)/2))
		}
		g.pinch = dist
		g.moved = true
		g.touching = false
		g.drag = false
	} else {
		if g.pinch > 0 {
			g.host.persist(g)
			g.pinch = 0
			g.moved = true
			g.touching = false
			g.drag = false
			return nil
		}
		if len(ids) == 1 && !g.touching {
			g.touch = ids[0]
			tx, ty := ebiten.TouchPosition(g.touch)
			p := g.mapLocal(float64(tx), float64(ty))
			if p.X >= 0 && p.Y >= 0 && p.X < g.Camera.Width && p.Y < g.Camera.Height {
				g.start = p
				g.last = p
				g.moved = false
				g.touching = true
			}
		}
		if g.touching {
			if inpututil.IsTouchJustReleased(g.touch) {
				g.finish(g.last)
				g.touching = false
			} else {
				tx, ty := ebiten.TouchPosition(g.touch)
				g.move(g.mapLocal(float64(tx), float64(ty)))
			}
		}
	}
	return nil
}

func (g *Game) newGamePresentation() {
	g.discardServiceSearch()
	g.serviceRanking = ui.ServiceRanking{}
	g.State = ui.NewGameState(g.State, g.Frame)
	g.Camera = g.regionCamera(g.State.Portrait)
	g.drag, g.touching, g.moved = false, false, false
	g.pinch, g.tick = 0, 0
	g.replayTick = 0
	g.dirty = true
}
func (g *Game) mapLocal(x, y float64) mapview.Point {
	return mapview.Point{X: x - g.mapOrigin.X, Y: y - g.mapOrigin.Y}
}
func (g *Game) move(at mapview.Point) {
	if math.Hypot(at.X-g.start.X, at.Y-g.start.Y) > 6 {
		g.moved = true
	}
	if g.moved {
		g.Camera.PanX += at.X - g.last.X
		g.Camera.PanY += at.Y - g.last.Y
		g.Camera.Clamp()
	}
	g.last = at
}
func (g *Game) finish(at mapview.Point) {
	if !g.moved {
		if g.State.View == "map" || g.State.View == "build" {
			if id := render.PickProspect(
				g.Camera,
				g.Frame,
				g.State,
				at,
			); id != "" &&
				ui.SelectProspect(&g.State, g.Frame, id) {
				g.dirty = true
				g.host.layout(g)
				g.host.changed(g)
				g.drag = false
				g.host.persist(g)
				return
			}
		}
		if g.State.Watching && g.State.View != "build" {
			if id := render.PickTrain(g.Camera, g.Frame, g.replayTime(), at); id != "" {
				g.State.SelectedTrain = id
				g.dirty = true
				g.host.changed(g)
				g.drag = false
				return
			}
		}
		id := render.Pick(g.Camera, g.Frame.Cities, at)
		stop := ""
		if g.Frame.PublicNetwork && g.State.View != "build" {
			stop = g.Map.PickPublicStop(g.Camera, at)
		}
		// Marker centers remain inspectable; a tap on track between markers
		// selects service planning instead of requiring a route-list search.
		onMarker := false
		for _, city := range g.Frame.Cities {
			if city.ID == id {
				q := render.CityPoint(g.Camera, city)
				onMarker = math.Hypot(q.X-at.X, q.Y-at.Y) <= 10
				break
			}
		}
		if stop != "" {
			q := g.Map.PublicStop(stop)
			if q != nil {
				point := g.Camera.Screen(mapview.Project(q.Lat, q.Lon))
				onMarker = onMarker || math.Hypot(point.X-at.X, point.Y-at.Y) <= 8
			}
		}
		if !onMarker && (g.State.View == "map" || g.State.View == "services" || g.State.View == "build") {
			if ids := render.PickServiceSections(g.Camera, g.Frame, at); ui.SelectMapService(&g.State, g.Frame, ids) {
				g.dirty = true
				g.host.layout(g)
				g.host.changed(g)
				g.drag = false
				g.host.persist(g)
				return
			}
		}
		if stop != "" {
			source := g.Map.PublicStop(stop)
			// A city hub wins if the source marker is the same station.
			same := false
			for _, city := range g.Frame.Cities {
				if city.ID == id && city.StationName == source.Name {
					same = true
				}
			}
			if !same {
				g.State.SourceStation = stop
				g.State.Selected = ""
				g.dirty = true
				g.host.changed(g)
				g.drag = false
				g.host.persist(g)
				return
			}
		}
		if id != "" {
			ui.Select(&g.State, id)
			g.dirty = true
			g.host.changed(g)
		}
	}
	g.drag = false
	g.host.persist(g)
}

// Empty id frames all idle lines; a row action frames just that named line.
func (g *Game) focusIdle(id string) {
	points := []mapview.Point{}
	for _, line := range ui.IdleHighlights(g.Frame) {
		if id != "" && line.ReviewID != id {
			continue
		}
		for _, p := range line.Path {
			points = append(points, mapview.Project(p.Lat, p.Lon))
		}
	}
	if len(points) > 1 {
		g.Camera.Focus(points)
	}
}

func (g *Game) focusSelected() {
	id := g.State.ServiceCorridor
	if g.State.View == "build" {
		if co, ok := ui.Pair(g.Frame, g.State.FromCity, g.State.ToCity); ok {
			id = co.ID
		}
	}
	if g.State.View != "build" {
		if route, ok := ui.ServiceRoute(g.Frame, id); ok {
			points := []mapview.Point{}
			for _, section := range route.Sections {
				for _, co := range g.Frame.Corridors {
					if co.ID == section {
						a := co.Alignments[0]
						for _, p := range g.Frame.Projects {
							if p.CorridorID == section {
								for _, candidate := range co.Alignments {
									if candidate.ID == p.AlignmentID {
										a = candidate
									}
								}
							}
						}
						for _, p := range a.Path {
							points = append(points, mapview.Project(p.Lat, p.Lon))
						}
					}
				}
			}
			if len(points) > 0 {
				g.Camera.Focus(points)
				return
			}
		}
	}
	for _, co := range g.Frame.Corridors {
		if co.ID != id {
			continue
		}
		a := co.Alignments[0]
		for _, al := range co.Alignments {
			if al.ID == g.State.AlignmentID {
				a = al
			}
		}
		paths := [][]gameapi.GeoPoint{a.Path}
		if g.State.View == "build" {
			paths = ui.MapForProspect(g.Frame, co, a).Paths
		}
		points := []mapview.Point{}
		for _, path := range paths {
			for _, p := range path {
				points = append(points, mapview.Project(p.Lat, p.Lon))
			}
		}
		g.Camera.Focus(points)
	}
}
func (g *Game) replayTime() float64 {
	return math.Mod(float64(g.State.ReplayMinute)+g.replayTick/6, 1440)
}
func (g *Game) Draw(screen *ebiten.Image) { g.mu.Lock(); defer g.mu.Unlock(); g.host.draw(g, screen) }
func (g *Game) Layout(w, h int) (int, int) {
	g.mu.Lock()
	defer g.mu.Unlock()
	if g.width != w || g.height != h {
		g.width = w
		g.height = h
		g.dirty = true
		g.host.layout(g)
	}
	return w, h
}

// Preserve an explicitly chosen region when rotating or starting over.
func (g *Game) regionCamera(portrait bool) mapview.Camera {
	return g.cameraForRegion(g.State.Region, portrait)
}
func (g *Game) cameraForRegion(id string, portrait bool) mapview.Camera {
	if id != "" && id != "east-asia" {
		return mapview.RegionCamera(g.Camera.Width, g.Camera.Height, id, portrait)
	}
	return g.Map.Camera(g.Camera.Width, g.Camera.Height, portrait)
}
