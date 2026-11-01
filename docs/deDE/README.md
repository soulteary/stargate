# Dokumentationsindex

Willkommen zur Dokumentation des Stargate Forward Auth Service.

## 🌐 Mehrsprachige Dokumentation

- [English](../enUS/README.md) | [中文](../zhCN/README.md) | [Français](../frFR/README.md) | [Italiano](../itIT/README.md) | [日本語](../jaJP/README.md) | [Deutsch](README.md) | [한국어](../koKR/README.md)

## 📚 Dokumentenliste

### Kerndokumente

- **[README.md](../../README.deDE.md)** - Projektübersicht und Schnellstartanleitung
- **[ARCHITECTURE.md](ARCHITECTURE.md)** - Technische Architektur und Designentscheidungen

### Detaillierte Dokumente

- **[API.md](API.md)** - Vollständige API-Endpunkt-Dokumentation
  - Authentifizierungsprüfungs-Endpunkte
  - Login- und Logout-Endpunkte
  - Sitzungsaustausch-Endpunkte
  - Gesundheitsprüfungs-Endpunkte
  - Fehlerantwortformate
  - Authentifizierungsflussbeispiele

- **[CONFIG.md](CONFIG.md)** - Konfigurationsreferenz
  - Konfigurationsmethoden
  - Erforderliche Konfigurationselemente
  - Optionale Konfigurationselemente
  - Passwort-Konfigurationsdetails
  - Konfigurationsbeispiele
  - Best Practices für die Konfiguration

- **[DEPLOYMENT.md](DEPLOYMENT.md)** - Bereitstellungsanleitung
  - Docker-Bereitstellung
  - Docker Compose-Bereitstellung
  - Traefik-Integration
  - Produktionsbereitstellung
  - Überwachung und Wartung
  - Fehlerbehebung

## 🚀 Schnellnavigation

### Erste Schritte

