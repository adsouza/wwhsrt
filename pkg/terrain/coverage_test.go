package terrain

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"io/fs"
	"math"
	"strings"
	"testing"

	"wwhsrt/assets"
)

// Each line of the game is a path of longitude and latitude. A line is every path of the existing
// network, every alignment of a buildable prospect, and the alignments of the two older buildable routes.
type gameLine struct {
	name string
	path [][2]float64
}

func gameLines(t *testing.T) []gameLine {
	t.Helper()
	var lines []gameLine
	var existing struct {
		Lines [][][2]float64 `json:"lines"`
	}
	if err := json.Unmarshal(assets.PlayableRailMap, &existing); err != nil {
		t.Fatal(err)
	}
	for _, path := range existing.Lines {
		lines = append(lines, gameLine{"an existing line", path})
	}
	type corridor struct {
		ID         string `json:"id"`
		Existing   bool   `json:"existing"`
		PartOf     string `json:"part_of"`
		Alignments []struct {
			ID   string `json:"id"`
			Path []struct {
				Lat float64 `json:"lat"`
				Lon float64 `json:"lon"`
			} `json:"path"`
		} `json:"alignments"`
	}
	add := func(corridors []corridor) {
		for _, c := range corridors {
			if c.Existing || c.PartOf != "" {
				continue
			}
			for _, a := range c.Alignments {
				line := gameLine{name: c.ID + " / " + a.ID}
				for _, p := range a.Path {
					line.path = append(line.path, [2]float64{p.Lon, p.Lat})
				}
				lines = append(lines, line)
			}
		}
	}
	var network struct {
		Corridors []corridor `json:"corridors"`
	}
	if err := json.Unmarshal(assets.PublicNetwork, &network); err != nil {
		t.Fatal(err)
	}
	before := len(lines)
	add(network.Corridors)
	if before < 290 || len(lines)-before < 101 {
		t.Fatal("Missing public tracks or construction alignments", before, len(lines)-before)
	}
	return lines
}

func kmBetween(a, b [2]float64) float64 {
	rad := math.Pi / 180
	v := math.Pow(math.Sin((b[1]-a[1])*rad/2), 2) +
		math.Cos(a[1]*rad)*math.Cos(b[1]*rad)*math.Pow(math.Sin((b[0]-a[0])*rad/2), 2)
	return 2 * 6371 * math.Asin(math.Min(1, math.Sqrt(v)))
}

func committed(t *testing.T) *Set {
	t.Helper()
	s, err := Load(assets.Terrain)
	if err != nil {
		t.Fatal(err)
	}
	return s
}

func TestAtFindsTheTileHoldingAPlace(t *testing.T) {
	if got := At(8, 35.68, 139.76); got != (Address{8, 227, 100}) { // Tokyo
		t.Fatal(got)
	}
	lat, lon := Corner(Address{6, 20, 30}, 0.5, 0.5)
	if got := At(6, lat, lon); got != (Address{6, 20, 30}) {
		t.Fatal(got)
	}
	if got := At(3, 89, 180); got != (Address{3, 7, 0}) { // past the grid's edge: the last tile
		t.Fatal(got)
	}
}

// Adding or moving a line without regenerating the tiles fails here.
func TestEveryLineLiesOnLevel8Tiles(t *testing.T) {
	s := committed(t)
	for _, line := range gameLines(t) {
		for i := 1; i < len(line.path); i++ {
			a, b := line.path[i-1], line.path[i]
			steps := int(math.Ceil(kmBetween(a, b))) // one point per kilometre
			for step := 0; step <= steps; step++ {
				f := float64(step) / math.Max(1, float64(steps))
				lon, lat := a[0]+(b[0]-a[0])*f, a[1]+(b[1]-a[1])*f
				if at := At(MaxLevel, lat, lon); !s.has(at) {
					t.Fatalf("%s is not covered at %.4f N, %.4f E (tile %v): run tools/prepare-terrain.py --fetch and --write",
						line.name, lat, lon, at)
				}
			}
		}
	}
}

