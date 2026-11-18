"""Geometry shared by the public-network compiler and German infrastructure survey."""

import hashlib
import json
import math

TRUNK = "public_frankfurt_am_main__koln"
FRANKFURT = "public_frankfurt_am_main__stop_node_2453992382"
COLOGNE = "public_koln__stop_node_2453992382"


def path_hash(points):
    return hashlib.sha256(
        json.dumps(points, separators=(",", ":")).encode()
    ).hexdigest()


def montabaur_paths(points, station):
    # The station is between two vertices of the simplified Cologne–Frankfurt line.
    scale = math.cos(math.radians(station[1]))

    def projection(i):
        a, b = points[i : i + 2]
        dx, dy = (b[0] - a[0]) * scale, b[1] - a[1]
        x, y = (station[0] - a[0]) * scale, station[1] - a[1]
        t = max(0, min(1, (x * dx + y * dy) / (dx * dx + dy * dy))) if dx or dy else 0
        return math.hypot(x - t * dx, y - t * dy), i

    gap, i = min(projection(i) for i in range(len(points) - 1))
    assert gap * 111 < 0.4, "Montabaur station is off the Cologne–Frankfurt line"
    from_frankfurt = list(points[: i + 1])
    from_cologne = list(reversed(points[i + 1 :]))
    for leg in (from_frankfurt, from_cologne):
        if leg[-1] != station:
            leg.append(station)
    return {
        FRANKFURT: from_frankfurt,
        COLOGNE: from_cologne,
        TRUNK: from_frankfurt + list(reversed(from_cologne[:-1])),
    }


def correct_montabaur(rows, cities, km):
    by_id = {row["id"]: row for row in rows}
    trunk = by_id[TRUNK]
    station = next(city for city in cities if city["id"] == by_id[FRANKFURT]["to_city"])
    main = trunk["alignments"][0]
    paths = montabaur_paths(
        [[p["lon"], p["lat"]] for p in main["path"]],
        [station["station_lon"], station["station_lat"]],
    )
    lengths = {
        key: sum(km(a, b) for a, b in zip(path, path[1:]))
        for key, path in paths.items()
    }
    trunk["shared_sections"] = [FRANKFURT, COLOGNE]
    for key, path in paths.items():
        align = by_id[key]["alignments"][0]
        align["path"] = [dict(lon=x, lat=y) for x, y in path]
        if key != TRUNK:
            fraction = lengths[key] / lengths[TRUNK]
            # Keep the accepted Frankfurt leg's timetable and allocate the
            # balance to Cologne. Every existing fleet still covers its plan.
            if key == COLOGNE:
                align["minutes"] = (
                    main["minutes"] - by_id[FRANKFURT]["alignments"][0]["minutes"]
                )
            for field in ("km", "surface_km"):
                align[field] = round(main[field] * fraction)


def apply_majority_rule(rows, patterns, profiles, superseded, km, rates, hamm):
    from occitanie import works

    retired = {}
    for co in list(rows):
        id = co["id"]
        if id not in profiles:
            continue
        alt = co["alignments"][0]
        points = [[p["lon"], p["lat"]] for p in alt["path"]]
        assert path_hash(points) == profiles[id]["path_sha256"], (
            "Stale track survey: " + id
        )
        meters = profiles[id]["meters"]
        if id == hamm["id"]:
            points = hamm["path"]
            meters = hamm["meters"]
            distance = round(sum(meters.values()) / 1000)
            alt.update(
                path=[dict(lon=x, lat=y) for x, y in points],
                km=distance,
                surface_km=distance,
                minutes=160,
                connector_km=round(meters["conventional"] / 1000),
            )
            co["via"] = hamm["via"]
        fast = meters.get("dedicated", 0) + meters.get("upgraded", 0)
        total = sum(meters.values())
        drawn = sum(km(a, b) for a, b in zip(points, points[1:]))
        if total < 0.99 * drawn * 1000:
            continue  # A German-only survey cannot decide a cross-border line.
        if fast * 2 > total:
            co["infrastructure_kind"] = "hsr"
            continue
        co["retired"] = True
        retired[id] = None
        if fast:
            distance = max(2, round(total / 1000))
            retained = min(distance - 1, round(fast / 1000))
            id_new = id.replace("public_", "upgrade_de_", 1)
            co_new = works(
                id_new,
                co["from_city"],
                co["to_city"],
                alt["path"],
                distance,
                math.ceil(distance * 60 / 250) + 8,
                min(72, max(12, math.ceil((distance - retained) / 5))),
                "central_europe",
                rates["central_europe"],
                retained,
            )
            co_new["construction_kind"] = "upgrade"
            co_new["parallel_to"] = [id]
            rows.append(co_new)
            retired[id] = id_new
    # Keep the public approaches of the named international services available
    # to their through trains. Removed lines never appear as standalone offers.
    by_id = {c["id"]: c for c in rows}
    for r in list(patterns):
        affected = [id for id in r["sections"] if id in retired]
        if not affected:
            continue
        if r["id"] in ("network_brussels__dusseldorf", "network_dusseldorf__randstad"):
            for id in affected:
                by_id[id]["through_only"] = True
            continue
        if r["id"] not in superseded:
            superseded.append(r["id"])
        if all(retired[id] for id in affected):
            patterns.append(
                {
                    **r,
                    "id": r["id"] + "_hsr",
                    "sections": [retired.get(id, id) for id in r["sections"]],
                }
            )
