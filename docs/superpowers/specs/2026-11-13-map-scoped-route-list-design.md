# Map-scoped route list (native Services panel)

Status: approved 2026-11-13 and implemented. Section 8 and the hint wording were
revised during implementation and review; those revisions await confirmation.
Size: a small change.

## Intent

The native Services panel lists every open route in one dropdown: 3,218 entries
on the current catalog. The widget library builds one button per entry, so each
rebuild of the panel takes about 71 ms (other panels take about 1 ms). That is
four dropped frames on every route-search keystroke or fare change, and it is
why the `pkg/app` tests take about 42 s.

A 3,218-entry list is also not usable by scrolling. The map should be the way
to narrow it: the list offers the routes you are looking at.

## Goals

- The dropdown never holds more than 25 routes, plus the selected route.
- Zooming and panning the map changes which routes are offered.
- Every route stays reachable: by framing it on the map, by route search, or by
  tapping a stretch of track.
- The `pkg/app` suite drops to roughly 11 s (measured with a trial cap).

## Non-goals

- The web client. Its `<select>` keeps listing every route; a browser handles
  that cheaply, and 14 of the 26 browser check scripts pick routes from it by ID.
- Changing the zoom limit. It stays at 32x; measurement showed more zoom does
  not make more routes reachable.
- Guaranteeing that zoom alone reaches every route. Up to 482 routes share one
  stretch of track, so no zoom level separates them.

## Anti-goals

- A selected route disappearing from its own dropdown because the map moved.
- The list changing under the pointer while the player is dragging the map.
- Any change to rules, saves, forecasts or the frame contract. This is
  presentation only.

## Behaviour (decided)

1. **Visible** means the whole route fits inside the map view. A route that is
   only partly in view is not listed.
2. **Ranking** is by span: the diagonal of the route's bounding box in map
   coordinates, largest first. Ties break by route ID so the order is stable.
3. **Cap** is 25. The selected route is always listed in addition, even when it
   is outside the view or beyond the cap.
4. **Route search** looks across all open routes, not only those in view. The
   matches are ranked and capped the same way.
5. **Track tap** (the existing "Routes using clicked track" filter) still
   narrows to routes through that section, regardless of the view, ranked and
   capped the same way.
6. **Hint line** under the dropdown, in English and Simplified Chinese:
   - more matches than shown: how many are shown of how many, and that zooming
     in or searching narrows the list;
   - nothing fits the view: zoom out or search.
7. **Refresh** happens when a pan, pinch or zoom settles, not on every frame of
   the gesture. Region changes and the fit, focus and zoom buttons refresh
   immediately.

8. **Map not shown** (decided: filter by region). On compact layouts the
   Services panel can be open with no map on screen. The list then behaves as
   if the map were showing the whole chosen region: a route is listed when it
   fits entirely inside what the map would show for that region, ranked by
   span and capped at 25.
   - It is the same region setting as the Map panel's picker, so choosing a
     region here also moves the map for when it is shown again.
   - Changed during implementation and review: the region's frame is what the
     Map panel would show for that region in the current window, not the
     bounding box of the region's defining points. Japan and Korea were
     defined by two corner points, and the projection narrows that box until
     it excluded Tokyo; a tall stand-in shape likewise clipped Seoul and Busan.
     On a 390x844 phone the regions hold: China 2,407 routes, Africa 266,
     Japan 160, North America East 92, Korea 18.
   - Changed 2026-11-14: Tokyo and Fukuoka were added as defining points for
     Japan, so the points' bounding box now holds all 14 Japanese cities. The
     region's view moved east, which dropped 64 routes in north-east China
     and the Yangtze delta from Japan's phone list (224 to 160) and no
     Japanese route. The frame stays the Map panel's view: Korea and China
     are still defined by two corners.
   - Changed during implementation: the ordinary phone plan must fit one
     screen (an existing requirement), so the Region dropdown and the
     "showing N of M" advice sit inside the route-search disclosure, titled
     "Narrow the route list" in this mode. The count itself rides on the Route
     label ("Route · showing 25 of 75"). An empty list always explains itself.
   - The dropdown offers only regions that have at least one open route
     fitting them, plus the current region.
   - A capped search or track tap gets its own wording ("add a word to the
     search"), since neither zooming nor changing region narrows it.
   - Added in review: while a search query is active its disclosure stays
     open, so a hidden query cannot silently switch map scoping off; a search
     that matches nothing says so; and the list does not rebuild while a
     dropdown is open.

## Approaches considered

| Approach | Trade-off | Outcome |
|---|---|---|
| Shared rule in `pkg/ui`, used by the native panel only | Fixes the slow panel and tests; the two clients differ in how long the list is | **Chosen** |
| Shared rule, both clients | Same behaviour everywhere, but 14 browser check scripts need rework and their evidence regenerated | Not now |
| Rule written inside the native panel code | Least code, but only testable through widgets | Rejected |

Earlier alternatives, also set aside: a plain cap at the first 100 matches
(does not use the map), and filling the dropdown lazily when opened (no visible
change, but reworks keyboard selection and dropdown event handling).

## Constraints

- `pkg/ui` holds presentation logic and does not import `pkg/mapview` today.
  The rule should take route bounding boxes and the view rectangle as plain
  numbers so that stays true; projecting geometry belongs to the caller.
- Route geometry is static per frame revision. Bounding boxes should be
  computed once per revision, not on every rebuild (47,569 path points).
- A built line uses its chosen alignment's path, as `focusSelected` already does.
- Existing native tests must keep passing without weakening their assertions.

## Left to the builder

- Names and signatures of the new function and its box type.
- Where the per-revision box cache lives on the native host.
- How "settled" is detected for wheel zoom (for example, a short quiet period).
- Exact wording of the two hint strings, following the tone of nearby strings.
- Unit tests for the rule: fully inside versus partly inside, span ordering,
  the cap, the always-listed selection, search and track-tap precedence, and the
  empty view.
- A native test that the Services dropdown holds at most 26 entries and that
  moving the camera changes it.

## Evidence

Measured on the shipped catalog, 3,218 open routes:

- Rule "fits entirely in view", 25 largest by span: 2,638 routes (82%) can be
  brought into the list by framing them. By track length instead: 77%.
- Rule "any part in view": 51%, even with unlimited zoom.
- Trial cap of 100 entries: `pkg/app` 41.9 s to 11.2 s, no test failures.
