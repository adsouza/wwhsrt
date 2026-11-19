"""The worldwide basemap's rule: detailed coast near a line, coarse coast elsewhere.

The basemap is one land outline. Within NEAR_KM of any line it follows the source to FINE degrees;
farther away to COARSE degrees. Small far islands and small or far lakes are left out. The tool that
applies the rule is tools/prepare-basemap.py; the checks that read its output share drawn_geometry.
"""

import math

import numpy as np
import shapely
from scipy.spatial import cKDTree
from shapely import STRtree
from shapely.geometry import LineString, Polygon, box, shape
from shapely.ops import transform, unary_union

NEAR_KM = 150
FINE = 0.003
# Between NEAR_KM and MID_KM the coast follows the source to MID degrees (about 1.4 km). A view of a
# line at regional zoom shows coast this far away: with nothing between FINE and COARSE, eastern
# Hokkaido, 150 to 350 km from Sapporo, was drawn with edges 70 km long. Of the rules measured this
# one bought the most for its points; finer steps out to 1,200 km cost twice as many for no more.
MID_KM = 400
MID = 0.0125
COARSE = 0.05
FAR_ISLAND_KM = 25
NEAR_LAKE_KM = 5
FAR_LAKE_KM = 100
# Tolerances tried in turn where FINE would put a buildable route offshore.
LADDER = [0.003, 0.001, 0.0003, 0.0001, 0]
MAX_POINTS = 93000
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


def on_map(geometry):
    """The geometry as the map draws it: Equal Earth, with straight edges between projected points.

    A long edge is a curve on the globe but a straight chord on the map, so a coarse outline that is
    valid in degrees can cross itself once projected (Antarctica's does). make_valid keeps the area.
    """
    return shapely.make_valid(transform(project, geometry))


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
    """Answers how far points are from any path, in the rule's terms. Inside a box counts as on a path."""

    def __init__(self, paths, boxes, km=NEAR_KM, mid_km=MID_KM):
        samples = [p for path in paths for p in sample(path)]
        self.tree = cKDTree(_unit(samples)) if samples else None
        self.chord = 2 * math.sin(km / EARTH_KM / 2)
        self.mid_chord = 2 * math.sin(mid_km / EARTH_KM / 2)
        self.boxes = list(boxes)

    def _within(self, ring):
        """For each point: within NEAR_KM, and within MID_KM."""
        points = np.asarray([p[:2] for p in ring], dtype=float)
        near = np.zeros(len(points), dtype=bool)
        for west, south, east, north in self.boxes:
            near |= (points[:, 0] >= west) & (points[:, 0] <= east) & (points[:, 1] >= south) & (points[:, 1] <= north)
        middle = near.copy()
        if self.tree is not None and len(points):
            distance, _ = self.tree.query(_unit(points), distance_upper_bound=self.mid_chord)
            near |= distance <= self.chord
            middle |= np.isfinite(distance)
        return near, middle

    def flags(self, ring):
        """Whether each point is near a line: within NEAR_KM."""
        return [bool(v) for v in self._within(ring)[0]]

    def tolerances(self, ring, fine=FINE):
        """The tolerance for each point: fine near a line, MID out to MID_KM, COARSE beyond."""
        near, middle = self._within(ring)
        return [fine if n else MID if m else COARSE for n, m in zip(near, middle)]

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
    # Topology-preserving, so a run cannot cross itself: plain Douglas-Peucker does in fjords and deltas,
    # and one crossing anywhere would make a whole continent's outline invalid.
    return [tuple(p) for p in LineString(points).simplify(tolerance, preserve_topology=True).coords]


def _joined(points, tolerances):
    n = len(points)
    if len(set(tolerances)) == 1:
        closed = points + [points[0]]
        if tolerances[0] <= 0:
            return closed
        return [tuple(p) for p in Polygon(closed).simplify(tolerances[0], preserve_topology=True).exterior.coords]
    # Start where the detail changes, so every run of one tolerance is simplified as one open line. A
    # run ends on the first point of the next, and simplifying keeps end points, so the runs still join.
    start = next(i for i in range(n) if tolerances[i] != tolerances[i - 1])
    order = [(start + k) % n for k in range(n)] + [start]
    out, run = [], [order[0]]
    for index in order[1:]:
        run.append(index)
        if index == start or tolerances[index] != tolerances[run[0]]:
            out += _simplify_run([points[i] for i in run], tolerances[run[0]])[:-1]
            run = [index]
    return out + [out[0]]


