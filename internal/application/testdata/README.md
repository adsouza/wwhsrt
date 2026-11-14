# Historical save fixtures

`economy-v4.json` is an anonymous fresh Sunrise Rail campaign after two resolved
months, exported by the previous v4 headless executable before introducing the
economic calibration. It contains no player campaign or user-specific details.

Generation: `build/headless -save build/economics-probe/frozen-v4.json -advance 2`.
The previous executable's SHA-256 was
`c9701e7712e7e40073c30787955f8a6f91771ccb819d190a7a369f30030f940b`.
The fixture's SHA-256 is
`8f145e8fec79404077855f0e8d65da83d14bf67dac5ecd78f7605c70a87aa8e3`.
It uses rules `network-v4` and the pinned expanded public catalog
`2a8d0522b5a678579d17d5f257d0f6c1fc1c4f4d79879414d720ee6249486e59`.

Do not regenerate it using the current encoder: it verifies that the v5 reader
retains actual legacy company/fleet/service state and both original reports,
then commits the new policy at month index 2 with failed-write atomicity.

`milestone2-v2.json` is the earlier fixture exported by the previous version-2
headless binary. It continues to cover original-catalog/network upgrades.

`topology-v5.json` was exported before the topology repair using a copy of the
previous v5 headless executable (`build/network-repair/headless-v5`), resolving two
months of an anonymous Sunrise Rail company. It pins the original expanded catalog
and calibrated economy. It verifies read-only opening, unchanged historical
reports/plans/fleet, failed-write atomicity and the independent v6 topology cutover.

The v5 fixture SHA-256 is
`abe65fd44242b8d646102b593ebcd3afcb93f67d8393c2ef9c398ba8cb9f8ed0`;
its exporting executable SHA-256 is
`9c0595122621ef0ec0c87102b4bc824b27619aac215b73ac29a701e6fbc46a61`.


`connectivity-v6.json` was exported by the pre-completion version-6 executable
using the exact frozen v2 catalog (SHA `9fcd93c3356bd5545896684e98e102cf7d22144884bf984abd80998cffe942f5`).
It is an anonymous two-month campaign with Tokyo–Nagoya and Beijing–Shanghai
services and six leased trains. It verifies v6→v7 history/fleet preservation,
failed-write atomicity, current-month continuation and strict cutover/hash checks.
No player save was read or modified to create it.

`connectivity-mixed-v6.json` was exported by that same prior executable after
opening the anonymous v5 fixture and resolving one month. It has three reports
and `topology_since=2`, testing distinct v1→v2→current historical intervals.


`launch-v7.json` is an anonymous two-month campaign exported by the pre-expansion
v7 executable with catalog SHA `b5940f7c0c130ec255f0783da9c09e12355b6c61131396a1b5f590c4c72ce2f8`.
`morocco-v7.json` was exported by that same executable with active four-stop
Al Boraq and Casablanca–Rabat services, then two resolved months. These establish
that v8 retains historical accounts, retires the now-excluded conventional-service
plans and leased trains, preserves the unrelated Japanese service, commits
atomically and continues/reloads. No player campaign was used.


`regional-v8.json`, `regional-cutover-v8.json`, and `regional-assets-v8.json`, `regional-texas-v8.json`, and `regional-korea-v8.json`
are anonymous saves exported by the genuine pre-regional v8 executable using the
frozen catalog `0d74a64753d4ec70d353decfd80584cc17c6fc49adfb7c4bd640864d331ba5e9`.
They cover a fresh-world two-month history, a v7→v8 construction cutover, and an
ongoing Dallas–Houston build with debt and pending capital flows, and a 51-month
world with three completed Texas projects and an active Houston–Dallas–Austin–San
Antonio service, plus an operating pre-fix Busan–Ulsan service. They qualify
v8→v9 historical account/asset retention, failed-commit atomicity and reload;
no user company data is included.
