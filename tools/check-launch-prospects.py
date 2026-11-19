"""Independent catalog admission checks for all launch construction pairs."""

import gzip, hashlib, importlib.util, json, math, pathlib, re, sys
from shapely.geometry import shape, LineString
from shapely.ops import unary_union, transform

ROOT = pathlib.Path(__file__).resolve().parents[1]
sys.path.insert(0, str(pathlib.Path(__file__).resolve().parent))
import basemap_rule as rule  # noqa: E402


def read(p):
    return json.loads((ROOT / p).read_bytes())


def km(a, b):
    x, y, z, t = map(math.radians, (*a, *b))
    return 12742 * math.asin(
        min(1, math.sqrt(math.sin((t - y) / 2) ** 2 + math.cos(y) * math.cos(t) * math.sin((z - x) / 2) ** 2))
    )


def length(line):
    return sum(km(a, b) for a, b in zip(line.coords, line.coords[1:]))


def main():
    spec = importlib.util.spec_from_file_location("japan_geography", ROOT / "tools/check-route-geography.py")
    geo = importlib.util.module_from_spec(spec)
    spec.loader.exec_module(geo)
    source = read("data/construction/launch-prospects.v1.json")
    catalog = read("assets/scenario/public-network.json")
    byid = {c["id"]: c for c in catalog["corridors"]}
    cities = {c["id"]: c for c in catalog["cities"]}
    mask = json.loads(gzip.decompress((ROOT / "data/construction/route-land.v1.json.gz").read_bytes()))
    land = shape(mask["land"])
    projected = transform(geo.project, land)
    drawn = rule.drawn_geometry(read("assets/terrain/world-land.json"))
    projected_drawn = rule.on_map(drawn)
    profiles = {p["id"]: p for p in read("data/costs/construction_costs.v1.json")["profiles"]}
    prior = json.loads(gzip.decompress((ROOT / "assets/scenario/public-network-v3.json.gz").read_bytes()))
    old_markets = {c["id"]: c for c in prior["network"]["cities"]}
    additional_markets = {"antwerp", "rotterdam", "schiphol", "liege", "aachen", "duisburg", "arnhem", "utrecht", "narbonne", "carcassonne", "petrovaradin"}
    additional_markets |= set(read("data/imports/network-expansion.v1.json")["hubs"]) - {c["id"] for c in prior["cities"]}
    additional_markets |= set(read("data/imports/branch-projects.v1.json")["hubs"]) - {c["id"] for c in prior["cities"]}
    additional_markets |= set(read("data/imports/operating-hsr-additions.v1.json")["hubs"]) - {c["id"] for c in prior["cities"]}
    assert set(cities) == additional_markets | set(old_markets) | {
        "taipei",
        "maibara",
        "camp_de_tarragona",
        "figueres",
    }, "Unexpected city roster change"
    assert cities["taipei"]["station_name"] == "Taipei"
    for c in catalog["network"]["cities"]:
        if c["id"] in additional_markets | {"maibara", "camp_de_tarragona", "figueres"}:
            assert c["daily_trips"] == c["attraction"] == 500
        elif c["id"] == "taipei":
            assert c["daily_trips"] == 7000 and c["attraction"] == 7000 and c["road_region"] == "taiwan"
        else:
            assert all(
                c[k] == old_markets[c["id"]][k] for k in ["daily_trips", "attraction", "road_region", "lat", "lon"]
            ), "Expansion changes pre-existing demand inputs"
    declared = re.findall(r"^\| (\d+) \| ([^|]+) \|", (ROOT / "docs/LAUNCH_CORRIDORS.md").read_text(), re.M)
    assert [(int(i), n.strip()) for i, n in declared] == [
        (r["number"], r["name"]) for r in source["pairs"]
    ], "Spec/authoring coverage differs"
    assert len(source["pairs"]) == 74 and len([c for c in byid.values() if not c["existing"] and not c.get("part_of") and not c.get("retired")]) == 101
    checks = []

    def admit(path, allowed, bridge_km):
        line = LineString(path)
        ground = land.union(allowed)
        assert ground.covers(line), "Unpriced sea/lake crossing"
        assert projected.union(transform(geo.project, allowed)).covers(
            LineString([geo.project(*p) for p in path])
        ), "Rendered sea/lake crossing"
        assert drawn.union(allowed).covers(line), "Path appears over water on the actual basemap"
        assert projected_drawn.union(transform(geo.project, allowed)).covers(
            LineString([geo.project(*p) for p in path])
        ), "Projected path leaves actual drawn land"
        water = line.difference(land)
        pieces = list(water.geoms) if hasattr(water, "geoms") else [water]
        width = sum(length(s) for s in pieces if s.geom_type == "LineString")
        assert bridge_km >= math.ceil(width - 1e-8), "Water crossing has no priced structure quantity"
        return width

    for row in source["pairs"]:
        id = row.get("legacy_id") or row["id"]
        co = byid[id]
        assert not co["existing"] and {co["from_city"], co["to_city"]} == {row["from_city"], row["to_city"]}
        if row.get("legacy_id"):
            continue  # Existing Japan mask checker covers these unchanged alternatives.
        a = co["alignments"][0]
        path = [(p["lon"], p["lat"]) for p in a["path"]]
        assert len(path) <= 64 and a["surface_km"] + a["hill_km"] + a["viaduct_km"] + a["tunnel_km"] == a["km"]
        assert 1 <= a["months"] <= 120 and 1 <= a["km"] <= 6000
        for end, id in [(0, co["from_city"]), (-1, co["to_city"])]:
            c = cities[id]
            assert path[end] == (
                c.get("station_lon", c["lon"]),
                c.get("station_lat", c["lat"]),
            ), "Detached construction terminal"
        allowed = unary_union([shape(c["geometry"]) for c in row.get("crossings", [])])
        try:
            water = admit(path, allowed, a["viaduct_km"] + a["tunnel_km"])
        except AssertionError as error:
            raise AssertionError((row["number"], co["id"], str(error))) from error
        assert length(LineString(path)) <= a["km"]
        for key in ["surface_km", "hill_km", "viaduct_km", "tunnel_km"]:
            assert sum(i[key] for i in a["cost_intervals"]) == a[key]
        for interval in a["cost_intervals"]:
            assert (
                interval["country"]
                and interval["base_per_km"]
                == profiles[interval["profile_id"]]["baseline_million_2024_usd_per_route_km"] * 100_000_000
            )
        checks.append(
            dict(
                number=row["number"],
                id=co["id"],
                priced_km=a["km"],
                water_km=round(water, 3),
                countries=[p["country"] for p in a["cost_intervals"]],
            )
        )
    # Deliberate failures: straight Gulf of Guinea chord; missing Johor structures.
    gulf = next(r for r in source["pairs"] if r["number"] == 68)
    co = byid[gulf["id"]]
    a = co["alignments"][0]
    try:
        admit(
            [(p["lon"], p["lat"]) for p in [a["path"][0], a["path"][-1]]],
            unary_union([shape(c["geometry"]) for c in gulf["crossings"]]),
            a["viaduct_km"],
        )
    except AssertionError:
        pass
    else:
        raise AssertionError("Straight Gulf of Guinea shortcut was accepted")
    row = source["pairs"][69]
    a = byid[row["id"]]["alignments"][0]
    try:
        admit(
            [(p["lon"], p["lat"]) for p in a["path"]], unary_union([shape(c["geometry"]) for c in row["crossings"]]), 0
        )
    except AssertionError:
        pass
    else:
        raise AssertionError("Unpriced Johor Strait crossing was accepted")
    for id in ["public_casablanca__rabat", "public_kenitra__rabat"]:
        assert id not in byid and all(
            id not in r["sections"] for r in catalog["network"]["routes"]
        ), "Conventional Moroccan railway reclassified as public HSR"
    moroccan = {c["id"] for c in catalog["network"]["cities"] if c["road_region"] == "morocco"}
    assert {
        co["id"]
        for co in byid.values()
        if co["existing"] and (co["from_city"] in moroccan or co["to_city"] in moroccan)
    } == {"public_kenitra__tangier"}, "Unexpected Moroccan starter HSR"
    assert byid["public_kenitra__tangier"]["existing"]
    assert (
        byid["public_lisbon__porto"]["existing"]
        and byid["public_lisbon__porto"]["infrastructure_kind"] == "upgraded_conventional"
    )
    assert byid[source["pairs"][25]["id"]]["parallel_to"] == ["public_lisbon__porto"]
    result = dict(
        passed=True,
        prospects=74,
        new_alignments_checked=len(checks),
        catalog_sha256=hashlib.sha256((ROOT / "assets/scenario/public-network.json").read_bytes()).hexdigest(),
        checks=checks,
        negative_regressions=["Gulf of Guinea shortcut rejected", "Unpriced Johor Strait crossing rejected"],
        limits=[
            "Natural Earth cartographic screening and authored engineering quantities, "
            "not surveyed gradients, curvature, rights or procurement estimates.",
            "Legacy Japanese paths use their separate pinned detailed mask and unchanged costs.",
        ],
    )
    (ROOT / "docs/validation/launch-prospects-geography.json").write_text(json.dumps(result, indent=2) + "\n")
    print(
        "Passed all 74 spec pairs, 72 new alignments in both projections, pricing inputs and negative crossing fixtures"
    )


if __name__ == "__main__":
    main()
