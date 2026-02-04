# Konfigurationsreferenz

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
export PASSWORDS=plaintext:yourpassword
```

**Docker:**

```bash
docker run -e AUTH_HOST=auth.example.com -e PASSWORDS=plaintext:yourpassword stargate:latest
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
PASSWORDS=plaintext:test123

# Mehrere Klartext-Passwörter
PASSWORDS=plaintext:test123|admin456|user789

# BCrypt-Hash
PASSWORDS=bcrypt:$2a$10$N9qo8uLOickgx2ZMRZoMyeIjZAgcfl7p92ldGxad68LJZdL17lhWy

# SHA512-Hash
PASSWORDS=sha512:5e884898da28047151d0e56f8dc6292773603d0d6aabbdd62a11ef721d1542d8
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
LOGIN_PAGE_TITLE=Mein Authentifizierungsdienst
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
LOGIN_PAGE_FOOTER_TEXT=© 2024 Meine Firma
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
- Der Wert des Headers ist `authenticated`
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

### `PORT`

Abhörport des Dienstes (nur für lokale Entwicklung). Wird wie andere Optionen über die config-Paket-Umgebungsvariablen geladen und validiert.

| Attribut | Wert |
|----------|------|
| **Typ** | String |
| **Erforderlich** | Nein |
| **Standard** | Leer (wenn leer, verwendet der Server den Standardport `:80`) |

**Beschreibung:**

- Nur für lokale Entwicklungsumgebung
- In Docker-Containern normalerweise nicht erforderlich (verwendet Standard-Port 80)
- Format: Portnummer (z. B. `8080`) oder `:port` (z. B. `:8080`)

**Beispiel:**

```bash
PORT=8080
```

## Passwort-Konfiguration

Stargate unterstützt mehrere Passwort-Verschlüsselungsalgorithmen. Passwort-Konfigurationsformat: `algorithm:password1|password2|password3`

### Unterstützte Algorithmen

#### `plaintext` - Klartext-Passwort

**Beschreibung:**

- Im Klartext gespeichert, keine Verschlüsselung
- **Nur für Testumgebung**
- Für Produktion stark nicht empfohlen

**Beispiel:**

```bash
PASSWORDS=plaintext:test123|admin456
```

#### `bcrypt` - BCrypt-Hash

**Beschreibung:**

- Verwendet BCrypt-Algorithmus für Hashing
- Hohe Sicherheit, für Produktion empfohlen
- Passwort muss BCrypt-Hash-Wert verwenden

**BCrypt-Hash generieren:**

```bash
# Mit Go
go run -c 'golang.org/x/crypto/bcrypt' <<< 'password'

# Mit Online-Tools oder anderen Tools
```

**Beispiel:**

```bash
PASSWORDS=bcrypt:$2a$10$N9qo8uLOickgx2ZMRZoMyeIjZAgcfl7p92ldGxad68LJZdL17lhWy
```

#### `md5` - MD5-Hash

**Beschreibung:**

- Verwendet MD5-Algorithmus für Hashing
- Niedrigere Sicherheit, für Produktion nicht empfohlen
- Passwort muss MD5-Hash-Wert verwenden (32-stellige Hexadezimalzeichenkette)

**MD5-Hash generieren:**

```bash
# Linux/macOS
echo -n "password" | md5sum

# Oder Online-Tools verwenden
```

**Beispiel:**

```bash
PASSWORDS=md5:5e884898da28047151d0e56f8dc6292773603d0d6aabbdd62a11ef721d1542d8
```

#### `sha512` - SHA512-Hash

**Beschreibung:**

- Verwendet SHA512-Algorithmus für Hashing
- Hohe Sicherheit, für Produktion empfohlen
- Passwort muss SHA512-Hash-Wert verwenden (128-stellige Hexadezimalzeichenkette)

**SHA512-Hash generieren:**

```bash
# Linux/macOS
echo -n "password" | shasum -a 512

# Oder Online-Tools verwenden
```

**Beispiel:**

```bash
PASSWORDS=sha512:5e884898da28047151d0e56f8dc6292773603d0d6aabbdd62a11ef721d1542d8
```

### Passwort-Überprüfungsregeln

1. **Passwort-Normalisierung**: Leerzeichen werden entfernt und in Großbuchstaben umgewandelt, bevor die Überprüfung erfolgt
2. **Unterstützung mehrerer Passwörter**: Mehrere Passwörter können konfiguriert werden, jedes Passwort, das die Überprüfung besteht, ist akzeptabel
3. **Algorithmus-Konsistenz**: Alle Passwörter müssen denselben Algorithmus verwenden

## Konfigurationsbeispiele

### Grundlegende Konfiguration

```bash
# Erforderliche Konfiguration
AUTH_HOST=auth.example.com
PASSWORDS=plaintext:test123

# Optionale Konfiguration
DEBUG=false
LANGUAGE=en
```

### Produktionskonfiguration

```bash
# Erforderliche Konfiguration
AUTH_HOST=auth.example.com
PASSWORDS=bcrypt:$2a$10$N9qo8uLOickgx2ZMRZoMyeIjZAgcfl7p92ldGxad68LJZdL17lhWy

# Optionale Konfiguration
DEBUG=false
LANGUAGE=de
LOGIN_PAGE_TITLE=Mein Authentifizierungsdienst
LOGIN_PAGE_FOOTER_TEXT=© 2024 Meine Firma
USER_HEADER_NAME=X-Forwarded-User
COOKIE_DOMAIN=.example.com
```

### Docker Compose Konfiguration

```yaml
services:
  stargate:
    image: stargate:latest
    environment:
      # Erforderliche Konfiguration
      - AUTH_HOST=auth.example.com
      - PASSWORDS=bcrypt:$2a$10$N9qo8uLOickgx2ZMRZoMyeIjZAgcfl7p92ldGxad68LJZdL17lhWy
      
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
PASSWORDS=plaintext:test123|admin456

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
   - Verwenden Sie die Algorithmen `bcrypt` oder `sha512`, vermeiden Sie `plaintext`
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