func TestFineTilesHaveTheirParents(t *testing.T) {
	s := committed(t)
	counts := map[int]int{}
	for a := range s.Tiles {
		counts[a.Z]++
		if a.Z > CompleteLevel+1 && !s.has(a.Parent()) {
			t.Errorf("%v has no parent tile", a)
		}
	}
	// Detail falls away from a line in steps: levels 7 and 6 each reach beyond the parents of the level
	// above. Without those rings sharp relief would end against relief eight times coarser.
	for z := MaxLevel - 1; z > CompleteLevel; z-- {
		parents := map[Address]bool{}
		for a := range s.Tiles {
			if a.Z == z+1 {
				parents[a.Parent()] = true
			}
		}
		if ring := counts[z] - len(parents); ring < len(parents)/4 {
			t.Errorf("level %d has %d tiles of which %d are parents: its wider ring is missing", z, counts[z], len(parents))
		}
	}
	if counts[MaxLevel] < 900 {
		t.Errorf("only %d level-8 tiles", counts[MaxLevel])
	}
}

// A tile is drawn only by walking down from level 2, so a gap above a land tile would hide it.
func TestEveryLandTileCanBeReached(t *testing.T) {
	s := committed(t)
	for a, tile := range s.Tiles {
		if tile.Kind == Sea {
			continue
		}
		for up := a; up.Z > MinLevel; up = up.Parent() {
			if !s.has(up.Parent()) {
				t.Errorf("%v holds land but %v above it is not listed", a, up.Parent())
				break
			}
		}
	}
}

// drawnLand answers whether a place is on the land the basemap draws. The renderer fills by the
// even-odd rule over every outline (land, lakes, islands in lakes), and so does this.
type drawnLand struct {
	rings [][][2]float64
	boxes [][4]float64 // west, south, east, north
}

func loadDrawnLand(t *testing.T) *drawnLand {
	t.Helper()
	raw, err := assets.Terrain.ReadFile("terrain/world-land.json")
	if err != nil {
		t.Fatal(err)
	}
	var basemap struct {
		Land    [][][2]float64 `json:"land"`
		Lakes   [][][2]float64 `json:"lakes"`
		Islands [][][2]float64 `json:"lake_islands"`
	}
	if err = json.Unmarshal(raw, &basemap); err != nil {
		t.Fatal(err)
	}
	d := &drawnLand{}
	for _, ring := range append(append(basemap.Land, basemap.Lakes...), basemap.Islands...) {
		box := [4]float64{math.Inf(1), math.Inf(1), math.Inf(-1), math.Inf(-1)}
		for _, p := range ring {
			box[0], box[1] = math.Min(box[0], p[0]), math.Min(box[1], p[1])
			box[2], box[3] = math.Max(box[2], p[0]), math.Max(box[3], p[1])
		}
		d.rings, d.boxes = append(d.rings, ring), append(d.boxes, box)
	}
	return d
}

func (d *drawnLand) holds(lon, lat float64) bool {
	inside := false
	for i, ring := range d.rings {
		if box := d.boxes[i]; lon < box[0] || lon > box[2] || lat < box[1] || lat > box[3] {
			continue
		}
		for j, k := 0, len(ring)-1; j < len(ring); k, j = j, j+1 {
			a, b := ring[j], ring[k]
			if (a[1] > lat) != (b[1] > lat) && lon < (b[0]-a[0])*(lat-a[1])/(b[1]-a[1])+a[0] {
				inside = !inside
			}
		}
	}
	return inside
}

