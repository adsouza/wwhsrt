package render

import (
	"image/color"
	"math"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/vector"
)

// markerPhases is how many sub-pixel positions each marker sprite is drawn at
// along each axis, so a marker lands within 1/8 px of where it was asked for.
const markerPhases = 4

type markerKey struct {
	radius, width float32 // width 0 fills the disc
	phaseX        int
	phaseY        int
}

// markers draws the map's circular dots and rings from small cached sprites.
//
// Ebitengine's antialiased FillCircle and StrokeCircle build a curved path and
// fill it in eight passes: about 300 vertices per dot at any radius, plus a
// stencil region. A world map draws thousands of them per scene (cities,
// stations, prospect dots), which made dots the second-largest share of the
// vertex buffer Ebitengine keeps for the life of the process. A tinted sprite
// costs four vertices and batches into one draw.
type markers struct {
	sprites map[markerKey]*ebiten.Image
}

func (m *markers) disc(dst *ebiten.Image, x, y, r float32, clr color.Color) {
	m.draw(dst, x, y, r, 0, clr)
}

func (m *markers) ring(dst *ebiten.Image, x, y, r, width float32, clr color.Color) {
	m.draw(dst, x, y, r, width, clr)
}

func (m *markers) draw(dst *ebiten.Image, x, y, r, width float32, clr color.Color) {
	px, phaseX := markerPlace(x)
	py, phaseY := markerPlace(y)
	key := markerKey{radius: r, width: width, phaseX: phaseX, phaseY: phaseY}
	sprite, pad := m.sprite(key)
	op := &ebiten.DrawImageOptions{}
	op.GeoM.Translate(float64(px-pad), float64(py-pad))
	op.ColorScale.ScaleWithColor(clr)
	dst.DrawImage(sprite, op)
}

// sprite returns the white marker for key and the offset of its centre's pixel.
func (m *markers) sprite(key markerKey) (*ebiten.Image, int) {
	pad := int(math.Ceil(float64(key.radius+key.width))) + 2
	if s, ok := m.sprites[key]; ok {
		return s, pad
	}
	if m.sprites == nil {
		m.sprites = map[markerKey]*ebiten.Image{}
	}
	s := ebiten.NewImage(2*pad+1, 2*pad+1)
	cx := float32(pad) + float32(key.phaseX)/markerPhases
	cy := float32(pad) + float32(key.phaseY)/markerPhases
	if key.width == 0 {
		vector.FillCircle(s, cx, cy, key.radius, color.White, true)
	} else {
		vector.StrokeCircle(s, cx, cy, key.radius, key.width, color.White, true)
	}
	m.sprites[key] = s
	return s, pad
}

// markerPlace splits a coordinate into a whole pixel and the nearest sub-pixel
// phase, so that pixel + phase/markerPhases is within 1/(2*markerPhases) of v.
func markerPlace(v float32) (pixel, phase int) {
	q := int(math.Floor(float64(v)*markerPhases + .5))
	pixel = q / markerPhases
	if q%markerPhases < 0 {
		pixel--
	}
	return pixel, q - pixel*markerPhases
}
