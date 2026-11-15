const { city, control, language, prospect } = require('./ui-helpers.cjs');
// Milestone 2: existing-track start and standalone build/operate/save through actual HUD controls.
const assert = require('node:assert/strict'),
  fs = require('node:fs'),
  path = require('node:path'),
  crypto = require('node:crypto');
const { execFileSync } = require('node:child_process');
const { chromium } = require('playwright');
const url = process.argv[2] || 'http://127.0.0.1:8765/',
  out = path.resolve('docs/validation');
const snap = (p) => p.evaluate(() => railSnapshot());
async function ready(p) {
  await p.goto(url);
  await p.waitForFunction(() => !!window.railSnapshot, null, { timeout: 60000 });
}
async function act(p, id, trace, command) {
  const rev = (await snap(p)).frame.revision;
  await (await control(p, '#' + id)).click();
  await p.waitForFunction(async (r) => (await railSnapshot()).frame.revision === r + 1, rev);
  await p.waitForFunction(() => document.getElementById('game').getAttribute('aria-busy') === 'false');
  trace.push(command);
}
async function hint(p, text) {
  await p.waitForFunction(
    (t) =>
      [...document.querySelectorAll('.learning-cue')].some(
        (e) => e.getClientRects().length && e.textContent.includes(t),
      ),
    text,
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
      'Phone/touch/IME are emulated; physical phones and native assistive technology remain unqualified.',
      'This check covers the earlier business flow under the expanded network-v5 rules; ' +
        'network-specific scenarios are in check-milestone3.cjs.',
    ],
  };
  try {
    const ctx = await browser.newContext({
        viewport: { width: 1280, height: 800 },
        colorScheme: 'dark',
        reducedMotion: 'reduce',
      }),
      p = await ctx.newPage(),
      trace = [],
      errors = [];
    p.on('pageerror', (e) => errors.push(e.message));
    await ready(p);
    let f = (await snap(p)).frame;
    assert(f.public_network && f.cities.length >= 250);
    assert(['nagoya', 'osaka', 'tokyo'].every((id) => f.stations.some((s) => s.city_id === id)));
    const startingStations = f.stations.length;
    assert(f.stations.every((s) => s.public && s.open));
    assert.equal(f.train_family.seats, 400);
    // The lease comes from the economy catalog, so the check follows calibration changes.
    const economy = JSON.parse(fs.readFileSync('assets/scenario/economy.json', 'utf8'));
    assert.equal(f.train_family.lease_monthly, economy.lease_monthly);
    await hint(p, 'passenger income and costs');
    await act(p, 'advance', trace, { kind: 'advance_month' });
    f = (await snap(p)).frame;
    assert(f.reports[0].revenue > 0);
    assert.equal(f.reports[0].station_costs, f.reports[0].passengers * f.train_family.station_per_journey);
    const checkpoint = f.checkpoint;
    await p.locator('#view-report .learning-cue:visible').waitFor();
    await p.locator('#view-report').getByRole('button', { name: 'Got it' }).click();
    await p.waitForFunction(
      () => JSON.parse(localStorage.getItem('wwhsrt-business-ui-v2')).help.concepts.monthly_report.dismissed,
    );
    await (await control(p, '#tab-services')).click();
    await (await control(p, '#train-details summary')).click();
    assert(await p.locator('#train-details').evaluate((d) => d.open));
    const leaseDollars = (economy.lease_monthly / 100).toLocaleString('en-US');
    const trainValues = await p.locator('#train-values').innerText();
    assert(trainValues.includes(leaseDollars), trainValues);
    assert((await p.locator('#train-summary').innerText()).includes('Swift 400'));
    assert.equal((await snap(p)).frame.checkpoint, checkpoint);
    await (await control(p, '#tab-company')).click();
    await (await control(p, '#guidance-settings')).click();
    await p.waitForFunction(() => JSON.parse(localStorage.getItem('wwhsrt-business-ui-v2')).help.muted);
    await (await control(p, '#guidance-settings')).click();
    assert.equal((await snap(p)).frame.checkpoint, checkpoint);
    await (await control(p, '#tab-map')).click();
    await (await city(p, 'sapporo')).click();
    assert((await p.locator('#city-details').innerText()).includes('No station yet'));
    await (await city(p, 'tokyo')).click();
    await control(p, '#city-details dl');
    assert((await p.locator('#city-details').innerText()).includes('Public network'));
    result.checks.push(
      'Imported worldwide cities and operational public stations, ' +
        'named 400-seat train family and actual costs; ' +
        'ordinary first-month control earns starter revenue; ' +
        'per-concept dismissal/settings and fleet detail disclosure leave checkpoint unchanged; ' +
        'stationless/public city inspection',
    );

    await (await control(p, '#tab-build')).click();
    await prospect(p, 'hokkaido-extension');
    f = (await snap(p)).frame;
    const c = f.corridors.find((c) => c.id === 'hokkaido-extension'),
      a = c.alignments.find((a) => a.id === 'mountain');
    assert.equal(
      Object.values(a.estimate).reduce((sum, x) => sum + x, 0),
      a.cost,
    );
    await control(p, '#engineering-values');
    assert((await p.locator('#engineering-values').innerText()).includes('23,035,650,000'));
    assert((await p.locator('#station-package').innerText()).includes('station works'));
    await (
      await control(p, '#alignments button')
    )
      .locator('..')
      .locator('button')
      .filter({ hasText: 'Valley detour' })
      .click();
    const valley = c.alignments.find((a) => a.id === 'valley');
    assert(valley.cost < a.cost && valley.months > a.months && valley.tunnel_km < a.tunnel_km);
    assert.equal((await snap(p)).frame.checkpoint, checkpoint);
    await (
      await control(p, '#alignments button')
    )
      .locator('..')
      .locator('button')
      .filter({ hasText: 'Tunnel route' })
      .click();
    await (await control(p, '#build-pane-costs')).click();
    await hint(p, 'Tunnels, viaducts');
    await p.locator('#build-pane-plan').click();
    await (await control(p, '#finance-link')).click();
    assert(await p.locator('#view-company').isVisible());
    const gap = a.cost - a.grant - f.player.cash;
    await act(p, 'borrow-shortfall', trace, { kind: 'take_loan', amount: gap });
    await (await control(p, '#review-funding-plan')).click();
    await act(p, 'build-line', trace, { kind: 'build_line', corridor_id: c.id, alignment_id: a.id });
    f = (await snap(p)).frame;
    // The public network now reaches Shin-Hakodate-Hokuto, so the extension reuses that open public station and
    // builds a private one only where no public line stops (Sapporo).
    const publicCities = new Set(f.corridors.filter((c) => c.existing).flatMap((c) => [c.from_city, c.to_city]));
    const newEnds = [c.from_city, c.to_city].filter((id) => !publicCities.has(id));
    assert(newEnds.length > 0);
    assert.equal(f.stations.length, startingStations + newEnds.length);
    const readyMonth = f.projects[0].ready;
    for (const id of [c.from_city, c.to_city]) {
      const here = f.stations.filter((s) => s.city_id === id);
      assert.equal(here.length, 1);
      const s = here[0];
      if (publicCities.has(id)) assert(s.open && s.public);
      else assert(!s.open && !s.public && s.ready === readyMonth && s.owner === f.player.id);
    }
    await (await control(p, '#focus-route')).click();
    await p.screenshot({ path: path.join(out, 'milestone2-construction-desktop.png') });
    await language(p, 'zh-Hans');
    await p.waitForFunction(() => document.documentElement.lang === 'zh-Hans');
    await p.setViewportSize({ width: 390, height: 844 });
    await (await control(p, '#tab-map')).click();
    await (await city(p, 'sapporo')).click();
    assert((await p.locator('#city-details').innerText()).includes('正在建设'));
    assert(await p.evaluate(() => document.documentElement.scrollWidth <= innerWidth));
    await (await control(p, '#city-details')).scrollIntoViewIfNeeded();
    await p.screenshot({ path: path.join(out, 'milestone2-station-phone-zh.png') });
    await (await control(p, '#tab-build')).click();
    await hint(p, '建设需要');
    await act(p, 'advance-year', trace, { kind: 'advance_year' });
    for (let i = 0; i < 3; i++) await act(p, 'advance-year', trace, { kind: 'advance_year' });
    f = (await snap(p)).frame;
    assert.equal(f.months_resolved, readyMonth);
    for (const id of [c.from_city, c.to_city]) assert(f.stations.find((s) => s.city_id === id).open);
    await hint(p, '所需列车数');
    await (await control(p, '#project-service')).click();
    assert.equal(await p.locator('#service-corridor').inputValue(), c.id);
    await hint(p, '开通线路没有列车');
    await act(p, 'lease-train', trace, { kind: 'lease_train', corridor_id: c.id });
    // The suggested plan now opens at 8 trains a day, which needs two leases; one train runs the 4 the trace records.
    await (await control(p, '#frequency')).fill('4');
    await (await control(p, '#fare')).fill('110');
    await p.waitForFunction(() => !document.getElementById('apply-service').disabled);
    await act(p, 'apply-service', trace, { kind: 'set_service', corridor_id: c.id, frequency: 4, fare: 11000 });
    await act(p, 'advance', trace, { kind: 'advance_month' });
    f = (await snap(p)).frame;
    const run = f.reports.at(-1).services.find((s) => s.corridor_id === c.id);
    assert(run.revenue > 0 && run.passengers > 0);
    assert(f.reports.at(-1).maintenance > 0);
    for (const r of f.reports)
      assert.equal(
        r.closing_cash,
        r.opening_cash + r.revenue - r.costs + r.grants + r.borrowed - r.repaid - r.construction,
      );
    await p.screenshot({ path: path.join(out, 'milestone2-revenue-phone-zh.png') });
    result.checks.push(
      'Substantial barrier and priced terrain alternatives reconcile; ' +
        'one full-gap loan funds a standalone project; ' +
        'the stationless endpoint receives timed station works and the public terminus is reused; ' +
        'ordinary guidance leads through construction, opening, local fleet lease, ' +
        'applied service and positive revenue; every report reconciles on a Chinese phone layout',
    );

    const beforeReload = f.checkpoint,
      stations = f.stations;
    await p.reload();
    await p.waitForFunction(() => !!window.railSnapshot, null, { timeout: 60000 });
    assert.equal((await snap(p)).frame.checkpoint, beforeReload);
    assert.deepEqual((await snap(p)).frame.stations, stations);
    await (await control(p, '#tab-company')).click();
    const download = p.waitForEvent('download');
    await (await control(p, '#export')).click();
    const file = await download;
    await file.saveAs(path.resolve('build/m2-export.json'));
    const restored = JSON.parse(
      execFileSync(path.resolve('build/headless'), ['-save', path.resolve('build/m2-export.json')], {
        encoding: 'utf8',
        maxBuffer: 32 * 1024 * 1024,
        stdio: ['ignore', 'pipe', 'pipe'],
      }),
    );
    assert.equal(restored.checkpoint, beforeReload);
    assert.deepEqual(restored.stations, stations);
    await p.evaluate(() => navigator.serviceWorker.ready);
    await p.waitForFunction(() => !!navigator.serviceWorker.controller);
    await ctx.setOffline(true);
    await p.reload();
    await p.waitForFunction(() => !!window.railSnapshot, null, { timeout: 60000 });
    assert.equal((await snap(p)).frame.checkpoint, beforeReload);
    await (await control(p, '#tab-report')).click();
    await act(p, 'advance', trace, { kind: 'advance_month' });
    fs.writeFileSync('build/m2-commands.json', JSON.stringify(trace));
    const headless = JSON.parse(
      execFileSync(path.resolve('build/headless'), ['-commands', path.resolve('build/m2-commands.json')], {
        encoding: 'utf8',
        maxBuffer: 32 * 1024 * 1024,
        stdio: ['ignore', 'pipe', 'pipe'],
      }),
    );
    assert.equal(headless.checkpoint, (await snap(p)).frame.checkpoint);
    assert.deepEqual(errors, []);
    result.checkpoint = headless.checkpoint;
    result.checks.push(
      'Station/fleet/business state resumes identically; exported browser save opens headlessly; ' +
        'cached offline continuation works; ' +
        'accepted commands replay to an identical native-headless/WASM checkpoint',
    );
    result.result = 'passed';
    fs.writeFileSync(path.join(out, 'milestone2-checks.json'), JSON.stringify(result, null, 2) + '\n');
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
