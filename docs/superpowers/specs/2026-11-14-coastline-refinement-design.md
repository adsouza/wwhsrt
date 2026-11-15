# Coastline refinement (worldwide basemap)

Status: approved 2026-11-14; implemented 2026-11-15 on branch
worktree-coastline-refinement. See "As built" at the end for what differs from
the text below. The browser coastline check has not run; hosted CI runs it.
Amended 2026-11-14 at the user's request: the regional Japan files and the
test-only frames that draw them are folded into the new basemap (Behaviour 5).
Size: a small project (one tool reworked, one asset replaced, a small renderer
change, checks and documents).
Order: built before worldwide terrain
(`2026-11-14-worldwide-terrain-design.md`), so that the terrain's look is judged
against the final coast.

## Intent

The worldwide basemap is Natural Earth 1:110m outside Japan, China, Taiwan,
Australia, South Africa and Chile. At that scale Istanbul and Tangier are drawn
as sea, San Francisco Bay and the Thames estuary are drawn as land, and the
line into Istanbul runs across open water. Worldwide terrain will follow the
real coast at 0.6 km per pixel, which makes these errors obvious.

This change replaces the basemap with one detailed land outline, kept detailed
only where the game has lines.

## Goals

- Within 150 km of any line, the drawn coast matches Natural Earth 1:10m to
  about 300 m (simplification 0.003°, the setting Japan uses today).
- Farther away the coast is coarse (simplification 0.05°, about 5 km), and the
  two meet without a gap or a step.
- One rule for the whole world. The six countries refined earlier follow it
  too.
- The basemap stays small: about 73,000 points in about 2,000 outlines
  (today 45,628 points in 729 outlines).
- Panning at 3737×2029 in the native app is not visibly slower. Land redraw
  time is measured before and after.
- Every buildable route is still drawn on land or on its approved crossing.
- No scenario, network or campaign hash changes, and saves are unaffected.

## Non-goals

- National borders. Today they appear as a side effect of filling and
  outlining each country. The user chose to drop them (2026-11-14). If wanted
  later they return as their own line layer, drawn once per border.
- Rivers.
- The land mask used for pricing and route review
  (`data/construction/route-land.v1.json.gz`). It is read, not changed.
- Running the browser check. `tools/check-coastlines.cjs` is updated but not
  run, following the user's decision for terrain under the 2 GiB checker cap
  (my call).
- How relief is drawn. One thing does change in the terrain design as a
  result of this one: relief is cut to the land this basemap draws (decided by
  the user, 2026-11-14; see Behaviour 4).

## Anti-goals

- A buildable route drawn offshore.
- A changed hash for anything outside the display basemap.
- Gaps, overlaps or doubled coastlines where detail changes.
- Lakes hidden under terrain relief, or relief with no land drawn beneath it.
- A basemap whose size or draw cost grows with coast nobody will look at.

## Behaviour (decided)

### 1. Source

The outline is the `land` geometry in `data/construction/route-land.v1.json.gz`:
Natural Earth 1:10m countries merged into physical land, with Natural Earth
1:10m lakes cut out as holes. It has 4,297 land pieces and 1,334 lakes. Both
sources are public domain, already pinned by hash in that file, and already
credited in `assets/terrain/ATTRIBUTION.md`.

### 2. Where it is detailed

"A line" means, followed along its length:

- every path in `assets/scenario/playable-rail-map.json` (300);
- every alignment path in `data/construction/compiled-prospects.v1.json` (72
  buildable routes);
- every alignment path of `hokkaido-extension` and `tsuruga-osaka` in
  `assets/scenario/public-network.json`. These two older buildable routes, with
  two alignments each, appear in no other file (my call).

The terrain design uses the same definition. The Hokkaido and Kansai relief
boxes named there also count as near a line throughout, so the coast under
Japan's detailed relief stays detailed (my call).

- A stretch of outline within 150 km of a line is simplified at 0.003°.
- A stretch farther away is simplified at 0.05°.
- Detail can change along one outline. Where it changes, both stretches share
  the same point, so the outline stays closed and continuous.

### 3. What is left out

| Piece | Kept when |
|---|---|
| Island or land piece near a line | always |
| Island far from every line | it is at least about 20 km across |
| Lake near a line | it is at least about 5 km across |
| Lake far from every line | it is at least about 100 km across |

