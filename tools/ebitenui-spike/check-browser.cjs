// Focused checks for the isolated renderer spike; not a game test suite.
const assert = require('node:assert/strict');
const fs = require('node:fs');
const path = require('node:path');
const crypto = require('node:crypto');
const { chromium } = require('playwright');

const url = process.argv[2] || 'http://127.0.0.1:8765/';
const evidence = path.resolve(__dirname, '../../docs/validation');
const snap = page => page.evaluate(() => JSON.parse(wwhsrtSpikeSnapshot()));
const settle = page => page.waitForTimeout(100);
async function ready(page) {
  await page.goto(url);
  await page.waitForFunction(() => window.wwhsrtSpikeSnapshot && JSON.parse(wwhsrtSpikeSnapshot()).frames > 20, null, { timeout: 60000 });
}
async function click(page, id) {
  const r = (await snap(page)).widgets[id];
  await page.mouse.move(r.x + r.w / 2, r.y + r.h / 2);
  await page.mouse.down(); await page.waitForTimeout(70); await page.mouse.up(); await settle(page);
}
async function touch(page, cdp, id) {
  const r = (await snap(page)).widgets[id];
  await cdp.send('Input.dispatchTouchEvent', { type: 'touchStart', touchPoints: [{ x: r.x + r.w / 2, y: r.y + r.h / 2 }] });
  await page.waitForTimeout(120);
  await cdp.send('Input.dispatchTouchEvent', { type: 'touchEnd', touchPoints: [] }); await settle(page);
}
async function replace(page, id, value) {
  await click(page, id);
  await page.keyboard.press('ControlOrMeta+A', { delay: 70 });
  await page.keyboard.type(String(value), { delay: 60 }); await settle(page);
}

