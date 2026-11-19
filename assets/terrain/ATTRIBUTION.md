# Terrain credits

Elevation: Mapzen Terrain Tiles, distributed through the AWS Open Data registry.
SRTM and GMTED2010 terrain data courtesy of the U.S. Geological Survey.
Ocean bathymetry source: NOAA ETOPO1; oceans are transparent in these relief overlays.

Dataset: https://registry.opendata.aws/terrain-tiles/
Formats and decoding: https://github.com/tilezen/joerd/blob/master/docs/formats.md
Source and license details: https://github.com/tilezen/joerd/blob/master/docs/attribution.md

SRTM and GMTED2010 are United States Government material, not subject to copyright
protection under Title 17 U.S.C. within the United States. SRTM's source terms
also state: copyright by the United States Government; no copyright claimed under
Title 17 U.S.C. See the linked source terms for distribution outside the US.

Worldwide land outline, its lakes, and the Lake Biwa reference outline:
Natural Earth 1:10m countries and lakes, public domain.
https://www.naturalearthdata.com/about/terms-of-use/
https://www.naturalearthdata.com/downloads/10m-cultural-vectors/10m-admin-0-countries/

Regional reference labels use the Japan Ministry of the Environment's park maps:
https://www.env.go.jp/park/content/000169509.pdf
Labels and small lake point symbols are approximate context, not surveyed locations.

Raster sources, SHA-256 identities, imagery-source headers and generation details
are recorded in data/terrain/sources.v1.json, terrain.json and tools/prepare-terrain.py. Ordinary game builds
and offline play use the checked-in derived assets without downloading terrain.


The worldwide land outline is derived from the pinned mask in
`data/construction/route-land.v1.json.gz`: Natural Earth 1:10m countries merged
into land, with Natural Earth 1:10m lakes cut out (public domain). It is screened
against every buildable route; its sources and hashes are in
`world-land.metadata.json`.

The shorelines around København H and Helsingborg C use the 15 November 2026
OpenFreeMap water layer. The local land polygons are the complement of mapped
water within each harbour window, simplified to approximately 25 metres.
The windows, source tile addresses and SHA-256 digests are pinned in
`data/terrain/oresund-harbors.v1.geojson`.
[OpenFreeMap](https://openfreemap.org/) · [© OpenMapTiles](https://openmaptiles.org/) ·
[© OpenStreetMap contributors, ODbL 1.0](https://www.openstreetmap.org/copyright).
