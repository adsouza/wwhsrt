const { control, language } = require('./ui-helpers.cjs');
// Focused restart acceptance through real DOM controls and the shared Go/WASM port.
const assert = require('node:assert/strict');
const fs = require('node:fs'),
  path = require('node:path'),
  crypto = require('node:crypto');
const { execFileSync } = require('node:child_process');
const { chromium } = require('playwright');
const url = process.argv[2] || 'http://127.0.0.1:8765/';
const out = path.resolve('docs/validation');
const snap = (p) => p.evaluate(() => railSnapshot());
const idle = (p) => p.waitForFunction(() => document.getElementById('game').getAttribute('aria-busy') === 'false');
async function ready(p) {
  await p.goto(url);
  await p.waitForFunction(() => !!window.railSnapshot, null, { timeout: 60000 });
}
async function accept(p, id, trace, c) {
  const revision = (await snap(p)).frame.revision;
  await (await control(p, '#' + id)).click();
  await p.waitForFunction(async (r) => (await railSnapshot()).frame.revision === r + 1, revision);
  await idle(p);
  trace.push(c);
}
function fresh(f) {
  assert.equal(f.months_resolved, 0);
  assert.deepEqual(f.month, { year: 2029, month: 1 });
  assert.equal(f.player.name, 'Sunrise Rail');
  assert.equal(f.player.cash, 10000000000);
  assert.equal(f.debt, 0);
  assert.deepEqual(f.projects, []);
  assert.deepEqual(f.reports, []);
  assert.deepEqual(f.idle_lines, []);
  assert.deepEqual(f.pending, { construction: 0, grants: 0, borrowed: 0, repaid: 0 });
  assert.deepEqual(f.services, [{ corridor_id: 'tokyo-nagoya', frequency: 6, fare: 9000 }]);
  assert.equal(f.fleet.length, 2);
}
(async () => {
  const browser = await chromium.launch({ headless: true, args: ['--enable-unsafe-swiftshader'] });
  const watchdog = setTimeout(() => browser.close(), 180000);
  const result = {
    date: new Date().toISOString(),
    browser: browser.version(),
    wasm_sha256: crypto.createHash('sha256').update(fs.readFileSync('web/main.wasm')).digest('hex'),
    checks: [],
    limitations: ['Synthetic browser inputs; physical phones and native OS input are not qualified by this check.'],
  };
  try {
    const ctx = await browser.newContext({ viewport: { width: 1280, height: 800 }, reducedMotion: 'reduce' });
    const p = await ctx.newPage(),
      errors = [],
      trace = [];
    p.on('pageerror', (e) => errors.push(e.message));
    await ready(p);
    fresh((await snap(p)).frame);
    await accept(p, 'advance', trace, { kind: 'advance_month' });
    await (await control(p, '#tab-company')).click();
    await (await control(p, '#company-name')).fill('Previous Railway');
    await accept(p, 'rename', trace, { kind: 'rename_company', name: 'Previous Railway' });
    await (await control(p, '#loan-amount')).fill('2,000,000,000');
    await accept(p, 'borrow', trace, { kind: 'take_loan', amount: 200000000000 });
    await (await control(p, '#tab-build')).click();
    await (await control(p, '[data-corridor=tsuruga-osaka]')).click();
    await (
      await control(p, '#alignments button')
    )
      .locator('..')
      .locator('button')
      .filter({ hasText: 'Valley detour' })
      .click();
    await accept(p, 'build-line', trace, { kind: 'build_line', corridor_id: 'tsuruga-osaka', alignment_id: 'valley' });
    await (await control(p, '#tab-company')).click();
    await (await control(p, '#company-name')).fill('Unapplied draft');
    const before = await snap(p),
      durable = await p.evaluate(() => railStorageLoad());
    await (await control(p, '#new-game')).click();
    assert.equal(await p.locator('#cancel-new-game').evaluate((e) => e === document.activeElement), true);
    assert((await p.locator('#new-game-company').innerText()).includes('Previous Railway'));
    await p.screenshot({ path: path.join(out, 'new-game-desktop.png') });
    await p.keyboard.press('Escape');
    assert.equal(await p.locator('#new-game-dialog').isVisible(), false);
    assert.equal((await snap(p)).frame.checkpoint, before.frame.checkpoint);
    assert.equal(await p.locator('#company-name').inputValue(), 'Unapplied draft');
    await (await control(p, '#new-game')).click();
    await (await control(p, '#cancel-new-game')).click();
    assert.equal(await p.evaluate(() => railStorageLoad()), durable);
    result.checks.push(
      'Company entry opens a labeled confirmation with current company and date; ' +
        'Cancel and Escape preserve accepted state, drafts and save bytes; ' +
        'Cancel receives initial focus',
    );

    await language(p, 'zh-Hans');
    await p.waitForFunction(() => document.documentElement.lang === 'zh-Hans');
    await (await control(p, '#motion')).uncheck();
    await (await control(p, '#terrain-toggle')).click();
    await p.setViewportSize({ width: 320, height: 480 });
    await (await control(p, '#new-game')).click();
    assert.equal(await p.locator('#new-game-title').innerText(), '开始新游戏？');
    assert(await p.evaluate(() => document.documentElement.scrollWidth <= innerWidth));
    await (await control(p, '#confirm-new-game')).scrollIntoViewIfNeeded();
    const box = await p.locator('#confirm-new-game').boundingBox();
    assert(box.x >= 0 && box.x + box.width <= 320 && box.y >= 0 && box.y + box.height <= 480);
    await p.screenshot({ path: path.join(out, 'new-game-phone-zh.png') });
    await p.evaluate(() => {
      window.originalCommit = railStorageCommit;
      window.railStorageCommit = () =>
        new Promise((_, reject) => setTimeout(() => reject(Error('injected write failure')), 300));
    });
    await (await control(p, '#confirm-new-game')).click();
    await p.keyboard.press('Escape');
    await idle(p);
    assert(await p.locator('#new-game-dialog').isVisible());
    assert(await p.locator('#new-game-error').isVisible());
    assert((await p.locator('#new-game-error').innerText()).includes('之前的游戏已保留'));
    assert.equal((await snap(p)).frame.checkpoint, before.frame.checkpoint);
    assert.equal(await p.evaluate(() => railStorageLoad()), durable);
    assert.equal(await p.locator('#company-name').inputValue(), 'Unapplied draft');
    await p.evaluate(() => (window.railStorageCommit = window.originalCommit));
    await accept(p, 'confirm-new-game', trace, { kind: 'new_game' });
    let s = await snap(p);
    fresh(s.frame);
    assert.equal(await p.locator('#new-game-dialog').isVisible(), false);
    assert.equal(s.state.view, 'map');
    assert.equal(s.state.draft_name, 'Sunrise Rail');
    assert.equal(s.state.hint_dismissed, false);
    assert.equal(s.state.expanded, false);
    assert.equal(s.state.locale, 'zh-Hans');
    assert.equal(s.state.motion, false);
    assert.equal(s.state.hide_terrain, true);
    assert.equal(s.state.service_corridor, 'tokyo-nagoya');
    assert.equal(s.state.frequency, 6);
    assert.equal(await p.locator('#opportunities [data-corridor=tsuruga-osaka]').count(), 1);
    result.checks.push(
      '320x480 Simplified Chinese dialog reflows and actions remain reachable; ' +
        'in-flight restart cannot be dismissed; ' +
        'failed save preserves current company and durable bytes, displays the error, ' +
        'and can be retried; ' +
        'successful restart clears all business state/drafts and restores onboarding while retaining language, ' +
        'reduced motion and terrain preferences',
    );
    const checkpoint = s.frame.checkpoint;
    await p.reload();
    await p.waitForFunction(() => !!window.railSnapshot, null, { timeout: 60000 });
    assert.equal((await snap(p)).frame.checkpoint, checkpoint);
    assert.equal(await p.locator('#company-name').inputValue(), 'Sunrise Rail');
    await accept(p, 'advance', trace, { kind: 'advance_month' });
    s = await snap(p);
    assert.equal(s.frame.reports[0].revenue, 714132000);
    await p.evaluate(() => navigator.serviceWorker.ready);
    await p.waitForFunction(() => !!navigator.serviceWorker.controller);
    await ctx.setOffline(true);
    await p.reload();
    await p.waitForFunction(() => !!window.railSnapshot, null, { timeout: 60000 });
    await (await control(p, '#tab-company')).click();
    await (await control(p, '#new-game')).click();
    await accept(p, 'confirm-new-game', trace, { kind: 'new_game' });
    fresh((await snap(p)).frame);
    await accept(p, 'advance', trace, { kind: 'advance_month' });
    s = await snap(p);
    fs.mkdirSync('build', { recursive: true });
    const log = path.resolve('build/new-game-commands.json');
    fs.writeFileSync(log, JSON.stringify(trace));
    const headless = JSON.parse(
      execFileSync(path.resolve('build/headless'), ['-commands', log], {
        encoding: 'utf8',
        maxBuffer: 32 * 1024 * 1024,
        stdio: ['ignore', 'pipe', 'pipe'],
      }),
    );
    assert.equal(headless.checkpoint, s.frame.checkpoint);
    assert.deepEqual(errors, []);
    result.checkpoint = s.frame.checkpoint;
    result.checks.push(
      'Restarted save survives reload and earns starter-service revenue; ' +
        'cached app can restart and continue offline; ' +
        'identical restart command sequence matches native headless checkpoint',
    );
    result.result = 'passed';
    fs.writeFileSync(path.join(out, 'new-game-checks.json'), JSON.stringify(result, null, 2) + '\n');
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
