# Map-Scoped Route List Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** The native Services panel's route dropdown offers at most 25 routes (plus the selected one), chosen by what fits in the map view and ranked by span.

**Architecture:** A pure rule in `pkg/ui` picks and ranks routes from plain bounding boxes and an optional view rectangle. The native host projects route geometry into boxes once per frame revision, passes the camera's view rectangle, and marks the panel for rebuild when the view has settled.

**Tech Stack:** Go 1.26+, Ebitengine, ebitenui (existing). No new dependencies.

**Spec:** `docs/superpowers/specs/2026-11-13-map-scoped-route-list-design.md`

## Global Constraints

- Native client only. Do not change `web/`, `pkg/app/host_js.go`, or any `tools/*.cjs`.
- `pkg/ui` must not import `pkg/mapview`. `python3 tools/check-boundaries.py` must pass.
- Presentation only: no change to `internal/`, `pkg/gameapi`, saves or forecasts.
- Cap is exactly 25 (`ui.RouteChoiceLimit`); the selected route is listed in addition.
- Zoom limit stays 32x (`maxZoom` in `pkg/mapview/camera.go` is not touched).
- New strings go in both `pkg/ui/en.json` and `pkg/ui/zh-Hans.json`; the locale key parity test in `pkg/ui/state_test.go` must pass.
- Existing tests keep their assertions. Run with `GOMEMLIMIT=768MiB GOMAXPROCS=2 go test -p 1 ... -count=1`.
- Lines stay within 120 columns (`gofmt` clean).

## Review Focus

1. **Map not on screen** (compact layout, Services open without the map): the list follows the chosen region, a Region dropdown is offered, and a region with no fitting route explains itself. Tests in Tasks 1 and 3.
2. **Nothing fits the view** (zoomed far in): the selected route stays listed and the "zoom out or search" hint shows. Tests in Tasks 1 and 3.
3. **Route with no geometry** (missing box): never panics; left out of the in-view list, still found by search. Test in Task 1.
4. **Stale selection** (`ServiceCorridor` names a route that is closed or gone, for example after a new game): not appended, no panic. Test in Task 1.
5. **A line opens** (frame revision changes): boxes are rebuilt so the new route can appear. Test in Task 2.

---

### Task 1: The selection rule

**Files:**
- Create: `pkg/ui/route_choices.go`
- Test: `pkg/ui/route_choices_test.go`
- Modify: `pkg/ui/en.json`, `pkg/ui/zh-Hans.json`

**Interfaces:**
- Consumes: `routeUsesSections` (`pkg/ui/map_services.go`), `RouteName` (`pkg/ui/network.go`), `Text`.
- Produces:

```go
const RouteChoiceLimit = 25

// Box is an axis-aligned rectangle in the map's projected coordinates.
type Box struct{ MinX, MinY, MaxX, MaxY float64 }

func (b Box) Contains(o Box) bool // o lies entirely inside b, edges inclusive
func (b Box) Span() float64       // length of the diagonal

type RouteChoices struct {
	Routes   []gameapi.Route // what the dropdown lists, ranked; selected route last if it was added
	Matching int             // routes that passed the filter, before the cap
	InView   bool            // the view rectangle was the filter that applied
}

func ServiceRouteChoices(f gameapi.Frame, s State, boxes map[string]Box, view *Box) RouteChoices
func RouteChoiceHint(locale string, c RouteChoices, mapShown bool) string
```

Rule for `ServiceRouteChoices`, in this order:
1. Candidates are routes with `Open == true`.
2. If `s.ServiceMapSections` is non-empty, keep routes using any of those sections.
3. If `s.ServiceQuery` has words (`strings.Fields(strings.ToLower(...))`), keep routes whose lower-cased `RouteName(r, s.Locale)` contains every word.
4. Only if neither 2 nor 3 applied and `view != nil`: keep routes that have a box and whose box `view.Contains`; set `InView`.
5. `Matching` is the count now. Sort by `boxes[r.ID].Span()` descending (no box counts as 0), ties by `ID` ascending. Keep the first 25.
6. If `s.ServiceCorridor` is an open route and not already kept, append it.

`RouteChoiceHint`: when `Matching > 25`, `fmt.Sprintf(Text(locale, key), 25, Matching)` with key `routeChoicesMore`; when `InView && Matching == 0`, `Text(locale, "routeChoicesNone")`; otherwise `""`. When `mapShown` is false and `InView` is true, use the `...Region` keys instead.

Strings (exact):

