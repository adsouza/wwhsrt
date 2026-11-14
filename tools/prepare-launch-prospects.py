"""Compile the authored launch game prospects; offline, deterministic, no save access.

The land/lake mask is a cartographic screen. Authored terrain quantities and
crossings are game design inputs, not surveyed alignments or procurement costs.
"""

import gzip, hashlib, heapq, json, math, pathlib, sys
from shapely import prepare
from shapely.geometry import shape, Point, LineString, box, mapping
from shapely.ops import unary_union, nearest_points, transform
from shapely import STRtree

ROOT = pathlib.Path(__file__).resolve().parents[1]


def read(p):
    return json.loads((ROOT / p).read_bytes())


def sha(p):
    return hashlib.sha256((ROOT / p).read_bytes()).hexdigest()


def km(a, b):
    x, y, z, t = map(math.radians, (*a, *b))
    return 12742 * math.asin(
        min(1, math.sqrt(math.sin((t - y) / 2) ** 2 + math.cos(y) * math.cos(t) * math.sin((z - x) / 2) ** 2))
    )


def length(points):
    return sum(km(a, b) for a, b in zip(points, points[1:]))


def allocate(n, weights):
    raw = [n * w / sum(weights) for w in weights]
    out = [int(v) for v in raw]
    for i in sorted(range(len(out)), key=lambda i: (-(raw[i] - out[i]), i))[: n - sum(out)]:
        out[i] += 1
    return out


def project(x, y, z=None):
    # Also accepts vector coordinates through shapely.ops.transform.
    try:
        return tuple(project(a, b)[0] for a, b in zip(x, y)), tuple(project(a, b)[1] for a, b in zip(x, y))
    except TypeError:
        pass
    t = math.asin(math.sqrt(3) / 2 * math.sin(math.radians(y)))
    t2 = t * t
    t6 = t**6
    return 2 * math.sqrt(3) * math.radians(x) * math.cos(t) / (
        3 * (1.340264 - 3 * 0.081106 * t2 + t6 * (7 * 0.000893 + 9 * 0.003796 * t2))
    ), t * (1.340264 - 0.081106 * t2 + t6 * (0.000893 + 0.003796 * t2))


