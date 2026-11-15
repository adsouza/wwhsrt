package render

import (
	"image"
	"math"
	"sync/atomic"
	"testing"
	"time"

	"github.com/hajimehoshi/ebiten/v2"
	"wwhsrt/pkg/gameapi"
	"wwhsrt/pkg/mapview"
	"wwhsrt/pkg/terrain"
	"wwhsrt/pkg/ui"
)

// reliefMap is a map whose tiles are decoded for real but never sent to a GPU.
func reliefMap(t *testing.T) *Map {
	t.Helper()
	m, err := New()
	if err != nil {
		t.Fatal(err)
	}
	m.terrain.cache.close()
	freed := &atomic.Int32{}
	m.terrain.cache = newTileCache(tileCap, m.terrain.decode, func(*image.RGBA) texture { return fakeTexture{freed} })
	t.Cleanup(m.terrain.cache.close)
	return m
}

func bogotaMedellin(w, h float64) mapview.Camera {
	return mapview.Fit(w, h, []mapview.Point{mapview.Project(4.711, -74.072), mapview.Project(6.244, -75.581)})
}

func wholeWorld(w, h float64) mapview.Camera {
	return mapview.Fit(w, h, []mapview.Point{mapview.Project(70, -170), mapview.Project(-55, 175)})
}

func TestBuildingTheMapDecodesNoTile(t *testing.T) {
	m, err := New()
	if err != nil {
		t.Fatal(err)
	}
	defer m.terrain.cache.close()
	if m.terrain.cache.len() != 0 || len(m.terrain.cache.queue) != 0 || len(m.terrain.cache.pending) != 0 {
		t.Fatal("a map that has drawn nothing has loaded or asked for tiles")
	}
	if len(m.terrain.set.Tiles) == 0 {
		t.Fatal("the map knows of no tiles")
	}
}

func TestReliefLevelFollowsTheZoom(t *testing.T) {
	m := reliefMap(t)
	if level := m.terrain.level(wholeWorld(1440, 960)); level != 2 {
		t.Errorf("the whole world at 1440 px is drawn from level %d, want 2", level)
	}
	if level := m.terrain.level(bogotaMedellin(1440, 960)); level != 8 {
		t.Errorf("Bogota-Medellin filling the window is drawn from level %d, want 8", level)
	}
	last := 0
	for zoom := 1.0; zoom <= 32; zoom *= 1.3 { // zooming in never coarsens
		c := wholeWorld(1440, 960)
		c.Zoom = zoom
		level := m.terrain.level(c)
		if level < last {
			t.Errorf("zoom %.1f gives level %d after level %d", zoom, level, last)
		}
		last = level
	}
}

func TestPieceMeshUsesOnlyItsPartOfTheSource(t *testing.T) {
	m := reliefMap(t)
	c := bogotaMedellin(1440, 960)
	area := terrain.Address{Z: 8, X: 227, Y: 93}
	for _, want := range []struct {
		source         terrain.Address
		x0, x1, y0, y1 float32
	}{
		{area, 0, 256, 0, 256},
		{terrain.Address{Z: 7, X: 113, Y: 46}, 128, 256, 128, 256},
		{terrain.Address{Z: 6, X: 56, Y: 23}, 192, 256, 64, 128},
	} {
		vertices := m.terrain.mesh(c, terrain.Piece{Area: area, Source: want.source}, 1, false)
		if len(vertices) != 81 {
			t.Fatalf("a piece has %d vertices, want 81", len(vertices))
		}
		x0, x1, y0, y1 := float32(math.Inf(1)), float32(math.Inf(-1)), float32(math.Inf(1)), float32(math.Inf(-1))
		for _, v := range vertices {
			x0, x1, y0, y1 = min(x0, v.SrcX), max(x1, v.SrcX), min(y0, v.SrcY), max(y1, v.SrcY)
		}
		// Half a pixel is kept back at the image's own edge, where there is nothing beyond to blend with.
		inset := func(v float32) float32 { return min(255.5, max(0.5, v)) }
		if x0 != inset(want.x0) || x1 != inset(want.x1) || y0 != inset(want.y0) || y1 != inset(want.y1) {
			t.Errorf("from %+v the piece reads x %v to %v, y %v to %v; want x %v to %v, y %v to %v",
				want.source, x0, x1, y0, y1, inset(want.x0), inset(want.x1), inset(want.y0), inset(want.y1))
		}
	}
}

