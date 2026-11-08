# Testing

The repository is at the specification/prototype stage. There is no root game Go module, simulation test suite, dedicated server, or CI workflow yet. The runnable checks below cover the existing data and rendering prototypes. Passing them does not qualify the future game.

Use the [architecture](ARCHITECTURE.md) for system boundaries, [game design verification contract](docs/DESIGN.md#13-verification-and-performance) for domain properties and scenarios, and [UX acceptance criteria](docs/UX.md#10-acceptance-and-questions-to-test) for player workflows. Existing results and limitations are recorded in the [validation report](docs/validation/README.md).

## Local verification policy

- Use focused checks appropriate to the change. Documentation-only edits need document/reference checks, not game builds or browser runs.
- Do not run a full local suite on this machine without the user's express advance permission. Long/full checks belong in CI once it exists.
- Do not run local TypeScript compilation, directly or indirectly through scripts or hooks. Use hosted/CI typechecks if TypeScript is introduced.
- Inspect scripts, hooks, and child commands before running automatic checks. Serialize memory-heavy work. For Node-based checks, cap old space at 1536 MB per process; stop if combined memory-heavy check processes exceed 2 GiB RSS.
- State exactly which checks ran and any remaining limits. A build is not an interaction check; browser touch emulation is not physical-device qualification.

## Existing checks

| Area | Runnable check / evidence | What it establishes |
| --- | --- | --- |
| Metro inputs | Python structural checks in [data/README.md](data/README.md#reproduce-the-metro-structural-checks) | Raw hash/schema, identity/ranges, derived selection, and current 5M minimum; not accuracy of demographic estimates |
| Map prototype | [tools/check-map-preview.cjs](tools/check-map-preview.cjs), [results](docs/validation/map-checks.json) | Equal Earth projection, source values, 93 markers, responsive framing, region exclusions, latitude cushion, gestures, saved views, and timezone defaults |
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

Keep the raw 96-record attachment unchanged. Current derived output contains 93 major metros, retains Phoenix at exactly 5M, and excludes Brasília at 4.8M as well as the Manila/Taipei scope exclusions. Camera changes must not alter source coordinates or identities. Cost changes need reconciliation of the relevant observations, units, regional assignment, and engineering/overhead arithmetic against the [cost research](docs/CONSTRUCTION_COSTS.md); a JSON parse alone is insufficient.

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
| Multiplayer | Shared OD demand and finite capacity, company authorization/private frames, idempotent retries, acknowledgments, atomic month commits, and crash recovery |
| Scheduling | Injected clocks test 15-minute boundaries, absent players, backward/forward clock changes, and the proposed bounded outage catch-up; do not wait real hours in ordinary correctness checks |
| Full workflows | Construction-led and existing-track starts, disconnected networks and later joins, service/report/cash flows, auctions/open access, grants/loans, insolvency, and continuation |
| Presentation | Shared intent behavior, draft preservation, drawing/picking agreement, text/DPI reflow, reduced motion, phone rotation/keyboard/back/resume, and meaningful semantic controls |

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

No CI configuration is present yet. When implementation starts, establish hosted gates for native/browser builds, domain/API tests, bounded fuzzing, architecture boundaries, save compatibility, browser workflows, and cross-target canonical checkpoints. Include the dedicated server without a display/audio device and offline single-player with networking unavailable. Add full suites to hosted CI rather than silently running them through local hooks.

Measure performance on named hardware and pinned browsers using the workload in [the design](docs/DESIGN.md#initial-budgets-to-qualify). The 60 FPS ordinary / 30 FPS stress replay goals, cached preview under 200 ms, and full forecast/month under 2 seconds are targets, not prototype results. Record peak memory and compressed shipped WASM size. Do not treat the renderer spike's raw size or Africa2ice's budgets as qualification of the game.

Before release, qualify actual iOS Safari and Android Chrome, software keyboards/IME, safe-area and keyboard occlusion, background termination/resume, storage recovery, and the complete 320px workflow. Repeat on desktop with keyboard-only use, text scaling, and fractional/2× device scales. Test screen-reader and native accessibility through actual semantic controls. The current canvas exposes no semantic management interface; touch emulation and a native compile do not close these gates.

First-time-player sessions must cover both starting strategies without a required tutorial, plus experienced play with hints disabled. A stable online company must survive the proposed eight-hour unattended target without repetitive acknowledgments. Correctness should use accelerated/injected clocks; separately qualify real host timing and operational behavior.

For meaningful verification, record source/data identities, toolchain/browser/OS versions, executed commands, outcomes, and remaining gaps. Keep screenshots and ledgers tied to the build they describe. An unresolved required gate is reported as unresolved rather than hidden behind a prototype `passed` flag.
