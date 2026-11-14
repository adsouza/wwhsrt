# Service selection from map track

Verified on 2026-11-12 (browser ledger timestamps are UTC, 2026-11-13).

Clicking an operating railway opens Services → Plan. The named route selector
shows open direct and through-running patterns using the clicked section(s),
with an explicit scope label and Show all service routes control. Reclicking a
matching current pattern preserves pending fare/frequency and extension choices.
Choosing an unrelated route clears an incompatible map scope. Selecting a through
pattern highlights its complete route, rather than just the clicked section.

Picking follows the actual rendered alignment, including a completed project's
chosen alternative. The hit target is 18 screen pixels; overlapping sections
within four pixels of the nearest hit remain available for named disambiguation.
Future construction cannot be selected for operation. Drags preserve selection;
city/station marker centers and representative train markers retain inspection.
The phone retains a compact map when service controls open and after an explicit
service action. Geographic camera center/scale survive changes to the visible
map rectangle. Hidden/initial one-pixel surfaces do not supply a fit scale.

The optional section filter is local UI state, persisted with preferences and
cleared on New game. Campaign schema, economics, service commands and source data
are unchanged. No company reset or migration is required. Both English and
Simplified Chinese help describe track selection and scope clearing.

Verification:

- Focused `go test -p 1 ./pkg/ui ./pkg/mapview` and
  `go test -p 1 ./pkg/app ./pkg/render -count=1` passed with
  `GOMEMLIMIT=768MiB GOMAXPROCS=2`. New checks cover curved geometry versus
  endpoint chords, closed/open project boundaries, committed alternatives,
  overlaps, named direct/through choices, stale clicks, retained drafts, filter
  clearing, New game and geographic-center preservation on phone/desktop resize.
- Native widget/input tests cover map clicks versus drags, scoped service
  controls and clearing, English/Chinese phone/desktop widths and unchanged
  campaign exports. The ordinary primary service form still fits the phone
  viewport. The native map-service check passed again after retaining map context
  across a service command result when no idle-line warning exists.
- [Browser ledger](map-services-checks.json): real Japanese and Chinese track
  clicks enter scoped Services. Tokyo–Nagoya edits survive reclicks both from Map
  and directly within Services; selecting Tokyo–Nagoya–Osaka retains its extension
  behavior and produces 44 highlighted-pixel samples beyond the clicked section.
  The Chinese phone keeps the selected Beijing–Tianjin track visible (15 pixel
  samples) after its inspector opens. Scope clearing and reload work. All these
  inspections preserve exact durable campaign bytes/checkpoint. A subsequent
  explicitly clicked lease, in the isolated test company, retains map context.
- The existing construction-prospect browser regression also passed on the same
  final WASM: both dotted paths remain selectable and starting a project removes
  its prospect from the list/map, including after reload.
- Web/native builds, architecture boundaries, native bundle metadata, JS syntax
  and `git diff --check` passed.
- Final WASM SHA-256:
  `af9aa14ba1c4cb4b5f133985f08f15e782f419b50b91928c5a67e0f7c240997e`.
- Final native executable SHA-256:
  `69ec43831dcf9cab5ca3a317711d1b47aa16e67a73da19210da815fb0c1e4678`.

Screenshots: [Through service](map-services-through.png),
[Chinese phone](map-services-phone-zh.png).

Browser inputs are synthetic; native widgets were exercised directly without an
OS mouse-interaction pass. Physical phones remain unqualified. No full root suite
or TypeScript compilation ran. Browser campaigns are isolated from the player's
save. Restart the native app to load the rebuilt interaction.
