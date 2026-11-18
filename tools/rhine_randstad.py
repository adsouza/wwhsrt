"""Join the pinned international railway paths to the released public network."""

from math import hypot

from shapely.geometry import LineString, Point
from shapely.ops import substring


def repair_station_approaches(source, rows):
    # These released lines used suburban anchors. Join them at the main station
    # too, retaining the economic inputs and capacity IDs established in saves.
    for co in rows:
        for a in ("from_city", "to_city"):
            city_id = co[a]
            if city_id not in ("koln", "brussels"):
                continue
            hub = source["hubs"][city_id]
            pin = (hub["lon"], hub["lat"])
            alignment = co["alignments"][0]
            coords = [(p["lon"], p["lat"]) for p in alignment["path"]]
            if a == "to_city":
                coords.reverse()
            line = LineString(coords)
            if line.distance(Point(pin)) < 0.003:
                tail = substring(line, line.project(Point(pin)), line.length)
                coords = [pin, *list(tail.coords)[1:]]
            else:
                # The southern Cologne approaches diverge at Köln West.
                assert city_id == "koln"
                lead = source["cologne_west_approach"]["path"]
                i = min(
                    range(len(coords)), key=lambda j: hypot(coords[j][0] - lead[-1][0], coords[j][1] - lead[-1][1])
                )
                assert hypot(coords[i][0] - lead[-1][0], coords[i][1] - lead[-1][1]) < 0.001
                coords = [pin, *lead[1:], *coords[i + 1 :]]
            alignment["path"] = [dict(lon=x, lat=y) for x, y in (coords if a == "from_city" else reversed(coords))]
