"""Fixed boolean signatures, not raw log output or a root-cause determination."""
import json


def smtp_signatures(raw):
    result = {"starttls_failure_seen": False, "starttls_x509_failure_seen": False,
              "smtp_dial_failure_seen": False, "authentication_failure_signature_seen": False}
    # Only bounded structured error fields are examined. Other message/user fields
    # cannot cause a classification and none of the input is returned.
    for line in raw[-131072:].splitlines():
        if len(line) > 8192:
            continue
        try:
            entry = json.loads(line)
        except (ValueError, UnicodeDecodeError):
            continue
        error = entry.get("error") if isinstance(entry, dict) else None
        if not isinstance(error, str):
            continue
        starttls = "STARTTLS failed:" in error
        result["starttls_failure_seen"] |= starttls
        result["starttls_x509_failure_seen"] |= starttls and "x509:" in error
        result["smtp_dial_failure_seen"] |= "SMTP dial failed:" in error
        result["authentication_failure_signature_seen"] |= "authentication failed:" in error
    return result


def post_history_cache_miss(error, playwright_error_type):
    """Only the specific Playwright POST-history sentinel permits a coverage gap."""
    return isinstance(error, playwright_error_type) and "net::ERR_CACHE_MISS" in str(error)


def playwright_failure_category(error, playwright_error_type):
    if not isinstance(error, playwright_error_type):
        return "not_playwright"
    for marker, category in (("net::ERR_CACHE_MISS", "post_history_cache_miss"),
                             ("interrupted by another navigation", "navigation_interrupted"),
                             ("net::ERR_ABORTED", "navigation_aborted"),
                             ("Target page, context or browser has been closed", "target_closed"),
                             ("Timeout", "timeout")):
        if marker in str(error):
            return category
    return "other_playwright_error"
