package render

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"math"
	"testing"

	"wwhsrt/assets"
	"wwhsrt/pkg/gameapi"
	"wwhsrt/pkg/mapview"
	"wwhsrt/pkg/terrain"
)

// ringArea is the ring's signed area in degrees: positive when it winds counter-clockwise.
func ringArea(ring [][2]float64) float64 {
	sum := 0.0
	for i := 1; i < len(ring); i++ {
		sum += ring[i-1][0]*ring[i][1] - ring[i][0]*ring[i-1][1]
	}
	return sum / 2
}

// ringContains reports whether p is strictly inside ring (even-odd rule).
func ringContains(ring [][2]float64, p [2]float64) bool {
	in := false
	for i, j := 0, len(ring)-1; i < len(ring); j, i = i, i+1 {
		a, b := ring[i], ring[j]
		if (a[1] > p[1]) != (b[1] > p[1]) && p[0] < (b[0]-a[0])*(p[1]-a[1])/(b[1]-a[1])+a[0] {
			in = !in
		}
	}
	return in
}

func inAny(rings [][][2]float64, lon, lat float64) bool {
	for _, ring := range rings {
		if ringContains(ring, [2]float64{lon, lat}) {
			return true
		}
	}
	return false
}

func (b *basemap) landRings() [][][2]float64   { return b.rings[:b.land] }
func (b *basemap) lakeRings() [][][2]float64   { return b.rings[b.land : b.land+b.lakes] }
func (b *basemap) islandRings() [][][2]float64 { return b.rings[b.land+b.lakes:] }

// inLake reports whether the point is on a drawn lake's water.
func inLake(b *basemap, lon, lat float64) bool {
	return inAny(b.lakeRings(), lon, lat) && !inAny(b.islandRings(), lon, lat)
}

// drawnLand reports whether the map paints land at the point.
func drawnLand(b *basemap, lon, lat float64) bool {
	return inAny(b.landRings(), lon, lat) && !inLake(b, lon, lat)
}

func loadBasemap(t *testing.T) *basemap {
	t.Helper()
	b, err := newBasemap()
	if err != nil {
		t.Fatal(err)
	}
	return b
}

func TestBasemapMatchesItsMetadata(t *testing.T) {
	b := loadBasemap(t)
	raw, err := assets.Terrain.ReadFile("terrain/world-land.json")
	if err != nil {
		t.Fatal(err)
	}
	var metadata struct {
		Output string `json:"output_sha256"`
	}
	text, err := assets.Terrain.ReadFile("terrain/world-land.metadata.json")
	if err != nil {
		t.Fatal(err)
	}
	if err = json.Unmarshal(text, &metadata); err != nil {
		t.Fatal(err)
	}
	if sum := sha256.Sum256(raw); hex.EncodeToString(sum[:]) != metadata.Output {
		t.Fatal("world-land.json does not match the hash in its metadata; run tools/prepare-basemap.py")
	}
	if b.land == 0 || b.lakes == 0 || b.islands == 0 || b.land+b.lakes+b.islands != len(b.rings) {
		t.Fatalf("land %d, lakes %d, lake islands %d, rings %d", b.land, b.lakes, b.islands, len(b.rings))
	}
	points := 0
	for i, ring := range b.rings {
		points += len(ring)
		if len(ring) < 4 || ring[0] != ring[len(ring)-1] {
			t.Fatalf("ring %d is not a closed outline of at least four points", i)
		}
		for _, p := range ring {
			if math.Abs(p[0]) > 180 || math.Abs(p[1]) > 90 {
				t.Fatalf("ring %d has a point outside the globe: %v", i, p)
			}
		}
	}
	t.Logf("%d points in %d outlines", points, len(b.rings))
	if points > 80000 || len(b.rings) > 2200 {
		t.Fatalf("basemap has %d points in %d outlines; the budget is 80000 and 2200", points, len(b.rings))
	}
}

