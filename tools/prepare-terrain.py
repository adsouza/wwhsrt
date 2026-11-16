"""Build the display-only relief tiles from pinned Mapzen Terrarium tiles.

Developer tool; ordinary builds use the committed tiles and download nothing. The rules (which tiles,
shading, land) are in tools/terrain_rule.py. Needs tools/requirements-terrain.txt and
tools/requirements-geography.txt in an isolated environment.

    prepare-terrain.py --plan      print how many tiles each level has and how many are not yet cached
    prepare-terrain.py --fetch     download the missing source tiles to build/terrain-source
    prepare-terrain.py --survey    list the elevation datasets the cached tiles cite; fail on an uncredited one
    prepare-terrain.py --write     write assets/terrain/tiles and assets/terrain/terrain.json
    prepare-terrain.py --write --only CORRIDOR ... --box W S E N ...
                                   write a sample: level 8 only for those buildable corridors and boxes

Every source tile is pinned by SHA-256 in data/terrain/sources.v1.json. A tile that no longer matches
its pin stops the run: review the change before repinning.
"""

from concurrent.futures import ThreadPoolExecutor
import argparse
import collections
import hashlib
import importlib.util
import io
import json
import pathlib
import subprocess
import sys

import numpy as np
from PIL import Image
from shapely.geometry import LineString
from skimage.measure import find_contours

sys.path.insert(0, str(pathlib.Path(__file__).resolve().parent))
import basemap_rule as rule  # noqa: E402
import terrain_rule as T  # noqa: E402

ROOT = pathlib.Path(__file__).resolve().parents[1]
OUT = ROOT / "assets/terrain"
CACHE = ROOT / "build/terrain-source"
PINS = ROOT / "data/terrain/sources.v1.json"
BASEMAP = OUT / "world-land.json"
ATTRIBUTION = OUT / "ATTRIBUTION.md"
SOURCE = "https://s3.amazonaws.com/elevation-tiles-prod/terrarium/{}/{}/{}.png"


def basemap_tool():
    spec = importlib.util.spec_from_file_location("prepare_basemap", ROOT / "tools/prepare-basemap.py")
    module = importlib.util.module_from_spec(spec)
    spec.loader.exec_module(module)
    return module


def routes_and_lines():
    """The buildable routes as (corridor, alignment, path), and every line's path: existing ones and routes."""
    _, existing, routes = basemap_tool().inputs()
    routes = [(corridor, alignment, path) for corridor, alignment, path, _ in routes]
    return routes, existing + [path for _, _, path in routes]


def drawn_land():
    return rule.drawn_geometry(json.loads(BASEMAP.read_text()))


def tiles_of(levels):
    return sorted((z, x, y) for z, tiles in levels.items() for x, y in tiles)


def load_pins():
    return json.loads(PINS.read_text())["tiles"] if PINS.exists() else {}


def save_pins(pins):
    PINS.parent.mkdir(parents=True, exist_ok=True)
    PINS.write_text(json.dumps({"version": 1, "tiles": dict(sorted(pins.items()))}, indent=1) + "\n")


def key(z, x, y):
    return f"{z}/{x}/{y}"


def cached(z, x, y):
    return CACHE / f"{z}-{x}-{y}.png"


def headers(z, x, y):
    file = cached(z, x, y).with_suffix(".headers")
    if not file.exists():
        return {}
    return {
        name.lower().strip(): value.strip()
        for line in file.read_text().splitlines()
        if ":" in line
        for name, value in [line.split(":", 1)]
    }


