const assert = require('node:assert/strict'),
  fs = require('node:fs'),
  crypto = require('node:crypto');
const { chromium } = require('playwright'),
  { control } = require('./ui-helpers.cjs');
const snap = (p) => p.evaluate(() => railSnapshot()),
  url = process.argv[2] || 'http://127.0.0.1:8765/';
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
  const b = await chromium.launch({ headless: true, args: ['--enable-unsafe-swiftshader'] }),
    timer = setTimeout(() => b.close(), 120000);
  try {
    const ctx = await b.newContext({ viewport: { width: 1440, height: 960 }, reducedMotion: 'reduce' }),
      p = await ctx.newPage(),
      errors = [];
    p.on('pageerror', (e) => errors.push(e.message));
    await p.goto(url);
    await p.waitForFunction(() => !!window.railSnapshot, null, { timeout: 60000 });
    const initial = await snap(p),
      saved = await p.evaluate(() => railStorageLoad());
    await (await control(p, '#region-select')).selectOption('china');
    await (await control(p, '#tab-services')).click();
    await p.locator('#service-corridor').selectOption('public_harbin__shenyang');
    await (await control(p, '#focus-route')).click();
    for (let i = 0; i < 3; i++) await (await control(p, '#zoom-in')).click();
    await (await control(p, '#tab-map')).click();
    const s = await snap(p),
      route = s.frame.corridors.find((c) => c.id === 'public_harbin__shenyang');
    assert(route.existing);
    const path = route.alignments[0].path,
      good = path.slice(1).map((q, i) => point(s.camera, (q.lat + path[i].lat) / 2, (q.lon + path[i].lon) / 2)),
      bad = [
        [43.0289, 124.271],
        [43.257, 124.4899],
        [43.746, 125.1136],
      ].map(([lat, lon]) => point(s.camera, lat, lon));
    const box = await p.locator('#map-frame').boundingBox(),
      png = await p.screenshot({ clip: box }),
      ink = s.palette.existing;
    const pixels = await p.evaluate(
      async (q) => {
        const im = await createImageBitmap(
            new Blob([Uint8Array.from(atob(q.png), (c) => c.charCodeAt(0))], { type: 'image/png' }),
          ),
          c = new OffscreenCanvas(im.width, im.height),
          ctx = c.getContext('2d');
        ctx.drawImage(im, 0, 0);
        const d = ctx.getImageData(0, 0, im.width, im.height).data;
        const hit = ([x, y]) => {
          x = Math.round(x);
          y = Math.round(y);
          if (x < 3 || y < 3 || x >= im.width - 3 || y >= im.height - 3) return null;
          for (let dx = -2; dx <= 2; dx++)
            for (let dy = -2; dy <= 2; dy++) {
              const i = ((y + dy) * im.width + x + dx) * 4;
              if (q.ink.every((v, n) => Math.abs(d[i + n] - v) < 5)) return true;
            }
          return false;
        };
        return { keptRoute: q.good.map(hit).filter((v) => v !== null), unusedFragment: q.bad.map(hit) };
      },
      { png: png.toString('base64'), ink: [ink.R, ink.G, ink.B], good, bad },
    );
    console.log(pixels);
    assert(pixels.keptRoute.filter(Boolean).length >= 5);
    assert(pixels.unusedFragment.some((v) => v !== null));
    assert(pixels.unusedFragment.every((v) => v === false || v === null));
    await p.screenshot({ path: 'docs/validation/playable-map-northeast-china.png' });
    assert.equal((await snap(p)).frame.checkpoint, initial.frame.checkpoint);
    assert.equal(await p.evaluate(() => railStorageLoad()), saved);
    await p.reload();
    await p.waitForFunction(() => !!window.railSnapshot, null, { timeout: 60000 });
    assert.equal((await snap(p)).frame.checkpoint, initial.frame.checkpoint);
    assert.deepEqual(errors, []);
    const result = {
      date: new Date().toISOString(),
      passed: true,
      wasm_sha256: crypto.createHash('sha256').update(fs.readFileSync('web/main.wasm')).digest('hex'),
      map_sha256: crypto
        .createHash('sha256')
        .update(fs.readFileSync('assets/scenario/playable-rail-map.json'))
        .digest('hex'),
      pixels,
      checks: [
        'Actual Go map retains Harbin–Shenyang through track while unused parallel fragment samples have no rail pixels',
        'Map focus/zoom/selection/reload preserve campaign checkpoint and exact durable save bytes',
      ],
      limitations: [
        'Browser input is synthetic; native OS interaction and production topology qualification remain separate.',
      ],
    };
    fs.writeFileSync('docs/validation/playable-map-checks.json', JSON.stringify(result, null, 2) + '\n');
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