(async () => {
  fs.mkdirSync(evidence, { recursive: true });
  const browser = await chromium.launch({ headless: true, args: ['--enable-unsafe-swiftshader'] });
  const result = { date: '2026-11-08', browser: browser.version(), platform: process.platform + '/' + process.arch, scope: 'Actual Go/WASM + ebitenui v0.7.3 + Ebitengine v2.10.5; synthetic browser inputs, not physical mobile devices', checks: [] };
  try {
    const context = await browser.newContext({ viewport: { width: 1280, height: 800 } });
    const page = await context.newPage(); const errors = [];
    page.on('pageerror', e => errors.push(e.message));
    page.on('console', m => { if (/panic:|fatal error:/.test(m.text())) errors.push(m.text()); });
    await ready(page);
    await page.screenshot({ path: path.join(evidence, 'ebitenui-desktop.png') });
    const baseline = await snap(page);
    assert.equal(await page.locator('canvas').count(), 1);
    assert.equal(await page.locator('button').count(), 0, 'controls are not HTML replacements');
    await click(page, 'frequencyPlus'); assert.equal((await snap(page)).frequency, '7');
    assert.notEqual((await snap(page)).operating, baseline.operating);
    await replace(page, 'fare', 75); assert.equal((await snap(page)).fare, '75'); assert.equal((await snap(page)).focused, 'fare');
    await replace(page, 'frequency', 9); assert.equal((await snap(page)).valid, false); assert.equal((await snap(page)).widgets.apply.disabled, true);
    await replace(page, 'frequency', 8); assert.equal((await snap(page)).valid, true);
    await click(page, 'apply'); assert.equal((await snap(page)).applied, true);
    await click(page, 'dismissHint'); assert.equal((await snap(page)).hintDismissed, true); assert.equal((await snap(page)).frequency, '8');
    result.checks.push('real canvas widgets; pointer changes forecast; keyboard editing and platform select-all adapter; invalid fleet disables Apply; apply acknowledgment; hint dismissal preserves draft');

    for (const [w, h] of [[320, 740], [390, 844], [768, 1024], [1280, 800], [1920, 1080]]) {
      await page.setViewportSize({ width: w, height: h });
      await page.waitForFunction(width => JSON.parse(wwhsrtSpikeSnapshot()).width === width, w); await settle(page);
      const s = await snap(page); assert.equal(s.frequency, '8'); assert.equal(s.fare, '75');
      for (const id of ['header', 'footer', 'navigation', 'scroll', 'apply', 'servicesTab', 'reportsTab']) {
        const r = s.widgets[id]; assert.ok(r.x >= 0 && r.y >= 0 && r.x + r.w <= w + 1 && r.y + r.h <= h + 1, `${id} bounds at ${w}`);
      }
      assert.ok(s.widgets.scroll.y + s.widgets.scroll.h < s.widgets.footer.y, 'scroll content has its own space above footer');
    }
    result.checks.push('320, 390, 768, 1280, 1920 logical widths; inputs survive reflow; footer and navigation fit outside clipped scrolling region');
    await page.setViewportSize({ width: 390, height: 844 }); await settle(page);
    await page.reload(); await page.waitForFunction(() => window.wwhsrtSpikeSnapshot && JSON.parse(wwhsrtSpikeSnapshot()).frames > 20); // fresh visual evidence
    await page.screenshot({ path: path.join(evidence, 'ebitenui-phone.png') });
    await page.mouse.move(360, 600); await page.mouse.wheel(0, 500); await settle(page); assert.ok((await snap(page)).scroll > 0);
    await click(page, 'reportsTab'); assert.equal((await snap(page)).view, 'Reports');
    await page.screenshot({ path: path.join(evidence, 'ebitenui-report-phone.png') });
    await page.setViewportSize({ width: 1280, height: 800 }); await settle(page);
    await page.screenshot({ path: path.join(evidence, 'ebitenui-report-desktop.png') });
    result.checks.push('wheel scrolling; actual monthly report widgets render in both compositions');

    const retina = await browser.newContext({ viewport: { width: 390, height: 844 }, deviceScaleFactor: 2 });
    const rp = await retina.newPage(); await ready(rp); assert.equal((await snap(rp)).scale, 2);
    assert.equal((await snap(rp)).widgets.frequencyPlus.h, 44);
    await rp.screenshot({ path: path.join(evidence, 'ebitenui-phone-2x.png') }); await retina.close();
    result.checks.push('2x device scale: native-resolution rasterization with unchanged logical 44px targets');

    const mobile = await browser.newContext({ viewport: { width: 390, height: 844 }, hasTouch: true, isMobile: true, userAgent: 'Mozilla/5.0 (Linux; Android 14) AppleWebKit/537.36 Chrome/130.0.0.0 Mobile Safari/537.36' });
    const mp = await mobile.newPage(); mp.on('pageerror', e => errors.push(e.message)); await ready(mp); const cdp = await mobile.newCDPSession(mp);
    await touch(mp, cdp, 'frequencyPlus'); assert.equal((await snap(mp)).frequency, '7');
    await touch(mp, cdp, 'fare');
    const proxy = await mp.evaluate(() => ({ id: document.activeElement.id, mode: document.getElementById('tempInput').inputMode }));
    assert.equal(proxy.id, 'tempInput'); assert.equal(proxy.mode, 'numeric');
    await mp.keyboard.press('Backspace', { delay: 60 }); await mp.keyboard.press('Backspace', { delay: 60 }); await mp.keyboard.type('70', { delay: 60 }); await settle(mp);
    assert.equal((await snap(mp)).fare, '70', 'hidden mobile input routes into Go text field');
    await cdp.send('Input.dispatchTouchEvent', { type: 'touchStart', touchPoints: [{ x: 368, y: 630 }] });
    for (let y = 600; y >= 270; y -= 30) { await cdp.send('Input.dispatchTouchEvent', { type: 'touchMove', touchPoints: [{ x: 368, y }] }); await mp.waitForTimeout(35); }
    await cdp.send('Input.dispatchTouchEvent', { type: 'touchEnd', touchPoints: [] }); await settle(mp);
    assert.ok((await snap(mp)).scroll > 0, 'application drag policy scrolls real widget container');
    result.mobileInputProxy = proxy;
    result.checks.push('touch-emulated 120ms tap; numeric mobile input proxy focused; typing reaches Go state; application-owned one-finger scroll');
    result.accessibility = await mp.evaluate(() => ({ semanticButtons: document.querySelectorAll('button').length, labeledInputs: document.querySelectorAll('input[aria-label],input[aria-labelledby]').length, canvases: document.querySelectorAll('canvas').length }));
    assert.deepEqual(errors, []);
    result.wasmSha256 = crypto.createHash('sha256').update(fs.readFileSync(path.join(__dirname, 'web/main.wasm'))).digest('hex');
    result.wasmBytes = fs.statSync(path.join(__dirname, 'web/main.wasm')).size;
    result.passed = true;
    fs.writeFileSync(path.join(evidence, 'ebitenui-checks.json'), JSON.stringify(result, null, 2) + '\n');
    console.log(JSON.stringify(result, null, 2));
  } finally { await browser.close(); }
})().catch(error => { console.error(error); process.exitCode = 1; });
