# World data inputs

## Prototype basemap

[ui/theme.v1.json](ui/theme.v1.json) records the cheerful light/dark palette and semantic color roles. The [presentation contract](../docs/PRESENTATION.md) defines its contrast targets and construction/train animation and sound behavior.

[prototypes/map-region-presets.v1.json](prototypes/map-region-presets.v1.json) records the user-defined region labels, landscape/portrait metro anchor sets, latitude cushion, and timezone-selected initial portrait views for map framing. Bounds are derived from raw metro coordinates at runtime; no metro records are renamed or edited by these presentation presets.

[prototypes/natural-earth-110m.countries.geojson](prototypes/natural-earth-110m.countries.geojson) contains 177 Natural Earth country geometries and names for the Equal Earth world/region mockup. Its [manifest](prototypes/natural-earth-110m.metadata.json) records the public-domain source, source/processed hashes, snapshot date, and transformation. This is overview artwork, not qualified terrain, station positions, rail topology, or production regional detail. The raw seed preserves all 96 records. The [playable catalog](prototypes/playable-metros.v1.json) retains the original values of 93 metros with populations of at least 5M, excluding Manila, Taipei, and Brasília according to the [selection policy](prototypes/playable-metro-selection.v1.json). Display names omit a trailing “Metropolitan Area” without changing source names; [validation evidence](../docs/validation/README.md) describes projection and raw-data checks.

## Major-metro seed

[raw/metropolitan_areas_5m.json](raw/metropolitan_areas_5m.json) is an unchanged copy of the user's supplied file, received on 2026-11-08. The [sidecar manifest](raw/metropolitan_areas_5m.metadata.json) records its SHA-256, units, checked properties, and unknown metadata. Preserve this snapshot; derived game catalogs should carry explicit mappings and transformation versions.

One record represents a labeled metro or urban region, with an ID, country label, representative latitude/longitude, population in people, and estimated GDP in USD billions. It is a candidate anchor list for the game map. Smaller cities and existing rail stops remain necessary to complete playable corridors.

The 96 records pass the basic field, uniqueness, numeric, and coordinate-range checks below. There are 94 populations strictly above 5M. Phoenix is exactly 5M and Brasília is 4.8M; both source records remain unchanged. The playable catalog excludes Brasília for being under 5M and retains Phoenix. Passing these checks does not verify the estimates, map accuracy, worldwide completeness, or non-overlap of metropolitan markets.

Population and GDP reference years, upstream sources, GDP basis/price year, coordinate reference system, and redistribution terms are not present in the supplied file. GDP is described as estimated by the user. Use unknown values in metadata until qualified. In particular, do not treat receipt date as observation year or assume that USD implies a common nominal/PPP basis.

Before using the file for demand totals, review geographic coverage and potential overlap between broad regional labels and nearby city records. Coordinates are map markers, not surveyed station sites. Preserve country labels as raw input and normalize them separately for scenario regions and concessions.

The [specification](../docs/DESIGN.md) describes the seed's role in the broader city catalog and conditional use of GDP in bounded demand indices. No game runtime loader or calibrated demand model is implemented yet.

## Construction-cost inputs

[costs/construction_costs.v1.json](costs/construction_costs.v1.json) records the `hsr-construction-2024usd-v1` model: 22 project/sample observations, primary/institutional source URLs, frozen CPI and market-FX inputs, four worked conventional-rail extrapolations, engineering factors, and 20 regional profiles. Country-label assignments cover the supplied metro seed explicitly.

Published costs and inferred game parameters are separate fields. Mixed-price observations are flagged as approximate, while year-of-expenditure forecasts and incomplete supplier shares are excluded from constant-price normalization. Regional point values and sensitivity bounds are design assumptions informed by evidence, not measured country averages or confidence intervals. See the [research appendix](../docs/CONSTRUCTION_COSTS.md) and the [spec's construction rules](../docs/DESIGN.md) for scope, equations, and interpretation.

## Reproduce the metro structural checks

Run from the repository root with Python's standard library:

```sh
python3 - <<'PY'
from pathlib import Path
import hashlib, json, math

p = Path('data/raw/metropolitan_areas_5m.json')
raw = p.read_bytes()
rows = json.loads(raw)
metadata = json.loads(p.with_name('metropolitan_areas_5m.metadata.json').read_text())
fields = {'id', 'name', 'country', 'lat', 'lng', 'population', 'gdp_usd_billion'}
assert len(rows) == metadata['record_count'] == 96
assert hashlib.sha256(raw).hexdigest() == metadata['sha256']
assert all(set(r) == fields for r in rows)
assert all(isinstance(r[k], str) and r[k].strip()
           for r in rows for k in ('id', 'name', 'country'))
for key in ('id', 'name'):
    assert len({r[key].strip().casefold() for r in rows}) == len(rows)
assert len({(r['lat'], r['lng']) for r in rows}) == len(rows)
assert all(type(r['population']) is int and r['population'] > 0 for r in rows)
assert all(type(r[k]) in (int, float) and math.isfinite(r[k])
           for r in rows for k in ('lat', 'lng', 'gdp_usd_billion'))
assert all(-90 <= r['lat'] <= 90 and -180 <= r['lng'] <= 180
           and r['gdp_usd_billion'] > 0 for r in rows)
exceptions = [{'id': r['id'], 'population': r['population']}
              for r in rows if r['population'] <= 5_000_000]
assert exceptions == metadata['strict_population_gt_5000000']['exceptions_retained']
assert sum(r['population'] > 5_000_000 for r in rows) == 94
policy = json.loads(Path('data/prototypes/playable-metro-selection.v1.json').read_text())
playable = json.loads(Path('data/prototypes/playable-metros.v1.json').read_text())
assert len(playable) == policy['playable_count'] == 93
assert playable == [r for r in rows if r['population'] >= policy['minimum_population']
                    and r['id'] not in policy['excluded_metro_ids']]
assert all(r['population'] >= 5_000_000 for r in playable)
print('Structural checks passed; geographic and economic estimates remain unverified.')
print('Threshold exceptions retained:', exceptions)
PY
```
