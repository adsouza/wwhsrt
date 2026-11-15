package terrain

import (
	"math"
	"slices"
	"testing"
)

func all(Address) bool  { return true }
func none(Address) bool { return false }

// family is a level-5 tile, one of its level-6 children, one level-7 grandchild under that child, and
// the coarse tiles above them. Levels 6 and up exist only near lines, so most of the level-5 tile has
// nothing finer.
func family() (s *Set, coarsest, top, child, grandchild Address) {
	coarsest, top, child, grandchild = Address{2, 1, 1}, Address{5, 8, 8}, Address{6, 16, 16}, Address{7, 33, 33}
	s = &Set{Tiles: map[Address]Tile{
		coarsest: {Kind: Land}, {3, 2, 2}: {Kind: Land}, {4, 4, 4}: {Kind: Land},
		top: {Kind: Land}, child: {Kind: Coast}, grandchild: {Kind: Land},
	}}
	return
}

// cells counts how many pieces cover each level-7 cell under the level-5 tile {5, 8, 8}, which holds
// cells x 32 to 35 and y 32 to 35.
func cells(t *testing.T, pieces []Piece) map[[2]int]int {
	t.Helper()
	count := map[[2]int]int{}
	for _, p := range pieces {
		if !p.Area.Within(p.Source) {
			t.Fatalf("piece %+v draws an area from a tile that does not hold it", p)
		}
		if p.Area.Z > 7 {
			t.Fatalf("piece %+v is finer than the level asked for", p)
		}
		d := 7 - p.Area.Z
		for x := p.Area.X << d; x < (p.Area.X+1)<<d; x++ {
			for y := p.Area.Y << d; y < (p.Area.Y+1)<<d; y++ {
				count[[2]int{x, y}]++
			}
		}
	}
	return count
}

func wantEachCellOnce(t *testing.T, count map[[2]int]int, except ...[2]int) {
	t.Helper()
	for x := 32; x < 36; x++ {
		for y := 32; y < 36; y++ {
			want := 1
			if slices.Contains(except, [2]int{x, y}) {
				want = 0
			}
			if count[[2]int{x, y}] != want {
				t.Errorf("cell %d,%d is drawn %d times, want %d", x, y, count[[2]int{x, y}], want)
			}
		}
	}
	if len(count) > 16 {
		t.Errorf("pieces reach outside the level-5 tile: %v", count)
	}
}

func TestLevelForKeepsTilesBetweenOneAndTwoTimesTheirSize(t *testing.T) {
	for _, c := range []struct {
		pixelsPerDegree float64
		want            int
	}{{0.5, 2}, {512.0 / 90, 2}, {512.0/90 + 0.01, 3}, {300, 8}, {5000, 8}, {0, 2}} {
		if got := LevelFor(c.pixelsPerDegree); got != c.want {
			t.Errorf("LevelFor(%v) = %d, want %d", c.pixelsPerDegree, got, c.want)
		}
	}
	// At every scale between the ends, a tile of the chosen level is more than 256 and at most 512 pixels wide.
	for ppd := 6.0; ppd < 360; ppd *= 1.07 {
		z := LevelFor(ppd)
		if width := ppd * 360 / math.Exp2(float64(z)); width <= 256 || width > 512 {
			t.Errorf("at %.1f pixels per degree level %d tiles are %.0f pixels wide", ppd, z, width)
		}
	}
}

func TestCornerMatchesWebMercator(t *testing.T) {
	lat, lon := Corner(Address{8, 227, 93}, 0, 0) // the north-west corner of the old Hokkaido image
	if math.Abs(lon-139.21875) > 1e-8 || math.Abs(lat-44.087585) > 1e-5 {
		t.Fatal("tile origin misplaced", lat, lon)
	}
	lat2, lon2 := Corner(Address{8, 227, 93}, 1, 1)
	nextLat, nextLon := Corner(Address{8, 228, 94}, 0, 0)
	if lat2 >= lat || lon2 <= lon || lat2 != nextLat || lon2 != nextLon {
		t.Fatal("a tile's far corner is not its neighbour's near corner", lat2, lon2, nextLat, nextLon)
	}
}

func TestAddressFamily(t *testing.T) {
	a := Address{8, 227, 93}
	if a.Parent() != (Address{7, 113, 46}) {
		t.Fatal("wrong parent", a.Parent())
	}
	if siblings := a.Parent().Children(); !slices.Contains(siblings[:], a) {
		t.Fatal("a tile is not among its parent's children")
	}
	if !a.Within(a) || !a.Within(Address{6, 56, 23}) || a.Within(Address{6, 56, 22}) || a.Parent().Within(a) {
		t.Fatal("Within is wrong")
	}
}

// A tile's image is 256 pixels across but only the middle 254 are its own area; the outermost pixel
// on each side belongs to its neighbour, so a piece is read from 1 to 255.
func TestAPieceReadsItsOwnPartOfTheSource(t *testing.T) {
	area := Address{8, 227, 93}
	for _, c := range []struct {
		source          Address
		left, top, size float64
	}{
		{area, 1, 1, 254},
		{Address{7, 113, 46}, 128, 128, 127},    // the south-east quarter of its parent
		{Address{6, 56, 23}, 191.5, 64.5, 63.5}, // fourth column, second row of sixteenths
	} {
		left, top, size := Piece{area, c.source}.Rect()
		if left != c.left || top != c.top || size != c.size {
			t.Errorf("from %+v: left %v top %v size %v, want %v %v %v", c.source, left, top, size, c.left, c.top, c.size)
		}
	}
}

