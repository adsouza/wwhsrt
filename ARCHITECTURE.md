# Architecture

This document describes the architecture of World-wide High-speed Rail Tycoon. The [first-business slice](docs/validation/FIRST_BUSINESS.md) implements domain/application/API boundaries, network demand, transfers, capacity and finance, construction and funding, dedicated train leases, native ebitenui controls, an Ebitengine map, a semantic browser HUD, a headless CLI and development saves. The prototype public-network import now feeds the playable catalog; production data qualification, the production save contract and multiplayer server remain planned. Earlier widget prototypes remain reference artifacts.

Milestone 2 derives station ownership/readiness from the pinned starting network and committed projects; stations open with their project. A single named train-family projection publishes the constants used by fleet validation and settlement. Construction component projections use the same existing cost calculation and cent rounding. These projections and pure UI guidance are excluded from canonical saves/checkpoints, so this milestone needs no version-2 save migration or financial reset. Detailed station capex/upgrades and multiple train families remain outside this introductory model. Milestone 3 adds shared track/station call limits and a network solver, as described below.

The [game design](docs/DESIGN.md) owns gameplay rules and domain invariants; [UX](docs/UX.md) owns interaction behavior; [testing](TESTING.md) records verification commands and release gates. The [state/storage contract](docs/STATE_AND_STORAGE.md) defines Go type shapes, portable save v1, the SQLite server backend and migration policy. Numeric camera settings and cost assumptions live in their linked manifests. This document explains the system boundaries that realize those requirements. Proposed balance and recovery policies retain their status from the design; listing them here does not make them calibrated or implemented.

## Architecture style

Use Clean Architecture, domain-driven design (DDD), command/query responsibility segregation (CQRS), and explicit decorators for cross-cutting concerns. These are design commitments for the forthcoming game implementation; the isolated prototypes do not yet implement this architecture.

| Pattern | Application to this game |
| --- | --- |
| Clean Architecture | Dependencies point inward toward rules and use cases. Application-owned ports isolate storage, scheduling, networking, and diagnostics. Composition roots wire concrete adapters; the domain remains headless and independently testable. |
| DDD | Model the railway business using its own terms: projects, alignments, services, trainsets, access rights, obligations, and journeys. Entities have stable IDs; value types encode quantities and invariants. `World` is the aggregate root for atomic monthly/shared-market consistency, with company ownership enforced inside its operations. Construction, operations, demand, and finance are cooperating domain areas within this initial model. |
| CQRS | Separate state-changing commands from read-only inspections, forecasts, and frame projection. Both operate over the same authoritative world; UI-facing read models are isolated projections. Separate command/query databases and event sourcing are not required. |
| AOP through decorators | Wrap selected use-case and storage ports for structured logging, timing, and diagnostic outcomes, composed explicitly in Go. Keep authorization, business validation, reservations, and monthly transaction order visible in application/domain code. Runtime weaving or reflection-driven interception is not required. |

