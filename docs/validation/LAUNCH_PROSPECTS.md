# Worldwide construction prospects

The game now includes **all 71 pairs in the launch spec**, with **73 alignments**:
69 new strategic alignments and the four unchanged alternatives for the two
Japanese projects. The catalog has 288 cities, 293 starting public corridors and
3,327 multi-section service templates, including 429 future templates that combine
construction with other projects or inherited track. The public map has 2,533
associated source-station markers. New construction is shown as dotted prospects;
started projects leave the picker and selectable prospect layer.

## Data and costs

- [Authoring manifest](../../data/construction/launch-prospects.v1.json) records all
  71 identities, route controls, named barriers, construction months, terrain
  shares, country profile overrides and explicit major crossing envelopes.
- [Pinned land/lake/country mask](../../data/construction/route-land.v1.json.gz)
  derives from Natural Earth 1:10m countries and lakes. Its embedded metadata pins
  the source hashes, URLs and public-domain license. It is a build input, not
  shipped runtime terrain or a digital elevation model.
- `prepare-launch-prospects.py` routes between authored controls using land-valid
  graph edges in geographic and Equal Earth space. Shortcuts through bays/lakes
  cannot be accepted merely because both endpoints are on land. Named bridges
  and the small Oriente cartographic station adapter are explicit local
  exceptions. The Congo/Nile and other named river works are costed, even where
  the land mask does not resolve the river. The Johor Strait bridge is a game
  concept; it is not a claim of an existing railway bridge.
- Australia, South Africa and Chile use detailed replacement shorelines. Four
  otherwise valid routes appeared offshore on the old coarse map; the replacement
  is checked against every affected route in geographic and projected space,
  with coarse rings removed rather than overlaid. The independent admission
  checker also tests all 69 new paths against the actual drawn world basemap.
- Terrain quantities are authored game estimates. Regional rates come from the
  researched 2024-USD ledger. Country intersections determine cost apportionment;
  each country's rate is applied to its assigned quantities. Work categories sum
  exactly to priced route kilometres. System allowance and engineering factors
  are applied once, then 15% management and 20% contingency. The prototype's 90%
  public grant remains a game rule. There is no procurement quote or guarantee of
  profitability; land acquisition and detailed station engineering remain outside
  the estimate. Strategic moving time uses 250 km/h plus terminal allowance,
  except the Lisbon–Porto 75-minute project target.
- The previous Japanese costs, alignments and durations remain exact compatibility
  inputs. The original small scenario assets are unchanged.

## Starting infrastructure and projects

**Morocco starts with Tangier–Kenitra dedicated HSR only.** The conventional
Kenitra–Rabat–Casablanca continuation is excluded from the public HSR map and
service catalog. The raw conventional station inventory is retained as source
context. Kenitra–Casablanca and Casablanca–Marrakech are new HSR projects. The
project endpoints are the game's called markets; every intermediate real-world
station is not modeled as a service stop in those project packages.
[AFD's operating-infrastructure description](https://www.afd.fr/fr/projets/la-ligne-ferroviaire-grande-vitesse-lgv-entre-tanger-et-kenitra)
explicitly separates the dedicated line and conventional continuation.

**Lisbon–Porto is distinct:** upgraded conventional public railway remains
available, labeled accordingly in the service picker. The new dedicated-HSR
project has independent capacity, a separate construction quote and a 75-minute
game journey. [IP's project description](https://www.infraestruturasdeportugal.pt/pt-pt/rede-de-alta-velocidade)
identifies a new passenger railway integrated with the conventional network.

New routes may be independent components. Future through-service templates stay
closed until every section is operational. They include Tokyo–Sapporo after the
Hokkaido extension, the California chain and Toronto–Boston links; these templates
are not additional construction projects or free track.

## Interaction and compatibility

Both clients expose all 71 projects through a complete picker while showing at
most two nearby quick choices in the ordinary plan. The native popup is capped
at 240 px. Map picking, expanded-map comparison, engineering details and exact-gap
funding continue to use the same project identity. A new parallel HSR project
cannot accidentally select its existing conventional service. English and
Simplified Chinese labels distinguish the scopes and regional cost basis.

Native controlled dropdowns use `InitialEntry`; queued initialization/stale
selection events cannot overwrite a draft or trigger repeated panel rebuilds.
Selection is accepted once per widget state before the next rebuild. The tests
exercise non-first entries and stale old-widget events at 320 and 1440 px.

Save v8 archives the exact v7 catalog and applies a separate construction cutover.
Past financial reports keep their old rules. Unrelated plans/projects/trains/cash
survive; now-ineligible Moroccan plans and their leased trains are retired.
Current animation omits those retired services, while historical reports retain
their accounts. Opening writes nothing, failed commits preserve state/bytes, and
the next accepted command upgrades atomically. No new game is required.

## Verification

Reproduce sequentially under the repository memory policy:

```sh
python3 tools/prepare-launch-prospects.py --check
python3 tools/import-public-network.py --check
python3 tools/check-launch-prospects.py
python3 tools/test-network-coverage.py
python3 tools/audit-public-network.py --check --strict
```

The independent checker matches all 71 spec identities, verifies 69 new paths in
both projections, requires priced crossing structures and exact regional work
coverage, and deliberately rejects the straight Lagos–Kinshasa ocean chord and
an unpriced Johor Strait crossing. The unchanged Japanese alternatives retain
their separate detailed mask check. CI includes these gates; local results do not
establish that remote CI has run.

Focused Go tests cover every funded build command and opening boundary, real
loan/construction/monthly revenue/reload campaigns across regions, Tokyo–Sapporo
through passengers, ordinary and Moroccan v7 migrations, failed-write atomicity,
regional quote arithmetic, duplicate-project rejection, independent parallel
capacity, snapshot immutability, native selection/layout and rendered prospects.
[Original 70-project local ledger](launch-prospects-checks.json) records hashes and actual
verification results.

Browser and native builds are separate from interaction qualification. Browser
WASM interaction was not rerun: the previous startup check exceeded the required
2 GiB combined checker RSS cap, and this work does not resolve that memory issue.
No new browser screenshot or physical-phone/native-OS input pass is claimed.
Full root-suite, races, extended fuzzing and remote CI remain hosted work.

These are strategic game alignments with cartographic screening, not surveyed
railway grades, curvature, geotechnical conditions, land rights or approved
projects. Construction months, terrain shares, grants and demand remain game
assumptions. The 35 older proximity-review candidates remain separate; no new
arbitrary junction snapping is introduced.


The [regional update](REGIONAL_NETWORK.md) adds Houston–Austin as project 71,
restores Taipei and repairs the Yeongcheon detour using pinned source links.
Current successful commands write save v9, with an additional regional epoch for
historical demand. Earlier v8 migration results above retain their original scope.
The regenerated geography ledger covers all 71 current projects; current full
verification is recorded in the regional-update ledger.
