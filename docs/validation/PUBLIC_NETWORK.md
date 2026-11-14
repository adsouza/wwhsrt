# Public-network integration

Later [topology repairs](NETWORK_TOPOLOGY.md) supersede the inventory counts and connectivity limits below; this document retains the earlier verification record.

Verified on 2026-11-12 against macOS 27.0.1 / Apple M4, Go 1.27.1,
Node 26.13.0 and Chromium 153.0.8010.12. This is the existing prototype
made playable, with its provisional source/data limits retained.

## Implemented

The default game uses the [pinned prototype import](../../data/imports/README.md):
274 strategic market/city hubs, 146 existing corridor records, the original two
buildable corridors, 1,990 multi-section service templates, 2,397 source polylines
and 4,323 visible source station markers. The 164 source station pins include hubs
on fragments without a second admitted hub; a pin alone does not open a station
or create a service. Other cities remain unserved. Confirmed source gaps retain
hollow markers; tapping a source marker explains the difference from a service hub.
Fallback operating hubs require source stop-tag/operator evidence, a qualifying
station partner and at least 5 km separation. Yards/depots are excluded as hubs.

Chinese routes are operational game assets. Map controls exposes China and other
regional presets; Services → Find a service accepts English or Simplified Chinese
endpoint names. Beijing–Shanghai leases trains, applies fares/frequency, carries
end-to-end and intermediate passengers, consumes shared capacity and pays access
charges in the monthly ledger. Called stops remain in route details. Non-hub source
stations are inspectable cartography, aggregated into city catchments for demand.
Station details also name each selected city's mapped primary station.

Native, browser and headless use one catalog/domain solver. Integer distances are
pinned in the data bundle. Indexed lookups, all-destination itinerary traversal,
sparse outside-market projection, clipped paths and cached static map scenes keep
the larger catalog practical. Animated representative trains remain separate from
the cached scene. Raw source gaps do not become invented intercity chords.

Development save v4 pins the imported catalog bytes. Version-2/3 saves retain the
company, fleet, projects and historical accounts; the frozen original catalog
validates/rebuilds their old reports. Opening and failed writes preserve the old
save bytes. The next accepted command atomically writes v4. New game uses the
expanded catalog. The actual player's native save was not reset during this work.

## Verification

- Focused Go packages passed: `./internal/domain ./internal/application ./pkg/ui
  ./pkg/mapview ./pkg/app ./pkg/render`. New application checks exercise Chinese
  through passengers/charges, exact global demand conservation, sparse outside-mode
  inspection, shared Kyoto resources, source-stop admission, failed-save atomicity,
  v3→v4 historical-account preservation, continuation and foreign-catalog rejection.
  Native widget checks cover route search, saved-region rotation and sizing;
  rendering checks cover viewport clipping and scene-cache invalidation.
- `python3 tools/import-public-network.py --check` reproduced both generated
  network/coastline assets byte-for-byte. Boundary checks, JS syntax checks and
  `git diff --check` passed. CI now includes import reproduction and the new browser
  workflow; this does not claim that hosted CI has run.
- Web, headless and native targets built. The macOS app's Info.plist validates.
  Native executable SHA-256: `6e793589306bc5b6a19cb27bff791f0e97f163c90dc8c5ad07e195e630334672`.
- [Public-network browser ledger](public-network-checks.json): actual Chinese rail
  pixels (2,323 matching samples), non-hub station picking/name/status, accepted
  Beijing–Shanghai service and monthly accounting, English/Chinese route search,
  phone reflow, read-only outside-market inspection, retained region, reload,
  offline continuation, and identical native-headless/WASM checkpoint **and**
  detailed network report. Observed cold readiness 3235 ms;
  accepted month with the Chinese service 1654 ms
  including browser/UI work. These are observations, not worst-case budgets.
- [Workspace regression](workspace-checks.json) passed on the same final WASM:
  tabs/disclosures, top/left dock, pointer/keyboard resizing, drafts, 320–1920 px
  English/Chinese layouts, applied fare/revenue, reports, reload, offline New game.
  The ordinary edited plan fits the 390×844 phone with a 390×652 inspector and no
  inspector scrolling in either language. Expanded search/help may scroll.
- [M3 regression](milestone3-checks.json) passed on that WASM: Tokyo–Nagoya–Osaka,
  transfers, shared capacity, branch construction/operation, ledgers, representative
  day controls, failed-save atomicity, export, real v2 upgrade, offline use and
  native/WASM replay. Larger headless frames use a bounded 32 MiB subprocess buffer.

Final WASM SHA-256: `4e31ba9167920f561bb33a084614fe8d1253720513d671b202431f51ba970d72`.
Catalog SHA-256: `2a8d0522b5a678579d17d5f257d0f6c1fc1c4f4d79879414d720ee6249486e59`.
Pinned prototype SHA-256: `c4468a0d23cf3ff6a53ba63fb15ce246152a757bb672037d1fbe95c9f5cced28`.
The three browser ledgers pin the same WASM. Prior validation ledgers retain their
historical scope/build identity.

[China desktop](public-network-china-desktop.png) ·
[Chinese phone service](public-network-service-phone-zh.png) ·
[Chinese phone map](public-network-china-phone-zh.png)

## Remaining qualification

This imports the reviewed prototype, rather than independently certifying current
HSR operations. Simplified geometry can omit real junctions and terminal approaches;
components without a mapped connection stay separate even where a real railway may
connect them. Endpoint joins up to 150 m and station approach snaps up to 400 m are
explicit approximations, not engineering surveys. Timings (250 km/h plus dwell),
train compatibility, demand and capacities are game assumptions. Generated service
patterns are not published operator timetables. Full worldwide construction,
production data admission and multiplayer are outside this change.

Browser phone/touch inputs were emulated. Native builds/widget events passed;
actual native OS interaction and graphics were not exercised in this pass. Physical
phones, complete source-name translation/font coverage, screen readers, novice
usability, and dense-network/frame-rate stress qualification remain open. No full
root suite or TypeScript compilation ran locally. Isolated browser contexts and
memory/temporary stores protected the player's campaign; the owned test server was
stopped after verification.

Subsequent [station connection correction](OSAKA_STATION.md) replaces the original
Japan schematic drawings in frame projections. The public-network/save fingerprint
is unchanged; the Osaka ledger pins that newer presentation build.

Subsequent [playable map filtering](PLAYABLE_MAP.md) limits the default map to 146
admitted corridor paths and their 1,867 associated source stations. The full
2,397-line/4,323-station source inventory remains unchanged for audits, and the
public catalog/save fingerprint is unchanged. The newer map ledger supersedes the
initial raw-overlay rendering scope described above.
