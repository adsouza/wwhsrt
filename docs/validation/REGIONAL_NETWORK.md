# Yeongcheon repair, Houston–Austin and Taipei

Native map captures: [Daegu before](regional-daegu-before.png),
[Daegu after](regional-daegu-after.png), [Taiwan](regional-taiwan.png),
[Texas prospects](regional-texas.png).

The current catalog has **288 cities, 293 existing corridors, 71 construction
prospects and 3,327 multi-section service templates**. Its playable public map
has 2,533 associated source station markers. This update changes three areas:

- **Daegu–Ulsan:** eight mapped OSM ways restore the Gyeongju triangular connector
  and crossovers omitted by the fast-track extract. All junctions use shared
  source node IDs. The final visual check also found Ulsan pinned to Bugulsan rather than its
  KTX/Tongdosa station. Correcting that anchor and aggregating its two platforms
  removes the remaining spur. The drawing is now about 79 km, with the associated
  Ulsan service distances/times corrected prospectively under save v9.
- **Houston–Austin:** project 71 is a 257 km, 36-month game alignment with the
  North America regional rate and authored river/floodplain allowances. It is
  dotted, pickable and buildable through the ordinary workflow. A new
  Houston–Austin–San Antonio service requires both construction sections to open.
- **Taipei:** the stale city exclusion is removed. The original 7M metro estimate
  supplies 7,000 daily trips/attraction; the city is anchored at Taipei HSR station
  with a required visible marker. Taipei–Taichung–Tainan–Kaohsiung is a required
  ordered operating service, staying separate from mainland rail networks.

## Sources and station resources

`data/imports/rail-regional-links.v1.jsonl.gz` contains the eight source ways,
ordered node identities, query/response hashes and ODbL attribution. The missing
Gyeongju branch is mapped at 100 km/h; its short crossover approaches use a slower
game speed. These are connections used by the operating graph, not new dedicated
HSR inventory. The combined source has 464,515 ways and 3,432,739 vertices.

Taiwan previously pinned a station to only one of its parallel HSR tracks. That
created implausible long detours and prevented a simple ordered through-service.
The manifest now names the existing platform ways at four station complexes.
Nine local adapters, 2.48–46.74 m long and bounded to 100 m, aggregate those
platforms as strategic station resources. They are not arbitrary junction snaps
or claims of physical crossovers. Two similarly reviewed Ulsan KTX platform
adapters are 2.3 and 6.8 m. Other rail junctions still require shared OSM
node IDs. [THSR's official station/service description](https://en.thsrc.com.tw/corp/c33f8d02-7c8a-44f0-8866-b3f8c9707ff9)
supports the Taiwan stop chain. Intermediate source stations remain inspectable
map features; the four cities are the game's called markets.

## Compatibility

All prior corridor IDs remain. Construction quotes and unrelated operating
inputs remain unchanged; the three reanchored Ulsan services use corrected route
distances/times (Daegu 79 km/19 min, Busan 52 km/13 min, Seoul 358 km/87 min).
Past accounts retain their old values under the archived catalog. All **3,323 pre-update service templates keep their exact section,
city and stop lists**, including the longer Houston services via Dallas. The new
shorter through-service uses `launch_route_houston__san_antonio_regional`; adding
a prospect cannot silently reroute an accepted service onto unfinished track.

Save v9 adds `regional_since` and the exact v8 archive hash. Historical months
select the archived v8 catalog before earlier catalog/economy boundaries; Taipei's
new demand applies prospectively. Cash, projects, plans, fleet, debt and pending
capital flows survive. Reads and failed commits do not write; the next accepted
command upgrades atomically. Versions 2–8 remain readable. Fresh games reset all
cutovers to zero. No player save is used in verification.

## Verification

[Data admission](regional-network-checks.json) independently checks the corrected
Daegu alignment, all accepted service/economic identities, Taipei's ordered chain
and marker, and the new Texas alternative. Negative fixtures reject the old
Yeongcheon route and a severed Taipei connection. The general construction checker
covers all 71 pairs and 69 new alignments against geographic, Equal Earth and
actually drawn land geometry, including crossing budgets and regional quantities.

Focused Go checks cover the domain, application, UI, map controls, native widgets
and renderer. Actual commands construct Houston–Austin, carry direct and through
passengers, operate Taipei–Kaohsiung and reload. Genuine anonymous v8 fixtures
exercise ordinary history, a prior v7 cutover, pending construction/debt, and a
51-month campaign already running Houston–San Antonio via Dallas, plus an active
pre-fix Busan–Ulsan service. They verify
historical accounts, asset/route retention, failed-write atomicity and malformed
metadata rejection. Native picker tests include the new prospect at 320/1440 px
in English/Simplified Chinese.

Final results, build hashes, source memory receipts and native map captures are
recorded in [the local ledger](regional-update-checks.json). Browser and native
builds do not establish interactive input qualification. Complete browser/WASM
startup was not retried because of its existing 2 GiB memory-cap limitation;
physical phones and native OS interaction were not requalified. Houston–Austin
geometry, terrain shares and duration remain game concepts, not surveyed or
approved railway engineering. Full suites/race/fuzz and remote CI are not claimed.
