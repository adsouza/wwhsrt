'use strict';
// All business state and commands belong to Go. This file manages semantic
// controls, local presentation preferences, and the browser storage adapter.
const $ = id => document.getElementById(id);
let db, bridge, frame, texts = {}, busy = false, composing = false, message = '';
const preferenceKey = 'wwhsrt-business-ui-v2';
let prefs = {locale: initialLocale(), selected:'', view:'map', query:'', draft_name:'', hint_dismissed:false, motion:!matchMedia('(prefers-reduced-motion: reduce)').matches,from_city:'tsuruga',to_city:'osaka',choosing:'from',alignment_id:'mountain',service_corridor:'tokyo-nagoya',frequency:6,fare:9000,expanded:false,loan_amount:'',hide_terrain:false,appearance:'system'};
function initialLocale() { return /^(zh$|zh-Hans|zh-CN|zh-SG)/i.test(navigator.language) ? 'zh-Hans' : 'en'; }
try {
  const saved = JSON.parse(localStorage.getItem(preferenceKey));
  if (saved && typeof saved === 'object') {
    for (const k of Object.keys(prefs)) if (typeof saved[k] === typeof prefs[k]) prefs[k] = saved[k];
  }
} catch {}
prefs.locale = prefs.locale === 'zh-Hans' ? 'zh-Hans' : 'en';
prefs.view = ['map','build','services','report','company'].includes(prefs.view) ? prefs.view : 'map';
prefs.appearance = ['light','dark'].includes(prefs.appearance) ? prefs.appearance : 'system';
const systemAppearance = matchMedia('(prefers-color-scheme: dark)');
function applyAppearance(palette) {
 const dark=prefs.appearance==='dark'||prefs.appearance==='system'&&systemAppearance.matches;
 document.documentElement.dataset.theme=dark?'dark':'light';
 document.querySelector('meta[name=theme-color]').content=dark?'#0f1d2e':'#1264d5';
 if(palette)for(const [key,c]of Object.entries(palette))document.documentElement.style.setProperty('--'+key,`rgb(${c.R}, ${c.G}, ${c.B})`);
}
applyAppearance();
systemAppearance.addEventListener('change',async()=>{if(prefs.appearance==='system'){applyAppearance();await syncUI();render();}});
function savePreferences() { try { localStorage.setItem(preferenceKey,JSON.stringify(prefs)); } catch {} }
function money(cents) {return new Intl.NumberFormat(prefs.locale,{style:'currency',currency:'USD',currencyDisplay:'code',maximumFractionDigits:2}).format(cents/100);}
function month(m) {return new Intl.DateTimeFormat(prefs.locale,{year:'numeric',month:'long',timeZone:'UTC'}).format(new Date(Date.UTC(m.year,m.month-1,1)));}
function label(key) {return texts[key] || key;}
function startupError(code) {
  const zh = prefs.locale === 'zh-Hans';
  const fallback = {locked:zh?'游戏已在另一个标签页中打开。请关闭后重新加载。':'This game is open in another tab. Close it and reload here.',storageUnavailable:zh?'本地存储不可用。请允许此网站使用存储后重试。':'Local storage is unavailable. Enable storage for this site and reload.',invalid_save:zh?'存档无效或版本不兼容。原始存档已保留。':'The save is invalid or incompatible. Your original data has been preserved.'};
  $('loading').textContent = texts[code] || fallback[code] || (zh?'无法打开游戏。原始存档已保留。':'Could not open this game. Saved data has been preserved.');
  $('startup').hidden=false;$('game').hidden=true;$('reload').hidden=false;
}
window.railStartupError=startupError;
$('reload').onclick=()=>location.reload();
window.railStorageLoad=()=>new Promise((resolve,reject)=>{
  const tx=db.transaction('snapshots','readonly');let value='';
  const r=tx.objectStore('snapshots').get('campaign');r.onsuccess=()=>{value=r.result||'';};
  tx.oncomplete=()=>resolve(value);tx.onabort=()=>reject(tx.error);tx.onerror=()=>{};
});
window.railStorageCommit=value=>new Promise((resolve,reject)=>{
  const tx=db.transaction('snapshots','readwrite',{durability:'strict'});
  tx.objectStore('snapshots').put(value,'campaign');
  tx.oncomplete=()=>resolve(true);tx.onabort=()=>reject(tx.error);tx.onerror=()=>{};
});
async function call(kind,body={}) {return JSON.parse(await bridge(kind,JSON.stringify(body)));}
async function syncUI() {if(bridge){const s=await call('ui',{...prefs,portrait:innerHeight>innerWidth,dark:prefs.appearance==='dark'||prefs.appearance==='system'&&systemAppearance.matches});texts=s.texts;applyAppearance(s.palette);return s;}}
function render() {
  if(!frame)return;
  $('game').setAttribute('aria-busy',String(busy));
  document.documentElement.lang=prefs.locale;
  document.title=label('title');document.body.dataset.view=prefs.view;
  for(const e of document.querySelectorAll('[data-t]'))e.textContent=label(e.dataset.t);
  $('month').textContent=month(frame.month);$('cash').textContent=money(frame.player.cash);
  $('appearance').value=prefs.appearance;
  $('language').value=prefs.locale;$('search').value=prefs.query;$('company-name').value=prefs.draft_name;$('motion').checked=prefs.motion;$('terrain-toggle').setAttribute('aria-pressed',String(!prefs.hide_terrain));
  $('hint').hidden=prefs.hint_dismissed;
  for(const name of ['map','build','services','report','company']) {
    $('view-'+name).hidden=prefs.view!==name;
    if(prefs.view===name)$('tab-'+name).setAttribute('aria-current','page');else $('tab-'+name).removeAttribute('aria-current');
  }
  renderCities();renderReport();renderBusiness();renderIdleLines();
  $('status').textContent=label(message || 'saved');
  $('advance').disabled=busy||frame.months_resolved>=frame.campaign_months||(prefs.view==='services'&&serviceDirty());
  $('new-game').disabled=busy||composing;$('confirm-new-game').disabled=busy;$('cancel-new-game').disabled=busy;$('new-game-dialog').setAttribute('aria-busy',String(busy));
  $('rename').disabled=busy||composing;$('advance-year').disabled=busy||frame.months_resolved>=frame.campaign_months||(prefs.view==='services'&&serviceDirty());
  $('expand').textContent=label(document.body.classList.contains('expanded')?'exitFullScreen':'fullScreen');
  $('map-panel').setAttribute('aria-label',label('map'));
  $('map-frame').title=label('map')+' · Equal Earth';
}
function renderCities() {
  const list=$('city-list');list.replaceChildren();
  const query=prefs.query.toLocaleLowerCase().trim();
  for(const c of frame.cities) {
    const aliases=c.name+' '+label(c.id)+' '+({shenyang:'沈阳',beijing:'北京',shanghai:'上海',chengdu:'成都',hong_kong:'香港',seoul:'首尔',tokyo:'东京',nagoya:'名古屋',osaka:'大阪',tsuruga:'敦贺',shin_hakodate_hokuto:'新函馆北斗',sapporo:'札幌'}[c.id]||'');
    if(!aliases.toLocaleLowerCase().includes(query))continue;
    const b=document.createElement('button');b.textContent=label(c.id);b.dataset.city=c.id;
    b.setAttribute('aria-pressed',String(prefs.selected===c.id));b.onclick=async()=>{prefs.selected=c.id;savePreferences();await syncUI();renderCities();};list.append(b);
  }
  $('no-results').hidden=list.childElementCount>0;
  const details=$('city-details');details.replaceChildren();const c=frame.cities.find(c=>c.id===prefs.selected);
  if(c){const heading=document.createElement('h2');heading.textContent=label('selected');const name=document.createElement('strong');name.textContent=label(c.id);const coords=document.createElement('p');coords.textContent=label('latitude')+': '+c.lat.toFixed(4)+'° · '+label('longitude')+': '+c.lon.toFixed(4)+'°';details.append(heading,name,coords);}
  else{const p=document.createElement('p');p.textContent=label('selectCity');details.append(p);}
}
function renderReport(){
  $('resolved').textContent=String(frame.months_resolved);const report=frame.reports.at(-1);$('no-report').hidden=!!report;$('report-month').textContent=report?month(report.month):'';
  const ledger=$('ledger');ledger.replaceChildren();if(!report)return;
  for(const [key,value]of [['opening',report.opening_cash],['passengers',report.passengers],['revenue',report.revenue],['runningCosts',report.running_costs],['accessCosts',report.access_costs],['stationCosts',report.station_costs],['leaseCosts',report.lease_costs],['maintenance',report.maintenance],['interest',report.interest],['construction',report.construction],['grants',report.grants],['borrowed',report.borrowed],['repaid',report.repaid],['closing',report.closing_cash]]){const row=document.createElement('div'),dt=document.createElement('dt'),dd=document.createElement('dd');dt.textContent=label(key);dd.textContent=key==='passengers'?new Intl.NumberFormat(prefs.locale).format(value):money(value);row.append(dt,dd);ledger.append(row);}
}
async function command(kind,extra={}) {
  if(busy||composing||$('new-game-dialog').open&&kind!=='new_game')return;busy=true;message='advancing';$('new-game-error').hidden=true;render();
  try {const s=await call('command',{kind,revision:frame.revision,name:prefs.draft_name,corridor_id:prefs.service_corridor,frequency:prefs.frequency,fare:prefs.fare,...extra});frame=s.frame;message=s.error||kind;if(kind==='advance_month'&&!s.error)prefs.view='report';
    if(kind==='new_game'&&!s.error){prefs=s.state;applyAppearance(s.palette);document.body.classList.remove('expanded');$('expand').setAttribute('aria-pressed','false');$('new-game-dialog').close();$('inspector').scrollTop=0;scrollTo(0,0);}
    savePreferences();}
  catch {message='storage_failed';}
  finally {busy=false;render();if(kind==='new_game'&&$('new-game-dialog').open){$('new-game-error').textContent=label(message);$('new-game-error').hidden=false;}}

}
$('new-game').onclick=()=>{if(busy||composing)return;$('new-game-company').textContent=frame.player.name+' · '+month(frame.month);$('new-game-error').hidden=true;$('new-game-dialog').showModal();$('cancel-new-game').focus();};
$('cancel-new-game').onclick=()=>{if(!busy)$('new-game-dialog').close();};
$('new-game-dialog').addEventListener('cancel',e=>{if(busy)e.preventDefault();});
$('confirm-new-game').onclick=()=>command('new_game');
$('advance').onclick=()=>command('advance_month');
$('company-form').onsubmit=e=>{e.preventDefault();if(!composing)command('rename_company');};
for(const e of document.querySelectorAll('[data-view]'))e.onclick=()=>showView(e.dataset.view);
$('dismiss').onclick=()=>{prefs.hint_dismissed=true;savePreferences();$('hint').hidden=true;};
$('search').oninput=e=>{prefs.query=e.target.value;savePreferences();renderCities();};
$('company-name').oninput=e=>{prefs.draft_name=e.target.value;savePreferences();};
$('company-name').oncompositionstart=()=>{composing=true;$('rename').disabled=true;};
$('company-name').oncompositionend=()=>{composing=false;$('rename').disabled=busy;};
$('company-name').onkeydown=e=>{if(e.key==='Enter'&&(e.isComposing||composing||e.keyCode===229)){e.preventDefault();e.stopPropagation();}};
$('language').onchange=async e=>{prefs.locale=e.target.value;savePreferences();await syncUI();render();};
$('appearance').onchange=async e=>{prefs.appearance=e.target.value;savePreferences();applyAppearance();await syncUI();render();};
$('motion').onchange=async e=>{prefs.motion=e.target.checked;savePreferences();await syncUI();};
for(const [id,kind]of [['fit','fit'],['focus-route','focus_route'],['zoom-in','zoom_in'],['zoom-out','zoom_out']])$(id).onclick=()=>call(kind);
$('expand').onclick=()=>{const expanded=document.body.classList.toggle('expanded');prefs.expanded=expanded;$('expand').setAttribute('aria-pressed',String(expanded));$('expand').textContent=label(expanded?'exitFullScreen':'fullScreen');};
document.addEventListener('keydown',e=>{if(e.key==='Escape'&&document.body.classList.contains('expanded'))$('expand').click();});
$('export').onclick=async()=>{const s=await call('export');const url=URL.createObjectURL(new Blob([s.save],{type:'application/json'}));const a=document.createElement('a');a.href=url;a.download='wwhsrt-development-snapshot.json';a.click();setTimeout(()=>URL.revokeObjectURL(url),1000);};
window.railMapChanged=s=>{const snapshot=JSON.parse(s);Object.assign(prefs,{selected:snapshot.state.selected,from_city:snapshot.state.from_city,to_city:snapshot.state.to_city,choosing:snapshot.state.choosing,alignment_id:snapshot.state.alignment_id});savePreferences();renderCities();renderBuild();};
window.railReady=()=>startGame().catch(()=>startupError('startupFailed'));
async function startGame(){
  // Give Ebitengine a visible, nonzero viewport before waiting for its first
  // Update; hiding the iframe until after that Update would deadlock startup.
  $('game').hidden=false;
  bridge=$('map-frame').contentWindow.rail;
  const s=await call('snapshot');frame=s.frame;
  if(!prefs.draft_name)prefs.draft_name=frame.player.name;
  await syncUI();render();$('startup').hidden=true;$('game').hidden=false;
  window.railSnapshot=()=>call('snapshot'); // Read-only probe; test actions use real controls.
  if('serviceWorker'in navigator)navigator.serviceWorker.register('sw.js').catch(()=>{});
}
document.addEventListener('visibilitychange',()=>{savePreferences();if(document.hidden)return;if(bridge)call('snapshot').then(s=>{frame=s.frame;render();});});
addEventListener('resize',()=>{if(bridge)syncUI();});
addEventListener('pagehide',savePreferences);
async function open(attempt=0) {
  try {
    if(!navigator.locks)throw Error('storageUnavailable');
    await navigator.locks.request('wwhsrt-first-business-v2',{ifAvailable:true},async lock=>{
      if(!lock){
        // A closing/reloading tab can retain its lock briefly after the next
        // document starts. Allow that handoff before reporting a live rival tab.
        if(attempt<20){setTimeout(()=>open(attempt+1),100);return;}
        startupError('locked');return;
      }
      db=await new Promise((resolve,reject)=>{const r=indexedDB.open('wwhsrt-first-business',1);r.onupgradeneeded=()=>r.result.createObjectStore('snapshots');r.onsuccess=()=>resolve(r.result);r.onerror=()=>reject(r.error);r.onblocked=()=>reject(Error('blocked'));});
      $('map-frame').src='map.html';
      await new Promise(resolve=>addEventListener('pagehide',resolve,{once:true}));
      db.close(); // Release ownership promptly during reload/close.
    });
  } catch {startupError('storageUnavailable');}
}
open();
addEventListener('pageshow',e=>{if(e.persisted)location.reload();});

