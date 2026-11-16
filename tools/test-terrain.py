"""Tests for the worldwide terrain rules and tool. Run as a script; needs tools/requirements-terrain.txt."""

import hashlib
import importlib.util
import pathlib
import sys
import tempfile

import numpy as np
from PIL import Image
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


def test_detail_falls_away_from_a_line_in_steps():
    # Sharp relief must not end against relief eight times coarser. A short line near 1 E, 0.7 N:
    # level 8 reaches 60 km from it, level 7 180 km and level 6 400 km, each a halving of detail.
    levels = T.tile_list([[(1.0, 0.7), (1.2, 0.7)]], shapely.geometry.box(-20, -20, 20, 20))
    assert T.RINGS == {8: 60, 7: 180, 6: 400}
    assert {(x, y) for x, y in levels[8] if y == 127} == {(128, 127), (129, 127)}  # 60 km is 0.54 degrees
    seven, six = T.parents(levels[8]), T.parents(T.parents(levels[8]))
    assert (65, 63) in levels[7] and (65, 63) not in seven  # 180 km east of the line, beyond level 8's parents
    assert (64, 64) in levels[7] and (64, 64) not in seven  # 180 km south
    assert (66, 63) not in levels[7]  # starts at 5.6 E, more than 180 km away
    assert (32, 32) in levels[6] and (33, 31) not in levels[6]  # 400 km is 3.6 degrees; 5.6 E is beyond it
    assert seven <= levels[7] and six <= levels[6] and T.parents(levels[7]) <= levels[6]
    # A tile is 2.8 degrees at level 7 and 5.6 at level 6, so each ring is at least one tile wide.
    assert len(levels[7]) > len(seven) and len(levels[6]) >= len(six)


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


def tile_of_surface(first_col):
    """The shading of a tile whose west edge is at source column first_col, sampled on the written grid."""
    at = (np.arange(-1, 257) - 0.5) * 256 / T.OWN
    return T.shade(surface(at, first_col + at), 8, 100)


def test_neighbouring_tiles_share_two_columns_of_ground():
    # A written tile's outermost pixels lie in its neighbours: the last two columns of one tile are the
    # same ground as the first two of the next, so blending across the shared edge has no seam.
    west, east = tile_of_surface(0), tile_of_surface(256)
    for mine, theirs in ((254, 0), (255, 1)):
        difference = np.abs(west[:, mine].astype(int) - east[:, theirs].astype(int))
        assert difference.max() <= 1, (mine, theirs, difference.max())
    # The surface is steep enough that columns one apart do differ, so the agreement above means something.
    assert np.abs(west[:, 255].astype(int) - east[:, 0].astype(int)).max() > 10


def test_resampling_keeps_a_slope_exact():
    # Source pixel j of a tile is centred j + 0.5 from its edge; with HALO around it, index i is centred i + 0.5 - HALO.
    index = np.arange(256 + 2 * T.HALO, dtype=np.float64) + 0.5 - T.HALO
    padded = 10 * index[None, :] + 1000 * index[:, None]  # 10 m per pixel east, 1,000 south
    out = T.overlapped(padded)
    assert out.shape == (258, 258)
    centre = (np.arange(-1, 257) - 0.5) * 256 / T.OWN  # where each written pixel is centred
    assert np.allclose(out, 10 * centre[None, :] + 1000 * centre[:, None])


def test_resampling_keeps_fine_detail():
    # A ridge-and-valley pattern four source pixels long is close to the finest the source can hold.
    # Blending only the two nearest pixels would flatten it by a third where a written pixel falls
    # between source pixels (an error of 29 here); a curve through the four nearest keeps it within 12.
    index = np.arange(256 + 2 * T.HALO, dtype=np.float64) + 0.5 - T.HALO
    padded = np.tile(100 * np.sin(index * np.pi / 2), (len(index), 1))
    out = T.overlapped(padded)[1:-1, 1:-1]
    centre = (np.arange(256) - 0.5) * 256 / T.OWN
    exact = 100 * np.sin(centre * np.pi / 2)
    assert np.abs(out[100] - exact).max() < 15, np.abs(out[100] - exact).max()


def test_a_place_on_a_tile_edge_is_one_pixel_in():
    west, _, east, north = rule.tile_bounds(8, 128, 127, 128, 127)
    assert np.allclose(T.texel_exact(8, 128, 127, west, north), (1, 1))
    assert np.allclose(T.texel_exact(8, 128, 127, east, north), (255, 1))
    assert T.texel(8, 128, 127, west + 1e-9, north - 1e-9) == (1, 1)


