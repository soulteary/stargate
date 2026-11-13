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
 const config=el('auth-ui-config');config.dataset={locale:'en',messages:JSON.stringify({sent:'sent',send:'send',sending:'sending',authentication_failed:'auth failed',rate_limited:'wait',unknown_result:'unknown',format:'format',secret_cleared:'cleared',backup_hint:'save'})};
 el('error-container');el('code');el('button');const form=el(kind==='confirm'?'confirmForm':'passwordOnlyForm');form.action='https://auth.example/totp/enroll/confirm';el('backupCodes').hidden=true;el('backupCodesList');el('copyCodes');el('downloadCodes');el('codesSaved');el('savedHome');el('backupHint');el('reauthSendBtn');el('deliver_via').value='email';
 let copied=[];const win={document:{getElementById:id=>elements[id]||null,querySelectorAll:()=>[],createElement:()=>({textContent:''})},location:{href:'https://auth.example/totp/enroll',origin:'https://auth.example',assign:url=>navigations.push(url)},fetch,setTimeout:()=>1,clearTimeout(){},addEventListener:(name,fn)=>{(windowEvents[name] ||= []).push(fn);},navigator:{clipboard:{writeText:async value=>{copied.push(value);}}}};
 vm.runInNewContext(fs.readFileSync(script,'utf8'),{window:win,AbortController,URL,Blob,FormData:class{constructor(){this.fields={};}set(k,v){this.fields[k]=v;}}});return {elements,form,copied,navigations,dispatch:(name,event)=>{for(const fn of windowEvents[name]||[])fn(event||{});}};
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

test('confirmation pins the rendered locale into the request query above any older cookie preference',async()=>{
 let target;const f=fixture(async url=>{target=new URL(url);return {ok:false,status:429,type:'basic',json:async()=>({ok:false,error:'rate_limited'})};});
 await f.form.events.submit({preventDefault(){}});assert.equal(target.origin,'https://auth.example');assert.equal(target.searchParams.get('lang'),'en');
});

