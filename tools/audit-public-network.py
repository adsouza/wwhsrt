"""Read-only connectivity audit; proximity candidates are never auto-joined.

Uses the compiler's exact graph before output emission, then independently
queries disconnected geometry and checks operator-backed service expectations.
Default writes audit artifacts only. --check verifies those artifacts;
--strict additionally fails if expected real-world connections remain missing.
"""

import argparse
import collections
import csv
import hashlib
import importlib.util
import io
import json
import math
import pathlib

from scipy.spatial import cKDTree
from shapely.geometry import Point

ROOT = pathlib.Path(__file__).resolve().parents[1]
OUT = ROOT / "docs/validation/network-topology-audit.json"


def sha(path):
    return hashlib.sha256((ROOT / path).read_bytes()).hexdigest()


def audit():
    spec = importlib.util.spec_from_file_location("rail_compiler", ROOT / "tools/import-public-network.py")
    compiler = importlib.util.module_from_spec(spec)
    spec.loader.exec_module(compiler)
    graph = compiler.main(audit=True)
    coords, components = graph["coords"], graph["components"]
    xyz = lambda p: (
        math.cos(math.radians(p[1])) * math.cos(math.radians(p[0])),
        math.cos(math.radians(p[1])) * math.sin(math.radians(p[0])),
        math.sin(math.radians(p[1])),
    )
    hub_cities = collections.defaultdict(list)
    for city, (node, _) in graph["city_pins"].items():
        hub_cities[components[node]].append(city)
    summaries = {}
    for cid, nodes in enumerate(graph["comp_nodes"]):
        stations = graph["by_component"][cid]
        summaries[cid] = dict(
            nodes=len(nodes),
            stations=len(stations),
            independently_evidenced_or_tagged_stations=sum(bool(s["operator"]) or s["tagged"] for s in stations),
            city_hubs=sorted(hub_cities[cid]),
        )
    endpoints = sorted(
        {graph["index"][tuple(round(x, 7) for x in p)] for line in graph["lines"] for p in (line[0], line[-1])}
    )
    nearest = cKDTree([xyz(coords[n]) for n in endpoints])
    station_tree = cKDTree([xyz((s["lon"], s["lat"])) for s in graph["stations"]])

    def nearby_station(p):
        _, index = station_tree.query(xyz(p))
        station = graph["stations"][index]
        return dict(
            id=station["id"],
            name=station["name"],
            distance_km=round(compiler.km(p, (station["lon"], station["lat"])), 2),
        )

    candidates = []

    def add(kind, u, v, p, q):
        a, b = components[u], components[v]
        meters = compiler.km(p, q) * 1000
        if a == b or meters > 2000 or not summaries[a]["stations"] or not summaries[b]["stations"]:
            return
        candidates.append(
            dict(
                kind=kind,
                meters=round(meters, 2),
                a=p,
                b=q,
                components=[a, b],
                nearby_stations=[nearby_station(p), nearby_station(q)],
            )
        )

    for left, right in sorted(nearest.query_pairs(2000 / 6371000)):
        u, v = endpoints[left], endpoints[right]
        add("endpoint_to_endpoint", u, v, coords[u], coords[v])
    for u in endpoints:
        p = coords[u]
        if not summaries[components[u]]["stations"]:
            continue
        radius = 2000 / (111000 * max(0.1, math.cos(math.radians(p[1]))))
        point = Point(p)
        for line_id in sorted(graph["tree"].query(point.buffer(radius))):
            line = graph["lines"][line_id]
            v = graph["index"][tuple(round(x, 7) for x in line[0])]
            if components[u] == components[v] or not summaries[components[v]]["stations"]:
                continue
            shape = graph["shapes"][line_id]
            projected = shape.interpolate(shape.project(point))
            q = (projected.x, projected.y)
            if min(compiler.km(q, line[0]), compiler.km(q, line[-1])) * 1000 < 10:
                continue  # Already represented by the endpoint search.
            add("endpoint_to_interior", u, v, p, q)
    groups = {}
    for row in candidates:
        # Cluster repeated parallel-track findings; this is a review-site count,
        # not a count of missing junctions or certified repairs.
        key = tuple(sorted(row["components"])) + (round(row["a"][0] * 40), round(row["a"][1] * 40))
        if key not in groups or row["meters"] < groups[key]["meters"]:
            groups[key] = row
    candidates = sorted(groups.values(), key=lambda r: (r["meters"], r["components"], r["a"]))
    repairs = json.loads((ROOT / "data/imports/network-repairs.v2.json").read_bytes())
    reviewed = {(r["kind"], tuple(r["a"]), tuple(r["b"])): r for r in repairs["review_sites"]}
    for row in candidates:
        repair = reviewed.get((row["kind"], tuple(row["a"]), tuple(row["b"])), {})
        row["status"] = repair.get("status", "requires_junction_or_source_review")
        row["source_node_distance_m"] = repair.get("source_distance_m", [])
        row["component_details"] = [summaries[c] for c in row["components"]]

    catalog = json.loads((ROOT / "assets/scenario/public-network.json").read_text())
    adjacency = collections.defaultdict(set)
    for corridor in catalog["corridors"]:
        if corridor["existing"]:
            a, b = corridor["from_city"], corridor["to_city"]
            adjacency[a].add(b)
            adjacency[b].add(a)
    cities = {c["id"]: c for c in catalog["cities"]}
    expectations = json.loads((ROOT / "data/imports/network-connectivity-expectations.v1.json").read_text())
    checks = []
    for pair in expectations["pairs"]:
        seen, queue = {pair["from"]}, [pair["from"]]
        for node in queue:
            for other in sorted(adjacency[node] - seen):
                seen.add(other)
                queue.append(other)
        row = dict(pair)
        row["status"] = (
            "endpoint_missing_from_catalog"
            if pair["from"] not in cities or pair["to"] not in cities
            else "connected"
            if pair["to"] in seen
            else "missing_expected_connection"
        )
        row["selected_stations"] = {id: cities.get(id, {}).get("station_name") for id in (pair["from"], pair["to"])}
        checks.append(row)
    from network_coverage import checks as network_checks

    operating_checks = network_checks(
        catalog, json.loads((ROOT / "data/imports/operating-networks.v2.json").read_bytes())
    )
    raw_source = json.loads((ROOT / "data/imports/hsr-prototype.v1.json").read_text())
    admitted = {s["id"] for s in graph["stations"]}
    excluded = [
        dict(id=id, name=name, lon=lon, lat=lat, tagged=bool(tag), operator=operator)
        for id, name, lon, lat, tag, operator in raw_source["stations"]
        if id not in admitted
    ]
    return dict(
        repair_sha256=sha("data/imports/network-repairs.v2.json"),
        source_topology_sha256=sha("data/imports/rail-topology-source.v1.jsonl.gz"),
        source_sha256=sha("data/imports/hsr-prototype.v1.json"),
        connector_source_sha256=sha("data/imports/rail-connectors.v2.jsonl.gz"),
        network_manifest_sha256=sha("data/imports/operating-networks.v2.json"),
        operating_network_checks=operating_checks,
        playable_catalog_sha256=sha("assets/scenario/public-network.json"),
        expectations_sha256=sha("data/imports/network-connectivity-expectations.v1.json"),
        scope=dict(
            polylines=len(graph["lines"]),
            source_stations=len(raw_source["stations"]),
            admitted_stations=len(graph["stations"]),
            graph_components=len(summaries),
            station_bearing_components=sum(v["stations"] > 0 for v in summaries.values()),
            components_with_no_city_hub=sum(not v["city_hubs"] for v in summaries.values()),
            review_sites_within_2km=len(candidates),
            restored_source_sites=sum(r["status"] == "connected_in_source_node_topology" for r in candidates),
            unresolved_review_sites=sum(r["status"] == "requires_junction_or_source_review" for r in candidates),
            review_sites_by_kind=dict(sorted(collections.Counter(r["kind"] for r in candidates).items())),
            review_sites_within_150m=sum(r["meters"] <= 150 for r in candidates),
        ),
        service_expectations=checks,
        proximity_review_sites=candidates,
        excluded_source_stations=excluded,
        limits=[
            "Proximity does not establish a junction, compatible gauge or real train service.",
            "The 2 km search is not a global completeness certificate; "
            "longer missing approaches and tunnels need source restoration.",
            "A city pinned to one station/component can lose other genuine terminals and branches.",
            "Operator service evidence establishes required connectivity, not the exact route geometry to add.",
        ],
    )


