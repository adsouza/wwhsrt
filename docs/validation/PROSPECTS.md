The current catalog has 70 prospects. [LAUNCH_PROSPECTS.md](LAUNCH_PROSPECTS.md) supersedes the two-project counts and binary qualification below; the older screenshots remain historical evidence.

# Selectable construction prospects

Verified on 2026-11-12 (browser ledger timestamps are UTC, 2026-11-13).

Map and Build now show every currently supported unstarted corridor as a dotted
prospect along its reviewed alignment. Selecting the path opens Build → Plan with
both endpoints, alignment, engineering/funding and operating estimate. The selected
alignment has a solid highlight. The native/browser sidebar lists the same named
prospects, with selection styling and a Browse prospects on the map action.
Arbitrary city selectors and Choose A/B controls are removed; city inspection
preserves the construction draft. The current catalog has two prospects:
Tsuruga–Osaka and Shin-Hakodate-Hokuto–Sapporo. Catalog coverage is unchanged.

`ui.ConstructionOpportunities` excludes existing and committed corridors.
Drawing and hit testing share the same actual path, including detours and the
selected alternative. Picking uses an 18-pixel tolerance along the continuous
line, including spaces between dots. A construction commitment removes the
prospect immediately and shows accepted construction geometry instead. Persisted
endpoint fields remain compatible; no save migration or company reset is needed.
The UX/design specification and English/Simplified Chinese help use this model.

Verification:

- `GOMEMLIMIT=768MiB GOMAXPROCS=2 go test -p 1 ./pkg/ui ./pkg/app ./pkg/render -count=1`
  passed. New checks cover actual detours versus endpoint chords, selected
  alternatives, touch tolerance, public/started/completed exclusions, stale
  selection, native pointer/list events, drag separation, map expansion, native
  English/Chinese phone/desktop layout and unchanged campaign exports. The UI
  package passed again after the final contextual-help wording adjustment.
- `sh tools/build-web.sh` and `sh tools/build-native.sh` passed. Architecture
  boundaries, native bundle metadata, changed JS syntax and `git diff --check`
  passed.
- [Prospect browser ledger](prospects-checks.json): actual blue dot pixels at 17
  samples on Tsuruga–Osaka and nine on the Hokkaido extension; clicking each path
  opens its complete project. Drags preserve selection. Full-map browse/restore
  and the Chinese 390px list work without horizontal overflow. All inspection
  actions preserve exact durable campaign bytes/checkpoint. An explicitly funded
  construction order in the isolated test company removes its prospect from the
  list and map picking, including after reload.
- [Existing Chinese connection ledger](existing-connection-checks.json) passed
  on the same build: Beijing–Tianjin, Chengdu–Chongqing and Chengdu–Xi'an remain
  public service opportunities and absent from construction prospects. Service
  selection preserves exact campaign bytes/checkpoint in English desktop and
  Chinese phone layouts.
- The broader `check-workspace.cjs` sweep was stopped during its responsive loop
  when combined browser-check RSS exceeded the repository's 2 GiB cap. It did not
  complete and its older ledger does not qualify this build. The focused prospect
  workflow above completed separately; no full root suite or TypeScript
  compilation ran.
- Final WASM SHA-256:
  `af9aa14ba1c4cb4b5f133985f08f15e782f419b50b91928c5a67e0f7c240997e`.

Screenshots: [Map overview](prospects-overview.png),
[selected project](prospects-desktop.png), [Chinese phone](prospects-phone-zh.png).

Browser inputs are synthetic; native widgets were tested directly and built,
without a native OS mouse-interaction pass. Physical phones remain unqualified.
Test browser companies are isolated from the player's save. Restart the native
app to load the rebuilt controls and map.
