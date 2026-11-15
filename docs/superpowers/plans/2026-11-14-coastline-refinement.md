# Coastline Refinement Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Replace the worldwide basemap with one land outline from the pinned Natural Earth 1:10m data, detailed within 150 km of any line and coarse beyond, drawn in every frame, with the regional Japan files retired.

**Architecture:** A small Python module holds the rule (what is near a line, how an outline is simplified, what is left out) and a new tool applies it to the `land` geometry in `data/construction/route-land.v1.json.gz`, writing `assets/terrain/world-land.json` as three lists of outlines. A new `basemap` type in `pkg/render` loads and draws that file for every frame; the per-country outlines, the regional file and the test-only drawing branch go.

**Tech Stack:** Go 1.26+, Ebitengine v2.10.5 (`vector`), Python 3.12 with shapely 2.1.2 and scipy 1.14.1.

**Spec:** `docs/superpowers/specs/2026-11-14-coastline-refinement-design.md`

**Revised 2026-11-15** against `b6ecac4`. After the plan was written, land drawing moved to `pkg/render/land.go` (one fill path and one coast path for all outlines, to keep Ebitengine's stencil small), outlines are simplified in screen space, and CI began running the browser and data checks on push. Tasks 2 to 5 below are the revised ones.

## Global Constraints

- Within 150 km of a line the outline is simplified at 0.003°; farther away at 0.05°. Detail may change along one outline; both stretches share the point where it changes.
- "A line" is every path in `assets/scenario/playable-rail-map.json` (`lines`, `[lon, lat]`), every alignment path in `data/construction/compiled-prospects.v1.json` (`corridors[].alignments[].path`, `{lon, lat}`), and every alignment path of `hokkaido-extension` and `tsuruga-osaka` in `assets/scenario/public-network.json` (`corridors[].alignments[].path`, `{lat, lon}`).
- The Hokkaido and Kansai relief boxes count as near a line throughout: zoom-8 tiles x 227–228, y 93–95 (lon 139.21875 to 142.03125, lat 40.9799 to 44.08759) and x 224–225, y 100–101 (lon 135.0 to 137.8125, lat 34.30714 to 36.59789).
- Kept: every land piece near a line; a far island at least 20 km across; a lake near a line at least 5 km across; a far lake at least 100 km across. A lake left out is drawn as land.
- Budget: at most 80,000 points and 2,200 outlines in all (the design's goal is about 73,000 and 2,000). The size thresholds may be raised to hold it; the tolerances and the 150 km distance may not change.
- Every alignment of all 74 buildable routes lies on drawn land or its reviewed crossing, in geographic and Equal Earth coordinates. Where 0.003° fails, tighten locally through 0.001°, 0.0003°, 0.0001°, then unsimplified.
- No lines between countries. No change to `assets/scenario/public-network.json`, `assets/scenario/playable-rail-map.json`, `data/construction/*`, `assets/terrain/features.json`, any scenario file, or any save or checkpoint.
- No downloads. Every input is already in the repository.
- Python is `python3` (3.12, `/Library/Frameworks`); shapely and scipy are in its user site, so do not pass `-I`.
- Run memory-heavy tools one at a time. `tools/import-public-network.py` peaks near 1,768 MiB. Stop if combined check processes pass 2 GiB.
- No local browser or WASM run, and no TypeScript compilation. CI runs `tools/check-coastlines.cjs` on push; it is updated here and first verified by this branch's CI.
- Land is drawn in two antialiased passes, one fill and one coast (`pkg/render/land.go`). Do not add a third: each pass adds a map-sized box to Ebitengine's stencil.
- In `world-land.json`, land and lake-island outlines wind counter-clockwise and lakes clockwise, so the renderer's one nonzero fill cuts lakes out of land.
- `TestWorldViewGeometryStaysNearScreenResolution` keeps its limit of 22,000 land vertices at the World view.
- `pkg/terrain` and the headless composition stay graphics-free (`python3 tools/check-boundaries.py`).
- Lines stay within 120 columns; `gofmt` clean; `go vet ./...` and `GOOS=js GOARCH=wasm go vet ./...` clean.
- Go test command for a task unless it says otherwise:
  `GOMEMLIMIT=768MiB GOMAXPROCS=2 go test -p 1 ./pkg/render ./pkg/terrain ./pkg/app -count=1`
- Commit once per task on branch `worktree-coastline-refinement` (the user's choice, 2026-11-15); never push without being asked.

## Review Focus

1. **An outline that reaches the antimeridian.** Russia, Fiji and Antarctica touch ±180°. No drawn outline may have an edge spanning more than 180° of longitude, which would paint a streak across the map. (Task 2 Go test.)
2. **A route that ends at the waterfront.** If a path is still off land with the outline unsimplified, the tool stops and names the route and the point; it does not loop or write a file. (Task 2 tool test.)
3. **Islands inside lakes, and lakes that are left out.** An island in a kept lake is drawn as land over the lake; an island in a lake that was left out is not emitted at all. (Task 1 test.)
4. **Winding.** A lake wound the same way as its land is filled as land by the nonzero rule, with no error. Every outline's direction is checked. (Task 1 test and Task 2 Go test.)
5. **A window of zero size, or a camera looking at open sea.** Drawing with no visible outline, or a 0×0 map area, produces no outline and does not panic. (Task 4 Go test.)

---

### Task 1: The basemap rule

**Files:**
- Create: `tools/basemap_rule.py`
- Test: `tools/test-basemap.py` (plain asserts, run as a script, like `tools/test-network-coverage.py`)

**Interfaces:**
- Produces, all in `tools/basemap_rule.py`:
  - constants `NEAR_KM = 150`, `FINE = 0.003`, `COARSE = 0.05`, `FAR_ISLAND_KM = 20`, `NEAR_LAKE_KM = 5`, `FAR_LAKE_KM = 100`, `LADDER = [0.003, 0.001, 0.0003, 0.0001, 0]`, `MAX_POINTS = 80000`, `MAX_OUTLINES = 2200`, `JAPAN_BOXES = [(8, 227, 93, 228, 95), (8, 224, 100, 225, 101)]`
  - `tile_bounds(z, x0, y0, x1, y1) -> (west, south, east, north)`
  - `project(x, y, z=None)`: the Equal Earth transform moved unchanged from `tools/launch_shorelines.py`
  - `class NearIndex(paths, boxes, km=NEAR_KM)` with `near(lon, lat) -> bool`; `paths` is a list of lists of `(lon, lat)`, `boxes` a list of `(west, south, east, north)`
  - `across_km(ring) -> float`: great-circle length of the ring's bounding-box diagonal
  - `simplify_ring(ring, flags, fine=FINE, coarse=COARSE) -> list[(lon, lat)]`: `ring` closed, `flags[i]` true where vertex `i` is near
  - `compose(polygons, index, fine_for=None) -> dict`: `polygons` is the `coordinates` of a GeoJSON MultiPolygon; `fine_for` maps a polygon's index to a tighter fine tolerance; returns `{"land": [...], "lakes": [...], "lake_islands": [...], "sources": {...}}` where each list holds closed rings of `[lon, lat]` rounded to 6 places and `sources["land"][i]` is the index of the source polygon for `land[i]`
  - `drawn_geometry(basemap) -> shapely geometry`: union of `land`, minus union of `lakes`, plus union of `lake_islands`
  - `signed_area(ring) -> float`: positive for a counter-clockwise ring. `compose` emits `land` and `lake_islands` counter-clockwise and `lakes` clockwise.

- [ ] **Step 1: Write the failing tests in `tools/test-basemap.py`**

Each is a function called from `main()`; the script prints `basemap rule tests passed` and exits 0.

```python
def test_tile_bounds():
    w, s, e, n = rule.tile_bounds(8, 227, 93, 228, 95)
    assert (w, e) == (139.21875, 142.03125) and abs(s - 40.9799) < 1e-4 and abs(n - 44.08759) < 1e-4

def test_near_index():
    index = rule.NearIndex([[(0, 0), (10, 0)]], [(100, 10, 101, 11)])
    assert index.near(5, 0.9)            # about 100 km north of the path
    assert not index.near(5, 1.8)        # about 200 km
    assert index.near(100.5, 10.5)       # inside the box, far from any path
    assert not index.near(-170, -60)

def test_simplify_keeps_detail_only_where_near():
    # A 2-degree square whose south and north edges each carry a 0.01-degree zigzag.
    ring = zigzag_square()
    all_near = rule.simplify_ring(ring, [True] * len(ring))
    all_far = rule.simplify_ring(ring, [False] * len(ring))
    south_near = rule.simplify_ring(ring, [lat < 1 for _, lat in ring])
    assert len(all_far) == 5 and all_far[0] == all_far[-1]
    assert len(all_near) > 40
    assert len(all_far) < len(south_near) < len(all_near)
    assert south_near[0] == south_near[-1] and Polygon(south_near).is_valid
    assert all(lat < 1.02 for p in south_near if p not in all_far for lat in [p[1]])   # detail only in the south

def test_across_km():
    assert 150 < rule.across_km([(0, 0), (1, 0), (1, 1), (0, 1), (0, 0)]) < 160

def test_compose_leaves_out_what_the_rule_says():
    index = rule.NearIndex([[(0, 0), (4, 0)]], [])
    polygons = [
        [square(-1, -1, 6, 3), square(1, 0.5, 0.02), square(2, 0.5, 0.2)],   # mainland, 2 km lake, 22 km lake
        [square(2.07, 0.57, 0.01)],          # island in the 22 km lake
        [square(1.005, 0.505, 0.005)],       # island in the 2 km lake, which is left out
        [square(30, 30, 0.05)],              # far island about 7 km across
        [square(40, 30, 0.5)],               # far island about 70 km across
        [square(50, 30, 6, 6), square(51, 31, 0.5), square(53, 31, 2)],  # far land, 70 km lake, 280 km lake
    ]
    out = rule.compose(polygons, index)
    assert len(out["land"]) == 3 and out["sources"]["land"] == [0, 4, 5]
    assert len(out["lakes"]) == 2          # the 22 km near lake and the 280 km far lake
    assert len(out["lake_islands"]) == 1
    assert all(rule.signed_area(r) > 0 for r in out["land"] + out["lake_islands"])
    assert all(rule.signed_area(r) < 0 for r in out["lakes"])
    drawn = rule.drawn_geometry(out)
    assert drawn.contains(Point(2.075, 0.575)) and not drawn.contains(Point(2.15, 0.65))
    assert drawn.contains(Point(1.01, 0.51)) and drawn.contains(Point(51.2, 31.2))
    assert not drawn.contains(Point(30.02, 30.02))

def test_compose_tightens_one_polygon():
    loose = rule.compose(wiggly(), rule.NearIndex([[(0, 0), (4, 0)]], []))
    tight = rule.compose(wiggly(), rule.NearIndex([[(0, 0), (4, 0)]], []), fine_for={0: 0})
    assert len(tight["land"][0]) > len(loose["land"][0])
```

`square(west, south, width, height=None)` returns a closed ring; `zigzag_square()` and `wiggly()` are local fixtures.

- [ ] **Step 2: Run `python3 tools/test-basemap.py`; expect `ModuleNotFoundError: No module named 'basemap_rule'`**

- [ ] **Step 3: Implement `tools/basemap_rule.py`**

Decisions the tests do not fix:
- `NearIndex` samples each path at most every 10 km, stores the samples as unit-sphere vectors in a `scipy.spatial.cKDTree`, and tests a point by chord length `2·sin(km / 6371 / 2)`.
- `simplify_ring` rotates the ring to start where the flag changes, simplifies each run of equal flags as a `LineString(...).simplify(tolerance, preserve_topology=False)` (which keeps its end points), and joins the runs. A tolerance of 0 returns the run unchanged. If the joined ring is not a valid polygon, retry that ring with the coarse tolerance halved, down to the fine tolerance.
- `compose` handles each source polygon as exterior plus holes. Each ring, exterior or hole, is judged on its own vertices: it is near if any of them is near. A land polygon whose representative point lies inside a hole of another polygon is a lake island: emit it only if that lake is kept. A kept lake must stay inside its simplified exterior; if it does not, retry that lake at the fine tolerance.
- Output order is the source order, so the file is reproducible.

- [ ] **Step 4: Run `python3 tools/test-basemap.py`; expect `basemap rule tests passed`**

- [ ] **Step 5: Commit**

```bash
git add tools/basemap_rule.py tools/test-basemap.py
git commit -m "Add the basemap rule: near lines detailed, elsewhere coarse"
```

---


### Task 2: Generate the basemap and draw it

**Files:**
- Create: `tools/prepare-basemap.py`, `pkg/render/basemap.go`, `pkg/render/basemap_test.go`
- Modify: `tools/import-public-network.py` (delete the block from `rings = []` to the end of the `world-land.metadata.json` emit, about lines 777–808), `tools/check-launch-prospects.py:36-38`, `pkg/render/public_network.go` (remove `land`, `landCoast`, `landLayer`, `landCamera`, `landColor`, `drawLand` and the `world-land.json` read), `pkg/render/map.go:173-174`, `pkg/render/simplify_test.go` (`TestWorldViewGeometryStaysNearScreenResolution` reads `basemap.rings`), `tools/test-basemap.py`, `.github/workflows/milestone1.yml` (data job)
- Delete: `tools/launch_shorelines.py`, `pkg/render/coastline_test.go` (it tests regional rings replacing coarse ones inside `world-land.json`, which no longer happens)
- Regenerate: `assets/terrain/world-land.json`, `assets/terrain/world-land.metadata.json`, `docs/validation/launch-prospects-geography.json`

**Interfaces:**
- Consumes: everything Task 1 produces.
- Produces:
  - `assets/terrain/world-land.json`: `{"land": [...], "lakes": [...], "lake_islands": [...]}`, each a list of closed rings of `[lon, lat]`; land and lake islands counter-clockwise, lakes clockwise.
  - `assets/terrain/world-land.metadata.json` with keys `role`, `source_mask_sha256`, `source_countries_sha256`, `source_lakes_sha256`, `line_sources` (path → sha256 for the three line files), `near_km`, `fine_degrees`, `coarse_degrees`, `far_island_km`, `near_lake_km`, `far_lake_km`, `japan_boxes`, `tightened` (list of `{"route", "polygon", "tolerance"}`), `counts` (`land`, `lakes`, `lake_islands`, `points`, `left_out_islands`, `left_out_lakes`), `existing_line_points_off_land` (`before`, `after`), `output_sha256`.
  - `python3 tools/prepare-basemap.py` writes both files; `--check` exits non-zero with `Generated asset differs: <path>` if either differs.
  - In `pkg/render/basemap.go`:

    ```go
    // basemap is the one worldwide land outline.
    type basemap struct {
        rings                [][][2]float64 // land, then lakes, then lake islands
        land, lakes, islands int            // how many of each
        layer                layer
        camera               mapview.Camera
        color                color.NRGBA
    }
    func newBasemap() (*basemap, error)
    func (b *basemap) draw(dst *ebiten.Image, c mapview.Camera, pal ui.Palette)
    ```
  - `Map.basemap *basemap`, set in `render.New`.

- [ ] **Step 1: Add the tool's failing tests to `tools/test-basemap.py`**

```python
def test_offshore_route_stops_the_tool():
    tool = load("prepare-basemap.py")           # importlib, as check-launch-prospects.py loads its sibling
    polygons = [[square(0, 0, 1)]]
    routes = [("test-route", "launch", [(0.5, 0.5), (1.5, 0.5)], None)]   # id, alignment, path, allowed crossing
    try:
        tool.fit(polygons, rule.NearIndex([r[2] for r in routes], []), routes)
    except SystemExit as e:
        assert "test-route" in str(e) and "1.5" in str(e)
    else:
        raise AssertionError("an offshore route was accepted")

def test_fit_tightens_only_the_polygon_a_route_touches():
    basemap, tightened = tool.fit(bay_polygons(), index, bay_routes())   # the route hugs a bay 0.003 degrees would fill
    assert tightened == [{"route": "bay-route", "polygon": 0, "tolerance": 0.001}]
    assert rule.drawn_geometry(basemap).covers(LineString(bay_routes()[0][2]))
```

Run `python3 tools/test-basemap.py`; expect `FileNotFoundError` for `prepare-basemap.py`.

- [ ] **Step 2: Implement `tools/prepare-basemap.py`**

- `fit(polygons, index, routes) -> (basemap, tightened)`: compose; find each route whose path is not covered by `drawn ∪ allowed` in geographic coordinates, or whose projected path is not covered by the projected union (`rule.project`); for each, move every source polygon the path intersects one step down `rule.LADDER` through `fine_for`; repeat. When a failing route's polygons are all at 0, raise `SystemExit` naming the route, the alignment and the first off-land point.
- `main()`: read the mask's `land`, the three line sources (Global Constraints), and the reviewed crossings (`compiled-prospects.v1.json` → `review[].crossings`, keyed by `corridor_id`; the two older routes have none). `routes` is every alignment of the 74 buildable routes: the 72 in `compiled-prospects.v1.json` and both alignments of each older route. Build `NearIndex(paths, [rule.tile_bounds(*b) for b in rule.JAPAN_BOXES])`. Call `fit`. Assert the budget. Count sampled existing-line points (every 10 km) off drawn land against the committed file (`before`, accepting both the old flat list and the new shape) and the new one (`after`). Write or `--check` both outputs with the byte format `json.dumps(obj, separators=(",", ":")) + "\n"`; metadata with `indent=2`.

Run `python3 tools/test-basemap.py`; expect `basemap rule tests passed`.

- [ ] **Step 3: Remove the old emission**

In `tools/import-public-network.py` delete the block that builds `rings` from `source["countries"]`, calls `refine`, and emits `world-land.json` and `world-land.metadata.json`. Delete `tools/launch_shorelines.py`. In `tools/check-launch-prospects.py` replace the `drawn = unary_union([...world-land.json...])` expression with `rule.drawn_geometry(read("assets/terrain/world-land.json"))`, importing `basemap_rule` as `rule`.

- [ ] **Step 4: Write the failing Go tests in `pkg/render/basemap_test.go`**

Helpers: `ringArea(ring [][2]float64) float64` (signed, positive counter-clockwise), `drawnLand(b *basemap, lon, lat float64) bool` (inside a land ring and not inside a lake, or inside a lake island) and `inLake(b, lon, lat) bool`, built on the existing `ringContains`.

```go
func TestBasemapMatchesItsMetadata(t *testing.T)
// sha256 of terrain/world-land.json equals metadata.output_sha256; total points <= 80000; outlines <= 2200;
// b.land, b.lakes and b.islands are all > 0 and sum to len(b.rings);
// every ring has >= 4 points, first == last, lon in [-180,180], lat in [-90,90].

func TestBasemapOutlinesWindSoLakesAreCutOut(t *testing.T)
// ringArea > 0 for every land ring and lake island, < 0 for every lake.

func TestBasemapHasNoEdgeAcrossTheMap(t *testing.T)
// no consecutive pair of points in any ring differs by more than 180 degrees of longitude.

func TestBasemapKnownPoints(t *testing.T)
// land:  Istanbul Europe (28.95, 41.02), Istanbul Asia (29.06, 41.00), Tangier (-5.80, 35.76),
//        Folkestone (1.15, 51.10), Bogota (-74.07, 4.71), Manitoulin Island (-82.2, 45.75)
// sea:   San Francisco Bay (-122.25, 37.70), Thames estuary (0.90, 51.48), Gulf of Izmit (29.60, 40.73),
//        Osaka Bay (135.25, 34.55), and the eight water points tools/check-coastlines.cjs samples
//        (lat, lon there): (140.09867836, 42.95577938), (138.1263546, 37.31509792), (141.60869049, 40.63917142),
//        (140.54059505, 41.59805279), (121.15774495, 30.47348219), (119.10184264, 37.29738934),
//        (121.17869533, 28.26677417), (110.48955745, 20.07817143)
// lake:  Lake Michigan (-87.0, 43.5), Lake Biwa (136.08, 35.25), Lake Geneva (6.55, 46.45)

func TestCatalogueCitiesAreOnDrawnLand(t *testing.T)
// every city in assets.PublicNetwork ("cities": station_lon, station_lat) is on drawn land, except ids in
// a map[string]string of id -> reason. The test also fails for an exception that is on drawn land.
```

The first thirteen known points were checked against the full-detail source on 2026-11-14 and give the stated answer. If one fails, the outline is wrong, not the test. The eight checker points are water in today's basemap by that checker's own evidence.

Run the Go test command; expect a build failure: `undefined: basemap`.

- [ ] **Step 5: Implement `pkg/render/basemap.go` and switch the renderer to it**

`newBasemap` reads `terrain/world-land.json` from `assets.Terrain` and joins the three lists in order. `draw` is the removed `publicLayer.drawLand`: redraw `b.layer` only when the layer is empty or the camera or `pal.Land` changed, by calling `drawLand(layer, c, b.rings, coast, pal)` with every entry of `coast` true, then draw the layer's view. Because lakes wind the other way, the one nonzero fill leaves them clear and the scene's water fill shows through; the one coast path outlines land, lakes and lake islands alike. In `render.New` set `m.basemap`; in `drawScene` replace `m.public.drawLand(dst, c, s)` with `m.basemap.draw(dst, c, pal)`, leaving the `else` branch for Task 3.

- [ ] **Step 6: Generate the basemap and run the checks, one at a time**

```bash
python3 tools/prepare-basemap.py
python3 tools/prepare-basemap.py --check
python3 tools/import-public-network.py --check
python3 tools/check-launch-prospects.py
```

Expected: the first prints the counts; the second and third print nothing that says `differs`; the fourth ends with its usual pass line. Then run the Go test command. `TestCatalogueCitiesAreOnDrawnLand` will name the cities that fail: for each, confirm against the mask at full detail that its station marker really is in the sea (the design expects five to nine, among them Hong Kong, Dongguan, Zhuhai, Lisbon and Miami), then add it to the exception map with the reason. A city that is on land at full detail but off the drawn land is a defect in the rule: fix the rule, not the list. If `TestWorldViewGeometryStaysNearScreenResolution` exceeds 22,000 land vertices, raise the far-island or far-lake size threshold; do not raise the limit.

Expected after that: PASS.

- [ ] **Step 7: Add the new checks to CI's data job**

In `.github/workflows/milestone1.yml`, the `preparation` lane gains `tools/test-basemap.py` and `'tools/prepare-basemap.py --check'`.

- [ ] **Step 8: Confirm nothing else changed, then commit**

`git status --short` must list only the files named in this task. `git diff --no-ext-diff --stat -- assets/scenario data` must be empty.

```bash
git add -A tools pkg/render assets/terrain/world-land.json assets/terrain/world-land.metadata.json docs/validation/launch-prospects-geography.json .github/workflows/milestone1.yml
git commit -m "Draw one worldwide land outline with lakes"
```

---

### Task 3: One basemap for every frame; retire the regional files

**Files:**
- Modify: `pkg/render/map.go` (remove `rings`, `ringCoast`, their load, and the `else` branch in `drawScene`; add `land`), `pkg/render/land.go` (`drawLand` loses its `coast` parameter; `coastRings` goes; `ringContains` moves to `basemap_test.go`), `pkg/render/simplify_test.go` (remove `TestCoastRingsHideOnlyOutlinesALaterRingPaintedOver`), `assets/assets.go` (remove `Land`), `pkg/render/terrain.go` (stop drawing `lakes`; remove the field and its load in `newRelief`), `tools/check-route-geography.py:55-60,170`, `tools/prepare-route-geography.py:10-12`, `tools/prepare-terrain.py:167-169`, `tools/prepare-skeleton-assets.py` (remove the `land.json` block ending at line 44), `tools/check-coastlines.cjs:113`, `pkg/render/basemap_test.go`
- Delete: `assets/terrain/land.json`, `assets/terrain/land.metadata.json`, `assets/scenario/land.json`, `tools/prepare-shorelines.py`
- Regenerate: `docs/validation/route-geography-checks.json`

**Interfaces:**
- Consumes: `Map.basemap`, `rule.drawn_geometry`, the metadata key `source_countries_sha256`.
- Produces: `func (m *Map) land(f gameapi.Frame) *basemap`, which `drawScene` uses for its land; `func drawLand(dst *ebiten.Image, c mapview.Camera, rings [][][2]float64, pal ui.Palette)`.

- [ ] **Step 1: Write the failing Go tests in `pkg/render/basemap_test.go`**

```go
func TestEveryFrameDrawsTheOneBasemap(t *testing.T) {
    m, err := New()
    // ...
    with, without := m.land(gameapi.Frame{PublicNetwork: true}), m.land(gameapi.Frame{})
    if with == nil || with != without || len(with.rings) == 0 { t.Fatal(...) }
}

func TestReliefNoLongerDrawsItsOwnLake(t *testing.T)
// reflect.TypeOf(relief{}) has no field named "lakes"; inLake(m.basemap, 136.08, 35.25) is true.
```

Run the Go test command; expect a build failure: `m.land undefined`.

- [ ] **Step 2: Make the renderer use one basemap**

`drawScene` becomes `dst.Fill(pal.Water); m.land(f).draw(dst, c, pal)` with no branch. Remove `Map.rings`, `Map.ringCoast`, `assets.Land`, `coastRings` and `relief.lakes`. `pkg/terrain` and `features.json` are unchanged; `terrain.Data.Lakes` stays because `tools/prepare-route-geography.py` and `pkg/terrain` tests use the file. In `land.go`'s comment, the note about slivers between neighbouring country rings no longer applies; say what is true now.

- [ ] **Step 3: Repoint the Python tools and the browser checker**

- `tools/check-route-geography.py`: `coastline` is `assets/terrain/world-land.json`; its metadata is `world-land.metadata.json`; the source assertion compares `coast_meta["source_countries_sha256"]` with `meta["coast_source_sha256"]`; `drawn` is `rule.drawn_geometry(...)`. The result key stays `drawn_coast_sha256`.
- `tools/prepare-route-geography.py`: read `source_countries_sha256` from `world-land.metadata.json` in place of `source_sha256` from `land.metadata.json`. Do not run it.
- `tools/prepare-terrain.py`: delete the closing `subprocess.run([... "prepare-shorelines.py"])` and its comment.
- `tools/check-coastlines.cjs:113`: `all = JSON.parse(fs.readFileSync('assets/terrain/world-land.json')).land`. Its eight water points are pinned as sea by Task 2's Go test, and its two coast boxes (Hokkaido, and Shanghai at lon 120–122.5, lat 29–32) lie near lines, so they stay detailed. Not run locally.

- [ ] **Step 4: Delete the retired files, then run everything**

```bash
git rm -q assets/terrain/land.json assets/terrain/land.metadata.json assets/scenario/land.json tools/prepare-shorelines.py
python3 tools/check-route-geography.py
python3 tools/check-launch-prospects.py
python3 -m py_compile tools/prepare-route-geography.py tools/prepare-terrain.py tools/prepare-skeleton-assets.py
node --check tools/check-coastlines.cjs
grep -rn "terrain/land\.json\|scenario/land\.json\|land\.metadata\.json\|prepare-shorelines\|launch_shorelines\|assets\.Land\b\|coastRings" \
  --include='*.go' --include='*.py' --include='*.cjs' --include='*.sh' --include='*.yml' .
```

Expected: both checks pass, with `check-route-geography.py` reporting its six alignments; `py_compile` and `node --check` print nothing; `grep` prints nothing outside `build/` and `.claude/`. If `check-route-geography.py` reports a sea or lake intersection for one of the four older alignments, that alignment is off the new drawn land: go back to `tools/prepare-basemap.py`, whose `fit` should have tightened it, and find why it did not. Then run the Go test command; expect PASS.

- [ ] **Step 5: Commit**

```bash
git add -A pkg/render assets tools docs/validation/route-geography-checks.json
git commit -m "Draw the one basemap in every frame and retire the regional files"
```

---

### Task 4: Skip what is off screen, and measure

**Files:**
- Modify: `pkg/render/basemap.go`, `pkg/render/land.go`, `pkg/render/clip.go`, `pkg/render/clip_test.go`, `pkg/render/simplify_test.go`, `pkg/render/basemap_test.go`
- Create (not committed): timing overlays in the session scratchpad

**Interfaces:**
- Consumes: `basemap`, `drawLand`.
- Produces:
  - `type outline struct { points []mapview.Point; min, max mapview.Point }`: an outline projected once at load, with its bounds. `basemap.outlines []outline` sits beside `rings`, one per ring.
  - `func (b *basemap) visible(c mapview.Camera) [][]mapview.Point`: the outlines to draw, in order, each clipped to the screen and simplified; an outline whose bounds lie wholly off screen is skipped without touching its points.
  - `func clipRing(c mapview.Camera, projected []mapview.Point) []mapview.Point` replaces `screenRing`.
  - `func drawLand(dst *ebiten.Image, outlines [][]mapview.Point, pal ui.Palette)` draws what `visible` returns, still as one fill path and one coast path.

- [ ] **Step 1: Write the failing tests**

```go
func TestBasemapSkipsOutlinesOffScreen(t *testing.T)
// camera := mapview.Fit(1280, 850, Tokyo and Osaka projected), Zoom 8:
//   0 < len(b.visible(camera)) < len(b.rings)/10, and every returned outline has >= 3 points inside
//   [-4, Width+4] x [-4, Height+4].
// a camera fitted on two mid-Pacific points (-150, 0) and (-149, 1), Zoom 32: len(b.visible(camera)) == 0.
// a camera with Width 0 and Height 0: len(b.visible(camera)) == 0 and no panic.

func TestProjectedPointsMatchTheRing(t *testing.T)
// for every outline, len(points) == len(ring), points[i] == mapview.Project(ring[i][1], ring[i][0]),
// and min/max bound every point.
```

`clip_test.go` calls `clipRing` with projected input in place of `screenRing`; `TestWorldViewGeometryStaysNearScreenResolution` counts the points `visible` returns. Run the Go test command; expect a build failure: `b.visible undefined`.

- [ ] **Step 2: Implement**

Bounds test: map the outline's `min` and `max` through `c.Screen` (it is a scale and a shift, so the box maps to a box) and skip when it does not meet `[-4, Width+4] × [-4, Height+4]`.

Run the Go test command; expect PASS.

- [ ] **Step 3: Measure land redraw time and memory before and after**

Follow the recipe in the memory note `measure-native-memory-on-a-save-copy`: build with `go build -overlay`, run on a copy of the save under a scratch `HOME`, never the real one. Baseline is commit `b6ecac4` in `git worktree add --detach <scratch>/base b6ecac4`; remove that worktree afterwards.

The overlay adds, in each tree, a file `pkg/render/exp_timing.go` holding the samples and a function that prints count, mean and 95th percentile in milliseconds to stderr every 300 samples, and replaces the file holding the land layer's cache-miss path with a copy that times it with `time.Now()` / `time.Since` (`publicLayer.drawLand` at baseline, `basemap.draw` now). A second overlaid file, `pkg/app/host_native.go`, adds one line at the top of `host.update`: `g.Camera.PanX += 1`, so every frame redraws the scene.

Run each build for 30 seconds at the saved window size (3737×2029), once on the World view and once on the Japan region. Record the four mean and 95th-percentile figures, and each run's physical footprint from `vmmap --summary` at the end.

Pass: no timing for the new build is more than twice its baseline, and no footprint is more than 100 MB above its baseline. `land.go` records how sensitive Ebitengine's stencil and vertex buffers are; a footprint jump means the two-pass rule was broken somewhere.

- [ ] **Step 4: Commit**

```bash
git add pkg/render
git commit -m "Skip basemap outlines that are off screen"
```

Put the measured figures in the commit body.

---

### Task 5: Documents and final verification

**Files:**
- Modify: `assets/terrain/README.md`, `assets/terrain/ATTRIBUTION.md`, `assets/scenario/README.md:7`, `ARCHITECTURE.md` (the shoreline paragraph), `data/imports/README.md:20,145,200`, `TESTING.md` (the alignment-geometry, legacy-geometry and coastline passages), `docs/validation/README.md:83`, `docs/superpowers/specs/2026-11-14-coastline-refinement-design.md` (Status, Constraints and Evidence), `docs/superpowers/specs/2026-11-14-worldwide-terrain-design.md` (the browser-check non-goal)

**Interfaces:**
- Consumes: the counts in `world-land.metadata.json`, the exception list from Task 2, the timings from Task 4.

- [ ] **Step 1: Update the documents**

Each passage that describes per-country outlines, the regional files, `prepare-shorelines.py`, `launch_shorelines.py` or "legacy fixtures draw the regional layer" is rewritten to say what is now true:
- one land outline from the `land` geometry of `route-land.v1.json.gz`, with the tolerances, the 150 km distance, the size thresholds and the real counts;
- no lines between countries; lakes are holes in the one fill;
- every frame draws it;
- how to regenerate and check it (`python3 tools/prepare-basemap.py`, `--check`, `python3 tools/test-basemap.py`);
- `ATTRIBUTION.md` credits Natural Earth 1:10m countries and lakes for the whole basemap and drops the per-country paragraphs;
- `docs/validation/coastline-checks.json` and its screenshots are refreshed by CI, not here.

In the coastline design, set Status to implemented; replace "Not measured" in Evidence with the real counts, the tightened routes, the city exceptions, the existing-line points off land before and after, and the timings and footprints; and bring Constraints up to date (land is drawn by `land.go` in two passes; the browser check runs in CI). In the terrain design, the non-goal about not running the browser check is replaced by: CI runs the browser checks on push, and they are not run locally.

- [ ] **Step 2: Run the full verification, one command at a time**

```bash
python3 tools/test-basemap.py
python3 tools/prepare-basemap.py --check
python3 tools/check-boundaries.py
gofmt -l assets cmd internal pkg
go vet ./...
GOOS=js GOARCH=wasm go vet ./...
GOMEMLIMIT=768MiB GOMAXPROCS=2 go test -p 1 ./... -count=1
sh tools/build-native.sh
sh tools/build-web.sh
git diff --no-ext-diff --stat b6ecac4 -- assets/scenario/public-network.json assets/scenario/playable-rail-map.json data assets/terrain/features.json
```

Expected: the first prints `basemap rule tests passed`; `gofmt -l` and the last `git diff` print nothing; everything else exits 0. Check each command's own exit status; do not pipe a test into a commit.

- [ ] **Step 3: Look at it in the native app**

Run the built app on a copy of the save (scratch `HOME`) and look at: the Sea of Marmara (Istanbul on land, the line no longer over open water), San Francisco Bay, Lake Biwa with terrain on and off, the Great Lakes, and the World view in light and dark. Screenshot each for the user. This is the only local check of the drawn result; the Go tests do not draw, and the browser check runs only in CI.

- [ ] **Step 4: Commit**

```bash
git add -A assets ARCHITECTURE.md TESTING.md data/imports/README.md docs
git commit -m "Document the single-outline basemap"
```

Report to the user: what ran, the counts, the timings and footprints, the exceptions, and that the browser coastline check has not run and will first run in this branch's CI.
