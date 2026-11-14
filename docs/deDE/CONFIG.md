# Konfigurationsreferenz

> **Historische Übersetzung**: Dieser Leitfaden enthält nicht alle aktuellen Änderungen. Die [aktuelle englische Konfiguration](../enUS/CONFIG.md) beschreibt die Redis-Anforderungen und Upgrade-Grenzen.

Dieses Dokument beschreibt alle Konfigurationsoptionen für Stargate im Detail.

## Inhaltsverzeichnis

- [Konfigurationsmethoden](#konfigurationsmethoden)
- [Erforderliche Konfiguration](#erforderliche-konfiguration)
- [Optionale Konfiguration](#optionale-konfiguration)
- [Passwort-Konfiguration](#passwort-konfiguration)
- [Konfigurationsbeispiele](#konfigurationsbeispiele)

## Konfigurationsmethoden

Stargate wird über Umgebungsvariablen konfiguriert. Alle Konfigurationselemente werden über Umgebungsvariablen definiert, keine Konfigurationsdatei ist erforderlich.

### Definition von Umgebungsvariablen

**Linux/macOS:**

```bash
export AUTH_HOST=auth.example.com
export PASSWORDS='plaintext:yourpassword'
```

**Docker:**

```bash
docker run -e AUTH_HOST=auth.example.com -e PASSWORDS=plaintext:yourpassword ghcr.io/soulteary/stargate:1.3.8
```

**Docker Compose:**

```yaml
services:
  stargate:
    environment:
      - AUTH_HOST=auth.example.com
      - PASSWORDS=plaintext:yourpassword
```

## Erforderliche Konfiguration

Die folgenden Konfigurationselemente sind erforderlich. Wenn sie nicht definiert sind, wird der Dienst nicht starten.

### `AUTH_HOST`

Hostname des Authentifizierungsdienstes.

| Attribut | Wert |
|----------|------|
| **Typ** | String |
| **Erforderlich** | Ja |
| **Standard** | Keine |
| **Beispiel** | `auth.example.com` |

**Beschreibung:**

- Wird verwendet, um Callback-URLs für die Anmeldung zu erstellen
- Normalerweise auf den Hostnamen des Stargate-Dienstes gesetzt
- Unterstützt Wildcard `*` (für Produktion nicht empfohlen)

**Beispiel:**

```bash
AUTH_HOST=auth.example.com
```

### `PASSWORDS`

Passwort-Konfiguration, die den Verschlüsselungsalgorithmus und die Liste der Passwörter angibt.

| Attribut | Wert |
|----------|------|
| **Typ** | String |
| **Erforderlich** | Ja |
| **Standard** | Keine |
| **Format** | `algorithm:password1|password2|password3` |

**Beschreibung:**

- Format: `algorithm:password1|password2|password3`
- Unterstützt mehrere Passwörter, getrennt durch `|`
- Jedes Passwort, das die Überprüfung besteht, ermöglicht die Anmeldung
- Unterstützte Algorithmen siehe Abschnitt [Passwort-Konfiguration](#passwort-konfiguration)

**Beispiele:**

```bash
# Einzelnes Klartext-Passwort
PASSWORDS='plaintext:test123'

# Mehrere Klartext-Passwörter
PASSWORDS='plaintext:test123|admin456|user789'

# BCrypt-Hash
PASSWORDS='bcrypt:$2a$10$N9qo8uLOickgx2ZMRZoMyeIjZAgcfl7p92ldGxad68LJZdL17lhWy'

```

## Optionale Konfiguration

Die folgenden Konfigurationselemente sind optional. Wenn sie nicht definiert sind, werden Standardwerte verwendet.

### `DEBUG`

Debug-Modus aktivieren.

| Attribut | Wert |
|----------|------|
| **Typ** | Boolean |
| **Erforderlich** | Nein |
| **Standard** | `false` |
| **Mögliche Werte** | `true`, `false` |

**Beschreibung:**

- Wenn aktiviert, wird die Protokollierungsebene auf `DEBUG` gesetzt
- Zeigt detailliertere Debug-Informationen an
- Empfohlen, in Produktion auf `false` zu setzen

**Beispiel:**

```bash
DEBUG=true
```

### `LANGUAGE`

Interface-Sprache.

| Attribut | Wert |
|----------|------|
| **Typ** | String |
| **Erforderlich** | Nein |
| **Standard** | `en` |
| **Mögliche Werte** | `en` (Englisch), `zh` (Chinesisch), `fr` (Französisch), `it` (Italienisch), `ja` (Japanisch), `de` (Deutsch), `ko` (Koreanisch) |

**Beschreibung:**

- Beeinflusst die Sprache der Fehlermeldungen und des Interface-Textes
- Groß-/Kleinschreibung nicht beachtend (`EN`, `en`, `En` funktionieren alle)

**Beispiel:**

```bash
LANGUAGE=de
```

### `LOGIN_PAGE_TITLE`

Titel der Anmeldeseite.

| Attribut | Wert |
|----------|------|
| **Typ** | String |
| **Erforderlich** | Nein |
| **Standard** | `Stargate - Login` |

**Beschreibung:**

- Wird an der Titelposition der Anmeldeseite angezeigt
- Unterstützt HTML-Tags (nicht empfohlen)

**Beispiel:**

```bash
LOGIN_PAGE_TITLE='Mein Authentifizierungsdienst'
```

### `LOGIN_PAGE_FOOTER_TEXT`

Fußzeilentext der Anmeldeseite.

| Attribut | Wert |
|----------|------|
| **Typ** | String |
| **Erforderlich** | Nein |
| **Standard** | `Copyright © 2024 - Stargate` |

**Beschreibung:**

- Wird an der Fußzeilenposition der Anmeldeseite angezeigt
- Unterstützt HTML-Tags (nicht empfohlen)

**Beispiel:**

```bash
LOGIN_PAGE_FOOTER_TEXT='© 2024 Meine Firma'
```

### `USER_HEADER_NAME`

Name des Benutzer-Headers, der nach erfolgreicher Authentifizierung gesetzt wird.

| Attribut | Wert |
|----------|------|
| **Typ** | String |
| **Erforderlich** | Nein |
| **Standard** | `X-Forwarded-User` |

**Beschreibung:**

- Nach erfolgreicher Authentifizierung setzt Stargate diesen Header in der Antwort
- Der Wert ist die ID des authentifizierten Benutzers, sofern verfügbar; reine Passwortauthentifizierung verwendet `authenticated`
- Backend-Dienste können über diesen Header bestimmen, ob ein Benutzer authentifiziert ist
- Muss eine nicht-leere Zeichenkette sein

**Beispiel:**

```bash
USER_HEADER_NAME=X-Authenticated-User
```

### `TRUSTED_PROXIES` und `PROXY_HEADER`

`TRUSTED_PROXIES` listet die Quell-IPs oder CIDR-Bereiche der Reverse Proxys auf, deren Forwarded-Headern Stargate vertraut. Der Standardwert ist leer; `PROXY_HEADER` benennt den Header mit der Client-IP (Standard: `X-Forwarded-For`).

Setzen Sie `TRUSTED_PROXIES` hinter einem Reverse Proxy, auch wenn Stargate ohne diesen Wert startet. Solange der Proxy nicht aufgeführt ist, sieht Stargate nur die Anfrage, die der Proxy an `/_auth` gesendet hat, nicht die des Benutzers:

- die Weiterleitung zur Anmeldung enthält keinen `callback` und wird mit `http://` gebildet, sodass Benutzer nach der Anmeldung auf dem Auth-Host bleiben, statt zur Anwendung zurückzukehren;
- wenn der Proxy HTTPS terminiert, können sich Benutzer gar nicht anmelden: Der Browser sendet `Origin: https://...`, Stargate vergleicht ihn mit der `http://`-Anfrage, die es vom Proxy erhalten hat, und `POST /_login`, `/_logout`, `/_step_up`, `/_send_verify_code`, `/_send_reauth_code` und `/totp/*` werden mit `403` und `cross-site request rejected` beantwortet;
- Step-up-Pfadregeln können nicht ausgewertet werden (deshalb erfordert `STEP_UP_ENABLED=true` den Wert `TRUSTED_PROXIES`);
- jeder Client wird der Proxy-Adresse zugeordnet und teilt sich ein Rate-Limit-Kontingent und eine Adresse in den Audit-Einträgen.

Stargate protokolliert beim Start eine Warnung, wenn `TRUSTED_PROXIES` leer ist, und eine weitere, wenn es zum ersten Mal Forwarded-Header einer nicht aufgeführten Gegenstelle ignoriert. Außerdem protokolliert es einmalig eine Warnung, wenn es eine Anfrage als Cross-Site-Anfrage ablehnt, deren `Origin` zu den ignorierten Headern `X-Forwarded-Proto` und `X-Forwarded-Host` passt; die `403`-Antwort selbst nennt die Ursache nicht.

### `COOKIE_DOMAIN`

Cookie-Domain, verwendet für Cross-Domain-Sitzungsfreigabe.

| Attribut | Wert |
|----------|------|
| **Typ** | String |
| **Erforderlich** | Nein |
| **Standard** | Leer (nicht definiert) |

**Beschreibung:**

- Wenn definiert, werden Sitzungs-Cookies auf die angegebene Domain gesetzt
- Unterstützt Cross-Subdomain-Sitzungsfreigabe
- Format: `.example.com` (beachten Sie den anfänglichen Punkt)
- Wenn leer gesetzt, sind Cookies nur für die aktuelle Domain gültig

**Beispiel:**

```bash
# Sitzungsfreigabe auf allen Subdomains *.example.com ermöglichen
COOKIE_DOMAIN=.example.com
```

**Cross-Domain-Sitzungsfreigabe-Szenario:**

Angenommen, die folgenden Domains:
- `auth.example.com` - Authentifizierungsdienst
- `app1.example.com` - Anwendung 1
- `app2.example.com` - Anwendung 2

Nach dem Setzen von `COOKIE_DOMAIN=.example.com`:
1. Der Benutzer meldet sich bei `auth.example.com` an
2. Das Sitzungs-Cookie wird auf die Domain `.example.com` gesetzt
3. Der Benutzer kann dieselbe Sitzung auf `app1.example.com` und `app2.example.com` verwenden

### `SESSION_EXCHANGE_REDIRECT_PATH`

Zielpfad innerhalb des Callback-Ursprungs nach einem verifizierten Sitzungsaustausch. Standard: `/`. Für den nativen Anmeldeabschluss kann `/auth/login/example/` verwendet werden. Nur vom Administrator konfigurierte absolute Pfade ohne Query, Fragment, Prozentkodierung oder Steuerzeichen sind erlaubt. Callback-Whitelist und Wiedergabeschutz bleiben unverändert.

### `PORT`

Abhörport des Dienstes für native und Container-Bereitstellungen. Wird wie andere Optionen über die config-Paket-Umgebungsvariablen geladen und validiert.

| Attribut | Wert |
|----------|------|
| **Typ** | String |
| **Erforderlich** | Nein |
| **Standard** | Leer beim nativen Programm (`:80`); offizielle Container-Images setzen `PORT=8080` |

**Beschreibung:**

- In Entwicklung und Produktion konfigurierbar
- Offizielle Container verwenden standardmäßig `8080`. Für Überschreibungen nur eine Zahl wie `PORT=8080` verwenden, da der Healthcheck daraus eine URL erstellt.
- Format beim nativen Programm: Portnummer (z. B. `8080`) oder `:port` (z. B. `:8080`)

**Beispiel:**

```bash
PORT=8080
```

## Sicherheits- und Integrationskonfiguration

Die folgende Tabelle ist mit den tatsächlich in `internal/config` registrierten sicherheitsrelevanten Variablen synchronisiert. Leere optionale Werte deaktivieren die jeweilige Integration.

| Variable | Werte | Standard | Zweck |
|----------|-------|----------|-------|
| `TRUSTED_PROXIES` | IP/CIDR-Liste | leer | Quellen, deren Forwarded-Header vertraut werden |
| `PROXY_HEADER` | Headername | `X-Forwarded-For` | Quell-IP-Header |
| `COOKIE_SECURE` | true/false | true | Secure-Attribut der Sitzungs-Cookies |
| `CALLBACK_ALLOWED_HOSTS` | Host-Liste | leer | Explizite Ziele für Rückleitungen |
| `SESSION_EXCHANGE_SECRET` | Geheimnis, mind. 32 Zeichen | leer | Signiert kurzlebige domänenübergreifende Tickets |
| `SESSION_EXCHANGE_REDIRECT_PATH` | Pfad | `/` | Nein |
| `PASSWORD_HEADER_AUTH_ENABLED` | true/false | false | Aktiviert den vertrauenswürdigen Legacy-Passwort-Header |
| `LOG_LEVEL` | debug/info/warn/error | info | Start-Protokollstufe |
| `HEADER_AUTH_ENABLED` | true/false | false | Vertrauenswürdige Header-Authentifizierung |
| `HEADER_AUTH_SHARED_SECRET` | Geheimnis, mindestens 32 Zeichen | leer | Gemeinsames Geheimnis für Header-Authentifizierung |
| `HEADER_AUTH_SECRET_HEADER` | Headername | `X-Stargate-Header-Auth` | Transportiert das gemeinsame Geheimnis |
| `WARDEN_ENABLED` | true/false | false | Warden-Integration |
| `WARDEN_URL` | URL | leer | Warden-Endpunkt |
| `WARDEN_API_KEY` | Geheimnis | leer | API-Key-Authentifizierung |
| `WARDEN_HMAC_KEY_ID` | String | leer | HMAC-Schlüsselkennung |
| `WARDEN_HMAC_SECRET` | Geheimnis | leer | HMAC-Signaturgeheimnis |
| `WARDEN_TLS_CA_CERT_FILE` | Pfad | leer | Benutzerdefinierte CA |
| `WARDEN_TLS_CLIENT_CERT_FILE` | Pfad | leer | mTLS-Clientzertifikat |
| `WARDEN_TLS_CLIENT_KEY_FILE` | Pfad | leer | mTLS-Clientschlüssel |
| `WARDEN_TLS_SERVER_NAME` | String | leer | Erwarteter TLS-Servername |
| `WARDEN_CACHE_TTL` | positive Ganzzahl (Sekunden) | 300 | Nur aus Kompatibilitätsgründen akzeptiert; ohne Wirkung |
| `WARDEN_SOURCE_MAX_AGE` | Dauer | 0s | Maximales Alter der Warden-Quelldaten; 0s deaktiviert die Prüfung |
| `WARDEN_SOURCE_CLOCK_SKEW` | Dauer | 0s | Tolerierte Zeitabweichung; kleiner als das maximale Alter |
| `METRICS_API_KEY` | Geheimnis | leer | Separater Bearer-Schlüssel für /metrics, mindestens 32 Bytes |
| `HERALD_ENABLED` | true/false | false | Herald-Integration |
| `HERALD_URL` | URL | leer | Herald-Endpunkt |
| `HERALD_API_KEY` | Geheimnis | leer | API-Key-Authentifizierung |
| `HERALD_HMAC_KEY_ID` | String | leer | HMAC-Schlüsselkennung |
| `HERALD_HMAC_SECRET` | Geheimnis | leer | HMAC-Signaturgeheimnis |
| `HERALD_CLIENT_IP_HEADER` | Header-Name | leer | Header, in dem Stargate die Client-Adresse an Herald sendet; muss `HERALD_TRUSTED_PROXY_HEADER` von Herald entsprechen, und Stargate muss in `HERALD_TRUSTED_PROXIES` stehen |
| `HERALD_TLS_CA_CERT_FILE` | Pfad | leer | Benutzerdefinierte CA |
| `HERALD_TLS_CLIENT_CERT_FILE` | Pfad | leer | mTLS-Clientzertifikat |
| `HERALD_TLS_CLIENT_KEY_FILE` | Pfad | leer | mTLS-Clientschlüssel |
| `HERALD_TLS_SERVER_NAME` | String | leer | Erwarteter TLS-Servername |
| `HERALD_TOTP_ENABLED` | true/false | false | TOTP-Anmeldung und -Verwaltung |
| `LOGIN_SMS_ENABLED` | true/false | true | SMS-Anmeldekanal |
| `LOGIN_EMAIL_ENABLED` | true/false | true | E-Mail-Anmeldekanal |
| `LOGIN_DINGTALK_REQUIRE_USERID` | true/false | false | Nur explizite Warden-Userid für DingTalk verlangen |
| `LOGIN_UNIFORM_RESPONSES` | true/false | false | Gleiche Antworten von `/_send_verify_code` und der Warden-Anmeldung, ob die Kennung in Warden gelistet ist oder nicht; die Antwortzeit wird nicht angeglichen |
| `LOGIN_UNIFORM_RESEND_COOLDOWN` | Dauer | `60s` | Wartezeit für erneutes Senden je Kennung bei `LOGIN_UNIFORM_RESPONSES=true`; mindestens `1s` und nicht kürzer als `RESEND_COOLDOWN` von Herald |
| `SESSION_STORAGE_ENABLED` | true/false | false | Redis-Sitzungsspeicher |
| `SESSION_STORAGE_REDIS_ADDR` | host:port | `localhost:6379` | Redis-Adresse |
| `SESSION_STORAGE_REDIS_PASSWORD` | Geheimnis | leer | Redis-Passwort |
| `SESSION_STORAGE_REDIS_DB` | Ganzzahl | 0 | Redis-Datenbank |
| `SESSION_STORAGE_REDIS_KEY_PREFIX` | String | `stargate:session:` | Redis-Schlüsselpräfix |
| `SESSION_STORAGE_REDIS_USERNAME` | String | leer | Redis-ACL-Benutzer; erfordert `SESSION_STORAGE_REDIS_PASSWORD` |
| `SESSION_STORAGE_REDIS_TLS_ENABLED` | true/false | false | TLS für die Redis-Verbindung mit Prüfung des Serverzertifikats |
| `SESSION_STORAGE_REDIS_TLS_CA_CERT_FILE` | Pfad | leer | Benutzerdefinierte CA für Redis; erfordert `SESSION_STORAGE_REDIS_TLS_ENABLED=true` |
| `SESSION_STORAGE_REDIS_TLS_SERVER_NAME` | String | leer | Erwarteter TLS-Servername für Redis; erfordert `SESSION_STORAGE_REDIS_TLS_ENABLED=true` |
| `AUDIT_LOG_ENABLED` | true/false | true | Audit-Protokollierung |
| `AUDIT_LOG_FORMAT` | json/text | json | Audit-Format |
| `STEP_UP_ENABLED` | true/false | false | Zusätzliche Authentifizierung |
| `STEP_UP_PATHS` | Pfad-Liste | leer | Geschützte Geschäftspfade |
| `OTLP_ENABLED` | true/false | false | OpenTelemetry-Export |
| `OTLP_ENDPOINT` | URL | leer | OTLP-Endpunkt |
| `AUTH_REFRESH_ENABLED` | true/false | true | Regelmäßige Autorisierungsaktualisierung |
| `AUTH_REFRESH_INTERVAL` | Dauer | `5m` | Aktualisierungsintervall |
| `REQUEST_CONTEXT_TIMEOUT` | Dauer | `10s` | Request-Deadline für Upstream-Aufrufe; keine Garantie für Abbruch bei Client-Trennung |
| `RATE_LIMIT_LOGIN_MAX` | Ganzzahl | 10 | Anmeldeversuche pro Fenster und Client; 0 deaktiviert |
| `RATE_LIMIT_VERIFICATION_MAX` | Ganzzahl | 5 | Verifizierungsanfragen pro Fenster und Client; 0 deaktiviert |
| `RATE_LIMIT_WINDOW` | Dauer | `1m` | Festes Ratenbegrenzungsfenster |

`LOGIN_DINGTALK_REQUIRE_USERID`: Standard: `false` (Kompatibilitätsmodus). Bei `true` ist nur die serverseitig von Warden verwaltete `dingtalk_userid` zulässig: keine Telefonnummer als Ersatz, und fehlende oder ungültige Zuordnungen werden abgelehnt. Dieselbe Regel gilt beim Anfordern eines Codes und beim Prüfen einer bestehenden Bindung. Dies aktiviert die Option nicht automatisch und ändert weder bestehende Sitzungen noch die Cache-/Widerrufsgrenzen. Siehe das [englische Userid-Profil](../enUS/DINGTALK_USERID_ONLY.md).

`PORT` gilt für Entwicklung und Produktion. Ohne Umgebungswert nutzt die native Binärdatei `:80`; offizielle Images setzen `PORT=8080`. In Containern nur eine Zahl verwenden, etwa `PORT=8080`: Der Healthcheck baut daraus eine URL, daher ist die native Schreibweise `:8080` dort ungeeignet.

Warden verlangt HMAC-Key-ID und Secret gemeinsam. Bei Herald muss die Key-ID für einen Server mit `HERALD_HMAC_KEYS` explizit gesetzt sein; ein einzelnes `HMAC_SECRET` unterstützt den impliziten Standard. mTLS-Clientzertifikat und Schlüssel müssen für beide Dienste ein vollständiges Paar bilden.

Ist `WARDEN_URL` oder `HERALD_URL` eine `http://`-URL und ein API-Key oder HMAC-Secret konfiguriert, protokolliert Stargate beim Start eine Warnung: Ein API-Key sowie alle Anfrage- und Antwortdaten werden dann unverschlüsselt über das Netzwerk übertragen. Verwenden Sie `https://`, sofern die Verbindung zu Warden bzw. Herald nicht in einem privaten Netzwerk bleibt, etwa einem Compose-internen.

`WARDEN_CACHE_TTL` wird nur aus Kompatibilitätsgründen akzeptiert und hat keine Wirkung auf das Verhalten von Stargate. Der Wert wird dem Warden-Client als Lebensdauer seines Benutzerlisten-Caches übergeben, Stargate fordert die Benutzerliste jedoch nie an: Anmeldung, Code-Versand, vertrauenswürdige Header-Authentifizierung und die Autorisierungsaktualisierung schlagen jeweils einen einzelnen Benutzer per Telefonnummer oder E-Mail-Adresse nach, und diese Abfragen werden nicht zwischengespeichert. Jede davon ist eine Anfrage an Warden. Wie schnell ein deaktivierter oder entfernter Benutzer eine bestehende Sitzung verliert und wie oft eine Sitzung eine Warden-Anfrage auslöst, bestimmt `AUTH_REFRESH_INTERVAL`; ein längeres Intervall verringert die Anfragen an Warden, verzögert aber auch die Wirkung einer Deaktivierung. Mit `HEADER_AUTH_ENABLED=true` löst jede weitergeleitete Anfrage mit einem vertrauenswürdigen Identitäts-Header eine Warden-Abfrage aus. Der Wert muss weiterhin eine positive Ganzzahl sein, sonst startet Stargate nicht.

Die `SESSION_STORAGE_REDIS_*`-Einstellungen wirken nur mit `SESSION_STORAGE_ENABLED=true`. Ist eine davon auf einen Nicht-Standardwert gesetzt, während `SESSION_STORAGE_ENABLED` `false` ist, startet Stargate, hält Sitzungen und Rate-Limit-Zähler im Arbeitsspeicher und protokolliert beim Start eine Warnung, die die ignorierten Variablen auflistet.

Der Benutzer aus `SESSION_STORAGE_REDIS_USERNAME` benötigt Lese-, Schreib- und Skriptrechte (`EVAL`/`EVALSHA`) für die Schlüssel unter `SESSION_STORAGE_REDIS_KEY_PREFIX`. Ohne `SESSION_STORAGE_REDIS_TLS_ENABLED=true` werden Sitzungen, Verifizierungskontexte und Rate-Limit-Zähler unverschlüsselt an Redis übertragen; verwenden Sie das nur in einem privaten Netzwerk. Mit TLS (1.2 oder neuer) wird das Zertifikat gegen die Systemwurzeln und den Host aus `SESSION_STORAGE_REDIS_ADDR` geprüft, sofern `SESSION_STORAGE_REDIS_TLS_CA_CERT_FILE` (PEM-Datei mit zusätzlichen CA-Zertifikaten) und `SESSION_STORAGE_REDIS_TLS_SERVER_NAME` nichts anderes festlegen. Die Prüfung lässt sich nicht abschalten; Clientzertifikate werden nicht unterstützt.

## Passwort-Konfiguration

Stargate akzeptiert `bcrypt` für Produktionsumgebungen und `plaintext` ausschließlich für lokale Tests. MD5 und ungesalzenes SHA-512 werden bei der Konfigurationsprüfung abgelehnt. Passwortwerte sind groß-/kleinschreibungssensitiv; Leerzeichen bleiben erhalten. Die im Compose-Beispiel dieses Repositorys veröffentlichten Demo-Passwörter (`test1234`, `test1337`) werden beim Start abgelehnt, wenn `COOKIE_SECURE` `true` ist; sie werden nur für die lokale HTTP-Demo akzeptiert.

```bash
PASSWORDS='bcrypt:<bcrypt-hash>'
# Nur lokal:
PASSWORDS='plaintext:test123'
```

## Konfigurationsbeispiele

### Grundlegende Konfiguration

```bash
# Erforderliche Konfiguration
AUTH_HOST=auth.example.com
PASSWORDS='plaintext:test123'

# Optionale Konfiguration
DEBUG=false
LANGUAGE=en
```

### Produktionskonfiguration

```bash
# Erforderliche Konfiguration
AUTH_HOST=auth.example.com
PASSWORDS='bcrypt:$2a$10$N9qo8uLOickgx2ZMRZoMyeIjZAgcfl7p92ldGxad68LJZdL17lhWy'

# Optionale Konfiguration
DEBUG=false
LANGUAGE=de
LOGIN_PAGE_TITLE='Mein Authentifizierungsdienst'
LOGIN_PAGE_FOOTER_TEXT='© 2024 Meine Firma'
USER_HEADER_NAME=X-Forwarded-User
COOKIE_DOMAIN=.example.com
```

### Docker Compose Konfiguration

```yaml
services:
  stargate:
    image: ghcr.io/soulteary/stargate:1.3.8
    environment:
      # Erforderliche Konfiguration
      - AUTH_HOST=auth.example.com
      - PASSWORDS=bcrypt:$$2a$$10$$N9qo8uLOickgx2ZMRZoMyeIjZAgcfl7p92ldGxad68LJZdL17lhWy
      
      # Optionale Konfiguration
      - DEBUG=false
      - LANGUAGE=de
      - LOGIN_PAGE_TITLE=Mein Authentifizierungsdienst
      - LOGIN_PAGE_FOOTER_TEXT=© 2024 Meine Firma
      - COOKIE_DOMAIN=.example.com
```

### Lokale Entwicklungskonfiguration

```bash
# Erforderliche Konfiguration
AUTH_HOST=localhost
PASSWORDS='plaintext:test123|admin456'

# Optionale Konfiguration
DEBUG=true
LANGUAGE=de
PORT=8080
```

## Konfigurationsvalidierung

Stargate validiert alle Konfigurationselemente beim Start:

1. **Erforderliche Konfigurationsprüfung**: Wenn die erforderliche Konfiguration nicht definiert ist, schlägt der Dienst beim Start fehl und zeigt eine Fehlermeldung an
2. **Format-Validierung**: Ein falsches Passwort-Konfigurationsformat führt zu einem Startfehler
3. **Algorithmus-Validierung**: Nicht unterstützte Passwort-Algorithmen führen zu einem Startfehler
4. **Wert-Validierung**: Einige Konfigurationselemente haben Wertbeschränkungen (z. B. `LANGUAGE`, `DEBUG`)
5. **Regel-Validierung**: Einige Konfigurationselemente hängen von anderen ab oder haben eine Mindestlänge oder ein vorgeschriebenes Format (z. B. erfordert `HERALD_ENABLED` den Wert `WARDEN_ENABLED=true`)

Die Meldung nennt die Variable und gibt an, welcher dieser Fälle vorliegt. Eine Regel wird als Grund gemeldet; sie wird nicht als konfigurierter Wert angezeigt, und konfigurierte Werte von Geheimnissen werden nie ausgegeben.

**Fehlerbeispiele:**

```bash
# Fehlende erforderliche Konfiguration
Error: Configuration error: environment variable 'AUTH_HOST' is required but not set.

# Falsches Passwort-Format
Error: Configuration error: invalid value for environment variable 'PASSWORDS': 'invalid_format'

# Nicht unterstützter Algorithmus
Error: Configuration error: invalid value for environment variable 'PASSWORDS': 'unknown:password'

# Wert außerhalb der akzeptierten Menge
Error: Configuration error: invalid value for environment variable 'DEBUG': 'maybe'.
  Accepted values: [true false]

# Regel, die eine andere Einstellung betrifft
Error: Configuration error: environment variable 'HERALD_ENABLED': requires WARDEN_ENABLED=true to resolve authenticated users.
```

## Best Practices für die Konfiguration

1. **Produktionssicherheit**:
   - Verwenden Sie `bcrypt` in Produktion; `plaintext` nur lokal
   - Setzen Sie `DEBUG=false`
   - Verwenden Sie starke Passwörter

2. **Cross-Domain-Sitzungen**:
   - Wenn Sie Sitzungen zwischen Subdomains teilen müssen, setzen Sie `COOKIE_DOMAIN`
   - Format: `.example.com` (beachten Sie den anfänglichen Punkt)

3. **Mehrsprachige Unterstützung**:
   - Setzen Sie `LANGUAGE` entsprechend der Benutzerbasis
   - Unterstützt `en`, `zh`, `fr`, `it`, `ja`, `de`, `ko`

4. **Angepasste Oberfläche**:
   - Verwenden Sie `LOGIN_PAGE_TITLE` und `LOGIN_PAGE_FOOTER_TEXT`, um die Anmeldeseite anzupassen

5. **Überwachung und Debugging**:
   - Setzen Sie `DEBUG=true` in der Entwicklungsumgebung für detaillierte Protokolle
   - Setzen Sie `DEBUG=false` in der Produktionsumgebung, um die Protokollausgabe zu reduzieren

## Legacy-Authentifizierung per Passwort-Header

Die `Stargate-Password`-Authentifizierung ist standardmäßig deaktiviert. Aktivieren Sie sie nur für vertrauenswürdige Legacy-Clients mit `PASSWORD_HEADER_AUTH_ENABLED=true`. Verwenden Sie HTTPS und Rate Limits und entfernen Sie den Header im Reverse Proxy, bevor die Anfrage an den Backend-Dienst weitergeleitet wird.

Fehlgeschlagene Header-Versuche auf `/_auth` werden pro Client-Adresse auf `RATE_LIMIT_LOGIN_MAX` angerechnet. Nur falsche Werte bleiben gezählt. Ist das Kontingent aufgebraucht, wird jede Anfrage dieser Adresse, die den Header enthält, bis zum Ende des Fensters mit `429` und `Retry-After` beantwortet, bevor das Passwort verglichen wird, also auch mit dem richtigen Passwort. Eine Anfrage wird gezählt, während ihr Passwort verglichen wird, und bei richtigem Passwort wieder abgezogen; gleichzeitig gesendete falsche Werte können das Kontingent daher nicht gemeinsam überschreiten. Aus demselben Grund erhält ein Client, der mehr Anfragen mit dem Header gleichzeitig offen hat, als das Kontingent noch zulässt, für die überzähligen `429` und kann sie wiederholen. Clients mit derselben Adresse teilen sich dieses Kontingent:

- Clients hinter einem gemeinsamen NAT oder ausgehenden Proxy haben eine Adresse; sendet einer von ihnen falsche Werte, sperrt er die anderen für den Rest des Fensters aus und kann das in jedem Fenster wiederholen;
- ist `TRUSTED_PROXIES` leer oder steht Stargate hinter einem Proxy oder CDN, das dort nicht aufgeführt ist, wird jede Anfrage dem Proxy zugeordnet, sodass dies für alle Header-Clients der Installation gilt. Stargate protokolliert beim Start eine Warnung für `PASSWORD_HEADER_AUTH_ENABLED=true` mit leerem `TRUSTED_PROXIES`.

Setzen Sie `TRUSTED_PROXIES`, damit das Kontingent pro Client gilt. Cookie-Sitzungen sind von diesem Kontingent nicht betroffen.

## Quellenalter und Monitoring

`WARDEN_SOURCE_MAX_AGE=0s` bleibt aus Kompatibilitätsgründen der Standard.
Ein positiver Wert erfordert Warden-Snapshot-Metadaten v1 und
`AUTH_REFRESH_ENABLED=true`. `WARDEN_SOURCE_CLOCK_SKEW` wird von der Gültigkeit
abgezogen. Abgelaufene Quelldaten werden auch bei noch gültigen Sitzungen erneut
geprüft; bei Quellenfehlern wird der Zugriff mit HTTP 503 abgelehnt.

`/metrics` liefert ohne `METRICS_API_KEY` HTTP 404. Vor dem Update einen separaten,
zufälligen Schlüssel mit mindestens 32 Bytes einrichten und den Scraper auf
`Authorization: Bearer <secret>` umstellen. Keine Anmelde- oder Dienstschlüssel
wiederverwenden. Für Metadaten zwischen Hosts verifiziertes HTTPS verwenden;
HTTP setzt ein vertrauenswürdiges, isoliertes lokales Netz voraus.
Die vollständigen Regeln und Migrationshinweise stehen in der
[englischen Konfigurationsreferenz](../enUS/CONFIG.md#source-freshness-and-scrape-credentials).


### `LOGIN_SUPPORT_URL`

Optionaler HTTPS-Supportlink oder ein einzelnes `mailto:help@example.com`-Postfach. Leer blendet den Link aus. Zugangsdaten, Steuerzeichen und Mailto-Header sind nicht erlaubt.
