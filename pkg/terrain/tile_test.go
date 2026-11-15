package terrain

import (
	"math"
	"slices"
	"testing"
)

func all(Address) bool  { return true }
func none(Address) bool { return false }

// family is a level-2 tile, one of its level-3 children, and one level-4 grandchild under that child.
func family() (*Set, Address, Address, Address) {
	top, child, grandchild := Address{2, 1, 1}, Address{3, 2, 2}, Address{4, 5, 5}
	return &Set{Tiles: map[Address]Tile{top: {Kind: Land}, child: {Kind: Coast}, grandchild: {Kind: Land}}}, top, child, grandchild
}

// cells counts how many pieces cover each level-4 cell under the level-2 tile {2, 1, 1}, which holds
// cells x 4 to 7 and y 4 to 7.
func cells(t *testing.T, pieces []Piece) map[[2]int]int {
	t.Helper()
	count := map[[2]int]int{}
	for _, p := range pieces {
		if !p.Area.Within(p.Source) {
			t.Fatalf("piece %+v draws an area from a tile that does not hold it", p)
		}
		if p.Area.Z > 4 {
			t.Fatalf("piece %+v is finer than the level asked for", p)
		}
		d := 4 - p.Area.Z
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
	for x := 4; x < 8; x++ {
		for y := 4; y < 8; y++ {
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
		t.Errorf("pieces reach outside the level-2 tile: %v", count)
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

func TestCoverDrawsEveryAreaOnce(t *testing.T) {
	s, top, child, grandchild := family()
	pieces, wanted := s.Cover(4, all, all)
	wantEachCellOnce(t, cells(t, pieces))
	for _, tile := range []Address{top, child, grandchild} {
		if !slices.Contains(wanted, tile) {
			t.Errorf("%+v supplies part of the view but is not wanted", tile)
		}
	}
	if !slices.Contains(pieces, Piece{grandchild, grandchild}) {
		t.Error("the finest tile is not drawn from itself")
	}
	// Three quarters of the child have no finer tile: each is drawn from the child, not from the top.
	if !slices.Contains(pieces, Piece{Address{4, 4, 4}, child}) {
		t.Errorf("a quarter with no tile of its own is not drawn from its parent: %+v", pieces)
	}
	// At a coarser level the finer tiles are not used.
	pieces, wanted = s.Cover(3, all, all)
	if slices.Contains(wanted, grandchild) || len(pieces) != 4 {
		t.Errorf("level 3 gave %+v wanting %+v", pieces, wanted)
	}
	pieces, _ = s.Cover(2, all, all)
	if !slices.Equal(pieces, []Piece{{top, top}}) {
		t.Errorf("level 2 gave %+v", pieces)
	}
}

func TestCoverFallsBackToTheFinestReadyAncestor(t *testing.T) {
	s, top, child, grandchild := family()
	pieces, wanted := s.Cover(4, all, func(a Address) bool { return a == top })
	wantEachCellOnce(t, cells(t, pieces))
	for _, p := range pieces {
		if p.Source != top {
			t.Errorf("%+v is drawn from a tile that is not ready", p)
		}
	}
	if !slices.Contains(wanted, grandchild) || !slices.Contains(wanted, child) {
		t.Errorf("tiles that are not ready are not asked for: %+v", wanted)
	}
	pieces, _ = s.Cover(4, all, func(a Address) bool { return a != grandchild })
	wantEachCellOnce(t, cells(t, pieces))
	if !slices.Contains(pieces, Piece{grandchild, child}) {
		t.Errorf("the missing tile's area is not drawn from its parent: %+v", pieces)
	}
	pieces, wanted = s.Cover(4, all, none)
	if len(pieces) != 0 {
		t.Errorf("drew %+v with nothing ready", pieces)
	}
	if !slices.Equal(wanted, []Address{top, child, grandchild}) {
		t.Errorf("wanted %+v, want every tile with the coarsest first", wanted)
	}
}

func TestSeaTilesCountAsCoverageAndDrawNothing(t *testing.T) {
	s, top, child, _ := family()
	sea := Address{4, 4, 5}
	s.Tiles[sea] = Tile{Kind: Sea}
	pieces, wanted := s.Cover(4, all, all)
	wantEachCellOnce(t, cells(t, pieces), [2]int{4, 5})
	if slices.Contains(wanted, sea) {
		t.Error("a sea tile has nothing to load but is wanted")
	}
	// A whole sea branch draws nothing beneath it either.
	s.Tiles[child] = Tile{Kind: Sea}
	count := cells(t, func() []Piece { p, _ := s.Cover(4, all, all); return p }())
	for x := 4; x < 6; x++ {
		for y := 4; y < 6; y++ {
			if count[[2]int{x, y}] != 0 {
				t.Errorf("cell %d,%d under a sea tile is drawn", x, y)
			}
		}
	}
	_ = top
}

func TestCoverSkipsWhatIsNotVisible(t *testing.T) {
	s, top, child, grandchild := family()
	pieces, wanted := s.Cover(4, func(a Address) bool { return a.Within(top) && !a.Within(child) || a == top }, all)
	for _, p := range pieces {
		if p.Area.Within(child) {
			t.Errorf("%+v is off screen but drawn", p)
		}
	}
	if slices.Contains(wanted, grandchild) || slices.Contains(wanted, child) {
		t.Errorf("tiles off screen are wanted: %+v", wanted)
	}
	if pieces, wanted = s.Cover(4, none, all); len(pieces) != 0 || len(wanted) != 0 {
		t.Errorf("an empty view gave %+v, %+v", pieces, wanted)
	}
}

func TestCoverAlsoAsksForTheCoarsestTile(t *testing.T) {
	// The level-2 tile shows first while finer ones load, so it is wanted even when none of it is drawn whole.
	s, top, _, _ := family()
	for _, c := range top.Children() {
		s.Tiles[c] = Tile{Kind: Land}
	}
	_, wanted := s.Cover(3, all, none)
	if len(wanted) == 0 || wanted[0] != top {
		t.Errorf("wanted %+v, want the level-2 tile first", wanted)
	}
}
