// Isolated browser saves only. Exercise the visible planner, then replay the
// accepted commands in the graphics-free native client against the same assets.
const { control, language } = require('./ui-helpers.cjs');
const assert = require('node:assert/strict'),
  fs = require('node:fs'),
  path = require('node:path'),
  crypto = require('node:crypto');
const { execFileSync } = require('node:child_process'),
  { chromium } = require('playwright');
const url = process.argv[2] || 'http://127.0.0.1:8765/',
  out = path.resolve('docs/validation');
const snap = (p) => p.evaluate(() => railSnapshot());
async function ready(p) {
  await p.goto(url);
  await p.waitForFunction(() => !!window.railSnapshot, null, { timeout: 60000 });
}
async function act(p, id, trace, command) {
  const r = (await snap(p)).frame.revision;
  await (await control(p, '#' + id)).click();
  await p.waitForFunction(async (r) => (await railSnapshot()).frame.revision === r + 1, r, { timeout: 30000 });
  await p.waitForFunction(() => document.getElementById('game').getAttribute('aria-busy') === 'false');
  trace.push(command);
}
async function target(p, value) {
  await p.waitForFunction((v) => document.getElementById('forecast-values').dataset.breakEven === String(v), value);
}
const forecast = (p, q) =>
  p
    .evaluate((q) => call('forecast', q), q)
    .then((r) => {
      assert(!r.error, r.error);
      return r.forecast;
    });
