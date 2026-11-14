const assert = require('node:assert/strict'),
  fs = require('node:fs'),
  crypto = require('node:crypto');
const { chromium } = require('playwright'),
  { control, language } = require('./ui-helpers.cjs');
const snap = (p) => p.evaluate(() => railSnapshot()),
  url = process.argv[2] || 'http://127.0.0.1:8765/';
(async () => {
  const b = await chromium.launch({ headless: true, args: ['--enable-unsafe-swiftshader'] }),
    timer = setTimeout(() => b.close(), 120000);
  try {
    const ctx = await b.newContext({ viewport: { width: 1280, height: 850 }, reducedMotion: 'reduce' }),
      p = await ctx.newPage(),
      errors = [];
    p.on('pageerror', (e) => errors.push(e.message));
    await p.goto(url);
    await p.waitForFunction(() => !!window.railSnapshot, null, { timeout: 60000 });
    const initial = await snap(p),
      save = await p.evaluate(() => railStorageLoad()),
      pairs = [
        ['beijing', 'tianjin', 'public_beijing__tianjin'],
        ['chengdu', 'chongqing', 'public_chengdu__chongqing'],
        ['chengdu', 'xi_an', 'public_chengdu__xi_an'],
      ];
    for (const locale of ['en', 'zh-Hans']) {
      await language(p, locale);
      await p.setViewportSize({ width: locale === 'en' ? 1280 : 390, height: 850 });
      for (const [from, to, id] of pairs) {
        await (await control(p, '#tab-build')).click();
        assert.equal(await p.locator('#from-city,#to-city').count(), 0);
        assert.equal(await p.locator('#opportunities [data-corridor="' + id + '"]').count(), 0);
        assert(initial.frame.corridors.find((c) => c.id === id)?.existing);
        await (await control(p, '#tab-services')).click();
        await p.locator('#service-corridor').selectOption(id);
        await p.waitForFunction(async (id) => (await railSnapshot()).state.service_corridor === id, id);
        if (id === 'public_beijing__tianjin')
          await p.screenshot({ path: 'docs/validation/existing-connection-' + locale + '.png' });
        assert.equal((await snap(p)).frame.checkpoint, initial.frame.checkpoint);
        assert.equal(await p.evaluate(() => railStorageLoad()), save);
        assert(await p.evaluate(() => document.documentElement.scrollWidth <= innerWidth));
      }
    }
    assert.deepEqual(errors, []);
    const result = {
      date: new Date().toISOString(),
      passed: true,
      wasm_sha256: crypto.createHash('sha256').update(fs.readFileSync('web/main.wasm')).digest('hex'),
      pairs,
      checks: [
        'All three Chinese pairs are existing public assets, excluded from prospects; ' +
          'arbitrary endpoint controls are absent',
        'Service selector opens each existing route without changing checkpoint, money, ' +
          'projects or exact durable save bytes',
        'English desktop and Simplified Chinese phone layouts remain within the viewport',
      ],
      limitations: [
        'Synthetic browser input; ' +
          'native shortcut is covered by real-widget unit tests and a native build, ' +
          'not OS interaction.',
      ],
    };
    fs.writeFileSync('docs/validation/existing-connection-checks.json', JSON.stringify(result, null, 2) + '\n');
    console.log(JSON.stringify(result, null, 2));
    await ctx.close();
  } finally {
    clearTimeout(timer);
    await b.close();
  }
})().catch((e) => {
  console.error(e);
  process.exitCode = 1;
});