def review_markdown(result):
    sites = [r for r in result["proximity_review_sites"] if r["status"] == "requires_junction_or_source_review"]
    lines = [
        "# Unverified proximity review sites",
        "",
        "These are " + str(len(sites)) + " review candidates from the older display-geometry audit, "
        "not confirmed missing railway links. "
        "Each compares fragments within 2 km that remain unconnected by the current source-node assessment. "
        "Close parallel tracks, grade-separated crossings, "
        "unused fragments and omitted approaches can all produce candidates.",
        "",
        "Nearest station names are location hints, not verified junction names. "
        "Some markers are more than 20 km from the candidate. "
        "The coordinate locates the first compared fragment; "
        "separation measures fragment-to-fragment distance, not distance between stations. "
        "Older display component memberships are not current playable-catalog connectivity.",
        "",
        "The whole-network contracts pass independently. For example, "
        "the Kaifeng candidate concerns the old conventional-station area; "
        "the playable city now uses Kaifeng North and is connected. "
        "These sites are deliberately not snapped together.",
        "",
        "| # | Nearest source station label(s) | Separation | Comparison | Location (latitude, longitude) |",
        "| --- | --- | ---: | --- | --- |",
    ]
    for i, r in enumerate(sites, 1):
        names = " / ".join(dict.fromkeys(s["name"] for s in r["nearby_stations"])).replace("|", " / ")
        lon, lat = r["a"]
        url = f"https://www.openstreetmap.org/?mlat={lat:.5f}&mlon={lon:.5f}#map=15/{lat:.5f}/{lon:.5f}"
        lines.append(
            f"| {i} | {names} | {r['meters']:,.2f} m | {r['kind'].replace('_to_',' to ').replace('_',' ')} | [{lat:.5f}, "
            f"{lon:.5f}]({url}) |"
        )
    lines += [
        "",
        "Source: [full JSON audit](network-topology-audit.json), including both fragment coordinates, "
        "nearest-marker distances and raw display component details. "
        "[Repair scope and qualification](NETWORK_TOPOLOGY.md).",
        "",
    ]
    return "\n".join(lines).encode()


