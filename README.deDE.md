# Stargate - Forward Auth Service

[![License](https://img.shields.io/badge/license-Apache%202.0-blue.svg)](LICENSE)
[![Go Version](https://img.shields.io/badge/go-1.26+-blue.svg)](https://golang.org)
[![codecov](https://codecov.io/gh/soulteary/stargate/branch/main/graph/badge.svg)](https://codecov.io/gh/soulteary/stargate)
[![Go Report Card](https://goreportcard.com/badge/github.com/soulteary/stargate)](https://goreportcard.com/report/github.com/soulteary/stargate)

> **🚀 Ihr Gateway zu sicheren Microservices**

![Stargate](.github/assets/banner.jpg)

Stargate ist ein produktionsreifer, leichtgewichtiger Forward Auth Service, der als **einzelner Authentifizierungspunkt** für Ihre gesamte Infrastruktur konzipiert wurde. Mit Go entwickelt und für Leistung optimiert, integriert sich Stargate nahtlos mit Traefik und anderen Reverse-Proxies, um Ihre Backend-Services zu schützen—**ohne eine einzige Zeile Authentifizierungscode in Ihren Anwendungen zu schreiben**.

## 🌐 Mehrsprachige Dokumentation

- [English](README.md) | [中文](README.zhCN.md) | [Français](README.frFR.md) | [Italiano](README.itIT.md) | [日本語](README.jaJP.md) | [Deutsch](README.deDE.md) | [한국어](README.koKR.md)

![Vorschau](.github/assets/preview.png)

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
- **Unternehmensauthentifizierung**: Integration mit Warden (Benutzer-Whitelist) und Herald (OTP/Verifizierungscodes) für produktionsreife Authentifizierung

## ✨ Funktionen

### 🔐 Unternehmensgrade Sicherheit

- **Mehrere Passwort-Verschlüsselungsalgorithmen**: Wählen Sie aus Plaintext (Test), bcrypt, MD5, SHA512 und mehr
- **Sichere Sitzungsverwaltung**: Cookie-basierte Sitzungen mit anpassbarer Domain und Ablaufzeit
- **Flexible Authentifizierung**: Unterstützung für passwortbasierte und sitzungsbasierte Authentifizierung
- **OTP/Verifizierungscode-Unterstützung**: Integration mit Herald-Service für SMS/Email-Verifizierungscodes
- **Benutzer-Whitelist-Verwaltung**: Integration mit Warden-Service für Benutzerzugriffskontrolle

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

## 📋 Inhaltsverzeichnis

- [Schnellstart](#-schnellstart)
- [Dokumentation](#-dokumentation)
- [Grundkonfiguration](#-grundkonfiguration)
- [Optionale Service-Integration](#-optionale-service-integration)
- [Produktions-Checkliste](#-produktions-checkliste)
- [Lizenz](#-lizenz)

## 🚀 Schnellstart

Stargate in **weniger als 2 Minuten** zum Laufen bringen!

### Verwendung von Docker Compose (Empfohlen)

**Schritt 1:** Repository klonen
```bash
git clone <repository-url>
cd stargate
```

**Schritt 2:** Authentifizierung konfigurieren (`docker-compose.yml` bearbeiten)

**Option A: Passwort-Authentifizierung (Einfach)**
```yaml
services:
  stargate:
    environment:
      - AUTH_HOST=auth.example.com
      - PASSWORDS=plaintext:yourpassword1|yourpassword2
```

**Option B: Warden + Herald OTP-Authentifizierung (Produktion)**
```yaml
services:
  stargate:
    environment:
      - AUTH_HOST=auth.example.com
      - WARDEN_ENABLED=true
      - WARDEN_URL=http://warden:8080
      - WARDEN_API_KEY=your-warden-api-key
      - HERALD_ENABLED=true
      - HERALD_URL=http://herald:8080
      - HERALD_HMAC_SECRET=your-herald-hmac-secret
```

**Schritt 3:** Service starten
```bash
docker-compose up -d
```

**Das war's!** Ihr Authentifizierungsservice läuft jetzt. 🎉

### Lokale Entwicklung

Für die lokale Entwicklung stellen Sie sicher, dass Go 1.26+ installiert ist, dann:

```bash
chmod +x start-local.sh
./start-local.sh
```

Zugriff auf die Login-Seite unter `http://localhost:8080/_login?callback=localhost`

## 📚 Dokumentation

Umfassende Dokumentation ist verfügbar, um Ihnen zu helfen, das Beste aus Stargate herauszuholen:

### Kern-Dokumente

- 📐 **[Architekturdokument](docs/deDE/ARCHITECTURE.md)** - Tiefere Einblicke in technische Architektur und Designentscheidungen
- 🔌 **[API-Dokument](docs/deDE/API.md)** - Vollständige API-Endpunkt-Referenz mit Beispielen
- ⚙️ **[Konfigurationsreferenz](docs/deDE/CONFIG.md)** - Detaillierte Konfigurationsoptionen und Best Practices
- 🚀 **[Bereitstellungsanleitung](docs/deDE/DEPLOYMENT.md)** - Produktionsbereitstellungsstrategien und Empfehlungen

### Schnellreferenz

- **API-Endpunkte**: `GET /_auth` (Auth-Prüfung), `GET /_login` (Login-Seite), `POST /_login` (Login), `POST /_send_verify_code` (OTP senden), `GET /_logout` (Logout), `GET /_session_exchange` (Cross-Domain), `GET /totp/enroll`, `POST /totp/enroll/confirm`, `GET /totp/revoke`, `POST /totp/revoke` (TOTP bei Herald), `GET /health` (Gesundheitsprüfung), `GET /metrics` (Prometheus)
- **Bereitstellung**: Docker Compose wird für den Schnellstart empfohlen. Siehe [DEPLOYMENT.md](docs/deDE/DEPLOYMENT.md) für die Produktionsbereitstellung.
- **Entwicklung**: Für entwicklungsbezogene Dokumentation siehe [ARCHITECTURE.md](docs/deDE/ARCHITECTURE.md)

## ⚙️ Grundkonfiguration

Stargate verwendet Umgebungsvariablen für die Konfiguration. Hier sind die häufigsten Einstellungen:

### Erforderliche Konfiguration

- **`AUTH_HOST`**: Hostname des Authentifizierungsservices (z.B. `auth.example.com`)
- **`PASSWORDS`**: Passwort-Konfiguration, Format: `algorithm:password1|password2|password3`

### Häufige Konfigurationsbeispiele

```bash
# Einfache Passwort-Authentifizierung
AUTH_HOST=auth.example.com
PASSWORDS=plaintext:test123|admin456

# BCrypt-Hash verwenden
PASSWORDS=bcrypt:$2a$10$N9qo8uLOickgx2ZMRZoMyeIjZAgcfl7p92ldGxad68LJZdL17lhWy

# Cross-Domain-Sitzungsteilung
COOKIE_DOMAIN=.example.com

# Login-Seite anpassen
LOGIN_PAGE_TITLE=Mein Auth-Service
LANGUAGE=de  # oder 'en'
```

**Unterstützte Passwort-Algorithmen:** `plaintext` (nur Test), `bcrypt`, `md5`, `sha512`

**Für vollständige Konfigurationsreferenz siehe: [docs/deDE/CONFIG.md](docs/deDE/CONFIG.md)**

## 🔗 Optionale Service-Integration

Stargate kann vollständig unabhängig verwendet werden oder optional mit den folgenden Services integriert werden:

### Warden-Integration (Benutzer-Whitelist)

Bietet Benutzer-Whitelist-Verwaltung und Benutzerinformationen. Wenn aktiviert, fragt Stargate Warden ab, um zu überprüfen, ob ein Benutzer in der erlaubten Liste ist.

```bash
WARDEN_ENABLED=true
WARDEN_URL=http://warden:8080
WARDEN_API_KEY=your-api-key
```

### Herald-Integration (OTP/Verifizierungscodes)

Bietet OTP/Verifizierungscode-Services. Wenn aktiviert, ruft Stargate Herald auf, um Verifizierungscodes (SMS/Email) zu erstellen, zu senden und zu überprüfen.

```bash
HERALD_ENABLED=true
HERALD_URL=http://herald:8080
HERALD_HMAC_SECRET=your-hmac-secret  # Produktion
# oder
HERALD_API_KEY=your-api-key  # Entwicklung
```

**Hinweis:** Beide Integrationen sind optional. Stargate kann unabhängig mit Passwort-Authentifizierung verwendet werden.

**Vollständige Integrationsanleitung siehe: [docs/deDE/ARCHITECTURE.md](docs/deDE/ARCHITECTURE.md)**

## ⚠️ Produktions-Checkliste

Vor dem Bereitstellen in der Produktion:

- ✅ Verwenden Sie starke Passwort-Algorithmen (`bcrypt` oder `sha512`, vermeiden Sie `plaintext`)
- ✅ Aktivieren Sie HTTPS über Traefik oder Ihren Reverse-Proxy
- ✅ Setzen Sie `COOKIE_DOMAIN` für ordnungsgemäße Sitzungsverwaltung über Subdomains
- ✅ Für erweiterte Funktionen optional Warden + Herald für OTP-Authentifizierung integrieren
- ✅ Verwenden Sie HMAC-Signaturen oder mTLS für Stargate ↔ Herald/Warden-Kommunikation
- ✅ Richten Sie angemessene Protokollierung und Überwachung ein
- ✅ Halten Sie Stargate auf dem neuesten Stand

## 🎯 Designprinzipien

Stargate ist für die unabhängige Verwendung konzipiert:

- **Eigenständige Nutzung**: Stargate kann unabhängig mit Passwort-Authentifizierungsmodus ausgeführt werden, ohne externe Abhängigkeiten
- **Optionale Integration**: Kann optional mit Warden (Benutzer-Whitelist) und Herald (OTP/Verifizierungscodes) integriert werden
- **Hohe Leistung**: Der forwardAuth-Hauptpfad überprüft nur die Sitzung und gewährleistet schnelle Antwortzeiten
- **Flexibilität**: Unterstützt mehrere Authentifizierungsmodi, wählen Sie je nach Bedarf

## 📝 Lizenz

Dieses Projekt ist unter der Apache License 2.0 lizenziert. Siehe die [LICENSE](LICENSE)-Datei für Details.

## 🤝 Beitragen

Wir begrüßen Beiträge! Ob es sich handelt um:
- 🐛 Fehlerberichte
- 💡 Funktionsvorschläge
- 📝 Dokumentationsverbesserungen
- 🔧 Code-Beiträge

Bitte zögern Sie nicht, ein Issue zu öffnen oder einen Pull Request einzureichen.