let forecastTicket=0,forecastTimer;
function corridorName(c){return label(c.from_city)+' – '+label(c.to_city);}
function reviewIdleLine(id){if(document.body.classList.contains('expanded'))$('expand').click();loadService(id);showView('services');}
function renderIdleLines(){
 const lines=frame.idle_lines||[],total=lines.reduce((sum,line)=>sum+line.maintenance+line.leases,0),active=lines.length>0;
 document.body.classList.toggle('has-idle-lines',active);$('idle-warning').hidden=!active;$('idle-footer').hidden=!active;$('idle-details').hidden=!active;
 if(!active){$('idle-line-list').replaceChildren();return;}
 $('idle-title').textContent=label('idleLinesTitle')+' · '+lines.length;$('idle-summary').textContent=label('idleCosts')+': '+money(total)+label('monthlyUnit');
 $('idle-footer').textContent=label('idleShort')+' · '+lines.length+' · '+money(total)+label('monthlyUnit');
 $('review-idle').onclick=$('idle-footer').onclick=()=>reviewIdleLine(lines[0].corridor_id);
 const list=$('idle-line-list');list.replaceChildren();for(const line of lines){const b=document.createElement('button');b.dataset.idleCorridor=line.corridor_id;b.textContent=corridorName(line)+' · '+money(line.maintenance+line.leases)+label('monthlyUnit')+' · '+label(line.needs_trains?'idleNeedsTrains':'idleNeedsService');b.onclick=()=>reviewIdleLine(line.corridor_id);list.append(b);}
}
function isOpen(c){return c.existing||frame.projects.some(p=>p.corridor_id===c.id&&p.ready<=frame.months_resolved);}
function chosenPair(){return frame.corridors.find(c=>(c.from_city===prefs.from_city&&c.to_city===prefs.to_city)||(c.to_city===prefs.from_city&&c.from_city===prefs.to_city));}
function selectOptions(element,items,value){element.replaceChildren();for(const [id,title]of items){const o=document.createElement('option');o.value=id;o.textContent=title;element.append(o);}element.value=value;}
function ledgerRows(element,rows){element.replaceChildren();for(const [title,value]of rows){const row=document.createElement('div'),dt=document.createElement('dt'),dd=document.createElement('dd');dt.textContent=title;dd.textContent=value;row.append(dt,dd);element.append(row);}}
async function showView(view){prefs.view=view;savePreferences();await syncUI();render();$('inspector').scrollTop=0;scrollTo(0,0);}
function renderBusiness(){
  $('debt').textContent=label('debt')+': '+money(frame.debt);if(!prefs.loan_amount)prefs.loan_amount=dollarInput(fundingShortfall()||100000000);$('loan-amount').value=prefs.loan_amount;renderLoan();$('repay').disabled=busy||frame.debt<=0||frame.player.cash<Math.min(50000000000,frame.debt);
  renderBuild();renderServices();
}
function renderBuild(){
 const cityOptions=frame.cities.map(c=>[c.id,label(c.id)]);selectOptions($('from-city'),cityOptions,prefs.from_city);selectOptions($('to-city'),cityOptions,prefs.to_city);
 const list=$('opportunities');list.replaceChildren();for(const c of frame.corridors.filter(c=>!c.existing&&!frame.projects.some(p=>p.corridor_id===c.id))){const b=document.createElement('button');b.textContent=corridorName(c);b.dataset.corridor=c.id;b.onclick=async()=>{prefs.from_city=c.from_city;prefs.to_city=c.to_city;prefs.alignment_id=c.alignments[0].id;savePreferences();await syncUI();renderBuild();};list.append(b);}
 const co=chosenPair(),project=co&&frame.projects.find(p=>p.corridor_id===co.id);$('alignments').replaceChildren();$('quote').replaceChildren();$('build-preview').replaceChildren();$('funding-gap').textContent='';$('build-status').textContent='';$('terrain-explanation').textContent='';$('build-line').disabled=true;$('finance-link').hidden=true;$('borrow-shortfall-build').hidden=true;$('project-service').hidden=true;$('project-progress').hidden=true;
 if(!co){$('build-status').textContent=label('no_route');return;}
 if(co.existing){$('build-status').textContent=label('publicTrack');return;}
 if(project){$('project-progress').hidden=false;$('project-progress').max=project.ready-project.started;$('project-progress').value=Math.min(frame.months_resolved-project.started,project.ready-project.started);$('project-service').hidden=project.ready>frame.months_resolved;$('project-service').onclick=()=>{loadService(co.id);showView('services');};$('build-status').textContent=project.ready<=frame.months_resolved?label('readyForService'):label('underConstruction')+' · '+label('opens')+' '+month({year:2029+Math.floor(project.ready/12),month:1+project.ready%12})+' · '+(project.ready-frame.months_resolved)+' '+label('monthsLeft');return;}
 if(!co.alignments.some(a=>a.id===prefs.alignment_id))prefs.alignment_id=co.alignments[0].id;
 for(const a of co.alignments){const b=document.createElement('button');b.textContent=label(a.id)+' · '+a.km+' km · '+a.minutes+' min · '+label('duration')+' '+a.months;b.setAttribute('aria-pressed',String(a.id===prefs.alignment_id));b.onclick=async()=>{prefs.alignment_id=a.id;savePreferences();await syncUI();renderBuild();};$('alignments').append(b);}
 const a=co.alignments.find(a=>a.id===prefs.alignment_id),net=a.cost-a.grant;
 $('terrain-explanation').textContent=label((co.id==='hokkaido-extension'?'hokkaido':'kansai')+(a.id==='mountain'?'MountainWhy':'ValleyWhy'));
 const quoteID=co.id,variant=a.id;call('forecast',{corridor_id:co.id,alignment_id:a.id,frequency:6,fare:9000}).then(result=>{if(chosenPair()?.id===quoteID&&prefs.alignment_id===variant&&!result.error){ledgerRows($('build-preview'),[[label('potential'),money(result.forecast.contribution)]]);}}).catch(()=>{});ledgerRows($('quote'),fundingRows(selectedFunding()));
 if(net>frame.player.cash){$('funding-gap').textContent=(frame.player.cash<0?label('fundingNegativeCash')+' ':'')+label('fundingFormula')+' '+label('shortfallInterest')+': '+money(Math.ceil((frame.debt+net-frame.player.cash)/600)-Math.ceil(frame.debt/600));$('finance-link').hidden=false;$('borrow-shortfall-build').hidden=false;$('borrow-shortfall-build').textContent=label('borrowShortfall')+' · '+money(net-frame.player.cash);$('borrow-shortfall-build').disabled=busy||net-frame.player.cash>frame.debt_limit-frame.debt;}
 $('build-line').disabled=busy||net>frame.player.cash;
}
function renderServices(){
 const available=frame.corridors.filter(isOpen);if(!available.some(c=>c.id===prefs.service_corridor)&&available.length)loadService(available[0].id);
 selectOptions($('service-corridor'),available.map(c=>[c.id,corridorName(c)]),prefs.service_corridor);$('frequency').value=prefs.frequency;$('fare').value=prefs.fare/100;
 $('service-draft-note').textContent=serviceDirty()?label('draft_not_applied'):label('saved');$('restore-plan').hidden=!serviceDirty();
 $('fleet-count').textContent=label('fleet')+': '+frame.fleet.filter(f=>f.corridor_id===prefs.service_corridor).length;
 $('lease-train').disabled=busy;$('return-train').disabled=busy;queueForecast();
}
function loadService(id){prefs.service_corridor=id;const s=frame.services.find(s=>s.corridor_id===id);prefs.frequency=s?s.frequency:4;prefs.fare=s?s.fare:9000;}
function queueForecast(){
 const ticket=++forecastTicket;clearTimeout(forecastTimer);$('apply-service').disabled=true;
 forecastTimer=setTimeout(async()=>{const result=await call('forecast',{corridor_id:prefs.service_corridor,frequency:prefs.frequency,fare:prefs.fare});if(ticket!==forecastTicket)return;
  const f=result.forecast;$('forecast-status').textContent=result.error?label(result.error):(f.fleet_needed>f.fleet_available?label('fleet_shortage'):'');
  if(!result.error){ledgerRows($('forecast-values'),[[label('passengers'),new Intl.NumberFormat(prefs.locale).format(f.passengers)],[label('needFleet'),String(f.fleet_needed)],[label('revenue'),money(f.revenue)],[label('forecastCosts'),money(f.costs)],[label('operatingContribution'),money(f.contribution)]]);}
  $('apply-service').disabled=busy||!!result.error||!f.open||f.fleet_needed>f.fleet_available;
 },120);
}
for(const [id,key]of [['from-city','from_city'],['to-city','to_city']])$(id).onchange=async e=>{prefs[key]=e.target.value;prefs.alignment_id='';savePreferences();await syncUI();renderBuild();};
for(const [id,which]of [['choose-from','from'],['choose-to','to']])$(id).onclick=async()=>{prefs.choosing=which;await syncUI();if(!document.body.classList.contains('expanded'))$('expand').click();scrollTo(0,0);};
$('finance-link').onclick=()=>{prefs.loan_amount=dollarInput(fundingShortfall());showView('company');};
$('build-line').onclick=()=>{const co=chosenPair();if(co)command('build_line',{corridor_id:co.id,alignment_id:prefs.alignment_id});};
$('service-corridor').onchange=async e=>{loadService(e.target.value);savePreferences();await syncUI();renderServices();};
$('frequency').oninput=e=>{prefs.frequency=Number(e.target.value);savePreferences();updateDraftGuard();queueForecast();};
$('fare').oninput=e=>{prefs.fare=Math.round(Number(e.target.value)*100);savePreferences();updateDraftGuard();queueForecast();};
$('apply-service').onclick=()=>command('set_service');$('lease-train').onclick=()=>command('lease_train');$('return-train').onclick=()=>command('return_train');
$('borrow').onclick=()=>command('take_loan',{amount:parseDollars(prefs.loan_amount)});$('repay').onclick=()=>command('repay_loan');$('advance-year').onclick=()=>command('advance_year');

