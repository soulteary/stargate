# Authentication themes

Native themes require a Stargate build containing this format; earlier releases
ignore these settings. The theme and UI contracts are both `1`.

Set `THEME_DIR` to one theme directory, for example `/app/themes/deep-space`.
Mount it read-only in Docker:

```yaml
services:
  stargate:
    environment:
      THEME_DIR: /app/themes/deep-space
    volumes:
      - ./themes/deep-space:/app/themes/deep-space:ro
```

The runtime user (UID 10001 in the image) needs read access to the files and
search access to the directories. A theme is read and validated at startup.
Restart after changing it. An invalid selected theme prevents startup.
`THEME_SAFE_MODE=true` skips the theme and all legacy template, asset and favicon
overrides, using the built-in interface. It can recover an instance with an
unreadable or invalid theme directory.

With `THEME_DIR` set, all seven authentication pages use built-in templates,
JavaScript and favicon. The theme supplies only validated data. This avoids
combining a new form with a stale external script. Without either setting, the
existing external-directory precedence remains unchanged. Root, ForwardAuth,
login APIs, callbacks, sessions and rate limits retain their existing behavior.

## Package

```text
deep-space/
  theme.yml
  assets/
    deep-space.jpg
  LICENSE.txt
```

`theme.yml` is a single UTF-8 YAML document, at most 64 KiB:

```yaml
contract: 1
ui_contract: 1
id: deep-space
name: Deep Space
version: "1.0.0"
author: soulteary
license: Apache-2.0
layout: split
tokens:
  background: "#101419"
  surface: "#161c22"
  text: "#edf2f5"
  muted: "#96a3af"
  accent: "#8ed9e8"
  accent-text: "#10232b"
  border: "#596875"
  danger: "#f08d83"
  focus: "#8ed9e8"
  art: "#0b1018"
  art-text: "#f4f7fa"
  radius: "6px"
  font-scale: "1"
brand:
  image: deep-space.jpg
  eyebrow: BEYOND THE ORDINARY
  heading: Welcome to your universe.
  caption: Your gateway to connected services.
```

Required metadata: `contract`, `ui_contract`, `id`, `name`, `version`, `author`,
`license`, `layout`. The ID is at most 32 lowercase ASCII letters/digits with
single hyphens between groups. Metadata strings are nonempty and at most 128
bytes. Layout is `split` or `centered`; split becomes stacked below 900px.

`tokens`, `modes` and `brand` are optional. Omitted tokens use the defaults in
`src/internal/theme/theme.go`. Color tokens are `background`, `surface`, `text`,
`muted`, `accent`, `accent-text`, `border`, `danger`, `focus`, `art`, `art-text`.
They accept quoted `#RRGGBB` only. `radius` accepts integer pixels from `0px` to
`32px`. `font-scale` accepts 0.8–1.5, at most four characters. These values become
`--sg-<token>` variables in a generated same-origin stylesheet, not inline styles.

Text, muted text, links and error text must contrast at least 4.5:1 with both
background and surface. Button text must contrast 4.5:1 with accent, and brand
text with art. The image behind brand text is not covered by this check: review
its readability with the built-in scrim at desktop and mobile widths.

`modes.light` and `modes.dark` override base tokens under the corresponding
`prefers-color-scheme` media query. Each merged palette passes the same checks.
Without modes the palette is fixed, irrespective of the device preference.

Brand fields are `image`, `heading`, `caption`, `eyebrow`. Text is plain text,
at most 256 bytes per field, without newlines or NUL, and is HTML-escaped.
The site name still comes from `LOGIN_PAGE_TITLE`; footer and support links
still come from `LOGIN_PAGE_FOOTER_TEXT` and `LOGIN_SUPPORT_URL`. Authentication
labels and messages continue to use Stargate's seven languages. Decorative
brand text is the same in each language.

The optional image is a PNG or JPEG in `assets/`, at most 2 MiB, at most 8192
pixels per side and 32 million pixels in total. Its extension must match its
header. Its ASCII filename starts with a letter or digit and contains only
letters, digits, dots, underscores and hyphens, at most 64 characters.
The image header is checked, not the complete image stream.

Only `theme.yml`, its declared image and optional `LICENSE.txt` (64 KiB) are
accepted. Links, unknown YAML fields, extra files, CSS, HTML, scripts, fonts and
SVG are rejected. No archive import, remote download or theme discovery runs in
the gateway. Themes are installed by the operator; nothing is fetched or updated
automatically. The directory must remain under the operator's control while it
is loaded.

Validated bytes are held in an immutable in-memory snapshot. CSS and image URLs
use a SHA-256 of the manifest and image contents and return fixed MIME types,
`nosniff` and a one-year immutable cache policy. The manifest and license are
never served. Authentication HTML retains `Cache-Control: no-store` and the
existing CSP. A running process does not reread files. Only its current digest
is served, so deploy a theme consistently across replicas; an old page whose
resources were not loaded may need reloading after a restart.

## Validation and distribution

From the Stargate source:

```sh
go run ./src/cmd/themecheck /path/to/deep-space
```

Exit codes: `0` accepted, `1` at least one invalid theme, `2` no arguments.
The command uses the same loader as the server and makes no network requests.
Run it from the build being deployed: matching format numbers do not make
unknown future fields available to an older build.

Themes are maintained separately at <https://github.com/soulteary/stargate-themes>.
Their repository packages the directory for distribution and records checksums.
Extract it before mounting; the server reads directories, not ZIP files.
Review visual readability and resource rights before publishing a theme.
