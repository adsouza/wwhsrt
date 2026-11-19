"""Connect the two construction contracts to existing public approaches."""

from occitanie import works
from pinned_services import add_sections


def add_branch_projects(source, rows, patterns, rates):
    hubs = source["hubs"]
    approaches = []
    add_sections(source, approaches)
    for row in approaches:
        endpoints = [hubs[row[k]] for k in ("from_city", "to_city")]
        row.update(retired=True, through_only=True, infrastructure_kind="conventional",
                   station_names=[h["station"] for h in endpoints])
        if any(h.get("alternate_terminal") for h in endpoints):
            row["terminal_ids"] = [h["station_id"] if h.get("alternate_terminal") else "" for h in endpoints]
        rows.append(row)

    for p in source["projects"]:
        a, b = p["cities"]
        terrain = dict(hill=0, viaduct=0, tunnel=p["tunnel_km"] / p["km"]) if "tunnel_km" in p else {}
        row = works(p["id"], a, b, [dict(lon=x, lat=y) for x, y in p["path"]],
                    p["km"], p["minutes"], p["months"], "central_europe", rates["central_europe"],
                    country=p["country"], **terrain)
        row.update(construction_kind=p["kind"], opening_service=p["opening_service"], station_names=[hubs[k]["station"] for k in (a, b)])
        if p.get("via"):
            row["via"] = p["via"]
        # Psary is an operating junction without passenger platforms.
        if a == "psary":
            row["through_only"] = True
        rows.append(row)

    def service(id, cities, stops, sections, names):
        patterns.append(dict(id=id, cities=cities, stops=stops, sections=sections,
                             station_names=names, infrastructure_kind="new_hsr"))

    service("warsaw__krakow", ["warsaw", "wloszczowa", "psary", "krakow"],
            ["warsaw", "wloszczowa", "krakow"],
            ["public_warsaw__wloszczowa", "public_wloszczowa__psary", "upgrade_psary__krakow"],
            ["Warszawa Centralna", "Włoszczowa Północ", "Kraków Główny"])
    cities = ["vienna", "gloggnitz", "muerzzuschlag", "graz"]
    track = ["public_vienna__gloggnitz", "semmering_base_tunnel", "public_muerzzuschlag__graz"]
    stops = ["vienna", "graz"]
    names = [hubs[k]["station"] for k in stops]
    service("vienna__graz", cities, stops, track, names)
    service("vienna__graz_via_gloggnitz", cities, ["vienna", "gloggnitz", "graz"], track,
            [names[0], hubs["gloggnitz"]["station"], names[1]])
    service("vienna__graz_via_both_portals", cities, cities, track, [hubs[k]["station"] for k in cities])
    service("vienna__klagenfurt", cities + ["klagenfurt"], stops + ["klagenfurt"],
            track + ["public_graz__klagenfurt"], names + ["Klagenfurt Hauptbahnhof"])
