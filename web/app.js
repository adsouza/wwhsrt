'use strict';
// All business state and commands belong to Go. This file manages semantic
// controls, local presentation preferences, and the browser storage adapter.
const $ = (id) => document.getElementById(id);
let db,
  bridge,
  frame,
  texts = {},
  busy = false,
  composing = false,
  message = '';
// Reuse the existing semantic inputs/handlers; group them around player tasks.
function setupWorkspace() {
  const el = (tag, id, key) => {
    const n = document.createElement(tag);
    if (id) n.id = id;
    if (key) n.dataset.t = key;
    return n;
  };
  const move = (parent, ...nodes) => {
    for (const n of nodes.flat()) {
      const node = typeof n === 'string' ? $(n) : n;
      if (node) parent.append(node);
    }
  };
  const range = (parent, start, end) => {
    const a = $(start);
    let b = $(end);
    if (!a || !b) return;
    while (b.parentElement && b.parentElement !== a.parentElement) b = b.parentElement;
    if (b.parentElement !== a.parentElement) return;
    const nodes = [];
    for (let n = a; n; n = n.nextSibling) {
      nodes.push(n);
      if (n === b) break;
    }
    for (const n of nodes) if (n !== parent) parent.append(n);
  };
  const labelFor = (id) => document.querySelector('label[for="' + id + '"]');
  const tabs = (view, choices) => {
    const nav = el('div', view + '-panes');
    nav.className = 'task-tabs';
    nav.setAttribute('role', 'tablist');
    for (const id of choices) {
      const b = el('button', view + '-pane-' + id, 'pane_' + id);
      b.type = 'button';
      b.dataset.pane = id;
      b.dataset.task = view;
      b.setAttribute('role', 'tab');
      nav.append(b);
    }
    return nav;
  };
  const pane = (view, id) => {
    const n = el('section', view + '-section-' + id);
    n.dataset.taskSection = view;
    n.dataset.pane = id;
    n.setAttribute('role', 'tabpanel');
    n.setAttribute('aria-labelledby', view + '-pane-' + id);
    return n;
  };
  const details = (id, key, ...nodes) => {
    const d = el('details', id),
      s = el('summary', null, key);
    d.append(s);
    move(d, ...nodes);
    return d;
  };
  const anchor = (view, id) => document.querySelector('#view-' + view + ' [data-help-anchor="' + id + '"]');
  const main = document.querySelector('main'),
    dock = el('section', 'workspace-dock');
  dock.setAttribute('aria-label', 'Management');
  move(dock, document.querySelector('#inspector>nav'));
  main.prepend(dock);
  const splitter = el('div', 'inspector-resize');
  splitter.tabIndex = 0;
  splitter.setAttribute('role', 'separator');
  splitter.setAttribute('aria-orientation', 'vertical');
  main.insertBefore(splitter, $('inspector'));
  const regionPicker = el('div', 'region-picker'),
    regionLabel = el('label', null, 'region'),
    regionSelect = el('select', 'region-select');
  regionLabel.htmlFor = 'region-select';
  regionPicker.append(regionLabel, regionSelect);
  const tools = details('dock-map-tools', 'mapControls');
  dock.append(tools);
  move(tools, regionPicker, $('fit'), $('focus-route'), $('zoom-in'), $('zoom-out'), $('terrain-toggle'));
  document.querySelector('.map-tools').classList.add('map-expand-only');
  const phoneTools = details('phone-map-tools', 'mapControls'),
    phoneActions = el('div', 'phone-map-actions');
  phoneTools.append(phoneActions);
  document.querySelector('.map-tools').append(phoneTools);
  phoneTools.addEventListener('click', (e) => {
    if (e.target.closest('button')) phoneTools.open = false;
  });
  const header = document.querySelector('header');
  move(header, $('idle-warning'));
  document.querySelector('.section-head').hidden = true;
  const quick = el('button', 'language-quick');
  quick.type = 'button';
  header.append(quick);
  const companyLabel = el('span', 'company-label');
  document.querySelector('.wordmark span').replaceWith(companyLabel);
  // Services: shared route selector, focused timetable, fleet tools, then analysis.
  const svc = $('view-services'),
    svcTabs = tabs('services', ['plan', 'fleet', 'analysis']);
  svc.querySelector('h1').after(svcTabs);
  const serviceHeader = el('div', 'service-header');
  move(serviceHeader, labelFor('service-corridor'), 'service-corridor');
  svcTabs.after(serviceHeader);
  const mapChoice = el('p', 'map-service-choices', 'mapServiceChoices'),
    allRoutes = el('button', 'all-service-routes', 'allServiceRoutes');
  mapChoice.hidden = true;
  allRoutes.hidden = true;
  serviceHeader.append(mapChoice, allRoutes);
  const issue = el('p', 'service-issue');
  issue.className = 'task-issue';
  issue.setAttribute('role', 'status');
  issue.hidden = true;
  serviceHeader.append(issue);
  const findRoute = details('route-finder', 'routeSearch'),
    routeQuery = el('input', 'route-query');
  routeQuery.type = 'search';
  routeQuery.placeholder = '';
  routeQuery.setAttribute('aria-label', 'Search route names');
  findRoute.append(routeQuery);
  serviceHeader.append(findRoute);
  svc.querySelector(':scope>h2[data-t=forecast]')?.remove();
  const distance = el('p', 'service-distance');
  serviceHeader.append(distance, el('p', 'service-via'));
  const plan = pane('services', 'plan'),
    fleet = pane('services', 'fleet'),
    analysis = pane('services', 'analysis');
  svc.append(plan, fleet, analysis);
  const idle = details('idle-disclosure', 'idleReview', 'idle-details');
  plan.append(idle);
  const path = details('service-path-details', 'calledStops', 'service-path', anchor('services', 'through'));
  plan.append(path);
  move(plan, 'extension-choice', 'extension-price');
  labelFor('fare').dataset.t = 'fareShort';
  const fields = el('div', 'service-fields');
  fields.className = 'two-fields';
  for (const id of ['frequency', 'fare']) {
    const f = el('div');
    move(f, labelFor(id), id);
    fields.append(f);
  }
  plan.append(fields);
  const capacity = $('service-capacity'),
    capTitle = $('capacity-title');
  capTitle.remove();
  const capDetails = el('details', 'capacity-disclosure'),
    capSummary = el('summary', 'capacity-summary');
  capDetails.append(capSummary);
  move(capDetails, 'capacity-values');
  capacity.append(capDetails);
  capacity.setAttribute('aria-labelledby', 'capacity-summary');
  plan.append(capacity);
  move(plan, 'fleet-count', document.querySelector('#view-services .service-actions'));
  plan.querySelector('.service-actions').prepend(el('button', 'optimize-frequency', 'optimizeFrequency'));
  $('lease-train').dataset.t = 'leaseShort';
  $('return-train').dataset.t = 'returnShort';
  const forecast = el('div', 'plan-forecast');
  forecast.className = 'plan-forecast';
  const forecastLabel = el('h2', null, 'serviceSummary');
  forecastLabel.className = 'sr-only';
  forecast.append(forecastLabel);
  plan.append(forecast);
  move(forecast, 'forecast-status', 'forecast-values');
  const apply = el('div', 'service-commit');
  apply.className = 'task-commit';
  $('service-draft-note').classList.add('sr-only');
  move(apply, 'service-draft-note', 'restore-plan', 'apply-service');
  plan.append(apply);
  for (const id of ['service-plan', 'fleet', 'fares', 'capacity', 'idle']) move(plan, anchor('services', id));
  anchor('services', 'fleet').dataset.helpAnchor = 'fleet-plan';
  plan.prepend(anchor('services', 'service-plan'));
  path.after(anchor('services', 'through'));
  fields.after(anchor('services', 'fares'));
  capacity.after(anchor('services', 'capacity'));
  plan.querySelector('.service-actions').after(anchor('services', 'fleet-plan'));
  idle.after(anchor('services', 'idle'));
  move(fleet, 'train-summary', 'train-details');
  const fleetNotes = el('p', null, 'serviceHint');
  fleet.append(fleetNotes);
  fleet.append(el('div', 'fleet-overview'));
  const fleetActions = el('div');
  fleetActions.className = 'service-actions';
  fleetActions.append(el('button', 'fleet-lease', 'leaseTrain'), el('button', 'fleet-return', 'returnTrain'));
  fleet.append(fleetActions);
  const fleetHelp = el('div');
  fleetHelp.dataset.helpAnchor = 'fleet';
  fleet.append(fleetHelp);
  for (const id of ['capacity', 'fares']) {
    const help = el('div');
    help.dataset.helpAnchor = id;
    analysis.append(help);
  }
  analysis.append(
    el('h2', null, 'operatingCostDetails'),
    el('dl', 'economics-values'),
    el('p', 'connector-note'),
    el('p', 'break-even-assumptions', 'breakEvenAssumptions'),
  );
  analysis.append(
    el('h2', null, 'companyChange'),
    el('dl', 'analysis-values'),
    el('h2', null, 'capacityTitle'),
    el('dl', 'analysis-capacity-values'),
  );
  // Keep baseline network/service notes available without filling the plan.
  for (const n of [...svc.querySelectorAll(':scope>p')]) if (n !== $('train-summary')) n.hidden = true;
  // Construction separates commit decisions from engineering and project history.
  const build = $('view-build');
  build.querySelector(':scope>h2[data-t=plans]')?.remove();
  const buildTabs = tabs('build', ['plan', 'costs', 'projects']);
  build.querySelector('h1').after(buildTabs);
  const bp = pane('build', 'plan'),
    bc = pane('build', 'costs'),
    bj = pane('build', 'projects');
  build.append(bp, bc, bj);
  move(
    bp,
    'selected-prospect',
    anchor('build', 'endpoints'),
  );
  move(
    bp,
    'build-status',
    'project-progress',
    'project-service',
    anchor('build', 'project'),
    'build-package',
  );
  const existingService = el('button', 'plan-existing-service', 'planExistingService');
  existingService.hidden = true;
  $('build-status').after(existingService);
  $('build-package').querySelector('h2[data-t=engineeringBreakdown]')?.remove();
  const engineering = el('h2', null, 'engineeringBreakdown');
  bc.append(engineering);
  move(bc, 'engineering-values', 'station-package', 'terrain-explanation');
  for (const p of [...$('build-package').querySelectorAll(':scope>p')])
    if (p.dataset.t === 'estimateNote' || p.dataset.t === 'buildHint') bc.append(p);
  const costHelp = el('div');
  costHelp.dataset.helpAnchor = 'engineering';
  bc.append(costHelp);
  bj.append(el('div', 'project-list'));
  const funding = details('build-funding-details', 'fundingPlan', 'quote', 'funding-gap', 'finance-link');
  $('borrow-shortfall-build').before(funding); // quote details live by the exact-gap action
  const quoteCompact = el('p', 'build-cost-summary');
  $('build-package').prepend(quoteCompact, $('alignments'), $('build-line'));
  bp.append($('browse-prospects'));
  // Accounts retain full ledger without making it the default report.
  const report = $('view-report'),
    rt = tabs('report', ['overview', 'accounts', 'network', 'markets']);
  report.querySelector('h1').after(rt);
  const ro = pane('report', 'overview'),
    ra = pane('report', 'accounts'),
    rn = pane('report', 'network'),
    rm = pane('report', 'markets');
  report.append(ro, ra, rn, rm);
  const overviewHelp = el('div');
  overviewHelp.dataset.helpAnchor = 'accounts';
  ro.append(el('dl', 'report-summary'), overviewHelp);
  move(ra, 'ledger', anchor('report', 'accounts'));
  move(rn, 'network-report', anchor('report', 'capacity'));
  const market = el('div', 'market-tools');
  const nr = $('network-report');
  const heading = nr.querySelector('h2');
  if (heading) {
    let n = heading;
    while (n) {
      const next = n.nextSibling;
      market.append(n);
      n = next;
    }
  }
  rm.append(market);
  nr.querySelector('summary').hidden = true;
  nr.open = true;
  move(ro, 'no-report', 'report-month');
  for (const n of [...report.querySelectorAll(':scope>p')]) n.hidden = true;
  // Company identity and device preferences no longer sit above financing.
  const company = $('view-company'),
    ct = tabs('company', ['finance', 'identity', 'settings']);
  company.querySelector('h1').after(ct);
  const cf = pane('company', 'finance'),
    ci = pane('company', 'identity'),
    cs = pane('company', 'settings');
  company.append(cf, ci, cs);
  range(cf, 'debt', 'repay');
  move(cf, document.querySelector('#view-company [data-t=loanTerms]'));
  move(ci, 'company-form', 'new-game');
  move(
    cs,
    labelFor('appearance'),
    'appearance',
    'guidance-settings',
    'reset-learning',
    document.querySelector('header .language'),
    document.querySelector('#view-company .motion'),
  );
  const financeDetails = details(
    'company-funding-details',
    'fundingPlan',
    'company-funding',
    'no-funding-plan',
    'choose-funding-plan',
  );
  cf.append(financeDetails);
  anchor('company', 'finance')?.remove();
  const financeHelp = el('div');
  financeHelp.dataset.helpAnchor = 'finance';
  cf.append(financeHelp);
  // Move remaining display/accessibility/export controls to Settings.
  range(cs, 'motion', 'export'); // motion's wrapping label is moved above; fallback below
  if ($('motion')?.parentElement !== cs && $('motion')?.closest('label')) cs.append($('motion').closest('label'));
  for (const n of [...company.childNodes])
    if (n.nodeType === 1 && !n.matches('h1,.task-tabs,[data-task-section]')) {
      if (n.matches('h2') && n.dataset.t === 'finance') n.remove();
      else cs.append(n);
    }
  const layout = el('fieldset', 'layout-settings');
  layout.append(el('legend', null, 'workspace'));
  const dl = el('label', null, 'commandDock');
  dl.htmlFor = 'command-dock';
  const ds = el('select', 'command-dock');
  for (const id of ['top', 'left']) {
    const o = el('option', null, 'dock_' + id);
    o.value = id;
    ds.append(o);
  }
  const wl = el('label', null, 'panelWidth');
  wl.htmlFor = 'panel-width';
  const width = el('input', 'panel-width');
  width.type = 'range';
  width.min = '320';
  width.max = '800';
  width.step = '20';
  const reset = el('button', 'layout-reset', 'resetWorkspace');
  reset.type = 'button';
  layout.append(el('p', null, 'workspaceResponsive'), dl, ds, wl, width, el('output', 'panel-width-value'), reset);
  cs.prepend(layout);
  // City browsing is optional once a metro is selected on the map.
  const map = $('view-map');
  const cityBrowser = details('city-browser', 'showCities', labelFor('search'), 'search', 'city-list', 'no-results');
  map.querySelector('h1').after(cityBrowser);
  // Static repeated prose remains in its task's optional detail pane.
  for (const n of [...plan.querySelectorAll(':scope>.hint')]) n.hidden = true;
}

