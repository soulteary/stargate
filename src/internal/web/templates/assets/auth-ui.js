/* Authentication requests are never retried automatically. */
(function (root) {
  'use strict';
  const MAX_WAIT = 86400;
  function waitSeconds(value) { return Number.isInteger(value) && value > 0 && value <= MAX_WAIT ? value : null; }
  function contextKey(c) { return JSON.stringify([c.phone.trim(), c.mail.trim(), c.channel, c.mode]); }
  function createState(context) {
    return { context: context, revision: 0, challenge: null, sendPending: false, submitPending: false, cooldowns: new Map() };
  }
  function changeContext(state, next) {
    if (state.submitPending) return false;
    if (contextKey(state.context) === contextKey(next)) return false;
    state.context = next; state.revision++; state.challenge = null; return true;
  }
  function beginSend(state) {
    if (state.sendPending || state.submitPending || state.context.mode !== 'delivery') return null;
    state.sendPending = true; return { revision: state.revision, key: contextKey(state.context), context: { ...state.context } };
  }
  function acceptSend(state, ticket, result, now) {
    if (ticket.revision !== state.revision || ticket.key !== contextKey(state.context)) return false;
    if (!result || result.success !== true || typeof result.challenge_id !== 'string' || !result.challenge_id || result.challenge_id.length > 256) throw new Error('ambiguous');
    state.challenge = { id: result.challenge_id, revision: ticket.revision, key: ticket.key };
    const wait = waitSeconds(result.next_resend_in);
    if (wait !== null) state.cooldowns.set(ticket.key, now + wait * 1000);
    return true;
  }
  function beginSubmit(state) {
    if (state.submitPending || state.sendPending) return false;
    state.submitPending = true; return true;
  }
  function reasonFor(result, status) {
    const reasons = { identifier_required: 'identifier_required', identifier_invalid: 'identifier_invalid', verification_required: 'challenge_required', invalid: 'invalid', invalid_code: 'invalid', expired: 'expired', locked: 'locked', too_many_attempts: 'locked', rate_limited: 'rate_limited', resend_cooldown: 'rate_limited', lock_contention: 'service_unavailable', service_unavailable: 'service_unavailable', connection_failed: 'service_unavailable', backend_unavailable: 'service_unavailable', totp_not_enrolled: 'totp_not_enrolled', invalid_enrollment: 'expired', recent_authentication_required: 'authentication_failed', unauthorized: 'authentication_failed', authentication_failed: 'authentication_failed', user_not_in_list: 'authentication_failed' };
    const reason = result && (result.reason || result.error);
    return reasons[reason] || (status === 429 ? 'rate_limited' : status >= 500 ? 'service_unavailable' : 'authentication_failed');
  }
  const api = { waitSeconds, createState, changeContext, beginSend, acceptSend, beginSubmit, contextKey, reasonFor };
  if (typeof module !== 'undefined' && module.exports) module.exports = api;
  if (!root.document) return;
  const document = root.document;
  const config = document.getElementById('auth-ui-config');
  if (!config) return;
  let words;
  try { words = JSON.parse(config.dataset.messages); } catch (_) { return; }
  const text = (key) => words[key] || words.authentication_failed;
  const message = document.getElementById('error-container');
  function announce(key, field) {
    message.textContent = text(key); message.classList.add('show');
    if (field) { field.setAttribute('aria-invalid', 'true'); field.focus(); }
  }
  function clearMessage() { message.classList.remove('show'); message.textContent = ''; document.querySelectorAll('[aria-invalid]').forEach(el => el.removeAttribute('aria-invalid')); }
  async function request(url, data) {
    const target = new URL(url, root.location.href);
    if (target.origin !== root.location.origin) throw new Error('ambiguous');
    target.searchParams.set('lang', config.dataset.locale);
    const controller = new AbortController();
    const timer = root.setTimeout(() => controller.abort(), 45000);
    try {
      const response = await root.fetch(target.href, { method: 'POST', body: data, signal: controller.signal, credentials: 'same-origin', redirect: 'manual', headers: { Accept: 'application/json', 'X-Language': config.dataset.locale } });
      if (response.type === 'opaqueredirect') throw new Error('ambiguous');
      const result = await response.json();
      if (!result || typeof result !== 'object' || Array.isArray(result)) throw new Error('ambiguous');
      return { response, result };
    } finally { root.clearTimeout(timer); }
  }

  function navigate(raw) {
    const url = new URL(raw || '/', root.location.href);
    if (!['http:', 'https:'].includes(url.protocol) || url.username || url.password) throw new Error('ambiguous');
    root.location.assign(url.href);
  }
  function freeze(form, frozen) {
    form.setAttribute('aria-busy', String(frozen));
    form.querySelectorAll('input, select, button').forEach(el => {
      if (frozen) { el.dataset.previousDisabled = String(el.disabled); el.disabled = true; }
      else { el.disabled = el.dataset.previousDisabled === 'true'; delete el.dataset.previousDisabled; }
    });
  }
  const form = document.getElementById('loginForm');
  if (form) {
    const phone = document.getElementById('phone'), mail = document.getElementById('mail');
    const mode = document.getElementById('auth-mode'), channel = document.getElementById('deliver_via');
    const code = document.getElementById('verify_code'), otp = document.getElementById('otp_code'), backup = document.getElementById('recovery_code');
    const send = document.getElementById('sendCodeBtn'), submit = form.querySelector('[type="submit"]');
    const passwordForm = document.getElementById('passwordLoginForm');
    const debug = document.getElementById('debug-code-container');
    const readContext = () => ({ phone: phone.value.trim(), mail: mail.value.trim(), mode: mode.value, channel: channel ? channel.value : '' });
    const state = createState(readContext());
    let navigationStarted = false;
    function render() {
      document.getElementById('delivery-fields').hidden = state.context.mode !== 'delivery';
      document.getElementById('totp-fields').hidden = state.context.mode !== 'totp';
      document.getElementById('recovery-fields').hidden = state.context.mode !== 'recovery';
      if (send) {
        const remaining = Math.max(0, Math.ceil(((state.cooldowns.get(contextKey(state.context)) || 0) - Date.now()) / 1000));
        send.disabled = state.sendPending || state.submitPending || remaining > 0 || !state.context.phone && !state.context.mail;
        send.textContent = state.sendPending ? text('sending') : remaining > 0 ? text('resend').replace('{seconds}', String(remaining)) : text('send');
      }
      submit.disabled = state.submitPending || state.sendPending || !['delivery', 'totp', 'recovery'].includes(state.context.mode);
      submit.textContent = state.submitPending ? text('submitting') : text('signin');
    }
    function changed() {
      if(state.submitPending) { phone.value=state.context.phone;mail.value=state.context.mail;mode.value=state.context.mode;if(channel)channel.value=state.context.channel;return; }
      if (changeContext(state, readContext())) {
        [code, otp, backup].forEach(el => { if (el) el.value = ''; });
        if (debug) { debug.hidden = true; debug.textContent = ''; }
        announce('account_changed');
      }
      render();
    }
    [phone, mail].forEach(el => el.addEventListener('input', changed));
    [mode, channel].filter(Boolean).forEach(el => el.addEventListener('change', changed));
    if (send) send.addEventListener('click', async function () {
      if (!state.context.phone && !state.context.mail) { announce('identifier_required', phone); return; }
      const ticket = beginSend(state); if (!ticket) return;
      clearMessage(); render();
      const data = new FormData();
      if (ticket.context.phone) data.set('phone', ticket.context.phone);
      if (ticket.context.mail) data.set('mail', ticket.context.mail);
      data.set('deliver_via', ticket.context.channel);
      try {
        const { response, result } = await request('/_send_verify_code', data);
        if (ticket.revision !== state.revision) return;
        if (response.ok && result.success === true) {
          if (acceptSend(state, ticket, result, Date.now())) {
            code.value = ''; announce('sent');
            if (config.dataset.debug === 'true' && typeof result.debug_code === 'string' && debug) {
              debug.textContent = text('debug') + ': ' + result.debug_code; debug.hidden = false;
            }
          }
        } else {
          if (response.status !== 429) { state.challenge = null; code.value = ''; }
          const wait = waitSeconds(result.next_resend_in);
          if (response.status === 429 && wait !== null) state.cooldowns.set(ticket.key, Date.now() + wait * 1000);
          announce(reasonFor(result, response.status));
        }
      } catch (_) { if (ticket.revision === state.revision) { state.challenge = null; code.value = ''; announce('unknown_result'); } }
      finally { state.sendPending = false; render(); }
    });
    async function login(event, localPassword) {
      event.preventDefault();
      if (state.submitPending || state.sendPending) return;
      clearMessage();
      const target = localPassword ? passwordForm : form;
      if (!localPassword) {
        if (!state.context.phone && !state.context.mail) { announce('identifier_required', phone); return; }
        const field = state.context.mode === 'delivery' ? code : state.context.mode === 'totp' ? otp : backup;
        if (!field.value.trim()) { announce('code_required', field); return; }
        if (state.context.mode === 'delivery' && (!state.challenge || state.challenge.revision !== state.revision)) { announce('challenge_required', send); return; }
        if (state.context.mode === 'totp' && !/^\d{6}$/.test(field.value.trim())) { announce('format', field); return; }
        if (state.context.mode === 'recovery' && !/^(?:[A-Za-z0-9]{16}|(?:[A-Za-z0-9]{4}-){3}[A-Za-z0-9]{4})$/.test(field.value.trim())) { announce('recovery_hint', field); return; }
      }
      // Capture every successful control before disabling any inputs.
      const data = new FormData(target);
      if (!localPassword) {
        if (state.context.mode === 'delivery') { data.set('challenge_id', state.challenge.id); data.delete('use_otp'); data.delete('otp_code'); }
        else { data.set('use_otp', 'true'); data.set('otp_code', (state.context.mode === 'totp' ? otp : backup).value.trim()); data.delete('challenge_id'); data.delete('verify_code'); }
      }
      if (!beginSubmit(state)) return;
      freeze(form, true); if (passwordForm) freeze(passwordForm, true); render();
      try {
        const { response, result } = await request('/_login', data);
        if (response.ok && result.success === true) { navigate(result.redirect || '/'); navigationStarted = true; return; }
        announce(reasonFor(result, response.status));
      } catch (_) { announce('unknown_result'); }
      finally { if (!navigationStarted) { state.submitPending = false; freeze(form, false); if (passwordForm) freeze(passwordForm, false); render(); } }
    }
    form.addEventListener('submit', event => login(event, false));
    if (passwordForm) passwordForm.addEventListener('submit', event => login(event, true));
    root.setInterval(render, 1000); document.addEventListener('visibilitychange', render);
    root.addEventListener('pageshow', event => { if (event.persisted && navigationStarted) { navigationStarted = false; state.submitPending = false; state.challenge = null; code && (code.value = ''); freeze(form,false); if(passwordForm) freeze(passwordForm,false); } if (!state.submitPending) render(); });
    render();
  }
  const passwordOnly = document.getElementById('passwordOnlyForm');
  if (passwordOnly) {
    let pending = false, navigationStarted = false;
    passwordOnly.addEventListener('submit', async event => {
      event.preventDefault(); if (pending) return;
      const data = new FormData(passwordOnly); pending = true; clearMessage(); freeze(passwordOnly, true);
      try { const {response,result} = await request('/_login', data); if(response.ok && result.success === true) { navigate(result.redirect || '/'); navigationStarted = true; } else announce(reasonFor(result,response.status)); }
      catch (_) { announce('unknown_result'); }
      finally { if (!navigationStarted) { pending = false; freeze(passwordOnly,false); } }
    });
    root.addEventListener('pageshow', event => { if(event.persisted && navigationStarted) { navigationStarted=false;pending=false;freeze(passwordOnly,false); } });
  }
  let managementBusy = false;
  const nativeForms = Array.from(document.querySelectorAll('[data-logout], [data-native-once]'));
  nativeForms.forEach(nativeForm => {
    const action = new URL(nativeForm.action, root.location.href);
    if(action.origin === root.location.origin) { action.searchParams.set('lang',config.dataset.locale);nativeForm.action=action.href; }
    nativeForm.addEventListener('submit', event => {
    if(managementBusy) { event.preventDefault();return; }
    managementBusy=true;nativeForms.forEach(f=>freeze(f,true));
    });
  });
  root.addEventListener('pageshow', event => { if(event.persisted) { managementBusy=false;nativeForms.forEach(f=>freeze(f,false)); } });
  const management = document.getElementById('confirmForm') || document.getElementById('revokeForm');
  if (management) {
    let pending = false;
    let backupCodes = [], sensitiveCleared = false;
    management.addEventListener('submit', async event => {
      event.preventDefault(); if (pending || managementBusy) return; clearMessage();
      const data = new FormData(management);
      const code = management.querySelector('[name="code"]');
      if (management.id === 'confirmForm' && !/^\d{6}$/.test(code.value.trim())) { announce('format', code); return; }
      pending = true; managementBusy=true;nativeForms.forEach(f=>freeze(f,true));freeze(management, true);
      try {
        const { response, result } = await request(management.action, data);
        if (response.ok && result.ok === true) {
          management.hidden = true; management.querySelectorAll('input').forEach(el => { el.value = ''; });
          const qr = document.getElementById('qrcode'); if (qr) { qr.replaceChildren(); qr.removeAttribute('data-uri'); qr.hidden = true; }
          if (management.id === 'confirmForm') {
            if (!Array.isArray(result.backup_codes) || !result.backup_codes.length || result.backup_codes.some(c => typeof c !== 'string')) throw new Error('ambiguous');
            backupCodes = result.backup_codes.slice(); sensitiveCleared = false;
            if (copy) copy.disabled = false; if (download) download.disabled = false;
            if (saved) { saved.disabled = false; saved.checked = false; document.getElementById('savedHome').hidden = true; }
            const hint = document.getElementById('backupHint'); if (hint) hint.textContent = text('backup_hint');
            const list = document.getElementById('backupCodesList'); list.replaceChildren();
            backupCodes.forEach(value => { const li = document.createElement('li'); li.textContent = value; list.appendChild(li); });
            const section = document.getElementById('backupCodes'); section.hidden = false; section.focus(); announce('backup_hint');
          } else announce('success');
        } else announce(reasonFor(result, response.status));
      } catch (_) { announce('unknown_result'); }
      finally { pending = false;managementBusy=false;nativeForms.forEach(f=>freeze(f,false));freeze(management, false); }
    });
    const copy = document.getElementById('copyCodes');
    if (copy) copy.addEventListener('click', async () => { if (!backupCodes.length) return; try { await root.navigator.clipboard.writeText(backupCodes.join('\n')); } catch (_) { announce('copy_failed'); } });
    const download = document.getElementById('downloadCodes');
    if (download) download.addEventListener('click', () => {
      if (!backupCodes.length) return;
      const url = URL.createObjectURL(new Blob([backupCodes.join('\n') + '\n'], { type: 'text/plain;charset=utf-8' }));
      const link = document.createElement('a'); link.href = url; link.download = 'stargate-backup-codes.txt'; link.click(); root.setTimeout(() => URL.revokeObjectURL(url), 0);
    });
    const saved = document.getElementById('codesSaved');
    if (saved) saved.addEventListener('change', () => { document.getElementById('savedHome').hidden = !saved.checked; });
    root.addEventListener('pagehide', () => {
      const qr = document.getElementById('qrcode');
      if (backupCodes.length || qr && qr.dataset.uri) sensitiveCleared = true;
      backupCodes = [];
      const list = document.getElementById('backupCodesList'); if (list) list.replaceChildren();
      if (copy) copy.disabled = true; if (download) download.disabled = true;
      if (saved) { saved.checked = false; saved.disabled = true; document.getElementById('savedHome').hidden = true; }
      if (qr) { qr.replaceChildren(); qr.removeAttribute('data-uri'); }
    });
    root.addEventListener('pageshow', event => {
      if (event.persisted && sensitiveCleared) {
        announce('secret_cleared');
        const hint = document.getElementById('backupHint'); if(hint) hint.textContent = text('secret_cleared');
      }
    });
    const qr = document.getElementById('qrcode');
    if (qr && qr.dataset.uri) { try { if (typeof root.QRCode !== 'function') throw new Error('QR'); new root.QRCode(qr, { text: qr.dataset.uri, width: 200, height: 200 }); } catch (_) { announce('qr_failed'); } }
  }
})(typeof window !== 'undefined' ? window : globalThis);
