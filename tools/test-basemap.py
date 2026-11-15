"""Tests for the worldwide basemap rule and its tool. Run as a script; needs shapely and scipy."""

import importlib.util
import pathlib
import sys

from shapely.geometry import LineString, Point, Polygon

sys.path.insert(0, str(pathlib.Path(__file__).resolve().parent))
import basemap_rule as rule  # noqa: E402


def square(west, south, width, height=None):
    height = width if height is None else height
    return [
        (west, south),
        (west + width, south),
        (west + width, south + height),
        (west, south + height),
        (west, south),
    ]


def zigzag_square():
    """A 2-degree square whose south and north edges each carry a 0.01-degree zigzag."""
    south = [(i * 0.02, 0.01 if i % 2 else 0.0) for i in range(101)]
    north = [(2 - i * 0.02, 1.99 if i % 2 else 2.0) for i in range(101)]
    return south + north + [south[0]]


def wiggly():
    """One land piece beside the path whose south coast wiggles by less than the fine tolerance."""
    south = [(i * 0.01, 0.2 + (0.002 if i % 2 else 0.0)) for i in range(101)]
    return [[south + [(1, 1), (0, 1), south[0]]]]


def test_tile_bounds():
    w, s, e, n = rule.tile_bounds(8, 227, 93, 228, 95)
    assert (w, e) == (139.21875, 142.03125) and abs(s - 40.9799) < 1e-4 and abs(n - 44.08759) < 1e-4


def test_near_index():
    index = rule.NearIndex([[(0, 0), (10, 0)]], [(100, 10, 101, 11)])
    assert index.near(5, 0.9)  # about 100 km north of the path
    assert not index.near(5, 1.8)  # about 200 km
    assert index.near(100.5, 10.5)  # inside the box, far from any path
    assert not index.near(-170, -60)


def test_simplify_keeps_detail_only_where_near():
    ring = zigzag_square()
    all_near = rule.simplify_ring(ring, [True] * len(ring))
    all_far = rule.simplify_ring(ring, [False] * len(ring))
    south_near = rule.simplify_ring(ring, [lat < 1 for _, lat in ring])
    assert len(all_far) == 5 and all_far[0] == all_far[-1], all_far
    assert len(all_near) > 40
    assert len(all_far) < len(south_near) < len(all_near)
    assert south_near[0] == south_near[-1] and Polygon(south_near).is_valid
    assert all(p[1] < 1.02 for p in south_near if p not in all_far)  # detail only in the south


def test_a_ring_that_crosses_itself_is_split_not_abandoned():
    # Runs simplified apart can cross each other at a sand spit (Afro-Eurasia's outline does, in the Nile
    # delta). Falling back to less simplification for the whole continent tripled the basemap.
    bowtie = [(0, 0), (2, 2), (2, 0), (0, 2), (0, 0)]
    rings = rule.valid_rings(bowtie)
    assert len(rings) == 2 and all(Polygon(r).is_valid and r[0] == r[-1] for r in rings), rings
    assert abs(sum(abs(rule.signed_area(r)) for r in rings) - 2) < 1e-9
    assert rule.valid_rings(square(0, 0, 1)) == [square(0, 0, 1)]


def test_across_km():
    assert 150 < rule.across_km([(0, 0), (1, 0), (1, 1), (0, 1), (0, 0)]) < 160


