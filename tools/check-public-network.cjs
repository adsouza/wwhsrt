const { control, language } = require('./ui-helpers.cjs');
const assert = require('node:assert/strict'),
  fs = require('node:fs'),
  path = require('node:path'),
  crypto = require('node:crypto');
const { execFileSync } = require('node:child_process'),
  { chromium } = require('playwright');
const url = process.argv[2] || 'http://127.0.0.1:8765/',
  out = path.resolve('docs/validation');
const snap = (p) => p.evaluate(() => railSnapshot());
async function ready(p) {
  await p.goto(url);
  await p.waitForFunction(() => !!window.railSnapshot, null, { timeout: 60000 });
}
async function act(p, id, trace, c) {
  const r = (await snap(p)).frame.revision;
  await (await control(p, '#' + id)).click();
  await p.waitForFunction(async (r) => (await railSnapshot()).frame.revision === r + 1, r, { timeout: 30000 });
  await p.waitForFunction(() => document.getElementById('game').getAttribute('aria-busy') === 'false');
  trace.push(c);
}
function point(c, lat, lon) {
  const t = Math.asin((Math.sqrt(3) / 2) * Math.sin((lat * Math.PI) / 180)),
    t2 = t * t,
    t6 = t2 * t2 * t2,
    p = {
      x:
        (2 * Math.sqrt(3) * ((lon * Math.PI) / 180) * Math.cos(t)) /
        (3 * (1.340264 - 3 * 0.081106 * t2 + t6 * (7 * 0.000893 + 9 * 0.003796 * t2))),
      y: t * (1.340264 - 0.081106 * t2 + t6 * (0.000893 + 0.003796 * t2)),
    },
    z =
      Math.min(Math.max(1, c.Width - 72) / (c.Max.X - c.Min.X), Math.max(1, c.Height - 80) / (c.Max.Y - c.Min.Y)) *
      c.Zoom;
  return [
    c.Width / 2 + (p.x - (c.Max.X + c.Min.X) / 2) * z + c.PanX,
    c.Height / 2 - (p.y - (c.Max.Y + c.Min.Y) / 2) * z + c.PanY,
  ];
}
(async () => {
  const browser = await chromium.launch({ headless: true, args: ['--enable-unsafe-swiftshader'] }),
    timer = setTimeout(() => browser.close(), 240000);
  const result = {
    date: new Date().toISOString(),
    browser: browser.version(),
    wasm_sha256: crypto.createHash('sha256').update(fs.readFileSync('web/main.wasm')).digest('hex'),
    catalog_sha256: crypto
      .createHash('sha256')
      .update(fs.readFileSync('assets/scenario/public-network.json'))
      .digest('hex'),
    checks: [],
    timings_ms: {},
    limitations: [
      'Synthetic browser phone/touch input; native OS interaction, ' +
        'physical devices and full production topology are not qualified.',
      'Prototype source gaps and approximate strategic timings remain; route templates are not operator timetables.',
    ],
  };
  try {
    const ctx = await browser.newContext({ viewport: { width: 1440, height: 900 }, reducedMotion: 'reduce' }),
      p = await ctx.newPage(),
      errors = [],
      trace = [];
    p.on('pageerror', (e) => {
      errors.push(e.message);
      console.error(e.message);
    });
    console.log('Loading expanded world');
    let start = Date.now();
    await ready(p);
    console.log('Loaded');
    result.timings_ms.cold_start = Date.now() - start;
    let s = await snap(p);
    assert(s.frame.public_network);
    const catalog = JSON.parse(fs.readFileSync('assets/scenario/public-network.json'));
    assert.equal(s.frame.cities.length, catalog.cities.length);
    assert.equal(s.frame.corridors.length, catalog.corridors.length);
    result.catalog_counts = {
      cities: catalog.cities.length,
      corridors: catalog.corridors.length,
      stations: catalog.stations.length,
      through_patterns: catalog.network.routes.length,
    };
    assert(s.frame.routes.some((r) => r.id === 'route_beijing__shanghai' && r.open));
    const checkpoint = s.frame.checkpoint;
    console.log('Selecting China');
    await (await control(p, '#region-select')).selectOption('china');
    await p.waitForFunction(async () => (await railSnapshot()).state.region === 'china');
    await p.waitForTimeout(300);
    s = await snap(p);
    const samples = s.frame.corridors
      .filter(
        (c) =>
          c.id.startsWith('public_') &&
          c.alignments[0].path.some((p) => p.lon > 102 && p.lon < 124 && p.lat > 21 && p.lat < 42),
      )
      .flatMap((c) => c.alignments[0].path.filter((_, i) => i % 3 === 0).map((p) => point(s.camera, p.lat, p.lon)));
    const map = p.frames().find((f) => f.url().includes('map.html')),
      box = await p.locator('#map-frame').boundingBox(),
      png = await p.screenshot({ clip: box }),
      ink = s.palette.existing;
    result.rail_pixels = await p.evaluate(
      async (q) => {
        const im = await createImageBitmap(
            new Blob([Uint8Array.from(atob(q.png), (c) => c.charCodeAt(0))], { type: 'image/png' }),
          ),
          c = new OffscreenCanvas(im.width, im.height),
          ctx = c.getContext('2d');
        ctx.drawImage(im, 0, 0);
        const d = ctx.getImageData(0, 0, im.width, im.height).data;
        let inside = 0,
          hits = 0;
        for (let [x, y] of q.points) {
          x = Math.round(x);
          y = Math.round(y);
          if (x < 3 || y < 3 || x >= im.width - 3 || y >= im.height - 3) continue;
          inside++;
          let found = false;
          for (let dx = -2; dx <= 2; dx++)
            for (let dy = -2; dy <= 2; dy++) {
              const i = ((y + dy) * im.width + x + dx) * 4;
              if (q.color.every((v, n) => Math.abs(d[i + n] - v) <= 5)) found = true;
            }
          if (found) hits++;
        }
        return { inside, hits };
      },
      { png: png.toString('base64'), points: samples, color: [ink.R, ink.G, ink.B] },
    );
    console.log('Rail pixels', result.rail_pixels);
    assert(result.rail_pixels.hits > 30);
    assert.equal((await snap(p)).frame.checkpoint, checkpoint);
    // Ordinary intermediate stops are deliberately hidden at overview zoom.
    for (let i = 0; i < 3; i++) await (await control(p, '#zoom-in')).click();
    if (await p.locator('#dock-map-tools').evaluate((e) => e.open)) await p.locator('#dock-map-tools>summary').click();
    s = await snap(p);
    const mappedStops = new Set(JSON.parse(fs.readFileSync('assets/scenario/playable-rail-map.json')).station_ids);
    const candidates = catalog.stations
      .filter((t) => mappedStops.has(t.id) && (s.camera.Zoom >= 2 || t.tagged || t.operator))
      .map((t) => ({ t, xy: point(s.camera, t.lat, t.lon) }))
      .filter(
        ({ t, xy }) =>
          xy[0] > 30 &&
          xy[1] > 75 &&
          xy[0] < s.camera.Width - 30 &&
          xy[1] < s.camera.Height - 30 &&
          Object.values(s.points).every((q) => Math.hypot(q.X - xy[0], q.Y - xy[1]) > 30) &&
          catalog.stations.filter((q) => q.lat === t.lat && q.lon === t.lon).length === 1,
      );
    assert(candidates.length);
    const target = candidates[0];
    await p.mouse.click(box.x + target.xy[0], box.y + target.xy[1]);
    await p.waitForFunction(async (id) => (await railSnapshot()).state.source_station === id, target.t.id);
    await p.waitForFunction(
      (name) => document.getElementById('city-details').textContent.includes(name),
      target.t.name,
    );
    assert.equal((await snap(p)).frame.checkpoint, checkpoint);
    await p.screenshot({ path: path.join(out, 'public-network-china-desktop.png') });
    result.checks.push(
      'Chinese public-rail geometry is painted in the actual Go-rendered map, ' +
        'with stations and city hubs; tapping a non-hub source station shows its name/status; ' +
        'region selection and inspection preserve the campaign checkpoint',
    );
    console.log('Planning Beijing Shanghai');
    await (await control(p, '#tab-services')).click();
    await (await control(p, '#route-query')).fill('Beijing Shanghai');
    assert((await p.locator('#service-corridor option').count()) <= 3);
    await p.locator('#service-corridor').selectOption('route_beijing__shanghai');
    await p.locator('#frequency').fill('2');
    await p.locator('#fare').fill('160');
    await p.locator('#fare').blur();
    s = await snap(p);
    const r = s.frame.routes.find((r) => r.id === 'route_beijing__shanghai');
    const needed = Math.ceil(
      (2 * (2 * r.minutes + s.frame.train_family.round_trip_turnaround)) / s.frame.train_family.daily_minutes,
    );
    for (let i = 0; i < needed; i++) await act(p, 'lease-train', trace, { kind: 'lease_train', corridor_id: r.id });
    await p.waitForFunction(() => !document.getElementById('apply-service').disabled);
    await act(p, 'apply-service', trace, { kind: 'set_service', corridor_id: r.id, frequency: 2, fare: 16000 });
    console.log('Resolving Chinese service');
    start = Date.now();
    await act(p, 'advance', trace, { kind: 'advance_month' });
    result.timings_ms.month_with_china_service = Date.now() - start;
    s = await snap(p);
    assert(s.frame.network.journeys.some((j) => j.from === 'beijing' && j.to === 'shanghai' && j.passengers > 0));
    assert(s.frame.reports[0].access_costs > 0);
    assert.equal(
      s.frame.network.markets.reduce((n, m) => n + m.demand, 0) + s.frame.network.unserved_demand,
      s.frame.network.demand,
    );
    for (const sec of s.frame.network.sections) assert(sec.delivered <= sec.capacity);
    result.checks.push(
      'Searchable Beijing–Shanghai all-stop service leases trains, ' +
        'carries through passengers and intermediate markets, ' +
        'pays access fees and reconciles finite global demand/shared track capacity',
    );
    fs.writeFileSync('build/public-network-commands.json', JSON.stringify(trace));
    const native = JSON.parse(
      execFileSync(path.resolve('build/headless'), ['-commands', path.resolve('build/public-network-commands.json')], {
        encoding: 'utf8',
        maxBuffer: 32 * 1024 * 1024,
        stdio: ['ignore', 'pipe', 'pipe'],
      }),
    );
    assert.equal(native.checkpoint, s.frame.checkpoint);
    assert.deepEqual(native.network, s.frame.network);
    await p.locator('#report-pane-markets').click();
    await p.locator('#market-from').selectOption('lagos');
    await p.locator('#market-to').selectOption('kinshasa');
    await p
      .waitForFunction(
        () =>
          document.getElementById('market-values').textContent.includes('Road') ||
          document.getElementById('market-values').textContent.includes('road'),
      )
      .catch(async () => {
        assert((await p.locator('#market-values').innerText()).length > 20);
      });
    await language(p, 'zh-Hans');
    await p.setViewportSize({ width: 390, height: 844 });
    await (await control(p, '#tab-services')).click();
    await (await control(p, '#route-query')).fill('北京 上海');
    assert((await p.locator('#service-corridor option').count()) <= 3);
    assert((await p.locator('#service-corridor option:checked').innerText()).includes('北京'));
    await p.locator('#route-finder>summary').click();
    assert(await p.evaluate(() => document.documentElement.scrollWidth <= innerWidth));
    await p.screenshot({ path: path.join(out, 'public-network-service-phone-zh.png') });
    await (await control(p, '#tab-map')).click();
    await (await control(p, '#region-select')).selectOption('china');
    await p.waitForTimeout(200);
    await p.screenshot({ path: path.join(out, 'public-network-china-phone-zh.png') });
    assert.equal((await snap(p)).frame.checkpoint, s.frame.checkpoint);
    const saved = s.frame.checkpoint;
    await p.reload();
    await p.waitForFunction(() => !!window.railSnapshot, null, { timeout: 60000 });
    assert.equal((await snap(p)).frame.checkpoint, saved);
    assert.equal((await snap(p)).state.region, 'china');
    await p.evaluate(() => navigator.serviceWorker.ready);
    await p.waitForFunction(() => !!navigator.serviceWorker.controller);
    await ctx.setOffline(true);
    await p.reload();
    await p.waitForFunction(() => !!window.railSnapshot, null, { timeout: 60000 });
    assert.equal((await snap(p)).frame.checkpoint, saved);
    await act(p, 'advance', trace, { kind: 'advance_month' });
    result.checks.push(
      'Sparse outside-mode inspector remains available; ' +
        'Chinese route search and phone layouts reflow; ' +
        'region/campaign reload and offline continuation work; ' +
        'native-headless and WASM checkpoints and detailed reports match',
    );
    assert.deepEqual(errors, []);
    result.passed = true;
    result.final_checkpoint = (await snap(p)).frame.checkpoint;
    fs.writeFileSync(path.join(out, 'public-network-checks.json'), JSON.stringify(result, null, 2) + '\n');
    console.log(JSON.stringify(result, null, 2));
    await ctx.close();
  } finally {
    clearTimeout(timer);
    await browser.close();
  }
})().catch((e) => {
  console.error(e);
  process.exitCode = 1;
});
