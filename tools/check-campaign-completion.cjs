// Exercises the production footer renderer without starting the large WASM client.
const assert = require('node:assert/strict'),
  fs = require('node:fs'),
  vm = require('node:vm');
const source = fs.readFileSync('web/app.js', 'utf8');
const renderer = source.slice(
  source.indexOf('function renderCampaignCompletion()'),
  source.indexOf('function renderCities()'),
);
assert(renderer.startsWith('function renderCampaignCompletion()'));
for (const locale of ['en', 'zh-Hans']) {
  const texts = JSON.parse(fs.readFileSync('pkg/ui/' + locale + '.json', 'utf8'));
  const nodes = Object.fromEntries(
    ['campaign-ended', 'campaign-new-game', 'campaign-end-detail', 'advance', 'advance-year'].map((id) => [id, {}]),
  );
  const context = vm.createContext({
    frame: { months_resolved: 359, campaign_months: 360 },
    busy: false,
    composing: false,
    $: (id) => nodes[id],
    label: (key) => texts[key],
  });
  vm.runInContext(renderer, context);
  const render = () => vm.runInContext('renderCampaignCompletion()', context);
  render();
  assert.equal(nodes['campaign-ended'].hidden, true);
  assert.equal(nodes.advance.hidden, false);
  context.frame.months_resolved = 360;
  render();
  assert.equal(nodes['campaign-ended'].hidden, false);
  assert.equal(nodes['campaign-new-game'].hidden, false);
  assert.equal(nodes.advance.hidden, true);
  assert.equal(nodes['advance-year'].hidden, true);
  assert.equal(nodes['campaign-end-detail'].textContent, texts.campaignEndDetail.replace('{years}', '30'));
  context.busy = true;
  render();
  assert.equal(nodes['campaign-new-game'].disabled, true);
  context.busy = false;
  context.composing = true;
  render();
  assert.equal(nodes['campaign-new-game'].disabled, true);
  context.composing = false;
  context.frame = { months_resolved: 360, campaign_months: 360 };
  render();
  assert.equal(nodes['campaign-ended'].hidden, false);
  context.frame.months_resolved = 0;
  render();
  assert.equal(nodes['campaign-ended'].hidden, true);
  assert.equal(nodes['campaign-new-game'].hidden, true);
  assert.equal(nodes.advance.hidden, false);
}
// Compare without whitespace so reformatting app.js does not break the wiring check.
assert(source.replace(/\s+/g, '').includes("$('campaign-new-game').onclick=()=>$('new-game').click()"));
console.log(
  'Campaign completion footer: final-month boundary, reload/reset, ' +
    'both locales and busy/composition states passed (DOM model; ' +
    'no browser layout/WASM qualification).',
);

if (process.argv.includes('--layout'))
  (async () => {
    const { chromium } = require('playwright'),
      { execFileSync } = require('node:child_process');
    const browser = await chromium.launch({ headless: true }),
      results = [];
    let peak = 0,
      overBudget = false;
    const guard = setInterval(() => {
      const rows = execFileSync('ps', ['-axo', 'pid=,ppid=,rss='], { encoding: 'utf8' })
        .trim()
        .split('\n')
        .map((s) => s.trim().split(/\s+/).map(Number));
      const ids = new Set([process.pid]);
      let prior = 0;
      while (prior !== ids.size) {
        prior = ids.size;
        for (const [id, parent] of rows) if (ids.has(parent)) ids.add(id);
      }
      const rss = rows.filter(([id]) => ids.has(id)).reduce((n, row) => n + row[2], 0) / 1024;
      peak = Math.max(peak, rss);
      if (rss > 2048) {
        overBudget = true;
        browser.close();
      }
    }, 500);
    const timeout = setTimeout(() => browser.close(), 60000);
    try {
      const page = await browser.newPage();
      const html = fs.readFileSync('web/index.html', 'utf8'),
        footer = html.slice(html.indexOf('<footer>'), html.indexOf('</footer>') + 9);
      const css = fs
        .readFileSync('web/style.css', 'utf8')
        .replace(
          "url('rail-ui.ttf')",
          "url('data:font/ttf;base64," + fs.readFileSync('assets/fonts/rail-ui.ttf').toString('base64') + "')",
        );
      for (const locale of ['en', 'zh-Hans'])
        for (const width of [320, 390, 800, 1440]) {
          await page.setViewportSize({ width, height: 844 });
          await page.setContent(
            '<style>' + css + '</style><div id="game"><header></header><main></main>' + footer + '</div>',
          );
          await page.evaluate(
            async ({ renderer, texts }) => {
              window.$ = (id) => document.getElementById(id);
              window.frame = { months_resolved: 360, campaign_months: 360 };
              window.busy = false;
              window.composing = false;
              window.label = (key) => texts[key];
              for (const e of document.querySelectorAll('[data-t]')) e.textContent = label(e.dataset.t);
              $('status').textContent = texts.saved;
              (0, eval)(renderer + ';renderCampaignCompletion()');
              await document.fonts.ready;
            },
            { renderer, texts: JSON.parse(fs.readFileSync('pkg/ui/' + locale + '.json', 'utf8')) },
          );
          const layout = await page.evaluate(() => {
            const ids = ['campaign-ended', 'campaign-new-game'],
              box = (id) => {
                const r = $(id).getBoundingClientRect();
                return { x: r.x, y: r.y, right: r.right, bottom: r.bottom, width: r.width, height: r.height };
              };
            return {
              boxes: ids.map(box),
              clipped: ids.some(
                (id) => $(id).scrollWidth > $(id).clientWidth || $(id).scrollHeight > $(id).clientHeight,
              ),
              overflow: document.documentElement.scrollWidth > innerWidth,
            };
          });
          assert.equal(layout.clipped, false, JSON.stringify({ locale, width, layout }));
          assert.equal(layout.overflow, false);
          for (const b of layout.boxes) {
            assert(b.x >= 0 && b.right <= width && b.bottom <= 844, JSON.stringify({ locale, width, layout }));
            assert(b.height <= 44, JSON.stringify({ locale, width, layout }));
          }
          assert(layout.boxes[0].right <= layout.boxes[1].x);
          results.push({ locale, width, ...layout });
        }
      assert.equal(overBudget, false, 'combined browser checker RSS exceeded 2 GiB');
      fs.mkdirSync('build/campaign-completion', { recursive: true });
      fs.writeFileSync(
        'build/campaign-completion/browser-footer.json',
        JSON.stringify(
          {
            passed: true,
            peak_rss_mib: Math.ceil(peak),
            results,
            scope:
              'Isolated production footer HTML/CSS/font and renderer; ' +
              'no full application/WASM startup or save interaction',
          },
          null,
          2,
        ) + '\n',
      );
      console.log(
        'Isolated browser footer layout passed all eight locale/viewport cases. Peak checker RSS: ' +
          Math.ceil(peak) +
          ' MiB.',
      );
    } finally {
      clearInterval(guard);
      clearTimeout(timeout);
      await browser.close();
    }
  })().catch((e) => {
    console.error(e);
    process.exitCode = 1;
  });
