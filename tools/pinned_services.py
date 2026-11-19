"""Import operator services with pinned station-to-station railway geometry."""

from math import hypot

from shapely.geometry import LineString


def add_hubs(source, cities, market, stations):
    by_city = {c["id"]: c for c in cities}
    by_station = {s["id"]: s for s in stations}
    for city_id, hub in source["hubs"].items():
        lon, lat = hub["lon"], hub["lat"]
        station_id = hub["station_id"]
        if station_id not in by_station:
            station = dict(id=station_id, name=hub["station"], lon=lon, lat=lat, tagged=False, operator="")
            stations.append(station)
            by_station[station_id] = station
        by_station[station_id]["map_gap"] = False
        if city_id not in by_city:
            city = dict(id=city_id, name=hub["city"], lon=lon, lat=lat)
            cities.append(city)
            by_city[city_id] = city
            market[city_id] = dict(daily_trips=500, attraction=500, road_region=hub["region"])
        if hub.get("alternate_terminal"):
            continue
        by_city[city_id].update(station_name=hub["station"], station_lon=lon, station_lat=lat)
        market[city_id].update(by_city[city_id])


def add_sections(source, rows):
    def points(coords):
        return [dict(lon=x, lat=y) for x, y in coords]

    kind = {"infrastructure_kind": source["infrastructure_kind"]} if "infrastructure_kind" in source else {}
    for section in source["sections"]:
        a, b = section["cities"]
        coords = list(section["path"])
        assert LineString(coords).is_simple, ("Railway doubles back", a, b)
        for i, city_id in [(0, a), (-1, b)]:
            hub = source["hubs"][city_id]
            assert hypot(coords[i][0] - hub["lon"], coords[i][1] - hub["lat"]) < 0.001
            coords[i] = (hub["lon"], hub["lat"])
        distance = section["km"]
        alignment = dict(
            id="public_rail",
            km=distance,
            minutes=section["minutes"],
            months=0,
            surface_km=distance,
            hill_km=0,
            viaduct_km=0,
            tunnel_km=0,
            path=points(coords),
        )
        rows.append(
            dict(
                id=f"public_{a}__{b}",
                from_city=a,
                to_city=b,
                existing=True,
                daily_market=5000,
                outside_cost=15000,
                alignments=[alignment],
                **kind,
                **({"via": section["via"]} if section.get("via") else {}),
            )
        )


def add_patterns(source, rows, patterns):
    kind = {"infrastructure_kind": source["infrastructure_kind"]} if "infrastructure_kind" in source else {}
    sections = {frozenset((c["from_city"], c["to_city"])): c["id"] for c in rows if c["existing"]}
    superseded = []
    for service in source["services"]:
        path = service["cities"]
        if "replaces" in service:
            prior = next(p for p in patterns if p["id"] == service["replaces"])
            assert [prior["cities"][0], prior["cities"][-1]] == [path[0], path[-1]]
            superseded.append(prior["id"])
        patterns.append(
            dict(
                id=service["id"],
                cities=path,
                **kind,
                stops=[c for c in path if c not in service.get("skip_stops", ())],
                sections=[sections[frozenset(pair)] for pair in zip(path, path[1:])],
            )
        )
    return superseded


def add_services(source, cities, market, stations, rows, patterns):
    add_hubs(source, cities, market, stations)
    add_sections(source, rows)
    return add_patterns(source, rows, patterns)
