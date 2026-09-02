# Migrazione da v0.12.0 a v1.0.0

Stargate v1.0.0 stabilisce i contratti di deployment e HTTP. Verificare prima l'aggiornamento in staging.

## Prima dell'aggiornamento

1. Registrare immagine, variabili d'ambiente, route proxy, host di callback e probe.
2. Conservare l'immagine `v0.12.0` e il container precedente per il rollback.
3. Individuare tutti i client di `/_logout`, `/totp/enroll` e `/_session_exchange`.
4. Preparare Redis se più processi servono traffico, se processi vecchi e nuovi si sovrappongono durante il rollout o se sessioni e ticket consumati devono sopravvivere a un riavvio. Un singolo processo attivo può usare lo storage in memoria anche con callback cross-domain, ma tutto lo stato è locale al processo e si perde al riavvio.

## Conservare la configurazione di rollback e creare l'ambiente v1

Non avviare v1 con il vecchio file di ambiente. Lasciare invariato `stargate.env` e conservare il container v0.12.0. I comandi seguenti creano una copia protetta per il rollback e un file v1 separato, rifiutano di sovrascrivere output esistenti e rimuovono le impostazioni ritirate solo dal file v1:

```bash
set -eu
old_env=./stargate.env
rollback_env=./stargate-v0.12.0.env
v1_env=./stargate-v1.env

test -f "$old_env"
test ! -e "$rollback_env"
test ! -e "$v1_env"
chmod 600 "$old_env"
umask 077
(set -C; cat "$old_env" > "$rollback_env")
(set -C; awk '!/^[[:space:]]*WARDEN_OTP_(ENABLED|SECRET_KEY)[[:space:]]*=/' "$old_env" > "$v1_env")
```

Il TOTP tramite Herald richiede che Stargate risolva gli utenti autenticati tramite Warden; impostare quindi anche `WARDEN_ENABLED=true` e `WARDEN_URL`.

v1 rifiuta `WARDEN_OTP_ENABLED` e `WARDEN_OTP_SECRET_KEY` quando una variabile è presente, anche vuota o `false`. Non aggiungerle a `stargate-v1.env`. Per TOTP configurare Stargate con `HERALD_ENABLED=true`, `HERALD_TOTP_ENABLED=true`, `HERALD_URL` e un'autenticazione del servizio Herald supportata. Configurare Herald per inoltrare TOTP a herald-totp e assegnare a herald-totp un proprio `HERALD_TOTP_ENCRYPTION_KEY`; non riutilizzare automaticamente il vecchio secret Warden. Vedere [Configurazione](CONFIG.md).

Verificare e distribuire v1 con `--env-file ./stargate-v1.env`. Conservare `stargate-v0.12.0.env`, il file originale invariato e il vecchio container fino alla chiusura della finestra di rollback.

## Modifiche obbligatorie

| Contratto | v0.12.0 | v1.0.0 |
| --- | --- | --- |
| Porta container | `80` | `8080` |
| Utente runtime | root | UID/GID `10001` |
| Liveness | `/health` | `/healthz` |
| Readiness | `/health` | `/readyz` |

Traefik ForwardAuth deve usare `http://stargate:8080/_auth`. Con più repliche abilitare `SESSION_STORAGE_ENABLED=true` e condividere Redis e i segreti dei ticket.

## Contratti di sicurezza e HTTP

- Lo scambio tra domini usa `?ticket=` invece dell'ID sessione grezzo `?id=`. Configurare `CALLBACK_ALLOWED_HOSTS` e un `SESSION_EXCHANGE_SECRET` di almeno 32 caratteri.
- Gli header inoltrati sono accettati solo dalle origini in `TRUSTED_PROXIES`.
- `Stargate-Password` è disabilitato per impostazione predefinita; usare `PASSWORD_HEADER_AUTH_ENABLED=true` solo con proxy fidati.
- Gli header di identità richiedono Warden, `HEADER_AUTH_ENABLED=true` e un segreto condiviso forte.
- `GET /_logout` diventa `POST /_logout`.
- `GET /totp/enroll` mostra una pagina di conferma; `POST /totp/enroll` avvia la registrazione.
- `GET /metrics` rimane invariato.

## Rollout e verifica

```bash
curl --fail http://127.0.0.1:8080/healthz
curl --fail http://127.0.0.1:8080/readyz
curl --fail http://127.0.0.1:8080/metrics
```

Verificare login, logout, TOTP, callback consentiti e rifiutati e ticket scaduti o riutilizzati. Non mescolare v0.12.0 e v1.0.0 nello stesso pool.

## Rollback

Ripristinare insieme immagine, porta e probe v0.12.0. Invalidare i ticket v1 in corso e richiedere un nuovo login.
