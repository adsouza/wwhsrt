# Worldwide terrain relief

Status: approved 2026-11-14; built 2026-11-15 (see "As built" at the end).
Coastline refinement was built first.
Amended 2026-11-14 and the amendment approved the same day (and on 2026-11-15
for what the coastline build found; see "After the coastline build" at the end): relief is cut to the land the refined
basemap draws (the user's decision), and "a line" gained the two older
Japanese routes (my call). See Behaviour 1 and 4.
Size: a project (a new tile format, a reworked generator, an on-demand loader,
renderer changes, documentation and checks).
Depends on: coastline refinement
(`2026-11-14-coastline-refinement-design.md`). The tile generator reads the
basemap that project produces, so this is built second.

## Intent

The map shows shaded relief so a player can see why a line curves around
mountains. Today that relief exists only in two hand-picked boxes, Hokkaido and
Kansai. Everywhere else the land is one flat colour, so a route such as
Bogotá–Medellín (238 km in a straight line, 421 km as routed through the
Magdalena valley) looks like an unexplained detour.

This change shows relief wherever the game has lines, at the detail Japan has
today, and coarser relief on all other land.

## Goals

- Every line in the game, buildable or existing, has relief at Japan's current
  detail (zoom-8 source tiles, about 0.6 km per pixel at the equator) within
  60 km of the line.
- All land has relief at wide views, so no continent is flat when zoomed out.
- The native app and the browser bundle each grow by no more than about 25 MB.
  The estimate is 21 to 23 MB (see Evidence).
- Memory used by terrain depends on the window size, not on how much of the
  world is covered. At a 3737×2029 window, terrain holds at most about 80 MB of
  GPU memory, and panning across continents for a minute does not raise the
  footprint further.
- Startup decodes no tile that is not on screen in the first frame.
- Play is fully offline on both targets, as now. In the browser this is by
  construction (the tiles are inside the binary) and is not verified.
- The Go suite still passes under its 768 MiB cap, and tests that build the
  map do not get slower.

## Non-goals

- Contours outside Hokkaido and Kansai. Japan keeps the contours it has.
- Named peaks, uplands and lakes outside Japan. Those labels are hand-written.
- Making the drawn routes follow the terrain. Route paths are authored sketches
  with straight legs between a few waypoints (Bogotá–Medellín has five points).
  On detailed relief some legs will visibly cross ridges. The alignment
  descriptions already say such crossings need tunnels.
- Terrain affecting costs, feasibility, schedules, saves or scenario hashes. It
  stays display-only.
- Detail finer than zoom 8. Closer zooms enlarge zoom-8 tiles, as Japan does
  today.
- Sea-floor relief. The sea stays clear.
- Refining the basemap coastline. Outside Japan, China, Taiwan, Australia,
  South Africa and Chile the basemap is Natural Earth 1:110m: it draws Istanbul
  and Tangier as sea, and San Francisco Bay and the Thames estuary as land.
  Relief follows the real coast, so the two disagree there. The user chose to
  fix this by refining coastlines for every country with lines, as a separate
  project that is built first. This design does not change the basemap; it
  reads the refined one (Behaviour 4).
- Running the browser check locally. `TESTING.md` blocks full browser/WASM
  startup under the unresolved 2 GiB cap, and the user chose not to run it.
  Since 2026-11-15 hosted CI runs the browser checks on push, so
  `tools/check-terrain.cjs` must be right for the new assets and is first
  verified there.
- Reducing the app's existing idle memory footprint.

## Anti-goals

- Memory or startup time that grows with coverage.
- Any download while playing.
- Ribbons of relief with a hard edge against flat land.
- A visible change in brightness where sharp tiles meet coarse ones.
- Tiles left on the GPU for Go's collector to free. The collector does not see
  GPU memory; that caused the 11 GB session fixed in commit 2820824.
- Wording or visuals that suggest relief proves a route is feasible.

## Behaviour (decided)

### 1. What is covered

Relief is stored as 256×256 tiles on the Web Mercator grid, at seven levels:

| Level | Detail at the equator | Covers |
|---|---|---|
| 2 to 5 | 39 km to 4.9 km per pixel | every tile that contains land north of 60° S |
| 6 and 7 | 2.4 km and 1.2 km per pixel | the parents of every level-8 tile |
| 8 | 0.6 km per pixel | every tile within 60 km of a line, plus today's Hokkaido and Kansai boxes |

"A line" means every path in `assets/scenario/playable-rail-map.json`, every
alignment path in `data/construction/compiled-prospects.v1.json`, and every
alignment path of `hokkaido-extension` and `tsuruga-osaka` in
`assets/scenario/public-network.json`. Paths are followed along their length,
not only at their vertices. The coastline design uses the same definition.

The Hokkaido box is tiles x 227–228, y 93–95 and the Kansai box is x 224–225,
y 100–101, both at level 8. They stay covered even where no line passes.

### 2. Which level is drawn

The map draws the finest level whose pixels are not smaller than screen pixels,
so tiles appear at one to two times their stored size. Level 8 is the finest;
beyond it tiles are enlarged further, as now.

Where the chosen level has no tile, the nearest coarser level that has one is
drawn for that area. Every piece of land is drawn exactly once. The result is
sharp relief near lines and softer relief beyond them, with no step in
brightness between the two.

### 3. Loading and memory

- A tile is decoded the first time it is needed and uploaded to the GPU.
- Until a tile is ready, the coarser tile under it is shown. When it arrives
  the map updates without the player moving it.
- Decoding is spread over frames so that a pan does not stall.
- Loaded tiles are held in a cache with a fixed cap. Tiles pushed out are
  freed explicitly with `Deallocate`.
- The decoded Go image is not kept after upload. Today's loader keeps it.
- With terrain hidden, no tile is loaded.
- Targets on the native app at 3737×2029 (my call): tile loading makes no
  frame longer than 33 ms during a pan, and a view that has stopped moving is
  fully sharp within one second. The first build step is to measure decode
  time per tile on native.
- No target is set for the browser, because it cannot be checked (see
  Non-goals). If Go's decoder proves slow there, the remedy is slower fill-in
  with the coarser tile shown for longer. A different format or a browser-side
  decoder is outside this design and needs the user's agreement.
- Every frame draws relief the same way. The coastline project gives frames
  without the public network, which only tests produce, the same basemap as
  the rest (amended 2026-11-14).

### 4. Storage

- Each tile's colour is a JPEG at quality 80.
- A tile that is not all land has a one-bit mask. A pixel is marked land when
  its elevation is above 1 m, as today, and it lies on land drawn by the
  refined basemap (`assets/terrain/world-land.json`: land outlines minus drawn
  lakes). Other pixels are drawn fully clear. Relief therefore never appears
  without drawn land beneath it, and never on a drawn lake.
- The manifest records the basemap's hash. A Go test fails if it differs from
  the embedded basemap, so a changed coast forces the tiles to be regenerated.
- In the tile's JPEG, clear pixels carry the flat lowland colour so there is
  no hard edge for compression to smear.
- Land keeps today's fixed 225/255 opacity, applied when the tile is loaded or
  drawn, and the existing zoom-dependent opacity on top of that.
- All tiles ship inside the binary on both targets.

### 5. Colour scale

The current scale reaches its palest colour at 2,600 m, which suits Japan. In
the Andes it makes Bogotá's flat plateau and 5,000 m peaks the same pale tone.

Proposed scale (my proposal, shown to the user as an image; the exact colours
may still be adjusted): unchanged up to 1,800 m, then brown at 2,600 m, grey at
3,600 m and near-white at 4,800 m.

| Metres | Colour (R, G, B) |
|---|---|
| 0 | 231, 244, 204 |
| 150 | 218, 231, 174 |
| 400 | 199, 211, 145 |
| 800 | 189, 190, 135 |
| 1200 | 183, 168, 133 |
| 1800 | 177, 157, 139 |
| 2600 | 166, 141, 128 |
| 3600 | 196, 186, 180 |
| 4800 | 244, 243, 240 |

The legend keeps five swatches, showing the colours at 0, 400, 1,200, 2,600
and 4,800 m, and its upper caption becomes "4800+ m" (my call). That caption is
the same length as today's "2000+ m", so the browser legend positions pinned by
`TestMapLegendsUnchangedWithoutAScale` hold. Hillshading (northwest light,
relief exaggerated 2.5 times) and the dark-palette tint are unchanged.

Colour is baked into the tiles, so a later change rewrites every tile in git.
Before the full set is generated and committed, the builder generates a small
sample (Bogotá–Medellín, Kansai, and one line through high ground in each of
Europe and Asia) and the user approves the look in the native app (my call).
The user is also the judge of "no step in brightness" and "no visible
difference from JPEG".

### 6. Japan

Hokkaido and Kansai move onto the same tiles as everywhere else. Their
contours, peak symbols, lake outline and labels are unchanged. Three things
look slightly different there: land above 1,800 m is browner, the 12-pixel
fade at the old box edges is gone because there are no boxes, and the image is
JPEG-compressed.

### 7. Generator

`tools/prepare-terrain.py` is reworked to:

- derive the tile list from the two line files and the two Japan boxes;
- fetch each level's tiles from Mapzen Terrain Tiles at that level, cached under
  the ignored `build/terrain-source`, each pinned by SHA-256 as now (about 2,000
  source tiles, a few hundred megabytes of cache);
- shade them with the current method and the new colour scale;
- write the tiles, masks and a manifest, and keep the per-source pins (URL,
  hash, imagery sources, last-modified) in a file that is not embedded;
- fail if any source tile cites an elevation dataset that
  `assets/terrain/ATTRIBUTION.md` does not credit.

If that check fails, the builder stops and reports the dataset names and their
terms to the user; adding a credit is a licence decision and is not the
builder's to make (my call). A new credit touches `ATTRIBUTION.md`, the credits
text both build scripts copy into the bundles, the legend caption
"Mapzen / USGS" in `pkg/render/terrain.go`, and the
`includes('Mapzen Terrain Tiles')` assertion in `tools/check-terrain.cjs`.

Running it is a developer step. Ordinary builds and play still need no Python
and no network.

### 8. Checks

- A Go test fails if any point along any line lies outside a level-8 tile in
  the manifest. Adding a line without regenerating terrain therefore fails.
- A Go test fails if a level-8 or level-7 tile lacks its parent, or if any
  level 2 to 5 land tile is missing.
- Each tile's hash is checked when it is loaded, as images are today.
- A test shows the cache cap holds: after loading more tiles than the cap, the
  number held does not exceed it and the rest were deallocated.
- `tools/check-terrain.cjs` is updated for the new assets and for a line
  outside Japan. It is not run locally (decided by the user; see Non-goals);
  hosted CI runs it and produces `docs/validation/terrain-checks.json`. The web
  bundle is built.
- Memory is measured in the native app at 3737×2029 on a copy of a real save,
  before and after, while panning between continents.

### 9. Documents

`assets/terrain/README.md`, `assets/terrain/ATTRIBUTION.md` and the terrain
paragraph in `docs/DESIGN.md` are updated. The README sentence saying only two
regions have coverage is replaced by the new coverage rule, and the statement
that relief cannot prove a route feasible stays.

## Approaches considered

**How it is built**

1. **Built-in pyramid, loaded as you look (chosen by the user).** Described
   above. Memory depends on the window; fully offline; one code path for both
   targets. Costs about 21 to 23 MB of download and a new loader and cache.
2. **Bigger regional images, all loaded at startup.** Extends today's code with
   more boxes. Smallest change, but about 266 MB on the GPU plus the same again
   in Go memory, every tile decoded before the first frame, and 13 tests build
   the map under a 768 MiB cap. Rejected.
3. **Separate tile files fetched when needed.** Keeps the binary at 38 MB and
   starts fastest in the browser, but terrain is missing offline for places not
   yet visited, and native and browser need different loading paths. Rejected
   because offline play is promised.

**How sharp**

The user compared quarter, half and Japan's detail on Bogotá–Medellín. Framed
on the whole route, half and full look the same; about 2.5 times closer, full
is clearly sharper. The user chose Japan's detail.

**How it is stored**

PNG, as today, averages 126 KB per tile on all-land tiles; JPEG at quality 80
averages 12 KB with no visible difference. JPEG was chosen (my call, accepted
with the size table). JPEG has no transparency, hence the separate sea mask.

## Constraints

- Assets are embedded with `go:embed` in `assets/assets.go`; its terrain
  pattern lists `*.json`, `*.png` and the credits file and must take in the new
  files.
- `pkg/terrain` must stay free of graphics imports, and the headless
  composition must stay graphics-free; `tools/check-boundaries.py` enforces it.
- The map redraws its whole scene whenever the camera changes
  (`pkg/render/cache.go`), so during a pan every visible tile is drawn every
  frame. A late-arriving tile must also cause a redraw, which the scene key
  does not do today.
- Source tiles are Web Mercator; the map is Equal Earth. Tiles are drawn as
  meshes whose vertices are reprojected, as now.
- JPEG decoding uses Go's standard library. No new Go dependency.
- The generator's Python dependencies are pinned in
  `tools/requirements-terrain.txt` and need an isolated environment (this
  machine's default Python has a different numpy).
- The repository gains about 21 to 23 MB of binary files.
- In the browser, embedded files sit in WebAssembly memory, so the larger
  binary adds about 22 MB that is always resident. Its effect is not measured.
- Terrain stays outside scenario hashes and save compatibility.

## Left to the builder

- File layout (one file per tile, or a single pack with an index) and the
  manifest's shape and version.
- The mask's file format.
- The exact level-switch threshold, within "tiles appear at one to two times
  their stored size".
- The cache cap and eviction order, within the memory goal.
- How many tiles are decoded per frame, and whether decoding runs on another
  goroutine on native.
- How "drawn exactly once" is achieved: drawing part of a parent tile where a
  child is missing, or composing the layer in an offscreen image. If an
  offscreen image is used it must be one image reused across frames.
- Mesh density per level.
- Whether levels 6 and 7 are fetched from the source or reduced from level 8.
- How the band edge is treated beyond the rule that brightness must not step.

## Evidence

Measured on 2026-11-14 with a throwaway spike (not in the repository):

- 18 source tiles around Bogotá–Medellín at levels 6, 7 and 8, shaded with the
  current method. Mean size per tile: PNG 91, 114 and 126 KB; JPEG quality 80
  9, 11 and 12 KB. Mean absolute error of the level-8 JPEG against the PNG:
  3.3 of 255.
- The four Kansai tiles, all coastal: JPEG 10.4 KB per tile plus a 1.0 KB
  one-bit mask, against 101 KB as PNG. Composited over the sea colour the mean
  error is 2.4 of 255, with no halo visible at the coast.
- Land tiles north of 60° S, counted against `assets/terrain/world-land.json`:
  102 at level 4, 338 at level 5.
- Tiles within 60 km of all 372 paths (300 existing, 72 buildable). My count,
  sampling a 60 km offset around points along each path, was about 1,015 at
  level 8. The design reviewer's count by buffer intersection, including the
  Japan boxes and the parent rule, was 952 at level 8, 360 at level 7 and 158
  at level 6. No path crosses the antimeridian or nears the Mercator limit.
- Against `assets/terrain/world-land.json`, Istanbul (both shores) and Tangier
  test as sea, and mid San Francisco Bay and the Thames estuary test as land.
- Source headers for four sampled tiles cite only SRTM, GMTED2010 and ETOPO1,
  which `ATTRIBUTION.md` already credits. This is a sample, not the full set;
  the generator check in Behaviour 7 covers the rest.
- Size estimate: about 2,000 tiles at 9 to 12 KB, plus masks, is 21 to 23 MB.
  The native binary is 37.5 MB and `web/main.wasm` is 38.8 MB today.

Not measured: decode time per tile on native or in the browser, the real tile
counts after the generator runs, and memory with the pyramid in place.

## After the coastline build (2026-11-15)

Building coastlines first taught three things this design must respect. None
changes what the user approved; all change how it is built.

- **Memory baseline.** Other work on `main` cut the native map's memory, and
  the coastline build measured 2.5 GB while panning the World view at
  3737×2029 and 1.2 GB at the Japan region. Those, not the 3.6 GB idle figure
  quoted in the conversation that produced this design, are the numbers the
  terrain goals are measured against.
- **Ebitengine's costs are in the shape of what is drawn.** A third antialiased
  pass adds a map-sized stencil box, and one large sub-path fans map-spanning
  triangles (5 GB of GPU memory, found by measurement). Relief tiles are drawn
  with `DrawTriangles`, not `vector`, but anything this design strokes or fills
  must be measured the same way, at a large window while panning.
- **Lake Biwa.** The terrain layer still draws its own outline of the lake,
  because today's relief tints the lake's surface. Once tiles are cut to the
  basemap (Behaviour 4) that outline and `relief.lakes` are removed, together
  with `TestLakeBiwaHasOneShoreInBothLayers`.

## As built (2026-11-15)

Built to the plan in `docs/superpowers/plans/2026-11-15-worldwide-terrain.md`.
What differs from the text above, and what was measured:

- **Coverage is graded (the user's decision at the look approval).** Level 8
  alone ended 60 km from a line against level 5, eight times coarser, along a
  hard straight edge. Level 7 now reaches 180 km from a line and level 6 reaches
  400 km, in addition to holding the parents of finer tiles. The distances are
  the builder's.
- **Counts and size.** 2,637 tiles: 12, 44, 135, 415, 365, 626 and 1,040 at
  levels 2 to 8, of which 82 are all sea and have no files. 14.4 MB of tile
  files against the 21 to 23 MB estimated; coast tiles and coarse levels
  compress better than the all-land sample did. Levels 4 and 5 hold 135 and
  415 tiles, more than the 102 and 338 counted here, because the refined
  basemap draws many more islands than the outline those were counted against.
- **Sources.** 2,637 source tiles, 198 MB in the ignored cache. Every one cites
  only SRTM, GMTED2010 and ETOPO1, so no credit was added.
- **A tile's own area is its middle 254 pixels.** The outermost pixel on each
  side lies in the neighbouring tile. Blending between pixel centres cannot
  reach a tile's edge without reading outside the image, which showed as a
  faint line between tiles; with shared border pixels each tile is drawn from
  1 to 255 and the join is exact. A 258-pixel tile would not do: Ebitengine
  rounds an unmanaged image's texture up to a power of two. The source is
  resampled onto that grid with a Catmull-Rom curve before shading.
- **Builder's choices from "Left to the builder".** One file per tile with a
  one-bit PNG mask; one level per view, the coarsest whose tiles are at most
  twice their stored width at the view's centre; part of a parent drawn where a
  child is absent or not ready, with no offscreen image; an 8 by 8 mesh per
  piece, cut finer where a coarse piece meets fine ones; a cache of 288 tiles
  (75 MB), least recently drawn freed first; one worker goroutine on both
  targets and at most 8 uploads a frame; levels 6 and 7 fetched from the
  source; ground distance per pixel taken row by row for the shading.
- **Land is the drawn basemap, whatever its height.** Behaviour 4 also asked
  for elevation above 1 m, as before. With tiles cut to the basemap that test
  only removed real land: 3.3% of it at level 5, including Dutch polders and the
  Jiangsu plain, which showed as darker blotches in the dark palette. It is
  gone (the builder's call after the branch review, confirmed by screenshot).
- **Ice and lake beds.** The level-5 source is a bedrock grid. Greenland came
  out as ice-free lowland with its interior masked away. Where a tile lies, on
  a 33-pixel average, more than 150 m below its parent, the parent's surface is
  blended in. Only Greenland, a little of Novaya Zemlya and the deepest lakes
  are touched.
- **The cache follows the view.** It keeps twice what the scene on screen uses,
  between 64 and 288 tiles, and never frees a tile that scene uses, so a
  window above about 4K can hold more than 288. The first version protected
  only tiles drawn in the current frame, which lapsed as soon as the cached
  scene was shown.
- **Wide views are not flat.** The shading method was kept as it is and levels
  2 to 5 show clear relief.
- **`features.json` is unchanged.** Its Lake Biwa outline is no longer drawn by
  the terrain layer, but the route-geography tools read it and pin the file.
- **Decode time on native:** 0.75 ms for a coast tile.
- **Size:** the native binary went from 39.6 MB to 53.9 MB and `web/main.wasm`
  from 41.0 MB to 55.1 MB: 14.3 and 14.1 MB more, against a goal of 25.
- **Memory, measured on native at the saved window size (3737x2029) on a copy
  of a real save.** A scripted camera glided between eight views on three
  continents, holding each for two seconds, on `main` and on this build.
  Physical footprint stayed between 1.6 and 2.2 GB here against 1.8 to 2.3 GB
  on `main`, and did not grow over five laps (276 s). GPU memory
  (`IOAccelerator (graphics)`) was about 870 MB on `main`; here it was 910 MB
  in the first lap and 965 MB from the second lap on, level for three more
  laps. That is about 95 MB for terrain against the "about 80 MB" goal: 75 MB
  is the 288 tiles, and the rest was not traced. The cache never held more
  than 288 tiles. (Measured before the cache was changed to follow the view;
  at this window size the two behave the same.)
- **Frame time.** Frames more than 33 ms apart: 1 to 4 per four-second glide
  on both builds (mean 2.1 on `main`, 1.9 here), with a worst gap of about
  600 ms on both. That gap is a map redraw that takes about 600 ms once or
  twice a glide on `main` too, so it is not tile loading; it was not
  investigated here. No difference attributable to tiles was measured.
- **Time to sharp.** After an instant jump to a view with nothing cached, no
  tile was wanted or in hand after 25 to 250 ms (worst: Europe, 67 new tiles),
  against a goal of one second. After a glide the view was already sharp.
- **Browser.** `tools/check-terrain.cjs` gained a check that relief changes
  the middle of the Bogotá–Medellín map. Run locally under the 2 GiB footprint
  watch, that section passed in the browser before the watch stopped the run at
  a later step; hosted CI is its test. Browser decode speed and memory were
  not measured.
