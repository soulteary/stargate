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
docker run -e AUTH_HOST=auth.example.com -e PASSWORDS=plaintext:yourpassword ghcr.io/soulteary/stargate:1.3.5
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
| `WARDEN_CACHE_TTL` | Sekunden | 300 | Warden-Cache-Laufzeit |
| `WARDEN_SOURCE_MAX_AGE` | Dauer | 0s | Maximales Alter der Warden-Quelldaten; 0s deaktiviert die Prüfung |
| `WARDEN_SOURCE_CLOCK_SKEW` | Dauer | 0s | Tolerierte Zeitabweichung; kleiner als das maximale Alter |
| `METRICS_API_KEY` | Geheimnis | leer | Separater Bearer-Schlüssel für /metrics, mindestens 32 Bytes |
| `HERALD_ENABLED` | true/false | false | Herald-Integration |
| `HERALD_URL` | URL | leer | Herald-Endpunkt |
| `HERALD_API_KEY` | Geheimnis | leer | API-Key-Authentifizierung |
| `HERALD_HMAC_KEY_ID` | String | leer | HMAC-Schlüsselkennung |
| `HERALD_HMAC_SECRET` | Geheimnis | leer | HMAC-Signaturgeheimnis |
| `HERALD_TLS_CA_CERT_FILE` | Pfad | leer | Benutzerdefinierte CA |
| `HERALD_TLS_CLIENT_CERT_FILE` | Pfad | leer | mTLS-Clientzertifikat |
| `HERALD_TLS_CLIENT_KEY_FILE` | Pfad | leer | mTLS-Clientschlüssel |
| `HERALD_TLS_SERVER_NAME` | String | leer | Erwarteter TLS-Servername |
| `HERALD_TOTP_ENABLED` | true/false | false | TOTP-Anmeldung und -Verwaltung |
| `LOGIN_SMS_ENABLED` | true/false | true | SMS-Anmeldekanal |
| `LOGIN_EMAIL_ENABLED` | true/false | true | E-Mail-Anmeldekanal |
| `SESSION_STORAGE_ENABLED` | true/false | false | Redis-Sitzungsspeicher |
| `SESSION_STORAGE_REDIS_ADDR` | host:port | `localhost:6379` | Redis-Adresse |
| `SESSION_STORAGE_REDIS_PASSWORD` | Geheimnis | leer | Redis-Passwort |
| `SESSION_STORAGE_REDIS_DB` | Ganzzahl | 0 | Redis-Datenbank |
| `SESSION_STORAGE_REDIS_KEY_PREFIX` | String | `stargate:session:` | Redis-Schlüsselpräfix |
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

`PORT` gilt für Entwicklung und Produktion. Ohne Umgebungswert nutzt die native Binärdatei `:80`; offizielle Images setzen `PORT=8080`. In Containern nur eine Zahl verwenden, etwa `PORT=8080`: Der Healthcheck baut daraus eine URL, daher ist die native Schreibweise `:8080` dort ungeeignet.

Warden verlangt HMAC-Key-ID und Secret gemeinsam. Bei Herald muss die Key-ID für einen Server mit `HERALD_HMAC_KEYS` explizit gesetzt sein; ein einzelnes `HMAC_SECRET` unterstützt den impliziten Standard. mTLS-Clientzertifikat und Schlüssel müssen für beide Dienste ein vollständiges Paar bilden.

## Passwort-Konfiguration

Stargate akzeptiert `bcrypt` für Produktionsumgebungen und `plaintext` ausschließlich für lokale Tests. MD5 und ungesalzenes SHA-512 werden bei der Konfigurationsprüfung abgelehnt. Passwortwerte sind groß-/kleinschreibungssensitiv; Leerzeichen bleiben erhalten.

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
    image: ghcr.io/soulteary/stargate:1.3.5
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

**Fehlerbeispiele:**

```bash
# Fehlende erforderliche Konfiguration
Error: Configuration error: environment variable 'AUTH_HOST' is required but not set.

# Falsches Passwort-Format
Error: Configuration error: invalid value for environment variable 'PASSWORDS': 'invalid_format'

# Nicht unterstützter Algorithmus
Error: Configuration error: invalid value for environment variable 'PASSWORDS': 'unknown:password'
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