setupWorkspace();
const preferenceKey = 'wwhsrt-business-ui-v2';
let prefs = {
  region: 'east-asia',
  service_query: '',
  service_map_sections: [],
  panel_width: 0,
  command_dock: '',
  service_pane: '',
  company_pane: '',
  build_pane: '',
  report_pane: '',
  disclosures: {},
  review_map: false,
  help: {},
  locale: initialLocale(),
  selected: '',
  view: 'map',
  query: '',
  draft_name: '',
  hint_dismissed: false,
  motion: !matchMedia('(prefers-reduced-motion: reduce)').matches,
  from_city: 'tsuruga',
  to_city: 'osaka',
  choosing: 'from',
  alignment_id: 'mountain',
  service_corridor: 'tokyo-nagoya',
  frequency: 6,
  fare: 9000,
  expanded: false,
  loan_amount: '',
  hide_terrain: false,
  appearance: 'system',
  fleet_details: false,
  replace_service_id: '',
  network_details: false,
  watching: false,
  replay_playing: false,
  replay_minute: 360,
  selected_train: '',
  market_from: 'tokyo',
  market_to: 'osaka',
  market_class: 'business',
};
function initialLocale() {
  return /^(zh$|zh-Hans|zh-CN|zh-SG)/i.test(navigator.language) ? 'zh-Hans' : 'en';
}
try {
  const saved = JSON.parse(localStorage.getItem(preferenceKey));
  if (saved && typeof saved === 'object') {
    for (const k of Object.keys(prefs)) if (typeof saved[k] === typeof prefs[k]) prefs[k] = saved[k];
  }
} catch {}
prefs.locale = prefs.locale === 'zh-Hans' ? 'zh-Hans' : 'en';
prefs.view = ['map', 'build', 'services', 'report', 'company'].includes(prefs.view) ? prefs.view : 'map';
prefs.appearance = ['light', 'dark'].includes(prefs.appearance) ? prefs.appearance : 'system';
const systemAppearance = matchMedia('(prefers-color-scheme: dark)');
function applyAppearance(palette) {
  const dark = prefs.appearance === 'dark' || (prefs.appearance === 'system' && systemAppearance.matches);
  document.documentElement.dataset.theme = dark ? 'dark' : 'light';
  document.querySelector('meta[name=theme-color]').content = dark ? '#0f1d2e' : '#1264d5';
  if (palette)
    for (const [key, c] of Object.entries(palette))
      document.documentElement.style.setProperty('--' + key, `rgb(${c.R}, ${c.G}, ${c.B})`);
}
applyAppearance();
systemAppearance.addEventListener('change', async () => {
  if (prefs.appearance === 'system') {
    applyAppearance();
    await syncUI();
    render();
  }
});
function savePreferences() {
  try {
    localStorage.setItem(preferenceKey, JSON.stringify(prefs));
  } catch {}
}
function money(cents) {
  return new Intl.NumberFormat(prefs.locale, {
    style: 'currency',
    currency: 'USD',
    currencyDisplay: 'code',
    maximumFractionDigits: 2,
  }).format(cents / 100);
}
function month(m) {
  return new Intl.DateTimeFormat(prefs.locale, { year: 'numeric', month: 'long', timeZone: 'UTC' }).format(
    new Date(Date.UTC(m.year, m.month - 1, 1)),
  );
}
function label(key) {
  return texts[key] || key;
}
function startupError(code) {
  const zh = prefs.locale === 'zh-Hans';
  const fallback = {
    locked: zh
      ? '游戏已在另一个标签页中打开。请关闭后重新加载。'
      : 'This game is open in another tab. Close it and reload here.',
    storageUnavailable: zh
      ? '本地存储不可用。请允许此网站使用存储后重试。'
      : 'Local storage is unavailable. Enable storage for this site and reload.',
    invalid_save: zh
      ? '存档无效或版本不兼容。原始存档已保留。'
      : 'The save is invalid or incompatible. Your original data has been preserved.',
  };
  $('loading').textContent =
    texts[code] ||
    fallback[code] ||
    (zh ? '无法打开游戏。原始存档已保留。' : 'Could not open this game. Saved data has been preserved.');
  $('startup').hidden = false;
  $('game').hidden = true;
  $('reload').hidden = false;
}
window.railStartupError = startupError;
$('reload').onclick = () => location.reload();
window.railStorageLoad = () =>
  new Promise((resolve, reject) => {
    const tx = db.transaction('snapshots', 'readonly');
    let value = '';
    const r = tx.objectStore('snapshots').get('campaign');
    r.onsuccess = () => {
      value = r.result || '';
    };
    tx.oncomplete = () => resolve(value);
    tx.onabort = () => reject(tx.error);
    tx.onerror = () => {};
  });
window.railStorageCommit = (value) =>
  new Promise((resolve, reject) => {
    const tx = db.transaction('snapshots', 'readwrite', { durability: 'strict' });
    tx.objectStore('snapshots').put(value, 'campaign');
    tx.oncomplete = () => resolve(true);
    tx.onabort = () => reject(tx.error);
    tx.onerror = () => {};
  });
