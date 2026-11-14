"""Merge source-vertex repairs while retaining every accepted service identity."""

import copy, hashlib, heapq, json, pathlib, gzip

ROOT = pathlib.Path(__file__).resolve().parents[1]


def km(a, b):
    import math

    x, y, z, t = map(math.radians, (*a, *b))
    return 12742 * math.asin(
        min(1, math.sqrt(math.sin((t - y) / 2) ** 2 + math.cos(y) * math.cos(t) * math.sin((z - x) / 2) ** 2))
    )


def merge(rows):
    patch = json.loads((ROOT / "data/imports/network-repairs.v2.json").read_bytes())
    assert (
        patch["base_catalog_sha256"]
        == hashlib.sha256(
            gzip.decompress((ROOT / "assets/scenario/public-network-v2.json.gz").read_bytes())
        ).hexdigest()
    )
    assert (
        patch["source_sha256"]
        == hashlib.sha256((ROOT / "data/imports/rail-topology-source.v1.jsonl.gz").read_bytes()).hexdigest()
    )
    prior = json.loads(gzip.decompress((ROOT / "assets/scenario/public-network-v2.json.gz").read_bytes()))
    assert (
        patch["connectors_sha256"]
        == hashlib.sha256((ROOT / "data/imports/rail-connectors.v2.jsonl.gz").read_bytes()).hexdigest()
    )
    assert (
        patch["regional_links_sha256"]
        == hashlib.sha256((ROOT / "data/imports/rail-regional-links.v1.jsonl.gz").read_bytes()).hexdigest()
    )
    assert (
        patch["bridge_approaches_sha256"]
        == hashlib.sha256((ROOT / "data/imports/rail-bridge-approaches.v1.jsonl.gz").read_bytes()).hexdigest()
    )
    assert all(c["connected"] for c in patch["network_checks"])
    assert (
        patch["manifest_sha256"]
        == hashlib.sha256((ROOT / "data/imports/operating-networks.v2.json").read_bytes()).hexdigest()
    )
    old = {frozenset((co["from_city"], co["to_city"])): co for co in prior["corridors"] if co["existing"]}
    physical = []
    cities = {c["id"]: c for c in patch["cities"]}
    for co in copy.deepcopy(patch["corridors"]):
        saved = old.get(frozenset((co["from_city"], co["to_city"])))
        chord = km(
            (cities[co["from_city"]]["lon"], cities[co["from_city"]]["lat"]),
            (cities[co["to_city"]]["lon"], cities[co["to_city"]]["lat"]),
        )
        if co["alignments"][0]["km"] > 2000 or co["alignments"][0]["km"] > chord * 2.5 + 30:
            continue
        if saved:
            co["id"] = saved["id"]
            co["alignments"][0]["id"] = saved["alignments"][0]["id"]
            if co["from_city"] != saved["from_city"]:
                co["from_city"], co["to_city"] = co["to_city"], co["from_city"]
                co["alignments"][0]["path"].reverse()
        physical.append(co)
    adj = {}
    for co in physical:
        for a, b in [(co["from_city"], co["to_city"]), (co["to_city"], co["from_city"])]:
            adj.setdefault(a, []).append((b, co))
    byid = {c["id"]: c for c in physical}
    aliases = []
    retained = []
    for original in prior["corridors"]:
        if not original["existing"] or original["id"] in byid:
            continue
        a, b = original["from_city"], original["to_city"]
        ds = {a: 0}
        prev = {}
        q = [(0, a)]
        while q:
            cost, n = heapq.heappop(q)
            if ds[n] != cost:
                continue
            if n == b:
                break
            for nxt, co in sorted(adj.get(n, []), key=lambda x: x[1]["id"]):
                v = cost + co["alignments"][0]["minutes"]
                if v < ds.get(nxt, 10**12):
                    ds[nxt] = v
                    prev[nxt] = (n, co)
                    heapq.heappush(q, (v, nxt))
        if b not in ds:
            # Do not erase separately qualified old source corridors outside repair regions.
            retained.append(copy.deepcopy(original))
            continue
        parts = []
        n = b
        while n != a:
            p, co = prev[n]
            parts.append((p, co))
            n = p
        parts.reverse()
        chord = km(
            (cities[a]["station_lon"], cities[a]["station_lat"]), (cities[b]["station_lon"], cities[b]["station_lat"])
        )
        if sum(part["alignments"][0]["km"] for p, part in parts) > chord * 2.5 + 30:
            retained.append(copy.deepcopy(original))
            continue
        co = copy.deepcopy(original)
        shape = []
        al = co["alignments"][0]
        al.update(km=0, minutes=0, surface_km=0, connector_km=0)
        co["shared_sections"] = []
        for p, part in parts:
            co["shared_sections"].append(part["id"])
            alt = part["alignments"][0]
            path = alt["path"] if part["from_city"] == p else list(reversed(alt["path"]))
            shape.extend(path if not shape else path[1:])
            for k in ["km", "minutes", "surface_km", "connector_km"]:
                al[k] += alt.get(k, 0)
        al["path"] = shape
        aliases.append(co)
    patch["physical_corridors"] = copy.deepcopy(physical)
    # Keep published gameplay lengths/timings for accepted IDs. The repair adds
    # source paths/resources, not retroactive changes to leased-fleet requirements.
    for co in physical + aliases + retained:
        saved = next((c for c in prior["corridors"] if c["id"] == co["id"]), None)
        if saved and "ulsan" not in (co["from_city"], co["to_city"]):
            path = co["alignments"][0]["path"]
            connector = min(saved["alignments"][0]["km"], co["alignments"][0].get("connector_km", 0))
            co["alignments"] = copy.deepcopy(saved["alignments"])
            co["alignments"][0]["path"] = path
            if connector:
                co["alignments"][0]["connector_km"] = connector
    for co in physical + aliases + retained:
        frozen = json.loads((ROOT / "data/imports/rail-display-paths.v1.json").read_bytes()).get(co["id"], {})
        for a in co["alignments"]:
            if (
                a["path"]
                == next(
                    (
                        old["path"]
                        for c in prior["corridors"]
                        if c["id"] == co["id"]
                        for old in c["alignments"]
                        if old["id"] == a["id"]
                    ),
                    None,
                )
                and a["id"] in frozen
            ):
                a["path"] = frozen[a["id"]]
    allrows = physical + aliases + retained
    byid = {c["id"]: c for c in allrows}

    def leaves(id, seen):
        assert id not in seen, "Cyclic capacity alias: " + id
        row = byid[id]
        return (
            [id]
            if not row.get("shared_sections")
            else [leaf for child in row["shared_sections"] for leaf in leaves(child, seen | {id})]
        )

    for co in allrows:
        if co.get("shared_sections"):
            co["shared_sections"] = list(
                dict.fromkeys(leaf for id in co["shared_sections"] for leaf in leaves(id, {co["id"]}))
            )
    # Existing legacy express resource definitions are replaced by the same source
    # path decomposition as all other aliases; no nested/self resource aliases.
    withdrawn = set(
        json.loads((ROOT / "data/imports/operating-networks.v2.json").read_bytes()).get("non_hsr_connectors", [])
    )
    patch["physical_corridors"] = [c for c in patch["physical_corridors"] if c["id"] not in withdrawn]
    return [copy.deepcopy(c) for c in rows if not c["existing"]] + [
        c for c in physical + aliases + retained if c["id"] not in withdrawn
    ], patch
