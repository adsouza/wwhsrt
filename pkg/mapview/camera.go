// Package mapview contains pure Equal Earth drawing/picking transforms.
package mapview

import "math"

type Point struct{ X, Y float64 }

// Regional terrain and short corridors need closer inspection on narrow screens.
const maxZoom = 32

func Project(lat, lon float64) Point {
	const a1, a2, a3, a4 = 1.340264, -0.081106, 0.000893, 0.003796
	p := lat * math.Pi / 180
	lambda := lon * math.Pi / 180
	t := math.Asin(math.Sqrt(3) / 2 * math.Sin(p))
	t2 := t * t
	t6 := t2 * t2 * t2
	return Point{2 * math.Sqrt(3) * lambda * math.Cos(t) / (3 * (a1 + 3*a2*t2 + t6*(7*a3+9*a4*t2))), t * (a1 + a2*t2 + t6*(a3+a4*t2))}
}

type Camera struct {
	Width, Height    float64
	Zoom, PanX, PanY float64
	Min, Max         Point
}

func Fit(w, h float64, points []Point) Camera {
	c := Camera{Width: w, Height: h, Zoom: 1, Min: points[0], Max: points[0]}
	for _, p := range points {
		c.Min.X = math.Min(c.Min.X, p.X)
		c.Min.Y = math.Min(c.Min.Y, p.Y)
		c.Max.X = math.Max(c.Max.X, p.X)
		c.Max.Y = math.Max(c.Max.Y, p.Y)
	}
	return c
}
func New(w, h float64) Camera {
	return Fit(w, h, []Point{Project(22.3193, 114.1694), Project(41.8057, 123.4315), Project(30.5728, 104.0668), Project(35.685, 139.7514)})
}
func (c Camera) scale() float64 {
	a, b := c.Min, c.Max
	return math.Min(math.Max(1, c.Width-72)/(b.X-a.X), math.Max(1, c.Height-80)/(b.Y-a.Y)) * c.Zoom
}
func (c Camera) Screen(p Point) Point {
	center := Point{(c.Min.X + c.Max.X) / 2, (c.Min.Y + c.Max.Y) / 2}
	s := c.scale()
	return Point{c.Width/2 + (p.X-center.X)*s + c.PanX, c.Height/2 - (p.Y-center.Y)*s + c.PanY}
}
func (c Camera) World(p Point) Point {
	center := Point{(c.Min.X + c.Max.X) / 2, (c.Min.Y + c.Max.Y) / 2}
	s := c.scale()
	return Point{center.X + (p.X-c.Width/2-c.PanX)/s, center.Y - (p.Y-c.Height/2-c.PanY)/s}
}

// Resize keeps the same geographical center and scale when an inspector or
// compact task changes the map's available rectangle. Reusing pixel pan offsets
// with a different fit scale can otherwise move a closely inspected route away.
func (c *Camera) Resize(w, h float64) {
	if w == c.Width && h == c.Height {
		return
	}
	// Hidden/initial one-pixel surfaces have no useful fit scale to preserve.
	if c.Zoom == 0 || c.Width <= 72 || c.Height <= 80 || w <= 72 || h <= 80 {
		c.Width, c.Height = w, h
		return
	}
	center, scale := c.World(Point{X: c.Width / 2, Y: c.Height / 2}), c.scale()
	c.Width, c.Height = w, h
	c.Zoom, c.PanX, c.PanY = 1, 0, 0
	c.Zoom = math.Max(1, math.Min(maxZoom, scale/c.scale()))
	at := c.Screen(center)
	c.PanX, c.PanY = w/2-at.X, h/2-at.Y
	c.Clamp()
}
func (c *Camera) ZoomAt(factor float64, at Point) {
	before := c.World(at)
	c.Zoom = math.Max(1, math.Min(maxZoom, c.Zoom*factor))
	after := c.Screen(before)
	c.PanX += at.X - after.X
	c.PanY += at.Y - after.Y
	c.Clamp()
}
func (c *Camera) Clamp() {
	c.PanX = math.Max(-c.Width*c.Zoom/2, math.Min(c.Width*c.Zoom/2, c.PanX))
	c.PanY = math.Max(-c.Height*c.Zoom/2, math.Min(c.Height*c.Zoom/2, c.PanY))
}
func (c *Camera) Focus(points []Point) {
	if len(points) < 2 {
		return
	}
	// Leave room below the alignment for the phone's wrapped map controls.
	bottomSpace := math.Min(130, c.Height*.3)
	fit := Fit(c.Width, c.Height-bottomSpace, points)
	base := *c
	base.Zoom = 1
	base.PanX = 0
	base.PanY = 0
	c.Zoom = math.Max(1, math.Min(maxZoom, fit.scale()/base.scale()))
	c.PanX = 0
	c.PanY = 0
	center := Point{X: (fit.Min.X + fit.Max.X) / 2, Y: (fit.Min.Y + fit.Max.Y) / 2}
	p := c.Screen(center)
	c.PanX = c.Width/2 - p.X
	c.PanY = c.Height/2 - p.Y - bottomSpace/2
	c.Clamp()
}
