# Regional terrain context

Native and browser maps share offline shaded relief around the current Hokkaido
and Kansai construction corridors. Elevation tint, northwest hillshade, 500 m
contours, named peaks/uplands and lowland labels help explain route choices.
Lake Biwa has a Natural Earth outline; Toya and Shikotsu use **point symbols**,
not invented lake boundaries. Japan's finer shoreline keeps world-scale coastal
polygons from obscuring local terrain.

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
derived coastline hash. Japan exterior rings, simplified by 0.003 degrees, replace
its old 1:110m rings; other countries retain their previous geometry. Sources cache
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
