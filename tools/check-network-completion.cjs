const { control, language } = require('./ui-helpers.cjs');
const assert = require('node:assert/strict'),
  fs = require('node:fs'),
  path = require('node:path'),
  crypto = require('node:crypto');
const { execFileSync } = require('node:child_process'),
  { chromium } = require('playwright');

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
  fs.mkdirSync('build/network-completion', { recursive: true });
  const result = {
    date: new Date().toISOString(),
    catalog_sha256: sha('assets/scenario/public-network.json'),
    wasm_sha256: sha('web/main.wasm'),
    passed: false,
    checks: [],
    limitations: [
      'Synthetic desktop browser interaction; physical phones and native OS input are not qualified.',
      'Source/operator completeness outside the declared network manifest remains unqualified.',
    ],
  };
  const b = await chromium.launch({
    headless: true,
    args: ['--enable-unsafe-swiftshader', '--js-flags=--max-old-space-size=256'],
  });
  let peak = 0,
    limited = false;
  const timer = setTimeout(() => b.close(), 180000);
  const monitor = setInterval(() => {
    const rows = execFileSync('ps', ['-axo', 'pid=,ppid=,rss='], { encoding: 'utf8' })
      .trim()
      .split('\n')
      .map((l) => l.trim().split(/\s+/).map(Number));
    const ids = new Set([process.pid]);
    let more = true;
    while (more) {
      more = false;
      for (const [id, parent] of rows)
        if (ids.has(parent) && !ids.has(id)) {
          ids.add(id);
          more = true;
        }
    }
    const rss = rows.filter(([id]) => ids.has(id)).reduce((s, r) => s + r[2], 0);
    peak = Math.max(peak, rss);
    if (rss > 2 * 1024 * 1024 && !limited) {
      limited = true;
      console.error('Checker RSS cap exceeded', Math.ceil(rss / 1024), 'MiB');
      b.close();
    }
  }, 500);
  try {
    const ctx = await b.newContext({ viewport: { width: 1280, height: 900 }, reducedMotion: 'reduce' }),
      p = await ctx.newPage(),
      errors = [],
      trace = [];
    p.on('pageerror', (e) => errors.push(e.message));
    await ready(p);
    let f = (await snap(p)).frame;
    const manifest = JSON.parse(fs.readFileSync('data/imports/operating-networks.v2.json'));
    for (const net of manifest.networks) {
      const adj = new Map();
      for (const c of f.corridors)
        if (c.existing) {
          for (const [a, b] of [
            [c.from_city, c.to_city],
            [c.to_city, c.from_city],
          ]) {
            if (!adj.has(a)) adj.set(a, []);
            adj.get(a).push(b);
          }
        }
      const seen = new Set([net.cities[0]]),
        q = [net.cities[0]];
      for (const n of q)
        for (const b of adj.get(n) || [])
          if (!seen.has(b)) {
            seen.add(b);
            q.push(b);
          }
      for (const c of net.cities) assert(seen.has(c), net.id + ' missing ' + c);
      if (net.ordered) {
        const declared = JSON.parse(
            fs.readFileSync('assets/scenario/public-network.json'),
          ).operating_network_checks.find((c) => c.id === net.id),
          r = f.routes.find((r) => r.id === declared.service_id);
        assert(r);
        assert(
          r.stops.join('|') === net.cities.join('|') ||
            (r.stops.length === 2 && [...r.stops].reverse().join('|') === net.cities.join('|')),
        );
      }
    }
    result.checks.push(
      'All ten operating networks and ordered whole-network services are present in the actual WASM frame',
    );
    const nec = f.routes.find((r) => r.id === 'network_nec_all_stops');
    assert(nec.open);
    await (await control(p, '#service-corridor')).selectOption(nec.id);
    await (await control(p, '#focus-route')).click();
    for (const id of ['dock-map-tools', 'phone-map-tools'])
      if (await p.locator('#' + id).evaluate((e) => e.open)) await p.locator('#' + id + '>summary').click();
    await p.waitForTimeout(150);
    result.nec_highlight_samples = await pixels(p, nec);
    assert(result.nec_highlight_samples > 8);
    await p.screenshot({ path: 'docs/validation/network-completion-nec.png' });
    await (await control(p, '#frequency')).fill('1');
    await (await control(p, '#fare')).fill('10');
    await p.locator('#fare').blur();
    const q = { corridor_id: nec.id, frequency: 1, fare: 1000 },
      fc = await p.evaluate((q) => call('forecast', q), q);
    assert(!fc.error);
    for (let i = 0; i < fc.forecast.additional_leases; i++)
      await act(p, 'lease-train', trace, { kind: 'lease_train', corridor_id: nec.id });
    await p.waitForFunction(() => !document.getElementById('apply-service').disabled);
    await act(p, 'apply-service', trace, { kind: 'set_service', ...q });
    await act(p, 'advance', trace, { kind: 'advance_month' });
    f = (await snap(p)).frame;
    assert(
      f.network.journeys.some(
        (j) =>
          ((j.from === 'washington' && j.to === 'boston') || (j.to === 'washington' && j.from === 'boston')) &&
          j.passengers > 0,
      ),
    );
    assert(
      f.network.journeys.some(
        (j) =>
          ((j.from === 'philadelphia' && j.to === 'washington') ||
            (j.to === 'philadelphia' && j.from === 'washington')) &&
          j.passengers > 0,
      ),
    );
    result.checks.push(
      'Normal NEC controls lease trains, ' +
        'accept service and carry Washington–Boston and Philadelphia–Washington passengers in a settled month',
    );
    const uz = f.routes.find((r) => r.id === 'network_uzbek_main_all_stops');
    await (await control(p, '#service-corridor')).selectOption(uz.id);
    await (await control(p, '#focus-route')).click();
    await p.waitForTimeout(150);
    result.uzbek_highlight_samples = await pixels(p, uz);
    assert(result.uzbek_highlight_samples > 8);
    await p.screenshot({ path: 'docs/validation/network-completion-uzbekistan.png' });
    assert.deepEqual(errors, []);
    result.checks.push('Complete NEC and Tashkent–Khiva paths visibly highlight on the rendered map');
    const checkpoint = f.checkpoint,
      network = f.network;
    await b.close();
    clearTimeout(timer);
    assert(!limited, 'RSS limit');
    // The browser is closed before the native comparison, keeping the two large
    // runtimes out of memory together. No player campaign is read or written.
    fs.writeFileSync('build/network-completion/browser-commands.json', JSON.stringify(trace));
    const native = JSON.parse(
      execFileSync(
        path.resolve('build/headless'),
        ['-commands', path.resolve('build/network-completion/browser-commands.json')],
        {
          encoding: 'utf8',
          maxBuffer: 32 * 1024 * 1024,
          stdio: ['ignore', 'pipe', 'pipe'],
          env: { ...process.env, GOMEMLIMIT: '768MiB', GOMAXPROCS: '2' },
        },
      ),
    );
    assert.equal(native.checkpoint, checkpoint);
    assert.deepEqual(native.network, network);
    result.checkpoint = checkpoint;
    result.checks.push('Native-headless and WASM checkpoints and detailed passenger reports agree');
    result.passed = true;
  } catch (e) {
    result.error = String(e);
    throw e;
  } finally {
    clearInterval(monitor);
    clearTimeout(timer);
    await b.close();
    result.peak_rss_mib = Math.ceil(peak / 1024);
    result.status = limited ? 'memory_limit_stopped' : result.passed ? 'passed' : 'failed';
    fs.writeFileSync('docs/validation/network-completion-browser-checks.json', JSON.stringify(result, null, 2) + '\n');
    console.log(JSON.stringify(result, null, 2));
  }
})().catch((e) => {
  console.error(e);
  process.exitCode = 1;
});