// The host sends the frame only when its revision changes. Snapshot-shaped
// replies (the ones carrying a camera) reuse the last frame received.
let lastFrame, rankedProspects = [], nextService = '';
function withFrame(s) {
  if (s && typeof s === 'object' && 'camera' in s) {
    if (s.frame) {
      rankedProspects = s.prospects;
      nextService = s.next_service;
      lastFrame = s.frame;
    } else if (lastFrame) s.frame = lastFrame;
  }
  return s;
}
async function call(kind, body = {}) {
  return withFrame(JSON.parse(await bridge(kind, JSON.stringify(body))));
}
async function syncUI() {
  if (bridge) {
    const s = await call('ui', {
      ...prefs,
      viewport_width: innerWidth,
      viewport_height: innerHeight,
      portrait: innerHeight > innerWidth,
      dark: prefs.appearance === 'dark' || (prefs.appearance === 'system' && systemAppearance.matches),
    });
    texts = s.texts;
    // Go drops a saved region that no longer exists, so follow its state.
    prefs.region = s.state.region || 'east-asia';
    selectOptions(
      $('region-select'),
      (s.regions || []).map((r) => [r.id, prefs.locale === 'zh-Hans' ? r.chinese : r.name]),
      prefs.region || 'east-asia',
    );
    acceptHelp(s.state.help);
    applyWorkspace(s.workspace);
    applyAppearance(s.palette);
    return s;
  }
}
function render() {
  if (!frame) return;
  $('game').setAttribute('aria-busy', String(busy));
  document.documentElement.lang = prefs.locale;
  document.title = label('title');
  document.body.dataset.view = prefs.view;
  $('company-label').textContent = frame.player.name;
  $('language-quick').textContent = prefs.locale === 'en' ? 'EN' : '简';
  $('language-quick').title = label('language');
  renderTaskPanes();
  for (const e of document.querySelectorAll('[data-t]')) e.textContent = label(e.dataset.t);
  $('month').textContent =
    innerWidth < 800 ? String(frame.month.year) + '-' + String(frame.month.month).padStart(2, '0') : month(frame.month);
  $('cash').textContent = money(frame.player.cash);
  $('appearance').value = prefs.appearance;
  $('language').value = prefs.locale;
  $('search').value = prefs.query;
  $('company-name').value = prefs.draft_name;
  $('motion').checked = prefs.motion;
  $('terrain-toggle').setAttribute('aria-pressed', String(!prefs.hide_terrain));
  $('guidance-settings').setAttribute('aria-pressed', String(!prefs.help.muted));
  $('guidance-settings').textContent =
    label('guidanceSettings') + ' · ' + label(prefs.help.muted ? 'motionOff' : 'motionOn');
  for (const name of ['map', 'build', 'services', 'report', 'company']) {
    $('view-' + name).hidden = prefs.help.open || prefs.view !== name;
    if (!prefs.help.open && prefs.view === name) $('tab-' + name).setAttribute('aria-current', 'page');
    else $('tab-' + name).removeAttribute('aria-current');
  }
  renderCities();
  renderReport();
  renderBusiness();
  renderIdleLines();
  renderGuidance();
  renderNetwork();
  renderReplay();
  $('status').textContent = label(message || 'saved');
  $('advance').disabled =
    busy || frame.months_resolved >= frame.campaign_months || (prefs.view === 'services' && serviceDirty());
  $('new-game').disabled = busy || composing;
  $('confirm-new-game').disabled = busy;
  $('cancel-new-game').disabled = busy;
  $('new-game-dialog').setAttribute('aria-busy', String(busy));
  $('rename').disabled = busy || composing;
  $('advance-year').disabled =
    busy || frame.months_resolved >= frame.campaign_months || (prefs.view === 'services' && serviceDirty());
  renderCampaignCompletion();
  placeMapTools();
  $('expand').textContent = label(document.body.classList.contains('expanded') ? 'exitFullScreen' : 'fullScreen');
  $('map-panel').setAttribute('aria-label', label('map'));
  $('map-frame').title = label('map') + ' · Equal Earth';
}
function renderCampaignCompletion() {
  const complete = frame.campaign_months > 0 && frame.months_resolved >= frame.campaign_months;
  $('campaign-ended').hidden = !complete;
  $('campaign-new-game').hidden = !complete;
  $('campaign-end-detail').textContent = label('campaignEndDetail').replace(
    '{years}',
    String(Math.floor(frame.campaign_months / 12)),
  );
  $('campaign-new-game').disabled = busy || composing;
  $('advance').hidden = complete;
  $('advance-year').hidden = complete;
}
function renderCities() {
  const list = $('city-list');
  list.replaceChildren();
  const query = prefs.query.toLocaleLowerCase().trim();
  for (const c of $('city-browser').open ? frame.cities : []) {
    const aliases =
      c.name +
      ' ' +
      label(c.id) +
      ' ' +
      ({
        shenyang: '沈阳',
        beijing: '北京',
        shanghai: '上海',
        chengdu: '成都',
        hong_kong: '香港',
        seoul: '首尔',
        tokyo: '东京',
        nagoya: '名古屋',
        osaka: '大阪',
        tsuruga: '敦贺',
        shin_hakodate_hokuto: '新函馆北斗',
        sapporo: '札幌',
      }[c.id] || '');
    if (!aliases.toLocaleLowerCase().includes(query)) continue;
    const b = document.createElement('button');
    b.textContent = label(c.id);
    b.dataset.city = c.id;
    b.setAttribute('aria-pressed', String(prefs.selected === c.id));
    b.onclick = async () => {
      prefs.source_station = '';
      prefs.selected = c.id;
      savePreferences();
      await syncUI();
      renderCities();
      renderGuidance();
    };
    list.append(b);
  }
  $('no-results').hidden = list.childElementCount > 0;
  const details = $('city-details');
  details.replaceChildren();
  if (prefs.source_station) {
    call('snapshot').then((s) => {
      if (!s.source_station || s.state.source_station !== prefs.source_station) return;
      const stop = s.source_station,
        heading = document.createElement('h2'),
        note = document.createElement('p');
      heading.textContent = stop.Name;
      note.textContent = label('sourceStationInfo') + (stop.map_gap ? ' ' + label('sourceStationGap') : '');
      details.replaceChildren(heading, note);
    });
  }
  const c = frame.cities.find((c) => c.id === prefs.selected);
  if (c) {
    const heading = document.createElement('h2');
    heading.textContent = label('selected');
    const name = document.createElement('strong');
    name.textContent = label(c.id);
    const coords = document.createElement('p');
    coords.textContent =
      label('latitude') + ': ' + c.lat.toFixed(4) + '° · ' + label('longitude') + ': ' + c.lon.toFixed(4) + '°';
    details.append(heading, name);
    const station = frame.stations.find((s) => s.city_id === c.id),
      facts = document.createElement('dl'),
      rows = station
        ? [
            [label('station'), label(station.open ? 'stationOpen' : 'stationBuilding')],
            [label('stationOwner'), station.public ? label('stationPublic') : frame.player.name],
          ]
        : [[label('station'), label('stationAbsent')]];
    if (station && !station.open)
      rows.push([
        label('opens'),
        month({ year: 2029 + Math.floor(station.ready / 12), month: 1 + (station.ready % 12) }),
      ]);
    if (c.station_name) rows.push([label('mappedStation'), c.station_name]);
    if (station) rows.push([label('stationHandling'), money(frame.train_family.station_per_journey)]);
    ledgerRows(facts, rows);
    const status = document.createElement('p');
    status.textContent = station ? label(station.open ? 'stationOpen' : 'stationBuilding') : label('stationAbsent');
    const extra = document.createElement('details'),
      summary = document.createElement('summary');
    summary.textContent = label('moreDetails');
    extra.append(summary, coords, facts);
    extra.open = !!prefs.disclosures?.station;
    extra.ontoggle = () => saveDisclosure('station', extra.open);
    details.append(status, extra);
  } else {
    const p = document.createElement('p');
    p.textContent = label('selectCity');
    details.append(p);
  }
}
function renderReport() {
  $('resolved').textContent = String(frame.months_resolved);
  const report = frame.reports.at(-1);
  $('no-report').hidden = !!report;
  $('report-month').textContent = report ? month(report.month) : '';
  const ledger = $('ledger');
  ledger.replaceChildren();
  $('report-summary').replaceChildren();
  if (!report) return;
  ledgerRows($('report-summary'), [
    [label('passengers'), String(report.passengers)],
    [label('revenue'), money(report.revenue)],
    [label('forecastCosts'), money(report.costs)],
    [label('closing'), money(report.closing_cash)],
  ]);
  for (const [key, value] of [
    ['opening', report.opening_cash],
    ['passengers', report.passengers],
    ['revenue', report.revenue],
    ['runningCosts', report.running_costs],
    ['accessCosts', report.access_costs],
    ['stationCosts', report.station_costs],
    ['leaseCosts', report.lease_costs],
    ['maintenance', report.maintenance],
    ['interest', report.interest],
    ['construction', report.construction],
    ['grants', report.grants],
    ['borrowed', report.borrowed],
    ['repaid', report.repaid],
    ['closing', report.closing_cash],
  ]) {
    const row = document.createElement('div'),
      dt = document.createElement('dt'),
      dd = document.createElement('dd');
    dt.textContent = label(key);
    dd.textContent = key === 'passengers' ? new Intl.NumberFormat(prefs.locale).format(value) : money(value);
    row.append(dt, dd);
    ledger.append(row);
  }
}
async function command(kind, extra = {}) {
  if (busy || composing || ($('new-game-dialog').open && kind !== 'new_game')) return;
  busy = true;
  message = 'advancing';
  $('new-game-error').hidden = true;
  render();
  try {
    const s = await call('command', {
      kind,
      revision: frame.revision,
      name: prefs.draft_name,
      corridor_id: prefs.service_corridor,
      frequency: prefs.frequency,
      fare: prefs.fare,
      ...extra,
    });
    frame = s.frame;
    acceptHelp(s.state.help);
    message = s.error || kind;
    if (kind === 'extend_service' && !s.error) prefs.replace_service_id = '';
    if (kind === 'advance_month' && !s.error) prefs.view = 'report';
    if (kind === 'build_line' && !s.error) {
      $('inspector').scrollTop = 0;
      scrollTo(0, 0);
    }
    if (kind === 'new_game' && !s.error) {
      prefs = s.state;
      applyAppearance(s.palette);
      document.body.classList.remove('expanded');
      $('expand').setAttribute('aria-pressed', 'false');
      $('new-game-dialog').close();
      $('inspector').scrollTop = 0;
      scrollTo(0, 0);
    }
    savePreferences();
  } catch {
    message = 'storage_failed';
  } finally {
    busy = false;
    if (prefs.view !== 'services' || (!(frame.idle_lines || []).length && !prefs.service_map_sections?.length))
      prefs.review_map = false;
    savePreferences();
    await syncUI();
    render();
    if (kind === 'new_game' && $('new-game-dialog').open) {
      $('new-game-error').textContent = label(message);
      $('new-game-error').hidden = false;
    }
  }
}
$('new-game').onclick = () => {
  if (busy || composing) return;
  $('new-game-company').textContent = frame.player.name + ' · ' + month(frame.month);
  $('new-game-error').hidden = true;
  $('new-game-dialog').showModal();
  $('cancel-new-game').focus();
};
$('campaign-new-game').onclick = () => $('new-game').click();
$('cancel-new-game').onclick = () => {
  if (!busy) $('new-game-dialog').close();
};
$('new-game-dialog').addEventListener('cancel', (e) => {
  if (busy) e.preventDefault();
});
$('confirm-new-game').onclick = () => command('new_game');
$('advance').onclick = () => command('advance_month');
$('company-form').onsubmit = (e) => {
  e.preventDefault();
  if (!composing) command('rename_company');
};
for (const e of document.querySelectorAll('[data-view]')) e.onclick = () => showView(e.dataset.view);

$('guidance-settings').onclick = () => helpEvent({ kind: prefs.help.muted ? 'unmute' : 'mute' });
$('reset-learning').onclick = () => helpEvent({ kind: 'reset' });
$('search').oninput = (e) => {
  prefs.query = e.target.value;
  savePreferences();
  renderCities();
  renderGuidance();
};
$('company-name').oninput = (e) => {
  prefs.draft_name = e.target.value;
  savePreferences();
};
$('company-name').oncompositionstart = () => {
  composing = true;
  $('rename').disabled = true;
};
$('company-name').oncompositionend = () => {
  composing = false;
  $('rename').disabled = busy;
};
$('company-name').onkeydown = (e) => {
  if (e.key === 'Enter' && (e.isComposing || composing || e.keyCode === 229)) {
    e.preventDefault();
    e.stopPropagation();
  }
};
$('language').onchange = async (e) => {
  prefs.locale = e.target.value;
  savePreferences();
  await syncUI();
  render();
};
$('appearance').onchange = async (e) => {
  prefs.appearance = e.target.value;
  savePreferences();
  applyAppearance();
  await syncUI();
  render();
};
$('motion').onchange = async (e) => {
  prefs.motion = e.target.checked;
  savePreferences();
  await syncUI();
};
for (const [id, kind] of [
  ['fit', 'fit'],
  ['focus-route', 'focus_route'],
  ['zoom-in', 'zoom_in'],
  ['zoom-out', 'zoom_out'],
])
  $(id).onclick = () => call(kind);
function placeMapTools() {
  const expanded = document.body.classList.contains('expanded'),
    phone = innerWidth < 800,
    target = phone ? $('phone-map-actions') : expanded ? document.querySelector('.map-tools') : $('dock-map-tools');
  $('phone-map-tools').hidden = !phone;
  for (const id of ['region-picker', 'fit', 'focus-route', 'zoom-in', 'zoom-out', 'terrain-toggle'])
    if ($(id).parentElement !== target) target.append($(id));
}
$('expand').onclick = () => {
  const expanded = document.body.classList.toggle('expanded');
  prefs.expanded = expanded;
  placeMapTools();
  $('expand').setAttribute('aria-pressed', String(expanded));
  $('expand').textContent = label(expanded ? 'exitFullScreen' : 'fullScreen');
  savePreferences();
  syncUI().then(renderGuidance);
  renderReplay();
};
document.addEventListener('keydown', (e) => {
  if (e.key === 'Escape' && document.body.classList.contains('expanded')) $('expand').click();
});
$('export').onclick = async () => {
  const s = await call('export');
  const url = URL.createObjectURL(new Blob([s.save], { type: 'application/json' }));
  const a = document.createElement('a');
  a.href = url;
  a.download = 'wwhsrt-development-snapshot.json';
  a.click();
  setTimeout(() => URL.revokeObjectURL(url), 1000);
};
window.railMapChanged = (s) => {
  const snapshot = withFrame(JSON.parse(s)),
    sameService = snapshot.state.service_corridor === prefs.service_corridor;
  Object.assign(prefs, {
    view: snapshot.state.view,
    build_pane: snapshot.state.build_pane,
    service_corridor: snapshot.state.service_corridor,
    service_pane: snapshot.state.service_pane,
    service_query: snapshot.state.service_query,
    service_map_sections: snapshot.state.service_map_sections || [],
    frequency: sameService ? prefs.frequency : snapshot.state.frequency,
    fare: sameService ? prefs.fare : snapshot.state.fare,
    replace_service_id: sameService ? prefs.replace_service_id : snapshot.state.replace_service_id,
    review_map: snapshot.state.review_map,
    expanded: snapshot.state.expanded,
    selected_train: snapshot.state.selected_train,
    source_station: snapshot.state.source_station,
    selected: snapshot.state.selected,
    from_city: snapshot.state.from_city,
    to_city: snapshot.state.to_city,
    choosing: snapshot.state.choosing,
    alignment_id: snapshot.state.alignment_id,
  });
  savePreferences();
  document.body.classList.toggle('expanded', !!prefs.expanded);
  $('expand').setAttribute('aria-pressed', String(!!prefs.expanded));
  render();
  syncUI();
};
window.railReady = () => startGame().catch(() => startupError('startupFailed'));
async function startGame() {
  // Give Ebitengine a visible, nonzero viewport before waiting for its first
  // Update; hiding the iframe until after that Update would deadlock startup.
  $('game').hidden = false;
  bridge = $('map-frame').contentWindow.rail;
  const s = await call('snapshot');
  frame = s.frame;
  if (!prefs.draft_name) prefs.draft_name = frame.player.name;
  await syncUI();
  document.body.classList.toggle('expanded', !!prefs.expanded);
  $('expand').setAttribute('aria-pressed', String(!!prefs.expanded));
  render();
  $('startup').hidden = true;
  $('game').hidden = false;
  window.railSnapshot = () => call('snapshot'); // Read-only probe; test actions use real controls.
  if ('serviceWorker' in navigator) navigator.serviceWorker.register('sw.js').catch(() => {});
}
document.addEventListener('visibilitychange', () => {
  savePreferences();
  if (document.hidden) return;
  if (bridge)
    call('snapshot').then((s) => {
      frame = s.frame;
      render();
    });
});
addEventListener('resize', () => {
  if (bridge) syncUI().then(() => render());
});
addEventListener('pagehide', savePreferences);
async function open(attempt = 0) {
  try {
    if (!navigator.locks) throw Error('storageUnavailable');
    await navigator.locks.request('wwhsrt-first-business-v2', { ifAvailable: true }, async (lock) => {
      if (!lock) {
        // A closing/reloading tab can retain its lock briefly after the next
        // document starts. Allow that handoff before reporting a live rival tab.
        if (attempt < 20) {
          setTimeout(() => open(attempt + 1), 100);
          return;
        }
        startupError('locked');
        return;
      }
      db = await new Promise((resolve, reject) => {
        const r = indexedDB.open('wwhsrt-first-business', 1);
        r.onupgradeneeded = () => r.result.createObjectStore('snapshots');
        r.onsuccess = () => resolve(r.result);
        r.onerror = () => reject(r.error);
        r.onblocked = () => reject(Error('blocked'));
      });
      $('map-frame').src = 'map.html';
      await new Promise((resolve) => addEventListener('pagehide', resolve, { once: true }));
      db.close(); // Release ownership promptly during reload/close.
    });
  } catch {
    startupError('storageUnavailable');
  }
}
open();
addEventListener('pageshow', (e) => {
  if (e.persisted) location.reload();
});

