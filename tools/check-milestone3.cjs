const { control, language } = require('./ui-helpers.cjs');
// Real semantic-HUD/Go-WASM workflow, with native-headless replay and v2 upgrade.
const assert = require('node:assert/strict');
const fs = require('node:fs'),
  path = require('node:path'),
  crypto = require('node:crypto');
const { execFileSync } = require('node:child_process');
const { chromium } = require('playwright');
const url = process.argv[2] || 'http://127.0.0.1:8765/';
const out = path.resolve('docs/validation');
const snap = (p) => p.evaluate(() => railSnapshot());
async function ready(p) {
  await p.goto(url);
  await p.waitForFunction(() => !!window.railSnapshot, null, { timeout: 60000 });
  await p.waitForFunction(async () => (await railSnapshot()).camera.Width > 100);
}
async function settled(p) {
  await p.waitForFunction(() => document.querySelector('#game').getAttribute('aria-busy') === 'false');
}
async function act(p, id, trace, command) {
  console.log('Action', id, command.kind, command.corridor_id || '');
  const revision = (await snap(p)).frame.revision;
  await (await control(p, '#' + id)).click();
  await p
    .waitForFunction(async (r) => (await railSnapshot()).frame.revision === r + 1, revision, { timeout: 20000 })
    .catch(async (e) => {
      console.error(
        'Action status:',
        await p.locator('#status').innerText(),
        await p.locator('#forecast-status').innerText(),
      );
      throw e;
    });
  await settled(p);
  trace.push(command);
}
async function apply(p, trace, command) {
  await p.waitForFunction(() => !document.querySelector('#apply-service').disabled);
  await act(p, 'apply-service', trace, command);
}
function conserved(n) {
  for (const m of n.markets) {
    assert.equal(m.carried + m.road + m.air + m.conventional + m.no_trip, m.demand);
    assert.equal(m.carried + m.rejected, m.requested);
  }
  for (const s of n.services) for (const l of s.legs) assert(l.passengers <= l.capacity);
  for (const s of n.sections) assert(s.delivered <= s.capacity);
  assert.equal(
    n.journeys.reduce((sum, j) => sum + j.passengers, 0),
    n.carried,
  );
  assert.equal(
    n.services.reduce((sum, s) => sum + s.revenue, 0),
    n.revenue,
  );
}
(async () => {
  const browser = await chromium.launch({ headless: true, args: ['--enable-unsafe-swiftshader'] });
  const watchdog = setTimeout(() => browser.close(), 240000);
  const result = {
    date: new Date().toISOString(),
    browser: browser.version(),
    wasm_sha256: crypto.createHash('sha256').update(fs.readFileSync('web/main.wasm')).digest('hex'),
    checks: [],
    limitations: [
      'Browser phone/touch inputs are emulated; physical devices, ' +
        'native OS interaction/IME and assistive technology remain separate qualification.',
      'Representative departures and aggregate loads are a strategic replay, not a validated dispatch timetable.',
    ],
  };
  try {
    let ctx = await browser.newContext({ viewport: { width: 1280, height: 900 }, reducedMotion: 'reduce' });
    let p = await ctx.newPage();
    p.setDefaultTimeout(20000);
    const errors = [],
      trace = [];
    p.on('pageerror', (e) => {
      errors.push(e.message);
      console.error('Page error:', e.message);
    });
    await ready(p);
    await (await control(p, '#tab-services')).click();
    await (await control(p, '#service-corridor')).selectOption('tokyo-osaka');
    await p.waitForFunction(() => !document.querySelector('#apply-service').disabled);
    assert(await p.locator('#extend-starter').isChecked());
    await control(p, '#service-path');
    assert((await p.locator('#service-path').innerText()).includes('Tokyo → Nagoya → Osaka'));
    await p.locator('#fare').fill('138');
    await p.locator('#fare').blur();
    await p.waitForFunction(() => !document.getElementById('apply-service').disabled);
    assert((await p.locator('#extension-price').innerText()).includes('360,000'));
    const initial = (await snap(p)).frame.checkpoint;
    await (await control(p, '#service-path')).scrollIntoViewIfNeeded();
    await p.screenshot({ path: path.join(out, 'milestone3-through-service-desktop.png') });
    await p.evaluate(() => {
      window.m3Commit = railStorageCommit;
      window.railStorageCommit = () => Promise.reject(Error('injected save failure'));
    });
    await (await control(p, '#apply-service')).click();
    await p.waitForFunction(() => document.querySelector('#status').textContent.includes('Could not save'));
    assert.equal((await snap(p)).frame.checkpoint, initial);
    await p.evaluate(() => (window.railStorageCommit = window.m3Commit));
    await apply(p, trace, {
      kind: 'extend_service',
      corridor_id: 'tokyo-osaka',
      replace_service_id: 'tokyo-nagoya',
      frequency: 6,
      fare: 13800,
      lease_count: 1,
    });
    let f = (await snap(p)).frame;
    assert.equal(f.fleet.length, 3);
    assert(f.fleet.every((t) => t.corridor_id === 'tokyo-osaka'));
    await act(p, 'advance', trace, { kind: 'advance_month' });
    f = (await snap(p)).frame;
    conserved(f.network);
    assert(
      f.network.journeys.some(
        (j) => j.from === 'tokyo' && j.to === 'osaka' && j.services.length === 1 && j.services[0] === 'tokyo-osaka',
      ),
    );
    assert(f.network.journeys.some((j) => j.from === 'nagoya' && j.passengers > 0));
    assert.equal(f.day.length, 12);
    assert.deepEqual(
      f.day[0].calls.map((c) => c.city_id),
      ['tokyo', 'nagoya', 'osaka'],
    );
    assert.equal(f.day[0].calls[1].departure - f.day[0].calls[1].arrival, 4);
    result.checks.push(
      'Through service calls at Nagoya, carries both end-to-end and intermediate markets, ' +
        'and derives twelve daily departures with a real intermediate dwell; ' +
        'explicit extension transfers two starter trains and leases exactly one, ' +
        'with failed-save atomicity',
    );

    await (await control(p, '#tab-company')).click();
    await (await control(p, '#motion')).check();
    await (await control(p, '#tab-map')).click();
    await (await control(p, '#watch-day')).click();
    await (await control(p, '#replay-minute')).fill('50');
    await p.locator('#replay-minute').dispatchEvent('input');
    await (await control(p, '#replay-train')).selectOption('tokyo-osaka/1/0');
    await (await control(p, '#focus-route')).click();
    const replayCheckpoint = (await snap(p)).frame.checkpoint;
    await (await control(p, '#replay-play')).click();
    await p.waitForFunction(() => document.querySelector('#replay-clock').textContent !== '06:50');
    await (await control(p, '#replay-play')).click();
    assert.equal((await snap(p)).frame.checkpoint, replayCheckpoint);
    await (await control(p, '#replay-minute')).fill('50');
    await p.locator('#replay-minute').dispatchEvent('input');
    await p.screenshot({ path: path.join(out, 'milestone3-operating-day-desktop.png') });
    result.checks.push(
      'Representative day supports departure selection, station calls, ' +
        'play/pause and time scrubbing on actual paths; ' +
        'replay controls leave the campaign checkpoint unchanged',
    );

    await (await control(p, '#tab-services')).click();
    await (await control(p, '#service-corridor')).selectOption('tokyo-nagoya');
    await (await control(p, '#frequency')).fill('16');
    await (await control(p, '#fare')).fill('90');
    for (let i = 0; i < 4; i++)
      await act(p, 'lease-train', trace, { kind: 'lease_train', corridor_id: 'tokyo-nagoya' });
    await apply(p, trace, { kind: 'set_service', corridor_id: 'tokyo-nagoya', frequency: 16, fare: 9000 });
    await act(p, 'advance', trace, { kind: 'advance_month' });
    f = (await snap(p)).frame;
    conserved(f.network);
    const east = f.network.sections.find((s) => s.corridor_id === 'tokyo-nagoya');
    assert.equal(east.requested, 22);
    assert.equal(east.delivered, 16);
    assert(f.network.services.some((s) => s.delivered < s.requested));

    await (await control(p, '#tab-services')).click();
    await (await control(p, '#service-corridor')).selectOption('tokyo-osaka');
    await (await control(p, '#frequency')).fill('0');
    await apply(p, trace, { kind: 'set_service', corridor_id: 'tokyo-osaka', frequency: 0, fare: 13800 });
    await (await control(p, '#service-corridor')).selectOption('nagoya-osaka');
    await (await control(p, '#frequency')).fill('8');
    await (await control(p, '#fare')).fill('50');
    for (let i = 0; i < 2; i++)
      await act(p, 'lease-train', trace, { kind: 'lease_train', corridor_id: 'nagoya-osaka' });
    await apply(p, trace, { kind: 'set_service', corridor_id: 'nagoya-osaka', frequency: 8, fare: 5000 });
    await act(p, 'advance', trace, { kind: 'advance_month' });
    f = (await snap(p)).frame;
    conserved(f.network);
    assert(f.network.transfers > 0);
    assert(
      f.network.journeys.some(
        (j) => j.from === 'tokyo' && j.to === 'osaka' && j.transfers[0] === 'nagoya' && j.fare === 14000,
      ),
    );
    await p.locator('#report-pane-markets').click();
    await (await control(p, '#market-from')).selectOption('tokyo');
    await (await control(p, '#market-to')).selectOption('osaka');
    assert(await p.locator('#market-journeys [data-journey="tokyo-nagoya|nagoya-osaka"]').count());
    await language(p, 'zh-Hans');
    await p.waitForFunction(() => document.documentElement.lang === 'zh-Hans');
    await p.setViewportSize({ width: 390, height: 844 });
    await (await control(p, '#market-journeys')).scrollIntoViewIfNeeded();
    await p.screenshot({ path: path.join(out, 'milestone3-transfer-report-phone-zh.png') });
    for (const width of [320, 390, 768, 1280, 1920]) {
      await p.setViewportSize({ width, height: 844 });
      assert(await p.evaluate(() => document.documentElement.scrollWidth <= innerWidth));
    }
    await p.setViewportSize({ width: 1280, height: 900 });
    await language(p, 'en');
    result.checks.push(
      'Overlapping local/through services share the same sixteen return paths; ' +
        'transfers at Nagoya consume directed seats on both services and one company journey; ' +
        'outside road/air/conventional/no-trip shares conserve finite OD/class demand; ' +
        'report and inspector reflow at 320–1920px including Simplified Chinese',
    );

    await (await control(p, '#tab-build')).click();
    await (await control(p, '[data-corridor=tsuruga-osaka]')).click();
    await (
      await control(p, '#alignments button')
    )
      .locator('..')
      .locator('button')
      .filter({ hasText: 'Valley detour' })
      .click();
    f = (await snap(p)).frame;
    const a = f.corridors.find((c) => c.id === 'tsuruga-osaka').alignments.find((a) => a.id === 'valley'),
      gap = Math.max(0, a.cost - a.grant - f.player.cash);
    if (gap) await act(p, 'borrow-shortfall-build', trace, { kind: 'take_loan', amount: gap });
    await act(p, 'build-line', trace, { kind: 'build_line', corridor_id: 'tsuruga-osaka', alignment_id: 'valley' });
    for (let i = 0; i < 4; i++) await act(p, 'advance-year', trace, { kind: 'advance_year' });
    await (await control(p, '#project-service')).click();
    await act(p, 'lease-train', trace, { kind: 'lease_train', corridor_id: 'tsuruga-osaka' });
    await (await control(p, '#fare')).fill('50');
    await apply(p, trace, { kind: 'set_service', corridor_id: 'tsuruga-osaka', frequency: 4, fare: 5000 });
    await act(p, 'advance', trace, { kind: 'advance_month' });
    f = (await snap(p)).frame;
    conserved(f.network);
    assert(f.network.journeys.some((j) => (j.from === 'tsuruga' || j.to === 'tsuruga') && j.services.length > 1));
    for (const r of f.reports)
      assert.equal(
        r.closing_cash,
        r.opening_cash + r.revenue - r.costs + r.grants + r.borrowed - r.repaid - r.construction,
      );
    const checkpoint = f.checkpoint,
      network = f.network;
    await p.reload();
    await p.waitForFunction(() => !!window.railSnapshot, null, { timeout: 60000 });
    assert.equal((await snap(p)).frame.checkpoint, checkpoint);
    assert.deepEqual((await snap(p)).frame.network, network);
    fs.writeFileSync('build/m3-commands.json', JSON.stringify(trace));
    const native = JSON.parse(
      execFileSync(path.resolve('build/headless'), ['-commands', path.resolve('build/m3-commands.json')], {
        encoding: 'utf8',
        maxBuffer: 32 * 1024 * 1024,
        stdio: ['ignore', 'pipe', 'pipe'],
      }),
    );
    assert.equal(native.checkpoint, checkpoint);
    assert.deepEqual(native.network, network);
    result.checkpoint = checkpoint;
    await (await control(p, '#tab-company')).click();
    const download = p.waitForEvent('download');
    await (await control(p, '#export')).click();
    await (await download).saveAs(path.resolve('build/m3-export.json'));
    const restored = JSON.parse(
      execFileSync(path.resolve('build/headless'), ['-save', path.resolve('build/m3-export.json')], {
        encoding: 'utf8',
        maxBuffer: 32 * 1024 * 1024,
        stdio: ['ignore', 'pipe', 'pipe'],
      }),
    );
    assert.equal(restored.checkpoint, checkpoint);
    assert.deepEqual(errors, []);
    await ctx.close();
    result.checks.push(
      'A completed branch feeds the existing network through up to two transfers; ' +
        'every monthly ledger reconciles; reload/export restore the same detailed results; ' +
        'accepted UI commands replay to an identical native-headless/WASM checkpoint and network report',
    );

    ctx = await browser.newContext({ viewport: { width: 390, height: 844 } });
    p = await ctx.newPage();
    await ready(p);
    const legacy = fs.readFileSync('internal/application/testdata/milestone2-v2.json', 'utf8');
    await p.evaluate((save) => railStorageCommit(save), legacy);
    await p.reload();
    await p.waitForFunction(() => !!window.railSnapshot, null, { timeout: 60000 });
    f = (await snap(p)).frame;
    assert.equal(f.months_resolved, 2);
    assert.equal(f.reports[0].revenue, 1150596000);
    await act(p, 'advance', [], { kind: 'advance_month' });
    f = (await snap(p)).frame;
    assert.equal(f.reports[0].revenue, 1150596000);
    assert(f.network.available);
    await p.evaluate(() => navigator.serviceWorker.ready);
    await p.waitForFunction(() => !!navigator.serviceWorker.controller);
    await ctx.setOffline(true);
    await p.reload();
    await p.waitForFunction(() => !!window.railSnapshot, null, { timeout: 60000 });
    assert.equal((await snap(p)).frame.checkpoint, f.checkpoint);
    await act(p, 'advance', [], { kind: 'advance_month' });
    await ctx.close();
    result.checks.push(
      'A real version-2 fixture opens without rewriting historical revenue, ' +
        'commits version-5 network months, then reloads and advances offline',
    );
    result.result = 'passed';
    fs.writeFileSync(path.join(out, 'milestone3-checks.json'), JSON.stringify(result, null, 2) + '\n');
    console.log(JSON.stringify(result, null, 2));
  } finally {
    clearTimeout(watchdog);
    await browser.close();
  }
})().catch((e) => {
  console.error(e);
  process.exitCode = 1;
});