A lake that is left out is drawn as land.

With these thresholds the estimate is 1,762 land outlines and 275 lakes,
73,496 points in all (see Evidence). The thresholds may be tuned to hold the
goal of about 73,000 points and 2,000 outlines.

### 4. How it is drawn

- Land is filled in the land colour with a one-pixel coast line, as today.
- Lakes are filled in the water colour with the same coast line. An island
  inside a lake stays land; the source has 245 of them.
- Terrain relief is cut to this basemap when its tiles are generated: relief
  appears only where land is drawn, and not on a drawn lake. So nothing shows
  relief without a coast, and lakes need no special place in the drawing
  order. The terrain generator reads the basemap file this project produces.
- Lakes show with terrain on or off.
- There are no lines between countries.
- Lake Biwa, which the terrain layer already outlines from the same source, is
  not drawn twice with visibly different shapes.

### 5. What it replaces

`assets/terrain/world-land.json` is replaced entirely, with its metadata file.
The earlier per-country replacements for Japan, China, Taiwan, Australia, South
Africa and Chile inside it are superseded by the single rule.

There is one basemap. Every frame draws it, with or without the public
network. Frames without the public network are produced only by tests: the
native app, the browser build and the headless tool all open the game through
`application.Open`, which always turns the network on. So:

- the branch in `pkg/render/map.go` that draws the regional outlines is
  removed;
- `assets/terrain/land.json`, `assets/terrain/land.metadata.json` and
  `tools/prepare-shorelines.py` are removed, and the regional file leaves the
  binary (345 KB);
- `assets/scenario/land.json`, the coarse regional base, is removed if nothing
  still needs it once its two tool readers are changed;
- `tools/check-route-geography.py`, which checks the six Japanese alignments
  against the regional file as "the drawn coast", checks the new basemap
  instead, and its recorded results are regenerated;
- `pkg/render/coastline_test.go` is rewritten for the new basemap.

### 6. Buildable routes stay on land

Today `tools/launch_shorelines.py` tightens the simplification for a country
until every construction path there lies on drawn land or on its reviewed
crossing, in both geographic and Equal Earth coordinates. The same guarantee
holds for all 74 buildable routes and every one of their alignments, including
the two older Japanese routes, which today are checked only against the
regional file (my call). Where 0.003° would put a path offshore, the
outline is tightened locally (0.001°, 0.0003°, 0.0001°, then unsimplified)
until it does not. `tools/check-launch-prospects.py` keeps checking the drawn
geometry, now as land minus lakes, and the two older routes get the same check
against the new basemap. A route must not be drawn across a lake that this
basemap now shows, Lake Biwa included.

Existing lines are not held to this rule, because real bridges and tunnels
cross water. The builder reports how many sampled points of existing lines lie
off drawn land before and after; the number should fall.

### 7. Tooling

- The basemap is composed by a tool that can run without re-importing the
  whole public network. `tools/import-public-network.py` handles about 3.4
  million points and has peaked at 1,768 MiB.
- The metadata file records the source hashes, both tolerances, the 150 km
  distance, the size thresholds, any local tightening, the counts and the
  output hash.
- A `--check` mode confirms the committed file is what the tool produces.

### 8. Checks

- A Go test confirms the embedded basemap matches the hash in its metadata and
  stays within the point and outline budget.
- A Go test checks known points: Istanbul (both shores), Tangier and Folkestone
  are on land; mid San Francisco Bay, the Thames estuary, the Gulf of Izmit and
  Osaka Bay are water; a point in Lake Michigan is a lake.
- A Go test checks that every city in the public catalogue is on drawn land,
  with any exceptions listed by name and reason.
- A Go test confirms a frame without the public network draws the same land
  as one with it.
- No file in the repository still refers to the removed regional files.
- The hashes of `public-network.json`, the scenario files and the campaign
  checkpoints are shown unchanged.
- Land redraw time is measured in the native app at 3737×2029 on a copy of a
  real save, before and after, during a pan. If it more than doubles, outlines
  wholly off screen are skipped before projection, and it is measured again.
- `tools/check-coastlines.cjs` is updated for the new file and not run.

### 9. Documents

`assets/terrain/README.md`, `assets/terrain/ATTRIBUTION.md`, `ARCHITECTURE.md`,
`data/imports/README.md` and the route-geography passages in `TESTING.md` and
`docs/validation` describe the basemap or the regional files and are updated.

