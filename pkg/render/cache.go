package render

import (
	"image"

	"github.com/hajimehoshi/ebiten/v2"
	"wwhsrt/pkg/gameapi"
	"wwhsrt/pkg/mapview"
	"wwhsrt/pkg/ui"
)

type sceneKey struct {
	Camera                                                              mapview.Camera
	Checkpoint, Locale, View, Region, SourceStation, Service, Alignment string
	Selected, From, To                                                  gameapi.CityID
	Dark, HideTerrain, Portrait                                         bool
	Text                                                                ui.TextScale
	// Terrain counts relief tiles' arrivals while terrain is shown, so a tile that loads late redraws the scene.
	Terrain uint64
}

func mapSceneKey(c mapview.Camera, f gameapi.Frame, s ui.State) sceneKey {
	return sceneKey{
		Camera:        c,
		Checkpoint:    f.Checkpoint,
		Locale:        s.Locale,
		View:          s.View,
		Region:        s.Region,
		SourceStation: s.SourceStation,
		Service:       s.ServiceCorridor,
		Alignment:     s.AlignmentID,
		Selected:      s.Selected,
		From:          s.FromCity,
		To:            s.ToCity,
		Dark:          ui.IsDark(s),
		HideTerrain:   s.HideTerrain,
		Portrait:      s.Portrait,
	}
}

// sceneKeyFor adds what the map itself holds: the text scale decides every label's size and position,
// and relief tiles arrive after the view that asked for them.
func (m *Map) sceneKeyFor(c mapview.Camera, f gameapi.Frame, s ui.State) sceneKey {
	key := mapSceneKey(c, f, s)
	key.Text = m.Text
	if !s.HideTerrain && m.terrain != nil {
		key.Terrain = m.terrain.arrivals
	}
	return key
}

// The scene is immutable between state/camera changes. Representative trains
// are drawn separately so animation never rerasterizes thousands of stations.
func (m *Map) Draw(dst *ebiten.Image, c mapview.Camera, f gameapi.Frame, s ui.State, minute float64) {
	if m.terrain != nil {
		m.terrain.tick(s.HideTerrain)
	}
	key := m.sceneKeyFor(c, f, s)
	if m.scene.view == nil || m.sceneKey != key || f.Checkpoint == "" {
		m.sceneKey = key
		m.drawScene(m.scene.reset(int(c.Width), int(c.Height)), c, f, s)
	}
	dst.DrawImage(m.scene.view, nil)
	m.drawDay(dst, c, f, s, minute)
	m.drawIdleOverlay(dst, c, f, s)
}

// layer is one of the map's window-sized images, kept between redraws.
//
// The map's layers are redrawn on every pan and zoom. Ebitengine packs a
// managed image into a shared atlas page, and small images (glyphs, marker
// sprites) then fill the strip beside it. Freeing the layer leaves that page
// held by the small images, so the next layer took a new 34 MB page: two dozen
// of them after a minute at 3737x2029. Reusing the image avoids the churn, and
// an unmanaged image has a texture of its own that is freed with it.
//
// The image behind the layer is a whole number of 16 px blocks wide and tall.
// Ebitengine rounds each antialiased path's box up to 16 px but caps its
// stencil buffer at twice the destination's width. A path spanning a
// 3032 px map needs 6080 px of a buffer capped at 6064, so Ebitengine freed
// and reallocated the 54 MB buffer on every redraw, and the pages the old
// buffers sat on stayed pinned. Drawing into a view of a 3040 px image lifts
// the cap to 6080.
type layer struct {
	backing *ebiten.Image
	view    *ebiten.Image // backing cut to the size asked for
}

// reset returns the layer's image cleared at w x h, allocating a new one only
// when the size changed.
func (l *layer) reset(w, h int) *ebiten.Image {
	w, h = max(1, w), max(1, h)
	if l.view != nil && l.view.Bounds().Dx() == w && l.view.Bounds().Dy() == h {
		l.view.Clear()
		return l.view
	}
	if l.backing != nil {
		l.backing.Deallocate()
	}
	l.backing = ebiten.NewImageWithOptions(image.Rect(0, 0, roundUp16(w), roundUp16(h)), &ebiten.NewImageOptions{Unmanaged: true})
	l.view = l.backing.SubImage(image.Rect(0, 0, w, h)).(*ebiten.Image)
	return l.view
}

func roundUp16(n int) int { return (n + 15) &^ 15 }
