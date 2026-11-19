package render

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"image/color"
	"math"
	"testing"

	"wwhsrt/assets"
	"wwhsrt/pkg/gameapi"
	"wwhsrt/pkg/mapview"
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
		// Counted, not measured: an area computed in floating point differs between arm64, which
		// fuses a multiply and a subtract, and amd64, which does not.
		distinct := map[[2]float64]bool{}
		for _, p := range ring {
			distinct[p] = true
		}
		if len(distinct) < 3 {
			t.Fatalf("ring %d has only %d distinct points: %v", i, len(distinct), ring)
		}
		for _, p := range ring {
			if math.Abs(p[0]) > 180 || math.Abs(p[1]) > 90 {
				t.Fatalf("ring %d has a point outside the globe: %v", i, p)
			}
		}
	}
	t.Logf("%d points in %d outlines", points, len(b.rings))
	if points > 93000 || len(b.rings) > 2200 {
		t.Fatalf("basemap has %d points in %d outlines; the budget is 93000 and 2200", points, len(b.rings))
	}
}

// The file winds land and lake islands one way and lakes the other, for any reader that fills by
// winding. The renderer fills by the even-odd rule and does not depend on it. An area of zero means an
// outline collapsed to a line when its coordinates were rounded.
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
		{"Copenhagen inner harbour", 12.575, 55.6675, "sea"},
		{"Helsingborg ferry basin", 12.6867, 56.041, "sea"},
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

// A continent's outline stroked as one closed sub-path took 5 GB of GPU memory at 3737x2029: Ebitengine
// fans triangles from a sub-path's first point, so every coast segment made a triangle across the map.
func TestCoastIsStrokedInShortRuns(t *testing.T) {
	ring := []mapview.Point{}
	for i := range 100 {
		ring = append(ring, mapview.Point{X: float64(i), Y: float64(i * i % 7)})
	}
	closed := append(append([]mapview.Point{}, ring...), ring[0])
	for name, outline := range map[string][]mapview.Point{"closed": closed, "left open by clipping": ring} {
		runs := coastRuns([][]mapview.Point{outline})
		segments := 0
		for i, run := range runs {
			if len(run) < 2 || len(run) > coastRun+1 {
				t.Fatalf("%s: run %d has %d points; a run is 2 to %d", name, i, len(run), coastRun+1)
			}
			if i > 0 && run[0] != runs[i-1][len(runs[i-1])-1] {
				t.Fatalf("%s: run %d does not start where run %d ended", name, i, i-1)
			}
			segments += len(run) - 1
		}
		if segments != 100 || runs[0][0] != ring[0] || runs[len(runs)-1][len(runs[len(runs)-1])-1] != ring[0] {
			t.Fatalf("%s: %d segments in %d runs; the whole outline is 100 segments and returns to its start",
				name, segments, len(runs))
		}
	}
	if runs := coastRuns([][]mapview.Point{closed, closed}); len(runs) != 2*len(coastRuns([][]mapview.Point{closed})) {
		t.Fatal("two outlines should give twice the runs of one")
	}
}

// parityGrid marks, at the centre of every third pixel, whether an odd number of the given shapes cover
// it. Each shape is a closed polygon; a sample is covered when a ray to its right crosses the shape's
// edges an odd number of times. An edge is always measured from its upper end, so two shapes sharing
// an edge agree exactly on which side of it a sample lies.
func parityGrid(shapes [][]mapview.Point, w, h float64) []bool {
	const stride = 3
	cols, rows := int(w)/stride, int(h)/stride
	flips := make([]int32, (cols+1)*rows)
	for _, points := range shapes {
		for i := range points {
			p, q := points[i], points[(i+1)%len(points)]
			if p.Y == q.Y {
				continue
			}
			if p.Y > q.Y {
				p, q = q, p
			}
			first := max(0, int(math.Ceil((p.Y-1.5)/stride)))
			for r := first; r < rows && float64(r*stride)+1.5 < q.Y; r++ {
				y := float64(r*stride) + 1.5
				if y < p.Y {
					continue
				}
				x := p.X + (y-p.Y)/(q.Y-p.Y)*(q.X-p.X)
				// Samples left of the edge see it to their right.
				flips[r*(cols+1)+min(cols, max(0, int(math.Ceil((x-1.5)/stride))))]++
			}
		}
	}
	covered := make([]bool, cols*rows)
	for r := range rows {
		crossings := int32(0)
		for c := cols - 1; c >= 0; c-- {
			crossings += flips[r*(cols+1)+c+1]
			covered[r*cols+c] = crossings%2 == 1
		}
	}
	return covered
}