def source(z, x, y, pins, download):
    """The bytes of one source tile, checked against its pin; downloaded first if allowed and missing."""
    file, url = cached(z, x, y), SOURCE.format(z, x, y)
    if not file.exists():
        if not download:
            raise SystemExit(f"Source tile {key(z, x, y)} is not cached; run --fetch")
        partial = file.with_suffix(".part")
        subprocess.run(
            ["curl", "--fail", "--silent", "--show-error", "--location", "--max-time", "60", "--retry", "3"]
            + ["--dump-header", str(file.with_suffix(".headers")), "--output", str(partial), url],
            check=True,
        )
        partial.rename(file)
    raw = file.read_bytes()
    digest = hashlib.sha256(raw).hexdigest()
    pin = pins.get(key(z, x, y))
    if pin and pin["sha256"] != digest:
        raise SystemExit(f"Source changed: {url}; review before repinning")
    return raw, digest


def pin_of(z, x, y, digest, old):
    head = headers(z, x, y)
    return {
        "url": SOURCE.format(z, x, y),
        "sha256": digest,
        "imagery_sources": head.get("x-amz-meta-x-imagery-sources", (old or {}).get("imagery_sources", "")),
        "last_modified": head.get("last-modified", (old or {}).get("last_modified", "")),
    }


def plan(levels):
    for z in sorted(levels):
        print(f"level {z}: {len(levels[z])} tiles")
    tiles = tiles_of(levels)
    missing = [t for t in tiles if not cached(*t).exists()]
    print(f"{len(tiles)} source tiles, {len(missing)} not cached, from {SOURCE.format('z', 'x', 'y')}")


def fetch(levels):
    CACHE.mkdir(parents=True, exist_ok=True)
    pins = load_pins()
    tiles = tiles_of(levels)

    def one(tile):
        return tile, source(*tile, pins, True)[1]

    with ThreadPoolExecutor(8) as pool:
        for done, (tile, digest) in enumerate(pool.map(one, tiles), 1):
            pins[key(*tile)] = pin_of(*tile, digest, pins.get(key(*tile)))
            if done % 200 == 0:
                print(f"{done} of {len(tiles)}", flush=True)
    wanted = {key(*tile) for tile in tiles}
    save_pins({name: pin for name, pin in pins.items() if name in wanted})
    size = sum(cached(*t).stat().st_size for t in tiles)
    print(f"{len(tiles)} source tiles cached and pinned, {size / 1e6:.0f} MB")


def credit_gaps(citations, attribution_text):
    """Count the datasets the citations name, and return what has no credit.

    A source with no dataset header is a gap too: it cannot be shown to be credited.
    """
    counts, unknown = collections.Counter(), 0
    for cited in citations:
        names = T.datasets(cited or "")
        unknown += not names
        counts.update(names)
    gaps = T.uncredited(set(counts), attribution_text)
    if unknown:
        gaps.add("a source tile with no dataset header")
    return dict(counts), gaps


def survey(levels):
    """Print each cited dataset with its tile count. Returns what has no credit."""
    pins, citations = load_pins(), []
    for tile in tiles_of(levels):
        cited = headers(*tile).get("x-amz-meta-x-imagery-sources") or pins.get(key(*tile), {}).get("imagery_sources")
        if cited is None:
            raise SystemExit(f"Source tile {key(*tile)} is not cached; run --fetch")
        citations.append(cited)
    counts, gaps = credit_gaps(citations, ATTRIBUTION.read_text())
    for name, count in sorted(counts.items(), key=lambda item: -item[1]):
        print(f"{name}: {count} tiles")
    if gaps:
        print("uncredited:", ", ".join(sorted(gaps)))
    return gaps


def write_tile(out, z, x, y, rgb, mask):
    """Write one tile's files under out and return its manifest entry.

    A sea tile has no files. A coast tile has a one-bit mask beside its JPEG, and its sea pixels carry
    the lowland colour so compression has no hard edge to smear onto the land.
    """
    kind = T.kind(mask)
    entry = [z, x, y, kind]
    if kind == "sea":
        return entry
    folder = out / str(z) / str(x)
    folder.mkdir(parents=True, exist_ok=True)
    pixels = rgb.copy()
    pixels[~mask] = T.LOWLAND
    Image.fromarray(pixels, "RGB").save(folder / f"{y}.jpg", quality=T.JPEG_QUALITY, optimize=True)
    entry.append(hashlib.sha256((folder / f"{y}.jpg").read_bytes()).hexdigest())
    if kind == "coast":
        Image.fromarray(mask).convert("1").save(folder / f"{y}.png", optimize=True)
        entry.append(hashlib.sha256((folder / f"{y}.png").read_bytes()).hexdigest())
    return entry


