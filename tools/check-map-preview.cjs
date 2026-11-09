// Focused checks against a locally served visualization preview.
const assert = require('node:assert/strict');
const fs = require('node:fs');
const path = require('node:path');
const { chromium } = require('playwright');
const url = process.argv[2] || 'http://127.0.0.1:8766/equal-earth-world-map-preview.html';
const root = path.resolve(__dirname, '..');
const raw = JSON.parse(fs.readFileSync(path.join(root, 'data/raw/metropolitan_areas_5m.json')));
const manifest = JSON.parse(fs.readFileSync(path.join(root, 'data/prototypes/map-region-presets.v1.json')));
const latitudeCushion = 3;
const latitudeSouth = raw.find(m => m.id === 'melbourne').lat - latitudeCushion;
const latitudeNorth = manifest.portrait_limits.northern_reference.lat + latitudeCushion;
const original = raw.filter(m => m.population >= 5_000_000 && !['manila', 'taipei', 'brasilia'].includes(m.id));
(async () => {
  const browser = await chromium.launch({ headless: true });
  const checks = { date: '2026-11-08', browser: browser.version(), projection: 'Equal Earth', checks: [] };
  try {
    const page = await browser.newPage({ viewport: { width: 1280, height: 900 }, timezoneId: 'America/New_York', ignoreHTTPSErrors: true });
    const errors = []; page.on('pageerror', e => errors.push(e.message));
    await page.goto(url); const f = page.frameLocator('iframe');
    await f.locator('.wm-metro').first().waitFor();
    assert.equal(await f.locator('.wm-metro').count(), 93);
    const theme = JSON.parse(fs.readFileSync(path.join(root, 'data/ui/theme.v1.json')));
    assert.deepEqual(JSON.parse(await f.locator('#wm-theme').textContent()), theme);
    assert.equal(await f.locator('#equal-earth-rail-map').getAttribute('data-theme'), 'cheerful-rail-v1');
    assert.deepEqual(JSON.parse(await f.locator('#wm-metros').textContent()), original);
    assert.equal(await f.locator('[data-preview="portrait"]').getAttribute('aria-pressed'), 'true');
    assert.equal(await f.locator('.wm-stage').getAttribute('data-orientation'), 'portrait');
    assert.equal(await f.locator('[data-action="world"]').count(), 0, 'no portrait World button in the DOM');
    assert.equal(await f.locator('[data-region] option[value="americas"]').count(), 0, 'initial Americas overview is not selectable');
    assert.ok(!(await f.locator('[data-region]').innerText()).includes('World'));
    assert.equal(await f.locator('[data-region]').inputValue(), '');
    assert.equal(await f.locator('.wm-stage').getAttribute('data-camera-preset'), 'americas');
    const initialIds = ['buenos_aires', 'toronto', 'san_francisco', 'rio_de_janeiro'];
    const initialAnchors = initialIds.map(id => original.find(m => m.id === id));
    const initialBounds = [Math.min(...initialAnchors.map(m => m.lng)), Math.min(...initialAnchors.map(m => m.lat)), Math.max(...initialAnchors.map(m => m.lng)), Math.max(...initialAnchors.map(m => m.lat))];
    assert.deepEqual(JSON.parse(await f.locator('.wm-stage').getAttribute('data-preset-bounds')), initialBounds);
    assert.ok(await f.locator('.wm-stage').evaluate((stage, ids) => ids.every(id => {
      const dot = stage.querySelector(`[data-city-id="${id}"]`), x = Number(dot.getAttribute('cx')), y = Number(dot.getAttribute('cy')), r = Number(dot.getAttribute('r'));
      return x >= r && x <= stage.clientWidth - r && y >= r && y <= stage.clientHeight - r;
    }), initialIds));
    for (const id of initialIds) assert.equal(await f.locator(`.wm-label[data-city-label="${id}"]`).count(), 1);

    assert.equal(await f.locator('[data-city-id="manila"], [data-city-id="taipei"], [data-city-id="brasilia"]').count(), 0);
    assert.ok(!(await f.locator('[data-city]').innerText()).includes('Metropolitan Area'));
    await page.screenshot({ path: path.join(root, 'docs/validation/equal-earth-portrait-preview.png'), fullPage: true });
    const initialZoom = Number(await f.locator('.wm-stage').getAttribute('data-zoom'));
    await f.locator('[data-action="in"]').click(); await page.waitForTimeout(350);
    assert.ok(Number(await f.locator('.wm-stage').getAttribute('data-zoom')) > initialZoom * 1.5);
    await f.locator('[data-action="out"]').click(); await page.waitForTimeout(350);
    assert.ok(Math.abs(Number(await f.locator('.wm-stage').getAttribute('data-zoom')) - initialZoom) < 0.002);
    const phoneStage = await f.locator('.wm-stage').boundingBox();
    await page.mouse.move(phoneStage.x + phoneStage.width / 2, phoneStage.y + phoneStage.height / 2);
    await page.mouse.wheel(0, -150); await page.waitForTimeout(400);
    assert.ok(Number(await f.locator('.wm-stage').getAttribute('data-zoom')) > initialZoom);
    checks.checks.push('portrait starts at hidden Americas bounds with all four anchor labels; World absent from DOM; dropdown has only specific regions; manual +/- and wheel zoom remain usable');
    await f.locator('[data-city]').selectOption('shanghai');
    assert.match(await f.locator('.wm-detail').innerText(), /Shanghai/);
    assert.ok(!(await f.locator('.wm-detail').innerText()).includes('Metropolitan Area'));
    await f.locator('[data-preview="auto"]').click();
    await f.locator('[data-action="world"]').click(); await page.waitForTimeout(850);
    assert.equal(await f.locator('.wm-stage').getAttribute('data-visible-count'), '93');
    const geometry = JSON.parse(await f.locator('#wm-countries').textContent()); assert.equal(geometry.features.length, 177);
    const context = page.frames().find(frame => frame !== page.mainFrame());
    const coordinateError = await context.evaluate(async () => {
      const { geoEqualEarth } = await import('https://cdn.jsdelivr.net/npm/d3@7.9.0/+esm');
      const projection = geoEqualEarth().scale(1).translate([0, 0]);
      const matrix = document.querySelector('.wm-stage svg>g').transform.baseVal.consolidate().matrix;
      return Math.max(...[...document.querySelectorAll('.wm-metro')].map(dot => {
        const p = projection([dot.__data__.lng, dot.__data__.lat]);
        return Math.hypot(Number(dot.getAttribute('cx')) - (matrix.a * p[0] + matrix.e), Number(dot.getAttribute('cy')) - (matrix.d * p[1] + matrix.f));
      }));
    });
    assert.ok(coordinateError < 0.001); checks.maxMarkerProjectionErrorPx = coordinateError;
    checks.checks.push('93 playable records retain raw values and all meet the 5M minimum; Phoenix retained at exactly 5M; Brasília/Manila/Taipei excluded; display suffix removed; portrait preview defaults correctly; 177 sourced country geometries; all markers match Equal Earth forward projection');
    await page.screenshot({ path: path.join(root, 'docs/validation/equal-earth-world.png'), fullPage: true });
    const dot = f.locator('[data-city-id="delhi"]'); const initialRadius = await dot.getAttribute('r');
    await f.locator('[data-region]').selectOption('south-asia'); await page.waitForTimeout(850);
    assert.ok(Number(await f.locator('.wm-stage').getAttribute('data-zoom')) > 2);
    assert.equal(await f.locator('.wm-metro').count(), 93); assert.equal(await dot.getAttribute('r'), initialRadius);
    await f.locator('[data-city]').selectOption('delhi'); assert.match(await f.locator('.wm-detail').innerText(), /Delhi/);
    const delhi = original.find(m => m.id === 'delhi'); assert.ok((await f.locator('.wm-detail').innerText()).includes('$' + delhi.gdp_usd_billion.toLocaleString('en-US') + 'B'));
    await page.screenshot({ path: path.join(root, 'docs/validation/equal-earth-south-asia.png'), fullPage: true });
    await f.locator('[data-action="world"]').click(); await page.waitForTimeout(800);
    assert.equal(await f.locator('.wm-stage').getAttribute('data-visible-count'), '93');
    const cluster = f.locator('[data-city-id="guangzhou"]'); const stage = await f.locator('.wm-stage').boundingBox();
    await page.mouse.click(stage.x + Number(await cluster.getAttribute('cx')), stage.y + Number(await cluster.getAttribute('cy')));
    assert.ok(await f.locator('.wm-candidates button').count() > 1, 'overlapping markers remain selectable');
    assert.equal(await f.locator('[data-city] option[value="brasilia"]').count(), 0);
    assert.ok(original.every(m => m.population >= 5_000_000));
    await f.locator('[data-city]').selectOption('phoenix'); assert.match(await f.locator('.wm-detail').innerText(), /5\.0M/);
    checks.checks.push('regional zoom uses camera scaling; marker sizes and raw records stay fixed; metro inspection shows source values; world reset; overlapping metro disambiguation');
    for (const width of [352, 422, 800, 1060]) {
      await page.setViewportSize({ width, height: 1100 }); await page.waitForTimeout(100);
      assert.ok(await f.locator('.wm-app').evaluate(e => e.scrollWidth <= e.clientWidth), 'no overflow at ' + width);
      assert.equal(await f.locator('.wm-metro').count(), 93);
      assert.ok(await f.locator('.wm-label').evaluateAll(labels => labels.every(e => parseFloat(getComputedStyle(e).fontSize) >= 12)));
    }
    checks.checks.push('narrow layouts, resize persistence, and labels remain at readable pixel sizes');
    await page.emulateMedia({ colorScheme: 'dark', reducedMotion: 'reduce' });
    await f.locator('[data-region]').selectOption('east-asia'); await page.waitForTimeout(100);
    assert.ok(Number(await f.locator('.wm-stage').getAttribute('data-zoom')) > 1);
    await page.screenshot({ path: path.join(root, 'docs/validation/equal-earth-east-asia-dark.png'), fullPage: true });
    checks.checks.push('bright cheerful theme matches its manifest; dark appearance and reduced-motion regional navigation');

    // Exercise real window aspect ratios in a top-level document as well as
    // the inline preview. The preview host sizes its iframe to its content.
    const fragmentResponse = await page.request.get(new URL('equal-earth-world-map.html', url).href);
    assert.ok(fragmentResponse.ok());
    const full = await browser.newPage({ viewport: { width: 1280, height: 800 }, timezoneId: 'America/New_York', ignoreHTTPSErrors: true, reducedMotion: 'reduce' });
    full.on('pageerror', e => errors.push(e.message));
    await full.setContent(await fragmentResponse.text());
    await full.locator('.wm-metro').first().waitFor();
    await full.locator('[data-preview="auto"]').click();
    const seed = new Map(original.map(m => [m.id, m]));
    const bandState = (target = full) => target.evaluate(async ([south, north]) => {
      const { geoEqualEarth } = await import('https://cdn.jsdelivr.net/npm/d3@7.9.0/+esm');
      const projection = geoEqualEarth().scale(1).translate([0, 0]);
      const stage = document.querySelector('.wm-stage'), matrix = stage.querySelector('svg>g').transform.baseVal.consolidate().matrix;
      return { northY: matrix.d * projection([0, north])[1] + matrix.f, southY: matrix.d * projection([0, south])[1] + matrix.f, height: stage.clientHeight, k: matrix.d };
    }, [latitudeSouth, latitudeNorth]);
    const assertBand = async (label, target = full) => {
      const state = await bandState(target);
      assert.ok(state.northY <= 0.001 && state.southY >= state.height - 0.001, `${label}: portrait camera remains within the latitude band ${JSON.stringify(state)}`);
      return state;
    };

    const expected = {
      landscape: {
        mashreq: ['alexandria', 'tehran', 'riyadh', 'dubai'],
        'north-america': ['toronto', 'new_york', 'mexico_city', 'san_francisco'],
        europe: ['moscow', 'saint_petersburg', 'madrid'],
        africa: ['khartoum', 'luanda', 'abidjan', 'dar_es_salaam'],
        'south-asia': ['bengaluru', 'dhaka', 'lahore', 'riyadh'],
        'east-asia': ['shenyang', 'chengdu', 'hong_kong', 'tokyo'],
        'south-america': ['lima', 'buenos_aires', 'rio_de_janeiro'],
        'south-east-asia': ['ho_chi_minh', 'singapore', 'yangon']
      },
      portrait: {
        europe: ['madrid', 'milan', 'randstad'],
        africa: ['abidjan', 'johannesburg', 'tehran', 'dubai'],
        'south-asia': ['bengaluru', 'dhaka', 'lahore', 'karachi'],
        'east-asia': ['shenyang', 'chengdu', 'hong_kong', 'tokyo', 'yangon'],
        'south-america': ['bogota', 'lima', 'buenos_aires', 'rio_de_janeiro'],
        'north-america-east': ['toronto', 'miami', 'atlanta', 'new_york'],
        'north-america-west': ['san_francisco', 'mexico_city', 'houston'],
        'south-east-asia': ['ho_chi_minh', 'singapore', 'yangon']
      }
    };
    assert.deepEqual(JSON.parse(await full.locator('#wm-region-presets').textContent()), manifest);
    assert.equal(manifest.portrait_limits.cushion_degrees, latitudeCushion);
    checks.orientationCases = [];checks.portraitFraming = [];
    for (const [orientation, viewport] of [['landscape', { width: 1280, height: 800 }], ['portrait', { width: 390, height: 844 }]]) {
      await full.setViewportSize(viewport); await full.waitForTimeout(80);
      assert.equal(await full.locator('.wm-stage').getAttribute('data-orientation'), orientation);
      assert.equal(await full.locator('[data-action="world"]').count(), orientation === 'landscape' ? 1 : 0);
      assert.equal(await full.locator('[data-region] option[value="americas"]').count(), 0);
      assert.equal(await full.locator('[data-region] option[value="europe"]').textContent(), 'Frangistan');
      assert.equal(await full.locator('[data-region] option[value="africa"]').textContent(), orientation === 'landscape' ? 'Africa (East+West)' : 'Africa+Arabia');
      assert.equal(await full.locator('[data-region] option[value="oceania"]').count(), 0);
      assert.equal(await full.locator('[data-region] option[value="mashreq"]').count(), orientation === 'landscape' ? 1 : 0);
      assert.equal(await full.locator('[data-region] option[value="south-east-asia"]').textContent(), 'SouthEast Asia');
      assert.equal(await full.locator('[data-region] option[value="north-america"]').count(), orientation === 'landscape' ? 1 : 0);
      for (const side of ['east', 'west']) assert.equal(await full.locator(`[data-region] option[value="north-america-${side}"]`).count(), orientation === 'portrait' ? 1 : 0);
      if (orientation === 'portrait') assert.equal(await full.locator('[data-region]').inputValue(), 'north-america-east', 'first portrait North America choice defaults East');
      for (const [region, ids] of Object.entries(expected[orientation])) {
        const entry = manifest.regions.find(r => r.id === region), definition = { ...entry, ...entry[orientation] };
        assert.deepEqual((entry[orientation] || entry).metro_ids, ids);
        await full.locator('[data-region]').selectOption(region); await full.waitForTimeout(60);
        const anchors = ids.map(id => seed.get(id));
        const bounds = [Math.min(...anchors.map(m => m.lng)), Math.min(...anchors.map(m => m.lat)), Math.max(...anchors.map(m => m.lng)), Math.max(...anchors.map(m => m.lat))];
        assert.deepEqual(JSON.parse(await full.locator('.wm-stage').getAttribute('data-preset-bounds')), bounds);
        const visible = await full.locator('.wm-stage').evaluate((stage, ids) => ids.every(id => {
          const dot = stage.querySelector(`[data-city-id="${id}"]`), x = Number(dot.getAttribute('cx')), y = Number(dot.getAttribute('cy')), r = Number(dot.getAttribute('r'));
          return x >= r && x <= stage.clientWidth - r && y >= r && y <= stage.clientHeight - r;
        }), ids);
        assert.ok(visible, `${region} ${orientation} includes every anchor and its marker`);
        for (const id of ids) assert.equal(await full.locator(`.wm-label[data-city-label="${id}"]`).count(), 1, `${region} ${orientation} labels ${id}`);
        const fitsRectangle = await full.evaluate(async ({bounds, ids, pointFit}) => {
          const { geoEqualEarth } = await import('https://cdn.jsdelivr.net/npm/d3@7.9.0/+esm');
          const projection = geoEqualEarth().scale(1).translate([0, 0]);
          const stage = document.querySelector('.wm-stage'), matrix = stage.querySelector('svg>g').transform.baseVal.consolidate().matrix;
          const lats = [bounds[1], bounds[3], Math.max(bounds[1], Math.min(bounds[3], 0))];
          const points = pointFit ? ids.map(id => { const d = stage.querySelector(`[data-city-id="${id}"]`); return [Number(d.getAttribute('cx')), Number(d.getAttribute('cy'))]; }) : [bounds[0], bounds[2]].flatMap(lng => lats.map(lat => { const p = projection([lng, lat]); return [matrix.a * p[0] + matrix.e, matrix.d * p[1] + matrix.f]; }));
          const xs = points.map(p => p[0]), ys = points.map(p => p[1]);
          return Math.min(...xs) >= 25.9 && Math.max(...xs) <= stage.clientWidth - 25.9 && Math.min(...ys) >= 20.9 && Math.max(...ys) <= stage.clientHeight - 20.9 && (Math.abs(Math.min(...xs) - 26) < .2 || Math.abs(Math.min(...ys) - 21) < .2);
        }, {bounds, ids, pointFit: definition.fit === 'metro-points'});
        if (region === 'east-asia') {
          assert.ok(await full.locator('[data-city-id="dhaka"]').evaluate(d => Number(d.getAttribute('cx')) + Number(d.getAttribute('r')) < 0), 'Dhaka is outside East Asia camera');
        }
        if (orientation === 'portrait' && region === 'africa') {
          assert.ok(await full.locator('[data-city-id="karachi"]').evaluate(d => Number(d.getAttribute('cx')) - Number(d.getAttribute('r')) > d.closest('.wm-stage').clientWidth), 'Karachi is fully outside Africa+Arabia on the east');
        }
        if (orientation === 'portrait' && ['south-asia', 'north-america-west'].includes(region)) {
          const framing = await full.evaluate(async ({ bounds, region }) => {
            const { geoEqualEarth } = await import('https://cdn.jsdelivr.net/npm/d3@7.9.0/+esm');
            const projection = geoEqualEarth().scale(1).translate([0, 0]);
            const stage = document.querySelector('.wm-stage'), matrix = stage.querySelector('svg>g').transform.baseVal.consolidate().matrix;
            const ps = [bounds[0], bounds[2]].flatMap(lng => [bounds[1], bounds[3], Math.max(bounds[1], Math.min(bounds[3], 0))].map(lat => projection([lng, lat])));
            const dx = Math.max(...ps.map(p => p[0])) - Math.min(...ps.map(p => p[0])), dy = Math.max(...ps.map(p => p[1])) - Math.min(...ps.map(p => p[1]));
            const baselineScale = Math.min((stage.clientWidth - 52) / dx, (stage.clientHeight - 42) / dy);
            return { region, scaleRatio: matrix.a / baselineScale, lahoreY: region === 'south-asia' ? Number(stage.querySelector('[data-city-id="lahore"]').getAttribute('cy')) : null };
          }, { bounds, region });
          assert.ok(framing.scaleRatio > (region === 'south-asia' ? 1.15 : 1.4), `${region} is visibly closer than the previous rectangle fit`);
          if (region === 'south-asia') assert.ok(Math.abs(framing.lahoreY - 21) < .1, 'Lahore sits at the top gutter');
          checks.portraitFraming.push(framing);
        }
        assert.ok(fitsRectangle, `${region} ${orientation} tightly fits its specified projected anchors/bounds`);
        if (orientation === 'portrait') await assertBand(region);
        checks.orientationCases.push({ orientation, region, anchors: ids, bounds });
        if (['mashreq', 'africa', 'south-asia', 'europe', 'north-america', 'north-america-east', 'north-america-west', 'south-east-asia'].includes(region)) await full.screenshot({ path: path.join(root, `docs/validation/equal-earth-${region}-${orientation}.png`), fullPage: true });
      }
      if (orientation === 'landscape') await full.locator('[data-region]').selectOption('north-america');
    }
    // A landscape-only Mashreq selection becomes a geographic camera in portrait.
    await full.setViewportSize({ width: 1280, height: 800 }); await full.waitForTimeout(80);
    await full.locator('[data-region]').selectOption('mashreq');
    const geographicCamera = () => full.evaluate(async () => {
      const { geoEqualEarth } = await import('https://cdn.jsdelivr.net/npm/d3@7.9.0/+esm');
      const stage = document.querySelector('.wm-stage'), matrix = stage.querySelector('svg>g').transform.baseVal.consolidate().matrix;
      return geoEqualEarth().scale(1).translate([0, 0]).invert([(stage.clientWidth / 2 - matrix.e) / matrix.a, (stage.clientHeight / 2 - matrix.f) / matrix.d]);
    });
    const mashreqCenter = await geographicCamera();
    await full.setViewportSize({ width: 390, height: 844 }); await full.waitForTimeout(80);
    assert.equal(await full.locator('[data-region] option[value="mashreq"]').count(), 0);
    assert.equal(await full.locator('[data-region]').inputValue(), '');
    const portraitCenter = await geographicCamera();
    assert.ok(Math.hypot(...mashreqCenter.map((v, i) => v - portraitCenter[i])) < 0.0001, 'Mashreq rotation preserves geographic center');
    // A height-only resize must update labels/bounds even when the map's own
    // fixed width and height do not change. Preserve the selected region.
    await full.setViewportSize({ width: 1000, height: 700 }); await full.waitForTimeout(80);
    await full.locator('[data-region]').selectOption('africa');
    const landscapeBounds = await full.locator('.wm-stage').getAttribute('data-preset-bounds');
    await full.setViewportSize({ width: 1000, height: 1300 }); await full.waitForTimeout(80);
    assert.equal(await full.locator('.wm-stage').getAttribute('data-orientation'), 'portrait');
    assert.equal(await full.locator('[data-region]').inputValue(), 'africa');
    assert.notEqual(await full.locator('.wm-stage').getAttribute('data-preset-bounds'), landscapeBounds);
    assert.equal(await full.locator('[data-region] option:checked').textContent(), 'Africa+Arabia');
    await full.setViewportSize({ width: 1000, height: 700 }); await full.waitForTimeout(80);
    assert.equal(await full.locator('.wm-stage').getAttribute('data-preset-bounds'), landscapeBounds);
    // A stored preset's old camera cannot override its current orientation.
    await full.evaluate(() => dispatchEvent(new CustomEvent('openai:set_globals', { detail: { globals: { widgetState: { privateContent: { worldMap: { selected: 'delhi', region: 'africa', camera: { center: [0, 0], zoom: 1 } } } } } } })));
    assert.equal(await full.locator('.wm-stage').getAttribute('data-preset-bounds'), landscapeBounds);
    await full.setViewportSize({ width: 700, height: 700 }); await full.waitForTimeout(80);
    assert.equal(await full.locator('.wm-stage').getAttribute('data-orientation'), 'landscape');
    // Landscape North America becomes the remembered portrait side; a West
    // selection survives a round trip through the landscape overview.
    await full.setViewportSize({ width: 1000, height: 700 }); await full.waitForTimeout(80);
    await full.locator('[data-region]').selectOption('north-america');
    await full.setViewportSize({ width: 390, height: 844 }); await full.waitForTimeout(80);
    assert.equal(await full.locator('[data-region]').inputValue(), 'north-america-west', 'previously visited West remains remembered');
    await full.locator('[data-region]').selectOption('north-america-west');
    const westernBounds = await full.locator('.wm-stage').getAttribute('data-preset-bounds');
    await full.setViewportSize({ width: 1000, height: 700 }); await full.waitForTimeout(80);
    assert.equal(await full.locator('[data-region]').inputValue(), 'north-america');
    await full.setViewportSize({ width: 390, height: 844 }); await full.waitForTimeout(80);
    assert.equal(await full.locator('[data-region]').inputValue(), 'north-america-west');
    assert.equal(await full.locator('.wm-stage').getAttribute('data-preset-bounds'), westernBounds);
    // Previously saved Oceania presets migrate to their replacement.
    await full.evaluate(() => dispatchEvent(new CustomEvent('openai:set_globals', { detail: { globals: { widgetState: { privateContent: { worldMap: { selected: null, region: 'oceania', camera: { center: [140, -30], zoom: 2 } } } } } } })));
    assert.equal(await full.locator('[data-region]').inputValue(), 'south-east-asia');
    assert.equal(await full.locator('[data-region] option:checked').textContent(), 'SouthEast Asia');
    // A landscape World view enters portrait at the hidden Americas overview.
    await full.setViewportSize({ width: 1280, height: 800 }); await full.waitForTimeout(80);
    await full.locator('[data-action="world"]').click();
    await full.setViewportSize({ width: 390, height: 844 }); await full.waitForTimeout(80);
    assert.equal(await full.locator('[data-action="world"]').count(), 0);
    assert.equal(await full.locator('.wm-stage').getAttribute('data-camera-preset'), 'americas');
    assert.deepEqual(JSON.parse(await full.locator('.wm-stage').getAttribute('data-preset-bounds')), initialBounds);
    // Old saved World views migrate; current manually chosen portrait cameras survive.
    await full.evaluate(() => dispatchEvent(new CustomEvent('openai:set_globals', { detail: { globals: { widgetState: { privateContent: { worldMap: { selected: null, region: null, camera: { center: [0, 0], zoom: 1 }, previewMode: 'portrait' } } } } } })));
    assert.equal(await full.locator('.wm-stage').getAttribute('data-camera-preset'), 'americas');
    await full.evaluate(() => dispatchEvent(new CustomEvent('openai:set_globals', { detail: { globals: { widgetState: { privateContent: { worldMap: { selected: null, region: null, camera: { center: [-100, 25], zoom: 8 }, previewMode: 'portrait', viewPolicyRevision: 7 } } } } } })));
    assert.equal(await full.locator('.wm-stage').getAttribute('data-camera-preset'), '');
    assert.ok(Math.abs(Number(await full.locator('.wm-stage').getAttribute('data-zoom')) - 8) < 0.001);
    const restoredCenter = await geographicCamera();
    assert.ok(Math.hypot(restoredCenter[0] + 100, restoredCenter[1] - 25) < 0.0001);
    await full.setViewportSize({ width: 420, height: 900 }); await full.waitForTimeout(80);
    const resizedCenter = await geographicCamera();
    assert.ok(Math.hypot(resizedCenter[0] + 100, resizedCenter[1] - 25) < 0.0001);
    // Check every rendered animation frame while zooming and panning against the limits.
    await full.emulateMedia({ reducedMotion: 'no-preference' });
    await full.evaluate(async ([south, north]) => {
      const { geoEqualEarth } = await import('https://cdn.jsdelivr.net/npm/d3@7.9.0/+esm');
      const projection = geoEqualEarth().scale(1).translate([0, 0]);
      window.latitudeSamples = { frames: 0, violations: 0, stop: false };
      const tick = () => {
        if (window.latitudeSamples.stop) return;
        const stage = document.querySelector('.wm-stage'), matrix = stage.querySelector('svg>g').transform.baseVal.consolidate().matrix;
        if (stage.dataset.orientation === 'portrait') {
          window.latitudeSamples.frames++;
          if (matrix.d * projection([0, north])[1] + matrix.f > .001 || matrix.d * projection([0, south])[1] + matrix.f < stage.clientHeight - .001) window.latitudeSamples.violations++;
        }
        requestAnimationFrame(tick);
      };
      tick();
    }, [latitudeSouth, latitudeNorth]);
    for (let i = 0; i < 12 && !(await full.locator('[data-action="out"]').isDisabled()); i++) {
      await full.locator('[data-action="out"]').click(); await full.waitForTimeout(300);
      await assertBand('zoom-out button');
    }
    assert.ok(await full.locator('[data-action="out"]').isDisabled());
    const atLimit = await assertBand('minimum zoom');
    assert.ok(Math.abs(atLimit.northY) < .001 && Math.abs(atLimit.southY - atLimit.height) < .001, 'minimum zoom exactly spans the cushioned latitude limits');
    let mapBox = await full.locator('.wm-stage').boundingBox();
    await full.mouse.move(mapBox.x + mapBox.width / 2, mapBox.y + 30);
    await full.mouse.wheel(0, 6000); await full.waitForTimeout(300);
    const afterWheel = await assertBand('wheel overscroll at northern edge');
    assert.ok(Math.abs(afterWheel.k - atLimit.k) < .001);
    await full.locator('[data-action="in"]').click(); await full.waitForTimeout(300);
    const drag = async dy => {
      mapBox = await full.locator('.wm-stage').boundingBox();
      await full.mouse.move(mapBox.x + mapBox.width / 2, mapBox.y + mapBox.height / 2);
      await full.mouse.down(); await full.mouse.move(mapBox.x + mapBox.width / 2, mapBox.y + mapBox.height / 2 + dy, { steps: 8 }); await full.mouse.up();
    };
    await drag(900); await assertBand('pan north');
    await drag(-900); await assertBand('pan south');
    for (const city of ['saint_petersburg', 'melbourne']) {
      await full.locator('[data-city]').selectOption(city);
      await full.locator('[data-action="focus"]').click(); await full.waitForTimeout(750);
      await assertBand(`focus ${city}`);
    }
    // Current saved cameras cannot bypass either boundary or the zoom floor.
    for (const lat of [-80, 80]) {
      await full.evaluate(lat => dispatchEvent(new CustomEvent('openai:set_globals', { detail: { globals: { widgetState: { privateContent: { worldMap: { selected: null, region: null, camera: { center: [0, lat], zoom: 1 }, previewMode: 'portrait', viewPolicyRevision: 7 } } } } } })), lat);
      await assertBand(`restore invalid latitude ${lat}`);
    }
    await full.setViewportSize({ width: 320, height: 844 }); await full.waitForTimeout(100); await assertBand('320px resize');
    await full.setViewportSize({ width: 1000, height: 1300 }); await full.waitForTimeout(100); await assertBand('wide portrait resize');
    const samples = await full.evaluate(() => { window.latitudeSamples.stop = true; return window.latitudeSamples; });
    assert.ok(samples.frames > 30); assert.equal(samples.violations, 0, 'no animation frame exposes latitude beyond the limits');
    checks.portraitLatitudeLimits = { south: latitudeSouth, north: latitudeNorth, cushionDegrees: latitudeCushion, animationFrames: samples.frames, violations: samples.violations };
    // Exercise actual touch pinch events in a touch-capable browser context.
    const touch = await browser.newPage({ viewport: { width: 390, height: 844 }, hasTouch: true, isMobile: true, timezoneId: 'America/New_York', ignoreHTTPSErrors: true, reducedMotion: 'reduce' });
    touch.on('pageerror', e => errors.push(e.message));
    await touch.setContent('<meta name="viewport" content="width=device-width, initial-scale=1">' + await fragmentResponse.text()); await touch.locator('.wm-metro').first().waitFor();
    const cdp = await touch.context().newCDPSession(touch), touchBox = await touch.locator('.wm-stage').boundingBox();
    const cx = touchBox.x + touchBox.width / 2, cy = touchBox.y + touchBox.height / 2;
    const points = d => [{ x: cx - d, y: cy, id: 1 }, { x: cx + d, y: cy, id: 2 }];
    const beforePinch = await bandState(touch);
    await cdp.send('Input.dispatchTouchEvent', { type: 'touchStart', touchPoints: points(100) });
    for (const d of [80, 60, 40, 20, 10]) await cdp.send('Input.dispatchTouchEvent', { type: 'touchMove', touchPoints: points(d) });
    await cdp.send('Input.dispatchTouchEvent', { type: 'touchEnd', touchPoints: [] }); await touch.waitForTimeout(200);
    const afterPinch = await assertBand('touch pinch zoom-out', touch);
    assert.ok(afterPinch.k < beforePinch.k && Math.abs(afterPinch.northY) < .001, 'pinch reaches and respects the zoom floor');
    await touch.close();
    checks.checks.push('portrait latitude limits apply to buttons, edge-focused wheel zoom, touch pinch, north/south pan, metro focus, animation frames, invalid saved cameras, and narrow/wide portrait resize; zoom-out disables at the floor');
    checks.checks.push('landscape World transitions to portrait Americas; old saved World cameras migrate; current manual portrait cameras restore and resize without resetting');
    checks.checks.push('all 16 requested landscape/portrait anchor sets, visible anchor labels, tight projected bounds/anchor fits, Dhaka and Karachi camera exclusions, closer portrait South Asia/Western North America, Lahore top alignment, orientation-specific menus, Mashreq camera-preserving portrait transition, height-only resize, current-orientation state restore, square viewport, remembered North America side, and legacy Oceania migration');
    // Initial defaults depend on the device timezone, without changing saved views.
    checks.timezoneDefaults = [];
    const timezoneFixtures = [
      ['America/New_York', -300, 'americas', '2026-07-15'],
      ['Atlantic/Reykjavik', 0, 'dhaka-edge', '2026-01-15'],
      ['Europe/London', 0, 'dhaka-edge', '2026-01-15'],
      ['Europe/London', 0, 'dhaka-edge', '2026-07-15'],
      ['Asia/Calcutta', 330, 'dhaka-edge', '2026-07-15'],
      ['Asia/Kolkata', 330, 'dhaka-edge', '2026-01-15'],
      ['Asia/Kathmandu', 345, 'karachi-edge', '2026-01-15'],
      ['Asia/Dhaka', 360, 'karachi-edge', '2026-07-15'],
      ['Asia/Yangon', 390, 'karachi-edge', '2026-01-15'],
      ['Asia/Shanghai', 480, 'karachi-edge', '2026-01-15'],
      ['Asia/Tokyo', 540, 'karachi-edge', '2026-07-15'],
      ['Australia/Darwin', 570, 'americas', '2026-07-15'],
      ['Atlantic/Azores', -60, 'americas', '2026-07-15']
    ];
    const fragment = await fragmentResponse.text();
    for (const [timezoneId, offset, view, date] of timezoneFixtures) {
      const target = await browser.newPage({ viewport: { width: 390, height: 844 }, timezoneId, ignoreHTTPSErrors: true, reducedMotion: 'reduce' });
      target.on('pageerror', e => errors.push(e.message));
      await target.clock.setFixedTime(new Date(date + 'T12:00:00Z'));
      await target.setContent(fragment); await target.locator('.wm-metro').first().waitFor();
      assert.equal(await target.locator('.wm-stage').getAttribute('data-standard-utc-offset-minutes'), String(offset), timezoneId);
      assert.equal(await target.locator('.wm-stage').getAttribute('data-camera-preset'), view, timezoneId);
      for (const hidden of ['americas', 'dhaka-edge', 'karachi-edge']) assert.equal(await target.locator(`[data-region] option[value="${hidden}"]`).count(), 0);
      if (view !== 'americas') {
        const edge = view === 'dhaka-edge' ? 'east' : 'west', id = edge === 'east' ? 'dhaka' : 'karachi';
        for (const width of [390, 320, 1000]) {
          await target.setViewportSize({ width, height: width === 1000 ? 1300 : 844 }); await target.waitForTimeout(120);
          const band = await assertBand(`${timezoneId} ${width}`, target);
          assert.ok(Math.abs(band.northY) < .001 && Math.abs(band.southY - band.height) < .001, 'timezone view uses maximum permitted zoom-out');
          assert.ok(await target.locator(`[data-city-id="${id}"]`).evaluate(d => {
            const x = Number(d.getAttribute('cx')), r = Number(d.getAttribute('r')); return x >= r && x <= d.closest('.wm-stage').clientWidth - r;
          }));
          assert.equal(await target.locator(`[data-city-label="${id}"]`).count(), 1);
          const outward = await target.locator('.wm-stage').evaluate((stage, { id, edge }) => {
            const anchor = stage.querySelector(`[data-city-id="${id}"]`).__data__, direction = edge === 'east' ? 1 : -1;
            return [...stage.querySelectorAll('.wm-metro')].filter(d => direction * (d.__data__.lng - anchor.lng) > 0).filter(d => {
              const x = Number(d.getAttribute('cx')), r = Number(d.getAttribute('r')); return edge === 'east' ? x - r <= stage.clientWidth : x + r >= 0;
            }).map(d => d.__data__.id);
          }, { id, edge });
          assert.deepEqual(outward, [], `${timezoneId} keeps outward metros fully offscreen`);
          assert.ok(await target.locator('[data-action="out"]').isDisabled());
        }
        if (['Asia/Kolkata', 'Asia/Tokyo'].includes(timezoneId)) {
          await target.setViewportSize({ width: 390, height: 844 }); await target.waitForTimeout(120);
          await target.screenshot({ path: path.join(root, `docs/validation/equal-earth-${view}-portrait.png`), fullPage: true });
        }
        // A restored explicit preset and a manual camera take precedence.
        await target.evaluate(() => dispatchEvent(new CustomEvent('openai:set_globals', { detail: { globals: { widgetState: { privateContent: { worldMap: { selected: null, region: 'north-america-west', previewMode: 'portrait' } } } } } })));
        assert.equal(await target.locator('[data-region]').inputValue(), 'north-america-west');
        await target.evaluate(() => dispatchEvent(new CustomEvent('openai:set_globals', { detail: { globals: { widgetState: { privateContent: { worldMap: { selected: null, region: null, camera: { center: [-100, 25], zoom: 8 }, previewMode: 'portrait', viewPolicyRevision: 7 } } } } } })));
        assert.equal(await target.locator('.wm-stage').getAttribute('data-camera-preset'), '');
        assert.ok(Math.abs(Number(await target.locator('.wm-stage').getAttribute('data-zoom')) - 8) < .001);
      }
      checks.timezoneDefaults.push({ timezoneId, standardUtcOffsetMinutes: offset, view, date });
      await target.close();
    }
    // Missing timezone APIs use the Americas fallback.
    const unknownTimezone = await browser.newPage({ viewport: { width: 390, height: 844 }, ignoreHTTPSErrors: true });
    await unknownTimezone.addInitScript(() => { Intl.DateTimeFormat = () => { throw new Error('Timezone unavailable'); }; });
    await unknownTimezone.goto(new URL('equal-earth-world-map.html', url).href); await unknownTimezone.locator('.wm-metro').first().waitFor();
    assert.equal(await unknownTimezone.locator('.wm-stage').getAttribute('data-camera-preset'), 'americas');
    await unknownTimezone.close();
    checks.checks.push('timezone portrait defaults: UTC0 to +05:30 inclusive Dhaka east; greater than +05:30 through +09:00 Karachi west; fractional offsets, aliases, summer/winter stability, unknown/outside fallback, maximum zoom-out, edge metro labels, outward metro exclusion at 320/390/1000px, and saved-view precedence');
    await full.close();
    assert.deepEqual(errors, []); checks.passed = true;
    fs.writeFileSync(path.join(root, 'docs/validation/map-checks.json'), JSON.stringify(checks, null, 2) + '\n'); console.log(JSON.stringify(checks, null, 2));
  } finally { await browser.close(); }
})().catch(error => { console.error(error); process.exitCode = 1; });
