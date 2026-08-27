# Migration von v0.12.0 auf v1.0.0

Stargate v1.0.0 definiert stabile Bereitstellungs- und HTTP-Verträge. Die Umstellung zuerst in einer Staging-Umgebung prüfen.

## Vor dem Upgrade

1. Image, Umgebungsvariablen, Proxy-Routen, Callback-Hosts und Probes dokumentieren.
2. Das Image `v0.12.0` und den unveränderten Container für einen Rollback behalten.
3. Alle Aufrufer von `/_logout`, `/totp/enroll` und `/_session_exchange` erfassen.
4. Bei mehreren Replikaten Redis-Sitzungsspeicher vorbereiten.

## Erforderliche Änderungen

| Vertrag | v0.12.0 | v1.0.0 |
| --- | --- | --- |
| Container-Port | `80` | `8080` |
| Laufzeitbenutzer | root | UID/GID `10001` |
| Liveness | `/health` | `/healthz` |
| Readiness | `/health` | `/readyz` |

Traefik ForwardAuth muss `http://stargate:8080/_auth` verwenden. Für mehrere Replikate `SESSION_STORAGE_ENABLED=true` und dieselben Redis- und Ticket-Einstellungen verwenden.

## Sicherheits- und HTTP-Verträge

- Cross-Domain-Austausch verwendet `?ticket=` statt der rohen Sitzungs-ID `?id=`. `CALLBACK_ALLOWED_HOSTS` und ein mindestens 32 Zeichen langes `SESSION_EXCHANGE_SECRET` konfigurieren.
- Weiterleitungsheader werden nur von Quellen in `TRUSTED_PROXIES` akzeptiert.
- `Stargate-Password` ist standardmäßig deaktiviert; für vertrauenswürdige Proxy-Clients `PASSWORD_HEADER_AUTH_ENABLED=true` setzen.
- Vertrauenswürdige Identitätsheader benötigen Warden, `HEADER_AUTH_ENABLED=true` und ein starkes `HEADER_AUTH_SHARED_SECRET`.
- `GET /_logout` wird durch `POST /_logout` ersetzt.
- `GET /totp/enroll` zeigt nur eine Bestätigungsseite; `POST /totp/enroll` startet die Registrierung.
- `GET /metrics` bleibt unverändert verfügbar.

## Rollout und Prüfung

```bash
curl --fail http://127.0.0.1:8080/healthz
curl --fail http://127.0.0.1:8080/readyz
curl --fail http://127.0.0.1:8080/metrics
```

Login, Logout, TOTP, erlaubte und abgelehnte Callback-Hosts sowie abgelaufene oder erneut verwendete Tickets prüfen. v0.12.0 und v1.0.0 nicht im selben Instanzpool mischen.

## Rollback

Altes Image, Port und Probes gemeinsam wiederherstellen. Offene v1-Tickets verwerfen und Benutzer neu anmelden lassen; aktive Sitzungen sind zwischen gemischten Versionen nicht portabel.
