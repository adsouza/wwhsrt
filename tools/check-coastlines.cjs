const assert = require('node:assert/strict'),
  fs = require('node:fs'),
  crypto = require('node:crypto');
const { chromium } = require('playwright'),
  { control, prospect } = require('./ui-helpers.cjs');
const snap = (p) => p.evaluate(() => railSnapshot()),
  url = process.argv[2] || 'http://127.0.0.1:8765/';
function screen(c, lat, lon) {
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
async function closeTools(p) {
  if (await p.locator('#dock-map-tools').evaluate((e) => e.open)) await p.locator('#dock-map-tools>summary').click();
}
async function appearance(p, mode, view) {
  await (await control(p, '#appearance')).selectOption(mode);
  await p.waitForFunction(async (d) => (await railSnapshot()).state.dark === d, mode === 'dark');
  await (await control(p, '#tab-' + view)).click();
}
async function terrain(p, hidden) {
  if ((await snap(p)).state.hide_terrain !== hidden) {
    await (await control(p, '#terrain-toggle')).click();
    await p.waitForFunction(async (h) => (await railSnapshot()).state.hide_terrain === h, hidden);
  }
  await closeTools(p);
}
async function pixels(p, qa, coast) {
  const s = await snap(p),
    box = await p.locator('#map-frame').boundingBox(),
    png = await p.screenshot({ clip: box });
  return p.evaluate(
    async (q) => {
      const im = await createImageBitmap(
          new Blob([Uint8Array.from(atob(q.png), (c) => c.charCodeAt(0))], { type: 'image/png' }),
        ),
        c = new OffscreenCanvas(im.width, im.height),
        ctx = c.getContext('2d');
      ctx.drawImage(im, 0, 0);
      const d = ctx.getImageData(0, 0, im.width, im.height).data;
      const inside = ([x, y]) => x > 8 && y > 100 && x < im.width - 8 && y < im.height - 50;
      let good = 0,
        checked = 0,
        samples = [];
      for (let xy of q.waterPoints) {
        if (!inside(xy)) continue;
        checked++;
        const x = Math.round(xy[0]),
          y = Math.round(xy[1]),
          i = (y * im.width + x) * 4;
        samples.push({ pixel: [x, y], rgb: [d[i], d[i + 1], d[i + 2]], expected: q.water });
        if (q.water.every((v, n) => Math.abs(d[i + n] - v) < 5)) good++;
      }
      let shore = 0,
        boundary = 0;
      for (let xy of q.coastPoints) {
        if (!inside(xy)) continue;
        const x = Math.round(xy[0]),
          y = Math.round(xy[1]);
        let hit = false;
        for (let dx = -2; dx <= 2; dx++)
          for (let dy = -2; dy <= 2; dy++) {
            const i = ((y + dy) * im.width + x + dx) * 4;
            if (q.coast.every((v, n) => Math.abs(d[i + n] - v) < 5)) hit = true;
          }
        if (hit) shore++;
        let wet = false,
          dry = false;
        for (let dx = -4; dx <= 4; dx++)
          for (let dy = -4; dy <= 4; dy++) {
            const i = ((y + dy) * im.width + x + dx) * 4;
            if (q.water.every((v, n) => Math.abs(d[i + n] - v) < 5)) wet = true;
            else dry = true;
          }
        if (wet && dry) boundary++;
      }
      return { checkedWater: checked, correctWater: good, shoreHits: shore, boundaryHits: boundary, samples };
    },
    {
      png: png.toString('base64'),
      waterPoints: qa.map(([lat, lon]) => screen(s.camera, lat, lon)),
      coastPoints: coast.map(([lon, lat]) => screen(s.camera, lat, lon)),
      water: [s.palette.water.R, s.palette.water.G, s.palette.water.B],
      coast: [s.palette.coast.R, s.palette.coast.G, s.palette.coast.B],
    },
  );
}
(async () => {
  const b = await chromium.launch({ headless: true, args: ['--enable-unsafe-swiftshader'] }),
    timer = setTimeout(() => b.close(), 180000);
  try {
    const ctx = await b.newContext({ viewport: { width: 1440, height: 960 }, reducedMotion: 'reduce' }),
      p = await ctx.newPage(),
      errors = [];
    p.on('pageerror', (e) => errors.push(e.message));
    await p.goto(url);
    await p.waitForFunction(() => !!window.railSnapshot, null, { timeout: 60000 });
    const before = await snap(p),
      saved = await p.evaluate(() => railStorageLoad()),
      all = JSON.parse(fs.readFileSync('assets/terrain/land.json'));
    const checks = [];
    for (const country of ['Japan', 'China']) {
      if (country === 'Japan') {
        await (await control(p, '#region-select')).selectOption('japan');
        await prospect(p, 'hokkaido-extension');
        await (await control(p, '#focus-route')).click();
      } else {
        await (await control(p, '#region-select')).selectOption('china');
        await (await control(p, '#tab-services')).click();
        await p.locator('#service-corridor').selectOption('public_hangzhou__shanghai');
        await (await control(p, '#focus-route')).click();
      }
      await closeTools(p);
      const qa =
        country === 'Japan'
          ? [
              [42.95577938, 140.09867836],
              [37.31509792, 138.1263546],
              [40.63917142, 141.60869049],
              [41.59805279, 140.54059505],
            ]
          : [
              [30.47348219, 121.15774495],
              [37.29738934, 119.10184264],
              [28.26677417, 121.17869533],
              [20.07817143, 110.48955745],
            ];
      const coast = all
        .flat()
        .filter(([lon, lat]) =>
          country === 'Japan'
            ? lon > 139 && lon < 144 && lat > 41 && lat < 44
            : lon > 120 && lon < 122.5 && lat > 29 && lat < 32,
        )
        .filter((_, i) => i % 3 === 0);
      for (const mode of ['light', 'dark']) {
        await appearance(p, mode, country === 'Japan' ? 'build' : 'services');
        for (const hide of [true, false]) {
          await terrain(p, hide);
          const result = await pixels(p, qa, coast);
          console.log(country, mode, hide, JSON.stringify(result));
          if (result.correctWater !== result.checkedWater) await p.screenshot({ path: 'build/coast-debug.png' });
          assert(result.checkedWater > 0);
          assert.equal(result.correctWater, result.checkedWater);
          assert(result.boundaryHits > 15);
          if (hide) assert(result.shoreHits >= result.boundaryHits * 0.25);
          checks.push({ country, mode, terrain: !hide, ...result });
          if (mode === 'light' && !hide)
            await p.screenshot({ path: 'docs/validation/coastline-' + country.toLowerCase() + '.png' });
        }
      }
    }
    assert.equal((await snap(p)).frame.checkpoint, before.frame.checkpoint);
    assert.equal(await p.evaluate(() => railStorageLoad()), saved);
    assert.deepEqual(errors, []);
    const result = {
      date: new Date().toISOString(),
      passed: true,
      wasm_sha256: crypto.createHash('sha256').update(fs.readFileSync('web/main.wasm')).digest('hex'),
      world_land_sha256: crypto
        .createHash('sha256')
        .update(fs.readFileSync('assets/terrain/world-land.json'))
        .digest('hex'),
      checks,
      verified: [
        'Former coarse-land coastal points are water; detailed shorelines remain visible in Japan and China',
        'Light/dark and terrain on/off all use the same replacement geometry; exact save bytes/checkpoint unchanged',
      ],
      limitations: [
        'Synthetic browser interaction; native OS graphics/input and surveyed shoreline accuracy are not qualified.',
      ],
    };
    fs.writeFileSync('docs/validation/coastline-checks.json', JSON.stringify(result, null, 2) + '\n');
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
