"""Derive candidate catalog cities served by the provisional fast-track station layer.

It also applies a corridor screen: track from the same preview is grouped into connected
components, each station is matched to a component within STATION_TRACK_KM, and each city
is classed by whether its track can form a corridor with another catalog city.

This is a screening aid for the planned ~250-city catalog, not qualified starting
infrastructure. Each station from the launch network preview is assigned to the most
populous GeoNames populated place within ASSIGN_KM. Cities within METRO_KM of one of
the playable metros are reported as covered rather than listed.

Usage (run with python3 -I):
  python3 -I tools/derive-hsr-stop-cities.py \
      "$WWHSRT_MAP_PREVIEW_DIR/hsr-network-overlay.html" /path/to/cities15000.txt \
      [/path/to/GHS_UCDB_THEME_GENERAL_CHARACTERISTICS_GLOBE_R2024A.gpkg]

With the optional GHSL Urban Centre Database file, each city is matched to the urban
centre containing its GeoNames point, giving a comparable 2025 population and showing
when a candidate shares an urban centre with a playable metro.
"""

import collections
import csv
import datetime
import hashlib
import json
import math
from pathlib import Path
import sys

ASSIGN_KM = 12
METRO_KM = 40
STATION_TRACK_KM = 5
JOIN_CELL_DEG = 0.002
CORRIDOR_ORDER = ["reaches_metro", "candidate_pair", "isolated", "no_track_match"]
TIER_ORDER = {"operator_published": 0, "tagged_stop": 1, "proximity_only": 2}
ROOT = Path(__file__).resolve().parents[1]
OUT_CSV = ROOT / "data/prototypes/hsr-stop-city-candidates.v1.csv"
OUT_META = ROOT / "data/prototypes/hsr-stop-city-candidates.v1.metadata.json"
METROS = ROOT / "data/prototypes/playable-metros.v1.json"
EVIDENCE = ROOT / "data/prototypes/hsr-service-evidence.v1.jsonl"
EVIDENCE_FIELDS = ["hsr_service", "service_kind", "verified_station", "source_url", "source_type", "resolved_place"]


def km(lat1, lon1, lat2, lon2):
    p1, p2 = math.radians(lat1), math.radians(lat2)
    a = math.sin((p2 - p1) / 2) ** 2 + math.cos(p1) * math.cos(p2) * math.sin(math.radians(lon2 - lon1) / 2) ** 2
    return 12742 * math.asin(math.sqrt(a))


