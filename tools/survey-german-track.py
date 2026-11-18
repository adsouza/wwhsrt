"""Refresh the German track profiles using the pinned DB InfraGO archive and Natural Earth mask.

Run with the archive path after preparing the terrain sources. Requires shapely, numpy and pyproj.
The public-network compiler consumes the checked-in profiles without these survey inputs.
"""

import argparse
import collections
import csv
import gzip
import hashlib
import io
import json
from pathlib import Path
import zipfile

import numpy as np
from pyproj import Transformer
import shapely as geo
from shapely.geometry import shape
from shapely.ops import transform
from german_track import path_hash

ROOT = Path(__file__).resolve().parents[1]
ARCHIVE_SHA = "1fa5267cab954d8393878f7bb2396bf9ebb16f914701733c145c073f7e707f36"
# New alignments identified by DB line number. For IDs spanning multiple
# projects, only the documented high-speed new-build portion is admitted.
NEW = {"1733", "2690", "4080", "5934"}
NEW_FAST = {"5919", "6185", "4813", "4280"}
# Existing main lines with upgraded fast sections. On these lines the local
# operating limit identifies the fast portions of the playable alignment.
MAIN = set(
    "1283 1700 1710 1720 2200 2600 2650 3600 4010 4020 5300 5302 5501 5503 5850 5910 6100 6107 6132 6135 6340 6363 6411".split()
)


def track_class(number, speed):
    if number in NEW:
        return "dedicated"
    # The Nantenbach connecting curve has a 200 km/h rating on new track.
    # Neither label for fast track describes its construction and standard.
    if number == "5216" or not speed.isdigit():
        return "unresolved"
    speed = int(speed)
    if number in NEW_FAST:
        return "dedicated" if speed >= 250 else "unresolved"
    if speed >= 200:
        return "upgraded" if number in MAIN else "unresolved"
    return "conventional"


def load_tracks(archive):
    assert hashlib.sha256(archive.read_bytes()).hexdigest() == ARCHIVE_SHA, (
        "Unexpected DB edition"
    )
    tracks, classes = [], []
    with zipfile.ZipFile(archive) as z, z.open("M1 Streckennetz.csv") as stream:
        for row in csv.DictReader(
            io.TextIOWrapper(stream, encoding="utf-8-sig"), delimiter=";"
        ):
            if (
                row["DB Betrieb"] != "In DB Netz Betrieb"
                or row["Bauzustand"] == "im Bau"
            ):
                continue
            wkt = row["WKT UTM 32N (EPSG 25832)"]
            if not wkt:
                continue
            line = geo.from_wkt(wkt)
            if line.geom_type == "LineString" and line.length >= 1:
                tracks.append(line)
                classes.append(
                    track_class(
                        row["Streckennummer"], row["Geschwindigkeit"].split(" ")[0]
                    )
                )
    return geo.STRtree(tracks), classes


