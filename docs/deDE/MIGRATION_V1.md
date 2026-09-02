# Migration von v0.12.0 auf v1.0.0

Stargate v1.0.0 definiert stabile Bereitstellungs- und HTTP-Verträge. Die Umstellung zuerst in einer Staging-Umgebung prüfen.

## Vor dem Upgrade

1. Image, Umgebungsvariablen, Proxy-Routen, Callback-Hosts und Probes dokumentieren.
2. Das Image `v0.12.0` und den unveränderten Container für einen Rollback behalten.
3. Alle Aufrufer von `/_logout`, `/totp/enroll` und `/_session_exchange` erfassen.
4. Redis vorbereiten, wenn mehrere Prozesse Traffic bedienen, alte und neue Prozesse während des Rollouts überlappen oder Sitzungen und verbrauchte Tickets einen Neustart überdauern müssen. Ein einzelner laufender Prozess darf auch bei Cross-Domain-Callbacks In-Memory-Speicher verwenden; der Zustand ist jedoch prozesslokal und geht beim Neustart verloren.

## Rollback-Konfiguration erhalten und v1-Umgebung erstellen

v1 nicht mit der alten Umgebungsdatei starten. `stargate.env` unverändert lassen und den v0.12.0-Container behalten. Diese Befehle erstellen eine geschützte Rollback-Kopie und eine getrennte v1-Datei, überschreiben keine vorhandene Ausgabe und entfernen die veralteten Einstellungen nur aus der v1-Datei:

```bash
set -eu
old_env=./stargate.env
rollback_env=./stargate-v0.12.0.env
v1_env=./stargate-v1.env
old_container=${STARGATE_OLD_CONTAINER:-stargate}

test ! -e "$rollback_env"
test ! -e "$v1_env"
umask 077

# Falls die Datei fehlt, die Umgebung des vorhandenen Containers geschützt exportieren.
if [ ! -e "$old_env" ]; then
  export_tmp=$(mktemp "${old_env}.tmp.XXXXXX")
  trap 'rm -f "$export_tmp"' 0 1 2 15
  docker inspect --format '{{range .Config.Env}}{{println .}}{{end}}' "$old_container" > "$export_tmp"
  ln "$export_tmp" "$old_env"
  rm -f "$export_tmp"
  trap - 0 1 2 15
fi

test -f "$old_env"
chmod 600 "$old_env"
(set -C; cat "$old_env" > "$rollback_env")
(set -C; awk '
  /^[[:space:]]*WARDEN_OTP_(ENABLED|SECRET_KEY)[[:space:]]*(=|$)/ { next }
  /^[[:space:]]*PORT[[:space:]]*(=|$)/ { print "PORT=8080"; next }
  { print }
' "$old_env" > "$v1_env")
```

Herald-basiertes TOTP erfordert, dass Stargate authentifizierte Benutzer über Warden auflöst. Daher auch `WARDEN_ENABLED=true` und `WARDEN_URL` setzen.

v1 lehnt `WARDEN_OTP_ENABLED` und `WARDEN_OTP_SECRET_KEY` ab, sobald eine Variable vorhanden ist, auch leer oder `false`. Nicht zu `stargate-v1.env` hinzufügen. Für TOTP in Stargate `HERALD_ENABLED=true`, `HERALD_TOTP_ENABLED=true`, `HERALD_URL` und eine unterstützte Herald-Serviceauthentifizierung konfigurieren. Herald muss TOTP an herald-totp weiterleiten; herald-totp benötigt einen eigenen `HERALD_TOTP_ENCRYPTION_KEY`. Das alte Warden-Secret nicht automatisch wiederverwenden. Siehe [Konfiguration](CONFIG.md).

v1 mit `--env-file ./stargate-v1.env` prüfen und starten. `stargate-v0.12.0.env`, die unveränderte Originaldatei und den alten Container bis zum Ende des Rollback-Fensters behalten.

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
