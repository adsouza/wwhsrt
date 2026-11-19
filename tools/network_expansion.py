"""Compile single-purchase corridors, their physical legs and through services."""

from pinned_services import add_hubs, add_sections

PROFILES = {
    "Switzerland": "central_europe",
    "Italy": "central_europe",
    "India": "india",
    "Bangladesh": "other_south_asia",
    "Democratic Republic of the Congo": "west_central_africa",
    "Angola": "west_central_africa",
}
QUANTITIES = ("surface_km", "hill_km", "viaduct_km", "tunnel_km")


def construction_row(project, legs, rates, id, *, part=False):
    intervals, path = [], []
    retained = 0
    for leg in legs:
        points = [dict(lon=x, lat=y) for x, y in leg["path"]]
        assert not path or path[-1] == points[0]
        path.extend(points[1:] if path else points)
        if leg.get("public"):
            retained += leg["km"]
            continue
        assert sum(i["km"] for i in leg["cost"]) == leg["km"]
        for entry in leg["cost"]:
            profile = PROFILES[entry["country"]]
            quantities = {k: entry[k] for k in QUANTITIES if k != "surface_km"}
            quantities["surface_km"] = entry["km"] - sum(quantities.values())
            intervals.append(
                dict(
                    country=entry["country"],
                    profile_id=profile,
                    base_per_km=rates[profile],
                    hill_factor_percent=150,
                    tunnel_factor=5,
                    **quantities,
                )
            )
    alignment = dict(
        id="launch",
        km=sum(s["km"] for s in legs),
        minutes=sum(s["minutes"] for s in legs),
        months=project["months"],
        cost_intervals=intervals,
        path=path,
        **{k: sum(i[k] for i in intervals) for k in QUANTITIES},
    )
    if retained:
        alignment.update(retained_km=retained, connector_km=retained)
    row = dict(
        id=id,
        from_city=legs[0]["cities"][0],
        to_city=legs[-1]["cities"][1],
        construction_kind=project["kind"],
        existing=False,
        daily_market=5000,
        outside_cost=15000,
        alignments=[alignment],
    )
    if part:
        row["part_of"] = project["id"]
    else:
        row.update(via=project["via"], opening_service=project["opening_service"])
    return row


def add_expansion(
    source, cities, market, stations, rows, patterns, rates, dwell_minutes
):
    add_hubs(source, cities, market, stations)
    hubs = source["hubs"]
    city = {c["id"]: c for c in cities}

    def station(k):
        return (
            hubs[k]["station"]
            if k in hubs
            else city[k].get("station_name", city[k]["name"])
        )

    def terminals(row):
        ends = [row["from_city"], row["to_city"]]
        row["station_names"] = [station(k) for k in ends]
        ids = [
            hubs[k]["station_id"] if hubs.get(k, {}).get("alternate_terminal") else ""
            for k in ends
        ]
        if any(ids):
            row["terminal_ids"] = ids

    public = []
    add_sections({**source, "infrastructure_kind": "hsr"}, public)
    for row in public:
        terminals(row)
        if row["id"] == "public_stockholm__flemingsberg":
            row.update(infrastructure_kind="conventional", through_only=True)
        rows.append(row)

    def pattern(id, nodes, stops, track, kind):
        patterns.append(
            dict(
                id=id,
                cities=nodes,
                stops=stops,
                sections=track,
                infrastructure_kind=kind,
                station_names=[station(k) for k in stops],
            )
        )

    # A shared physical itinerary can be extended in either direction while
    # technical bridge endpoints remain outside passenger stop lists.
    paths = {}

    def remember(nodes, track):
        paths[nodes[0], nodes[-1]] = (nodes, track)
        paths[nodes[-1], nodes[0]] = (nodes[::-1], track[::-1])

    for co in rows:
        if not co.get("retired"):
            remember([co["from_city"], co["to_city"]], [co["id"]])
    for project in source["projects"]:
        legs, id = project["legs"], project["id"]
        parent = construction_row(project, legs, rates, id)
        rows.append(parent)
        terminals(parent)
        if len(legs) == 1:
            remember(legs[0]["cities"], [id])
            continue
        parent["through_only"] = True
        parent["alignments"][0]["minutes"] += dwell_minutes * (
            len(project["stops"]) - 2
        )
        track = []
        for leg in legs:
            a, b = leg["cities"]
            if leg.get("public"):
                sections = []
                add_sections(dict(hubs=hubs, sections=[leg]), sections)
                co = sections[0]
                co.update(
                    infrastructure_kind="conventional", through_only=True, retired=True
                )
            else:
                co = construction_row(
                    project, [leg], id=id + "_" + a + "__" + b, rates=rates, part=True
                )
                if a.startswith("padma_") or b.startswith("padma_"):
                    co["through_only"] = True
            rows.append(co)
            terminals(co)
            track.append(co["id"])
            remember([a, b], [co["id"]])
        nodes = [legs[0]["cities"][0]] + [leg["cities"][1] for leg in legs]
        stops = project["stops"]
        pattern(project["opening_service"], nodes, stops, track, "new_hsr")
        for a, b in zip(stops, stops[1:]):
            start, end = nodes.index(a), nodes.index(b)
            remember(nodes[start : end + 1], track[start:end])

    public_ids = {c["id"] for c in rows if c["existing"]}
    for service in source["services"]:
        stops = service["stops"]
        nodes, track = [stops[0]], []
        for a, b in zip(stops, stops[1:]):
            part, sections = paths[a, b]
            nodes.extend(part[1:])
            track.extend(sections)
        pattern(
            service["id"],
            nodes,
            stops,
            track,
            "hsr" if public_ids.issuperset(track) else "new_hsr",
        )


def connect_western_vienna(rows, patterns, superseded):
    # The legacy western services sold Vienna tickets as far as Tullnerfeld.
    # Keep their timetable IDs for saved trains, sharing the same western track.
    approach = "public_tullnerfeld__vienna"
    western = {}
    for city in ("dresden", "munich", "nuremberg"):
        prior = next(co for co in rows if co["id"] == f"public_{city}__vienna")
        section = f"public_{city}__tullnerfeld"
        rows.append(dict(prior, id=section, to_city="tullnerfeld"))
        prior.update(retired=True, shared_sections=[section])
        western[prior["id"]] = section
        patterns.append(dict(id=f"{city}__vienna_via_tullnerfeld", cities=[city, "tullnerfeld", "vienna"],
                             stops=[city, "tullnerfeld", "vienna"], sections=[section, approach]))

    for route in list(patterns):
        if route["id"] in superseded or western.keys().isdisjoint(route["sections"]):
            continue
        sections = [western.get(id, id) for id in route["sections"]]
        cities = ["tullnerfeld" if k == "vienna" else k for k in route["cities"]]
        if cities[0] == "tullnerfeld":
            cities.insert(0, "vienna")
            sections.insert(0, approach)
        else:
            assert cities[-1] == "tullnerfeld"
            cities.append("vienna")
            sections.append(approach)
        patterns.append(dict(route, id=route["id"] + "_wien_hbf", cities=cities, sections=sections))
        superseded.append(route["id"])
