"""Tests for the worldwide terrain rules and tool. Run as a script; needs tools/requirements-terrain.txt."""

import math
import pathlib
import sys

import numpy as np
import shapely.geometry

sys.path.insert(0, str(pathlib.Path(__file__).resolve().parent))
import basemap_rule as rule  # noqa: E402
import terrain_rule as T  # noqa: E402


def test_colour_scale_matches_the_design():
    assert T.STOPS == [0, 150, 400, 800, 1200, 1800, 2600, 3600, 4800]
    assert T.COLORS[0] == [231, 244, 204] and T.COLORS[6] == [166, 141, 128]
    assert T.COLORS[7] == [196, 186, 180] and T.COLORS[8] == [244, 243, 240]
    assert len(T.COLORS) == len(T.STOPS)


def test_level8_covers_60_km_either_side_of_a_line():
    # A line along the equator from 0E to 1E: 60 km is 0.54 degrees, a level-8 tile is 1.40625 degrees.
    tiles = T.level8_tiles([[(0.0, 0.0), (1.0, 0.0)]])
    assert (128, 127) in tiles and (128, 128) in tiles  # north and south of the equator
    assert (127, 127) in tiles  # 60 km west of the start
    assert (130, 127) not in tiles  # starts at 2.8E, more than 60 km east of the end


def test_level8_follows_a_long_leg_between_its_vertices():
    # Two vertices 10 degrees apart: the tiles in the middle are covered, not only the ends.
    tiles = T.level8_tiles([[(0.0, 0.0), (10.0, 0.0)]])
    assert all((x, 127) in tiles for x in range(128, 136))


def test_japan_boxes_are_always_covered():
    assert {(227, 93), (228, 95), (224, 100), (225, 101)} <= T.level8_tiles([])


def test_every_fine_tile_has_its_parents():
    levels = T.tile_list([[(0.0, 0.0), (1.0, 0.0)]], shapely.geometry.box(-10, -10, 10, 10))
    assert sorted(levels) == [2, 3, 4, 5, 6, 7, 8]
    assert all((x >> 1, y >> 1) in levels[7] for x, y in levels[8])
    assert all((x >> 1, y >> 1) in levels[6] for x, y in levels[7])


def test_coarse_levels_cover_land_north_of_60_south_only():
    antarctica, europe = shapely.geometry.box(0, -80, 10, -70), shapely.geometry.box(0, 42, 10, 50)
    assert T.land_tiles(4, antarctica) == set()
    assert T.land_tiles(4, europe) == {(8, 5)}


def test_terrarium_elevation():
    rgb = np.array([[[128, 0, 0], [131, 232, 128]]], dtype=np.float64)
    assert T.elevation(rgb).tolist() == [[0.0, 1000.5]]


def test_unproject_reverses_the_map_projection():
    for lon, lat in [(0, 0), (140.8, 42.8), (-75.6, 6.2), (30, -59), (-170, 80)]:
        back = T.unproject(*rule.project(lon, lat))
        assert abs(back[0] - lon) < 1e-9 and abs(back[1] - lat) < 1e-9, (lon, lat, back)


def surface(rows, cols):
    """A hilly surface defined for any pixel, so differently placed tiles can be cut from it."""
    return 1500 + 1000 * np.sin(cols[None, :] / 4.0) * np.cos(rows[:, None] / 7.0)


def tile_of_surface(first_col, halo):
    pad = 1 if halo else 0
    rows = np.arange(-pad, 256 + pad, dtype=np.float64)
    cols = np.arange(first_col - pad, first_col + 256 + pad, dtype=np.float64)
    return T.shade(surface(rows, cols), 8, 100, halo)


def test_shading_is_continuous_across_a_tile_edge():
    # Columns 255 and 256 of the surface sit on a tile edge in tiles A and B, and inside tile C.
    a, b, c = tile_of_surface(0, True), tile_of_surface(256, True), tile_of_surface(128, True)
    assert np.array_equal(a[:, 255], c[:, 127]) and np.array_equal(b[:, 0], c[:, 128])
    # Without the neighbour's pixels the edge column is shaded from a guessed slope, and differs.
    assert not np.array_equal(tile_of_surface(0, False)[:, 255], c[:, 127])


def test_flat_ground_gets_the_plain_colour_times_flat_shade():
    rgb = T.shade(np.full((258, 258), 400.0), 8, 100, True)
    assert rgb.shape == (256, 256, 3) and rgb.dtype == np.uint8
    assert tuple(rgb[128, 128]) == tuple(int(c * (0.74 + 0.34 * 0.7071)) for c in (199, 211, 145))


