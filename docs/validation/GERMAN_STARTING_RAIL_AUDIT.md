# German starting railway audit

17 November 2026 · Playable public track · DB InfraGO infrastructure data, May 2026

### Playable map update

The starting map classifies whole domestic lines. It admits 15 when confirmed dedicated and upgraded track together exceed half of the mapped distance. The 21 lines with some confirmed fast track below the threshold are upgrade prospects. Their construction quotes cover the remaining distance. Three wholly conventional lines have no standalone offer. The denominator includes unresolved track. The domestic rule covers paths with at least 99% of their drawn length in the German survey. International through services retain the approaches they require. Company service status uses the ordinary map colours.

**Hamburg–Hamm runs via Bremen Hbf, Osnabrück Hbf and Münster (Westf) Hbf.** The corrected 322 km railway has 222 km of confirmed upgraded track, 82 km of conventional track and 18 km unresolved. At 69% upgraded it qualifies for starting service. The Oldenburg–Leer alignment in the original audit below has been withdrawn. The two Montabaur services share the Cologne–Frankfurt high-speed line. [DB InfraGO track data](https://data.gov.de/suche/daten/infrastrukturdaten-der-db-infrago) · [DB Hamburg departures](https://www.bahnhof.de/downloads/schedule/Regeltafel_2514.pdf)

To verify the survey, run `python3 tools/survey-german-track.py --check PATH_TO_INFRASTRUKTURDATEN_ZIP` with the Natural Earth mask present in `build/terrain-source`. The survey covers both the archived source inventory and the corrected Hamburg alignment. The old speed tables below describe the earlier map.

### Original speed audit

Of the 50 playable corridors that enter Germany, **28 have a majority of their German map distance on track rated below 200 km/h**, including 12 with at least 90% below that speed. Eleven have a majority at 200 km/h or above. The remaining eleven are mixed or too close to parallel tracks for a confident majority. These counts held under both sets of matching assumptions below. Each corridor counts once, even if another service uses the same rails. [DB InfraGO dataset published on GovData](https://data.gov.de/suche/daten/infrastrukturdaten-der-db-infrago)

### Where the distinction changes the game

- **Cologne–Montabaur and Frankfurt–Montabaur need a route correction.** DB runs passenger trains via Montabaur on its 300 km/h Cologne–Frankfurt line. The game’s Cologne–Frankfurt alignment picks up that line, with about 139 km in the 250+ band. But the separate Cologne–Montabaur path goes through Bonn, Koblenz and the Limburg area, with about 162 of its 168 drawn kilometres below 200. The Frankfurt branch also follows slower tracks toward Limburg. Use the Montabaur stop on the shared high-speed main line for both services. [DB’s operating notice naming the stops](https://www.deutschebahn.com/de/presse/presse-regional/pr-duesseldorf-de/presseinformationen-regional/Bauarbeiten-puenktlich-beendet-Reisende-fahren-zwischen-Koeln-und-Frankfurt-wieder-ueber-die-erneuerte-Schnellfahrstrecke-13008506)
- **Cologne–Mannheim, Cologne–Hamm and Düsseldorf Hbf–Hamm need review as starting HSR assets.** Cologne–Mannheim goes through Koblenz, Trier and Saarbrücken. About 414 of 420 drawn km match track below 200. For Cologne–Hamm the figure is 226 of 243 km, and for Düsseldorf–Hamm it is 99 of 101 km. A conventional passenger service could use these rails. Inventorying them as existing high-speed infrastructure would give the player the wrong starting network. [DB InfraGO track sections and speeds](https://data.gov.de/suche/daten/infrastrukturdaten-der-db-infrago)
- **Hamburg–Hamm is mixed and mostly conventional as drawn.** About 319 of 404 km match track below 200, including the detour along Oldenburg–Leer and the Emsland line. About 74 km match the 200–249 band. The importer’s 333 “connector km” is a separate game estimate. **Hamburg–Hannover has the opposite issue**: its mapped path has a substantial 200 km/h section, but the game assigns 518 km to a path measuring about 178 km. Berlin–Dresden also needs an endpoint check; its Dresden pin is Großenhain Cottbuser Bahnhof. [DB InfraGO data](https://data.gov.de/suche/daten/infrastrukturdaten-der-db-infrago) · [Playable inventory](../../assets/scenario/public-network.json)
- **Berlin–Hamburg and Nuremberg–Munich illustrate two different kinds of fast infrastructure.** About 243 km of the Berlin–Hamburg route match the 200–249 band. About 68 km of the Nuremberg–Munich route match the 250+ band on the Nuremberg–Ingolstadt line. Preserve those fast sections while representing conventional approaches at their own speeds. [DB InfraGO data](https://data.gov.de/suche/daten/infrastrukturdaten-der-db-infrago)
- **A border crossing can reverse the interpretation.** Nearly all of the roughly 8 German km of Liège–Aachen are on slower track. Belgian infrastructure manager Infrabel documents HSL 3 from Chênée to Hergenrath, open since 2009. The German percentage alone cannot classify the international service. [Infrabel line history](https://infrabel.be/fr/notre-histoire)

### Reading the corridor tables

All distances are approximate kilometres rounded independently. **Drawn** covers the whole playable path. **DE** covers the part inside Germany. The three speed columns and **Unresolved** partition the DE distance. Unresolved includes unassigned geometry, missing numeric speeds and plausible parallel tracks with different speed bands. **DB line IDs** are the largest identified contributors by matched length; a line can have several local speeds. Düsseldorf Airport is the station used for the game’s Rhine-Ruhr market.

The groups describe the speed mix of the physical path. A mixed route can support a legitimate through service. A majority at or above 200 km/h alone cannot tell whether a route is a dedicated high-speed line, an upgraded conventional line or several of each joined together. The game’s admission rule also requires a named alignment and dated commercial passenger operation. [Game design, existing HSR](../DESIGN.md#existing-infrastructure-and-access) · [UIC definition](https://uic.org/com/enews/nr/596-high-speed/article/the-definition-of-high-speed-rail)

### Mostly below 200 km/h (28 corridors)

| Game corridor | Drawn | DE | <200 | 200–249 | 250+ | Unresolved | DB line IDs |
|---|---:|---:|---:|---:|---:|---:|---|
| Augsburg – Colmar | 483 | 202 | 179 | 0 | 15 | 8 | 5360, 5304, 4570 |
| Cologne – Hamm (Westf) | 243 | 243 | 226 | 0 | 0 | 17 | 2800, 2651, 2880 |
| Cologne – Mannheim | 420 | 420 | 414 | 0 | 0 | 6 | 3010, 3280, 2630 |
| Cologne – Montabaur | 168 | 168 | 162 | 0 | 0 | 6 | 2630, 3710, 3731 |
| Cologne – Sélestat | 495 | 217 | 211 | 0 | 0 | 7 | 3010, 2630, 3140 |
| Dresden – Nuremberg | 444 | 376 | 315 | 50 | 0 | 11 | 5903, 6363, 6362 |
| Dresden – Vienna | 552 | 89 | 88 | 0 | 0 | 1 | 6240, 6248, 6239 |
| Duisburg – Arnhem | 99 | 79 | 78 | 0 | 0 | 0 | 2270, 2321, 2320 |
| Düsseldorf Airport – Hamm (Westf) | 100 | 100 | 62 | 18 | 0 | 20 | 2650, 2158, 2160 |
| Düsseldorf Hbf – Hamm (Westf) | 101 | 101 | 99 | 0 | 0 | 3 | 2550, 2525, 2932 |
| Frankfurt – Eisenach | 207 | 207 | 176 | 15 | 2 | 14 | 3600, 6340, 3680 |
| Frankfurt – Hamm (Westf) | 321 | 321 | 311 | 0 | 2 | 9 | 3900, 2550, 2972 |
| Frankfurt – Hannover | 352 | 352 | 197 | 0 | 129 | 27 | 3900, 1733, 3684 |
| Frankfurt – Montabaur | 103 | 103 | 95 | 0 | 2 | 6 | 3610, 3731, 3730 |
| Frankfurt – Nuremberg | 237 | 237 | 162 | 36 | 26 | 13 | 5910, 5200, 1733 |
| Hamburg – Hamm (Westf) | 404 | 404 | 319 | 74 | 0 | 11 | 2931, 2200, 1520 |
| Hannover – Bielefeld | 110 | 110 | 64 | 28 | 0 | 18 | 1700, 2990, 2981 |
| Hannover – Gotha | 218 | 218 | 118 | 0 | 84 | 16 | 1733, 6296, 6343 |
| Hannover – Hamm (Westf) | 370 | 370 | 223 | 0 | 122 | 24 | 1733, 2550, 3903 |
| Karlsruhe – Colmar | 226 | 167 | 116 | 0 | 21 | 30 | 4000, 4280, 4020 |
| Liège – Aachen | 49 | 8 | 7 | 0 | 0 | 1 | 2600, 2550 |
| Mannheim – Sélestat | 346 | 137 | 134 | 0 | 0 | 3 | 3280, 3250, 3231 |
| Munich – Vienna | 397 | 129 | 128 | 0 | 0 | 1 | 5600, 5510, 5531 |
| Nuremberg – Augsburg | 141 | 141 | 93 | 34 | 0 | 15 | 5320, 5300, 5310 |
| Nuremberg – Eisenach | 269 | 269 | 238 | 27 | 0 | 4 | 5910, 5240, 6311 |
| Nuremberg – Karlsruhe | 334 | 334 | 238 | 19 | 67 | 10 | 4080, 4930, 5902 |
| Nuremberg – Mannheim | 282 | 282 | 249 | 27 | 0 | 7 | 4120, 5910, 4111 |
| Nuremberg – Vienna | 583 | 144 | 138 | 0 | 0 | 5 | 5903, 5926, 5922 |

### Mostly 200 km/h or above (11 corridors)

| Game corridor | Drawn | DE | <200 | 200–249 | 250+ | Unresolved | DB line IDs |
|---|---:|---:|---:|---:|---:|---:|---|
| Augsburg – Karlsruhe | 276 | 276 | 107 | 38 | 121 | 10 | 5302, 4080, 4813 |
| Berlin – Gotha | 279 | 279 | 62 | 120 | 81 | 16 | 6132, 5919, 6340 |
| Berlin – Hamburg | 293 | 293 | 37 | 243 | 0 | 13 | 6100, 6107, 1244 |
| Dresden – Gotha | 237 | 237 | 57 | 61 | 108 | 12 | 5919, 6363, 6340 |
| Frankfurt – Cologne | 168 | 168 | 8 | 2 | 139 | 20 | 2690, 2651, 2621 |
| Frankfurt – Mannheim | 72 | 72 | 2 | 62 | 0 | 7 | 4010, 4011, 3656 |
| Gotha – Eisenach | 29 | 29 | 2 | 26 | 0 | 0 | 6340, 6697, 6311 |
| Hamburg – Bielefeld | 287 | 287 | 107 | 168 | 0 | 12 | 2200, 1283, 2992 |
| Hamburg – Hannover | 178 | 178 | 19 | 118 | 0 | 41 | 1720, 1710, 2200 |
| Mannheim – Karlsruhe | 60 | 60 | 8 | 19 | 28 | 5 | 4080, 4020, 4000 |
| Nuremberg – Gotha | 221 | 221 | 81 | 13 | 105 | 22 | 5919, 5900, 6340 |

### Mixed or unresolved majority (11 corridors)

| Game corridor | Drawn | DE | <200 | 200–249 | 250+ | Unresolved | DB line IDs |
|---|---:|---:|---:|---:|---:|---:|---|
| Aachen – Cologne | 70 | 70 | 33 | 1 | 3 | 32 | 2600, 2613, 2630 |
| Berlin – Dresden | 140 | 140 | 70 | 54 | 0 | 16 | 6135, 6248, 6172 |
| Berlin – Hannover | 268 | 268 | 50 | 69 | 72 | 77 | 6185, 6107, 1730 |
| Bielefeld – Hamm (Westf) | 67 | 67 | 6 | 9 | 0 | 52 | 1700, 2990, 2013 |
| Cologne – Düsseldorf Hbf | 41 | 41 | 9 | 10 | 0 | 22 | 2650, 2670, 2413 |
| Düsseldorf Airport – Duisburg | 15 | 15 | 2 | 2 | 0 | 11 | 2650, 2670, 2310 |
| Düsseldorf Airport – Düsseldorf Hbf | 8 | 8 | 1 | 1 | 0 | 6 | 2650, 2400, 2670 |
| Hannover – Eisenach | 246 | 246 | 97 | 0 | 127 | 22 | 1733, 6340, 3900 |
| Karlsruhe – Sélestat | 126 | 76 | 31 | 0 | 23 | 22 | 4280, 4260, 4020 |
| Munich – Augsburg | 55 | 55 | 16 | 10 | 0 | 29 | 5503, 5543, 5581 |
| Munich – Nuremberg | 165 | 165 | 60 | 6 | 68 | 31 | 5934, 5501, 5544 |

### Method and limits

1. Intersected all 311 public paths in the playable rail map with the Natural Earth 1:10m German boundary, including corridors whose endpoints have synthetic region IDs. Measured the resulting track in ETRS89 / UTM zone 32N. Cross-border rows include only the German portion in the speed assessment. Overlapping corridors share physical track, so adding these rows would overstate the size of the network. [Playable rail geometry](../../assets/scenario/playable-rail-map.json) · [Natural Earth country boundaries](https://www.naturalearthdata.com/downloads/10m-cultural-vectors/10m-admin-0-countries/)
2. Used the May 2026 DB InfraGO section file `M1 Streckennetz.csv`, filtering to records in DB network operation and excluding construction records. Sampled the drawn paths at intervals of at most 200 m and weighted each sample by the distance it represents. Matched to DB geometry within 80 m with a local heading difference of at most 26° in either direction. Kept a speed band only when all eligible tracks within 12 m of the best match agreed on the band. DB’s speed field is an infrastructure rating. Timetabled speeds, rolling-stock limits and temporary restrictions can differ. [DB InfraGO dataset and publication date](https://data.gov.de/suche/daten/infrastrukturdaten-der-db-infrago) · [Source CSV archive](https://mobilithek.info/mdp-api/files/aux/922109165921083392/Infrastrukturdaten.zip)
3. Repeated the match at up to 100 m intervals, with a 50 m search radius and an 18 m parallel-track tolerance. Assigned a majority only when the same band or combination exceeded half the **entire** German path in both runs, including unresolved distance in the denominator. The distance columns report the first run. Bielefeld–Hamm, Aachen–Cologne and Munich–Augsburg are particularly sensitive to parallel tracks. Those need track or line IDs before assigning speeds to individual game sections.
4. This is a speed and geometry audit of the current playable inventory. Commercial operation at a historical scenario cutoff requires separate evidence for each candidate. The 200 km/h screen captures upgraded routes, while the 250+ band highlights faster infrastructure. Neither a high-speed train name nor a corridor endpoint determines the track class.

Reviewed map SHA-256 `1ed05fead27b7d9ab250c33e2280aba8d53f07df1e43f3565aad19ed2783a8f6`. DB source archive SHA-256 `1fa5267cab954d8393878f7bb2396bf9ebb16f914701733c145c073f7e707f36`.
