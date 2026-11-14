# Playable economy calibration

Verified locally on 2026-11-12 (America/New_York; ledger timestamps use UTC).
This is a gameplay balance pass, not empirical transport-demand certification.

## Changes

The expanded world has bounded, larger city trip budgets, stronger nearby and
domestic demand, and a waiting penalty that improves at every supported positive
frequency step. Lease cost is USD 120,000/train/month (formerly 400,000), running
USD 6.50/train-km (formerly 12), and public access USD 2.50/train-km (formerly 5).
Station handling, construction, line maintenance, finance and physical capacity
rules are unchanged. The exact inputs live in
[the policy asset](../../assets/scenario/economy.json); formulas and starting-plan
heuristic are in [the design](../DESIGN.md#playable-economy-calibration-implemented).

Unconfigured routes receive distance-based fares and demand-based frequencies.
Accepted plans stay intact. Through extension keeps the starter frequency and
previews explicitly accepted extra leases. Construction previews state the same
service assumptions. Both clients show monthly break-even in the primary plan;
Analysis separates all five cost components and explains the fixed-fare,
fixed-passenger-mix assumption. Through routes include intermediate short rides.
Unavailable targets use an em dash; invalid drafts clear stale forecasts.

## Reproducible comparison

`GOMEMLIMIT=768MiB GOMAXPROCS=2 go run -p 1 ./cmd/balancecheck -original-catalog -out docs/validation/economy-calibration.json`

This pre-topology-repair [comparison](economy-calibration.json) pins the frozen original public catalog and evaluates every then-existing direct
corridor independently, alongside the same starter service in January 2029.
It does not put all services into one company or optimize fares.

| Policy and plan | Contribution-positive corridors / 146 |
| --- | ---: |
| Legacy, 4 return trips/day at USD 90 | 14 |
| Rebalanced, same 4 / USD 90 | 85 |
| Rebalanced, suggested plans | 107 |

Lyon–Marseille starts at one return/day, USD 44.46: projected 8,825 boardings,
USD 392,359.50 revenue, USD 285,674.50 costs and USD 106,685 contribution.
The break-even target is 6,399 boardings/month. The suggested Lille–Lyon service
still loses money: the policy does not make every corridor profitable.

## Verification

Focused Go packages passed:
`GOMEMLIMIT=768MiB GOMAXPROCS=2 go test -p 1 ./internal/domain ./internal/application ./pkg/ui ./pkg/app -count=1`.
These cover finite demand conservation, seat limits, fare-mix/round-up targets,
suggestions, forecast/settlement accounting, unchanged accepted plans, native
widget sizing and strict/atomic v4-to-v5 compatibility. The frozen
[fixture provenance](../../internal/application/testdata/README.md) identifies
the actual pre-change executable and save bytes.

After building native/web/headless,
`NODE_OPTIONS=--max-old-space-size=1536 node tools/check-economy.cjs <server-url>`
passed. The [ledger](economy-checks.json) pins the tested WASM and policy hashes.
It exercises visible controls, actual leases/Apply/monthly settlement,
direct/through forecasts, exact native-headless/WASM checkpoint and report
parity, English/Chinese draft retention, invalid/suspended forecasts, Chinese
phone reflow, old-report preservation, v5 policy cutover and offline continuation.
All browser saves are isolated; the player's native campaign was not rewritten.

The public-network browser workflow also passed against the same WASM: actual
Chinese rail pixels, Beijing–Shanghai service settlement, finite demand/shared
capacity, native replay and offline continuation. Final focused UI/native checks
for economics, suggested plans, primary-plan fit and layered help passed after
the last presentation changes. JavaScript syntax, import boundaries, diff
whitespace, new document links and the native app's Info.plist checks passed.

- [Desktop plan](economy-plan-desktop.png)
- [Desktop Analysis](economy-analysis-desktop.png)
- [Chinese phone plan](economy-phone-zh.png)
- [Chinese phone Analysis](economy-analysis-phone-zh.png)

Native and browser targets compiled. Native widget tests use the actual ebitenui
layout; OS pointer/keyboard interaction and physical phones were not exercised.
The native app was rebuilt without replacing the player's running process.
The full root suite, race checks, long fuzzing and remote CI were not run locally.

The [known network topology gaps](NETWORK_TOPOLOGY.md) remain unchanged. Long-term
campaign finance, many-service competitive equilibrium and multiplayer balance
remain unqualified. Break-even assumes the current schedule, leases, passenger
mix and fare: it is neither a demand guarantee nor company-wide solvency. Owned
infrastructure upkeep attribution may overlap across service forecasts; company
incremental cash is shown separately.
