# Testing

The repository has a root Go module and a playable first-business slice, alongside earlier prototypes. Focused checks exercise construction, network demand/revenue, funding, fleet, cash reconciliation and continuation. The dedicated server, worldwide data qualification and full production save format remain planned. A hosted workflow builds native/browser targets and runs the initial rules and browser checks; adding the workflow does not establish that it has run remotely.

Use the [architecture](ARCHITECTURE.md) for system boundaries, [game design verification contract](docs/DESIGN.md#13-verification-and-performance) for domain properties and scenarios, and [UX acceptance criteria](docs/UX.md#10-acceptance-and-questions-to-test) for player workflows. Existing results and limitations are recorded in the [validation report](docs/validation/README.md).

## Local verification policy

- Use focused checks appropriate to the change. Documentation-only edits need document/reference checks, not game builds or browser runs.
- The full Go suite may be run locally, for example `GOMEMLIMIT=768MiB GOMAXPROCS=2 go test -p 1 ./... -count=1`.
- Do not run local TypeScript compilation, directly or indirectly through scripts or hooks. Use hosted/CI typechecks if TypeScript is introduced.
- Inspect scripts, hooks, and child commands before running automatic checks. Serialize memory-heavy work. For Node-based checks, cap old space at 1536 MB per process; stop if combined memory-heavy check processes exceed 2 GiB.
- Measure that 2 GiB as physical footprint, not RSS. A browser checker is a tree of processes (Node, the local server, and the headless browser's browser, GPU and renderer processes) that map the same libraries; summed `ps` RSS counts those shared pages once per process and peaked 10% to 46% above the footprint in the runs below. Take the process IDs from `ps -axo pid=,ppid=` and the sizes from one `top -l 1 -stats pid,mem -pid …` call, about once a second.
- Browser checkers may run locally one at a time under that watch, each against a freshly started local server on a free port, with the tree killed when it passes 2 GiB. On 2026-11-15, at the single-outline basemap, 17 of the 20 checkers hosted CI runs finished and passed this way, with peak footprints of 1,440 to 1,968 MiB (`check-campaign-completion.cjs --layout` 124 MiB). The same watch on summed RSS had stopped 18 of the 20 within 43 seconds. `check-terrain.cjs`, `check-business.cjs` and `check-milestone3.cjs` cross 2 GiB in most runs (terrain fitted in 3 of 14, the other two in 1 of 3 each), so a local stop of those three is not a result; hosted CI qualifies them. Peaks vary by a few hundred MiB between runs of the same checker.
- A checker rewrites tracked evidence in `docs/validation` even when it is stopped part-way. Look at what changed and restore partial files before committing.
- State exactly which checks ran and any remaining limits. A build is not an interaction check; browser touch emulation is not physical-device qualification.

## Existing checks

The Taipei/Yeongcheon/Texas update adds `python3 tools/check-regional-network.py`
after ordinary source preparation/import. Run it sequentially with the existing
geography, coverage and strict audit gates; the operating manifest now has eleven
network contracts. `TestRegional*` in the application package covers genuine v8
migration, active older Texas service retention and new Taiwan/Texas through
passengers. See [REGIONAL_NETWORK.md](docs/validation/REGIONAL_NETWORK.md).

Campaign-end feedback has a focused native widget regression and an isolated
browser footer check: `NODE_OPTIONS=--max-old-space-size=1536 node tools/check-campaign-completion.cjs --layout`.
See [CAMPAIGN_COMPLETION.md](docs/validation/CAMPAIGN_COMPLETION.md) for exact
targets and qualification limits. This does not retry full WASM startup.

For the economy calibration, run the focused domain/application/UI/native-app packages, then `go run -p 1 ./cmd/balancecheck -out docs/validation/economy-calibration.json` with the Go memory/process caps above. After rebuilding web/native/headless, run `NODE_OPTIONS=--max-old-space-size=1536 node tools/check-economy.cjs <test-server-url>`. It exercises suggested plans, break-even/expense reconciliation, invalid/suspended drafts, through-running, native-headless/WASM replay, a genuine v4 save cutover, Chinese phone layouts and offline continuation. See [ECONOMY.md](docs/validation/ECONOMY.md).


Public-network repairs are documented in [NETWORK_TOPOLOGY.md](docs/validation/NETWORK_TOPOLOGY.md). Run `python3 tools/prepare-network-repairs.py --check`, `python3 tools/import-public-network.py --check`, `python3 tools/test-network-coverage.py` and `python3 tools/audit-public-network.py --check --strict` sequentially. All ten whole-network contracts, ordered NEC/Uzbekistan/Al Boraq services, required station markers and map endpoints must pass, alongside the earlier seven pair checks. Negative regressions sever the NEC, isolate Tashkent, damage stop/section lists and remove a marker. Focused Go application tests operate complete services, carry end-to-end/intermediate passengers, check capacity/ledger/reload and v6→v7 historical migration/atomicity. The domain/UI/mapview/app/render packages cover shared behavior and drawings.

Source preparation handles about 3.43 million vertices; serialize it with all heavy checks and monitor RSS. The successful local reproducibility run peaked at 1,768 MiB. After building web/native/headless, the optional focused desktop check is `NODE_OPTIONS=--max-old-space-size=1536 node tools/check-network-completion.cjs <server-url>`. It uses a fresh isolated campaign, checks visible NEC/Uzbekistan paths and normal service controls, and closes the browser before native-headless parity. It stops above the 2 GiB combined checker cap. Earlier broad browser/phone runs were stopped at that cap as summed RSS; the local verification policy above now measures footprint, under which most checkers fit. Current historical-catalog parsing is lazy; this does not establish physical-phone or repeated-reload memory qualification.

For service selection from map track, run the focused UI/mapview/app/render
packages or `-run 'Test(MapService|ServicePicking|MapResize|NativeMapService)'`.
After building both clients, run
`NODE_OPTIONS=--max-old-space-size=1536 node tools/check-map-services.cjs <server-url>`.
It covers open-track picking, direct/through choices, retained drafts, full-route
highlight pixels, Chinese phone visibility, clearing/reloading the local scope
and unchanged campaign saves. See [MAP_SERVICES.md](docs/validation/MAP_SERVICES.md).

For prospect selection, use focused `go test -p 1 ./pkg/ui ./pkg/app ./pkg/render`
or `-run 'Test(SelectProspect|Prospect|ProductionProspect|NativeProspects)'`.
After rebuilding web/native, run
`NODE_OPTIONS=--max-old-space-size=1536 node tools/check-prospects.cjs <server-url>`.
It checks actual dotted pixels, whole-path selection, drag separation, list and
full-map browsing, Chinese phone layout and removal after a real construction
commit/reload. See [PROSPECTS.md](docs/validation/PROSPECTS.md) for evidence and limits.

For native representative-day Play, run
`GOMEMLIMIT=768MiB GOMAXPROCS=2 go test -p 1 ./pkg/app -run '^TestNativeReplay' -count=1`.
It renders the real slider after Play/rebuilds, checks deferred-event isolation,
pause/scrub/resume and unchanged campaign exports. See
[REPLAY_PLAYBACK.md](docs/validation/REPLAY_PLAYBACK.md) for evidence and limits.

### First business slice

For the expanded public network, see [PUBLIC_NETWORK.md](docs/validation/PUBLIC_NETWORK.md). Focused checks: `go test -p 1 ./internal/domain ./internal/application ./pkg/ui ./pkg/mapview ./pkg/app ./pkg/render`; `TestPublicNetwork*` covers Chinese through-running, finite demand, shared Kyoto capacity, source inventory, atomic v2–v6→v7 upgrade, historical account retention and pinned-catalog restore. `TestNativePublicNetwork*` covers route search and region rotation. Build web/native/headless, then run `NODE_OPTIONS=--max-old-space-size=1536 node tools/check-public-network.cjs <server-url>` for actual Chinese rail pixels, route search/service/revenue, Chinese phone layouts, native/WASM replay, reload and offline continuation. Import regeneration uses `tools/requirements-public-network.txt`; its scope/limits are in `data/imports/README.md`.


Task workspace checks and qualification limits are in [WORKSPACE.md](docs/validation/WORKSPACE.md). Use the focused `./pkg/ui ./pkg/app` packages for shared layout, native widget sizing, draft preservation and map-origin picking. After building the web bundle, run `NODE_OPTIONS=--max-old-space-size=1536 node tools/check-workspace.cjs <test-server-url>` for real tabs, disclosures, pointer/keyboard resizing, top/left placement, compact fallback, English/Chinese reflow, business actions, help links, reload, offline continuation and New game preference retention. The ordinary plan fits at 390×844 and 1280×850; expanded detail and short screens can scroll. Legacy workflow scripts use `tools/ui-helpers.cjs` to follow normal tabs and disclosures before an action. Native OS pointer/keyboard interaction and physical devices remain separate qualification.

Contextual onboarding checks and qualification limits are in [ONBOARDING.md](docs/validation/ONBOARDING.md). Run the focused `./pkg/ui ./pkg/app` packages, build the web bundle, then run `NODE_OPTIONS=--max-old-space-size=1536 node tools/check-onboarding.cjs` against a temporary local server (optional URL argument). It checks per-concept persistence, explanations/reference, drafts, Chinese search/composition, failed/accepted learning actions and offline use; it does not qualify novice usability or native OS input.

Milestone 3 details and limits are in [MILESTONE_3.md](docs/validation/MILESTONE_3.md). Focused checks are `GOMEMLIMIT=768MiB GOMAXPROCS=2 go test -p 1 ./internal/domain ./internal/application ./pkg/ui ./pkg/app ./pkg/render -count=1`. These include network conservation/fuzz seeds, through/transfer/feeder and bottleneck scenarios, migration/atomic extension, Chinese native layout and representative train timing/picking. For the real browser workflow, build web/headless targets and start the server below, then run `NODE_OPTIONS=--max-old-space-size=1536 node tools/check-milestone3.cjs` (optional URL argument). It writes `milestone3-checks.json` and screenshots, and compares both the checkpoint and detailed network report with native-headless replay. The `testdata/milestone2-v2.json` application fixture was exported by the prior version-2 headless binary. Races and extended fuzzing remain hosted gates.

For native Company name editing, run `GOMEMLIMIT=768MiB GOMAXPROCS=2 go test -p 1 ./pkg/app -run '^TestNative(Name|Panels|Wrapped|ReportScroll|NewGame)' -count=1`. `TestNativeName*` covers Unicode/repeated-substring selections, grapheme deletion, pointer placement/double-click, Shift/word navigation, fake-clipboard copy/cut/paste, failed/empty clipboard preservation, undo/redo, UI focus traversal, draft-only persistence and resize retention. Clipboard tests use an in-memory adapter and never inspect or overwrite the host clipboard. Native OS shortcuts, physical drag/keyboard/IME and actual clipboard exchange still need a manual pass: automated CUA input did not reliably activate even the existing native controls during the isolated observation.

Milestone 2 evidence is in [MILESTONE_2.md](docs/validation/MILESTONE_2.md). After building browser/headless targets and starting the test server, run `NODE_OPTIONS=--max-old-space-size=1536 node tools/check-milestone2.cjs` (optional URL argument). It drives the normal starter-month control, per-concept dismissal and separate global visibility, station/family inspection, cost/alternative comparison, independent project/station construction, opening, fleet, service, revenue, export/headless restore and offline continuation on desktop/Chinese phone layouts. It records `milestone2-checks.json`, screenshots and native-headless/WASM parity. Run the focused Go application/UI/native-widget packages below; `TestM2*` adds station lifecycle/reuse and projection/accounting coverage, and `TestContextGuidance*` covers cue transitions and dismissal without state mutation. Native macOS service rendering was visually inspected in isolation, but automated native input did not reliably activate controls. Native interaction and physical-device qualification remain open.

Use Go 1.27.1 (module minimum 1.26.4). From the root, these are focused checks for the implemented slice:

```sh
GOMEMLIMIT=768MiB GOMAXPROCS=2 go test -p 1 ./assets ./internal/application ./internal/adapters/filestore ./pkg/mapview ./pkg/ui -count=1
python3 tools/check-boundaries.py
```

For the real WASM/browser acceptance checks, build first, then start the server in a separate terminal:

```sh
sh tools/build-web.sh
mkdir -p build
GOMEMLIMIT=768MiB GOMAXPROCS=2 go build -p 1 -o build/headless ./cmd/headless
go run ./cmd/webserve
```

In another terminal:

```sh
npm ci --ignore-scripts --no-audit --no-fund
npx playwright install chromium
NODE_OPTIONS=--max-old-space-size=1536 npm run check:business
```

The checker uses actual DOM inputs and the Go-rendered map. It verifies selection/drag separation, zoom, keyboard controls and accessibility-tree names, CDP Chinese composition, draft retention, narrow layouts, hidden-map commands, IndexedDB write failure, Web Lock ownership, reload, offline cold launch, corrupt-save preservation, and native-headless/WASM checkpoints. It writes `docs/validation/business-checks.json` and screenshots. Synthetic inputs cannot qualify an actual phone keyboard, VoiceOver experience, native IME or OS termination. The native app was launched/visually inspected on macOS; its canvas management controls are absent from the accessibility tree.

For text size, run `GOMEMLIMIT=768MiB GOMAXPROCS=2 go test -p 1 ./pkg/ui ./pkg/render ./pkg/app -run 'Text|Workspace|Chrome|StepEquals|ResizeDrops|MapLegend|MapScene|DividerDrag' -count=1`. `TestTextLadders` and its neighbours pin the desktop and compact ladders, the four steps and the largest step a window has room for. `TestWorkspaceStepEqualsSmallerWindow` and `TestNativeStepEqualsSmallerWindow` check that a step is the layout of a proportionally smaller window, rectangle for rectangle, and that no hand-placed text is clipped; `TestNativeChromeTextFitsItsBoxes` checks the same at 100% from 320×480 to 1920×1080 in both languages. `TestNativeFontsComeFromTheScale` fails on any font made outside the two face helpers. `TestMapLegendsUnchangedWithoutAScale` pins the positions the browser client draws. These build widgets without opening a window: sharpness, the caret in text fields at large steps and the look of map legends need a hand check in the rebuilt native app on a large display.

For native sidebar sizing and scrolling, run `GOMEMLIMIT=768MiB GOMAXPROCS=2 go test -p 1 ./pkg/app -run '^TestNative(ReportScroll|Panels|Wrapped)' -count=1`. These checks measure actual ebitenui content/footer preferred sizes across five views, English/Chinese and 320–1920px windows, and check wrapped catalog text with the bundled font. They reproduce the Build panel's former 448px content width inside a 360px sidebar. Populated-report tests dispatch the actual widget wheel/slider events at 320×480 through 1200×800: fractional wheel deltas, both scroll limits, dragging, view rebuilds, stale events, expanded-map isolation, footer/gutter separation and no-overflow behavior. Scrolling preserves the simulation checkpoint. These tests do not open a window or qualify OS input; manually check wheel/trackpad, scrollbar dragging and Page Up/Down/Home/End in the rebuilt native app. Text fields retain their editing keys.

Hosted [CI](.github/workflows/milestone1.yml) also runs the full root suite, race checks, bounded fuzz campaigns, the data checks and macOS/Linux/Windows builds of every command, each in its own job. Browser checkers run against one shared web build; long checkers get a runner each and short ones share one, so every job starts at once under the account's 20-job concurrency limit. Runners set `WWHSRT_UNCAPPED=1`, which makes `tools/build-web.sh` and `tools/build-native.sh` skip the memory caps that local builds keep. On `main`, once every job has passed, the `pages` job publishes that same checked `web/` bundle to [GitHub Pages](https://adsouza.github.io/wwhsrt/); a failure anywhere leaves the previous build live. Stop a test server when finished; preserve an interactive development instance the user is using.

For idle-line warnings, run `go test -p 1 ./internal/application -run '^TestIdle' -count=1` and `go test -p 1 ./pkg/app -run '^TestNativeIdle' -count=1`. They cover authoritative fees, opening boundaries, accepted versus previewed service, leasing alone, suspension, reload, public-track false alarms, frame isolation and native warning/header sizing. The business browser checker exercises the warning, service shortcut and its lifecycle at desktop and 320px Chinese layouts and saves `business-idle-line-*.png`. Notices do not block deliberate suspension or mutate finances. The focused `TestIdleHighlights*`, `TestIdleThroughFleet*` and `TestNativeIdle*` checks also verify committed geometry, through-fleet section coverage, all-line and individual review framing, and unchanged state. After rebuilding web/native targets, run `NODE_OPTIONS=--max-old-space-size=1536 node tools/check-idle-review.cjs` against the test server (optional URL argument). It checks actual Go-rendered warning-colour pixels on two idle lines, excludes operating track, exercises row focus, 320px Chinese map visibility, leasing versus accepted service, idle fleets on shared track, through-route coverage and reload. Results are `docs/validation/idle-review-*`.

For New game, run `go test -p 1 ./internal/application ./pkg/ui ./pkg/app -run 'Test(NewGame|NativeNewGame)' -count=1`. This covers a populated campaign reset, atomic failed writes, restart after campaign completion, stale command rejection, durable reload, first-month income, cleared drafts, retained preferences, and native confirmation/cancel sizing down to 320×480. After building browser/headless targets and starting the test server, run `NODE_OPTIONS=--max-old-space-size=1536 node tools/check-new-game.cjs` (optional URL argument). It uses actual Company/confirmation controls for cancellation, errors/retry, English/Chinese layouts, reload, offline restart and native-headless/WASM replay; evidence is `docs/validation/new-game-*`. Its browser context is isolated from the player's real save.

For native window preferences, run `go test -p 1 ./pkg/app -run '^TestNativeWindow' -count=1`. Checks cover normal and invalid bounds, disconnected/reordered/duplicate-named displays, legacy flat UI preferences, reload, failed writes/retry, New game preservation and unchanged campaign state. After rebuilding the native app, move/resize it through OS window controls, quit and reopen, then compare the observed `ui.json` placement. [Recorded macOS verification](docs/validation/window-checks.json) restored exact bounds. Physical monitor unplugging, Windows/Linux runtime, maximize/fullscreen transitions and forced termination remain separate qualification; the debounce can lose the latest fraction of a second on a crash.

For appearance, run `go test -p 1 ./pkg/ui ./pkg/app -count=1`. These focused packages cover explicit/system overrides, representative text contrast of at least 4.5:1 in both palettes, preference retention and native sizing at 320–1920px in both languages/modes. Build browser/native targets, serve the browser bundle, then run `NODE_OPTIONS=--max-old-space-size=1536 node tools/check-theme.cjs` (optional URL argument). It checks real System/Light/Dark controls, emulated live OS changes, changed map pixels, shared palette use, all five views at 320px Chinese, dialogs, expanded planning, restart, offline use, New game retention and unchanged campaign checkpoints for theme-only actions. Results and screenshots are `docs/validation/theme-*`; native system-dark rendering was also visually inspected on macOS. Physical devices, live native OS transitions and other desktop operating systems remain unqualified.

For construction funding explanations, `go test -p 1 ./pkg/ui -run '^TestFunding' -count=1` checks cents, reversed city selection, the named default alignment, available/negative cash, zero-floor behavior and committed/public exclusions. Native sizing uses the existing panel checks. After rebuilding and serving the browser bundle, `NODE_OPTIONS=--max-old-space-size=1536 node tools/check-funding.cjs` checks named-route arithmetic, signed deductions, review/alternative selection, exact-gap borrowing, clearing a real simulated cash deficit, committed-line exclusion and the 320px Chinese dark layout. Evidence is `docs/validation/funding-*`. The native build compiled; its updated funding flow was not visually qualified because the user was actively playing the running prior build.

For regional terrain changes, run `go test -p 1 ./pkg/terrain ./pkg/mapview ./pkg/ui`, rebuild the browser bundle and run `NODE_OPTIONS=--max-old-space-size=1536 node tools/check-terrain.cjs` against that server. Inspect Hokkaido desktop/phone and Kansai screenshots. The checker records toggle persistence, actual canvas changes, unchanged quotes/checkpoints, Chinese layouts and offline terrain/credits. See [terrain provenance](assets/terrain/README.md); shading is context, not engineering qualification.

For alignment geometry, install `tools/requirements-geography.txt` and run `python3 tools/check-route-geography.py`. This checks all six current paths against pinned land/lake masks in geographic and Equal Earth space, including the drawn coastline, with negative fixtures for the former bay/lake shortcuts and missing coverage. See [scope and provenance](data/geography/README.md). The terrain browser checker also compares every live WASM path with the checked catalog. Inspect both Hokkaido options and the Lake Biwa approaches after changes.

The earlier prototype checks below remain independently runnable.

| Area | Runnable check / evidence | What it establishes |
| --- | --- | --- |
| Metro inputs | Python structural checks in [data/README.md](data/README.md#reproduce-the-metro-structural-checks) | Raw hash/schema, identity/ranges, derived selection, and current 5M minimum; not accuracy of demographic estimates |
| State/save contracts | Focused Go package under [docs/contracts](docs/contracts), [Python checker](tools/check-state-contracts.py), [results](docs/validation/state-contract-checks.json) | Compile-checked type shapes, schema/fixture coherence, numeric precision/integrity checks, SQLite transaction/retry/backup contract; not a production restore validator or Go SQL adapter |
| Map prototype | [tools/check-map-preview.cjs](tools/check-map-preview.cjs), [results](docs/validation/map-checks.json) | Equal Earth projection, source values, 94 playable markers, responsive framing, region exclusions, latitude cushion, gestures, saved views, and timezone defaults |
| Actual ebitenui renderer | [tools/ebitenui-spike/check-browser.cjs](tools/ebitenui-spike/check-browser.cjs), [results](docs/validation/ebitenui-checks.json) | Real Go/WASM widgets, selected pointer/keyboard/touch-emulated paths, layout, scrolling, reports, and high DPI |
| Native rendering spike | Build instructions below and [spike notes](tools/ebitenui-spike/README.md) | The macOS arm64 target compiled during the recorded investigation; its window was not exercised |

The current map ledger covers 16 regional framing cases and 13 timezone fixtures, plus other interaction checks. The ebitenui ledger records the exact tested WASM hash and size. These ledgers are dated observations, not guarantees about later source edits; rerun the relevant checker after changing its inputs.

### Documentation changes

From the repository root:

```sh
git diff --check
```

Also inspect touched Markdown links, code examples, and claims against the owning files. Check new/untracked documents explicitly: `git diff --check` alone does not inspect their contents. Do not describe an intended package, command, workflow, or release gate as already implemented.

### Data changes

Run the standard-library Python block in [data/README.md](data/README.md#reproduce-the-metro-structural-checks). For catalog or camera changes, also run the map checker against a preview containing the updated embedded data. Check that the embedded catalogs/manifests match their repository records; editing a JSON file alone does not update the current HTML prototype.

Keep the raw 96-record attachment unchanged. Current derived output contains 94 major metros, retains Phoenix at exactly 5M, and excludes Brasília at 4.8M as well as the Manila scope exclusion. Camera changes must not alter source coordinates or identities. Cost changes need reconciliation of the relevant observations, units, regional assignment, and engineering/overhead arithmetic against the [cost research](docs/CONSTRUCTION_COSTS.md); a JSON parse alone is insufficient.

### State and save contracts

From the repository root, run only the reference package:

```sh
GO111MODULE=off GOWORK=off GOMEMLIMIT=768MiB GOMAXPROCS=2 go test -p 1 ./docs/contracts -count=1
python3 tools/check-state-contracts.py
```

The Python checker needs `jsonschema` as pinned in [tools/requirements-state-contracts.txt](tools/requirements-state-contracts.txt); install that file in a virtual environment if it is not already available. SQLite comes from Python's standard library. These focused checks do not download or qualify the selected production Go SQLite driver.

The [structural JSON Schema](docs/contracts/save-v1.schema.json) is generated from the reference DTOs. After an intentional unreleased-contract change, regenerate it and the example using the same focused Go command with `-update-contracts`, review those outputs, then rerun without that flag and run the Python checker. Do not use regeneration to hide an unintended compatibility change after v1 is released. The [state/storage design](docs/STATE_AND_STORAGE.md) defines when real migrations become necessary.

The reference codec is deliberately partial: complete required-field/schema checks and full domain/bundle validation remain gates for the production restore path. SQL fault tests use Python SQLite and injected transaction errors; abrupt-process/power-loss behavior, the real Go driver, process ownership, browser/native adapters, and complete server recovery still need qualification.

### Map preview

Prerequisites: Node, a resolvable Playwright installation, and its Chromium browser. This repository does not yet contain a Node dependency manifest or pinned browser-install command. If Playwright is installed outside normal resolution, set `NODE_PATH` to that installation's `node_modules` directory.

The current map HTML lives in the chat-owned visualization directory, outside this checkout. The checker is not yet self-contained in a fresh clone. Set `WWHSRT_MAP_PREVIEW_DIR` to the directory containing the current fragment `equal-earth-world-map.html` and its generated standalone wrapper `equal-earth-world-map-preview.html`. Regenerate the wrapper after changing the fragment; the wrapper embeds a copy. Packaging the source/generation workflow within the repository remains a reproducibility task.

In a dedicated terminal, serve that directory:

```sh
python3 -m http.server 8766 --bind 127.0.0.1 --directory "$WWHSRT_MAP_PREVIEW_DIR"
```

From the repository root in another terminal:

```sh
NODE_OPTIONS=--max-old-space-size=1536 node tools/check-map-preview.cjs http://127.0.0.1:8766/equal-earth-world-map-preview.html
```

The checker exercises both the wrapper iframe and the original fragment, writes `docs/validation/map-checks.json`, and refreshes map screenshots. It uses deterministic browser timezone fixtures instead of relying on the tester's timezone. Inspect affected screenshots after framing or label changes. CDN imports mean this prototype check needs access to its pinned D3 module; it does not establish offline cold launch of the eventual game.

Check region visibility and anchors in both orientations, Dhaka/Karachi camera exclusions, the hidden portrait initial views, absent portrait World button, manual zoom and the 3° latitude cushion, saved-camera precedence, and UTC+05:30's Dhaka-edge tie-break. The owning values are in the [preset manifest](data/prototypes/map-region-presets.v1.json), not this document. Stop the local HTTP server when finished.

### Connection-planning preview

The separate two-endpoint preview uses `connection-planner.html` and its generated `connection-planner-preview.html` wrapper in the same chat-owned directory. Serve that directory as above, regenerate the wrapper when the source changes, then run only its focused checks:

```sh
NODE_OPTIONS=--max-old-space-size=1536 node tools/check-connection-planner.cjs http://127.0.0.1:8766/connection-planner-preview.html
```

This checks the A/B selection state machine, marker selection and drag separation, dense-marker choices, stale estimate suppression, authored-route absence, alignment changes, funding review, draft restore, expanded-map fallback, native browser full-screen, and narrow layouts. It also checks fixture arithmetic/endpoint coordinates and records screenshots plus `docs/validation/connection-planner-checks.json`. The illustrative route and forecast fixtures do not validate real-world engineering or simulation results. Actual Ebitengine full-screen, physical-device Back/safe areas/software keyboards, and accessibility remain implementation gates. This prototype shares the region manifest but does not replace the full regional/timezone checker above.

### Go/ebitenui rendering spike

This is a separate module under `tools/ebitenui-spike`, not the future game's root build. Its [go.mod](tools/ebitenui-spike/go.mod) declares Go 1.26.4 and pins ebitenui 0.7.3 / Ebitengine 2.10.5. Recorded validation used Go 1.27.1. Use the `wasm_exec.js` shim from the same toolchain that built the WASM.

From `tools/ebitenui-spike`:

```sh
GOMEMLIMIT=768MiB GOMAXPROCS=2 GOOS=js GOARCH=wasm go build -p 1 -o web/main.wasm .
cp "$(go env GOROOT)/lib/wasm/wasm_exec.js" web/wasm_exec.js
python3 -m http.server 8765 --bind 127.0.0.1 --directory web
```

In another terminal, from the repository root:

```sh
NODE_OPTIONS=--max-old-space-size=1536 node tools/ebitenui-spike/check-browser.cjs http://127.0.0.1:8765/
```

The checker writes `docs/validation/ebitenui-checks.json` and screenshots. Review the generated evidence with the exact built WASM. Build outputs and the runtime shim are ignored; do not commit them accidentally. For native interaction, `go run .` from the spike directory opens a window and requires manual inspection; it is not a headless check. Stop the HTTP server after checking.

## Required checks as the game is implemented

These are planned gates, not existing scripts. Add concrete commands when their packages/workflows exist, and choose focused targets for local work rather than copying a nonexistent root test command.

| Layer | Required evidence |
| --- | --- |
| Architecture | Domain has no graphics, storage, network, or wall-clock dependency; public contracts do not expose mutable domain state; headless server starts without client graphics/audio |
| Domain examples/properties | Checked quantities and rounding, allocation bounds, connected services, independent construction, complete engineering coverage, cash/reservation reconciliation, and failed-command atomicity |
| Geography/cost data | Finite geodesics at poles/seams/near-antipodes; reviewed whole-route engineering; explicit river/ocean treatments; location-specific regional pricing without double charging |
| Corridor planner | Kinshasa–Lagos coastline-detour fixture: unsupported direct geometry does not block a qualified land alternative or force intermediate stations; absent authored coverage is reported accurately |
| Public application API | Quotes/forecasts are read-only; frame mutation cannot change world state; stale drafts and structured errors preserve accepted state |
| Persistence | Save/reload continuation equals uninterrupted play; failed writes/loads preserve committed/live state; data-version compatibility, single-writer browser storage, and portable export/import |
| Determinism | Identical command logs and versioned data yield matching canonical checkpoints across supported native targets and WebAssembly; replay/help/forecast behavior cannot affect them |
| Multiplayer | Shared OD demand and finite capacity, public construction and private drafts, concurrent construction conflicts and reservation release, paid track rights and settlement, idempotent retries, atomic month commits, and crash recovery |
| Diagnostics/audit | Correlated outcomes, atomic world/receipt/audit persistence, exactly-once business events after retry, redaction/export authorization, bounded queues/rotation, reproducible command histories, and unchanged checkpoints with logging disabled/verbose/failed sinks |
| Scheduling | Injected clocks test host start, the 30-second and 15-minute month intervals, auction cutoff (the longer of three minutes and one interval) and next eligible settlement, minimum-notice conversion, explicit pause and restart, clock jumps, and bounded outage catch-up |
| Full workflows | Construction-led and existing-track starts, disconnected networks and later joins, service/report/cash flows, auctions/open access, grants/loans, insolvency, and continuation |
| Presentation | Shared intent behavior, draft preservation, drawing/picking agreement, text/DPI reflow, reduced motion, phone rotation/keyboard/back/resume, and meaningful semantic controls |
| Localization | Complete Simplified Chinese message catalog, matching typed placeholders, offline fonts and translations, and identical simulation checkpoints across UI languages |
| Palette/motion/audio | Light/dark contrast and color-independent states; state-grounded rail/train effects, cue deduplication, no reconnect backlog, audio unlock/background/mute handling, bounded voices/markers, and unchanged results across effect/replay preferences |

For the [Simplified Chinese localization release check](docs/DESIGN.md#localization), have fluent reviewers complete a construction, service, and monthly-report flow in Simplified Chinese. Check missing glyphs, punctuation and wrapping at 320 CSS px and larger text settings, city search across aliases, and IME composition on native desktop and physical phones. Verify that switching languages preserves an open draft and that two clients using different languages see equivalent results and notices from the same world.

The [design's end-to-end scenarios](docs/DESIGN.md#end-to-end-scenarios) remain the full scenario checklist. Extend that list and relevant checks together when a gameplay rule changes. Use example tests and targeted Go fuzz/property tests for mathematical/allocation boundaries; a third-party property framework or blanket mutation-testing requirement is not necessary to begin.

## Property-based testing

Property tests are a core part of the planned domain/API suite. Use Go's native fuzzing with bounded generators and deterministic regression seeds. Generate meaningful valid worlds, networks, service plans, and command sequences alongside deliberately invalid boundary inputs. Check invariants after each accepted or rejected command, not only at the end of a generated campaign.

Priority properties include:

- Geometry is finite, nonnegative, symmetric where applicable, and handles seams and extreme coordinates. Engineering interval lengths reconcile with their alignment without gaps or double charges.
- Complete itineraries consume seats on every leg; carried passengers never exceed the shared market or capacity; disconnected components cannot create phantom journeys.
- Ledgers, access-payment transfers, obligations, and protected reservations reconcile under the documented rounding rules. Reservations cannot be spent twice.
- Rejected commands, forecasts, and inspections preserve the canonical world hash. Mutating returned frames cannot mutate the world.
- Retrying the same operation identity cannot duplicate a purchase, payment, or monthly transition. Stable ordering removes packet-arrival order from shared allocation outcomes.
- Save/reload continuation and uninterrupted execution produce the same checkpoint for the same versioned inputs and command log. Replay and contextual-help choices cannot change that checkpoint.

Use independent expected values or simpler reference models where practical; assertions that repeat the production algorithm provide weak evidence. Retain minimized failing inputs in the regression corpus with the seed and data/algorithm versions. Keep CI fuzz campaigns time/resource bounded and make failures reproducible. Cross-target determinism also needs the explicit native/WASM checkpoint gate; one host's fuzz results cannot establish it.

## Mutation testing

Use targeted mutation testing to assess whether the tests detect plausible defects in critical rules. Introduce it once the first domain/application slice and its example/property tests exist. It is planned verification work, not tooling that is already installed or a requirement to mutate every rendering/data file.

Prioritize money and rounding, reservation handling, capacity allocation, company authorization, atomic command failure, and idempotent persistence/monthly commits. Representative mutations change an eligibility boundary, omit a reserved-funds deduction, skip a company-ownership check, allocate passengers beyond seats, or allow a retried operation to post twice. The corresponding behavior tests should fail for those defects.

Run bounded, scoped mutation jobs in hosted CI or deliberately scheduled qualification runs. Review surviving mutants: add a behavior test when a survivor exposes a real gap, and document equivalent/unreachable mutations separately. Report tested scope and surviving defects; a single repository-wide mutation percentage must not substitute for evidence that the critical rules are protected. Select and pin the Go mutation tool after checking its compatibility with the chosen toolchain. Local runs remain subject to the verification policy above.

## CI and release qualification

The [walking-skeleton workflow](.github/workflows/milestone1.yml) establishes hosted gates for initial domain/API tests, race checks, bounded fuzzing, architecture boundaries, native/browser builds, browser workflows, and the native-headless/WASM checkpoint. Extend it with production-save compatibility, full scenario policies, and dedicated-server/display-free checks as those systems arrive. Do not run full suites silently through local hooks.

Measure performance on named hardware and pinned browsers using the workload in [the design](docs/DESIGN.md#initial-budgets-to-qualify). The 60 FPS ordinary / 30 FPS stress replay goals, cached preview under 200 ms, and full forecast/month under 2 seconds are targets, not prototype results. Record peak memory and compressed shipped WASM size. Do not treat the renderer spike's raw size or Africa2ice's budgets as qualification of the game.

Before release, qualify actual iOS Safari and Android Chrome, software keyboards/IME, safe-area and keyboard occlusion, background termination/resume, storage recovery, and the complete 320px workflow. Repeat on desktop with keyboard-only use, text scaling, and fractional/2× device scales. Test screen-reader and native accessibility through actual semantic controls. The native canvas exposes no semantic management interface; the browser now has semantic HTML controls. Touch emulation and a native compile do not close these gates.

First-time-player sessions must cover both starting strategies without a required tutorial, plus experienced play with hints disabled. Verify that drafts carry from the planning lobby across host start and several live month boundaries. A stable online company must survive a 30-minute absence at the proposed default and eight hours at the 15-minute host setting. Use injected clocks for correctness checks and qualify real host timing separately.

For meaningful verification, record source/data identities, toolchain/browser/OS versions, executed commands, outcomes, and remaining gaps. Keep screenshots and ledgers tied to the build they describe. An unresolved required gate is reported as unresolved rather than hidden behind a prototype `passed` flag.

For capacity diagnostics, run the focused domain/application/UI/app packages or `go test -p 1 ./internal/domain ./internal/application ./pkg/ui ./pkg/app -run 'Test(Capacity|NativeCapacity)' -count=1`. After building web/native/headless targets, run `NODE_OPTIONS=--max-old-space-size=1536 node tools/check-capacity.cjs <test-server-url>`. It reproduces a 4+4+4+1 private-section conflict through real controls, verifies named contributors, available draft frequency, blocked Apply, cleared stale forecast, Chinese phone layout/reload, a real remedy and headless/WASM parity. See [CAPACITY.md](docs/validation/CAPACITY.md).

For station-connected legacy geometry, run `go test -p 1 ./internal/application ./pkg/render`
and `python3 tools/check-route-geography.py`. The latter checks all six frozen paths
and their six displayed replacements, including water masks in geographic/Equal
Earth space and unchanged priced kilometre bounds. After building web, run
`NODE_OPTIONS=--max-old-space-size=1536 node tools/check-osaka-map.cjs <server-url>`.
It checks actual selected-track pixels at Kyoto/Shin-Osaka, absence of the old
city-centre approach, common endpoints, reload and unchanged save bytes. See
[OSAKA_STATION.md](docs/validation/OSAKA_STATION.md).

For existing public pairs in Build, use focused `TestExistingChineseConnections`,
`TestNativeExistingPublicConnection`, and `TestConstructionOpportunities` checks.
After building web, run `NODE_OPTIONS=--max-old-space-size=1536 node tools/check-existing-connections.cjs <server-url>`.
The [validation note](docs/validation/EXISTING_CONNECTIONS.md) covers the three
reported Chinese pairs, no duplicate construction, explicit existing-line status,
service navigation and unchanged company state in both languages.

For unused-source map filtering, run `go test -p 1 ./pkg/render` and
`python3 tools/import-public-network.py --check`, then build web/native and run
`NODE_OPTIONS=--max-old-space-size=1536 node tools/check-playable-map.cjs <server-url>`.
The renderer test covers all admitted paths, visible station association, retained
Harbin–Shenyang/Changchun–Jilin and removal of the reported unused parallel fragment.
The browser check verifies actual retained/removed rail pixels and unchanged saves.
See [PLAYABLE_MAP.md](docs/validation/PLAYABLE_MAP.md).

For the basemap, run focused `go test -p 1 ./pkg/render`,
`python3 tools/test-basemap.py`, `python3 tools/prepare-basemap.py --check`,
`python3 tools/check-launch-prospects.py` and the route geography checker. A change
to what `pkg/render` draws as land also needs the native app's memory compared
before and after while panning at a large window: the Go tests do not draw.
`tools/check-coastlines.cjs` checks Japan/China coastal water and fine shoreline
pixels in light/dark modes with terrain on/off, plus unchanged save bytes/checkpoint.
It passed locally under the footprint watch (peak 1,652 MiB) and runs in hosted CI, which uploads a fresh `coastline-checks.json` and screenshots
as evidence; the copies in `docs/validation` predate the single-outline basemap
until that evidence is committed. See [COASTLINES.md](docs/validation/COASTLINES.md).


## Worldwide construction catalog

See [LAUNCH_PROSPECTS.md](docs/validation/LAUNCH_PROSPECTS.md). Run sequentially:
`python3 tools/prepare-launch-prospects.py --check`,
`python3 tools/import-public-network.py --check`,
`python3 tools/check-launch-prospects.py`,
`python3 tools/test-network-coverage.py` and the strict public-network audit.
The new mask checker covers all 68 new alignments in geographic and Equal Earth
coordinates, explicit water structures, regional quantities, all 70 spec identities,
Morocco's HSR boundary and deliberate Gulf/Johor errors. The unchanged Japanese
alternatives retain their separate detailed route mask.

Focused domain/application/UI/mapview/app/render packages cover all 70 funded
build commands, opening boundaries, real regional construction/loan/revenue/reload
campaigns, Tokyo–Sapporo through-running, v7→v8 migration including retired Moroccan
services, and native 70-entry selection in both languages at 320/1440 px. The
native dropdown test exercises controlled initialization and stale deferred events.
No TypeScript compilation is authorized by this section.
Rebuild native, web and headless after data changes. Browser interaction checks
run under the footprint watch in the local verification policy above.

## Network bridge expansion

The connector update uses focused `TestConnectors*` application checks for v9→v10
migration, nested historical accounts, failure atomicity, ordered public services,
new construction/opening boundaries, through passengers and reload. Native
`TestNativeWorldwideProspectPickerFitsAndSelectsDedicatedHSR` checks all three new
projects in English/Simplified Chinese at 320/1440 px. Shared-resource assertions
now include the two physical Maibara sections beneath older Kyoto–Nagoya aliases.

Run `python3 tools/check-network-bridges.py` with the existing offline preparation,
import, launch geography, regional, coverage and strict audit checks. The
operating manifest has 14 network contracts; bridge cuts must fail even when old
express aliases remain. Focused Go packages are domain, application, UI, mapview,
app and render, with the existing memory/process limits. Rebuild native, web and
headless after catalog changes. Browser checks follow the footprint watch in the
local verification policy above. See
[NETWORK_BRIDGES.md](docs/validation/NETWORK_BRIDGES.md).
