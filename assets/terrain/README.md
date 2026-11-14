# Regional terrain context

Native and browser maps share offline shaded relief around the current Hokkaido
and Kansai construction corridors. Elevation tint, northwest hillshade, 500 m
contours, named peaks/uplands and lowland labels help explain route choices.
Lake Biwa has a Natural Earth outline; Toya and Shikotsu use **point symbols**,
not invented lake boundaries. The finer Japan and China/Taiwan shorelines replace the world-scale outlines.
A single basemap prevents coarse coast polygons from obscuring local terrain.

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
image hashes. `land.metadata.json` pins the Natural Earth countries source and
derived coastline hash. Japan, China (including Hong Kong/Macao physical land)
and Taiwan exterior rings, simplified by 0.003 degrees, replace their old 1:110m
rings; other countries retain their previous geometry. Hong Kong/Macao are unioned
with the adjacent physical land to avoid internal overlay seams. Sources cache
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

`python3 tools/prepare-shorelines.py` regenerates only those shoreline refinements
from the pinned local Natural Earth 1:10m source (fetching/hash-checking it if the
cache is absent). `--check` verifies identical output. The full relief preparer
calls the same helper. Then run `python3 tools/import-public-network.py` to compose
`world-land.json`: remove all 17 replaced coarse regional rings, append the refined
regional set exactly once, and record source/output hashes in
`world-land.metadata.json`. Runtime public-network frames draw that combined
basemap only; legacy fixture frames draw only the regional set. Neither depends on
the terrain visibility toggle, and neither changes economic/save inputs.


The worldwide map also replaces coarse Australia, South Africa and Chile
shorelines using the same pinned Natural Earth 1:10m country source in the launch
construction mask. These corrections keep four new land alignments from appearing
offshore. The public compiler records simplification/source metadata and removes
the old rings; the construction checker tests the actual drawn geometry in both
coordinate spaces. This remains presentation-only data, outside campaign hashes.
