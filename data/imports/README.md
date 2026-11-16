# Prototype public-network import

`hsr-prototype.v1.json` pins the audited payload from the existing
`hsr-network-overlay.html` prototype (2026-11-12 import). It contains the original
compressed/delta-encoded rail polylines, source station records and review status,
rail/station source hashes, operator source links, city anchors, camera manifests,
and Natural Earth country outlines. It is an input snapshot, not an instruction
file. The source audit remains in `data/prototypes/hsr-overlay-audit.v1.json` and
`data/prototypes/hsr-station-review.v1.csv`.

From the repository root, with Python 3.12:

```sh
python3 -m pip install -r tools/requirements-public-network.txt
python3 tools/import-public-network.py
python3 tools/import-public-network.py --check
```

The deterministic compiler writes `assets/scenario/public-network.json`,
`assets/scenario/rail-display-paths.json`, and
`assets/scenario/playable-rail-map.json`. The first file records the exact input SHA-256.
`assets/scenario/regions.json` is an authored orientation-aware camera projection
of the existing manifests, with an additional World view. Camera changes
do not alter the network graph or save identity.

## Map and operating model

The [connectivity repairs](../../docs/validation/NETWORK_TOPOLOGY.md) restore all
seven required metropolitan journeys using unsimplified OSM node identities and
bounded slower connecting track, with ten whole-network contracts now enforced. The audit still retains 35 unverified proximity
sites. `--check --strict` now enforces passing connectivity in addition to
reproducibility; it is not global qualification.

- The source inventory preserves all 2,397 reviewed polylines and 4,323 candidate
  station markers. The default game map draws the 293 admitted public corridor
  paths and 2,525 associated source stations. Unused source fragments and their
  unrelated station dots stay in the audit input instead of appearing as usable
  game infrastructure. Associated known map gaps retain their hollow markers.
- Service endpoints combine the earlier scenario, the prototype's metro/launch
  selection, and real station pairs on smaller disconnected components. The three
  explicitly removed metros (Manila, Brasilia) remain excluded as city IDs.
  Non-hub source stations remain inspectable map features. They are aggregated into
  city catchments for demand and are not 4,323 independently selectable service stops.
- The older prototype fallback stage attaches stations within 400 m of source geometry.
  Nearby platform approaches are joined at the station. Its polyline endpoints
  can bridge gaps of at most 150 m;
  every such prototype join is listed in `endpoint_gaps`. The newer source-node
  repair stage does not use those proximity joins. Crossings alone create no junction.
  A component must offer two station endpoints at least 5 km apart before the
  compiler adds fallback city hubs. Additional hubs require a source high-speed
  stop tag or operator evidence; yards/depots are excluded from operating hubs.
  Mere geometric proximity cannot turn a depot or nearby local stop into a new
  passenger market. Source gaps do not create intercity chords.
- A multi-source shortest-path forest contracts the graph into adjacent city hubs.
  Direct links and shortest connected all-stop patterns are playable immediately
  on public open-access track, using the existing lease/fare/frequency system.
  Through-running, transfers, shared capacity, access charges and reports use the
  same domain rules as the original Japan slice. The legacy Nagoya–Osaka express
  shares the imported Nagoya–Kyoto/ Kyoto–Osaka capacity resources.
- The resulting catalog has 288 market/city hubs, 293 existing corridor records,
  71 buildable corridors, and 3,327 multi-section route templates (including closed future services).
  A selectable city does not imply it already has a connected station.
- This is the provisional prototype data made playable, not a newly certified
  real-world timetable or topology. Simplified source geometry can omit switches,
  urban approaches and connections; separate fragments stay separate. Operator
  station evidence is not proof of an adjacent mapped link. The source snapshot's
  cutoff is retained; opening dates have not been independently refreshed.
- Timings (250 km/h distance approximation plus existing strategic dwell rules),
  capacities, train compatibility, demand and fares are game assumptions. Source
  operator names do not claim that an actual operator runs each generated pattern.
  Country labels/codes normalize to one road market, with the original Honshu,
  Hokkaido and Hong Kong distinctions retained.

The original 12-city scenario/business/network assets remain frozen compatibility
inputs. New games use the imported catalog. Save v9 pins the repaired and frozen original catalogs; v2–v8 saves
upgrade on the next successful command and retain their historical accounts.

## Source topology repair stage