def with_halo(heights, z, x, y, width=T.HALO):
    """A tile's source heights with width pixels around them: from each neighbour heights has, else its own edge."""
    padded = np.pad(heights[(x, y)], width, mode="edge")
    n = 2**z
    mine = {-1: slice(0, width), 0: slice(width, width + 256), 1: slice(width + 256, None)}
    theirs = {-1: slice(256 - width, None), 0: slice(None), 1: slice(0, width)}
    for dx in (-1, 0, 1):
        for dy in (-1, 0, 1):
            other = heights.get(((x + dx) % n, y + dy))  # columns wrap at the date line; rows do not
            if other is not None and (dx, dy) != (0, 0):
                padded[mine[dy], mine[dx]] = other[theirs[dy], theirs[dx]]
    return padded


def selected_level8(routes, corridors, boxes, available):
    """Level-8 tiles for a sample: near the named buildable corridors, and those of the full list in the boxes."""
    known = {corridor for corridor, _, _ in routes}
    missing = sorted(set(corridors) - known)
    if missing:
        raise SystemExit("Not a buildable corridor: " + ", ".join(missing))
    tiles = T.level8_tiles([path for corridor, _, path in routes if corridor in corridors])
    for west, south, east, north in boxes:
        x0, y0 = T.tile_at(8, west, north)
        x1, y1 = T.tile_at(8, east, south)
        tiles.update((x, y) for x in range(x0, x1 + 1) for y in range(y0, y1 + 1) if (x, y) in available)
    return tiles


def contours():
    """The 500, 1,000 and 1,500 m contours of the two Japan boxes, as before the tile pyramid."""
    pins, regions = load_pins(), []
    for name, (z, x0, y0, x1, y1) in zip(("hokkaido", "kansai"), rule.JAPAN_BOXES):
        heights = np.block(
            [[decode(z, x, y, pins) for x in range(x0, x1 + 1)] for y in range(y0, y1 + 1)]
        ).astype(np.float64)
        lines = []
        for metres in (500, 1000, 1500):
            for line in find_contours(heights[::4, ::4], metres):
                if len(line) < 8:
                    continue
                simplified = LineString([(p[1] * 4, p[0] * 4) for p in line]).simplify(1.5)
                points = [T.place(z, x0, y0, col, row) for col, row in simplified.coords]
                lines.append({"metres": metres, "points": [[round(lat, 6), round(lon, 6)] for lon, lat in points]})
        regions.append({"id": name, "contours": lines})
    return regions


def decode(z, x, y, pins):
    raw, _ = source(z, x, y, pins, False)
    pixels = np.array(Image.open(io.BytesIO(raw)).convert("RGB"), dtype=np.float64)
    if pixels.shape != (256, 256, 3):
        raise SystemExit(f"Unexpected tile dimensions: {key(z, x, y)}")
    return T.elevation(pixels).astype(np.float32)


def remove_unlisted(out, entries):
    """Delete tile files the entries do not name, and folders left empty."""
    listed = set()
    for z, x, y, *hashes in (entry[:3] + entry[4:] for entry in entries):
        listed.update(out / str(z) / str(x) / f"{y}{suffix}" for suffix, _ in zip((".jpg", ".png"), hashes))
    for file in sorted(out.rglob("*"), reverse=True):
        if file.is_file() and file not in listed:
            file.unlink()
        elif file.is_dir() and not any(file.iterdir()):
            file.rmdir()


