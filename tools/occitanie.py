"""Starting RFN services and a hypothetical Toulouse–Narbonne high-speed project."""

from math import ceil


def works(
    id,
    a,
    b,
    coords,
    length,
    minutes,
    months,
    profile,
    base,
    retained=0,
    hill=0.1,
    viaduct=0.08,
    tunnel=0.03,
    country=None,
):
    build_km = length - retained
    h, v, t = (round(build_km * fraction) for fraction in (hill, viaduct, tunnel))
    quantities = dict(
        surface_km=build_km - h - v - t, hill_km=h, viaduct_km=v, tunnel_km=t
    )
    interval = dict(
        country=country or ("Germany" if profile == "central_europe" else "France"),
        profile_id=profile,
        base_per_km=base,
        hill_factor_percent=150,
        tunnel_factor=5,
        **quantities,
    )
    alt = dict(
        id="launch",
        km=length,
        minutes=minutes,
        months=months,
        path=coords,
        cost_intervals=[interval],
        **quantities,
    )
    if retained:
        alt["retained_km"] = retained
    return dict(
        id=id,
        from_city=a,
        to_city=b,
        existing=False,
        daily_market=5000,
        outside_cost=15000,
        construction_kind="dedicated_hsr",
        alignments=[alt],
    )


def add_occitanie(
    source, cities, market, stations, rows, patterns, superseded, km, rates
):
    by_city = {c["id"]: c for c in cities}
    for id, hub in source["hubs"].items():
        pin = {"lon": hub["lon"], "lat": hub["lat"]}
        if id not in by_city:
            by_city[id] = dict(id=id, name=hub["station"], **pin)
            cities.append(by_city[id])
            market[id] = dict(
                daily_trips=500,
                attraction=500,
                road_region=market["toulouse"]["road_region"],
            )
        by_city[id].update(
            station_lon=pin["lon"], station_lat=pin["lat"], station_name=hub["station"]
        )
        market[id].update(by_city[id])
        stations.append(
            dict(
                id="rfn/" + id,
                name=hub["station"],
                operator="SNCF",
                tagged=False,
                map_gap=False,
                **pin,
            )
        )
        for co in rows:
            for side, i in (("from_city", 0), ("to_city", -1)):
                if co[side] == id:
                    for alt in co["alignments"]:
                        alt["path"][i] = pin.copy()

    def point(id):
        c = by_city[id]
        return dict(lon=c["station_lon"], lat=c["station_lat"])

    def points(coords):
        return [dict(lon=x, lat=y) for x, y in coords]

    def length(coords):
        return sum(km(a, b) for a, b in zip(coords, coords[1:]))

    public = []
    for s in source["sections"]:
        a, b = s["cities"]
        n = round(length(s["path"]))
        id = "public_" + a + "__" + b
        public.append(id)
        rows.append(
            dict(
                id=id,
                from_city=a,
                to_city=b,
                existing=True,
                daily_market=5000,
                outside_cost=15000,
                infrastructure_kind="conventional",
                alignments=[
                    dict(
                        id="rfn",
                        km=n,
                        minutes=s["minutes"],
                        months=0,
                        surface_km=n,
                        hill_km=0,
                        viaduct_km=0,
                        tunnel_km=0,
                        path=points(s["path"]),
                    )
                ],
            )
        )
    fast = "occitanie_toulouse__narbonne"
    # Strategic design follows the Lauragais and Aude valleys. Durations, engineering
    # shares and the 8% alignment allowance are game assumptions.
    alignment = [
        point("toulouse"),
        *points([(1.49, 43.58), (1.71, 43.40), (1.96, 43.32)]),
        point("carcassonne"),
        *points([(2.55, 43.21), (2.76, 43.18)]),
        point("narbonne"),
    ]
    rail_km = ceil(length([(p["lon"], p["lat"]) for p in alignment]) * 1.08)
    row = works(
        fast,
        "toulouse",
        "narbonne",
        alignment,
        rail_km,
        ceil(rail_km * 60 / 250) + 8,
        48,
        "france_benelux",
        rates["france_benelux"],
    )
    row["construction_kind"] = "hypothetical_hsr"
    rows.append(row)
    # Narbonne is a junction on one coastal contract. Purchasing that contract
    # opens both parts and consumes the same capacity as its end-to-end service.
    coast = next(c for c in rows if c["id"] == "launch_25_beziers__perpignan")
    alt = coast["alignments"][0]
    pin = point("narbonne")
    path = [alt["path"][0], pin, *alt["path"][2:]]
    alt["path"] = path
    south, north = "occitanie_narbonne__perpignan", "occitanie_beziers__narbonne"
    for id, a, b, segment, mins in (
        (north, "beziers", "narbonne", path[:2], 10),
        (south, "narbonne", "perpignan", path[1:], 19),
    ):
        n = round(length([(p["lon"], p["lat"]) for p in segment]))
        part = works(
            id,
            a,
            b,
            segment,
            n,
            mins,
            alt["months"],
            "france_benelux",
            rates["france_benelux"],
        )
        part["part_of"] = coast["id"]
        if id == north:
            part["parallel_to"] = [public[2]]
        rows.append(part)

    def service(id, stops, sections, kind=""):
        r = dict(id="occitanie_" + id, cities=stops, stops=stops, sections=sections)
        if kind:
            r["infrastructure_kind"] = kind
        patterns.append(r)

    service(
        "toulouse__narbonne_local",
        ["toulouse", "carcassonne", "narbonne"],
        public[:2],
        "conventional",
    )
    service(
        "toulouse__beziers",
        ["toulouse", "carcassonne", "narbonne", "beziers"],
        public,
        "conventional",
    )
    service(
        "toulouse__beziers_fast",
        ["toulouse", "narbonne", "beziers"],
        [fast, public[2]],
        "new_hsr",
    )
    service(
        "toulouse__beziers_coastal",
        ["toulouse", "narbonne", "beziers"],
        [fast, north],
        "new_hsr",
    )
    service(
        "toulouse__montpellier_via_beziers",
        ["toulouse", "narbonne", "beziers", "montpellier"],
        [fast, public[2], "launch_24_montpellier__beziers"],
    )
    service(
        "toulouse__montpellier",
        ["toulouse", "narbonne", "beziers", "montpellier"],
        [fast, north, "launch_24_montpellier__beziers"],
    )
    service("toulouse__perpignan", ["toulouse", "narbonne", "perpignan"], [fast, south])
    spanish = next(r for r in patterns if r["id"] == "launch_route_barcelona__toulouse")
    end = spanish["cities"].index("perpignan")
    service(
        "barcelona__toulouse",
        spanish["cities"][: end + 1] + ["narbonne", "toulouse"],
        spanish["sections"][:end] + [south, fast],
    )
    for r in list(patterns):
        if r["id"].startswith("launch_route_") and {
            "toulouse",
            "beziers",
            "paris",
        } <= set(r["cities"]):
            # Provide the corresponding shortcut even for distant Spanish origins.
            if r["id"] not in superseded:
                superseded.append(r["id"])
            ordered = r["cities"]
            sections = r["sections"]
            if ordered[0] == "toulouse":
                ordered = ordered[::-1]
                sections = sections[::-1]
            if "perpignan" in ordered and ordered[0] not in ("barcelona", "perpignan"):
                i = ordered.index("perpignan")
                service(
                    ordered[0] + "__toulouse",
                    ordered[: i + 1] + ["narbonne", "toulouse"],
                    sections[:i] + [south, fast],
                )
