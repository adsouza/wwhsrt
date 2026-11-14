# Existing public connections in construction planning

Beijing–Tianjin, Chengdu–Chongqing and Chengdu–Xi'an are existing public corridors
in the imported catalog. The inspected running native binary also embeds those
classifications. They were already excluded from `ConstructionOpportunities`, and
the domain rejects attempts to construct them with `already_built`. The current
buildable catalog contains only Tsuruga–Osaka and Shin-Hakodate-Hokuto–Sapporo.

The Build endpoint selectors still allow inspecting any city pair. A terse
"Public track" message made the distinction from a construction proposal unclear.
Both clients now explicitly explain that the selected cities already have public
HSR, require no construction, and can support paid-access train services. A **Plan
trains on this existing line** action opens the appropriate Services draft. Public
pairs display no construction alignment, quote or commit action. The existing
construction list/domain guards remain in place; no public lines were removed.

Verification on 2026-11-12:

- Focused `TestExistingChineseConnections*` application checks verify all three
  public classifications, rejected duplicate construction, unchanged durable save
  bytes/checkpoint/projects, and availability for service forecasts.
- `TestNativeExistingPublicConnection*` verifies native widget layout at 320 and
  1280 px in English/Chinese, absence of construction commit actions, the exact
  service-draft destination, and unchanged company exports.
- Existing construction-filter and catalog/locale checks passed. Web/native builds,
  architecture boundaries, JS syntax, bundle metadata and `git diff --check` passed.
- [Browser ledger](existing-connection-checks.json) drives all three pairs in English
  desktop and Chinese phone layouts: explicit status, no quote/alternatives,
  disabled construction, exclusion from opportunities, working service shortcut,
  viewport bounds and unchanged exact save bytes/checkpoint.
- WASM SHA-256: `de0c6e183e018ef8131b05093eb94a84817c72c013fa5406a8a8fffff1ee2129`.

[English desktop](existing-connection-en.png) ·
[Chinese phone](existing-connection-zh-Hans.png).

No economic catalog, save format or company state changed. Restart the native game
to load the new explanation/action. Actual native OS interaction was not exercised;
its widgets were tested directly. Browser input was synthetic. No full root suite
or TypeScript compilation ran locally.