// The renderer fills every outline as one nonzero path, so a lake is only cut out of its land when it
// winds the other way. A lake wound like land would be filled as land with no error.
func TestBasemapOutlinesWindSoLakesAreCutOut(t *testing.T) {
	b := loadBasemap(t)
	for i, ring := range b.rings {
		lake := i >= b.land && i < b.land+b.lakes
		if area := ringArea(ring); area == 0 || (area < 0) != lake {
			t.Fatalf("ring %d (lake: %v) has signed area %v", i, lake, area)
		}
	}
}

func TestBasemapHasNoEdgeAcrossTheMap(t *testing.T) {
	b := loadBasemap(t)
	for i, ring := range b.rings {
		for j := 1; j < len(ring); j++ {
			// Antarctica's outline runs along the pole from one edge of the map to the other. On an
			// Equal Earth map the pole is a line, so that edge is drawn where it belongs.
			if math.Abs(ring[j][1]) == 90 && ring[j][1] == ring[j-1][1] {
				continue
			}
			if math.Abs(ring[j][0]-ring[j-1][0]) > 180 {
				t.Fatalf("ring %d jumps from %v to %v", i, ring[j-1], ring[j])
			}
		}
	}
}

func TestBasemapKnownPoints(t *testing.T) {
	b := loadBasemap(t)
	for _, c := range []struct {
		name     string
		lon, lat float64
		want     string
	}{
		{"Istanbul, European shore", 28.95, 41.02, "land"},
		{"Istanbul, Asian shore", 29.06, 41.00, "land"},
		{"Tangier", -5.80, 35.76, "land"},
		{"Folkestone", 1.15, 51.10, "land"},
		{"Bogota", -74.07, 4.71, "land"},
		{"Manitoulin Island", -82.2, 45.75, "land"},
		{"San Francisco Bay", -122.25, 37.70, "sea"},
		{"Thames estuary", 0.90, 51.48, "sea"},
		{"Gulf of Izmit", 29.60, 40.73, "sea"},
		{"Osaka Bay", 135.25, 34.55, "sea"},
		// The water points tools/check-coastlines.cjs samples in the browser.
		{"checker: west of Hokkaido", 140.09867836, 42.95577938, "sea"},
		{"checker: Sea of Japan off Joetsu", 138.1263546, 37.31509792, "sea"},
		{"checker: off Hachinohe", 141.60869049, 40.63917142, "sea"},
		{"checker: Tsugaru Strait", 140.54059505, 41.59805279, "sea"},
		{"checker: Hangzhou Bay", 121.15774495, 30.47348219, "sea"},
		{"checker: Laizhou Bay", 119.10184264, 37.29738934, "sea"},
		{"checker: off Wenzhou", 121.17869533, 28.26677417, "sea"},
		{"checker: Qiongzhou Strait", 110.48955745, 20.07817143, "sea"},
		{"Lake Michigan", -87.0, 43.5, "lake"},
		{"Lake Biwa", 136.08, 35.25, "lake"},
		{"Lake Geneva", 6.55, 46.45, "lake"},
	} {
		got := "sea"
		if inLake(b, c.lon, c.lat) {
			got = "lake"
		} else if drawnLand(b, c.lon, c.lat) {
			got = "land"
		}
		if got != c.want {
			t.Errorf("%s (%v, %v) is drawn as %s, want %s", c.name, c.lon, c.lat, got, c.want)
		}
	}
}

// cityOffLand names the cities whose station marker is not on drawn land, and why that is accepted.
var cityOffLand = map[string]string{
	"hong_kong": "its station marker is 347 m off Natural Earth 1:10m land",
	"dongguan":  "its station marker is 209 m off Natural Earth 1:10m land",
	"miami":     "its marker is 473 m off Natural Earth 1:10m land",
	"lisbon":    "its station marker is 396 m off Natural Earth 1:10m land",
	"zhuhai":    "its station marker is 10 m off Natural Earth 1:10m land",
}

