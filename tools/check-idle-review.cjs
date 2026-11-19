const { control, language, prospect } = require('./ui-helpers.cjs');
// Actual controls + Go-rendered pixels: identify every idle built line, not an
// arbitrary selected service or an unbuilt alignment. Uses isolated browser data.
const assert = require('node:assert/strict'),
  fs = require('node:fs'),
  path = require('node:path'),
  crypto = require('node:crypto');
const { chromium } = require('playwright');
const url = process.argv[2] || 'http://127.0.0.1:8765/',
  out = path.resolve('docs/validation');
// Keep only the inspected lines when crossing the browser/Node boundary.
const snap = (p, ids = []) => p.evaluate(async (ids) => {
  const { camera, palette, frame } = await railSnapshot();
  return {
    camera, palette,
    frame: {
      revision: frame.revision, checkpoint: frame.checkpoint,
      idle_lines: frame.idle_lines, services: frame.services,
      projects: frame.projects.filter((project) => ids.includes(project.corridor_id)),
      corridors: frame.corridors.filter((corridor) => ids.includes(corridor.id)),
    },
  };
}, ids);
async function command(p, id) {
  const rev = (await snap(p)).frame.revision;
  await (await control(p, '#' + id)).click();
  await p.waitForFunction(async (r) => (await railSnapshot()).frame.revision === r + 1, rev);
  await p.waitForFunction(() => document.getElementById('game').getAttribute('aria-busy') === 'false');
}
function project(lat, lon) {
  const t = Math.asin((Math.sqrt(3) / 2) * Math.sin((lat * Math.PI) / 180)),
    t2 = t * t,
    t6 = t2 * t2 * t2;
  return {
    X:
      (2 * Math.sqrt(3) * ((lon * Math.PI) / 180) * Math.cos(t)) /
      (3 * (1.340264 + 3 * -0.081106 * t2 + t6 * (7 * 0.000893 + 9 * 0.003796 * t2))),
    Y: t * (1.340264 + -0.081106 * t2 + t6 * (0.000893 + 0.003796 * t2)),
  };
}
function screen(c, p) {
  const s =
    Math.min(Math.max(1, c.Width - 72) / (c.Max.X - c.Min.X), Math.max(1, c.Height - 80) / (c.Max.Y - c.Min.Y)) *
    c.Zoom;
  return [
    c.Width / 2 + (p.X - (c.Min.X + c.Max.X) / 2) * s + c.PanX,
    c.Height / 2 - (p.Y - (c.Min.Y + c.Max.Y) / 2) * s + c.PanY,
  ];
}
function samples(s, id) {
  const co = s.frame.corridors.find((c) => c.id === id),
    pr = s.frame.projects.find((p) => p.corridor_id === id),
    a = co.alignments.find((a) => a.id === pr?.alignment_id) || co.alignments[0],
    points = [];
  for (let i = 1; i < a.path.length; i++) {
    for (const u of [0.1, 0.2, 0.3, 0.4, 0.5, 0.6, 0.7, 0.8, 0.9]) {
      const p = project(a.path[i - 1].lat, a.path[i - 1].lon),
        q = project(a.path[i].lat, a.path[i].lon);
      points.push(screen(s.camera, { X: p.X + (q.X - p.X) * u, Y: p.Y + (q.Y - p.Y) * u }));
    }
  }
  return points;
}
async function pixels(p, ids) {
  const s = await snap(p, ids),
    map = p.frames().find((f) => f.url().includes('map.html')),
    png = await (await control(map, 'canvas')).screenshot(),
    color = s.palette['warning-text'];
  return p.evaluate(
    async (q) => {
      const im = await createImageBitmap(
          new Blob([Uint8Array.from(atob(q.png), (c) => c.charCodeAt(0))], { type: 'image/png' }),
        ),
        c = new OffscreenCanvas(im.width, im.height),
        ctx = c.getContext('2d');
      ctx.drawImage(im, 0, 0);
      const d = ctx.getImageData(0, 0, im.width, im.height).data,
        out = {};
      for (const [id, points] of Object.entries(q.samples)) {
        let hits = 0,
          inside = 0;
        for (let [x, y] of points) {
          x = Math.round(x);
          y = Math.round(y);
          if (x < 3 || y < 3 || x >= im.width - 3 || y >= im.height - 3) continue;
          inside++;
          let hit = false;
          for (let dx = -2; dx <= 2; dx++)
            for (let dy = -2; dy <= 2; dy++) {
              const i = ((y + dy) * im.width + x + dx) * 4;
              if (q.color.every((v, n) => Math.abs(d[i + n] - v) <= 4)) hit = true;
            }
          if (hit) hits++;
        }
        out[id] = { hits, inside };
      }
      return out;
    },
    {
      png: png.toString('base64'),
      color: [color.R, color.G, color.B],
      samples: Object.fromEntries(ids.map((id) => [id, samples(s, id)])),
    },
  );
}
async function reviewAll(p) {
  await (await control(p, '#review-idle')).click();
  await p.waitForFunction(() => document.getElementById('map-panel').dataset.idleFocus === 'all');
}