func TestNeighbouringPiecesShareTheirEdgeVertices(t *testing.T) {
	m := reliefMap(t)
	c := bogotaMedellin(1440, 960)
	west := append([]vertexAt{}, corners(m.terrain.mesh(c, piece(8, 74, 123), 1, false))...)
	east := corners(m.terrain.mesh(c, piece(8, 75, 123), 1, false))
	for row := range 9 {
		if a, b := west[row*9+8], east[row*9]; a != b {
			t.Fatalf("row %d: the west tile's east edge is at %v and the east tile's west edge at %v", row, a, b)
		}
	}
	// A coarse piece beside fine ones: its corner is exactly the fine pieces' corner, so no gap opens.
	coarse := corners(m.terrain.mesh(c, piece(7, 37, 62), 1, false)) // holds 8/74..75/124..125
	fine := corners(m.terrain.mesh(c, piece(8, 74, 123), 1, false))
	if coarse[0] != fine[8*9] {
		t.Fatalf("a level-7 piece's corner %v is not the level-8 corner %v beside it", coarse[0], fine[8*9])
	}
}

type vertexAt struct{ x, y float32 }

func piece(z, x, y int) terrain.Piece {
	a := terrain.Address{Z: z, X: x, Y: y}
	return terrain.Piece{Area: a, Source: a}
}

func corners(vertices []ebiten.Vertex) []vertexAt {
	out := make([]vertexAt, len(vertices))
	for i, v := range vertices {
		out[i] = vertexAt{v.DstX, v.DstY}
	}
	return out
}

func TestReliefIsDimmedAtWideViewsAndTintedInTheDark(t *testing.T) {
	m := reliefMap(t)
	c := bogotaMedellin(1440, 960)
	c.Zoom = 2
	light := m.terrain.mesh(c, piece(8, 74, 123), reliefOpacity(c), false)[0]
	if want := float32(0.42 + 0.075*2); math.Abs(float64(light.ColorA-want)) > 1e-6 || light.ColorR != 1 {
		t.Errorf("at zoom 2 relief has opacity %v and red %v, want %v and 1", light.ColorA, light.ColorR, want)
	}
	dark := m.terrain.mesh(c, piece(8, 74, 123), reliefOpacity(c), true)[0]
	if dark.ColorR != .30 || dark.ColorG != .38 || dark.ColorB != .40 {
		t.Errorf("dark relief is tinted %v, %v, %v", dark.ColorR, dark.ColorG, dark.ColorB)
	}
	c.Zoom = 30
	if got := reliefOpacity(c); got != .98 {
		t.Errorf("relief opacity at zoom 30 is %v, want its ceiling of 0.98", got)
	}
}

func TestOnlyTilesInViewAreAskedFor(t *testing.T) {
	m := reliefMap(t)
	c := bogotaMedellin(1440, 960)
	pieces, wanted := m.terrain.cover(c)
	if len(pieces) != 0 {
		t.Fatalf("%d pieces are drawn before any tile has loaded", len(pieces))
	}
	if len(wanted) == 0 || len(wanted) > 40 {
		t.Fatalf("a view of one route wants %d tiles", len(wanted))
	}
	for _, a := range wanted {
		if !m.terrain.visible(c, a) {
			t.Errorf("%+v is wanted but is not in view", a)
		}
	}
	if !m.terrain.visible(c, terrain.Address{Z: 8, X: 74, Y: 123}) || m.terrain.visible(c, terrain.Address{Z: 8, X: 227, Y: 93}) {
		t.Error("visibility is wrong for a tile under the route or for one in Japan")
	}
}

