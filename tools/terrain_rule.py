"""The rules for worldwide terrain tiles: which tiles exist, how they are shaded, and what counts as land.

Pure functions; tools/prepare-terrain.py does the reading, fetching and writing. Tiles are 256x256 on the
Web Mercator grid. Longitude comes before latitude in every pair, as in basemap_rule.
"""

import math

import numpy as np
from PIL import Image, ImageDraw
from scipy.ndimage import uniform_filter
import shapely
from shapely.geometry import MultiPolygon, Polygon, box

import basemap_rule as rule

# Elevation in metres and the colour at it. Unchanged from the Japan-only scale up to 1,800 m; above
# that it keeps going, so a 2,600 m plateau and a 4,800 m peak are different tones.
STOPS = [0, 150, 400, 800, 1200, 1800, 2600, 3600, 4800]
COLORS = [
    [231, 244, 204],
    [218, 231, 174],
    [199, 211, 145],
    [189, 190, 135],
    [183, 168, 133],
    [177, 157, 139],
    [166, 141, 128],
    [196, 186, 180],
    [244, 243, 240],
]
LOWLAND = tuple(COLORS[0])
JPEG_QUALITY = 80
# How far either side of a line each fine level reaches, in km. Level 8 alone would end against
# level 5, eight times coarser, along a hard edge; two wider rings make each step a halving.
RINGS = {8: 60, 7: 180, 6: 400}
NEAR_KM = RINGS[8]
SAMPLE_KM = 5
LEVELS = range(2, 9)
COARSE = range(2, 6)  # levels that cover all land
SOUTH_LIMIT = -60.0
EDGE_KM = 2  # outline edges are followed along the map's straight line in steps no longer than this


def _mercator_y(z, lat):
    return (1 - math.asinh(math.tan(math.radians(lat))) / math.pi) / 2 * 2**z


def pixel_exact(z, x, y, lon, lat):
    """Where a place falls in tile z/x/y, in pixels from its north-west corner."""
    return ((lon + 180) / 360 * 2**z - x) * 256, (_mercator_y(z, lat) - y) * 256


# A written tile is 256 pixels across, but only the middle 254 are its own area: the outermost pixel on
# each side lies in the neighbouring tile. Two tiles side by side therefore hold the same two columns
# of ground along their shared edge, and a renderer that blends between pixel centres can draw each
# from 1 to 255 with no seam and without reading outside the image. Sizes stay a power of two, which
# the game's graphics library would otherwise round up to 512.
OWN = 254
HALO = 4  # source pixels needed around a tile to resample it and its border


def texel_exact(z, x, y, lon, lat):
    """Where a place falls in the written tile z/x/y, in its pixels from the north-west corner."""
    col, row = pixel_exact(z, x, y, lon, lat)
    return col * OWN / 256 + 1, row * OWN / 256 + 1


def texel(z, x, y, lon, lat):
    """The pixel of the written tile z/x/y that holds a place, as (column, row)."""
    col, row = texel_exact(z, x, y, lon, lat)
    return math.floor(col), math.floor(row)


def overlapped(padded, width=HALO):
    """Resample a tile's source heights, with width source pixels around them, onto the written grid.

    Returns 258x258: the 256 written pixels and one more around them for the shading's slopes. Written
    pixel k is centred (k - 0.5) * 256 / 254 source pixels from the tile's edge. The grid is less than
    1% coarser than the source, so a written pixel falls anywhere between two source pixels; a
    Catmull-Rom curve through the four nearest keeps the detail that blending the nearest two would
    blur away.
    """
    assert padded.shape == (256 + 2 * width, 256 + 2 * width), padded.shape
    position = (np.arange(-1, 257) - 0.5) * 256 / OWN - 0.5 + width
    index = np.floor(position).astype(int)
    t = position - index
    weights = [(-t**3 + 2 * t**2 - t) / 2, (3 * t**3 - 5 * t**2 + 2) / 2, (-3 * t**3 + 4 * t**2 + t) / 2, (t**3 - t**2) / 2]
    rows = sum(w[:, None] * padded[index + k - 1, :] for k, w in enumerate(weights))
    return sum(w[None, :] * rows[:, index + k - 1] for k, w in enumerate(weights))


def place(z, x, y, col, row):
    """Longitude and latitude of a pixel position in tile z/x/y; the reverse of pixel_exact."""
    n = 2**z
    return (x + col / 256) / n * 360 - 180, math.degrees(math.atan(math.sinh(math.pi * (1 - 2 * (y + row / 256) / n))))


def tile_at(z, lon, lat):
    n = 2**z
    return min(n - 1, max(0, math.floor((lon + 180) / 360 * n))), min(n - 1, max(0, math.floor(_mercator_y(z, lat))))


