"""Draft the allocation of the ~156 catalog places beyond the playable metros.

The output is a reviewable draft, not an approved catalog. Groups and quotas follow the
allocation proposed on 2026-11-09:
- launch-shortlist endpoints that are not playable metros;
- non-China cities of 1M+ with checked high-speed service;
- hand-picked smaller stops outside China (service not yet checked);
- China: provincial capitals, then junctions and line ends, then a provisional
  population fill with a spacing rule.

Usage (run with python3 -I):
  python3 -I tools/draft-catalog-selection.py \
      "$WWHSRT_MAP_PREVIEW_DIR/hsr-network-overlay.html" /path/to/cities15000.txt
"""

import collections
import csv
import datetime
import hashlib
import json
import math
from pathlib import Path
import sys

ROOT = Path(__file__).resolve().parents[1]
PROTO = ROOT / "data/prototypes"
OUT = PROTO / "catalog-city-selection.v1.json"
TOTAL_TARGET = 250
METRO_KM = 40
FILL_SPACING_KM = 50
QUOTAS = {"launch_endpoints": 60, "checked_stops_outside_china": 32, "smaller_stops_outside_china": 14, "china": 50}

SMALLER_STOPS = [
    2925533,
    2910831,
    2861650,
    2998324,
    2995469,
    3181928,
    3176959,
    3172394,
    3104324,
    2509954,
    2510911,
    1854383,
    1855431,
    1860243,
]
CHINA_CAPITALS = [
    2037013,
    2038180,
    1795270,
    1793511,
    2036892,
    1808722,
    1810821,
    1800163,
    1799869,
    1804651,
    1809461,
    1804430,
    1788852,
    1786657,
    1529102,
    1809078,
]
# Checked against line maps on 2026-11-09 (china-junction-roles.v1.jsonl). Baoji, Liuzhou,
# Sanya and Ningbo proved to be through stops and were replaced by Guilin, Huizhou,
# Huzhou and Fuyang by user decision.
CHINA_JUNCTIONS = [
    10630003,
    1796134,
    1808370,
    1805528,
    1787858,
    1791388,
    1814087,
    1787093,
    2037799,
    1795940,
    1809498,
    1806776,
    1806535,
    1810845,
]
# Rows the service check flagged as misplaced, inflated, or overlapping another city.
CHINA_FILL_EXCLUDE = {
    1790842: "GeoNames population (7.2M) far exceeds the city",
    1798425: "Point is Pujiang, Zhejiang; population is Puyang, Henan",
    1793036: "Village name standing in for Tengzhou",
    1808316: "Heshan District stands in for Yiyang",
    1797658: "Overlaps Quanzhou",
    1806602: "Station is Yuyao North in neighbouring Yuyao",
}


# Reviewed decisions to keep a city that shares a playable metro's GHSL urban centre.
KEEP_DESPITE_SHARED_CENTRE = {
    1857910: "Kyoto: kept as its own catalog city by user decision on 2026-11-09",
}


def km(lat1, lon1, lat2, lon2):
    p1, p2 = math.radians(lat1), math.radians(lat2)
    a = math.sin((p2 - p1) / 2) ** 2 + math.cos(p1) * math.cos(p2) * math.sin(math.radians(lon2 - lon1) / 2) ** 2
    return 12742 * math.asin(math.sqrt(a))


def sha256(path):
    return hashlib.sha256(Path(path).read_bytes()).hexdigest()


