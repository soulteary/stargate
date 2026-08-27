# Migrazione da v0.12.0 a v1.0.0

Stargate v1.0.0 stabilisce i contratti di deployment e HTTP. Verificare prima l'aggiornamento in staging.

## Prima dell'aggiornamento

1. Registrare immagine, variabili d'ambiente, route proxy, host di callback e probe.
2. Conservare l'immagine `v0.12.0` e il container precedente per il rollback.
3. Individuare tutti i client di `/_logout`, `/totp/enroll` e `/_session_exchange`.
4. Preparare lo storage Redis delle sessioni per deployment con più repliche.

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
