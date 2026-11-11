# World-wide High-speed Rail Tycoon — Product & Architecture Specification

**Status:** Draft 0.9 — Equal Earth world/region prototype and actual ebitenui rendering evidence, alongside the product, geography, and cost specifications.

**Date:** 2026-11-08

**Scope:** A complete strategic railway business game, delivered first as a small playable network and then as a worldwide campaign.

## 1. The game

Build a railway company that makes high-speed rail the best way to travel between the world's cities. Begin in one region, identify a promising corridor, finance and construct it, buy trains, and attract passengers with a useful combination of speed, frequency, price, and connections. Reinvest in a network whose connected journeys become more valuable than its individual lines.

The central tension is **network usefulness versus capital and operating cost**. The shortest line may cross expensive terrain. The fastest train may be uneconomic on a short route. A new branch may lose money itself but feed a profitable trunk. Frequent trains improve the passenger offer while consuming fleet and infrastructure capacity.

The game uses real-world geography and recognizable cities. Its financial values, passenger behavior, project delivery, and policies are explicit game abstractions. It does not present its outputs as engineering estimates or transport forecasts.

### Decisions confirmed by the user

| Decision | Direction |
| --- | --- |
| Time | Monthly business turns with an animated operating-day view |
| Management depth | City connections, stations, fleet, fares, and service frequency |
| Campaign | Start in one region on a real-world map and expand worldwide from a contemporary setting |
| Starting world | Existing high-speed rail lines are available to operate or expand; independent new lines can also be built |
| External competition | Abstract road and air offers compete for passengers |
| Human competition | An optional server-connected multiplayer world contains competing player companies |
| Offline play | Single-player runs entirely locally without a server connection |
| Finance | Private operator/developer with loans and public grants for useful projects |
| Legacy ownership | Existing HSR starts public; players may bid to acquire lines subject to reasonable reserve prices |
| Multiplayer clock | A persistent world advances on a fixed schedule while players may be offline |
| First server release | Small private games on a hostable dedicated server, with invite codes |
| Purchased legacy lines | Retain open access at regulated fees; the owner earns access revenue and pays maintenance |
| Rival construction | All companies can see committed projects and planned openings. The planner flags an existing or committed line for the same city pair before another company builds. |
| Access to new private lines | The owner can offer paid access to other operators with a shared path quota. Operators share the quota; the owner keeps the remaining capacity. |
| Station access | Every station is open access at regulated per-call fees. A company that owns a station earns those fees but cannot refuse access. |
| Notices and auctions | Online notices that can suspend another company's services cover the unattended target. Auctions are opportunities that an absent player may miss. |
| Insolvent and removed companies | Their track and stations keep operating under a public caretaker. Accepted access rights continue at pinned fees. |
| Lines under construction at the cutoff | They are not in the starting world. Qualified ones become buildable catalog corridors. |
| Train orders | Unpaid installments are reserved when the order is placed, like construction. |
| Geographic difficulty | Real-world barriers shape route feasibility and cost; mountain crossings are very expensive |
| Independent construction | Either or both endpoint cities may be unserved; a new line need not connect to an existing network |
| Learning to play | Onboarding is integrated transparently into normal gameplay, without a separate required tutorial |
| Responsive interface | Complete core gameplay on smartphone screens; larger displays add map area and useful simultaneous views |
| Localization | Provide a Simplified Chinese player interface |
| Projection constraint | Do not use Mercator for the game map |

### Proposed detail

The confirmed choices above govern this draft. Remaining mechanics and defaults are proposed design details, with a short review list in §15. Numbers labeled **initial** are starting balance values, not approved realism claims or measured performance. This is a specification task; no game implementation, deployment, or server hosting has been performed.

### Product principles

1. Geography creates choices: terrain, distance, water, and city distribution determine feasible routes, construction cost, completion time, and service performance. Mountain crossings require a major investment or a deliberate detour.
2. Connections create value: through passengers, transfers, and shared trunks are core mechanics.
3. Every major result is explainable: show why a corridor is expensive, why a train is empty, and why cash fell.
4. Strategy stays manageable: the player defines service patterns rather than signals or individual departure times.
5. The operating day makes the business tangible: trains and station activity illustrate the accepted service plan.
6. One simulation supports desktop, browser, saves, forecasts, and automated play.

## 2. Campaign and scope

### Main campaign

Proposed initial campaign: start in January 2025 and operate for 30 years, with an option to continue after the final report. The scenario includes high-speed lines open at its declared cutoff, reduced to a reviewed strategic graph of corridors between catalog cities. Projects under construction and planned lines are classified separately and do not appear as already operational. In the first version they are not in the starting world. A qualified project still under construction at the cutoff becomes an ordinary buildable catalog corridor, priced as new construction. Scripted public construction is a possible later extension. The date can change to suit a qualified data snapshot. Real geography and existing infrastructure provide the starting point; future development and commercial ownership follow the game's rules.

Choose a starting region and a home city. A regional offer provides a starting financial package, access to appropriate existing infrastructure, and several plausible expansion corridors. Regions without existing HSR offer a funded construction-led start. The rest of the world is visible from the start; market research has no exploration fog. Expansion requires capital and a simple regional operating concession, not geographical discovery.

The company may build and run multiple disconnected networks, within one region or across different regions. It can start a new A–B railway even when neither city belongs to the public network or to any company network, including its own. Each network must stand on its own rail connectivity: a passenger cannot transfer between separate rail components merely because the same company owns both. Worldwide success does not require a continuous railway across oceans, and joining an existing trunk is a strategic choice rather than a construction prerequisite.

Initial release target: **about 250 curated metropolitan areas** across Europe, Africa, Asia, North America, South America, and Oceania. A milestone is not called worldwide until all six have playable starting or expansion opportunities. Antarctica is outside the passenger-market scope.

