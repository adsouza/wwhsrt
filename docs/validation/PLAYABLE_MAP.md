# Playable public-rail map

Later [topology repairs](NETWORK_TOPOLOGY.md) supersede the inventory counts and connectivity limits below; this document retains the earlier verification record.

The former background drew the full 2,397-polyline source extract, including a
separate unused parallel fragment southwest of Changchun (roughly 124.27 E,
43.03 N through 125.11 E, 43.75 N). Those sample points are several kilometres
from any admitted operating corridor. The nearby Harbin–Shenyang path is a usable
through corridor and remains visible, as does Changchun–Jilin.

`playable-rail-map.json` now supplies exact displayed geometry for all 146 existing
corridor records and 1,867 source stations associated within 400 m. The renderer
checks the catalog fingerprint. Unused source fragments and unrelated station dots
are omitted from this presentation layer. The original 2,397 polylines/4,323 station
records remain available in the pinned audit input. This does not claim that the
hidden rail does not exist in the real world.

The rule is independent of the viewport: a through corridor stays visible when
it has no intermediate game city or both endpoint cities are offscreen. It is also
independent of the player's current train services. Construction/owned lines keep
their existing state-driven overlays. Map selection/highlights and train replay
use the same station-connected paths. Economic catalog bytes, source identity,
fares, journey times, service templates, passenger simulation and saves are
unchanged; no company reset or migration is needed.

Verified on 2026-11-12:

- `go test -p 1 ./pkg/render -count=1` passed. New coverage compares every rendered
  public line against the corresponding application-frame path, requires every
  public corridor exactly once, keeps the two northeast operating routes, excludes
  three reported-fragment sample points and independently checks all 1,867 visible
  source-station associations in a local metre projection.
- Import `--check` reproduced all presentation/economic outputs byte-for-byte.
  Economic catalog SHA-256 remains
  `2a8d0522b5a678579d17d5f257d0f6c1fc1c4f4d79879414d720ee6249486e59`.
- Web/native builds, native bundle metadata, architecture boundaries, JS syntax and
  `git diff --check` passed.
- [Targeted browser ledger](playable-map-checks.json): 40 visible retained-route
  samples match actual rail pixels; all three unused-fragment samples have no rail
  pixels. Normal route selection/focus/zoom/reload preserve exact durable save
  bytes and checkpoint. [Northeast China screenshot](playable-map-northeast-china.png).
- [Public-network regression](public-network-checks.json) passed on the same WASM:
  Chinese rail pixels, source station inspection at detailed zoom, Beijing–Shanghai
  operation, capacity, monthly charges, Chinese phone layouts, reload/offline use,
  and identical native-headless/WASM checkpoint and detailed network report.
- WASM SHA-256: `b9cfe9632d4ceacd294a3e24ab7db919be6c1da0f99b11efefc30dec883c7377`.
- Map SHA-256: `8320e8a8ff270f6a82f5f855bb193325a594248bf0594e748dbad6a4f7966948`.

Native OS interaction/graphics were not exercised in this pass. Browser inputs
were synthetic; world-data certification and physical-device qualification remain
open. No full root suite or TypeScript compilation ran locally. Test browser data
was isolated from the player's campaign; restart the native game to load the new
map. The owned test server was stopped after checking.
