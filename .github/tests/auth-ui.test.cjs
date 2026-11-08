'use strict';
const test = require('node:test');
const assert = require('node:assert/strict');
const fs = require('node:fs');
const path = require('node:path');
const vm = require('node:vm');
const script = path.resolve(__dirname, '../../src/internal/web/templates/assets/auth-ui.js');
const ui = require(script);
const context = (mail = 'a@example.test', mode = 'delivery', channel = 'email') => ({ phone: '', mail, mode, channel });
test('account, channel and mode changes invalidate challenge and late send replies', () => {
 for (const next of [context('b@example.test'), context('a@example.test','delivery','sms'),context('a@example.test','totp')]) {
  const state=ui.createState(context());const ticket=ui.beginSend(state);
  assert.equal(ui.changeContext(state,next),true);
  assert.equal(ui.acceptSend(state,ticket,{success:true,challenge_id:'old',next_resend_in:120},100),false);
  assert.equal(state.challenge,null);assert.equal(state.cooldowns.size,0);
 }
});
test('one pending send or login blocks duplicate and competing submissions',()=>{
 const state=ui.createState(context());assert.ok(ui.beginSend(state));assert.equal(ui.beginSend(state),null);assert.equal(ui.beginSubmit(state),false);
 state.sendPending=false;assert.equal(ui.beginSubmit(state),true);assert.equal(ui.beginSubmit(state),false);assert.equal(ui.beginSend(state),null);
 assert.equal(ui.changeContext(state,context('b@example.test')),false);assert.equal(state.context.mail,'a@example.test');
});
test('resend keeps the known challenge while pending and uses only authoritative seconds',()=>{
 for(const seconds of [30,60,120]){const state=ui.createState(context());state.challenge={id:'previous'};const ticket=ui.beginSend(state);assert.equal(state.challenge.id,'previous');assert.equal(ui.acceptSend(state,ticket,{success:true,challenge_id:'new',next_resend_in:seconds},1000),true);assert.equal(state.cooldowns.get(ui.contextKey(context())),1000+seconds*1000);}
 for(const seconds of [null,undefined,0,-1,1.5,'60',Infinity,86401])assert.equal(ui.waitSeconds(seconds),null);
 const state=ui.createState(context());const ticket=ui.beginSend(state);ui.acceptSend(state,ticket,{success:true,challenge_id:'new'},1000);assert.equal(state.cooldowns.size,0);
});
test('cooldown survives returning to account but challenge does not',()=>{
 const state=ui.createState(context());const t=ui.beginSend(state);ui.acceptSend(state,t,{success:true,challenge_id:'new',next_resend_in:120},0);state.sendPending=false;
 ui.changeContext(state,context('b@example.test'));ui.changeContext(state,context());assert.equal(state.challenge,null);assert.equal(state.cooldowns.get(ui.contextKey(context())),120000);
});
test('stable reasons take precedence and natural language is never parsed',()=>{
 assert.equal(ui.reasonFor({reason:'expired',error:'unrelated'},401),'expired');
 assert.equal(ui.reasonFor({error:'rate_limited'},429),'rate_limited');
 assert.equal(ui.reasonFor({error:'expired private user details'},401),'authentication_failed');
 assert.equal(ui.reasonFor({error:'<script>private</script>'},502),'service_unavailable');
});
function fixture(fetch, kind='confirm'){
 const elements={}, windowEvents={}, navigations=[];
 function el(id){return elements[id]={id,hidden:false,disabled:false,dataset:{},value:'123456',checked:false,textContent:'',children:[],events:{},classList:{add(){},remove(){}},setAttribute(){},removeAttribute(){},focus(){},addEventListener(k,fn){this.events[k]=fn;},querySelector(){return elements.code;},querySelectorAll(){return [elements.code,elements.button];},replaceChildren(){this.children=[];},appendChild(child){this.children.push(child);}};}
 const config=el('auth-ui-config');config.dataset={locale:'en',messages:JSON.stringify({authentication_failed:'auth failed',rate_limited:'wait',unknown_result:'unknown',format:'format',secret_cleared:'cleared',backup_hint:'save'})};
 el('error-container');el('code');el('button');const form=el(kind==='confirm'?'confirmForm':'passwordOnlyForm');form.action='https://auth.example/totp/enroll/confirm';el('backupCodes').hidden=true;el('backupCodesList');el('copyCodes');el('downloadCodes');el('codesSaved');el('savedHome');el('backupHint');
 let copied=[];const win={document:{getElementById:id=>elements[id]||null,querySelectorAll:()=>[],createElement:()=>({textContent:''})},location:{href:'https://auth.example/totp/enroll',origin:'https://auth.example',assign:url=>navigations.push(url)},fetch,setTimeout:()=>1,clearTimeout(){},addEventListener:(name,fn)=>{(windowEvents[name] ||= []).push(fn);},navigator:{clipboard:{writeText:async value=>{copied.push(value);}}}};
 vm.runInNewContext(fs.readFileSync(script,'utf8'),{window:win,AbortController,URL,Blob,FormData:class{constructor(){}}});return {elements,form,copied,navigations,dispatch:(name,event)=>{for(const fn of windowEvents[name]||[])fn(event||{});}};
}
test('actual confirmation listener sends once during double submit and restores after 429',async()=>{
 let resolve;let calls=0;const f=fixture(()=>{calls++;return new Promise(r=>{resolve=r;});});const event={preventDefault(){}};
 const first=f.form.events.submit(event);await f.form.events.submit(event);assert.equal(calls,1);assert.equal(f.elements.button.disabled,true);
 resolve({ok:false,status:429,type:'basic',json:async()=>({ok:false,error:'rate_limited'})});await first;
 assert.equal(f.elements['error-container'].textContent,'wait');assert.equal(f.elements.button.disabled,false);assert.equal(f.form.hidden,false);
});
test('ambiguous confirmation is not retried and does not fabricate backup codes',async()=>{
 let calls=0;const f=fixture(async()=>{calls++;throw new Error('connection lost');});await f.form.events.submit({preventDefault(){}});
 assert.equal(calls,1);assert.equal(f.elements['error-container'].textContent,'unknown');assert.equal(f.elements.backupCodes.hidden,true);assert.equal(f.elements.backupCodesList.children.length,0);assert.equal(f.elements.button.disabled,false);
});
test('backup codes render as text and clipboard is used only after explicit click',async()=>{
 const f=fixture(async()=>({ok:true,status:200,type:'basic',json:async()=>({ok:true,backup_codes:['ABCD-EFGH-IJKL-MNOP']})}));
 await f.form.events.submit({preventDefault(){}});assert.equal(f.form.hidden,true);assert.equal(f.copied.length,0);assert.equal(f.elements.backupCodesList.children[0].textContent,'ABCD-EFGH-IJKL-MNOP');
 await f.elements.copyCodes.events.click();assert.deepEqual(f.copied,['ABCD-EFGH-IJKL-MNOP']);
});