let forecastTicket = 0,
  forecastTimer;
function corridorName(c) {
  return texts['projectName_' + c.id] || (label(c.from_city) + ' – ' + label(c.to_city));
}
let idleReviewTicket = 0;
async function reviewIdleLine(id) {
  const lines = frame.idle_lines || [];
  if (!lines.length || (id && !lines.some((l) => l.corridor_id === id))) return;
  if (document.body.classList.contains('expanded')) $('expand').click();
  const ticket = ++idleReviewTicket,
    selected = lines.find((l) => l.corridor_id === id) || lines[0],
    service = (selected.maintenance && frame.corridors.find((c) => c.id === selected.corridor_id)?.opening_service) || selected.corridor_id;
  delete $('map-panel').dataset.idleFocus;
  prefs.service_pane = 'plan';
  prefs.review_map = true;
  saveDisclosure('idle', true);
  loadService(service);
  await showView('services');
  // Give the now-visible phone map its real viewport before fitting the paths.
  await new Promise((resolve) => requestAnimationFrame(() => requestAnimationFrame(resolve)));
  if (
    ticket !== idleReviewTicket ||
    prefs.view !== 'services' ||
    prefs.help.open ||
    prefs.service_corridor !== service
  )
    return;
  await call('focus_idle', { corridor_id: id || '' });
  if (ticket === idleReviewTicket) $('map-panel').dataset.idleFocus = id || 'all';
}
function idleName(line) {
  const fleetOnly = line.maintenance === 0 && line.leases > 0,
    route = frame.routes.find((r) => r.id === line.corridor_id);
  return (
    label(fleetOnly ? 'idleFleetLabel' : 'idleLineLabel') +
    ': ' +
    (fleetOnly && route ? routeName(route) : corridorName(line))
  );
}
function renderIdleLines() {
  const lines = frame.idle_lines || [],
    total = lines.reduce((sum, line) => sum + line.maintenance + line.leases, 0),
    active = lines.length > 0;
  document.body.classList.toggle('has-idle-lines', active);
  document.body.classList.toggle('idle-services-map', active && prefs.view === 'services' && prefs.review_map);
  $('idle-warning').hidden = !active;
  $('idle-footer').hidden = !active;
  $('idle-details').hidden = !active;
  $('idle-disclosure').hidden = !active;
  if (!active) {
    delete $('map-panel').dataset.idleFocus;
    $('idle-line-list').replaceChildren();
    return;
  }
  $('review-idle').textContent = label('idleShort') + ' · ' + lines.length;
  $('review-idle').title = label('reviewIdleLines');
  $('review-idle').setAttribute('aria-label', label('reviewIdleLines') + ' · ' + lines.length);
  $('idle-title').textContent = label('idleLinesTitle') + ' · ' + lines.length;
  $('idle-summary').textContent = label('idleCosts') + ': ' + money(total) + label('monthlyUnit');
  $('idle-footer').textContent =
    label('idleShort') + ' · ' + lines.length + ' · ' + money(total) + label('monthlyUnit');
  $('review-idle').onclick = $('idle-footer').onclick = () => reviewIdleLine();
  $('idle-highlight-label').textContent = label('idleLinesTitle');
  const list = $('idle-line-list');
  list.replaceChildren();
  for (const line of lines) {
    const b = document.createElement('button');
    b.dataset.idleCorridor = line.corridor_id;
    b.setAttribute('aria-pressed', String(line.corridor_id === prefs.service_corridor));
    b.textContent =
      idleName(line) +
      ' · ' +
      money(line.maintenance + line.leases) +
      label('monthlyUnit') +
      ' · ' +
      label(line.needs_trains ? 'idleNeedsTrains' : 'idleNeedsService');
    b.onclick = () => reviewIdleLine(line.corridor_id);
    list.append(b);
  }
}
function isOpen(c) {
  return (c.existing && !c.retired) || frame.projects.some((p) => p.corridor_id === (c.part_of || c.id) && p.ready <= frame.months_resolved);
}
function chosenPair() {
  const pairs = frame.corridors.filter(
    (c) =>
      !c.part_of && !c.retired && !(c.existing && c.through_only) && ((c.from_city === prefs.from_city && c.to_city === prefs.to_city) ||
      (c.to_city === prefs.from_city && c.from_city === prefs.to_city)),
  );
  return pairs.find((c) => !c.existing) || pairs[0];
}
function selectOptions(element, items, value) {
  element.replaceChildren();
  for (const [id, title] of items) {
    const o = document.createElement('option');
    o.value = id;
    o.textContent = title;
    element.append(o);
  }
  element.value = value;
}
function ledgerRows(element, rows) {
  element.replaceChildren();
  for (const [title, value] of rows) {
    const row = document.createElement('div'),
      dt = document.createElement('dt'),
      dd = document.createElement('dd');
    dt.textContent = title;
    dd.textContent = value;
    row.append(dt, dd);
    element.append(row);
  }
}
async function showView(view) {
  $('dock-map-tools').open = false;
  $('phone-map-tools').open = false;
  await helpEvent({ kind: 'leave', value: view }, true);
  $('inspector').scrollTop = 0;
  scrollTo(0, 0);
}
function renderBusiness() {
  $('debt').textContent = label('debt') + ': ' + money(frame.debt);
  if (!prefs.loan_amount) prefs.loan_amount = dollarInput(fundingShortfall() || 100000000);
  $('loan-amount').value = prefs.loan_amount;
  renderLoan();
  $('repay').disabled = busy || frame.debt <= 0 || frame.player.cash < Math.min(50000000000, frame.debt);
  renderBuild();
  renderServices();
}
async function chooseProspect(row) {
  prefs.from_city = row.from;
  prefs.to_city = row.to;
  prefs.alignment_id = row.alignment;
  prefs.build_pane = 'plan';
  await showView('build');
  await call('focus_route');
}

let boardKey = '';
function renderProspectBoard() {
  const rows = rankedProspects;
  const affordable = rows.length && rows[0].cash <= frame.player.cash;
  const finished = frame.months_resolved >= frame.campaign_months;
  const next = $('next-investment');
  next.textContent = label(finished ? 'campaignEndTitle' : affordable ? 'reviewCheapest' : nextService ? 'earnWithService' : 'finance');
  next.disabled = busy || finished;
  next.onclick = async () => {
    if (affordable) return chooseProspect(rows[0]);
    if (!nextService) {
      prefs.company_pane = 'finance';
      return showView('company');
    }
    prefs.service_pane = 'plan';
    prefs.service_query = '';
    if (prefs.service_corridor !== nextService) loadService(nextService);
    await showView('services');
    await call('focus_route');
  };
  const selected = prefs.view === 'build' ? chosenPair()?.id : '';
  const key = JSON.stringify([frame.checkpoint, prefs.locale, selected]);
  if (boardKey === key) return;
  boardKey = key;
  const list = $('opportunities');
  const fragment = new DocumentFragment();
  for (const row of rows) {
    const b = document.createElement('button');
    const name = document.createElement('span');
    name.textContent = corridorName({ id: row.id, from_city: row.from, to_city: row.to });
    const price = document.createElement('span');
    price.className = 'prospect-price';
    price.textContent = money(row.cash);
    if (row === rows[0] || row.cash <= frame.player.cash) {
      const ready = document.createElement('small');
      ready.className = row.cash <= frame.player.cash ? 'affordable' : '';
      ready.textContent = row.cash <= frame.player.cash ? label('withinBudget') :
        label('saveForLine').replace('%s', money(row.cash - frame.player.cash));
      price.append(ready);
    }
    b.dataset.corridor = row.id;
    b.setAttribute('aria-pressed', String(selected === row.id));
    b.onclick = () => chooseProspect(row);
    b.append(name, price);
    fragment.append(b);
  }
  list.replaceChildren(fragment);
  if (!rows.length) list.textContent = label('noProspects');
}