def main():
    input_shas = {
        p: sha(p)
        for p in [
            "data/construction/launch-prospects.v1.json",
            "data/construction/route-land.v1.json.gz",
            "data/costs/construction_costs.v1.json",
            "data/imports/operating-networks.v2.json",
            "assets/scenario/public-network-v5.json.gz",
        ]
    }
    authored = read("data/construction/launch-prospects.v1.json")
    base_bytes = gzip.decompress((ROOT / "assets/scenario/public-network-v3.json.gz").read_bytes())
    base = json.loads(base_bytes)
    cities = {c["id"]: dict(c) for c in base["cities"]}
    accepted = {
        c["id"]: c
        for c in json.loads(gzip.decompress((ROOT / "assets/scenario/public-network-v5.json.gz").read_bytes()))[
            "corridors"
        ]
        if not c["existing"]
    }
    # New junction markets are pinned in the operating manifest, shared with the
    # existing-rail importer. Construction endpoints must use those same anchors.
    from rail_source import catalog

    operating, manifest = catalog()
    station_hubs = {h["city_id"] for h in manifest["hubs"]}
    for c in operating["cities"]:
        if c["id"] not in cities:
            cities[c["id"]] = dict(c)
        elif c["id"] in station_hubs:
            cities[c["id"]].update({k: c[k] for k in ["station_name", "station_lon", "station_lat"]})
    mask = json.loads(gzip.decompress((ROOT / "data/construction/route-land.v1.json.gz").read_bytes()))
    land = shape(mask["land"])
    prepare(land)
    projected_land = transform(project, land)
    prepare(projected_land)
    countries = mask["countries"]
    country_shapes = [shape(c["geometry"]) for c in countries]
    country_tree = STRtree(country_shapes)
    profiles = {p["id"]: p for p in read("data/costs/construction_costs.v1.json")["profiles"]}
    overrides = authored.get("terminals", [])
    for c in overrides:
        cities[c["id"]].update(c)

    def point(id):
        c = cities[id]
        return (c.get("station_lon", c["lon"]), c.get("station_lat", c["lat"]))

    def solve(a, b, ground, projected_ground):
        def covers(a, b):
            return ground.covers(LineString([a, b])) and projected_ground.covers(LineString([project(*a), project(*b)]))

        if covers(a, b):
            return [a, b]
        # Visibility-constrained A*: edges must stay on admitted land/crossing geometry,
        # not merely have land at their endpoints. No diagonal ocean shortcuts.
        for padding, step in [(0.28, 0.025), (0.65, 0.025), (1.2, 0.04)] + ([(0.3, 0.004)] if km(a, b) < 100 else []):
            w, e = min(a[0], b[0]) - padding, max(a[0], b[0]) + padding
            s, n = min(a[1], b[1]) - padding, max(a[1], b[1]) + padding
            nx, ny = math.ceil((e - w) / step) + 1, math.ceil((n - s) / step) + 1
            pos = lambda q: (w + q[0] * step, s + q[1] * step)
            cache = {}

            def ok(q):
                if q not in cache:
                    cache[q] = 0 <= q[0] < nx and 0 <= q[1] < ny and ground.covers(Point(pos(q)))
                return cache[q]

            def anchor(p):
                q = (round((p[0] - w) / step), round((p[1] - s) / step))
                found = []
                for dx in range(-4, 5):
                    for dy in range(-4, 5):
                        v = q[0] + dx, q[1] + dy
                        if ok(v) and covers(p, pos(v)):
                            found.append(v)
                return sorted(found, key=lambda v: km(p, pos(v)))[:4]

            starts, ends = anchor(a), set(anchor(b))
            dist = {q: km(a, pos(q)) for q in starts}
            prev = {}
            queue = [(v + km(pos(q), b), v, q) for q, v in dist.items()]
            heapq.heapify(queue)
            end = None
            while queue:
                _, cost, q = heapq.heappop(queue)
                if cost != dist[q]:
                    continue
                if q in ends:
                    end = q
                    break
                for dx, dy in [(-1, -1), (-1, 0), (-1, 1), (0, -1), (0, 1), (1, -1), (1, 0), (1, 1)]:
                    v = q[0] + dx, q[1] + dy
                    if not ok(v) or not covers(pos(q), pos(v)):
                        continue
                    value = cost + km(pos(q), pos(v))
                    if value < dist.get(v, 1e20):
                        dist[v] = value
                        prev[v] = q
                        heapq.heappush(queue, (value + km(pos(v), b), value, v))
            if end is not None:
                out = [end]
                while out[-1] in prev:
                    out.append(prev[out[-1]])
                return [a, *[pos(q) for q in reversed(out)], b]
        raise ValueError("No land path: " + repr((a, b)))

    result = []
    reviews = []
    cache_file = ROOT / "build/prospect-catalog/geometry-cache.json"
    cache_file.parent.mkdir(parents=True, exist_ok=True)
    cache = json.loads(cache_file.read_bytes()) if cache_file.exists() else {}
    algorithm_sha = sha("tools/prepare-launch-prospects.py")
    for row in authored["pairs"]:
        if "legacy_id" in row:
            continue
        points = [point(row["from_city"]), *[tuple(p) for p in row["via"]], point(row["to_city"])]
        # A via point is a route control, not a city or station: author it on land.
        for p in points[1:-1]:
            assert land.covers(Point(p)) or any(
                shape(c["geometry"]).covers(Point(p)) for c in row.get("crossings", [])
            ), ("Via point in water", row["number"], p)
        crossings = row.get("crossings", [])
        allowed = unary_union([shape(c["geometry"]) for c in crossings]) if crossings else None
        ground = land if allowed is None else land.union(allowed)
        prepare(ground)
        projected_ground = projected_land if allowed is None else projected_land.union(transform(project, allowed))
        prepare(projected_ground)
        for p in [points[0], points[-1]]:
            assert ground.covers(Point(p)), ("Terminal outside land/mapped adapter", row["number"], p)
        cache_key = hashlib.sha256(
            json.dumps(
                [row, points, sha("data/construction/route-land.v1.json.gz"), algorithm_sha], sort_keys=True
            ).encode()
        ).hexdigest()
        if cache_key in cache:
            path = cache[cache_key]
        else:
            # Preserve authored controls where they encode intended mountain detours.
            path = []
            for a, b in zip(points, points[1:]):
                part = solve(a, b, ground, projected_ground)
                reduced = [part[0]]
                i = 0
                while i < len(part) - 1:
                    nxt = next(
                        j
                        for j in range(len(part) - 1, i, -1)
                        if ground.covers(LineString([part[i], part[j]]))
                        and projected_ground.covers(LineString([project(*part[i]), project(*part[j])]))
                    )
                    reduced.append(part[nxt])
                    i = nxt
                path.extend(reduced if not path else reduced[1:])
            cache[cache_key] = path
            cache_file.write_text(json.dumps(cache, separators=(",", ":")))
        assert ground.covers(LineString(path)) and projected_ground.covers(
            LineString([project(*p) for p in path])
        ), "Invalid cached route"
        assert len(path) <= 64, ("Too many reviewed vertices", row["number"], len(path))
        rail_km = math.ceil(length(path) * 1.08)
        h, v, t = allocate(
            rail_km,
            [
                row["hill_percent"],
                row["viaduct_percent"],
                row["tunnel_percent"],
                100 - row["hill_percent"] - row["viaduct_percent"] - row["tunnel_percent"],
            ],
        )[:3]
        surface = rail_km - h - v - t
        line = LineString(path)
        shares = []
        specific = row.get("country_profiles", {})
        for ix in sorted(country_tree.query(line)):
            section = line.intersection(country_shapes[ix])
            pieces = list(section.geoms) if hasattr(section, "geoms") else [section]
            distance = sum(length(list(p.coords)) for p in pieces if p.geom_type == "LineString")
            if distance > 0.01:
                country = {"United States of America": "United States"}.get(
                    countries[ix]["name"], countries[ix]["name"]
                )
                shares.append([country, specific.get(country, row["profile"]), distance])
        assert shares, ("No country cost assignment", row["number"])
        if specific:
            assert all(c in specific for c, p, v in shares), ("Unexpected crossed jurisdiction", row["number"], shares)
        jurisdictions = shares
        parts = [
            dict(
                country=c,
                profile_id=p,
                surface_km=0,
                hill_km=0,
                viaduct_km=0,
                tunnel_km=0,
                base_per_km=int(profiles[p]["baseline_million_2024_usd_per_route_km"] * 100_000_000),
                hill_factor_percent=row.get("hill_factor_percent", 150),
                tunnel_factor=row.get("tunnel_factor", 5),
            )
            for c, p, _ in jurisdictions
        ]
        # Allocate every work quantity by measured country intersections, exactly once.
        # This is a strategic apportionment, not a surveyed chainage/terrain schedule.
        for key, count in [("surface_km", surface), ("hill_km", h), ("viaduct_km", v), ("tunnel_km", t)]:
            for part, n in zip(parts, allocate(count, [j[2] for j in jurisdictions])):
                part[key] = n
        mins = row.get("minutes", math.ceil(rail_km * 60 / 250) + 8)
        alt = dict(
            id="launch",
            km=rail_km,
            minutes=mins,
            months=row["months"],
            surface_km=surface,
            hill_km=h,
            viaduct_km=v,
            tunnel_km=t,
            path=[dict(lon=p[0], lat=p[1]) for p in path],
            cost_intervals=parts,
        )
        # Station approach repairs must not change an accepted construction contract.
        # Retain its published engineering quantities, prices and opening schedule.
        if row["id"] in accepted:
            saved = accepted[row["id"]]["alignments"][0]
            alt = {**saved, "path": alt["path"]}
            assert length(path) <= alt["km"], "Reanchored path exceeds accepted priced length"
        co = dict(
            id=row["id"],
            from_city=row["from_city"],
            to_city=row["to_city"],
            existing=False,
            daily_market=5000,
            outside_cost=15000,
            construction_kind=row["kind"],
            construction_note=row["barrier"],
            alignments=[alt],
        )
        if row.get("parallel_to"):
            co["parallel_to"] = row["parallel_to"]
        result.append(co)
        reviews.append(
            dict(
                number=row["number"],
                corridor_id=co["id"],
                vertices=len(path),
                drawn_km=round(length(path), 3),
                priced_km=alt["km"],
                water_km=round(
                    length(list(LineString(path).difference(land).coords))
                    if LineString(path).difference(land).geom_type == "LineString"
                    else sum(
                        length(list(s.coords))
                        for s in getattr(LineString(path).difference(land), "geoms", [])
                        if s.geom_type == "LineString"
                    ),
                    3,
                ),
                crossings=crossings,
            )
        )
        print(row["number"], len(path), "vertices", rail_km, "km", flush=True)
    assert all(sha(p) == value for p, value in input_shas.items()), "Inputs changed during compilation"
    output = dict(
        version=1,
        source_catalog_sha256=hashlib.sha256(base_bytes).hexdigest(),
        authoring_sha256=sha("data/construction/launch-prospects.v1.json"),
        land_mask_sha256=sha("data/construction/route-land.v1.json.gz"),
        cost_ledger_sha256=sha("data/costs/construction_costs.v1.json"),
        operating_manifest_sha256=sha("data/imports/operating-networks.v2.json"),
        accepted_catalog_sha256=hashlib.sha256(
            gzip.decompress((ROOT / "assets/scenario/public-network-v5.json.gz").read_bytes())
        ).hexdigest(),
        terminals=overrides,
        corridors=result,
        review=reviews,
        services=authored.get("services", []),
    )
    target = ROOT / "data/construction/compiled-prospects.v1.json"
    data = (json.dumps(output, ensure_ascii=False, separators=(",", ":")) + "\n").encode()
    if "--check" in sys.argv:
        assert target.read_bytes() == data, "Prospect compilation differs"
    else:
        target.write_bytes(data)
    print("Compiled", len(result), "additional prospects; two legacy Japanese projects retained")


if __name__ == "__main__":
    main()
