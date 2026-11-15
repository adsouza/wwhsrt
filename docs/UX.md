# Interaction, layout, and learning through play

**Status:** Proposed UX specification, 2026-11-08. Supplements [DESIGN.md](DESIGN.md), draft 0.9. The user has confirmed integrated onboarding and smartphone usability alongside larger displays; the layouts, breakpoints, copy, and usability targets below are proposals.

## 1. The experience

The player should feel like they are building and running a railway company: seeing a promising connection, weighing an expensive crossing against a detour, opening a service, and watching passengers use it. The map is the workplace. Financial and operating information should explain decisions on that map, rather than requiring a tour of disconnected management screens.

The recurring interaction is **inspect → try a plan → understand its consequences → commit → see the result**. Trying a plan is free. Buying something is explicit. The UI distinguishes a city connection, its physical railway, and the passenger services using it. A drawn line alone does not imply that trains or passengers exist.

Keep one clear next action for the selected task. Offer expert detail on demand. Experienced players can immediately use all available tools; learning cues never unlock gameplay features, demand a prescribed route, or block a business month.

**Starting over (implemented in the local first-business slice):** Company → New game opens a confirmation identifying the company and date being replaced, with explicit loss and fresh-start details. Cancel receives initial focus; Cancel/Escape leave the save and drafts untouched. Confirmation replaces the local campaign only after its save commits. A storage error leaves the existing game intact and offers retry. The fresh company returns to the East Asia map with onboarding cues restored; language, motion and terrain preferences remain. Both native and semantic browser controls support this flow, including narrow portrait layouts. This is also available after the campaign ends and is not an online world-deletion control.

### A few minutes of ordinary play

1. Open the map and see the company objective, available cash, and a few relevant opportunities. Select a city or search for one.
2. Pick a dotted construction prospect on the map or choose its named list entry, and compare reviewed alignments. Both cities may be unserved, even when the company has a different network elsewhere.
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

The local first-business slice's funding preview always identifies its city pair and alignment. Build and Company show total construction cost, a signed grant deduction, the resulting company contribution, available cash, any existing negative-cash deficit and the remaining amount needed (minimum zero). Company offers **Review this route** to inspect/change that plan without a financial command. **Borrow funding shortfall** takes the whole displayed gap in one loan; its added monthly interest is labeled separately from the editable custom loan's interest. Accepted construction/public lines do not generate another construction-funding preview. These displays explain current scenario prices and finance rules.

The top status strip always identifies the business month and unrestricted cash. Offline it provides **Advance month**; online it provides the next deadline and the player's acknowledgment state. Wider layouts add last-month operating cash flow, cash runway, and debt. On phones those additional values are one tap away in Company. Never shorten billions to millions without an explicit unit or conceal negative signs.

Reports and the selected-object inspector are views of the same objects. **Show on map**, **Edit service**, **Inspect bottleneck**, and **View payment schedule** preserve their originating context and provide a return path. Opening a report does not discard an unfinished service or construction plan.

## 3. Responsive screen layout

Layout depends on available logical width and height, text scale, and pointer capabilities, not device labels or raw framebuffer pixels. Proposed starting breakpoints are below; refine them against real content. A large text setting can select a narrower composition on a physically wide display.

| Available logical width | Composition | Use of extra room |
| --- | --- | --- |
| 320–799 px | Compact: status, command dock, focused task with tabs | Map and city inspector share space; service/finance/report tasks use the available height |
| 800–999 px | Map plus a resizable right inspector; command dock across the top | Keep at least 320 px of usable map width |
| 1000+ px and 600+ px high | Map plus right inspector; command dock at the top or left, as preferred | Wider map and inspector; choose the dock location in Company → Settings |

### Implemented task workspace

Separate global navigation and time from the task being edited. A 48 px status strip shows company (desktop), month, cash, language and a compact idle warning. The command dock contains Map, Build, Services, Report, Company and Field Notes (one row at 390 px and wider, two rows on narrower phones); its active task is tinted. The footer owns status and month/year advancement. Only the inspector scrolls; these global controls remain available.

On desktop the right inspector defaults to 440 logical px. Drag its 8 px inner divider or use **Company → Settings → Panel width** to choose 320–800 px; browser keyboard users can focus the separator and use arrows/Home/End. Clamp the displayed inspector width to leave at least 320 px for the map. The left dock is 164 px wide. Save the preferred width and top/left placement locally across restarts and new campaigns. Narrow/short windows temporarily use the top dock and compact inspector without overwriting desktop preferences. Phone map controls use a folded menu in a strip below the canvas so they do not cover a reviewed line. Expanded map hides both management panels; Restore/Escape returns to the same draft, and focus/zoom controls remain available.

The city/station browsing task is labeled **Explore** (**探索** in Simplified Chinese) in both clients. It changes the inspector task; the map and full-screen map control remain separate. The internal `map` view identifier stays stable for saved preferences.

Use task tabs to separate decisions that were previously stacked in one long column:

| Task | Default tab | Secondary tabs |
| --- | --- | --- |
| Services | **Plan**: route, frequency, fare, assigned/required trains, quick lease/return, three forecast values and Apply | **Fleet**: family/specifications and lease management; **Analysis**: complete financial and capacity forecasts |
| Build | **Plan**: available prospects, selected corridor/alignment, contribution/funding gap and commitment | **Costs**: component engineering and terrain; **Projects**: committed works and service links |
| Company | **Finance**: debt, available credit, custom loan and repayment | **Identity**: name and New game; **Settings**: workspace, appearance, motion, terrain and learning preferences |
| Report | **Overview**: passengers, revenue, costs and closing cash | **Accounts**: full reconciliation; **Network**: sections/stations; **Markets**: OD/traveler demand and journeys |

