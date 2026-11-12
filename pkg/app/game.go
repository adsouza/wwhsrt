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
	mu            sync.Mutex
	Port          gameapi.Game
	Frame         gameapi.Frame
	State         ui.State
	Map           *render.Map
	Camera        mapview.Camera
	width, height int
	tick          float64
	dirty         bool
	drag          bool
	moved         bool
	start, last   mapview.Point
	touch         ebiten.TouchID
	touching      bool
	pinch         float64
	host          host
}

func New(port gameapi.Game) (*Game, error) {
	m, e := render.New()
	if e != nil {
		return nil, e
	}
	g := &Game{Port: port, Frame: port.Snapshot(), Map: m, Camera: m.Camera(1, 1, false), dirty: true, State: ui.State{Locale: "en", View: "map", Motion: true}}
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
	if g.host.blocksMap() {
		return nil
	}
	x, y := ebiten.CursorPosition()
	at := mapview.Point{X: float64(x), Y: float64(y)}
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
		dist := math.Hypot(float64(x1-x2), float64(y1-y2))
		if g.pinch > 0 {
			g.Camera.ZoomAt(dist/g.pinch, mapview.Point{X: float64(x1+x2) / 2, Y: float64(y1+y2) / 2})
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
			p := mapview.Point{X: float64(tx), Y: float64(ty)}
			if p.X < g.Camera.Width && p.Y < g.Camera.Height {
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
				g.move(mapview.Point{X: float64(tx), Y: float64(ty)})
			}
		}
	}
	return nil
}

func (g *Game) newGamePresentation() {
	g.State = ui.NewGameState(g.State, g.Frame)
	g.Camera = g.Map.Camera(g.Camera.Width, g.Camera.Height, g.State.Portrait)
	g.drag, g.touching, g.moved = false, false, false
	g.pinch, g.tick = 0, 0
	g.dirty = true
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
		id := render.Pick(g.Camera, g.Frame.Cities, at)
		if id != "" {
			ui.Select(&g.State, id)
			g.dirty = true
			g.host.changed(g)
		}
	}
	g.drag = false
	g.host.persist(g)
}
func (g *Game) focusSelected() {
	id := g.State.ServiceCorridor
	if g.State.View == "build" {
		if co, ok := ui.Pair(g.Frame, g.State.FromCity, g.State.ToCity); ok {
			id = co.ID
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
		points := []mapview.Point{}
		for _, p := range a.Path {
			points = append(points, mapview.Project(p.Lat, p.Lon))
		}
		g.Camera.Focus(points)
	}
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
