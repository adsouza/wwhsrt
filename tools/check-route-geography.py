"""Check every shipped business alignment against pinned regional water geometry.

Requires shapely==2.1.2. No downloads, game writes or runtime route generation.
"""
import hashlib
import json
import math
from pathlib import Path

from shapely.geometry import LineString, Polygon, box, shape
from shapely.ops import transform, unary_union

ROOT = Path(__file__).resolve().parents[1]


def sha(path):
    return hashlib.sha256(path.read_bytes()).hexdigest()


def project(lon, lat):
    # Equal Earth coordinates: check the straight segments the renderer draws,
    # not just the different straight segments in unprojected lon/lat space.
    t = math.asin(math.sqrt(3) / 2 * math.sin(math.radians(lat)))
    t2, t6 = t*t, t**6
    return (2*math.sqrt(3)*math.radians(lon)*math.cos(t) /
            (3*(1.340264-3*.081106*t2+t6*(7*.000893+9*.003796*t2))),
            t*(1.340264-.081106*t2+t6*(.000893+.003796*t2)))


def kilometres(points):
    total = 0
    for (lon, lat), (lon2, lat2) in zip(points, points[1:]):
        p, q = math.radians(lat), math.radians(lat2)
        a = math.sin((q-p)/2)**2 + math.cos(p)*math.cos(q)*math.sin(math.radians(lon2-lon)/2)**2
        total += 12742*math.asin(min(1, math.sqrt(a)))
    return total


def main():
    source = ROOT / 'data/geography/japan-route-mask.v1.json'
    meta = json.loads(source.with_suffix('.metadata.json').read_text())
    assert sha(source) == meta['output_sha256'], 'Route mask changed without provenance review'
    mask = json.loads(source.read_text())
    assert mask['schema_version'] == 1
    bounds = box(*mask['coverage_bbox'])
    land = shape(mask['detailed_land'])
    assert land.is_valid
    lakes = {x['id']: shape(x['geometry']) for x in mask['lakes']}
    coastline = ROOT / 'assets/terrain/land.json'
    coast_meta = json.loads(coastline.with_name('land.metadata.json').read_text())
    assert sha(coastline) == coast_meta['output_sha256'], 'Drawn coastline changed without provenance review'
    assert coast_meta['source_sha256'] == meta['coast_source_sha256']
    assert sha(ROOT/'assets/terrain/features.json') == meta['lake_reference_sha256']
    drawn = unary_union([Polygon(r) for r in json.loads(coastline.read_text())])
    spaces = [('geographic', lambda p: p, land, lakes),
              ('equal_earth_source', lambda p: [project(*x) for x in p], transform(project, land),
               {k: transform(project, v) for k, v in lakes.items()}),
              ('equal_earth_drawn', lambda p: [project(*x) for x in p], transform(project, drawn),
               {k: transform(project, v) for k, v in lakes.items()})]

    def check_path(points):
        line = LineString(points)
        if not bounds.covers(line):
            return ['outside qualified mask coverage']
        errors = []
        for name, convert, ground, waters in spaces:
            rendered = LineString(convert(points))
            if rendered.difference(ground).length > 1e-12:
                errors.append(name+': sea intersection')
            for key, water in waters.items():
                if rendered.intersection(water).length > 1e-12:
                    errors.append(name+': '+key+' intersection')
        return errors

    # Independent negative fixtures reproduce the old bay/lake shortcuts and
    # ensure missing mask coverage cannot silently become flat, buildable land.
    regressions = [
        ('Uchiura Bay', [(140.648,41.904),(140.38,42.514)], 'sea intersection'),
        ('Lake Biwa', [(136.25,35.3),(135.76,34.985)], 'lake_biwa intersection'),
        ('unknown coverage', [(0,0),(1,1)], 'coverage'),
    ]
    for name, points, expected in regressions:
        assert any(expected in e for e in check_path(points)), 'Negative fixture accepted: '+name

    cities = {c['id']: (c['lon'],c['lat']) for c in json.loads((ROOT/'assets/scenario/scenario.json').read_text())['cities']}
    business = ROOT/'assets/scenario/business.json'
    rows, failures = [], []
    for corridor in json.loads(business.read_text()):
        for alignment in corridor['alignments']:
            points = [(p['lon'],p['lat']) for p in alignment['path']]
            assert 2 <= len(points) <= 64
            assert all(math.isfinite(x) and math.isfinite(y) for x,y in points)
            assert points[0] == cities[corridor['from_city']] and points[-1] == cities[corridor['to_city']], 'Detached route endpoints'
            errors = check_path(points)
            # A coarse drawing can omit rail curves; it must not already be
            # longer than all the rail kilometres priced by the scenario.
            length = kilometres(points)
            if length > alignment['km']:
                errors.append('drawn route exceeds priced rail kilometres')
            row = {'corridor':corridor['id'],'alignment':alignment['id'],'existing':corridor['existing'],
                   'vertices':len(points),'drawn_geodesic_km':round(length,2),'scenario_rail_km':alignment['km'],'errors':errors}
            rows.append(row)
            if errors:
                failures.append(row)
    assert rows, 'Empty route catalog'
    if failures:
        raise AssertionError(json.dumps(failures,indent=2))
    result = {'result':'passed','business_sha256':sha(business),'mask_sha256':sha(source),'drawn_coast_sha256':sha(coastline),
              'routes':rows,'checks':['All shipped paths, including existing lines and both alternatives, connect their endpoints',
              'No mapped sea/lake or conservative lake-envelope crossings in geographic and Equal Earth space',
              'Both source shoreline and actually rendered coast checked; unknown coverage rejected',
              'Old Uchiura Bay and Lake Biwa shortcuts rejected by negative fixtures',
              'Drawn geodesic lengths do not exceed priced scenario rail kilometres'],
              'limitations':['Cartographic regional review, not a surveyed engineering alignment or gradient/curvature qualification.',
              'Natural Earth coastline and Lake Biwa plus authored avoidance envelopes for Toya, Shikotsu and Onuma; small rivers/other unmapped waters are outside this mask.',
              'There are no approved sea/lake crossing exceptions in the current scenario. Future exceptions need explicit reviewed engineering, not a generic water bypass.']}
    (ROOT/'docs/validation/route-geography-checks.json').write_text(json.dumps(result,indent=2)+'\n')
    print('Passed geographic and Equal Earth water checks for',len(rows),'current alignments; negative fixtures rejected')


if __name__ == '__main__':
    main()