test('fresh confirmation after page restore re-enables only the newly received backup codes',async()=>{
 let calls=0;const f=fixture(async()=>{calls++;return {ok:true,status:200,type:'basic',json:async()=>({ok:true,backup_codes:['NEWC-ODES-ONLY-HERE']})};});
 f.dispatch('pagehide');f.dispatch('pageshow',{persisted:true});
 assert.equal(f.elements.copyCodes.disabled,true);assert.equal(f.elements.codesSaved.disabled,true);
 await f.form.events.submit({preventDefault(){}});
 assert.equal(calls,1);assert.equal(f.elements.copyCodes.disabled,false);assert.equal(f.elements.downloadCodes.disabled,false);assert.equal(f.elements.codesSaved.disabled,false);
 assert.equal(f.elements.codesSaved.checked,false);assert.equal(f.elements.savedHome.hidden,true);assert.equal(f.elements.backupHint.textContent,'save');
 await f.elements.copyCodes.events.click();assert.deepEqual(f.copied,['NEWC-ODES-ONLY-HERE']);
 f.elements.codesSaved.checked=true;f.elements.codesSaved.events.change();assert.equal(f.elements.savedHome.hidden,false);
 f.dispatch('pagehide');f.dispatch('pageshow',{persisted:true});assert.equal(f.elements.copyCodes.disabled,true);assert.equal(f.elements.codesSaved.checked,false);assert.equal(f.elements.savedHome.hidden,true);
});
test('re-authentication send posts only the channel, once, and reports the outcome',async()=>{
 let resolve;const calls=[];const f=fixture((url,init)=>{calls.push({url:new URL(url),fields:init.body.fields});return new Promise(r=>{resolve=r;});});
 const button=f.elements.reauthSendBtn;const first=button.events.click();await button.events.click();
 assert.equal(calls.length,1);assert.equal(calls[0].url.pathname,'/_send_reauth_code');assert.deepEqual(calls[0].fields,{deliver_via:'email'});assert.equal(button.disabled,true);
 resolve({ok:true,status:200,type:'basic',json:async()=>({success:true,challenge_id:'c1'})});await first;
 assert.equal(f.elements['error-container'].textContent,'sent');assert.equal(button.disabled,false);assert.equal(button.textContent,'send');
 const second=button.events.click();resolve({ok:false,status:429,type:'basic',json:async()=>({success:false,reason:'rate_limited'})});await second;
 assert.equal(f.elements['error-container'].textContent,'wait');assert.equal(calls.length,2);
});
test('a send ticket carries the challenge held for the same context and nothing otherwise',()=>{
 const state=ui.createState(context());const first=ui.beginSend(state);assert.equal(first.previous,null);
 ui.acceptSend(state,first,{success:true,challenge_id:'held'},0);state.sendPending=false;
 assert.equal(ui.beginSend(state).previous,'held');state.sendPending=false;
 assert.equal(ui.changeContext(state,context('b@example.test')),true);assert.equal(ui.beginSend(state).previous,null);state.sendPending=false;
 for(const stale of [{id:'other',revision:state.revision,key:ui.contextKey(context())},{id:'older',revision:state.revision-1,key:ui.contextKey(state.context)},{id:'bare'}]){
  state.challenge=stale;assert.equal(ui.beginSend(state).previous,null);state.sendPending=false;
 }
});
function loginFixture(fetch,dataset={}){
 const elements={};
 function el(id,value=''){return elements[id]={id,hidden:false,disabled:false,dataset:{},value,textContent:'',events:{},classList:{add(){},remove(){}},setAttribute(){},removeAttribute(){},focus(){},addEventListener(k,fn){this.events[k]=fn;},querySelector(){return elements.submit;},querySelectorAll(){return [];}};}
 el('auth-ui-config').dataset={locale:'en',messages:JSON.stringify({sent:'sent',send:'send',sending:'sending',resend:'wait {seconds}',signin:'signin',submitting:'submitting',authentication_failed:'auth failed',rate_limited:'wait',account_changed:'changed',unknown_result:'unknown',identifier_hint:'one identifier',code_required:'code required'}),...dataset};
 el('error-container');el('loginForm');el('phone');el('mail','a@example.test');el('auth-mode','delivery');el('deliver_via','email');el('verify_code');el('otp_code');el('recovery_code');el('sendCodeBtn');el('submit');el('delivery-fields');el('totp-fields');el('recovery-fields');
 const win={document:{getElementById:id=>elements[id]||null,querySelectorAll:()=>[],addEventListener(){}},location:{href:'https://auth.example/_login',origin:'https://auth.example',assign(){}},fetch,setTimeout:()=>1,clearTimeout(){},setInterval(){},addEventListener(){}};
 vm.runInNewContext(fs.readFileSync(script,'utf8'),{window:win,AbortController,URL,FormData:class{constructor(){this.fields={};}set(k,v){this.fields[k]=v;}}});return elements;
}
test('the send request presents the held challenge on a resend, also after a refused one, and never for another account',async()=>{
 const replies=[{ok:true,status:200,json:{success:true,challenge_id:'h1'}},{ok:false,status:429,json:{success:false,reason:'rate_limited',next_resend_in:30}},{ok:true,status:200,json:{success:true,challenge_id:'h2'}},{ok:true,status:200,json:{success:true,challenge_id:'h3'}},{ok:true,status:200,json:{success:true,challenge_id:'h4'}}];
 const sent=[];const elements=loginFixture(async(url,init)=>{const reply=replies[sent.length];sent.push({path:new URL(url).pathname,fields:init.body.fields});return {ok:reply.ok,status:reply.status,type:'basic',json:async()=>reply.json};});
 const send=elements.sendCodeBtn;const account={mail:'a@example.test',deliver_via:'email'};
 await send.events.click();await send.events.click();await send.events.click();await send.events.click();
 assert.deepEqual(sent.map(call=>call.path),Array(4).fill('/_send_verify_code'));
 assert.deepEqual(sent[0].fields,account);assert.deepEqual(sent[1].fields,{...account,challenge_id:'h1'});
 assert.deepEqual(sent[2].fields,{...account,challenge_id:'h1'});assert.deepEqual(sent[3].fields,{...account,challenge_id:'h2'});
 elements.mail.value='b@example.test';elements.mail.events.input();await send.events.click();
 assert.deepEqual(sent[4].fields,{mail:'b@example.test',deliver_via:'email'});
});
test('with the single-identifier flag the page asks for one identifier and sends nothing while both are filled in',async()=>{
 const sent=[];const fetch=async(url,init)=>{sent.push({path:new URL(url).pathname,fields:init.body.fields});return {ok:true,status:200,type:'basic',json:async()=>({success:true,challenge_id:'h1'})};};
 const both=(elements)=>{elements.phone.value='13800000001';elements.phone.events.input();return elements;};
 const shown=(elements)=>elements['error-container'].textContent;
 const submit=(elements)=>elements.loginForm.events.submit({preventDefault(){}});
 let elements=both(loginFixture(fetch,{singleIdentifier:'true'}));
 await elements.sendCodeBtn.events.click();assert.equal(sent.length,0);assert.equal(shown(elements),'one identifier');
 elements['error-container'].textContent='';await submit(elements);assert.equal(sent.length,0);assert.equal(shown(elements),'one identifier');
 elements.mail.value='';elements.mail.events.input();await elements.sendCodeBtn.events.click();
 assert.deepEqual(sent.map(call=>call.fields),[{phone:'13800000001',deliver_via:'email'}]);
 for(const dataset of [{singleIdentifier:'false'},{}]){
  sent.length=0;elements=both(loginFixture(fetch,dataset));
  await submit(elements);assert.equal(shown(elements),'code required');
  await elements.sendCodeBtn.events.click();
  assert.deepEqual(sent.map(call=>call.fields),[{phone:'13800000001',mail:'a@example.test',deliver_via:'email'}]);
 }
});
