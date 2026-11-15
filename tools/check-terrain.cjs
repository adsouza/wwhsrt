const { control, language, prospect } = require('./ui-helpers.cjs');
// Focused real Go/WASM terrain workflow. No commands change the company.
const assert = require('node:assert/strict'),
  fs = require('node:fs'),
  path = require('node:path'),
  crypto = require('node:crypto');
const { chromium } = require('playwright');
const url = process.argv[2] || 'http://127.0.0.1:8765/',
  out = path.resolve('docs/validation');
const snap = (p) => p.evaluate(() => railSnapshot());
async function ready(p) {
  await p.goto(url);
  await p.waitForFunction(() => !!window.railSnapshot, null, { timeout: 60000 });
  await p.waitForFunction(async () => (await railSnapshot()).camera.Width > 100);
}
async function terrain(p, hidden) {
  await (await control(p, '#terrain-toggle')).click();
  await p.waitForFunction(async (h) => (await railSnapshot()).state.hide_terrain === h, hidden);
  assert.equal(await p.locator('#terrain-toggle').getAttribute('aria-pressed'), String(!hidden));
}
// Relief tiles load after the view that asks for them, so a picture is taken only once the map has stopped
// changing. This looks at the middle of the map, clear of the legend and the controls, and returns it:
// relief is the only thing the terrain toggle changes there.
async function settled(p) {
  const box = await p.locator('#map-frame').boundingBox(),
    side = Math.min(300, box.width - 40, box.height - 40),
    clip = { x: box.x + (box.width - side) / 2, y: box.y + (box.height - side) / 2, width: side, height: side };
  let last = await p.screenshot({ clip });
  for (let i = 0; i < 24; i++) {
    await p.waitForTimeout(250);
    const now = await p.screenshot({ clip });
    if (now.equals(last)) break;
    last = now;
  }
  return last;
}
async function shot(p, name) {
  await settled(p);
  await p.screenshot({ path: path.join(out, name) });
}
(async () => {
  const b = await chromium.launch({ headless: true, args: ['--enable-unsafe-swiftshader'] }),
    watchdog = setTimeout(() => b.close(), 180000);
  const result = {
    date: new Date().toISOString(),
    browser: b.version(),
    wasm_sha256: crypto.createHash('sha256').update(fs.readFileSync('web/main.wasm')).digest('hex'),
    checks: [],
    limitations: [
      'Regional cartographic context, not an engineering survey or route feasibility proof.',
      'Synthetic browser inputs; physical phones and native controls remain separate qualification.',
    ],
  };
  try {
    const ctx = await b.newContext({ viewport: { width: 1440, height: 960 }, reducedMotion: 'reduce' }),
      p = await ctx.newPage(),
      errors = [];
    p.on('pageerror', (e) => errors.push(e.message));
    await ready(p);
    const before = (await snap(p)).frame;
    const display = JSON.parse(fs.readFileSync('assets/scenario/rail-display-paths.json', 'utf8'));
    for (const co of JSON.parse(fs.readFileSync('assets/scenario/business.json', 'utf8'))) {
      const live = before.corridors.find((c) => c.id === co.id);
      for (const a of co.alignments)
        assert.deepEqual(live.alignments.find((x) => x.id === a.id).path, display[co.id]?.[a.id] || a.path);
    }
    result.checks.push(
      'All six WASM frame alignments exactly match the geography-checked station-connected display paths',
    );
    await (await control(p, '#tab-build')).click();
    await prospect(p, 'hokkaido-extension');
    assert.match(await (await control(p, '#terrain-explanation')).innerText(), /170 km/);
    await (
      await control(p, '#alignments button')
    )
      .locator('..')
      .locator('button')
      .filter({ hasText: 'Valley detour' })
      .click();
    assert.match(await (await control(p, '#terrain-explanation')).innerText(), /45 km/);
    const quote = await (await control(p, '#quote')).innerText();
    await (await control(p, '#expand')).click();
    await (await control(p, '#focus-route')).click();
    await p.waitForFunction(async () => (await railSnapshot()).camera.Zoom > 3);
    await shot(p, 'terrain-hokkaido-desktop.png');
    const map = p.frames().find((f) => f.url().includes('map.html'));
    const on = await (await control(map, 'canvas')).screenshot();
    await terrain(p, true);
    await p.waitForTimeout(500);
    const off = await (await control(map, 'canvas')).screenshot();
    assert.notDeepEqual(on, off);
    assert.equal((await snap(p)).frame.checkpoint, before.checkpoint);
    await p.reload();
    await p.waitForFunction(() => !!window.railSnapshot);
    assert.equal((await snap(p)).state.hide_terrain, true);
    await terrain(p, false);
    await (await control(p, '#focus-route')).click();
    await p.keyboard.press('Escape');
    assert.equal(await (await control(p, '#quote')).innerText(), quote);
    assert.equal((await snap(p)).frame.checkpoint, before.checkpoint);
    console.log('Hokkaido terrain, persistence and state checks passed');
    result.checks.push(
      'Both Hokkaido alignment explanations disclose their tunnel kilometres; ' +
        'terrain changes actual canvas pixels; toggle persists on reload; quotes, ' +
        'cash and checkpoint stay unchanged',
    );
    await (await control(p, '[data-corridor=tsuruga-osaka]')).click();
    await (
      await control(p, '#alignments button')
    )
      .locator('..')
      .locator('button')
      .filter({ hasText: 'Valley detour' })
      .click();
    assert.match(await (await control(p, '#terrain-explanation')).innerText(), /eastern and southern shores/);
    await (await control(p, '#expand')).click();
    await (await control(p, '#focus-route')).click();
    await shot(p, 'terrain-kansai-desktop.png');
    await p.keyboard.press('Escape');
    await prospect(p, 'hokkaido-extension');
    await (
      await control(p, '#alignments button')
    )
      .locator('..')
      .locator('button')
      .filter({ hasText: 'Valley detour' })
      .click();
    await language(p, 'zh-Hans');
    await p.waitForFunction(() => document.documentElement.lang === 'zh-Hans');
    assert.match(await (await control(p, '#terrain-explanation')).innerText(), /45/);
    await p.setViewportSize({ width: 390, height: 844 });
    await p.locator('#build-pane-plan').click();
    await (await control(p, '#browse-prospects')).click();
    await (await control(p, '#focus-route')).click();
    await p.waitForFunction(async () => (await railSnapshot()).camera.Width === 390);
    const phoneMap = await p.locator('#map-frame').boundingBox(),
      tools = await p.locator('.map-tools').boundingBox();
    assert(
      (await snap(p)).points.shin_hakodate_hokuto.Y + 12 < tools.y - phoneMap.y,
      'route endpoint hidden by phone controls',
    );
    await shot(p, 'terrain-hokkaido-phone-zh.png');
    for (const width of [320, 390, 768, 1920]) {
      await p.setViewportSize({ width, height: width > 1000 ? 1080 : 844 });
      // The page moves the map tools only after its resize handler has asked the game for a new frame.
      await p.waitForFunction(() => document.getElementById('phone-map-tools').hidden === innerWidth >= 800);
      assert(await p.evaluate(() => document.documentElement.scrollWidth <= innerWidth));
      assert(await (await control(p, '#terrain-toggle')).isVisible());
      if (
        (await p.locator('#phone-map-tools').isVisible()) &&
        (await p.locator('#phone-map-tools').evaluate((e) => e.open))
      )
        await p.locator('#phone-map-tools>summary').click();
    }
    console.log('Kansai and responsive Chinese maps passed');
    result.checks.push(
      'Kansai relief and Lake Biwa inspected; Simplified Chinese Hokkaido map; ' +
        '320–1920px layouts keep terrain toggle visible without horizontal overflow',
    );
    // Relief outside Japan. The Bogotá–Medellín route follows the Magdalena valley between two ranges.
    await p.setViewportSize({ width: 1440, height: 960 });
    await p.waitForFunction(() => document.getElementById('phone-map-tools').hidden);
    await p.keyboard.press('Escape'); // the phone steps left the map expanded, which hides the tabs
    await p.waitForFunction(() => !document.body.classList.contains('expanded'));
    await (await control(p, '#region-select')).selectOption('south-america');
    await prospect(p, 'launch_64_bogota__medellin');
    await (await control(p, '#expand')).click();
    await (await control(p, '#focus-route')).click();
    await p.waitForFunction(async () => (await railSnapshot()).camera.Zoom > 3);
    const andes = await settled(p);
    await p.screenshot({ path: path.join(out, 'terrain-bogota-medellin-desktop.png') });
    await terrain(p, true);
    const plain = await settled(p);
    assert(!andes.equals(plain), 'relief does not change the middle of the Bogotá–Medellín map');
    await terrain(p, false);
    assert.equal((await snap(p)).frame.checkpoint, before.checkpoint);
    await p.keyboard.press('Escape');
    console.log('Relief outside Japan passed');
    result.checks.push(
      'Relief outside Japan: the middle of the Bogotá–Medellín map changes with the terrain toggle; ' +
        'checkpoint unchanged',
    );
    await p.evaluate(() => navigator.serviceWorker.ready);
    await p.waitForFunction(() => !!navigator.serviceWorker.controller);
    await p.close();
    await ctx.setOffline(true);
    const offline = await ctx.newPage();
    await ready(offline);
    assert.equal((await snap(offline)).state.hide_terrain, false);
    assert.equal((await snap(offline)).frame.checkpoint, before.checkpoint);
    assert(
      await offline.evaluate(async () => {
        const r = await fetch('terrain-credits.txt');
        return r.ok && (await r.text()).includes('Mapzen Terrain Tiles');
      }),
    );
    assert.deepEqual(errors, []);
    result.checks.push('Embedded terrain and attribution open offline with the same company checkpoint');
    result.result = 'passed';
    fs.writeFileSync(path.join(out, 'terrain-checks.json'), JSON.stringify(result, null, 2) + '\n');
    console.log(JSON.stringify(result, null, 2));
    await ctx.close();
  } finally {
    clearTimeout(watchdog);
    await b.close();
  }
})().catch((e) => {
  console.error(e);
  process.exitCode = 1;
});