function renderBuild() {
  renderProspectBoard();
  const co = chosenPair(),
    project = co && frame.projects.find((p) => p.corridor_id === co.id);
  $('selected-prospect').textContent = co ? corridorName(co) : label('chooseProspect');
  $('build-cost-summary').textContent = '';
  $('build-package').hidden = true;
  $('alignments').replaceChildren();
  $('quote').replaceChildren();
  $('engineering-values').replaceChildren();
  $('station-package').hidden = true;
  $('build-preview').replaceChildren();
  $('funding-gap').textContent = '';
  $('build-status').textContent = '';
  $('plan-existing-service').hidden = true;
  $('terrain-explanation').textContent = '';
  $('build-line').disabled = true;
  $('finance-link').hidden = true;
  $('borrow-shortfall-build').hidden = true;
  $('project-service').hidden = true;
  $('project-progress').hidden = true;
  if (!co) {
    $('build-status').textContent = label('chooseProspect');
    return;
  }
  if (co.existing) {
    $('build-status').textContent = label('existingPublicConnection');
    $('plan-existing-service').hidden = false;
    return;
  }
  if (project) {
    $('project-progress').hidden = false;
    $('project-progress').max = project.ready - project.started;
    $('project-progress').value = Math.min(frame.months_resolved - project.started, project.ready - project.started);
    $('project-service').hidden = project.ready > frame.months_resolved;
    $('project-service').onclick = () => {
      loadService(co.opening_service || co.id);
      prefs.service_pane = 'plan';
      showView('services');
    };
    $('build-status').textContent =
      project.ready <= frame.months_resolved
        ? label('readyForService')
        : label('underConstruction') +
          ' · ' +
          label('opens') +
          ' ' +
          month({ year: 2029 + Math.floor(project.ready / 12), month: 1 + (project.ready % 12) }) +
          ' · ' +
          (project.ready - frame.months_resolved) +
          ' ' +
          label('monthsLeft');
    return;
  }
  $('build-package').hidden = false;
  $('build-status').textContent = co.opening_service ? label('projectScope_' + co.id) :
    co.construction_kind ? label('constructionScope_' + co.construction_kind) : '';
  if (!co.alignments.some((a) => a.id === prefs.alignment_id)) prefs.alignment_id = co.alignments[0].id;
  for (const a of co.alignments) {
    const b = document.createElement('button');
    b.textContent = label(a.id) + ' · ' + a.km + ' km · ' + a.minutes + ' min · ' + label('duration') + ' ' + a.months;
    b.setAttribute('aria-pressed', String(a.id === prefs.alignment_id));
    b.onclick = async () => {
      prefs.alignment_id = a.id;
      savePreferences();
      await syncUI();
      renderBuild();
      renderGuidance();
    };
    $('alignments').append(b);
  }
  const a = co.alignments.find((a) => a.id === prefs.alignment_id),
    net = a.cost - a.grant;
  $('build-cost-summary').textContent = label('contribution') + ' · ' + money(net) + ' · ' +
    (net <= frame.player.cash ? label('withinBudget') : label('fundingGap') + ' ' + money(net - frame.player.cash));
  const e = a.estimate;
  ledgerRows($('engineering-values'), [
    ...(a.cost_intervals || []).map((p) => [
      label('regionalCostBasis') + ' · ' + p.country,
      money(p.base_per_km) + ' / km',
    ]),
    [label('surfaceWorks') + ' · ' + a.surface_km + ' km', money(e.surface)],
    [label('hillWorks') + ' · ' + a.hill_km + ' km', money(e.hills)],
    [label('viaductWorks') + ' · ' + a.viaduct_km + ' km', money(e.viaducts)],
    [label('tunnelWorks') + ' · ' + a.tunnel_km + ' km', money(e.tunnels)],
    [label('managementAllowance'), money(e.management)],
    [label('contingencyAllowance'), money(e.contingency)],
    [label('totalCost'), money(a.cost)],
  ]);
  $('station-package').hidden = false;
  $('terrain-explanation').textContent = co.construction_note
    ? co.construction_note + ' · ' + label('authoredAlignmentNote')
    : co.opening_service ? label('projectScope_' + co.id) : label(
        (co.id === 'hokkaido-extension' ? 'hokkaido' : 'kansai') + (a.id === 'mountain' ? 'MountainWhy' : 'ValleyWhy'),
      );
  const quoteID = co.id,
    variant = a.id,
    plan = frame.routes.find((r) => r.id === (co.opening_service || co.id));
  call('forecast', {
    corridor_id: co.id,
    alignment_id: a.id,
    frequency: plan?.suggested_frequency || 6,
    fare: plan?.suggested_fare || 9000,
  })
    .then((result) => {
      if (chosenPair()?.id === quoteID && prefs.alignment_id === variant && !result.error) {
        ledgerRows($('build-preview'), [
          [label('potential'), money(result.forecast.contribution)],
          [label('previewPlan'), (plan?.suggested_frequency || 6) + ' · ' + money(plan?.suggested_fare || 9000)],
        ]);
      }
    })
    .catch(() => {});
  ledgerRows($('quote'), fundingRows(selectedFunding()));
  if (net > frame.player.cash) {
    $('funding-gap').textContent =
      (frame.player.cash < 0 ? label('fundingNegativeCash') + ' ' : '') +
      label('fundingFormula') +
      ' ' +
      label('shortfallInterest') +
      ': ' +
      money(Math.ceil((frame.debt + net - frame.player.cash) / 600) - Math.ceil(frame.debt / 600));
    $('finance-link').hidden = false;
    $('borrow-shortfall-build').hidden = false;
    $('borrow-shortfall-build').textContent = label('borrowShortfall') + ' · ' + money(net - frame.player.cash);
    $('borrow-shortfall-build').disabled = busy || net - frame.player.cash > frame.debt_limit - frame.debt;
  }
  $('build-line').disabled = busy || net > frame.player.cash;
}
function renderServices() {
  const t = frame.train_family;
  $('train-summary').textContent = label(t.name_key) + ' · ' + t.seats + ' ' + label('seatUnit');
  $('train-details').open = prefs.fleet_details;
  ledgerRows($('train-values'), [
    [label('trainFamily'), label(t.name_key)],
    [label('seatsPerTrain'), String(t.seats)],
    [label('trainLease'), money(t.lease_monthly)],
    [label('trainRunning'), money(t.running_per_km)],
    [label('trackAccess'), money(t.public_access_per_km)],
    [label('stationHandling'), money(t.station_per_journey)],
    [label('trainDay'), String(t.daily_minutes / 60)],
    [label('trainTurnaround'), String(t.round_trip_turnaround)],
  ]);
  const scoped = !!prefs.service_map_sections?.length;
  let available = frame.routes.filter(
    (r) => r.open && (!scoped || r.sections.some((id) => prefs.service_map_sections.includes(id))),
  );
  if (!available.length && scoped) {
    prefs.service_map_sections = [];
    available = frame.routes.filter((r) => r.open);
  }
  $('map-service-choices').hidden = $('all-service-routes').hidden = !prefs.service_map_sections?.length;
  if (!available.some((c) => c.id === prefs.service_corridor) && available.length) loadService(available[0].id);
  document.querySelector('label[for=service-corridor]').title = label('mapServiceHint');
  $('route-query').setAttribute('aria-label', label('routeSearch'));
  const words = (prefs.service_query || '').toLowerCase().trim().split(/\s+/).filter(Boolean);
  const stationNames = new Map(words.length ? frame.cities.map((c) => [c.id, c.station_name]) : []);
  $('route-query').value = prefs.service_query || '';
  selectOptions(
    $('service-corridor'),
    available
      .filter((r) => {
        if (!words.length || r.id === prefs.service_corridor) return true;
        const searchable = [routeName(r), ...r.stops.map(label), ...(r.station_names || r.stops.map((id) => stationNames.get(id))), ...(r.via || [])].join(' ').toLowerCase();
        return words.every((word) => searchable.includes(word));
      })
      .map((r) => [r.id, routeName(r)]),
    prefs.service_corridor,
  );
  $('frequency').value = prefs.frequency;
  $('fare').value = prefs.fare / 100;
  $('service-draft-note').hidden = !serviceDirty();
  $('service-draft-note').textContent = label('draftChanges');
  $('restore-plan').hidden = !serviceDirty();
  const route = frame.routes.find((r) => r.id === prefs.service_corridor);
  $('service-path-details').hidden = !route || (route.stops.length <= 2 && !route.station_names?.length);
  $('service-distance').hidden = paneFor('services') === 'plan';
  $('service-distance').textContent = route ? route.km + ' km · ' + route.minutes + ' min' : '';
  $('service-via').hidden = !route?.via?.length;
  $('service-via').textContent = route?.via?.length ? label('viaPlaces') + ' ' + route.via.join(' → ') : '';
  $('connector-note').textContent = route?.connector_km && route.infrastructure_kind !== 'hsr'
    ? label('conventionalConnectors') + ': ' + route.connector_km + ' km. ' + label('connectorHint')
    : '';
  $('service-path').textContent = route
    ? label('calledStops') +
      ': ' +
      (route.station_names || route.stops.map(label)).join(' → ') +
      ' · ' +
      route.km +
      ' km · ' +
      route.minutes +
      ' min'
    : '';
  const starter = frame.services.some((s) => s.corridor_id === 'tokyo-nagoya' && s.frequency > 0),
    target = frame.services.some((s) => s.corridor_id === 'tokyo-osaka' && s.frequency > 0);
  $('extension-choice').hidden = prefs.service_corridor !== 'tokyo-osaka' || !starter || target;
  $('extend-starter').checked = !!prefs.replace_service_id;
  $('optimize-frequency').disabled = busy || !route?.open;
  $('lease-train').disabled = busy;
  $('return-train').disabled = busy;
  renderIdleLines();
  queueForecast();
}
function routeName(r) {
  if (!r || r.cities.length < 2) return label('retiredService');
  const stops = r.stops?.length >= 2 ? r.stops : r.cities;
  return (
    (stops.length > 4
      ? label(r.cities[0]) +
        ' → ' +
        label(r.cities.at(-1)) +
        ' · ' +
        label('routeStopCount').replace('%d', r.stops.length)
      : stops.map(label).join(' → ')) +
    (r.stops.length < r.cities.length ? ' · ' + label('express') : '') +
    (r.infrastructure_kind ? ' · ' + label('trackKind_' + r.infrastructure_kind) : '')
  );
}
function loadService(id) {
  if (!frame.routes.find((r) => r.id === id)?.sections.some((id) => prefs.service_map_sections?.includes(id)))
    prefs.service_map_sections = [];
  prefs.service_corridor = id;
  prefs.replace_service_id = '';
  const s = frame.services.find((s) => s.corridor_id === id),
    route = frame.routes.find((r) => r.id === id);
  prefs.frequency = s ? s.frequency : route?.suggested_frequency || 4;
  prefs.fare = s
    ? s.fare
    : route?.suggested_fare || (route?.sections.length > 1 ? Math.min(30000, Math.max(1000, route.km * 25)) : 9000);
  if (!s && id === 'tokyo-osaka') {
    const starter = frame.services.find((s) => s.corridor_id === 'tokyo-nagoya' && s.frequency > 0);
    if (starter) {
      prefs.replace_service_id = starter.corridor_id;
      prefs.frequency = starter.frequency;
    }
  }
  draftStatusChanged();
}
let lastServiceForecast = {},
  forecastKey = '';