function ledger(f) {
  for (const r of f.reports)
    assert.equal(
      r.closing_cash,
      r.opening_cash + r.revenue - r.costs + r.grants + r.borrowed - r.repaid - r.construction,
    );
}
function expenses(f) {
  assert.equal(f.costs, f.running_costs + f.access_costs + f.handling_costs + f.lease_costs + f.maintenance_costs);
}
const sha = (file) => crypto.createHash('sha256').update(fs.readFileSync(file)).digest('hex');
(async () => {
  const browser = await chromium.launch({ headless: true, args: ['--enable-unsafe-swiftshader'] }),
    timer = setTimeout(() => browser.close(), 240000);
  const result = {
    date: new Date().toISOString(),
    browser: browser.version(),
    wasm_sha256: sha('web/main.wasm'),
    economy_sha256: sha('assets/scenario/economy.json'),
    checks: [],
    limitations: [
      'Isolated browser and emulated phone; native OS interaction and physical devices were not exercised.',
      'Gameplay calibration, not empirical ridership certification; topology gaps remain separately documented.',
    ],
  };
  try {
    let ctx = await browser.newContext({ viewport: { width: 1280, height: 900 }, reducedMotion: 'reduce' }),
      p = await ctx.newPage();
    const errors = [],
      trace = [];
    p.on('pageerror', (e) => errors.push(e.message));
    await ready(p);
    let f = (await snap(p)).frame;
    assert.equal(f.train_family.lease_monthly, 12000000);
    assert.equal(f.train_family.running_per_km, 650);
    assert.equal(f.train_family.public_access_per_km, 250);
    await (await control(p, '#service-corridor')).selectOption('public_lyon__marseille');
    await target(p, 6399);
    assert.equal(await p.locator('#frequency').inputValue(), '1');
    assert.equal(await p.locator('#fare').inputValue(), '44.46');
    assert((await p.locator('#forecast-values').innerText()).includes('Break-even/mo'));
    const q = { corridor_id: 'public_lyon__marseille', frequency: 1, fare: 4446 },
      direct = await forecast(p, q);
    expenses(direct);
    // Same forecast as internal/application/economy_test.go, which pins the rule-level value.
    assert.equal(direct.passengers, 8723);
    assert.equal(direct.break_even_passengers, 6399);
    await p.screenshot({ path: path.join(out, 'economy-plan-desktop.png') });
    await (await control(p, '#economics-values')).scrollIntoViewIfNeeded();
    assert.equal(await p.locator('#economics-values>div').count(), 7);
    assert((await p.locator('#break-even-assumptions').innerText()).includes('mix'));
    await p.screenshot({ path: path.join(out, 'economy-analysis-desktop.png') });
    await (await control(p, '#frequency')).fill('0');
    await p.waitForFunction(() => document.getElementById('forecast-values').dataset.breakEvenAvailable === 'false');
    assert((await p.locator('#forecast-values').innerText()).includes('—'));
    await p.locator('#frequency').fill('1');
    await p.locator('#fare').fill('1');
    await p.locator('#fare').blur();
    await p.waitForFunction(
      () =>
        document.getElementById('forecast-status').textContent.length > 0 &&
        !document.getElementById('forecast-values').dataset.breakEven,
    );
    assert.equal(await p.locator('#economics-values>div').count(), 0);
    await p.locator('#fare').fill('44.46');
    await p.locator('#fare').blur();
    await target(p, 6399);
    await language(p, 'zh-Hans');
    await target(p, 6399);
    assert.equal(await p.locator('#fare').inputValue(), '44.46');
    await language(p, 'en');
    await target(p, 6399);
    result.checks.push(
      'New direct-route defaults are distance/demand based; ' +
        'visible monthly break-even and seven cost rows agree with Go forecast; ' +
        'suspension is undefined and invalid input clears stale results; ' +
        'locale switching preserves the draft',
    );
    await act(p, 'lease-train', trace, { kind: 'lease_train', corridor_id: q.corridor_id });
    await p.waitForFunction(() => !document.getElementById('apply-service').disabled);
    await act(p, 'apply-service', trace, { kind: 'set_service', ...q });
    await act(p, 'advance', trace, { kind: 'advance_month' });
    f = (await snap(p)).frame;
    ledger(f);
    const service = f.network.services.find((s) => s.id === q.corridor_id);
    assert.equal(service.boardings, direct.passengers);
    assert.equal(service.revenue, direct.revenue);
    assert.equal(service.running + service.access + service.handling + 12000000, direct.costs);
    await (await control(p, '#service-corridor')).selectOption('tokyo-osaka');
    await p.waitForFunction(() => !document.getElementById('apply-service').disabled);
    assert(await p.locator('#extend-starter').isChecked());
    assert.equal(await p.locator('#frequency').inputValue(), '6');
    assert.equal(await p.locator('#fare').inputValue(), '81.28');
    const through = await forecast(p, {
      corridor_id: 'tokyo-osaka',
      frequency: 6,
      fare: 8128,
      replace_service_id: 'tokyo-nagoya',
    });
    expenses(through);
    assert(through.break_even_available && through.break_even_uses_observed_mix);
    assert.equal(through.additional_leases, 1);
    await act(p, 'apply-service', trace, {
      kind: 'extend_service',
      corridor_id: 'tokyo-osaka',
      replace_service_id: 'tokyo-nagoya',
      frequency: 6,
      fare: 8128,
      lease_count: 1,
    });
    await act(p, 'advance', trace, { kind: 'advance_month' });
    f = (await snap(p)).frame;
    ledger(f);
    assert.equal(f.reports[1].revenue, through.company_revenue);
    assert(f.network.journeys.some((j) => j.from === 'tokyo' && j.to === 'osaka' && j.passengers > 0));
    fs.writeFileSync('build/economy-commands.json', JSON.stringify(trace));
    const native = JSON.parse(
      execFileSync(path.resolve('build/headless'), ['-commands', path.resolve('build/economy-commands.json')], {
        encoding: 'utf8',
        maxBuffer: 32 * 1024 * 1024,
        stdio: ['ignore', 'pipe', 'pipe'],
      }),
    );
    assert.equal(native.checkpoint, f.checkpoint);
    assert.deepEqual(native.network, f.network);
    result.checkpoint = f.checkpoint;
    result.checks.push(
      'Leasing/applying the suggested Lyon–Marseille plan settles the forecast passenger count, ' +
        'revenue and expenses; through extension retains the starter frequency, ' +
        'uses mixed passenger fares for break-even and reconciles company revenue; ' +
        'native-headless/WASM commands produce identical checkpoints and detailed reports',
    );
    assert.deepEqual(errors, []);
    await ctx.close();
    ctx = await browser.newContext({ viewport: { width: 390, height: 844 }, reducedMotion: 'reduce' });
    p = await ctx.newPage();
    p.on('pageerror', (e) => errors.push(e.message));
    await ready(p);
    const raw = fs.readFileSync('internal/application/testdata/economy-v4.json', 'utf8'),
      legacy = JSON.parse(raw).payload;
    await p.evaluate((save) => railStorageCommit(save), raw);
    await p.reload();
    await p.waitForFunction(() => !!window.railSnapshot, null, { timeout: 60000 });
    f = (await snap(p)).frame;
    assert.equal(f.months_resolved, 2);
    assert.deepEqual(f.reports, legacy.reports);
    assert.deepEqual(f.player, legacy.company);
    assert.equal(f.train_family.lease_monthly, 12000000);
    await language(p, 'zh-Hans');
    await (await control(p, '#service-corridor')).selectOption('public_lyon__marseille');
    await p.waitForFunction(() => document.getElementById('forecast-values').dataset.breakEvenAvailable === 'true');
    assert((await p.locator('#forecast-values').innerText()).includes('保本'));
    assert(await p.evaluate(() => document.documentElement.scrollWidth <= innerWidth));
    await p.screenshot({ path: path.join(out, 'economy-phone-zh.png') });
    await (await control(p, '#economics-values')).scrollIntoViewIfNeeded();
    assert((await p.locator('#break-even-assumptions').innerText()).includes('客'));
    await p.screenshot({ path: path.join(out, 'economy-analysis-phone-zh.png') });
    await (await control(p, '#tab-report')).click();
    await act(p, 'advance', [], { kind: 'advance_month' });
    f = (await snap(p)).frame;
    assert.deepEqual(f.reports.slice(0, 2), legacy.reports);
    assert.equal(f.reports[2].lease_costs, 24000000);
    ledger(f);
    const saved = JSON.parse(await p.evaluate(() => railStorageLoad()));
    // Expanded worlds save as version 10 (internal/application/game.go; pinned by economy_test.go).
    assert.equal(saved.version, 10);
    assert.equal(saved.payload.economy_since, 2);
    assert.equal(saved.payload.economy_sha256, result.economy_sha256);
    await p.evaluate(() => navigator.serviceWorker.ready);
    await p.waitForFunction(() => !!navigator.serviceWorker.controller);
    await ctx.setOffline(true);
    await p.reload();
    await p.waitForFunction(() => !!window.railSnapshot, null, { timeout: 60000 });
    assert.equal((await snap(p)).frame.checkpoint, f.checkpoint);
    await act(p, 'advance', [], { kind: 'advance_month' });
    assert.deepEqual((await snap(p)).frame.reports.slice(0, 2), legacy.reports);
    assert.deepEqual(errors, []);
    await ctx.close();
    result.checks.push(
      'Simplified Chinese phone planner and Analysis expose the target and assumptions without horizontal overflow; ' +
        'a genuine v4 save preserves company and reports, ' +
        'commits a hashed v7 cutover on the next command, and reloads/continues offline',
    );
    result.direct = direct;
    result.through = through;
    result.passed = true;
    fs.writeFileSync(path.join(out, 'economy-checks.json'), JSON.stringify(result, null, 2) + '\n');
    console.log(JSON.stringify({ passed: true, checks: result.checks, checkpoint: result.checkpoint }, null, 2));
  } finally {
    clearTimeout(timer);
    await browser.close();
  }
})().catch((e) => {
  console.error(e);
  process.exitCode = 1;
});
