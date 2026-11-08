# Construction-cost evidence and calibration

**Research date:** 2026-11-08. **Model:** `hsr-construction-2024usd-v1`.

This appendix supports the regional tables in [DESIGN.md](DESIGN.md). The [machine-readable ledger](../data/costs/construction_costs.v1.json) preserves 22 observations, source URLs, currency/price assumptions, arithmetic, conventional-rail extrapolations, and 20 regional profiles. The profiles are **inferred game inputs**, not observed national averages or a statistical estimate of country effects. Some underlying totals are completed costs; others are budgets, forecasts, or narrow contracts, identified below.

## Scope and normalization

The game baseline `B` prices **one route-kilometre of new rural at-grade, double-track, electrified 300 km/h railway**, including track and standard railway systems. A 100 km double-track route uses 100 route-km, not 200. Land, major stations, depots, trainsets, special structures, design/management, contingency, financing, and tax are outside this baseline. Explicit structure quantities replace surface quantities. The complete construction quote adds the excluded applicable items once.

Use constant **2024 USD** for construction and the game's common money basis. This is independent of a scenario starting in 2025. Historical comparisons use:

```text
screening_cost_2024_usd_per_route_km
    = reported_cost / historical_LCU_per_USD / route_km
      × US_CPI_2024 / US_CPI_price_year
```

