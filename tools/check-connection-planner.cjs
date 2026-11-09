/* Focused interaction-prototype checks. No game, Go, or TypeScript build. */
const { chromium } = require('playwright');
const fs = require('node:fs');
const path = require('node:path');
const assert = require('node:assert/strict');
const url = process.argv[2];
if (!url) throw new Error('Usage: node tools/check-connection-planner.cjs <served-preview-url> [evidence-dir]');
const evidence = process.argv[3] || path.resolve(__dirname, '../docs/validation');
const results = [], errors = [];
const record = (name, detail = true) => results.push({ name, detail });
const rootSelector = '#rail-connection-planner';
const stageSelector = '[data-stage]';
const sleep = ms => new Promise(resolve => setTimeout(resolve, ms));
const camera = f => f.locator(stageSelector).evaluate(el => JSON.parse(el.dataset.camera));
const ready = f => f.waitForSelector(rootSelector + '[data-quote-state=ready]');
const endpoint = (f, slot) => f.locator('[data-name=' + slot + ']').innerText();
async function choose(f, slot, id) {
  await f.locator('[data-slot=' + slot + ']').click();
  await f.locator('[data-city]').selectOption(id);
}
async function saved(f) {
  return f.evaluate(() => window.__plannerSaved?.privateContent?.connectionPlanner);
}
async function attach(page) {
  await page.goto(url);
  const frame = page.frames().find(f => f.parentFrame());
  assert(frame, 'render.py preview should contain an iframe');
  await ready(frame);
  await frame.evaluate(() => {
    const original = window.openai?.setWidgetState;
    window.openai = { ...window.openai, setWidgetState: async state => {
      window.__plannerSaved = state;
      if (original) return original(state);
    }};
  });
  return frame;
}
async function noOverflow(f, label) {
  const bad = await f.locator(rootSelector).evaluate(root => {
    const r = root.getBoundingClientRect();
    return [...root.querySelectorAll('button,select,.cp-strip strong')].filter(el => {
      if (!el.getClientRects().length) return false;
      const b = el.getBoundingClientRect();
      return b.left < r.left - 1 || b.right > r.right + 1 || el.scrollWidth > el.clientWidth + 2;
    }).map(el => el.outerHTML.slice(0, 160));
  });
  assert.deepEqual(bad, [], label);
}
(async () => {
  fs.mkdirSync(evidence, { recursive: true });
  const metros = JSON.parse(fs.readFileSync(path.resolve(__dirname, '../data/prototypes/playable-metros.v1.json')));
  const fixture = JSON.parse(fs.readFileSync(path.resolve(__dirname, '../data/prototypes/connection-planner-demo.v1.json')));
  const capacity = 2 * fixture.return_trips_per_day * fixture.seats_per_train * fixture.operating_days;
  for (const p of fixture.pairs) for (const a of p.alternatives) {
    assert.equal(a.passengers * p.average_fare_usd / 1e6, a.revenue_usd_million);
    assert(a.passengers <= capacity);
    for (const [i, endpointIndex] of [[0, 0], [a.coordinates.length - 1, 1]]) {
      const m = metros.find(m => m.id === p.metro_ids[endpointIndex]);
      assert.deepEqual(a.coordinates[i], [m.lng, m.lat]);
    }
  }
  record('Fixture units, revenue arithmetic, directional capacity and exact endpoint coordinates');
  const browser = await chromium.launch({ headless: true });
  try {
    const context = await browser.newContext({ viewport: { width: 1280, height: 1000 }, ignoreHTTPSErrors: true });
    const page = await context.newPage();
    page.on('pageerror', e => errors.push(e.message));
    const f = await attach(page);
    assert.equal(await f.locator('[data-city] option').count(), 94);
    assert.equal(await f.locator('[data-metro]').count(), 94);
    assert.equal(await endpoint(f, 'a'), 'Delhi');
    assert.equal(await endpoint(f, 'b'), 'Mumbai');
    assert.equal(await f.locator('[data-build]').innerText(), '$14.8B');
    record('94 named/map metros, initial distinct endpoints and consistent opening quote');
    await f.locator(rootSelector).screenshot({ path: path.join(evidence, 'connection-planner-desktop.png') });
    await choose(f, 'b', 'delhi');
    assert(await f.locator('[data-error]').isVisible());
    assert.equal(await endpoint(f, 'b'), 'Mumbai');
    await choose(f, 'b', 'tokyo');
    await choose(f, 'a', 'mumbai');
    await choose(f, 'b', 'delhi');
    await ready(f);
    assert.equal(await endpoint(f, 'a'), 'Mumbai');
    assert.equal(await endpoint(f, 'b'), 'Delhi');
    assert.equal(await f.locator('[data-build]').innerText(), '$14.8B');
    await choose(f, 'b', 'tokyo');
    await choose(f, 'a', 'delhi');
    await choose(f, 'b', 'mumbai');
    await ready(f);
    await f.locator('[data-alignment=direct-plateau]').click();
    assert.equal(await f.locator('[data-build]').innerText(), '$19.6B');
    assert.equal(await f.locator('[data-revenue]').innerText(), '$48M');
    assert.equal(await f.locator('[data-contribution]').innerText(), '+$17M');
    assert.equal(await f.locator('.cp-route').getAttribute('data-route'), 'direct-plateau');
    record('Duplicate endpoint rejection, order-independent quote, atomic alignment/estimate update');
    const beforeCamera = await camera(f);
    await f.locator('[data-action=focus]').click(); await sleep(150);
    assert(await f.locator(rootSelector).evaluate(el => el.classList.contains('cp-focus')));
    assert(await f.locator('.cp-inspector').isHidden());
    assert(await f.locator('.cp-strip').isVisible());
    assert(await f.locator('[data-compact-alignment]').isVisible());
    const focusedCamera = await camera(f);
    assert(Math.abs(beforeCamera.k - focusedCamera.k) < 1e-6);
    await f.locator('[data-compact-alignment]').selectOption('western-plains');
    assert.equal(await f.locator('[data-build]').innerText(), '$14.8B');
    await page.keyboard.press('Escape'); await sleep(100);
    assert(await f.locator('.cp-inspector').isVisible());
    assert(Math.abs((await camera(f)).k - beforeCamera.k) < 1e-6);
    assert.equal(await endpoint(f, 'a'), 'Delhi');
    record('Iframe full-screen fallback, alignment switching while expanded, Escape and draft/scale preservation');
    await f.locator('[data-action=review]').click();
    assert(await f.locator('.cp-review').isVisible());
    assert((await f.locator('[data-review-details]').innerText()).includes('$15.5B'));
    assert((await f.locator('[data-review-details]').innerText()).includes('$7.5B'));
    assert(await f.getByRole('button', { name: 'Commit project · funding needed' }).isDisabled());
    await f.locator('[data-action=keep]').click();
    assert(await f.locator('[data-saved]').isVisible());
    assert.equal(await endpoint(f, 'a'), 'Delhi');
    record('Read-only funding review, explicit fleet cost/gap, disabled commitment and local-only draft');
    await choose(f, 'b', 'tokyo');
    assert.equal(await f.locator('[data-build]').innerText(), '—');
    assert(await f.locator('[data-action=review]').isDisabled());
    await f.waitForSelector(rootSelector + '[data-quote-state="No authored route yet"]');
    assert.equal(await f.locator('.cp-route').count(), 0);
    await choose(f, 'b', 'mumbai');
    await f.locator('[data-city]').selectOption('tokyo');
    await sleep(220);
    assert.equal(await endpoint(f, 'b'), 'Tokyo');
    assert.equal(await f.locator(rootSelector).getAttribute('data-quote-state'), 'No authored route yet');
    assert.equal(await f.locator('[data-revenue]').innerText(), '—');
    await f.locator('[data-action=clear]').click();
    assert.equal(await endpoint(f, 'b'), 'Choose metro');
    assert(await f.locator('[data-action=review]').isDisabled());
    record('Missing alignment coverage, clear endpoint, immediate stale-value removal and latest-pair-only async results');
    await choose(f, 'a', 'lagos'); await choose(f, 'b', 'kinshasa'); await ready(f);
    assert.equal(await f.locator('[data-contribution]').innerText(), '−$5M');
    await f.locator('[data-action=pair]').click();
    await choose(f, 'a', 'delhi'); await choose(f, 'b', 'mumbai'); await ready(f);
    await f.locator('[data-action=pair]').click();
    record('Independent unserved-pair planning and clearly negative operating contribution');
    const stage = await f.locator(stageSelector).boundingBox();
    const aBefore = await endpoint(f, 'a'), bBefore = await endpoint(f, 'b');
    await page.mouse.move(stage.x + stage.width / 2, stage.y + stage.height / 2);
    await page.mouse.down(); await page.mouse.move(stage.x + stage.width / 2 + 75, stage.y + stage.height / 2 + 40, { steps: 8 }); await page.mouse.up();
    assert.equal(await endpoint(f, 'a'), aBefore); assert.equal(await endpoint(f, 'b'), bBefore);
    await f.locator('[data-action=pair]').click();
    const point = await f.locator('[data-metro=ahmedabad]').evaluate(el => ({ x: +el.getAttribute('cx'), y: +el.getAttribute('cy') }));
    await f.locator('[data-slot=b]').click();
    const rect = await f.locator(stageSelector).boundingBox();
    await page.mouse.click(rect.x + point.x, rect.y + point.y);
    if (await f.locator('[data-candidates]').isVisible()) await f.locator('[data-choose=ahmedabad]').click();
    assert.equal(await endpoint(f, 'b'), 'Ahmedabad');
    record('Map marker selection and drag pan preserving endpoint identities');
    await f.locator('[data-region]').selectOption('east-asia');
    const guangzhou = await f.locator('[data-metro=guangzhou]').evaluate(el => ({ x: +el.getAttribute('cx'), y: +el.getAttribute('cy') }));
    const eastRect = await f.locator(stageSelector).boundingBox();
    await page.mouse.click(eastRect.x + guangzhou.x, eastRect.y + guangzhou.y);
    assert(await f.locator('[data-candidates]').isVisible());
    assert(await f.locator('[data-choose=guangzhou]').isVisible());
    record('Overlapping markers require named disambiguation before assignment');
    await choose(f, 'a', 'delhi'); await choose(f, 'b', 'mumbai'); await ready(f);
    await f.locator('[data-action=pair]').click();
    const restored = await saved(f);
    await choose(f, 'b', 'tokyo');
    await f.evaluate(v => window.dispatchEvent(new CustomEvent('openai:set_globals', { detail: { globals: { widgetState: { privateContent: { connectionPlanner: { ...v, active: 'a' } } } } } })), restored);
    await ready(f); assert.equal(await endpoint(f, 'b'), 'Mumbai');
    record('Host state restore without camera/default resets or committed operations');
    for (const w of [320, 390, 768, 1280]) {
      await page.setViewportSize({ width: w, height: w < 500 ? 844 : 1000 }); await sleep(130);
      await noOverflow(f, 'Normal at ' + w);
      if (w < 500) assert(await f.locator('[data-action=world]').isHidden());
      if (w === 390) await f.locator(rootSelector).screenshot({ path: path.join(evidence, 'connection-planner-phone.png') });
      await f.locator('[data-action=focus]').click(); await sleep(100);
      await noOverflow(f, 'Focused at ' + w);
      if (w === 390) await f.locator(rootSelector).screenshot({ path: path.join(evidence, 'connection-planner-phone-expanded.png') });
      assert(await f.locator('[data-action=focus]').isVisible());
      assert(await f.locator('.cp-strip').isVisible());
      await f.locator('[data-action=focus]').click();
    }
    record('Normal/expanded layouts fit 320, 390, 768 and 1280px, with portrait World removed');
    await page.setViewportSize({ width: 390, height: 844 });
    for (let i = 0; i < 15; i++) {
      const out = f.locator('[data-action=out]');
      if (await out.isDisabled()) break;
      await out.click();
    }
    assert(await f.locator('[data-action=out]').isDisabled());
    const cap = await f.locator(stageSelector).evaluate(el => ({ limits: JSON.parse(el.dataset.limits), camera: JSON.parse(el.dataset.camera) }));
    assert.deepEqual(cap.limits, [-40.8136, 66.63292]);
    record('Portrait min zoom and cushioned Melbourne–Hörnefors latitude band', cap.limits);
    await page.emulateMedia({ colorScheme: 'dark', reducedMotion: 'reduce' });
    await f.locator('[data-action=pair]').click(); await noOverflow(f, 'Dark reduced motion');
    await f.locator(rootSelector).screenshot({ path: path.join(evidence, 'connection-planner-dark.png') });
    record('Dark appearance and reduced-motion layout');
    // Native fullscreen is exercised separately, outside render.py's restrictive iframe.
    const native = await context.newPage();native.on('pageerror', e => errors.push(e.message));
    // The source is an inline fragment; supply the UTF-8 document encoding its host normally supplies.
    await native.route('**/connection-planner.html', async route => {
      const response = await route.fetch();
      await route.fulfill({ response, headers: { ...response.headers(), 'content-type': 'text/html; charset=utf-8' } });
    });
    await native.goto(url.replace('connection-planner-preview.html', 'connection-planner.html'));
    await native.waitForSelector(rootSelector + '[data-quote-state=ready]');
    await native.locator('[data-action=focus]').click(); await sleep(150);
    assert(await native.evaluate(() => !!document.fullscreenElement));
    const fullSize = await native.locator(rootSelector).evaluate(el => ({ h: el.clientHeight, scroll: el.scrollHeight, window: window.innerHeight }));
    assert(fullSize.scroll <= fullSize.h + 2, JSON.stringify(fullSize));
    await native.evaluate(() => document.exitFullscreen()); await sleep(100);
    assert.equal(await native.locator('[data-focus-label]').innerText(), 'Expand map');
    record('Native browser full-screen entry, viewport fit and system exit synchronizing map focus', fullSize);
    await native.setViewportSize({ width: 390, height: 844 });
    await native.locator('[data-action=focus]').click(); await sleep(150);
    const phoneFull = await native.locator(rootSelector).evaluate(el => ({ h: el.clientHeight, scroll: el.scrollHeight, stage: el.querySelector('[data-stage]').clientHeight, window: window.innerHeight }));
    assert(phoneFull.scroll <= phoneFull.h + 2, JSON.stringify(phoneFull));
    assert(phoneFull.stage >= 260);
    await native.screenshot({ path: path.join(evidence, 'connection-planner-phone-fullscreen.png') });
    await native.evaluate(() => document.exitFullscreen());
    record('390×844 full-screen phone keeps map, endpoint controls, alignment and financial review in viewport', phoneFull);
    assert.deepEqual(errors, []);
    const report = { passed: true, prototypeOnly: true, browser: browser.version(), checks: results, limitations: ['Illustrative alignments and financials, not route engineering or economic calibration', 'Browser checks, not Ebitengine implementation or physical-phone qualification', 'Static CDN import does not establish offline cold launch'] };
    fs.writeFileSync(path.join(evidence, 'connection-planner-checks.json'), JSON.stringify(report, null, 2) + '\n');
    console.log(JSON.stringify(report, null, 2));
  } finally { await browser.close(); }
})().catch(e => { console.error(e); process.exit(1); });
