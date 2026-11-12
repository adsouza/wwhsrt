# World-wide High-speed Rail Tycoon

A strategic railway business game about operating and expanding the world's high-speed rail networks. Plan monthly business turns, watch an animated operating day, and compete for passengers through fares, frequency, speed, and useful connections.

![World-wide High-speed Rail Tycoon social card with a streamlined train in front of a globe](assets/branding/social-preview.png)

Build independent lines between unserved cities, expand existing railways, or join separate networks. A new line can operate without connecting to the legacy network.

Real geography shapes expansion: compare longer valley routes with expensive mountain works, tunnels, and bridges, balancing construction time and upkeep against journey speed.

Single-player runs offline. Optional multiplayer connects competing companies to a private dedicated server. Players prepare in a planning lobby before the host starts the clock. The proposed default is 30 seconds per business month for a three-hour campaign, with a fixed interval chosen at world creation. Existing public lines can be acquired through reserve-price auctions while retaining regulated open access.

Committed construction is visible to every company. Owners can offer paid track access on new private lines. Online auctions give players at least three real minutes to bid. Every station is open access at regulated fees.

The first playable business slice now runs in East Asia: operate the Tokyo–Nagoya starter service, adjust fares and frequency, lease trains, borrow for construction, compare terrain-aware alignments, build a new line, and collect passenger revenue in monthly reports. Native controls use ebitenui; the browser has semantic HTML controls over the same Go/Ebitengine map and rules. English and Simplified Chinese are bundled.

To start over, open **Company → New game** and confirm replacing the current save. This restores the January 2029 starting company and starter service while keeping display preferences.

Choose **Company → Appearance** for System, Light or Dark mode. System follows your device; the saved choice covers controls, the map and terrain, including offline play.

Run the native client from this checkout:

```sh
go run ./cmd/wwhsrt
```

Or build and serve the browser version:

```sh
sh tools/build-web.sh
go run ./cmd/webserve
```

Open `http://127.0.0.1:8765/`. Once its offline cache has installed, the browser client can reload and play without networking. Native play has no network requirement. `sh tools/build-native.sh` also produces a native executable and a macOS development app bundle on macOS.

The [first-business notes](docs/validation/FIRST_BUSINESS.md) explain controls, funding, model assumptions, save locations and remaining qualification. [TESTING.md](TESTING.md#first-business-slice) lists focused checks. The graphics-free CLI uses the same application: `go run ./cmd/headless -advance 2`.

[Milestone 2](docs/validation/MILESTONE_2.md) adds explicit station readiness, the Swift 400 train family and costs, reconciled engineering breakdowns, and contextual guidance through the first-business loop. Existing version-2 saves and economic balances are preserved. Through-running and transfers belong to milestone 3.

The [game design specification](docs/DESIGN.md) records the current design, confirmed choices, proposed rules, and delivery milestones. The broader worldwide game and multiplayer server remain planned.

The [architecture](ARCHITECTURE.md) describes system boundaries, offline/server execution, data preparation, presentation, and persistence. The [testing guide](TESTING.md) separates current runnable prototype checks from planned implementation and release gates.

The [state and storage contract](docs/STATE_AND_STORAGE.md) defines concrete Go reference types, JSON save v1 and schema, SQLite server transactions, recovery, and migrations only when a released format actually changes.

The [interaction and UX specification](docs/UX.md) defines map-centered decisions, contextual learning during ordinary play, and complete phone workflows that expand into simultaneous map, planning, and comparison views on larger displays.

The [presentation specification](docs/PRESENTATION.md) defines the bright palette and construction/train motion and sound. The [diagnostics contract](docs/DIAGNOSTICS.md) defines pervasive event logging, durable audit history and support bundles.

The [world-data notes](data/README.md) describe the supplied 96-metro seed, its population and GDP estimates, and the track and station audits for the launch map. The [launch corridor shortlist](docs/LAUNCH_CORRIDORS.md) records the 70 proposed links for alignment authoring and qualification.

The [construction-cost research](docs/CONSTRUCTION_COSTS.md) supports 20 regional cost profiles, with HSR benchmarks, conventional-rail extrapolations, and explicit price/scope assumptions.

The [map and widget validation report](docs/validation/README.md) contains Equal Earth world/regional views of the playable metros derived from the 96-record seed (94 after Taipei's restoration) and screenshots/checks from an actual Go/WebAssembly ebitenui renderer. The [isolated rendering spike](tools/ebitenui-spike/README.md) runs independently of the future game.
