# Stargate - Forward Auth Service

[![License](https://img.shields.io/badge/license-Apache%202.0-blue.svg)](LICENSE)
[![Go Version](https://img.shields.io/badge/go-1.25+-blue.svg)](https://golang.org)
[![codecov](https://codecov.io/gh/soulteary/stargate/branch/main/graph/badge.svg)](https://codecov.io/gh/soulteary/stargate)

> **🚀 Ihr Gateway zu sicheren Microservices**

Stargate ist ein produktionsreifer, leichtgewichtiger Forward Auth Service, der als **einzelner Authentifizierungspunkt** für Ihre gesamte Infrastruktur konzipiert wurde. Mit Go entwickelt und für Leistung optimiert, integriert sich Stargate nahtlos mit Traefik und anderen Reverse-Proxies, um Ihre Backend-Services zu schützen—**ohne eine einzige Zeile Authentifizierungscode in Ihren Anwendungen zu schreiben**.

## 🌐 Mehrsprachige Dokumentation

- [English](README.md) | [中文](README.zhCN.md) | [Français](README.frFR.md) | [Italiano](README.itIT.md) | [日本語](README.jaJP.md) | [Deutsch](README.deDE.md) | [한국어](README.koKR.md)

### 🎯 Warum Stargate?

Müde davon, Authentifizierungslogik in jedem Service zu implementieren? Stargate löst dies, indem es die Authentifizierung am Edge zentralisiert und Ihnen ermöglicht:

- ✅ **Mehrere Services schützen** mit einer einzigen Authentifizierungsschicht
- ✅ **Code-Komplexität reduzieren** durch Entfernen der Auth-Logik aus Ihren Anwendungen
- ✅ **In Minuten bereitstellen** mit Docker und einfacher Konfiguration
- ✅ **Mühelos skalieren** mit minimalem Ressourcen-Footprint
- ✅ **Sicherheit aufrechterhalten** mit mehreren Verschlüsselungsalgorithmen und sicherer Sitzungsverwaltung

### 💼 Anwendungsfälle

Stargate ist perfekt für:

- **Microservices-Architektur**: Mehrere Backend-Services schützen, ohne Anwendungscode zu ändern
- **Multi-Domain-Anwendungen**: Authentifizierungssitzungen über verschiedene Domains und Subdomains teilen
- **Interne Tools & Dashboards**: Schnell Authentifizierung zu internen Services und Admin-Panels hinzufügen
- **API-Gateway-Integration**: Mit Traefik, Nginx oder anderen Reverse-Proxies als einheitliche Auth-Schicht verwenden
- **Entwicklung & Testing**: Einfache passwortbasierte Authentifizierung für Entwicklungsumgebungen

## 📋 Inhaltsverzeichnis

- [Funktionen](#funktionen)
- [Schnellstart](#schnellstart)
- [Konfiguration](#konfiguration)
- [Dokumentation](#dokumentation)
- [API-Dokumentation](#api-dokumentation)
- [Bereitstellungsanleitung](#bereitstellungsanleitung)
- [Entwicklungsleitfaden](#entwicklungsleitfaden)
- [Lizenz](#lizenz)

## ✨ Funktionen

### 🔐 Unternehmensgrade Sicherheit

- **Mehrere Passwort-Verschlüsselungsalgorithmen**: Wählen Sie aus Plaintext (Test), bcrypt, MD5, SHA512 und mehr
- **Sichere Sitzungsverwaltung**: Cookie-basierte Sitzungen mit anpassbarer Domain und Ablaufzeit
- **Flexible Authentifizierung**: Unterstützung für passwortbasierte und sitzungsbasierte Authentifizierung

### 🌐 Erweiterte Fähigkeiten

- **Cross-Domain-Sitzungsteilung**: Nahtlos Authentifizierungssitzungen über verschiedene Domains/Subdomains teilen
- **Mehrsprachige Unterstützung**: Integrierte englische und chinesische Benutzeroberflächen, leicht erweiterbar für weitere Sprachen
- **Anpassbare Benutzeroberfläche**: Branden Sie Ihre Login-Seite mit benutzerdefinierten Titeln und Fußzeilentexten

### 🚀 Leistung & Zuverlässigkeit

- **Leichtgewichtig & Schnell**: Auf Go und Fiber-Framework aufgebaut für außergewöhnliche Leistung
- **Minimaler Ressourcenverbrauch**: Geringer Speicher-Footprint, perfekt für containerisierte Umgebungen
- **Produktionsbereit**: Erprobte Architektur, die für Zuverlässigkeit entwickelt wurde

### 📦 Entwicklererfahrung

- **Docker First**: Vollständiges Docker-Image und docker-compose-Konfiguration sofort einsatzbereit
- **Traefik Native**: Zero-Konfiguration Traefik Forward Auth Middleware-Integration
- **Einfache Konfiguration**: Umgebungsvariablen-basierte Konfiguration, keine komplexen Dateien erforderlich

## 🚀 Schnellstart

Stargate in **weniger als 2 Minuten** zum Laufen bringen!

### Verwendung von Docker Compose (Empfohlen)

**Schritt 1:** Repository klonen
```bash
git clone <repository-url>
cd forward-auth
```

**Schritt 2:** Authentifizierung konfigurieren (`codes/docker-compose.yml` bearbeiten)
```yaml
services:
  stargate:
    environment:
      - AUTH_HOST=auth.example.com
      - PASSWORDS=plaintext:yourpassword1|yourpassword2
```

**Schritt 3:** Service starten
```bash
cd codes
docker-compose up -d
```

**Das war's!** Ihr Authentifizierungsservice läuft jetzt. 🎉

### Lokale Entwicklung

1. Stellen Sie sicher, dass Go 1.25 oder höher installiert ist

2. Navigieren Sie zum Projektverzeichnis:
```bash
cd codes
```

3. Lokales Startskript ausführen:
```bash
chmod +x start-local.sh
./start-local.sh
```

4. Auf die Login-Seite zugreifen:
```
http://localhost:8080/_login?callback=localhost
```

## ⚙️ Konfiguration

Stargate verwendet ein einfaches, umgebungsvariablen-basiertes Konfigurationssystem. Keine komplexen YAML-Dateien oder Konfigurationsparsing—setzen Sie einfach Umgebungsvariablen und Sie sind bereit.

### Erforderliche Konfiguration

| Umgebungsvariable | Beschreibung | Beispiel |
|-------------------|--------------|----------|
| `AUTH_HOST` | Hostname des Authentifizierungsservices | `auth.example.com` |
| `PASSWORDS` | Passwort-Konfiguration, Format: `algorithm:password1\|password2\|password3` | `plaintext:test123\|admin456` |

### Optionale Konfiguration

| Umgebungsvariable | Beschreibung | Standard | Beispiel |
|-------------------|--------------|----------|----------|
| `DEBUG` | Debug-Modus aktivieren | `false` | `true` |
| `LANGUAGE` | Interface-Sprache | `en` | `de` (Deutsch), `zh` (Chinesisch), `en` (Englisch), `fr` (Französisch), `it` (Italienisch), `ja` (Japanisch), `ko` (Koreanisch) |
| `LOGIN_PAGE_TITLE` | Titel der Login-Seite | `Stargate - Login` | `Mein Auth-Service` |
| `LOGIN_PAGE_FOOTER_TEXT` | Fußzeilentext der Login-Seite | `Copyright © 2024 - Stargate` | `© 2024 Mein Unternehmen` |
| `USER_HEADER_NAME` | Benutzer-Header-Name, der nach erfolgreicher Authentifizierung gesetzt wird | `X-Forwarded-User` | `X-Authenticated-User` |
| `COOKIE_DOMAIN` | Cookie-Domain (für Cross-Domain-Sitzungsteilung) | Leer (nicht gesetzt) | `.example.com` |
| `PORT` | Service-Listening-Port (nur lokale Entwicklung) | `80` | `8080` |

### Passwort-Konfigurationsformat

Die Passwort-Konfiguration verwendet das folgende Format:
```
algorithm:password1|password2|password3
```

Unterstützte Algorithmen:
- `plaintext`: Klartext-Passwort (nur Test)
- `bcrypt`: BCrypt-Hash
- `md5`: MD5-Hash
- `sha512`: SHA512-Hash

Beispiele:
```bash
# Klartext-Passwörter (mehrere)
PASSWORDS=plaintext:test123|admin456|user789

# BCrypt-Hash
PASSWORDS=bcrypt:$2a$10$N9qo8uLOickgx2ZMRZoMyeIjZAgcfl7p92ldGxad68LJZdL17lhWy

# MD5-Hash
PASSWORDS=md5:5e884898da28047151d0e56f8dc6292773603d0d6aabbdd62a11ef721d1542d8
```

**Für detaillierte Konfiguration siehe: [docs/deDE/CONFIG.md](docs/deDE/CONFIG.md)**

## 📚 Dokumentation

Umfassende Dokumentation ist verfügbar, um Ihnen zu helfen, das Beste aus Stargate herauszuholen:

- 📐 **[Architekturdokument](docs/deDE/ARCHITECTURE.md)** - Tiefere Einblicke in technische Architektur und Designentscheidungen
- 🔌 **[API-Dokument](docs/deDE/API.md)** - Vollständige API-Endpunkt-Referenz mit Beispielen
- ⚙️ **[Konfigurationsreferenz](docs/deDE/CONFIG.md)** - Detaillierte Konfigurationsoptionen und Best Practices
- 🚀 **[Bereitstellungsanleitung](docs/deDE/DEPLOYMENT.md)** - Produktionsbereitstellungsstrategien und Empfehlungen

## 📚 API-Dokumentation

### Authentifizierungsprüfungs-Endpunkt

#### `GET /_auth`

Der Haupt-Authentifizierungsprüfungs-Endpunkt für Traefik Forward Auth.

**Anfrage-Header:**
- `Stargate-Password` (optional): Passwort-Authentifizierung für API-Anfragen
- `Cookie: stargate_session_id` (optional): Sitzungs-Authentifizierung für Web-Anfragen

**Antwort:**
- `200 OK`: Authentifizierung erfolgreich, setzt `X-Forwarded-User`-Header (oder konfigurierter Benutzer-Header-Name)
- `401 Unauthorized`: Authentifizierung fehlgeschlagen
- `500 Internal Server Error`: Serverfehler

**Hinweise:**
- HTML-Anfragen leiten bei Authentifizierungsfehler zur Login-Seite weiter
- API-Anfragen (JSON/XML) geben bei Authentifizierungsfehler 401-Fehler zurück

### Login-Endpunkt

#### `GET /_login`

Zeigt die Login-Seite an.

**Abfrageparameter:**
- `callback` (optional): Callback-URL nach erfolgreichem Login

**Antwort:**
- Gibt Login-Seiten-HTML zurück

#### `POST /_login`

Verarbeitet Login-Anfragen.

**Formulardaten:**
- `password`: Benutzerpasswort
- `callback` (optional): Callback-URL nach erfolgreichem Login

**Callback-Abrufpriorität:**
1. Vom Cookie (wenn zuvor gesetzt)
2. Von Formulardaten
3. Von Abfrageparametern
4. Wenn keines der oben genannten vorhanden ist und die Ursprungsdomain sich von der Authentifizierungsservice-Domain unterscheidet, verwenden Sie die Ursprungsdomain als Callback

**Antwort:**
- `200 OK`: Login erfolgreich
  - Wenn Callback vorhanden ist, leitet zu `{callback}/_session_exchange?id={session_id}` weiter
  - Wenn kein Callback vorhanden ist, gibt Erfolgsmeldung zurück (HTML- oder JSON-Format, abhängig vom Anfragetyp)
- `401 Unauthorized`: Falsches Passwort
- `500 Internal Server Error`: Serverfehler

### Logout-Endpunkt

#### `GET /_logout`

Meldet den aktuellen Benutzer ab und zerstört die Sitzung.

**Antwort:**
- `200 OK`: Logout erfolgreich, gibt "Logged out" zurück

### Sitzungsaustausch-Endpunkt

#### `GET /_session_exchange`

Wird für Cross-Domain-Sitzungsteilung verwendet. Setzt das angegebene Sitzungs-ID-Cookie und leitet weiter.

**Abfrageparameter:**
- `id` (erforderlich): Zu setzende Sitzungs-ID

**Antwort:**
- `302 Redirect`: Leitet zum Root-Pfad weiter
- `400 Bad Request`: Sitzungs-ID fehlt

### Gesundheitsprüfungs-Endpunkt

#### `GET /health`

Service-Gesundheitsprüfungs-Endpunkt.

**Antwort:**
- `200 OK`: Service ist gesund

### Root-Endpunkt

#### `GET /`

Root-Pfad, zeigt Service-Informationen an.

**Für detaillierte API-Dokumentation siehe: [docs/deDE/API.md](docs/deDE/API.md)**

## 🐳 Bereitstellungsanleitung

### Docker-Bereitstellung

#### Image erstellen

```bash
cd codes
docker build -t stargate:latest .
```

#### Container ausführen

```bash
docker run -d \
  --name stargate \
  -p 80:80 \
  -e AUTH_HOST=auth.example.com \
  -e PASSWORDS=plaintext:yourpassword \
  stargate:latest
```

### Docker Compose-Bereitstellung

Das Projekt bietet eine `docker-compose.yml`-Beispielkonfiguration, einschließlich Stargate-Service und Beispiel-whoami-Service:

```bash
cd codes
docker-compose up -d
```

### Traefik-Integration

Traefik-Labels in `docker-compose.yml` konfigurieren:

```yaml
services:
  stargate:
    image: stargate:latest
    environment:
      - AUTH_HOST=auth.example.com
      - PASSWORDS=plaintext:yourpassword
    networks:
      - traefik
    labels:
      - "traefik.enable=true"
      - "traefik.docker.network=traefik"
      - "traefik.http.routers.auth.entrypoints=http"
      - "traefik.http.routers.auth.rule=Host(`auth.example.com`) || Path(`/_session_exchange`)"
      - "traefik.http.middlewares.stargate.forwardauth.address=http://stargate/_auth"

  your-service:
    image: your-service:latest
    networks:
      - traefik
    labels:
      - "traefik.enable=true"
      - "traefik.docker.network=traefik"
      - "traefik.http.routers.your-service.entrypoints=http"
      - "traefik.http.routers.your-service.rule=Host(`your-service.example.com`)"
      - "traefik.http.routers.your-service.middlewares=stargate"  # Stargate-Middleware verwenden

networks:
  traefik:
    external: true
```

### Produktionsempfehlungen

1. **HTTPS verwenden**: In der Produktion sicherstellen, dass HTTPS über Traefik konfiguriert ist
2. **Starke Passwort-Algorithmen verwenden**: `plaintext` vermeiden, `bcrypt` oder `sha512` empfehlen
3. **Cookie-Domain setzen**: Wenn Sie Sitzungen über mehrere Subdomains teilen müssen, `COOKIE_DOMAIN` setzen
4. **Log-Verwaltung**: Angemessene Log-Rotation und Überwachung konfigurieren
5. **Ressourcenlimits**: Angemessene CPU- und Speicherlimits für Container setzen

**Für detaillierte Bereitstellungsanleitung siehe: [docs/deDE/DEPLOYMENT.md](docs/deDE/DEPLOYMENT.md)**

## 💻 Entwicklungsleitfaden

### Projektstruktur

```
codes/
├── src/
│   ├── cmd/
│   │   └── stargate/          # Hauptprogramm-Einstiegspunkt
│   │       ├── main.go        # Programmeinstieg
│   │       ├── server.go      # Serverkonfiguration
│   │       └── constants.go  # Konstantendefinitionen
│   ├── internal/
│   │   ├── auth/              # Authentifizierungslogik
│   │   ├── config/            # Konfigurationsverwaltung
│   │   ├── handlers/          # HTTP-Handler
│   │   ├── i18n/              # Internationalisierung
│   │   ├── middleware/        # Middleware
│   │   ├── secure/            # Passwort-Verschlüsselungsalgorithmen
│   │   └── web/               # Web-Vorlagen und statische Ressourcen
│   ├── go.mod
│   └── go.sum
├── Dockerfile
├── docker-compose.yml
└── start-local.sh
```

### Lokale Entwicklung

1. Abhängigkeiten installieren:
```bash
cd codes
go mod download
```

2. Tests ausführen:
```bash
go test ./...
```

3. Entwicklungsserver starten:
```bash
./start-local.sh
```

### Neue Passwort-Algorithmen hinzufügen

1. Neue Algorithmus-Implementierung im Verzeichnis `src/internal/secure/` erstellen:
```go
package secure

type NewAlgorithmResolver struct{}

func (r *NewAlgorithmResolver) Check(h string, password string) bool {
    // Passwort-Verifizierungslogik implementieren
    return false
}
```

2. Algorithmus in `src/internal/config/validation.go` registrieren:
```go
SupportedAlgorithms = map[string]secure.HashResolver{
    // ...
    "newalgorithm": &secure.NewAlgorithmResolver{},
}
```

### Neue Sprachunterstützung hinzufügen

1. Sprachkonstante in `src/internal/i18n/i18n.go` hinzufügen:
```go
const (
    LangEN Language = "en"
    LangZH Language = "zh"
    LangDE Language = "de"  // Neu
)
```

2. Übersetzungszuordnung hinzufügen:
```go
var translations = map[Language]map[string]string{
    // ...
    LangDE: {
        "error.auth_required": "Authentifizierung erforderlich",
        // ...
    },
}
```

3. Sprachoption in `src/internal/config/config.go` hinzufügen:
```go
Language = EnvVariable{
    PossibleValues: []string{"en", "zh", "de"},  // Neue Sprache hinzufügen
}
```

## 📝 Lizenz

Dieses Projekt ist unter der Apache License 2.0 lizenziert. Siehe die [LICENSE](codes/LICENSE)-Datei für Details.

## 🤝 Beitragen

Wir begrüßen Beiträge! Ob es sich handelt um:
- 🐛 Fehlerberichte
- 💡 Funktionsvorschläge
- 📝 Dokumentationsverbesserungen
- 🔧 Code-Beiträge

Bitte zögern Sie nicht, ein Issue zu öffnen oder einen Pull Request einzureichen. Jeder Beitrag macht Stargate besser!

---

## ⚠️ Produktions-Checkliste

Vor dem Bereitstellen in der Produktion stellen Sie sicher, dass Sie diese Sicherheits-Best-Practices abgeschlossen haben:

- ✅ **Starke Passwörter verwenden**: `plaintext` vermeiden, `bcrypt` oder `sha512` für Passwort-Hashing verwenden
- ✅ **HTTPS aktivieren**: HTTPS über Traefik oder Ihren Reverse-Proxy konfigurieren
- ✅ **Cookie-Domain setzen**: `COOKIE_DOMAIN` für ordnungsgemäße Sitzungsverwaltung über Subdomains konfigurieren
- ✅ **Überwachen & Protokollieren**: Angemessene Protokollierung und Überwachung für Ihre Bereitstellung einrichten
- ✅ **Regelmäßige Updates**: Stargate auf die neueste Version aktualisieren, um Sicherheitspatches zu erhalten