// Levels 2 to 5 must list every tile that holds drawn land, or that land is flat at wide views. Each
// level-5 tile is probed at 64 places, and a place on land needs its tile at all four levels.
func TestCoarseLevelsCoverAllDrawnLand(t *testing.T) {
	s, land := committed(t), loadDrawnLand(t)
	if !land.holds(-74.07, 4.71) || land.holds(-30, 30) || land.holds(136.1, 35.3) {
		t.Fatal("the land test is wrong for Bogota, the mid-Atlantic or Lake Biwa")
	}
	n, onLand := 1<<CompleteLevel, 0
	for x := range n {
		for y := range n {
			for i := range 64 {
				lat, lon := Corner(Address{CompleteLevel, x, y}, (float64(i%8)+.5)/8, (float64(i/8)+.5)/8)
				if lat <= -60 || !land.holds(lon, lat) {
					continue
				}
				onLand++
				for z := MinLevel; z <= CompleteLevel; z++ {
					if at := At(z, lat, lon); !s.has(at) {
						t.Fatalf("drawn land at %.3f N, %.3f E has no level-%d tile %v", lat, lon, z, at)
					}
				}
			}
		}
	}
	if onLand < 10000 {
		t.Fatalf("only %d probes fell on land", onLand)
	}
}

// Every committed tile opens: its files match the manifest's hashes and decode to a full tile. The
// game checks this only when a tile is first shown, which for most of the world is never in a test.
func TestEveryCommittedTileDecodes(t *testing.T) {
	s := committed(t)
	for a, tile := range s.Tiles {
		if tile.Kind == Sea {
			continue
		}
		img, err := s.Decode(assets.Terrain, a)
		if err != nil {
			t.Fatalf("%v: %v", a, err)
		}
		land := 0
		for i := 3; i < len(img.Pix); i += 4 {
			if img.Pix[i] != 0 {
				land++
			}
		}
		if land == 0 || (tile.Kind == Land && land != TileSize*TileSize) {
			t.Fatalf("%v is listed as %v but %d of its pixels are land", a, tile.Kind, land)
		}
	}
}

// A changed coast fails here until the tiles are cut to it again.
func TestTilesWereCutToThisBasemap(t *testing.T) {
	s := committed(t)
	raw, err := assets.Terrain.ReadFile("terrain/world-land.json")
	if err != nil {
		t.Fatal(err)
	}
	if sum := sha256.Sum256(raw); s.BasemapSHA != hex.EncodeToString(sum[:]) {
		t.Fatal("the tiles were cut to a different basemap: run tools/prepare-terrain.py --write")
	}
	if !s.Complete {
		t.Fatal("the committed tiles are a sample, not the full set")
	}
}

func TestEveryListedTileHasItsFiles(t *testing.T) {
	s := committed(t)
	want := map[string]bool{}
	for a, tile := range s.Tiles {
		if tile.Kind != Sea {
			want[Path(a)+".jpg"] = true
		}
		if tile.Kind == Coast {
			want[Path(a)+".png"] = true
		}
	}
	found, unlisted := 0, 0
	err := fs.WalkDir(assets.Terrain, "terrain/tiles", func(path string, d fs.DirEntry, err error) error {
		if err != nil || d.IsDir() {
			return err
		}
		if !want[path] {
			if unlisted++; unlisted <= 10 {
				t.Errorf("%s is not listed in the manifest", path)
			}
		}
		if !strings.HasSuffix(path, ".jpg") && !strings.HasSuffix(path, ".png") {
			t.Errorf("%s is not a tile file", path)
		}
		found++
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
	if found != len(want) {
		t.Fatalf("%d tile files are embedded but the manifest lists %d", found, len(want))
	}
}

// The whole point: a route through mountains is not drawn on flat-looking land.
func TestBogotaToMedellinHasSharpRelief(t *testing.T) {
	s := committed(t)
	for _, place := range [][2]float64{{4.711, -74.072}, {6.244, -75.581}, {5.5, -74.7}} {
		at := At(MaxLevel, place[0], place[1])
		if tile, ok := s.Tiles[at]; !ok || tile.Kind == Sea {
			t.Errorf("%.2f N, %.2f E has no level-8 land tile (%v)", place[0], place[1], at)
		}
	}
}
