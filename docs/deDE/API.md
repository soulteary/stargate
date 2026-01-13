# API-Dokumentation

Dieses Dokument beschreibt im Detail alle API-Endpunkte des Stargate Forward Auth-Dienstes.

## Inhaltsverzeichnis

- [Authentifizierungsprüfungs-Endpunkt](#authentifizierungsprüfungs-endpunkt)
- [Anmeldungs-Endpunkt](#anmeldungs-endpunkt)
- [Abmelde-Endpunkt](#abmelde-endpunkt)
- [Sitzungsaustausch-Endpunkt](#sitzungsaustausch-endpunkt)
- [Gesundheitsprüfungs-Endpunkt](#gesundheitsprüfungs-endpunkt)
- [Root-Endpunkt](#root-endpunkt)

## Authentifizierungsprüfungs-Endpunkt

### `GET /_auth`

Der Haupt-Endpunkt für die Authentifizierungsprüfung für Traefik Forward Auth. Dieser Endpunkt ist die Hauptfunktion von Stargate und wird verwendet, um zu überprüfen, ob ein Benutzer authentifiziert wurde.

#### Authentifizierungsmethoden

Stargate unterstützt zwei Authentifizierungsmethoden, die in folgender Prioritätsreihenfolge überprüft werden:

1. **Header-Authentifizierung** (API-Anfragen)
   - Anfrage-Header: `Stargate-Password: <password>`
   - Geeignet für API-Anfragen, Automatisierungsskripte usw.

2. **Cookie-Authentifizierung** (Web-Anfragen)
   - Cookie: `stargate_session_id=<session_id>`
   - Geeignet für Web-Anwendungen, die über Browser zugänglich sind

#### Anfrage-Header

| Header | Typ | Erforderlich | Beschreibung |
|--------|-----|--------------|--------------|
| `Stargate-Password` | String | Nein | Passwort-Authentifizierung für API-Anfragen |
| `Cookie` | String | Nein | Sitzungs-Cookie mit `stargate_session_id` |
| `Accept` | String | Nein | Wird verwendet, um den Anfragetyp (HTML/API) zu bestimmen |

#### Antwort

**Erfolgreiche Antwort (200 OK)**

Wenn die Authentifizierung erfolgreich ist, setzt Stargate den Benutzerinformations-Header und gibt einen Statuscode 200 zurück:

```
HTTP/1.1 200 OK
X-Forwarded-User: authenticated
```

Der Name des Benutzer-Headers kann über die Umgebungsvariable `USER_HEADER_NAME` konfiguriert werden (Standard: `X-Forwarded-User`).

**Fehlerantwort**

| Statuscode | Beschreibung | Antwort-Text |
|------------|--------------|--------------|
| `401 Unauthorized` | Authentifizierung fehlgeschlagen | Fehlermeldung (JSON-Format für API-Anfragen) oder Weiterleitung zur Anmeldeseite (HTML-Anfragen) |
| `500 Internal Server Error` | Server-Fehler | Fehlermeldung |

#### Behandlung des Anfragetyps

- **HTML-Anfragen**: Weiterleitung zu `/_login?callback=<originalURL>` bei fehlgeschlagener Authentifizierung
- **API-Anfragen** (JSON/XML): Rückgabe einer 401-Fehlerantwort bei fehlgeschlagener Authentifizierung

#### Beispiele

**Verwendung der Header-Authentifizierung (API-Anfrage)**

```bash
curl -H "Stargate-Password: yourpassword" \
     http://auth.example.com/_auth
```

**Verwendung der Cookie-Authentifizierung (Web-Anfrage)**

```bash
curl -H "Cookie: stargate_session_id=<session_id>" \
     http://auth.example.com/_auth
```

## Anmeldungs-Endpunkt

### `GET /_login`

Zeigt die Anmeldeseite an.

#### Abfrageparameter

| Parameter | Typ | Erforderlich | Beschreibung |
|-----------|-----|--------------|--------------|
| `callback` | String | Nein | Callback-URL nach erfolgreicher Anmeldung (normalerweise die Domain der ursprünglichen Anfrage) |

#### Verhalten

- Wenn der Benutzer bereits angemeldet ist, wird automatisch zum Sitzungsaustausch-Endpunkt weitergeleitet
- Wenn der Benutzer nicht angemeldet ist, wird die Anmeldeseite angezeigt
- Wenn die URL einen Parameter `callback` enthält und die Domain unterschiedlich ist, wird der Callback im Cookie `stargate_callback` gespeichert (läuft nach 10 Minuten ab)

#### Callback-Abrufpriorität

1. **Aus den Abfrageparametern**: Der Parameter `callback` in der URL (höchste Priorität)
2. **Aus dem Cookie**: Wenn nicht in den Abfrageparametern vorhanden, aus dem Cookie `stargate_callback` abrufen

#### Antwort

**200 OK** - Gibt das HTML der Anmeldeseite zurück

Die Seite enthält:
- Anmeldeformular
- Anpassbarer Titel (`LOGIN_PAGE_TITLE`)
- Anpassbarer Fußzeilentext (`LOGIN_PAGE_FOOTER_TEXT`)

#### Beispiel

```bash
# Auf die Anmeldeseite zugreifen
curl http://auth.example.com/_login?callback=app.example.com
```

### `POST /_login`

Verarbeitet Anmeldeanfragen, überprüft das Passwort und erstellt eine Sitzung.

#### Anfrage-Text

Formulardaten (`application/x-www-form-urlencoded`):

| Feld | Typ | Erforderlich | Beschreibung |
|------|-----|--------------|--------------|
| `password` | String | Ja | Benutzerpasswort |
| `callback` | String | Nein | Callback-URL nach erfolgreicher Anmeldung |

#### Callback-Abrufpriorität

Die Anmeldeverarbeitung ruft den Callback in folgender Prioritätsreihenfolge ab:

1. **Aus dem Cookie**: Wenn die Domain beim vorherigen Zugriff auf die Anmeldeseite unterschiedlich war, wird der Callback im Cookie `stargate_callback` gespeichert
2. **Aus den Formulardaten**: Das Feld `callback` in den Formulardaten der POST-Anfrage
3. **Aus den Abfrageparametern**: Der `callback` in den Abfrageparametern der URL
4. **Automatische Inferenz**: Wenn keines der oben genannten vorhanden ist und die ursprüngliche Domain (`X-Forwarded-Host`) sich von der Domain des Authentifizierungsdienstes unterscheidet, die ursprüngliche Domain als Callback verwenden

#### Antwort

**Erfolgreiche Antwort (200 OK)**

Die Antwort variiert je nachdem, ob ein Callback vorhanden ist und welcher Anfragetyp:

1. **Mit Callback**:
   - Weiterleitung zu `{callback}/_session_exchange?id={session_id}`
   - Statuscode: `302 Found`

2. **Ohne Callback**:
   - **HTML-Anfrage**: Gibt eine HTML-Seite mit Meta-Refresh zurück, die automatisch zur ursprünglichen Domain weiterleitet
   - **API-Anfrage**: Gibt eine JSON-Antwort zurück
     ```json
     {
       "success": true,
       "message": "Login successful",
       "session_id": "<session_id>"
     }
     ```

**Fehlerantwort**

| Statuscode | Beschreibung | Antwort-Text |
|------------|--------------|--------------|
| `401 Unauthorized` | Falsches Passwort | Fehlermeldung im JSON/XML/Text-Format je nach Accept-Header |
| `500 Internal Server Error` | Server-Fehler | Fehlermeldung |

#### Beispiele

```bash
# Anmeldeformular absenden (mit Callback)
curl -X POST \
     -d "password=yourpassword&callback=app.example.com" \
     -c cookies.txt \
     http://auth.example.com/_login

# Anmeldeformular absenden (ohne Callback, wird automatisch inferiert)
curl -X POST \
     -d "password=yourpassword" \
     -H "X-Forwarded-Host: app.example.com" \
     -c cookies.txt \
     http://auth.example.com/_login
```

## Abmelde-Endpunkt

### `GET /_logout`

Meldet den aktuellen Benutzer ab und zerstört die Sitzung.

#### Antwort

**Erfolgreiche Antwort (200 OK)**

```
HTTP/1.1 200 OK
Content-Type: text/plain

Logged out
```

Das Sitzungs-Cookie wird gelöscht.

#### Beispiel

```bash
curl -b cookies.txt http://auth.example.com/_logout
```

## Sitzungsaustausch-Endpunkt

### `GET /_session_exchange`

Wird für die Cross-Domain-Sitzungsfreigabe verwendet. Setzt das angegebene Sitzungs-ID-Cookie und leitet zum Root-Pfad weiter.

Dieser Endpunkt wird hauptsächlich verwendet, um Authentifizierungssitzungen zwischen mehreren Domains/Subdomains zu teilen. Nachdem ein Benutzer sich auf einer Domain angemeldet hat, kann dieser Endpunkt verwendet werden, um das Sitzungs-Cookie auf einer anderen Domain zu setzen.

#### Abfrageparameter

| Parameter | Typ | Erforderlich | Beschreibung |
|-----------|-----|--------------|--------------|
| `id` | String | Ja | Zu setzende Sitzungs-ID |

#### Antwort

**Erfolgreiche Antwort (302 Redirect)**

```
HTTP/1.1 302 Found
Location: /
Set-Cookie: stargate_session_id=<session_id>; Path=/; HttpOnly; SameSite=Lax; Domain=<cookie_domain>; Expires=<expiry>
```

**Fehlerantwort**

| Statuscode | Beschreibung | Antwort-Text |
|------------|--------------|--------------|
| `400 Bad Request` | Sitzungs-ID fehlt | Fehlermeldung |

#### Cookie-Domain

Wenn die Umgebungsvariable `COOKIE_DOMAIN` konfiguriert ist, wird das Cookie auf die angegebene Domain gesetzt, was die Cross-Subdomain-Freigabe ermöglicht.

#### Beispiel

```bash
# Sitzungs-Cookie setzen (für Cross-Domain-Szenarien)
curl "http://auth.example.com/_session_exchange?id=<session_id>"
```

**Typisches Verwendungsszenario:**

1. Der Benutzer meldet sich bei `auth.example.com` an
2. Nach erfolgreicher Anmeldung wird zu `app.example.com/_session_exchange?id=<session_id>` weitergeleitet
3. Das Sitzungs-Cookie wird auf die Domain `.example.com` gesetzt (wenn `COOKIE_DOMAIN=.example.com` konfiguriert ist)
4. Weiterleitung zu `app.example.com/`
5. Der Benutzer kann diese Sitzung auf allen Subdomains `*.example.com` verwenden

## Gesundheitsprüfungs-Endpunkt

### `GET /health`

Gesundheitsprüfungs-Endpunkt des Dienstes. Wird verwendet, um den Status des Dienstes zu überwachen.

#### Antwort

**Erfolgreiche Antwort (200 OK)**

```
HTTP/1.1 200 OK
```

#### Beispiel

```bash
curl http://auth.example.com/health
```

**Typische Verwendungen:**

- Docker-Gesundheitsprüfungen
- Kubernetes-Liveness-Proben
- Load-Balancer-Gesundheitsprüfungen

## Root-Endpunkt

### `GET /`

Root-Pfad, zeigt die Dienstinformationen an.

#### Antwort

**200 OK** - Gibt die Dienstinformationsseite zurück

#### Beispiel

```bash
curl http://auth.example.com/
```

## Fehlerantwort-Format

Alle API-Fehlerantworten wählen automatisch das Format basierend auf dem `Accept`-Header des Clients:

### JSON-Format (`Accept: application/json`)

```json
{
  "error": "Error message",
  "code": 401
}
```

### XML-Format (`Accept: application/xml`)

```xml
<errors>
  <error code="401">Error message</error>
</errors>
```

### Text-Format (Standard)

```
Error message
```

Fehlermeldungen unterstützen Internationalisierung und geben je nach Umgebungsvariable `LANGUAGE` Nachrichten auf Chinesisch oder Englisch zurück.

## Authentifizierungsfluss-Beispiele

### Web-Anwendungs-Authentifizierungsfluss

1. Der Benutzer greift auf eine geschützte Ressource zu (z. B. `https://app.example.com/dashboard`)
2. Traefik fängt die Anfrage ab und leitet sie an `https://auth.example.com/_auth` weiter
3. Stargate überprüft die Sitzung im Cookie
4. Wenn nicht authentifiziert, Weiterleitung zu `https://auth.example.com/_login?callback=app.example.com`
5. Der Benutzer gibt das Passwort ein und sendet es ab
6. Stargate überprüft das Passwort, erstellt eine Sitzung, setzt das Cookie
7. Weiterleitung zu `https://app.example.com/_session_exchange?id=<session_id>`
8. Das Sitzungs-Cookie wird auf die Domain `app.example.com` gesetzt
9. Der Benutzer greift erneut auf die geschützte Ressource zu, die Authentifizierung ist erfolgreich

### API-Authentifizierungsfluss

1. Der API-Client sendet eine Anfrage an eine geschützte Ressource
2. Traefik fängt die Anfrage ab und leitet sie an `https://auth.example.com/_auth` weiter
3. Der API-Client fügt `Stargate-Password: <password>` in den Anfrage-Header ein
4. Stargate überprüft das Passwort
5. Wenn die Überprüfung erfolgreich ist, setzt den Header `X-Forwarded-User` und gibt 200 zurück
6. Traefik erlaubt der Anfrage, zum Backend-Dienst fortzufahren

## Hinweise

1. **Sitzungsablaufzeit**: Standardmäßig 24 Stunden, erfordert eine erneute Anmeldung nach Ablauf
2. **Cookie-Sicherheit**: Alle Cookies werden mit den Flags `HttpOnly` und `SameSite=Lax` gesetzt
3. **Passwort-Überprüfung**: Passwörter werden vor der Überprüfung normalisiert (Leerzeichen entfernen, in Großbuchstaben umwandeln)
4. **Unterstützung mehrerer Passwörter**: Mehrere Passwörter können konfiguriert werden, jedes Passwort, das die Überprüfung besteht, ist akzeptabel
5. **Cross-Domain-Sitzungen**: Die Umgebungsvariable `COOKIE_DOMAIN` muss konfiguriert werden, um die Cross-Domain-Sitzungsfreigabe zu aktivieren
