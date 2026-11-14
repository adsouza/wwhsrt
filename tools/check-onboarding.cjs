const { control, language } = require('./ui-helpers.cjs');
const assert = require('node:assert/strict'),
  fs = require('node:fs'),
  path = require('node:path'),
  crypto = require('node:crypto');
const { chromium } = require('playwright');
const url = process.argv[2] || 'http://127.0.0.1:8765/',
  out = path.resolve('docs/validation');
const snap = (p) => p.evaluate(() => railSnapshot());
const waitHelp = (p, fn, arg) => p.waitForFunction(fn, arg);
async function ready(p) {
  await p.goto(url);
  await p.waitForFunction(() => !!window.railSnapshot, null, { timeout: 60000 });
  await p.waitForSelector('#tab-map');
}
async function state(p) {
  return p.evaluate(() => JSON.parse(localStorage.getItem('wwhsrt-business-ui-v2')));
}
async function settled(p) {
  await p.waitForFunction(() => document.getElementById('game').getAttribute('aria-busy') === 'false');
}
(async () => {
  const browser = await chromium.launch({ headless: true, args: ['--enable-unsafe-swiftshader'] });
  const watchdog = setTimeout(() => browser.close(), 240000);
  const result = {
    date: new Date().toISOString(),
    browser: browser.version(),
    wasm_sha256: crypto.createHash('sha256').update(fs.readFileSync('web/main.wasm')).digest('hex'),
    ui_sha256: crypto
      .createHash('sha256')
      .update(Buffer.concat(['web/app.js', 'web/style.css', 'web/index.html'].map((f) => fs.readFileSync(f))))
      .digest('hex'),
    checks: [],
    limitations: ['Phone and IME inputs are emulated. Native OS input and novice usability remain unqualified.'],
  };
  try {
    const ctx = await browser.newContext({ viewport: { width: 1280, height: 850 }, reducedMotion: 'reduce' }),
      p = await ctx.newPage(),
      errors = [];
    p.on('pageerror', (e) => errors.push(e.message));
    await ready(p);
    const initial = await snap(p),
      save = await p.evaluate(() => railStorageLoad());
    assert.equal(await p.locator('.learning-cue:visible').count(), 1);
    await (await control(p, '#tab-build')).click();
    const terrain = p.locator('#view-build [data-help-anchor=alignments]');
    await terrain.locator('.learning-cue').waitFor();
    await terrain.getByRole('button', { name: 'Got it' }).click();
    await p.locator('#view-build [data-help-anchor=funding] .learning-cue').waitFor();
    assert.equal((await state(p)).help.concepts.terrain.dismissed, true);
    assert.equal((await state(p)).help.muted, false);
    assert.equal(await p.locator('.learning-cue:visible').count(), 1);
    await p.locator('#help-open').click();
    await p.locator('#help-search').fill('alignment');
    await p.locator('#help-results button').filter({ hasText: 'Why this alignment costs more' }).click();
    assert((await p.locator('#help-article').innerText()).includes('engineering breakdown'));
    await p.locator('#help-article button').last().click();
    await terrain.locator('details[open]').waitFor();
    const funding = p.locator('#view-build [data-help-anchor=funding]');
    await funding.locator('summary').click();
    await funding.locator('details[open]').waitFor();
    assert((await funding.locator('dl').innerText()).includes('Additional cash needed'));
    await funding.getByRole('button', { name: 'Field Notes', exact: true }).click();
    await p.locator('#field-notes').waitFor({ state: 'visible' });
    await (await control(p, '#help-search')).fill('tunnel');
    await p.waitForFunction(() => document.querySelectorAll('#help-results button').length < 14);
    assert((await p.locator('#help-results').innerText()).includes('alignment'));
    await (await control(p, '#help-back')).click();
    await p.locator('#view-build').waitFor({ state: 'visible' });
    assert.equal((await state(p)).from_city, 'tsuruga');
    assert.equal((await state(p)).alignment_id, 'mountain');
    assert.equal((await snap(p)).frame.checkpoint, initial.frame.checkpoint);
    assert.equal(await p.evaluate(() => railStorageLoad()), save);
    result.checks.push(
      'Single anchored cue; dismissing terrain exposes funding without global mute; ' +
        'Explain remains available with quote facts; ' +
        'reference search/back preserves draft and exact durable game bytes',
    );
    await p.reload();
    await p.waitForFunction(() => !!window.railSnapshot, null, { timeout: 60000 });
    await p.locator('#view-build [data-help-anchor=funding] .learning-cue').waitFor();
    assert.equal((await state(p)).help.concepts.terrain.dismissed, true);
    await (await control(p, '#tab-company')).click();
    await (await control(p, '#guidance-settings')).click();
    await p.waitForFunction(() => JSON.parse(localStorage.getItem('wwhsrt-business-ui-v2')).help.muted);
    await (await control(p, '#tab-services')).click();
    await p.waitForTimeout(300);
    assert.equal(await p.locator('.learning-cue:visible').count(), 0);
    await (await control(p, '#frequency')).fill('8');
    await (await control(p, '#fare')).fill('123');
    await p.locator('#fare').dispatchEvent('change');
    await (await control(p, '#help-open')).click();
    await (await control(p, '#help-search')).fill('fare');
    await p.locator('#help-results button').filter({ hasText: 'Fare and demand' }).click();
    await p.locator('#help-article h2').waitFor();
    assert((await p.locator('#help-article').innerText()).includes('123.00'));
    await (await control(p, '#help-back')).click();
    assert.equal(await p.locator('#frequency').inputValue(), '8');
    assert.equal(await p.locator('#fare').inputValue(), '123');
    assert.equal((await snap(p)).frame.checkpoint, initial.frame.checkpoint);
    result.checks.push(
      'Reload retains per-concept dismissals; global mute hides only unsolicited cues; ' +
        'on-demand forecast explanations and service fare/frequency drafts survive help navigation',
    );
    // Phone, Chinese search/composition, content width, expanded planner.
    await language(p, 'zh-Hans');
    await p.setViewportSize({ width: 320, height: 740 });
    await (await control(p, '#help-open')).click();
    await p.locator('#help-search').dispatchEvent('compositionstart');
    await (await control(p, '#help-search')).fill('票价');
    await p.locator('#help-search').dispatchEvent('keydown', { key: 'Enter', isComposing: true, keyCode: 229 });
    await p.locator('#help-search').dispatchEvent('compositionend');
    await p.waitForFunction(() => document.getElementById('help-results').textContent.includes('票价和需求'));
    await p.locator('#help-results button').filter({ hasText: '票价和需求' }).click();
    await p.locator('#help-article h2').waitFor();
    assert.equal(await p.evaluate(() => document.documentElement.scrollWidth <= innerWidth), true);
    assert.equal((await snap(p)).frame.revision, initial.frame.revision);
    await p.screenshot({ path: path.join(out, 'onboarding-field-notes-phone-zh.png'), fullPage: true });
    await (await control(p, '#help-back')).click();
    await (await control(p, '#tab-build')).click();
    await (await control(p, '#browse-prospects')).click();
    await p.locator('#expanded-help').waitFor({ state: 'visible' });
    const rect = await p.locator('#map-frame').boundingBox(),
      helpRect = await p.locator('#expanded-help').boundingBox();
    assert(rect.height > 0);
    assert(rect.y + rect.height <= helpRect.y + 1, 'help overlaps map canvas');
    await p.locator('#expanded-help button').last().click();
    await (await control(p, '#help-back')).click();
    assert((await p.locator('body').getAttribute('class')).includes('expanded'));
    assert.equal((await state(p)).to_city, 'osaka');
    await (await control(p, '#expand')).click();
    result.checks.push(
      '320px Simplified Chinese reference/search and composition Enter do not submit business commands; ' +
        'expanded endpoint help reserves space outside map; Back restores expanded draft',
    );
    await p.setViewportSize({ width: 1280, height: 850 });
    await language(p, 'en');
    await (await control(p, '#tab-company')).click();
    await (await control(p, '#guidance-settings')).click();
    await p.waitForFunction(() => !JSON.parse(localStorage.getItem('wwhsrt-business-ui-v2')).help.muted);
    await (await control(p, '#tab-build')).click();
    await p.locator('#view-build [data-help-anchor=funding] .learning-cue').waitFor();
    await (await control(p, '#view-build [data-help-anchor=funding]')).scrollIntoViewIfNeeded();
    await p.screenshot({ path: path.join(out, 'onboarding-funding-desktop.png') });
    assert.equal((await snap(p)).frame.checkpoint, initial.frame.checkpoint);
    // A rejected write must not complete learning or alter durable business state.
    await (await control(p, '#tab-map')).click();
    await p.evaluate(() => {
      window.originalOnboardingCommit = railStorageCommit;
      window.railStorageCommit = () => Promise.reject(Error('test write failure'));
    });
    await (await control(p, '#advance')).click();
    await p.waitForFunction(() => document.getElementById('status').textContent.includes('save'));
    await settled(p);
    assert.equal((await snap(p)).frame.checkpoint, initial.frame.checkpoint);
    assert(!(await state(p)).help.concepts.getting_started?.completed);
    assert.equal(await p.locator('.learning-cue:visible').count(), 0);
    await p.evaluate(() => {
      window.railStorageCommit = window.originalOnboardingCommit;
      delete window.originalOnboardingCommit;
    });
    result.checks.push(
      'Rejected storage write does not complete a learning action; blocking error suppresses the unsolicited cue',
    );
    // Normal accepted actions complete concepts; a monthly report gets its own cue.
    await (await control(p, '#tab-map')).click();
    await (await control(p, '#advance')).click();
    await p.waitForFunction(async () => (await railSnapshot()).frame.months_resolved === 1);
    await settled(p);
    await p.locator('#view-report [data-help-anchor=accounts]:visible .learning-cue').waitFor();
    assert.equal((await state(p)).help.concepts.getting_started.completed, true);
    assert.equal((await state(p)).help.concepts.terrain.dismissed, true);
    await (await control(p, '#view-report [data-help-anchor=accounts]:visible summary')).click();
    await p.locator('#view-report [data-help-anchor=accounts]:visible details[open]').waitFor();
    await p.screenshot({ path: path.join(out, 'onboarding-report-desktop.png') });

    await (await control(p, '#tab-map')).click();
    await (await control(p, '[data-city=tokyo]')).click();
    const replayCheckpoint = (await snap(p)).frame.checkpoint;
    await (await control(p, '#watch-day')).click();
    await p.locator('#view-map [data-help-anchor=replay] .learning-cue').waitFor();
    await p.locator('#view-map [data-help-anchor=replay]').getByRole('button', { name: 'Got it' }).click();
    await p.locator('#view-map [data-help-anchor=city] .learning-cue').waitFor();
    assert.equal((await snap(p)).frame.checkpoint, replayCheckpoint);
    result.checks.push(
      'Choosing operating-day replay teaches replay beside its controls; ' +
        'dismissing it allows city guidance and leaves the settlement unchanged',
    );
    // Offline cold reload from the service-worker bundle.
    await p.waitForFunction(async () => !!(await navigator.serviceWorker.getRegistration())?.active);
    await ctx.setOffline(true);
    await p.reload();
    await p.waitForFunction(() => !!window.railSnapshot, null, { timeout: 60000 });
    await (await control(p, '#help-open')).click();
    await (await control(p, '#help-search')).fill('fleet');
    await p.waitForFunction(() => document.getElementById('help-results').textContent.includes('Trains'));
    assert.equal((await state(p)).help.concepts.terrain.dismissed, true);
    result.checks.push(
      'Accepted month advances learning without reviving dismissed concepts; ' +
        'report explains settlement; ' +
        'Field Notes and persisted progression work after offline cold reload',
    );
    assert.deepEqual(errors, []);
    await ctx.close();
    // Legacy global dismissal migrates into a separate mute setting.
    const legacy = await browser.newContext(),
      q = await legacy.newPage();
    await q.addInitScript(() =>
      localStorage.setItem('wwhsrt-business-ui-v2', JSON.stringify({ hint_dismissed: true })),
    );
    await ready(q);
    assert.equal((await state(q)).help.muted, true);
    await (await control(q, '#help-open')).click();
    await q.locator('#field-notes').waitFor({ state: 'visible' });
    assert.equal(await q.locator('#help-results button').count(), 14);
    await legacy.close();
    result.checks.push('Legacy global dismissal imports as mute while all fourteen reference topics remain available');
    result.passed = true;
    fs.writeFileSync(path.join(out, 'onboarding-checks.json'), JSON.stringify(result, null, 2) + '\n');
    console.log(JSON.stringify(result, null, 2));
  } finally {
    clearTimeout(watchdog);
    await browser.close();
  }
})().catch((e) => {
  console.error(e);
  process.exitCode = 1;
});
