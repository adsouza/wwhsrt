"""Compose the worldwide display basemap: one land outline, detailed near lines.

Reads the pinned Natural Earth 1:10m land (countries merged, lakes cut out) from
data/construction/route-land.v1.json.gz and applies tools/basemap_rule.py. Every buildable route must
lie on the drawn land or its reviewed crossing; where the usual tolerance would put one offshore, the
land it touches is simplified less. No downloads. Needs shapely and scipy.

    python3 tools/prepare-basemap.py             write assets/terrain/world-land.json and its metadata
    python3 tools/prepare-basemap.py --check     fail if the committed files differ
    python3 tools/prepare-basemap.py --compare F report existing-line points off the land in basemap file F
"""

import gzip
import hashlib
import json
import pathlib
import sys

import shapely
from shapely import STRtree
from shapely.geometry import LineString, Point, Polygon, shape
from shapely.ops import unary_union

sys.path.insert(0, str(pathlib.Path(__file__).resolve().parent))
import basemap_rule as rule  # noqa: E402

ROOT = pathlib.Path(__file__).resolve().parents[1]
MASK = "data/construction/route-land.v1.json.gz"
LOCAL_SHORES = "data/terrain/oresund-harbors.v1.geojson"
STOCKHOLM = "data/terrain/stockholm-shore.v1.geojson"
EXISTING = "assets/scenario/playable-rail-map.json"
PROSPECTS = "data/construction/compiled-prospects.v1.json"
NETWORK = "assets/scenario/public-network.json"
# Buildable routes that predate the prospect catalogue and appear only in the public network.
OLDER_ROUTES = ("hokkaido-extension", "tsuruga-osaka")
OUTPUT = "assets/terrain/world-land.json"
METADATA = "assets/terrain/world-land.metadata.json"


def sha(path):
    return hashlib.sha256((ROOT / path).read_bytes()).hexdigest()


def off_land(route, drawn, projected):
    """The first point of a route's path that is not on drawn land or its allowed crossing, or None."""
    _, _, path, allowed = route
    ground, flat = drawn, projected
    if allowed is not None and not allowed.is_empty:
        ground, flat = drawn.union(allowed), projected.union(rule.on_map(allowed))
    if ground.covers(LineString(path)) and flat.covers(LineString([rule.project(*p) for p in path])):
        return None
    samples = rule.sample(path, 1)
    for point in samples:  # where it leaves the land on the globe, if it does
        if not ground.covers(Point(point)):
            return point
    for point in samples:  # otherwise where the drawn, projected leg does
        if not flat.covers(Point(rule.project(*point))):
            return point
    return path[0]  # every sample is on land, so a leg leaves it between samples


def fit(polygons, index, routes):
    """Compose the basemap, simplifying less wherever a route would otherwise leave the drawn land.

    routes is a list of (route id, alignment id, path, allowed crossing or None). Returns the basemap
    and the list of {"route", "polygon", "tolerance"} tightenings, one per source polygon.
    """
    exteriors = [Polygon(p[0]) for p in polygons]
    tree = STRtree(exteriors)
    fine_for, reason = {}, {}
    while True:
        basemap = rule.compose(polygons, index, fine_for)
        drawn = rule.drawn_geometry(basemap)
        projected = rule.on_map(drawn)
        failing = [(route, point) for route in routes if (point := off_land(route, drawn, projected)) is not None]
        if not failing:
            tightened = [{"route": reason[i], "polygon": i, "tolerance": fine_for[i]} for i in sorted(fine_for)]
            return basemap, tightened
        progressed = False
        for route, point in failing:
            for i in sorted(int(k) for k in tree.query(LineString(route[2]), predicate="intersects")):
                step = rule.LADDER.index(fine_for.get(i, rule.FINE))
                if step + 1 < len(rule.LADDER):
                    fine_for[i] = rule.LADDER[step + 1]
                    reason.setdefault(i, route[0])
                    progressed = True
        if not progressed:
            route, point = failing[0]
            raise SystemExit(
                f"Route {route[0]} ({route[1]}) is off the drawn land at {point[0]}, {point[1]} "
                "with the outline unsimplified; it needs a reviewed crossing or a corrected path"
            )


def points_off_land(basemap, lines):
    """How many points sampled every 10 km along existing lines are off the drawn land."""
    if isinstance(basemap, list):  # the earlier format: a flat list of per-country outlines
        drawn = unary_union([Polygon(r).buffer(0) for r in basemap])
    else:
        drawn = rule.drawn_geometry(basemap)
    samples = [p for line in lines for p in rule.sample(line)]
    covered = shapely.covers(drawn, shapely.points(samples))
    return {"sampled": len(samples), "off_land": int(len(samples) - covered.sum())}