function queueForecast() {
  const key = JSON.stringify([
    frame.checkpoint,
    prefs.service_corridor,
    prefs.frequency,
    prefs.fare,
    prefs.replace_service_id,
    prefs.locale,
    busy,
  ]);
  if (key === forecastKey) return;
  forecastKey = key;
  $('fleet-count').textContent =
    label('fleet') + ': ' + frame.fleet.filter((f) => f.corridor_id === prefs.service_corridor).length;
  $('fleet-overview').textContent = $('fleet-count').textContent;
  delete $('fleet-count').dataset.needed;
  const ticket = ++forecastTicket;
  clearTimeout(forecastTimer);
  $('apply-service').disabled = true;
  $('forecast-values').replaceChildren();
  $('economics-values').replaceChildren();
  delete $('forecast-values').dataset.breakEven;
  delete $('forecast-values').dataset.breakEvenAvailable;
  $('analysis-values').replaceChildren();
  $('forecast-status').textContent = '';
  $('service-capacity').hidden = true;
  forecastTimer = setTimeout(async () => {
    const result = await call('forecast', {
      corridor_id: prefs.service_corridor,
      frequency: prefs.frequency,
      fare: prefs.fare,
      replace_service_id: prefs.replace_service_id,
    });
    if (ticket !== forecastTicket) return;
    const f = result.forecast;
    lastServiceForecast = f;
    $('fleet-lease').disabled = busy;
    $('fleet-return').disabled = busy;
    $('service-capacity').hidden = !(result.capacity_details || []).length;
    $('service-capacity').classList.toggle(
      'over-capacity',
      (f.capacity_usage || []).some((c) => c.excess > 0),
    );
    ledgerRows(
      $('capacity-values'),
      (result.capacity_details || []).map((row) => [row.label, row.value]),
    );
    $('forecast-status').textContent = result.error
      ? label(result.error)
      : f.fleet_needed > f.fleet_available && !prefs.replace_service_id
        ? label('fleet_shortage')
        : '';
    if (!result.error) {
      ledgerRows($('economics-values'), [
        [label('passengers'), String(f.passengers)],
        [
          label('breakEvenMonthly'),
          f.break_even_available ? String(f.break_even_passengers) : label('breakEvenUnavailable'),
        ],
        [label('runningCosts'), money(f.running_costs)],
        [label('accessCosts'), money(f.access_costs)],
        [label('stationCosts'), money(f.handling_costs)],
        [label('leaseCosts'), money(f.lease_costs)],
        [label('maintenance'), money(f.maintenance_costs)],
      ]);
      $('forecast-values').dataset.breakEven = String(f.break_even_passengers);
      $('forecast-values').dataset.breakEvenAvailable = String(f.break_even_available);
    }
    if (!result.error) {
      ledgerRows($('analysis-values'), [
        [label('passengers'), new Intl.NumberFormat(prefs.locale).format(f.passengers)],
        [label('needFleet'), String(f.fleet_needed)],
        [label('revenue'), money(f.revenue)],
        [label('forecastCosts'), money(f.costs)],
        [label('operatingContribution'), money(f.contribution)],
        [label('deliveredFrequency'), f.delivered_frequency + ' / ' + prefs.frequency],
        [label('companyChange'), money(f.company_change)],
        [label('feederRevenue'), money(f.feeder_revenue)],
      ]);
    }
    if (!result.error) {
      ledgerRows($('forecast-values'), [
        [label('profitChange'), money(f.company_change)],
        [label('forecastContribution'), money(f.contribution)],
        [label('ridersBreakEven'), f.break_even_available ? String(f.break_even_passengers) : '—'],
      ]);
      $('fleet-count').textContent =
        label('fleetShort') +
        ': ' +
        f.fleet_available +
        ' / ' +
        f.fleet_needed +
        ' · ' +
        money(frame.train_family.lease_monthly) +
        label('monthlyUnit');
      $('lease-train').textContent = label('leaseShort');
      $('return-train').textContent = label('returnShort');
      $('fleet-count').dataset.needed = f.fleet_needed;
      $('fleet-overview').textContent =
        label('fleet') +
        ': ' +
        f.fleet_available +
        ' / ' +
        f.fleet_needed +
        ' · ' +
        label('trainLease') +
        ': ' +
        money(frame.train_family.lease_monthly);
    }
    $('capacity-summary').textContent =
      (result.capacity_summary || []).map((r) => r.label + ' · ' + r.value).join(' · ') || label('capacityTitle');
    $('capacity-summary').title = label('capacityBreakdown');
    ledgerRows(
      $('analysis-capacity-values'),
      (result.capacity_details || []).map((r) => [r.label, r.value]),
    );
    $('service-issue').hidden = !result.error;
    $('service-issue').textContent = result.error
      ? label(result.error) + ' ' + (result.capacity_summary || []).map((r) => r.label + ' · ' + r.value).join(' · ')
      : '';
    $('extension-price').textContent =
      prefs.replace_service_id && !result.error
        ? label('moveFleet') +
          ': ' +
          f.fleet_available +
          ' · ' +
          label('additionalLeases') +
          ': ' +
          f.additional_leases +
          ' · ' +
          label('trainLeaseTotal') +
          ': ' +
          money(Math.max(f.fleet_needed, f.fleet_available) * frame.train_family.lease_monthly)
        : '';
    $('apply-service').textContent =
      label(prefs.replace_service_id ? 'applyExtension' : 'applyService') +
      (prefs.replace_service_id && f.additional_leases > 0
        ? ' · +' + money(f.additional_leases * frame.train_family.lease_monthly) + label('monthlyUnit')
        : '');
    $('apply-service').disabled =
      busy ||
      !!result.error ||
      !f.open ||
      (prefs.replace_service_id
        ? f.additional_leases > 0 && frame.player.cash < 0
        : f.fleet_needed > f.fleet_available);
  }, 120);
}
$('browse-prospects').onclick = () => {
  if (!document.body.classList.contains('expanded')) $('expand').click();
  scrollTo(0, 0);
};
$('finance-link').onclick = () => {
  prefs.loan_amount = dollarInput(fundingShortfall());
  prefs.company_pane = 'finance';
  showView('company');
};
$('build-line').onclick = () => {
  const co = chosenPair();
  if (co) command('build_line', { corridor_id: co.id, alignment_id: prefs.alignment_id });
};
$('plan-existing-service').onclick = () => {
  const co = chosenPair();
  if (co?.existing) {
    loadService(co.id);
    prefs.service_pane = 'plan';
    savePreferences();
    showView('services');
  }
};
$('route-query').oninput = (e) => {
  prefs.service_query = e.target.value;
  savePreferences();
  renderServices();
};
$('all-service-routes').onclick = async () => {
  prefs.service_map_sections = [];
  savePreferences();
  await syncUI();
  renderServices();
};
$('service-corridor').onchange = async (e) => {
  loadService(e.target.value);
  savePreferences();
  await syncUI();
  renderServices();
  renderGuidance();
};
$('frequency').oninput = (e) => {
  prefs.frequency = Number(e.target.value);
  savePreferences();
  updateDraftGuard();
  queueForecast();
  renderGuidance();
};
$('fare').oninput = (e) => {
  prefs.fare = Math.round(Number(e.target.value) * 100);
  savePreferences();
  updateDraftGuard();
  queueForecast();
  renderGuidance();
};
$('optimize-frequency').onclick = async () => {
  const ticket = forecastTicket;
  const button = $('optimize-frequency');
  button.disabled = true;
  try {
    const result = await call('best_frequency', {
      corridor_id: prefs.service_corridor,
      fare: prefs.fare,
      replace_service_id: prefs.replace_service_id,
    });
    if (ticket !== forecastTicket || prefs.view !== 'services') return;
    if (result.available) {
      prefs.frequency = result.frequency;
      $('frequency').value = result.frequency;
      savePreferences();
      updateDraftGuard();
      await syncUI();
      queueForecast();
    }
    message = result.available ? (result.frequency ? 'frequencyOptimized' : 'zeroBestFrequency') : 'noFeasibleFrequency';
    $('status').textContent = label(message);
  } finally {
    button.disabled = busy;
  }
};
$('apply-service').onclick = () =>
  prefs.replace_service_id
    ? command('extend_service', {
        replace_service_id: prefs.replace_service_id,
        lease_count: lastServiceForecast.additional_leases,
      })
    : command('set_service');
$('lease-train').onclick = () => command('lease_train');
$('return-train').onclick = () => command('return_train');
$('fleet-lease').onclick = () => command('lease_train');
$('fleet-return').onclick = () => command('return_train');
$('borrow').onclick = () => command('take_loan', { amount: parseDollars(prefs.loan_amount) });
$('repay').onclick = () => command('repay_loan');
$('advance-year').onclick = () => command('advance_year');

function serviceDirty() {
  if (prefs.replace_service_id) return true;
  const accepted = frame.services.find((s) => s.corridor_id === prefs.service_corridor);
  return !accepted || accepted.frequency !== prefs.frequency || accepted.fare !== prefs.fare;
}
function draftStatusChanged() {
  if (
    serviceDirty() &&
    [
      '',
      'set_service',
      'extend_service',
      'advance_month',
      'advance_year',
      'take_loan',
      'lease_train',
      'return_train',
      'build_line',
      'rename_company',
      'new_game',
      'frequencyOptimized',
      'zeroBestFrequency',
      'noFeasibleFrequency',
    ].includes(message)
  )
    message = 'draft_not_applied';
  else if (!serviceDirty() && message === 'draft_not_applied') message = '';
  $('status').textContent = label(message || 'saved');
}
function updateDraftGuard() {
  draftStatusChanged();
  $('service-draft-note').hidden = !serviceDirty();
  $('service-draft-note').textContent = label('draftChanges');
  $('restore-plan').hidden = !serviceDirty();
  $('advance').disabled = busy || (prefs.view === 'services' && serviceDirty());
  $('advance-year').disabled = $('advance').disabled;
}
$('restore-plan').onclick = () => {
  loadService(prefs.service_corridor);
  savePreferences();
  render();
};

function parseDollars(value) {
  const s = value.trim();
  if (!/^(?:[0-9]+|[0-9]{1,3}(?:,[0-9]{3})+)(?:\.[0-9]{0,2})?$/.test(s)) return 0;
  const [whole, fraction = ''] = s.replaceAll(',', '').split('.');
  const cents = BigInt(whole) * 100n + BigInt((fraction + '00').slice(0, 2));
  return cents > BigInt(Number.MAX_SAFE_INTEGER) ? 0 : Number(cents);
}
function dollarInput(cents) {
  return (BigInt(cents) / 100n).toString() + '.' + (BigInt(cents) % 100n).toString().padStart(2, '0');
}
function selectedFunding() {
  const co = chosenPair();
  if (!co || co.existing || frame.projects.some((p) => p.corridor_id === co.id)) return null;
  const a = co.alignments.find((a) => a.id === prefs.alignment_id) || co.alignments[0];
  if (!a) return null;
  const contribution = a.cost - a.grant,
    available = Math.max(0, frame.player.cash),
    deficit = Math.max(0, -frame.player.cash);
  return { co, a, contribution, available, deficit, gap: Math.max(0, contribution - available + deficit) };
}
function fundingRows(q) {
  if (!q) return [];
  const rows = [
    [label('totalCost'), money(q.a.cost)],
    [label('grant'), '− ' + money(q.a.grant)],
    [label('contribution'), '= ' + money(q.contribution)],
    [label('fundingCash'), '− ' + money(q.available)],
  ];
  if (q.deficit > 0) rows.push([label('fundingDeficit'), '+ ' + money(q.deficit)]);
  rows.push([label('fundingGap'), '= ' + money(q.gap)]);
  return rows;
}
function fundingShortfall() {
  return selectedFunding()?.gap || 0;
}
function renderLoan() {
  const amount = parseDollars(prefs.loan_amount),
    credit = frame.debt_limit - frame.debt,
    q = selectedFunding(),
    gap = q?.gap || 0;
  $('credit-available').textContent = label('creditAvailable') + ': ' + money(credit);
  $('borrow').disabled = busy || amount <= 0 || amount > credit || frame.months_resolved >= frame.campaign_months;
  $('loan-preview').textContent =
    amount <= 0
      ? label('invalid_loan_amount')
      : amount > credit
        ? label('loan_limit')
        : label('loanInterest') + ': ' + money(Math.ceil((frame.debt + amount) / 600) - Math.ceil(frame.debt / 600));
  $('company-funding').hidden = !q;
  $('no-funding-plan').hidden = !!q;
  $('choose-funding-plan').hidden = !!q;
  if (q) {
    $('company-funding-route').textContent = corridorName(q.co) + ' · ' + label(q.a.id);
    ledgerRows($('company-funding-values'), fundingRows(q));
    $('company-funding-note').textContent =
      (q.deficit > 0 ? label('fundingNegativeCash') + ' ' : '') + label('fundingFormula');
  }
  $('shortfall-interest').hidden = gap <= 0;
  $('shortfall-interest').textContent =
    label('shortfallInterest') + ': ' + money(Math.ceil((frame.debt + gap) / 600) - Math.ceil(frame.debt / 600));
  $('borrow-shortfall').hidden = gap <= 0;
  $('borrow-shortfall').textContent = label('borrowShortfall') + ' · ' + money(gap);
  $('borrow-shortfall').disabled = busy || gap > credit;
}
$('review-funding-plan').onclick = () => {
  const q = selectedFunding();
  if (q) {
    prefs.alignment_id = q.a.id;
    prefs.build_pane = 'plan';
    showView('build');
  }
};
$('choose-funding-plan').onclick = () => {
  prefs.build_pane = 'plan';
  showView('build');
};
$('loan-amount').oninput = (e) => {
  prefs.loan_amount = e.target.value;
  savePreferences();
  renderLoan();
};
async function borrowShortfall() {
  const amount = fundingShortfall();
  if (amount <= 0) return;
  prefs.loan_amount = dollarInput(amount);
  savePreferences();
  await command('take_loan', { amount });
}
$('borrow-shortfall').onclick = borrowShortfall;
$('borrow-shortfall-build').onclick = borrowShortfall;

$('terrain-toggle').onclick = async () => {
  prefs.hide_terrain = !prefs.hide_terrain;
  savePreferences();
  await syncUI();
  $('terrain-toggle').setAttribute('aria-pressed', String(!prefs.hide_terrain));
};

// Go owns concept progression. Revisions prevent old async replies from restoring
// dismissed cues; events are serialized so rapid interactions cannot lose progress.
let guidanceTicket = 0,
  helpQueue = Promise.resolve(),
  helpComposing = false,
  helpObserver;
