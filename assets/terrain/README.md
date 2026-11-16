# Terrain context

Native and browser maps share offline shaded relief: elevation tint and northwest
hillshade wherever the game has a line, and coarser relief on all other land.
Hokkaido and Kansai also keep their 500 m contours, named peaks and uplands, and
lowland labels. Toya and Shikotsu use **point symbols**, not invented lake
boundaries. The land under the relief is one worldwide outline, described under
Basemap below.

This is presentation data, outside scenario hashes, costs, gradients, schedules
and save compatibility. Railway sketches and engineering quantities remain authored
game estimates, not surveyed alignments: route paths are a few straight legs between
waypoints, and on detailed relief some legs visibly cross ridges. Relief cannot prove
an alternative feasible or optimal.

## Coverage

Relief is a pyramid of 256x256 tiles on the Web Mercator grid.

| Level | Detail at the equator | Covers |
|---|---|---|
| 2 to 5 | 39 km to 4.9 km per pixel | every tile that holds drawn land north of 60 S |
| 6 | 2.4 km per pixel | within 400 km of a line |
| 7 | 1.2 km per pixel | within 180 km of a line |
| 8 | 0.6 km per pixel | within 60 km of a line, and the Hokkaido and Kansai boxes |

"A line" is defined under Basemap. Levels 6 and 7 also hold the parents of every
finer tile. Detail therefore halves in steps away from a line; level 8 alone would
end against level 5, eight times coarser, along a hard edge. The set is 2,637 tiles
and 14.4 MB: 12, 44, 135, 415, 365, 626 and 1,040 at levels 2 to 8.

The map draws one level per view, the coarsest whose tiles are at most twice their
stored width on screen, and where that level has no tile it draws that part of the
nearest coarser one. Level 8 is the finest and is enlarged at closer zooms.

## Files

`tiles/{level}/{column}/{row}.jpg` is a tile's colour at JPEG quality 80. A tile
that is part sea has a one-bit mask beside it, `{row}.png`. Land is what the
basemap draws, whatever its height, so relief never shows on sea or on a drawn lake
and polders and basins below sea level are not left bare; sea pixels carry the
lowland colour in the JPEG so compression has no edge to smear. A tile with no land
has no files and is listed only, so that a line across a strait counts as covered.

Only the middle 254 pixels of a tile are its own area. The outermost pixel on each
side lies in the neighbouring tile, so two tiles side by side hold the same ground
along their shared edge and are drawn from 1 to 255 without a seam. The size stays
256 because the graphics library rounds textures up to a power of two.

`terrain.json` (schema 2) lists every tile with its kind and the SHA-256 of its
files, the hash of the basemap the tiles were cut to, and Japan's contours.
`data/terrain/sources.v1.json`, which is not embedded, pins each source tile's URL,
hash, imagery-source header and last-modified date. `features.json` holds the
approximate reference labels and the Natural Earth outline of Lake Biwa that the
route-geography tools read; no script generates it.

## Loading

Starting the game reads `terrain.json` and decodes no tile. A tile is decoded on a
worker the first time a view needs it, checked against its hashes, and uploaded a
few per frame; until it arrives the coarser tile under it is shown, and its arrival
redraws the map. "The coarser tile" is the level-2 tile, which is asked for first,
or anything finer that is still held. The cache keeps twice what the scene on screen
uses, no fewer than 64 tiles and no more than 288 (75 MB), so memory follows the
window; it frees the tiles whose last scene is oldest, explicitly, and never one the
scene on screen uses, so a window larger than about 4K can hold more than 288.
Nothing is loaded while terrain is hidden (`pkg/terrain`,
`pkg/render/tilecache.go`, `pkg/render/terrain.go`).

Measured on the native app at 3737x2029 on a copy of a real save, gliding between
eight views on three continents: GPU memory was about 95 MB above the build
without tiles and stopped rising after the second lap, the physical footprint
stayed in the same 1.6 to 2.2 GB range as before, tile loading added no frames
over 33 ms, and after an instant jump to an unseen view the relief was fully
sharp within 250 ms. The browser was not measured.

## Reproducing

Install `tools/requirements-terrain.txt` and `tools/requirements-geography.txt`
into an isolated Python environment. Ordinary builds and play need no Python or
network.

    python tools/prepare-terrain.py --plan     tiles per level, and how many sources are not cached
    python tools/prepare-terrain.py --fetch    download and pin the missing sources (2,637 files, 198 MB)
    python tools/prepare-terrain.py --survey   the elevation datasets the sources cite
    python tools/prepare-terrain.py --write    write tiles/ and terrain.json

Sources are Terrarium tiles from Mapzen Terrain Tiles on AWS, cached under the
ignored `build/terrain-source`, with metres decoded as `(R*256 + G + B/256)-32768`.
A source that no longer matches its pin stops the run; review the change before
repinning. `--survey` fails if a source cites a dataset `ATTRIBUTION.md` does not
credit or carries no dataset header, and `--write` refuses to run while it would;
adding a credit is a licence decision. Today every source cites only SRTM,
GMTED2010 and ETOPO1.

