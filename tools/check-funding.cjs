const { control, language } = require('./ui-helpers.cjs');
// Explain selected-route funding using real controls, including a negative cash balance.
const assert = require('node:assert/strict'),
  fs = require('node:fs'),
  path = require('node:path'),
  crypto = require('node:crypto');
const { chromium } = require('playwright');
const url = process.argv[2] || 'http://127.0.0.1:8765/',
  out = path.resolve('docs/validation');
const snap = (p) => p.evaluate(() => railSnapshot());
async function command(p, id) {
  const r = (await snap(p)).frame.revision;
  await (await control(p, '#' + id)).click();
  await p.waitForFunction(async (r) => (await railSnapshot()).frame.revision === r + 1, r);
  await p.waitForFunction(() => document.getElementById('game').getAttribute('aria-busy') === 'false');
}
const amounts = async (p) =>
  (await p.locator('#company-funding-values dd').allTextContents()).map((s) => Number(s.replace(/[^0-9]/g, '')));
(async () => {
  const browser = await chromium.launch({ headless: true, args: ['--enable-unsafe-swiftshader'] });
  const watchdog = setTimeout(() => browser.close(), 180000);
  const result = {
    date: new Date().toISOString(),
    wasm_sha256: crypto.createHash('sha256').update(fs.readFileSync('web/main.wasm')).digest('hex'),
    checks: [],
    limitations: ['Browser controls and phone viewport are emulated; physical devices remain unqualified.'],
  };
  try {
    const ctx = await browser.newContext({
        viewport: { width: 1280, height: 800 },
        colorScheme: 'dark',
        reducedMotion: 'reduce',
      }),
      p = await ctx.newPage(),
      errors = [];
    p.on('pageerror', (e) => errors.push(e.message));
    await p.goto(url);
    await p.waitForFunction(() => !!window.railSnapshot, null, { timeout: 60000 });
    await (await control(p, '#tab-company')).click();
    const before = (await snap(p)).frame.checkpoint;
    await control(p, '#company-funding');
    assert((await (await control(p, '#company-funding-route')).innerText()).includes('Tsuruga – Osaka · Tunnel route'));
    assert.deepEqual(await amounts(p), [1981611000000, 1783449900000, 198161100000, 10000000000, 188161100000]);
    const signs = await p.locator('#company-funding-values dd').allTextContents();
    assert(
      signs[1].startsWith('− ') && signs[2].startsWith('= ') && signs[3].startsWith('− ') && signs[4].startsWith('= '),
    );
    await (await control(p, '#review-funding-plan')).click();
    assert(await p.locator('#view-build').isVisible());
    assert.equal((await snap(p)).state.from_city, 'tsuruga');
    assert.equal((await snap(p)).state.to_city, 'osaka');
    await (
      await control(p, '#alignments button')
    )
      .locator('..')
      .locator('button')
      .filter({ hasText: 'Valley detour' })
      .click();
    await (await control(p, '#tab-company')).click();
    assert((await (await control(p, '#company-funding-route')).innerText()).includes('Valley detour'));
    assert.deepEqual(await amounts(p), [1163340000000, 1047006000000, 116334000000, 10000000000, 106334000000]);
    assert.equal((await snap(p)).frame.checkpoint, before);
    await p.setViewportSize({ width: 1280, height: 1600 });
    await (await control(p, '#company-funding')).screenshot({ path: path.join(out, 'funding-breakdown-desktop.png') });
    await p.setViewportSize({ width: 1280, height: 800 });
    await command(p, 'borrow-shortfall');
    let f = (await snap(p)).frame;
    assert.equal(f.debt, 106334000000);
    assert.equal(f.player.cash, 116334000000);
    assert.equal((await amounts(p)).at(-1), 0);
    assert(await p.locator('#borrow-shortfall').isHidden());
    await (await control(p, '#review-funding-plan')).click();
    await command(p, 'build-line');
    await (await control(p, '#tab-company')).click();
    assert(await p.locator('#company-funding').isHidden());
    assert(await p.locator('#no-funding-plan').isVisible());
    result.checks.push(
      'Company names default route/alignment; ' +
        'signed cost minus grant minus cash reconciles to exact gap; ' +
        'review link and alternative choice update the quote without mutation; ' +
        'one exact-gap loan leaves zero gap; committed project stops offering construction funding',
    );
    await (await control(p, '#new-game')).click();
    await command(p, 'confirm-new-game');
    await (await control(p, '#tab-services')).click();
    await (await control(p, '#frequency')).fill('0');
    await p.waitForFunction(() => !document.getElementById('apply-service').disabled);
    await command(p, 'apply-service');
    // The starter's two idle leases no longer exhaust the starting cash within the campaign, so lease eight more
    // on the suspended service and run until their real lease costs leave the company overdrawn.
    for (let i = 0; i < 8; i++) await command(p, 'lease-train');
    assert.equal((await snap(p)).frame.fleet.length, 10);
    await (await control(p, '#tab-report')).click();
    for (let year = 0; year < 15 && (await snap(p)).frame.player.cash >= 0; year++) await command(p, 'advance-year');
    f = (await snap(p)).frame;
    assert(f.player.cash < 0);
    const deficit = -f.player.cash;
    await (await control(p, '#tab-company')).click();
    let rows = await amounts(p);
    assert.equal(rows.length, 6);
    assert.equal(rows[3], 0);
    assert.equal(rows[4], deficit);
    assert.equal(rows[5], 198161100000 + deficit);
    assert((await (await control(p, '#company-funding-note')).innerText()).includes('negative'));
    await language(p, 'zh-Hans');
    await p.waitForFunction(() => document.documentElement.lang === 'zh-Hans');
    await p.setViewportSize({ width: 320, height: 740 });
    assert(await p.evaluate(() => document.documentElement.scrollWidth <= innerWidth));
    await p.setViewportSize({ width: 320, height: 1400 });
    await (await control(p, '#company-funding')).screenshot({ path: path.join(out, 'funding-breakdown-phone-zh.png') });
    await p.setViewportSize({ width: 320, height: 740 });
    await command(p, 'borrow-shortfall');
    f = (await snap(p)).frame;
    assert.equal(f.player.cash, 198161100000);
    assert.equal(f.debt, 198161100000 + deficit);
    assert.equal((await amounts(p)).at(-1), 0);
    assert.deepEqual(errors, []);
    result.checks.push(
      'Idle leases on an actual suspended service produce a negative balance; ' +
        'breakdown separates zero available cash and existing deficit; ' +
        'exact-gap loan clears that deficit and funds the company contribution; ' +
        'Chinese dark phone layout fits 320px',
    );
    result.result = 'passed';
    fs.writeFileSync(path.join(out, 'funding-checks.json'), JSON.stringify(result, null, 2) + '\n');
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
