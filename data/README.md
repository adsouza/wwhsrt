# World data inputs

## Prototype basemap

[ui/theme.v1.json](ui/theme.v1.json) records the cheerful light/dark palette and semantic color roles. The [presentation contract](../docs/PRESENTATION.md) defines its contrast targets and construction/train animation and sound behavior.

[prototypes/map-region-presets.v1.json](prototypes/map-region-presets.v1.json) records the user-defined region labels, landscape/portrait metro anchor sets, latitude cushion, and timezone-selected initial portrait views for map framing. Bounds are derived from raw metro coordinates at runtime; no metro records are renamed or edited by these presentation presets.

[prototypes/launch-map-framing.v1.json](prototypes/launch-map-framing.v1.json) extends that base for the launch network preview. It records the framing changes for Frangistan, East Asia (Harbin as the northern anchor in both orientations), landscape North America, portrait SouthEast Asia, Australia, and Uzbekistan. Frangistan uses the OpenStreetMap station coordinates for Hörnefors and Cortadura as its latitude anchors in both orientations. Uzbekistan references the existing OpenStreetMap station IDs for Khanki, Bukhara (Kogon), Samarkand, and Tashkent Central. The major-metro preview uses the base manifest.

The launch network preview sources fast-track candidates and station geometry from [OpenStreetMap via QLever](https://wiki.openstreetmap.org/wiki/QLever). The [overlay audit](prototypes/hsr-overlay-audit.v1.json) records the reviewed exclusions. The remaining tracks are provisional candidates. Station candidates include tagged service stops and named rail stations within 400 m of the filtered track. The inspector gives the evidence for the selected station. Qualify starting infrastructure, commercial status, and service stops separately for production. This layer selects track by tags, not by the cities it joins. The game admits only corridors between catalog cities, as described in the [design's admission rule](../docs/DESIGN.md#supplementary-geography-sources).

The [KCIC Whoosh stop list](https://kcic.co.id/syarat-ketentuan-frequent-whooser-card/) identifies Halim, Karawang, Padalarang, and Tegalluar Summarecon. The preview displays the western terminus as **Jakarta (Halim)** at the existing Halim station coordinates.

The preview identifies **Tashkent Central**, **Samarkand**, and **Bukhara (Kogon)** as published Afrosiyob stops. It uses the existing OpenStreetMap station coordinates. The passenger operator identifies Kogon as the location of Bukhara's station. [Afrosiyob timetable](https://eticket.uzrailpass.uz/uz/pages/schedule) · [Bukhara station](https://uzrailpass.uz/vip/buxoro-temir-yol-vokzali)

The preview shows **Orlando (Brightline)** at the airport intermodal terminal. The mapped fast section begins east of the station, so the preview marks the terminal with a hollow dot. Cocoa's planned station is outside the existing station layer. [Brightline Orlando](https://www.gobrightline.com/orlando) · [Cocoa station announcement](https://www.gobrightline.com/press-room/2026/city-of-cocoa-space-coast-transportation-planning-organization-announce-historic-federal-grant-award-for-cocoa-multimodal-rail-station)

The Poland preview shows **Warsaw Central**, **Kraków Główny**, **Katowice**, and **Gdańsk Główny** as published passenger stops whose approaches fall outside the extracted fast-track geometry. Warsaw–Gdańsk has sections in service at up to 200 km/h. The Central Rail Line (CMK) toward Kraków and Katowice has previously carried 200 km/h service, with speeds reduced during upgrades and 250 km/h service pending. Use the scenario cutoff to qualify operating speeds. [Polish rail program](https://naszakolej.gov.pl/szybkosc/) · [PKP PLK station directory](https://portalpasazera.pl/en/KatalogStacji)

The Finland and Sweden follow-up checked the retained track components with no displayed stops. It restored **Parkano**, **Falkenberg**, and **Västeraspby** and added nearby published stops on the reviewed routes. Kouvola and Lappeenranta anchor the retained Kouvola–Luumäki track. Örebro Central and Arboga anchor the retained Mälarbanan track. The relevant station approaches are outside the fast-track extract. [VR timetables](https://www.vr.fi/en/timetables) · [Norrtåg timetable](https://norrtag.se/app/uploads/2026/10/2026-10-Tidtabell-Sundsvall-Umea-T26-Host.pdf) · [Öresundståg routes](https://www.oresundstag.se/en/about-oresundstag/where-do-oresundstag-operate) · [Mälartåg service](https://www.malardalstrafik.se/maelartaag/wifi-ombord/)

The [station connectivity review](prototypes/hsr-station-review.v1.csv) lists the 558 exceptions among 4,833 station candidates. The initial audit uses the 155,128 unsimplified ways remaining after the overlay exclusions. It joins ways at shared coordinates and associates stations within 400 m of a way. A candidate passes if its component also reaches another mapped station at least 5 km away in straight-line distance. This threshold screens out nearby platforms and stubs. Follow-up stations are checked against the retained components. A geometric match alone leaves service and city boundaries to be verified.

The preview shows 4,275 candidates with a mapped link as filled dots. It keeps 48 published stops without a mapped link as hollow dots and links to their sources. It hides 510 other markers, including San Severiano, a Renfe commuter stop. The CSV records why each exception was flagged and any independently checked evidence. Its component length is the sum of mapped way lengths, so parallel tracks can count separately.

The preview filters stationless track after the station review and before simplifying the linework. It keeps components within 400 m of a displayed station, separately documented passenger corridors, and nearby track reachable from either group across gaps of at most 150 m. The last rule preserves adjacent tracks and small breaks in the extract for display. It does not add track or establish a routable connection. The [overlay audit](prototypes/hsr-overlay-audit.v1.json) records the display counts, source hashes, and documented corridors.

[prototypes/natural-earth-110m.countries.geojson](prototypes/natural-earth-110m.countries.geojson) contains 177 Natural Earth country geometries and names for the Equal Earth world/region mockup. Its [manifest](prototypes/natural-earth-110m.metadata.json) records the public-domain source, source/processed hashes, snapshot date, and transformation. This is overview artwork, not qualified terrain, station positions, rail topology, or production regional detail. The raw seed preserves all 96 records. The [playable catalog](prototypes/playable-metros.v1.json) retains the original values of 94 metros with populations of at least 5M, excluding Manila and Brasília according to the [selection policy](prototypes/playable-metro-selection.v1.json). Display names omit a trailing “Metropolitan Area” without changing source names; [validation evidence](../docs/validation/README.md) describes projection and raw-data checks.

## Major-metro seed

[prototypes/connection-planner-demo.v1.json](prototypes/connection-planner-demo.v1.json) supplies two illustrative endpoint pairs, each with two alignment choices, for the connection-planning interaction preview. Monetary values are USD millions; forecast passenger counts are monthly. These are authored UI fixtures, not researched prices, qualified terrain alignments, corridor-catalog entries, or economic simulation outputs. Polyline endpoints use the original metro coordinates. Revenue equals passengers times the stated average fare; capacity is two directions × return trips/day × seats × operating days. The Lagos–Kinshasa example deliberately includes a negative operating contribution. The preview never extrapolates these numbers to unlisted pairs.

[raw/metropolitan_areas_5m.json](raw/metropolitan_areas_5m.json) is an unchanged copy of the user's supplied file, received on 2026-11-08. The [sidecar manifest](raw/metropolitan_areas_5m.metadata.json) records its SHA-256, units, checked properties, and unknown metadata. Preserve this snapshot; derived game catalogs should carry explicit mappings and transformation versions.

One record represents a labeled metro or urban region, with an ID, country label, representative latitude/longitude, population in people, and estimated GDP in USD billions. It is a candidate anchor list for the game map. Smaller cities and existing rail stops remain necessary to complete playable corridors.

The 96 records pass the basic field, uniqueness, numeric, and coordinate-range checks below. There are 94 populations strictly above 5M. Phoenix is exactly 5M and Brasília is 4.8M; both source records remain unchanged. The playable catalog excludes Brasília for being under 5M and retains Phoenix. Passing these checks does not verify the estimates, map accuracy, worldwide completeness, or non-overlap of metropolitan markets.

Population and GDP reference years, upstream sources, GDP basis/price year, coordinate reference system, and redistribution terms are not present in the supplied file. GDP is described as estimated by the user. Use unknown values in metadata until qualified. In particular, do not treat receipt date as observation year or assume that USD implies a common nominal/PPP basis.

Before using the file for demand totals, review geographic coverage and potential overlap between broad regional labels and nearby city records. Coordinates are map markers, not surveyed station sites. Preserve country labels as raw input and normalize them separately for scenario regions and concessions.

The [specification](../docs/DESIGN.md) describes the seed's role in the broader city catalog and conditional use of GDP in bounded demand indices. No game runtime loader or calibrated demand model is implemented yet.

## Existing-HSR stop-city candidates

[prototypes/hsr-stop-city-candidates.v1.csv](prototypes/hsr-stop-city-candidates.v1.csv) screens which cities the provisional fast-track station layer touches. It is input to the planned catalog of about 250 cities, which includes the 94 major metros. [tools/derive-hsr-stop-cities.py](../tools/derive-hsr-stop-cities.py) generates it, and the [metadata](prototypes/hsr-stop-city-candidates.v1.metadata.json) records input hashes, parameters and counts. The script needs the external launch-network preview and a GeoNames `cities15000.txt` download.

Method:
- Each of the 4,833 station candidates is assigned to the most populous GeoNames place within 12 km. 1,876 rural stations have none.
- Cities within 40 km of a playable metro count as covered; 160 do.
- The remaining 1,361 cities are ranked by GeoNames population. Each row records the strongest station evidence (operator-published, tagged high-speed stop, or proximity only) and whether its stations link to mapped fast track.

Findings:
- 161 candidate cities have population of 1M or more, 425 have 250k or more, and 690 have 100k or more. 382 of the 690 are in China.
- Cities with 1M+ alone would fill the roughly 156 catalog places left after the 94 metros. The catalog therefore needs a regional allocation rule, not a population cut.
- Several major cities on existing high-speed lines are missing from the 5M seed, including Berlin, Hamburg, Munich, Vienna, Warsaw, Ankara, Casablanca, Jeddah, Harbin and Hefei.

Caveats:
- GeoNames populations are not a common metropolitan definition. Many Chinese entries are prefecture totals.
- Evidence tiers follow OpenStreetMap tagging habits: French and German stops are often tagged, Chinese and Japanese stops rarely are.
- A station near tagged track is not proof of high-speed service.

Screens and checks:
- **Corridor screen:** groups the preview's track into connected pieces and classes each city by whether its track can form a corridor with another catalog city (`corridor_class`). It is informational only. The service check decides inclusion (decision of 2026-11-09), because gaps in the tagged track made the screen drop real high-speed cities in China and elsewhere.
- **Service check:** all 161 cities of 1M+ were checked on 2026-11-09 for scheduled high-speed passenger service on 2025-01-01. [hsr-service-evidence.v1.jsonl](prototypes/hsr-service-evidence.v1.jsonl) records the station, lines, service kind, opening year and sources, and the generator joins it onto the candidate list.
  - **Result:** 157 have service, 3 do not (Budapest, Bayan Nur, Fushun) and 1 is uncertain (Zoucheng).
  - **Spot-checks:** nine rows are flagged `needs_spot_check` because a cited URL was not opened during the check.
  - **Decisions:** Casablanca, Rabat, Oslo and Belgrade stay in by user decision despite debatable speed or date evidence.
  - **Sources:** almost all are Wikipedia line and station articles plus some Chinese news. DESIGN §12 still requires dated official evidence before a line becomes a starting asset.
- **Second check:** on 2026-11-09, the same check covered the 60 non-metro shortlist endpoints and the 14 smaller stops outside China (`checked_for` is `launch_endpoint` or `smaller_stop`).
  - **Endpoints already served:** Bordeaux, Boston, Dresden, Kenitra, Lisbon, Lyon, Montpellier, Perpignan, Plasencia, Porto, Shin-Hakodate-Hokuto, Tsuruga and Turin.
  - **Build-only:** the other 47 endpoints had no high-speed service on 2025-01-01.
  - **Smaller stops:** all 14 are confirmed.
  - **Judgement calls:**
    - Boston (Acela) and the Portuguese Alfa Pendular are classed as upgraded lines.
    - Birmingham (110 mph into New Street) and Dammam (180 km/h) fall just below the threshold.
    - Plasencia counts as dedicated because its line is designed for 350 km/h, although trains run at 200.
  - **Spot-checks:** nine of these rows are flagged `needs_spot_check` for links seen only in search results.
- **How the corridor screen scored against the check:**
  - **China:** it dropped 9 cities, of which 8 have service; only Bayan Nur was a correct drop. It also kept Fushun (no service) and Zoucheng (uncertain).
  - **Outside China:** it dropped 7, of which 6 have service; only Budapest was a correct drop.
- **Misplaced or ambiguous points found by the check:**
  - The "Puyang" point is Pujiang, Zhejiang, while its population is Puyang, Henan's.
  - "Chengtangcun" is Tengzhou, and "Heshan" stands for Yiyang.
  - Several same-name cities (two Taizhous, two Fuzhous, Suzhou in Anhui) are correctly placed but easy to confuse.

- **GHSL urban centres:** given the GHSL Urban Centre Database R2024A V1.2 (General Characteristics theme), the generator matches each city to the urban centre containing it. It tries the GeoNames point first, then the city's assigned stations, then anything within 5 km of an outline. Xuzhou needs one reviewed override.
  - **Columns:** `ghsl_pop_2025` gives a comparable 2025 urban population. `ghsl_shared_metro` and `ghsl_shared_candidates` show cities that share an urban centre with a metro or another candidate.
  - **Effect on the inflated figures:** Weifang falls from 9.4M (GeoNames) to 0.96M and Wuzhong from 7.2M to 245k.
  - **Same urban area:** Zhongshan, Jiangmen, Wuxi, Shaoxing and Kyoto sit inside a metro's urban centre.
  - **Gaps:** 409 smaller candidates match no urban centre.

GeoNames data is used under [CC BY 4.0](https://creativecommons.org/licenses/by/4.0/) with attribution to [GeoNames](https://www.geonames.org/). GHSL data is © European Union, 1995–2026, used under [CC BY 4.0](https://creativecommons.org/licenses/by/4.0/) from the [GHSL Urban Centre Database R2024A](https://human-settlement.emergency.copernicus.eu/ucdb2024Overview.php) (Mari Rivero et al.); this repository stores derived matches and populations only.

## Draft catalog allocation

[prototypes/catalog-city-selection.v1.json](prototypes/catalog-city-selection.v1.json) is a draft allocation of the 156 catalog places beyond the 94 playable metros, under the 250-city target. It is for review, not an approved catalog. [tools/draft-catalog-selection.py](../tools/draft-catalog-selection.py) generates it from the candidate list, the service evidence, the launch preview's shortlist points and GeoNames.

| Group | Places | Rule |
| --- | ---: | --- |
| Launch endpoints | 60 | Shortlist endpoints more than 40 km from a playable metro. Bandung, Belgrade, Casablanca and Warsaw are counted in the next group instead. 13 already have service; 47 are build-only endpoints. |
| Checked stops outside China | 32 | Every non-China city of 1M+ with checked service on 2025-01-01 |
| Smaller stops outside China | 14 | Picks on existing lines that the 1M cut misses: Frankfurt, Hanover, Nuremberg, Lille, Marseille, Bologna, Florence, Naples, Zaragoza, Valencia, Seville, Okayama, Niigata and Kanazawa. All 14 have checked service. |
| China | 50 | 16 provincial capitals, then 14 checked junctions and line ends, then 20 picks ranked by GHSL urban population. The ranking skips cities inside a metro's urban centre, cities sharing an urban centre with one already chosen, and cities within 50 km of a metro or chosen city. |

The remaining 75 Chinese cities with service are not selected; they stay as pass-through stations on starting corridors. Kyoto shares Osaka's GHSL urban centre but stays a catalog city of its own by decision on 2026-11-09.

The Chinese population fill excludes rows the service check flagged: Wuzhong (inflated population), Puyang (misplaced point), Chengtangcun, Heshan, Jinjiang and Cixi.

[china-junction-roles.v1.jsonl](prototypes/china-junction-roles.v1.jsonl) records the network-role check on 2026-11-09, against high-speed lines open on 2025-01-01. It classifies each city as a junction, a line end or a through stop, with the number of directions, the lines and the sources.
- **Original picks:** of the 14 drafted, 8 are junctions and 2 are line ends (Dalian, Shantou). Baoji, Liuzhou, Sanya and Ningbo are through stops.
- **Replacements:** by decision, the through stops gave way to Guilin, Huizhou and Huzhou (4 directions each) and Fuyang (3). Yiwu also has 4 directions but sits within 50 km of Jinhua.
- **Knock-on effect:** Ningbo and Liuzhou re-enter through the GHSL ranking, replacing Yancheng and Zunyi.
- **Date-sensitive calls:** Datong is a junction only because Ulanqab–Datong opened on 2024-12-31. Shangqiu's fourth line opened in 2026, after the reference date.

Two endpoints lack GeoNames data:
- Albury-Wodonga's ID is not in `cities15000`.
- Shin-Hakodate-Hokuto is an OSM station point.

Open items:
- Spot-check the evidence rows flagged `needs_spot_check`.
- Replace Wikipedia-only evidence with dated official sources before any line becomes a starting asset (DESIGN §12).

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
assert len(playable) == policy['playable_count'] == 94
assert playable == [r for r in rows if r['population'] >= policy['minimum_population']
                    and r['id'] not in policy['excluded_metro_ids']]
assert all(r['population'] >= 5_000_000 for r in playable)
print('Structural checks passed; geographic and economic estimates remain unverified.')
print('Threshold exceptions retained:', exceptions)
PY
```