function serviceDirty(){const accepted=frame.services.find(s=>s.corridor_id===prefs.service_corridor);return !accepted||accepted.frequency!==prefs.frequency||accepted.fare!==prefs.fare;}
function updateDraftGuard(){$('service-draft-note').textContent=serviceDirty()?label('draft_not_applied'):label('saved');$('restore-plan').hidden=!serviceDirty();$('advance').disabled=busy||prefs.view==='services'&&serviceDirty();$('advance-year').disabled=$('advance').disabled;}
$('restore-plan').onclick=()=>{loadService(prefs.service_corridor);savePreferences();render();};

function parseDollars(value){
 const s=value.trim();if(!/^(?:[0-9]+|[0-9]{1,3}(?:,[0-9]{3})+)(?:\.[0-9]{0,2})?$/.test(s))return 0;
 const [whole,fraction='']=s.replaceAll(',','').split('.');const cents=BigInt(whole)*100n+BigInt((fraction+'00').slice(0,2));return cents>BigInt(Number.MAX_SAFE_INTEGER)?0:Number(cents);
}
function dollarInput(cents){return (BigInt(cents)/100n).toString()+'.'+(BigInt(cents)%100n).toString().padStart(2,'0');}
function selectedFunding(){
 const co=chosenPair();if(!co||co.existing||frame.projects.some(p=>p.corridor_id===co.id))return null;
 const a=co.alignments.find(a=>a.id===prefs.alignment_id)||co.alignments[0];if(!a)return null;
 const contribution=a.cost-a.grant,available=Math.max(0,frame.player.cash),deficit=Math.max(0,-frame.player.cash);
 return {co,a,contribution,available,deficit,gap:Math.max(0,contribution-available+deficit)};
}
function fundingRows(q){if(!q)return [];const rows=[[label('totalCost'),money(q.a.cost)],[label('grant'),'− '+money(q.a.grant)],[label('contribution'),'= '+money(q.contribution)],[label('fundingCash'),'− '+money(q.available)]];if(q.deficit>0)rows.push([label('fundingDeficit'),'+ '+money(q.deficit)]);rows.push([label('fundingGap'),'= '+money(q.gap)]);return rows;}
function fundingShortfall(){return selectedFunding()?.gap||0;}
function renderLoan(){
 const amount=parseDollars(prefs.loan_amount),credit=frame.debt_limit-frame.debt,q=selectedFunding(),gap=q?.gap||0;
 $('credit-available').textContent=label('creditAvailable')+': '+money(credit);$('borrow').disabled=busy||amount<=0||amount>credit||frame.months_resolved>=frame.campaign_months;
 $('loan-preview').textContent=amount<=0?label('invalid_loan_amount'):amount>credit?label('loan_limit'):label('loanInterest')+': '+money(Math.ceil((frame.debt+amount)/600)-Math.ceil(frame.debt/600));
 $('company-funding').hidden=!q;$('no-funding-plan').hidden=!!q;$('choose-funding-plan').hidden=!!q;
 if(q){$('company-funding-route').textContent=corridorName(q.co)+' · '+label(q.a.id);ledgerRows($('company-funding-values'),fundingRows(q));$('company-funding-note').textContent=(q.deficit>0?label('fundingNegativeCash')+' ':'')+label('fundingFormula');}
 $('shortfall-interest').hidden=gap<=0;$('shortfall-interest').textContent=label('shortfallInterest')+': '+money(Math.ceil((frame.debt+gap)/600)-Math.ceil(frame.debt/600));
 $('borrow-shortfall').hidden=gap<=0;$('borrow-shortfall').textContent=label('borrowShortfall')+' · '+money(gap);$('borrow-shortfall').disabled=busy||gap>credit;
}
$('review-funding-plan').onclick=()=>{const q=selectedFunding();if(q){prefs.alignment_id=q.a.id;showView('build');}};
$('choose-funding-plan').onclick=()=>showView('build');
$('loan-amount').oninput=e=>{prefs.loan_amount=e.target.value;savePreferences();renderLoan();};
async function borrowShortfall(){const amount=fundingShortfall();if(amount<=0)return;prefs.loan_amount=dollarInput(amount);savePreferences();await command('take_loan',{amount});}
$('borrow-shortfall').onclick=borrowShortfall;$('borrow-shortfall-build').onclick=borrowShortfall;

$('terrain-toggle').onclick=async()=>{prefs.hide_terrain=!prefs.hide_terrain;savePreferences();await syncUI();$('terrain-toggle').setAttribute('aria-pressed',String(!prefs.hide_terrain));};
