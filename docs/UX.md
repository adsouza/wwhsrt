# Interaction, layout, and learning through play

**Status:** Proposed UX specification, 2026-11-08. Supplements [DESIGN.md](DESIGN.md), draft 0.8. The user has confirmed integrated onboarding and smartphone usability alongside larger displays; the layouts, breakpoints, copy, and usability targets below are proposals.

## 1. The experience

The player should feel like they are building and running a railway company: seeing a promising connection, weighing an expensive crossing against a detour, opening a service, and watching passengers use it. The map is the workplace. Financial and operating information should explain decisions on that map, rather than requiring a tour of disconnected management screens.

The recurring interaction is **inspect → try a plan → understand its consequences → commit → see the result**. Trying a plan is free. Buying something is explicit. The UI distinguishes a city connection, its physical railway, and the passenger services using it. A drawn line alone does not imply that trains or passengers exist.

Keep one clear next action for the selected task. Offer expert detail on demand. Experienced players can immediately use all available tools; learning cues never unlock gameplay features, demand a prescribed route, or block a business month.

### A few minutes of ordinary play

1. Open the map and see the company objective, available cash, and a few relevant opportunities. Select a city or search for one.
2. Choose **Plan connection**, select another city, and compare qualified alignments. Both cities may be unserved, even when the company has a different network elsewhere.
3. Compare the faster mountain crossing with the longer valley route. The quote names the barrier, engineering works, regional cost profile, stations, opening date, and total commitment.
4. Commit a funded project, or save the draft and instead explore running trains on public track. Progress is measured in normal business months.
5. When infrastructure and trains are available, create a service, choose stops, fare, and daily return trips, preview its costs and demand, and apply it.
6. Resolve a month offline, or let the scheduled online month close. The report explains what happened and links directly to the affected services. Watching the representative day makes the result tangible.

Construction-led starts can take several business months to produce trains. Explain that honestly and show project milestones; do not insert a hidden fast-forward, free assets, or a compulsory nearly finished railway. A scenario may openly advertise a different starting package, but that is scenario content rather than tutorial logic.

## 2. Information architecture

Use four stable destinations on every device. On desktop they form a compact left navigation rail; on a phone they form a labeled bottom navigation bar. Icons supplement words.

| Destination | Primary purpose | Important contents |
| --- | --- | --- |
| Map | Find markets, plan infrastructure, inspect the railway | City search, layers, selected-object inspector, projects, representative day |
| Services | Design and improve the passenger offer | Service list/editor, fleet inventory/orders, capacity and passenger-flow links |
| Company | Finance and expand the business | Cash runway, ledger, loans, grants, concessions, assets and auctions, objectives |
| Reports | Understand resolved results | Monthly summary, trends, changes, project milestones, contextual explanations |

Fleet is a named section within Services and is directly reachable from a fleet shortage. Finance is a named section within Company and is directly reachable from any funding gap. Avoid another permanent navigation item for each subsystem. Settings, saves, help, and leave-world actions live in the company/menu control; campaign finance and presentation settings remain distinct.

The top status strip always identifies the business month and unrestricted cash. Offline it provides **Advance month**; online it provides the next deadline and the player's acknowledgment state. Wider layouts add last-month operating cash flow, cash runway, and debt. On phones those additional values are one tap away in Company. Never shorten billions to millions without an explicit unit or conceal negative signs.

Reports and the selected-object inspector are views of the same objects. **Show on map**, **Edit service**, **Inspect bottleneck**, and **View payment schedule** preserve their originating context and provide a return path. Opening a report does not discard an unfinished service or construction plan.

## 3. Responsive screen layout

Layout depends on available logical width and height, text scale, and pointer capabilities, not device labels or raw framebuffer pixels. Proposed starting breakpoints are below; refine them against real content. A large text setting can select a narrower composition on a physically wide display.

| Available logical width | Composition | Use of extra room |
| --- | --- | --- |
| 320–599 px | Compact: status, map or focused task, bottom navigation | One inspector at a time; summary sheet can expand to a full task screen |
| 600–1023 px | Medium: map plus a compact inspector when height/width permit | Otherwise use the compact task flow; landscape phones remain supported |
| 1024–1599 px | Desktop: navigation rail, map, roughly 320–400 px inspector | Optional results drawer below the map; panels have readable line lengths |
| 1600+ px | Wide: desktop composition plus an optional pinned comparison/report pane | Compare alternatives or services without repeatedly replacing the selection |

### Compact / phone