## Approaches considered

1. **One land outline, borders dropped (chosen by the user).** One rule
   everywhere, no seams between countries, about 73,000 points.
2. **Refine each country separately, borders kept.** About 93,000 points.
   Neighbours are simplified separately, so shared borders do not meet: up to
   about 300 m apart near lines and up to about 5 km where coarse, unless
   borders get special handling. Rejected by the user's choice.
3. **Detailed everywhere.** About 277,000 points for the single outline, or
   218,000 per country. Canada and Russia alone add about 73,000, most of it
   Arctic islands. Rejected; the user asked for far coast to stay coarse.

## Constraints

- The basemap is embedded and read by `pkg/render/public_network.go` as a flat
  list of outlines. Lakes need either a second list or a second file, and
  every reader changes with it: the renderer, `tools/check-launch-prospects.py`
  and `tools/check-coastlines.cjs`.
- Land is drawn into an image cached per camera (`publicLayer.drawLand`), then
  relief is drawn over the scene. Because relief is cut to the basemap, lakes
  can be part of that cached image.
- `tools/import-public-network.py --check` regenerates and compares
  `world-land.json` today; that moves to the new tool.
  `docs/validation/launch-prospects-checks.json` and
  `docs/validation/coastline-checks.json` record earlier browser checks and
  hold the old basemap's hash. They are left as records and described as such.
- Every outline is projected and clipped on each redraw, on screen or not, and
  each outline is its own fill and stroke. Outline count matters as well as
  point count.
- The basemap is display-only and outside scenario hashes.
- The Python tools need shapely 2.1.2 (`tools/requirements-geography.txt`) and
  scipy. Both are installed for this machine's `python3` (3.12).
- Memory-heavy tools are run one at a time under the limits in `TESTING.md`.

## Left to the builder

- The file layout for land and lakes, and the metadata's shape.
- Whether the basemap tool is new or a mode of an existing one, and what
  happens to `tools/launch_shorelines.py`.
- How detail is varied along one outline while keeping it valid (no
  self-crossings).
- The exact size thresholds, within the budget.
- How lakes and the islands inside them are drawn.
- How Lake Biwa's two sources are reconciled.
- Whether off-screen outlines are skipped regardless of the timing result.

## Evidence

Measured on 2026-11-14 with throwaway scripts (not in the repository), using a
plain Douglas–Peucker simplification written for the purpose. Against the
recorded results of the real tool it gave 11,424 points for Chile (recorded
11,389), 8,771 for Australia (8,748), 4,703 for Japan (4,680) and 1,173 for
South Africa (1,173).

- Against today's `world-land.json`, Istanbul (both shores) and Tangier test as
  sea; mid San Francisco Bay and the Thames estuary test as land.
- Single outline, all at 0.003°: 276,888 points. With the near and far rule:
  71,416 points for land alone.
- With lakes and the thresholds in Behaviour 3: 381 land outlines near lines
  (38,216 points), 1,381 far (22,781), 132 lakes near lines (7,264) and 143 far
  (5,235); 2,535 far islands and 1,059 lakes left out.
- Per-country refinement of the 43 unrefined countries that contain lines:
  172,487 added points all fine, 47,777 with far islands and far stretches
  coarse.

From the design reviewer's read-only probes of the same files:

- `hokkaido-extension` reaches 141.8 km from the nearest line under the first
  two sources alone, and 36 of 130 land sample points in the Hokkaido relief
  box are more than 150 km from any such line. Hence the third source and the
  boxes in Behaviour 2.
- The four older alignments lie on source land at full resolution (none of
  about 3,800 samples off).
- 19 catalogue cities are more than 150 km from any line, among them Chicago,
  Miami, Karachi, Oslo and Stockholm. With a coarse stand-in coast all stay on
  land except Miami.
- At full source resolution the station markers for Hong Kong, Dongguan,
  Zhuhai and Lisbon, and Miami's marker, test as sea. The exception list in
  Behaviour 8 will have about five to nine names.

## As built (2026-11-15)

Between approval and the build, 17 commits on `main` rewrote land drawing
(`pkg/render/land.go`: one fill path and one coast path, to keep Ebitengine's
stencil small) and made hosted CI run the browser and data checks on push. The
build follows that code. Where it differs from the text above:

- **Far islands: 25 km, not 20.** At 20 km the basemap had 2,343 outlines against
  the budget of 2,200. At 25 km it has 2,002 outlines and 74,361 points: 1,505
  land, 292 lakes, 205 islands inside lakes; 2,552 islands and 1,042 lakes left
  out.
- **All of Hokkaido is detailed.** Its east and north are more than 150 km from
  any line, and at the Japan region's zoom a 5 km coast drew the Shiretoko and
  Nemuro peninsulas as spikes, where the whole island had been detailed before.
  The same angular look remains on other coasts far from lines when they are
  viewed closely; that is the rule as approved.
- **No route needed tightening.** All 76 alignments of the 74 buildable routes
  lie on the drawn land at 0.003 degrees.
- **Lakes are holes, by the even-odd rule.** The land fill is a fan of triangles
  per piece, each flipping the pixels it covers, so a lake inside land is clear
  and an island inside the lake is land again. It is not Ebitengine's own fill:
  that counts overlapping triangles in four bits, a continent as one outline
  put 20 to 40 over pixels deep inland, and the map showed water-coloured
  streaks from Spain to the Baltic (found in review, confirmed on screen).
  Pieces are cut to 256 px so that no triangle crosses the map. The fill is not
  antialiased; the antialiased coast over its edge hides that. The even-odd
  rule needs land outlines that never overlap, which the tool now guarantees,
  and an island inside a lake is cut to the lake as drawn.
- **Detail is chosen by zoom, to within one pixel.** At the earlier quarter-pixel
  tolerance the World view drew 45,849 land vertices against a guard of 22,000
  (`TestWorldViewGeometryStaysNearScreenResolution`), because the finer coast has
  more to show. Each point now carries a rank computed once at load, and the map
  draws the points whose rank exceeds one pixel at the current zoom: 21,749
  vertices at the World view. The choice does not depend on where the camera is,
  so the coast does not shimmer when dragged. An outline with no shape at the
  current zoom is drawn as a sliver when it is at least three pixels long and
  not at all when shorter.
- **The coast is stroked in runs of 32 segments.** The planned before-and-after
  measurement found 7.3 GB of memory at the World view against 2.6 GB before,
  5.1 GB of it on the GPU. Ebitengine fills a sub-path as a fan of triangles
  from its first point, and one land outline makes a continent a single
  sub-path. Cutting the stroke into short open runs inside one path fixed it.
- **Lake Biwa is drawn by both layers for now.** Today's relief tiles tint the
  lake's surface as land, so the terrain layer keeps its own outline until
  relief is cut to the basemap. A test holds the two shores within 0.004
  degrees.
- **Existing lines off land rose, and that is correct.** Sampled every 10 km,
  411 of 48,417 points are off the drawn land; the full-detail source says 420
  are over water. The old basemap had 252 off but disagreed with the source at
  214 points; the new one disagrees at 21. Behaviour 6's "the number should
  fall" was wrong: the old coast painted land over 191 points of real water.
- **City exceptions:** Hong Kong, Dongguan, Miami, Lisbon and Zhuhai, whose
  markers sit 10 to 473 m off Natural Earth's land.
- **Browser check.** The decision not to run it locally stands, but hosted CI
  now runs `tools/check-coastlines.cjs` on push. It was updated to sample the
  new file and has not run yet. Its eight water points are pinned as sea by a Go
  test.

Measured at 3737x2029 on a copy of a real save, panning every frame for 30
seconds (land redraw mean, 95th percentile, footprint at the end):

| View | Before | After |
|---|---|---|
| World | 9.9 ms, 10.6 ms, 2.6 GB | 3.1 ms, 3.5 ms, 2.5 GB |
| Japan | 3.0 ms, 3.1 ms, 1.2 GB | 0.5 ms, 0.6 ms, 1.2 GB |

The After figures are the final build, with the flip fill; with Ebitengine's fill
and the coast in runs they were 7.4 ms and 1.5 ms at the same footprints. The
tool takes about 7 seconds and 613 MB. `world-land.json` is 1.69 MB, up from
1.05 MB; the regional file it retires was 345 KB.

A fresh review of the whole branch found the streaks, an outline that rounding
had collapsed to a line (it would have failed a Go test on amd64 only), long
thin islands that vanished, and eight islands in lakes drawn as water. All four
are fixed with tests.