def track_components(encoded):
    import base64
    import zlib
    lines = []
    for deltas in json.loads(zlib.decompress(base64.b64decode(encoded))):
        x = y = 0
        coords = []
        for i in range(0, len(deltas), 2):
            x += deltas[i]
            y += deltas[i + 1]
            coords.append((x / 1e4, y / 1e4))
        lines.append(coords)
    parent = list(range(len(lines)))

    def find(a):
        while parent[a] != a:
            parent[a] = parent[parent[a]]
            a = parent[a]
        return a

    cells = collections.defaultdict(set)
    for index, coords in enumerate(lines):
        for lon, lat in coords:
            cells[(round(lon / JOIN_CELL_DEG), round(lat / JOIN_CELL_DEG))].add(index)
    for (cx, cy), members in cells.items():
        group = set(members)
        for dx in (-1, 0, 1):
            for dy in (-1, 0, 1):
                group |= cells.get((cx + dx, cy + dy), set())
        group = sorted(group)
        for other in group[1:]:
            parent[find(other)] = find(group[0])
    vertices = collections.defaultdict(list)
    for index, coords in enumerate(lines):
        component = find(index)
        for lon, lat in coords:
            vertices[(int(lat // 0.25), int(lon // 0.25))].append((lat, lon, component))
    return vertices


def nearest_component(vertices, lat, lon, limit_km):
    best = None
    gi, gj = int(lat // 0.25), int(lon // 0.25)
    for a in range(gi - 1, gi + 2):
        for b in range(gj - 1, gj + 2):
            for vlat, vlon, component in vertices.get((a, b), ()):
                dist = km(lat, lon, vlat, vlon)
                if dist <= limit_km and (best is None or dist < best[0]):
                    best = (dist, component)
    return None if best is None else best[1]


GHSL_TABLE = "GHSL_UCDB_THEME_GENERAL_CHARACTERISTICS_GLOBE_R2024A"
GHSL_FALLBACK_KM = 5
# Reviewed matches where the GeoNames point and its assigned stations miss the outline.
GHSL_OVERRIDES = {
    10630003: (10143, "Xuzhou: GeoNames point lies 6 km outside the Xuzhou urban centre; its stations are on the edge"),
}


def mollweide(lat, lon, radius=6378137.0):
    phi, lam = math.radians(lat), math.radians(lon)
    theta = phi
    for _ in range(50):
        if abs(abs(phi) - math.pi / 2) < 1e-12:
            break
        delta = -(2 * theta + math.sin(2 * theta) - math.pi * math.sin(phi)) / (2 + 2 * math.cos(2 * theta))
        theta += delta
        if abs(delta) < 1e-12:
            break
    return 2 * math.sqrt(2) / math.pi * radius * lam * math.cos(theta), math.sqrt(2) * radius * math.sin(theta)


def wkb_polygons(blob):
    import struct
    flags = blob[3]
    envelope = {0: 0, 1: 32, 2: 48, 3: 48, 4: 64}[(flags >> 1) & 7]
    data, pos = blob[8 + envelope:], 0

    def read(fmt, little):
        nonlocal pos
        fmt = ("<" if little else ">") + fmt
        value = struct.unpack_from(fmt, data, pos)
        pos += struct.calcsize(fmt)
        return value

    def geometry():
        little = data[pos] == 1
        read("B", little)
        kind = read("I", little)[0] % 1000
        if kind == 3:
            rings = []
            for _ in range(read("I", little)[0]):
                count = read("I", little)[0]
                coords = read("%dd" % (2 * count), little)
                rings.append(list(zip(coords[0::2], coords[1::2])))
            return [rings]
        if kind == 6:
            polygons = []
            for _ in range(read("I", little)[0]):
                polygons += geometry()
            return polygons
        raise ValueError(f"unsupported WKB type {kind}")

    return geometry()


def inside(x, y, polygons):
    for rings in polygons:
        hit = False
        for ring in rings:
            for (x1, y1), (x2, y2) in zip(ring, ring[1:] + ring[:1]):
                if (y1 > y) != (y2 > y) and x < (x2 - x1) * (y - y1) / (y2 - y1) + x1:
                    hit = not hit
        if hit:
            return True
    return False


def load_ghsl(path):
    import sqlite3
    db = sqlite3.connect(f"file:{path}?mode=ro", uri=True)
    centres = []
    for uc_id, name, country, pop, blob in db.execute(
            f"SELECT ID_UC_G0, GC_UCN_MAI_2025, GC_CNT_UNN_2025, GC_POP_TOT_2025, geom FROM {GHSL_TABLE}"):
        polygons = wkb_polygons(blob)
        xs = [x for rings in polygons for x, _ in rings[0]]
        ys = [y for rings in polygons for _, y in rings[0]]
        centres.append({"id": uc_id, "name": name, "country": country, "population": round(pop),
                        "bbox": (min(xs), min(ys), max(xs), max(ys)), "polygons": polygons})
    return centres


def match_ghsl(centres, lat, lon, stations=()):
    for label, (plat, plon) in [("inside", (lat, lon))] + [("station_inside", p) for p in stations]:
        px, py = mollweide(plat, plon)
        for centre in centres:
            x0, y0, x1, y1 = centre["bbox"]
            if x0 <= px <= x1 and y0 <= py <= y1 and inside(px, py, centre["polygons"]):
                return centre, label
    x, y = mollweide(lat, lon)
    # Mollweide metres: adequate for a few-kilometre tolerance away from the poles.
    reach = GHSL_FALLBACK_KM * 1000
    best = None
    for centre in centres:
        x0, y0, x1, y1 = centre["bbox"]
        if x0 - reach <= x <= x1 + reach and y0 - reach <= y <= y1 + reach:
            gap = min(math.hypot(x - px, y - py) for rings in centre["polygons"] for px, py in rings[0])
            if gap <= reach and (best is None or gap < best[0]):
                best = (gap, centre)
    return (best[1], "near_edge") if best else (None, "none")


def sha256(path):
    return hashlib.sha256(Path(path).read_bytes()).hexdigest()


def main(overlay_path, geonames_path, ghsl_path=None):
    html = Path(overlay_path).read_text(encoding="utf-8")
    marker = 'id="wwhsrt-network-data">'
    start = html.index(marker) + len(marker)
    data = json.loads(html[start:html.index("</script>", start)])
    unlinked = set(data["stationMetadata"]["stationsWithoutMappedLink"])

    cities = []
    for line in open(geonames_path, encoding="utf-8"):
        f = line.rstrip("\n").split("\t")
        if f[6] == "P":
            cities.append((int(f[0]), f[1], float(f[4]), float(f[5]), f[8], int(f[14] or 0)))
    grid = collections.defaultdict(list)
    for c in cities:
        grid[(int(c[2] // 0.5), int(c[3] // 0.5))].append(c)

    def nearby(lat, lon):
        gi, gj = int(lat // 0.5), int(lon // 0.5)
        for a in range(gi - 1, gi + 2):
            for b in range(gj - 1, gj + 2):
                for c in grid.get((a, b), ()):
                    dist = km(lat, lon, c[2], c[3])
                    if dist <= ASSIGN_KM:
                        yield c, dist

    vertices = track_components(data["existing"])
    agg, unassigned = {}, 0
    for sid, name, lon, lat, tagged, operator in data["stations"]:
        tier = "operator_published" if operator else "tagged_stop" if tagged else "proximity_only"
        candidates = list(nearby(lat, lon))
        if not candidates:
            unassigned += 1
            continue
        city, _ = max(candidates, key=lambda t: (t[0][5], -t[1], -t[0][0]))
        entry = agg.setdefault(city[0], {"city": city, "names": set(), "tiers": collections.Counter(), "linked": 0,
                                         "components": set(), "points": []})
        entry["points"].append((lat, lon))
        component = nearest_component(vertices, lat, lon, STATION_TRACK_KM)
        if component is not None:
            entry["components"].add(component)
        entry["names"].add(name)
        entry["tiers"][tier] += 1
        entry["linked"] += sid not in unlinked

    metros = json.loads(METROS.read_text(encoding="utf-8"))
    component_metros = collections.defaultdict(set)
    for m in metros:
        gi, gj = int(m["lat"] // 0.25), int(m["lng"] // 0.25)
        reach = int(METRO_KM / 25) + 2
        for a in range(gi - reach, gi + reach + 1):
            for b in range(gj - reach, gj + reach + 1):
                for vlat, vlon, component in vertices.get((a, b), ()):
                    if km(m["lat"], m["lng"], vlat, vlon) <= METRO_KM:
                        component_metros[component].add(m["id"])
    rows, covered = [], 0
    for gid, entry in agg.items():
        c = entry["city"]
        dist, metro = min((km(c[2], c[3], m["lat"], m["lng"]), m["id"]) for m in metros)
        if dist <= METRO_KM:
            covered += 1
            continue
        rows.append({
            "geonameid": gid, "city": c[1], "country_code": c[4], "population": c[5],
            "lat": c[2], "lng": c[3],
            "evidence": min(entry["tiers"], key=TIER_ORDER.get),
            "stations": sum(entry["tiers"].values()),
            "stations_with_mapped_link": entry["linked"],
            "tagged_or_published": entry["tiers"]["tagged_stop"] + entry["tiers"]["operator_published"],
            "sample_stations": "; ".join(sorted(entry["names"])[:3]),
            "nearest_metro": metro, "nearest_metro_km": round(dist),
            "_components": entry["components"],
            "_points": entry["points"],
        })
    component_cities = collections.defaultdict(set)
    for row in rows:
        for component in row["_components"]:
            component_cities[component].add(row["geonameid"])
    for row in rows:
        comps = row.pop("_components")
        reached = sorted(set().union(*(component_metros[c] for c in comps))) if comps else []
        others = set().union(*(component_cities[c] for c in comps)) - {row["geonameid"]} if comps else set()
        row["corridor_class"] = ("no_track_match" if not comps else "reaches_metro" if reached
                                 else "candidate_pair" if others else "isolated")
        row["component_metros"] = "; ".join(reached[:4]) + ("; …" if len(reached) > 4 else "")
        row["component_other_cities"] = len(others)
    evidence = {}
    if EVIDENCE.exists():
        for line in EVIDENCE.read_text(encoding="utf-8").splitlines():
            if line.strip():
                item = json.loads(line)
                evidence[int(item["geonameid"])] = item
    for row in rows:
        item = evidence.get(row["geonameid"], {})
        row["hsr_service"] = item.get("hsr_service", "unchecked")
        row["service_kind"] = item.get("service_kind", "")
        row["verified_station"] = item.get("station", "")
        urls = item.get("source_url", [])
        row["source_url"] = " ".join(urls) if isinstance(urls, list) else urls
        row["needs_spot_check"] = item.get("needs_spot_check", "")
        row["source_type"] = item.get("source_type", "")
        row["resolved_place"] = item.get("resolved_place", "")
    if ghsl_path:
        centres = load_ghsl(ghsl_path)
        metro_centre = {}
        for m in metros:
            centre, _ = match_ghsl(centres, m["lat"], m["lng"])
            if centre:
                metro_centre.setdefault(centre["id"], []).append(m["id"])
        for row in rows:
            if row["geonameid"] in GHSL_OVERRIDES:
                centre, how = next(c for c in centres if c["id"] == GHSL_OVERRIDES[row["geonameid"]][0]), "override"
            else:
                centre, how = match_ghsl(centres, row["lat"], row["lng"], row["_points"])
            row["ghsl_id"] = centre["id"] if centre else ""
            row["ghsl_name"] = centre["name"] if centre else ""
            row["ghsl_pop_2025"] = centre["population"] if centre else ""
            row["ghsl_match"] = how
            row["ghsl_shared_metro"] = "; ".join(metro_centre.get(centre["id"], [])) if centre else ""
        shared = collections.Counter(r["ghsl_id"] for r in rows if r["ghsl_id"] != "")
        for row in rows:
            row["ghsl_shared_candidates"] = shared[row["ghsl_id"]] - 1 if row["ghsl_id"] != "" else ""
    for row in rows:
        row.pop("_points", None)
    rows.sort(key=lambda r: (-r["population"], r["geonameid"]))
    with OUT_CSV.open("w", newline="", encoding="utf-8") as fh:
        writer = csv.DictWriter(fh, fieldnames=["rank", *rows[0]], lineterminator="\n")
        writer.writeheader()
        for rank, row in enumerate(rows, 1):
            writer.writerow({"rank": rank, **row})

    strong = [r for r in rows if r["evidence"] != "proximity_only"]
    thresholds = {f">={t}": {"all": sum(r["population"] >= t for r in rows), "tagged_or_published": sum(r["population"] >= t for r in strong)}
                  for t in (1_000_000, 500_000, 250_000, 100_000)}
    meta = {
        "schema_version": 1,
        "generated_on": datetime.date.today().isoformat(),
        "status": "Screening candidates for the planned city catalog; not qualified starting infrastructure or service stops",
        "inputs": {
            "station_layer": {"description": "Launch network preview station candidates (OpenStreetMap via QLever)",
                              "source_hashes": data["stationMetadata"]["sourceHashes"], "station_count": len(data["stations"])},
            "geonames": {"file": Path(geonames_path).name, "sha256": sha256(geonames_path),
                         "source": "https://download.geonames.org/export/dump/cities15000.zip",
                         "license": "CC BY 4.0", "attribution": "GeoNames (https://www.geonames.org/)"},
            "playable_metros": {"file": "playable-metros.v1.json", "sha256": sha256(METROS)},
            "ghsl_ucdb": ({"file": Path(ghsl_path).name, "sha256": sha256(ghsl_path), "release": "R2024A V1.2",
                           "population_field": "GC_POP_TOT_2025", "edge_tolerance_km": GHSL_FALLBACK_KM,
                           "match_order": ["GeoNames point inside", "assigned station inside", "within edge_tolerance_km of the outline"],
                           "overrides": {str(k): {"ucdb_id": v[0], "reason": v[1]} for k, v in GHSL_OVERRIDES.items()},
                           "source": "https://jeodpp.jrc.ec.europa.eu/ftp/jrc-opendata/GHSL/GHS_UCDB_GLOBE_R2024A/",
                           "license": "CC BY 4.0", "attribution": "European Commission, GHSL Urban Centre Database R2024A"}
                          if ghsl_path else None),
            "service_evidence": ({"file": EVIDENCE.name, "sha256": sha256(EVIDENCE), "rows": len(evidence)}
                                 if EVIDENCE.exists() else None),
        },
        "method": {"assign_km": ASSIGN_KM, "metro_coverage_km": METRO_KM, "station_track_km": STATION_TRACK_KM,
                   "track_join_cell_deg": JOIN_CELL_DEG,
                   "corridor_class_use": "Informational only; the service check decides inclusion",
                   "corridor_classes": {"reaches_metro": "Track component comes within metro_coverage_km of a playable metro",
                                        "candidate_pair": "No metro, but another candidate city has a station on the same component",
                                        "isolated": "Only this city has a station on the component",
                                        "no_track_match": "No station within station_track_km of preview track"},
                   "assignment": "Most populous GeoNames populated place within assign_km of each station",
                   "evidence_order": list(TIER_ORDER)},
        "summary": {"stations_unassigned": unassigned, "cities_with_station": len(agg),
                    "covered_by_playable_metro": covered, "candidate_cities": len(rows),
                    "with_tagged_or_published_stop": len(strong), "population_thresholds": thresholds,
                    "service_yes_1m_by_country": dict(collections.Counter(
                        r["country_code"] for r in rows if r["hsr_service"] == "yes" and r["population"] >= 1_000_000).most_common()),
                    "service_checked": {value: sum(r["hsr_service"] == value for r in rows)
                                        for value in ("yes", "no", "uncertain", "unchecked")},
                    "corridor_classes": {cls: {"all": sum(r["corridor_class"] == cls for r in rows),
                                               ">=1000000": sum(r["corridor_class"] == cls and r["population"] >= 1_000_000 for r in rows)}
                                         for cls in CORRIDOR_ORDER},
                    "candidates_100k_by_country": dict(collections.Counter(r["country_code"] for r in rows if r["population"] >= 100_000).most_common())},
        "caveats": [
            "GeoNames populations are not a common metropolitan definition; many Chinese entries are prefecture totals.",
            "Evidence tiers reflect OpenStreetMap tagging practice; Chinese and Japanese stations are rarely tagged as high-speed stops.",
            "The station layer includes upgraded conventional lines and stops within 400 m of tagged track; a station is not proof of high-speed service.",
            "A fixed metro-coverage radius mishandles broad metro regions and large cities just outside it.",
            "Track components come from a speed-tag extract; terminal approaches below the tag threshold split real networks, so metro reach (not track continuity) bridges those gaps.",
        ],
    }
    OUT_META.write_text(json.dumps(meta, indent=2, ensure_ascii=False) + "\n", encoding="utf-8")
    print(json.dumps(meta["summary"], indent=2))


if __name__ == "__main__":
    main(*sys.argv[1:4])