Keep aggregate boundaries aligned with invariants rather than creating a repository for every entity. A command changes domain state only through its aggregate operations. Dependency boundaries, query purity, frame isolation, and invariant preservation must be verified as described in [TESTING.md](TESTING.md#required-checks-as-the-game-is-implemented).

## One rules engine, two execution modes

Use a shared Go application/domain implementation for offline single-player and an authoritative dedicated multiplayer server. Desktop and browser clients use the same driving contract. Composition selects an in-process implementation or a remote adapter; presentation does not implement a second economy.

```mermaid
flowchart LR
    UI[Native or browser presentation] --> API[gameapi driving contract]
    API --> Local[In-process application]
    API --> Remote[Remote adapter]
    Remote --> Server[Headless server application]
    Local --> Rules[Shared domain code]
    Server --> Rules
    Local --> LocalStore[Local persistence port]
    Server --> ServerStore[Server persistence port]
```

Local and server worlds are separate instances using the same rules code. Offline play has no server, login, network, or wall-clock progression requirement. The online server owns the shared world, private acknowledged plans, and schedule; clients submit intentions and receive authorized results. Clients animate the representative operating day from resolved summaries rather than receiving authoritative train positions every frame.

## Package boundaries

The following is the target layout. Milestone 1 implements a small subset; its actual inventory and qualification are linked above.

| Package | Responsibility | Boundary |
| --- | --- | --- |
| `internal/domain` | World, construction, fleet, services, demand, itineraries, finance, monthly resolution | Pure rules; no renderer, storage, networking, audio, or wall clock |
| `internal/application` | Commands, forecasts, validation orchestration, frame projection, persistence mapping and coordination | Imports domain and public contracts; owns required adapter ports |
| `pkg/gameapi` | Commands, isolated frame DTOs, stable error codes, driving ports | No dependency on domain entities, rendering, or transport |
| `internal/adapters` | Filesystem/IndexedDB, remote game port, server transport/storage, diagnostic sinks | Implements ports; does not become a second owner of mutable world state |
| `pkg/ui` | Selection, drafts, scene stack, presentation preferences | Emits shared intents and consumes frames |
| `pkg/hud` | Management controls and intent translation | No direct domain mutation |
| `pkg/render` | Equal Earth map, layers, replay illustration, draw/pick transforms | Screen geometry cannot determine engineering distances or simulation outcomes |
| `pkg/app`, `pkg/audio` | Client composition, input/lifecycle, platform audio | Platform effects remain outside the domain |
| `cmd/server` | Headless composition and world coordinators | Must not import the client host or initialize graphics/audio |
| `internal/verification` | Public-API player policies and canonical checkpoints | Exercises driving contracts rather than privileged world mutation |
| `data`, `tools` | Versioned scenario inputs and offline preparation/verification | Prototype inputs are not automatically qualified production data |

Only application code translates public commands into domain operations and domain state into frames. Presentation never receives a mutable `World` pointer. Returned frames must be isolated from domain state, carry relevant revisions, and contain only information authorized for that company. Storage and network work use immutable copies rather than concurrently mutating a live world.

Africa2ice supplies reference patterns for these boundaries, not a runtime dependency. Reuse isolated frames, shared intents, guarded drafts, consistent coordinate transforms, asynchronous storage, and logging decorators selectively. Preserve source notices when copying code. Replace its ecological rules, fixed desktop layout, and sequential onboarding. The [design's reference notes](docs/DESIGN.md#11-architecture-informed-by-africa2ice) identify the inspected components.

## World ownership and monthly resolution

One `World` is the atomic consistency boundary. It owns companies, infrastructure and access rights, fleet, accepted plans, construction, auctions, protected funds, finance, and resolved history. Cities, corridor alternatives, and balance tables are immutable versioned inputs. A company can own several disconnected rail components; individual services and passenger journeys still require physical connectivity.

Commands either produce a complete valid result or leave the prior world unchanged. Use typed IDs and quantities, checked integer money, explicit units and rounding, and stable ordering for consequential allocation and tie-breaks. Forecasts and inspections are read-only.

Monthly resolution follows the [canonical order](docs/DESIGN.md#monthly-resolution-order): freeze plans and eligible assets, allocate shared capacity, derive delivered service, allocate the shared OD market to complete itineraries, post and reconcile cash flows, advance obligations/projects/orders, settle due auctions, complete next-month assets, and publish reports/checkpoints. An asset completed in this transition becomes usable next month.

The same seed, data/algorithm versions, and accepted commands must produce matching canonical checkpoints on supported native and WebAssembly targets. The domain does not read a wall clock or consume randomness during queries, forecasts, saves, or replay. Any future randomness must have serializable, versioned ownership. The operating-day replay is presentation state and cannot advance the business simulation.

## Commands, drafts, and multiplayer

Expose commands and queries through the game port, including construction quotes, service forecasts, accepted plan changes, snapshots, and storage operations. Keep stable machine-readable constraint codes separate from UI wording. Local/admin state hashes and full-world snapshots are not ordinary multiplayer player queries.

A UI draft is separate from an accepted plan. Previewing or editing does not spend funds. Applying online must receive a durable acknowledgment naming the revision and first effective month. After stale assumptions or an ambiguous response, retain the draft, refresh authorized state, and query the original operation identity before retrying a financial action.

The server serializes accepted operations within each world. Requests have version/world/phase/revision identities and idempotency keys; company identity comes from the authenticated session. Authorization covers queries and forecasts as well as mutations. Invite codes provide admission to private games, not a substitute for revocable sessions.

Publish committed project and access-offer summaries from the authoritative world after durable commit. The coordinator settles concurrent construction claims against the unordered pair, reviewed competition, and physical section identities. Service authorization and fees use effective track agreements and station rights before shared capacity is allocated.

New multiplayer worlds begin in a planning lobby with a fixed opening snapshot and a host-controlled start. The server scheduler sets the first deadline one full interval after the host starts. Online months then resolve on the fixed interval chosen at world creation, including while players are absent. The proposed default is 30 real seconds. Persist the interval, deadline, phase, acknowledged plans, and resolution identity. Commit the complete month durably before publishing it. The proposed recovery policy completes an already committed operation idempotently, resolves at most one overdue month after an outage, and schedules the next deadline one interval after recovery. Qualify that policy before release. Standing services and commitments continue according to acknowledged plans. Shared-world spending requires server acknowledgment.

Online auction cutoffs belong to the server host record. The server enforces the bid deadline (the longer of three real minutes and one interval) and includes the sealed auction IDs in the durable monthly lock. The domain settles those pinned bids using the same deterministic clearing rules as offline play.

## Geography and data preparation

Keep three distinct representations:

1. Geographic city coordinates and alignment polylines describe world locations.
2. Reviewed engineering intervals and rail speed profiles determine feasibility, length, cost, upkeep, construction duration, and running time.
3. Equal Earth projection and camera transforms determine drawing and picking only.

The first version uses a precomputed corridor catalog with curated alternatives. Passenger/service routing runs at runtime; global civil-engineering route generation does not. A direct line crossing water cannot invalidate a qualified overland alternative. Kinshasa–Lagos is the coastline-detour acceptance case: one continuous project may bend without adding intermediate stations. Missing catalog coverage must be distinguished from geographic impossibility.

The offline data pipeline must qualify full-route engineering coverage, terrain, shorelines, crossings, borders, and rights. Starting track is admitted as corridors between catalog cities, never by track tags alone. Missing terrain is unqualified data, never free flat land. A coarse Natural Earth overview cannot price tunnels or establish buildability. Cross-border costs use the jurisdiction of each engineering interval, not whichever endpoint is cheaper.

Current inputs and their scope:

| Input | Owning record |
| --- | --- |
| Original 96-record metro attachment and unresolved provenance | [Raw metadata](data/raw/metropolitan_areas_5m.metadata.json) |
| Current 94 major playable metros, each at least 5M; Manila and Brasília excluded | [Selection policy](data/prototypes/playable-metro-selection.v1.json) and [derived catalog](data/prototypes/playable-metros.v1.json) |
| Regional names, anchors, exclusions, orientation rules, latitude cushion, timezone defaults | [Camera preset manifest](data/prototypes/map-region-presets.v1.json) |
| Prototype country outlines | [Natural Earth manifest](data/prototypes/natural-earth-110m.metadata.json) |
| Regional construction assumptions, observations, normalization, and engineering factors | [Cost ledger](data/costs/construction_costs.v1.json) and [research](docs/CONSTRUCTION_COSTS.md) |

Preserve raw inputs and derive game catalogs with explicit transformations. Record source/version/hash identities in released scenario bundles and saves. The 94-metro prototype is not the complete worldwide corridor catalog; smaller intermediate cities and existing HSR stops remain part of the planned broader catalog.

## Presentation and camera state

Ebitengine is the proposed desktop/browser renderer, with ebitenui a demonstrated visual-widget candidate. The HTML/D3 map prototype is a feasibility/reference artifact, not an existing Go map implementation. Both product hosts must share the game port and rules regardless of their eventual control-rendering technology.

Use one forward/inverse camera transform for map drawing and picking. Layout uses logical space; framebuffer scale improves raster quality without shrinking controls. Phone task views reflow instead of scaling down a fixed desktop canvas. Contextual guidance reads frames and presentation preferences; enabling it cannot change simulation results. The shared `pkg/ui` concept model supplies inline disclosures and searchable Field Notes to both hosts. Seen/dismissed/explained/completed flags, global mute and reference return context live in local UI preferences, outside campaign exports and hashes. Browser help events are serialized and versioned independently of game revisions so asynchronous rendering cannot restore dismissed suggestions. Successful accepted commands record learning progress; help navigation itself cannot dispatch a business command.

Camera, selection, dismissed hints, replay position, and drafts are local UI state, separate from authoritative campaign state. Timezone-selected portrait defaults are calculated locally and give way to saved views. Region framing and the cushioned zoom floor follow the [UX camera contract](docs/UX.md#regional-camera-presets); retain those details in the manifest rather than duplicating regional tables here.

Localize in the client using stable message keys, typed arguments, and display names keyed by catalog ID. Frames, commands, saves, and server events carry canonical IDs, quantities, and reason codes so the client can render them in its own language. Keep the language preference outside campaign state. Bundle translation catalogs and fonts with the desktop and web clients. Use English as the fallback for missing interface messages and the source label for untranslated catalog names. [Localization requirements](docs/DESIGN.md#localization)

Milestone 1 selects ebitenui for native management controls and semantic HTML for the browser HUD; both use the same application commands and frame DTOs. Browser command handling runs independently of the renderer's update loop, so hiding the map cannot suspend a business command. Native company-name entry adapts Ebitengine's IME field into the ebitenui presentation instead of relying on raw character input. The native macOS accessibility tree currently exposes the window but no canvas management controls; browser semantics provide the accessible host for this slice. Physical-phone and actual native IME/screen-reader qualification remain open.

The [presentation contract](docs/PRESENTATION.md) requires a cheerful bright palette and state-grounded construction/train animation and sound. Renderer/audio consume frames and deduplicated presentation cues, never mutate campaign state, and honor reduced motion, mute and lifecycle preferences.

## Diagnostics and audit

Pervasive structured diagnostics are required, through explicit Go `log/slog` decorators/adapters and domain-returned facts. The [diagnostics contract](docs/DIAGNOSTICS.md) defines coverage, correlation, retention, privacy, support exports and reproducibility. Rolling diagnostic sinks are bounded and non-authoritative. Server canonical game requests and ordered audit events persist with world/host/receipt commits in SQLite; retry cannot duplicate a business event. Audit data is admin-private and separately projected for player-visible activity. Logging verbosity, dropped verbose events and audio/render tracing cannot alter canonical outcomes.

## Persistence and lifecycle

Use versioned JSON snapshots behind application-owned persistence ports, with filesystem and IndexedDB adapters. Persist a consistent world, not independent per-city/train repositories. Include data and algorithm identities, obligations, protected reservations, owned assets, plans, reports, and replay summaries. Server storage additionally owns scheduling and operation identities; authentication secrets stay separate from downloadable game data.

The dedicated server uses embedded SQLite via `database/sql` and `modernc.org/sqlite`, with a single connection/writer and transactional world/host-state/receipt commits. Entities stay in a versioned whole-world JSON snapshot; sessions and invite hashes stay in separate operational tables. Save, host-record and database schemas begin at v1; no historical game-save migration is needed yet. See the [detailed contract and reference artifacts](docs/STATE_AND_STORAGE.md).

Decode and validate into an isolated replacement world before swapping live state. Incompatible or corrupt loads preserve the current campaign. Serialize writes; acknowledge saves only after durable commit; preserve the previous slot on failure. Support portable browser export/import and a single browser storage writer. UI preferences/drafts use separate versioned records.

Native and browser lifecycle adapters handle audio activation, visibility, storage failures, and resume. Browser background timers do not control online deadlines. On reconnect/resume, refresh world, schedule, and uncertain operation status before enabling commitments.

## Decisions still to qualify

- Accessible browser/native control strategy and physical iOS/Android keyboard, safe-area, and lifecycle behavior.
- Native/browser transport, and implementation/qualification of the chosen SQLite server adapter and platform storage adapters.
- Worked deterministic rounding/shared-capacity allocation and cross-target checkpoint equality.
- Qualified existing-HSR inventory, corridor catalog, economic calibration, and data redistribution rights.
- Measured mobile/stress performance, memory, and shipped WebAssembly size.

Use [TESTING.md](TESTING.md) to distinguish existing prototype evidence from these implementation and release obligations.

### Native text editing

The Company name editor participates in Ebiten UI layout, input layers and focus traversal, with Ebitengine `textinput.Field` retaining IME composition. The pinned Ebiten UI 0.7.3 `TextInput` has no native clipboard/IME integration and uses rune positions as byte slices in selection extraction/deletion; using it unchanged would corrupt non-ASCII selections. `pkg/app/name_input_native.go` supplies grapheme-safe editing/selection rendering and bounded undo history; `github.com/atotto/clipboard` supplies explicit clipboard reads/writes only on editing shortcuts, with failures preserving the draft. Clipboard contents are not logged. The browser keeps its semantic HTML input. This adapter can be replaced when the upstream widget meets the same Unicode/IME/clipboard contract.

### Network game and development save v9

`internal/domain/routes.go` resolves immutable service patterns onto physical corridor sections. `journeys.go` builds called-station rides and bounded transfer itineraries; `allocation.go` conserves finite markets and rations complete requests across shared resources with stable rounding. `network.go` uses the same solver for forecasts, settlement and read models. Track paths are allocated before directed-leg seats. `day.go` derives a repeating representative day from the last committed plan and aggregate loads; `pkg/render/day.go` draws/picks it without changing the business world.

The network catalog is pinned separately in `assets/scenario/network.json`. It declares game trip budgets, attraction/road regions, rounded outside-mode distances, stopping patterns and resource limits. It is not a surveyed railway/demographic dataset. Forecasts replace one plan on a copied world and compare whole-company outcomes. `extend_service` atomically suspends a source pattern, transfers its fleet, orders only explicitly approved additional leases and applies the longer service, subject to the same revision, capacity and persist-before-acknowledgment rules.

The bounded development save is version 10, rules `network-v10`, with `public_sha256`, `catalog_since`, `network_sha256`, `network_since`, `economy_sha256` and `economy_since`. Version-2/3 restores retain their original reports and company state; the next resolved month uses the expanded public catalog. The frozen original catalog validates earlier reports. Opening a legacy save does not write it; the next successful command commits the upgrade atomically. Restore recomputes network-era monthly service results from requested plans and pinned data. Ephemeral detailed markets, route frames and replay positions stay out of economic saves/checkpoints. Existing `corridor_id` service/fleet fields retain their wire names as stable service-pattern IDs; construction corridors remain physical assets.

The embedded `assets/scenario/economy.json` pins the expanded-world gameplay calibration. `Catalog.WithEconomy` validates bounds and caches destination-weight totals; `World` selects legacy or calibrated rules at the saved monthly cutover. Version-4 worlds open with `economy_since = months`: old reports replay using the old demand and cost rules, future months use the calibrated policy. The v5 reader requires the exact policy hash and a valid cutover. Opening and failed commands preserve the original bytes; the next successful command persists the upgrade. Frozen v2/v3 catalogs keep their legacy economics while validating earlier history. Policy changes need an explicit new identity/compatibility path, never silent substitution.

Read-only route suggestions are computed for the frame, not inside every simulation lookup. Accepted fare/frequency plans remain authoritative. Forecast cost components and fixed-mix break-even are domain projections shared by both clients; they are excluded from canonical saves/checkpoints. See [economic calibration and verification](docs/validation/ECONOMY.md).

Public topology repairs are a second independent historical boundary. The immutable `public-network-v1.json` validates pre-repair expanded reports; the repaired catalog carries `WithPrevious` as well as the original 12-city `WithPrior`. `World.atMonth` selects topology first and the original-catalog epoch second, then the economy cutover controls demand and costs. Save v6 adds `topology_since` and `prior_public_sha256`; v4/v5 opens validate with the frozen expanded catalog before upgrading in memory. Published route IDs and timings remain stable so active fleets retain valid requirements. That original v6 transition is retained for historical validation; current successful commands persist v9 atomically. The [source-node pipeline and connectivity gates](docs/validation/NETWORK_TOPOLOGY.md) document provenance and limitations.

Capacity validation and its explanation share one domain projection (`ownedCapacityUsage`). Forecasts publish requested track paths/station calls, each contributing service, the current draft contribution and the maximum frequency for that resource with other services unchanged. Typed capacity errors carry the violated-resource projection while retaining stable error codes. Both clients format those values through `pkg/ui/CapacityDetails`; neither recreates the validation rule. These diagnostic projections are excluded from saves/checkpoints and do not alter public-capacity allocation or private-capacity limits.

The task workspace uses `pkg/ui/workspace.go` for the unobscured map/inspector rectangles, width clamps and responsive dock choice in both clients. Local `ui.State` stores preferred panel width, dock placement, task tabs and bounded disclosure flags. Temporary compact fallback never rewrites desktop preferences. Native drawing translates an offscreen map into that rectangle; picking, dragging and pinch centers subtract the same origin. Map widgets and open combo popups consume their own pointer input. Native selection lists use ebitenui comboboxes; browser tablists and an accessible separator use semantic HTML. Task links choose the relevant subtab, and ordinary tab changes retain unapplied drafts. These preferences and projections require no campaign schema migration and cannot dispatch business commands.

## Prototype public network

`tools/import-public-network.py` compiles a pinned source snapshot into an immutable catalog and map layer. See [the import contract](data/imports/README.md). Real station locations are separate from metro coordinates. Strategic hub contraction limits service/demand graph size while retaining every visible source station and polyline for cartography. Source station inspection never changes company state. Operating patterns use graph paths, not geographic endpoint chords; small endpoint/platform joins and remaining source gaps are explicit.

The catalog has indexed city/corridor/distance lookups and source-pinned integer distances for native/WASM determinism. Journey enumeration finds itineraries for all reachable destinations in one traversal per origin/class. Sparse reports retain all company-reachable markets; `unserved_demand` accounts for the entire remaining world budget, and the read-only market inspector projects outside-mode detail on demand. An origin with no company departure bypasses OD allocation while preserving its full demand budget. Disconnected source components cannot produce through services. Service templates are game plans, not imported operator timetables. Original service/fleet identities stay stable; the legacy Nagoya–Osaka express explicitly shares the new Kyoto sections' physical capacity.

The map caches global land and the static public-rail layer by camera/theme, then overlays the selected service, active projects, city stations and representative trains. Both clients expose the same orientation-aware region manifest and optional route search. Native search uses the existing Unicode selection/paste/IME adapter. Camera, search, station inspection and language are presentation preferences, excluded from canonical saves.

The legacy Japan route IDs use `rail-display-paths.json` for frame geometry. This
presentation-only projection follows the imported railway/station hubs; the frozen
catalog still owns costs, lengths, timings and save validation. Correcting a map
endpoint therefore does not invalidate v4 saves. All map/highlight/train consumers
use the same projected frame paths. See [Osaka correction](docs/validation/OSAKA_STATION.md).

`playable-rail-map.json` compiles the admitted public corridor paths and associated
source station IDs into the cached background layer. The renderer validates its
catalog fingerprint, and rendering tests compare every line with the application
frame. Source-only geometry remains in the pinned input for audits; neither it nor
unrelated station dots are painted as playable infrastructure. This projection is
excluded from economic saves, so map decluttering requires no campaign migration.

The basemap is one worldwide land outline, `world-land.json`, written by
`tools/prepare-basemap.py` from the pinned Natural Earth 1:10m land. It is detailed
within 150 km of any line, less so out to 400 km and coarse beyond, with lakes
cut out and no lines between countries, and every frame draws it. Data checks enforce its source
hashes, its point budget, known places on land and water, and every buildable
route on drawn land; pixel checks in CI verify coastal water across
palette/terrain settings.
### Construction prospect presentation

`ui.ConstructionOpportunities` supplies the current unstarted, non-public corridor
set. `render/prospects.go` uses one reviewed alignment per candidate for both
drawing and screen-distance picking; selecting an alternative changes that shared
geometry. `ui.SelectProspect` validates current availability and updates only the
local draft. Both hosts show the same named candidates, and map selection opens
Build → Plan. City inspection no longer rewrites construction endpoints. Accepted
projects leave the prospect set and keep authoritative project rendering. Existing
endpoint fields remain the save-compatible draft representation; no business
rules, catalogue coverage or persistence schema are changed by this interaction.
### Selecting services from track

`render.PickServiceSections` follows open sections' actual displayed alignments,
including a completed project's committed alternative. It retains nearby
overlapping hits for named disambiguation. `ui.SelectMapService` scopes the open
route selector to those sections and updates only local draft preferences.
`ServiceMapSections` is optional UI state; ordinary selection of an incompatible
route and New game clear it. The existing service command/forecast path remains
authoritative. Marker centers retain city/station inspection, and drags never
select track. Both hosts use `Camera.Resize` to preserve geographic center/scale
when the visible map rectangle changes as the service inspector opens.


Whole-network completion adds a separate `WithConnectivityPrior` catalog and
`connectivity_since` boundary. The exact v6 catalog is shipped as frozen gzip;
`previous_repair_sha256` pins its decompressed bytes. `World.atMonth` selects
that history first, then the older topology and original-catalog boundaries.
v6 restores validate their historical reports against that exact catalog before
upgrading in memory. Archived catalogs are parsed lazily, avoiding their geometry
allocation for fresh games. Both Python compiler gates and actual Go monthly
service tests cover complete operator-backed networks; source component checks
alone cannot establish a usable city/station chain. See the updated topology
validation report for the ten-network manifest and deliberate missing-link tests.


Worldwide construction adds 69 authored alignments to the two frozen Japanese
projects. `ConstructionInterval` assigns work quantities and regional baseline
rates to physical country scopes. `constructionEstimate` validates complete
quantity coverage and calculates the shared engineering/management/contingency
quote; legacy intervals retain the exact Japan formula. One explicitly declared
new dedicated HSR project may parallel inherited conventional rail between the
same cities; duplicate ordinary projects remain invalid and independent physical
tracks receive independent capacity. No cost calculation lives in either client.

Save v8 adds a `WithConstructionPrior` catalog and construction epoch, preserving
v7 historical reports while withdrawing conventional Moroccan starter assets.
Report validation and representative-day calculations select the historical
catalog; current frame animation filters retired services. All 71 prospects use
the same command, station-opening, service, demand, forecast and accounting ports
in native, WASM and headless clients. Source preparation and independent admission
checks run offline from pinned Natural Earth/cost/authoring inputs.


Save v9 adds `regional_since` and `regional_prior_sha256` through `regionalPayload`.
`WithRegionalPrior` retains the exact v8 catalog from `public-network-v4.json.gz`;
`World.atMonth` selects that catalog before applying earlier construction,
connectivity, topology and original-catalog boundaries. This preserves historical
demand/accounting when Taipei returns and Houston–Austin is added. Versions 2–8
remain readable; reads/failed commits do not write and the next accepted command
commits v9 atomically. New games reset all boundaries to zero. Reviewed Taiwan
station-platform aggregation is confined to named source way IDs at four HSR
stations; general railway junctions still require shared OSM node IDs.

The regional update also pins Ulsan to KTX/Tongdosa and aggregates its two mapped
platforms. Three Ulsan distances/timings change in current rules; v8 reports still
validate against the exact archived inputs.


Save v10 adds `connector_since` and `connector_prior_sha256` through
`connectorPayload`. `WithConnectorPrior` retains the exact v9 catalog in
`public-network-v5.json.gz`. Historical selection applies this boundary before
all earlier boundaries. Junction markets, repaired public resources and future
through-service alternatives apply prospectively; accepted service IDs, project
quotes/dates, fleets and resolved accounts are retained. v2–v9 saves remain
readable. Opening is read-only and the next accepted command commits atomically.
Named, bounded station-platform aggregations at Maibara, Kyoto, Nagoya,
Shin-Osaka, Barcelona, Camp de Tarragona and Figueres join reviewed HSR platforms
as strategic station resources. All intercity railway joins still require shared
source node IDs. The pinned approach supplement covers operational Turin/Milan
and Perpignan connections; these are existing public infrastructure, not private
construction projects.