def write(levels, complete):
    pins, land = load_pins(), T.densified(drawn_land())
    tiles_out = OUT / "tiles"
    entries, counts, above = [], collections.Counter(), {}
    for z in sorted(levels):
        # A level's own heights, and how far the level above lies over them (see terrain_rule.ice_weight).
        own = {(x, y): decode(z, x, y, pins) for x, y in sorted(levels[z])}
        gaps = {
            (x, y): (np.maximum(T.from_parent(above[(x >> 1, y >> 1)], x, y), 0) - np.maximum(tile, 0)).astype(np.float32)
            for (x, y), tile in own.items()
            if (x >> 1, y >> 1) in above
        }
        heights, iced = {}, []
        for (x, y), tile in own.items():
            heights[(x, y)] = tile
            if (x, y) in gaps:
                weight = T.ice_weight(with_halo(gaps, z, x, y, T.ICE_WINDOW // 2))
                if weight.any():
                    heights[(x, y)] = (np.maximum(tile, 0) + weight * gaps[(x, y)]).astype(np.float32)
                    iced.append(f"{x}/{y} ({100 * weight.mean():.0f}%)")
        del own, gaps
        for x, y in sorted(heights):
            samples = T.overlapped(with_halo(heights, z, x, y))
            mask = T.land_mask(land, z, x, y)
            rgb = T.shade(samples, z, y)
            entry = write_tile(tiles_out, z, x, y, rgb, mask)
            entries.append(entry)
            counts[(z, entry[3])] += 1
        under_ice = f"; {len(iced)} take some surface from level {z - 1}: {' '.join(iced)}" if iced else ""
        print(f"level {z}: " + ", ".join(f"{counts[(z, k)]} {k}" for k in ("land", "coast", "sea")) + under_ice, flush=True)
        above = heights
    remove_unlisted(tiles_out, entries)
    manifest = {
        "schema_version": 2,
        "role": "Display context only; never pricing, feasibility or route generation.",
        "source": "Mapzen Terrain Tiles / USGS SRTM + GMTED2010, NOAA ETOPO1",
        "source_documentation": "https://github.com/tilezen/joerd/blob/master/docs/formats.md",
        "attribution": "Elevation: Mapzen / USGS; bathymetry: NOAA. All overlays reprojected to Equal Earth.",
        "basemap_sha256": hashlib.sha256(BASEMAP.read_bytes()).hexdigest(),
        "complete": complete,
        "tiles": entries,
        "regions": contours(),
    }
    (OUT / "terrain.json").write_text(json.dumps(manifest, separators=(",", ":")) + "\n")
    size = sum(f.stat().st_size for f in tiles_out.rglob("*") if f.is_file())
    print(f"{len(entries)} tiles listed, {size / 1e6:.1f} MB of tile files")


def main():
    parser = argparse.ArgumentParser(description=__doc__, formatter_class=argparse.RawDescriptionHelpFormatter)
    mode = parser.add_mutually_exclusive_group(required=True)
    for name in ("plan", "fetch", "survey", "write"):
        mode.add_argument("--" + name, action="store_true")
    parser.add_argument("--only", nargs="+", default=[], metavar="CORRIDOR")
    parser.add_argument("--box", nargs=4, type=float, action="append", default=[], metavar=("W", "S", "E", "N"))
    args = parser.parse_args()
    routes, lines = routes_and_lines()
    levels = T.tile_list(lines, drawn_land())
    sample = bool(args.only or args.box)
    if sample:
        if not args.write:
            parser.error("--only and --box go with --write")
        levels[8] = selected_level8(routes, args.only, args.box, levels[8])
        levels[7] = T.parents(levels[8])
        levels[6] = T.parents(levels[7])
    if args.plan:
        plan(levels)
    elif args.fetch:
        fetch(levels)
    elif survey(levels):
        raise SystemExit(1)
    elif args.write:
        write(levels, not sample)


if __name__ == "__main__":
    main()