`rail-topology-source.v1.jsonl.gz` pins 297,661 source ways: each row contains its
OSM way ID, strategic speed, slower-connector flag, WKT and ordered OSM node IDs.
The first row records the QLever OSM endpoint, regional SPARQL queries, exact raw
response/member-batch hashes, exclusions and the single collinear-node expansion.
The fast-track part matches the previously audited source hash; slower additions
are bounded to the reviewed service regions and compatible standard gauge.
`rail-display-paths.v1.json` preserves the prior legacy station-connected drawings.
`assets/scenario/public-network-v1.json` is the exact frozen pre-repair catalog,
SHA-256 `2a8d0522b5a678579d17d5f257d0f6c1fc1c4f4d79879414d720ee6249486e59`.

Run `python3 tools/prepare-network-repairs.py` to regenerate
`network-repairs.v2.json`, then the ordinary importer above. Both accept `--check`.
No fetch is part of compilation. The preparation stage joins shared OSM node
identities, never geometric crossings, then simplifies only the final drawings.
`network_repair_catalog.py` preserves published route identities and game timings,
retains qualified old paths when the raw source would impose an implausible detour,
and maps decomposed old corridors onto explicit shared capacity resources.
The full provenance, scope and remaining terminal limitations are in the topology
report. Regeneration is a data-development operation; runtime clients use only the
compiled assets and work offline.

## Attribution