def test_slopes_facing_the_light_are_brighter():
    # Light comes from the northwest: ground rising to the southeast faces it, ground rising to the northwest does not.
    rows, cols = np.mgrid[0:258, 0:258].astype(np.float64)
    lit = T.shade(1000 + 200 * (rows + cols), 8, 100, True)
    shadowed = T.shade(1000 + 200 * (516 - rows - cols), 8, 100, True)
    assert int(lit[128, 128, 0]) > int(shadowed[128, 128, 0])


def test_high_ground_is_not_all_one_tone():
    low, high = T.shade(np.full((258, 258), 2600.0), 8, 100, True), T.shade(np.full((258, 258), 4800.0), 8, 100, True)
    assert int(high[0, 0, 2]) - int(low[0, 0, 2]) > 90  # Bogota's plateau and a 4,800 m peak differ


def test_mask_needs_both_height_and_drawn_land():
    land = T.densified(shapely.geometry.box(0, 0, 0.7, 1.5))  # tile 8/128/127 spans 0 to 1.40625 E: its western half
    mask = T.land_mask(np.full((256, 256), 50.0), land, 8, 128, 127)
    assert mask.dtype == bool and mask.shape == (256, 256)
    assert mask[128, 10] and not mask[128, 250]
    assert not T.land_mask(np.full((256, 256), 0.5), land, 8, 128, 127).any()


def test_mask_leaves_lakes_clear_and_keeps_their_islands():
    lake = shapely.geometry.box(0.4, 0.4, 1.0, 1.0)
    island = shapely.geometry.box(0.6, 0.6, 0.8, 0.8)
    land = T.densified(shapely.geometry.box(0, 0, 1.40625, 1.5).difference(lake).union(island))
    mask = T.land_mask(np.full((256, 256), 50.0), land, 8, 128, 127)

    def at(lon, lat):
        return mask[T.pixel(8, 128, 127, lon, lat)[::-1]]

    assert at(0.2, 0.2) and not at(0.5, 0.5) and at(0.7, 0.7)


def test_mask_follows_map_straight_edges():
    # The map draws an outline edge as a straight line between its projected ends. For a 370 km edge at
    # 60 N, far from the map's central meridian, that line is neither straight in degrees nor straight on
    # a Mercator tile. (Near the central meridian the three nearly coincide, so the edge is at 100 E.)
    a, b = (100.0, 58.0), (105.0, 62.0)
    ax, ay = rule.project(*a)
    bx, by = rule.project(*b)
    lon, lat = T.unproject((ax + bx) / 2, (ay + by) / 2)  # the middle of the edge as drawn
    z = 8
    x, y = T.tile_at(z, lon, lat)
    land = T.densified(shapely.geometry.Polygon([a, b, (105.0, 58.0)]))  # land lies south-east of the edge
    mask = T.land_mask(np.full((256, 256), 50.0), land, z, x, y)
    col, row = T.pixel(z, x, y, lon, lat)
    edge = int(np.argmax(mask[row]))  # the first land pixel in that row
    assert mask[row].any() and not mask[row].all() and abs(edge - col) <= 1, (edge, col)
    # The same edge taken as straight in degrees, or straight on the tile, would be drawn somewhere else.
    degrees = T.pixel_exact(z, x, y, a[0] + (lat - a[1]) / (b[1] - a[1]) * (b[0] - a[0]), lat)[0]
    assert abs(degrees - col) > 2, (degrees, col)
    naive = shapely.geometry.LineString([T.pixel_exact(z, x, y, *a), T.pixel_exact(z, x, y, *b)])
    assert naive.distance(shapely.geometry.Point(col, row)) > 2


def test_tile_kind():
    assert T.kind(np.ones((256, 256), bool)) == "land" and T.kind(np.zeros((256, 256), bool)) == "sea"
    part = np.zeros((256, 256), bool)
    part[0, 0] = True
    assert T.kind(part) == "coast"


def test_datasets_are_read_from_the_source_header():
    header = "srtm/N43E140.tif, gmted/30N120E_20101117_gmted_mea075.tif, etopo1/ETOPO1_Bed_g.tif"
    assert T.datasets(header) == {"srtm", "gmted", "etopo1"}
    assert T.datasets("") == set()


def main():
    for name, test in sorted(globals().items()):
        if name.startswith("test_"):
            test()
    print("terrain rule tests passed")


if __name__ == "__main__":
    main()
