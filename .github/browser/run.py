"""Real Chromium acceptance; never persist browser state, messages or credentials."""
import base64
import hashlib
import hmac
import json
import os
from pathlib import Path
import re
import secrets
import struct
import subprocess
import time
import urllib.request
from urllib.parse import parse_qs, urlparse

from playwright.sync_api import sync_playwright
from validate_inputs import validate, callback_matches, hello_text_url
from diagnostics import smtp_signatures

ROOT = Path.cwd()
STATE = Path(os.environ["RUNNER_TEMP"]) / ("browser-" + secrets.token_hex(6))
PROJECT = "browser-" + secrets.token_hex(6)
AUTH = "http://auth.test.localhost:8088"
APP = "http://hello.test.localhost:8088"
RESULT = {"stage": "preflight", "status": "running", "scope": "Chromium local HTTP, real SMTP capture and HMAC service chain; no production TLS or device coverage", "checks": []}
OPENER = urllib.request.build_opener(urllib.request.ProxyHandler({}))


def check(name, condition):
    if not condition:
        RESULT["failed_check"] = name
        save()
        raise AssertionError(name)
    RESULT["checks"].append({"name": name, "status": "passed"})
    save()


def save():
    Path("browser-results.json").write_text(json.dumps(RESULT, indent=2) + "\n")


def stage(name):
    print("Acceptance stage: " + name, flush=True)
    RESULT["stage"] = name
    save()


def health_summary():
    try:
        data = compose("ps", "--all", "--format", "json").decode()
        entries = json.loads(data) if data.lstrip().startswith("[") else [json.loads(line) for line in data.splitlines() if line]
        return [{key.lower(): item.get(key, "") for key in ("Service", "State", "Health", "ExitCode")} for item in entries]
    except Exception:
        return [{"status": "unavailable"}]


def command(args, cwd=None, env=None):
    # Docker/configuration failures may contain secrets: do not forward output.
    completed = subprocess.run(args, cwd=cwd, env=env, stdout=subprocess.PIPE, stderr=subprocess.PIPE, timeout=600)
    if completed.returncode:
        raise RuntimeError("Stack command failed; sensitive output withheld")
    return completed.stdout


def compose(*args):
    return command(["docker", "compose", "--project-name", PROJECT, *args], cwd=STATE)


def mailbox_code():
    deadline = time.monotonic() + 30
    while time.monotonic() < deadline:
        with OPENER.open("http://127.0.0.1:8025/api/v1/messages", timeout=3) as response:
            messages = json.load(response).get("messages", [])
        if messages:
            with OPENER.open("http://127.0.0.1:8025/api/v1/message/" + messages[0]["ID"], timeout=3) as response:
                message = json.load(response)
            match = re.search(r"(?<!\d)\d{6}(?!\d)", message.get("Text", ""))
            if match:
                return match.group()
        time.sleep(.5)
    raise RuntimeError("SMTP message was not delivered within deadline")