Keep branding lightweight. The app name can appear at the top of a phone view, but scrolls out of the way with the content. A compact sticky status strip retains the business month and unrestricted cash; online it also retains the deadline and acknowledgment state. Returning to the top restores the name. Prefer this stable positional behavior over making the header appear and disappear on every change of scroll direction. Its height follows content and text scale, and it must not obscure focused controls, errors, or financial review actions. Bottom navigation remains reachable while content scrolls. In focused editors, the task title takes priority over app branding.

The resting view is a map with a small status strip, a search control, and bottom navigation. Tapping an object opens a summary sheet with its name, defining facts, and primary action. The sheet has **Summary**, **Details**, and **Close** controls; dragging is an additional convenience. Expansion uses available space rather than a fixed percentage of every screen. If the visible map would become too small, Details becomes a full task view with a **Back to map** control.

A construction or service editor is a focused screen. Its header retains the task name and back control, its contents scroll vertically, and its footer contains the current financial summary and primary action above the safe area. Intermediate steps may collect endpoints, alignment, or service settings, but the review remains directly accessible and backtracking preserves inputs. The final review is ordinary page content, not another small dialog stacked on a sheet.

Comparison tables become vertically arranged alternatives with the same metric order and an explicit difference from the selected option. Ledger and service lists prioritize key columns and open row detail for the remainder. Ordinary controls and financial explanations require no horizontal page scrolling. The spatial map still pans in two dimensions.

Respect browser chrome, notches, home indicators, the on-screen keyboard, and short landscape viewports. Opening city search scrolls its field and results into view and temporarily removes competing bottom controls. Dismissing the keyboard restores the task. Rotation and resizing preserve draft, selection, navigation history, and map framing. Portrait is a complete play mode.

### Desktop and large monitors

The map occupies the remaining workspace after navigation and the current inspector. It should gain usable geographical area as the window grows. The inspector stays readable instead of stretching paragraphs across an ultrawide display. An optional pinned pane compares an earlier alignment, a service, or a report with the active selection. Limit the first design to one active editor and one read-only pinned reference.

Panels are resizable within safe limits and can be collapsed. Store these preferences per layout class. Compact layout does not inherit an impractically wide desktop pane. A small results drawer can remain open while a service is edited on desktop; on phones the same report is a separate view with a return link.

Reframing the map considers the actual unobscured rectangle. Opening a sheet or inspector does not hide a selected endpoint beneath it. Automatic camera changes occur on explicit focus/selection commands, not whenever a forecast refreshes. Let the player stop a camera transition and retain their chosen zoom.

## 4. Map interaction and visual language

### Selecting and moving

- Single tap/click selects an object and opens its summary. Background tap clears selection unless a named endpoint-selection task is active. It never buys, demolishes, or applies a plan.
- One-finger drag pans; pinch zooms. Mouse/trackpad pan and zoom have visible control equivalents. Zoom buttons and **Fit selected railway** work without gestures.
- Tapping crowded markers opens a short named list with city/line/service type labels. Search and list pickers offer equivalent access without precise map targeting.
- Hover can preview a label, but no necessary information or command depends on hover, double-click, right-click, long press, or a multipointer gesture.
- Entering endpoint selection displays **Choose destination for [city]**, a search/list alternative, and Cancel. Existing network attachment is never an eligibility filter.
- Pointer-down on a sheet, button, or modal belongs to that surface through pointer-up; it cannot also pan or select the map. A pan or pinch ending over a city does not activate it.

### Showing the railway