func TestCatalogueCitiesAreOnDrawnLand(t *testing.T) {
	b := loadBasemap(t)
	var catalogue struct {
		Cities []struct {
			ID         string  `json:"id"`
			Lon        float64 `json:"lon"`
			Lat        float64 `json:"lat"`
			StationLon float64 `json:"station_lon"`
			StationLat float64 `json:"station_lat"`
		} `json:"cities"`
	}
	if err := json.Unmarshal(assets.PublicNetwork, &catalogue); err != nil {
		t.Fatal(err)
	}
	if len(catalogue.Cities) < 200 {
		t.Fatalf("only %d cities in the catalogue", len(catalogue.Cities))
	}
	seen := map[string]bool{}
	for _, c := range catalogue.Cities {
		seen[c.ID] = true
		// A city is marked at its station where the catalogue has one, and at its centre otherwise.
		lon, lat := c.StationLon, c.StationLat
		if lon == 0 && lat == 0 {
			lon, lat = c.Lon, c.Lat
		}
		_, excused := cityOffLand[c.ID]
		switch on := drawnLand(b, lon, lat); {
		case !on && !excused:
			t.Errorf("%s's marker (%v, %v) is not on drawn land", c.ID, lon, lat)
		case on && excused:
			t.Errorf("%s is on drawn land; remove it from cityOffLand", c.ID)
		}
	}
	for id := range cityOffLand {
		if !seen[id] {
			t.Errorf("cityOffLand names %s, which is not in the catalogue", id)
		}
	}
}

// Frames without the public network come only from tests, but they used to draw a regional outline of
// their own, and the Japanese route check verified alignments against that file, not what players see.
func TestEveryFrameDrawsTheOneBasemap(t *testing.T) {
	m, err := New()
	if err != nil {
		t.Fatal(err)
	}
	with, without := m.land(gameapi.Frame{PublicNetwork: true}), m.land(gameapi.Frame{})
	if with == nil || with != without || len(with.rings) == 0 {
		t.Fatal("a frame without the public network does not draw the same basemap as one with it")
	}
}

// Lake Biwa is drawn twice for now. Today's relief tiles tint the lake's surface as land, so with
// terrain on, the terrain layer's own outline is what keeps the lake visible; with terrain off, the
// basemap's is. Both come from Natural Earth 1:10m, and they must stay close enough to read as one
// shore. The terrain layer's copy goes when relief is cut to the basemap.
func TestLakeBiwaHasOneShoreInBothLayers(t *testing.T) {
	b := loadBasemap(t)
	var shore [][2]float64
	for _, ring := range b.lakeRings() {
		if ringContains(ring, [2]float64{136.08, 35.25}) {
			shore = ring
		}
	}
	if shore == nil {
		t.Fatal("the basemap does not draw Lake Biwa")
	}
	data, err := terrain.Load(assets.Terrain)
	if err != nil {
		t.Fatal(err)
	}
	if len(data.Lakes) != 1 || data.Lakes[0].Key != "lake_biwa" {
		t.Fatalf("expected the terrain layer to outline Lake Biwa alone, got %d lakes", len(data.Lakes))
	}
	worst := 0.0
	for _, p := range data.Lakes[0].Points { // latitude first
		at, nearest := mapview.Point{X: p[1], Y: p[0]}, math.Inf(1)
		for i := 1; i < len(shore); i++ {
			a, c := mapview.Point{X: shore[i-1][0], Y: shore[i-1][1]}, mapview.Point{X: shore[i][0], Y: shore[i][1]}
			nearest = min(nearest, math.Sqrt(segmentDistance2(at, a, c)))
		}
		worst = max(worst, nearest)
	}
	if worst > 0.004 {
		t.Fatalf("the two outlines of Lake Biwa are %.4f degrees apart at their worst; the limit is 0.004", worst)
	}
}

