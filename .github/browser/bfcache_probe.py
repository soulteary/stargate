"""One controlled BFCache experiment; product code and response headers stay intact."""
import base64
import hashlib
import hmac
import json
from pathlib import Path
import struct
import time
from urllib.parse import parse_qs, urlparse

import production_tls as fixture
from validate_inputs import callback_matches

IGNORED_DEFAULTS = ['--disable-back-forward-cache', '--disable-prompt-on-repost']
REASONS = {'BackForwardCacheDisabled', 'HTTPMethodNotGET', 'HTTPStatusNotOK', 'ErrorDocument',
           'SchemeNotHTTPOrHTTPS', 'Loading', 'Timeout', 'JavaScriptExecution', 'CacheLimit',
           'CacheControlNoStore', 'CacheControlNoStoreCookieModified', 'CacheControlNoStoreHTTPOnlyCookieModified',
           'CacheControlNoStoreDeviceBoundSessionTerminated', 'JsNetworkRequestReceivedCacheControlNoStoreResource',
           'MainResourceHasCacheControlNoStore', 'MainResourceHasCacheControlNoCache', 'RelatedActiveContentsExist',
           'BrowsingInstanceNotSwapped', 'RendererProcessKilled', 'RendererProcessCrashed', 'CookieDisabled',
           'CookieFlushed', 'WebSocket', 'WebTransport', 'WebRTC', 'OutstandingNetworkRequestFetch',
           'OutstandingNetworkRequestXHR', 'OutstandingNetworkRequestOthers', 'SessionRestored',
           'ForwardCacheDisabled', 'NavigationCancelledWhileRestoring'}
BASE_SAFE_FAILURE = fixture.safe_failure
STAMP_SCRIPT = """window.__u11bf = {document:crypto.randomUUID(), shows:[]};
addEventListener('pageshow', e => window.__u11bf.shows.push({persisted:e.persisted, trusted:e.isTrusted}));"""


def exact_cdp_body_limit(error, playwright_error_type):
    if not isinstance(error, playwright_error_type):
        return False
    message = str(error)
    return 'Network.getResponseBody' in message and any(marker in message for marker in (
        'No resource with given identifier found', 'No data found for resource with given identifier'))


def safe_experiment_failure(error):
    result = BASE_SAFE_FAILURE(error)
    name = type(error).__name__
    result['exception_type'] = name if name in {'Error','TimeoutError','JSONDecodeError','HTTPError','URLError','KeyError','RuntimeError','AssertionError','TypeError','ValueError'} else 'other'
    if isinstance(error, json.JSONDecodeError):
        result['reason'] = 'json_decode_failed'
    elif 'Network.getResponseBody' in str(error):
        result['reason'] = 'cdp_response_body_read_failed'
    else:
        # Fixed browser error codes only; never preserve the message or URL.
        codes = ('ERR_CONNECTION_REFUSED', 'ERR_CONNECTION_RESET', 'ERR_CONNECTION_CLOSED',
                 'ERR_NAME_NOT_RESOLVED', 'ERR_CERT_AUTHORITY_INVALID', 'ERR_CERT_COMMON_NAME_INVALID',
                 'ERR_CERT_DATE_INVALID', 'ERR_ABORTED', 'ERR_EMPTY_RESPONSE', 'ERR_CACHE_MISS',
                 'ERR_TOO_MANY_REDIRECTS', 'ERR_HTTP_RESPONSE_CODE_FAILURE')
        matched = [code for code in codes if 'net::'+code in str(error)]
        if matched:
            result['browser_error_codes'] = matched
        if 'interrupted by another navigation' in str(error):
            result['navigation_interrupted'] = True
        if 'Target page, context or browser has been closed' in str(error):
            result['target_closed'] = True
    return result


def approved_login_entry(login_url, callbacks):
    return fixture.url_matches(login_url, fixture.AUTH+'/_login') and bool(callbacks) and all(callback_matches(value, fixture.APP) for value in callbacks)