Use **Equal Earth** as the proposed world projection, including the initial regional zoom view. Mercator/Web Mercator are excluded. Keep projection parameters stable during ordinary pan/zoom so routes and targets do not shift shape mid-gesture. This preserves relative land areas rather than making northern regions visually dominant, while still accepting that shapes and distances have projection distortion. See [PROJ's description](https://proj.org/en/stable/operations/projections/eqearth.html). The Natural Earth basemap source is independent of the projection choice.

Displayed route kilometres, engineering quantities, journey times, and costs come from geographic alignment data and the simulation, never measurements of the projected line on screen. Clip picks to the valid world outline and handle routes at the map seam explicitly. A future regional projection would need a deliberate, stable transition and the same geographic selection/draft; the initial design uses one projection through zoom levels.

#### Regional camera presets

The user-defined [preset manifest](../data/prototypes/map-region-presets.v1.json) names the Europe selector entry **Frangistan** and gives the following orientation-specific framing. Bounds are the minimum/maximum supplied longitude/latitude of each listed anchor set, with a small screen-space gutter for markers. These are camera presets, not changes to metro identity, geography data, concessions, or construction regions.

| Preset | Landscape viewport | Portrait viewport |
| --- | --- | --- |
| Frangistan | Moscow, Saint Petersburg, Madrid | Madrid, Milan, Randstad |
| Africa | **Africa (East+West)**: Khartoum, Luanda, Abidjan, Dar es Salaam | **Africa+Arabia**: Abidjan, Gauteng, Tehran, Dubai-Sharjah-Ajman; exclude Karachi from the viewport |
| Mashreq | Alexandria, Tehran, Riyadh, Dubai | Not offered |
| South Asia | Bengaluru, Dhaka, Lahore, Riyadh | Bengaluru, Dhaka, Lahore, Karachi |
| East Asia | Shenyang, Chengdu, Hong Kong, Tokyo | Same anchors plus Yangon |
| South America | Lima, Buenos Aires, Rio de Janeiro | Same anchors plus Bogotá |
| North America | Toronto, NYC, Mexico City, SF Bay | **Eastern North America**: Toronto, Miami, Atlanta, NYC; **Western North America**: SF Bay, Mexico City, Houston |
| SouthEast Asia | Ho Chi Minh City, Singapore, Yangon | Same anchors |

In portrait, omit the World button entirely. Choose the initial portrait overview from the device timezone when no saved view exists. Standard UTC offsets from 00:00 through +05:30 inclusive use maximum permitted zoom-out with Dhaka as the easternmost visible metro. Offsets strictly greater than +05:30 through +09:00 inclusive use maximum permitted zoom-out with Karachi as the westernmost visible metro. Chennai and Kolkata share UTC+05:30; the user-selected tie-break is the Dhaka-edge view. All other or unavailable timezones fall back to the Americas overview bounded by Buenos Aires, Toronto, SF Bay, and Rio de Janeiro. None of these initial overviews appears in the region selector. The selector offers only specific regions. Keep manual pan, pinch/wheel zoom, the +/− controls, and metro focus available. Landscape retains its World button. Rotating from a landscape World view into portrait opens the timezone-selected overview; existing specific-region selections follow their ordinary orientation rules. Saved manual portrait cameras retain their center and zoom within the portrait camera limits. Legacy saved World views migrate to the timezone-selected starting view.

Read the OS/browser timezone locally without geolocation or network access; this also works offline. The browser uses `Intl.DateTimeFormat().resolvedOptions().timeZone` and device-local `Date` offsets; native clients use the corresponding platform timezone data. For the initial mapping, approximate standard offset as the smaller east-of-UTC offset at local noon on January 15 and July 15 of the current year. This avoids seasonal changes at the range boundaries. Preserve valid saved regional and manual views ahead of timezone defaults, and do not recenter an active map when timezone data changes. Edge defaults use the shared cushioned latitude zoom floor. Keep the edge metro's marker fully visible and choose its screen gutter so metros geographically beyond that edge are wholly offscreen; labels can face inward. These camera choices do not change the playable catalog.


Portrait zoom-out uses Melbourne (37.8136°S) and Saint Petersburg (59.9311°N) as reference latitudes, with a 3° cushion beyond each end: the effective latitude band is 40.8136°S to 62.9311°N. Compute the minimum scale from the projected latitude span and actual map height; clamp vertical translation so neither edge exposes latitude outside that band. Apply the same constraint to buttons, wheel/pinch gestures, panning, metro focus, animated transitions, resizing, and restored cameras. Disable the minus button at minimum zoom. Landscape retains its existing world extent. The portrait map height responds to available width so the initial Americas framing can retain all four anchors within this band.

Mashreq appears only in landscape. Rotating from an active Mashreq preset to portrait clears the preset selection and preserves the geographic camera, following the ordinary manual-camera resize rules. SouthEast Asia replaces Oceania in the region selector. Manila and Taipei are excluded from the playable catalog by game-scope choice; Ho Chi Minh City replaces Manila as the SouthEast Asia eastern anchor. The raw 96-record source remains unchanged, with 93 playable records derived from it. Brasília (4.8M) is excluded by the current major-metro catalog’s 5M minimum; Phoenix remains eligible at exactly 5M. Remove the trailing “Metropolitan Area” suffix in display names while retaining original source names. The supplied Australian metros remain part of the world map. In portrait, Eastern and Western North America replace the single overview option; landscape retains the overview. Rotation from either side returns to the overview, and rotating back restores the last portrait side. With no previous choice, default to Eastern North America. A saved Oceania preset resolves to SouthEast Asia.

Portrait means the game document viewport is taller than it is wide; a square uses landscape. Use viewport orientation rather than phone detection, a width breakpoint, or the map rectangle's own aspect ratio. A selected preset refits when the viewport changes, including height-only changes. Manual pan/zoom leaves preset mode and preserves the player's geographic camera on resize. Restored presets use the current orientation instead of reapplying a camera fitted for another shape. Fit the projected geographic rectangle, including its curved edges and equatorial extrema, so all requested anchors remain visible. East Asia fits the projected anchor points and shifts within their available padding to keep Dhaka outside the viewport in both orientations. Portrait Africa+Arabia also fits projected anchors, includes Abidjan on the west, and keeps Karachi entirely outside the eastern edge. Portrait South Asia and Western North America fit projected city positions for a closer view while retaining every anchor. South Asia aligns its northernmost anchor, Lahore, to the top gutter to reduce the space above it. These exclusions are camera constraints, not filters that hide metros from the map. Prioritize the selected metro and preset anchors when decluttering labels, and move labels below or left of a marker when needed at the viewport edge. The Equal Earth projection stays fixed. The conversation preview opens in a 390px portrait layout; its Portrait/Auto review controls can force that layout or follow the actual window. These review controls are outside the product UI; the game itself uses viewport orientation.

Default to geographical context with restrained terrain, city labels, and infrastructure. Owned, public/open-access, rival, under-construction, and draft lines use consistent line styles and labels, not color alone. A service is a highlighted path over infrastructure with called stops emphasized. Do not draw every service as another permanent competing line at world scale.

Show one analytical layer at a time: potential market, journey-time advantage, actual loads, shared capacity, or construction difficulty. Selection adds relevant detail without changing the meaning of the base layer. Demand is a potential market; actual passenger flow and seat capacity are separate measures. Cluster labels by zoom and selection priority, never silently remove important objects from search or lists.

The construction layer names barriers and highlights authored engineering intervals. A compact profile and cost breakdown link the mountain crossing on the map to its tunnel/viaduct expense. The same alignment's regional cost basis is inspectable. Coarse terrain artwork is contextual; the reviewed corridor profile determines price and feasibility.

## 5. Task flows and commitment

### Plan a connection

**Endpoints → alignment → project package → funding/review → commit.** The desktop inspector may expose these as sections; the phone may use focused subviews. Every stage belongs to one persistent draft.

The planner lists qualified candidate corridors, with explicit reasons when an endpoint pair has no reviewed alignment. It must distinguish **no authored route yet**, **missing concession**, **unsupported crossing**, and **insufficient funds**. It must not mislabel a disconnected city as unreachable merely because it is outside the current rail graph.

When a direct endpoint line crosses ocean but a qualified land detour exists, show the curved land alignment as a selectable project. Kinshasa–Lagos is the coastline-detour acceptance example. Explain the additional distance and actual engineering costs; an unsupported direct crossing does not block the overland alternative. Do not require the player to add intermediate stations merely to make the line bend. Missing catalog coverage is **no authored route yet**, not proof that no land route exists.

Alignment comparison first shows total commitment, opening date, distance, journey-time estimate, and major structures. Details explain direct works, regional cost assumptions, land, stations, depot provision, overheads, contingency, future installments, and upkeep. No cheap headline excludes mandatory structures and then reveals them at purchase.

Station additions needed by the selected project appear in the package. Show current unrestricted cash, protected reservations, any secured grants, payment timing, and the funding gap. Loans and grants are explicit separate decisions with published terms. The final action says, for example, **Commit project · $X total**, with **$Y reserved now; schedule below** adjacent. If a stale quote changes, highlight the differences and require renewed review before accepting the updated commitment.

Committing a project ends the draft and displays a factual receipt with opening target, payment schedule, and **View project**. It does not secretly order trains or create services. A project completion report links to **Create service**, with remaining fleet and station requirements visible.

### Create or improve a service

**Stops/path → fleet → return trips/day and fare → preview → apply.** Start from a city, selected track, completed project, or Services. A service needs a continuous operational usable path; that constraint is distinct from the freedom to build independent infrastructure.

Prefer labeled stepper controls plus numeric entry for frequency and fares. Sliders may supplement them. Write **8 return trips/day — 8 departures each way**, and show units beside every editable value. Fleet selection exposes location, delivery/relocation readiness, and required trainsets. An existing train on another disconnected network cannot teleport into use.

The live preview initially shows journey time, estimated passengers, operating cash contribution, required fleet, and the first binding constraint. Expand for leg loads, feeder effect, outside-mode comparison, upkeep, and company cash scenario. Show **requested** versus **estimated allocated** frequency online; the final allocation is not guaranteed by a preview.

Explain constraints beside the affected control: **12 trips need 4 trainsets; 3 available here** with **View fleet**. Disable only the affected committing action when a genuine rule prevents it, and make the reason visible and accessible. A disabled control cannot rely on a hover tooltip to explain itself. Suggestions remain optional.

**Apply service plan** submits one validated revision; it does not advance a month. Explicit fleet purchases remain monetary commitments. A review that includes a purchase names it and its cost; no editor setting may silently take a loan or order extra trains. Suspending a service previews continuing fleet/infrastructure overhead and its effect on connections.

### Undo, drafts, and failure

Draft fields support undo/reset without simulation effects. Accepted operating plans can be revised before cutoff under ordinary rules. Financial commitments use their real cancellation/refund rules; do not label an unrecoverable transaction **Undo**. Reserve confirmations for commitments with material financial or destructive consequences. A preview and its clearly labeled review action supply confirmation; do not stack a generic “Are you sure?” on every edit.

Navigating away offers **Keep draft**, **Apply** when valid, or **Discard**. Kept drafts persist locally, are marked **Not applied**, and survive reload where local storage is available. Multiple named infrastructure concepts may be kept, but only one editor is active. Each kept draft has a source revision and is revalidated on reopen. Offline month advancement requires the active dirty editor to be resolved; saved inactive concepts do not block time. Online drafts never hold the world clock. UI draft persistence and preferences live separately from campaign/world state.

Forecasts show a pending indicator without blanking the last completed result; outdated numbers are labeled. Only the newest matching draft/revision result may replace them. An older slow response cannot overwrite a later edit. Errors preserve inputs and offer a specific remedy or retry.

## 6. Onboarding inside the game

There is no separate tutorial, forced camera tour, required checklist, or sequential Next-button lesson. A new campaign opens into ordinary play with a small number of understandable opportunities and truthful starting resources.

### Teach at the decision

| Natural event | Integrated explanation | Next action |
| --- | --- | --- |
| Company has no services | Empty state: “Operate on public track or plan a new connection.” | Browse available services / Plan connection |
| City selected | City summary distinguishes potential demand from actual passengers and identifies current rail access | Plan connection / Inspect existing track |
| First expensive crossing | One short explanation beside its cost: “The direct route needs major tunneling. Compare a longer alignment.” | Compare alignments |
| Project committed | Receipt explains installments, the opening target, and that trains/services are still needed | View project |
| No suitable train available | Service preview names train count and delivery location; explains why it cannot operate yet | View fleet |
| First service applied | Inline acknowledgment: “This plan runs in the next business month.” | Advance month offline / View deadline online |
| First resolved revenue | Report connects carried passengers, fares, running costs, and cash change; links to service and day replay | Inspect service / Watch day |
| Low load or capacity denied | Report explains the measured constraint, with relevant comparisons rather than a generic failure message | Inspect market / Inspect bottleneck |

Opportunity suggestions use qualified scenario data and normal forecast algorithms. Show why an offer is interesting, plus cost/risk, rather than claiming “best route.” An existing-track service can demonstrate operation with less construction exposure. A construction-led region gets different relevant suggestions. Do not recommend an unaffordable project as the next required action. Online suggestions cannot consume an exclusive grant or reserve shared capacity until the player explicitly commits.

### Guidance policy

At most one unsolicited learning cue appears in the active task. It is a short, dismissible sentence anchored next to the unfamiliar choice. It never covers the map target, steals focus, opens a modal, or waits for an animation to finish. Hard errors and time-critical financial information take priority. Already understandable empty states and labels often remove the need for a separate hint.

Track concepts encountered, explanations dismissed, and actions completed in local presentation state. Progress follows actual behavior rather than an expected route sequence. A player starting with an independent line, an acquisition, or an existing-track service receives relevant explanations. A returning player can hide suggestions immediately; help and **Why this number?** remain available. A dismissed cue does not return on every reload, month, or device rotation. Cross-device hint synchronization is optional and never a prerequisite for offline play.

Help is layered: concise label → visible contextual reason → **Explain** disclosure → searchable railway/business reference with a link back to the current task. Adapt Africa2ice's Field Notes into this contextual reference, rather than opening a permanent textbook drawer on a small screen. Critical numbers and reasons remain in the primary interface.

Guidance never changes prices, passenger allocation, time, finances, RNG, or world hashes. In multiplayer there is no special frozen tutorial month. Explain the running clock before joining and show it throughout the first task.

## 7. Results, attention, and the two clocks

After resolution, show a compact **Month closed** summary with operating cash flow, total cash change and its largest causes, passenger change, and the most actionable exception. Positive progress and project openings matter alongside warnings. The detailed ledger must reconcile the summary, including capital spending and debt payments. A construction payment is not mislabeled an operating loss.

Make progress visible on the railway itself: completed project sections change their line style, a newly usable station gains its operating marker, and opening a service offers a view of its first resolved representative day. A useful new connection can highlight newly available journeys and genuinely added served population, using the simulation's definitions and avoiding double-counting. These moments are brief, optional, and replayable from Reports; they do not interrupt play with compulsory celebration dialogs or animate construction as if it already carries passengers.

One issue links to one actionable explanation: **Service A lost passengers → compare journey offer**, **Track section full → contributing services**, **Cash runway falling → payment schedule**. Explain why before recommending a remedy. The player may dismiss a warning without “fixing” a deliberate strategy.

Reports accumulate when several online months pass. Reconnecting shows **Since your last visit: N months**, current state, important changes, and report history. It does not replay a stack of old dialogs or demand acknowledgment of each month. Attention tiers distinguish a blocking rule, imminent monetary obligation, and optional opportunity using text and symbols as well as color.

The representative operating day is a map viewing mode with a distinct label **Replay of [month] · Representative day**. Its play/pause, speed, and scrub controls affect only the illustration. **Advance month** remains a separate offline business action. Pausing replay never pauses an online world. Switching back to live planning preserves the selected service and camera where possible.

### Persistent multiplayer

The status strip distinguishes **Draft — not submitted**, **Submitting**, **Plan acknowledged for [month]**, **Locked / resolving**, and **Connection lost — acknowledgment uncertain**. A local storage save is not a server acknowledgment. A compact phone view retains the current status and expands for revision details.

After a rejected stale plan or a month closing mid-edit, retain the draft, refresh the snapshot, show changed assumptions, and require review before submitting against the new month. After an ambiguous response, query the original operation identity; never infer success or blindly buy again. Show the server-derived deadline, and explicitly flag an estimated/stale countdown while disconnected. On resume, refresh world, schedule, and operation status before enabling new commitments.

Standing services continue while the player is away. The interface helps prepare through cash runway, upcoming obligations, and disclosed safeguards. No recurring “ready” click, attendance reward, or compulsory per-month housekeeping is introduced by the UX. Offline pause/reload remains possible; online closure keeps following server rules while the phone app is backgrounded.

## 8. Input, accessibility, and visual density

Use a cheerful contemporary transport-map style with bright railway blue, teal, sunflower yellow, coral and violet accents, fresh map colors, and readable deep-ink typography. Clear labels, distinct railway line styles, readable numerals, light and dark appearances, and purposeful motion keep it usable. Station activity and the representative day provide character. Keep financial panels calm enough to read. High resolution improves detail and available workspace rather than shrinking controls. The [presentation specification](PRESENTATION.md) and [theme manifest](../data/ui/theme.v1.json) define the initial palette and semantic roles.

All visible actions, keyboard commands, and touch controls emit the same intent with the same validation. Escape/Back closes one transient layer or returns within a task; it never silently applies or discards a draft. Text editing owns its shortcuts. Avoid a bare Space shortcut for month advancement, where a player may intend to pause replay. Any advance shortcut is documented, guarded against key-repeat, and unavailable inside editors/modals.

Proposed defaults: 16 logical px body text, larger text options, at least 44×44 logical px effective touch targets, and no essential meaning conveyed by sound or color alone. Financial tables may use a compact desktop density only when readability remains intact. Browser zoom and text scaling must reflow controls; city search/list access remains available when map labels are crowded.

For web interfaces, W3C's [Reflow guidance](https://www.w3.org/WAI/WCAG22/Understanding/reflow.html) describes 320 CSS px reflow while recognizing genuinely two-dimensional content such as maps. Its [Target Size guidance](https://www.w3.org/WAI/WCAG22/Understanding/target-size-minimum.html) sets a 24 CSS px AA minimum with exceptions and identifies 44 CSS px as the enhanced threshold. The proposed 44 px touch default is a product choice; these sources do not establish that the game already conforms to WCAG.

Accessible labels, focus order, visible focus, error association, selected states, and meaningful announcements are required for management controls. City, service, route-alternative, and report lists provide a non-spatial path through core decisions. Reduced motion replaces train movement with static positions and aggregate counts, removes camera tweening, and retains full operating information. Audio has separate controls and never carries a unique warning.

### Railway motion and sound

Show short track/sleeper reveals and work-front effects when authoritative construction progress changes, then an opening highlight when the line actually becomes operational. V1 timeline-derived progress is labeled scheduled; animation does not make unfinished track usable. Train markers follow resolved operating-day paths/timing, with appropriate electric-train departure, rolling and pass-by sounds plus selected station ambience. Refer to [PRESENTATION.md](PRESENTATION.md) for cues, timing, mixing, deduplication, reduced motion and background behavior. Keep master mute and independent music/effects/ambience controls available. Never replay a backlog of sounds after reconnect or make the player wait for an effect to finish.

## 9. Implementation constraints and early feasibility work

Preserve Africa2ice's isolated frames, shared intents, guarded draft lifecycle, scene ownership, and consistent drawing/hit-test transforms. Replace its fixed 1280×720 presentation assumptions and guided-first-turn sequence. Logical layout rectangles are recomputed from available space; both rendering and picking consume the same geometry. Framebuffer/device scale controls raster quality, not breakpoint choice. Rendering, accessibility views, and help consume projected frames rather than a mutable world.

Reference: Africa2ice's [HUD interaction design](/Users/adsouza/Code/africa2ice/docs/superpowers/specs/2026-09-02-hud-redesign-design.md) illustrates shared intents, contextual diagnostics, and camera space accounting. Its fixed panel and sequential guide are reference-specific choices that this design replaces.

The Go/domain and local/remote game-port boundaries remain. Choosing an Ebitengine renderer does not establish smartphone browser lifecycle, text input, or screen-reader support. Milestone 1 must prove a narrow-screen task with touch, software keyboard, browser back behavior, text scaling, background/resume, and semantic controls. Evaluate iOS Safari and Android Chrome on actual devices. Mobile browser single-player runs locally, with explicit save/export behavior; native iOS/Android packaging is a separate delivery decision.

A canvas-only HUD cannot be assumed to expose accessible semantics. Prototype an accessible management-control layer early: if Ebitengine UI tooling cannot meet the requirements, use a semantic HTML HUD for the browser around the shared Go/WebAssembly game port and renderer. Keep intents/frames aligned across hosts. Qualify the native accessibility bridge separately; report gaps explicitly and resolve the architecture before promising equivalent accessibility. This decision precedes building dozens of custom canvas forms.

The [2026-11-08 evidence](validation/README.md) confirms that real ebitenui widgets can reproduce the service/report style and reflow between desktop and phone compositions. The spike implements custom theme images, explicit stretch/wrapping rules, persistent edits across layout changes, an application scroll policy, and platform select-all adaptation. Pointer/keyboard, 2× DPI, and touch-emulated numeric input pass focused checks. The canvas exposes no semantic button/form structure, so accessible browser controls and physical phone/keyboard/lifecycle checks remain required. Visual feasibility does not settle the final host architecture.

Local draft preservation uses the existing storage adapters with separate versioned UI records. Browser background termination is expected: save active edits opportunistically and expose storage failures. Online backgrounding does not stop server time. Browser resume treats cached deadlines and forecasts as stale until refreshed. Do not rely on background JavaScript timers as the source of truth.

Phone performance is a separate qualification target. Reduce basemap detail, visible train markers, and decorative animation before reducing information or removing actions. Stop replay drawing when hidden, and keep forecasts bounded/cancelable. Pin representative mobile hardware and workload measurements during the vertical slice rather than inheriting desktop performance claims.

## 10. Acceptance and questions to test

These are proposed research targets, not measured results. Test with first-time players without a spoken tutorial and with experienced players who disable hints. Include both a public-track start and a construction-led start.

1. A new player can identify available money, the current business month, the company goal, and a plausible next action within the first minute.
2. In a public-track start, most first-time testers can preview and apply a valid service within ten minutes without outside instruction, then explain why the first report's cash change differs from operating result.
3. In a construction-led start, testers can compare a crossing and detour, find the funding gap/payment schedule, and commit a viable independent project. They can explain why trains do not appear immediately.
4. The complete core loop works at 320 and 390 CSS px widths and on short landscape viewports: city selection, independent construction, financing, fleet, service editing, reports, save/export, and online acknowledgment. No essential action requires hover, precision dragging, or desktop mode.
5. At 768, 1280, 1920 and 2560 logical px widths, additional space increases map/comparison utility. Repeat at increased text scale and 1×/fractional/2× device scale; controls and hit areas agree. A 4K display at 2× scale is approximately a 1920-logical-pixel layout.
6. Rotate/resize or open the software keyboard midway through each editor. Preserve inputs and focus sensibly; financial actions and errors remain visible without overlapping system controls.
7. First-time learning works after choosing an unexpected route. Disabling all suggestions leaves a complete usable game. Reloading does not resurrect dismissed cues, and guidance does not change a campaign checkpoint.
8. A player can follow a low-ridership report to its cause, edit the relevant fare/frequency, compare the forecast, and return to the report with context intact.
9. Close an online month during editing, lose a response after a financial commitment, and background/resume a phone. Preserve the draft, refresh the world, and show correct acknowledgment without duplicate purchases or false success.
10. Keyboard-only and assistive-technology users can navigate and perform core management tasks through semantic controls and lists. Validate this on the chosen browser/native implementations; a visually correct canvas is insufficient evidence.
11. A stable online company can be left operating without repetitive acknowledgments. On return, one concise summary explains elapsed months, important changes, and the next deadline.

The first usability pass should settle whether the phone summary sheet leaves enough map visible, how much forecast detail belongs before commitment, and whether players distinguish infrastructure from services. Refine these from observed confusion before adding more permanent navigation or instructional text.

## 11. Service planning and monthly-report screen examples

The proposed service editor and report use one continuous decision loop. An illustrative operating scenario starts with a public-track Lyon–Turin service, locally available trains, and a completed first operating month. This is a separate scenario from the construction example, not an automatic completion of that project. Its fare, demand, cost, route, and fleet figures illustrate interaction and are not calibrated balance inputs or real-world service claims.

### Service editor

The title identifies the service and the business month being planned. The selected route states its stops, travel time, track access, and local train availability. Frequency and fare have numeric fields and large plus/minus controls; the frequency helper spells out departures in each direction. A fleet line includes the maintenance spare so the player does not have to discover an unexplained extra train later.

On a phone, settings and detailed forecasts are sequential. A compact action area retains the forecast operating cash contribution and **Apply plan** while the player scrolls. The detailed forecast shows passenger journeys, seat occupancy, trainsets, and the change versus the accepted plan using the same forecast month. Expanding costs separates ticket income, train running, track access, and fixed overhead. It states that no purchase or loan is included.

The first frequency interaction can show: “One return trip means one departure each way. More trips can attract passengers and need more trains.” The fare interaction can instead show: “A lower fare can fill more seats, but each passenger pays less. Compare the cash forecast before applying.” Only one cue appears at a time. Each has **Got it**; dismissing one concept does not dismiss necessary validation or require opening another explanation.

If frequency exceeds local fleet capacity, the rule replaces the optional learning cue: **9 return trips need 4 trainsets; only 3 are available.** The action is unavailable with its reason visible. The player can reduce frequency or inspect the fleet; the interface never silently orders a train. A hypothetical infeasible plan is not presented as an operable profit forecast.

Applying gives an inline acknowledgment naming the month. It leaves the clock unchanged. **Advance month** is a separate offline business action. Leaving a changed editor offers Keep draft, Apply, or Discard before navigating; Reports retains historical values even while a draft is edited. The editor and local-fleet subview keep the same active-draft guard.

### Monthly report

Start with the month and two distinct outcomes: **Operating cash** and **Total cash change**. The first report can show positive operating cash alongside a cash decline caused by a station investment. A contextual sentence beside those values explains the distinction and offers **Explain cash change**, which expands the reconciliation in place rather than launching a lesson or opening a blocking dialog.

The reconciliation is opening cash + operating cash − capital spending − interest − principal repayment = closing cash. The operating-cost disclosure separately reconciles ticket revenue and operating expenses. Displayed figures disclose rounding; underlying entries reconcile exactly. Do not label operating cash as accounting profit.

The service section shows carried journeys, offered seats, occupancy, applied fare/frequency, and the number of days in the resolved month. **Why aren't all seats filled?** explains demand versus capacity and links to **Explore fare & frequency**. That link opens the next month's editor and preserves any kept draft. The report remains an immutable result, not a forecast that changes when a fare is typed.

Reports include an investment milestone and useful progress alongside financial warnings. When multiple reports exist, a month selector provides history; one report needs no redundant single-option selector. After closing a month, the top status advances to the next planning month while the report clearly names the month just resolved.

### Phone behavior to qualify

Retain the approved scrolling header: the app name scrolls away, while date/cash and bottom navigation stay available. Contextual explanations are in the content flow and never cover a field or action. The action area must not overlap bottom navigation, errors, or the software keyboard. Text editing keeps focus and cursor position while forecasts refresh. Increasing frequency, dismissing a cue, switching reports, and returning to the editor must not lose a draft or unexpectedly scroll to the top.

The mockup exercises these layouts and transitions with a small deterministic UI model. Production forecasts and reports must come from the shared game port/domain algorithms specified in DESIGN.md, including server acknowledgment and capacity uncertainty online.