def test_flat_ground_gets_the_plain_colour_times_flat_shade():
    rgb = T.shade(np.full((258, 258), 400.0), 8, 100)
    assert rgb.shape == (256, 256, 3) and rgb.dtype == np.uint8
    assert tuple(rgb[128, 128]) == tuple(int(c * (0.74 + 0.34 * 0.7071)) for c in (199, 211, 145))


def test_slopes_facing_the_light_are_brighter():
    # Light comes from the northwest: ground rising to the southeast faces it, ground rising to the northwest does not.
    rows, cols = np.mgrid[0:258, 0:258].astype(np.float64)
    lit = T.shade(1000 + 200 * (rows + cols), 8, 100)
    shadowed = T.shade(1000 + 200 * (516 - rows - cols), 8, 100)
    assert int(lit[128, 128, 0]) > int(shadowed[128, 128, 0])


def test_high_ground_is_not_all_one_tone():
    low, high = T.shade(np.full((258, 258), 2600.0), 8, 100), T.shade(np.full((258, 258), 4800.0), 8, 100)
    assert int(high[0, 0, 2]) - int(low[0, 0, 2]) > 90  # Bogota's plateau and a 4,800 m peak differ


def test_a_parents_quarter_is_enlarged_in_place():
    rows, cols = np.mgrid[0:256, 0:256].astype(np.float64)
    parent = 10 * cols + 1000 * rows  # parent pixel j is centred j + 0.5 from its edge
    child = T.from_parent(parent, 5, 4)  # odd column, even row: the north-east quarter
    assert child.shape == (256, 256)
    # Child pixel j is centred (j + 0.5) / 2 parent pixels into the quarter, which starts at column 128.
    assert np.isclose(child[100, 100], 10 * (128 + 50.25 - 0.5) + 1000 * (50.25 - 0.5))
    assert np.isclose(child[0, 255], 10 * 255) and np.isclose(child[255, 0], 10 * (128 - 0.25) + 1000 * 127.25)


def test_ground_under_ice_takes_the_surface_from_the_level_above():
    # The level-5 source is a bedrock grid. Under Greenland's ice it lies up to 2.8 km below the surface
    # the level above shows. The gap between the two, averaged over about 160 km, says where the ice is.
    half = T.ICE_WINDOW // 2
    size = 256 + 2 * half
    assert np.allclose(T.ice_weight(np.full((size, size), 2000.0)), 1)  # deep under the ice sheet
    assert np.allclose(T.ice_weight(np.zeros((size, size))), 0)  # ordinary ground
    # Mountains are not ice: valleys lie far below the coarse level's average and peaks far above it, but
    # the gap averages out, so nothing is filled in.
    rows, cols = np.mgrid[0:size, 0:size]
    ranges = 900 * np.sin(cols / 3.0) * np.cos(rows / 2.0)
    assert np.abs(ranges).max() > 800 and np.allclose(T.ice_weight(ranges), 0)
    # A lake's surface above the lake bed the coarser level shows is a negative gap: left alone.
    assert np.allclose(T.ice_weight(np.full((size, size), -300.0)), 0)
    # Across the ice margin the weight rises smoothly, with no step for the shading to catch.
    margin = np.tile(np.clip((np.arange(size) - 60) * 20.0, 0, 3000), (size, 1))
    weight = T.ice_weight(margin)
    assert weight.shape == (256, 256) and weight[100, 0] == 0 and weight[100, 255] == 1
    assert (np.diff(weight[100]) >= 0).all() and np.diff(weight[100]).max() < 0.1


def test_the_ice_margin_does_not_step_at_a_tile_edge():
    # Two tiles side by side, with ice thickening eastwards across both: the weights at their shared
    # edge agree, because each tile sees its neighbour's gap.
    P = tool()
    half = T.ICE_WINDOW // 2
    cols = np.arange(512, dtype=np.float64)
    west, east = np.tile(cols[:256], (256, 1)), np.tile(cols[256:], (256, 1))  # 255 m at the shared edge
    gaps = {(5, 5): west.astype(np.float32), (6, 5): east.astype(np.float32)}
    a = T.ice_weight(P.with_halo(gaps, 8, 5, 5, half))
    b = T.ice_weight(P.with_halo(gaps, 8, 6, 5, half))
    assert 0 < a[100, 255] < 1 and abs(a[100, 255] - b[100, 0]) < 0.02


