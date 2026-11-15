"""The worldwide basemap's rule: detailed coast near a line, coarse coast elsewhere.

The basemap is one land outline. Within NEAR_KM of any line it follows the source to FINE degrees;
farther away to COARSE degrees. Small far islands and small or far lakes are left out. The tool that
applies the rule is tools/prepare-basemap.py; the checks that read its output share drawn_geometry.
"""

import math

import numpy as np
from scipy.spatial import cKDTree
from shapely import STRtree
from shapely.geometry import LineString, Polygon
from shapely.ops import unary_union

NEAR_KM = 150
FINE = 0.003
COARSE = 0.05
FAR_ISLAND_KM = 20
NEAR_LAKE_KM = 5
FAR_LAKE_KM = 100
# Tolerances tried in turn where FINE would put a buildable route offshore.
LADDER = [0.003, 0.001, 0.0003, 0.0001, 0]
MAX_POINTS = 80000
MAX_OUTLINES = 2200
# The Hokkaido and Kansai relief boxes (zoom, x0, y0, x1, y1). Relief there is detailed, so the coast is too.
JAPAN_BOXES = [(8, 227, 93, 228, 95), (8, 224, 100, 225, 101)]
EARTH_KM = 6371.0
SAMPLE_KM = 10


def tile_bounds(z, x0, y0, x1, y1):
    """West, south, east and north of a block of Web Mercator tiles."""
    n = 2**z

    def lat(y):
        return math.degrees(math.atan(math.sinh(math.pi * (1 - 2 * y / n))))

    return x0 / n * 360 - 180, lat(y1 + 1), (x1 + 1) / n * 360 - 180, lat(y0)


def project(x, y, z=None):
    """Equal Earth, as pkg/mapview projects it. Accepts scalars or shapely's coordinate arrays."""
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


def km_between(a, b):
    lon1, lat1, lon2, lat2 = map(math.radians, (a[0], a[1], b[0], b[1]))
    v = math.sin((lat2 - lat1) / 2) ** 2 + math.cos(lat1) * math.cos(lat2) * math.sin((lon2 - lon1) / 2) ** 2
    return 2 * EARTH_KM * math.asin(min(1, math.sqrt(v)))


def sample(path, step_km=SAMPLE_KM):
    """Points along a path no more than step_km apart, including its vertices."""
    points = [tuple(path[0][:2])] if path else []
    for a, b in zip(path, path[1:]):
        steps = max(1, math.ceil(km_between(a, b) / step_km))
        points += [(a[0] + (b[0] - a[0]) * i / steps, a[1] + (b[1] - a[1]) * i / steps) for i in range(1, steps + 1)]
    return points


def _unit(points):
    lon, lat = np.radians(np.asarray(points, dtype=float).reshape(-1, 2)).T
    return np.column_stack([np.cos(lat) * np.cos(lon), np.cos(lat) * np.sin(lon), np.sin(lat)])


class NearIndex:
    """Answers whether a point is within km of any path, or inside any box."""

    def __init__(self, paths, boxes, km=NEAR_KM):
        samples = [p for path in paths for p in sample(path)]
        self.tree = cKDTree(_unit(samples)) if samples else None
        self.chord = 2 * math.sin(km / EARTH_KM / 2)
        self.boxes = list(boxes)

    def flags(self, ring):
        points = np.asarray([p[:2] for p in ring], dtype=float)
        near = np.zeros(len(points), dtype=bool)
        for west, south, east, north in self.boxes:
            near |= (points[:, 0] >= west) & (points[:, 0] <= east) & (points[:, 1] >= south) & (points[:, 1] <= north)
        if self.tree is not None and len(points):
            distance, _ = self.tree.query(_unit(points), distance_upper_bound=self.chord)
            near |= np.isfinite(distance)
        return [bool(v) for v in near]

    def near(self, lon, lat):
        return self.flags([(lon, lat)])[0]


def across_km(ring):
    """Great-circle length of the ring's bounding-box diagonal."""
    lons, lats = [p[0] for p in ring], [p[1] for p in ring]
    return km_between((min(lons), min(lats)), (max(lons), max(lats)))


def signed_area(ring):
    """Positive for a counter-clockwise ring."""
    return sum(a[0] * b[1] - b[0] * a[1] for a, b in zip(ring, ring[1:])) / 2


def _simplify_run(points, tolerance):
    if tolerance <= 0 or len(points) < 3:
        return list(points)
    return [tuple(p) for p in LineString(points).simplify(tolerance, preserve_topology=False).coords]


