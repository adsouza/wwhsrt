# Current route water checks

The first-business scenario has six alignments, all in Japan. Five formerly cut
across sea or Lake Biwa: both Hokkaido options, both Tsuruga–Osaka options and the
existing Nagoya–Osaka line. Tokyo–Nagoya was already clear. The current authored
paths add intermediate geometry points; these are not extra stations or projects.

Hokkaido's tunnel route follows an inland corridor through the Oshima uplands,
Yakumo/Oshamambe, the Kutchan area and the inland Otaru approach, informed by
[JRTT's extension overview](https://www.jrtt.go.jp/english/project/hokkaido.html).
The longer alternative follows the western and northern shore of Uchiura Bay,
then the Muroran/Shiraoi–Tomakomai–Chitose approach, inspired by the
[JR Hokkaido network](https://www.jrhokkaido.co.jp/global/pdf/e-accessibility-map.pdf).
Nagoya–Osaka and the Tsuruga valley alternative go around Lake Biwa's eastern and
southern shores via the Maibara area; the Tsuruga tunnel option stays inland west
of the lake. These are regional game drawings, not surveyed rail alignments.
The existing line's Maibara–Kyoto sequence is documented in
[JR Central's fact sheets](https://global.jr-central.co.jp/en/company/ir/factsheets/_pdf/factsheets2025.pdf).

`japan-route-mask.v1.json` contains clipped Natural Earth 1:10m Japanese land
geometry, the Natural Earth Lake Biwa polygon already used by the map, and
conservative authored avoidance envelopes for Toya, Shikotsu and Onuma. The
envelopes are explicitly **not lake shorelines** and are not drawn by the game.
Metadata pins the source and derived bytes. Natural Earth is public domain.

To reproduce the mask, first obtain the pinned source cache using
`tools/prepare-terrain.py`, then run `python3 tools/prepare-route-geography.py`.
For ordinary verification, install `tools/requirements-geography.txt` and run
`python3 tools/check-route-geography.py`; no downloads are needed. CI runs this
check. The game itself has no Shapely or Python dependency.

The checker audits every current path, including both alternatives and existing
track, in geographic coordinates and in **Equal Earth**, where the renderer joins
projected waypoints with straight segments. It checks both the detailed source
shoreline and the coastline actually drawn, endpoint attachment, mask coverage and
that priced rail kilometres are at least the coarse drawing's geodesic length.
Negative fixtures reject the old Uchiura Bay/Lake Biwa shortcuts and absent coverage.
Results are in `docs/validation/route-geography-checks.json`.

There are no approved sea/lake crossing exceptions in this scenario. Add explicit
reviewed engineering and scoped crossing geometry before admitting one; do not
turn off water checks for an entire route. Small rivers and other unmapped waters
are outside this regional mask. Passing it does not prove gradients, curvature,
geology, clearances or legal buildability. Full production corridor engineering
remains separate. Authored rail lengths, quantities, prices and durations remain
balance assumptions and were not recalculated from artwork.

This data revision changes the business catalog hash. During the authorized testing
reset, the native company was replaced with a fresh company. No save migration was
introduced; earlier development saves with the old hash still fail validation.
Browser qualification uses fresh storage; an older development browser profile
needs its `wwhsrt-first-business` IndexedDB data cleared before starting this catalog.
