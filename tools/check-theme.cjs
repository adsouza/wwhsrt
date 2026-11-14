const {control,language}=require('./ui-helpers.cjs');
// Appearance acceptance against actual Go/WASM and semantic controls, in isolated storage.
const assert=require('node:assert/strict'),fs=require('node:fs'),path=require('node:path'),crypto=require('node:crypto');
const {chromium}=require('playwright');
const url=process.argv[2]||'http://127.0.0.1:8765/',out=path.resolve('docs/validation');
const snap=p=>p.evaluate(()=>railSnapshot());
async function theme(p,dark){await p.waitForFunction(async d=>document.documentElement.dataset.theme===(d?'dark':'light')&&(await railSnapshot()).state.dark===d,dark);}
async function choice(p,mode,dark){await (await control(p,'#appearance')).selectOption(mode);await theme(p,dark);}
async function command(p,id){const r=(await snap(p)).frame.revision;await (await control(p,'#'+id)).click();await p.waitForFunction(async r=>(await railSnapshot()).frame.revision===r+1,r);await p.waitForFunction(()=>document.getElementById('game').getAttribute('aria-busy')==='false');}
(async()=>{
 const browser=await chromium.launch({headless:true,args:['--enable-unsafe-swiftshader']});const watchdog=setTimeout(()=>browser.close(),180000);
 const result={date:new Date().toISOString(),browser:browser.version(),wasm_sha256:crypto.createHash('sha256').update(fs.readFileSync('web/main.wasm')).digest('hex'),ui_sha256:crypto.createHash('sha256').update(['web/app.js','web/style.css','web/index.html','web/map.html'].map(f=>fs.readFileSync(f)).reduce((a,b)=>Buffer.concat([a,b]),Buffer.alloc(0))).digest('hex'),checks:[],limitations:['Browser inputs and OS appearance are emulated; physical phones and native OS theme transitions are not qualified here.']};
 try{
  const ctx=await browser.newContext({viewport:{width:1280,height:800},colorScheme:'dark',reducedMotion:'reduce'}),p=await ctx.newPage(),errors=[];p.on('pageerror',e=>errors.push(e.message));
  await p.goto(url);await p.waitForFunction(()=>!!window.railSnapshot,null,{timeout:60000});await theme(p,true);
  const checkpoint=(await snap(p)).frame.checkpoint;
  const darkMap=await (await control(p,'#map-frame')).screenshot();
  await p.emulateMedia({colorScheme:'light'});await theme(p,false);
  const lightMap=await (await control(p,'#map-frame')).screenshot();assert(!darkMap.equals(lightMap),'map ignored appearance');
  await (await control(p,'#tab-company')).click();assert.equal(await p.locator('#appearance').inputValue(),'system');
  await choice(p,'dark',true);await p.emulateMedia({colorScheme:'light'});await theme(p,true);
  await choice(p,'light',false);await p.emulateMedia({colorScheme:'dark'});await theme(p,false);
  await choice(p,'system',true);assert.equal((await snap(p)).frame.checkpoint,checkpoint);
  const colors=(await snap(p)).palette;
  assert.equal(await p.locator('header').evaluate(e=>getComputedStyle(e).backgroundColor),`rgb(${colors.surface.R}, ${colors.surface.G}, ${colors.surface.B})`);
  assert.equal(await p.locator('#loan-amount').evaluate(e=>getComputedStyle(e).backgroundColor),`rgb(${colors.control.R}, ${colors.control.G}, ${colors.control.B})`);
  result.checks.push('System default follows live dark/light browser changes; explicit overrides take precedence; semantic UI uses shared Go palette; actual map pixels change; theme-only interactions preserve campaign checkpoint');
  await command(p,'advance'); // A populated report, with actual revenue.
  await (await control(p,'#tab-services')).click();await (await control(p,'#frequency')).fill('0');await p.waitForFunction(()=>!document.getElementById('apply-service').disabled);await command(p,'apply-service');assert(await p.locator('#idle-warning').isVisible());
  await p.screenshot({path:path.join(out,'theme-dark-services-desktop.png')});
  await (await control(p,'#tab-build')).click();await (await control(p,'[data-corridor=hokkaido-extension]')).click();await (await control(p,'#focus-route')).click();
  await p.screenshot({path:path.join(out,'theme-dark-terrain-desktop.png')});
  await (await control(p,'#tab-company')).click();await choice(p,'dark',true);await (await control(p,'#company-name')).fill('Appearance draft');
  await p.reload();await p.waitForFunction(()=>!!window.railSnapshot,null,{timeout:60000});await theme(p,true);assert.equal(await p.locator('#appearance').inputValue(),'dark');assert.equal(await p.locator('#company-name').inputValue(),'Appearance draft');
  await language(p,'zh-Hans');await p.waitForFunction(()=>document.documentElement.lang==='zh-Hans');
  await p.setViewportSize({width:320,height:740});
  for(const view of ['company','build','services','report','map']){await p.locator('#tab-'+view).click();assert(await p.evaluate(()=>document.documentElement.scrollWidth<=innerWidth),view+' overflowed');}
  await (await control(p,'#tab-report')).click();await p.screenshot({path:path.join(out,'theme-dark-report-phone-zh.png')});
  await (await control(p,'#tab-company')).click();await (await control(p,'#new-game')).click();assert(await p.locator('#new-game-dialog').isVisible());await p.screenshot({path:path.join(out,'theme-dark-dialog-phone-zh.png')});await (await control(p,'#cancel-new-game')).click();
  await (await control(p,'#tab-build')).click();await (await control(p,'#browse-prospects')).click();assert(await p.locator('#inspector').isHidden());await theme(p,true);await p.keyboard.press('Escape');
  result.checks.push('Dark services, idle-cost warning, terrain route planning, finance forms, populated report, confirmation and expanded map; all five views fit 320px Simplified Chinese; preference and company draft survive reload');
  const before=(await snap(p)).frame.checkpoint;
  await p.evaluate(()=>navigator.serviceWorker.ready);await p.waitForFunction(()=>!!navigator.serviceWorker.controller);await ctx.setOffline(true);await p.reload();await p.waitForFunction(()=>!!window.railSnapshot,null,{timeout:60000});await theme(p,true);assert.equal((await snap(p)).frame.checkpoint,before);
  await (await control(p,'#tab-company')).click();await choice(p,'light',false);await choice(p,'dark',true);assert.equal((await snap(p)).frame.checkpoint,before);
  await (await control(p,'#new-game')).click();await command(p,'confirm-new-game');await theme(p,true);assert.equal((await snap(p)).state.appearance,'dark');assert.equal((await snap(p)).frame.months_resolved,0);
  assert.deepEqual(errors,[]);result.checks.push('Cached offline launch retains dark choice; offline switching does not mutate campaign; New game retains selected appearance');
  result.result='passed';fs.writeFileSync(path.join(out,'theme-checks.json'),JSON.stringify(result,null,2)+'\n');console.log(JSON.stringify(result,null,2));await ctx.close();
 }finally{clearTimeout(watchdog);await browser.close();}
})().catch(e=>{console.error(e);process.exitCode=1;});
