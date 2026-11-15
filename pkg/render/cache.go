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

// sceneKeyFor adds what the map itself holds: the text scale decides every label's size and position.
func (m *Map) sceneKeyFor(c mapview.Camera, f gameapi.Frame, s ui.State) sceneKey {
	key := mapSceneKey(c, f, s)
	key.Text = m.Text
	return key
}

// The scene is immutable between state/camera changes. Representative trains
// are drawn separately so animation never rerasterizes thousands of stations.
func (m *Map) Draw(dst *ebiten.Image, c mapview.Camera, f gameapi.Frame, s ui.State, minute float64) {
	key := m.sceneKeyFor(c, f, s)
	if m.scene == nil || m.sceneKey != key || f.Checkpoint == "" {
		m.scene = layerImage(m.scene, int(c.Width), int(c.Height))
		m.sceneKey = key
		m.drawScene(m.scene, c, f, s)
	}
	dst.DrawImage(m.scene, nil)
	m.drawDay(dst, c, f, s, minute)
}

// layerImage returns img cleared for reuse at w x h, or a new image when the
// size changed.
//
// The map's window-sized layers are redrawn on every pan and zoom. Ebitengine
// packs a managed image into a shared atlas page, and small images (glyphs,
// marker sprites) then fill the strip beside it. Freeing the layer leaves that
// page held by the small images, so the next layer took a new 34 MB page: two
// dozen of them after a minute at 3737x2029. Reusing the image avoids the
// churn, and an unmanaged image has a texture of its own that is freed with it.
func layerImage(img *ebiten.Image, w, h int) *ebiten.Image {
	w, h = max(1, w), max(1, h)
	if img != nil && img.Bounds().Dx() == w && img.Bounds().Dy() == h {
		img.Clear()
		return img
	}
	if img != nil {
		img.Deallocate()
	}
	return ebiten.NewImageWithOptions(image.Rect(0, 0, w, h), &ebiten.NewImageOptions{Unmanaged: true})
}