| Key | en | zh-Hans |
|---|---|---|
| `routeChoicesMore` | `Showing %d of %d routes. Zoom in or search to narrow the list.` | `显示 %d / %d 条线路。放大地图或搜索可缩小范围。` |
| `routeChoicesNone` | `No route fits in this map view. Zoom out or search by name.` | `当前地图视野内没有完整线路。请缩小地图或按名称搜索。` |
| `routeChoicesMoreRegion` | `Showing %d of %d routes. Choose a smaller region or search to narrow the list.` | `显示 %d / %d 条线路。选择更小的区域或搜索可缩小范围。` |
| `routeChoicesNoneRegion` | `No route fits in this region. Choose another region or search by name.` | `该区域内没有完整线路。请选择其他区域或按名称搜索。` |

- [ ] **Step 1: Write the failing tests** in `pkg/ui/route_choices_test.go`. Build a frame with `gameapi.Route` values only (`ID`, `Open`, `Sections`, `Cities`) and hand-made boxes.

```go
func TestRouteChoicesKeepOnlyRoutesFullyInsideView(t *testing.T)
// inside {0,0,10,10}: box {1,1,5,5} kept; box {8,8,12,12} (partly out) dropped; InView true; Matching 1

func TestRouteChoicesRankBySpanThenIDAndCapAt25(t *testing.T)
// 40 open routes, spans 1..40, two with equal span: Routes has 25 entries,
// first is the largest span, equal spans ordered by ID, Matching == 40

func TestRouteChoicesAlwaysListOpenSelection(t *testing.T)
// selected route outside view: len(Routes) == kept+1 and last ID is the selection;
// selected route closed or absent from f.Routes: not appended, no panic

func TestRouteChoicesSearchAndTrackTapIgnoreView(t *testing.T)
// ServiceQuery matching a route outside the view returns it with InView false;
// ServiceMapSections likewise; a route with no box is returned by search with span 0

func TestRouteChoicesWithoutViewRankAllOpenRoutes(t *testing.T)
// view == nil: closed routes excluded, InView false, capped at 25

func TestRouteChoiceHint(t *testing.T)
// Matching 40 -> en "Showing 25 of 40 routes. Zoom in or search to narrow the list."
// InView && Matching 0 -> en "No route fits in this map view. Zoom out or search by name."
// Matching 3 -> ""; mapShown false with InView true uses the two ...Region strings;
// all four keys are non-empty and distinct in zh-Hans
```

- [ ] **Step 2: Run to verify failure**

Run: `go test ./pkg/ui -run 'TestRouteChoice' -count=1`
Expected: FAIL to compile, `undefined: ServiceRouteChoices`.

- [ ] **Step 3: Implement** the types and two functions in `pkg/ui/route_choices.go` and add the four keys to both locale files. Use `sort.SliceStable` after computing spans once.

- [ ] **Step 4: Run to verify pass**

Run: `go test ./pkg/ui -count=1 && python3 tools/check-boundaries.py`
Expected: `ok  wwhsrt/pkg/ui` and the boundary check's "passed" line.

- [ ] **Step 5: Commit** `git commit -m "Add map-scoped route choice rule"` with the four files.

---

### Task 2: Route boxes and the view rectangle

**Files:**
- Create: `pkg/app/route_choices_native.go` (`//go:build !js`)
- Test: `pkg/app/route_choices_native_test.go` (`//go:build !js`)
- Modify: `pkg/app/host_native.go` (three fields on `host`)

**Interfaces:**
- Consumes: `ui.Box` (Task 1), `mapview.Project`, `mapview.Camera.World`.
- Produces:

```go
func routeBoxes(f gameapi.Frame) map[string]ui.Box // every route in f.Routes whose sections all have geometry
func viewBox(c mapview.Camera) ui.Box              // the camera's visible rectangle in projected coordinates
func regionFrame(c mapview.Camera) ui.Box          // c.Min..c.Max grown by 10% of each dimension on every side
func (h *host) boxesFor(f gameapi.Frame) map[string]ui.Box
```

`host` gains `routeBoxCache map[string]ui.Box`, `routeBoxRevision uint64`, `routeBoxReady bool`.

- A corridor's box covers `mapview.Project(lat, lon)` of every point on one alignment: the alignment named by its entry in `f.Projects` if there is one, otherwise `Alignments[0]` (the same choice `focusSelected` in `pkg/app/game.go` makes).
- A route's box is the union of its sections' corridor boxes.
- `viewBox` uses `c.World` at `(0,0)` and `(c.Width,c.Height)` and normalises min/max (screen Y grows downward).
- `regionFrame` is independent of window size: a region camera's `Min` and `Max` are the bounds of the points that define the region.
- `boxesFor` recomputes when `!routeBoxReady` or `f.Revision != routeBoxRevision`.

- [ ] **Step 1: Write the failing tests** using `application.Open(nil, nil)` and `p.Snapshot()` as the other native tests do.

