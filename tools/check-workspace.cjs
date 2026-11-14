// Task workspace acceptance: real semantic controls and Go/WASM, isolated save.
const assert = require('node:assert/strict'),
  fs = require('node:fs'),
  path = require('node:path'),
  crypto = require('node:crypto');
const { chromium } = require('playwright'),
  { control, language } = require('./ui-helpers.cjs');
const url = process.argv[2] || 'http://127.0.0.1:8765/',
  out = path.resolve('docs/validation');
const snap = (p) => p.evaluate(() => railSnapshot());
async function ready(p) {
  await p.goto(url);
  await p.waitForFunction(() => !!window.railSnapshot, null, { timeout: 60000 });
  await p.waitForFunction(() => document.getElementById('startup').hidden);
}
async function planReady(p) {
  await p.waitForFunction(
    () =>
      document.querySelectorAll('#forecast-values>div').length === 3 &&
      document.querySelector('#services-section-plan .learning-cue')?.getClientRects().length > 0,
  );
}
async function accepted(p, id) {
  const rev = (await snap(p)).frame.revision;
  await (await control(p, '#' + id)).click();
  await p.waitForFunction(async (r) => (await railSnapshot()).frame.revision === r + 1, rev);
  await p.waitForFunction(() => document.getElementById('game').getAttribute('aria-busy') === 'false');
}
async function fits(p) {
  assert(
    await p.evaluate(
      () =>
        document.documentElement.scrollWidth <= innerWidth && document.documentElement.scrollHeight <= innerHeight + 1,
    ),
    'body overflow',
  );
  const data = await p
    .locator('#inspector')
    .evaluate((e) => ({ width: e.clientWidth, height: e.clientHeight, scroll: e.scrollHeight, top: e.scrollTop }));
  assert.equal(data.top, 0);
  assert(data.scroll <= data.height + 1, 'normal service plan needs scrolling: ' + JSON.stringify(data));
  const b = await p.locator('#apply-service').boundingBox(),
    a = await p.locator('#inspector').boundingBox();
  assert(b.y >= a.y && b.y + b.height <= a.y + a.height, 'Apply offscreen');
  return data;
}
(async () => {
  const browser = await chromium.launch({ headless: true, args: ['--enable-unsafe-swiftshader'] }),
    watchdog = setTimeout(() => browser.close(), 360000);
  const result = {
    date: new Date().toISOString(),
    browser: browser.version(),
    wasm_sha256: crypto.createHash('sha256').update(fs.readFileSync('web/main.wasm')).digest('hex'),
    ui_sha256: crypto
      .createHash('sha256')
      .update(Buffer.concat(['web/app.js', 'web/style.css', 'web/index.html'].map((f) => fs.readFileSync(f))))
      .digest('hex'),
    checks: [],
    layouts: [],
    limitations: [
      'Browser touch, keyboard and phone layouts are emulated; physical devices, ' +
        'screen readers and native OS input require separate qualification.',
    ],
  };
  try {
    console.log('Starting workspace');
    const ctx = await browser.newContext({ viewport: { width: 1280, height: 850 }, reducedMotion: 'reduce' }),
      p = await ctx.newPage(),
      errors = [];
    p.setDefaultTimeout(15000);
    p.setDefaultNavigationTimeout(30000);
    p.on('pageerror', (e) => errors.push(e.message));
    await ready(p);
    const initial = (await snap(p)).frame.checkpoint,
      bytes = await p.evaluate(() => railStorageLoad());
    await p.locator('#tab-services').click();
    await planReady(p);
    assert.equal(await p.locator('#capacity-disclosure').evaluate((e) => e.open), false);
    assert(await p.locator('#service-capacity').isHidden());
    assert(await p.locator('#train-summary').isHidden());
    result.layouts.push({ name: 'desktop-top-plan', ...(await fits(p)) });
    await p.screenshot({ path: path.join(out, 'workspace-services-desktop.png') });
    console.log('Editing draft and tabs');
    await p.locator('#frequency').fill('7');
    await p.locator('#fare').fill('95');
    for (const pane of ['fleet', 'analysis', 'plan']) {
      await p.locator('#services-pane-' + pane).click();
      assert.equal((await snap(p)).state.frequency, 7);
      assert.equal((await snap(p)).state.fare, 9500);
      assert.equal(await p.locator('#services-pane-' + pane).getAttribute('aria-selected'), 'true');
    }
    console.log('Keyboard tabs');
    await p.locator('#services-pane-plan').focus();
    await p.keyboard.press('ArrowRight');
    await p.waitForFunction(
      () => document.getElementById('services-pane-fleet').getAttribute('aria-selected') === 'true',
    );
    assert.equal(await p.locator('#services-pane-fleet').evaluate((e) => e === document.activeElement), true);
    await p.keyboard.press('End');
    await p.waitForFunction(
      () => document.getElementById('services-pane-analysis').getAttribute('aria-selected') === 'true',
    );
    await p.keyboard.press('Home');
    await p.waitForFunction(
      () => document.getElementById('services-pane-plan').getAttribute('aria-selected') === 'true',
    );
    console.log('Company and layout');
    await p.locator('#tab-company').click();
    assert(await p.locator('#loan-amount').isVisible());
    assert(await p.locator('#company-name').isHidden());
    assert(await p.locator('#company-funding').isHidden());
    await p.locator('#loan-amount').fill('12345.67');
    await p.locator('#company-pane-identity').click();
    await p.locator('#company-name').fill('Workspace 草案');
    await p.locator('#company-pane-settings').click();
    await p.locator('#command-dock').selectOption('left');
    await p.waitForFunction(() => document.body.dataset.commandDock === 'left');
    const dock = await p.locator('#workspace-dock').boundingBox(),
      inspector = await p.locator('#inspector').boundingBox();
    assert.equal(dock.x, 0);
    assert.equal(dock.width, 164);
    assert(inspector.x > dock.width);
    console.log('Dragging divider');
    const divider = await p.locator('#inspector-resize').boundingBox();
    await p.mouse.move(divider.x + 4, divider.y + 100);
    await p.mouse.down();
    await p.mouse.move(680, divider.y + 100, { steps: 5 });
    await p.mouse.up();
    await p.waitForFunction(async () => (await railSnapshot()).state.panel_width === 600);
    assert.equal((await p.locator('#inspector').boundingBox()).width, 600);
    console.log('Keyboard divider');
    await p.locator('#inspector-resize').focus();
    await p.keyboard.press('ArrowLeft');
    await p.waitForFunction(async () => (await railSnapshot()).state.panel_width === 620);
    await p.keyboard.press('ArrowRight');
    await p.waitForFunction(async () => (await railSnapshot()).state.panel_width === 600);
    console.log('Left service plan');
    await p.locator('#tab-services').click();
    await p.locator('#frequency').fill('6');
    await planReady(p);
    result.layouts.push({ name: 'desktop-left-plan', ...(await fits(p)) });
    await p.screenshot({ path: path.join(out, 'workspace-left-dock-desktop.png') });
    assert(await p.locator('#train-values').isHidden());
    assert.equal((await snap(p)).frame.checkpoint, initial);
    assert.equal(await p.evaluate(() => railStorageLoad()), bytes);
    result.checks.push(
      'Focused service plan fits at 1280x850; ' +
        'fleet and analysis tabs plus collapsed capacity remove secondary detail; ' +
        'pointer and keyboard tabs preserve fare/frequency; finance, ' +
        'identity and settings separate; UI-only actions leave exact save bytes unchanged',
    );
    await p.locator('#tab-company').click();
    await p.locator('#company-pane-finance').click();
    assert.equal(await p.locator('#loan-amount').inputValue(), '12345.67');
    await p.locator('#company-pane-identity').click();
    assert.equal(await p.locator('#company-name').inputValue(), 'Workspace 草案');
    console.log('Responsive language loop');
    for (const locale of ['en', 'zh-Hans']) {
      await language(p, locale);
      for (const size of [
        { width: 390, height: 844 },
        { width: 320, height: 740 },
        { width: 1920, height: 1080 },
      ]) {
        await p.setViewportSize(size);
        await p.locator('#tab-services').click();
        await p.locator('#services-pane-plan').click();
        await planReady(p);
        if (size.width === 390) result.layouts.push({ name: 'phone-plan-' + locale, ...(await fits(p)) });
        for (const pane of ['plan', 'fleet', 'analysis']) {
          await p.locator('#services-pane-' + pane).click();
          assert(await p.evaluate(() => document.documentElement.scrollWidth <= innerWidth));
        }
        await p.locator('#services-pane-plan').click();
        await p.waitForFunction(
          async (w) => (await railSnapshot()).workspace.PanelWidth === (w < 800 ? w : 600),
          size.width,
        );
        assert.equal(await p.locator('body').getAttribute('data-command-dock'), size.width < 800 ? 'top' : 'left');
        assert.equal((await snap(p)).state.command_dock, 'left');
        assert.equal((await snap(p)).state.panel_width, 600);
      }
    }
    await p.setViewportSize({ width: 390, height: 844 });
    await p.locator('#tab-services').click();
    await planReady(p);
    await p.screenshot({ path: path.join(out, 'workspace-services-phone-zh.png') });
    console.log('Build and expanded map');
    await language(p, 'en');
    await p.locator('#tab-build').click();
    await p.locator('#build-pane-costs').click();
    assert(await p.locator('#engineering-values').isVisible());
    assert.equal(await p.locator('#from-city').count(), 0);
    await p.locator('#build-pane-plan').click();
    assert(await p.locator('#opportunities').isVisible());
    await p.locator('#browse-prospects').click();
    await p.waitForFunction(() => document.body.classList.contains('expanded'));
    assert(await p.locator('#workspace-dock').isHidden());
    await control(p, '#zoom-in');
    assert(await p.locator('#zoom-in').isVisible());
    const zoom = (await snap(p)).camera.Zoom;
    await p.locator('#zoom-in').click();
    await p.waitForFunction(async (z) => (await railSnapshot()).camera.Zoom > z, zoom);
    await (await control(p, '#focus-route')).click();
    await p.keyboard.press('Escape');
    assert(await p.locator('#inspector').isVisible());
    assert.equal((await snap(p)).state.to_city, 'osaka');
    console.log('Reference task jump');
    await p.locator('#tab-services').click();
    await p.locator('#help-open').click();
    await p.locator('#help-search').fill('train');
    await p.locator('#help-results button').filter({ hasText: 'Trains for the timetable' }).click();
    await p.locator('#help-article button').last().click();
    await p.waitForFunction(
      () => document.getElementById('services-pane-fleet').getAttribute('aria-selected') === 'true',
    );
    assert(await p.locator('#train-summary').isVisible());
    result.checks.push(
      '320–1920px English/Chinese panes avoid horizontal overflow; ' +
        '390x844 service plan fits without scrolling; ' +
        'phone fallback preserves left-dock/600px desktop preferences; ' +
        'full map retains focus and zoom controls; ' +
        'Field Notes task link opens the relevant Fleet tab',
    );
    console.log('Business and reports');
    await p.locator('#services-pane-plan').click();
    await planReady(p);
    await accepted(p, 'apply-service');
    assert.equal((await snap(p)).frame.services[0].fare, 9500);
    await accepted(p, 'advance');
    assert(await p.locator('#report-summary').isVisible());
    assert.equal(await p.locator('#report-summary>div').count(), 4);
    assert(await p.locator('#ledger').isHidden());
    await p.locator('#report-pane-accounts').click();
    assert(await p.locator('#ledger').isVisible());
    await p.locator('#report-pane-network').click();
    assert(await p.locator('#network-totals').isVisible());
    await p.locator('#report-pane-markets').click();
    assert(await p.locator('#market-from').isVisible());
    console.log('Reload and offline');
    const checkpoint = (await snap(p)).frame.checkpoint;
    await p.reload();
    await p.waitForFunction(() => !!window.railSnapshot, null, { timeout: 60000 });
    assert.equal((await snap(p)).frame.checkpoint, checkpoint);
    await p.locator('#tab-company').click();
    await p.locator('#company-pane-identity').click();
    assert.equal(await p.locator('#company-name').inputValue(), 'Workspace 草案');
    await p.evaluate(() => navigator.serviceWorker.ready);
    await p.waitForFunction(() => !!navigator.serviceWorker.controller);
    await ctx.setOffline(true);
    await p.reload();
    await p.waitForFunction(() => !!window.railSnapshot, null, { timeout: 60000 });
    assert.equal((await snap(p)).frame.checkpoint, checkpoint);
    assert.equal((await snap(p)).state.panel_width, 600);
    await accepted(p, 'advance');
    await p.locator('#tab-company').click();
    await p.locator('#company-pane-identity').click();
    await p.locator('#new-game').click();
    await accepted(p, 'confirm-new-game');
    const fresh = await snap(p);
    assert.equal(fresh.frame.months_resolved, 0);
    assert.equal(fresh.state.command_dock, 'left');
    assert.equal(fresh.state.panel_width, 600);
    assert.equal(fresh.state.service_corridor, 'tokyo-nagoya');
    assert.equal(fresh.state.fare, 9000);
    result.checks.push(
      'Applied fare plan earns revenue; Overview, Accounts, ' +
        'Network and Markets expose the intended report detail; ' +
        'reload and offline continuation retain campaign/drafts/layout; ' +
        'New game retains workspace preferences while clearing business drafts',
    );
    assert.deepEqual(errors, []);
    result.passed = true;
    fs.writeFileSync(path.join(out, 'workspace-checks.json'), JSON.stringify(result, null, 2) + '\n');
    console.log(JSON.stringify(result, null, 2));
    await ctx.close();
  } finally {
    clearTimeout(watchdog);
    await browser.close();
  }
})().catch((e) => {
  console.error(e);
  process.exitCode = 1;
});
