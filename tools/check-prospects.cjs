const assert = require('node:assert/strict'),
  fs = require('node:fs'),
  crypto = require('node:crypto');
const { chromium } = require('playwright'),
  { control, language, prospect } = require('./ui-helpers.cjs');
async function snap(p) {
  return p.evaluate(async () => {
    const { camera, state, palette, frame } = await railSnapshot();
    const { revision, checkpoint, projects } = frame;
    const corridors = frame.corridors.filter((c) => !c.existing);
    return { camera, state, palette, frame: { revision, checkpoint, projects, corridors } };
  });
}
const url = process.argv[2] || 'http://127.0.0.1:8765/';
// Construction prospects in the catalog; pkg/app and pkg/render tests pin the same count.
const PROSPECTS = 101;
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
async function closeTools(p) {
  if (await p.locator('#dock-map-tools').evaluate((e) => e.open)) await p.locator('#dock-map-tools>summary').click();
}
function samples(s, path) {
  return path
    .slice(1)
    .map((q, i) => {
      const a = screen(s.camera, path[i]),
        b = screen(s.camera, q);
      return [(a[0] + b[0]) / 2, (a[1] + b[1]) / 2];
    })
    .filter(([x, y]) => x > 60 && x < s.camera.Width - 40 && y > 130 && y < s.camera.Height - 70);
}
async function clickMap(p, at) {
  const r = await p.locator('#map-frame').boundingBox();
  await p.mouse.click(r.x + at[0], r.y + at[1]);
}
async function dots(p, points, ink) {
  const box = await p.locator('#map-frame').boundingBox(),
    png = await p.screenshot({ clip: box });
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
      for (const [x, y] of q.points) {
        let found = false;
        for (let dx = -9; dx <= 9; dx++)
          for (let dy = -9; dy <= 9; dy++) {
            const i = ((Math.round(y) + dy) * im.width + Math.round(x) + dx) * 4;
            if (q.ink.every((v, n) => Math.abs(v - d[i + n]) < 5)) found = true;
          }
        if (found) hits++;
      }
      return hits;
    },
    { png: png.toString('base64'), points, ink: [ink.R, ink.G, ink.B] },
  );
}
(async () => {
  const b = await chromium.launch({ headless: true, args: ['--enable-unsafe-swiftshader'] }),
    timer = setTimeout(() => b.close(), 180000),
    checks = [];
  try {
    const ctx = await b.newContext({ viewport: { width: 1440, height: 960 }, reducedMotion: 'reduce' }),
      p = await ctx.newPage(),
      errors = [];
    p.on('pageerror', (e) => {
      errors.push(e.message);
      console.error('pageerror', e.message);
    });
    await p.goto(url);
    await p.waitForFunction(() => !!window.railSnapshot, null, { timeout: 60000 });
    const initial = await snap(p),
      saved = await p.evaluate(() => railStorageLoad());
    const prospects = initial.frame.corridors.filter(
      (c) => !c.existing && !c.part_of && !c.retired && !initial.frame.projects.some((v) => v.corridor_id === c.id),
    );
    assert.equal(prospects.length, PROSPECTS);
    const cashNeeded = (c) => Math.min(...c.alignments.map((a) => a.cost - a.grant));
    const expected = [...prospects].sort((a, b) => cashNeeded(a) - cashNeeded(b)).map((c) => c.id);
    assert.deepEqual(await p.locator('#opportunities button').evaluateAll((rows) => rows.map((row) => row.dataset.corridor)), expected);
    assert(await p.locator('#opportunities').evaluate((list) => list.scrollHeight > list.clientHeight));
    await (await control(p, '#tab-build')).click();
    assert.equal(await p.locator('#opportunities button').count(), PROSPECTS);
    await (await control(p, '#region-select')).selectOption('east-asia');
    await closeTools(p);
    await p.screenshot({ path: 'docs/validation/prospects-overview.png' });
    // Each prospect can be found, clicked and reviewed without selecting cities.
    for (const co of prospects.filter((c) => ['tsuruga-osaka', 'hokkaido-extension'].includes(c.id))) {
      console.log('Checking prospect', co.id);
      await (await control(p, '#tab-build')).click();
      await prospect(p, co.id);
      await (await control(p, '#focus-route')).click();
      await closeTools(p);
      await p.locator('#tab-map').click();
      const s = await snap(p),
        points = samples(s, co.alignments[0].path);
      assert(points.length > 2, 'visible route body');
      const hits = await dots(p, points, s.palette.action);
      assert(hits > 2, 'actual dotted prospect pixels');
      const before = await snap(p);
      const box = await p.locator('#map-frame').boundingBox(),
        at = points[Math.floor(points.length / 2)];
      await p.mouse.move(box.x + at[0], box.y + at[1]);
      await p.mouse.down();
      await p.mouse.move(box.x + at[0] + 30, box.y + at[1] + 20, { steps: 6 });
      await p.mouse.up();
      assert.equal((await snap(p)).state.view, 'map');
      // Restore the camera after deliberately panning, then pick the visible line.
      await p.locator('#tab-build').click();
      await (await control(p, '#focus-route')).click();
      await closeTools(p);
      await p.locator('#tab-map').click();
      const chosen = samples(await snap(p), co.alignments[0].path);
      await clickMap(p, chosen[Math.floor(chosen.length / 2)]);
      await p.waitForFunction(async (id) => {
        const s = await railSnapshot();
        return s.state.view === 'build' && s.state.from_city === s.frame.corridors.find((c) => c.id === id).from_city;
      }, co.id);
      const selected = await snap(p);
      assert.equal(selected.state.to_city, co.to_city);
      const cheapest = co.alignments.reduce((a, b) => (b.cost - b.grant < a.cost - a.grant ? b : a));
      assert.equal(selected.state.alignment_id, cheapest.id);
      assert(await p.locator('#build-package').isVisible());
      assert.equal(await p.locator('#from-city,#to-city,#choose-from,#choose-to').count(), 0);
      assert.equal(selected.frame.checkpoint, before.frame.checkpoint);
      checks.push({ corridor: co.id, dot_samples: hits });
    }
    await p.screenshot({ path: 'docs/validation/prospects-desktop.png' });
    await p.locator('#browse-prospects').click();
    assert(await p.locator('#inspector').isHidden());
    await p.keyboard.press('Escape');
    assert(await p.locator('#inspector').isVisible());
    await language(p, 'zh-Hans');
    await p.setViewportSize({ width: 390, height: 844 });
    await p.locator('#tab-build').click();
    assert.equal(await p.locator('#opportunities button').count(), PROSPECTS);
    await p.locator('[data-corridor=tsuruga-osaka]').click();
    assert(await p.evaluate(() => document.documentElement.scrollWidth <= innerWidth));
    await p.screenshot({ path: 'docs/validation/prospects-phone-zh.png' });
    await p.locator('#browse-prospects').click();
    assert(await p.locator('#inspector').isHidden());
    assert(await p.locator('#expand').isVisible());
    await p.keyboard.press('Escape');
    assert.equal((await snap(p)).frame.checkpoint, initial.frame.checkpoint);
    assert.equal(await p.evaluate(() => railStorageLoad()), saved);
    // Start one real project: it leaves both the list and map's selectable set.
    await language(p, 'en');
    await p.setViewportSize({ width: 1440, height: 960 });
    await p.locator('[data-corridor=tsuruga-osaka]').click();
    if (await p.locator('#borrow-shortfall-build').isVisible()) {
      const rev = (await snap(p)).frame.revision;
      await p.locator('#borrow-shortfall-build').click();
      await p.waitForFunction(async (r) => (await railSnapshot()).frame.revision > r, rev);
    }
    const rev = (await snap(p)).frame.revision;
    await p.locator('#build-line').click();
    await p.waitForFunction(async (r) => (await railSnapshot()).frame.revision > r, rev);
    assert.equal(await p.locator('[data-corridor=tsuruga-osaka]').count(), 0);
    assert.equal(await p.locator('#opportunities button').count(), PROSPECTS - 1);
    await (await control(p, '#focus-route')).click();
    await closeTools(p);
    await p.locator('#tab-map').click();
    const s = await snap(p),
      co = s.frame.corridors.find((c) => c.id === 'tsuruga-osaka'),
      at = samples(s, co.alignments[0].path);
    await clickMap(p, at[Math.floor(at.length / 2)]);
    // A built line is no longer a prospect, so the click must not open it in Build. Open public track under the
    // click may legitimately pick a service instead, depending on where the platform's fonts put the map.
    await p.waitForTimeout(500);
    assert.notEqual((await snap(p)).state.view, 'build');
    await p.reload();
    await p.waitForFunction(() => !!window.railSnapshot, null, { timeout: 60000 });
    await p.locator('#tab-build').click();
    assert.equal(await p.locator('[data-corridor=tsuruga-osaka]').count(), 0);
    assert.deepEqual(errors, []);
    const result = {
      date: new Date().toISOString(),
      passed: true,
      wasm_sha256: crypto.createHash('sha256').update(fs.readFileSync('web/main.wasm')).digest('hex'),
      checks,
      verified: [
        'All ' + PROSPECTS + ' prospects are available in the cash-ranked scrollable list; ' +
          'the two Japanese sample prospects have actual dotted map pixels and can be selected by their curved paths; ' +
          'drag does not select',
        'Selection opens the complete project and preserves the campaign; no arbitrary endpoint controls',
        'Full-map browse/restore and Simplified Chinese phone list fit the viewport',
        'A real construction commitment removes its prospect from list/map selection, including after reload',
      ],
      limitations: [
        'Synthetic browser events and native widget tests; native OS input and physical phones not qualified.',
      ],
    };
    fs.writeFileSync('docs/validation/prospects-checks.json', JSON.stringify(result, null, 2) + '\n');
    console.log(JSON.stringify(result, null, 2));
    await ctx.close();
  } finally {
    clearTimeout(timer);
    await b.close();
  }
})().catch((e) => {
  console.error(e);
  process.exitCode = 1;
});
