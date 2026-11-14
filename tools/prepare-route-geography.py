"""Prepare pinned regional water-check geometry; ordinary builds need no GIS tools."""

import hashlib
import json
from pathlib import Path
from shapely.geometry import box, mapping, shape

ROOT = Path(__file__).resolve().parents[1]
source = ROOT / "build/terrain-source/ne_10m_admin_0_countries.geojson"
coast_meta = json.loads((ROOT / "assets/terrain/land.metadata.json").read_text())
raw = source.read_bytes()
assert hashlib.sha256(raw).hexdigest() == coast_meta["source_sha256"], "Unreviewed coastline input"
country = next(f for f in json.loads(raw)["features"] if f["properties"]["ADMIN"] == "Japan")
bounds = [134, 34, 142.5, 44.5]
land = shape(country["geometry"]).intersection(box(*bounds))
references = ROOT / "assets/terrain/features.json"
features = json.loads(references.read_text())
biwa = features["lakes"][0]
assert biwa["key"] == "lake_biwa"
lakes = [
    {
        "id": "lake_biwa",
        "kind": "shoreline",
        "geometry": {"type": "Polygon", "coordinates": [[[lon, lat] for lat, lon in biwa["points"]]]},
    }
]
# Deliberately conservative review envelopes from the regional reference maps.
# These are avoidance areas, not claimed lake shorelines or rendered polygons.
for name, bbox in [
    ("lake_toya", [140.78, 42.555, 140.94, 42.655]),
    ("lake_shikotsu", [141.22, 42.68, 141.50, 42.84]),
    ("lake_onuma", [140.66, 41.96, 140.72, 42.02]),
]:
    lakes.append({"id": name, "kind": "conservative_avoidance_envelope", "geometry": mapping(box(*bbox))})
mask = {
    "schema_version": 1,
    "coverage_bbox": bounds,
    "detailed_land": mapping(land),
    "lakes": lakes,
    "role": "Current Japan game corridors only; sea/lake cartographic checks, "
    "not engineering qualification. Small rivers and other unmapped waters are outside this mask.",
}
out = ROOT / "data/geography/japan-route-mask.v1.json"
out.write_text(json.dumps(mask, separators=(",", ":")) + "\n")
meta = {
    "schema_version": 1,
    "output_sha256": hashlib.sha256(out.read_bytes()).hexdigest(),
    "coast_source_url": coast_meta["source_url"],
    "coast_source_sha256": coast_meta["source_sha256"],
    "lake_reference_sha256": hashlib.sha256(references.read_bytes()).hexdigest(),
    "lake_source_sha256": features["natural_earth_source_sha256"],
    "lake_reference_sources": features["sources"],
    "envelopes_note": "Approximate conservative review bounds for Toya, Shikotsu and Onuma; "
    "not surveyed or drawn shorelines.",
    "license": "Natural Earth public domain; envelopes are authored reference bounds.",
}
out.with_suffix(".metadata.json").write_text(json.dumps(meta, indent=2) + "\n")
print("Prepared pinned Japan route mask:", out.stat().st_size, "bytes")