The level-5 source is a bedrock grid: under Greenland's ice it lies up to 2.8 km
below the surface that level 4 shows, and it holds lake beds, not lake surfaces.
Where a tile's heights, averaged over 33 pixels, lie more than 150 m below its
parent's, the parent's surface is blended in, fully from 400 m. The average tells
ice from mountains, whose valleys differ from a coarser level as much but in both
directions. `--write` names the tiles this touches: Greenland, a little of Novaya
Zemlya, and the deepest lakes.

The rules are in `tools/terrain_rule.py` and tested by `tools/test-terrain.py`,
which CI runs. Hillshade exaggerates vertical relief 2.5 times. The colour scale
is unchanged from the Japan-only relief up to 1,800 m and then runs on through
brown at 2,600 m and grey at 3,600 m to near-white at 4,800 m, so a high plateau
and a higher peak are different tones. Colour is baked into the tiles, so changing
it rewrites every tile. Contours are simplified display lines at 500, 1,000 and
1,500 m. These coarse display samples do not replace the future engineering profile
pipeline.

Adding or moving a line, or changing the basemap, needs the tiles regenerated:
`go test ./pkg/terrain` fails until every kilometre of every line lies on a level-8
tile and the manifest names the embedded basemap's hash.

See [credits](ATTRIBUTION.md) for source terms. Credits ship in the native bundle
and browser offline cache. The Terrain button persists the visibility preference.
City labels take priority over terrain text. Alignment descriptions disclose that
the detours still need tunnels.

The shared camera allows up to 32x zoom so short corridors can fill a phone map
instead of stopping at the old 8x limit. Fit selected line uses the same projection
and padding as manual zoom; changing it has no effect on simulation state.

## Focused verification

`go test -p 1 ./pkg/terrain ./pkg/render ./pkg/mapview ./pkg/ui` checks the manifest,
coverage of every line, decoding and hashes, what is drawn for a view, the tile
cache's limit, georeferencing, projection and localization. Go tests cannot read
pixels back, so joins between tiles were checked by screenshot in the native app.
After `sh tools/build-web.sh`, `node tools/check-terrain.cjs` against a local server
records the WASM hash, toggle persistence, unchanged financial state/quotes, narrow
layouts, Chinese labels, relief around Bogota-Medellin and offline continuation in
`docs/validation/terrain-checks.json` and screenshots. It usually exceeds the local
2 GiB footprint limit, so hosted CI is its test. Inspect the screenshots; synthetic
browser checks cannot qualify physical phones or native accessibility.

## Basemap

`world-land.json` is one worldwide land outline, drawn in every frame. It comes
from the `land` geometry pinned in `data/construction/route-land.v1.json.gz`:
Natural Earth 1:10m countries merged into physical land, with Natural Earth 1:10m
lakes cut out. There are no lines between countries.

Within 150 km of any line the outline follows the source to 0.003 degrees (about
300 m); from there to 400 km, to 0.0125 degrees (about 1.4 km); farther away, to
0.05 degrees (about 5 km). Detail can change along one outline. The middle step
is there because a view of a line at regional zoom shows coast that far away:
without it eastern Hokkaido was drawn with edges 70 km long. "A line" is every
existing path in `playable-rail-map.json`, every buildable alignment in
`compiled-prospects.v1.json`, and both alignments of `hokkaido-extension` and
`tsuruga-osaka`. The Hokkaido and Kansai relief boxes count as on a line
throughout. Islands under 25 km across and more than 150 km from every line are
left out, as are lakes under 5 km across within 150 km of a line or under 100 km
across beyond it; a lake left out is drawn as land. No two land outlines
overlap, and an island inside a lake is cut to the lake as drawn.

The result is 84,164 points in 1,997 outlines: 1,500 land, 292 lakes and 205
islands inside lakes. Every buildable route lies on the drawn land or its
reviewed crossing. The tool simplifies less where one would not; none needed it.

`python3 tools/prepare-basemap.py` regenerates the file and its metadata, with no
downloads. `--check` verifies the committed files, and
`python3 tools/test-basemap.py` tests the rule in `tools/basemap_rule.py`. CI runs
both.

The renderer fills land by the even-odd rule, so a lake inside land is left
clear and an island inside the lake is land again (`pkg/render/land.go`). The file
also winds land and the islands inside lakes one way and lakes the other, for
readers that fill by winding. The renderer chooses how much of each outline to
draw from the zoom, to within one screen pixel (`pkg/render/basemap.go`). Relief
tiles are cut to this outline, so the terrain layer no longer draws a lake outline
of its own.

The basemap does not depend on the terrain toggle and changes no economic or
save input. It is presentation-only data, outside campaign hashes.
