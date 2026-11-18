"""Compile the pinned prototype into a strategic, offline public rail catalog.

Geometry is imported verbatim as a map layer. Operating links follow its connected
polyline graph; small endpoint gaps are recorded, never arbitrary city chords.
Stations without mapped links retain their source status. No network fetches.
"""

import base64, collections, hashlib, heapq, json, math, pathlib, re, sys, zlib, unicodedata, gzip
from shapely import STRtree
from shapely.geometry import LineString, Point
from scipy.spatial import cKDTree

ROOT = pathlib.Path(__file__).resolve().parents[1]


def read(p):
    return json.loads((ROOT / p).read_text(encoding="utf-8"))


PENDING = {}


def emit(p, obj):
    PENDING[p] = (json.dumps(obj, ensure_ascii=False, separators=(",", ":")) + "\n").encode("utf-8")


def output_hash(p):
    return hashlib.sha256(PENDING[p] if p in PENDING else (ROOT / p).read_bytes()).hexdigest()


def flush():
    for p, data in PENDING.items():
        target = ROOT / p
        if "--check" in sys.argv:
            if not target.exists() or target.read_bytes() != data:
                raise SystemExit("Generated asset differs: " + p)
        else:
            target.write_bytes(data)


def km(a, b):
    lon1, lat1, lon2, lat2 = map(math.radians, (*a, *b))
    v = math.sin((lat2 - lat1) / 2) ** 2 + math.cos(lat1) * math.cos(lat2) * math.sin((lon2 - lon1) / 2) ** 2
    return 12742 * math.asin(min(1, math.sqrt(v)))


def normal(s):
    return re.sub(
        r"[^a-z0-9]+",
        "_",
        unicodedata.normalize("NFKD", s.translate(str.maketrans({"Ł": "L", "ł": "l"})))
        .encode("ascii", "ignore")
        .decode()
        .lower(),
    ).strip("_")