function acceptHelp(h) {
  if (h && (!prefs.help.version || h.revision >= prefs.help.revision)) {
    prefs.help = h;
    prefs.hint_dismissed = false;
    savePreferences();
  }
}
function scrollSurface() {
  return $('inspector').scrollHeight > $('inspector').clientHeight + 1 ? $('inspector') : document.scrollingElement;
}
function taskScroll() {
  const el = scrollSurface();
  return el.scrollTop / Math.max(1, el.scrollHeight - el.clientHeight);
}
function helpEvent(event, navigate = false) {
  const execute = async () => {
    const before = !!prefs.help.open;
    if (event.kind === 'open') event.scroll = taskScroll();
    const r = await call('help_event', { state: prefs, event, issue: message });
    acceptHelp(r.state.help);
    if (navigate || ['open', 'close', 'task', 'leave'].includes(event.kind)) {
      prefs.view = r.state.view;
      prefs.expanded = r.state.expanded;
      if (event.kind === 'task') {
        for (const k of ['service_pane', 'company_pane', 'build_pane', 'report_pane']) prefs[k] = r.state[k];
      }
      if (event.kind === 'leave' && event.value !== 'services') prefs.review_map = false;
      document.body.classList.toggle('expanded', !!prefs.expanded);
      $('expand').setAttribute('aria-pressed', String(!!prefs.expanded));
    }
    await syncUI();
    savePreferences();
    if (event.kind === 'seen') return;
    render();
    await renderGuidance();
    if (event.kind === 'open' && !before) {
      $('inspector').scrollTop = 0;
      scrollTo(0, 0);
      $('help-search').focus();
    }
    if (event.kind === 'close') {
      const surface = scrollSurface();
      surface.scrollTop = prefs.help.return_scroll * Math.max(0, surface.scrollHeight - surface.clientHeight);
      requestAnimationFrame(() => focusHelpAnchor(prefs.help.return_anchor, false));
    }
    if (event.kind === 'task') requestAnimationFrame(() => focusHelpAnchor(prefs.help.return_anchor, true));
  };
  helpQueue = helpQueue.then(execute, execute);
  return helpQueue;
}
function focusHelpAnchor(anchor, scroll) {
  const el = [...document.querySelectorAll('[data-help-anchor]')].find(
    (e) => e.dataset.helpAnchor === anchor && e.getClientRects().length,
  );
  const target = el?.querySelector('summary,button');
  if (target) {
    if (scroll) target.scrollIntoView({ block: 'center' });
    target.focus({ preventScroll: !scroll });
  } else $('tab-' + prefs.view)?.focus({ preventScroll: true });
}
function helpButton(key, fn) {
  const b = document.createElement('button');
  b.textContent = label(key);
  b.type = 'button';
  b.onclick = fn;
  return b;
}
function helpFacts(facts) {
  const dl = document.createElement('dl');
  ledgerRows(
    dl,
    facts.map((r) => [r.label, r.value]),
  );
  return dl;
}
async function renderGuidance() {
  const ticket = ++guidanceTicket;
  if (!bridge) return;
  const r = await call('help', { state: prefs, issue: message });
  if (ticket !== guidanceTicket) return;
  acceptHelp(r.state.help);
  const m = r.model;
  const oldFocus = document.activeElement,
    focusAnchor = oldFocus?.closest('[data-help-anchor]')?.dataset.helpAnchor,
    focusRole = oldFocus?.dataset.helpRole;
  helpObserver?.disconnect();
  $('map-frame').inert = !!prefs.help.open;
  $('field-notes').hidden = !prefs.help.open;
  $('help-open').setAttribute('aria-pressed', String(!!prefs.help.open));
  for (const el of document.querySelectorAll('[data-help-anchor]')) el.replaceChildren();
  document.body.classList.toggle('help-open', !!prefs.help.open);
  document.body.classList.toggle('expanded-learning', !!prefs.expanded && prefs.view === 'build');
  $('expanded-help').hidden = !prefs.expanded || prefs.view !== 'build';
  helpObserver = new IntersectionObserver(
    (entries) => {
      for (const e of entries)
        if (e.isIntersecting) {
          const concept = e.target.dataset.concept;
          if (!prefs.help.concepts?.[concept]?.seen) {
            helpEvent({ kind: 'seen', concept });
          }
          helpObserver.unobserve(e.target);
        }
    },
    { threshold: 0.6 },
  );
  for (const slot of m.slots) {
    // Empty help blocks intentionally have display:none. Use their task's
    // visibility while rebuilding, so a previous render cannot hide help forever.
    const el = [...document.querySelectorAll('[data-help-anchor]')].find(
      (e) =>
        e.dataset.helpAnchor === slot.anchor &&
        (slot.anchor === 'expanded-endpoint' || e.closest('#view-' + prefs.view)) &&
        e.parentElement?.getClientRects().length,
    );
    if (!el) continue;
    const topic = slot.topic;
    const fallback =
      prefs.view === 'services'
        ? paneFor('services') === 'fleet'
          ? 'fleet'
          : paneFor('services') === 'analysis'
            ? 'capacity'
            : 'service-plan'
        : prefs.view === 'build'
          ? paneFor('build') === 'costs'
            ? 'engineering'
            : 'endpoints'
          : prefs.view === 'company'
            ? 'finance'
            : prefs.view === 'report'
              ? 'accounts'
              : 'city';
    if (
      slot.anchor !== prefs.help.expanded &&
      slot.anchor !== (m.cue.anchor || fallback) &&
      slot.anchor !== 'expanded-endpoint'
    )
      continue;
    el.className = 'context-help';
    const cue = m.cue.concept === topic.id && m.cue.anchor === slot.anchor;
    if (cue) {
      const p = document.createElement('p');
      p.className = 'learning-cue';
      p.dataset.concept = topic.id;
      p.textContent = label(topic.summary_key);
      el.append(p);
      helpObserver.observe(p);
      const dismiss = helpButton('dismissShort', () => helpEvent({ kind: 'dismiss', concept: topic.id }));
      dismiss.dataset.helpRole = 'dismiss';
      el.append(dismiss);
    }
    if (slot.anchor === 'expanded-endpoint') {
      const task = document.createElement('strong');
      task.textContent = label('chooseProspect') + ' · ' + label(prefs.from_city) + ' – ' + label(prefs.to_city);
      el.prepend(task);
      el.append(helpButton('fieldNotes', () => helpEvent({ kind: 'open', concept: topic.id, anchor: slot.anchor })));
      continue;
    }
    const details = document.createElement('details'),
      summary = document.createElement('summary');
    summary.dataset.helpRole = 'explain';
    summary.textContent = label('explainShort');
    details.append(summary);
    details.open = prefs.help.expanded === slot.anchor;
    if (cue && innerWidth < 800) {
      summary.replaceChildren(el.querySelector('.learning-cue'));
      summary.setAttribute('aria-label', label('explainShort') + ' · ' + label(topic.title_key));
    }
    const p = document.createElement('p');
    p.textContent = label(topic.detail_key);
    details.append(
      p,
      helpFacts(slot.facts),
      helpButton('fieldNotes', () => helpEvent({ kind: 'open', concept: topic.id, anchor: slot.anchor })),
    );
    details.ontoggle = () => {
      if (details.isConnected && details.open !== (prefs.help.expanded === slot.anchor))
        helpEvent({ kind: details.open ? 'expand' : 'collapse', concept: topic.id, anchor: slot.anchor });
    };
    el.append(details);
  }
  if (prefs.help.open) {
    if (!helpComposing && document.activeElement !== $('help-search')) $('help-search').value = prefs.help.query || '';
    const list = $('help-results');
    list.replaceChildren();
    if (!m.results.length) {
      const p = document.createElement('p');
      p.textContent = label('noHelpResults');
      list.append(p);
    }
    for (const topic of m.results)
      list.append(helpButton(topic.title_key, () => helpEvent({ kind: 'topic', concept: topic.id })));
    const article = $('help-article');
    article.replaceChildren();
    if (m.article.id) {
      const t = m.article,
        h = document.createElement('h2'),
        p = document.createElement('p'),
        detail = document.createElement('p');
      h.textContent = label(t.title_key);
      p.textContent = label(t.summary_key);
      detail.textContent = label(t.detail_key);
      article.append(h, p, detail, helpFacts(m.article_facts));
      const progress = prefs.help.concepts?.[t.id] || {};
      for (const [flag, key] of [
        ['seen', 'helpProgressSeen'],
        ['dismissed', 'helpProgressDismissed'],
        ['completed', 'helpProgressCompleted'],
      ])
        if (progress[flag]) {
          const p = document.createElement('p');
          p.className = 'eyebrow';
          p.textContent = label(key);
          article.append(p);
        }
      article.append(helpButton('goToTask', () => helpEvent({ kind: 'task', concept: t.id }, true)));
    }
  }
  if (focusAnchor && focusRole) {
    const el = [...document.querySelectorAll('[data-help-anchor]')].find(
      (e) => e.dataset.helpAnchor === focusAnchor && e.getClientRects().length,
    );
    el?.querySelector('[data-help-role="' + focusRole + '"]')?.focus({ preventScroll: true });
  }
}
$('help-open').onclick = () => helpEvent({ kind: 'open' });
$('help-back').onclick = () => helpEvent({ kind: 'close' }, true);
$('help-search').oncompositionstart = () => {
  helpComposing = true;
};
$('help-search').oncompositionend = (e) => {
  helpComposing = false;
  helpEvent({ kind: 'query', value: e.target.value });
};
$('help-search').oninput = (e) => {
  if (!helpComposing && !e.isComposing) helpEvent({ kind: 'query', value: e.target.value });
};
$('help-search').onkeydown = (e) => {
  if (e.key === 'Enter') {
    e.preventDefault();
    e.stopPropagation();
  }
  if (e.key === 'Escape' && !helpComposing && !e.isComposing) {
    e.preventDefault();
    helpEvent({ kind: 'close' }, true);
  }
};
$('train-details').ontoggle = () => {
  const open = $('train-details').open;
  if (prefs.fleet_details !== open) {
    prefs.fleet_details = open;
    savePreferences();
    syncUI();
  }
};

$('extend-starter').onchange = async (e) => {
  prefs.replace_service_id = e.target.checked ? 'tokyo-nagoya' : '';
  savePreferences();
  await syncUI();
  renderServices();
  renderGuidance();
};
function renderNetwork() {
  const n = frame.network;
  $('network-report').open = true;
  $('network-report').hidden = !n.available;
  if (!n.available) return;
  const load = n.seat_km ? n.passenger_km / n.seat_km : 0;
  const rows = [
    [label('companyJourneys'), n.carried],
    [label('transferCount'), n.transfers],
    [label('railRequested'), n.requested],
    [label('alternateServed'), n.alternate_served],
    [label('railRejected'), n.rejected],
    [label('networkLoad'), (load * 100).toFixed(1) + '%'],
  ];
  for (const s of n.sections) {
    const co = frame.corridors.find((c) => c.id === s.corridor_id);
    rows.push([
      corridorName(co),
      label('pathsDelivered') +
        ' ' +
        s.delivered +
        '/' +
        s.capacity +
        ' · ' +
        label('pathsRequested') +
        ' ' +
        s.requested,
    ]);
  }
  for (const st of n.stations)
    rows.push([
      label(st.city_id),
      label('stationCalls') +
        ' ' +
        st.calls +
        '/' +
        st.capacity +
        ' · ' +
        label('boardings') +
        ' ' +
        st.boardings +
        ' · ' +
        label('transferCount') +
        ' ' +
        st.transfers,
    ]);
  ledgerRows($('network-totals'), rows);
  const options = frame.cities.map((c) => [c.id, label(c.id)]);
  selectOptions($('market-from'), options, prefs.market_from || 'tokyo');
  selectOptions($('market-to'), options, prefs.market_to || 'osaka');
  $('market-class').value = prefs.market_class || 'business';
  renderMarket();
}
let marketTicket = 0;
async function renderMarket() {
  const ticket = ++marketTicket;
  let m = frame.network.markets.find(
    (m) => m.from === $('market-from').value && m.to === $('market-to').value && m.class === $('market-class').value,
  );
  const list = $('market-journeys');
  list.replaceChildren();
  if (!m && frame.network.available) {
    const reply = await call('market', {
      From: $('market-from').value,
      To: $('market-to').value,
      Class: $('market-class').value,
    });
    if (ticket !== marketTicket) return;
    if (!reply.error) m = reply.market;
  }
  if (!m) {
    $('market-values').replaceChildren();
    return;
  }
  const rows = [
    [label('finiteDemand'), m.demand],
    [label('railRequested'), m.requested],
    [label('companyJourneys'), m.carried],
    [label('alternateServed'), m.alternate_served],
    [label('railRejected'), m.rejected],
  ];
  for (const mode of ['road', 'air', 'conventional', 'no_trip']) rows.push([label('mode_' + mode), m[mode]]);
  for (const o of m.offers)
    rows.push([
      label('mode_' + o.mode),
      money(o.fare) + ' · ' + o.minutes + ' min · ' + label('generalizedCost') + ' ' + money(o.generalized),
    ]);
  ledgerRows($('market-values'), rows);
  for (const j of frame.network.journeys.filter((j) => j.from === m.from && j.to === m.to && j.class === m.class)) {
    const p = document.createElement('p');
    p.dataset.journey = j.services.join('|');
    p.textContent =
      j.passengers +
      ' · ' +
      j.services.map((id) => routeName(frame.routes.find((r) => r.id === id))).join(' / ') +
      ' · ' +
      money(j.fare) +
      ' · ' +
      label('transferCount') +
      ' ' +
      j.transfers.length +
      (j.alternate ? ' · ' + label('alternateServed') : '');
    list.append(p);
  }
}
for (const [id, key] of [
  ['market-from', 'market_from'],
  ['market-to', 'market_to'],
  ['market-class', 'market_class'],
])
  $(id).onchange = (e) => {
    prefs[key] = e.target.value;
    savePreferences();
    renderMarket();
  };