// Choosing points by rank must give what simplifying the outline at that tolerance would, so the map
// shows the same shape the per-frame simplification drew, without depending on where the camera is.
func TestOutlineRanksSelectWhatSimplifyingWould(t *testing.T) {
	ring := []mapview.Point{}
	seed := uint32(7)
	for i := range 400 {
		seed = seed*1664525 + 1013904223
		angle := 2 * math.Pi * float64(i) / 400
		radius := 100 + float64(seed>>20)/4096*30
		ring = append(ring, mapview.Point{X: radius * math.Cos(angle), Y: radius * math.Sin(angle)})
	}
	ring = append(ring, ring[0])
	rank := rankOutline(ring)
	if len(rank) != len(ring) || !math.IsInf(rank[0], 1) || !math.IsInf(rank[len(rank)-1], 1) {
		t.Fatal("an outline's first and last points must always be kept")
	}
	for _, tolerance := range []float64{0.5, 2, 8, 25} {
		want := simplifyScreen(ring, tolerance)
		got := []mapview.Point{}
		for i, p := range ring {
			if rank[i] > tolerance {
				got = append(got, p)
			}
		}
		if len(got) != len(want) {
			t.Fatalf("tolerance %v keeps %d points by rank, %d by simplifying", tolerance, len(got), len(want))
		}
		for i := range want {
			if got[i] != want[i] {
				t.Fatalf("tolerance %v: point %d differs", tolerance, i)
			}
		}
	}
}

func TestProjectedPointsMatchTheRing(t *testing.T) {
	b := loadBasemap(t)
	if len(b.outlines) != len(b.rings) {
		t.Fatalf("%d outlines for %d rings", len(b.outlines), len(b.rings))
	}
	for i, o := range b.outlines {
		ring := b.rings[i]
		if len(o.points) != len(ring) || len(o.rank) != len(ring) {
			t.Fatalf("outline %d has %d points and %d ranks for %d ring points", i, len(o.points), len(o.rank), len(ring))
		}
		for j, p := range o.points {
			if p != mapview.Project(ring[j][1], ring[j][0]) {
				t.Fatalf("outline %d point %d is not the ring's point projected", i, j)
			}
			if p.X < o.min.X || p.X > o.max.X || p.Y < o.min.Y || p.Y > o.max.Y {
				t.Fatalf("outline %d point %d lies outside its bounds", i, j)
			}
		}
	}
}

func TestBasemapSkipsOutlinesOffScreen(t *testing.T) {
	b := loadBasemap(t)
	japan := mapview.Fit(1280, 850, []mapview.Point{mapview.Project(35.68, 139.77), mapview.Project(34.69, 135.50)})
	japan.Zoom = 8
	drawn := b.visible(japan)
	if len(drawn) == 0 || len(drawn) >= len(b.rings)/10 {
		t.Fatalf("a view between Tokyo and Osaka draws %d of %d outlines", len(drawn), len(b.rings))
	}
	for i, points := range drawn {
		if len(points) < 3 {
			t.Fatalf("outline %d has %d points; it cannot be filled", i, len(points))
		}
		for _, p := range points {
			// Clipping computes where an edge meets the margin, so allow for rounding there.
			const edge = 4 + 1e-6
			if p.X < -edge || p.Y < -edge || p.X > japan.Width+edge || p.Y > japan.Height+edge {
				t.Fatalf("outline %d has a point off screen: %v", i, p)
			}
		}
	}
	pacific := mapview.Fit(1280, 850, []mapview.Point{mapview.Project(0, -150), mapview.Project(1, -149)})
	pacific.Zoom = 32
	if n := len(b.visible(pacific)); n != 0 {
		t.Fatalf("open ocean draws %d outlines", n)
	}
	if n := len(b.visible(mapview.Camera{Zoom: 1, Max: mapview.Point{X: 1, Y: 1}})); n != 0 {
		t.Fatalf("a map with no area draws %d outlines", n)
	}
}

// Which points are drawn depends on the zoom alone. If it also depended on where the camera is, the
// coast would shimmer while the map is dragged.
func TestBasemapDetailDoesNotChangeWhenPanned(t *testing.T) {
	b := loadBasemap(t)
	c := mapview.RegionCamera(3032, 1863, "world", false)
	count := func(c mapview.Camera) int {
		n := 0
		for i := range b.outlines {
			n += len(b.chosen(i, c))
		}
		return n
	}
	still := count(c)
	c.PanX, c.PanY = 311, -57
	if moved := count(c); moved != still || still == 0 {
		t.Fatalf("%d points chosen before the pan, %d after", still, moved)
	}
}