def _joined(points, flags, fine, coarse):
    n = len(points)
    if all(flags) or not any(flags):
        tolerance = fine if flags[0] else coarse
        closed = points + [points[0]]
        if tolerance <= 0:
            return closed
        return [tuple(p) for p in Polygon(closed).simplify(tolerance, preserve_topology=True).exterior.coords]
    # Start where the detail changes, so every run of equal flags is simplified as one open line. A run
    # ends on the first point of the next, and simplifying keeps end points, so the runs still join.
    start = next(i for i in range(n) if flags[i] != flags[i - 1])
    order = [(start + k) % n for k in range(n)] + [start]
    out, run = [], [order[0]]
    for index in order[1:]:
        run.append(index)
        if index == start or flags[index] != flags[run[0]]:
            out += _simplify_run([points[i] for i in run], fine if flags[run[0]] else coarse)[:-1]
            run = [index]
    return out + [out[0]]


def simplify_ring(ring, flags, fine=FINE, coarse=COARSE):
    """Simplify a closed ring at fine where flags are true and at coarse elsewhere."""
    points = [tuple(p[:2]) for p in ring]
    flags = list(flags)
    if points[0] == points[-1]:
        points, flags = points[:-1], flags[: len(points) - 1]
    tolerance = coarse
    while True:
        out = _joined(points, flags, fine, tolerance)
        if len(out) >= 4 and Polygon(out).is_valid:
            return out
        if tolerance <= fine:
            return points + [points[0]]
        tolerance = max(fine, tolerance / 2)


def _wound(ring, counter_clockwise):
    ring = [[round(p[0], 6), round(p[1], 6)] for p in ring]
    return ring if (signed_area(ring) > 0) == counter_clockwise else ring[::-1]


def compose(polygons, index, fine_for=None):
    """Apply the rule to a MultiPolygon's coordinates.

    Returns land, lakes and lake_islands as closed rings of [lon, lat]: land and lake islands
    counter-clockwise, lakes clockwise. sources["land"][i] is the source polygon of land[i].
    fine_for maps a source polygon to a tighter fine tolerance for it and its lakes.
    """
    fine_for = fine_for or {}
    holes, owners = [], []
    for i, polygon in enumerate(polygons):
        for h, hole in enumerate(polygon[1:]):
            holes.append(Polygon(hole))
            owners.append((i, h))
    tree = STRtree(holes) if holes else None

    def lake_around(i):
        if tree is None:
            return None
        point = Polygon(polygons[i][0]).representative_point()
        hits = [owners[k] for k in tree.query(point, predicate="within") if owners[k][0] != i]
        return hits[0] if hits else None

    def kept_lake(i, h, exterior):
        hole = polygons[i][1 + h]
        flags = index.flags(hole)
        near = any(flags)
        if across_km(hole) < (NEAR_LAKE_KM if near else FAR_LAKE_KM):
            return None
        fine = fine_for.get(i, FINE)
        for ring in (simplify_ring(hole, flags, fine), simplify_ring(hole, flags, fine, fine), list(map(tuple, hole))):
            if Polygon(ring).is_valid and Polygon(ring).within(exterior):
                return ring
        return None

    land, lakes, islands, source_land = [], [], [], []
    left_out_islands = left_out_lakes = 0
    lake_rings, pending = {}, []
    for i, polygon in enumerate(polygons):
        if lake_around(i) is not None:
            pending.append(i)
            continue
        flags = index.flags(polygon[0])
        if not any(flags) and across_km(polygon[0]) < FAR_ISLAND_KM:
            left_out_islands += 1
            left_out_lakes += len(polygon) - 1
            continue
        exterior = simplify_ring(polygon[0], flags, fine_for.get(i, FINE))
        land.append(_wound(exterior, True))
        source_land.append(i)
        for h in range(len(polygon) - 1):
            ring = kept_lake(i, h, Polygon(exterior))
            if ring is None:
                left_out_lakes += 1
                continue
            lake_rings[(i, h)] = Polygon(ring)
            lakes.append(_wound(ring, False))
    for i in pending:
        polygon, lake = polygons[i], lake_rings.get(lake_around(i))
        left_out_lakes += len(polygon) - 1  # a lake on an island in a lake is not drawn
        if lake is None:
            continue  # its lake was left out, so the island is already part of the land around it
        flags = index.flags(polygon[0])
        fine = fine_for.get(i, FINE)
        for ring in (simplify_ring(polygon[0], flags, fine), list(map(tuple, polygon[0]))):
            if Polygon(ring).within(lake):
                islands.append(_wound(ring, True))
                break
    return {
        "land": land,
        "lakes": lakes,
        "lake_islands": islands,
        "sources": {"land": source_land, "left_out_islands": left_out_islands, "left_out_lakes": left_out_lakes},
    }


def drawn_geometry(basemap):
    """The land the renderer draws: land outlines, minus lakes, plus the islands inside those lakes."""

    def union(rings):
        return unary_union([Polygon(r).buffer(0) for r in rings])

    return union(basemap["land"]).difference(union(basemap["lakes"])).union(union(basemap["lake_islands"]))
