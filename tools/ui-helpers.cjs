// Follow the same tabs and disclosures a player uses. Assertions can still
// inspect hidden state directly; this helper only prepares an intended action.
async function control(p, selector) {
 const target=p.locator(selector).first();
 if(await target.isVisible())return target;
 const location=await target.evaluate(e=>({view:e.closest('.view')?.id,pane:e.closest('[data-task-section]')?.dataset.pane,task:e.closest('[data-task-section]')?.dataset.taskSection}));
 if(location.view?.startsWith('view-')){
  const view=location.view.slice(5);
  if(!await p.locator('#'+location.view).isVisible())await p.locator('#tab-'+view).click();
  if(location.pane&&!await p.locator('#'+location.task+'-section-'+location.pane).isVisible())await p.locator('#'+location.task+'-pane-'+location.pane).click();
 }
 const closed=await target.evaluate(e=>{const ids=[];for(let n=e.parentElement;n;n=n.parentElement)if(n.tagName==='DETAILS'&&!n.open)ids.unshift(n.id);return ids;});
 for(const id of closed)await p.locator('#'+id+'>summary').click();
 await target.waitFor({state:'visible'});
 return target;
}
async function language(p, value){
 if(await p.locator('html').getAttribute('lang')!==value)await p.locator('#language-quick').click();
 await p.waitForFunction(v=>document.documentElement.lang===v,value);
}
module.exports={control,language};