def _open(ring, detail, fine, coarse):
    """The ring without its closing point, and a tolerance for each remaining point.

    detail gives, per point, either a tolerance or a flag: true for fine, false for coarse.
    """
    points = [tuple(p[:2]) for p in ring]
    tolerances = [(fine if d else coarse) if isinstance(d, bool) else float(d) for d in detail]
    if points[0] == points[-1]:
        points, tolerances = points[:-1], tolerances[: len(points) - 1]
    return points, tolerances


def simplify_ring(ring, detail, fine=FINE, coarse=COARSE):
    """Simplify a closed ring to the tolerance given for each point (or fine and coarse by flag)."""
    points, tolerances = _open(ring, detail, fine, coarse)
    finest, cap = min(tolerances), max(tolerances)
    while True:
        out = _joined(points, [min(t, cap) for t in tolerances])
        if len(out) >= 4 and Polygon(out).is_valid:
            return out
        if cap <= finest:
            # Two runs crossed each other. Simplify the ring whole at its finest tolerance, which cannot.
            closed = points + [points[0]]
            if finest <= 0:
                return closed
            return [tuple(p) for p in Polygon(closed).simplify(finest, preserve_topology=True).exterior.coords]
        cap = max(finest, cap / 2)


def _polygons(geometry):
    if geometry.geom_type == "Polygon":
        return [] if geometry.is_empty else [geometry]
    return [p for part in getattr(geometry, "geoms", []) for p in _polygons(part)]


def valid_rings(ring):
    """The ring itself, or the valid rings it splits into where it crosses itself, largest first."""
    ring = [tuple(p[:2]) for p in ring]
    polygon = Polygon(ring)
    if polygon.is_valid:
        return [ring]
    parts = sorted(_polygons(shapely.make_valid(polygon)), key=lambda part: -part.area)
    return [[tuple(p) for p in part.exterior.coords] for part in parts]


def simplify_outline(ring, detail, fine=FINE, coarse=COARSE):
    """simplify_ring for a land outline, which may come back as more than one ring.

    Runs simplified apart can cross each other where the coast doubles back within the tolerance.
    Splitting the outline at such a crossing keeps the rest of it at the tolerances asked for.
    """
    return valid_rings(_joined(*_open(ring, detail, fine, coarse)))


def _emitted(ring, counter_clockwise):
    """The ring as it is written: rounded to six places and wound as asked.

    Rounding can collapse a small outline to a line or make it touch itself, so this returns a list:
    empty when nothing with area is left, and more than one ring when rounding split it.
    """
    rounded = []
    for p in ring:
        point = (round(p[0], 6), round(p[1], 6))
        if not rounded or rounded[-1] != point:
            rounded.append(point)
    if len(set(rounded)) < 3:
        return []
    if rounded[0] != rounded[-1]:
        rounded.append(rounded[0])
    out = []
    for part in valid_rings(rounded):
        if len(set(part)) >= 3 and signed_area(part) != 0:
            part = part if (signed_area(part) > 0) == counter_clockwise else part[::-1]
            out.append([[p[0], p[1]] for p in part])
    return out


def _apart(rings):
    """(index of the ring it came from, ring) for each ring, cut so that no two overlap.

    Coarse outlines of neighbouring islands can cross. A ring wholly inside another is dropped; where
    two cross, the later one loses the shared part. Rings that touch nothing come back unchanged.
    """
    shapes = [Polygon(ring) for ring in rings]
    if not shapes:
        return []
    tree = STRtree(shapes)
    crossing = {
        j: sorted(int(i) for i in tree.query(shape, predicate="intersects") if i != j)
        for j, shape in enumerate(shapes)
    }
    inside = {j for j, others in crossing.items() if any(shapes[j].within(shapes[i]) for i in others)}
    out = []
    for j, shape in enumerate(shapes):
        if j in inside:
            continue
        cut = shape
        for i in crossing[j]:
            if i < j and i not in inside and shapes[i].intersection(shape).area > 0:
                cut = cut.difference(shapes[i])
        if cut is shape:
            out.append((j, rings[j]))
        else:
            out += [(j, [tuple(p) for p in part.exterior.coords]) for part in _polygons(cut)]
    return out


