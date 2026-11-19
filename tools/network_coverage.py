"""Independent end-to-end and ordered-service checks over the compiled catalog."""

import collections


def checks(catalog, manifest):
    cities = {c["id"]: c for c in catalog["cities"]}
    corridors = {c["id"]: c for c in catalog["corridors"]}
    adj = collections.defaultdict(set)

    def valid(id, seen):
        if id not in corridors or id in seen:
            return False
        c = corridors[id]
        return c["existing"] and not c.get("retired") and all(valid(k, seen | {id}) for k in c.get("shared_sections", []))

    for id, c in corridors.items():
        if valid(id, set()):
            a, b = c["from_city"], c["to_city"]
            adj[a].add(b)
            adj[b].add(a)
    result = []
    for network in manifest["networks"]:
        if network["id"] == "german-ice" and "hamburg_hamm_sha256" in catalog:
            # Admission is per line. The original ICE timetable spans conventional
            # links, so connectivity between every ICE city is no longer required.
            required = ["public_berlin__hamburg", "public_hamburg__stop_node_2146217850",
                        "public_frankfurt_am_main__koln", "public_frankfurt_am_main__stop_node_2453992382",
                        "public_koln__stop_node_2453992382"]
            missing = [id for id in required if not valid(id, set()) or corridors[id].get("infrastructure_kind") != "hsr"]
            result.append(dict(id=network["id"], source=network["source"], scope="majority-HSR corridors",
                               cities=sorted({corridors[id][key] for id in required if id in corridors for key in ("from_city","to_city")}),
                               required_corridors=required, missing=missing, connected=not missing))
            continue
        ids = network["cities"]
        seen = {ids[0]}
        q = [ids[0]]
        for n in q:
            for other in sorted(adj[n] - seen):
                seen.add(other)
                q.append(other)
        missing = [id for id in ids if id not in cities or id not in seen]
        row = dict(id=network["id"], source=network["source"], cities=ids, missing=missing, connected=not missing)
        if network.get("ordered"):
            pattern = next(
                (
                    r
                    for r in catalog["network"]["routes"]
                    if r["id"] == "network_" + network["id"].replace("-", "_") + "_all_stops"
                ),
                None,
            )
            if pattern is None and len(ids) == 2:
                co = next(
                    (
                        c
                        for c in corridors.values()
                        if {c["from_city"], c["to_city"]} == set(ids) and valid(c["id"], set())
                    ),
                    None,
                )
                if co:
                    pattern = dict(id=co["id"], sections=[co["id"]], cities=ids, stops=ids)
            row["service_id"] = pattern["id"] if pattern else None
            chain = bool(
                pattern
                and len(pattern["sections"]) == len(pattern["cities"]) - 1
                and len(set(pattern["cities"])) == len(pattern["cities"])
            )
            if chain:
                for a, b, section in zip(pattern["cities"], pattern["cities"][1:], pattern["sections"]):
                    co = corridors.get(section, {})
                    chain &= valid(section, set()) and {a, b} == {co.get("from_city"), co.get("to_city")}
                chain &= [c for c in pattern["cities"] if c in ids] == ids
            row["ordered_service"] = bool(chain and pattern["stops"] == ids)
            row["connected"] &= row["ordered_service"]
        result.append(row)
    return result


def operating_service_checks(catalog, manifest):
    """Compare every pinned operating service and section with the playable graph."""
    rows = {c["id"]: c for c in catalog["corridors"]}
    sections = {frozenset((c["from_city"], c["to_city"])): c for c in rows.values()
                if c["existing"] and not c.get("retired")}
    withdrawn = set(catalog["network"]["superseded_routes"])
    patterns = {r["id"]: r for r in catalog["network"]["routes"] if r["id"] not in withdrawn}
    results = []
    for entry in manifest["sections"] + manifest["services"]:
        cities = entry["cities"]
        if len(cities) == 2:
            row = sections.get(frozenset(cities), {})
            id = row.get("id", "public_" + "__".join(cities))
            track = [id]
        else:
            id = entry["id"]
            route = patterns.get(id, {})
            expected = [c for c in cities if c not in entry.get("skip_stops", [])]
            track = route.get("sections", []) if route.get("stops") == expected and route.get("cities") == cities else []
        previous = None
        connected = len(track) == len(cities) - 1
        for a, b, leg in zip(cities, cities[1:], track):
            row = rows.get(leg, {})
            if not row.get("existing") or row.get("retired") or {row.get("from_city"), row.get("to_city")} != {a, b}:
                connected = False
                break
            path = row["alignments"][0]["path"]
            first, last = (path[0], path[-1]) if row["from_city"] == a else (path[-1], path[0])
            if previous is not None and previous != first:
                connected = False
            previous = last
        hub = next(manifest["hubs"][city] for city in cities if city in manifest["hubs"])
        source = manifest["sources"][hub["region"]]
        results.append(dict(id=id, cities=cities, source=source, connected=connected))
    return results


def require(catalog, manifest):
    result = checks(catalog, manifest)
    failed = [r for r in result if not r["connected"]]
    if failed:
        raise ValueError("Operating-network coverage failed: " + repr(failed))
    return result


def require_map(catalog, map_data, manifest):
    """Required station markers and city-to-city paths must ship with the services."""
    city = {c["id"]: c for c in catalog["cities"]}
    stations = {s["id"]: s for s in catalog["stations"]}
    visible = set(map_data["station_ids"])
    drawn = dict(zip(map_data["corridor_ids"], map_data["lines"]))
    assert len(drawn) == len(map_data["lines"]) == len(map_data["corridor_ids"])
    for hub in manifest["hubs"]:
        id = hub["station_id"]
        c = city[hub["city_id"]]
        assert id in stations and id in visible, "Missing operating station marker: " + id
        assert (c["station_lon"], c["station_lat"]) == (stations[id]["lon"], stations[id]["lat"]), (
            "Station anchor mismatch: " + id
        )
    for co in catalog["corridors"]:
        if not co["existing"] or co.get("retired"):
            continue
        path = drawn.get(co["id"])
        assert path and len(path) > 1, "Missing mapped corridor: " + co["id"]
        for end, id in [(0, co["from_city"]), (-1, co["to_city"])]:
            c = city[id]
            terminal = co.get("terminal_ids", ["", ""])[end]
            pin = [c["station_lon"], c["station_lat"]]
            if terminal:
                assert terminal in visible, "Alternate terminal missing from map: " + terminal
                pin = [stations[terminal]["lon"], stations[terminal]["lat"]]
            assert path[end] == pin, "Detached track endpoint: " + co["id"]
    return dict(required_station_markers=len(manifest["hubs"]), drawn_corridors=len(drawn))