def inputs():
    mask = json.loads(gzip.decompress((ROOT / MASK).read_bytes()))
    existing = [[tuple(p) for p in line] for line in json.loads((ROOT / EXISTING).read_text())["lines"]]
    prospects = json.loads((ROOT / PROSPECTS).read_text())
    crossings = {
        r["corridor_id"]: unary_union([shape(c["geometry"]) for c in r["crossings"]]) for r in prospects["review"]
    }
    expansion = json.loads((ROOT / "data/imports/network-expansion.v1.json").read_bytes())
    for project in expansion["projects"]:
        # Allow only the surveyed deck of an existing bridge in this contract.
        decks = [LineString(leg["path"]).buffer(.0005) for leg in project["legs"] if leg.get("public")]
        if decks:
            crossings[project["id"]] = unary_union(decks)
    catalog = json.loads((ROOT / NETWORK).read_text())["corridors"]
    routes = []
    for co in catalog:
        if co["existing"] or co.get("part_of"):
            continue
        for alt in co["alignments"]:
            path = [(p["lon"], p["lat"]) for p in alt["path"]]
            if co["id"].startswith("upgrade_de_"):
                # These works follow existing railway bridges at water crossings.
                existing.append(path)
            else:
                routes.append((co["id"], alt["id"], path, crossings.get(co["id"])))
    return mask, existing, routes


def main():
    mask, existing, routes = inputs()
    if "--compare" in sys.argv:
        other = json.loads(pathlib.Path(sys.argv[sys.argv.index("--compare") + 1]).read_text())
        print(json.dumps(points_off_land(other, existing)))
        return
    boxes = [rule.tile_bounds(*b) for b in rule.JAPAN_BOXES]
    index = rule.NearIndex(existing + [r[2] for r in routes], boxes)
    composed, tightened = fit(mask["land"]["coordinates"], index, routes)
    basemap = {k: composed[k] for k in ("land", "lakes", "lake_islands")}
    rule.apply_local_shores(basemap, json.loads((ROOT / LOCAL_SHORES).read_bytes())["features"])
    # Here the coarse shoreline is the edge of Lake Mälaren inside the mainland.
    for feature in json.loads((ROOT / STOCKHOLM).read_bytes())["features"]:
        rule.apply_lake_shore(basemap, feature)
    counts = {k: len(v) for k, v in basemap.items()}
    counts["points"] = sum(len(r) for v in basemap.values() for r in v)
    counts["left_out_islands"] = composed["sources"]["left_out_islands"]
    counts["left_out_lakes"] = composed["sources"]["left_out_lakes"]
    outlines = counts["land"] + counts["lakes"] + counts["lake_islands"]
    assert counts["points"] <= rule.MAX_POINTS, f"{counts['points']} points exceed the budget"
    assert outlines <= rule.MAX_OUTLINES, f"{outlines} outlines exceed the budget"
    for name, rings in basemap.items():
        for ring in rings:
            assert len({tuple(p) for p in ring}) >= 3 and rule.signed_area(ring) != 0, f"A {name} outline has no area"
    # The renderer fills by the even-odd rule: land drawn twice would come out as water. Rounding to
    # six places can leave a hair's width along a cut.
    doubled = rule.overlap_area(basemap["land"])
    assert doubled < 1e-6, f"Land outlines overlap by {doubled} square degrees"
    output = (json.dumps(basemap, separators=(",", ":")) + "\n").encode()
    metadata = {
        "role": "Single worldwide land outline with lakes cut out; display only, no political boundaries",
        "source_mask_sha256": sha(MASK),
        "local_shorelines": {p: sha(p) for p in (LOCAL_SHORES, STOCKHOLM)},
        "source_countries_sha256": mask["source_countries_sha256"],
        "source_lakes_sha256": mask["source_lakes_sha256"],
        "license": mask["license"],
        "local_shoreline_license": "OpenStreetMap contributors, ODbL 1.0",
        "line_sources": {p: sha(p) for p in (EXISTING, PROSPECTS, NETWORK)},
        "near_km": rule.NEAR_KM,
        "fine_degrees": rule.FINE,
        "mid_km": rule.MID_KM,
        "mid_degrees": rule.MID,
        "coarse_degrees": rule.COARSE,
        "far_island_km": rule.FAR_ISLAND_KM,
        "near_lake_km": rule.NEAR_LAKE_KM,
        "far_lake_km": rule.FAR_LAKE_KM,
        "japan_boxes": [list(b) for b in rule.JAPAN_BOXES],
        "winding": "land and lake_islands counter-clockwise, lakes clockwise",
        "tightened": tightened,
        "counts": counts,
        "existing_line_points_off_land": points_off_land(basemap, existing),
        "output_sha256": hashlib.sha256(output).hexdigest(),
    }
    products = {OUTPUT: output, METADATA: (json.dumps(metadata, indent=2) + "\n").encode()}
    for path, data in products.items():
        target = ROOT / path
        if "--check" in sys.argv:
            if not target.exists() or target.read_bytes() != data:
                raise SystemExit("Generated asset differs: " + path)
        else:
            target.write_bytes(data)
    print(json.dumps({"counts": counts, "tightened": tightened}, indent=2))


if __name__ == "__main__":
    main()
