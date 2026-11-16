package terrain

import (
	"math"
	"slices"
)

// Relief is a pyramid of 256x256 tiles on the Web Mercator grid. Level 8 is the finest.
const (
	MinLevel = 2
	MaxLevel = 8
	// CompleteLevel is the finest level that lists every tile holding land. Up to it, a quarter with no
	// tile is open sea; beyond it, relief exists only near lines and a missing tile means "use the parent".
	CompleteLevel = 5
	// TileSize is the width and height of a tile's image in pixels.
	TileSize = 256
	// OwnSize is how much of that image is the tile's own area; the rest is a border of its neighbours.
	OwnSize = 254
	border  = (TileSize - OwnSize) / 2
)

// Address names one tile: level, column from the date line eastward, row from the north.
type Address struct{ Z, X, Y int }

func (a Address) Parent() Address { return Address{a.Z - 1, a.X >> 1, a.Y >> 1} }

func (a Address) Children() [4]Address {
	z, x, y := a.Z+1, a.X<<1, a.Y<<1
	return [4]Address{{z, x, y}, {z, x + 1, y}, {z, x, y + 1}, {z, x + 1, y + 1}}
}

// Within reports whether a lies inside b, or is b.
func (a Address) Within(b Address) bool {
	d := a.Z - b.Z
	return d >= 0 && a.X>>d == b.X && a.Y>>d == b.Y
}

// Corner is the position of the point (u, v) of a tile, each from 0 to 1, west to east and north to
// south. The renderer then applies Equal Earth; the tiles' own projection is never a camera.
func Corner(a Address, u, v float64) (lat, lon float64) {
	n := math.Exp2(float64(a.Z))
	lon = (float64(a.X)+u)/n*360 - 180
	lat = math.Atan(math.Sinh(math.Pi*(1-2*(float64(a.Y)+v)/n))) * 180 / math.Pi
	return
}

// At is the tile of a level that holds a place. Places beyond the grid's edge belong to its last tile.
func At(z int, lat, lon float64) Address {
	n := math.Exp2(float64(z))
	x := math.Floor((lon + 180) / 360 * n)
	y := math.Floor((1 - math.Asinh(math.Tan(lat*math.Pi/180))/math.Pi) / 2 * n)
	return Address{z, int(math.Max(0, math.Min(n-1, x))), int(math.Max(0, math.Min(n-1, y)))}
}

// LevelFor is the level to draw when one degree of longitude at the view's centre is pixelsPerDegree
// wide: the coarsest level whose tiles are at most twice their stored width on screen. Tiles are then
// drawn at one to two times their size, except that level 8 is enlarged without limit.
func LevelFor(pixelsPerDegree float64) int {
	if pixelsPerDegree <= 0 {
		return MinLevel
	}
	z := int(math.Ceil(math.Log2(pixelsPerDegree * 360 / (2 * TileSize))))
	return min(MaxLevel, max(MinLevel, z))
}

// Piece draws Area using the part of Source's image that covers it. Source is Area or a tile that holds it.
type Piece struct{ Area, Source Address }

// Rect is the square of Source's image that covers Area, in pixels: its left and top edge and its size.
//
// A tile's image is TileSize pixels across but only the middle OwnSize are its own area; the outermost
// pixel on each side lies in the neighbouring tile. Tiles side by side therefore hold the same ground
// along their shared edge, and blending between pixel centres from 1 to 255 leaves no seam and never
// reads outside the image.
func (p Piece) Rect() (left, top, size float64) {
	shift := p.Area.Z - p.Source.Z
	size = OwnSize / float64(int(1)<<shift)
	return border + float64(p.Area.X-p.Source.X<<shift)*size, border + float64(p.Area.Y-p.Source.Y<<shift)*size, size
}

// Cover lists what to draw at a level and which tiles the view wants loaded, coarsest first.
//
// Every area of land is drawn once. A tile at the level, or one with nothing finer beneath it, is drawn
// whole. Where only some of a tile's quarters have finer tiles, the others are drawn as quarters of it,
// unless the quarter is open sea (see CompleteLevel). A sea tile draws nothing of its own. Each area comes from the tile that should supply it when that is ready and
// otherwise from the finest tile above it that is, so a view sharpens as tiles arrive; with none ready
// the area is left to the plain land beneath.
func (s *Set) Cover(level int, visible, ready func(Address) bool) (pieces []Piece, wanted []Address) {
	want := map[Address]bool{}
	emit := func(area, supplier Address) {
		want[supplier] = true
		for a := supplier; a.Z >= MinLevel; a = a.Parent() {
			tile, ok := s.Tiles[a]
			if !ok || tile.Kind == Sea {
				continue
			}
			if a.Z == MinLevel {
				want[a] = true // the coarsest tile is what shows while finer ones load
			}
			if ready(a) {
				pieces = append(pieces, Piece{area, a})
				break
			}
		}
	}
	var walk func(Address)
	walk = func(t Address) {
		if !visible(t) {
			return
		}
		// A sea tile has nothing of its own to draw, but a finer tile beneath it may resolve an island.
		land := s.Tiles[t].Kind != Sea
		children, finer := t.Children(), false
		for _, c := range children {
			if _, ok := s.Tiles[c]; ok {
				finer = true
			}
		}
		if t.Z >= level || !finer {
			if land {
				emit(t, t)
			}
			return
		}
		for _, c := range children {
			if !visible(c) {
				continue
			}
			if _, ok := s.Tiles[c]; ok {
				walk(c)
			} else if land && c.Z > CompleteLevel {
				emit(c, t)
			}
		}
	}
	n := 1 << MinLevel
	for x := range n {
		for y := range n {
			if t := (Address{MinLevel, x, y}); s.has(t) {
				walk(t)
			}
		}
	}
	for a := range want {
		wanted = append(wanted, a)
	}
	slices.SortFunc(wanted, func(a, b Address) int {
		if a.Z != b.Z {
			return a.Z - b.Z
		}
		if a.X != b.X {
			return a.X - b.X
		}
		return a.Y - b.Y
	})
	return pieces, wanted
}

func (s *Set) has(a Address) bool { _, ok := s.Tiles[a]; return ok }