func TestCoverDrawsEveryAreaOnce(t *testing.T) {
	s, coarsest, top, child, grandchild := family()
	pieces, wanted := s.Cover(7, all, all)
	wantEachCellOnce(t, cells(t, pieces))
	for _, tile := range []Address{top, child, grandchild} {
		if !slices.Contains(wanted, tile) {
			t.Errorf("%+v supplies part of the view but is not wanted", tile)
		}
	}
	if slices.Contains(wanted, coarsest) {
		t.Error("the coarsest tile is wanted although everything finer is ready")
	}
	if !slices.Contains(pieces, Piece{grandchild, grandchild}) {
		t.Error("the finest tile is not drawn from itself")
	}
	// Three quarters of the child have no finer tile: each is drawn from the child, not from the top.
	if !slices.Contains(pieces, Piece{Address{7, 32, 32}, child}) {
		t.Errorf("a quarter with no tile of its own is not drawn from its parent: %+v", pieces)
	}
	// Three quarters of the top have no level-6 tile: each is drawn as a quarter of the top.
	if !slices.Contains(pieces, Piece{Address{6, 17, 17}, top}) {
		t.Errorf("a quarter of the level-5 tile is not drawn from it: %+v", pieces)
	}
	// At a coarser level the finer tiles are not used.
	pieces, wanted = s.Cover(6, all, all)
	if slices.Contains(wanted, grandchild) || len(pieces) != 4 {
		t.Errorf("level 6 gave %+v wanting %+v", pieces, wanted)
	}
	pieces, _ = s.Cover(5, all, all)
	if !slices.Equal(pieces, []Piece{{top, top}}) {
		t.Errorf("level 5 gave %+v", pieces)
	}
	pieces, _ = s.Cover(2, all, all)
	if !slices.Equal(pieces, []Piece{{coarsest, coarsest}}) {
		t.Errorf("level 2 gave %+v", pieces)
	}
}

// Levels 2 to 5 list every tile that holds land. A quarter with no tile there is open sea: nothing is
// drawn for it, not a blank quarter of its parent.
func TestCoverLeavesOpenSeaUndrawn(t *testing.T) {
	s, _, _, _, _ := family()
	pieces, _ := s.Cover(5, all, all)
	if len(pieces) != 1 {
		t.Fatalf("one land tile at level 5 gave %d pieces: %+v", len(pieces), pieces)
	}
	pieces, _ = s.Cover(3, all, all)
	if !slices.Equal(pieces, []Piece{{Address{3, 2, 2}, Address{3, 2, 2}}}) {
		t.Fatalf("level 3 gave %+v", pieces)
	}
}

func TestCoverFallsBackToTheFinestReadyAncestor(t *testing.T) {
	s, coarsest, top, child, grandchild := family()
	pieces, wanted := s.Cover(7, all, func(a Address) bool { return a == coarsest })
	wantEachCellOnce(t, cells(t, pieces))
	for _, p := range pieces {
		if p.Source != coarsest {
			t.Errorf("%+v is drawn from a tile that is not ready", p)
		}
	}
	if !slices.Contains(wanted, grandchild) || !slices.Contains(wanted, child) || !slices.Contains(wanted, top) {
		t.Errorf("tiles that are not ready are not asked for: %+v", wanted)
	}
	pieces, _ = s.Cover(7, all, func(a Address) bool { return a != grandchild })
	wantEachCellOnce(t, cells(t, pieces))
	if !slices.Contains(pieces, Piece{grandchild, child}) {
		t.Errorf("the missing tile's area is not drawn from its parent: %+v", pieces)
	}
	pieces, wanted = s.Cover(7, all, none)
	if len(pieces) != 0 {
		t.Errorf("drew %+v with nothing ready", pieces)
	}
	if !slices.Equal(wanted, []Address{coarsest, top, child, grandchild}) {
		t.Errorf("wanted %+v, want the coarsest tile and the three suppliers, coarsest first", wanted)
	}
}

func TestSeaTilesCountAsCoverageAndDrawNothing(t *testing.T) {
	s, _, _, child, _ := family()
	sea := Address{7, 32, 33}
	s.Tiles[sea] = Tile{Kind: Sea}
	pieces, wanted := s.Cover(7, all, all)
	wantEachCellOnce(t, cells(t, pieces), [2]int{32, 33})
	if slices.Contains(wanted, sea) {
		t.Error("a sea tile has nothing to load but is wanted")
	}
	// A whole sea branch draws nothing beneath it either.
	s.Tiles[child] = Tile{Kind: Sea}
	pieces, _ = s.Cover(7, all, all)
	count := cells(t, pieces)
	for x := 32; x < 34; x++ {
		for y := 32; y < 34; y++ {
			if count[[2]int{x, y}] != 0 {
				t.Errorf("cell %d,%d under a sea tile is drawn", x, y)
			}
		}
	}
}

func TestCoverSkipsWhatIsNotVisible(t *testing.T) {
	s, _, top, child, grandchild := family()
	pieces, wanted := s.Cover(7, func(a Address) bool { return !a.Within(child) }, all)
	for _, p := range pieces {
		if p.Area.Within(child) {
			t.Errorf("%+v is off screen but drawn", p)
		}
	}
	if len(pieces) != 3 {
		t.Errorf("three quarters of the level-5 tile are in view but %d pieces are drawn", len(pieces))
	}
	if slices.Contains(wanted, grandchild) || slices.Contains(wanted, child) || !slices.Contains(wanted, top) {
		t.Errorf("wanted %+v: tiles off screen must not be asked for, and the tile in view must be", wanted)
	}
	if pieces, wanted = s.Cover(7, none, all); len(pieces) != 0 || len(wanted) != 0 {
		t.Errorf("an empty view gave %+v, %+v", pieces, wanted)
	}
}