def survey(points, border, project, tree, classes):
    coords = np.asarray(project(*np.asarray(points).T)).T
    vectors = np.diff(coords, axis=0)
    lengths = np.linalg.norm(vectors, axis=1)
    seg, start, end = [], [], []
    for i, length in enumerate(lengths):
        count = int(np.ceil(length / 200))
        for j in range(count):
            seg.append(i)
            start.append(j / count)
            end.append((j + 1) / count)
    seg, start, end = map(np.asarray, (seg, start, end))
    samples = geo.points(coords[seg] + vectors[seg] * ((start + end) / 2)[:, None])
    inside = geo.covers(border, samples)
    si, ti = tree.query(samples[inside], predicate="dwithin", distance=80)
    si = np.flatnonzero(inside)[si]
    rails = tree.geometries[ti]
    offset = geo.line_locate_point(rails, samples[si])
    low = geo.line_interpolate_point(rails, np.maximum(0, offset - 10))
    high = geo.line_interpolate_point(rails, np.minimum(geo.length(rails), offset + 10))
    bearings = geo.get_coordinates(high) - geo.get_coordinates(low)
    cosine = abs((bearings * vectors[seg[si]]).sum(axis=1)) / (
        np.linalg.norm(bearings, axis=1) * lengths[seg[si]] + 1e-8
    )
    candidates = collections.defaultdict(list)
    for i, j, d in zip(
        si[cosine >= 0.9],
        ti[cosine >= 0.9],
        geo.distance(rails, samples[si])[cosine >= 0.9],
    ):
        candidates[i].append((d, classes[j]))
    meters, runs = collections.Counter(), []
    for i in np.flatnonzero(inside):
        choices = candidates[i]
        closest = min((d for d, _ in choices), default=0)
        kinds = {k for d, k in choices if d <= closest + 12}
        kind = kinds.pop() if len(kinds) == 1 else "unresolved"
        meters[kind] += lengths[seg[i]] * (end[i] - start[i])
        a, b = (int(seg[i]) * 10000 + round(f * 10000) for f in (start[i], end[i]))
        if runs and runs[-1][0] == kind and runs[-1][2] == a:
            runs[-1][2] = b
        else:
            runs.append([kind, a, b])
    return dict(
        path_sha256=path_hash(points),
        meters={k: round(v) for k, v in sorted(meters.items())},
        runs=runs,
    )


def main():
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument("archive", type=Path)
    parser.add_argument(
        "--check",
        action="store_true",
        help="Verify the existing profile without writing",
    )
    args = parser.parse_args()
    tree, classes = load_tracks(args.archive)
    mask = ROOT / "build/terrain-source/ne_10m_admin_0_countries.geojson"
    boundary = mask.read_bytes()
    boundary_sha = hashlib.sha256(boundary).hexdigest()
    metadata = json.loads(
        (ROOT / "assets/terrain/world-land.metadata.json").read_bytes()
    )
    assert boundary_sha == metadata["source_countries_sha256"], (
        "Unexpected country boundary edition"
    )
    border = next(
        shape(f["geometry"])
        for f in json.loads(boundary)["features"]
        if f["properties"]["ADM0_A3"] == "DEU"
    )
    project = Transformer.from_crs(4326, 25832, always_xy=True).transform
    projected = transform(project, border)
    # The original inventory is pinned even after a conventional path leaves the map.
    archived = ROOT / "assets/scenario/public-network-v8.json.gz"
    catalog = json.loads(gzip.decompress(archived.read_bytes()))
    display = json.loads(
        (ROOT / "assets/scenario/rail-display-paths.json").read_bytes()
    )
    lines = {}
    for co in catalog["corridors"]:
        if co["existing"]:
            alt = co["alignments"][0]
            path = display.get(co["id"], {}).get(alt["id"], alt["path"])
            lines[co["id"]] = [[p["lon"], p["lat"]] for p in path]
    hamm = json.loads((ROOT / "data/imports/hamburg-hamm.v1.json").read_bytes())
    reviewed = survey(hamm["path"], projected, project, tree, classes)
    assert reviewed["meters"] == hamm["meters"] and hamm["db_sha256"] == ARCHIVE_SHA, (
        "Refresh Hamburg–Hamm survey"
    )
    profiles = {
        id: survey(points, projected, project, tree, classes)
        for id, points in lines.items()
        if border.intersects(geo.LineString(points))
    }
    output = dict(
        db_sha256=ARCHIVE_SHA,
        boundary_sha256=boundary_sha,
        profiles=profiles,
    )
    target = ROOT / "data/imports/german-track.v1.json"
    serialized = json.dumps(output, separators=(",", ":")) + "\n"
    if args.check:
        if target.read_text() != serialized:
            raise SystemExit("German track profiles differ")
    else:
        target.write_text(serialized)
    print(
        f"{len(profiles)} German track profiles verified"
        if args.check
        else f"Wrote {len(profiles)} German track profiles"
    )


if __name__ == "__main__":
    main()