def totp(secret):
    key = base64.b32decode(secret + "=" * (-len(secret) % 8))
    digest = hmac.new(key, struct.pack(">Q", int(time.time()) // 30), hashlib.sha1).digest()
    offset = digest[-1] & 15
    return str((struct.unpack(">I", digest[offset:offset+4])[0] & 0x7fffffff) % 1000000).zfill(6)


def authenticated(page, expected_user=None):
    response = page.context.request.get(AUTH + "/_auth", headers={"X-Forwarded-Uri": "/profile"}, max_redirects=0)
    check("actual session ForwardAuth", response.status == 200 and (expected_user is None or response.headers.get("x-auth-user") == expected_user))


def login_page(page):
    page.goto(AUTH + "/_login?lang=en")
    page.locator("#mail").fill("demo@example.test")


def app_login_page(page):
    page.goto(APP)
    page.wait_for_url(lambda u: urlparse(u).hostname == urlparse(AUTH).hostname and urlparse(u).path == "/_login")
    callbacks = page.locator('input[name="callback"]')
    states = page.locator('input[name="exchange_state"]')
    values = [value for value in callbacks.evaluate_all("els => els.map(el => el.value)") if value]
    if values:
        check("protected app callback has exact expected authority", all(callback_matches(value, APP) for value in values))
    else:
        check("protected app supplies opaque return context", any(states.evaluate_all("els => els.map(el => el.value.length > 0)")))
        RESULT["callback_context"] = "opaque state present; target is proven separately by actual app arrival"
    page.locator("#mail").fill("demo@example.test")


def protected_app(page):
    page.wait_for_url(lambda u: urlparse(u).hostname == urlparse(APP).hostname and urlparse(u).path == "/")
    check("real protected hello rendered", page.locator("h1").inner_text() == "hello from the terminal")
    # hello v2.3.0 gives Mozilla user agents HTML even with Accept:text/plain;
    # its explicit format parameter takes precedence over that browser default.
    response = page.context.request.get(hello_text_url(APP), headers={"Accept": "text/plain"}, max_redirects=0)
    observed = {"status": response.status, "project_matches": response.headers.get("project") == "https://github.com/soulteary/hello", "summary_matches": "Hello from soulteary/hello!" in response.text().splitlines()}
    RESULT["protected_app_probe"] = observed
    save()
    check("protected hello200 and published response contract", observed["status"] == 200 and observed["project_matches"] and observed["summary_matches"])
    authenticated(page, "demo-user-001")


def logout(page):
    response = page.goto(AUTH + "/_account?lang=en")
    check("authenticated HTML is not cacheable", "no-store" in response.headers.get("cache-control", ""))
    denied = page.context.request.post(AUTH + "/_logout", headers={"Origin": "https://other.example"}, max_redirects=0)
    check("cross-origin logout rejected", denied.status == 403)
    unchanged = page.context.request.get(AUTH + "/_logout", max_redirects=0)
    check("GET does not perform logout", unchanged.status == 405)
    with page.expect_response(lambda r: r.request.method == "POST" and urlparse(r.url).path == "/_logout") as response:
        page.locator("[data-logout] button").click()
    check("same-origin POST logout", response.value.status in (200, 302, 303))
    page.goto(APP)
    page.wait_for_url(lambda u: urlparse(u).hostname == urlparse(AUTH).hostname and urlparse(u).path == "/_login")
    check("protected app requires login after logout", urlparse(page.url).path == "/_login")


def error_controls(context):
    page = context.new_page()
    login_page(page)
    page.locator("#deliver_via").select_option("email")
    with page.expect_response(lambda r: urlparse(r.url).path == "/_send_verify_code") as response:
        page.locator("#sendCodeBtn").click()
    body = response.value.json()
    check("real cooldown429 carries bounded timing", response.value.status == 429 and type(body.get("next_resend_in")) is int and 1 <= body["next_resend_in"] <= 86400)
    page.wait_for_function("document.querySelector('#sendCodeBtn').disabled && /[0-9]/.test(document.querySelector('#sendCodeBtn').textContent)")
    check("known429 browser cooldown disables send", page.locator("#sendCodeBtn").is_disabled() and re.search(r"\d", page.locator("#sendCodeBtn").inner_text()) is not None)
    page.close()
    # UI compatibility branch: same real429 response with optional timing removed.
    page = context.new_page()
    login_page(page)
    page.locator("#deliver_via").select_option("email")
    def without_timing(route):
        upstream = route.fetch()
        data = upstream.json()
        data.pop("next_resend_in", None)
        headers = {k:v for k,v in upstream.headers.items() if k.lower() not in ("retry-after", "content-length")}
        route.fulfill(status=upstream.status, headers=headers, body=json.dumps(data))
    page.route("**/_send_verify_code*", without_timing)
    page.locator("#sendCodeBtn").click()
    page.locator("#error-container.show").wait_for()
    check("unknown-timing compatibility UI has no invented countdown", page.locator("#sendCodeBtn").is_enabled())
    page.close()
    page = context.new_page()
    login_page(page)
    page.locator("#deliver_via").select_option("email")
    attempts = []
    def disconnected(route):
        attempts.append(1)
        route.abort("connectionfailed")
    page.route("**/_send_verify_code*", disconnected)
    page.locator("#sendCodeBtn").click()
    page.locator("#error-container.show").wait_for()
    page.wait_for_timeout(1000)
    check("intercepted ambiguous network error never retries POST", len(attempts) == 1)
    page.close()


def run_browser():
    stage("chromium-authentication")
    with sync_playwright() as pw:
        browser = pw.chromium.launch(args=["--host-resolver-rules=MAP *.test.localhost 127.0.0.1"])
        context = browser.new_context(accept_downloads=True)
        context.add_init_script("window.__acceptanceRestored = false; addEventListener('pageshow', event => { window.__acceptanceRestored = event.persisted; });")
        page = context.new_page()
        page.set_default_timeout(15000)
        login_page(page)
        page.locator("#mail").fill("")
        page.locator("#loginForm [type=submit]").click()
        page.locator("#error-container.show").wait_for()
        check("empty identifier is explained", page.locator("#error-container").inner_text() == "Enter a phone number or email address." and page.locator("#phone").get_attribute("aria-invalid") == "true" and page.locator("#phone").evaluate("el => el === document.activeElement"))
        stage("password-login")
        page.locator("#local_password").fill(os.environ["BROWSER_TEST_PASSWORD"])
        page.locator("#passwordLoginForm [type=submit]").click()
        page.wait_for_url(lambda u: "/_login" not in urlparse(u).path)
        check("real password login", "/_login" not in urlparse(page.url).path)
        authenticated(page)
        logout(page)
        app_login_page(page)
        page.locator("#deliver_via").select_option("email")
        with page.expect_response(lambda r: urlparse(r.url).path == "/_send_verify_code") as sent:
            page.locator("#sendCodeBtn").click()
        RESULT["email_challenge_status"] = sent.value.status
        try:
            reason = sent.value.json().get("reason", "")
        except Exception:
            reason = ""
        RESULT["email_challenge_reason"] = reason if reason in {"identifier_required", "identifier_invalid", "invalid", "rate_limited", "resend_cooldown", "service_unavailable", "connection_failed", "backend_unavailable", "user_not_in_list", "authentication_failed", "provider_error", "unknown_error"} else "unknown"
        stage("email-challenge-response")
        check("real email challenge accepted", sent.value.status == 200)
        error_controls(context)
        stage("smtp-email-login")
        code = mailbox_code()
        page.locator("#verify_code").fill("000000" if code != "000000" else "000001")
        with page.expect_response(lambda r: r.request.method == "POST" and urlparse(r.url).path == "/_login") as wrong:
            page.locator("#loginForm [type=submit]").click()
        check("incorrect email code rejected by backend", wrong.value.status == 401)
        page.locator("#error-container.show").wait_for()
        check("incorrect real email code is explained", page.locator("#error-container").inner_text() != "")
        page.locator("#verify_code").fill(code)
        page.locator("#loginForm [type=submit]").click()
        page.wait_for_url(lambda u: "/_login" not in urlparse(u).path)
        check("real SMTP email OTP login", "/_login" not in urlparse(page.url).path)
        protected_app(page)
        page.goto(AUTH + "/_account?lang=en")
        check("account screenshot contains no secret controls", page.locator("input, #qrcode, #backupCodesList").count() == 0 and "demo@example.test" not in page.locator("body").inner_text())
        page.screenshot(path="browser-account.png", full_page=True)
        RESULT["screenshots"] = ["browser-account.png"]
        stage("totp-enrollment")
        page.goto(AUTH + "/totp/enroll?lang=en")
        page.locator("[data-native-once] button").click()
        uri = page.locator("#qrcode").get_attribute("data-uri")
        secret = parse_qs(urlparse(uri).query)["secret"][0]
        page.locator("#code").fill(totp(secret))
        page.locator("#confirmForm [type=submit]").click()
        page.locator("#backupCodes:not([hidden])").wait_for()
        codes = [value.strip() for value in page.locator("#backupCodesList li").all_text_contents()]
        check("real TOTP enrollment returns ten backup codes", len(codes) == 10 and all(len(c) == 19 for c in codes))
        with page.expect_download() as download:
            page.locator("#downloadCodes").click()
        # Playwright temporary download is deleted with context; never copy to artifacts.
        check("backup download generated", download.value.failure() is None)
        saved_text = Path(download.value.path()).read_text()
        check("download contains the displayed recovery set", all(code in saved_text for code in codes))
        del saved_text
        page.locator("#codesSaved").check()
        check("backup acknowledgement reveals continuation", page.locator("#savedHome").is_visible())
        page.goto(AUTH + "/_account?lang=en")
        page.go_back()
        if page.evaluate("window.__acceptanceRestored === true"):
            check("actual BFCache clears recovery controls", page.locator("#backupCodesList li").count() == 0 and page.locator("#copyCodes").is_disabled() and page.locator("#downloadCodes").is_disabled() and page.locator("#codesSaved").is_disabled())
        else:
            RESULT["bfcache"] = "not exercised: Chromium restored by navigation/reload rather than BFCache"
            save()
        logout(page)
        # Enrollment consumes current TOTP step; wait for next step rather than retry a consumed POST.
        stage("authenticator-login")
        time.sleep(31 - (time.time() % 30))
        login_page(page)
        page.locator("#auth-mode").select_option("totp")
        page.locator("#otp_code").fill(totp(secret))
        page.locator("#loginForm [type=submit]").click()
        page.wait_for_url(lambda u: "/_login" not in urlparse(u).path)
        check("real authenticator login", "/_login" not in urlparse(page.url).path)
        authenticated(page, "demo-user-001")
        logout(page)
        login_page(page)
        stage("recovery-login-or-reuse")
        page.locator("#auth-mode").select_option("recovery")
        page.locator("#recovery_code").fill(codes[0])
        page.locator("#loginForm [type=submit]").click()
        page.wait_for_url(lambda u: "/_login" not in urlparse(u).path)
        check("real backup recovery login", "/_login" not in urlparse(page.url).path)
        authenticated(page, "demo-user-001")
        logout(page)
        # Let the real fixed login quota window expire; a429 cannot prove consumed-code rejection.
        stage("natural-login-quota-expiry")
        time.sleep(61)
        login_page(page)
        stage("recovery-login-or-reuse")
        page.locator("#auth-mode").select_option("recovery")
        page.locator("#recovery_code").fill(codes[0])
        with page.expect_response(lambda r: r.request.method == "POST" and urlparse(r.url).path == "/_login") as reused:
            page.locator("#loginForm [type=submit]").click()
        check("used backup backend status", reused.value.status == 401)
        page.locator("#error-container.show").wait_for()
        check("used backup rejected by real backend", "/_login" in urlparse(page.url).path)
        page.locator("#recovery_code").fill(codes[1])
        page.locator("#loginForm [type=submit]").click()
        page.wait_for_url(lambda u: "/_login" not in urlparse(u).path)
        authenticated(page, "demo-user-001")
        check("unused recovery positive control after rejection", "/_login" not in urlparse(page.url).path)
        context.close()
        browser.close()


def main():
    validate(os.environ)
    RESULT["harness_checkout_sha"] = command(["git", "rev-parse", "HEAD"]).decode().strip()
    RESULT["workflow_github_sha"] = os.environ.get("GITHUB_SHA", "")
    RESULT.update({key.lower(): os.environ[key] for key in ("FULLSTACK_REF", "GATEWAY_REF", "HERALD_IMAGE")})
    save()
    try:
        for directory, key in (("fullstack", "FULLSTACK_REF"), ("gateway", "GATEWAY_REF")):
            check(directory + " exact source", command(["git", "rev-parse", "HEAD"], ROOT/directory).decode().strip() == os.environ[key])
        command(["python3", str(ROOT/"fullstack/examples/full-stack/configure.py"), "--mode", "totp", "--output", str(STATE), "--project", PROJECT])
        password = secrets.token_urlsafe(32)
        os.environ["BROWSER_TEST_PASSWORD"] = password
        override = STATE / "browser.override.yml"
        override.write_text("services:\n  stargate:\n    build:\n      args:\n        COMMIT: '" + os.environ["GATEWAY_REF"] + "'\n        VERSION: 'browser-verification'\n    environment:\n      PASSWORDS: 'plaintext:" + password + "'\n  herald:\n    image: " + os.environ["HERALD_IMAGE"] + "\n")
        override.chmod(0o600)
        envfile = STATE/".env"
        text = envfile.read_text()
        # Append to generated absolute Compose file list; never evaluate .env as shell.
        lines = text.splitlines()
        for index, line in enumerate(lines):
            if line.startswith("COMPOSE_FILE="):
                lines[index] = line.rstrip('"') + ":" + str(override) + ('"' if line.endswith('"') else '')
        envfile.write_text("\n".join(lines) + "\n")
        os.environ["STARGATE_SOURCE_DIR"] = str(ROOT/"gateway")
        compose("config", "--quiet")
        stage("real-stack-build-readiness")
        compose("up", "-d", "--build", "--wait", "--wait-timeout", "180")
        run_browser()
        RESULT["status"] = "passed"
    except Exception as failure:
        RESULT["failure_type"] = type(failure).__name__
        RESULT["status"] = "failed"
        RESULT["service_health"] = health_summary()
        try:
            RESULT["smtp_error_signatures"] = smtp_signatures(compose("logs", "--no-color", "--no-log-prefix", "--tail", "80", "herald"))
        except Exception:
            RESULT["smtp_error_signatures"] = {"status": "unavailable"}
        RESULT["error"] = "Acceptance failed; raw errors, browser data and service logs intentionally withheld"
        raise RuntimeError(RESULT["error"]) from None
    finally:
        try:
            compose("down", "--volumes", "--remove-orphans", "--timeout", "15")
            cleanup = {}
            for kind, args in (("containers", ["ps", "-aq"]), ("networks", ["network", "ls", "-q"]), ("volumes", ["volume", "ls", "-q"])):
                cleanup[kind] = command(["docker", *args, "--filter", "label=com.docker.compose.project=" + PROJECT]).decode().split()
            check("owned stack resources removed", not any(cleanup.values()))
            RESULT["cleanup"] = cleanup
        except Exception:
            RESULT["cleanup"] = {"status": "failed"}
            RESULT["status"] = "failed"
        save()
    if RESULT["status"] != "passed":
        raise SystemExit(1)


if __name__ == "__main__":
    main()
