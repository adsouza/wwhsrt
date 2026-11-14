# Expanded-world gameplay policy

`economy.json` is the immutable `regional-balance-v1` gameplay calibration. All
currency fields are USD cents. It applies to the expanded playable network, not
the frozen original catalogs used to validate historical saves.

The policy scales finite city trip budgets and emphasizes nearer domestic
destinations. It reduces train leasing, running and public access charges so
short regional services can work, and removes the previous frequency plateau.
It does not change construction costs, geography, loans, grants, station handling
or owned-line maintenance. The single development train family still has global
operating rates; the name does not imply regional wage or energy cost modeling.

Coefficients are authored game assumptions, not researched real-world fares,
ridership or operating accounts. The formulas, starting-plan heuristic and
break-even semantics are specified in [DESIGN.md](../../docs/DESIGN.md#playable-economy-calibration-implemented).
Reproduce the comparison with `cmd/balancecheck`; see
[ECONOMY.md](../../docs/validation/ECONOMY.md) for evidence and limitations.

Development save v5 pins the exact bytes by SHA-256 and records the first month
using this policy. Do not edit an existing policy as a silent save upgrade:
introduce an explicit rules identity/compatibility path for future revisions.