func TestALateTileChangesTheSceneKey(t *testing.T) {
	m := reliefMap(t)
	c, f, s := bogotaMedellin(1440, 960), gameapi.Frame{Checkpoint: "x"}, ui.State{Locale: "en"}
	m.terrain.tick(false)
	before := m.sceneKeyFor(c, f, s)
	_, wanted := m.terrain.cover(c)
	m.terrain.cache.want(wanted)
	for deadline := time.Now().Add(5 * time.Second); m.sceneKeyFor(c, f, s) == before; {
		if time.Now().After(deadline) {
			t.Fatal("tiles arrived, or should have, and the scene key never changed")
		}
		m.terrain.tick(false)
		time.Sleep(time.Millisecond)
	}
	// Once everything has arrived the key settles, so the scene is not redrawn for ever.
	for deadline := time.Now().Add(5 * time.Second); m.terrain.cache.len() < len(wanted); {
		if time.Now().After(deadline) {
			t.Fatalf("%d of %d tiles arrived", m.terrain.cache.len(), len(wanted))
		}
		m.terrain.tick(false)
		time.Sleep(time.Millisecond)
	}
	settled := m.sceneKeyFor(c, f, s)
	m.terrain.tick(false)
	if m.sceneKeyFor(c, f, s) != settled {
		t.Fatal("the scene key keeps changing after the last tile arrived")
	}
	// With the tiles in, the same view is drawn from them, each area once and nothing left wanted.
	pieces, _ := m.terrain.cover(c)
	if len(pieces) == 0 {
		t.Fatal("no pieces are drawn after the tiles arrived")
	}
	for _, p := range pieces {
		// An area with no tile of its own is drawn from its parent; every other area from its own tile.
		if _, listed := m.terrain.set.Tiles[p.Area]; listed && p.Source != p.Area {
			t.Errorf("%+v is still drawn from a coarser tile", p)
		}
	}
}

func TestHiddenTerrainLoadsNothing(t *testing.T) {
	m := reliefMap(t)
	c, f := bogotaMedellin(1440, 960), gameapi.Frame{Checkpoint: "x"}
	shown, hidden := ui.State{Locale: "en"}, ui.State{Locale: "en", HideTerrain: true}
	_, wanted := m.terrain.cover(c)
	m.terrain.cache.want(wanted)
	m.terrain.tick(true) // the player hides terrain while tiles are still wanted
	if len(m.terrain.cache.queue) != 0 {
		t.Fatal("tiles are still queued after terrain was hidden")
	}
	key := m.sceneKeyFor(c, f, hidden)
	time.Sleep(20 * time.Millisecond) // anything already in hand may finish
	for range 5 {
		m.terrain.tick(true)
	}
	if m.sceneKeyFor(c, f, hidden) != key {
		t.Fatal("the scene is redrawn for tiles while terrain is hidden")
	}
	if m.terrain.cache.len() != 0 {
		t.Fatal("tiles were taken in while terrain was hidden")
	}
	if m.sceneKeyFor(c, f, shown) == m.sceneKeyFor(c, f, hidden) {
		t.Fatal("hiding terrain does not change the scene")
	}
}

func TestTerrainLegendShowsTheNewTop(t *testing.T) {
	m := reliefMap(t)
	if got := m.terrainLegend(mapview.New(1440, 960), "en").labels[1].text; got != "4800+ m" {
		t.Fatalf("the legend's upper caption is %q", got)
	}
	if len(terrainLegendColours) != 5 || terrainLegendColours[3].R != 166 || terrainLegendColours[4].B != 240 {
		t.Fatalf("the legend's swatches are %v", terrainLegendColours)
	}
}

func TestJapanKeepsItsContours(t *testing.T) {
	m := reliefMap(t)
	if len(m.terrain.contours) != 2 || len(m.terrain.contours[0].lines) < 20 {
		t.Fatal("Japan's contour lines are missing")
	}
	kansai := mapview.Fit(1440, 960, []mapview.Point{mapview.Project(35.65, 136.07), mapview.Project(34.70, 135.50)})
	if !m.terrain.contours[1].onScreen(kansai) || m.terrain.contours[0].onScreen(kansai) {
		t.Fatal("a view of Kansai should hold Kansai's contours and not Hokkaido's")
	}
}
