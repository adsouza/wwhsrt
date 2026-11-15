# Regional terrain context

Native and browser maps share offline shaded relief around the current Hokkaido
and Kansai construction corridors. Elevation tint, northwest hillshade, 500 m
contours, named peaks/uplands and lowland labels help explain route choices.
Lake Biwa has a Natural Earth outline; Toya and Shikotsu use **point symbols**,
not invented lake boundaries. The land under the relief is one worldwide outline,
described under Basemap below.

This is presentation data, outside scenario hashes, costs, gradients, schedules
and save compatibility. Railway sketches and engineering quantities remain authored
game estimates, not surveyed alignments. Relief cannot prove an alternative feasible
or optimal. Only these two corridor regions have relief coverage; unshaded land
elsewhere does not imply flat terrain.

## Reproducing

Install `tools/requirements-terrain.txt` into an isolated Python environment, then
run `python3 tools/prepare-terrain.py`. Ordinary builds/play need no Python or network.

The tool fetches ten zoom-8 Terrarium source tiles from Mapzen Terrain Tiles on AWS,
checks existing source hashes, decodes metres as `(R*256 + G + B/256)-32768`, and
derives PNG relief and contours. Source Web Mercator pixels convert to geographic
coordinates and **Equal Earth** mesh vertices at render time; the camera and picking
remain Equal Earth. Hillshade exaggerates vertical relief 2.5 times for legibility.
Ocean/negative elevation is transparent. Sources include SRTM, GMTED2010 and ETOPO1.
These coarse display samples do not replace the future engineering profile pipeline.
Contours are simplified display lines at 500, 1000 and 1500 m; higher elevations
remain visible through relief tint and shading.

`terrain.json` pins source URLs, raw hashes, imagery headers, dimensions and derived
image hashes. `world-land.metadata.json` pins the basemap's sources, rule and
output hash. Sources cache
under ignored `build/terrain-source`. Changed inputs require provenance review before
repinning. `features.json` contains approximate reference labels and the Natural Earth
Lake Biwa extraction, with source links/hash; the script does not generate that file.

See [credits](ATTRIBUTION.md) for source terms. Credits ship in the native bundle
and browser offline cache. The Terrain button persists the visibility preference.
City labels take priority over terrain text. Alignment descriptions disclose that
the detours still need tunnels.

The shared camera allows up to 32× zoom so short corridors can fill a phone map
instead of stopping at the old 8× limit. Fit selected line uses the same projection
and padding as manual zoom; changing it has no effect on simulation state.

## Focused verification

`go test -p 1 ./pkg/terrain ./pkg/mapview ./pkg/ui` checks integrity, georeferencing,
projection and localization. After `sh tools/build-web.sh`, serve `web` on localhost
and run `node tools/check-terrain.cjs`. Screenshots and `docs/validation/terrain-checks.json`
record the WASM hash, toggle persistence, unchanged financial state/quotes, narrow
layouts, Chinese labels and offline continuation. Inspect the screenshots; synthetic
browser checks cannot qualify physical phones or native accessibility.

## Basemap

`world-land.json` is one worldwide land outline, drawn in every frame. It comes
from the `land` geometry pinned in `data/construction/route-land.v1.json.gz`:
Natural Earth 1:10m countries merged into physical land, with Natural Earth 1:10m
lakes cut out. There are no lines between countries.

Within 150 km of any line the outline follows the source to 0.003 degrees (about
300 m); farther away to 0.05 degrees (about 5 km). Detail can change along one
outline. "A line" is every existing path in `playable-rail-map.json`, every
buildable alignment in `compiled-prospects.v1.json`, and both alignments of
`hokkaido-extension` and `tsuruga-osaka`. The Hokkaido and Kansai relief boxes
count as near a line throughout, and so does the whole of Hokkaido, whose east
is otherwise drawn to 5 km on the island the first scenario is played on.
Islands under 25 km across and far from every line are left out, as are lakes
under 5 km across near a line or under 100 km across far from one; a lake left
out is drawn as land. No two land outlines overlap, and an island inside a lake
is cut to the lake as drawn.

The result is 74,361 points in 2,002 outlines: 1,505 land, 292 lakes and 205
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
draw from the zoom, to within one screen pixel (`pkg/render/basemap.go`). The
terrain layer still draws its own outline of Lake Biwa, because
today's relief tints the lake's surface; that goes when relief is cut to the
basemap.

The basemap does not depend on the terrain toggle and changes no economic or
save input. It is presentation-only data, outside campaign hashes.
