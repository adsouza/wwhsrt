The [regional update](REGIONAL_NETWORK.md) extends this repair with Taipei HSR and the Gyeongju connector. Current counts are 288 cities, 293 existing corridors, 71 prospects and eleven whole-network contracts. Older result ledgers below retain their original hashes/counts.

# Public-network connectivity repairs

Updated 2026-11-13. The compiled catalog now has **287 city hubs, 292 existing
corridors, 70 construction prospects and 3,323 service templates**.
The playable map draws those 292 public corridor paths and 2,525 associated station
markers. The full station inventory has 4,331 records; unused source fragments
remain excluded from the playable map. This is explicit network coverage, not
worldwide timetable or topological completeness certification.

## Whole-network coverage

The [operating manifest](../../data/imports/operating-networks.v2.json) defines
station anchors and required reachability independently of the graph compiler.

| Network | Required coverage | Primary service evidence |
| --- | --- | --- |
| Northeast Corridor | Washington, BWI, Baltimore, Wilmington, Philadelphia, Metropark, Newark, New York Penn, Stamford, New Haven, Providence, Route 128, Boston Back Bay, Boston South | [Amtrak's 14-stop Acela map](https://media.amtrak.com/wp-content/uploads/2025/08/NextGen-Acela_NEC-Map.pdf) |
| Uzbekistan main | Tashkent–Samarkand–Bukhara–Urgench–Khiva | [UZ Rail timetable](https://uzrailpass.uz/en/pages/poezd-jadvali), [Ministry of Transport Khiva service announcement](https://t.me/s/Mintrans_uz/28230) |
| Uzbekistan branch | Samarkand–Qarshi–Kitob/Shahrisabz | [UZ Rail timetable](https://uzrailpass.uz/en/pages/poezd-jadvali) |
| German ICE | Rhine-Ruhr, Köln, Frankfurt, Berlin, Munich, Hamburg | [DB ICE network](https://www.bahn.de/service/ueber-uns/zugtypen/ice) |
| Austrian Railjet | Vienna–Munich | [ÖBB services](https://www.oebb.at/en/reiseplanung-services/oebb-zugverbindungen) |
| Turkish YHT | Istanbul Söğütlüçeşme–Ankara–Konya | [TCDD YHT stops](https://www.tcddtasimacilik.gov.tr/seferler/yuksek-hizli-tren) |
| Sapsan | Moscow Leningradsky–Saint Petersburg | [RZD magazine, July 2025](https://rzdtour.com/assets/pdf/rzd-mag/RZD_Mag_07_2025.pdf) |
| Alfa Pendular | Lisbon Oriente–Porto Campanhã | [CP station/fare table](https://www.cp.pt/info/documents/d/cp/precos-alfa-pendular-lisboa-porto-braga-1) |
| Al Boraq | Tanger Ville–Kenitra (starting dedicated HSR only) | [ONCF operating plan](https://www.oncf.ma/content/download/120129/2802244/file/Plan%20de%20transport%20Al%20Boraq%20CAN.pdf) |
| Chinese corrected hubs | Kunming South, Kaifeng North and Xuchang East connected to Beijing/Shanghai/Hong Kong | [NRA Kunming South](https://source.nra.gov.cn/tlfc/tpsy/202204/t20220405_293015.shtml), [government Xuchang station distinction](https://sfj.pds.gov.cn/contents/33790/21232.html), [Kaifeng government map](https://www.kaifeng.gov.cn/kfsrmzfwz/szfbx/1742071489435447296/yO7R5VTV.pdf) |

The first three require explicit ordered all-stop templates; endpoint connectivity
alone cannot pass. Al Boraq checks the canonical direct Tangier–Kenitra corridor.
Conventional Rabat/Casablanca links are deliberately withdrawn as public HSR;
new southern HSR is a construction prospect. Khiva service is not labeled
Afrosiyob: current western service uses the Jaloliddin Manguberdi train.
The seven earlier required journeys remain covered: Sendai–Shin-Hakodate-Hokuto,
London–Paris, Paris–Lyon, Milan–Rome, Berlin–Munich, New York–Boston and
Changchun–Shenyang.

## Cause and repair

The earlier northeast connector bounds stopped before Philadelphia and Washington;
Uzbekistan lacked slower broad-gauge approaches. Other missing regional links and
wrong station assignments were uncovered in Germany, Austria, Turkey, Russia,
Portugal, Morocco and China. A larger global snap radius cannot safely fix these
missing mainlines or distinguish grade-separated crossings.

The immutable original source contains 297,661 OSM ways with ordered node IDs.
The new [pinned supplement](../../data/imports/rail-connectors.v2.jsonl.gz) adds
166,846 ways in eight explicit service regions, for 464,507 ways and 3,432,688
vertices after station projections. Gauge admission is regional: 1435 mm, 1520 mm
(Uzbekistan/Russia) and 1668 mm (Portugal). Explicit incompatible gauges, inactive
or construction railway, yards, sidings, spurs, and tagged freight/industrial/test/
military tracks are excluded. Crossovers remain eligible.

Morocco's source omits gauge and usage tags on some essential Kenitra approaches.
Within the operator-backed Al Boraq bounds only, missing gauges are inferred from
ONCF's standard-gauge national railway network after the active/passenger-capable
filters. This is a documented national-gauge inference, not measured track gauge.
The pinned header records the exact policy, queries and response/member hashes.

**Only shared OSM node IDs create source junctions.** Coordinate serialization
variance of at most 0.000001 degrees is tolerated only for an already identical
node ID. No geometric crossing, endpoint proximity or broad city boundary joins
separate railways. Stations project onto an actual source edge within the existing
400 m limit, splitting that edge; multiple stations on one way remain distinct.
Local station adapters are visible. Drawings simplify only after contraction.
Island networks are not joined to the mainland by invented links.

The NEC metro anchors now use New York Penn and Boston South. Metropark and Route
128 are separate stops, alongside the other intermediate stations. Uzbekistan adds
Urgench, Khiva, Qarshi and Kitob/Shahrisabz. Lisbon/Porto now anchor at Oriente and
Campanhã rather than unrelated outlying stations. Kunming/Kaifeng/Xuchang anchor
at the appropriate HSR stations; Xuchang's nearby metro station is not substituted.
All authored starting-network station anchors must also ship as visible markers on drawn track.
New small-city demand budgets (500 daily trips and attraction) are authored game
catchments, not census estimates. Existing city budgets are retained.

Connector speeds use tags where available, capped at 160 km/h, with a 100 km/h
fallback; original fast-source ways retain their strategic 250 km/h assumption.
New patterns use source traversal weights. Published IDs, original game lengths,
times, alignment IDs, accepted stop/section lists and fleet requirements remain
compatible. Historical express corridors decomposed onto new physical sections
share those capacity resources, rather than receiving a parallel allowance.
Qualified old paths outside complete source regions remain available when raw
source is missing or would require an implausible detour.

## Regression gates and reproduction

Run sequentially (source preparation is memory-heavy):

```sh
python3 tools/prepare-network-repairs.py --check
python3 tools/import-public-network.py --check
python3 tools/test-network-coverage.py
python3 tools/audit-public-network.py --check --strict
```

The source stage fails if any manifest hub is absent or its network is split.
The compiler stages all outputs until complete-network, ordered-service,
station-marker and drawing-endpoint checks pass. The independent checker validates
corridor resources and whole-network reachability. Negative regressions sever the
NEC's Wilmington–Philadelphia approach while retaining old express aliases,
isolate Tashkent, remove/reorder an intermediate service stop, damage a section
chain, and remove a required station marker. Each must fail; island separation is
also checked. CI runs these gates. Remote CI execution is not implied by local results.

Focused Go domain/application/UI/mapview/native-widget/render packages passed.
`TestConnectivityCompleteServicesCarryEndAndIntermediateRiders` leases trains and
operates complete services for the repaired network families, carries both endpoint
and intermediate passengers, checks capacity and monthly cash reconciliation, and
reloads the settled campaign. Prior seven-journey tests remain. These are actual
business-loop tests, not geometry-only assertions.

## Save compatibility

Development save **v8 / network-v8** adds a construction boundary and pins the
exact archived v7 catalog (`public-network-v3.json.gz`, decompressed SHA-256
`b5940f7c0c130ec255f0783da9c09e12355b6c61131396a1b5f590c4c72ce2f8`).
It also retains the v6, v1-expanded and small original catalogs for their own
historical boundaries. Past accounts replay with their exact old topology. Opening
writes nothing; failed commits preserve bytes/state; the next successful command
writes v8. Unrelated plans, projects, fleet and cash survive. Withdrawn conventional
Moroccan services are retired and their leased trains returned, without rewriting
past reports. Current map animation omits retired services.

The genuine v4/v5/v6 fixtures remain, alongside anonymous v7 fixtures for both an
ordinary company and an active Moroccan company. Tests cover atomic failure,
retirement, current continuation, malformed metadata and reload. The player's
campaign and running native application were not touched. See
[worldwide construction qualification](LAUNCH_PROSPECTS.md) for current results.

## Remaining limits

The original 92 proximity-review sites remain an independent review inventory:
57 have connected source components; 35 still need junction/source review and are
deliberately unjoined. [All 35 sites](PROXIMITY_REVIEW.md) have nearest-station labels, measured separations and map links. A source-component path may go around a candidate site and
does not certify a junction at that coordinate. Multiple terminals per metropolis,
full timetable/vehicle compatibility, dispatch detail and source completeness
outside the manifest remain unqualified.

Current build, source hashes and local evidence are recorded in
[network-completion-checks.json](network-completion-checks.json). The focused
browser attempt is recorded separately in
[network-completion-browser-checks.json](network-completion-browser-checks.json).
Earlier broad desktop/phone workflows exceeded the required 2 GiB checker cap;
those historical ledgers remain intact. New validation uses a fresh isolated
campaign and closes the browser before native comparison. Physical phones,
repeated browser reload memory, native OS input, full root-suite and remote CI
qualification remain separate work.


The current focused browser attempt stopped during startup at **2,068 MiB**
combined checker RSS, before any interaction assertions passed. Lazy history
loading did not establish a browser memory pass. No retry was performed. The
browser build is compiled but remains interaction-unqualified; no new screenshots
or native/WASM browser parity are claimed. Native widget/rules/data checks and
headless replay/save tests passed independently. The test server was stopped.


The 70-project construction expansion supersedes the v7 build counts and binaries
in the older completion ledgers. Current v8 local results are in
[launch-prospects-checks.json](launch-prospects-checks.json). Browser interaction
was not rerun for v8 under the existing memory restriction; no new browser pass
or screenshots are claimed.