// The fill is drawn as triangles that each flip the pixels they cover, so a pixel ends up land when an
// odd number cover it. This checks the triangles handed to the GPU against the outlines themselves.
//
// The fill used to be Ebitengine's own, which counts overlapping triangles in four bits. With one land
// outline a continent is a single shape, pixels deep in France sat under 33 to 39 triangles, and the
// map showed water-coloured streaks from Spain to the Baltic.
func TestLandFillPaintsExactlyTheLand(t *testing.T) {
	b := loadBasemap(t)
	const w, h = 3032, 1863
	zoomed := func(lat, lon, zoom float64) mapview.Camera {
		c := mapview.RegionCamera(w, h, "world", false)
		c.Zoom *= zoom
		at := c.Screen(mapview.Project(lat, lon))
		c.PanX, c.PanY = c.PanX+w/2-at.X, c.PanY+h/2-at.Y
		return c
	}
	for name, c := range map[string]mapview.Camera{
		"the World":                  mapview.RegionCamera(w, h, "world", false),
		"Europe at four times":       zoomed(48, 15, 4),
		"Britain at eight times":     zoomed(54, -2, 8),
		"North America at two times": zoomed(40, -95, 2),
		"the Great Lakes at eight":   zoomed(45, -84, 8),
		"Norway's fjords at sixteen": zoomed(62, 6, 16),
		"Japan at ten times":         zoomed(37.34, 137.09, 10),
	} {
		outlines := b.visible(c)
		vertices, indices := landFan(cells(outlines), color.NRGBA{R: 1, G: 2, B: 3, A: 255})
		if len(indices) == 0 || len(indices)%3 != 0 {
			t.Fatalf("%s: %d indices", name, len(indices))
		}
		triangles := make([][]mapview.Point, 0, len(indices)/3)
		for i := 0; i < len(indices); i += 3 {
			triangle := make([]mapview.Point, 3)
			for k := range 3 {
				v := vertices[indices[i+k]]
				triangle[k] = mapview.Point{X: float64(v.DstX), Y: float64(v.DstY)}
			}
			if side := max(triangle[0].X, triangle[1].X, triangle[2].X) - min(triangle[0].X, triangle[1].X, triangle[2].X); side > landCell+1 {
				t.Fatalf("%s: a triangle is %.0f px wide; pieces are cut to %d so that none crosses the map", name, side, landCell)
			}
			triangles = append(triangles, triangle)
		}
		// The outlines as the GPU receives them: single precision.
		single := make([][]mapview.Point, len(outlines))
		for i, points := range outlines {
			for _, p := range points {
				single[i] = append(single[i], mapview.Point{X: float64(float32(p.X)), Y: float64(float32(p.Y))})
			}
		}
		want, got := parityGrid(single, w, h), parityGrid(triangles, w, h)
		land, wrong := 0, 0
		for i := range want {
			if want[i] {
				land++
			}
			if want[i] != got[i] {
				wrong++
			}
		}
		// Cutting an outline makes new points on the cut, rounded to single precision, so a sample
		// within a millionth of a pixel of an edge can fall either way.
		if land == 0 || wrong > land/100000 {
			t.Errorf("%s: %d of %d land samples are painted wrongly", name, wrong, land)
		}
	}
}

// Coast between 150 and 400 km from a line is drawn to 1.4 km, not 5. Eastern Hokkaido is 150 to
// 350 km from Sapporo: with nothing between fine and coarse, the Shiretoko and Nemuro peninsulas came
// out as spikes at the Japan region's zoom, where the whole island had been detailed before.
func TestCoastJustBeyondALineIsNotCrude(t *testing.T) {
	b := loadBasemap(t)
	for _, ring := range b.landRings() {
		if !ringContains(ring, [2]float64{141.35, 43.06}) { // Sapporo
			continue
		}
		// A well-drawn coast has no long straight edges: here the longest is 29 km, on the smooth
		// Okhotsk shore. The spikes were drawn with edges up to 71 km.
		longest := 0.0
		for i := 1; i < len(ring); i++ {
			a, c := ring[i-1], ring[i]
			if a[0] > 143 && c[0] > 143 {
				km := 111 * math.Hypot((c[0]-a[0])*math.Cos(43.5*math.Pi/180), c[1]-a[1])
				longest = max(longest, km)
			}
		}
		if longest == 0 || longest > 40 {
			t.Fatalf("Hokkaido's coast east of 143°E has an edge %.0f km long; none should pass 40", longest)
		}
		return
	}
	t.Fatal("no land outline contains Sapporo")
}

// A long thin island has every point within a pixel of the line through its ends, so choosing points
// by rank alone leaves it two: at a 1440x800 World view a 61x2 px island and K'gari both vanished.
func TestThinOutlinesAreStillDrawn(t *testing.T) {
	b := loadBasemap(t)
	for _, size := range [][2]float64{{3032, 1863}, {1440, 800}} {
		c := mapview.RegionCamera(size[0], size[1], "world", false)
		drawn, specks := 0, 0
		for i := range b.outlines {
			o := &b.outlines[i]
			lo, hi := c.Screen(o.min), c.Screen(o.max)
			long := max(hi.X-lo.X, lo.Y-hi.Y)
			points := len(b.chosen(i, c))
			switch {
			case long >= landSpeck && points < 4:
				t.Fatalf("%vx%v: outline %d is %.1f px long and is not drawn", size[0], size[1], i, long)
			case long < landTolerance && points != 0: // between the two it depends on the outline's shape
				t.Fatalf("%vx%v: outline %d is %.2f px long and is drawn", size[0], size[1], i, long)
			case points == 0:
				specks++
			default:
				drawn++
			}
		}
		if drawn == 0 || specks == 0 {
			t.Fatalf("%vx%v: %d outlines drawn, %d too small to draw; expected some of each", size[0], size[1], drawn, specks)
		}
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
