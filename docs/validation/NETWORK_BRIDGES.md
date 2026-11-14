# Network connectors and inherited railway repairs

The playable catalog adds three prospects and restores the two missing public
HSR links. Native and browser clients consume the same catalog, construction
commands, service templates, forecasts and persistence adapter.

| Project | New works | Construction | Scope |
| --- | ---: | ---: | --- |
| 72 Tsuruga–Maibara | 55 km | 48 months; $3.6294bn | Historical Hokuriku alternative with mountain approaches and a Tokaido junction; onward Kyoto/Osaka and Nagoya use public track. |
| 73 Valencia–Camp de Tarragona | 271 km | 54 months; $6.84756bn | Dedicated-HSR/upgrade game package; reuse the existing 119 km public HSR onward to Barcelona. |
| 74 Porto–Vigo | 139 km | 54 months; $4.3884bn | Portugal–Galicia connector, with Minho crossing, hilly approaches and cross-border interfaces; official end-to-end service target 50 minutes. |

Prices include authored regional surface/hill/viaduct/tunnel quantities and the
current game markup. These are strategic game estimates, not engineering quotes.
The Maibara alternative is historically studied, not the selected Hokuriku
extension. The Mediterranean package prices only works south of Camp de Tarragona;
it does not rebuild Tarragona–Barcelona or buy existing conventional track.
Gauge/electrification and rolling-stock compatibility are abstracted by the
common strategic train family, with approach/interface works included in the
project scope. Production interoperability and engineering remain unqualified.

## Existing infrastructure and services

Turin now uses Torino Porta Susa and connects to the Italian network at Milan.
Perpignan connects through Figueres and Barcelona; Camp de Tarragona is a called
market on the existing line toward Zaragoza. Maibara is a called junction between
Kyoto and Nagoya. The catalog has 291 city markets, 300 public corridor records,
74 prospects (76 alignments including the Japanese alternatives), 3,937
multi-section templates and 2,548 associated map markers. The full station
inventory has 4,333 records. Corridor records include historical capacity aliases,
so their count is not a count of disjoint physical railways.

The new public links are immediately usable and never offered as construction.
Ordered public services cover Turin–Milan–Bologna–Florence–Rome–Naples,
Camp de Tarragona–Barcelona–Figueres–Perpignan and Shin-Osaka–Kyoto–Maibara–Nagoya.
Future connector patterns include Kanazawa–Tsuruga–Maibara–Kyoto–Osaka,
Kanazawa–Tsuruga–Maibara–Nagoya, Valencia–Tarragona–Barcelona (optionally onward
to Perpignan), and Lisbon–Porto–Vigo–Pontevedra. Public feeders are reused without
requiring another construction project. Every private section must open before
its through-service becomes usable.

The source supplement pins 2,012 active ways for Turin/Milan and Perpignan
approaches, including ordered OSM node IDs and query/response fingerprints.
Reviewed HSR platform aggregations at seven station complexes are at most
100 m. They represent one strategic station resource; they do not claim physical
crossovers between arbitrary parallel tracks. Other joins require shared OSM
node IDs. Kyoto–Nagoya and Nagoya–Osaka historical aliases consume the same
Maibara/Kyoto physical resources as new local and through trains.

## Company compatibility

Save v10 archives the exact v9 catalog and records `connector_since` plus its
hash. All 3,327 accepted multi-section templates retain their exact sections,
cities and stops. Existing corridor economics, construction quotes and opening
dates remain unchanged; drawings move to the actual station anchors. Shorter
new patterns receive separate identities. New station markets and topology apply
prospectively. Cash, projects, plans, fleet, debt, pending flows and past accounts
survive. v2–v9 histories remain readable. Opening and failed commits do not write;
a successful command commits the upgrade atomically. No player save is used in
verification.

## Financial screen

The [cash-flow receipt](network-bridge-economics.json) uses the actual shipped
costs, durations and service routes. A default starter operates during the
48/54 construction months in both worlds. It then compares the new project
against a baseline of profitable existing public feeder services, optimizing a
bounded grid of fares/frequencies and optionally extending one feeder. Both worlds
resolve a further 12 months with their selected plans fixed. The comparison keeps
baseline plans/fleets in the project world and includes construction maintenance,
train leases, running, access, station handling and actual borrowing interest.

| Connector | Mean additional company cash/month | Worst tested month |
| --- | ---: | ---: |
| Tsuruga–Maibara | +$0.790m | +$0.635m |
| Valencia–Camp de Tarragona | +$4.317m | +$3.806m |
| Porto–Vigo | +$0.625m | +$0.472m |

These are qualified portfolio examples under the current 90% grant and economy,
not standalone shuttle margins or an exhaustive optimum. They replace the earlier
schematic cost/route screen. Monthly cash excludes upfront capital/principal
repayment, discounting, competition and full-campaign investment recovery.
Positive operating cash does not guarantee recovery of construction expenditure
within the 30-year campaign, particularly for Maibara and Porto–Vigo.
Reproduce without touching player saves:

```sh
GOMEMLIMIT=768MiB GOMAXPROCS=2 go run -p 1 ./cmd/bridgecheck -out docs/validation/network-bridge-economics.json
```

## Verification

The [admission receipt](network-bridge-checks.json) checks public repairs,
construction/through-service scopes, all accepted identities/economics, named
station markers, exact v9 route retention, shared-capacity aliases and deliberately
severed Italian, Catalan and Maibara connections. The launch geography check
covers all 74 prospects in geographic, Equal Earth and actually drawn land space.
The operating manifest has 14 end-to-end contracts, including the earlier NEC,
Uzbekistan, China, Taiwan and other network repairs.

Focused Go packages exercise funded builds, opening boundaries, real through
passengers/revenue and reload; genuine v9 fixtures cover active Kyoto–Nagoya
service and an earlier v8→v9 transition with debt/construction. Migration checks
cover historical accounts, asset/plan retention, malformed metadata, failed-write
atomicity and fresh-game resets. Native picker checks include all three new
prospects in English/Simplified Chinese at 320/1440 px.

Executed gates, build identities and remaining limits are recorded in
[the result ledger](network-bridge-update-checks.json). Native map captures use
the actual renderer with isolated in-memory state. Full browser/WASM startup is
not retried under the documented unresolved 2 GiB cap. Builds and widget/map
checks do not establish physical-phone, native OS input, IME or accessibility
qualification. No full root suite, remote CI or full-campaign NPV is claimed.


Actual-renderer map captures: [Maibara](bridge-maibara.png),
[Mediterranean/Catalan network](bridge-mediterranean.png),
[Portugal–Galicia](bridge-portugal.png), and [Turin–Milan](bridge-turin.png).