test('pagehide and restored management pages never retain actionable empty save controls',async()=>{
 let calls=0;const f=fixture(async()=>{calls++;return {ok:true,status:200,type:'basic',json:async()=>({ok:true,backup_codes:['ABCD-EFGH-IJKL-MNOP']})};});
 await f.form.events.submit({preventDefault(){}});f.elements.codesSaved.checked=true;f.dispatch('pagehide');f.dispatch('pageshow',{persisted:true});
 assert.equal(f.elements.copyCodes.disabled,true);assert.equal(f.elements.downloadCodes.disabled,true);assert.equal(f.elements.codesSaved.checked,false);assert.equal(f.elements.savedHome.hidden,true);assert.equal(f.elements['error-container'].textContent,'cleared');
 await f.elements.copyCodes.events.click();assert.equal(f.copied.length,0);assert.equal(calls,1);
});
test('confirmed password login stays locked while navigation is deferred and resets only on page restore',async()=>{
 let calls=0;const f=fixture(async()=>{calls++;return {ok:true,status:200,type:'basic',json:async()=>({success:true,redirect:'https://app.example/_session_exchange?ticket=synthetic'})};},'password');
 const event={preventDefault(){}};await f.form.events.submit(event);assert.equal(f.elements.button.disabled,true);await f.form.events.submit(event);assert.equal(calls,1);assert.equal(f.navigations.length,1);
 f.dispatch('pageshow',{persisted:true});assert.equal(f.elements.button.disabled,false);assert.equal(calls,1);
});
