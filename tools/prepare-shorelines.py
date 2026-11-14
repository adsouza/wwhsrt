"""Prepare regional display coastlines from pinned Natural Earth data.

Independent of relief-tile generation. Uses the existing local source cache or
fetches the original source, checking its exact hash before producing geometry.
"""

import hashlib
import json
import pathlib
import subprocess
import sys
from shapely.geometry import Polygon, shape
from shapely.ops import unary_union

ROOT = pathlib.Path(__file__).resolve().parents[1]
SOURCE_URL = (
    "https://raw.githubusercontent.com/nvkelso/natural-earth-vector/master/geojson/ne_10m_admin_0_countries.geojson"
)
SOURCE_SHA = "239eec57ac17f100a11e2536cffc56752c318b50ae765b0918ff7aab4ce8f255"
CACHE = ROOT / "build/terrain-source/ne_10m_admin_0_countries.geojson"


def key(r):
    return json.dumps(r, separators=(",", ":"))


def main():
    if not CACHE.exists():
        CACHE.parent.mkdir(parents=True, exist_ok=True)
        subprocess.run(
            [
                "curl",
                "--fail",
                "--silent",
                "--show-error",
                "--location",
                "--max-time",
                "45",
                "--output",
                str(CACHE),
                SOURCE_URL,
            ],
            check=True,
        )
    raw = CACHE.read_bytes()
    if hashlib.sha256(raw).hexdigest() != SOURCE_SHA:
        raise SystemExit("Shoreline source changed; review before repinning")
    detailed = {f["properties"]["ADMIN"]: f["geometry"] for f in json.loads(raw)["features"]}
    coarse = json.loads((ROOT / "data/prototypes/natural-earth-110m.countries.geojson").read_text())
    targets = {"Japan", "China", "Taiwan"}
    removed = set()
    for f in coarse["features"]:
        if f["properties"]["name"] in targets:
            g = f["geometry"]
            ps = g["coordinates"] if g["type"] == "MultiPolygon" else [g["coordinates"]]
            removed.update(key(p[0]) for p in ps)
    base = json.loads((ROOT / "assets/scenario/land.json").read_text())
    assert removed <= {key(r) for r in base}
    rings = [r for r in base if key(r) not in removed]
    counts = {}
    for name in ["Japan", "China", "Taiwan"]:
        if name == "China":
            # Dissolve adjacent SAR boundaries; this layer describes physical
            # land/coast, without adding internal lines along Hong Kong/Macao.
            g = unary_union([shape(detailed[n]) for n in ["China", "Hong Kong S.A.R.", "Macao S.A.R"]])
            polys = list(g.geoms) if g.geom_type == "MultiPolygon" else [g]
            exteriors = [p.exterior.coords for p in polys]
        else:
            g = detailed[name]
            ps = g["coordinates"] if g["type"] == "MultiPolygon" else [g["coordinates"]]
            exteriors = [p[0] for p in ps]
        result = []
        for exterior in exteriors:
            coords = Polygon(exterior).simplify(0.003, preserve_topology=True).exterior.coords
            result.append([[round(x, 6), round(y, 6)] for x, y in coords])
        rings.extend(result)
        counts[name] = {"rings": len(result), "vertices": sum(len(r) for r in result)}
    output = (json.dumps(rings, separators=(",", ":")) + "\n").encode()
    metadata = {
        "source_url": SOURCE_URL,
        "source_sha256": SOURCE_SHA,
        "output_sha256": hashlib.sha256(output).hexdigest(),
        "role": "Display-only Japan and China/Taiwan shoreline refinement; no gameplay or political boundaries.",
        "simplification_degrees": 0.003,
        "license": "Natural Earth public domain",
        "refinements": counts,
        "china_geometry_note": "Physical land union includes Hong Kong/Macao to retain their coast and suppress internal overlay seams.",
    }
    products = {"land.json": output, "land.metadata.json": (json.dumps(metadata, indent=2) + "\n").encode()}
    for name, data in products.items():
        path = ROOT / "assets/terrain" / name
        if "--check" in sys.argv:
            if path.read_bytes() != data:
                raise SystemExit("Shoreline output differs: " + str(path))
        else:
            path.write_bytes(data)
    print(json.dumps(counts, indent=2))


if __name__ == "__main__":
    main()
