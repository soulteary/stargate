"""Chromium theme checks: real password/session flow and synthetic page views.

The view fixtures exercise layout and the core browser state machine. They do
not establish delivered-code or TOTP backend success; run run.py for that.
"""
import argparse
import json
import os
import re
from pathlib import Path
import secrets
import socket
import subprocess
import tempfile
import time
import urllib.request
from urllib.parse import urlparse, parse_qs

from playwright.sync_api import sync_playwright

ROOT = Path(__file__).resolve().parents[2]
PAGES = ("login", "login.warden", "account", "totp_start", "totp_enroll", "totp_revoke", "step_up")


def main():
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument("--theme", type=Path, required=True)
    parser.add_argument("--output", type=Path, required=True)
    args = parser.parse_args()
    args.output.mkdir(parents=True, exist_ok=True)
    result = {"status": "running", "checks": [], "scope": "Chromium; real password login, session and logout; synthetic Warden/TOTP/step-up views and browser transport fixtures"}
    def check(name, condition):
        if not condition:
            raise AssertionError(name)
        result["checks"].append(name)
    with tempfile.TemporaryDirectory(prefix="stargate-theme-") as temporary:
        binary = Path(temporary) / "server.test"
        subprocess.run(["go", "test", "-c", "-o", str(binary), "./src/cmd/stargate"], cwd=ROOT, check=True)
        with socket.socket() as listener:
            listener.bind(("127.0.0.1", 0))
            port = listener.getsockname()[1]
        address = "127.0.0.1:" + str(port)
        base = "http://" + address
        password = secrets.token_urlsafe(24)
        env = {**os.environ, "STARGATE_THEME_BROWSER_DIR": str(args.theme.resolve()), "STARGATE_THEME_BROWSER_ADDR": address, "STARGATE_THEME_BROWSER_PASSWORD": password}
        # Server logs can contain session details; keep them temporary, never upload.
        with (Path(temporary) / "server.log").open("wb") as log:
            process = subprocess.Popen([str(binary), "-test.run=^TestThemeBrowserServer$"], cwd=ROOT, env=env, stdout=log, stderr=log)
            try:
                opener = urllib.request.build_opener(urllib.request.ProxyHandler({}))
                for _ in range(100):
                    if process.poll() is not None:
                        raise RuntimeError("theme server exited")
                    try:
                        with opener.open(base + "/healthz", timeout=1) as response:
                            if response.status == 200:
                                break
                    except OSError:
                        time.sleep(.1)
                else:
                    raise RuntimeError("theme server not ready")
                with sync_playwright() as pw:
                    browser = pw.chromium.launch(executable_path=os.environ.get("PLAYWRIGHT_CHROMIUM_EXECUTABLE"))
                    context = browser.new_context()
                    external, errors, violations = [], [], []
                    context.on("request", lambda request: external.append(request.url) if urlparse(request.url).netloc not in (address, "") else None)
                    page = context.new_page()
                    page.on("pageerror", lambda error: errors.append(type(error).__name__))
                    page.on("console", lambda message: violations.append(True) if message.type == "error" and "Content Security Policy" in message.text else None)
                    page.set_default_timeout(10000)
                    for width, height in ((1440, 1000), (390, 844)):
                        page.set_viewport_size({"width": width, "height": height})
                        for lang in ("en", "zh"):
                            for name in PAGES:
                                response = page.goto(base + "/__theme_preview/" + name + "?lang=" + lang)
                                page.locator(".auth-brand img").evaluate("img => img.decode()")
                                check(f"{name} {lang} {width}: themed no-store page", response.status == 200 and response.headers.get("cache-control") == "no-store" and page.locator("body").get_attribute("class") == "auth-themed")
                                check(f"{name} {lang} {width}: no horizontal overflow", page.evaluate("document.documentElement.scrollWidth <= innerWidth"))
                                check(f"{name} {lang} {width}: visible form area", page.locator("main").is_visible())
                                check(f"{name} {lang} {width}: palette loaded", page.locator("body").evaluate("el => getComputedStyle(el).backgroundColor") == "rgb(16, 20, 25)")
                                check(f"{name} {lang} {width}: error initially hidden", not page.locator("#error-container").is_visible())
                                if name == "totp_enroll":
                                    check(f"{width} {lang}: recovery initially hidden", not page.locator("#backupCodes").is_visible() and not page.locator("#savedHome").is_visible())
                                if name == "login.warden":
                                    page.locator("#auth-mode").select_option("totp")
                                    check(f"{width} {lang}: TOTP switch", page.locator("#otp_code").is_visible() and not page.locator("#delivery-fields").is_visible())
                                    page.locator("#auth-mode").select_option("recovery")
                                    check(f"{width} {lang}: recovery switch", page.locator("#recovery_code").is_visible() and not page.locator("#totp-fields").is_visible())
                                    page.locator("#auth-mode").select_option("delivery")
                                    page.locator("#phone").focus()
                                    check(f"{width} {lang}: keyboard focus visible", page.locator("#phone").evaluate("el => getComputedStyle(el).outlineStyle !== 'none'"))
                    page.set_viewport_size({"width": 1440, "height": 1000})
                    page.goto(base + "/__theme_preview/login.warden?lang=en")
                    page.locator(".auth-brand img").evaluate("img => img.decode()")
                    page.emulate_media(reduced_motion="reduce")
                    check("reduced motion disables animation", page.locator(".auth-brand img").evaluate("el => getComputedStyle(el).animationName") == "none" and page.locator("main").evaluate("el => getComputedStyle(el).animationName") == "none")
                    page.screenshot(path=str(args.output / "deep-space-login-desktop.png"), full_page=True)
                    page.set_viewport_size({"width": 390, "height": 844})
                    page.goto(base + "/_login?lang=zh")
                    page.locator(".auth-brand img").evaluate("img => img.decode()")
                    page.screenshot(path=str(args.output / "deep-space-login-mobile.png"), full_page=True)
                    page.locator("#password").fill("incorrect-password")
                    with page.expect_response(lambda response: response.request.method == "POST" and urlparse(response.url).path == "/_login") as rejected:
                        page.locator("[type=submit]").click()
                    check("real wrong password rejected", rejected.value.status == 401)
                    check("wrong password feedback visible", page.locator("#error-container").is_visible() and bool(page.locator("#error-container").inner_text()))
                    page.locator("#password").fill(password)
                    page.locator("[type=submit]").click()
                    page.wait_for_url(base + "/")
                    auth = context.request.get(base + "/_auth")
                    check("real password creates authenticated session", auth.status == 200)
                    response = page.goto(base + "/_account?lang=en")
                    check("real account is themed and no-store", response.status == 200 and response.headers.get("cache-control") == "no-store" and page.locator(".auth-brand").count() == 1)
                    check("account has no secret controls", page.locator("input, #qrcode, #backupCodesList").count() == 0)
                    page.set_viewport_size({"width": 1440, "height": 1000})
                    page.screenshot(path=str(args.output / "deep-space-account.png"), full_page=True)
                    page.locator("[data-logout] button").click()
                    page.wait_for_url(lambda url: urlparse(url).path == "/_logout")
                    check("logout response shown", page.locator("body").inner_text() == "Logged out")
                    check("real logout revokes session", context.request.get(base + "/_auth", max_redirects=0).status != 200)
                    # Controlled transport fixture: prove that core JS binds the
                    # selected account/challenge and preserves the opaque return state.
                    page.goto(base + "/__theme_preview/login.warden?lang=en")
                    page.locator("#mail").fill("preview@example.test")
                    page.locator("#deliver_via").select_option("email")
                    requests = []
                    page.route("**/_send_verify_code*", lambda route: route.fulfill(json={"success": True, "challenge_id": "fixture-challenge", "next_resend_in": 2}))
                    def login_fixture(route):
                        requests.append({key: [value] for key, value in re.findall(r'name="([^"\r\n]+)"\r\n\r\n([^\r\n]*)', route.request.post_data)})
                        route.fulfill(status=401, json={"reason": "invalid", "error": "Invalid code"})
                    page.route("**/_login*", login_fixture)
                    page.locator("#sendCodeBtn").click()
                    page.wait_for_function("() => document.querySelector('#error-container').textContent === 'Code sent. Check your messages.'")
                    page.locator("#verify_code").fill("123456")
                    with page.expect_response(lambda response: response.request.method == "POST" and urlparse(response.url).path == "/_login"):
                        page.locator("#loginForm [type=submit]").click()
                    page.wait_for_function("() => !document.querySelector('#loginForm [type=submit]').disabled")
                    check("fixture: login carries bound challenge and return state", len(requests) == 1 and requests[0].get("challenge_id") == ["fixture-challenge"] and requests[0].get("exchange_state") == ["preview-state"] and requests[0].get("mail") == ["preview@example.test"])
                    check("fixture: failure restores form controls", page.locator("#mail").is_enabled() and page.locator("#loginForm [type=submit]").is_enabled())
                    page.locator("#mail").fill("other@example.test")
                    page.locator("#loginForm [type=submit]").click()
                    check("fixture: changed account cannot reuse challenge", len(requests) == 1)
                    check("no external resource requests", not external)
                    check("no browser script errors", not errors)
                    check("no CSP console errors", not violations)
                    result["browser"] = browser.version
                    context.close()
                    browser.close()
                result["status"] = "passed"
            finally:
                process.terminate()
                try:
                    process.wait(timeout=10)
                except subprocess.TimeoutExpired:
                    process.kill()
                    process.wait()
                (args.output / "theme-results.json").write_text(json.dumps(result, indent=2) + "\n")
    print("Theme browser checks passed: " + str(len(result["checks"])))


if __name__ == "__main__":
    main()