The ledger freezes annual-average [BLS CPI-U through FRED](https://fred.stlouisfed.org/series/CPIAUCNS) and [World Bank official average exchange rates](https://data.worldbank.org/indicator/PA.NUS.FCRF), including query URLs and values. CPI 2024 is `313.689`; the reference series is all-items, all urban consumers, not a local or construction-specific series. Historical market FX avoids applying today's exchange rate to old budgets.

This is an approximate common-dollar screen, not a construction purchasing-power model. A construction index and annual disbursements would be preferable. Where the source gives nominal spending across years or omits a price base, the ledger explicitly uses an audit, approval, or opening year as a **proxy**. A proxy cannot establish an exact constant-price cost. The calculations help locate broad orders of magnitude; they do not justify precise country rankings.

Year-of-expenditure (YOE) forecasts remain in their original basis unless a spending schedule allows deflation. PPP-converted comparator figures remain separate from market-FX figures. Financing amounts are not total project costs, expenditures to date are not completion costs, and grants/loans do not reduce the economic resource cost of construction.

## Evidence ledger

The final column is a whole-project or package comparison, **not `B`**. `≈` indicates that a mixed/unspecified price base required a proxy. Exact-looking source inputs do not remove scope uncertainty.

| IDs | Railway or sample | Published amount and denominator | Evidence/scope | Screening 2024 USD million/route-km |
| --- | --- | --- | --- | ---: |
| E01–E02 | China 350 / 250 km/h samples | $20.6m / $16.9m per route-km, standardized to 2017 by the study | Constructed double-track lines, including electrical/control systems and facilities; mixed terrain | 26.4 / 21.6 |
| E03–E04 | France LGV Est / Rhin–Rhône | €6.712bn / 406 km; €2.588bn / 138 km | Audited nominal totals at 2018; completed lines | ≈24.4 / ≈27.7 |
| E05 | Spain Madrid–Barcelona–border | €12.109bn / 797 km | Audited nominal line total | ≈22.4 |
| E06–E07 | Germany Berlin–Munich / Stuttgart–Munich | €14.682bn / 671 km; €13.273bn / 267 km | First includes mixed line works; second was incomplete at audit | ≈32.3 / ≈73.3 |
| E08 | Italy Turin–Salerno | €32.169bn / 1,007 km | Audited nominal line total | ≈47.1 |
| E09 | India Mumbai–Ahmedabad | ₹1.08 trillion / 508 km | Original budget reiterated in 2025, not completed outturn; 2017 approval-year proxy | ≈41.8 |
| E10 | Japan Nishi Kyushu | ¥620bn / 66 route-km | 2019 estimate; 67 construction-km is a different denominator; tunnel-heavy | ≈105.7 |
| E11 | Korea Honam phase 1 | ₩8.119tn / 182.3 km | Historical phase total on updated owner page; unstated price base, 2015-opening proxy | ≈52.1 |
| E12 | Morocco Tangier–Kenitra package | MAD22.9bn / 200 km of new HSR | 2015-status budget in January 2016 review; facilities/package scope; not the full through-service length | ≈15.5 |
| E13 | Indonesia Jakarta–Bandung | $7.27bn / 142.3 km | Reported investment; structures and facilities; 2023-opening proxy | ≈52.6 |
| E14 | Saudi Haramain | SAR60bn / source-rounded 450 km | Reported construction total; package and price base unspecified; 2018-opening proxy | ≈44.4 |
| E15 | Tanzania–Burundi corridor | $3.93bn / 651 km | Conventional electrified railway project estimate, 2023 approval proxy | ≈6.2 |
| E16 | Uruguay Ferrocarril Central | $839m / 273 km | 2019 reconstruction estimate; excludes a separately stated $97m of associated roads | ≈3.8 |
| E17 | Brazil FICO Mara Rosa–Água Boa | BRL2.539bn / 365 km | Contractual estimate at October 2020 prices; 80 km/h freight railway; includes specified land/environment/design costs | 1.6 |
| E18 | UAE Etihad Rail Package A | AED1.5bn / 139 km | 2019 double-track civil/track award; excludes a complete HSR electrical/equipment package | ≈3.6 |
| E19 | Australia Inland Rail | AUD31.4bn / roughly 1,700 km | 2022 forecast reviewed in 2023; mixed new/upgraded freight works, escalation and wider program costs | ≈13.7 |
| E20 | UK HS1 | £6.163bn / 109 km | Completed nominal multi-year line/depot total; excludes Underground remodelling; 2007-opening proxy | ≈171.2 |
| E21 | California Merced–Bakersfield | $36.75bn / 171 miles = 275.198 km | August 2025 P65 forecast; Table B.2 is YOE and includes broad scope/contingency | Not normalized; $133.5m/km YOE |
| E22 | Egypt Siemens contract share | €8.1bn over approximately 2,000 km | Supplier share includes equipment, trains and 15-year maintenance; **not total construction** | Excluded as a whole-line benchmark |

### Sources and interpretation

- **E01–E02:** [World Bank, China’s High-Speed Rail Development](https://documents1.worldbank.org/curated/en/933411559841476316/pdf/Chinas-High-Speed-Rail-Development.pdf), pp. 39–41 and note on p. 49. This is the most useful speed-specific constructed-network reference in this set. Its line averages already contain engineering complexity; they cannot be used as a flat rate with the same complexity added again.
- **E03–E08:** [European Court of Auditors, Special Report 19/2018, Table 5](https://op.europa.eu/webpub/eca/special-reports/high-speed-rail-19-2018/en/). The audit calls the figures nominal; this is why the screening conversions use an explicit proxy rather than claiming a common observed 2018 construction base.
- **E09:** [Indian Parliament, July 2025 Question 54](https://sansad.in/getFile/loksabhaquestions/annex/185/AS54_7WACnw.pdf?source=pqals). A restated original budget is not an updated estimate of final expenditure. Its 2017 conversion is a disclosed approximation.
- **E10:** [MLIT 2019 reevaluation](https://www.mlit.go.jp/tec/hyouka/public/kisha/190329h30/pdf/06.pdf), Nishi Kyushu row, combined with [JRTT route specifications](https://www.jrtt.go.jp/english/construction/shinkansen.html). JRTT gives 66 route-km and approximately 61% tunnel by construction length. This checks a structure-heavy reconstruction, not the price of flat Japanese countryside.
- **E11:** [Korea National Railway project profiles](https://www.kr.or.kr/sub/info.do?m=05010202), Honam phase 1 amount and length. Do not divide the combined two-phase budget by phase-1 length. The missing historical price base limits the screening result.
- **E12:** [Morocco Ministry of Finance review](https://www.finances.gov.ma/Maliya%20tawassol/file_160216_153707547.pdf), January 2016, printed p. 11, 2015-status budget, and [AFD project description](https://www.afd.fr/fr/projets/le-train-le-plus-rapide-dafrique), new-line length. This is a budget/package comparison rather than a final asset-level bill.
- **E13:** [Government report of KCIC investment](https://inp.polri.go.id/artikel/president-prabowo-to-issue-presidential-decree-to-settle-high-speed-rail-debt-luhut) and [KCIC route specification](https://kcic.co.id/kcic-siaran-pers/informasi-kereta-whoosh-makin-mudah-kcic-hadirkan-contact-center-via-telepon-hingga-whatsapp/). The investment is not an audited civil-only outturn series.
- **E14:** [Saudi Ministry of Media’s Haramain entry](https://saudipedia.com/en/haramain-high-speed-railway). Use its rounded route denominator consistently and keep unstated cost scope visible.
- **E15:** [AfDB project approval](https://www.afdb.org/en/news-and-events/press-releases/african-development-bank-approves-69641-million-financing-burundi-and-tanzania-build-650-kilometers-rail-infrastructure-develop-central-corridor-network-67028), cross-checked against the [AfDB 2025 project booklet](https://bos.afdb.org/documents/presentation/BOS%20Abidjan%202025%20Booklet.pdf). The $696.41m financing headline is not the project budget; the latter is the relevant numerator.
- **E16:** [Uruguay Presidency, June 2019](https://www.gub.uy/presidencia/comunicacion/noticias/astori-ferrocarril-central-busca-corregir-error-historico-garrafal-cometio). Retain the explicit distinction between rail investment and associated road works.
- **E17:** [ANTT Voto DFQ 82/2024](https://portal.antt.gov.br/documents/498202/0/Voto%2BDFQ%2B-%2B082.pdf/866c2a45-023a-c040-746c-508f4ab3f5ec?t=1729522360270), §§3.2–3.3. Low-speed freight specification and a stated 2020 contract base make this a useful conventional lower anchor, not a price for HSR.
- **E18:** [Etihad Rail’s 2019 Package A award](https://corporate.etihadrail.ae/en/newsroom/press/etihad-rail-launches-package-a-of-stage-two-to-connect-ghuweifat-and-ruwais). Double track is already in scope; an extrapolation must not add a second-track factor again.
- **E19:** [Australian Government’s Inland Rail review](https://www.infrastructure.gov.au/sites/default/files/documents/independent-review-of-inland-rail-report.pdf), pp. 11 and 52. More than 70% of the works are brownfield. Reuse does not automatically make a corridor cheap: maintaining traffic during works can add cost. The escalation-bearing forecast is a particularly approximate screening anchor.
- **E20:** [NAO’s 2024 lessons report, HS1 case](https://www.nao.org.uk/wp-content/uploads/2024/02/delivering-value-from-government-investment.pdf), Appendix 1. Publication in 2024 does not make a nominal 1998–2007 construction total a 2024-price total.
- **E21:** [California HSR’s August 2025 supplemental report](https://hsr.ca.gov/wp-content/uploads/2025/08/2025-Project-Update-Report-SUP-FINAL-081925-A11Y.pdf), Tables 1.5 and B.2. Keep P65/YOE scope separate from direct works, already-spent amounts, and funding gaps.
- **E22:** [Siemens’ 2022 Egypt announcement](https://press.siemens.com/global/en/pressrelease/siemens-mobility-finalizes-contract-2000-km-high-speed-rail-system-egypt). Dividing this contract share by network length does not establish a complete railway cost per kilometre.

## Converting conventional rail into HSR priors

Where comparable HSR evidence is missing, use an explicit bridge from conventional cost to the target scope. For the worked examples, all currency quantities below are **million 2024 USD per route-km**:

```text
B_candidate = ((C × retained_scope_fraction − included_systems_allowance)
               / source_terrain_factor)
              × greenfield_factor × track_factor × HSR_geometry_factor
              + target_systems_allowance
```

`C` is the screening cost above. Subtract already-included electrical/control scope before adding the target systems allowance. The track factor accounts for a proposed single-track-equivalent source versus double-track target; use `1.0` where source double track is already included. A greenfield factor accounts for a reconstruction source only when justified. Do not apply a universal “HSR costs twice as much” rule or automatically double the entire source cost because the target has two tracks.

The adjustments below are **authored assumptions**, not observed component breakdowns or universal engineering ratios. Conventional sources do not identify these factors independently. Round the result upward to the next $2m/km for the initial game point; preserve a wide sensitivity envelope rather than pretending the rounding is empirical precision.

| Profile | Source `C` | Scope retained | Existing systems removed | Terrain divisor | Greenfield | Track | HSR geometry | Target systems added | Result → adopted `B` |
| --- | ---: | ---: | ---: | ---: | ---: | ---: | ---: | ---: | ---: |
| East Africa | 6.215 | 0.85 | 0.8 | 1.2 | 1.0 | 1.6 | 1.4 | 3.0 | 11.37 → 12 |
| Other Latin America | 3.771 | 0.85 | 0 | 1.0 | 1.4 | 1.6 | 1.4 | 3.0 | 13.05 → 14 |
| Brazil | 1.635 | 0.90 | 0 | 1.0 | 1.0 | 1.6 | 1.5 | 3.0 | 6.53 → 8 |
| Australia / NZ | 13.733 | 0.75 | 0 | 1.0 | 1.0 | 1.6 | 1.25 | 3.0 | 23.60 → 24 |

The East Africa example assumes a single-track-equivalent conventional scope and makes a modeled allowance for included systems and terrain; confirm the full appraisal before treating that decomposition as an engineering estimate. The Latin America example includes a modeled greenfield uplift to a reconstruction source. Brazil's low freight benchmark produces a lower prior, with uncertainty for the substantial specification change. Australia's program-level forecast is discounted for scope/overhead before upgrading its technical offer; its mix of works does not justify a blanket reuse saving.

West/Central Africa transfers the East Africa point by `7/6`; Southern Africa by `4/3`. These are initial delivery/logistics scenarios, not measured regional cost gaps. Other unsupported country assignments are labeled `X` in the spec. No continent receives a claim of uniform costs, and lack of HSR history does not imply Chinese scale efficiencies or cheap imported systems.

## Regional priors and engineering consistency

The spec's table is the normative readable table; JSON contains the same points, ranges, overheads, country-label mappings, and evidence classes. All 46 distinct country labels in the supplied 96-metro seed have an explicit assignment. No automatic GDP-based rate or silent missing-country fallback is allowed. Country labels are game-data assignments, not a jurisdictional claim about borders or access rights.

A region's `B` is a rural direct-works prior after deliberately removing terrain and package effects by judgment. **HSR-informed points are not fitted by mechanically dividing an all-in source total**, because reliable component quantities are missing. The ledger preserves their rationale and the observations used. Transfer priors and conventional extrapolations remain separately identified. Low/high values are design sensitivity scenarios, not confidence intervals, percentiles, or random costs.

The revised component model separates a common `$3m/km` electrical/train-control/communications allowance from local civil/track cost. That common amount is an authored gameplay assumption, not a measured equipment-price index. It avoids making the whole railway package cheap solely because local labor is inexpensive. Specialized tunnel ventilation, bridge foundations and unusual power supply remain explicit component/protection scope rather than being hidden inside the common allowance.

The [DfT/HS2 international study](https://assets.publishing.service.gov.uk/media/5fb7de78e90e077ee0856f00/high-speed-rail-international-benchmarking-study-document.pdf), pp. 13–16, reports standardized PPP-based 2011 comparators: rural schemes £11–20m/km, urban/high-interface schemes £43–61m/km, complex schemes £72–93m/km; mean civil asset figures are approximately £6m/km earthworks, £31m/km viaducts and £36m/km tunnels. It also reports indirect and contingency allowances of 7.5–20.4% and 10–30%. Use those as scope/ratio evidence, **not market-FX national rates**. The game factors are still design choices; the old blanket `6 × B` viaduct and `12 × B` ordinary tunnel rates are replaced to avoid overpricing ordinary standardized structures and repeatedly multiplying systems costs. Deep base/underwater works retain higher classes and project-specific overrides.

Initial design/management fractions are 15% generally and 20% for the UK, US/Canada and Oceania. Initial contingencies are 20% generally, 25% for the UK, broad Eurasian transfers, East/Southern Africa, Latin America, US/Canada and Oceania, and 30% for West/Central Africa. These assignments are scenario parameters, not measured national risk probabilities. Apply them once to an explicit subtotal, rather than adding a second contingency to a contingency-inclusive project quote. Financing remains a separate ledger item.

## Practical limits and next qualification

This research establishes usable initial worldwide rates with inspectable evidence and extrapolations. It does not establish a tender-ready estimate for any real corridor. The most valuable next evidence is asset-level tender quantities and cost schedules, especially for transferred countries and for property, stations, systems, and current construction-price indices. A first-region corridor must reconstruct a plausible complete quote from its actual engineering quantities before the parameters are considered playtest-qualified. Do not force it to match an incompatible historical total by silently changing a region multiplier.

The first game's 12–48-month construction schedule remains deliberate time compression. The source projects' multi-year and sometimes multi-decade delivery periods do not empirically validate that gameplay duration. The release documentation should keep that distinction visible.