def test_a_source_with_no_header_is_not_assumed_credited():
    P = tool()
    text = "SRTM and GMTED2010 from USGS; ETOPO1 from NOAA."
    counts, gaps = P.credit_gaps(["srtm/a.tif, etopo1/b.tif", "gmted/c.tif"], text)
    assert gaps == set() and counts == {"srtm": 1, "etopo1": 1, "gmted": 1}
    _, gaps = P.credit_gaps(["srtm/a.tif", "", "ned/d.tif"], text)
    assert gaps == {"ned", "a source tile with no dataset header"}


def test_land_is_the_drawn_basemap_whatever_its_height():
    # Polders, deltas and basins lie at or below sea level and are land all the same. Before tiles were
    # cut to the basemap, height above 1 m was the only way to tell land from sea; kept as a second
    # test it left such ground without relief, which showed as dark blotches in the dark palette.
    land = T.densified(shapely.geometry.box(0, 0, 0.7, 1.5))  # tile 8/128/127 spans 0 to 1.40625 E: its western half
    mask = T.land_mask(land, 8, 128, 127)
    assert mask.dtype == bool and mask.shape == (256, 256)
    assert mask[128, 10] and not mask[128, 250]
    assert "heights" not in T.land_mask.__code__.co_varnames


def test_mask_leaves_lakes_clear_and_keeps_their_islands():
    lake = shapely.geometry.box(0.4, 0.4, 1.0, 1.0)
    island = shapely.geometry.box(0.6, 0.6, 0.8, 0.8)
    land = T.densified(shapely.geometry.box(0, 0, 1.40625, 1.5).difference(lake).union(island))
    mask = T.land_mask(land, 8, 128, 127)

    def at(lon, lat):
        return mask[T.texel(8, 128, 127, lon, lat)[::-1]]

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
    mask = T.land_mask(land, z, x, y)
    col, row = T.texel(z, x, y, lon, lat)
    edge = int(np.argmax(mask[row]))  # the first land pixel in that row
    assert mask[row].any() and not mask[row].all() and abs(edge - col) <= 1, (edge, col)
    # The same edge taken as straight in degrees, or straight on the tile, would be drawn somewhere else.
    degrees = T.texel_exact(z, x, y, a[0] + (lat - a[1]) / (b[1] - a[1]) * (b[0] - a[0]), lat)[0]
    assert abs(degrees - col) > 2, (degrees, col)
    naive = shapely.geometry.LineString([T.texel_exact(z, x, y, *a), T.texel_exact(z, x, y, *b)])
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


def test_uncredited_datasets_are_named():
    text = "SRTM and GMTED2010 from USGS; ETOPO1 from NOAA."
    assert T.uncredited({"srtm", "gmted", "etopo1"}, text) == set()
    assert T.uncredited({"srtm", "ned", "eudem"}, text) == {"ned", "eudem"}
    assert T.uncredited({"srtm"}, "nothing here") == {"srtm"}


def test_the_committed_credits_cover_the_credited_datasets():
    text = (pathlib.Path(__file__).resolve().parents[1] / "assets/terrain/ATTRIBUTION.md").read_text()
    assert T.uncredited(set(T.CREDITED), text) == set()


def tool():
    spec = importlib.util.spec_from_file_location("prepare_terrain", pathlib.Path(__file__).with_name("prepare-terrain.py"))
    module = importlib.util.module_from_spec(spec)
    spec.loader.exec_module(module)
    return module


def half_land():
    mask = np.zeros((256, 256), bool)
    mask[:, :128] = True
    return mask