1. Lesen Sie [README.deDE.md](../../README.deDE.md), um das Projekt zu verstehen
2. Überprüfen Sie den Abschnitt [Schnellstart](../../README.deDE.md#-schnellstart)
3. Beziehen Sie sich auf [Konfiguration](../../README.deDE.md#-grundkonfiguration), um den Dienst zu konfigurieren

### Entwickler

1. Lesen Sie [ARCHITECTURE.md](ARCHITECTURE.md), um die Architektur zu verstehen
2. Überprüfen Sie [API.md](API.md), um die API-Schnittstellen zu verstehen
3. Beziehen Sie sich auf den [Entwicklungsleitfaden](CONTRIBUTING.md) für die Entwicklung

### Betrieb

1. Lesen Sie [DEPLOYMENT.md](DEPLOYMENT.md), um die Bereitstellungsmethoden zu verstehen
2. Überprüfen Sie [CONFIG.md](CONFIG.md), um die Konfigurationsoptionen zu verstehen
3. Beziehen Sie sich auf [Fehlerbehebung](DEPLOYMENT.md#fehlerbehebung), um Probleme zu lösen

## 📖 Dokumentenstruktur

```
stargate/
├── README.md              # Hauptprojektdokument (Englisch)
├── README.zhCN.md         # Hauptprojektdokument (Chinesisch)
├── README.frFR.md         # Hauptprojektdokument (Französisch)
├── README.itIT.md         # Hauptprojektdokument (Italienisch)
├── README.jaJP.md         # Hauptprojektdokument (Japanisch)
├── README.deDE.md         # Hauptprojektdokument (Deutsch)
├── README.koKR.md         # Hauptprojektdokument (Koreanisch)
├── docs/
│   ├── enUS/
│   │   ├── README.md       # Dokumentationsindex (Englisch)
│   │   ├── ARCHITECTURE.md # Architekturdokument (Englisch)
│   │   ├── API.md          # API-Dokument (Englisch)
│   │   ├── CONFIG.md       # Konfigurationsreferenz (Englisch)
│   │   └── DEPLOYMENT.md   # Bereitstellungsanleitung (Englisch)
│   ├── zhCN/
│   │   ├── README.md       # Dokumentationsindex (Chinesisch)
│   │   ├── ARCHITECTURE.md # Architekturdokument (Chinesisch)
│   │   ├── API.md          # API-Dokument (Chinesisch)
│   │   ├── CONFIG.md       # Konfigurationsreferenz (Chinesisch)
│   │   └── DEPLOYMENT.md   # Bereitstellungsanleitung (Chinesisch)
│   └── deDE/
│       ├── README.md       # Dokumentationsindex (Deutsch, diese Datei)
│       ├── ARCHITECTURE.md # Architekturdokument (Deutsch)
│       ├── API.md          # API-Dokument (Deutsch)
│       ├── CONFIG.md       # Konfigurationsreferenz (Deutsch)
│       └── DEPLOYMENT.md   # Bereitstellungsanleitung (Deutsch)
└── ...
```

## 🔍 Nach Thema Suchen

### Konfiguration

- Umgebungsvariablen-Konfiguration: [CONFIG.md](CONFIG.md)
- Passwort-Konfiguration: [CONFIG.md#passwort-konfiguration](CONFIG.md#passwort-konfiguration)
- Konfigurationsbeispiele: [CONFIG.md#konfigurationsbeispiele](CONFIG.md#konfigurationsbeispiele)

### API

- API-Endpunktliste: [API.md](API.md)
- Authentifizierungsfluss: [API.md#authentifizierungsfluss-beispiele](API.md#authentifizierungsfluss-beispiele)
- Fehlerbehandlung: [API.md#fehlerantwort-format](API.md#fehlerantwort-format)

### Bereitstellung

- Docker-Bereitstellung: [DEPLOYMENT.md#docker-bereitstellung](DEPLOYMENT.md#docker-bereitstellung)
- Traefik-Integration: [DEPLOYMENT.md#traefik-integration](DEPLOYMENT.md#traefik-integration)
- Produktionsumgebung: [DEPLOYMENT.md#produktionsbereitstellung](DEPLOYMENT.md#produktionsbereitstellung)

### Architektur

- Technologie-Stack: [ARCHITECTURE.md#technologie-stack](ARCHITECTURE.md#technologie-stack)
- Projektstruktur: [ARCHITECTURE.md#projektstruktur](ARCHITECTURE.md#projektstruktur)
- Kernkomponenten: [ARCHITECTURE.md#hauptkomponenten](ARCHITECTURE.md#hauptkomponenten)

## 💡 Verwendungsempfehlungen

1. **Erstmalige Benutzer**: Beginnen Sie mit [README.deDE.md](../../README.deDE.md) und folgen Sie der Schnellstartanleitung
2. **Dienst konfigurieren**: Beziehen Sie sich auf [CONFIG.md](CONFIG.md), um alle Konfigurationsoptionen zu verstehen
3. **Traefik integrieren**: Überprüfen Sie den Traefik-Integrationsabschnitt in [DEPLOYMENT.md](DEPLOYMENT.md)
4. **Erweiterungen entwickeln**: Lesen Sie [ARCHITECTURE.md](ARCHITECTURE.md), um das Architekturdesign zu verstehen
5. **Fehlerbehebung**: Überprüfen Sie [DEPLOYMENT.md#fehlerbehebung](DEPLOYMENT.md#fehlerbehebung)

## 📝 Dokumentationsaktualisierungen

Die Dokumentation wird kontinuierlich aktualisiert, während sich das Projekt entwickelt. Wenn Sie Fehler finden oder Ergänzungen benötigen, senden Sie bitte ein Issue oder einen Pull Request.

## 🤝 Beitragen

Verbesserungen der Dokumentation sind willkommen:

1. Fehler oder Bereiche finden, die verbessert werden müssen
2. Ein Issue einreichen, das das Problem beschreibt
3. Oder direkt einen Pull Request einreichen
