# Regional shoreline replacement

Verified on 2026-11-12 (the browser ledger timestamp is UTC, 2026-11-13).

The global basemap previously retained coarse Japan polygons underneath the
regional detailed layer. China retained the coarse prototype geometry. The import
now replaces all 17 original regional rings with the refined regional dataset,
and the renderer draws either the composed global layer or the legacy regional
fixture layer. It no longer draws both. Shorelines remain independent of the
terrain visibility toggle.

The pinned Natural Earth 1:10m source supplies Japan (109 rings, 4,680 vertices),
China including Hong Kong and Macao (75 rings, 9,414 vertices), and Taiwan
(8 rings, 333 vertices), simplified at 0.003 degrees. This is display geometry,
not a surveyed construction alignment. The composed world layer has 474 rings.

Validation:

- Focused `go test -p 1 ./pkg/render -count=1` passed, including a regression
  requiring every replacement ring exactly once, no duplicate world rings, and
  no retained coarse regional rings.
- Shoreline and public-network import `--check` commands reproduced their
  outputs. All 12 current route geography checks passed, including negative
  fixtures. Architecture boundaries, JavaScript syntax, bundle metadata and
  `git diff --check` passed.
- Web and native builds passed. The [browser ledger](coastline-checks.json)
  records eight passing Japan/China × light/dark × terrain on/off cases. Points
  formerly covered by coarse coastal land now show water; detailed shoreline
  boundaries remain visible. Exact durable campaign bytes and checkpoint were
  unchanged. Browser test data was isolated from the player's campaign.
- Screenshots: [China](coastline-china.png), [Japan](coastline-japan.png).
- WASM SHA-256:
  `d59f34467de84a51d7f063f7916fc8707ce696cec6842335ccfb59b2092a879c`.
- World-land SHA-256:
  `bd178019874d9e692f2e1ba3e1ca0617d825a4e7e31615feb317554b446e74c4`.
- Economic catalog SHA-256 remains
  `2a8d0522b5a678579d17d5f257d0f6c1fc1c4f4d79879414d720ee6249486e59`;
  no save migration is required.

Native OS graphics/input and physical devices were not exercised in this pass.
No full root suite or TypeScript compilation ran locally. The owned test server
was stopped; the running native game was left untouched and needs a restart to
load the rebuilt map.