def test_compose_leaves_out_what_the_rule_says():
    index = rule.NearIndex([[(0, 0), (4, 0)]], [])
    polygons = [
        [square(-1, -1, 6, 3), square(1, 0.5, 0.02), square(2, 0.5, 0.2)],  # mainland, 2 km lake, 22 km lake
        [square(2.07, 0.57, 0.01)],  # island in the 22 km lake
        [square(1.005, 0.505, 0.005)],  # island in the 2 km lake, which is left out
        [square(30, 30, 0.05)],  # far island about 7 km across
        [square(40, 30, 0.5)],  # far island about 70 km across
        [square(50, 30, 6, 6), square(51, 31, 0.5), square(53, 31, 2)],  # far land, 70 km lake, 280 km lake
    ]
    out = rule.compose(polygons, index)
    assert len(out["land"]) == 3 and out["sources"]["land"] == [0, 4, 5], out["sources"]
    assert len(out["lakes"]) == 2  # the 22 km near lake and the 280 km far lake
    assert len(out["lake_islands"]) == 1
    assert all(rule.signed_area(r) > 0 for r in out["land"] + out["lake_islands"])
    assert all(rule.signed_area(r) < 0 for r in out["lakes"])
    drawn = rule.drawn_geometry(out)
    assert drawn.contains(Point(2.075, 0.575)) and not drawn.contains(Point(2.15, 0.65))
    assert drawn.contains(Point(1.01, 0.51)) and drawn.contains(Point(51.2, 31.2))
    assert not drawn.contains(Point(30.02, 30.02))


def test_compose_tightens_one_polygon():
    index = rule.NearIndex([[(0, 0), (4, 0)]], [])
    loose = rule.compose(wiggly(), index)
    tight = rule.compose(wiggly(), index, fine_for={0: 0})
    assert len(tight["land"][0]) > len(loose["land"][0])


def load(name):
    path = pathlib.Path(__file__).resolve().parent / name
    spec = importlib.util.spec_from_file_location(name.removesuffix(".py").replace("-", "_"), path)
    module = importlib.util.module_from_spec(spec)
    spec.loader.exec_module(module)
    return module


def toothed_coast():
    """Land whose south coast has teeth 0.002 degrees deep: real land that the fine tolerance shaves off."""
    south = [(i * 0.01, 0.2 if i % 2 else 0.202) for i in range(101)]
    return [[south + [(1, 1), (0, 1), south[0]]]]


def tooth_route():
    return [("tooth-route", "launch", [(0.51, 0.2005), (0.51, 0.6)], None)]  # id, alignment, path, allowed crossing


def test_offshore_route_stops_the_tool():
    tool = load("prepare-basemap.py")
    polygons = [[square(0, 0, 1)]]
    routes = [("test-route", "launch", [(0.5, 0.5), (1.5, 0.5)], None)]
    try:
        tool.fit(polygons, rule.NearIndex([r[2] for r in routes], []), routes)
    except SystemExit as e:
        assert "test-route" in str(e) and "off the drawn land at 1.0" in str(e), str(e)  # just past the coast
    else:
        raise AssertionError("an offshore route was accepted")


def test_fit_tightens_only_the_polygon_a_route_touches():
    tool = load("prepare-basemap.py")
    polygons = toothed_coast() + [[square(3, 0, 1)]]
    index = rule.NearIndex([r[2] for r in tooth_route()], [])
    assert not rule.drawn_geometry(rule.compose(polygons, index)).covers(LineString(tooth_route()[0][2]))
    basemap, tightened = tool.fit(polygons, index, tooth_route())
    assert tightened == [{"route": "tooth-route", "polygon": 0, "tolerance": 0.001}], tightened
    assert rule.drawn_geometry(basemap).covers(LineString(tooth_route()[0][2]))


def test_fit_survives_land_that_is_invalid_once_projected():
    # A spike that stops short of a long meridian edge on the globe crosses that edge's straight chord on
    # the map, so the projected outline intersects itself. Antarctica's coarse outline does the same.
    tool = load("prepare-basemap.py")
    spiked = [(100, -60), (100, 60), (90, 60), (90, 1), (99, 0.5), (90, 0), (90, -60), (100, -60)]
    assert Polygon(spiked).is_valid
    polygons = [[square(0, 0, 1)], [spiked]]
    routes = [("crossing-route", "launch", [(0.2, 0.5), (0.8, 0.5)], Polygon(square(0.4, 0.4, 0.2)))]
    basemap, tightened = tool.fit(polygons, rule.NearIndex([r[2] for r in routes], []), routes)
    assert tightened == [] and len(basemap["land"]) == 2


def main():
    for name, test in sorted(globals().items()):
        if name.startswith("test_"):
            test()
    print("basemap rule tests passed")


if __name__ == "__main__":
    main()