```go
func TestRouteBoxesCoverEveryOpenRouteAndItsSections(t *testing.T)
// every open route in the frame has a box; for a multi-section route the box
// Contains each section's box; Span() > 0

func TestViewBoxMatchesCamera(t *testing.T)
// c := mapview.New(1000, 600): viewBox(c) Contains the projected point of
// c.World at the screen centre; after c.ZoomAt(4, centre) the new box has a smaller Span()
// and is contained by the old one

func TestRegionFrameIgnoresWindowSize(t *testing.T)
// regionFrame(mapview.RegionCamera(390, 260, "japan", true)) equals the frame from
// RegionCamera(1920, 1080, "japan", true); it Contains the box of c.Min..c.Max and is larger

func TestBoxesForRecomputesOnNewRevision(t *testing.T)
// same revision returns the cached map (mutate an entry, call again, mutation still there);
// after f.Revision++ with a route removed from f.Routes, that ID is gone
```

- [ ] **Step 2: Run to verify failure**

Run: `go test ./pkg/app -run 'TestRouteBoxes|TestViewBox|TestRegionFrame|TestBoxesFor' -count=1`
Expected: FAIL to compile, `undefined: routeBoxes`.

- [ ] **Step 3: Implement** the three functions and add the fields.

- [ ] **Step 4: Run to verify pass**

Run: `go test ./pkg/app -run 'TestRouteBoxes|TestViewBox|TestRegionFrame|TestBoxesFor' -count=1`
Expected: `ok  wwhsrt/pkg/app`.

- [ ] **Step 5: Commit** `git commit -m "Compute route boxes and the map view rectangle"`.

---

### Task 3: Use the rule in the Services panel