def overlap_area(rings):
    """The area, in square degrees, that is covered by more than one of the rings."""
    shapes = [Polygon(ring) for ring in rings]
    tree = STRtree(shapes)
    total = 0.0
    for j, shape in enumerate(shapes):
        for i in tree.query(shape, predicate="intersects"):
            if i < j:
                total += shapes[i].intersection(shape).area
    return total


def without_overlaps(rings):
    """The rings, cut so that no two overlap. The renderer fills by the even-odd rule."""
    return [ring for _, ring in _apart(rings)]


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
        graded, even = index.tolerances(hole, fine), [fine] * len(hole)
        for ring in (simplify_ring(hole, graded), simplify_ring(hole, even), list(map(tuple, hole))):
            if Polygon(ring).is_valid and Polygon(ring).within(exterior):
                return ring
        return None

    land, lakes, islands, source_land, whole = [], [], [], [], []
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
        outline = simplify_outline(polygon[0], index.tolerances(polygon[0], fine_for.get(i, FINE)))
        exterior = unary_union([Polygon(ring) for ring in outline])
        kept = [(i, ring) for ring in outline]
        for h in range(len(polygon) - 1):
            ring = kept_lake(i, h, exterior)
            written = _emitted(ring, False) if ring is not None else []
            if len(written) != 1:  # left out by the rule, or lost to rounding
                left_out_lakes += 1
                continue
            lake_rings[(i, h)] = Polygon(written[0])
            lakes.append(written[0])
        whole += kept
    for at, ring in _apart([ring for _, ring in whole]):
        written = _emitted(ring, True)
        if not written:
            left_out_islands += 1
        for part in written:
            land.append(part)
            source_land.append(whole[at][0])
    for i in pending:
        polygon, lake = polygons[i], lake_rings.get(lake_around(i))
        left_out_lakes += len(polygon) - 1  # a lake on an island in a lake is not drawn
        if lake is None:
            continue  # its lake was left out, so the island is already part of the land around it
        island = Polygon(simplify_ring(polygon[0], index.tolerances(polygon[0], fine_for.get(i, FINE))))
        if not island.within(lake):
            # Simplifying the lake moved its shore across the island. Keep the part still in the lake.
            island = Polygon(polygon[0]).intersection(lake)
        written = [w for part in _polygons(island) for w in _emitted(part.exterior.coords, True)]
        if not written:
            left_out_islands += 1
        islands += written
    return {
        "land": land,
        "lakes": lakes,
        "lake_islands": islands,
        "sources": {"land": source_land, "left_out_islands": left_out_islands, "left_out_lakes": left_out_lakes},
    }


def apply_local_shores(basemap, features):
    """Replace the generalised shoreline within the surveyed harbour windows."""
    for feature in features:
        window = box(*feature["properties"]["bounds"])
        rings = basemap["land"]
        polygons = [Polygon(ring) for ring in rings]
        touched = sorted(STRtree(polygons).query(window, predicate="intersects"))
        # These coastal windows contain no lake features from the global basemap.
        assert touched and all(not window.intersects(Polygon(ring)) for layer in ("lakes", "lake_islands")
                               for ring in basemap[layer])
        outside = unary_union([polygons[i] for i in touched]).difference(window)
        shoreline = outside.union(shape(feature["geometry"]))
        parts = sorted(_polygons(shoreline), key=lambda p: (-p.area, p.bounds))
        holes = [Polygon(h) for part in parts for h in part.interiors]
        land = []
        for part in parts:
            layer = basemap["lake_islands"] if any(h.covers(part) for h in holes) else land
            layer.extend(_emitted(part.exterior.coords, True))
            for hole in part.interiors:
                basemap["lakes"].extend(_emitted(hole.coords, False))
        # Retain the order of the other outlines, including the major continents.
        for i in reversed(touched[1:]):
            del rings[i]
        rings[touched[0] : touched[0] + 1] = land


def drawn_geometry(basemap):
    """The land the renderer draws: land outlines, minus lakes, plus the islands inside those lakes."""

    def union(rings):
        return unary_union([Polygon(r).buffer(0) for r in rings])

    return union(basemap["land"]).difference(union(basemap["lakes"])).union(union(basemap["lake_islands"]))
