# Kyoto–Osaka station connection

The imported Osaka hub is **Shin-Ōsaka** at 34.73354, 135.50019. The old
Nagoya–Osaka and Tsuruga–Osaka drawings ended at the metropolitan reference point
34.6937, 135.5023, about 4.4 km south. Drawing both the old schematic approach and
the source railway left a stray branch that missed the station marker.

The frame projection now uses source track for the legacy existing services and
the common Kyoto–Shin-Osaka tail for Osaka construction approaches. Other legacy
endpoints meet their mapped stations too. Drawing, service highlights, route focus
and representative trains all consume those same frame paths. Only selected
existing services get an overlay; the cached source map supplies the other tracks.

Economics, service IDs, prices, timings, canonical saves and pinned public-network
bytes are unchanged. Existing v4 campaigns need no reset or migration. The running
native process keeps its compiled code until restarted; it was not interrupted.

Verification on 2026-11-12:

- Focused `./internal/application ./pkg/render` packages passed, including exact
  station endpoints, common Kyoto tail, matching forward/reverse train endpoints,
  unchanged non-geometry fields/save bytes/fingerprint, and frame-copy isolation.
- All 12 frozen/display paths passed the existing geographic and Equal Earth
  shoreline/lake checks and priced-length bounds; negative fixtures still fail.
- Import regeneration is byte-for-byte reproducible. Public catalog SHA-256 remains
  `2a8d0522b5a678579d17d5f257d0f6c1fc1c4f4d79879414d720ee6249486e59`.
- Web/headless/native targets built. Boundary and JS syntax checks passed.
- [Browser ledger](osaka-station-checks.json): 12 selected-route samples match the
  source approach, and the old approach has no selected-route pixels; DOM route
  selection/focus/reload preserve checkpoint and exact durable bytes.
- WASM: `5e5ec14adaf977dd168708611c6d5b883d5e2c8e7fb4607a5e9e42fe79a48547`.
- Display paths: `833898f638293f2d5b9823a5961c59ef97fcc1de3dac15129f1aba8a08a9f60c`.

[Corrected selected approach](osaka-station-desktop.png).

The browser check was isolated from the user's native campaign. It exercises the
actual Go map renderer, but it does not qualify native OS interaction or an
engineering survey. Original regional water-mask limitations still apply. No full
root suite or TypeScript compilation ran locally.