**Files:**
- Modify: `pkg/app/planner_native.go` (`buildServices`, the block that fills `options` and calls `h.selectChoice`)
- Modify: `pkg/app/host_native.go` (fields; the Map panel's region picker moves into a shared method)
- Test: `pkg/app/route_choices_native_test.go`

**Interfaces:**
- Consumes: `ui.ServiceRouteChoices`, `ui.RouteChoiceHint` (Task 1); `viewBox`, `regionFrame`, `h.boxesFor` (Task 2); `g.regionCamera`, `mapview.VisibleRegions`.
- Produces: `host.routeChoices ui.RouteChoices` (what the last build listed) and `host.routeView ui.Box`, `host.routeViewUsed bool` (the view that build used). Task 4 reads these.

Behaviour:
- The map is "shown" when `!h.workspace.Map.Empty()`. `view` is `viewBox(g.Camera)` when shown, otherwise `regionFrame(g.regionCamera(g.State.Portrait))`. It is never `nil` in the panel.
- Extract the Map panel's region dropdown (`host_native.go`, the `label(t("region"), 13)` block and its `selectChoice`) into `func (h *host) regionPicker(g *Game, label nativeLabel, offer func(id string) bool)`. The Map panel passes an `offer` that accepts every region, so its list is unchanged.
- When the map is not shown, Services calls `regionPicker` above the route dropdown with an `offer` that accepts the current region and any region for which at least one open route's box fits `regionFrame` of that region's camera.
- Pass `mapShown` to `ui.RouteChoiceHint`.
- `options` come from `choices.Routes`, in that order, as `choice{r.ID, ui.RouteName(r, g.State.Locale)}`.
- Directly after the dropdown, `label(hint, 13)` when `ui.RouteChoiceHint` is non-empty.
- Remove the old loop over `ui.MapServiceRoutes` with its inline query matching. `ui.MapServiceRoutes` itself stays; `SelectMapService` still uses it.

- [ ] **Step 1: Write the failing tests** (Services view, `Locale: "en"`, built with `g.host.layout(g); g.host.build(g)`).

```go
func TestNativeServiceRoutesFollowTheMapView(t *testing.T)
// 1280x850, default camera, ServiceCorridor "tokyo-nagoya":
//   len(g.host.routeChoices.Routes) <= ui.RouteChoiceLimit+1, InView true, selection listed
// then g.Camera.ZoomAt(32, centre) and rebuild:
//   routeChoices differs from before; selection still listed
// then State.ServiceQuery = "shanghai" and rebuild:
//   InView false and every listed route (other than the selection) has "shanghai" in its English name

func TestNativeServiceRoutesFollowTheRegionWithoutAMap(t *testing.T)
// 390x844, View "services", ReviewMap false (map rectangle empty), Region "japan", Portrait true:
//   two dropdowns are built (region, then route); InView true; 1 < len(Routes) <= ui.RouteChoiceLimit+1;
//   every listed route other than the selection has a box inside regionFrame of the japan camera
// Region "china": the listed routes differ from japan's
// Region "south-asia" (no open route fits): Routes is exactly the selection and the hint is the
//   routeChoicesNoneRegion text; the region dropdown still lists south-asia because it is current
// At 1280x850 (map shown) Services builds one dropdown only

func TestNativeServiceRoutesExplainAnEmptyView(t *testing.T)
// 1280x850, camera zoomed to 32x over open sea (PanX/PanY pushed to the clamp):
//   Matching == 0, Routes is exactly the selection, and
//   ui.RouteChoiceHint("en", g.host.routeChoices) is the routeChoicesNone text
```

- [ ] **Step 2: Run to verify failure**

Run: `go test ./pkg/app -run 'TestNativeServiceRoutes' -count=1`
Expected: FAIL to compile, `g.host.routeChoices undefined`.

- [ ] **Step 3: Implement** the change in `buildServices` and add the fields.

- [ ] **Step 4: Run the whole package**

Run: `GOMEMLIMIT=768MiB GOMAXPROCS=2 go test -p 1 ./pkg/app -count=1`
Expected: `ok  wwhsrt/pkg/app` in roughly 10 to 15 s (was about 42 s). If an existing test fails because it expected a specific route in the dropdown, stop and report it rather than editing that test.

- [ ] **Step 5: Commit** `git commit -m "List only map-scoped routes in the native Services panel"`.

---

### Task 4: Refresh when the view settles

**Files:**
- Modify: `pkg/app/route_choices_native.go`, `pkg/app/host_native.go` (`update`), `pkg/app/workspace_native.go` (map tool buttons)
- Test: `pkg/app/route_choices_native_test.go`

**Interfaces:**
- Consumes: `host.routeView`, `host.routeViewUsed` (Task 3), `viewBox` (Task 2).
- Produces:

```go
const routeViewSettleTicks = 12 // 0.2 s at 60 updates per second

// routeListStale reports true once when the Services list was built for a
// different view and the camera has then held still for routeViewSettleTicks calls.
func (h *host) routeListStale(g *Game) bool
```

`host` gains `routePendingView ui.Box` and `routeStableTicks int`.

- Returns false, and resets the counter, unless `g.State.View == "services"`, help is closed, and the map rectangle is non-empty. With the map hidden the list depends only on the region, and the region dropdown already rebuilds the panel.
- Returns false while `g.drag`, `g.touching` or `g.pinch > 0`.
- If the current `viewBox` equals `routeView` (and `routeViewUsed`): false, counter reset.
- If it differs from `routePendingView`: store it, counter to 0, false. Otherwise count up and return true exactly when the counter reaches `routeViewSettleTicks`.
- `host.update` calls it once per tick before its `if g.dirty` rebuild and sets `g.dirty = true` on true.
- The map tool buttons `fit`, `focusRoute`, `zoomIn`, `zoomOut` in `addMapTools` set `g.dirty = true` so they refresh at once.

- [ ] **Step 1: Write the failing test**

```go
func TestNativeServiceRouteListRefreshesAfterTheViewSettles(t *testing.T)
// build Services at 1280x850; routeListStale is false on repeated calls
// g.Camera.ZoomAt(2, centre): 11 calls false, the 12th true, the 13th false
// move the camera again halfway through a count: the count restarts
// with g.drag = true: stays false however many calls
// after g.host.build(g): false again
// with State.View = "map": false
```

- [ ] **Step 2: Run to verify failure**

Run: `go test ./pkg/app -run 'TestNativeServiceRouteListRefreshes' -count=1`
Expected: FAIL to compile, `routeListStale undefined`.

- [ ] **Step 3: Implement** `routeListStale`, the call in `update`, and the four `g.dirty = true` lines.

- [ ] **Step 4: Run the full suite and static checks**

Run: `GOMEMLIMIT=768MiB GOMAXPROCS=2 go test -p 1 ./... -count=1 && go vet ./... && GOOS=js GOARCH=wasm go vet ./... && gofmt -l assets cmd internal pkg docs tools && python3 tools/check-boundaries.py`
Expected: every package `ok`, no vet output, no file listed by `gofmt`, boundary check passes.

- [ ] **Step 5: Commit** `git commit -m "Refresh the route list when the map view settles"`.

---

### Task 5: Check it in the running app

- [ ] **Step 1:** Build with `sh tools/build-native.sh` and open the app.
- [ ] **Step 2:** Narrow the window below 800 px so the map is hidden in Services, and confirm the Region dropdown appears and changes the route list. Then, at full width, in Services confirm by hand: the dropdown is short; zooming in on Japan shortens it and zooming out lengthens it up to 25; the selected route never disappears; typing a route name finds a route outside the view; tapping track still narrows the list; the hint line appears when more than 25 match and when nothing fits.
- [ ] **Step 3:** Report what was and was not checked. Native pointer and keyboard interaction cannot be driven automatically in this project (see `TESTING.md`).