(async () => {
  const browser = await chromium.launch({ headless: true, args: ['--enable-unsafe-swiftshader'] });
  const timer = setTimeout(() => browser.close(), 240000);
  const result = {
    date: new Date().toISOString(),
    browser: browser.version(),
    wasm_sha256: crypto.createHash('sha256').update(fs.readFileSync('web/main.wasm')).digest('hex'),
    ui_sha256: crypto
      .createHash('sha256')
      .update(Buffer.concat(['web/app.js', 'web/style.css', 'web/index.html'].map((f) => fs.readFileSync(f))))
      .digest('hex'),
    checks: [],
    limitations: ['Phone layouts are emulated; native OS input and physical devices are not qualified.'],
  };
  try {
    const ctx = await browser.newContext({ viewport: { width: 1280, height: 850 }, reducedMotion: 'reduce' }),
      p = await ctx.newPage(),
      errors = [];
    p.on('pageerror', (e) => errors.push(e.message));
    await p.goto(url);
    await p.waitForFunction(() => !!window.railSnapshot, null, { timeout: 60000 });
    await (await control(p, '#tab-company')).click();
    await (await control(p, '#loan-amount')).fill('5,000,000,000');
    await command(p, 'borrow');
    for (const id of ['tsuruga-osaka', 'hokkaido-extension']) {
      await (await control(p, '#tab-build')).click();
      await prospect(p, id);
      await p.locator('#alignments button').filter({ hasText: 'Tunnel route' }).click();
      await command(p, 'build-line');
    }
    for (let i = 0; i < 4; i++) await command(p, 'advance-year');
    let s = await snap(p);
    assert.equal(s.frame.idle_lines.length, 2);
    const before = s.frame.checkpoint,
      bytes = await p.evaluate(() => railStorageLoad());
    await reviewAll(p);
    s = await snap(p);
    assert.equal(await p.locator('#idle-line-list button').count(), 2);
    for (const line of s.frame.idle_lines)
      assert(await p.locator('[data-idle-corridor="' + line.corridor_id + '"]').isVisible());
    assert.equal(await p.locator('#idle-line-list [aria-pressed=true]').count(), 1);
    let paint = await pixels(p, ['tsuruga-osaka', 'hokkaido-extension', 'tokyo-nagoya']);
    assert(paint['tsuruga-osaka'].hits >= 3);
    assert(paint['hokkaido-extension'].hits >= 3);
    assert.equal(paint['tokyo-nagoya'].hits, 0);
    assert.equal((await snap(p)).frame.checkpoint, before);
    assert.equal(await p.evaluate(() => railStorageLoad()), bytes);
    await p.screenshot({ path: path.join(out, 'idle-review-desktop.png') });
    result.initial_pixels = paint;
    result.checks.push(
      'Review frames both idle built alignments, ' +
        'paints their actual map pixels in the warning colour and lists both names; ' +
        'the operating Tokyo–Nagoya line is not marked idle; ' +
        'review leaves checkpoint and durable bytes unchanged',
    );
    await (await control(p, '[data-idle-corridor=hokkaido-extension]')).click();
    await p.waitForFunction(async () => {
      const s = await railSnapshot();
      return (
        s.state.service_corridor === 'hokkaido-extension' &&
        s.camera.Zoom > 3 &&
        document.getElementById('map-panel').dataset.idleFocus === 'hokkaido-extension'
      );
    });
    assert.equal(await p.locator('[data-idle-corridor=hokkaido-extension]').getAttribute('aria-pressed'), 'true');
    await p.setViewportSize({ width: 320, height: 740 });
    await language(p, 'zh-Hans');
    await (await control(p, '#tab-report')).click();
    await reviewAll(p);
    await p.waitForFunction(async () => (await railSnapshot()).camera.Width === 320);
    assert(await p.locator('#map-panel').isVisible());
    assert(await p.evaluate(() => document.documentElement.scrollWidth <= innerWidth));
    paint = await pixels(p, ['tsuruga-osaka', 'hokkaido-extension']);
    console.log('Phone idle paint', paint);
    assert(paint['tsuruga-osaka'].hits > 0 && paint['hokkaido-extension'].hits > 0);
    await p.screenshot({ path: path.join(out, 'idle-review-phone-zh.png'), fullPage: true });
    result.phone_pixels = paint;
    result.checks.push(
      'Each named idle row selects and focuses its line; ' +
        'Chinese 320px review reveals the map and highlights both lines without horizontal overflow',
    );
    await p.setViewportSize({ width: 1280, height: 850 });
    await language(p, 'en');
    await (await control(p, '[data-idle-corridor=tsuruga-osaka]')).click();
    await command(p, 'lease-train');
    await command(p, 'lease-train');
    assert.equal((await snap(p)).frame.idle_lines.length, 2);
    await (await control(p, '#frequency')).fill('4');
    await p.waitForFunction(() => !document.getElementById('apply-service').disabled);
    await command(p, 'apply-service');
    s = await snap(p);
    assert.deepEqual(
      s.frame.idle_lines.map((l) => l.corridor_id),
      ['hokkaido-extension'],
    );
    assert.equal(await p.locator('[data-idle-corridor=tsuruga-osaka]').count(), 0);
    await (await control(p, '#review-idle')).click();
    await p.waitForFunction(() => document.getElementById('map-panel').dataset.idleFocus === 'all');
    await (await control(p, '#fit')).click();
    await p.waitForFunction(async () => (await railSnapshot()).camera.Zoom === 1);
    paint = await pixels(p, ['tsuruga-osaka', 'hokkaido-extension']);
    assert(paint['tsuruga-osaka'].inside >= 3);
    assert.equal(paint['tsuruga-osaka'].hits, 0);
    assert(paint['hokkaido-extension'].hits > 0);
    result.resolved_pixels = paint;
    result.checks.push(
      'Leasing alone retains both idle highlights; ' +
        'applying a running service removes exactly its row and highlight while the other idle line remains marked',
    );

    // An idle leased fleet on shared track must not claim that the physical
    // section is idle when a through train is actively using it.
    await (await control(p, '#service-corridor')).selectOption('tokyo-osaka');
    await p.waitForFunction(() => document.getElementById('extend-starter').checked);
    await p.waitForFunction(() => !document.getElementById('apply-service').disabled);
    await command(p, 'apply-service');
    await (await control(p, '#service-corridor')).selectOption('tokyo-nagoya');
    await command(p, 'lease-train');
    s = await snap(p);
    const fleet = s.frame.idle_lines.find((l) => l.corridor_id === 'tokyo-nagoya');
    assert.equal(fleet.maintenance, 0);
    assert(fleet.leases > 0);
    assert(s.frame.services.some((r) => r.corridor_id === 'tokyo-osaka' && r.frequency > 0));
    await reviewAll(p);
    assert((await p.locator('[data-idle-corridor=tokyo-nagoya]').innerText()).startsWith('Idle trains:'));
    paint = await pixels(p, ['tokyo-nagoya', 'hokkaido-extension']);
    assert(paint['tokyo-nagoya'].hits > 0 && paint['hokkaido-extension'].hits > 0);
    result.shared_track_pixels = paint;
    await (await control(p, '#service-corridor')).selectOption('tokyo-osaka');
    await (await control(p, '#frequency')).fill('0');
    await p.waitForFunction(() => !document.getElementById('apply-service').disabled);
    await command(p, 'apply-service');
    await reviewAll(p);
    assert((await p.locator('[data-idle-corridor=tokyo-osaka]').innerText()).startsWith('Idle trains:'));
    paint = await pixels(p, ['tokyo-nagoya', 'nagoya-osaka']);
    assert(paint['tokyo-nagoya'].hits > 0 && paint['nagoya-osaka'].hits > 0);
    result.through_fleet_pixels = paint;
    result.checks.push(
      'Idle leased trains are named separately and use a dashed route highlight even while another service runs on the same public track; ' +
        'an idle through-service fleet highlights both sections',
    );
    const checkpoint = (await snap(p)).frame.checkpoint;
    await p.reload();
    await p.waitForFunction(() => !!window.railSnapshot, null, { timeout: 60000 });
    assert.equal((await snap(p)).frame.checkpoint, checkpoint);
    assert.equal(await p.locator('#idle-line-list button').count(), 3);
    assert.deepEqual(errors, []);
    result.passed = true;
    fs.writeFileSync(path.join(out, 'idle-review-checks.json'), JSON.stringify(result, null, 2) + '\n');
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