The city catalog is deliberately curated rather than selected only by population. The user-supplied [96-record metro seed](../data/raw/metropolitan_areas_5m.json) is the starting candidate list of major anchors, with the original IDs and estimates retained. The current playable selection excludes Manila by user-directed scope (the Philippines is outside the launch catalog) and Brasília because its 4.8M population is below the 5M minimum, leaving 94 of those major anchors. Taipei is playable; the raw source remains intact. Display names omit the trailing “Metropolitan Area” suffix. It supplements rather than replaces the proposed broader catalog: add smaller intermediate cities and existing-HSR stops needed for useful corridors. The seed is not yet a verified, exhaustive list of all metros above 5M; its recorded values include two threshold exceptions. All playable regions need at least one viable starting strategy under the normal financial rules; identical starting packages are not required. The [existing-HSR stop-city screen](../data/README.md#existing-hsr-stop-city-candidates) finds 161 non-metro cities of 1M+ on the provisional fast-track layer. That is more than the roughly 156 remaining places, so the broader catalog needs a regional allocation rule. A [draft allocation](../data/README.md#draft-catalog-allocation) splits the places by role: 60 launch endpoints, 32 checked stops outside China, 14 smaller stops outside China, and 50 Chinese cities chosen by network role.

### Progression

- First railway: launch a useful service on existing track, or build an independent corridor between two cities, and complete a business month. A construction-led start is available even in a region that already has HSR elsewhere.
- Regional network: serve intermediate cities, introduce connecting services, and expand a congested trunk.
- International operator: obtain another concession and run cross-border services.
- Worldwide company: develop multiple viable regional networks and meet the campaign's financial and connectivity objectives.

Concessions have a published fee, fixed lead time, and clear eligibility requirements. In the first version, they abstract commercial access and cross-border agreements. They do not attempt to simulate current international law or diplomatic negotiations.

A regional concession may require capital and a fee, but cannot require a physical connection back to the home city or another company line. The same principle applies to eligibility for building a standalone line in that region.

### Success and failure

The proposed campaign asks for both a viable company and a useful network. Each scenario declares numerical targets for rolling operating cash flow, passenger journeys, connected metropolitan population, and active regions. A region counts only when it has an operating service and meaningful rolling ridership; purchasing a concession alone earns no credit.

Report profit, cash, debt, passenger-kilometres, unique connected population, and regions served separately. Avoid a single score that conceals a bankrupt company behind track kilometres. Connected population counts each metro once and requires an operating station served by a live service.

Failure occurs when the company cannot meet mandatory payments after its disclosed credit facility and the restructuring window. The baseline window is three business months, extended in fast multiplayer worlds by the response-time rule in §10. During restructuring, time can advance, services can be reduced, trains sold, and new debt arranged within limits. New construction commitments are blocked during restructuring. Insolvency follows from unpaid obligations.

Sandbox offers configurable starting capital and optional insolvency. Difficulty and subsidy settings belong to the campaign and its save, and changing them marks the campaign as customized.

### First-version boundaries

Include the existing HSR network, passenger rail, construction projects, station capacity, a small train catalog, service patterns, fares, transfers, competing transport modes, debt, grants, maintenance, monthly reporting, saves, the operating-day view, offline single-player, and optional server-based multiplayer.

Defer freight, signaling, platform-by-platform control, exact timetables, individual passengers, AI companies building track, real-estate development, detailed tax/FX systems, stock trading, and speculative transoceanic megastructures. Multiplayer is part of the product scope; an official large public service is a separate hosting decision.

## 3. Player loop and the two clocks

### Monthly business turn

1. Read last month's results and the most urgent issues.
2. Inspect demand and compare corridor alternatives.
3. Commit construction, order or reassign trains, and edit services.
4. Preview the resulting offer and financial exposure.
5. Advance one month.
6. Inspect ridership, capacity, cash movements, and project progress; optionally watch the operating day.

Planning has no action-point limit. Accepted financial commitments immediately affect available funds. Service edits take effect in the next resolved month. Draft edits remain local until applied; leaving a dirty editor offers Apply, Keep draft, or Discard. Offline, a player can advance with warnings such as an idle train, but cannot advance with invalid accepted state or an unresolved active editor. Inactive kept concepts do not block time and are revalidated when reopened. Online, the server deadline advances the world using acknowledged plans; an unfinished local editor never blocks other companies or silently becomes an accepted order. See [the UX specification](UX.md) for draft persistence and review behavior.

### Animated operating day

The operating-day view is a replayable illustration of one representative day from a resolved month. It has pause, speed, scrub, and reduced-motion controls. Watching it never advances the campaign, spends money, generates another month's demand, or changes a random draw.

Display trains following their actual service paths, stopping at called stations, and carrying a visual load based on resolved leg occupancy. A selected train identifies its service, next stop, speed limit, and average load. Station activity reflects boardings, alightings, and transfers in aggregate.

The simulation does not retain thousands of train agents. The view derives a bounded animation schedule from the saved service plan and monthly report. Mark the view **Representative day — aggregate loads**. Departure spacing illustrates frequency; it is not a claim that a detailed dispatch timetable has been validated. Zoomed-out views may show a labeled sample of trains.

The same month and service plan produce the same illustration. Scrubbing, frame rate, audio, and the duration for which the view is open do not enter campaign state. The monthly report remains authoritative.

## 4. World, geography, and construction

### Map

Use a zoomable, pannable 2D world map with regional focus. Show recognizable coastlines, restrained terrain shading, cities, stations, and rail infrastructure. At world scale, cluster labels and summarize networks; at regional scale, show individual corridors and services. Keep the build tools usable without rotating a globe.

**Proposed projection: Equal Earth**, for the world overview and the initial regional zoom implementation. It preserves relative areas, giving Africa, South America, and other regions their proper visual weight in a worldwide game. [PROJ's Equal Earth documentation](https://proj.org/en/stable/operations/projections/eqearth.html) describes its global purpose and area-preserving property. Mercator and Web Mercator are excluded by the user's instruction. The Natural Earth basemap dataset and the choice of map projection are separate decisions.

The [regional camera specification](UX.md#regional-camera-presets), [base preset manifest](../data/prototypes/map-region-presets.v1.json), and [launch map overrides](../data/prototypes/launch-map-framing.v1.json) define regional bounds, the portrait latitude cushion, and device-timezone initial views.

Use the same projection while zooming and panning in the first implementation, so the railway does not unexpectedly reshape under the player's pointer. Equal Earth does not preserve every shape, angle, or distance. A distinct locally centered regional projection remains a possible later refinement, subject to demonstrating a worthwhile improvement and a stable, explicit transition; it is not a prerequisite for the initial map.

Geographic coordinates are authoritative; projected screen distances are never simulation distances. Compute alignment lengths from their geographic polylines under the declared geodesic model, and use the reviewed engineering intervals for construction costs and operating profiles. Equal projected area does not make screen lengths an engineering ruler. Any regional scale indicator is local to the indicated reference position, not a promise of constant scale across the world.

Handle the antimeridian explicitly in drawing, picking, and route geometry. Split geometry at the chosen map seam instead of drawing a line across the entire screen; do not invent connectivity between separate components. Map drawing and picking use the same forward/inverse projection and camera transforms; points outside the projected world outline are not valid geographic picks. Projection, zoom, and framing never alter demand, route kilometres, construction costs, or a campaign checkpoint.

### Infrastructure vocabulary

| Term | Meaning |
| --- | --- |
| City | One metropolitan market and one initial station site |
| Station | Company access to a city's passengers and a bounded number of daily train calls |
| Corridor | A buildable geographical connection between two cities |
| Alignment | One surveyed alternative for that corridor, with a fixed geometry and engineering breakdown |
| Track section | An owned operational edge with speed and directional train-path capacity |
| Project | A funded construction or upgrade commitment with progress and a completion date |
| Service | An ordered path through operational track, a stopping pattern, fleet assignment, frequency, and fare |
| Journey | One passenger's origin-to-destination itinerary, potentially using multiple services |

A service is not a track section. Many services may share track, and a service may traverse several sections. Passing through a station without stopping is possible. Lines crossing on the map do not form a junction. Graph topology uses explicit station and junction nodes; non-market junctions preserve real connections without inventing a city or passenger market. Passenger transfers require called stations rather than junctions. The player primarily works with city connections while the internal graph retains this topology.

### Existing infrastructure and access

The starting scenario records each existing section's operational status, strategic geometry, speed, capacity, stations/junctions, and ownership. Every starting section lies on a corridor between two catalog cities, under the admission rule in §12. Real line identities and opening dates are source-backed. Initial capacity and access charges may be authored abstractions where qualified operational data is unavailable; label that distinction.

Existing HSR starts as a neutral public network. Companies obtain regional concessions, station access, and train paths in exchange for published charges. A company may operate on existing infrastructure before it can afford a new line. Players may acquire existing lines through reserve-price auctions. Acquired legacy lines remain open-access infrastructure: ownership brings regulated access revenue, maintenance obligations, and investment opportunities rather than permission to exclude competitors. Public, acquired open-access, and newly built private infrastructure are visually distinct.

Existing infrastructure cannot be sold or deleted by a player who only has operating access. Access fees are charged by train-kilometre and/or station call as published, not as an additional ticket fare. An acquired line's owner receives rival operators' access payments and pays upkeep; its own services have no cash payment to themselves, with an internal transfer price available for reporting only. Acquisition does not increase capacity or exempt the owner's trains from shared allocation. Regulated fees are published scenario rules and cannot be raised unilaterally by the owner. Track access never creates passengers or grants an exclusive market. The first version has no autonomous HSR company expanding in the background; human companies provide rail competition in multiplayer. The legacy network is infrastructure, not an invisible fare-setting rival.

New private track starts with owner access. The owner can publish a paid access offer for an owned line, including while it is under construction. The offer names the sections it covers, the rate per train-kilometre, the shared path quota, and the notice period. The notice period follows the scenario rules and is never shorter than the online minimum notice in §10. Other eligible companies can accept posted terms once the track is operational. An accepted right takes effect at the next business month boundary and covers train paths on the named sections. Station calls use the regulated station access in §5, and concessions require their own permissions.

Accepted rights retain their fee until the parties accept replacement terms or the right ends. The owner can close the offer to new operators. Ending an accepted right requires the published notice and takes effect at a month boundary. Forecasts show the fee and any scheduled end, and affected services report a suspension or reroute requirement when the right expires. Charge for actual operated train-kilometres on each covered section. Credit the owner for the matching amount paid. The owner pays upkeep.

The shared path quota is the number of daily return pairs on each covered section that operators with accepted rights may use. Those operators share the quota under the fair allocation rules. The owner keeps the section's remaining capacity for its own services. Quota paths the operators leave unused in a month become available to the owner in the same resolution. The owner's remaining capacity is never offered beyond the quota. Raising the quota takes effect at the next month boundary. Lowering it requires the same notice as ending a right. Closing the offer to new operators leaves the quota in place for existing rights.

Grant-funded track follows any public-access conditions disclosed before grant acceptance. Acquired legacy lines retain their regulated open access. The voluntary offer applies to new private lines.

### Acquiring an existing line

A line is sold as an explicit bundle of track sections and maintenance obligations. Concurrent auction bundles cannot contain the same section, and nomination cannot remove infrastructure from service. Publish its geometry, capacity, regulated tariff, recent usage/access income, annual upkeep, upgrade options, reserve price, and closing deadline. Station ownership is excluded unless the listing explicitly includes it; buying the track must not silently monopolize a shared city station. Public lines and acquired open-access lines cannot be removed to evict rivals. Capacity-affecting works require advance notice of at least the online minimum notice in §10, and the same published disruption rules for every operator, including the owner.

Initial reserve-price proposal:

```text
reserve_price = max(
    40% × authored replacement_cost,
    8 × max(0, normalized_annual_net_access_income)
)
```

Replacement cost uses the same corridor engineering and upgrade tables as new construction. Net access income subtracts mandatory infrastructure upkeep; before 12 operating months, annualize the available history and label the shorter observation period. The asset-value floor prevents an initially idle public line from being sold for almost nothing. The percentage and income multiple are initial balance values to qualify against representative lines; the user requirement is a reasonable, visible reserve, not these particular coefficients. Bids cannot receive construction grants unless an offer explicitly allows acquisition spending.

Proposed auction format: sealed maximum bids with a second-price clearing rule. Players may nominate a public line for sale. Multiplayer bidding lasts the longer of **three real minutes** and one business-month interval from listing. Nominations appear in the world activity log and on reconnect. Unattended play protects a company's operations, not its opportunities, so an absent player can miss an auction. The server publishes the cutoff, seals the bids at that instant, and settles at the first monthly resolution whose cutoff is at or after the bidding deadline. The offline proposal uses 12 business months. Reserve the bidder's maximum amount in unrestricted cash. Total bid, construction, and train-order reservations cannot exceed available funding. Eligible companies may place, revise, or withdraw bids before the bidding cutoff, with the revised reservation validated atomically. Bids must meet the reserve. Qualified markets need multiple available listings.

At close, the highest eligible bidder pays the greater of the reserve and the second-highest bid plus a published bid increment, capped at their maximum. With one eligible bidder, the price is the reserve; with no bidder, the line stays public. Equal maximum bids use a deterministic rotating tie-break independent of request arrival time; the winner pays that maximum. Release unused reservations and losing bids immediately. Payment goes to the public treasury and creates a company infrastructure asset, not an operating expense or revenue to another player. There are no player-to-player line resales in the first version.

Ownership and maintenance responsibility change at the next month boundary; current-month access fees belong to the prior owner. The sale preserves all open-access rights and the line's capacity-allocation status. Offline uses the same mechanism without fictional rival bids, so a sole qualifying bid clears at reserve. A transaction history explains listing, reserve, winner, clearing price, and ownership change. Losing maximum bids stay private.

### Building a corridor

Select an origin and destination city. The planner offers up to three materially different surveyed alternatives, such as lower-cost surface routing, faster direct routing, or an engineered crossing. Show:

- Route length, achievable speed profile, and indicative end-to-end time.
- Surface, mountain, bridge, tunnel, and urban-approach components.
- Total construction cost, payment timing, completion month, and maintenance burden.
- Required stations, concessions, and prerequisite technology.
- Forecast passenger opportunity and the difference between corridor and company-wide effects.

#### Independent city-to-city lines

**Neither endpoint must already have a station, track, or a connection to an existing railway.** The planner accepts two distinct eligible cities whether both are unserved, one is served, or both already have rail. Building A–B may create the company's first rail component, a separate component alongside an existing C–D network, an extension, or a link joining components. All are ordinary uses of the same construction command and quote path.

The [two-endpoint planning interaction](UX.md#two-endpoints-and-a-persistent-connection-draft) keeps editable A/B slots and a persistent draft. [Expanded-map planning](UX.md#expanded-map-and-full-screen-behavior) retains endpoint/alignments controls and build-cost, monthly-revenue, and operating-contribution estimates while the large inspector is hidden. Full-screen/restore, panning, zooming, and estimate refreshes preserve the draft; only a separate funding review can lead to a financial commitment.

Eligibility depends on a qualified alignment, terrain/crossing feasibility, construction and operating rights, concessions, and funding. It does not depend on graph reachability from the company's home city, another owned station, or the public HSR network. A city being visible in the catalog does not certify that every other city has a feasible alignment to it; the curated corridor model still applies. Catalog selection must include independent pairs and new regional starting opportunities, rather than offering only extensions of legacy HSR.

#### Visible construction and competing routes

Publish the owning company, unordered endpoint pair, alignment, construction status, and expected opening as soon as a multiplayer project is accepted. Broadcast changes to the opening target, completion, cancellation, and any posted access offer. Players see the work in the map and corridor planner, including on reconnect. Drafts, unresolved requests, the owner's financing, and private service plans stay with that company.

Match the pair in either order and show all committed, building, or operating direct lines for it before quoting a rival build. Show whether the line offers access and let players inspect it or plan a service for its opening. A second route on a distinct, independently buildable alignment requires explicit confirmation of the competing line and a refreshed forecast. The forecast includes rival service where published and labels assumptions about service on a line still being built. Construction cannot claim a physical section already committed or owned. Use catalog section and conflict identities to detect overlapping works.

For multiplayer new-line requests, reserve funds on acknowledgment and settle construction claims together at the next month boundary. Evaluate concurrent requests in the published deterministic tie-break order. Recheck the reviewed competition and physical section claims against existing projects and awards earlier in the same resolution. If a rival award changes the reviewed competition, release the affected request's reserved funds and show the accepted project for review. A refreshed request can then confirm competition on a separately buildable alignment.

The quote bundles the new track and any required endpoint station construction or access. Neither station must be built separately before a quote can be requested. List missing concessions, station works, operating costs, and the fleet needed to open the line; let the player finance and commit a complete valid project package. No special connection fee or artificial disconnected-network penalty applies beyond the actual facilities, regional administration, and operating costs already specified.

Train procurement can deliver directly to a standalone railway's operating area at the disclosed cost and lead time. Servicing provision is included in the station/fleet cost model or an explicit local facility allowance; a new component does not need a rail connection to a distant existing depot. Moving an already deployed trainset between disconnected components uses a disclosed off-map relocation cost/lead time, without simulating shipping or requiring the player to build a delivery railway. A trainset remains unavailable for service during delivery/relocation and cannot serve two components at once.

Once its track and stations are operational, A–B can run its own service, attract its own OD passengers, and earn revenue independently of the rest of the company. Company cash, debt, and fleet ownership remain shared. Passenger routing remains strictly physical: A–B and C–D do not provide A–D rail journeys until an operating connection is actually built and valid services cover it. Adding B–C later may join the components and create new through-journey opportunities through the ordinary demand and transfer rules.

The build tool starts from either a map city or city search and allows an unserved origin. Label independent projects clearly and show the prospective component on the map. Do not disable endpoint selection because a city is outside the existing network. Offline and server-side validation enforce the same rule; multiplayer rivals' ownership matters only for assets/rights actually used by the chosen alignment.

The first version uses a **precomputed, versioned corridor catalog** with curated alternatives. Runtime pathfinding is for passenger journeys and services, not global civil-engineering design. This gives the player choices while making geography reviewable and avoiding a coarse world raster masquerading as an engineering model. Future waypoint routing can replace the planner behind the same quote contract.

The [proposed launch shortlist](LAUNCH_CORRIDORS.md) identifies 60 regional and 10 strategic city pairs for alignment authoring and qualification. Check each against the scenario cutoff and existing physical track before adding it to the playable catalog.

Every catalog entry must have an engineering classification for its entire route. Unsupported ocean crossings fail with a specific reason. Short water crossings may be built only through explicit bridge or tunnel alternatives, each with its own cost, duration, and eligibility. Rivers do not automatically prohibit construction, and drawing a straight line between two land cities does not certify a land route.

An ocean intersection in a straight endpoint preview or great-circle baseline invalidates that candidate geometry, not the endpoint pair. For a coastline-detour case such as Kinshasa–Lagos, corridor qualification must consider an overland alignment around the coast, with explicit treatment of rivers, terrain, borders, and urban approaches. Once qualified, offer that alignment as one continuous city-to-city project; geometry waypoints do not require intermediate metro stations or separate player construction commands. Price and time the actual detour, and do not add an ocean-crossing charge for water intersected only by the rejected direct geometry. If the catalog has no qualified alternative, report **no authored route yet**, rather than claiming that the cities cannot be connected overland.

Geodesic distance is a baseline and validation check. Actual alignment length is derived from its surveyed polyline. Running time uses the alignment's section speed profile plus a documented acceleration allowance; it does not divide great-circle distance by train top speed.

### Geographic difficulty and engineering choices

**Real terrain is part of the railway business model.** A short line through a mountain range must not be quoted like an equally long railway across open level land. Each catalog alternative follows real geography and carries a reviewed engineering profile; the profile determines cost, lead time, feasible speed, and infrastructure upkeep. A generic distance surcharge based only on the endpoint cities is insufficient.

| Geographic challenge | Required gameplay consequence | Player choice |
| --- | --- | --- |
| Flat open land and broad valleys | Baseline surface construction where the alignment is otherwise feasible | Build directly or follow a valley |
| Rolling hills | Additional earthworks; less direct alignments may reduce cost | Pay for cuts/embankments or accept a longer route |
| Rugged mountains and escarpments | Expensive earthworks, tunnels, and/or viaducts; a surface route may be infeasible at the requested speed | Use a pass, detour around the range, or fund a direct engineered crossing |
| Rivers, lakes, and straits | Explicit bridge/tunnel approaches and structures, with complete crossing coverage | Choose a crossing location and engineering alternative |
| Floodplains, wetlands, and unstable ground | Authored stabilization, elevated structures, or avoidance requirements | Pay for treatment or route around the difficult ground |
| Dense urban approaches | Land/relocation allowance and constrained approaches; underground/elevated alternatives may be required | Pay for central access or accept a longer/slower approach |
| Authored restricted areas | A published exclusion or mitigation requirement in the scenario data | Choose an eligible alignment; money alone cannot bypass a no-build rule |

Restriction and ground-condition layers are scenario inputs, not claims to reproduce current planning law or a site survey. Heat, frost, flood, or seismic exposure may contribute a documented protection/upkeep allowance when an alignment is authored; random disasters and maintenance micromanagement remain outside the first playable loop.

Mountain difficulty depends on local relief, slopes, and the chosen alignment rather than elevation above sea level alone. A high plateau may permit a relatively straightforward surface route; a short crossing of a steep escarpment may require major structures. Valley-following alternatives should remain meaningful even when they add kilometres.

#### Geometry constrains speed

Every alternative includes a strategic rail elevation/gradient profile, curvature or equivalent reviewed speed limits, and engineering components. A route that cannot satisfy its intended track class must either receive a lower speed limit, use a different alignment, require additional civil works, or be unavailable. Buying a faster train cannot turn a winding mountain pass into a high-speed direct line. Raising a geometric speed limit requires a priced alignment upgrade or replacement, not a free equipment toggle.

Distinguish the ground surface from the rail alignment. A tunnel passes beneath the surface and a viaduct above it; their track gradients follow their authored vertical alignment, not every sampled hill on the ground. The simplified model need not expose civil-engineering tools to the player, but its catalog must not contain impossible discontinuities or unexplained jumps over a barrier. Numerical gradient/curvature limits are versioned game-engineering parameters to select during qualification, not invented real-world universal limits.

#### Regional construction rates — initial researched table

Use model `hsr-construction-2024usd-v1`. The [research appendix](CONSTRUCTION_COSTS.md) and [cost ledger](../data/costs/construction_costs.v1.json) preserve the project evidence, scope distinctions, currency factors, arithmetic, assumptions, and country assignments. They cover real HSR samples and conventional railway projects where comparable HSR evidence is absent. **Every `B` below is an inferred game baseline, not an observed national average or a claimed tender price.**

`B` prices one **route-kilometre of new rural at-grade double-track, electrified 300 km/h railway**, including civil works, track and standard electrical/control systems. It excludes land, urban-specific works, major stations, depots, rolling stock, design/management, contingency, financing, and taxes. Dollar amounts are **constant 2024 USD**. Historical costs with an unclear price base are screened using an explicitly labeled proxy; the research does not silently treat nominal, PPP and year-of-expenditure figures as equivalent.

In the table, **H** means an HSR benchmark informs the prior, **H-plan** means its anchor is a budget/forecast, **C** means an explicit conventional-rail extrapolation, and **X** means a disclosed regional/country transfer. Even H priors require judgment to remove structure and package effects from published totals. Sensitivity bounds are alternative design scenarios, not confidence intervals or random quote ranges.

| Cost profile | `B`, $m/route-km | Sensitivity bounds | Evidence basis | Initial design/management | Initial contingency |
| --- | ---: | ---: | --- | ---: | ---: |
| China | 10 | 8–14 | H: World Bank network samples | 15% | 20% |
| India | 12 | 9–18 | H-plan: Mumbai–Ahmedabad original budget | 15% | 20% |
| Japan | 20 | 14–30 | H-plan: Nishi Kyushu estimate and engineering mix | 15% | 20% |
| Korea / Taiwan analogue | 15 | 11–24 | H/X: Honam phase 1; Taiwan transfer | 15% | 20% |
| Southeast Asia | 14 | 9–23 | H/X: Jakarta–Bandung, China reference; country transfers | 15% | 20% |
| Other South Asia | 12 | 8–21 | X: India/East Africa analogues | 15% | 20% |
| Iberia | 12 | 9–18 | H/X: Spanish audited line; Portugal transfer | 15% | 20% |
| France / Benelux | 14 | 10–22 | H/X: LGV Est and Rhin–Rhône; Benelux transfers | 15% | 20% |
| Central Europe / Italy | 18 | 12–30 | H/X: German/Italian audited lines; adjacent transfers | 15% | 20% |
| United Kingdom | 28 | 18–45 | H: HS1 outturn, with urban/package effects separated | 20% | 25% |
| Turkey / Iran / Iraq / Russia analogue | 14 | 8–27 | X: broad Eurasian analogue, not country-specific estimates | 15% | 25% |
| Gulf | 14 | 9–23 | H/C: Haramain package and Etihad conventional civil award | 15% | 20% |
| North Africa | 10 | 7–17 | H/X: Moroccan HSR budget; other-country transfers | 15% | 20% |
| East Africa | 12 | 8–21 | C: Tanzania–Burundi electrified conventional railway | 15% | 25% |
| West / Central Africa | 14 | 9–25 | X: East Africa extrapolation × `7/6` | 15% | 30% |
| Southern Africa | 16 | 10–27 | X: East Africa extrapolation × `4/3` | 15% | 25% |
| Brazil | 8 | 6–16 | C: FICO greenfield freight contract | 15% | 25% |
| Other Latin America | 14 | 9–24 | C/X: Uruguay reconstruction extrapolation and country transfers | 15% | 25% |
| US / Canada | 30 | 20–50 | H-plan/X: California scope check; Canada transfer | 20% | 25% |
| Australia / New Zealand | 24 | 16–40 | C/X: Inland Rail extrapolation; NZ transfer | 20% | 25% |

All 46 country labels in the supplied metro seed have explicit assignments in the ledger. Examples: Hong Kong uses China's rural baseline with explicit local urban works; Taiwan uses the declared Korea analogue; Mexico uses the Latin America profile rather than the US/Canada profile. Further countries require reviewed assignments. Do not derive rates automatically from metro GDP or assume a missing country is flat/cheap.

Apply a profile where the work physically occurs, splitting cross-border alignments into jurisdictional intervals. Choosing the cheaper endpoint must not price an entire international route. Terrain, city density, special access, remote mobilization, imported equipment and unusual procurement constraints remain explicit quantities or overrides. Generic regional differences do not substitute for the actual alignment, and the research does not establish that a country intrinsically causes a precise cost multiplier.

#### Extrapolating from conventional rail

The conventional-to-HSR method adjusts source scope, single/double track, electrification/control equipment, reconstruction versus greenfield, and high-speed geometry. It retains the source's price/scope uncertainty. The [appendix](CONSTRUCTION_COSTS.md#converting-conventional-rail-into-hsr-priors) gives all working factors. Initial examples, in $m of 2024 USD per route-km:

| Profile | Conventional screening anchor | Modeled HSR baseline before rounding | Adopted `B` |
| --- | ---: | ---: | ---: |
| East Africa | 6.215 | 11.37 | 12 |
| Other Latin America | 3.771 | 13.05 | 14 |
| Brazil | 1.635 | 6.53 | 8 |
| Australia / New Zealand | 13.733 | 23.60 | 24 |

These are explicit design extrapolations, not measured HSR premiums. The adopted point rounds upward to the next $2m/km. A double-track source such as Etihad Package A must not receive an additional single-to-double-track uplift. A program with brownfield works does not justify treating all reuse as cheap. Conventional local labor costs do not make imported railway systems free.

#### Engineering components and railway systems

Replace the draft's coarse whole-package structure multipliers with separate civil/track and electrical/control allowances. Let `I = $3m/route-km` be an **authored common systems allowance** for electrification, train control and communications; it is not a measured global equipment price. The local civil/track portion of the flat baseline is `B − I`.

```text
component_rate = speed_factor × [I + (B − I) × engineering_factor]
direct_line_works = sum(component_length_km × component_rate)
```

| Primary engineering component | Initial factor on `B − I` |
| --- | ---: |
| Flat/valley surface construction | 1.0 |
| Rolling-hill surface construction | 1.5 |
| Rugged mountain surface works | 4.0 |
| Standard bridge or viaduct | 2.5 |
| Ordinary mountain tunnel | 5.0 |
| Major deep/base tunnel | 8.0 |
| Eligible underwater tunnel | 12.0 |

These are initial modeled factors informed by civil-asset comparisons, not universal observed ratios. Standard repetitive viaducts differ from exceptional long-span bridges. Long/deep or unusually difficult crossings receive the appropriate class or an itemized project override, not an unlimited stacking of generic multipliers. Specialized foundations, ventilation and protection equipment are included explicitly in that component's scope or a separate named allowance once. The previous `6 × B` viaduct and `12 × B` ordinary-tunnel defaults are superseded.

For the same feasible alignment, initial speed factors are `0.9` at 250 km/h, `1.0` at 300 km/h, and `1.1` at 350 km/h. These modest equipment/standard effects are design assumptions; the larger cost of sustaining higher speed comes from geometry forcing more expensive engineering quantities. Do not apply an all-in empirical speed premium and then charge for the same additional tunnels again.

Partition route length into non-overlapping intervals. A mountain tunnel replaces the surface class for its kilometre; it is not charged as flat track plus mountain works plus tunnel. The standard systems allowance appears once per route-km. Existing lines retain their engineering inventory, so replacement-value calculations and civil upgrades use the same model without billing the owner for already built track.

Calculate the complete quote per jurisdictional subtotal:

```text
W = direct_line_works + stations + depots + explicit_special_works
D = design_management_fraction × W
P = land_acquisition_and_relocation
C = contingency_fraction × (W + D + P)
construction_quote = W + D + P + C
```

Urban civil works must be represented in `direct_line_works` or named special works once; `P` covers property/relocation rather than duplicating the same tunnelling. Profiles' indirect/reserve fractions are initial scenario parameters, not measured country risk probabilities. Apply them once. Trainsets and financing are separate purchases/cash flows. Existing fixed-price quotes cannot receive another contingency layer, and historical P65/contingency-inclusive totals are comparison checks rather than additional quote line items. Missing engineering data cannot fall back silently to the flat rate.

Construction lead time uses an authored work schedule with stated parallelism and bottlenecks. A long tunnel can dominate the completion date even if it is a minority of the route length. It is not enough to apply the same months-per-kilometre rule to every engineering class. Upkeep likewise uses the owned engineering component inventory, so a costly crossing remains a long-term business commitment. All of these effects appear in the quote before commitment.

#### Worked strategic choice

For a fictional pair of endpoints using the France/Benelux prior `B = $14m/km`, `I = $3m/km`, and the 300 km/h class where geometrically feasible, compare direct line works before station, land, indirect and contingency additions:

| Alternative | Engineering breakdown | Length | Direct line-works cost | Strategic effect |
| --- | --- | ---: | ---: | --- |
| Valley detour | 300 km flat/valley surface | 300 km | $4.20bn | Least expensive; longest distance |
| Mountain pass | 180 km flat + 60 km rugged surface | 240 km | $5.34bn | Shorter, but its curves/gradients can constrain speed |
| Direct base tunnel | 180 km flat + 40 km major base tunnel | 220 km | $6.16bn | Potentially the fastest, with higher capital cost, upkeep, and a longer construction bottleneck |

This illustrates the intended decision rather than describing a particular real-world route. The tunnel must earn its additional cost through a better transport offer or useful network connections; it must not be automatically optimal because it is shortest. Quoted running times come from the actual alternative's speed profile, not this illustrative table.

#### Map and quote feedback

Show a terrain/engineering overlay, named barriers, and a compact elevation profile when comparing alignments. Color and label surface works, tunnels, bridges, and viaducts separately. The cost breakdown identifies kilometres and cost by engineering type, the main construction bottleneck, maintenance burden, and any speed constraint. Selecting a costly component should highlight it on the map and explain the alternatives. A blocked option names the barrier and missing eligible crossing rather than returning a generic build failure.

Existing HSR retains its reviewed real route and engineering inventory. Terrain does not charge the player to rebuild already operational public track. It affects upkeep, future civil works, and the replacement-value component of auction reserves; a line's acquisition price remains governed by the auction rules. Every client uses the same pinned engineering data, and the server recomputes quotes and eligibility rather than trusting a client-drawn route.

### Costs and delivery

All monetary values use one constant-price game currency, displayed as dollars, with **2024 USD** as its price-year reference. A scenario may begin in 2025 without changing this fixed monetary base. There is no currency speculation or inflation in the first version. The researched construction model is a source-informed balance dataset and is explicitly distinguished from real construction quotations. Its price/proxy assumptions and version identity accompany the data.

An alignment's price is the sum of its mutually exclusive engineering components, stations, and stated contingency. A tunnel kilometre is not charged again as a surface kilometre. Country/region and urban approach modifiers have explicit application rules to prevent stacking the same cost twice.

Initial delivery target: ordinary projects take roughly 12–48 months; major engineered crossings may take longer. Balance must allow a useful opening within the first few minutes of play through a starter project, starter asset, or accelerated introductory scenario. The main campaign should not require dozens of empty turns before the first train runs.

Accepting a project locks its quote and reserves the entire construction budget. Reserved money remains in the bank but is unavailable for other spending. Each month, a scheduled installment reduces cash and the reservation by the same amount and increases construction-in-progress. Cancellation releases unspent reservations and writes off completed work; it creates no refund of already spent cash. A grant must declare whether its unpaid portion is guaranteed before it can fund a reservation.

Completed infrastructure becomes operational at the beginning of the following business month. Capacity upgrades reuse existing track geometry and publish whether service is disrupted during work. Capacity increases require paid upgrades; duplicate overlapping links cannot manufacture free capacity.

## 5. Stations, trains, and services

### Stations

The first version has one strategic station facility per city, with three capacity levels. A station provides market access, daily train-call capacity, a fixed local access-time allowance, and a monthly operating cost. Each facility has a public or company owner, and every facility is open access. Any eligible company with the relevant concession obtains station access at regulated per-call fees published in the scenario rules. The owner cannot refuse access or raise the fee. It receives the fees for other companies' calls, and its own calls make no cash payment to itself. Access rights are separate company records, so companies using one station share its call capacity under the joint allocation rather than receiving duplicate platforms. Building the first station in a city earns fee income and the facility's value, not an exclusive market. Upgrades increase train calls and transfer convenience. Multiple stations in the same metro are aggregated deliberately, with any connecting time retained; incompatible disconnected lines must not acquire a free instantaneous connection through aggregation.

A train stopping at an intermediate city consumes a station call there. Through services consume track capacity even where they do not stop. Capacity is directional for track and total daily calls for stations; the UI names the unit in both cases.

### Fleet

Use three fictional high-speed train families initially, differentiated by seats, maximum speed, purchase price, energy use per train-kilometre, and maintenance cost. These are gameplay equipment profiles, not manufacturer specifications. Avoid collecting dozens of near-identical train types.

Trains belong to the company, with distinct available, assigned, in-delivery, relocating, and retired/sold counts. Procurement has a disclosed delivery time and destination operating area. A company buys trainsets outright or, where the scenario publishes lease terms, leases them for a fixed monthly payment and term. Leased sets cannot be sold, and lease payments are mandatory obligations. A purchase order reserves its unpaid installments in unrestricted cash, like construction, and each installment draws down that reservation. Persist delivery/relocation commitments and their remaining time so saves and server restarts cannot make unavailable fleet appear early. A trainset is assigned to at most one service; spare trainsets are visible. Selling trains returns a published depreciated value and cannot leave an active service under-resourced without an explicit reduction or suspension.

Fleet requirements use round-trip running time, terminal turnaround, chosen frequency, and a maintenance reserve. With `f` daily departures in each direction over operating window `W`, the strategic planning rule is:

```text
headway_minutes = W / f
cycle_minutes = outbound_time + inbound_time + both_terminal_turnarounds
working_sets = ceil(cycle_minutes / headway_minutes)
required_sets = working_sets + ceil(working_sets × reserve_fraction)
```

This is a conservative fleet-planning abstraction, not a detailed rolling-stock diagram. Train speed is limited by both equipment and track. A faster fleet does not raise a section's infrastructure speed limit.

### Service editor

Choose terminal cities, a valid operational path, stopping cities, train family and trainsets, departures per direction per day, and a fare per passenger-kilometre. The fare applies to distance actually ridden on that service. Initial services are symmetric out-and-back patterns; directional and peak/off-peak scheduling are deferred.

Before Apply, show running time, required fleet, track and station usage, monthly operating cost, indicative load, and marginal company cash flow. Reject disconnected paths, repeated-city loops, insufficient fleet, zero/negative frequency, invalid fares, and overbooking of company-controlled capacity. For shared public capacity, validate the request and label the final allocation provisional until the common monthly resolution. Return a specific constraint and the changes that would resolve it.

Track capacity measures scheduled departures, not passenger load. A 10%-full train consumes the same train path as a full train. Empty scheduled trains still incur operating costs. A service may remain suspended without being deleted, but contributes no transport offer, fare revenue, or variable train-kilometres.

Research and progression initially expose a small set of station, track, and equipment upgrades. Prefer a published certification investment and lead time to a sprawling technology tree. Cross-border compatibility is abstracted through the concession; multiple voltages and track gauges are deferred.

## 6. Passengers and competing modes

### Demand has a bounded origin

Generate directed monthly origin–destination demand for business and leisure travelers. Every passenger request has one origin, one destination, and one class. A metro has a finite outbound trip budget; adding rail services does not independently generate a full new market for each line.

Replace the draft's raw population/GDP product with a normalized, calibratable model:

```text
outbound_trips[i,c] = population[i] × monthly_trip_rate[c]
                      × origin_affordability[i,c] × seasonality[i,c,m]

destination_weight[i,j,c] = attractiveness[j,c]
                            / (1 + distance_km[i,j] / distance_scale[c])^decay[c]

demand[i,j,c] = outbound_trips[i,c] × destination_weight[i,j,c]
               / sum(destination_weight[i,k,c], k != i)
```

Attractiveness uses population plus bounded authored business/tourism weights. Initial affordability is a documented regional game index. The supplied metro seed also provides estimated `gdp_usd_billion`, retained for economic inspection and calibration. Once GDP basis, reference years, and matching metropolitan coverage are qualified, GDP per resident (`gdp_usd_billion × 1e9 / population`) may inform a bounded relative affordability or business-attractiveness index. Aggregate GDP is not household disposable income, and the supplied currency label does not establish nominal versus PPP comparability. Until qualified, keep a declared game-index fallback; do not silently treat the estimates as precise 2025 facts or reintroduce an unbounded population-times-GDP multiplier. Preserve the units of people, trips per month, kilometres, and money throughout the model.

The first version keeps metro populations fixed during a campaign and uses a bounded scenario-level demand trend. Rail-induced urban growth is deferred, so a feedback loop cannot create unlimited passengers. Seasonality is repeatable and class-dependent; it is not a fresh random multiplier every query.

### Mode and itinerary choice

Passengers compare rail with abstract road, air, and conventional-rail options plus a no-trip/outside option. Each non-player option has explicit eligibility, door-to-door time, and price rules. Road requires a valid road-region relationship, not merely two points on the map. Air includes access and terminal time. Conventional rail exists only for authored eligible pairs. These are scenario offers rather than a live timetable database.

For each class, generalized journey cost combines fare, in-vehicle time, access/egress, expected waiting, transfer penalties, and reliability. Business travelers have a higher value of time; leisure travelers are more price-sensitive. Initial rail routing allows up to two transfers. Transfers require an operating station shared by the services; merely passing through without a stop does not qualify.

Use a frequency-based expected wait rather than requiring exact schedules. The waiting rule is capped and described as a strategic approximation, especially for low-frequency services. A route can attract through passengers even if its intermediate cities have modest local demand.

Generate a bounded set of distinct rail itineraries and select the best rail offer per company/OD/class before mode and company choice. This prevents duplicate near-identical services from multiplying a company's market share. A smooth, calibrated choice rule allocates the one shared OD demand among those company offers, eligible outside modes, and no-trip. An offline world has just one company offer. Rejected or unavailable modes have zero eligible share, not fabricated zero travel cost.

### Capacity assignment

Assign rail passengers jointly across the network. Each journey consumes one seat on every directed service leg it uses. Loads cannot exceed seats, trains cannot be counted twice, and a transferring passenger counts once as a company journey while producing a boarding on each service used.

The first version uses deterministic proportional rationing across competing OD/class requests with repeated residual allocation. For an oversubscribed leg, apply the same limiting fraction to all requests using it, reserve those complete itineraries, and repeat for remaining capacity. Specify rounding by stable IDs and largest remainders; never privilege the first city in a map iteration.

Unserved rail requests may try one distinct feasible alternate rail itinerary; remaining travelers revert to their original outside-mode proportions or no-trip. Reassignment must consume only remaining seats and may not regenerate OD demand. Persist reports of requested, carried, alternate-served, and rejected passengers.

Monthly seats per directed leg are derived from delivered departures × seats per train × calendar days. In the first release all validated scheduled departures run: spare trainsets provide the maintenance reserve, and do not also reduce seat capacity a second time. Random train-by-train breakdowns are deferred. Capacity reservations use scheduled frequency; any later disruption feature must distinguish scheduled paths from delivered seats.

### Revenue and attribution

One journey produces one total fare, equal to the sum of the fares for its ridden service legs. Allocate revenue to those services by that same sum; a transfer does not create a second full-journey fare. Passenger-kilometres sum actual ridden distance. Load factor is passenger-kilometres divided by seat-kilometres, not an unweighted average of segment percentages.

A line's own profit is not the only test of its value. The planner's incremental forecast compares the whole company with and without the proposed change over the same inputs. Show a branch's direct result separately from its estimated feeder effect.

## 7. Company finance and management

The player combines private infrastructure developer and train operator. Ownership entails infrastructure upkeep even when no trains run. Initial funding combines company capital, bounded loans, and specific public grants for useful projects. Running on public infrastructure incurs access charges rather than treating that track as a free company asset.

The monthly ledger separates:

| Cash flow | Treatment |
| --- | --- |
| Ticket sales | Revenue from carried passengers only |
| Grant receipts | Separate from commercial revenue; never counted twice |
| Train operation | Energy, crew, and variable maintenance per delivered train-kilometre |
| Infrastructure/station upkeep | Fixed cost of owned operational assets |
| Public track/station access | Published usage charge; distinct from upkeep of company-owned assets |
| Access income | Other companies' payments for acquired lines, paid private-track rights, and station calls, reconciled across payer and owner |
| Fleet overhead | Ownership, lease, and maintenance cost, including unassigned trainsets |
| Construction installments and purchases | Capital spending; not operating expenses |
| Borrowing and principal repayment | Financing; not revenue or operating expense |
| Interest | Finance expense and cash payment |

Show operating profit, depreciation, interest, net result, cash change, reserved cash, and unrestricted cash distinctly. Depreciation changes accounting value, not cash. Loan offers publish rate, maturity, repayment schedule, borrowing cap, and the effect on monthly payments. Credit is never an unexplained automatic refill.

Mandatory obligations that cannot be paid from unrestricted funds or an approved credit facility become explicit arrears and trigger restructuring. Do not spend another project's protected reservation or leave the monthly transition half-applied. The report reconciles paid expenses, outstanding obligations, and the steps required to recover.

For an unpaid intercompany access charge, book matching payable/receivable entries rather than crediting the infrastructure owner with cash that the operator did not pay. Cash transfers reconcile exactly across companies; bad debts and caretaker obligations have explicit entries. Sealed bid reservations remain protected until withdrawal, settlement, or a disclosed insolvency rule releases them.

Maintenance is automatic and mandatory in v1. Capacity and fleet upgrades create maintenance tradeoffs; defer manual inspection schedules and wear sliders. Fixed company administration scales with active regions so expansion has a visible cost.

Every grant offer has a named objective, fixed maximum award, payment schedule, deadline, and explicit failure consequences. Grant optimization must not become a larger game than running trains. Start with a small authored catalog, such as access to a currently unserved metro or a useful cross-border connection. In multiplayer, grants draw from a declared shared award pool or an explicitly company-specific program; the same exclusive award cannot be paid independently to every company.

Macroeconomic changes and disruptions are a later extension. The first playable slice should prove the construction–service–demand–cash loop without random crises. No forecast may silently assume future grants or future loans that have not been secured.

## 8. Forecasting and information design

### Forecast contract

A quote is read-only and contains an input world revision, plan identity, and balance/data versions. Committing a stale quote triggers a fresh validation; it never spends the old price against a changed world.

Construction and service previews use the same domain algorithms as the monthly simulation, evaluated on an isolated candidate state. The committed company, RNG, future event sequence, and save are unchanged by forecasting. Cache results by world revision and plan identity.

In multiplayer, forecasts use the player's own draft and rivals' last published operating plans. They cannot read sealed rival submissions. Label the assumption and uncertainty in future shared-capacity allocations; do not present an estimated allocation as a reserved train path.

Show construction cost, opening date, first operating-month ridership and operating result, required fleet, and a finite-horizon cash scenario. Call a payback estimate unavailable when the forecast horizon does not recover the investment. Baseline/upside/downside scenarios use explicit demand assumptions and are not statistical confidence intervals.

### Main screen

Use a cheerful bright visual identity with purposeful construction/train motion and sound, as defined in [PRESENTATION.md](PRESENTATION.md). Animation follows authoritative project progress and resolved operating-day timing; it cannot complete work or advance time. Reduced motion, master mute and separate sound channels preserve full playability.

The detailed [interaction, layout, and onboarding specification](UX.md) defines the screen compositions and task flows. Four stable destinations are Map, Services (including fleet), Company (including finance), and Reports. The map is the principal workplace; Watch operating day is a distinct replay mode within it.

- **Status strip:** business month and unrestricted cash; Advance month offline, or the server deadline and plan acknowledgment state online. Phone branding scrolls away while the compact status remains available. Wider layouts add operating cash flow, runway, and debt.
- **World map:** infrastructure, projects, stations, and selected services, with city search and list-based alternatives to precise picking.
- **Selected-object inspector:** relevant facts and the next action. It is a side panel on desktop and an expandable summary sheet/focused task on phones.
- **Results:** a compact month-close summary and linked detailed reports; a desktop drawer or pinned report can accompany the map, while phones retain a return path between views.
- **Progressive detail:** route comparisons, financial schedules, constraints, and contextual help expand without shrinking essential text or hiding actions.

The railway reference adapts Africa2ice's Field Notes into explanations beside real decisions: why transfers help, why speed gains diminish, why a tunnel dominates cost, and what a reported metric means. Essential numbers and constraint reasons stay visible; longer explanations are optional. Wide displays can pin a read-only comparison beside the active inspector, while small displays present the same decision in sequence.

Map layers cover potential demand, time advantage, occupancy, infrastructure capacity, and construction difficulty. The construction layer shows terrain, barriers, engineering components, and the selected alignment's profile and cost breakdown. Keep demand, carried passengers, and capacity visually distinct. Red is not the only signal: use labels, patterns, and explicit counts.

Selecting a service highlights every track section used and each called station. Selecting a congested section lists every contributing service. A passenger-flow inspector explains one OD market from demand generation through mode choice to seat allocation and revenue.

### Input and accessibility

All consequential actions have visible controls and keyboard equivalents routed through the same intent handler. Provide city search and list-based route selection so dense map picking is not the sole workflow. Escape dismisses one layer; modal dialogs block all underlying input. Destructive financial commitments display the amount and timing in the confirmation action itself.

Support complete portrait-phone play, touch/pointer/keyboard input, resizable windows, DPI-aware rendering, readable scalable text, reduced motion, independent music/effects controls, and no audio-dependent information. Layout reflows according to logical available space and text scale rather than scaling a fixed desktop canvas. Hover, long press, dragging, and pinch are optional conveniences with visible alternatives. Respect safe areas and software keyboards; preserve drafts across rotation and resume. Reduced motion uses static route positions and explicit counts in the operating-day view. Remember presentation preferences separately from campaign rules.

Onboarding happens through ordinary empty states, explainable forecasts, short contextual cues, project receipts, and reports linked to remedies. There is no separate required tutorial or prescribed route sequence. At most one unsolicited learning cue accompanies the active task, with dismissal and optional help. Starting on public track or building an independent line receives equally relevant guidance. Guidance never blocks time or changes simulation rules, prices, assets, or server allocation. See [UX.md](UX.md) for trigger examples and first-time-player acceptance targets.

## 9. Simulation and domain contract

### Aggregate and quantities

Use one `World` aggregate as the atomic consistency boundary. It owns all companies, the shared scenario clock, public and private infrastructure, projects, auctions and bid reservations, fleet ownership, accepted service plans, concessions, grants, and bounded report history. An offline world has one player company; a multiplayer world has several. Cities and the corridor catalog are immutable versioned scenario inputs. Every company-owned object carries a `CompanyID`; every modifying operation checks the acting company's authority.

Use typed identifiers and quantities for money, distance, minutes, seats, passengers, trainsets, departures, and basis points. Store money in checked integer minor units. Fare rates use a finer fixed-point unit per passenger-kilometre, and fare revenue rounds explicitly to cents. Use integer metres and simulation minutes or seconds at defined boundaries. Reject non-finite coordinates and calculations, invalid units, overflows, duplicate IDs, and broken references on construction and restore.

### Monthly resolution order

An accepted state is the beginning of a calendar month. `AdvanceMonth` resolves exactly that month:

1. Freeze its service plan and starting assets. Assets completed in the prior transition are now eligible.
2. Validate fleet assignment, track rights, concessions, and funding reservations; jointly allocate shared station and track capacity to the submitted service requests.
3. Derive delivered service, trip times, capacities, and scheduled operating costs.
4. Generate one shared OD market, compare all companies' offers with outside modes, and allocate complete rail itineraries to seats.
5. Record journey fares, train-kilometres, passenger-kilometres, and service/section results.
6. Post commercial cash flows and reconcile intercompany access payments, upkeep, interest, and debt payments. Pay reserved construction installments and advance existing projects and orders.
7. Settle due auctions, resolve pending construction claims, and apply scheduled access changes. Complete projects, ownership transfers, and deliveries for next month's use. Update asset accounting, concessions, restructuring, and campaign objectives.
8. Publish a reconciled monthly report and operating-day summary, advance the date, and request an autosave.

No mid-month service editing or partially built railway carries passengers. Use actual Gregorian month lengths, including leap years; rates and reports label monthly versus daily units. Resolving a month either produces a complete valid next world or leaves the prior world unchanged.

### Determinism

Given the same scenario and balance versions, initial seed, and accepted command sequence, all supported targets produce the same canonical checkpoints. No domain wall clock, network access, renderer, audio callback, or arbitrary goroutine scheduling influences outcomes.

Sort all consequential iterations and tie-breaks by documented stable keys. Quantize demand/choice outputs before allocation and use explicit rounding and checked integer ledger arithmetic. Cross-target equality is a release gate, not an assumption that floating-point expressions are automatically identical. If candidate formulas fail that gate, use deterministic tables or arithmetic before freezing the algorithm version.

Randomness is optional in the initial slice. When introduced, it has an owned serializable generator or versioned keyed draws by seed/month/entity/purpose. Querying, hovering, forecasting, saving, and watching trains consume no campaign randomness.

## 10. Multiplayer and offline play

### One rules engine, two execution modes

Offline single-player constructs the application service in-process and uses filesystem or IndexedDB storage. New games, forecasting, the operating day, saves, and loading require no login, server process, internet connection, or periodic license check.

Online play connects the same game UI to a remote implementation of the driving port. A headless Go server owns the entire shared `World`; clients submit intentions and receive authorized frames and reports. The server independently validates money, ownership, track rights, fleet, fares, demand, and results. Client-computed forecasts are advisory and client-supplied balances or passenger totals are never authoritative.

The headless server must start without importing Ebitengine, opening a graphics context, or initializing audio. One serialized world coordinator applies accepted operations and resolves months; storage and networking use immutable copies and never mutate the world concurrently. Separate worlds can run independently. This is a turn-based strategy protocol, not a high-frequency train-position stream: clients animate the operating day locally from resolved summaries.

### Persistent schedule and session ownership

First multiplayer target: **2–8 players in a private world**, using invites and a dedicated hostable server. A player may run that server separately on their own machine or deploy it to a server host. The browser can join an HTTPS server; browser-only peer hosting is not required. An official public persistent service and public matchmaking are outside the first release.

The proposed default for new multiplayer worlds is **one game month every 30 real seconds**. A game year takes six minutes, a four-year build takes 24 minutes, and the 30-year campaign takes three hours of running time. Hosts choose a fixed interval at world creation, with longer intervals available for groups that prefer a slower campaign. Offline single-player advances at the player's pace.

Before starting the clock, give the group a planning lobby. Players can inspect the starting map and finances, request quotes and forecasts, and prepare private construction, fleet, and service drafts against the opening state. The host explicitly starts the session when the group is ready. Budget about 10–15 minutes for this setup in a typical session and let the host give the group more time. Cash, reservations, and commitments first change through accepted commands after the start. The first monthly deadline is one full interval after the host starts, and private drafts carry into live play for explicit submission.

1. Every company plans against the same resolved world/month using its own private planning overlay.
2. Apply sends a validated plan revision to the server, which durably acknowledges it. Acknowledgment identifies the first month for which it can take effect.
3. The UI shows the authoritative next deadline and an estimated local countdown. The server's UTC clock controls the boundary; a client's clock, focus, or connectivity cannot accelerate it.
4. At the deadline, the coordinator locks the newest durably acknowledged plan for each company and resolves the month jointly. It never waits for Ready votes and never advances early because everyone is online.
5. Each company receives results immediately or on reconnect. The next deadline is published and the next planning window opens.

Other companies' next-month fares, financing, construction bids, and drafts stay private until their effects are committed. Their currently operating services and published infrastructure are visible. A draft can span several months. Update it against the current state and ask for renewed review when the decision's terms or eligibility change. A request that arrives after the deadline lock receives a closed-month rejection. The client keeps the draft and may resubmit it explicitly for the following planning window. Never silently apply an old command to a different month. Private-plan revisions do not invalidate an unrelated company's draft merely because that player clicked Apply.

### Standing plans and absent players

When a player does nothing, the last accepted service plan continues, projects and train orders progress, and scheduled debt/maintenance payments remain due. This applies equally to connected and disconnected companies. Unapplied local drafts disappear on disconnect unless restored locally; they never become server orders. Reconnecting restores the company's acknowledged plan, current date, current finances, and missed reports.

Provide a cash-runway estimate, pending bid/construction reservations, upcoming obligations, and an explicit **Leave company running** summary. Proposed standing safeguards allow a player to set a minimum unrestricted cash buffer, cancel a still-withdrawable bid that would breach it, or permit draws only from an already approved credit facility. Policies take effect only when enabled and acknowledged; there is no automatic new loan, asset purchase, or grant application. Disclose any automatic service reduction rule before it can act. Normal default is to continue the accepted plan and report its consequences.

The client has an activity log and overdue-action badges. Show consequential deadlines in business months and real time. Online auctions have a server bidding window of the longer of three real minutes and one interval, and a separate settlement month. Explicit host pause freezes the bidding countdown and resume restores the remaining time. After an outage, seal auctions whose persisted bidding deadline has elapsed and settle them on the next eligible resolution. For restructuring, retain the launch playtest target of at least **five real minutes**. At world creation, round that minimum up to whole months at the configured interval and take the larger of that count and the baseline business-month window. The default pace gives ten months for restructuring.

Notices that can suspend another company's services get the same conversion with a longer floor. Ending a track-access right, lowering a shared path quota, and starting capacity-affecting works need at least the **minimum notice**. The minimum notice is the larger of the months spanning 30 real minutes and 32 business months, fixed at world creation. That gives 60 months at the default pace and 32 months (eight hours) at a 15-minute interval, matching the unattended targets below. A longer scenario notice period still applies. Offline worlds have no rival rights, so the floor does not apply there.

A stable company should be able to run for at least 30 real minutes unattended in the standard session, equivalent to 60 business months. For a 15-minute host setting, retain the eight-hour unattended target. Test construction schedules, debt terms, auctions, and maintenance against both cases.

A bankrupt company stops operating and may remain as a spectator; its failure does not end the shared world. Its services release shared capacity. Keep its private assets and liabilities in a neutral inactive estate; no automatic gift to another player. The estate's operational track and stations keep running under a public caretaker, with the ownership liability recorded, so insolvency cannot strand rivals who depend on them. This covers acquired legacy lines, private lines, and owned stations. Acquired lines and all stations keep regulated open access. Accepted private-track rights continue at their pinned fees with the existing shared path quota, and the offer closes to new operators. The caretaker pays upkeep from the estate's access income and records any shortfall as an estate liability. Private estate auctions and player mergers are deferred.

The host can explicitly pause or end a private world and remove a departed player under disclosed lobby rules. Pause, resume, removal, and schedule changes appear in the world activity log. Removing a company follows the bankruptcy path. Its services stop and release capacity, and its assets and liabilities pass to an inactive estate under the same caretaker rules. Nothing transfers to the host. Sleeping or stopping a locally hosted server interrupts persistence; offer deployment instructions for an always-on machine if uninterrupted progression is desired.

### Server outage policy

Client absence and server absence are different. Proposed safe initial policy: on restart, resume any already committed resolution idempotently; if a deadline passed while the server was unavailable, resolve **at most one** overdue month using acknowledged standing plans, then schedule the next month one full interval after recovery. Do not silently replay hours or days of missed months and bankrupt every company before it can reconnect. Record the outage and adjusted deadline in the world log. This is an explicit exception to the normal running schedule, not an advance made by a client.

Store the next UTC deadline, world phase, schedule interval, and resolution identity durably. Server time moving backward cannot duplicate a month, and a large forward jump enters the same bounded recovery policy. The host cannot edit simulation time by supplying client timestamps. Before release, a restart test must prove the chosen downtime rule and deadline display on both native and browser clients.

### Shared markets and fair capacity

Generate OD demand once per world/month, then allocate it across companies and outside modes. For each OD/class, each company contributes at most one best rail offer to mode/company choice; creating duplicate services does not give that company extra choice weight. If equal offers have equal cost, shares and integer remainders use stable fair rules, independent of packet arrival order.

The first multiplayer version proposes same-company transfers only. A journey may use several services owned by its chosen company, but cannot rely on a rival's trains without a future explicit interline/ticket-sharing feature. Public track access is separate from passenger ticket sharing.

Private company fleet and infrastructure cannot be double-assigned or commandeered. Public infrastructure, acquired open-access lines, private sections with accepted paid rights, and shared stations have finite shared capacity. A service preview distinguishes **requested frequency** from **allocated frequency**. An access right makes a service eligible to request capacity. The joint resolution determines the final allocation.

Proposed allocation rule: use deterministic max-min fair allocation by company, then by service, subject to every section and station constraint along each complete service path. Allocate whole daily return pairs and expose requested, granted, and denied frequency with bottleneck reasons. Rotate indivisible remainders by month and stable company identity. Adding many duplicate services cannot win a larger company-level share. New capacity competes under the same published policy; first network arrival receives no priority. On a private section with accepted rights, the rule applies within the shared path quota.

Required fleet is checked against requested frequency. If final granted frequency is smaller, its unused trainsets remain assigned but idle and reported as such. A service granted zero departures is suspended for that month and earns no fare revenue or variable running cost. Fixed ownership, lease, and financial obligations still apply. Demand and cash are computed only after shared capacity is finalized. Owner-only track retains its owner's service allocation; it is never included in the public pool accidentally. Likewise, an owner's capacity outside the shared path quota is never allocated to operators.

The exact fair-allocation algorithm and rounding require worked shared-bottleneck examples before implementation. Open-access capacity is reallocated monthly in this first proposal; line acquisition does not grant an ownership priority. There are no permanent free path reservations or latency-based bidding wars. Limited grants, closing auctions, and mutually exclusive projects resolve at the joint boundary using published scenario rules, never client click order.

### Protocol, identity, and recovery

Use a versioned HTTPS command API plus resumable world-update notifications or bounded polling. Final transport selection follows a native/browser networking spike; WebSockets are optional, not a requirement for train animation. Requests carry a world ID, authenticated session, company identity derived by the server, turn/phase identity, company-plan revision, and idempotency key. Stale private plans return a conflict with enough information to refresh safely.

The protocol distinguishes **plan revision acknowledged**, **deadline locked**, **month committed**, and **result received**. A retried request cannot buy a train twice, issue duplicate debt, or advance a month twice. Slow or disconnected clients may miss notifications and then fetch the latest authorized snapshot. A client with uncertain acknowledgment queries operation status rather than assuming failure or resubmitting with a new identity.

Persist a committed month atomically before broadcasting its results. Durable commit includes world state and completion identity, so a crash after commit but before response cannot duplicate the resolution. On restart, load the latest complete checkpoint and acknowledged planning records; never load half the companies from one month and half from another. Backups and host export/import are whole-world operations, not player-local overwrites.

Players can inspect public networks, committed rival construction, posted track-access offers, scheduled services, fares, and capacity. Rivals receive public summaries. Drafts, full ledgers, and accepted access agreements are visible only to the parties authorized to see them. Apply authorization to forecasts and passenger inspectors as well as mutation endpoints. Bound request sizes, command rates, expensive forecasts, and join attempts. Invite codes admit players to private games, and revocable authenticated sessions identify subsequent requests. Server operators inherently control their hosted worlds.

Multiplayer saves are server-owned. A downloaded client cache cannot overwrite a shared world or spend online funds while offline. Offline games may be exported and deliberately imported by a host as a new world; that is not a claim of verified competitive history. Reconnecting a company uses its existing authorized identity and does not create a new grant or starting-capital package.

## 11. Architecture, informed by Africa2ice

See [ARCHITECTURE.md](../ARCHITECTURE.md) for the implementation boundaries and their current status. Gameplay rules and invariants remain defined in this design.

The implementation adopts Clean Architecture, DDD, CQRS within one authoritative world, and explicit Go decorators for AOP-style diagnostics. The [architecture-style decisions](../ARCHITECTURE.md#architecture-style) define the scope of each pattern.

Use Go and Ebitengine for the deterministic model and desktop/browser hosts, retaining Africa2ice's separation between game rules and presentation. Pin and smoke-test toolchain versions during implementation; the reference currently declares Go `1.26.4` and Ebitengine `2.10.5`, rather than the Gemini draft's broad Go `1.22+` baseline.

```text
main / main_js
    └── app: composition, input, lifecycle
         ├── local application service (offline)
         ├── remote game-port adapter ──► authoritative headless server (online)
         ├── application ──► domain
         │       └─────────► gameapi
         ├── storage adapter ──► application-owned persistence port
         ├── logging adapter ──► use-case and storage ports
         └── HUD / UI / renderer ──► gameapi frames and intents
```

Proposed layout:

```text
internal/domain/          world, geography, construction, services, fleet,
                          demand, itinerary allocation, finance, monthly resolution
internal/application/     command handling, queries/forecasts, frame projection,
                          save mapping, compatibility, autosave coordination
internal/adapters/        filesystem/IndexedDB storage, remote game port,
                          network API, server persistence, operational logging
internal/verification/    headless player policies and canonical checkpoints
pkg/gameapi/              dependency-free commands, frame DTOs, errors, use-case ports
pkg/app/                  Ebitengine host, composition, platform lifecycle
pkg/hud/                  panels and control-to-intent translation
pkg/ui/                   drafts, selection, scene stack, presentation preferences
pkg/render/               world map, layers, train illustration, viewport transforms
pkg/audio/                music/effects ports and platform output
cmd/server/               headless authoritative server composition and lifecycle
data/                     versioned cities, corridors, scenarios, balance manifests
tools/                    offline data preparation, release and browser checks
```

The domain uses concrete entities and pure algorithms. Persist the world as a consistent snapshot; do not introduce one repository per city, track, or train. Split commands from queries in the API without separate databases, distributed messaging, or event sourcing. Logging decorators implement the useful part of the draft's AOP proposal without adding a generic middleware framework to every calculation.

Example driving operations: `NewCampaign`, `Apply`, offline `AdvanceMonth`, `Snapshot`, `QuoteConstruction`, `ForecastServicePlan`, `InspectPassengerFlow`, local/admin `StateHash`, and asynchronous storage operations. Company commands include:

- Construction: `CommitProject` and `CancelProject` (new lines and alignment, capacity, or station upgrades), plus `UpgradeStation`.
- Access: `PublishTrackAccessOffer` (also republishes revised terms), `CloseTrackAccessOffer`, `AcceptTrackAccessOffer`, `EndTrackAccessAgreement`, `RequestStationAccess` (regulated terms at any station), and `ApplyForConcession`.
- Fleet: `OrderTrainsets`, `LeaseTrainsets`, `RelocateTrainsets`, and `SellTrainsets`.
- Services: `SetService` and `SuspendService`. Standing safeguards are part of the applied company plan.
- Finance and auctions: `AcceptLoan`, `AcceptGrant`, `NominateLineAuction`, `SetMaximumBid`, and `WithdrawBid`.

World operations are `CreateCompany`, `StartWorld`, `PauseWorld`, `ResumeWorld`, `EndWorld`, and `RemoveCompany`. They require a host session or server administration and never act through a player company's gameplay authority. The server scheduler owns online month advancement. Full-world hashes and snapshots are available to local or admin callers.

Only the application translates public commands to domain operations and domain state to frame DTOs. Presentation never holds a mutable `World` pointer. Frames are isolated snapshots with world/network revisions for caching and company-specific visibility. The local and remote ports expose the same gameplay semantics, while multiplayer adds sessions, acknowledged plan revisions, schedules, and operation-status queries. Commands return stable error codes and structured constraint details; UI strings and logging stay outside domain algorithms.

### What to reuse from Africa2ice

| Reference | Carry forward | Adapt or replace |
| --- | --- | --- |
| `pkg/gameapi`, `internal/application` | Small driving port, isolated frames, atomic commands | Rail-specific commands, reports, and forecasts |
| `internal/domain` | Pure aggregate, typed quantities, stable iteration | Entire ecological model and turn rules |
| `internal/adapters/storage` | Asynchronous saves, commit semantics, single-writer handling | Rail schema, namespace, data compatibility |
| `internal/adapters/logging` | Port decorators and local diagnostic sinks | Rail-specific outcome fields |
| `pkg/ui`, `pkg/hud`, `pkg/app` | Shared intents, draft lifecycle, scene ownership, contextual explanation/report patterns | Band checklist, sequential first-turn guide, fixed desktop panel composition, and migration workflows |
| `pkg/render` | Consistent drawing/picking transforms, DPI discipline, and bounded caching | Fixed 1280×720 presentation, Eurasian tile grid, fog, biome simulation, and camp scenes |
| `internal/verification`, CI tooling | Public-API-driven policies, canonical target comparison | Rail policies, limits, and performance workloads |

Use the reference as a source of proven patterns and selectively reusable components, not as an import dependency or an unreviewed whole-repository fork. Preserve applicable source notices when copying code. Do not carry over Africa2ice's numerical performance budgets as evidence for this game's workload.

Responsive layout, phone browser lifecycle, accessible management controls, and software-keyboard text entry require an early host feasibility spike. Ebitengine remains the proposed renderer/core host, but a canvas HUD is not evidence of screen-reader support. Qualify the native bridge and an accessible browser control layer; if necessary use a semantic HTML browser HUD around the shared Go/WebAssembly port. This decision and mobile performance measurements precede extensive form implementation; see [UX implementation constraints](UX.md#9-implementation-constraints-and-early-feasibility-work).

An [isolated ebitenui 0.7.3 / Ebitengine 2.10.5 spike](../tools/ebitenui-spike/README.md) now demonstrates the service/report style in a real Go/WebAssembly renderer. Focused checks cover responsive layout, editing, validation, scrolling, high DPI, and touch-emulated input; the native macOS target compiles. The [evidence report](validation/README.md) records the exact scope. The canvas does not supply the required semantic management controls, and physical phone/lifecycle qualification remains open. This supports ebitenui as the visual widget candidate; it does not close the accessibility gate or select the final browser HUD architecture.

## 12. Data, persistence, and platform behavior

### World data

Create an offline pipeline that emits compact reviewed game data with stable IDs. Each dataset records source URL, license/attribution, release and reference year, processing revision, and content hash. Validate city identity, metropolitan scope, coordinates, corridor endpoints, alternative geometry, engineering coverage, and required concessions.

#### Supplied metro seed

The user supplied `metropolitan_areas_5m.json` on 2026-11-08. Its [repository snapshot](../data/raw/metropolitan_areas_5m.json) preserves the original bytes and all 96 records. A [sidecar manifest](../data/raw/metropolitan_areas_5m.metadata.json) records its hash, field meanings, structural checks, and unresolved provenance. [Data notes](../data/README.md) explain intended use and reproducible checks.

Each row contains `id`, `name`, `country`, `lat`, `lng`, `population`, and `gdp_usd_billion`. All rows have the required fields, nonempty unique IDs/names after whitespace/case normalization, positive integer populations, positive finite GDP values, and finite in-range coordinates. These checks establish structural usability; they do not establish geographic accuracy or consistency of estimates. Of the 96 records, 94 have population strictly above 5M; Phoenix is exactly 5M and Brasília is 4.8M. Preserve both exceptions in the raw snapshot. Exclude Brasília from the current playable major-metro catalog for being under 5M; retain Phoenix at exactly 5M.

Use the supplied coordinates as representative map markers, with latitude and longitude kept explicit. They do not define station locations, metropolitan boundaries, or construction endpoints. The raw `country` strings are source labels, not a normalized country-code or concession catalog. Preserve raw IDs, map any future normalized identifiers explicitly, and avoid deriving identity from sort order or population rank.

The file supplies no population/GDP reference years, upstream citations, GDP price/basis metadata, coordinate reference-system declaration, or redistribution terms. Track those as unknown rather than inventing metadata from the receipt date. Some labels describe broad regions—for example Guangzhou (Pearl River Delta), Randstad, and Rhine-Ruhr—while other rows name nearby cities. Review potential boundary overlap and match GDP/population coverage before summing markets or dividing broad regions into multiple station cities. This is a qualification issue to investigate, not evidence that specific rows are duplicates.

#### Supplementary geography sources

Candidate basemap: [Natural Earth](https://www.naturalearthdata.com/about/terms-of-use/), whose vector and raster data are public domain. It supplies cartographic context, not engineering-grade terrain or automatic clearance for a crossing.

A [content-addressed 1:110m prototype snapshot](../data/prototypes/natural-earth-110m.metadata.json) now supports the Equal Earth mockup. It retains country geometry and names, with source/content hashes and processing notes. This does not qualify production regional detail, terrain costs, or railway topology. [World/regional evidence](validation/README.md) confirms use of the 94 playable metros derived from the 96-record source without altering their estimates. The playable selection policy records the Manila scope exclusion and Brasília population exclusion.

Candidate elevation input: [Copernicus DEM GLO-90](https://dataspace.copernicus.eu/explore-data/data-collections/copernicus-contributing-missions/collections-description/COP-DEM), a global 90-metre digital surface model. The provider states that it includes buildings, infrastructure, and vegetation, so surface samples are not automatically a bare-earth rail alignment. Use a pinned release to inform offline terrain classification and corridor review; no elevation product has been imported or qualified for the game yet. Record vertical units/datum, coverage, void handling, and applicable attribution/redistribution terms alongside the processed version.

The offline corridor pipeline combines terrain profiles with water/shoreline geometry and authored urban, ground-condition, and restriction data. It emits compact engineering intervals, rail profiles, named barrier crossings, speed limits, schedules, and maintenance quantities for every alternative. Review catalog crossings explicitly; elevation by itself does not identify ocean depth, rock conditions, bridge clearances, or legal buildability. Missing terrain/water coverage marks an alternative unqualified, rather than treating a void or missing tile as zero-elevation land. Ship reviewed profiles and map-level terrain detail sufficient for offline play instead of loading global full-resolution elevation rasters at runtime.

Existing HSR requires an additional source-backed inventory. The [UIC High-Speed Data and Atlas](https://uic.org/activities/high-speed-data-and-atlas) distinguishes infrastructure in operation, under construction, and planned, and is a candidate verification reference. Public download availability does not establish permission to redistribute its database or maps; establish dataset-specific reuse terms before bundling material. [OpenStreetMap](https://www.openstreetmap.org/copyright) is a candidate topology source with ODbL attribution and database-sharing requirements. Qualify a source and its redistribution path before importing; a map screenshot is not the game graph. Preserve the source cutoff and document deliberate strategic simplifications of junctions, intermediate stations, and connectivity.

Admit a starting HSR corridor only with a named line, section geometry, and dated evidence of qualifying commercial passenger service at the scenario cutoff.

**A starting corridor must join two cities in the game catalog.**
- Intermediate stops are catalog cities, and branch points are explicit junctions.
- Track tags such as OSM `highspeed` or speed limits set each section's speed and track class. They never admit track on their own.
- Where a slower approach breaks the tagged geometry, for example into a terminal, close the gap with a lower-speed section instead of splitting the line.
- A qualifying passenger line whose cities are missing from the catalog is a catalog question first: add those cities, or leave the line out.
- Track that is not on an admitted corridor is not a starting asset. This includes isolated upgraded stretches, bypasses, test track, and extract fragments.

Passenger demand exists only between cities, so track outside such a corridor could carry no one and would only add upkeep or a misleading auction listing. Distinguish dedicated high-speed lines from upgraded conventional corridors and their documented lower-speed approaches. Record station calls separately from nearby stations. OSM tags and speed limits identify review candidates. Confirm operating status and scope with the UIC atlas, infrastructure manager, or operator before publishing them as starting assets. The [preview audit](../data/prototypes/hsr-overlay-audit.v1.json) records examples of conventional services, test track, commissioning work, and erroneous speed tags that matched the preliminary extraction.

Candidate supplementary city/population reference: the European Commission's [GHSL Urban Centre Database](https://human-settlement.emergency.copernicus.eu/ucdb2024Overview.php), which provides harmonized urban-centre data. Its [download page](https://human-settlement.emergency.copernicus.eu/download.php?ds=ucdb) specifies acknowledgment of the source. Use it to help qualify the supplied anchors or add smaller cities after aligning definitions, rather than overwriting user estimates silently. Choose and record a common population epoch, inspect its metadata, and resolve mismatched metro definitions before importing. Names/aliases may use [GeoNames](https://www.geonames.org/export/), which requires attribution. These external sources have not been imported or qualified for this game; the supplied metro seed has only the structural checks described above.

Do not mix municipal population for one city with metropolitan population for another without explicit reconciliation. Do not derive precise mountain or tunnel costs from coarse map artwork. Authored corridor engineering and balance weights are labeled game data and reviewed independently of the basemap.

Bundle all runtime scenario inputs. Offline play requires no API key, network connection, live map tiles, real-time GDP feed, or geocoding request. Online sessions pin matching protocol, scenario, and balance identities; joining an incompatible world gives a specific error rather than silently substituting local data.

### Saves

The [state and storage contract](STATE_AND_STORAGE.md) defines compile-checked Go reference types, the complete structural save-v1 schema, a checked example, the server SQL schema, and restore/migration rules. Choose embedded SQLite for the dedicated server, storing whole-world JSON, scheduling state and operation receipts transactionally. This is a selected implementation design; production adapters and full semantic restore validation are still to build. There is no released game-save format to migrate yet.

Use versioned JSON snapshots behind an application-owned persistence port, with filesystem and IndexedDB adapters. Include scenario/data hashes, balance version, algorithm versions, date, company ledgers and commitments, infrastructure ownership/access status, auctions and protected bids, projects, fleet, services, RNG state if used, and enough of the latest report to restore its operating-day view. Server storage additionally owns schedule/phase records, acknowledged planning revisions, and resolution identities; authentication credentials are stored separately and never exposed through a player save download.

Manual checkpoints, quick save, and three rolling autosaves follow the same wire format. Autosave after each successful month, serialize writes, and announce completion only after durable commit. Failed saves preserve the previously committed slot and show a recoverable error. Browser users can export/import a portable save because browser storage may not be permanent.

Load into an isolated replacement world and validate fully before swapping it in. Unsupported schemas, unknown data versions, invalid references, and corrupt saves receive specific errors. Migrations must be explicit; never silently reinterpret a saved track layout against changed city or corridor IDs. Either retain the referenced released data bundle or provide a verified migration.

Multi-tab browser play has one storage writer; other tabs remain visibly read-only until ownership is safely transferred. Storage errors do not corrupt the current in-memory campaign. UI selection, map zoom, sound volume, and reduced motion live in separate preferences; difficulty and financial policy remain campaign data.

### Platforms and delivery

First targets: macOS arm64, Windows amd64, Linux amd64, and desktop plus smartphone browsers via WebAssembly. Complete core gameplay must reflow to a 320 CSS px width, with portrait and landscape support. Native phone-app packaging is a separate delivery decision. Start phone layout, touch/text entry, background/resume, and storage qualification in the first milestone rather than treating them as late polish. Larger displays add useful map area and optional simultaneous comparison/report views. See [UX.md](UX.md) for proposed layout bands and accessibility/validation requirements.

- For Flatpak, install the app icon at `/app/share/icons/hicolor/scalable/apps/<APP_ID>.svg`. Use the Flatpak application ID as the desktop entry's `Icon` value. [App icon SVG](../assets/branding/app-icon.svg)
- For the browser build, place `favicon.svg` in the web build's public root, serve it as `image/svg+xml`, and reference it from the document head with `rel="icon"`. [Favicon SVG](../assets/branding/favicon.svg)

Desktop and web share rules, content versions, saves, and checkpoints. Browser execution yields between bounded calculation stages so forecasts and monthly resolution do not freeze controls indefinitely. Native and browser builds must not depend on divergent domain code paths.

Pervasive diagnostics follow [DIAGNOSTICS.md](DIAGNOSTICS.md): correlate intents, validation, authoritative commits, construction/fleet changes, monthly phases, storage, networking and presentation failures. Keep rolling local diagnostics bounded; persist canonical game requests and audit events with authoritative server commits for after-the-fact investigation. Never log credentials or expose rival-private plans to players. No gameplay telemetry is uploaded by default. Audio begins after browser interaction and respects focus, mute, volume, and reduced-motion preferences where relevant.

### Localization

Ship English and Simplified Chinese (`zh-Hans`) in the worldwide release on desktop and web. The translation covers the complete player interface, including settings, map labels and search, contextual onboarding, construction and service planning, monthly reports, multiplayer notices, accessibility labels, and errors. Have fluent reviewers check railway and finance terminology in context. [W3C language tag guidance](https://www.w3.org/International/quicktips/doc/checker.en.php)

Bundle the translations and Chinese fonts for offline use. Format dates, numbers, and currency amounts for the selected language while preserving the game calendar, units, and settlement currency. Players in the same world can use different languages. Language changes affect presentation and search labels only. [Language selection and Chinese input](UX.md#language-and-chinese-text)

## 13. Verification and performance

See [TESTING.md](../TESTING.md) for current runnable checks, evidence limits, local verification policy, and planned CI/release gates.

### Correctness properties

- Geodesic calculations handle identical points, poles, antimeridian crossings, and near-antipodes; distances are finite, nonnegative, and symmetric.
- Every operational section has paid completed infrastructure; every accepted service has a connected path, eligible stations, sufficient fleet, and capacity.
- Every starting section lies on an admitted corridor whose two ends are catalog cities; no starting track is unreachable from a city.
- The world and a company's infrastructure graph may contain multiple components. A valid standalone construction project is not rejected for lacking a path to existing infrastructure; each individual service still requires a physically connected operational path.
- Every buildable alignment has complete, non-overlapping engineering intervals whose lengths reconcile with its geometry; water and mountain barriers have explicit treatments, and unqualified terrain cannot receive a flat-land default.
- Quotes, lead times, upkeep, speed limits, upgrades, and replacement valuations use the same versioned engineering inventory without double-charging overlapping terrain/structure classifications.
- Regional construction rates cover every scenario jurisdiction explicitly; cross-border sections use their own location's rate, double-track route length is not doubled, common systems are included once, and indirect/contingency charges reconcile against their specified bases.
- Ridden passengers never exceed OD requests; complete itineraries consume seats on every leg.
- Through passengers count once as journeys, while service boardings and revenue attribution reconcile.
- Total origin trips remain bounded; duplicate services do not duplicate market demand.
- Track/station capacities, trainset assignments, reservations, debt, cash, and accounting values reconcile exactly at their defined precision.
- Failed commands and forecasts leave the world hash unchanged; report/frame mutation cannot change the world.
- Save/reload continuation matches uninterrupted play; incompatible or corrupt loads preserve the live campaign.
- The operating-day view and reduced-motion mode cannot affect monthly results.

Use focused example tests, Go fuzz/property tests for the mathematical and allocation boundaries, architecture import checks, and scenario policies through the public game API. A third-party property framework is optional. Introduce targeted mutation testing only where it adds evidence for critical boundaries; it is not a prerequisite dependency for the first playable loop.

### End-to-end scenarios

1. Build and finance A–B, wait for completion, assign fleet, open service, earn a reconciled month of revenue, save, reload, and continue.
2. Add B–C and carry A–C through passengers without double fares or excess seats.
3. Add a feeder to a trunk and show its direct and company-wide effects separately.
4. Try an unsupported water crossing, accept an engineered alternative, and verify its price and timing.
5. Increase frequency until the preview identifies a fleet, station, or track constraint; buy the actual remedy.
6. Overextend financially, enter restructuring, recover or finish with an understandable insolvency report.
7. Start a second disconnected regional network and confirm there are no phantom intercontinental rail journeys.
8. Watch/scrub the same operating day at different speeds, then confirm identical next-month checkpoints.
9. Join two companies, submit different fares and overlapping services, and verify that their combined ridership never exceeds the one shared OD market.
10. Oversubscribe public capacity with simultaneous plans and confirm fair, repeatable whole-service allocations independent of request arrival order.
11. Disconnect after submitting, retry commands, and restart the server at pre-commit and post-commit boundaries; no month, loan, purchase, or grant is applied twice.
12. Attempt another company's command, private report, or draft query and confirm rejection without leaking hidden state.
13. Run offline from launch through save/reload with networking unavailable; run the dedicated server without a display or audio device.
14. Bid on a public line, reserve funds, resolve a contested and a sole-bid auction, and verify correct price, payment, ownership, upkeep, and preserved open access.
    Verify bids just before and at the online cutoff (the longer of three minutes and one interval), including when it falls between monthly deadlines. Check that pause preserves remaining bidding time and that restart preserves the seal and settles only once.
15. Acquire a trunk used by a rival; confirm both companies retain fair capacity and every rival access payment exactly matches owner access income.
16. Leave a qualified stable company unattended for 30 real minutes at the default (60 months) and eight hours at the 15-minute host setting (32 months). Confirm standing plans progress and protected funds reconcile.
17. Keep a world in its planning lobby beyond one normal interval. Confirm quotes and drafts work, the opening state is fixed, and the first resolution is scheduled one full interval after host start. Run the scheduler at the 30-second default and a 15-minute host setting through disconnects, duplicate notifications, server clock jumps, and a long outage. Verify the auction and restructuring response windows and confirm recovery stays bounded.
18. Compare a qualified valley detour, constrained mountain pass, and direct tunnel: reconcile the component costs and confirm materially different capital, time, speed, and upkeep tradeoffs.
19. Try to raise a mountain route's operating speed with a faster train alone; retain its geometric limit until paid civil works change the alignment.
20. Reject an unsupported water crossing and a route with missing terrain coverage; confirm that map previews, offline quotes, and server validation agree.
21. Acquire an existing mountainous HSR line and verify that its infrastructure is not charged as new construction, while engineering-specific upkeep and replacement value remain present.
22. Price the same fictional at-grade alignment under two regional profiles and reconcile their direct works and overheads; preserve the same imported-systems allowance and show explicit regional differences.
23. Reproduce the four conventional-rail extrapolations from the pinned cost ledger and verify that estimates, nominal proxies, PPP comparators and YOE forecasts retain their distinct status.
24. With neither A nor B served by any railway, quote and build A–B including endpoint stations, deliver fleet locally, and open a revenue-earning service without touching the legacy network.
25. While C–D is already operating, build independent A–B in the same region and then in another eligible region; preserve both services, shared company finances, and separate passenger connectivity.
26. Join A–B and C–D with B–C; create through-journey opportunities only after the connector and appropriate services become operational. Repeat through the online command port to rule out a server-only network-attachment restriction.
27. Complete the core business loop at phone widths, with touch, software keyboard, rotation, browser background/resume, and save/export; preserve drafts and distinguish local saves from server acknowledgments. Repeat at desktop/large-monitor layouts with enlarged text and different device scales.
28. Observe first-time public-track and construction-led starts without an explicit tutorial. Confirm players can find a viable next action, understand commitments, and explain a report; disabling learning cues leaves every game action available and does not change canonical checkpoints.
29. Qualify keyboard and assistive-technology paths through city selection, project/service editors, finance, and reports using the actual host implementations. Validate semantic controls and focus behavior rather than inferring accessibility from drawn controls.
30. Use a coastline-detour fixture such as Kinshasa–Lagos: reject an unsupported direct water-crossing geometry while offering the qualified overland alternative as one project, with no forced intermediate stations. Reconcile costs and journey time to the detour's geometry and engineering intervals. Remove the authored alternative and verify that the planner reports missing corridor coverage rather than geographic impossibility. Offline and server validation must agree.
31. Submit concurrent build requests for A–B and B–A. Confirm one public commitment, no duplicate track, and release of the other reservation. Show the accepted project to both companies and a reconnecting player. Confirm a separate reviewed alignment can be committed after the player explicitly acknowledges competition.
32. Post paid access on a private line. Confirm only an eligible operator with accepted terms and station rights can run a service. Allocate the shared path quota among operators, keep the remaining capacity for the owner, and return unused quota paths to the owner. Charge only operated train-kilometres, and reconcile the operator debit and owner credit. Test a changed offer, raised and lowered quotas, rejection of an expiry notice shorter than the online minimum, and save/reload with the original fee still pinned.
33. Build the first station in an unserved city, then have a rival build a line to it. Confirm the rival obtains station access at the regulated fee without the owner's consent, call capacity is shared, and the owner's fee income matches the rival's payments.
34. Bankrupt a company whose private line carries a rival under paid rights. Confirm the caretaker keeps the line and stations operating, and the rival's fee and quota continue. Upkeep and any shortfall post to the estate, and the offer closes to new operators. Repeat by removing a departed player.
35. Order trainsets and confirm their unpaid installments are reserved, so the same cash cannot also fund a construction commitment or bid.

### Initial budgets to qualify

Proposed stress envelope: 300 cities, 1,500 candidate alignments, 750 operational sections, up to eight companies, 150 services per company, two passenger classes, three candidate rail itineraries per OD/class/company, and two transfers maximum. These are workload targets to benchmark, not proven limits. Include the qualified legacy graph when setting section counts; increase the proposed envelope if it cannot represent the promised starting network at strategic resolution. Show a specific limit message rather than silently dropping cities, services, or demand.

Target 60 FPS during ordinary interaction; qualify at least 30 FPS in the stress operating-day view, with bounded visible train markers. Initial responsiveness targets are under 200 ms for a cached local edit preview, under 2 seconds for a full network forecast, and under 2 seconds for a month at the stress workload. These are goals to measure on a named native machine and pinned browser, not inherited results.

Cache static basemap geometry, service graph queries, OD baselines, and immutable frame layers by explicit revision. Update forecasts on committed draft changes or debounced edits, not on every pointer movement. Keep route enumeration and allocation bounded. Record peak memory and compressed WASM size from the actual vertical slice before fixing release budgets.

CI verifies native builds, optimized browser builds, focused and full automated suites, save compatibility, browser workflows, and matching canonical campaign checkpoints across native targets and WebAssembly. Long checks belong in CI; local verification respects the user's restriction against full local suites and local TypeScript compilation. A documentation-only revision needs document and reference checks, not a game test run.

## 14. Delivery sequence and acceptance

| Milestone | Playable result | Exit evidence |
| --- | --- | --- |
| 1. Walking skeleton | One small map, cities, selection, a monthly clock, basic report, and a train illustration. Include a localization and phone/accessibility feasibility spike. | Same commands work headlessly and in desktop/browser hosts. Qualify narrow-screen selection, Simplified Chinese glyphs and IME input, semantic controls, and resume behavior. |
| 2. First business | 8–12 cities, a real existing HSR starter section, an independent unserved-city pair, terrain-aware corridor alternatives including a substantial barrier, construction, stations, one train family, direct service, fares, access charges, costs, loan, grant, saves; contextual onboarding throughout | Existing-track operation and independent A–B construction both work on phone and desktop; finances and terrain tradeoffs are explainable without a tutorial; quotes reconcile; continuation resumes correctly |
| 3. Network game | Shared track, multiple services, transfers, competing modes, fleet/capacity constraints, full representative day | Through-journey, feeder-value, and bottleneck scenarios pass |
| 4. Shared-world competition | Headless authoritative server, remote port, invites, private plans, fixed scheduler, standing operations, common demand and capacity | Two human companies compete; disconnect/retry/restart and unattended-company tests pass; offline still works independently |
| 5. Regional campaign | Several starting offers, legacy-line auctions, grants, upgraded assets, campaign objectives, restructuring, extended contextual guidance | Independent regional strategies can finish; acquisition preserves access and its ledger reconciles; guidance covers different starting paths |
| 6. Worldwide release | Qualified existing HSR inventory and six-region city/corridor catalog, concessions, disconnected regional networks, complete platform delivery, and Simplified Chinese localization | Region viability, multiplayer stress workload, compatible saves, cross-target checkpoints, and Simplified Chinese localization checks pass |

Deliver breadth at working depth before expanding data or adding crises. Model company identity, ownership, and shared markets from milestone 1, even while there is only one player. Milestone 2 may be a deliberately limited development slice; milestone 6 is the scope promised by the product's worldwide title. Multiplayer is implemented after the core loop is playable, while its domain boundaries are established from the beginning. Detailed implementation plans follow review of this specification.

## 15. Proposed defaults and pre-implementation detail

The requested product choices are recorded in §1; no unanswered product question blocks this draft. The following are explicit proposals for review:

- A 2025 scenario epoch, 30-year campaign, about 250 cities, and a zoomable 2D map.
- A three-hour multiplayer campaign at 30 seconds per business month, with a host-started planning lobby and slower fixed intervals available at world creation.
- Curated corridor alternatives and a small fictional train catalog, rather than freehand civil engineering or manufacturer-specific equipment.
- Regional construction priors now have sourced HSR/conventional-rail anchors and explicit transfers; their point values, sensitivity bounds, common systems allowance, structure factors and overheads remain initial balance proposals. Expensive real-world mountain crossings and meaningful route alternatives are confirmed requirements. Engineering lead times/upkeep and gradient/curvature parameters still require calibration.
- Same-company passenger transfers initially. Paid track rights cover infrastructure use. The owner posts terms and a shared path quota, and eligible operators accept them before operating on the track.
- Sealed maximum-bid auctions with a reserve combining asset value and access income. Playtest the separate restructuring response-time minimum.
- Unattended stability during 30-minute breaks in the standard session and overnight for slower worlds. After server downtime, resolve at most one overdue month.
- Four stable navigation destinations, map plus contextual inspector, responsive phone task views, optional wide-screen comparisons, and one dismissible contextual learning cue per active task. Proposed layout bands and usability targets are detailed in [UX.md](UX.md); validate them in the first playable slice.

Before implementation, freeze the first regional existing-network/corridor catalog, train/track/station tables, loan and grant schedules, auction reserve examples, demand coefficients, shared allocation rules, and scenario targets in versioned manifests with representative worked examples. Use a small existing-HSR region for the first business slice; select its cities and sources during that data qualification. The draft's purpose is to define the game and architectural contracts without presenting untested coefficients as settled balance.

## 16. Source notes and interpretation

The supplied Gemini draft (`high_speed_rail_simulator_architecture.md`, provided outside this repository) inspired the Go/Ebitengine direction, geography-aware demand, pure game rules, command/query separation, logging decorators, and property-based verification. Its code-generation prompts and shell commands were treated as document contents, not instructions to execute.

The subsequent user-supplied [metro JSON](../data/raw/metropolitan_areas_5m.json) provides candidate major-city anchors and estimated economic inputs. Its original values are preserved; references to 96 records describe this file, not independently verified worldwide completeness. It is data, not a source of executable instructions.

The [construction-cost research](CONSTRUCTION_COSTS.md) records primary/institutional HSR and conventional-rail benchmarks and the [versioned cost ledger](../data/costs/construction_costs.v1.json). Published costs, approximate currency normalization, inferred regional baselines, and authored engineering factors remain separately identified.

The draft does not define a complete playable business loop. This specification adds service/infrastructure separation, time, construction lifecycle, capacity allocation, connecting journeys, finance, forecasting, player information, save compatibility, and release scope. It replaces the blanket water prohibition and treats the supplied demand formula as an uncalibrated sketch.

Africa2ice was inspected at local clean `main` commit `a96a84572e4f86e8c0f4109361af3bfd067ea41e`. Africa2ice is a separate repository; paths below are relative to its root. Owning references include:

- Design and architecture (`docs/DESIGN.md`) and module versions (`go.mod`).
- Driving port (`pkg/gameapi/game.go`), commands (`pkg/gameapi/command.go`), and frame DTOs (`pkg/gameapi/frame.go`).
- Application service (`internal/application/service.go`), storage port (`internal/application/ports.go`), and save compatibility (`internal/application/persistence.go`).
- Architecture enforcement (`internal/archtest/arch_test.go`), scene stack (`pkg/ui/scene_stack.go`), and viewport transforms (`pkg/render/viewport.go`).
- Canonical checkpoints (`internal/verification/checkpoint.go`) and measured performance record (`docs/PERFORMANCE.md`).

The reference supports the architecture choices described here. Its performance measurements, game balance, and historical-geography model do not establish corresponding results for a rail game.