def classify_back_landing(value):
    if fixture.url_matches(value, fixture.APP):
        return 'approved_app'
    if fixture.url_matches(value, fixture.AUTH+'/_login'):
        return 'auth_login'
    return 'other'


def accepted_login(status, body):
    return status == 200 and isinstance(body, dict) and body.get('success') is True


def safe_reasons(events):
    seen, unknown = set(), 0
    for event in events:
        for explanation in event.get('notRestoredExplanations', []):
            code = explanation.get('reason')
            if code in REASONS:
                seen.add(code)
            else:
                unknown += 1
    return {'codes': sorted(seen), 'unknown_count': unknown}


def field_presence(page):
    return page.evaluate("""() => Object.fromEntries(['mail','phone','verify_code','otp_code','recovery_code','local_password'].map(id => {
      const el=document.getElementById(id); return [id, el ? {present:true, nonempty:el.value.length>0,type:el.type,autocomplete:el.autocomplete} : {present:false}];
    }).concat([['send_button', document.querySelector('#sendCodeBtn') ? {present:true,disabled:document.querySelector('#sendCodeBtn').disabled} : {present:false}]]))""")


def totp(uri):
    query = parse_qs(urlparse(uri).query)
    if query.get('algorithm', ['SHA1'])[0] != 'SHA1' or query.get('period', ['30'])[0] != '30' or query.get('digits', ['6'])[0] != '6':
        raise AssertionError('Unexpected pinned TOTP parameters')
    secret = query['secret'][0]
    key = base64.b32decode(secret + '=' * (-len(secret) % 8))
    digest = hmac.new(key, struct.pack('>Q', int(time.time())//30), hashlib.sha1).digest()
    offset = digest[-1] & 15
    return str((struct.unpack('>I', digest[offset:offset+4])[0] & 0x7fffffff) % 1000000).zfill(6)


def genuine_restore(same_document, persisted, new_main_requests):
    return same_document is True and persisted is True and type(new_main_requests) is int and new_main_requests == 0


def restore(page, cdp, events, requests, name, original_stamp, entry_id, required, expected_path):
    events.clear()
    before = len(requests)
    cdp.send('Page.navigateToHistoryEntry', {'entryId': entry_id})
    deadline = time.monotonic() + 8
    observed = None
    while time.monotonic() < deadline:
        try:
            state = page.evaluate('window.__u11bf || null')
            if state and state['shows'] and urlparse(page.url).hostname == fixture.AUTH_HOST and urlparse(page.url).path == expected_path:
                observed = state
                break
        except Exception:
            pass  # Context replacement during navigation; no request or business retry.
        page.wait_for_timeout(100)
    same = bool(observed and observed['document'] == original_stamp)
    persisted = bool(observed and any(x['persisted'] and x['trusted'] for x in observed['shows']))
    result = {'same_document': same, 'trusted_pageshow_persisted': persisted,
              'new_main_requests': len(requests)-before, 'not_restored': safe_reasons(events)}
    result['status'] = 'restored' if genuine_restore(same, persisted, result['new_main_requests']) else 'not_restored'
    fixture.RESULT.setdefault('bfcache_cases', {})[name] = result
    fixture.save()
    if required:
        fixture.check(name+' genuinely restored', result['status'] == 'restored')
    return result


def accepted_account_observation(result):
    if result.get('status') == 'restored':
        return genuine_restore(result.get('same_document'), result.get('trusted_pageshow_persisted'), result.get('new_main_requests'))
    return (result.get('status') == 'not_restored' and
            result.get('not_restored') == {'codes':['CacheControlNoStoreHTTPOnlyCookieModified'], 'unknown_count':0})


def round_trip(page, cdp, events, requests, name, required=True):
    stamp = page.evaluate('window.__u11bf.document')
    original_path = urlparse(page.url).path
    history = cdp.send('Page.getNavigationHistory')
    entry_id = history['entries'][history['currentIndex']]['id']
    cookies = page.context.cookies()
    health = page.goto(fixture.AUTH+'/healthz')
    fixture.check(name+' inert health destination', health.status == 200 and 'set-cookie' not in health.headers)
    cookie_stable = cookies == page.context.cookies()  # Values compared only in memory, never hashed or returned.
    fixture.RESULT.setdefault('away_cookie_observations', {})[name] = {'unchanged':cookie_stable}
    fixture.save()
    result = restore(page, cdp, events, requests, name, stamp, entry_id, required, original_path)
    result['cookies_unchanged_during_away'] = cookie_stable
    fixture.save()
    return result


def browser_scenario(ca, home):
    from playwright.sync_api import sync_playwright, Error as PlaywrightError
    fixture.CA_SYSTEM_PATH = Path('/usr/local/share/ca-certificates')/(fixture.PROJECT+'.crt')
    fixture.check('Runner CA destination new', not fixture.CA_SYSTEM_PATH.exists())
    fixture.command(['sudo','install','-m','0644',ca,fixture.CA_SYSTEM_PATH])
    fixture.CA_SYSTEM_INSTALLED = True
    fixture.command(['sudo','update-ca-certificates'])
    nss = home/'.pki/nssdb'
    nss.mkdir(parents=True, mode=0o700)
    fixture.command(['certutil','-N','--empty-password','-d','sql:'+str(nss)])
    fixture.command(['certutil','-A','-d','sql:'+str(nss),'-n',fixture.PROJECT,'-t','C,,','-i',ca])
    with fixture.private_browser_environment(), sync_playwright() as pw:
        browser = pw.chromium.launch(channel='chromium', ignore_default_args=IGNORED_DEFAULTS,
            env=fixture.child_env({'HOME':str(home)}), args=['--host-resolver-rules=MAP '+fixture.AUTH_HOST+' 127.0.0.1, MAP '+fixture.APP_HOST+' 127.0.0.1'])
        arguments = browser.new_browser_cdp_session().send('Browser.getBrowserCommandLine')['arguments']
        fixture.RESULT['browser'] = {'version':browser.version, 'channel':'chromium', 'headless':True,
            'ignored_automation_defaults':IGNORED_DEFAULTS, 'bfcache_disable_flag_present':'--disable-back-forward-cache' in arguments,
            'repost_prompt_disable_flag_present':'--disable-prompt-on-repost' in arguments, 'certificate_bypass':False, 'certificate_ignore_flag_present':any(arg.startswith('--ignore-certificate-errors') or arg=='--allow-insecure-localhost' for arg in arguments), 'forced_cache_feature_present':any(arg.startswith('--enable-features=') and ('BackForwardCache' in arg or 'CacheControlNoStore' in arg) for arg in arguments)}
        fixture.check('Normal BFCache and repost policy restored', not any(flag in arguments for flag in IGNORED_DEFAULTS))
        fixture.check('No TLS bypass or forced cache feature', not fixture.RESULT['browser']['certificate_ignore_flag_present'] and not fixture.RESULT['browser']['forced_cache_feature_present'])
        context = browser.new_context(ignore_https_errors=False)
        context.add_init_script(STAMP_SCRIPT)
        page = context.new_page()
        page.set_default_timeout(15000)
        requests, events = [], []
        def request_seen(request):
            if request.is_navigation_request() and request.frame == page.main_frame:
                requests.append(request.method)  # Only counts/methods survive; URLs and bodies never saved.
        page.on('request', request_seen)
        cdp = context.new_cdp_session(page)
        cdp.send('Page.enable')
        cdp.on('Page.backForwardCacheNotUsed', lambda event: events.append(event))
        fixture.stage('get-login-bfcache-calibration')
        fixture.stage('navigate-protected-app-to-login')
        response = page.goto(fixture.APP)
        fixture.RESULT['initial_navigation'] = {'status':response.status if response else None, 'landing':classify_back_landing(page.url)}
        fixture.save()
        fixture.stage('wait-actual-login-form')
        page.locator('#loginForm').wait_for()
        fixture.stage('inspect-actual-login-entry')
        callbacks = page.locator('input[name="callback"]').evaluate_all('els => els.map(el => el.value).filter(Boolean)')
        fixture.check('Actual app entry retains approved callback', approved_login_entry(page.url, callbacks))
        fixture.check('Actual GET login keeps no-store', response.status == 200 and 'no-store' in response.headers.get('cache-control','') and response.request.method == 'GET')
        page.locator('#auth-mode').select_option('totp')
        page.locator('#otp_code').fill('123456')
        fixture.RESULT['unsubmitted_field_probe'] = {'kind':'synthetic input, never submitted', 'before':field_presence(page)}
        round_trip(page, cdp, events, requests, 'login_get',)
        fixture.RESULT['unsubmitted_field_probe']['after'] = field_presence(page)
        page.locator('#otp_code').fill('')
        page.locator('#auth-mode').select_option('delivery')
        page.locator('#mail').fill('production@example.test')
        page.locator('#deliver_via').select_option('email')
        fixture.stage('one-real-mail-login')
        with page.expect_response(lambda r: urlparse(r.url).path=='/_send_verify_code') as sent:
            page.locator('#sendCodeBtn').click()
        fixture.check('Actual mail challenge accepted', sent.value.status == 200)
        fixture.stage('read-captured-mail-code')
        code = fixture.mailbox_code()
        fixture.stage('fill-delivered-code')
        page.locator('#verify_code').fill(code)
        fixture.stage('read-login-history-before-submit')
        login_stamp = page.evaluate('window.__u11bf.document')
        history = cdp.send('Page.getNavigationHistory')
        login_entry = history['entries'][history['currentIndex']]['id']
        fixture.stage('submit-single-real-login')
        with page.expect_response(lambda r:r.request.method=='POST' and urlparse(r.url).path=='/_login') as logged_in:
            page.locator('#loginForm [type=submit]').click()
        fixture.RESULT['mail_login_observation'] = {'status':logged_in.value.status, 'success':None, 'body_limit':False}
        fixture.save()
        fixture.check('Single real login POST status200', logged_in.value.status == 200)
        fixture.stage('read-login-response-json')
        try:
            body = logged_in.value.json()
        except PlaywrightError as error:
            if not exact_cdp_body_limit(error, PlaywrightError):
                raise
            fixture.RESULT['mail_login_observation']['body_limit'] = True
            fixture.RESULT['mail_login_observation']['body_limit_kind'] = 'exact_cdp_resource_body_unavailable'
        else:
            fixture.RESULT['mail_login_observation']['success'] = isinstance(body,dict) and body.get('success') is True
            fixture.check('Single real login JSON success', accepted_login(logged_in.value.status, body))
        fixture.stage('wait-for-approved-app')
        page.wait_for_url(lambda value: fixture.url_matches(value, fixture.APP))
        fixture.check('Actual mail authentication reaches app', fixture.url_matches(page.url, fixture.APP))
        fixture.stage('actual-protected-app-contract')
        app = page.evaluate("""async()=>{const r=await fetch('/?format=text',{headers:{Accept:'text/plain'},redirect:'manual'});const text=await r.text();return {status:r.status,project:r.headers.get('Project')==='https://github.com/soulteary/hello',summary:text.split(String.fromCharCode(10)).includes('Hello from soulteary/hello!')};}""")
        fixture.RESULT['protected_app_positive'] = app
        fixture.check('Same page protected app200 and real hello contract', app=={'status':200,'project':True,'summary':True})
        fixture.stage('successful-login-history-observation')
        restore(page, cdp, events, requests, 'after_successful_login', login_stamp, login_entry, False, '/_login')
        landing = classify_back_landing(page.url)
        fixture.RESULT['successful_login_back_landing'] = landing
        fixture.check('Back ends at an approved product location', landing != 'other')
        fixture.RESULT['successful_login_field_presence'] = field_presence(page) if landing == 'auth_login' else {'observed':False,'reason':'not_login_document'}
        fixture.stage('authenticated-account-bfcache')
        response = page.goto(fixture.AUTH+'/_account?lang=en')
        fixture.check('Actual account keeps no-store', response.status==200 and 'no-store' in response.headers.get('cache-control',''))
        account = round_trip(page, cdp, events, requests, 'account_get', required=False)
        fixture.check('Account restores or has only explicit cookie-privacy exclusion', accepted_account_observation(account))
        fixture.check('Account remains at approved management document', fixture.url_matches(page.url, fixture.AUTH+'/_account'))
        account['coverage'] = 'genuine_restore' if account['status']=='restored' else 'not_exercised_cookie_privacy_eviction'
        fixture.save()
        auth = page.evaluate("""async()=>{const r=await fetch('/_auth',{headers:{'X-Forwarded-Uri':'/profile'}});return {status:r.status,user:r.headers.get('X-Auth-User')};}""")
        fixture.check('Actual account retains real identity', auth=={'status':200,'user':'production-user-001'})
        fixture.RESULT['mail_login_observation']['real_app_and_identity_positive'] = True
        fixture.save()
        fixture.stage('native-post-secret-leave')
        page.goto(fixture.AUTH+'/totp/enroll?lang=en')
        with page.expect_response(lambda r:r.request.method=='POST' and urlparse(r.url).path=='/totp/enroll') as started:
            page.locator('[data-native-once] button').click()
        fixture.check('Native enrollment document is actual POST no-store', started.value.status==200 and 'no-store' in started.value.headers.get('cache-control',''))
        uri = page.locator('#qrcode').get_attribute('data-uri')
        page.locator('#code').fill(totp(uri))
        page.locator('#confirmForm [type=submit]').click()
        page.locator('#backupCodes:not([hidden])').wait_for()
        fixture.check('Real confirmation displays ten backup entries', page.locator('#backupCodesList li').count()==10)
        hidden = []
        def cleanup_observed(message):
            if message.type=='debug' and message.text.startswith('__U11_PAGEHIDE__'):
                data=json.loads(message.text[len('__U11_PAGEHIDE__'):])
                if set(data)=={'trusted','persisted','codes_empty','qr_empty','copy_disabled','download_disabled','ack_clear'} and all(type(v) is bool for v in data.values()):
                    hidden.append(data)
        page.on('console', cleanup_observed)
        # Registered after product defer initialization and confirmation, so product cleanup runs first.
        page.evaluate("""() => addEventListener('pagehide', e => console.debug('__U11_PAGEHIDE__'+JSON.stringify({trusted:e.isTrusted,persisted:e.persisted,codes_empty:document.querySelectorAll('#backupCodesList li').length===0,qr_empty:!document.querySelector('#qrcode')?.dataset.uri&&document.querySelector('#qrcode').childNodes.length===0,copy_disabled:document.querySelector('#copyCodes').disabled,download_disabled:document.querySelector('#downloadCodes').disabled,ack_clear:!document.querySelector('#codesSaved').checked&&document.querySelector('#codesSaved').disabled})))""")
        page.goto(fixture.AUTH+'/healthz')
        page.wait_for_timeout(200)
        fixture.RESULT['native_post_leave']={'main_method':'POST','back_attempted':False,'chromium_source_exclusion':'HTTPMethodNotGET','exclusion_kind':'source-based rule, not an observed CDP reason','pagehide':hidden}
        fixture.check('Leaving actual backup page clears rendered secrets', bool(hidden) and hidden[-1]['trusted'] and all(hidden[-1][key] for key in ('codes_empty','qr_empty','copy_disabled','download_disabled','ack_clear')))
        context.close()
        browser.close()


def main():
    fixture.RESULT['scope']='Controlled real BFCache experiment; product no-store/TLS/auth unchanged; no POST history resubmission'
    fixture.save=lambda:Path('bfcache-results.json').write_text(json.dumps(fixture.RESULT,indent=2)+'\n')
    fixture.safe_failure=safe_experiment_failure
    fixture.trust_and_browser=browser_scenario  # Replace only the browser scenario; reuse fixture setup and owned cleanup unchanged.
    return fixture.main()


if __name__=='__main__':
    raise SystemExit(main())