def main(audit=False):
    source = read("data/imports/hsr-prototype.v1.json")
    lines = []
    for row in json.loads(zlib.decompress(base64.b64decode(source["existing"]))):
        x = y = 0
        p = []
        for i in range(0, len(row), 2):
            x += row[i]
            y += row[i + 1]
            p.append((x / 10000, y / 10000))
        lines.append(p)
    shapes = [LineString(p) for p in lines]
    tree = STRtree(shapes)
    hidden = set(source["stationMetadata"]["stationsWithoutMappedLink"])
    stations = [
        dict(id=sid, name=name, lon=lon, lat=lat, tagged=bool(tag), operator=op or "", map_gap=sid in hidden)
        for sid, name, lon, lat, tag, op in source["stations"]
        if sid not in hidden or op
    ]
    pins = collections.defaultdict(list)
    matches = collections.defaultdict(list)
    for s in stations:
        p = Point(s["lon"], s["lat"])
        best = None
        for i in tree.query(p.buffer(0.012)):
            q = shapes[i].interpolate(shapes[i].project(p))
            dist = km((p.x, p.y), (q.x, q.y))
            if dist <= 0.4 and (best is None or (dist, int(i)) < (best[0], best[1])):
                best = (dist, int(i), (q.x, q.y), shapes[i].project(p))
        if best:
            for i in tree.query(p.buffer(0.012)):
                q = shapes[i].interpolate(shapes[i].project(p))
                dist = km((p.x, p.y), (q.x, q.y))
                if dist <= 0.4:
                    key = s["id"] + "@" + str(int(i))
                    pins[int(i)].append((shapes[i].project(p), (q.x, q.y), key))
                    matches[s["id"]].append(key)
    graph = collections.defaultdict(dict)
    station_node = {}
    coords = []
    index = {}

    def node(p):
        p = tuple(round(x, 7) for x in p)
        if p not in index:
            index[p] = len(coords)
            coords.append(p)
        return index[p]

    for i, shape in enumerate(shapes):
        points = [(shape.project(Point(p)), p, None) for p in lines[i]] + pins[i]
        points.sort(key=lambda x: (x[0], x[2] or ""))
        prev = None
        for _, p, sid in points:
            n = node(p)
            if sid:
                station_node[sid] = n
            if prev is not None and n != prev:
                dist = max(1, round(km(coords[prev], coords[n]) * 1000))
                graph[n][prev] = min(dist, graph[n].get(prev, dist))
                graph[prev][n] = graph[n][prev]
            prev = n
    for station in stations:
        sid = station["id"]
        if sid not in matches:
            continue
        hub = node((station["lon"], station["lat"]))
        for key in matches[sid]:
            n = station_node.pop(key)
            d = max(1, round(km(coords[hub], coords[n]) * 1000))
            if n != hub:
                graph[hub][n] = graph[n][hub] = d
        station_node[sid] = hub
    # Join only close polyline endpoints, not arbitrary crossings/interiors.
    endpoints = sorted({node(p) for line in lines for p in (line[0], line[-1])})
    xyz = lambda p: (
        math.cos(math.radians(p[1])) * math.cos(math.radians(p[0])),
        math.cos(math.radians(p[1])) * math.sin(math.radians(p[0])),
        math.sin(math.radians(p[1])),
    )
    near = cKDTree([xyz(coords[n]) for n in endpoints])
    gaps = []
    for a, b in sorted(near.query_pairs(150 / 6371000)):
        u, v = endpoints[a], endpoints[b]
        if v in graph[u]:
            continue
        dist = round(km(coords[u], coords[v]) * 1000)
        if dist and dist <= 150:
            graph[u][v] = graph[v][u] = dist
            gaps.append([u, v, dist])
    components = {}
    comp_nodes = []
    for start in range(len(coords)):
        if start in components:
            continue
        cid = len(comp_nodes)
        q = [start]
        components[start] = cid
        for n in q:
            for nxt in graph[n]:
                if nxt not in components:
                    components[nxt] = cid
                    q.append(nxt)
        comp_nodes.append(q)
    by_component = collections.defaultdict(list)
    for s in stations:
        if s["id"] in station_node:
            by_component[components[station_node[s["id"]]]].append(s)
    cities = []
    city_by_id = {}
    market = {}
    legacy = read("assets/scenario/scenario.json")["cities"]
    oldmarket = read("assets/scenario/network.json")["cities"]

    def add(c, pop=100000, region=""):
        if c["id"] in city_by_id or c["id"] in {"manila", "brasilia"}:
            return
        cities.append(c)
        city_by_id[c["id"]] = c
        m = dict(
            c,
            daily_trips=max(500, min(50000, round(pop / 1000))),
            attraction=max(500, min(50000, round(pop / 1000))),
            road_region=road_region(region, c["lat"]) if region else "imported",
        )
        market[c["id"]] = m

    countries = {
        "AE": "United Arab Emirates",
        "AR": "Argentina",
        "AT": "Austria",
        "AU": "Australia",
        "BD": "Bangladesh",
        "BE": "Belgium",
        "BR": "Brazil",
        "CA": "Canada",
        "CL": "Chile",
        "CN": "China",
        "CO": "Colombia",
        "CZ": "Czechia",
        "DE": "Germany",
        "EG": "Egypt",
        "ES": "Spain",
        "FR": "France",
        "GB": "United Kingdom",
        "ID": "Indonesia",
        "IN": "India",
        "IT": "Italy",
        "JP": "Japan",
        "KE": "Kenya",
        "KR": "South Korea",
        "MA": "Morocco",
        "MX": "Mexico",
        "NG": "Nigeria",
        "NO": "Norway",
        "PE": "Peru",
        "PK": "Pakistan",
        "PL": "Poland",
        "PT": "Portugal",
        "RS": "Serbia",
        "SA": "Saudi Arabia",
        "SE": "Sweden",
        "TH": "Thailand",
        "TR": "Turkey",
        "TW": "Taiwan",
        "TZ": "Tanzania",
        "UG": "Uganda",
        "US": "United States",
        "UZ": "Uzbekistan",
        "ZA": "South Africa",
    }

    def road_region(country, lat):
        value = normal(countries.get(country, country))
        if value == "japan":
            return "hokkaido" if lat > 41 else "honshu"
        return "korea" if value == "south_korea" else value

    for c in legacy:
        add(dict(c))
    for m in oldmarket:
        market[m["id"]] = m
    for c in read("data/prototypes/playable-metros.v1.json"):
        name = c["name"].removesuffix(" Metropolitan Area")
        add(dict(id=c["id"], name=name, lat=c["lat"], lon=c["lng"]), c["population"], c["country"])
    selection = read("data/prototypes/catalog-city-selection.v1.json")
    for group, rows in selection["groups"].items():
        for c in rows:
            if c.get("lat") is None or c.get("lng") is None:
                continue
            id = normal(c["name"])
            add(
                dict(id=id, name=c["name"], lat=c["lat"], lon=c["lng"]),
                c.get("ghsl_pop_2025") or c.get("population") or 100000,
                c.get("country_code", ""),
            )
    station_xyz = cKDTree([xyz((s["lon"], s["lat"])) for s in stations])
    city_pins = {}
    claimed = {}
    # Prefer a station on a substantial component; city-level station aggregation
    # is explicit, while the rail path itself always stays on the source graph.
    for c in cities:
        candidates = station_xyz.query_ball_point(xyz((c["lon"], c["lat"])), 40 / 6371)
        choices = []
        for i in candidates:
            s = stations[i]
            if re.search(r"yard|depot|depository", s["name"], re.I):
                continue
            if s["id"] not in station_node:
                continue
            n = station_node[s["id"]]
            dist = km((c["lon"], c["lat"]), (s["lon"], s["lat"]))
            if dist > 40:
                continue
            size = len(by_component[components[n]])
            # Main stations beat tiny adjacent fragments, then use distance.
            names = {
                "tokyo": "Tokyo",
                "nagoya": "Nagoya",
                "osaka": "Shin-Osaka",
                "beijing": "Beijingnan",
                "shanghai": "Shanghai-Hongqiao",
                "hong_kong": "Hong Kong West Kowloon",
                "seoul": "Seoul",
            }
            expected = names.get(c["id"])
            match = s["name"] == expected if expected else normal(c["name"]) in normal(s["name"])
            choices.append((not match, size < 4, not bool(s["operator"]), -int(s["tagged"]), dist, s["id"], n))
        if choices:
            *_, sid, n = min(choices)
            if n not in claimed:
                city_pins[c["id"]] = (n, sid)
                claimed[n] = c["id"]
    # Retain small isolated networks through their actual stations, rather than
    # inventing connections to distant major metros.
    for cid, ss in sorted(by_component.items()):
        served = [c for c, (n, sid) in city_pins.items() if components[n] == cid]
        if len(served) >= 2:
            continue
        chosen = []
        hubs = [city_pins[c][0] for c in served]
        for s in sorted(
            (
                s
                for s in ss
                if (s["operator"] or s["tagged"]) and not re.search(r"yard|depot|depository", s["name"], re.I)
            ),
            key=lambda s: (not bool(s["operator"]), not s["tagged"], s["id"]),
        ):
            n = station_node[s["id"]]
            if n in claimed:
                continue
            if hubs and min(km(coords[n], coords[other]) for other in hubs) < 5:
                continue
            chosen.append(s)
            hubs.append(n)
            if len(hubs) == 2:
                break
        # A lone fragment never creates a new isolated service endpoint.
        if len(hubs) < 2:
            continue
        for s in chosen:
            n = station_node[s["id"]]
            id = "stop_" + re.sub(r"[^a-z0-9]+", "_", s["id"])
            add(dict(id=id, name=s["name"], lat=s["lat"], lon=s["lon"]), 100000, "component_" + str(cid))
            city_pins[id] = (n, s["id"])
            claimed[n] = id
    # Read-only diagnostics stop before any generated catalog/map output is written.
    if audit:
        return dict(
            lines=lines,
            shapes=shapes,
            tree=tree,
            stations=stations,
            station_node=station_node,
            coords=coords,
            index=index,
            graph=graph,
            components=components,
            comp_nodes=comp_nodes,
            by_component=by_component,
            city_pins=city_pins,
            cities=cities,
        )
    # Contract the connected rail graph to neighboring strategic station hubs.
    # A multi-source shortest-path forest avoids an all-pairs corridor explosion
    # and keeps shared trunk paths represented once between city catchments.
    owner = {}
    dist = {}
    prev = {}
    todo = []
    for city, (n, sid) in sorted(city_pins.items()):
        owner[n] = city
        dist[n] = 0
        heapq.heappush(todo, (0, city, n))
    while todo:
        cost, city, n = heapq.heappop(todo)
        if cost != dist[n] or owner[n] != city:
            continue
        for nxt, weight in sorted(graph[n].items()):
            v = (cost + weight, city)
            if v < (dist.get(nxt, 10**15), owner.get(nxt, "~")):
                dist[nxt] = v[0]
                owner[nxt] = city
                prev[nxt] = n
                heapq.heappush(todo, (v[0], city, nxt))
    links = {}
    for n in sorted(graph):
        for nxt, weight in sorted(graph[n].items()):
            if n >= nxt or n not in owner or nxt not in owner or owner[n] == owner[nxt]:
                continue
            a, b = owner[n], owner[nxt]
            key = tuple(sorted((a, b)))
            cost = dist[n] + weight + dist[nxt]
            if key in links and links[key][0] <= cost:
                continue
            left = [n]
            right = [nxt]
            while left[-1] in prev:
                left.append(prev[left[-1]])
            while right[-1] in prev:
                right.append(prev[right[-1]])
            path = list(reversed(left)) + right
            links[key] = (cost, path if key[0] == a else list(reversed(path)))
    out = []
    for (a, b), (meters, path) in sorted(links.items()):
        shape = LineString([coords[n] for n in path]).simplify(0.0005, preserve_topology=False)
        points = [dict(lon=round(x, 6), lat=round(y, 6)) for x, y in shape.coords]
        k = max(1, round(meters / 1000))
        out.append(
            dict(
                id="public_" + a + "__" + b,
                from_city=a,
                to_city=b,
                existing=True,
                daily_market=5000,
                outside_cost=15000,
                alignments=[
                    dict(
                        id="prototype",
                        km=k,
                        minutes=max(5, math.ceil(k * 60 / 250)),
                        months=0,
                        surface_km=k,
                        hill_km=0,
                        viaduct_km=0,
                        tunnel_km=0,
                        path=points,
                    )
                ],
            )
        )
    by_station = {s["id"]: s for s in stations}
    for id, (n, sid) in city_pins.items():
        s = by_station[sid]
        city_by_id[id]["station_lat"] = s["lat"]
        city_by_id[id]["station_lon"] = s["lon"]
        city_by_id[id]["station_name"] = s["name"]
        market[id].update(city_by_id[id])
    # Stable compatibility IDs keep existing companies' accepted services/fleet.
    legacy_rows = read("assets/scenario/business.json")
    rows = list(legacy_rows)
    existing_pairs = {frozenset((r["from_city"], r["to_city"])) for r in rows}
    for co in out:
        if frozenset((co["from_city"], co["to_city"])) not in existing_pairs:
            rows.append(co)
            existing_pairs.add(frozenset((co["from_city"], co["to_city"])))
    from network_repair_catalog import merge

    rows, repair_metadata = merge(rows)
    cities = repair_metadata["cities"]
    city_by_id = {c["id"]: c for c in cities}
    market = {c["id"]: c for c in repair_metadata["market_cities"]}
    manifest = read("data/imports/operating-networks.v2.json")
    station_by_id = {s["id"]: s for s in stations}
    for h in manifest["hubs"]:
        sid = h["station_id"]
        station = station_by_id.get(sid)
        if station is None:
            station = dict(
                id=sid,
                name=h["name"],
                lon=h["lon"],
                lat=h["lat"],
                tagged=False,
                operator="operator_confirmed",
                map_gap=False,
            )
            stations.append(station)
            station_by_id[sid] = station
        station["map_gap"] = False
        station["evidence"] = h["evidence"]

    # The 68 newly authored works extend the two immutable Japanese projects.
    launch = read("data/construction/compiled-prospects.v1.json")
    assert (
        launch["source_catalog_sha256"]
        == hashlib.sha256(
            gzip.decompress((ROOT / "assets/scenario/public-network-v3.json.gz").read_bytes())
        ).hexdigest()
    )
    assert launch["authoring_sha256"] == output_hash("data/construction/launch-prospects.v1.json")
    assert launch["land_mask_sha256"] == output_hash("data/construction/route-land.v1.json.gz")
    assert launch["cost_ledger_sha256"] == output_hash("data/costs/construction_costs.v1.json")
    assert launch["operating_manifest_sha256"] == output_hash("data/imports/operating-networks.v2.json")
    assert (
        launch["accepted_catalog_sha256"]
        == hashlib.sha256(
            gzip.decompress((ROOT / "assets/scenario/public-network-v5.json.gz").read_bytes())
        ).hexdigest()
    )
    rows.extend(launch["corridors"])
    for terminal in launch["terminals"]:
        id = terminal["id"]
        city_by_id[id].update(terminal)
        market[id].update(terminal)
    for co in rows:
        if co["id"] == "public_lisbon__porto":
            co["infrastructure_kind"] = "upgraded_conventional"
        if co["id"] == "public_kenitra__tangier":
            co["infrastructure_kind"] = "dedicated_hsr"
    existing_pairs = {frozenset((r["from_city"], r["to_city"])) for r in rows}
    out = [co for co in rows if co["existing"]]
    # Rendering corrections are separate from frozen economic/save inputs.
    # Reuse the source railway for legacy service IDs and station approaches.
    source_adj = collections.defaultdict(list)
    for co in out:
        source_adj[co["from_city"]].append((co["to_city"], co))
        source_adj[co["to_city"]].append((co["from_city"], co))

    def rail_path(a, b):
        ds = {a: 0}
        previous = {}
        queue = [(0, a)]
        while queue:
            cost, n = heapq.heappop(queue)
            if cost != ds[n]:
                continue
            if n == b:
                break
            for nxt, co in sorted(source_adj[n], key=lambda x: x[1]["id"]):
                value = cost + co["alignments"][0]["km"]
                if value < ds.get(nxt, 10**12):
                    ds[nxt] = value
                    previous[nxt] = (n, co)
                    heapq.heappush(queue, (value, nxt))
        if b not in ds:
            return None
        chunks = []
        n = b
        while n != a:
            prev, co = previous[n]
            path = co["alignments"][0]["path"]
            chunks.append(path if co["from_city"] == prev else list(reversed(path)))
            n = prev
        result = []
        for path in reversed(chunks):
            result.extend(path if not result else path[1:])
        return result

    display = {}
    for co in legacy_rows:
        display[co["id"]] = {}
        for alignment in co["alignments"]:
            path = rail_path(co["from_city"], co["to_city"]) if co["existing"] else None
            if not path:
                path = list(alignment["path"])
                if co["to_city"] == "osaka":
                    kyoto = city_by_id["kyoto"]
                    best = min(
                        range(len(path)),
                        key=lambda i: km(
                            (path[i]["lon"], path[i]["lat"]), (kyoto["station_lon"], kyoto["station_lat"])
                        ),
                    )
                    tail = rail_path("kyoto", "osaka")
                    if (
                        tail
                        and km((path[best]["lon"], path[best]["lat"]), (kyoto["station_lon"], kyoto["station_lat"])) < 1
                    ):
                        path = path[:best] + tail
                for end, id in [(0, co["from_city"]), (-1, co["to_city"])]:
                    city = city_by_id[id]
                    if city.get("station_name"):
                        path[end] = {"lon": city["station_lon"], "lat": city["station_lat"]}
            display[co["id"]][alignment["id"]] = path
    emit("assets/scenario/rail-display-paths.json", display)
    adjacency = collections.defaultdict(list)
    for co in repair_metadata["physical_corridors"]:
        if not co["existing"]:
            continue
        a, b = co["from_city"], co["to_city"]
        adjacency[a].append((b, co))
        adjacency[b].append((a, co))
    patterns = list(
        json.loads(gzip.decompress((ROOT / "assets/scenario/public-network-v2.json.gz").read_bytes()))["network"][
            "routes"
        ]
    )
    # Every connected strategic pair gets an authored shortest-rail pattern.
    # All intermediate strategic stops are called; direct corridor IDs stay stable.
    route_pairs = {frozenset(r["cities"][:: len(r["cities"]) - 1]) for r in patterns}
    for a in sorted(adjacency):
        ds = {a: 0}
        previous = {}
        queue = [(0, a)]
        while queue:
            cost, n = heapq.heappop(queue)
            if ds[n] != cost:
                continue
            for nxt, co in sorted(adjacency[n], key=lambda x: (x[0], x[1]["id"])):
                value = cost + co["alignments"][0]["minutes"]
                if value < ds.get(nxt, 10**12):
                    ds[nxt] = value
                    previous[nxt] = (n, co["id"])
                    heapq.heappush(queue, (value, nxt))
        for b in sorted(ds):
            pair = frozenset((a, b))
            if b <= a or pair in existing_pairs or pair in route_pairs:
                continue
            path = [b]
            sections = []
            while path[-1] != a:
                n, id = previous[path[-1]]
                path.append(n)
                sections.append(id)
            path.reverse()
            sections.reverse()
            if len(sections) > 64:
                continue
            patterns.append(dict(id="route_" + a + "__" + b, sections=sections, cities=path, stops=path))
            route_pairs.add(pair)
    for network in manifest["networks"]:
        if not network.get("ordered"):
            continue
        stops = network["cities"]
        path = [stops[0]]
        sections = []
        for a, b in zip(stops, stops[1:]):
            ds = {a: 0}
            previous = {}
            queue = [(0, a)]
            while queue:
                cost, n = heapq.heappop(queue)
                if ds[n] != cost:
                    continue
                if n == b:
                    break
                for nxt, co in sorted(adjacency[n], key=lambda p: p[1]["id"]):
                    value = cost + co["alignments"][0]["minutes"]
                    if value < ds.get(nxt, 10**12):
                        ds[nxt] = value
                        previous[nxt] = (n, co["id"])
                        heapq.heappush(queue, (value, nxt))
            if b not in ds:
                raise ValueError("No ordered source path: " + network["id"] + " " + a + " " + b)
            segment = [b]
            ids = []
            while segment[-1] != a:
                prev, id = previous[segment[-1]]
                segment.append(prev)
                ids.append(id)
            path.extend(list(reversed(segment))[1:])
            sections.extend(reversed(ids))
        if len(set(path)) != len(path):
            raise ValueError("Ordered source path repeats a city: " + network["id"])
        if len(sections) == 1:
            continue  # Two-stop ordered service is the canonical direct corridor.
        patterns.append(
            dict(
                id="network_" + network["id"].replace("-", "_") + "_all_stops",
                sections=sections,
                cities=path,
                stops=stops,
            )
        )
    # Precompute future through-running over inherited track and construction
    # chains. New patterns require at least one unbuilt section, so none become
    # operational until every section opens. Stable existing patterns win ties.
    build_adj = collections.defaultdict(list)
    private = {c["id"] for c in rows if not c["existing"]}
    for co in rows:
        if co.get("shared_sections"):
            continue  # use actual physical resources
        for a, b in [(co["from_city"], co["to_city"]), (co["to_city"], co["from_city"])]:
            build_adj[a].append((b, co))
    for a in sorted(build_adj):
        ds = {a: 0}
        previous = {}
        queue = [(0, a)]
        while queue:
            cost, n = heapq.heappop(queue)
            if ds[n] != cost:
                continue
            for nxt, co in sorted(build_adj[n], key=lambda p: p[1]["id"]):
                value = cost + co["alignments"][0]["minutes"]
                if value < ds.get(nxt, 10**12):
                    ds[nxt] = value
                    previous[nxt] = (n, co["id"])
                    heapq.heappush(queue, (value, nxt))
        for b in sorted(ds):
            pair = frozenset((a, b))
            if b <= a or pair in route_pairs or pair in existing_pairs:
                continue
            path = [b]
            sections = []
            while path[-1] != a:
                n, id = previous[path[-1]]
                path.append(n)
                sections.append(id)
            if len(sections) < 2 or not any(id in private for id in sections):
                continue
            assert len(sections) <= 64, "Future service exceeds route bound"
            path.reverse()
            sections.reverse()
            patterns.append(dict(id="launch_route_" + a + "__" + b, sections=sections, cities=path, stops=path))
            route_pairs.add(pair)
    # Accepted service IDs keep their exact paths. Adding a shorter prospect
    # must not reroute/close an existing company's Houston service over unbuilt
    # track. Retain the original Dallas detour and name the new alternative.
    prior_patterns = json.loads(gzip.decompress((ROOT / "assets/scenario/public-network-v4.json.gz").read_bytes()))[
        "network"
    ]["routes"]
    by_pattern = {p["id"]: p for p in patterns}
    corridor_ids = {c["id"] for c in rows}
    for old in prior_patterns:
        assert set(old["sections"]) <= corridor_ids, "Regional update removes an accepted service resource"
        current = by_pattern.get(old["id"])
        if current == old:
            continue
        if current is not None:
            if "launch_71_houston__austin" in current["sections"]:
                current["id"] += "_regional"
                assert current["id"] not in by_pattern, "Duplicate regional alternative"
                by_pattern[current["id"]] = current
            else:
                current.clear()
                current.update(old)
                continue
        patterns.append(old)
        by_pattern[old["id"]] = old
    # Preserve every service accepted before this connector expansion. Offer a
    # new identity for changed shortcuts instead of redirecting active trains.
    accepted = json.loads(gzip.decompress((ROOT / "assets/scenario/public-network-v5.json.gz").read_bytes()))[
        "network"
    ]["routes"]
    by_pattern = {p["id"]: p for p in patterns}
    for old in accepted:
        assert set(old["sections"]) <= corridor_ids, "Connector update removes an accepted service resource"
        current = by_pattern.get(old["id"])
        if current == old:
            continue
        if current is not None:
            current["id"] += "_connector"
            assert current["id"] not in by_pattern, "Duplicate connector alternative"
            by_pattern[current["id"]] = current
        patterns.append(old)
        by_pattern[old["id"]] = old
    # Deliberate feeder patterns must remain available even where a public
    # detour or another unbuilt project wins the global shortest-path search.
    for service in launch.get("services", []):
        path = service["cities"]
        sections = []
        for a, b in zip(path, path[1:]):
            choices = [
                co for co in rows if not co.get("shared_sections") and {co["from_city"], co["to_city"]} == {a, b}
            ]
            assert choices, ("Missing connector service leg", service["id"], a, b)
            co = min(choices, key=lambda c: (not c["existing"], c["alignments"][0]["minutes"], c["id"]))
            sections.append(co["id"])
        assert service["id"] not in by_pattern, "Duplicate authored connector service"
        patterns.append(dict(id=service["id"], sections=sections, cities=path, stops=path))
    from pinned_services import add_services
    from rhine_randstad import repair_station_approaches

    rhine_source = "data/imports/rhine-randstad.v1.json"
    rhine = read(rhine_source)
    repair_station_approaches(rhine, rows)
    superseded = add_services(rhine, cities, market, stations, rows, patterns)
    from german_track import correct_montabaur, apply_majority_rule, FRANKFURT, COLOGNE

    correct_montabaur(rows, cities, km)
    profile_file = "data/imports/german-track.v1.json"
    profiles = read(profile_file)["profiles"]
    for co in rows:
        if co["id"] in (FRANKFURT, COLOGNE):
            co["alignments"][0]["connector_km"] = round(profiles[co["id"]]["meters"].get("conventional", 0) / 1000)
    rates = {p["id"]: round(p["baseline_million_2024_usd_per_route_km"] * 100_000_000)
             for p in read("data/costs/construction_costs.v1.json")["profiles"]}
    hamm_source = "data/imports/hamburg-hamm.v1.json"
    apply_majority_rule(rows, patterns, profiles, superseded, km, rates, read(hamm_source))
    from occitanie import add_occitanie
    occitanie_source = "data/imports/toulouse-beziers.v1.json"
    add_occitanie(read(occitanie_source), cities, market, stations, rows, patterns, superseded, km, rates)
    serbia_source = "data/imports/belgrade-vrbas.v1.json"
    superseded.extend(add_services(read(serbia_source), cities, market, stations, rows, patterns))
    assert len(patterns) <= 8192, "Future service catalog exceeds route bound"
    ms = list(market.values())
    matrix = []
    for a in ms:
        for b in ms:
            matrix.append(0 if a["id"] == b["id"] else max(1, round(km((a["lon"], a["lat"]), (b["lon"], b["lat"])))))
    rules = dict(
        id="prototype-world-network-v1",
        cities=ms,
        routes=patterns,
        superseded_routes=superseded,
        conventional_pairs=read("assets/scenario/network.json")["conventional_pairs"],
        public_paths=16,
        private_paths=12,
        station_calls=64,
        dwell_minutes=4,
        transfer_minutes=20,
        distances=[],
        distance_km=matrix,
        sparse=True,
    )
    output = dict(
        repair_source_sha256=repair_metadata["source_sha256"],
        connector_source_sha256=repair_metadata["connectors_sha256"],
        regional_links_sha256=repair_metadata["regional_links_sha256"],
        bridge_approaches_sha256=repair_metadata["bridge_approaches_sha256"],
        launch_authoring_sha256=launch["authoring_sha256"],
        launch_land_mask_sha256=launch["land_mask_sha256"],
        construction_cost_ledger_sha256=launch["cost_ledger_sha256"],
        rhine_randstad_sha256=output_hash(rhine_source),
        german_track_sha256=output_hash(profile_file),
        occitanie_sha256=output_hash(occitanie_source),
        hamburg_hamm_sha256=output_hash(hamm_source),
        belgrade_vrbas_sha256=output_hash(serbia_source),
        id="prototype-world-v10",
        cities=cities,
        network=rules,
        corridors=rows,
        stations=stations,
        lines=lines,
        endpoint_gaps=[
            dict(from_lon=coords[a][0], from_lat=coords[a][1], to_lon=coords[b][0], to_lat=coords[b][1], meters=d)
            for a, b, d in gaps
        ],
        source_sha256=hashlib.sha256((ROOT / "data/imports/hsr-prototype.v1.json").read_bytes()).hexdigest(),
    )
    from network_coverage import require

    output["operating_network_checks"] = require(output, manifest)
    # Generation-only graph fields must never leak into the portable catalog.
    for s in stations:
        s.pop("line", None)
        s.pop("point", None)
    emit("assets/scenario/public-network.json", output)
    # The game map shows infrastructure that supports an admitted city corridor.
    # Keep the full extract above for audits, not as disconnected map decoration.
    map_lines = []
    map_ids = []
    for co in rows:
        if not co["existing"] or co.get("retired"):
            continue
        a = co["alignments"][0]
        path = display.get(co["id"], {}).get(a["id"], a["path"])
        map_ids.append(co["id"])
        map_lines.append([[p["lon"], p["lat"]] for p in path])
    map_shapes = [LineString(p) for p in map_lines]
    map_tree = STRtree(map_shapes)
    map_stops = []
    for station in stations:
        p = Point(station["lon"], station["lat"])
        for i in map_tree.query(p.buffer(0.012)):
            q = map_shapes[i].interpolate(map_shapes[i].project(p))
            if km((p.x, p.y), (q.x, q.y)) <= 0.4:
                map_stops.append(station["id"])
                break
    map_data = dict(
        source_catalog_sha256=output_hash("assets/scenario/public-network.json"),
        corridor_ids=map_ids,
        lines=map_lines,
        station_ids=map_stops,
    )
    from network_coverage import require_map

    require_map(output, map_data, manifest)
    emit("assets/scenario/playable-rail-map.json", map_data)
    print("Playable map:", len(map_lines), "corridor paths,", len(map_stops), "source stations")

    print(
        json.dumps(
            dict(
                cities=len(cities),
                stations=len(stations),
                mapped_stations=len(station_node),
                source_lines=len(lines),
                graph_components=len(comp_nodes),
                city_pins=len(city_pins),
                corridors=len(rows),
                patterns=len(patterns),
                endpoint_gaps=len(gaps),
                max_points=max(len(c["alignments"][0]["path"]) for c in out),
            ),
            indent=2,
        )
    )


if __name__ == "__main__":
    main()
    flush()
