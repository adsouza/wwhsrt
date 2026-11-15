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
	return Point{
		2 * math.Sqrt(3) * lambda * math.Cos(t) / (3 * (a1 + 3*a2*t2 + t6*(7*a3+9*a4*t2))),
		t * (a1 + a2*t2 + t6*(a3+a4*t2)),
	}
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

// Clearances for FitAvoiding, in pixels. When it pans, anchors stay
// AnchorClearance inside the screen, enough for a city's widest marker (the
// 13 px selection ring and its stroke), and avoided points end ExcludeClearance
// outside it: the radius of a plain city dot, so the dot lies wholly off
// screen. The renderer already skips any city whose centre is off screen,
// ring and label included, so that is the most that could ever show.
const (
	AnchorClearance  = 16.0
	ExcludeClearance = 6.0
	// avoidSlack absorbs rounding, so a cleared point counts as cleared.
	avoidSlack = 1e-6
)

// Clear reports whether screen point q lies at least ExcludeClearance beyond an edge.
func (c Camera) Clear(q Point) bool {
	m := ExcludeClearance - avoidSlack
	return q.X <= -m || q.X >= c.Width+m || q.Y <= -m || q.Y >= c.Height+m
}

// FitAvoiding fits points like Fit, then pans each avoided point at least
// ExcludeClearance beyond an edge of the screen, taking the smallest pan that
// keeps every fitted point AnchorClearance inside. Only the pan changes, never
// the scale; a point already outside, or one no such pan can clear, leaves the
// camera as it is. Fit's own padding is wider, so AnchorClearance only decides
// whether a pan is possible, never how far it goes.
func FitAvoiding(w, h float64, points, avoid []Point) Camera {
	c := Fit(w, h, points)
	for _, a := range avoid {
		c.panClear(points, a)
	}
	return c
}

func (c *Camera) panClear(points []Point, avoid Point) {
	a := c.Screen(avoid)
	if c.Clear(a) {
		return
	}
	lo, hi := Point{X: math.Inf(-1), Y: math.Inf(-1)}, Point{X: math.Inf(1), Y: math.Inf(1)}
	for _, p := range points {
		q := c.Screen(p)
		lo.X, hi.X = math.Max(lo.X, AnchorClearance-q.X-avoidSlack), math.Min(hi.X, c.Width-AnchorClearance-q.X+avoidSlack)
		lo.Y, hi.Y = math.Max(lo.Y, AnchorClearance-q.Y-avoidSlack), math.Min(hi.Y, c.Height-AnchorClearance-q.Y+avoidSlack)
	}
	best, dx, dy := math.Inf(1), 0.0, 0.0
	for _, m := range []struct{ dx, dy float64 }{
		{dx: -ExcludeClearance - a.X}, {dx: c.Width + ExcludeClearance - a.X},
		{dy: -ExcludeClearance - a.Y}, {dy: c.Height + ExcludeClearance - a.Y},
	} {
		if m.dx < lo.X || m.dx > hi.X || m.dy < lo.Y || m.dy > hi.Y {
			continue
		}
		if size := math.Abs(m.dx) + math.Abs(m.dy); size < best {
			best, dx, dy = size, m.dx, m.dy
		}
	}
	c.PanX += dx
	c.PanY += dy
}

// New frames the built-in default region: landscape East Asia, from Harbin in
// the north, with Dhaka kept off screen (see assets/scenario/camera.json).
func New(w, h float64) Camera {
	return FitAvoiding(
		w,
		h,
		[]Point{
			Project(22.3193, 114.1694),
			Project(45.75, 126.65),
			Project(30.5728, 104.0668),
			Project(35.685, 139.7514),
		},
		[]Point{Project(23.8103, 90.4125)},
	)
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

// Clamp keeps the view inside the area that dragging can show at zoom 1: the fitted frame and half a
// screen beyond it on each side. The limit grows with zoom so that the same area stays reachable;
// a limit of half a screen per zoom step would let a place be seen from afar but not approached.
func (c *Camera) Clamp() {
	reach := math.Max(1, c.Zoom) - 0.5
	c.PanX = math.Max(-c.Width*reach, math.Min(c.Width*reach, c.PanX))
	c.PanY = math.Max(-c.Height*reach, math.Min(c.Height*reach, c.PanY))
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
