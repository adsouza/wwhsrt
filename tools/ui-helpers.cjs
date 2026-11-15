// Follow the same tabs and disclosures a player uses. Assertions can still
// inspect hidden state directly; this helper only prepares an intended action.
async function control(p, selector) {
  const target = p.locator(selector).first();
  if (await target.isVisible()) return target;
  const location = await target.evaluate((e) => ({
    view: e.closest('.view')?.id,
    pane: e.closest('[data-task-section]')?.dataset.pane,
    task: e.closest('[data-task-section]')?.dataset.taskSection,
  }));
  if (location.view?.startsWith('view-')) {
    const view = location.view.slice(5);
    if (!(await p.locator('#' + location.view).isVisible())) await p.locator('#tab-' + view).click();
    if (location.pane && !(await p.locator('#' + location.task + '-section-' + location.pane).isVisible()))
      await p.locator('#' + location.task + '-pane-' + location.pane).click();
  }
  // Open the closed disclosures around the target, outermost first, by their own summaries (some have no id).
  const summaries = await target.evaluateHandle((e) => {
    const list = [];
    for (let n = e.parentElement; n; n = n.parentElement) {
      const summary = n.tagName === 'DETAILS' && !n.open && n.querySelector(':scope>summary');
      // A closed disclosure still shows its own summary, so opening it would only undo the caller's click.
      if (summary && !summary.contains(e)) list.unshift(summary);
    }
    return list;
  });
  for (const summary of (await summaries.getProperties()).values()) await summary.asElement()?.click();
  await summaries.dispose();
  await target.waitFor({ state: 'visible' });
  return target;
}
// City buttons exist only while the map's City browser is open, so open it the way a player would.
async function city(p, id) {
  const summary = await control(p, '#city-browser>summary');
  if (!(await p.locator('#city-browser').evaluate((d) => d.open))) await summary.click();
  return control(p, `[data-city=${id}]`);
}
// Only the two prospects nearest the chosen origin get a button. The picker lists every prospect and does the
// same thing as those buttons, so choose there and wait until the Build panel shows the choice.
async function prospect(p, id) {
  await (await control(p, '#prospect-picker')).selectOption(id);
  await p.waitForFunction(() => {
    const picker = document.getElementById('prospect-picker');
    return document.getElementById('selected-prospect').textContent === picker.selectedOptions[0]?.textContent;
  });
}
// A city picked on the map is saved and shown in the city details panel, whether or not the City browser is open.
async function selected(p, id) {
  await p.waitForFunction(
    (id) =>
      JSON.parse(localStorage.getItem('wwhsrt-business-ui-v2') || '{}').selected === id &&
      !!document.querySelector('#city-details strong')?.textContent,
    id,
  );
}
async function language(p, value) {
  if ((await p.locator('html').getAttribute('lang')) !== value) await p.locator('#language-quick').click();
  await p.waitForFunction((v) => document.documentElement.lang === v, value);
}
module.exports = { city, control, language, prospect, selected };