def main(overlay_path, geonames_path):
    html = Path(overlay_path).read_text(encoding="utf-8")
    marker = 'id="wwhsrt-network-data">'
    start = html.index(marker) + len(marker)
    overlay = json.loads(html[start : html.index("</script>", start)])
    metros = json.loads((PROTO / "playable-metros.v1.json").read_text(encoding="utf-8"))
    candidates = {
        int(r["geonameid"]): r
        for r in csv.DictReader(open(PROTO / "hsr-stop-city-candidates.v1.csv", encoding="utf-8"))
    }
    roles = {
        r["geonameid"]: r
        for r in map(json.loads, (PROTO / "china-junction-roles.v1.jsonl").read_text(encoding="utf-8").splitlines())
    }
    evidence = {
        e["geonameid"]: e
        for e in map(json.loads, (PROTO / "hsr-service-evidence.v1.jsonl").read_text(encoding="utf-8").splitlines())
    }
    checked_outside_china = {
        k: e
        for k, e in evidence.items()
        if isinstance(k, int) and e["country_code"] != "CN" and e.get("checked_for") in (None, "candidate_1m")
    }
    geonames = {}
    for line in open(geonames_path, encoding="utf-8"):
        f = line.rstrip("\n").split("\t")
        geonames[int(f[0])] = {
            "name": f[1],
            "lat": float(f[4]),
            "lng": float(f[5]),
            "country_code": f[8],
            "population": int(f[14] or 0),
        }

    def entry(gid, **extra):
        if gid in candidates:
            c = candidates[gid]
            base = {
                "geonameid": gid,
                "name": c["city"],
                "country_code": c["country_code"],
                "lat": float(c["lat"]),
                "lng": float(c["lng"]),
                "population": int(c["population"]),
            }
            if c.get("ghsl_id"):
                base.update(
                    {
                        "ghsl_id": int(c["ghsl_id"]),
                        "ghsl_name": c["ghsl_name"],
                        "ghsl_pop_2025": int(c["ghsl_pop_2025"]),
                        "ghsl_shared_metro": c["ghsl_shared_metro"] or None,
                    }
                )
        else:
            g = geonames[gid]
            base = {
                "geonameid": gid,
                "name": g["name"],
                "country_code": g["country_code"],
                "lat": g["lat"],
                "lng": g["lng"],
                "population": g["population"],
            }
        e = evidence.get(gid)
        if e:
            base.update({"hsr_service": e["hsr_service"], "service_kind": e["service_kind"], "station": e["station"]})
        base.update(extra)
        return base

    def near_metro(lat, lng):
        return min(km(lat, lng, m["lat"], m["lng"]) for m in metros) <= METRO_KM

    groups = {}
    checked = [entry(gid) for gid, e in sorted(checked_outside_china.items()) if e["hsr_service"] == "yes"]
    groups["checked_stops_outside_china"] = checked
    checked_ids = {c["geonameid"] for c in checked}

    endpoints, overlap = [], []
    names = sorted({n for pair in overlay["pairs"] for n in (pair[1], pair[2])})
    for name in names:
        lng, lat, source = overlay["points"][name]
        if near_metro(lat, lng):
            continue
        gid = int(source.split()[1]) if source.startswith("GeoNames ") else None
        if gid in candidates or gid in geonames:
            item = entry(gid, shortlist_name=name)
        elif gid is not None:
            item = {
                "geonameid": gid,
                "name": name,
                "country_code": None,
                "lat": lat,
                "lng": lng,
                "population": None,
                "shortlist_name": name,
                "note": "GeoNames ID is not in cities15000; coordinates from the shortlist",
            }
            if gid in evidence:
                e = evidence[gid]
                item.update(
                    {
                        "country_code": e["country_code"],
                        "hsr_service": e["hsr_service"],
                        "service_kind": e["service_kind"],
                        "station": e["station"],
                    }
                )
        else:
            item = {
                "geonameid": None,
                "source_ref": source,
                "name": name,
                "country_code": "JP",
                "lat": lat,
                "lng": lng,
                "population": None,
                "shortlist_name": name,
                "note": "Station point; its town is below the GeoNames cities15000 floor",
            }
            osm_key = "osm:" + source.split()[-1]
            if osm_key in evidence:
                e = evidence[osm_key]
                item.update(
                    {"hsr_service": e["hsr_service"], "service_kind": e["service_kind"], "station": e["station"]}
                )
        if item["geonameid"] in checked_ids or any(km(lat, lng, c["lat"], c["lng"]) <= 15 for c in checked):
            overlap.append(name)
            continue
        endpoints.append(item)
    groups["launch_endpoints"] = endpoints

    groups["smaller_stops_outside_china"] = [
        entry(gid, status="draft pick; service not yet checked") for gid in SMALLER_STOPS
    ]

    china = [entry(gid, basis="provincial capital") for gid in CHINA_CAPITALS]
    china += [
        entry(
            gid,
            basis="junction or line end (checked against line maps)",
            network_role=roles[gid]["role"],
            directions=roles[gid]["directions"],
        )
        for gid in CHINA_JUNCTIONS
    ]
    chosen = list(china)
    fill_quota = QUOTAS["china"] - len(china)

    def ghsl_pop(gid):
        value = candidates[gid].get("ghsl_pop_2025", "")
        return int(value) if value else 0

    pool = sorted(
        (
            e
            for e in evidence.values()
            if e["country_code"] == "CN"
            and e["hsr_service"] == "yes"
            and e["geonameid"] not in CHINA_FILL_EXCLUDE
            and e["geonameid"] not in {c["geonameid"] for c in chosen}
        ),
        key=lambda e: (-ghsl_pop(e["geonameid"]), e["geonameid"]),
    )
    skipped = []
    for e in pool:
        if fill_quota == 0:
            break
        item = entry(e["geonameid"], basis="population fill ranked by GHSL urban-centre population (2025)")
        reason = None
        if "ghsl_id" not in item:
            reason = "no GHSL urban centre match"
        elif item["ghsl_shared_metro"]:
            reason = f"inside the urban centre of metro {item['ghsl_shared_metro']}"
        elif any(c.get("ghsl_id") == item["ghsl_id"] for c in chosen):
            reason = "shares an urban centre with " + next(
                c["name"] for c in chosen if c.get("ghsl_id") == item["ghsl_id"]
            )
        else:
            close = [
                c["name"] for c in chosen + metros if km(item["lat"], item["lng"], c["lat"], c["lng"]) < FILL_SPACING_KM
            ]
            if close:
                reason = f"within {FILL_SPACING_KM} km of {close[0]}"
        if reason:
            skipped.append({"name": item["name"], "geonameid": item["geonameid"], "reason": reason})
            continue
        china.append(item)
        chosen.append(item)
        fill_quota -= 1
    groups["china"] = china

    counts = {k: len(v) for k, v in groups.items()}
    selected = {c["geonameid"] for v in groups.values() for c in v}
    china_unselected = sorted(
        (
            e["city"]
            for e in evidence.values()
            if e["country_code"] == "CN" and e["hsr_service"] == "yes" and e["geonameid"] not in selected
        )
    )
    manifest = {
        "schema_version": 1,
        "status": "Draft allocation for review; not an approved catalog",
        "generated_on": datetime.date.today().isoformat(),
        "total_target": TOTAL_TARGET,
        "playable_metros": len(metros),
        "places": TOTAL_TARGET - len(metros),
        "quotas": QUOTAS,
        "counts": counts,
        "service_by_group": {
            g: dict(collections.Counter(c.get("hsr_service", "unchecked") for c in v)) for g, v in groups.items()
        },
        "selected_total": sum(counts.values()),
        "inputs": {
            "candidate_list": {
                "file": "hsr-stop-city-candidates.v1.csv",
                "sha256": sha256(PROTO / "hsr-stop-city-candidates.v1.csv"),
            },
            "service_evidence": {
                "file": "hsr-service-evidence.v1.jsonl",
                "sha256": sha256(PROTO / "hsr-service-evidence.v1.jsonl"),
            },
            "junction_roles": {
                "file": "china-junction-roles.v1.jsonl",
                "sha256": sha256(PROTO / "china-junction-roles.v1.jsonl"),
            },
            "playable_metros": {"file": "playable-metros.v1.json", "sha256": sha256(PROTO / "playable-metros.v1.json")},
            "geonames": {"file": Path(geonames_path).name, "sha256": sha256(geonames_path), "license": "CC BY 4.0"},
            "ghsl_ucdb": "Joined through hsr-stop-city-candidates.v1.csv (GHSL UCDB R2024A V1.2, CC BY 4.0)",
        },
        "rules": {
            "launch_endpoints": f"Endpoints of the 70 shortlisted pairs more than {METRO_KM} km from a playable metro, "
            f"minus those already in checked_stops_outside_china",
            "checked_stops_outside_china": "Every non-China city of 1M+ with checked service on 2025-01-01",
            "smaller_stops_outside_china": "Hand-picked intermediate stops on existing high-speed lines that the 1M cut misses; "
            "service still to be checked",
            "china": (
                f"Provincial capitals with service, then junctions and line ends, "
                f"then the remaining cities with service ranked by "
                f"GHSL urban-centre population, skipping any inside a metro's urban centre, "
                f"sharing an urban centre with a "
                f"city already chosen, or within {FILL_SPACING_KM} km of a playable metro or chosen city"
            ),
        },
        "endpoint_overlap_with_checked_stops": overlap,
        "china_fill_excluded": [{"geonameid": k, "reason": v} for k, v in sorted(CHINA_FILL_EXCLUDE.items())],
        "china_fill_skipped": skipped,
        "selected_sharing_metro_urban_centre": [
            {
                "name": c["name"],
                "group": g,
                "metro": c["ghsl_shared_metro"],
                "decision": KEEP_DESPITE_SHARED_CENTRE.get(c["geonameid"]),
            }
            for g, v in groups.items()
            for c in v
            if c.get("ghsl_shared_metro")
        ],
        "china_with_service_not_selected": china_unselected,
        "open_items": [
            "Review selected cities that share a metro's GHSL urban centre.",
            "Unselected cities with service remain pass-through stations on starting corridors.",
        ],
        "groups": groups,
    }
    OUT.write_text(json.dumps(manifest, indent=2, ensure_ascii=False) + "\n", encoding="utf-8")
    print(
        json.dumps(
            {
                "counts": counts,
                "selected_total": manifest["selected_total"],
                "overlap": overlap,
                "china_skipped": [f"{s['name']}: {s['reason']}" for s in skipped],
                "sharing_metro": manifest["selected_sharing_metro_urban_centre"],
            },
            ensure_ascii=False,
            indent=1,
        )
    )


if __name__ == "__main__":
    main(*sys.argv[1:3])
