const { control, language, prospect, selected } = require('./ui-helpers.cjs');
// Focused real Go/WASM + semantic-HUD business workflow; no physical-device claim.
const assert = require('node:assert/strict'),
  fs = require('node:fs'),
  path = require('node:path'),
  crypto = require('node:crypto');
const { execFileSync } = require('node:child_process');
const { chromium } = require('playwright');
const url = process.argv[2] || 'http://127.0.0.1:8765/',
  out = path.resolve('docs/validation');
const snap = (p) =>
  p.evaluate(() =>
    Promise.race([railSnapshot(), new Promise((_, r) => setTimeout(() => r(Error('snapshot timeout')), 15000))]),
  );
async function ready(p) {
  await p.goto(url);
  await p.waitForFunction(() => !!window.railSnapshot, null, { timeout: 60000 });
  await p.waitForFunction(async () => (await railSnapshot()).camera.Width > 100);
}
async function idle(p) {
  await p.waitForFunction(() => document.getElementById('game').getAttribute('aria-busy') === 'false');
}
async function clickCommand(p, id, trace, command) {
  const revision = (await snap(p)).frame.revision;
  await (await control(p, '#' + id)).click();
  await p.waitForFunction(async (r) => (await railSnapshot()).frame.revision === r + 1, revision);
  await idle(p);
  trace.push(command);
}
(async () => {
  const browser = await chromium.launch({ headless: true, args: ['--enable-unsafe-swiftshader'] });
  // Hosted runners have taken 106 s to over 180 s; two were still in the offline cold launch at 180 s.
  const watchdog = setTimeout(() => browser.close(), 240000);
  const result = {
    date: new Date().toISOString(),
    browser: browser.version(),
    wasm_sha256: crypto.createHash('sha256').update(fs.readFileSync('web/main.wasm')).digest('hex'),
    checks: [],
    limitations: [
      'Synthetic desktop/touch/IME inputs; physical phones and native IME remain unqualified.',
      'Native canvas accessibility is still incomplete.',
    ],
  };
  try {
    const ctx = await browser.newContext({ viewport: { width: 1280, height: 800 }, reducedMotion: 'reduce' }),
      p = await ctx.newPage(),
      errors = [],
      trace = [];
    p.on('pageerror', (e) => errors.push(e.message));
    await ready(p);
    let f = (await snap(p)).frame;
    assert(f.public_network && f.cities.length >= 250);
    assert.equal(f.services[0].frequency, 6);
    assert.equal(f.fleet.length, 2);
    assert.equal(f.idle_lines.length, 0);
    assert(await p.locator('#idle-warning').isHidden());
    const cdp = await ctx.newCDPSession(p),
      ax = await cdp.send('Accessibility.getFullAXTree');
    assert(ax.nodes.some((n) => n.role?.value === 'button' && n.name?.value === 'Advance month'));
    const box = await p.locator('#map-frame').boundingBox(),
      at = (await snap(p)).points.seoul;
    await p.mouse.click(box.x + at.X, box.y + at.Y);
    await selected(p, 'seoul');
    await p.locator('#advance').focus();
    await p.keyboard.press('Enter');
    await p.waitForFunction(async () => (await railSnapshot()).frame.months_resolved === 1);
    trace.push({ kind: 'advance_month' });
    await idle(p);
    f = (await snap(p)).frame;
    assert(f.reports[0].revenue > 0 && f.reports[0].passengers > 0 && f.player.cash > 10000000000);
    await p.screenshot({ path: path.join(out, 'business-revenue-desktop.png') });
    result.checks.push(
      'Fresh company has an active starter service; map selection, ' +
        'AX names and keyboard advancement work; first month carries passengers, ' +
        'collects fares and posts costs',
    );

    await (await control(p, '#tab-company')).click();
    await (await control(p, '#company-name')).fill('');
    await cdp.send('Input.imeSetComposition', { text: '晨曦', selectionStart: 2, selectionEnd: 2 });
    const rev = (await snap(p)).frame.revision;
    await p.keyboard.press('Enter');
    assert.equal((await snap(p)).frame.revision, rev);
    await cdp.send('Input.insertText', { text: '晨曦铁路' });
    await p.waitForFunction(() => !document.getElementById('rename').disabled);
    await clickCommand(p, 'rename', trace, { kind: 'rename_company', name: '晨曦铁路' });
    await (await control(p, '#tab-build')).click();
    await (await control(p, '[data-corridor=tsuruga-osaka]')).click();
    await (
      await control(p, '#alignments button')
    )
      .locator('..')
      .locator('button')
      .filter({ hasText: 'Valley detour' })
      .click();
    assert(await (await control(p, '#quote')).innerText().then((s) => s.includes('11,633,400,000')));
    const priorLoan = (await snap(p)).frame,
      route = priorLoan.corridors.find((c) => c.id === 'tsuruga-osaka'),
      alternative = route.alignments.find((a) => a.id === 'valley'),
      shortfall = alternative.cost - alternative.grant - priorLoan.player.cash;
    await clickCommand(p, 'borrow-shortfall-build', trace, { kind: 'take_loan', amount: shortfall });
    assert.equal((await snap(p)).frame.debt, shortfall);
    await (await control(p, '#focus-route')).click();
    assert((await snap(p)).camera.Zoom > 1);
    await (await control(p, '#expand')).click();
    assert(await p.locator('#inspector').isHidden());
    await p.keyboard.press('Escape');
    assert(await p.locator('#inspector').isVisible());
    await clickCommand(p, 'build-line', trace, {
      kind: 'build_line',
      corridor_id: 'tsuruga-osaka',
      alignment_id: 'valley',
    });
    f = (await snap(p)).frame;
    assert.equal(f.projects.length, 1);
    assert.equal(f.projects[0].ready, 49);
    assert(f.pending.grants > 0 && f.pending.construction > f.pending.grants);
    assert.equal(await p.locator('#opportunities button[data-corridor=tsuruga-osaka]').count(), 0);
    assert.equal(await p.locator('#opportunities button[data-corridor=hokkaido-extension]').count(), 1);
    assert(await p.locator('#project-progress').isVisible());
    assert.equal(f.idle_lines.length, 0);
    assert(await p.locator('#review-idle').isHidden());
    for (let i = 0; i < 4; i++) await clickCommand(p, 'advance-year', trace, { kind: 'advance_year' });
    f = (await snap(p)).frame;
    assert.equal(f.months_resolved, 49);
    assert.equal(f.projects[0].ready, 49);
    assert(
      await p
        .locator('#build-status')
        .innerText()
        .then((s) => s.includes('ready for services')),
    );
    assert.equal(await p.locator('#opportunities button[data-corridor=tsuruga-osaka]').count(), 0);
    assert.equal(f.idle_lines.length, 1);
    assert.equal(f.idle_lines[0].maintenance, 193890000);
    assert.equal(f.idle_lines[0].needs_trains, true);
    assert(await p.locator('#idle-warning').isVisible());
    assert(await p.locator('#review-idle').isVisible());
    await p.screenshot({ path: path.join(out, 'business-idle-line-desktop.png') });
    await language(p, 'zh-Hans');
    await p.waitForFunction(() => document.documentElement.lang === 'zh-Hans');
    await p.setViewportSize({ width: 320, height: 740 });
    // The client lays out again after a bridge round trip, so wait for the phone layout rather than catch the old one.
    await p.waitForFunction(() => document.documentElement.scrollWidth <= innerWidth);
    const warningBox = await p.locator('#review-idle').boundingBox(),
      advanceBox = await p.locator('#advance').boundingBox();
    assert(warningBox.x >= 0 && warningBox.x + warningBox.width <= 320 && advanceBox.y + advanceBox.height <= 740);
    await p.screenshot({ path: path.join(out, 'business-idle-line-phone-zh.png') });
    await p.setViewportSize({ width: 1280, height: 800 });
    await language(p, 'en');
    await p.waitForFunction(() => document.documentElement.lang === 'en');
    await (await control(p, '#review-idle')).click();
    assert.equal(await p.locator('#service-corridor').inputValue(), 'tsuruga-osaka');
    assert(await p.locator('[data-idle-corridor=tsuruga-osaka]').isVisible());
    await clickCommand(p, 'lease-train', trace, { kind: 'lease_train', corridor_id: 'tsuruga-osaka' });
    assert.equal((await snap(p)).frame.idle_lines[0].needs_trains, false);
    const leased = (await snap(p)).frame;
    assert.equal(leased.idle_lines[0].leases, leased.train_family.lease_monthly); // one train's monthly lease
    assert(await p.locator('#idle-warning').isVisible());
    // The suggested plan now opens at 8 trains a day; enter the plan the trace records, which one train runs.
    await (await control(p, '#frequency')).fill('4');
    await (await control(p, '#fare')).fill('90');
    await p.waitForFunction(() => !document.getElementById('apply-service').disabled);
    await clickCommand(p, 'apply-service', trace, {
      kind: 'set_service',
      corridor_id: 'tsuruga-osaka',
      frequency: 4,
      fare: 9000,
    });
    assert.equal((await snap(p)).frame.idle_lines.length, 0);
    assert(await p.locator('#review-idle').isHidden());
    await (await control(p, '#frequency')).fill('0');
    await p.waitForFunction(() => !document.getElementById('apply-service').disabled);
    await clickCommand(p, 'apply-service', trace, {
      kind: 'set_service',
      corridor_id: 'tsuruga-osaka',
      frequency: 0,
      fare: 9000,
    });
    assert(await p.locator('#review-idle').isVisible());
    assert.equal((await snap(p)).frame.idle_lines.length, 1);
    assert(!(await p.locator('#advance').isDisabled()));
    await (await control(p, '#frequency')).fill('4');
    assert(await p.locator('#idle-warning').isVisible());
    await p.waitForFunction(() => !document.getElementById('apply-service').disabled);
    await clickCommand(p, 'apply-service', trace, {
      kind: 'set_service',
      corridor_id: 'tsuruga-osaka',
      frequency: 4,
      fare: 9000,
    });
    assert(await p.locator('#idle-warning').isHidden());
    await clickCommand(p, 'advance', trace, { kind: 'advance_month' });
    f = (await snap(p)).frame;
    const service = f.reports.at(-1).services.find((s) => s.corridor_id === 'tsuruga-osaka');
    assert(service.passengers > 0 && service.revenue > 0);
    assert(f.reports.at(-1).maintenance > 0);
    for (const r of f.reports)
      assert.equal(
        r.closing_cash,
        r.opening_cash + r.revenue - r.costs + r.grants + r.borrowed - r.repaid - r.construction,
      );
    result.checks.push(
      'Two-endpoint reverse-order planning and route focus; alternative engineering prices; ' +
        'single exact-shortfall loan, editable custom loans with cents and credit-limit validation, ' +
        'and 90% grant; ' +
        'committed lines disappear from opportunities immediately and stay absent after completion/reload while project progress remains visible; ' +
        'construction progress across 48 months; opening eligibility; ' +
        'persistent idle-line warnings with actual upkeep/idle-lease costs, 320px Chinese footer, ' +
        'service shortcut, accepted-service clearing and suspension reappearance; ' +
        'dedicated train lease; accepted service earns new-line revenue and pays upkeep; ' +
        'every report reconciles',
    );

    await (await control(p, '#tab-company')).click();
    await (await control(p, '#loan-amount')).fill('0');
    assert(await p.locator('#borrow').isDisabled());
    await (await control(p, '#loan-amount')).fill('6,000,000,000');
    assert(await p.locator('#borrow').isDisabled());
    await (await control(p, '#loan-amount')).fill('12,345.67');
    await clickCommand(p, 'borrow', trace, { kind: 'take_loan', amount: 1234567 });
    await (await control(p, '#company-name')).fill('保留此草稿');
    await language(p, 'zh-Hans');
    await p.waitForFunction(() => document.documentElement.lang === 'zh-Hans');
    assert.equal(await p.locator('#company-name').inputValue(), '保留此草稿');
    f = (await snap(p)).frame;
    fs.mkdirSync('build', { recursive: true });
    const log = path.resolve('build/business-commands.json');
    fs.writeFileSync(log, JSON.stringify(trace));
    const native = JSON.parse(
      execFileSync(path.resolve('build/headless'), ['-commands', log], {
        encoding: 'utf8',
        maxBuffer: 32 * 1024 * 1024,
        stdio: ['ignore', 'pipe', 'pipe'],
      }),
    );
    assert.equal(f.checkpoint, native.checkpoint);
    result.checkpoint = f.checkpoint;
    await p.setViewportSize({ width: 320, height: 740 });
    await (await control(p, '#tab-map')).click();
    await (await control(p, '#fit')).click();
    await p.waitForFunction(async () => (await railSnapshot()).camera.Width === 320);
    const touchMap = await p.locator('#map-frame').boundingBox(),
      touchCity = (await snap(p)).points.chengdu;
    await cdp.send('Input.dispatchTouchEvent', {
      type: 'touchStart',
      touchPoints: [{ x: touchMap.x + touchCity.X, y: touchMap.y + touchCity.Y }],
    });
    await p.waitForTimeout(100);
    await cdp.send('Input.dispatchTouchEvent', { type: 'touchEnd', touchPoints: [] });
    await selected(p, 'chengdu');
    await (await control(p, '#tab-report')).click();
    assert(await p.evaluate(() => document.documentElement.scrollWidth <= innerWidth));
    await p.evaluate(() => scrollTo(0, 0));
    await p.screenshot({ path: path.join(out, 'business-report-phone-zh.png') });
    await (await control(p, '#tab-build')).click();
    await prospect(p, 'hokkaido-extension');
    await p.evaluate(() => {
      const q = document.getElementById('quote');
      scrollTo(0, q.getBoundingClientRect().top + scrollY - 100);
    });
    await p.screenshot({ path: path.join(out, 'business-build-phone-zh.png') });
    for (const width of [390, 768, 1920]) {
      await p.setViewportSize({ width, height: width > 1000 ? 1080 : 844 });
      assert(await p.evaluate(() => document.documentElement.scrollWidth <= innerWidth));
    }
    await p.setViewportSize({ width: 1280, height: 800 });
    await (await control(p, '#tab-services')).click();
    await (await control(p, '#service-corridor')).selectOption('tokyo-nagoya');
    await (await control(p, '#frequency')).fill('16');
    await p.waitForFunction(() => document.getElementById('forecast-status').textContent.includes('租赁足够'));
    assert(await p.locator('#apply-service').isDisabled());
    result.checks.push(
      'Native-headless and real WASM replay identical accepted commands to an identical checkpoint; ' +
        'Chinese composition and draft retention; 320–1920px reflow; ' +
        'fleet shortage blocks an infeasible service',
    );

    await (await control(p, '#tab-report')).click();
    const before = (await snap(p)).frame.checkpoint;
    await p.evaluate(() => {
      window.commitBeforeFailure = railStorageCommit;
      window.railStorageCommit = () => Promise.reject(Error('injected disk full'));
    });
    await (await control(p, '#advance')).click();
    await p.waitForFunction(() => document.getElementById('status').textContent.includes('无法保存'));
    assert.equal((await snap(p)).frame.checkpoint, before);
    await p.evaluate(() => (window.railStorageCommit = window.commitBeforeFailure));
    const second = await ctx.newPage();
    await second.goto(url);
    await second.waitForFunction(() => document.getElementById('loading').textContent.includes('另一个标签页'));
    await second.close();
    await p.reload();
    await p.waitForFunction(() => !!window.railSnapshot);
    assert.equal((await snap(p)).frame.checkpoint, before);
    assert.equal(await p.locator('#company-name').inputValue(), '保留此草稿');
    assert.equal(await p.locator('#opportunities button[data-corridor=tsuruga-osaka]').count(), 0);
    await p.evaluate(() => navigator.serviceWorker.ready);
    await p.waitForFunction(() => !!navigator.serviceWorker.controller);
    await p.close();
    await ctx.setOffline(true);
    const offline = await ctx.newPage();
    await ready(offline);
    assert.equal((await snap(offline)).frame.checkpoint, before);
    const oldMonths = (await snap(offline)).frame.months_resolved;
    await (await control(offline, '#advance')).click();
    await offline.waitForFunction(async (n) => (await railSnapshot()).frame.months_resolved === n + 1, oldMonths);
    result.checks.push(
      'Rejected storage write preserves accepted financial state; second tab excluded; company, ' +
        'projects, fleet and drafts resume; ' +
        'cached application opens offline and continues earning revenue',
    );
    await ctx.setOffline(false);
    await offline.evaluate(() => railStorageCommit('{"bad":true}'));
    await offline.reload();
    await offline.waitForFunction(() => document.getElementById('loading').textContent.includes('存档无效'));
    assert.equal(await offline.evaluate(() => railStorageLoad()), '{"bad":true}');
    assert.deepEqual(errors, []);
    result.result = 'passed';
    fs.writeFileSync(path.join(out, 'business-checks.json'), JSON.stringify(result, null, 2) + '\n');
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
