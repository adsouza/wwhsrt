# Map and widget feasibility evidence

**Date:** 2026-11-08. Prototype evidence, not release qualification or a completed game.

## State and storage contracts

The [state/storage design](../STATE_AND_STORAGE.md) selects SQLite for the dedicated server and defines portable save v1. The focused Go reference checks compile its type declarations, compare generated schema/example output, and exercise wide-integer round trips, checksum/version handling and duplicate/trailing JSON rejection.

The [Python contract checker](../../tools/check-state-contracts.py) and [results](state-contract-checks.json) cover the structural schema, fixture bundle hashes, SQLite bootstrap/settings, atomic world/receipt/audit rollback, compare-and-swap conflicts, idempotency without duplicate audit events, reopen, locked-month atomicity and backup consistency. This uses Python's SQLite, not a production Go adapter. The reference decoder does not yet implement the full structural/domain restore pipeline. No crash-process/power-loss, authentication, network, native/browser storage, or full-game persistence qualification is claimed.

## Equal Earth map

The interactive mockup uses D3 7.9.0's `geoEqualEarth`, a content-addressed [Natural Earth snapshot](../../data/prototypes/natural-earth-110m.metadata.json), and the [metro seed](../../data/raw/metropolitan_areas_5m.json). Natural Earth's [terms](https://www.naturalearthdata.com/about/terms-of-use/) permit public-domain reuse. The [projection reference](https://proj.org/en/stable/operations/projections/eqearth.html) describes Equal Earth's global, equal-area purpose.

The raw source preserves all 96 records. The map embeds the 93 playable records with values unchanged, each at least 5M in population. Phoenix is retained at exactly 5M; Brasília (4.8M), Manila, and Taipei are excluded by the [playable selection policy](../../data/prototypes/playable-metro-selection.v1.json). Marker area is proportional to the supplied population. Selecting a metro shows its display name (without a trailing “Metropolitan Area”), country, coordinates, population, and estimated GDP. No new demographic estimates, railway inventory, prices, or station positions were inferred.

Region presets, zoom controls, wheel/pinch, and drag pan change the camera transform of one fixed projection. Labels and markers retain screen sizes; region zoom reveals more labels. Dense markers offer named disambiguation and all metros remain in the native picker. World and region views contain the same records, with out-of-frame markers clipped by the viewport.

Region labels and orientation-specific anchor sets are recorded in the [preset manifest](../../data/prototypes/map-region-presets.v1.json) and [UX spec](../UX.md#regional-camera-presets). Europe is displayed as Frangistan; the Africa label changes between Africa (East+West) and Africa+Arabia. Preset bounds use the supplied metro coordinates, and active presets refit after viewport changes.

The latest presets also use orientation-specific Frangistan anchors, separate Eastern/Western North America options in portrait, and SouthEast Asia in place of Oceania. Checks cover all 16 metro-anchor cases, anchor labels and marker visibility, projected rectangle fitting and the specified projected anchor-point fits, Dhaka remaining outside the East Asia viewport, menu changes, North America rotation with the last side remembered, and saved Oceania migration. Landscape-only Mashreq uses Alexandria, Tehran, Riyadh, and Dubai; checks cover its menu visibility and camera-preserving switch to portrait. Landscape North America uses Toronto, NYC, Mexico City, and SF Bay. SouthEast Asia uses Ho Chi Minh City, Singapore, and Yangon. Portrait Africa+Arabia includes Abidjan and excludes Karachi from the camera; portrait South Asia and Western North America use tighter projected-anchor fits, with Lahore aligned near the top. The initial portrait overview follows device standard UTC offset: 00:00–+05:30 selects a fully zoomed-out Dhaka eastern edge; greater than +05:30 through +09:00 selects a fully zoomed-out Karachi western edge. Other/unavailable timezones retain the Americas fallback. Saved views take priority, and all initial overviews remain outside the region list. Timezone fixtures cover boundary offsets, fractional offsets, daylight-saving stability, edge marker visibility, outward metro exclusion, and the latitude zoom floor. The World button is absent in portrait, the Americas overview is excluded from the region list, and manual pan/zoom remains available. Checks also cover World-to-portrait orientation changes, old saved World migration, and manual portrait camera restoration. Portrait camera limits add a 3° cushion beyond Melbourne and Saint Petersburg, yielding 40.8136°S–62.9311°N. Checks cover repeated zoom-out, edge-focused wheel input, emulated touch pinch, north/south pan, metro focus, intermediate animation frames, invalid restored cameras, and 320px/wide portrait resize; the minus button disables at the floor. The portrait map height responds to width to keep the Americas anchors visible. An Auto control follows actual window orientation; checks cover the shortened display names and removal of Manila/Taipei too.

Checks cover raw-record equality, 177 country geometries, forward-projection agreement for every marker, world reset, regional zoom, raw-value inspection, dense markers, narrow layouts, dark appearance, and reduced-motion navigation. [Map results](map-checks.json) include the subpixel projection comparison. The [checker](../../tools/check-map-preview.cjs) accepts a locally served visualization preview URL.

The current map preview applies [cheerful-rail-v1](../../data/ui/theme.v1.json). Its core normal-text/action/bright-fill text pairs were calculated at 4.5:1 or better in light and dark palettes; browser checks verify the manifest and actual action foreground colors. This is not full accessibility qualification. Construction/train motion and sound are specified in [PRESENTATION.md](../PRESENTATION.md), not implemented by this metro-only map or validated by these palette checks.

- [Cheerful dark palette](equal-earth-palette-dark.png)
- [Dhaka eastern-edge starting view](equal-earth-dhaka-edge-portrait.png)
- [Karachi western-edge starting view](equal-earth-karachi-edge-portrait.png)
- [Portrait Americas starting view](equal-earth-portrait-preview.png)
- [Mashreq in landscape](equal-earth-mashreq-landscape.png)
- [World overview](equal-earth-world.png)
- [South Asia with Delhi selected](equal-earth-south-asia.png)
- [East Asia in dark appearance](equal-earth-east-asia-dark.png)

The 1:110m geometry is an overview demonstration, not qualified terrain, railway alignment, urban detail, or production regional-zoom geometry. Metro estimates and CRS retain the qualification gaps in the raw-data manifest.

## Actual ebitenui rendering

The [isolated Go spike](../../tools/ebitenui-spike/README.md) executes **ebitenui 0.7.3 / Ebitengine 2.10.5**. Go **1.27.1**, macOS arm64, Chromium **153.0.8010.12**. The native macOS binary compiled; interaction checks executed the WebAssembly build with real library widgets.

| Case | Observed result |
| --- | --- |
| Appearance | Actual inputs, buttons, containers, rounded theme images, wrapped hints, forecasts, and report rows render in the proposed style |
| Responsive layout | Desktop columns become a narrow scrolling composition at 320, 390, 768, 1280, and 1920 logical px; edited values survive reflow |
| Footer and clipping | Scroll content has a distinct clipped rectangle above Apply and navigation |
| High DPI | 2× framebuffer rendering preserves 44-logical-pixel control heights |
| Pointer and keyboard | Buttons change forecasts, typing updates Go state and retains focus, invalid fleet disables Apply, application Ctrl/Cmd+A adapter works |
| Contextual help | Dismissing the hint preserves edits |
| Touch emulation | 120 ms taps operate widgets; numeric mobile input proxy gains focus and keyboard typing reaches Go state |
| Scrolling | Wheel and application-owned one-finger gestures move the masked ScrollContainer |
| Reports | Actual financial explanation and cash reconciliation render in desktop and phone compositions |
| Accessibility | The unaugmented canvas exposes zero semantic buttons and zero labeled inputs; the mobile proxy is not a semantic management interface |

The [result ledger](ebitenui-checks.json) includes checks, browser identity, WASM hash, and raw size. Its `passed` flag covers focused visual/input checks, **not** unresolved accessibility or physical-device requirements.

- [Service planner — actual desktop renderer](ebitenui-desktop.png)
- [Service planner — actual phone layout](ebitenui-phone.png)
- [Phone layout at 2× scale](ebitenui-phone-2x.png)
- [Monthly report — actual desktop renderer](ebitenui-report-desktop.png)
- [Monthly report — actual phone layout](ebitenui-report-phone.png)

**Conclusion:** ebitenui can render the proposed style and flexible compositions. Application code supplies responsive layout, explicit wrapping, theme images, input policy, state preservation, and platform adaptations. These visuals do not require an HTML-only widget renderer.

This does not establish screen-reader support, actual iOS/Android keyboard behavior, safe-area/keyboard occlusion, native accessibility, background/resume, persistent saves, offline cold launch, or full-game performance. Browser touch checks use emulation, not handsets. Keep the HTML HUD fallback and first-milestone device/accessibility work until those requirements are resolved.
