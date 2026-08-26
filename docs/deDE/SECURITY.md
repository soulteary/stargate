# Sicherheitsdokumentation

> 🌐 **Language / 语言**: [English](../enUS/SECURITY.md) | [中文](../zhCN/SECURITY.md) | [Français](../frFR/SECURITY.md) | [Italiano](../itIT/SECURITY.md) | [日本語](../jaJP/SECURITY.md) | [Deutsch](SECURITY.md) | [한국어](../koKR/SECURITY.md)

Dieses Dokument erläutert die Sicherheitsfunktionen von Stargate, die Sicherheitskonfiguration und bewährte Praktiken.

> ⚠️ **Hinweis**: Diese Dokumentation wird derzeit übersetzt. Für die vollständige Version konsultieren Sie die [englische Version](../enUS/SECURITY.md).

## Implementierte Sicherheitsfunktionen

1. **Forward Auth-Schutz**: Zentralisierte Authentifizierungsschicht zum Schutz von Backend-Diensten
2. **Mehrere Passwort-Algorithmen**: bcrypt sowie Plaintext nur für lokale Entwicklung; MD5 und ungesalzenes SHA-512 werden abgelehnt
3. **Sichere Sitzungsverwaltung**: Cookie-basierte Sitzungen mit konfigurierbarer Domain und Ablaufzeit
4. **Service-Integrationssicherheit**: Sichere Kommunikation mit Warden- und Herald-Diensten unter Verwendung von mTLS oder HMAC
5. **Sitzungsfreigabe-Sicherheit**: Sicherer Mechanismus zum Austausch von Sitzungen zwischen Domains
6. **Eingabevalidierung**: Strenge Validierung aller Eingabeparameter
7. **Fehlerbehandlung**: HTTP-Antworten verwenden begrenzte Fehlermeldungen; `DEBUG` steuert nur die Protokollierung
8. **Sicherheitsantwort-Header**: Fügt automatisch sicherheitsbezogene HTTP-Antwort-Header hinzu
9. **HTTPS-Erzwingung**: Produktionsumgebungen müssen HTTPS verwenden
10. **OTP-Integration**: Sichere Integration mit Herald für OTP/Verifizierungscode-Authentifizierung

Weitere Details finden Sie in der [englischen Version](../enUS/SECURITY.md).

## Sicherheitsrelevante Konfiguration

- Setzen Sie `COOKIE_SECURE=true` unter HTTPS und begrenzen Sie `TRUSTED_PROXIES` auf bekannte Proxy-Adressen.
- Konfigurieren Sie `CALLBACK_ALLOWED_HOSTS`; für domänenübergreifende Tickets ist ein `SESSION_EXCHANGE_SECRET` mit mindestens 32 Zeichen erforderlich.
- Bei Warden und Herald müssen HMAC-Schlüsselkennung und -Geheimnis sowie mTLS-Clientzertifikat und -schlüssel jeweils gemeinsam gesetzt werden.
- Die vollständige, synchronisierte Variablenliste finden Sie in [CONFIG.md](CONFIG.md#sicherheits--und-integrationskonfiguration).

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

## Vertrauensgrenze der Bereitstellung

Forward Auth schützt nur Anfragen, die tatsächlich über den konfigurierten Reverse Proxy laufen. Veröffentlichen Sie Backend-Ports nicht direkt. Erlauben Sie Zugriffe nur aus dem Proxy-Netz und entfernen Sie vom Client gesetzte Identitäts- oder Zugangsdaten-Header vor der Weiterleitung.