def main():
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument("--check", action="store_true")
    parser.add_argument("--strict", action="store_true")
    args = parser.parse_args()
    catalog_before = sha("assets/scenario/public-network.json")
    result = audit()
    data = (json.dumps(result, ensure_ascii=False, indent=2) + "\n").encode()
    stream = io.StringIO(newline="")
    writer = csv.writer(stream)
    writer.writerow(
        [
            "kind",
            "meters",
            "lon_a",
            "lat_a",
            "lon_b",
            "lat_b",
            "station_a",
            "station_b",
            "component_a",
            "component_b",
            "status",
        ]
    )
    for r in result["proximity_review_sites"]:
        writer.writerow(
            [
                r["kind"],
                r["meters"],
                *r["a"],
                *r["b"],
                *(s["name"] for s in r["nearby_stations"]),
                *r["components"],
                r["status"],
            ]
        )
    outputs = [
        (OUT, data),
        (OUT.with_suffix(".csv"), stream.getvalue().encode()),
        (OUT.with_name("PROXIMITY_REVIEW.md"), review_markdown(result)),
    ]
    for path, value in outputs:
        if args.check:
            if not path.exists() or path.read_bytes() != value:
                raise SystemExit("Audit artifact differs: " + str(path.relative_to(ROOT)))
        else:
            path.write_bytes(value)
    assert sha("assets/scenario/public-network.json") == catalog_before, "Audit mutated catalog"
    missing = [r for r in result["service_expectations"] if r["status"] != "connected"]
    print(json.dumps(dict(**result["scope"], missing_operator_backed_connections=len(missing)), indent=2))
    failed_networks = [r["id"] for r in result["operating_network_checks"] if not r["connected"]]
    print("Whole-network checks:", len(result["operating_network_checks"]), "failed:", failed_networks)
    if args.strict and failed_networks:
        raise SystemExit("Whole-network qualification FAILED: " + ", ".join(failed_networks))
    if args.strict and missing:
        raise SystemExit("Connectivity qualification FAILED: " + ", ".join(r["from"] + "–" + r["to"] for r in missing))


if __name__ == "__main__":
    main()
