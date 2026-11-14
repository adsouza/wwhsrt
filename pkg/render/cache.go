package render

import (
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
}

func mapSceneKey(c mapview.Camera, f gameapi.Frame, s ui.State) sceneKey {
	return sceneKey{Camera: c, Checkpoint: f.Checkpoint, Locale: s.Locale, View: s.View, Region: s.Region, SourceStation: s.SourceStation, Service: s.ServiceCorridor, Alignment: s.AlignmentID, Selected: s.Selected, From: s.FromCity, To: s.ToCity, Dark: ui.IsDark(s), HideTerrain: s.HideTerrain, Portrait: s.Portrait}
}

// The scene is immutable between state/camera changes. Representative trains
// are drawn separately so animation never rerasterizes thousands of stations.
func (m *Map) Draw(dst *ebiten.Image, c mapview.Camera, f gameapi.Frame, s ui.State, minute float64) {
	key := mapSceneKey(c, f, s)
	if m.scene == nil || m.sceneKey != key || f.Checkpoint == "" {
		if m.scene != nil {
			m.scene.Dispose()
		}
		m.scene = ebiten.NewImage(max(1, int(c.Width)), max(1, int(c.Height)))
		m.sceneKey = key
		m.drawScene(m.scene, c, f, s)
	}
	dst.DrawImage(m.scene, nil)
	m.drawDay(dst, c, f, s, minute)
}
