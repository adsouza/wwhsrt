# Public railway map and database

Railways and station data © OpenStreetMap contributors.
https://www.openstreetmap.org/copyright
Open Database License (ODbL): https://opendatacommons.org/licenses/odbl/1-0/

Country outlines: Natural Earth, public domain. https://www.naturalearthdata.com/

The original source/audit metadata and operator references are preserved in
https://github.com/adsouza/wwhsrt/tree/main/data/imports

The imported geometry is provisional cartography. Hollow station markers indicate
known map gaps. Service templates, times, capacity, demand and fares are game
assumptions, not published operator timetables. See data/imports/README.md.

City names/coordinates: GeoNames (https://www.geonames.org/), CC BY 4.0.
Supplementary urban populations: © European Union, 1995–2026, GHSL Urban Centre
Database R2024A V1.2 (Mari Rivero et al.), CC BY 4.0.
https://human-settlement.emergency.copernicus.eu/ucdb2024Overview.php
https://creativecommons.org/licenses/by/4.0/
Original user-supplied metro estimates remain distinct; see data/README.md.

Gyeongju regional connector/crossover geometry uses eight additional OSM ways,
with ordered source node IDs and query hashes pinned in
data/imports/rail-regional-links.v1.jsonl.gz. Taiwan station-platform aggregation
is a reviewed strategic station abstraction, not surveyed new track or a claim
of physical crossovers. Taiwan HSR service/station references:
https://en.thsrc.com.tw/corp/c33f8d02-7c8a-44f0-8866-b3f8c9707ff9

Ulsan KTX/Tongdosa gateway reference (distinct from Bugulsan):
https://english1.visitkorea.or.kr/enu/TRP/TP_ENG_4.jsp

The connector expansion adds `data/imports/rail-bridge-approaches.v1.jsonl.gz`:
2,012 pinned active railway ways for the operational Turin–Milan and Perpignan
station approaches, with ordered OSM node IDs, query/response fingerprints and
ODbL attribution. Refresh explicitly with
`python3 tools/fetch-network-connectors.py --bridge-approaches`; ordinary imports
remain offline. Source-region bounds/gauge policy are recorded in
`data/imports/bridge-approach-regions.v1.json`. These slow connecting approaches
are not classified as newly built dedicated HSR. The existing public HSR links
remain available at game start.

The operating manifest names the Maibara, Camp de Tarragona, Figueres, Turin
and Perpignan stations. Reviewed parallel HSR platform ways at Maibara, Kyoto,
Nagoya, Shin-Osaka, Barcelona, Camp de Tarragona and Figueres are aggregated as
strategic station resources within 100 m; other junctions require shared OSM node
IDs. Infrastructure scope references: [FS Turin–Milan history](https://www.fsitaliane.it/content/dam/fsitaliane/Documents/old-media-ed-eventi/comunicati-stampa-e-news/anno-2019/giugno/2019_06_03_NS_10_anni_AV.pdf),
[French ministry Perpignan–Figueres](https://www.ecologie.gouv.fr/sites/default/files/documents/CP_-_LVG_Perpignan-Figueras_-_21-12-2016.pdf),
[JR Central route map](https://global.jr-central.co.jp/en/info/route-map/) and
[Adif Barcelona–French border](https://www.adifaltavelocidad.es/w/lav-madrid-barcelona-frontera-francesa-tramo).

The Brussels, Düsseldorf and Randstad services use Amsterdam Centraal as the
Randstad terminal. Their operating itineraries follow
[Eurostar Brussels–Amsterdam](https://www.eurostar.com/be-en/train/brussels-to-amsterdam),
[NS International Amsterdam–Düsseldorf](https://www.nsinternational.com/en/germany/train-dusseldorf)
and the [Eurostar Brussels–Düsseldorf timetable](https://www.eurostar.com/be-en/reiseinformationen/fahrplanauskunft/8814001/8008094/brussel-midi-zuid/dusseldorf-hbf).
The pinned section geometry in `data/imports/rhine-randstad.v1.json` was
routed with the BRouter rail profile on 2026-11-17 from
[OpenStreetMap contributors' data (ODbL)](https://www.openstreetmap.org/copyright).
Each section includes its source query. Running minutes are game estimates and
station dwell is added by the simulation. The Brussels service shares the existing Cologne–Düsseldorf
track. The Randstad service uses the existing Düsseldorf airport approach.

German track profiles are derived from DB InfraGO AG, “Infrastrukturdaten der DB InfraGO”,
May 2026, M1 Streckennetz. The game matches sections to its map and assigns track classes.
[Source and licence](https://data.gov.de/suche/daten/infrastrukturdaten-der-db-infrago?ids=b88d0ec0-dbab-4990-8539-6461f32df53b) · [CC BY 4.0](https://creativecommons.org/licenses/by/4.0/)

The existing Toulouse–Carcassonne–Narbonne–Béziers railway is derived from SNCF Réseau’s
[Fichier de formes des lignes du Réseau Ferré National](https://ressources.data.sncf.com/explore/dataset/formes-des-lignes-du-rfn/),
operating line 640000. Geometry has been clipped and simplified. [SNCF Open Data licence (ODbL)](https://data.sncf.com/pages/licence)
The hypothetical Toulouse–Narbonne high-speed alignment, price and timetable are game designs.
The Hamburg–Hamm alignment follows BRouter's rail routing on OpenStreetMap through Bremen, Osnabrück
and Münster and is surveyed against the DB InfraGO data credited above.

Beograd Centar–Petrovaradin–Vrbas Nova uses the BRouter rail profile on
[OpenStreetMap contributors’ data (ODbL)](https://www.openstreetmap.org/copyright).
The two rail paths and source queries are pinned in `data/imports/belgrade-vrbas.v1.json`.
The 69-minute game service includes the intermediate stop and follows the IC 12540 elapsed time
in the [Srbija Voz timetable](https://www.srbvoz.rs/wp-content/redvoznje/bgcent_ns_su.pdf).