def unproject(x, y):
    """Longitude and latitude of a point on the Equal Earth map: the reverse of basemap_rule.project."""
    a1, a2, a3, a4 = 1.340264, -0.081106, 0.000893, 0.003796
    t = y / a1
    for _ in range(12):
        t2 = t * t
        t6 = t2 * t2 * t2
        step = (t * (a1 + a2 * t2 + t6 * (a3 + a4 * t2)) - y) / (a1 + 3 * a2 * t2 + t6 * (7 * a3 + 9 * a4 * t2))
        t -= step
        if abs(step) < 1e-14:
            break
    t2 = t * t
    t6 = t2 * t2 * t2
    lon = x * 3 * (a1 + 3 * a2 * t2 + t6 * (7 * a3 + 9 * a4 * t2)) / (2 * math.sqrt(3) * math.cos(t))
    return math.degrees(lon), math.degrees(math.asin(2 * math.sin(t) / math.sqrt(3)))


def near_tiles(z, lines, km):
    """Every tile of level z within km of a line."""
    reach = (km + SAMPLE_KM / 2) / 111.2  # degrees of latitude; half a sample step covers between samples
    tiles = set()
    for path in lines:
        for lon, lat in rule.sample(path, SAMPLE_KM):
            wide = reach / max(0.01, math.cos(math.radians(lat)))
            x0, y0 = tile_at(z, lon - wide, lat + reach)
            x1, y1 = tile_at(z, lon + wide, lat - reach)
            tiles.update((x, y) for x in range(x0, x1 + 1) for y in range(y0, y1 + 1))
    return tiles


def level8_tiles(lines):
    """Every level-8 tile within NEAR_KM of a line, and the two Japan boxes."""
    tiles = near_tiles(8, lines, NEAR_KM)
    for z, x0, y0, x1, y1 in rule.JAPAN_BOXES:
        assert z == 8
        tiles.update((x, y) for x in range(x0, x1 + 1) for y in range(y0, y1 + 1))
    return tiles


def parents(tiles):
    return {(x >> 1, y >> 1) for x, y in tiles}


def land_tiles(z, geometry):
    """Tiles at level z that hold some of the geometry north of 60 S."""
    north = geometry.intersection(box(-180, SOUTH_LIMIT, 180, 90))
    if north.is_empty:
        return set()
    shapely.prepare(north)
    found = set()
    for x in range(2**z):
        for y in range(2**z):
            tile = box(*rule.tile_bounds(z, x, y, x, y))
            if north.intersects(tile) and north.intersection(tile).area > 0:
                found.add((x, y))
    return found


def tile_list(lines, geometry):
    """The tiles of every level.

    Levels 2 to 5 hold all land. Level 8 holds what is near a line. Levels 7 and 6 hold the parents of
    the level above and a wider ring of their own (see RINGS), so detail falls away from a line in steps.
    """
    levels = {z: land_tiles(z, geometry) for z in COARSE}
    levels[8] = level8_tiles(lines)
    for z in (7, 6):
        levels[z] = parents(levels[z + 1]) | near_tiles(z, lines, RINGS[z])
    return levels


def elevation(rgb):
    """Metres from a Terrarium tile's red, green and blue."""
    return rgb[..., 0] * 256 + rgb[..., 1] + rgb[..., 2] / 256 - 32768


# The level-5 source is a bedrock grid: under Greenland's ice it lies up to 2.8 km below the surface
# the level above shows. Where a tile's heights, averaged over ICE_WINDOW pixels, lie more than
# ICE_FROM_M below its parent's, the parent's surface is blended in, fully from ICE_FULL_M. The average
# is what tells ice from mountains, whose valleys and peaks differ from a coarser level by as much but
# in both directions. Everywhere but Greenland the averaged gap stays far under the threshold.
ICE_FROM_M, ICE_FULL_M, ICE_WINDOW = 150, 400, 33


def ice_weight(padded_gap):
    """How much of the parent's surface each pixel takes: 0 for ordinary ground to 1 under an ice sheet.

    padded_gap is the parent's surface minus the tile's own heights, with ICE_WINDOW // 2 pixels around
    the tile (from its neighbours, so the weight does not step at tile edges). Returns 256x256.
    """
    half = ICE_WINDOW // 2
    assert padded_gap.shape == (256 + 2 * half, 256 + 2 * half), padded_gap.shape
    smooth = uniform_filter(padded_gap.astype(np.float64), size=ICE_WINDOW, mode="nearest")[half:-half, half:-half]
    return np.clip((smooth - ICE_FROM_M) / (ICE_FULL_M - ICE_FROM_M), 0, 1)


