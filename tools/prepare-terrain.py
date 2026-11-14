"""Build display-only shaded relief and contours from pinned Terrarium tiles.

Developer tool; runtime builds use the committed assets and perform no downloads.
Dependencies: tools/requirements-terrain.txt. Inputs cache under build/terrain-source.
On subsequent runs, downloaded/cached tiles must match the recorded source hashes.
"""
from datetime import datetime, timezone
from io import BytesIO
import hashlib
import json
import math
from pathlib import Path
import subprocess

import numpy as np
from PIL import Image
from shapely.geometry import LineString
from skimage.measure import find_contours

ROOT = Path(__file__).resolve().parents[1]
OUT = ROOT / "assets/terrain"
CACHE = ROOT / "build/terrain-source"
ZOOM = 8
AREAS = {"hokkaido": (227, 93, 228, 95), "kansai": (224, 100, 225, 101)}


def latitude(y):
    return math.degrees(math.atan(math.sinh(math.pi * (1 - 2 * y / 2**ZOOM))))


def location(px, py, x0, y0):
    return [latitude(y0 + py / 256), (x0 + px / 256) / 2**ZOOM * 360 - 180]


OUT.mkdir(parents=True, exist_ok=True)
CACHE.mkdir(parents=True, exist_ok=True)
previous = OUT / "terrain.json"
pins = {}
if previous.exists():
    for region in json.loads(previous.read_text())["regions"]:
        pins.update({s["url"]: s for s in region["sources"]})
regions = []
for name, (x0, y0, x1, y1) in AREAS.items():
    w, h = (x1 - x0 + 1) * 256, (y1 - y0 + 1) * 256
    heights = np.zeros((h, w), dtype=np.float64)
    sources = []
    for y in range(y0, y1 + 1):
        for x in range(x0, x1 + 1):
            url = f"https://s3.amazonaws.com/elevation-tiles-prod/terrarium/{ZOOM}/{x}/{y}.png"
            cached = CACHE / f"{ZOOM}-{x}-{y}.png"
            header_file = cached.with_suffix(".headers")
            headers = {}
            if cached.exists():
                raw = cached.read_bytes()
            else:
                subprocess.run(["curl","--fail","--silent","--show-error","--location","--max-time","30","--dump-header",str(header_file),"--output",str(cached),url],check=True)
                raw=cached.read_bytes()
            if header_file.exists():
                headers={k.lower().strip():v.strip() for line in header_file.read_text().splitlines() if ":" in line for k,v in [line.split(":",1)]}
            sha = hashlib.sha256(raw).hexdigest()
            if url in pins and pins[url]["sha256"] != sha:
                raise SystemExit(f"Source changed: {url}; review before repinning")
            im = np.array(Image.open(BytesIO(raw)).convert("RGB"), dtype=np.float64)
            if im.shape != (256, 256, 3):
                raise SystemExit(f"Unexpected tile dimensions: {url}")
            elevation = im[:, :, 0] * 256 + im[:, :, 1] + im[:, :, 2] / 256 - 32768
            heights[(y-y0)*256:(y-y0+1)*256, (x-x0)*256:(x-x0+1)*256] = elevation
            metadata = pins.get(url, {})
            sources.append({"url": url, "sha256": sha,
                "imagery_sources": headers.get("x-amz-meta-x-imagery-sources", metadata.get("imagery_sources", "SRTM, GMTED2010, ETOPO1")),
                "last_modified": headers.get("last-modified", metadata.get("last_modified", ""))})
    center_lat = latitude((y0+y1+1)/2)
    metres = math.cos(math.radians(center_lat)) * 2 * math.pi * 6378137 / (256 * 2**ZOOM)
    dy, dx = np.gradient(heights, metres)
    nx, ny, nz = -dx * 2.5, dy * 2.5, np.ones_like(heights)
    normal = np.sqrt(nx*nx + ny*ny + nz*nz)
    shade = np.clip(.74 + .34 * ((nx*-.5 + ny*.5 + nz*.7071) / normal), .55, 1.10)
    stops = [0, 150, 400, 800, 1200, 1800, 2600]
    colors = np.array([[231,244,204],[218,231,174],[199,211,145],[189,190,135],[183,168,133],[177,157,139],[234,230,216]])
    rgb = np.stack([np.interp(np.maximum(heights,0), stops, colors[:,i]) for i in range(3)], axis=-1)
    rgb = np.clip(rgb * shade[:,:,None],0,255).astype(np.uint8)
    # Only positive elevations are tinted. Sea/bathymetry and unknown areas are
    # transparent, never interpreted as free engineering surface.
    edge_y, edge_x = np.minimum(np.arange(h),np.arange(h)[::-1]), np.minimum(np.arange(w),np.arange(w)[::-1])
    fade = np.minimum(1,np.minimum(edge_y[:,None],edge_x[None,:])/12)
    alpha = (np.where(heights>1,225,0)*fade).astype(np.uint8)
    pixels = np.dstack([rgb,alpha])
    file = name+".png"
    Image.fromarray(pixels).save(OUT/file, optimize=True)
    contours=[]
    for elevation in [500,1000,1500]:
        for line in find_contours(heights[::4,::4],elevation):
            if len(line)<8: continue
            simplified=LineString([(p[1]*4,p[0]*4) for p in line]).simplify(1.5)
            points=[location(px,py,x0,y0) for px,py in simplified.coords]
            contours.append({"metres":elevation,"points":[[round(a,6),round(b,6)] for a,b in points]})
    regions.append({"id":name,"file":file,"sha256":hashlib.sha256((OUT/file).read_bytes()).hexdigest(),
        "zoom":ZOOM,"x0":x0,"y0":y0,"tiles_x":x1-x0+1,"tiles_y":y1-y0+1,
        "width":w,"height":h,"metres_per_pixel_at_center":round(metres),"sources":sources,"contours":contours})

manifest={"schema_version":1,"role":"Display context only; never pricing, feasibility or route generation.",
    "accessed_on":datetime.now(timezone.utc).date().isoformat(),
    "source":"Mapzen Terrain Tiles / USGS SRTM + GMTED2010, NOAA ETOPO1",
    "source_documentation":"https://github.com/tilezen/joerd/blob/master/docs/formats.md",
    "attribution":"Elevation: Mapzen / USGS; bathymetry: NOAA. All overlays reprojected to Equal Earth.",
    "regions":regions}
(OUT/"terrain.json").write_text(json.dumps(manifest,separators=(",",":"))+"\n")
print("Prepared",len(regions),"terrain regions from",sum(len(r["sources"]) for r in regions),"pinned tiles")

# Shoreline preparation is shared with the worldwide basemap and can also run
# independently, without rebuilding or downloading relief tiles.
subprocess.run(["python3",str(ROOT/"tools/prepare-shorelines.py")],check=True)
