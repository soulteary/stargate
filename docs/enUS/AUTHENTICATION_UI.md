# Authentication pages

The login page keeps delivered verification codes, authenticator codes and backup codes as separate modes. Changing the account, delivery channel or mode discards the page's challenge binding; request a code for the current selection. Backend account, purpose and channel checks still apply.

A pending login freezes the account and method until it completes. Requests are not retried automatically. If the connection or response fails, the operation may already have completed and a one-time code may have been consumed. Check your current login state before manually choosing another action. A rejected resend with HTTP 429 keeps a known earlier challenge; an ambiguous send result does not.

Resend timers use only server-provided seconds. A response without valid timing information shows a general rate-limit message. Reaching zero permits another manual attempt, not a guarantee of delivery: other server limits may still apply. Reloading a page does not reset server counters.

After TOTP setup succeeds, save the one-time backup-code list. Copying and downloading occur only on an explicit button click. Codes are not written to localStorage or sessionStorage; they remain sensitive in page memory until leaving. Leaving clears the rendered setup secrets; browser Back does not recover them or start another enrollment. If confirmation had an uncertain result or unsaved codes are lost, inspect account settings and contact your administrator rather than repeatedly starting setup.

Open `/_account` on the authentication host to manage the current session or use its same-origin POST logout button. Logout affects this browser session; it is not an all-device revocation feature. The root endpoint's existing plain-text behavior is unchanged.

An optional `LOGIN_SUPPORT_URL` provides a support link. See [configuration](CONFIG.md#login_support_url). With no configured link the pages omit it rather than displaying a placeholder destination.

## Standalone binary resources

The binary includes an explicit set of authentication templates and public assets. When no external template or asset directory exists, it serves these built-in resources without creating files in the working directory. Docker images still carry the same resource directory.

Existing external directories keep their precedence. Update a complete external template/asset tree with the binary, or remove that override to use the matching built-in UI. An old or partial override is not silently merged with new templates. Configuration, generated files and setup secrets are not embedded.