$('watch-day').onclick = async () => {
  prefs.watching = !prefs.watching;
  prefs.replay_playing = false;
  if (!prefs.selected_train && frame.day.length) prefs.selected_train = frame.day[0].id;
  savePreferences();
  await syncUI();
  renderReplay();
  renderGuidance();
};
$('replay-play').onclick = async () => {
  if (prefs.replay_playing) {
    const r = await call('replay');
    prefs.replay_minute = Math.floor(r.minute);
  }
  prefs.replay_playing = !prefs.replay_playing;
  savePreferences();
  await syncUI();
  renderReplay();
};
$('replay-minute').oninput = async (e) => {
  prefs.replay_minute = Number(e.target.value);
  prefs.replay_playing = false;
  savePreferences();
  await syncUI();
  renderReplay();
};
$('replay-train').onchange = async (e) => {
  prefs.selected_train = e.target.value;
  savePreferences();
  await syncUI();
  renderReplay();
};
function replayClock(m) {
  m = (Math.floor(m) + 360) % 1440;
  return String(Math.floor(m / 60)).padStart(2, '0') + ':' + String(m % 60).padStart(2, '0');
}
function renderReplay() {
  $('replay-controls').hidden = prefs.view !== 'map' && !document.body.classList.contains('expanded');
  $('watch-day').disabled = !frame.day.length;
  $('watch-day').setAttribute('aria-pressed', String(prefs.watching));
  $('replay-panel').hidden = !prefs.watching;
  $('replay-minute').value = prefs.replay_minute;
  $('replay-clock').textContent = replayClock(prefs.replay_minute);
  $('replay-play').textContent = label(prefs.replay_playing ? 'pauseDay' : 'playDay');
  $('replay-play').disabled = !prefs.motion;
  const trips = frame.day;
  if (!trips.some((t) => t.id === prefs.selected_train)) {
    prefs.selected_train = trips[0]?.id || '';
    if (bridge) syncUI();
  }
  selectOptions(
    $('replay-train'),
    trips.map((t) => [t.id, t.calls.map((c) => label(c.city_id)).join(' → ') + ' · ' + replayClock(t.departure)]),
    prefs.selected_train || trips[0]?.id,
  );
  if (!prefs.selected_train && trips.length) prefs.selected_train = trips[0].id;
  const t = trips.find((t) => t.id === prefs.selected_train);
  $('train-inspector').textContent = t
    ? label('representativeDay') +
      ' · ' +
      routeName(frame.routes.find((r) => r.id === t.service_id)) +
      ' · ' +
      t.calls.map((c) => label(c.city_id) + ' ' + replayClock(c.arrival)).join(' → ')
    : label('noNetworkReport');
}

setInterval(async () => {
  if (!bridge || !prefs.watching || !prefs.replay_playing || busy) return;
  try {
    const r = await call('replay');
    $('replay-clock').textContent = replayClock(r.minute);
    $('replay-minute').value = Math.floor(r.minute);
    if (r.description) $('train-inspector').textContent = r.description;
  } catch {}
}, 500);

$('network-report').ontoggle = () => {
  if (prefs.network_details !== $('network-report').open) {
    prefs.network_details = $('network-report').open;
    savePreferences();
    syncUI();
  }
};

installWorkspaceControls();

function paneFor(task) {
  const value =
    prefs[{ services: 'service_pane', company: 'company_pane', build: 'build_pane', report: 'report_pane' }[task]];
  return (
    {
      services: ['plan', 'fleet', 'analysis'],
      company: ['finance', 'identity', 'settings'],
      build: ['plan', 'costs', 'projects'],
      report: ['overview', 'accounts', 'network', 'markets'],
    }[task] || []
  ).includes(value)
    ? value
    : { services: 'plan', company: 'finance', build: 'plan', report: 'overview' }[task];
}
function renderTaskPanes() {
  for (const task of ['services', 'company', 'build', 'report']) {
    const active = paneFor(task);
    for (const node of document.querySelectorAll('[data-task-section="' + task + '"]'))
      node.hidden = node.dataset.pane !== active;
    for (const tab of document.querySelectorAll('[data-task="' + task + '"]')) {
      tab.setAttribute('aria-selected', String(tab.dataset.pane === active));
      tab.tabIndex = tab.dataset.pane === active ? 0 : -1;
      tab.setAttribute('aria-controls', task + '-section-' + tab.dataset.pane);
    }
  }
  for (const [id, key] of [
    ['idle-disclosure', 'idle'],
    ['capacity-disclosure', 'capacity'],
    ['service-path-details', 'service-path'],
    ['build-funding-details', 'funding'],
    ['company-funding-details', 'company-funding'],
    ['city-browser', 'city-list'],
  ])
    $(id).open = !!prefs.disclosures?.[key];
  $('command-dock').value = prefs.command_dock === 'left' ? 'left' : 'top';
  $('panel-width').value = prefs.panel_width || 440;
  $('panel-width-value').textContent = (prefs.panel_width || 440) + ' px';
  renderProjects();
}
function renderProjects() {
  if (!frame) return;
  const list = $('project-list');
  list.replaceChildren();
  for (const project of frame.projects) {
    const co = frame.corridors.find((c) => c.id === project.corridor_id);
    const b = document.createElement('button');
    b.textContent =
      corridorName(co) +
      ' · ' +
      label(project.ready <= frame.months_resolved ? 'readyForService' : 'underConstruction');
    b.onclick = async () => {
      prefs.from_city = co.from_city;
      prefs.to_city = co.to_city;
      prefs.alignment_id = project.alignment_id;
      prefs.build_pane = 'plan';
      savePreferences();
      await syncUI();
      render();
    };
    list.append(b);
  }
  if (!frame.projects.length) {
    const p = document.createElement('p');
    p.textContent = label('plans');
    list.append(p);
  }
}
function saveDisclosure(id, open) {
  if (!prefs.disclosures || typeof prefs.disclosures !== 'object') prefs.disclosures = {};
  if (open) prefs.disclosures[id] = true;
  else delete prefs.disclosures[id];
  savePreferences();
  if (bridge) syncUI();
}
function applyWorkspace(layout) {
  if (!layout) return;
  document.documentElement.style.setProperty('--prospect-height', (layout.Prospects.Max.Y - layout.Prospects.Min.Y) + 'px');
  document.body.dataset.commandDock = layout.DockLeft ? 'left' : 'top';
  document.documentElement.style.setProperty('--inspector-width', layout.PanelWidth + 'px');
  document.documentElement.style.setProperty(
    '--phone-map-height',
    Math.max(0, layout.Map.Max.Y - layout.Map.Min.Y) + 'px',
  );
  $('inspector-resize').setAttribute('aria-valuemin', '320');
  $('inspector-resize').setAttribute('aria-valuemax', '800');
  $('inspector-resize').setAttribute('aria-valuenow', String(layout.PanelWidth));
  $('inspector-resize').setAttribute('aria-label', label('resizeInspector'));
}
async function resizeInspector(width) {
  prefs.panel_width = Math.max(320, Math.min(800, Math.round(width)));
  savePreferences();
  await syncUI();
  $('panel-width').value = prefs.panel_width;
  $('panel-width-value').textContent = prefs.panel_width + ' px';
}
function installWorkspaceControls() {
  for (const b of document.querySelectorAll('[data-task]'))
    b.onclick = async () => {
      prefs[
        { services: 'service_pane', company: 'company_pane', build: 'build_pane', report: 'report_pane' }[
          b.dataset.task
        ]
      ] = b.dataset.pane;
      savePreferences();
      await syncUI();
      render();
      $('inspector').scrollTop = 0;
    };
  for (const nav of document.querySelectorAll('.task-tabs'))
    nav.onkeydown = (e) => {
      if (!['ArrowLeft', 'ArrowRight', 'Home', 'End'].includes(e.key)) return;
      e.preventDefault();
      const buttons = [...nav.querySelectorAll('button')],
        i = buttons.indexOf(document.activeElement);
      const at =
        e.key === 'Home'
          ? 0
          : e.key === 'End'
            ? buttons.length - 1
            : (i + (e.key === 'ArrowRight' ? 1 : -1) + buttons.length) % buttons.length;
      buttons[at].focus();
      buttons[at].click();
    };
  for (const [id, key] of [
    ['idle-disclosure', 'idle'],
    ['capacity-disclosure', 'capacity'],
    ['service-path-details', 'service-path'],
    ['build-funding-details', 'funding'],
    ['company-funding-details', 'company-funding'],
    ['city-browser', 'city-list'],
  ])
    $(id).ontoggle = () => {
      if ($(id).open !== !!prefs.disclosures?.[key]) saveDisclosure(key, $(id).open);
      if (id === 'city-browser') renderCities();
    };
  $('command-dock').onchange = async (e) => {
    prefs.command_dock = e.target.value;
    savePreferences();
    await syncUI();
  };
  $('panel-width').oninput = (e) => resizeInspector(Number(e.target.value));
  $('layout-reset').onclick = async () => {
    prefs.command_dock = 'top';
    prefs.panel_width = 0;
    savePreferences();
    await syncUI();
    renderTaskPanes();
  };
  $('region-select').onchange = async (e) => {
    prefs.region = e.target.value;
    savePreferences();
    await syncUI();
    await call('region', { ID: prefs.region });
    $('phone-map-tools').open = false;
    $('dock-map-tools').open = false;
  };
  $('language-quick').onclick = async () => {
    prefs.locale = prefs.locale === 'en' ? 'zh-Hans' : 'en';
    savePreferences();
    await syncUI();
    render();
  };
  const split = $('inspector-resize');
  let dragging = false;
  split.onpointerdown = (e) => {
    dragging = true;
    split.setPointerCapture(e.pointerId);
  };
  split.onpointermove = (e) => {
    if (dragging) resizeInspector(innerWidth - e.clientX);
  };
  split.onpointerup = split.onpointercancel = () => {
    dragging = false;
  };
  split.onkeydown = (e) => {
    if (!['ArrowLeft', 'ArrowRight', 'Home', 'End'].includes(e.key)) return;
    e.preventDefault();
    resizeInspector(
      e.key === 'Home' ? 320 : e.key === 'End' ? 800 : (prefs.panel_width || 440) + (e.key === 'ArrowLeft' ? 20 : -20),
    );
  };
  split.ondblclick = () => resizeInspector(440);
}