Optional disclosures contain opportunity lists, called stops, detailed station facts, project funding arithmetic and capacity contributors. Keep a named resource and its requested/available capacity visible whenever owned capacity is relevant; an overload remains a warning with Apply disabled even when its detailed explanation is collapsed. Header idle warnings open Services Plan, expand the affected-line list and show the map even on phones. Ordinary pane changes preserve drafts; explicit task links choose the pane containing the relevant controls.

Show one relevant inline concept cue (or one quiet Explain link) in a task pane. Expanding it reveals the explanation beside the controls; Field Notes supplies the complete searchable reference and can jump to the correct tab. Per-concept dismissal and global mute retain their earlier semantics. Optional content may require scrolling when expanded, on short screens, or with longer content/text; the ordinary service plan is designed to fit at 390×844 and desktop 1280×850 without inspector scrolling. This is a layout target, not first-time-player usability qualification.

### Compact / phone

Keep branding lightweight. The implemented phone header omits the app name and retains the business month and unrestricted cash; online it also retains the deadline and acknowledgment state. Its height follows content and text scale, and it must not obscure focused controls, errors, or financial review actions. The top command dock and bottom time controls remain reachable while the inspector scrolls. In focused editors, the task title takes priority over app branding.

The resting view is a map with a small status strip, command dock, and a city inspector with optional search/list browsing. Tapping an object opens a summary sheet with its name, defining facts, and primary action. The sheet has **Summary**, **Details**, and **Close** controls; dragging is an additional convenience. Expansion uses available space rather than a fixed percentage of every screen. If the visible map would become too small, Details becomes a full task view with a **Back to map** control.

A construction or service editor is a focused screen. Its header retains the task name and back control, its contents scroll vertically, and its footer contains the current financial summary and primary action above the safe area. Intermediate steps may collect endpoints, alignment, or service settings, but the review remains directly accessible and backtracking preserves inputs. The final review is ordinary page content, not another small dialog stacked on a sheet.

Comparison tables become vertically arranged alternatives with the same metric order and an explicit difference from the selected option. Ledger and service lists prioritize key columns and open row detail for the remainder. Ordinary controls and financial explanations require no horizontal page scrolling. The spatial map still pans in two dimensions.

Respect browser chrome, notches, home indicators, the on-screen keyboard, and short landscape viewports. Opening city search scrolls its field and results into view and temporarily removes competing bottom controls. Dismissing the keyboard restores the task. Rotation and resizing preserve draft, selection, navigation history, and map framing. Portrait is a complete play mode.

### Desktop and large monitors

The map occupies the remaining workspace after navigation and the current inspector. It should gain usable geographical area as the window grows. The inspector stays readable instead of stretching paragraphs across an ultrawide display. An optional pinned pane compares an earlier alignment, a service, or a report with the active selection. Limit the first design to one active editor and one read-only pinned reference.

Panels are resizable within safe limits and can be collapsed. Store the desktop preferences and retain them during compact fallback. Compact layout does not inherit an impractically wide desktop pane. A small results drawer can remain open while a service is edited on desktop; on phones the same report is a separate view with a return link.

Reframing the map considers the actual unobscured rectangle. Opening a sheet or inspector does not hide a selected endpoint beneath it. Automatic camera changes occur on explicit focus/selection commands, not whenever a forecast refreshes. Let the player stop a camera transition and retain their chosen zoom.

## 4. Map interaction and visual language

### Selecting and moving

- Single tap/click selects an object and opens its summary. Background taps preserve the construction draft. It never buys, demolishes, or applies a plan.
- One-finger drag pans; pinch zooms. Mouse/trackpad pan and zoom have visible control equivalents. Zoom buttons and **Fit selected railway** work without gestures.
- Tapping crowded markers opens a short named list with city/line/service type labels. Search and list pickers offer equivalent access without precise map targeting.
- Hover can preview a label, but no necessary information or command depends on hover, double-click, right-click, long press, or a multipointer gesture.
- Construction selection uses dotted prospects and a named corridor list. Existing network attachment is never an eligibility filter.
- Pointer-down on a sheet, button, or modal belongs to that surface through pointer-up; it cannot also pan or select the map. A pan or pinch ending over a city does not activate it.

### Showing the railway