def from_parent(parent, x, y):
    """The quarter of a parent tile's heights that covers child column x, row y, enlarged to 256x256."""
    at = (np.arange(256) + 0.5) / 2 - 0.5
    cols, rows = np.clip(at + 128 * (x & 1), 0, 255), np.clip(at + 128 * (y & 1), 0, 255)
    c0, r0 = np.floor(cols).astype(int), np.floor(rows).astype(int)
    c1, r1 = np.minimum(c0 + 1, 255), np.minimum(r0 + 1, 255)
    fc, fr = (cols - c0)[None, :], (rows - r0)[:, None]
    top = parent[r0][:, c0] * (1 - fc) + parent[r0][:, c1] * fc
    bottom = parent[r1][:, c0] * (1 - fc) + parent[r1][:, c1] * fc
    return top * (1 - fr) + bottom * fr


def shade(heights, z, y):
    """Colour and hillshade one written tile of row y at level z.

    heights is 258x258 on the written grid: the tile's 256 pixels and one more around them (see
    overlapped). Ground distance per pixel is taken row by row, so shading does not step between rows
    of tiles, and neighbouring tiles shade the ground they share identically.
    """
    assert heights.shape == (258, 258), heights.shape
    rows = y + (np.arange(256) - 0.5) / OWN
    lat = np.arctan(np.sinh(math.pi * (1 - 2 * rows / 2**z)))
    metres = (np.cos(lat) * 2 * math.pi * 6378137 / (OWN * 2**z))[:, None]
    dx = (heights[1:-1, 2:] - heights[1:-1, :-2]) / (2 * metres)
    dy = (heights[2:, 1:-1] - heights[:-2, 1:-1]) / (2 * metres)
    own = heights[1:-1, 1:-1]
    # Light from the northwest, relief exaggerated 2.5 times.
    nx, ny = -dx * 2.5, dy * 2.5
    light = np.clip(0.74 + 0.34 * ((nx * -0.5 + ny * 0.5 + 0.7071) / np.sqrt(nx * nx + ny * ny + 1)), 0.55, 1.10)
    colors = np.array(COLORS, dtype=np.float64)
    rgb = np.stack([np.interp(np.maximum(own, 0), STOPS, colors[:, i]) for i in range(3)], axis=-1)
    return np.clip(rgb * light[:, :, None], 0, 255).astype(np.uint8)


def densified(geometry, step_km=EDGE_KM):
    """The geometry with every edge following the straight line the map draws between its ends.

    The renderer joins an outline's projected points with straight lines. On a Mercator tile such a line
    is a curve, so long edges are cut into short ones along it before the mask is drawn.
    """

    def ring(points):
        out = [tuple(points[0][:2])]
        for a, b in zip(points, points[1:]):
            steps = math.ceil(rule.km_between(a, b) / step_km)
            if steps > 1:
                ax, ay = rule.project(a[0], a[1])
                bx, by = rule.project(b[0], b[1])
                out += [unproject(ax + (bx - ax) * i / steps, ay + (by - ay) * i / steps) for i in range(1, steps)]
            out.append(tuple(b[:2]))
        return out

    return MultiPolygon(
        [
            Polygon(ring(list(p.exterior.coords)), [ring(list(hole.coords)) for hole in p.interiors])
            for p in rule._polygons(geometry)
        ]
    )


def land_mask(geometry, z, x, y):
    """Which pixels of the written tile z/x/y are land: those inside the geometry (already densified).

    The geometry is the land the basemap draws. Height plays no part: ground at or below sea level is
    land where the basemap draws it, and sea floor is not where it does not.
    """
    west, south, east, north = rule.tile_bounds(z, x, y, x, y)
    margin = (east - west) / 32
    near = shapely.clip_by_rect(geometry, west - margin, max(-89.9, south - margin), east + margin, north + margin)
    inside = np.zeros((256, 256), dtype=bool)

    def filled(points):
        image = Image.new("1", (256, 256), 0)
        # A pixel is land when its centre is inside, so shift half a pixel.
        ImageDraw.Draw(image).polygon(
            [(c - 0.5, r - 0.5) for c, r in (texel_exact(z, x, y, p[0], p[1]) for p in points)], fill=1
        )
        return np.array(image, dtype=bool)

    for polygon in rule._polygons(near):
        if polygon.is_empty:
            continue
        part = filled(polygon.exterior.coords)
        for hole in polygon.interiors:
            part &= ~filled(hole.coords)
        inside |= part
    return inside


def kind(mask):
    return "land" if mask.all() else "coast" if mask.any() else "sea"


# Elevation datasets the credits file names, by the directory Mapzen's source headers cite them under.
CREDITED = {"srtm": "SRTM", "gmted": "GMTED2010", "etopo1": "ETOPO1"}


def uncredited(names, attribution_text):
    """Dataset names that have no credit: unknown ones, and known ones whose label the credits text lacks."""
    return {name for name in names if name not in CREDITED or CREDITED[name] not in attribution_text}


def datasets(imagery_sources):
    """The elevation datasets a source tile's header cites: the part of each entry before its first slash."""
    return {entry.strip().split("/")[0] for entry in imagery_sources.split(",") if entry.strip()}
