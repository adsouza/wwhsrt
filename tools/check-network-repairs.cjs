const { control, language } = require('./ui-helpers.cjs');
const assert = require('node:assert/strict'),
  fs = require('node:fs'),
  path = require('node:path'),
  crypto = require('node:crypto');
const { execFileSync } = require('node:child_process'),
  { chromium } = require('playwright');
const mode = process.argv[3] || 'desktop';
assert(['desktop', 'phone', 'offline'].includes(mode));
const snap = (p) => p.evaluate(() => railSnapshot()),
  url = process.argv[2] || 'http://127.0.0.1:8765/';
const sha = (p) => crypto.createHash('sha256').update(fs.readFileSync(p)).digest('hex');
async function ready(p) {
  await p.goto(url);
  await p.waitForFunction(() => !!window.railSnapshot, null, { timeout: 60000 });
}
function route(f, a, b) {
  return f.routes
    .filter(
      (r) => r.open && ((r.cities[0] === a && r.cities.at(-1) === b) || (r.cities[0] === b && r.cities.at(-1) === a)),
    )
    .sort((a, b) => a.minutes - b.minutes)[0];
}
async function act(p, id, trace, c) {
  const r = (await snap(p)).frame.revision;
  await (await control(p, '#' + id)).click();
  await p.waitForFunction(async (r) => (await railSnapshot()).frame.revision === r + 1, r);
  await p.waitForFunction(() => document.getElementById('game').getAttribute('aria-busy') === 'false');
  trace.push(c);
}
function screen(c, q) {
  const t = Math.asin((Math.sqrt(3) / 2) * Math.sin((q.lat * Math.PI) / 180)),
    t2 = t * t,
    t6 = t2 * t2 * t2,
    x =
      (2 * Math.sqrt(3) * ((q.lon * Math.PI) / 180) * Math.cos(t)) /
      (3 * (1.340264 - 3 * 0.081106 * t2 + t6 * (7 * 0.000893 + 9 * 0.003796 * t2))),
    y = t * (1.340264 - 0.081106 * t2 + t6 * (0.000893 + 0.003796 * t2)),
    z =
      Math.min(Math.max(1, c.Width - 72) / (c.Max.X - c.Min.X), Math.max(1, c.Height - 80) / (c.Max.Y - c.Min.Y)) *
      c.Zoom;
  return [
    c.Width / 2 + (x - (c.Max.X + c.Min.X) / 2) * z + c.PanX,
    c.Height / 2 - (y - (c.Max.Y + c.Min.Y) / 2) * z + c.PanY,
  ];
}
async function pixels(p, r) {
  const s = await snap(p),
    parts = r.sections.flatMap((id) => {
      const a = s.frame.corridors.find((c) => c.id === id).alignments[0].path;
      return a.slice(1).map((q, i) => {
        const x = screen(s.camera, a[i]),
          y = screen(s.camera, q);
        return [(x[0] + y[0]) / 2, (x[1] + y[1]) / 2];
      });
    }),
    box = await p.locator('#map-frame').boundingBox(),
    png = await p.screenshot({ clip: box }),
    ink = s.palette.draft;
  return p.evaluate(
    async (q) => {
      const im = await createImageBitmap(
          new Blob([Uint8Array.from(atob(q.png), (c) => c.charCodeAt(0))], { type: 'image/png' }),
        ),
        cv = new OffscreenCanvas(im.width, im.height),
        cx = cv.getContext('2d');
      cx.drawImage(im, 0, 0);
      const d = cx.getImageData(0, 0, im.width, im.height).data;
      let hits = 0;
      for (let [x, y] of q.points) {
        x = Math.round(x);
        y = Math.round(y);
        if (x < 4 || y < 4 || x >= im.width - 4 || y >= im.height - 4) continue;
        let ok = false;
        for (let dx = -3; dx <= 3; dx++)
          for (let dy = -3; dy <= 3; dy++) {
            const i = ((y + dy) * im.width + x + dx) * 4;
            if (q.ink.every((v, n) => Math.abs(d[i + n] - v) < 5)) ok = true;
          }
        if (ok) hits++;
      }
      return hits;
    },
    { png: png.toString('base64'), points: parts, ink: [ink.R, ink.G, ink.B] },
  );
}
(async () => {
  const options = { headless: true, args: ['--enable-unsafe-swiftshader', '--js-flags=--max-old-space-size=256'] };
  let profile;
  if (mode === 'phone') {
    profile = fs.mkdtempSync(path.resolve('build/network-repair/phone-profile-'));
    fs.writeFileSync('build/network-repair/phone-profile-path.json', JSON.stringify(profile));
  }
  if (mode === 'offline') {
    profile = JSON.parse(fs.readFileSync('build/network-repair/phone-profile-path.json'));
    assert.equal(path.dirname(profile), path.resolve('build/network-repair'));
    assert(path.basename(profile).startsWith('phone-profile-'));
  }
  const b =
      mode === 'desktop'
        ? await chromium.launch(options)
        : await chromium.launchPersistentContext(profile, {
            ...options,
            viewport: { width: 390, height: 844 },
            reducedMotion: 'reduce',
          }),
    timer = setTimeout(() => b.close(), 200000),
    result = {
      date: new Date().toISOString(),
      browser: (mode === 'desktop' ? b : b.browser()).version(),
      wasm_sha256: sha('web/main.wasm'),
      browser_js_heap_limit_mib: 256,
      catalog_sha256: sha('assets/scenario/public-network.json'),
      checks: [],
      limitations: [
        'Synthetic browser input and emulated phone; native OS interaction and physical devices not qualified.',
        '35 remaining proximity sites are unverified; station anchors are independently reviewed; ' +
          'published legacy game timings remain unchanged.',
        'Cold offline continuation uses a fresh browser process with the same isolated profile; ' +
          'repeated in-process WASM reload memory is not qualified.',
      ],
    };
  let peakRSS = 0,
    limited = false;
  const memory = setInterval(() => {
    const rows = execFileSync('ps', ['-axo', 'pid=,ppid=,rss='], { encoding: 'utf8' })
      .trim()
      .split('\n')
      .map((l) => l.trim().split(/\s+/).map(Number));
    const ids = new Set([process.pid]);
    let added = true;
    while (added) {
      added = false;
      for (const [id, parent] of rows)
        if (ids.has(parent) && !ids.has(id)) {
          ids.add(id);
          added = true;
        }
    }
    const rss = rows.filter(([id]) => ids.has(id)).reduce((s, r) => s + r[2], 0);
    peakRSS = Math.max(peakRSS, rss);
    if (rss > 2 * 1024 * 1024) {
      console.error(rows.filter(([id]) => ids.has(id)).map((r) => ({ pid: r[0], rss_mib: Math.ceil(r[2] / 1024) })));
      limited = true;
      console.error('Checker memory cap exceeded:', Math.ceil(rss / 1024), 'MiB');
      b.close();
    }
  }, 500);
  try {
    const errors = [],
      trace = [];
    let ctx, p;
    if (mode === 'desktop') {
      ctx = await b.newContext({ viewport: { width: 1280, height: 900 }, reducedMotion: 'reduce' });
      p = await ctx.newPage();
      p.on('pageerror', (e) => errors.push(e.message));
      await ready(p);
      let f = (await snap(p)).frame;
      const expected = JSON.parse(fs.readFileSync('data/imports/network-connectivity-expectations.v1.json'));
      for (const pair of expected.pairs) assert(route(f, pair.from, pair.to), pair.from + ' ' + pair.to);
      result.checks.push(
        'All seven expected city pairs have an operational direct/through service pattern in the actual browser frame',
      );
      const jp = route(f, 'sendai', 'shin_hakodate_hokuto');
      await (await control(p, '#service-corridor')).selectOption(jp.id);
      await (await control(p, '#focus-route')).click();
      for (const id of ['dock-map-tools', 'phone-map-tools'])
        if (await p.locator('#' + id).evaluate((e) => e.open)) await p.locator('#' + id + '>summary').click();
      await p.waitForFunction(() => document.getElementById('forecast-values').children.length === 3);
      await p.waitForTimeout(150);
      result.japan_highlight_samples = await pixels(p, jp);
      assert(result.japan_highlight_samples > 8);
      await p.screenshot({ path: 'docs/validation/network-repairs-japan.png' });
      await p.locator('#frequency').fill('1');
      await p.locator('#fare').fill(String(jp.suggested_fare / 100));
      await p.locator('#fare').blur();
      const q = { corridor_id: jp.id, frequency: 1, fare: jp.suggested_fare },
        forecast = await p.evaluate((q) => call('forecast', q), q);
      assert(!forecast.error);
      for (let i = 0; i < forecast.forecast.fleet_needed; i++)
        await act(p, 'lease-train', trace, { kind: 'lease_train', corridor_id: jp.id });
      await p.waitForFunction(() => !document.getElementById('apply-service').disabled);
      await act(p, 'apply-service', trace, { kind: 'set_service', ...q });
      await act(p, 'advance', trace, { kind: 'advance_month' });
      f = (await snap(p)).frame;
      assert(
        f.network.journeys.some(
          (j) =>
            ((j.from === 'sendai' && j.to === 'shin_hakodate_hokuto') ||
              (j.to === 'sendai' && j.from === 'shin_hakodate_hokuto')) &&
            j.passengers > 0,
        ),
      );
      fs.writeFileSync('build/network-repair/commands.json', JSON.stringify(trace));
      const native = JSON.parse(
        execFileSync(
          path.resolve('build/headless'),
          ['-commands', path.resolve('build/network-repair/commands.json')],
          { encoding: 'utf8', maxBuffer: 32 * 1024 * 1024, stdio: ['ignore', 'pipe', 'pipe'] },
        ),
      );
      assert.equal(native.checkpoint, f.checkpoint);
      assert.deepEqual(native.network, f.network);
      result.checkpoint = f.checkpoint;
      result.checks.push(
        'The repaired Sendai–Shin-Hakodate-Hokuto path is visibly highlighted, ' +
          'can lease/apply a service and earns real monthly passenger revenue; ' +
          'native/WASM checkpoints and detailed reports agree',
      );
      const europe = route(f, 'london', 'paris');
      assert(europe.connector_km > 0);
      await (await control(p, '#service-corridor')).selectOption(europe.id);
      await (await control(p, '#focus-route')).click();
      if (await p.locator('#dock-map-tools').evaluate((e) => e.open))
        await p.locator('#dock-map-tools>summary').click();
      await (await control(p, '#connector-note')).scrollIntoViewIfNeeded();
      assert((await p.locator('#connector-note').innerText()).includes('Conventional'));
      await p.screenshot({ path: 'docs/validation/network-repairs-frangistan.png' });
      assert.equal((await snap(p)).frame.checkpoint, f.checkpoint);
      await ctx.close();
    }
    if (mode === 'phone') {
      ctx = b;
      for (const page of ctx.pages()) await page.close();
      p = await ctx.newPage();
      p.on('pageerror', (e) => errors.push(e.message));
      const raw = fs.readFileSync('internal/application/testdata/topology-v5.json', 'utf8'),
        old = JSON.parse(raw).payload;
      await ctx.route('**/network-repair-seed', (r) =>
        r.fulfill({ contentType: 'text/html', body: '<!doctype html><title>Isolated save fixture</title>' }),
      );
      await p.goto(new URL('network-repair-seed', url).href);
      await p.evaluate(
        (raw) =>
          new Promise((resolve, reject) => {
            const r = indexedDB.open('wwhsrt-first-business', 1);
            r.onupgradeneeded = () => r.result.createObjectStore('snapshots');
            r.onerror = () => reject(r.error);
            r.onsuccess = () => {
              const db = r.result,
                tx = db.transaction('snapshots', 'readwrite');
              tx.objectStore('snapshots').put(raw, 'campaign');
              tx.oncomplete = () => {
                db.close();
                resolve();
              };
              tx.onerror = () => reject(tx.error);
            };
          }),
        raw,
      );
      await ready(p);
      let f = (await snap(p)).frame;
      assert.deepEqual(f.reports, old.reports);
      assert.deepEqual(f.services, old.services);
      assert.equal(await p.evaluate(() => railStorageLoad()), raw);
      await language(p, 'zh-Hans');
      await (await control(p, '#service-corridor')).selectOption(route(f, 'changchun', 'shenyang').id);
      await (await control(p, '#focus-route')).click();
      if (await p.locator('#phone-map-tools').evaluate((e) => e.open))
        await p.locator('#phone-map-tools>summary').click();
      assert(await p.evaluate(() => document.documentElement.scrollWidth <= innerWidth));
      await p.screenshot({ path: 'docs/validation/network-repairs-phone-zh.png' });
      await (await control(p, '#tab-report')).click();
      await act(p, 'advance', [], { kind: 'advance_month' });
      f = (await snap(p)).frame;
      assert.deepEqual(f.reports.slice(0, 2), old.reports);
      const save = JSON.parse(await p.evaluate(() => railStorageLoad()));
      assert.equal(save.version, 18);
      assert.equal(save.payload.topology_since, 2);
      assert.equal(save.payload.prior_public_sha256, sha('assets/scenario/public-network-v1.json'));
      await p.evaluate(() => navigator.serviceWorker.ready);
      await p.waitForFunction(() => !!navigator.serviceWorker.controller);
      await ctx.setOffline(true);
      await act(p, 'advance', [], { kind: 'advance_month' });
      f = (await snap(p)).frame;
      assert.deepEqual(f.reports.slice(0, 2), old.reports);
      result.checkpoint = f.checkpoint;
      assert.deepEqual(errors, []);
      result.checks.push(
        'Chinese phone route selection reflows; a genuine v5 save is seeded before loading Go, ' +
          'preserves original reports/plans and bytes on read, ' +
          'commits v7 with an explicit topology cutover, and advances offline',
      );
    }
    if (mode === 'offline') {
      ctx = b;
      for (const page of ctx.pages()) await page.close();
      await ctx.setOffline(true);
      p = await ctx.newPage();
      p.on('pageerror', (e) => errors.push(e.message));
      await ready(p);
      let f = (await snap(p)).frame;
      const before = JSON.parse(fs.readFileSync('docs/validation/network-repair-phone-checks.json')),
        old = JSON.parse(fs.readFileSync('internal/application/testdata/topology-v5.json')).payload;
      assert.equal(f.checkpoint, before.checkpoint);
      assert.deepEqual(f.reports.slice(0, 2), old.reports);
      await act(p, 'advance', [], { kind: 'advance_month' });
      f = (await snap(p)).frame;
      assert.deepEqual(f.reports.slice(0, 2), old.reports);
      result.checkpoint = f.checkpoint;
      assert.deepEqual(errors, []);
      result.checks.push(
        'A new browser process cold-loads the cached app and repaired v7 save offline from the same isolated profile, ' +
          'preserves the exact checkpoint and historical v5 reports, ' +
          'and resolves another business month',
      );
    }
    assert(!limited, '2 GiB checker memory limit');
    result.passed = true;
    result.phase = mode;
    result.peak_rss_mib = Math.ceil(peakRSS / 1024);
    fs.writeFileSync('docs/validation/network-repair-' + mode + '-checks.json', JSON.stringify(result, null, 2) + '\n');
    console.log(JSON.stringify(result, null, 2));
  } finally {
    if (limited) {
      result.passed = false;
      result.status = 'memory_limit_stopped';
      result.phase = mode;
      result.peak_rss_mib = Math.ceil(peakRSS / 1024);
      fs.writeFileSync(
        'docs/validation/network-repair-' + mode + '-limited-checks.json',
        JSON.stringify(result, null, 2) + '\n',
      );
    }
    clearInterval(memory);
    clearTimeout(timer);
    await b.close();
  }
})().catch((e) => {
  console.error(e);
  process.exitCode = 1;
});