Use **Equal Earth** as the proposed world projection, including the initial regional zoom view. Mercator/Web Mercator are excluded. Keep projection parameters stable during ordinary pan/zoom so routes and targets do not shift shape mid-gesture. This preserves relative land areas rather than making northern regions visually dominant, while still accepting that shapes and distances have projection distortion. See [PROJ's description](https://proj.org/en/stable/operations/projections/eqearth.html). The Natural Earth basemap source is independent of the projection choice.

Displayed route kilometres, engineering quantities, journey times, and costs come from geographic alignment data and the simulation, never measurements of the projected line on screen. Clip picks to the valid world outline and handle routes at the map seam explicitly. A future regional projection would need a deliberate, stable transition and the same geographic selection/draft; the initial design uses one projection through zoom levels.

#### Regional camera presets

Apply the [launch map framing overrides](../data/prototypes/launch-map-framing.v1.json) to the [base preset manifest](../data/prototypes/map-region-presets.v1.json) for the launch map. Resolve metro and station anchors using their own coordinates. Fit the minimum and maximum longitude and latitude of each anchor set with a small screen-space gutter for markers, except where a projected-point fit is specified below. The launch map uses GeoNames coordinates for Québec City, Lisbon, Brisbane, and Harbin, and OpenStreetMap station coordinates for the Frangistan latitude anchors and the Uzbekistan preset. Camera anchors do not add cities to the 94-metro roster.

| Preset | Landscape viewport | Portrait viewport |
| --- | --- | --- |
| Frangistan | Hörnefors (north), Cortadura (south), Lisbon (west), Moscow (east) | Hörnefors (north), Cortadura (south), Lisbon, Milan, Randstad |
| Africa | **Africa (East+West)**: Khartoum, Luanda, Abidjan, Dar es Salaam | **Africa+Arabia**: Abidjan, Gauteng, Tehran, Dubai-Sharjah-Ajman; exclude Karachi from the viewport |
| Mashreq | Alexandria, Tehran, Riyadh, Dubai | Not offered |
| South Asia | Bengaluru, Dhaka, Lahore, Riyadh | Bengaluru, Dhaka, Lahore, Karachi |
| East Asia | Harbin (north), Chengdu, Hong Kong, Tokyo | Same anchors plus Yangon |
| South America | Lima, Buenos Aires, Rio de Janeiro | Same anchors plus Bogotá |
| North America | Québec City, Mexico City, SF Bay | **Eastern North America**: Toronto, Miami, Atlanta, NYC; **Western North America**: SF Bay, Mexico City, Houston |
| SouthEast Asia | Ho Chi Minh City, Singapore, Yangon | Ho Chi Minh City, Singapore, Yangon, Surabaya |
| Australia | Melbourne, Brisbane | Not offered |
| Uzbekistan | Khanki, Bukhara (Kogon), Samarkand, Tashkent Central (station coordinates) | Not offered |

In portrait, omit the World button entirely. Choose the initial portrait overview from the device timezone when no saved view exists. Standard UTC offsets from 00:00 through +05:30 inclusive use maximum permitted zoom-out with Dhaka as the easternmost visible metro. Offsets strictly greater than +05:30 through +09:00 inclusive use maximum permitted zoom-out with Karachi as the westernmost visible metro. Chennai and Kolkata share UTC+05:30; the user-selected tie-break is the Dhaka-edge view. All other or unavailable timezones fall back to the Americas overview bounded by Buenos Aires, Toronto, SF Bay, and Rio de Janeiro. None of these initial overviews appears in the region selector. The selector offers only specific regions. Keep manual pan, pinch/wheel zoom, the +/− controls, and metro focus available. Landscape retains its World button. Rotating from a landscape World view into portrait opens the timezone-selected overview; existing specific-region selections follow their ordinary orientation rules. Saved manual portrait cameras retain their center and zoom within the portrait camera limits. Legacy saved World views migrate to the timezone-selected starting view.

Read the OS/browser timezone locally without geolocation or network access; this also works offline. The browser uses `Intl.DateTimeFormat().resolvedOptions().timeZone` and device-local `Date` offsets; native clients use the corresponding platform timezone data. For the initial mapping, approximate standard offset as the smaller east-of-UTC offset at local noon on January 15 and July 15 of the current year. This avoids seasonal changes at the range boundaries. Preserve valid saved regional and manual views ahead of timezone defaults, and do not recenter an active map when timezone data changes. Edge defaults use the shared cushioned latitude zoom floor. Keep the edge metro's marker fully visible and choose its screen gutter so metros geographically beyond that edge are wholly offscreen; labels can face inward. These camera choices do not change the playable catalog.


Portrait zoom-out uses Melbourne (37.8136°S) and Hörnefors (63.63292°N) as reference latitudes. A 3° cushion beyond each end gives an effective latitude band of 40.8136°S to 66.63292°N. Compute the minimum scale from the projected latitude span and actual map height. Clamp vertical translation so neither edge exposes latitude outside that band. Apply the same constraint to buttons, wheel/pinch gestures, panning, metro focus, animated transitions, resizing, and restored cameras. Disable the minus button at minimum zoom. Landscape retains its existing world extent. The portrait map height responds to available width so the initial Americas framing can retain all four anchors within this band.

Mashreq, Australia, and Uzbekistan appear only in landscape. Rotating from any of these presets to portrait clears the preset selection and preserves the geographic camera, subject to the ordinary portrait limits. SouthEast Asia replaces Oceania in the region selector. Manila is excluded from the playable catalog by game-scope choice, and Ho Chi Minh City is the SouthEast Asia eastern anchor. Taipei is playable and falls within the East Asia preset. The raw 96-record source yields 94 playable major metros. Brasília (4.8M) falls below the 5M minimum, while Phoenix remains eligible at exactly 5M. Remove the trailing “Metropolitan Area” suffix in display names while retaining original source names. In portrait, Eastern and Western North America replace the single overview. Rotation from either side returns to the overview, and rotating back restores the last portrait side. With no previous choice, default to Eastern North America. A saved Oceania preset resolves to SouthEast Asia.

Portrait means the game document viewport is taller than it is wide; a square uses landscape. Use viewport orientation rather than phone detection, a width breakpoint, or the map rectangle's own aspect ratio. A selected preset refits when the viewport changes, including height-only changes. Manual pan/zoom leaves preset mode and preserves the player's geographic camera on resize. Restored presets use the current orientation instead of reapplying a camera fitted for another shape. Fit the projected geographic rectangle, including its curved edges and equatorial extrema, so all requested anchors remain visible. East Asia fits the projected anchor points and shifts within their available padding to keep Dhaka outside the viewport in both orientations: the smallest pan that leaves Dhaka at least 16 px beyond an edge (wider than the largest city marker) while every anchor stays at least 16 px inside. That inner limit is narrower than the 26 px fit gutter so that 360 px-wide phones, where Dhaka is close to Yangon, can still exclude it. Portrait Africa+Arabia also fits projected anchors, includes Abidjan on the west, and keeps Karachi entirely outside the eastern edge. Portrait Frangistan, South Asia, and Western North America fit projected anchor positions for a closer view while retaining every anchor. South Asia aligns its northernmost anchor, Lahore, to the top gutter to reduce the space above it. These exclusions are camera constraints, not filters that hide metros from the map. Prioritize the selected metro and preset anchors when decluttering labels, and move labels below or left of a marker when needed at the viewport edge. The Equal Earth projection stays fixed. The conversation preview opens in a 390px portrait layout; its Portrait/Auto review controls can force that layout or follow the actual window. These review controls are outside the product UI; the game itself uses viewport orientation.

Default to geographical context with restrained terrain, city labels, and infrastructure. Owned, public/open-access, rival, under-construction, and draft lines use consistent line styles and labels, not color alone. A service is a highlighted path over infrastructure with called stops emphasized. Do not draw every service as another permanent competing line at world scale.

The launch network preview shows fast-track candidates as solid teal and proposed connections as dashed orange. The preview legend marks the teal layer as provisional. Hide isolated pieces of mapped fast track that lack both a displayed station and independent passenger corridor evidence. In the game, the existing HSR layer uses the qualified starting inventory: corridors between catalog cities. Isolated tagged fragments visible in this preview do not become game track. Station dots share the track color and have a separate layer toggle. At regional zoom, label stations where space permits. Give selected stations first priority for dots and labels, followed by preset anchors, operator-published stops, tagged service stops, and proximity-only stations. The map picker can focus a named station or a proposed corridor. A filled dot marks a station with a mapped rail link to another station. A hollow dot marks an operator-published stop whose connection is missing from the preview track layer. Hide candidates without either kind of evidence. The inspector identifies the evidence for each marker.

The [launch corridor shortlist](LAUNCH_CORRIDORS.md) records the proposed links. The [world-data notes](../data/README.md#prototype-basemap), [track audit](../data/prototypes/hsr-overlay-audit.v1.json), and [station review](../data/prototypes/hsr-station-review.v1.csv) record why the preview keeps or hides track and station candidates.

Show one analytical layer at a time: potential market, journey-time advantage, actual loads, shared capacity, or construction difficulty. Selection adds relevant detail without changing the meaning of the base layer. Demand is a potential market; actual passenger flow and seat capacity are separate measures. Cluster labels by zoom and selection priority, never silently remove important objects from search or lists.

The construction layer names barriers and highlights authored engineering intervals. A compact profile and cost breakdown link the mountain crossing on the map to its tunnel/viaduct expense. The same alignment's regional cost basis is inspectable. Coarse terrain artwork is contextual; the reviewed corridor profile determines price and feasibility.

## 5. Task flows and commitment

### Plan a connection

**Available prospect → alignment → project package → funding/review → commit.**
The construction UI starts from supported corridors rather than asking players to
invent a city pair. This replaces the earlier A/B endpoint-picking interaction.

#### Pick a prospect and retain its draft

Show every supported, unstarted corridor as a dotted line in Map and Build views,
using the actual reviewed alignment, including terrain and coastal detours. Show
one representative alignment per corridor until selected. Use a contrasting,
thicker highlight for the selected alternative and a text legend identifying the
dots as available construction prospects. Existing public rail, accepted projects
and operating lines retain their own styles; none is advertised as a new prospect.

Click or tap the line to select the whole corridor and open Build → Plan. Hit-test
the continuous path, including the gaps between dots, with a forgiving screen-pixel
target at every zoom level. Panning and drags never select a prospect. Map city
inspection must not replace either construction endpoint. The Build inspector
lists the same available corridors as named buttons, with a selected state, so
keyboard and phone users can choose without precise map pointing. Remove the
arbitrary-city dropdowns and Choose A/B controls. Preserve the persisted endpoint
IDs internally for draft/save compatibility, and retain existing project inspection.

Selecting a prospect identifies both endpoint names, the alternative alignment,
its package cost, opening estimate and operating forecast. Neither endpoint needs
existing railway access: supported disconnected corridors remain eligible. The
current playable catalog has two construction prospects, Tsuruga–Osaka and
Shin-Hakodate-Hokuto–Sapporo; expanding the reviewed catalog adds prospects through
the same UI. More dots must never imply unsupported construction coverage.

Compare alternative alignments after selecting a corridor. Remove stale quote
values while updating and reject late responses for older corridor/alignment/world
revisions. Selection and inspection cannot commit construction. Starting a project
immediately removes that corridor from the available list and dotted map layer;
show its accepted alignment and construction status instead, including after reload.

#### Expanded map and full-screen behavior

**Browse prospects on the map** or **Expand map** enters map-focused planning;
**Restore panels** returns with one action. Retain map/region controls, the selected
corridor identity, contextual prospect help and a stable restore control. On phones,
the named prospect list is the accessible alternative when paths are too small.
Escape/Back restores the ordinary workspace before abandoning a draft. Preserve the
selected corridor, alternative and geographic camera through expansion/restoration,
subject to the existing orientation-aware bounds. Selecting a different prospect
while expanded keeps the map expanded; restoring shows that project's estimate.

The first-use cue says “Pick a dotted construction prospect or choose one from the
list.” Explanations attach to prospect selection, alignment comparison and funding,
with per-concept dismissal and Field Notes. Do not instruct players to choose an
unsupported pair or suggest that a missing authored alignment is a gameplay mistake.

Accepted rival projects remain visible as **Under construction**, with company,
opening and any posted paid access. They leave the available-prospect set as soon
as accepted. An explicitly supported competing alignment must be a separately
reviewed offering with its competition forecast, rather than a duplicate dot.

Detours remain part of the selected alignment. Kinshasa–Lagos is the planned
coastline-detour acceptance example: a qualified land detour may be offered without
requiring an intermediate station. Missing catalog coverage is not proof that no
land route exists. The earlier A/B connection-planning HTML is a historical
illustration, superseded by this prospect selection model.

Station additions needed by the project appear in its package. Show unrestricted
cash, protected reservations, secured grants, payment timing and the funding gap.
Loans and grants are explicit decisions. A stale quote requires refreshed review;
picking a prospect cannot borrow money, lease trains or place a construction order.

### Create or improve a service

Click or tap an open railway's actual track path to enter **Services → Plan**.
Use an 18-pixel screen-space target and retain overlapping sections within four
pixels of the nearest hit for disambiguation. The existing named route selector
is scoped to open direct and through-running patterns using the clicked section(s);
show an explicit scope label and **Show all service routes** to clear it. Prefer a
direct pattern when entering from unrelated track; retain the current pattern and
its uncommitted fare/frequency when it already uses the clicked track. Selecting a
through pattern highlights its complete route, beyond the clicked section.

A drag cannot select a route. City/station marker centers retain inspection, and
representative train markers retain their replay inspector. Under-construction
track cannot create an operating service. Selecting from a full-map view restores
service controls; on phones keep a compact map visible beside the scoped chooser.
Changing the service selector through another shortcut clears an incompatible map
filter. Persist this local scope with UI preferences, and clear it on New game.
Map inspection cannot lease trains, apply a plan, spend money or advance time.

**Stops/path → fleet → return trips/day and fare → preview → apply.** Start from a city, selected track, completed project, or Services. A service needs a continuous operational usable path; that constraint is distinct from the freedom to build independent infrastructure.

For the implemented expanded-world planner, an unconfigured route opens with distance-based fare and demand-based frequency suggestions rather than a universal fare/frequency. Retain accepted plans and existing drafts; selecting the Tokyo–Osaka extension retains the starter's frequency and shows the additional leases before Apply. Construction previews use the same suggestions and state their fare/frequency assumptions.

The primary three-value forecast is **Revenue / Contribution / Break-even per month**. Keep company-wide cash change, projected passengers, feeder revenue, capacity and expense detail in **Analysis**. Analysis separates running, public access, station handling, leases and maintenance, and explains that break-even assumes the current fare, scheduled plan and passenger mix. Through services use short-trip fares in that mix. Show an em dash and explanation when a suspended service, absent through-route mix or nonpositive net fare makes a target unavailable. Invalid drafts clear old figures. The target is monthly service boardings, not unique company journeys or a claim that sufficient demand exists. The same information and contextual help are available in English and Simplified Chinese in native and browser clients.

An owned private line's inspector has **Offer paid access** with a rate per train-kilometre. Rivals see the posted rate, coverage, and notice terms on the map inspector and in service planning. **Accept access** shows the effective month and estimated cost at the proposed frequency. The forecast separates access charges from the operator's own running costs, shows available station access, and identifies any section where rights are missing or scheduled to end.

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

Help is layered: concise label → visible contextual reason → **Explain** disclosure → searchable railway/business reference with a link back to the current task. The playable clients implement this as inline disclosures and a **Field Notes** inspector with **Back to my task**, search and an explicit link to the topic’s controls. The reference preserves endpoint, alignment, company-name, loan and service drafts; returning restores expanded-map mode and the task scroll position. Expanded-map help occupies a reserved strip outside the map canvas. Adapt Africa2ice's Field Notes into this contextual reference, rather than opening a permanent textbook drawer on a small screen. Critical numbers and reasons remain in the primary interface.

The local concept record separates encountered, dismissed, explained and action-completed flags. New campaigns retain explicit dismissals and the separate global mute preference, clearing action completion. **Reset concept suggestions** explicitly clears learning flags without changing the mute setting. Legacy globally dismissed preferences migrate to muted suggestions. Successful accepted commands update action completion; rejected writes do not. See [implementation verification](validation/ONBOARDING.md).

Guidance never changes prices, passenger allocation, time, finances, RNG, or world hashes. In multiplayer there is no special frozen tutorial month. Explain the running clock before joining and show it throughout the first task.

## 7. Results, attention, and the two clocks

After resolution, show a compact **Month closed** summary with operating cash flow, total cash change and its largest causes, passenger change, and the most actionable exception. Positive progress and project openings matter alongside warnings. The detailed ledger must reconcile the summary, including capital spending and debt payments. A construction payment is not mislabeled an operating loss.

Make progress visible on the railway itself: completed project sections change their line style, a newly usable station gains its operating marker, and opening a service offers a view of its first resolved representative day. A useful new connection can highlight newly available journeys and genuinely added served population, using the simulation's definitions and avoiding double-counting. These moments are brief, optional, and replayable from Reports; they do not interrupt play with compulsory celebration dialogs or animate construction as if it already carries passengers.

One issue links to one actionable explanation: **Service A lost passengers → compare journey offer**, **Track section full → contributing services**, **Cash runway falling → payment schedule**. Explain why before recommending a remedy. The player may dismiss a warning without “fixing” a deliberate strategy.

Reports accumulate when several online months pass. Reconnecting shows **Since your last visit: N months**, current state, important changes, and report history. It does not replay a stack of old dialogs or demand acknowledgment of each month. Attention tiers distinguish a blocking rule, imminent monetary obligation, and optional opportunity using text and symbols as well as color.

The representative operating day is a map viewing mode with a distinct label **Replay of [month] · Representative day**. Its play/pause, speed, and scrub controls affect only the illustration. **Advance month** remains a separate offline business action. Pausing replay never pauses an online world. Switching back to live planning preserves the selected service and camera where possible.

### Persistent multiplayer

Before the host starts, show **Planning lobby** in place of the monthly countdown. Players can inspect the opening map and finances, compare alignments, and prepare private construction, fleet, or service drafts. The host has a distinct **Start game** action. When the clock starts, keep players in their current editor with the draft intact and show the first deadline. Players submit each draft explicitly when ready.

The status strip distinguishes **Draft — not submitted**, **Submitting**, **Plan acknowledged for [month]**, **Locked / resolving**, and **Connection lost — acknowledgment uncertain**. A local storage save is not a server acknowledgment. A compact phone view retains the current status and expands for revision details.

An online auction shows its server countdown (the longer of three minutes and one month interval) as soon as it is listed, and the nomination appears in the activity log. When bidding closes, switch to **Bidding closed · settlement pending** and show the expected settlement month. Keep existing bid reservations visible until settlement. The countdown freezes during a host pause.

Monthly results accumulate in Reports while the player works. Keep editors open across month boundaries and refresh the draft against the latest snapshot. Require a new review when terms or eligibility change, and show the effective month at submission. If the server rejects an in-flight request for a closed month, retain the draft and offer an explicit submission for the current month. After an ambiguous response, query the original operation identity. Show server-derived deadlines for auctions and restructuring, and flag an estimated or stale countdown while disconnected. On resume, refresh the world, schedule, and operation status before enabling new commitments.

Standing services continue while the player is away. The interface helps prepare through cash runway, upcoming obligations, and disclosed safeguards. No recurring “ready” click, attendance reward, or compulsory per-month housekeeping is introduced by the UX. Offline pause/reload remains possible; online closure keeps following server rules while the phone app is backgrounded.

## 8. Input, accessibility, and visual density

Use a cheerful contemporary transport-map style with bright railway blue, teal, sunflower yellow, coral and violet accents, fresh map colors, and readable deep-ink typography. Clear labels, distinct railway line styles, readable numerals, light and dark appearances, and purposeful motion keep it usable. Station activity and the representative day provide character. Keep financial panels calm enough to read. High resolution improves detail and available workspace rather than shrinking controls. The [presentation specification](PRESENTATION.md) and [theme manifest](../data/ui/theme.v1.json) define the initial palette and semantic roles.

The implemented local slice exposes **Company → Appearance: System / Light / Dark**, defaulting to System. The native control cycles between the three choices; the semantic browser control is a select. A shared runtime palette in `pkg/ui/theme.go` covers the map, terrain, stations/routes and management controls, including financial forms, warnings, active states, scrollbars and confirmation dialogs. System tracks OS/browser changes; an explicit choice takes precedence and persists across restart and New game. Dark mode uses deep navy surfaces, subdued terrain and bright railway accents with readable labels. It is a presentation preference and cannot change economic state.

All visible actions, keyboard commands, and touch controls emit the same intent with the same validation. Escape/Back closes one transient layer or returns within a task; it never silently applies or discards a draft. Text editing owns its shortcuts. Avoid a bare Space shortcut for month advancement, where a player may intend to pause replay. Any advance shortcut is documented, guarded against key-repeat, and unavailable inside editors/modals.

Proposed defaults: 16 logical px body text, larger text options, at least 44×44 logical px effective touch targets, and no essential meaning conveyed by sound or color alone.

In the native app, text sizes follow one rule. On a desktop layout (800 px and wider) nothing is below 14 px and body text is 16 px: sizes written below 14 become 14 and the rest gain 2 px. On a compact layout nothing is below 12 px and other sizes are as written. **Company → Settings → Text size** then multiplies every size by 100%, 125%, 150% or 200%. A step behaves as a window that much smaller: at 200% a 1920×1080 window shows the layout of a 960×540 one with every rectangle and font doubled, so layout breakpoints, panels, docks and map legends keep their proportions. A step is applied only while the window divided by it is at least 320×480; otherwise the largest step that fits is used and the setting says so. The preference is saved on the device and kept by New game. The ordinary plan fits one screen at 100%; larger steps may scroll. The browser client relies on browser zoom and is unchanged. Financial tables may use a compact desktop density only when readability remains intact. Browser zoom and text scaling must reflow controls; city search/list access remains available when map labels are crowded.

For web interfaces, W3C's [Reflow guidance](https://www.w3.org/WAI/WCAG22/Understanding/reflow.html) describes 320 CSS px reflow while recognizing genuinely two-dimensional content such as maps. Its [Target Size guidance](https://www.w3.org/WAI/WCAG22/Understanding/target-size-minimum.html) sets a 24 CSS px AA minimum with exceptions and identifies 44 CSS px as the enhanced threshold. The proposed 44 px touch default is a product choice; these sources do not establish that the game already conforms to WCAG.

Accessible labels, focus order, visible focus, error association, selected states, and meaningful announcements are required for management controls. City, service, route-alternative, and report lists provide a non-spatial path through core decisions. Reduced motion replaces train movement with static positions and aggregate counts, removes camera tweening, and retains full operating information. Audio has separate controls and never carries a unique warning.

### Language and Chinese text

Offer **English** and **简体中文** in the language selector. On first launch, use Simplified Chinese for device locales that resolve to `zh-Hans` or a bare `zh`. Default to English for unsupported locales. Save an explicit choice locally and apply it immediately, including in an active game with an open draft. Keep language selection independent of the device timezone used for the portrait camera. Set the web document's language for assistive technology.

Show reviewed localized names for cities, regions, lines, and stations, using the source name when a localized name is unavailable. Let city and station search match the source name and curated Simplified Chinese aliases. Preserve player-entered names as typed. Support Chinese IME composition in search and name fields. While composition is active, Enter selects or confirms the IME candidate and game shortcuts stay inactive.

Use script-appropriate font fallback and line breaking that respects Chinese punctuation without relying on spaces. Check dense map labels, station pickers, reports, and dialogs at phone widths and larger text settings. Keep financial values legible and show the settlement currency with an unambiguous symbol or code. [Unicode line breaking](https://www.unicode.org/reports/tr14/)

### Railway motion and sound

Show short track/sleeper reveals and work-front effects when authoritative construction progress changes, then an opening highlight when the line actually becomes operational. V1 timeline-derived progress is labeled scheduled; animation does not make unfinished track usable. Train markers follow resolved operating-day paths/timing, with appropriate electric-train departure, rolling and pass-by sounds plus selected station ambience. Refer to [PRESENTATION.md](PRESENTATION.md) for cues, timing, mixing, deduplication, reduced motion and background behavior. Keep master mute and independent music/effects/ambience controls available. Never replay a backlog of sounds after reconnect or make the player wait for an effect to finish.

## 9. Implementation constraints and early feasibility work

Preserve Africa2ice's isolated frames, shared intents, guarded draft lifecycle, scene ownership, and consistent drawing/hit-test transforms. Replace its fixed 1280×720 presentation assumptions and guided-first-turn sequence. Logical layout rectangles are recomputed from available space; both rendering and picking consume the same geometry. Framebuffer/device scale controls raster quality, not breakpoint choice. Rendering, accessibility views, and help consume projected frames rather than a mutable world.

Reference: Africa2ice's HUD interaction design (`docs/superpowers/specs/2026-09-02-hud-redesign-design.md` in that separate repository) illustrates shared intents, contextual diagnostics, and camera space accounting. Its fixed panel and sequential guide are reference-specific choices that this design replaces.

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
12. Show every eligible reviewed corridor as a dotted prospect. Select each by its actual path and its named list entry; verify the same project opens. Public, started and completed lines are absent from the prospect set. Dragging and unrelated city inspection preserve the construction draft. Commit one project and verify its prospect disappears before and after reload.
13. Expand/restore the map repeatedly, change prospects while expanded, restore and change alignment, pan away from the selected corridor, review funding, and return. Preserve the draft and camera; keep cost, revenue, operating contribution and restore available at 320/390px. Test allowed and denied platform full-screen, system exit, Escape/Back, safe areas, rotation, and portrait camera limits. A delayed estimate from an older pair must never overwrite the current pair, and no interaction in this sequence submits a financial commitment.

The first usability pass should settle whether the phone summary sheet leaves enough map visible, how much forecast detail belongs before commitment, and whether players distinguish infrastructure from services. Refine these from observed confusion before adding more permanent navigation or instructional text.

## 11. Service planning and monthly-report screen examples

The proposed service editor and report use one continuous decision loop. An illustrative operating scenario starts with a service on public track, locally available trains, and a completed first operating month. This is a separate scenario from the construction example, not an automatic completion of that project. The [rendering spike](../tools/ebitenui-spike/README.md) labels this placeholder service Lyon–Turin. In the [launch shortlist](LAUNCH_CORRIDORS.md#strategic-projects-10), Lyon–Turin is an unbuilt Alpine crossing, so the label does not imply existing high-speed track there. A playable public-track start uses a pair from the qualified starting inventory. Its fare, demand, cost, route, and fleet figures illustrate interaction and are not calibrated balance inputs or real-world service claims.

### Service editor

The title identifies the service and the business month being planned. The selected route states its stops, travel time, track access, and local train availability. Frequency and fare have numeric fields and large plus/minus controls; the frequency helper spells out departures in each direction. A fleet line includes the maintenance spare so the player does not have to discover an unexplained extra train later.

On a phone, Plan, Fleet and Analysis are separate tabs retaining the same route and draft. A compact action area retains the forecast operating cash contribution and **Apply plan** while the player scrolls. The detailed forecast shows passenger journeys, seat occupancy, trainsets, and the change versus the accepted plan using the same forecast month. Expanding costs separates ticket income, train running, track access, and fixed overhead. It states that no purchase or loan is included.

The first frequency interaction can show: “One return trip means one departure each way. More trips can attract passengers and need more trains.” The fare interaction can instead show: “A lower fare can fill more seats, but each passenger pays less. Compare the cash forecast before applying.” Only one cue appears at a time. Each has **Got it**; dismissing one concept does not dismiss necessary validation or require opening another explanation.

If frequency exceeds local fleet capacity, the rule replaces the optional learning cue: **9 return trips need 4 trainsets; only 3 are available.** The action is unavailable with its reason visible. The player can reduce frequency or inspect the fleet; the interface never silently orders a train. A hypothetical infeasible plan is not presented as an operable profit forecast.

Applying gives an inline acknowledgment naming the month. It leaves the clock unchanged. **Advance month** is a separate offline business action. Leaving a changed editor offers Keep draft, Apply, or Discard before navigating; Reports retains historical values even while a draft is edited. The editor and local-fleet subview keep the same active-draft guard.

### Monthly report

Start with the month and two distinct outcomes: **Operating cash** and **Total cash change**. The first report can show positive operating cash alongside a cash decline caused by a station investment. A contextual sentence beside those values explains the distinction and offers **Explain cash change**, which expands the reconciliation in place rather than launching a lesson or opening a blocking dialog.

The reconciliation is opening cash + operating cash − capital spending − interest − principal repayment = closing cash. The operating-cost disclosure separately reconciles ticket revenue and operating expenses. Displayed figures disclose rounding; underlying entries reconcile exactly. Do not label operating cash as accounting profit.

The service section shows carried journeys, offered seats, occupancy, applied fare/frequency, and the number of days in the resolved month. **Why aren't all seats filled?** explains demand versus capacity and links to **Explore fare & frequency**. That link opens the next month's editor and preserves any kept draft. The report remains an immutable result, not a forecast that changes when a fare is typed.

Reports include an investment milestone and useful progress alongside financial warnings. When multiple reports exist, a month selector provides history; one report needs no redundant single-option selector. After closing a month, the top status advances to the next planning month while the report clearly names the month just resolved.

### Phone behavior to qualify

Keep the app name out of the phone header; date/cash, the command dock and time controls stay available while the inspector scrolls. Contextual explanations are in the content flow and never cover a field or action. The action area must not overlap bottom navigation, errors, or the software keyboard. Text editing keeps focus and cursor position while forecasts refresh. Increasing frequency, dismissing a cue, switching reports, and returning to the editor must not lose a draft or unexpectedly scroll to the top.

The mockup exercises these layouts and transitions with a small deterministic UI model. Production forecasts and reports must come from the shared game port/domain algorithms specified in DESIGN.md, including server acknowledgment and capacity uncertainty online.

### Reviewing idle infrastructure and trains

**Review idle lines** opens Services, frames all current warnings together on the map and lists each affected line or fleet by name. Selecting a named row focuses that route. Services shows the map on phones when reviewing idle warnings, and expands the named warning list. Thick warning-colour strokes and endpoint rings mark the committed alignment of idle infrastructure. Idle leased fleets use dashed route strokes and an **Idle trains** label: shared infrastructure may still carry other services. Through fleets highlight every section in their actual route. The map legend explains the two styles; neither an unrelated selected route nor an unbuilt alternative is used as the idle highlight. Membership follows the authoritative current warnings, so operating a line removes its warning/highlight while leasing alone does not. Reviewing is presentation-only and changes neither accepted services nor finances.

### Explaining capacity conflicts

Place live capacity usage beside the service frequency controls. Name the affected physical section or station, distinguish daily return paths from station calls, show requested versus available capacity and identify each contributing service, including the current draft and express/through patterns. For an overload, show the excess, requests from other services, and the maximum frequency for this draft at that resource while other services remain unchanged. Explain that each return trip uses one path per section and two calls per served station; leasing more trains does not add paths. Counts use the hypothetical accepted plan, so an edited service replaces its previous frequency and an extension removes the source service before counting. Rejected previews clear old financial forecasts, disable Apply and suppress unsolicited learning cues. Valid previews retain their capacity totals so a full section is visible before another service is added. Public allocation remains provisional.

## Implemented public-network expansion

The playable map includes the prototype's public HSR geometry and visible station layer, including China. The region selector lives in Map controls (the phone map menu at compact sizes); it is available during full-map review as well as ordinary inspection. Orientation-aware presets retain the named regional bounds. East Asia remains the chosen initial region. Source station markers can be tapped for a name/status explanation. City hubs remain the service/demand endpoints; a mapped station alone cannot create a service over a missing connection.

Services has an optional **Find a service** disclosure so the expanded route dropdown does not require scrolling through thousands of patterns. Match endpoint names in English or Simplified Chinese. Long patterns show endpoints and stop count; called stops remain under route details. Selecting a route uses the existing lease/frequency/fare/forecast/Apply flow and highlights its actual source path. Map selection, region, search and inspection preserve the company checkpoint. Monthly reports expose Chinese services, passengers, public access charges and capacity through the same reporting controls. Sparse reports still allow inspecting outside-mode demand for a city pair without company rail.

When Build's endpoint selectors identify an existing public corridor, state that
HSR already connects the cities and no construction is needed. Offer **Plan trains
on this existing line** to open the matching service draft. Show no construction
quote/alternatives and keep the construction commit unavailable. Existing public
lines and committed projects stay out of Construction opportunities. A future
parallel-capacity project must be a separately authored, explicitly justified
alignment; the presence of an operating service template is not a build proposal.

The default public-rail map draws admitted, usable corridor geometry. Hide raw
source fragments and station markers that are unrelated to those corridors;
retain the complete input in the data audit. A through route remains visible
without an intermediate city and when its end cities are outside the current
viewport. Projected public geometry must match the paths used by service highlights
and trains; newly committed construction continues to draw from game state.

Coastline detail replaces the corresponding coarse basemap geometry. Use the same
refined shoreline with relief on or off and in both palettes; avoid duplicate
outlines or coarse land extending offshore beneath a finer layer. Japan and
China/Taiwan currently share the detailed source. Shading and shoreline remain
cartographic context, independent of construction prices and feasibility rules.

The repaired public-network planner shows slower connecting-track kilometres in
Analysis, with English and Simplified Chinese explanations. The ordinary Plan
stays compact. Existing rail remains one map colour; its source-based connecting
paths and associated station markers are now visible. Corridor selection,
through-running, route highlighting and new operating plans use the restored
network. Existing service settings remain intact after a save upgrade.


The public-network planner includes complete NEC and Uzbekistan all-stop services,
with their intermediate stations visible on the map. City labels retain familiar
metro names; station details identify the actual selected terminal. The NEC uses
New York Penn and Boston South rather than Metropark/Route 128 as metro anchors;
Metropark and Route 128 remain distinct service stops. Uzbekistan includes
Urgench/Khiva and Qarshi/Kitob (Shahrisabz). Conventional connecting approaches
remain visible in the same existing-rail colour, with their slower kilometres
explained in Analysis. These are game service templates, not an operator timetable.


The worldwide construction plan exposes the complete 70-project catalog in one
labeled picker with bounded native popup height (240 px), plus at most two nearby
quick choices. The dotted map still exposes every unstarted prospect. Selecting
any entry keeps the project identity, alignment, quote and expanded-map behavior
consistent; existing conventional service cannot intercept selection of a new
parallel HSR project. Started projects disappear from all selection affordances.
Native controls use their initial-selection API and ignore stale events across
rebuilds. The compact plan shows the scope; Costs discloses the terrain barrier,
regional country rates and engineering totals. Browser and native labels are
available in English and Simplified Chinese.

Lisbon–Porto service names distinguish upgraded conventional rail from the new HSR
project. Morocco's existing blue HSR ends at Kenitra; the southern projects remain
dotted until construction begins. Retired historical services keep their accounts
but are not offered as new services or animated on the current HSR map.

### Campaign endpoint in the playable clients

At the current 360-month limit, replace the footer's advance controls with a
persistent “Campaign complete” notice and “30 years · Time stopped.” explanation.
Offer “New game” beside it, opening the existing reset confirmation. Keep reports
and inspection available. Derive this state from the saved campaign frame on
load, independently of onboarding dismissal and transient “Saved” messages; clear
it after a new game starts. Use the same affordance in native/browser clients,
English/Simplified Chinese and compact/desktop layouts. Continuing beyond the
campaign remains a separate planned feature.


### Regional catalog update

Houston–Austin appears as a dotted prospect and in the same complete project
picker as the other 70 projects; it uses the ordinary selection, costs, funding,
construction and service flow. The shorter Houston–Austin–San Antonio service is
an additional route option, retaining older service paths via Dallas. Taipei is
inspectable/selectable as a city and HSR station, with Taiwan's ordered through
service available in Services. English and Simplified Chinese include Taipei,
Houston and Austin labels. Regional camera presets retain their existing bounds.
