# Worldwide Terrain Relief Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Show shaded relief at Japan's detail within 60 km of every line in the game and coarser relief on all other land, loaded as the player looks, with memory that depends on the window and not on coverage.

**Architecture:** A reworked Python generator writes a pyramid of 256×256 Web Mercator tiles (levels 2 to 8) as JPEG colour plus a one-bit land mask, cut to the drawn basemap, with a manifest. `pkg/terrain` parses the manifest at startup, decodes nothing, and offers pure functions for choosing a level, covering a view and decoding one tile. `pkg/render` decodes tiles on a worker goroutine, uploads a few per frame into a fixed-size cache that frees evicted textures explicitly, and draws each visible area once from the finest tile that is ready.

**Tech Stack:** Go 1.x standard library (`image/jpeg`, `image/png`, `embed`), Ebitengine v2.10.5 (`DrawTriangles`), Python 3.12 with Pillow 13.0.0, numpy 2.2.6, scikit-image 0.26.0, shapely 2.1.2 (all already pinned in `tools/requirements-terrain.txt`).

**Spec:** `docs/superpowers/specs/2026-11-14-worldwide-terrain-design.md`. Read it, including "After the coastline build", before Task 1.

## Global Constraints

- No new Go dependency. JPEG and PNG decoding use the standard library.
- `pkg/terrain` imports no graphics package; `cmd/headless` stays graphics-free. `tools/check-boundaries.py` enforces both after Task 4.
- Go tests run as `GOMEMLIMIT=768MiB GOMAXPROCS=2 go test -p 1 ./... -count=1` and must pass. Go tests cannot read pixels back; they test geometry, counts and decoded Go images.
- Terrain never enters scenario hashes, saves, costs or feasibility. No wording or visual suggests relief proves a route feasible.
- Tiles pushed out of the cache are freed with `Deallocate`. No tile texture is left for the garbage collector.
- With terrain hidden, no tile is decoded or uploaded.
- Startup decodes no tile.
- Level 8 is the finest level. Land opacity in a tile is 225/255. The existing zoom opacity `min(0.98, 0.42 + 0.075·Zoom)` and dark tint `(0.30, 0.38, 0.40)` are unchanged.
- JPEG quality 80. Land means elevation above 1 m and inside the drawn basemap (`rule.drawn_geometry` of `assets/terrain/world-land.json`).
- "A line" is every path in `assets/scenario/playable-rail-map.json`, every alignment path in `data/construction/compiled-prospects.v1.json`, and every alignment path of `hokkaido-extension` and `tsuruga-osaka` in `assets/scenario/public-network.json`. `tools/prepare-basemap.py` `inputs()` already reads exactly these.
- Japan boxes at level 8: x 227–228, y 93–95 and x 224–225, y 100–101 (`rule.JAPAN_BOXES`).
- The generator runs in an isolated environment built from `tools/requirements-terrain.txt` (the machine's default Python has a different numpy). Ordinary builds need no Python and no network.
- Three stops need the user and are marked **STOP**: the bulk download (Task 2), any uncredited elevation dataset (Task 2), and approval of the look (Task 7). Do not work past them.
- Browser checkers: follow the footprint rule in `TESTING.md`. `check-terrain.cjs` usually does not fit locally; hosted CI is its test.
- Do not commit locally generated files in `docs/validation`.
- Commit messages end with `Co-Authored-By: Claude Fable 5.2 <noreply@anthropic.com>`.

## Decisions this plan makes (the spec left them to the builder)

| Question | Decision |
|---|---|
| File layout | One file per tile under `assets/terrain/tiles/{z}/{x}/{y}.jpg`, mask beside it as `{y}.png`. Adding a line then adds files instead of rewriting one 22 MB pack. |
| Mask format | 1-bit greyscale PNG, 256×256, white = land. Present only on tiles that are part land. |
| Manifest | `assets/terrain/terrain.json`, `schema_version` 2 (shape in Task 3). |
| Source pins | `data/terrain/sources.v1.json`, not embedded. |
| Levels 6 and 7 | Fetched from the source at their own level, like every other level. |
| Level switch | One level per view: the smallest level whose tiles are at most 512 screen pixels wide at the view's centre latitude. |
| "Drawn exactly once" | Draw part of a parent tile where a child is absent or not yet ready. No offscreen image. |
| Mesh | 8×8 cells per drawn area, whatever its level. |
| Cache | 288 tiles (75 MB), least recently drawn evicted first, tiles drawn this frame never evicted. Each tile is an unmanaged 256×256 image. |
| Decoding | One worker goroutine on both targets; at most 8 uploads per frame. |
| Band edge | No special treatment. Sharp tiles end where coarser ones begin; both come from the same shading, so brightness does not step. |
| Hillshade spacing | Metres per pixel computed per pixel row, so shading is continuous across tile rows at every level. |

## Review Focus

- **A view that crosses the edge of level-8 coverage.** The player expects sharp relief to give way to softer relief with no line, gap or doubled strip. Test: `TestCoverDrawsEveryAreaOnce` (Task 4).
- **Panning fast enough that no wanted tile is ready.** The player expects coarser relief or plain land, never a hole in the land and never a stall. Tests: `TestCoverFallsBackToTheFinestReadyAncestor` (Task 4), `TestCacheNeverEvictsATileDrawnThisFrame` (Task 5).
- **A line over open sea (a strait tunnel).** Coverage must not fail there and nothing must be drawn. Tests: `TestSeaTilesCountAsCoverageAndDrawNothing` (Task 4), `test_all_sea_tile_is_listed_without_files` (Task 3).
- **A corrupt or truncated tile file.** The player expects that one tile to stay coarse and the game to keep running; a developer expects an error. Test: `TestDecodeRejectsATileThatFailsItsHash` (Task 4), `TestLoaderReportsAFailedTileOnceAndKeepsGoing` (Task 5).
- **Relief overhanging the drawn coast on long far-coast edges.** Basemap edges are straight on the Equal Earth map but the mask is cut in Mercator pixels. Test: `test_mask_follows_map_straight_edges` (Task 1).

---

### Task 1: Generator rules (pure functions)

**Files:**
- Create: `tools/terrain_rule.py`
- Create: `tools/test-terrain.py` (plain asserts, prints `terrain rule tests passed`, same style as `tools/test-basemap.py`)

**Interfaces:**
- Consumes: `tools/basemap_rule.py`: `sample(path, step_km)`, `tile_bounds(z, x0, y0, x1, y1)`, `drawn_geometry(basemap)`, `JAPAN_BOXES`.
- Produces (all in `terrain_rule.py`):
  - `STOPS = [0, 150, 400, 800, 1200, 1800, 2600, 3600, 4800]` and `COLORS` (the spec's table, in that order); `LOWLAND = (231, 244, 204)`; `NEAR_KM = 60`; `LEVELS = range(2, 9)`.
  - `level8_tiles(lines) -> set[tuple[int, int]]`: sample each path every 5 km; for each sample add every level-8 tile touched by the box of ±62.5 km around it (`dlat = 62.5 / 111.2`, `dlon = dlat / cos(lat)`); then add the Japan boxes.
  - `parents(tiles) -> set[tuple[int, int]]`: `(x >> 1, y >> 1)`.
  - `land_tiles(z, geometry) -> set[tuple[int, int]]`: tiles at level `z` that intersect `geometry` cut to latitudes north of 60° S.
  - `tile_list(lines, geometry) -> dict[int, set[tuple[int, int]]]`: levels 2–5 from `land_tiles`, 8 from `level8_tiles`, 7 and 6 from `parents`.
  - `elevation(rgb) -> ndarray`: Terrarium decode, `r*256 + g + b/256 - 32768`.
  - `shade(heights, z, y, halo) -> ndarray uint8 (256, 256, 3)`: `heights` is 258×258 when `halo` is true (one pixel from each neighbour) and 256×256 edge-replicated otherwise. Gradient spacing is per row: `cos(lat_row) * 2π * 6378137 / (256 * 2**z)`. Light and exaggeration as today: normals `(-dx*2.5, dy*2.5, 1)`, `clip(0.74 + 0.34 * (n · (-0.5, 0.5, 0.7071)), 0.55, 1.10)`. Colour by `np.interp` over `STOPS`/`COLORS`.
  - `land_mask(heights, geometry, z, x, y) -> ndarray bool (256, 256)`: `heights > 1` and inside `geometry`. Rasterise by transforming `geometry` to this tile's pixel coordinates, after densifying every edge so no segment is longer than 2 km along the straight line between its Equal Earth projections (`rule.project`), then `PIL.ImageDraw` polygons with holes.
  - `kind(mask) -> "land" | "coast" | "sea"`.
  - `datasets(imagery_sources: str) -> set[str]`: the part before `/` of each comma-separated entry.

- [ ] **Step 1: Write the failing tests** in `tools/test-terrain.py`:

```python
def test_colour_scale_matches_the_design():
    assert T.STOPS == [0, 150, 400, 800, 1200, 1800, 2600, 3600, 4800]
    assert T.COLORS[0] == [231, 244, 204] and T.COLORS[6] == [166, 141, 128]
    assert T.COLORS[7] == [196, 186, 180] and T.COLORS[8] == [244, 243, 240]

def test_level8_covers_60_km_either_side_of_a_line():
    # A line along the equator from 0E to 1E: 60 km is 0.54 degrees, a level-8 tile is 1.40625 degrees.
    tiles = T.level8_tiles([[(0.0, 0.0), (1.0, 0.0)]])
    assert (128, 127) in tiles and (128, 128) in tiles      # north and south of the equator
    assert (127, 127) in tiles                               # 60 km west of the start
    assert (130, 127) not in tiles                           # starts at 2.8E, more than 60 km east of the end

def test_japan_boxes_are_always_covered():
    assert {(227, 93), (228, 95), (224, 100), (225, 101)} <= T.level8_tiles([])

def test_every_fine_tile_has_its_parents():
    levels = T.tile_list([[(0.0, 0.0), (1.0, 0.0)]], shapely.geometry.box(-10, -10, 10, 10))
    assert all((x >> 1, y >> 1) in levels[7] for x, y in levels[8])
    assert all((x >> 1, y >> 1) in levels[6] for x, y in levels[7])

def test_coarse_levels_cover_land_north_of_60_south_only():
    antarctica, europe = shapely.geometry.box(0, -80, 10, -70), shapely.geometry.box(0, 42, 10, 50)
    assert T.land_tiles(4, antarctica) == set()
    assert T.land_tiles(4, europe) == {(8, 5)}

def test_shading_is_continuous_across_a_tile_edge():
    # One smooth slope split over two side-by-side tiles shades the same as when shaded in one piece.
    ...  # build a 256x512 ramp, shade each half with halo=True, compare the seam columns: max difference <= 1

def test_flat_ground_gets_the_plain_colour_times_flat_shade():
    rgb = T.shade(np.full((258, 258), 400.0), 8, 100, True)
    assert tuple(rgb[128, 128]) == tuple(int(c * (0.74 + 0.34 * 0.7071)) for c in (199, 211, 145))

def test_mask_needs_both_height_and_drawn_land():
    land = shapely.geometry.box(0, 0, 0.7, 1.5)   # tile 8/128/127 spans 0 to 1.40625 E, 0 to 1.4061 N: its western half
    mask = T.land_mask(np.full((256, 256), 50.0), land, 8, 128, 127)
    assert mask[128, 10] and not mask[128, 250]
    assert not T.land_mask(np.full((256, 256), 0.5), land, 8, 128, 127).any()

def test_mask_follows_map_straight_edges():
    # A 300 km edge at 60N: the mask boundary at mid-edge lies within 1 pixel of the Equal Earth chord.
    ...

def test_tile_kind():
    assert T.kind(np.ones((256, 256), bool)) == "land" and T.kind(np.zeros((256, 256), bool)) == "sea"

def test_datasets_are_read_from_the_source_header():
    assert T.datasets("srtm/N43E140.tif, gmted/30N120E_20101117_gmted_mea075.tif, etopo1/ETOPO1_Bed_g.tif") == {"srtm", "gmted", "etopo1"}
```

Write the two elided tests in full; their comments say what they assert.

- [ ] **Step 2: Run to verify they fail.** Run: `<venv>/bin/python tools/test-terrain.py`. Expected: `ModuleNotFoundError: No module named 'terrain_rule'`.
- [ ] **Step 3: Implement `tools/terrain_rule.py`** with the signatures above.
- [ ] **Step 4: Run to verify they pass.** Expected: `terrain rule tests passed`.
- [ ] **Step 5: Commit** `tools/terrain_rule.py tools/test-terrain.py` as "Add the rules for worldwide terrain tiles".

---

### Task 2: Fetch the source tiles and survey their datasets

**Files:**
- Modify: `tools/prepare-terrain.py` (rewritten in this task and the next; today's region loop goes)
- Create: `data/terrain/sources.v1.json`
- Modify: `tools/test-terrain.py`

**Interfaces:**
- Consumes: Task 1's `tile_list`, `datasets`; `prepare-basemap.py`'s `inputs()` for the lines (import it with `importlib`, as `check-route-geography.py` imports the rule).
- Produces:
  - `prepare-terrain.py --plan`: prints the tile count per level and the number of source tiles to fetch. No network.
  - `prepare-terrain.py --fetch`: downloads missing source tiles to `build/terrain-source/{z}-{x}-{y}.png` with their `.headers` file, exactly as today's `curl` call does, and fails with `Source changed: <url>; review before repinning` if a cached or downloaded tile differs from its pin.
  - `prepare-terrain.py --survey`: reads the cached headers and prints each dataset name with its tile count, then `uncredited: <names>` and exits 1 if any name is not in `CREDITED`.
  - `CREDITED = {"srtm": "SRTM", "gmted": "GMTED2010", "etopo1": "ETOPO1"}` in `terrain_rule.py`, and `uncredited(names, attribution_text) -> set[str]`, which also returns a credited key whose label is missing from `ATTRIBUTION.md`.
  - `data/terrain/sources.v1.json`: `{"version": 1, "tiles": {"8/227/93": {"url", "sha256", "imagery_sources", "last_modified"}, ...}}`, keys sorted. Today's 10 pins move here from `terrain.json` unchanged.

- [ ] **Step 1: Write the failing tests:**

```python
def test_uncredited_datasets_are_named():
    text = "SRTM and GMTED2010 from USGS; ETOPO1 from NOAA."
    assert T.uncredited({"srtm", "gmted", "etopo1"}, text) == set()
    assert T.uncredited({"srtm", "ned", "eudem"}, text) == {"ned", "eudem"}
    assert T.uncredited({"srtm"}, "nothing here") == {"srtm"}
```

- [ ] **Step 2: Run to verify it fails.** Expected: `AttributeError: module 'terrain_rule' has no attribute 'uncredited'`.
- [ ] **Step 3: Implement** `CREDITED`, `uncredited`, and the `--plan`, `--fetch` and `--survey` modes. Move the 10 existing pins into `data/terrain/sources.v1.json`.
- [ ] **Step 4: Run the tests and `--plan`.** Expected: tests pass; `--plan` prints seven levels, with level 8 between 900 and 1,100 tiles, level 4 near 102 and level 5 near 338 (the spec's counts). If a count is outside those ranges, find out why before going on.
- [ ] **Step 5: STOP — ask the user before downloading.** Report the `--plan` output: the number of files, the host (`s3.amazonaws.com/elevation-tiles-prod/terrarium`), and that the cache will be a few hundred megabytes in `build/terrain-source`. Run `--fetch` only on a yes.
- [ ] **Step 6: Run `--survey`.** If it exits 0, go on. **STOP** if it prints `uncredited:`. Report each dataset name, its tile count and its licence terms (from `https://github.com/tilezen/joerd/blob/master/docs/attribution.md`) to the user. A new credit is the user's decision. If they approve one, it changes `CREDITED`, `assets/terrain/ATTRIBUTION.md`, the legend caption `"Mapzen / USGS"` in `pkg/render/terrain.go`, and the `includes('Mapzen Terrain Tiles')` assertion in `tools/check-terrain.cjs`; make those changes in this task and rerun `--survey` to exit 0.
- [ ] **Step 7: Commit** the tool, the test and `data/terrain/sources.v1.json` as "Fetch and pin the source tiles for worldwide terrain".

---

### Task 3: Write tiles, masks and the manifest

**Files:**
- Modify: `tools/prepare-terrain.py`, `tools/test-terrain.py`
- Create (generated, not committed until Task 8): `assets/terrain/tiles/**`, `assets/terrain/terrain.json`

**Interfaces:**
- Consumes: Tasks 1 and 2.
- Produces:
  - `prepare-terrain.py --write [--only CORRIDOR_ID ...] [--box W S E N ...]`: writes tiles for the whole list, or, with `--only` and `--box`, for the level-8 tiles of the named buildable corridors and of the given longitude/latitude boxes, plus their parents, plus all of levels 2–5. Removes tile files that are no longer in the list. Refuses to write while `--survey` would fail.
  - `write_tile(out, z, x, y, rgb, mask) -> dict`: writes `{y}.jpg` (quality 80, `optimize=True`, sea pixels set to `LOWLAND` first) unless the tile is sea, writes `{y}.png` (mode `"1"`, `optimize=True`) only for a coast tile, and returns the manifest entry.
  - Manifest `assets/terrain/terrain.json`, written with `separators=(",", ":")` and tiles sorted by `(z, x, y)`:

```json
{"schema_version":2,
 "role":"Display context only; never pricing, feasibility or route generation.",
 "source":"Mapzen Terrain Tiles / USGS SRTM + GMTED2010, NOAA ETOPO1",
 "source_documentation":"https://github.com/tilezen/joerd/blob/master/docs/formats.md",
 "attribution":"Elevation: Mapzen / USGS; bathymetry: NOAA. All overlays reprojected to Equal Earth.",
 "basemap_sha256":"<sha256 of assets/terrain/world-land.json>",
 "complete":true,
 "tiles":[[8,227,93,"coast","<jpg sha256>","<mask sha256>"],[8,227,94,"land","<jpg sha256>"],[8,226,96,"sea"]],
 "regions":[{"id":"hokkaido","contours":[{"metres":500,"points":[[lat,lon],...]}]},{"id":"kansai","contours":[...]}]}
```

  `complete` is false when `--only` was used. Contours are computed as today (500, 1,000 and 1,500 m, `heights[::4, ::4]`, `simplify(1.5)`, lines under 8 points dropped) from the level-8 heights of each Japan box.
  - `accessed_on` is dropped from the manifest, so regenerating on another day changes no byte.

- [ ] **Step 1: Write the failing tests** (they write into a temporary directory):

```python
def test_coast_tile_has_a_jpeg_and_a_one_bit_mask():
    entry = P.write_tile(tmp, 8, 1, 2, rgb, half_land)
    assert entry[:4] == [8, 1, 2, "coast"] and len(entry) == 6
    assert Image.open(tmp / "8/1/2.png").mode == "1"
    jpeg = np.array(Image.open(tmp / "8/1/2.jpg"))
    assert abs(int(jpeg[128, 250, 0]) - 231) <= 3          # sea pixels carry the lowland colour

def test_all_land_tile_has_no_mask():
    entry = P.write_tile(tmp, 8, 1, 3, rgb, np.ones((256, 256), bool))
    assert entry[3] == "land" and len(entry) == 5 and not (tmp / "8/1/3.png").exists()

def test_all_sea_tile_is_listed_without_files():
    entry = P.write_tile(tmp, 8, 1, 4, rgb, np.zeros((256, 256), bool))
    assert entry == [8, 1, 4, "sea"] and not (tmp / "8/1/4.jpg").exists()

def test_writing_twice_gives_the_same_bytes():
    ...  # write_tile twice, compare file hashes
```

- [ ] **Step 2: Run to verify they fail.** Expected: `AttributeError` on `write_tile`.
- [ ] **Step 3: Implement** `write_tile` and `--write`. For each tile, take the one-pixel halo from a neighbour only when the neighbour is in the same level's list.
- [ ] **Step 4: Run the tests.** Expected: pass.
- [ ] **Step 5: Write the sample set.** Run: `<venv>/bin/python tools/prepare-terrain.py --write --only launch_64_bogota__medellin launch_66_lyon__turin tsuruga-osaka hokkaido-extension --box 103.5 30.3 109.2 34.5` (the box is Chengdu–Xi'an across the Qinling, an existing line with no corridor ID of its own here). Expected: it prints the counts written; `assets/terrain/tiles/8/224/100.jpg` exists; `terrain.json` has `"complete":false`. Do not commit the generated files.
- [ ] **Step 6: Commit** the tool and the test as "Write terrain tiles, masks and a manifest".

---

### Task 4: `pkg/terrain` — manifest, level choice, cover and decode

**Files:**
- Rewrite: `pkg/terrain/data.go` (manifest, features), `pkg/terrain/data_test.go`
- Create: `pkg/terrain/tile.go` (addresses, level, cover), `pkg/terrain/tile_test.go`, `pkg/terrain/decode.go`, `pkg/terrain/decode_test.go`
- Modify: `assets/assets.go` (embed pattern becomes `terrain/*.json terrain/tiles terrain/ATTRIBUTION.md`), `assets/terrain/features.json` (remove `lakes`), `tools/check-boundaries.py` (add `./pkg/terrain` to the packages that must not import `ebiten`)
- Modify: `pkg/render/terrain.go` (so the package keeps building: `newRelief` loads the `Set`, `draw` draws only the peak and lake markers, and the tile, contour and lake-outline code is removed until Task 6; `labels` and the legend are untouched), `pkg/render/basemap_test.go` (delete `TestLakeBiwaHasOneShoreInBothLayers` and its `pkg/terrain` import)
- Delete: `assets/terrain/hokkaido.png`, `assets/terrain/kansai.png`

**Interfaces:**
- Consumes: the manifest shape from Task 3.
- Produces:

```go
type Address struct{ Z, X, Y int }
func (a Address) Parent() Address
func (a Address) Children() [4]Address
// Within reports whether a lies inside b (or is b).
func (a Address) Within(b Address) bool
// Corner is the position of the point (u, v) of the tile, each from 0 to 1, west to east and north to south.
func Corner(a Address, u, v float64) (lat, lon float64)

type Kind uint8
const (Land Kind = iota; Coast; Sea)
type Tile struct{ Kind Kind; SHA, MaskSHA string }

type Region struct{ ID string; Contours []Contour }   // Contour and Feature keep today's fields
type Set struct {
    Tiles      map[Address]Tile
    BasemapSHA string
    Complete   bool
    Regions    []Region
    Features   []Feature
}
func Load(files fs.FS) (*Set, error)                  // reads terrain.json and features.json only

// LevelFor is the level to draw when one degree of longitude at the view's centre is pixelsPerDegree wide:
// the smallest level from 2 to 8 whose tiles are at most 512 pixels wide.
func LevelFor(pixelsPerDegree float64) int

// Piece draws Area using the part of Source's image that covers it. Source is Area or an ancestor of it.
type Piece struct{ Area, Source Address }
// Cover lists what to draw at level, and the tiles the view wants loaded, coarsest first.
func (s *Set) Cover(level int, visible, ready func(Address) bool) (pieces []Piece, wanted []Address)

// Decode reads one tile, checks its hashes, and returns it premultiplied: land at alpha 225, the rest clear.
func (s *Set) Decode(files fs.FS, a Address) (*image.RGBA, error)
func Path(a Address) string                           // "terrain/tiles/8/227/93"; ".jpg" and ".png" are appended
```

`Cover` rule, for every level-2 tile in the set that is visible: a sea tile draws nothing; a tile at `level`, or with no child in the set, is drawn whole; otherwise each visible child that is in the set is treated the same way, and each visible child that is not in the set is drawn as that quarter of its parent. The tile that should supply an area goes into `wanted`, with its level-2 ancestor before it; the piece's `Source` is the finest tile at or above it for which `ready` is true, and if none is ready the area gets no piece.

- [ ] **Step 1: Write the failing tests.** In `tile_test.go`, with a small hand-built `Set`:

```go
func TestLevelForKeepsTilesBetweenOneAndTwoTimesTheirSize(t *testing.T) {
    for _, c := range []struct{ ppd float64; want int }{{0.5, 2}, {512.0 / 90, 2}, {512.0/90 + 0.01, 3}, {300, 8}, {5000, 8}} { ... }
}
func TestCornerMatchesWebMercator(t *testing.T) {
    lat, lon := Corner(Address{8, 227, 93}, 0, 0)       // today's Hokkaido origin
    // want lon 139.21875 within 1e-8, lat 44.087585 within 1e-5
}
func TestCoverDrawsEveryAreaOnce(t *testing.T) {
    // Set: one level-2 land tile, one of its level-3 children, and one level-4 grandchild; all ready; level 4.
    // The pieces' areas, rasterised onto the level-4 grid under the level-2 tile, cover each of its 16 cells exactly once.
}
func TestCoverFallsBackToTheFinestReadyAncestor(t *testing.T) {
    // Same set; only the level-2 tile is ready: every piece has Source == the level-2 tile and the same 16 cells are covered once.
    // With nothing ready: no pieces, and wanted still lists every tile, level 2 first.
}
func TestSeaTilesCountAsCoverageAndDrawNothing(t *testing.T) {
    // A sea child: no piece covers its cells, it is not in wanted, and its parent's quarter is not drawn there.
}
func TestCoverSkipsWhatIsNotVisible(t *testing.T) { ... }
```

In `decode_test.go`, with an `fstest.MapFS` holding a JPEG and mask made in the test:

```go
func TestDecodeAppliesTheMaskAndLandOpacity(t *testing.T) {
    // Land pixel: alpha 225 and colour premultiplied (R <= 225). Sea pixel: 0, 0, 0, 0.
}
func TestDecodeRejectsATileThatFailsItsHash(t *testing.T) { ... }      // one flipped byte in the JPEG, and separately in the mask
func TestDecodeRefusesSeaAndUnknownTiles(t *testing.T) { ... }
func BenchmarkDecode(b *testing.B) { ... }                              // one coast tile from assets.Terrain
```

In `data_test.go`: `TestManifestLoadsWithoutDecodingATile` (load from an FS that holds only the two JSON files) and keep the Yotei feature check from today's test.

- [ ] **Step 2: Run to verify they fail.** Run: `go test ./pkg/terrain`. Expected: build failure, `undefined: Address`.
- [ ] **Step 3: Implement** the three files, the embed pattern, the boundary check, the `pkg/render` stub, and remove the two PNGs and `lakes`. Between this task and Task 6 the map shows no relief; the suite stays green.
- [ ] **Step 4: Run to verify they pass.** Run: `go test ./pkg/terrain -count=1`, `python3 tools/check-boundaries.py`, then the full capped suite. Expected: PASS, and the boundary script's usual line.
- [ ] **Step 5: Measure decode time** (the spec's first measurement). Run: `go test ./pkg/terrain -run '^$' -bench Decode -benchtime 200x`. Record ns/op in the commit message. If it is above 4 ms, say so to the user before Task 5: at 8 uploads a frame the worker would fall behind a pan.
- [ ] **Step 6: Commit** as "Load the terrain manifest and decode tiles one at a time".

---

### Task 5: Tile cache and loader

**Files:**
- Create: `pkg/render/tilecache.go`, `pkg/render/tilecache_test.go`

**Interfaces:**
- Consumes: `terrain.Address`, `(*terrain.Set).Decode`.
- Produces:

```go
const (tileCap = 288; uploadsPerFrame = 8)

type texture interface{ Deallocate() }

// tileCache holds uploaded tiles up to a fixed number and frees the ones it drops.
type tileCache struct{ ... }
func newTileCache(limit int, decode func(terrain.Address) (*image.RGBA, error), upload func(*image.RGBA) texture) *tileCache
func (c *tileCache) ready(a terrain.Address) bool
func (c *tileCache) get(a terrain.Address) texture          // marks a as drawn this frame
// want replaces the list of tiles to load, coarsest first; tiles no longer wanted and not yet decoded are dropped.
func (c *tileCache) want(tiles []terrain.Address)
// frame uploads up to uploadsPerFrame decoded tiles, evicts down to the limit, and reports whether anything arrived.
func (c *tileCache) frame() (arrived bool)
func (c *tileCache) len() int
func (c *tileCache) close()                                  // stops the worker and frees every texture
```

One goroutine takes the next wanted address, decodes it, and hands the image to `frame` through a channel of capacity `uploadsPerFrame`. The decoded image is dropped after upload. A tile that fails to decode is remembered as failed, reported once through `log.Printf`, and not retried. Eviction order is least recently drawn; a tile drawn since the last `frame` call is never evicted, even over the limit.

- [ ] **Step 1: Write the failing tests** with a fake texture that counts `Deallocate` calls and a decode function that returns a blank image:

```go
func TestCacheHoldsNoMoreThanItsLimitAndFreesTheRest(t *testing.T) {
    // limit 4; want and load 10 tiles (call frame until len stops changing, drawing none of them):
    // len() <= 4 and exactly 6 fake textures were deallocated.
}
func TestCacheNeverEvictsATileDrawnThisFrame(t *testing.T) {
    // limit 2; load 2 and get() both; want 3 more and call frame: the two drawn tiles are still ready.
}
func TestCacheUploadsAFewTilesAFrame(t *testing.T) {
    // 40 wanted tiles: after one frame() at most uploadsPerFrame are ready.
}
func TestCacheDropsWhatIsNoLongerWanted(t *testing.T) {
    // decode blocks on a channel; want(A...), then want(B...): once released, no A tile beyond the one in flight is decoded.
}
func TestLoaderReportsAFailedTileOnceAndKeepsGoing(t *testing.T) {
    // decode fails for one address: it never becomes ready, decode is called for it once, and the other tiles load.
}
func TestCloseFreesEverything(t *testing.T) { ... }
```

Poll with a deadline where a test waits for the worker; do not sleep for fixed times.

- [ ] **Step 2: Run to verify they fail.** Run: `go test ./pkg/render -run 'TestCache|TestLoader|TestClose'`. Expected: build failure, `undefined: newTileCache`.
- [ ] **Step 3: Implement `tilecache.go`.**
- [ ] **Step 4: Run to verify they pass**, with `-race` once: `go test -race ./pkg/render -run 'TestCache|TestLoader|TestClose' -count=1`. Expected: PASS, no race reported.
- [ ] **Step 5: Commit** as "Cache terrain tiles and load them off the frame".

---

### Task 6: Draw relief from tiles

**Files:**
- Rewrite: `pkg/render/terrain.go` (the `relief` type and `draw`; `labels`, the legend layout and `terrainSwatch` stay)
- Create: `pkg/render/terrain_test.go`
- Modify: `pkg/render/cache.go` (scene key), `pkg/render/map.go` (construction, `Draw`)

**Interfaces:**
- Consumes: Task 4's `Set`, `LevelFor`, `Cover`, `Corner`, `Piece`; Task 5's `tileCache`.
- Produces:

```go
type relief struct {
    set      *terrain.Set
    cache    *tileCache
    contours [][]mapview.Point       // both Japan regions, projected once
    features []terrain.Feature
    arrivals uint64                   // counts frames in which a tile arrived
}
func newRelief() (*relief, error)                                 // loads the manifest; decodes nothing
// level is the tile level to draw for a camera.
func (t *relief) level(c mapview.Camera) int
// visible reports whether any part of a tile is on screen, from nine projected points of it.
func (t *relief) visible(c mapview.Camera, a terrain.Address) bool
// mesh fills vertices and indices for one piece: 9x9 vertices, source coordinates inside Source's 256x256 image.
func pieceMesh(c mapview.Camera, p terrain.Piece, opacity float32, dark bool, vertices []ebiten.Vertex, indices []uint16) ([]ebiten.Vertex, []uint16)
func (t *relief) draw(dst *ebiten.Image, c mapview.Camera, s ui.State)
// tick lets arrived tiles in; called once per Map.Draw while terrain is shown.
func (t *relief) tick() uint64
```

- `sceneKey` gains `Terrain uint64`. `Map.Draw` sets it from `m.terrain.tick()` when `!s.HideTerrain` and leaves it 0 otherwise, before comparing keys, so a tile that arrives redraws the scene and hidden terrain loads nothing.
- `draw`: `level`, then `Cover(level, visible, cache.ready)`, `cache.want(wanted)`, then one `DrawTriangles` per piece with `FilterLinear` from `cache.get(p.Source)`. Contours are stroked at `Zoom >= 4` as today, only for regions whose box is on screen. The lake fill and outline are gone. Peaks and lake markers are unchanged.
- Legend swatches become `(231,244,204)`, `(199,211,145)`, `(183,168,133)`, `(166,141,128)`, `(244,243,240)` and the caption `"2000+ m"` becomes `"4800+ m"`.
- The upload function passed to the cache is `ebiten.NewImageWithOptions(image.Rect(0, 0, 256, 256), &ebiten.NewImageOptions{Unmanaged: true})` followed by `WritePixels(img.Pix)`.

- [ ] **Step 1: Write the failing tests** in `terrain_test.go`:

```go
func TestBuildingTheMapDecodesNoTile(t *testing.T) {
    // New() then m.terrain.cache.len() == 0 and nothing is wanted.
}
func TestReliefLevelFollowsTheZoom(t *testing.T) {
    // The World camera at 1440x960 gives a level from 2 to 4; a camera fitted to Bogota-Medellin at Zoom 8 gives 8.
}
func TestPieceMeshUsesOnlyItsPartOfTheSource(t *testing.T) {
    // Area {8, 227, 93} drawn from Source {7, 113, 46}: every SrcX is in [128, 256] and SrcY in [128, 256];
    // from Source {6, 56, 23}: SrcX in [192, 256], SrcY in [64, 128]; from itself: both in [0, 256].
}
func TestNeighbouringPiecesShareTheirEdgeVertices(t *testing.T) {
    // The east edge of one area and the west edge of the next have identical DstX/DstY, so no crack shows.
}
func TestHiddenTerrainLoadsNothing(t *testing.T) {
    // With HideTerrain, sceneKeyFor leaves Terrain at 0 after an arrival, and the cache's wanted list stays empty.
}
func TestALateTileChangesTheSceneKey(t *testing.T) {
    // After the cache reports an arrival, sceneKeyFor differs from the key before it.
}
func TestTerrainLegendShowsTheNewTop(t *testing.T) {
    // m.terrainLegend(c, "en").labels[1].text == "4800+ m"
}
```

- [ ] **Step 2: Run to verify they fail.** Run: `go test ./pkg/render -run 'Relief|PieceMesh|Neighbouring|HiddenTerrain|LateTile|TerrainLegend|BuildingTheMap'`. Expected: build failure on the new names.
- [ ] **Step 3: Implement.**
- [ ] **Step 4: Run the whole suite.** Run: `gofmt -l .`, `go vet ./...`, `GOOS=js GOARCH=wasm go vet ./...`, `GOMEMLIMIT=768MiB GOMAXPROCS=2 go test -p 1 ./... -count=1`. Expected: no output from gofmt, PASS everywhere. `TestMapLegendsUnchangedWithoutAScale` must pass unchanged.
- [ ] **Step 5: Build both targets.** Run: `sh tools/build-native.sh` and `sh tools/build-web.sh`. Expected: both succeed.
- [ ] **Step 6: Commit** the Go changes (not the generated tiles) as "Draw relief from tiles, finest ready level first".

---

### Task 7: Look at it, and get the look approved

**Files:** none committed, unless the user asks for a change.

- [ ] **Step 1: Run the native app on a copy of a real save** (recipe: memory note "Measure native memory on a save copy"; scratch `HOME`). Capture screenshots, and look at each before showing it, of: the World view; Bogotá–Medellín framed whole and at about 2.5 times closer; Kansai at the zoom of `docs/validation/terrain-kansai-desktop.png`; Lyon–Turin; Chengdu–Xi'an; one view that crosses the edge of level-8 coverage; one coast near a line; the dark palette on one of them.
- [ ] **Step 2: Check each against the spec's anti-goals:** no hard-edged ribbon, no step in brightness where sharp meets coarse, no relief on sea or on a drawn lake (Lake Biwa, Lake Tai), no tile seams, Japan's contours and labels where they were.
- [ ] **Step 3: STOP — show the user the screenshots** and ask them to approve the colour scale, the JPEG quality and the join between sharp and coarse relief. Say plainly whether relief at levels 2 to 5 looks flat: the shading method takes slopes in metres, so wide views may show colour with little shadow. If the user wants stronger shading at wide views, that is a change to the design (exaggeration by level); agree it with them, change `shade` and its test in `tools/terrain_rule.py`, rewrite the sample and repeat this task.
- [ ] **Step 4:** If anything changed, commit it as its own commit that says what the user asked for.

---

### Task 8: Generate everything, and pin coverage

**Files:**
- Create (generated, committed): `assets/terrain/tiles/**`, `assets/terrain/terrain.json`
- Create: `pkg/terrain/coverage_test.go`
- Modify: `.github/workflows/milestone1.yml` (the preparation lane also runs `tools/test-terrain.py`; add `-r tools/requirements-terrain.txt` to that lane's install if it is not already there)

**Interfaces:**
- Consumes: everything above.

- [ ] **Step 1: Write the failing tests** in `coverage_test.go` (they read `assets.PlayableRailMap`, `assets.PublicNetwork`, and `../../data/construction/compiled-prospects.v1.json` from disk):

```go
func TestEveryLineLiesOnLevel8Tiles(t *testing.T) {
    // Every point at 1 km steps along every line is inside a level-8 tile that is in the set (sea tiles count).
    // The failure names the line and the first uncovered point, and says to run tools/prepare-terrain.py.
}
func TestFineTilesHaveTheirParents(t *testing.T) {
    // Every level-8 tile has its level-7 parent and every level-7 tile its level-6 parent.
}
func TestCoarseLevelsCoverAllDrawnLand(t *testing.T) {
    // For levels 2 to 5: every tile that contains a vertex of a land outline in world-land.json north of 60 S is in the set.
}
func TestTilesWereCutToThisBasemap(t *testing.T) {
    // set.BasemapSHA equals the SHA-256 of the embedded world-land.json, and set.Complete is true.
}
func TestEveryListedTileHasItsFiles(t *testing.T) {
    // Land and coast tiles have a .jpg, coast tiles a .png, sea tiles neither; no file under tiles/ is unlisted.
}
```

- [ ] **Step 2: Run to verify they fail.** Run: `go test ./pkg/terrain -run 'EveryLine|FineTiles|CoarseLevels|CutToThisBasemap|EveryListedTile'`. Expected: `TestEveryLineLiesOnLevel8Tiles` and `TestTilesWereCutToThisBasemap` FAIL on the sample set (a line outside the sample is uncovered; `complete` is false).
- [ ] **Step 3: Generate the full set.** Run: `<venv>/bin/python tools/prepare-terrain.py --write`. Expected: it prints tile counts per level and kind; `du -sh assets/terrain/tiles` is under 25 MB. If it is over, stop and report the size to the user.
- [ ] **Step 4: Run the tests and the suite.** Run: the five tests, then the full capped suite. Expected: PASS.
- [ ] **Step 5: Build both targets and compare sizes** with the binaries from `main` (`build/` native binary and `web/main.wasm`). Expected: each grows by no more than about 25 MB. Record both numbers.
- [ ] **Step 6: Commit** the tiles, the manifest, the test and the workflow as "Add relief tiles for every line and all land", with the tile counts and sizes in the message.

---

### Task 9: Browser check and documents

**Files:**
- Modify: `tools/check-terrain.cjs`, `assets/terrain/README.md`, `assets/terrain/ATTRIBUTION.md` (only if Task 2 changed credits), `docs/DESIGN.md` (terrain paragraph), `TESTING.md`, `ARCHITECTURE.md` (if it describes the two regions), `docs/superpowers/specs/2026-11-14-worldwide-terrain-design.md` (add "As built")

- [ ] **Step 1: Update `tools/check-terrain.cjs`.** After the Kansai section and before the phone section, select `launch_64_bogota__medellin` with `prospect(p, …)`, expand, focus the route, wait until two canvas screenshots 500 ms apart are identical (tiles have arrived), save `terrain-bogota-medellin-desktop.png`, hide terrain, and assert the canvas changed; then show terrain again. Add to `result.checks`: "Relief outside Japan: Bogotá–Medellín canvas changes with the terrain toggle". The Hokkaido `shot` calls get the same wait-until-stable in place of the fixed 500 ms.
- [ ] **Step 2: Do not run it locally** unless the user asks; it usually exceeds the 2 GiB footprint limit here. Run `node --check tools/check-terrain.cjs`. Expected: no output.
- [ ] **Step 3: Update the documents.** `assets/terrain/README.md`: replace the sentence that only two regions have coverage with the coverage rule (levels, 60 km, the Japan boxes), the file layout, the manifest fields, how to regenerate, and keep the statement that relief cannot prove a route feasible. `docs/DESIGN.md`: the terrain paragraph. `TESTING.md`: the terrain paragraph names the new Go tests, `tools/test-terrain.py`, and that `check-terrain.cjs` is qualified by hosted CI. Spec: an "As built" section with the real tile counts, sizes and decode time.
- [ ] **Step 4: Run** `python3 tools/check-boundaries.py`, the full capped Go suite, and both builds once more. Expected: PASS.
- [ ] **Step 5: Commit** as "Check relief outside Japan in the browser and document the tile pyramid".

---

### Task 10: Measure memory and frame time

**Files:**
- Modify: `docs/superpowers/specs/2026-11-14-worldwide-terrain-design.md` ("As built" numbers), `assets/terrain/README.md` (one line with the measured memory)

- [ ] **Step 1: Measure before.** Build `main`'s native binary in a scratch copy (`git archive` into the scratchpad). On a copy of a real save, at 3737×2029, pan for one minute between continents (East Asia, Europe, South America and back) and record `vmmap --summary` physical footprint at the start and the end.
- [ ] **Step 2: Measure after**, the same way, with this branch's binary. Also record, with an instrumented build (`go build -overlay`, not committed): the longest frame during the pan, the time from the last camera change until no tile is wanted and none is in flight, and the largest number of tiles held.
- [ ] **Step 3: Compare with the goals.** Expected: tiles held never exceed 288 (75 MB); the footprint at the end of the minute is no higher than after the first pass over the same views; no frame over 33 ms is caused by tile loading; a stopped view is sharp within one second. The baseline to compare against is about 2.5 GB at the World view and 1.2 GB at Japan.
- [ ] **Step 4: If a goal is missed, stop and report the numbers** with the cause before changing the design. Known levers, in the order to try them: fewer uploads per frame; a smaller cache; one shared 2048×2048 page image per 64 tiles (needs a one-pixel gutter per tile) if several hundred draw calls per redraw prove slow.
- [ ] **Step 5: Write the measured numbers** into the two documents and commit as "Record measured memory and frame time for worldwide terrain".

---

## Not covered by this plan

- Browser decode speed and browser memory with the larger binary: the spec sets no target and neither can be measured locally. CI's `check-terrain.cjs` shows only that relief appears.
- Contours, peaks and labels outside Japan (spec non-goals).
- Regenerating tiles in CI. The source cache is a few hundred megabytes; CI checks the manifest against the lines and the basemap instead, so a new line or a changed coast fails `TestEveryLineLiesOnLevel8Tiles` or `TestTilesWereCutToThisBasemap` until a developer regenerates.
