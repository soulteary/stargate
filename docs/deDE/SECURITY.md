# Sicherheitsdokumentation

> 🌐 **Language / 语言**: [English](../enUS/SECURITY.md) | [中文](../zhCN/SECURITY.md) | [Français](../frFR/SECURITY.md) | [Italiano](../itIT/SECURITY.md) | [日本語](../jaJP/SECURITY.md) | [Deutsch](SECURITY.md) | [한국어](../koKR/SECURITY.md)

Dieses Dokument erläutert die Sicherheitsfunktionen von Stargate, die Sicherheitskonfiguration und bewährte Praktiken.

> ⚠️ **Hinweis**: Diese Dokumentation wird derzeit übersetzt. Für die vollständige Version konsultieren Sie die [englische Version](../enUS/SECURITY.md).

## Implementierte Sicherheitsfunktionen

1. **Forward Auth-Schutz**: Zentralisierte Authentifizierungsschicht zum Schutz von Backend-Diensten
2. **Mehrere Passwort-Algorithmen**: Unterstützung für bcrypt, SHA512, MD5 und Plaintext (nur Entwicklung)
3. **Sichere Sitzungsverwaltung**: Cookie-basierte Sitzungen mit konfigurierbarer Domain und Ablaufzeit
4. **Service-Integrationssicherheit**: Sichere Kommunikation mit Warden- und Herald-Diensten unter Verwendung von mTLS oder HMAC
5. **Sitzungsfreigabe-Sicherheit**: Sicherer Mechanismus zum Austausch von Sitzungen zwischen Domains
6. **Eingabevalidierung**: Strenge Validierung aller Eingabeparameter
7. **Fehlerbehandlung**: Produktionsmodus verbirgt detaillierte Fehlerinformationen
8. **Sicherheitsantwort-Header**: Fügt automatisch sicherheitsbezogene HTTP-Antwort-Header hinzu
9. **HTTPS-Erzwingung**: Produktionsumgebungen müssen HTTPS verwenden
10. **OTP-Integration**: Sichere Integration mit Herald für OTP/Verifizierungscode-Authentifizierung

Weitere Details finden Sie in der [englischen Version](../enUS/SECURITY.md).

## Meldung von Sicherheitslücken

Wenn Sie eine Sicherheitslücke entdecken, melden Sie diese bitte über:

1. **GitHub Security Advisory** (Bevorzugt)
   - Gehen Sie zur Registerkarte [Security](https://github.com/soulteary/stargate/security) im Repository
   - Klicken Sie auf "Report a vulnerability"
   - Füllen Sie das Sicherheitsberatungsformular aus

2. **E-Mail** (Wenn GitHub Security Advisory nicht verfügbar ist)
   - Senden Sie eine E-Mail an die Projektbetreuer
   - Fügen Sie eine detaillierte Beschreibung der Sicherheitslücke bei

**Bitte melden Sie Sicherheitslücken nicht über öffentliche GitHub Issues.**
