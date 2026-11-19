"""Join the audited operating lines to their existing stations, track and contracts."""

from copy import deepcopy

from shapely.geometry import LineString, Point
from shapely.ops import substring

from pinned_services import add_hubs, add_sections, add_patterns


def add_operating_hsr(source, cities, market, stations, rows, patterns, superseded, km, junction_splits=()):
    hubs = source["hubs"]
    old = {c["id"]: c for c in rows}
    add_hubs(source, cities, market, stations)
    city = {c["id"]: c for c in cities}
    station_index = {(s["lon"], s["lat"]): s for s in stations}

    def coords(points):
        return [(p["lon"], p["lat"]) for p in points]

    def points(path):
        return [dict(lon=x, lat=y) for x, y in path]

    def mark_stations(row):
        # Markets with several termini keep their demand and primary city pin.
        # Publish the actual platform at each end of this specific alignment.
        p = row["alignments"][0]["path"]
        ends = [p[0], p[-1]]
        keys = [row["from_city"], row["to_city"]]
        names, ids = [], []
        for end, key in zip(ends, keys):
            loc = (end["lon"], end["lat"])
            if loc == (city[key]["station_lon"], city[key]["station_lat"]):
                names.append(city[key]["station_name"])
                ids.append("")
            else:
                st = station_index[loc]
                names.append(hubs[key]["station"] if key in hubs and hubs[key]["station_id"] == st["id"] else st["name"])
                ids.append(st["id"])
        if any(ids):
            row["terminal_ids"] = ids
        row["station_names"] = names

    def distance(path):
        return sum(km(a, b) for a, b in zip(path, path[1:]))

    # Move the two San'yō approaches with Fukuoka's existing market.
    lead = points(source["hakata_approach"][::-1])
    for row in rows:
        if row["from_city"] == "fukuoka":
            alt = row["alignments"][0]
            assert alt["path"][0] == lead[-1]
            alt["path"] = lead[:-1] + alt["path"]

    # Old saves may have a tightly timed service on the retired station pair.
    # Give the corrected Hauptbahnhof alignments their own service identities.
    restored = {
        "public_munich__nuremberg": "public_munich_hbf__nuremberg_hbf",
        "public_berlin__hannover": "public_berlin_hbf__hannover_hbf",
        "public_munich__stop_node_1058668074": "public_munich_hbf__augsburg_hbf",
    }
    added = []
    add_sections(source, added)
    for row in added:
        mark_stations(row)
        if row["id"] in old:
            assert old[row["id"]].get("retired")
            row["replaces_corridor"] = row["id"]
            row["id"] = restored[row["id"]]
        rows.append(row)
    pairs = {frozenset((c["from_city"], c["to_city"])): c for c in rows if c["existing"]}

    def oriented(row, a):
        path = coords(row["alignments"][0]["path"])
        return path if row["from_city"] == a else path[::-1]

    def repair_express(a, junction, b):
        first, last = pairs[frozenset((a, junction))], pairs[frozenset((junction, b))]
        p, q = oriented(first, a), oriented(last, junction)
        assert p[-1] == q[0]
        row = deepcopy(pairs[frozenset((a, b))])
        row["replaces_corridor"] = row["id"]
        row["id"] = restored[row["id"]]
        rows.append(row)
        alt = row["alignments"][0]
        for key in ("km", "surface_km", "minutes"):
            alt[key] = first["alignments"][0][key] + last["alignments"][0][key]
        alt["path"] = points(p + q[1:])
        alt.pop("connector_km", None)
        row.pop("retired")
        row.update(infrastructure_kind="hsr", via=[hubs[junction]["station"]], shared_sections=[first["id"], last["id"]])
        mark_stations(row)

    repair_express("munich", "ingolstadt", "nuremberg")
    repair_express("berlin", "wolfsburg", "hannover")
    projects = {id.replace("public_", "upgrade_de_", 1): new for id, new in restored.items()}
    for id in projects:
        old[id]["retired"] = True  # Existing owners can finish and operate their contracts.
    all_rows = {c["id"]: c for c in rows}
    replacements = restored | projects

    def continuous(route, section_ids):
        previous = None
        for start, id in zip(route["cities"], section_ids):
            track = oriented(all_rows[id], start)
            if previous is not None and previous != track[0]:
                return False
            previous = track[-1]
        return True

    offered = {(tuple(r["sections"]), tuple(r["stops"])) for r in patterns if r["id"] not in superseded}
    for r in list(patterns):
        if replacements.keys().isdisjoint(r["sections"]):
            continue
        if r["id"] not in superseded:
            superseded.append(r["id"])
        sections = tuple(replacements.get(id, id) for id in r["sections"])
        key = (sections, tuple(r["stops"]))
        if key in offered or any(all_rows[id].get("retired") for id in sections):
            continue
        if continuous(r, sections):
            patterns.append({**r, "id": r["id"] + "_hbf", "sections": list(sections)})
            offered.add(key)

    def divide(id, junction, through_only=False):
        # Keep the released direct service and its timetable. It consumes both
        # physical sections, as do stopping and branching services.
        row = next(c for c in rows if c["id"] == id)
        a, b = row["from_city"], row["to_city"]
        main = row["alignments"][0]
        path = LineString(coords(main["path"]))
        pin = ((city[junction]["station_lon"], city[junction]["station_lat"]) if through_only
               else (hubs[junction]["lon"], hubs[junction]["lat"]))
        cut = path.project(Point(pin))
        assert path.distance(Point(pin)) < .003 and 0 < cut < path.length
        left = [*list(substring(path, 0, cut).coords)[:-1], pin]
        right = [pin, *list(substring(path, cut, path.length).coords)[1:]]
        legs = [(a, junction, left), (junction, b, right)]
        total = {key: main[key] for key in ("km", "minutes")}
        known = [pairs.get(frozenset((start, end))) for start, end, _ in legs]
        if any(known):
            i = 0 if known[0] else 1
            assert known[1-i] is None
            quantities = {k: known[i]["alignments"][0][k] for k in total}
            left_qty = quantities if i == 0 else {k: total[k] - quantities[k] for k in total}
        else:
            share = distance(left) / (distance(left) + distance(right))
            left_qty = {k: round(v * share) for k, v in total.items()}
        resources, joined = [], []
        for i, (start, end, track) in enumerate(legs):
            part = known[i]
            if part is None:
                q = left_qty if i == 0 else {k: total[k] - left_qty[k] for k in total}
                assert min(q.values()) > 0
                alt = dict(id="public_rail", months=0, path=points(track), surface_km=q["km"], hill_km=0, tunnel_km=0, viaduct_km=0, **q)
                part = dict(id=f"public_{start}__{end}", from_city=start, to_city=end, existing=True,
                            daily_market=row["daily_market"], outside_cost=row["outside_cost"], infrastructure_kind="hsr", alignments=[alt])
                if through_only:
                    part["through_only"] = True
                mark_stations(part)
                rows.append(part)
                pairs[frozenset((start, end))] = part
            track = oriented(part, start)
            assert not joined or joined[-1] == track[0]
            joined.extend(track[1:] if joined else track)
            resources.append(part["id"])
        main["path"] = points(joined)
        row["shared_sections"] = resources

    divide("public_paris__stop_node_304088466", "strasbourg")
    divide("public_madrid__sevilla", "cordoba_es")
    divide("public_madrid__valencia", "cuenca")
    for trunk in ("public_hengyang__shenzhen", "public_shenzhen__wuhan", "public_changsha__shenzhen", "public_dongguan__shenzhen"):
        divide(trunk, "guangzhou")

    for trunk, junction in junction_splits:
        divide(trunk, junction, through_only=True)

    add_patterns(source, rows, patterns)
    authored = {r["id"] for r in source["services"]}
    for r in patterns:
        if r["id"] in authored:
            r["station_names"] = [hubs[k]["station"] if k in hubs else city[k]["station_name"] for k in r["stops"]]
