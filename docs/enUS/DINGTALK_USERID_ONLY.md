# Opt-in userid-only work-IM login

`LOGIN_DINGTALK_REQUIRE_USERID=false` is the compatible default. It preserves
existing canonical-phone fallback for explicit DingTalk delivery. Opting in is
an operational choice; this change does not enable it on a running deployment.

An administrator must independently verify and maintain the association between
a stable Warden account and its `dingtalk_userid`. Request-provided userid,
receiver and contact values cannot create or override that association. The
userid is an opaque, case-sensitive identifier, not a guessed mobile number.

## Isolated profile

For an isolated synthetic evaluation, explicitly set:

```dotenv
LOGIN_DINGTALK_REQUIRE_USERID=true
LOGIN_SMS_ENABLED=false
LOGIN_EMAIL_ENABLED=false
WARDEN_ENABLED=true
HERALD_ENABLED=true
```

Retain the separately verified service authentication and TLS settings. These
lines are not a complete deployment manifest and contain no credentials or
service addresses. Use only a local mock Herald/provider in the isolated test.
If separately evaluating an actual herald-dingtalk adapter, its
`DINGTALK_LOOKUP_MODE=none` avoids mobile lookup; it cannot substitute for the
new gateway constraint. Do not send real messages as part of this profile.

Use synthetic active identities with a stable `user_id` and an explicit
single-recipient `dingtalk_userid`. If Warden has a response-field whitelist,
retain these and the existing required `status`/lookup fields. With strict mode,
missing bindings and comma/control/CRLF recipient lists fail before Herald is
called, even if the record also has a phone and email. Keeping SMS and email
disabled also prevents blank/other channel selection from finding an alternative
OTP path. Owlmail remains mail-chain verification; it is not a shared fallback
inbox for personal IM login.

A prior phone-derived binding, changed mapping or userid case mismatch cannot
reach Herald verification in strict mode. Native directory authorization still
runs first; missing or inactive identities are refused under the existing
cache/session boundaries. This flag does not promise instantaneous global
revocation and does not destroy old sessions, pending bindings or provider state.
It neither grants team/project roles nor changes Phorge permissions or optional
features. Mapping approval alone proves neither real mapping completion nor
200-person capacity or real IM delivery.

## Regression commands and scope

```sh
go test -race ./src/internal/config ./src/internal/handlers -run '^(TestDingtalkUserIDRequirementValidation|TestStrictDingtalk|TestVerificationDestinationUsesOnlyCanonicalWardenBindings|TestAuthRefreshRejectsMissingOrDisabledUser)' -count=1
```

Tests use in-process synthetic Warden and Herald HTTP servers, plus the native
challenge-binding store and actual send/login routes. They cover default-false
compatibility, strict missing/whitespace mappings, case preservation, malformed
recipient lists, inactive/missing users, untrusted form fields, disabled channels,
repeated idempotent missing-mapping attempts and stale phone-derived bindings.
Rejected login bindings remain intact and do not invoke verification; no real
message, deployment configuration or private mapping is used.