Rail geometry and station data: © OpenStreetMap contributors, available under the
[Open Database License](https://www.openstreetmap.org/copyright). The JSON snapshot
and compiled public-network database are OSM-derived databases and retain that
attribution/license; this notice does not change the license of the game's code.

Country outlines: Natural Earth, public domain; see
[`natural-earth-110m.metadata.json`](../prototypes/natural-earth-110m.metadata.json).
City coordinates/populations retain their source/estimate status in
[`data/README.md`](../README.md) and `data/prototypes/catalog-city-selection.v1.json`.
Independent operator references are preserved inside the pinned station metadata.

## Station-connected legacy drawings

`rail-display-paths.json` is a presentation-only projection. The original Japan
service IDs and construction alternatives predate the station-level import and
used city-centre endpoints. Their displayed paths now meet the mapped stations.
Existing-service drawings follow source rail; Osaka construction approaches reuse
the source Kyoto–Shin-Ōsaka tail. The immutable economic geometry, prices, distances,
journey times, public-network bytes and save fingerprint stay unchanged. Frames
use these paths consistently for map drawing, picking/focus and representative
trains. This requires no company reset or save migration. Non-selected existing
lines use the cached source map, avoiding duplicate schematic overlays.

## Playable map projection

`playable-rail-map.json` is presentation-only and pins its economic catalog's
SHA-256. Each rendered public line is the exact displayed geometry of one existing
corridor; source stations are retained only within the established 400 m association
radius. That also removes floating station dots left by unused track. This is a
game relevance filter, not a claim that the hidden source rail does not exist.
Through track stays visible even without an intermediate city or when both end
cities are offscreen. Built/private lines continue to draw from committed projects.
The full source inventory, economic catalog, services, fares, times and saves stay
unchanged. Add/qualify a corridor before promoting additional source geometry into
the playable map; do not patch a geographic fragment into the world arbitrarily.

## Basemap

The world basemap, `assets/terrain/world-land.json`, is no longer written by the
network compiler. `tools/prepare-basemap.py` composes it after the compiler has
run, because it reads the compiled lines to decide where the coast is detailed:
one land outline from the pinned Natural Earth 1:10m land, at 0.003 degrees within
150 km of any line, 0.0125 degrees out to 400 km and 0.05 degrees beyond. See
[the terrain README](../../assets/terrain/README.md#basemap). This is presentation
data outside the public-network save fingerprint.


## Whole-network completion (2026-11-13)

`connector-regions.v2.json` defines eight explicit service regions and compatible
gauges: 1435 mm, 1520 mm (Uzbekistan/Russia), and 1668 mm (Portugal).
`rail-connectors.v2.jsonl.gz` pins 166,846 additional active railway ways with
ordered node IDs, request hashes, gauge policy and operator sources. The combined
source has 464,515 ways and 3,432,739 vertices, including projected station anchors.
Explicit incompatible gauges and freight/yard/inactive/construction tracks remain
excluded. Morocco alone infers missing gauges from ONCF's standard-gauge national
mainline in the operator-backed Al Boraq region; this inference is recorded in
the source header. Even there, only shared node IDs create railway junctions.
`tools/fetch-network-connectors.py` is an explicit network refresh, never a runtime
or ordinary compilation dependency. Its cached HTTP receipts live under build/;
the pinned gzip is sufficient for offline reproduction.

`operating-networks.v2.json` supplies independent operator-backed station anchors
and eleven complete network requirements. It includes full NEC and Uzbekistan stop
chains. New small catchments use authored budgets of 500 daily trips/attraction,
not purported census populations. Corrections preserve published corridor and
service identities/times while adding physical resources. The exact prior v6
catalog is `assets/scenario/public-network-v2.json.gz`; it is an immutable save
compatibility input, not a source to overwrite with the current catalog.

Before writing any catalog/map output, compilation must pass whole-network,
ordered-service, station-marker and path-endpoint checks. Run
`python3 tools/test-network-coverage.py` to also prove failures for a severed NEC
approach, isolated Tashkent, missing/reordered stops and a missing station marker.
CI runs these and the strict audit. Focused application tests additionally operate
all complete network services, carry end-to-end/intermediate passengers, check
capacity/accounting and reload populated historical saves. This is an explicit
coverage contract, not worldwide timetable or source-data completeness certification.


The construction expansion is prepared separately from
`data/construction/launch-prospects.v1.json` and its pinned land/lake mask by
`tools/prepare-launch-prospects.py`. The public compiler verifies its authoring,
mask and cost-ledger hashes before incorporating the 69 new works; two Japanese
projects retain their original alternatives and prices. All 71 names/pairs match
the spec exactly. New service templates combine future track and inherited
physical sections; they stay closed until all sections open. The independent
`check-launch-prospects.py` rejects unpriced water shortcuts and requires only
Tangier–Kenitra as Moroccan starter HSR. Raw conventional station context remains
in the source inventory, not the playable blue network. Existing Lisbon–Porto is
explicitly upgraded conventional rail, and its distinct dedicated-HSR prospect
has separate track capacity. [Current construction report](../../docs/validation/LAUNCH_PROSPECTS.md).


The world basemap is derived from the same pinned construction mask, so the coast
a route is drawn on is the coast it was reviewed against. `tools/prepare-basemap.py`
requires every buildable route to lie on the drawn land or its reviewed crossing in
both projections, simplifying the coast less where one would not; it changes no
prices or saves. Metadata records the source mask, the rule and any such
tightening. The construction checker independently checks the actual rendered
world land as well as the detailed land/lake mask.


## Gyeongju connector, Taipei and Texas addition

`rail-regional-links.v1.jsonl.gz` pins eight active source ways: the 100 km/h
Gyeongju triangular branch and mapped crossovers. Its header contains way/node
query receipts and ODbL attribution. `rail_source.rows` includes these alongside
the unchanged source/connector archives. Real shared node IDs replace the 49 km
Yeongcheon out-and-back; no proximity junction is invented. The final Ulsan anchor uses its actual KTX/Tongdosa station rather than Bugulsan,
with two reviewed local platform adapters. Daegu–Ulsan is about 79 km and the three
Ulsan service distances/timings are corrected prospectively. Unrelated economic
inputs and all archived reports stay fixed.

Taipei uses its supplied 7M metro estimate and an explicit Taipei HSR station pin.
The four Taiwan HSR markets have nine explicitly reviewed platform-way adapters
of at most 100 m. These aggregate existing HSR platforms into strategic station
resources; they do not assert new physical crossovers or join arbitrary nearby
tracks. The ordinary source-node-only rule remains for other junctions. The
operator-backed ordered Taipei–Taichung–Tainan–Kaohsiung service is a required
whole-network contract; Taipei has a required visible station marker.

Houston–Austin is an authored 257 km, 36-month prospect with the North America
cost profile and explicit river/floodplain work quantities. The original 70 project
IDs and published economic inputs remain stable. Save v9 archives the exact v8
catalog and selects it before `regional_since`, retaining past monthly accounts
when the restored Taipei market changes demand. See the
[regional validation report](../../docs/validation/REGIONAL_NETWORK.md).

## Network bridge expansion

`rail-bridge-approaches.v1.jsonl.gz` supplements the source graph with the existing
Turin/Milan and Perpignan approaches. `bridge-approach-regions.v1.json` bounds the
explicit refresh (`tools/fetch-network-connectors.py --bridge-approaches`), which
preserves source node identity, known gauge and active passenger-mainline filters.
The supplement excludes IDs already present in the other three source archives;
its header pins their exact hashes. It is not fetched during game builds.

The operating manifest adds ordered Italian, Catalan and Tokaido-junction
contracts. Named HSR platform aggregations at seven stations are limited to
100 m. They represent one strategic station resource, not invented physical
crossovers. `tools/check-network-bridges.py` checks the compiled repairs, new
construction/through services, station markers, shared-capacity aliases, unchanged
accepted identities/economics and deliberately severed source connections.