def hills():
    rows, cols = np.mgrid[0:256, 0:256]
    return np.stack([100 + cols // 2, 120 + rows // 3, np.full((256, 256), 90)], axis=-1).astype(np.uint8)


def test_coast_tile_has_a_jpeg_and_a_one_bit_mask():
    with tempfile.TemporaryDirectory() as tmp:
        tmp = pathlib.Path(tmp)
        entry = tool().write_tile(tmp, 8, 1, 2, hills(), half_land())
        assert entry[:4] == [8, 1, 2, "coast"] and len(entry) == 6
        assert entry[4] == hashlib.sha256((tmp / "8/1/2.jpg").read_bytes()).hexdigest()
        assert entry[5] == hashlib.sha256((tmp / "8/1/2.png").read_bytes()).hexdigest()
        mask = Image.open(tmp / "8/1/2.png")
        assert mask.mode == "1" and mask.size == (256, 256)
        assert np.array_equal(np.array(mask, dtype=bool), half_land())
        jpeg = np.array(Image.open(tmp / "8/1/2.jpg"))
        assert jpeg.shape == (256, 256, 3)
        assert all(abs(int(jpeg[128, 250, i]) - T.LOWLAND[i]) <= 3 for i in range(3))  # sea carries the lowland colour
        assert abs(int(jpeg[128, 20, 0]) - 110) <= 6  # land keeps its own


def test_all_land_tile_has_no_mask():
    with tempfile.TemporaryDirectory() as tmp:
        tmp = pathlib.Path(tmp)
        entry = tool().write_tile(tmp, 8, 1, 3, hills(), np.ones((256, 256), bool))
        assert entry[3] == "land" and len(entry) == 5 and not (tmp / "8/1/3.png").exists()


def test_all_sea_tile_is_listed_without_files():
    with tempfile.TemporaryDirectory() as tmp:
        tmp = pathlib.Path(tmp)
        entry = tool().write_tile(tmp, 8, 1, 4, hills(), np.zeros((256, 256), bool))
        assert entry == [8, 1, 4, "sea"] and not (tmp / "8/1/4.jpg").exists()


def test_writing_twice_gives_the_same_bytes():
    with tempfile.TemporaryDirectory() as tmp:
        tmp = pathlib.Path(tmp)
        first = tool().write_tile(tmp / "a", 8, 1, 2, hills(), half_land())
        second = tool().write_tile(tmp / "b", 8, 1, 2, hills(), half_land())
        assert first == second


def test_files_the_list_no_longer_names_are_removed():
    # A tile that was part sea and is now all land keeps its colour image and loses its mask; a tile
    # that left the list loses both. The game checks that no unlisted file is embedded.
    P = tool()
    with tempfile.TemporaryDirectory() as tmp:
        tmp = pathlib.Path(tmp)
        was_coast = P.write_tile(tmp, 8, 1, 2, hills(), half_land())
        P.write_tile(tmp, 8, 9, 9, hills(), half_land())
        now_land = P.write_tile(tmp, 8, 1, 2, hills(), np.ones((256, 256), bool))
        assert len(was_coast) == 6 and len(now_land) == 5 and (tmp / "8/1/2.png").exists()
        P.remove_unlisted(tmp, [now_land, [8, 3, 3, "sea"]])
        assert (tmp / "8/1/2.jpg").exists() and not (tmp / "8/1/2.png").exists()
        assert not (tmp / "8/9").exists()


def test_a_tile_borrows_edge_pixels_only_from_neighbours_it_has():
    P = tool()
    own, east = np.full((256, 256), 100.0, np.float32), np.full((256, 256), 300.0, np.float32)
    padded = P.with_halo({(5, 5): own, (6, 5): east}, 8, 5, 5)
    h = T.HALO
    assert padded.shape == (256 + 2 * h, 256 + 2 * h)
    assert (padded[100, 256 + h :] == 300).all() and (padded[100, :h] == 100).all()  # east: the neighbour; west: its own edge
    assert padded[0, 100] == 100 and padded[h, h] == 100
    assert padded[0, -1] == 100  # no north-east neighbour: the corner repeats its own


def test_a_tile_at_the_date_line_borrows_from_the_other_side():
    P = tool()
    heights = {(0, 1): np.full((256, 256), 100.0, np.float32), (3, 1): np.full((256, 256), 700.0, np.float32)}
    assert (P.with_halo(heights, 2, 0, 1)[100, : T.HALO] == 700).all()


def test_sample_selection_takes_named_routes_and_boxes():
    P = tool()
    routes = [("near", "a", [(0.0, 0.0), (1.0, 0.0)]), ("far", "a", [(50.0, 0.0), (51.0, 0.0)])]
    in_box, also_in_box = T.tile_at(8, 100.2, 10.5), T.tile_at(8, 103.5, 10.5)
    assert in_box != also_in_box
    tiles = P.selected_level8(routes, ["near"], [(100.0, 10.0, 104.0, 11.0)], {in_box})
    assert (128, 127) in tiles and in_box in tiles
    assert also_in_box not in tiles  # a box takes only tiles the full list has: nothing else was fetched
    assert T.tile_at(8, 50.5, 0.0) not in tiles
    try:
        P.selected_level8(routes, ["missing"], [], set())
        raise AssertionError("an unknown route was accepted")
    except SystemExit:
        pass


def main():
    for name, test in sorted(globals().items()):
        if name.startswith("test_"):
            test()
    print("terrain rule tests passed")


if __name__ == "__main__":
    main()
