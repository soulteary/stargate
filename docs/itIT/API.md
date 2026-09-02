# Documentazione API

Questo documento descrive in dettaglio tutti gli endpoint API del servizio Stargate Forward Auth.

Per aggiornare da v0.12.0, leggere prima la [guida alla migrazione v1.0.0](MIGRATION_V1.md).

## Indice

- [Endpoint Verifica Autenticazione](#endpoint-verifica-autenticazione)
- [Endpoint Login](#endpoint-login)
- [Endpoint Logout](#endpoint-logout)
- [Endpoint Scambio Sessione](#endpoint-scambio-sessione)
- [Endpoint Verifica Salute](#endpoint-verifica-salute)
- [Endpoint Root](#endpoint-root)

## Endpoint Verifica Autenticazione

### `GET /_auth`

L'endpoint principale di verifica autenticazione per Traefik Forward Auth. Questo endpoint è la funzionalità principale di Stargate, utilizzato per verificare se un utente è stato autenticato.

#### Metodi di Autenticazione

Stargate supporta due metodi di autenticazione, verificati nel seguente ordine di priorità:

1. **Autenticazione Header** (richieste API)
   - Header richiesta: `Stargate-Password: <password>`
   - Disabilitata per impostazione predefinita. Impostare `PASSWORD_HEADER_AUTH_ENABLED=true` solo per client legacy attendibili.
   - Adatto per richieste API, script di automazione, ecc.

2. **Autenticazione Cookie** (richieste Web)
   - Cookie: `stargate_session_id=<session_id>`
   - Adatto per applicazioni Web accessibili tramite browser

#### Header Richiesta

| Header | Tipo | Richiesto | Descrizione |
|--------|------|-----------|-------------|
| `Stargate-Password` | String | No | Autenticazione password per richieste API |
| `Cookie` | String | No | Cookie sessione contenente `stargate_session_id` |
| `Accept` | String | No | Utilizzato per determinare il tipo di richiesta (HTML/API) |

#### Risposta

**Risposta di Successo (200 OK)**

Quando l'autenticazione riesce, Stargate imposta l'header informazione utente e restituisce un codice di stato 200:

```
HTTP/1.1 200 OK
X-Forwarded-User: authenticated
```

Il nome dell'header utente può essere configurato tramite la variabile d'ambiente `USER_HEADER_NAME` (predefinito: `X-Forwarded-User`).

**Risposta di Fallimento**

| Codice di Stato | Descrizione | Corpo Risposta |
|----------------|-------------|----------------|
| `401 Unauthorized` | Autenticazione fallita | Messaggio di errore (formato JSON per richieste API) o reindirizzamento alla pagina di login (richieste HTML) |
| `500 Internal Server Error` | Errore server | Messaggio di errore |

#### Gestione Tipo Richiesta

- **Richieste HTML**: Reindirizzamento a `/_login?callback=<originalURL>` in caso di fallimento autenticazione
- **Richieste API** (JSON/XML): Restituzione di una risposta di errore 401 in caso di fallimento autenticazione

#### Esempi

**Utilizzo Autenticazione Header (Richiesta API)**

```bash
# Prerequisito del server: PASSWORD_HEADER_AUTH_ENABLED=true
curl -H "Stargate-Password: yourpassword" \
     http://auth.example.com/_auth
```

**Utilizzo Autenticazione Cookie (Richiesta Web)**

```bash
curl -H "Cookie: stargate_session_id=<session_id>" \
     http://auth.example.com/_auth
```

## Endpoint Login

### `GET /_login`

Visualizza la pagina di login.

#### Parametri Query

| Parametro | Tipo | Richiesto | Descrizione |
|-----------|------|-----------|-------------|
| `callback` | String | No | URL callback dopo login riuscito (generalmente il dominio della richiesta originale) |

#### Comportamento

- Se l'utente è già connesso, reindirizza automaticamente all'endpoint scambio sessione
- Se l'utente non è connesso, visualizza la pagina di login
- Se l'URL contiene un parametro `callback` e il dominio differisce, il callback è memorizzato nel cookie `stargate_callback` (scade in 10 minuti)

#### Priorità Recupero Callback

1. **Dai parametri query**: Il parametro `callback` nell'URL (priorità più alta)
2. **Dal cookie**: Se non presente nei parametri query, recuperare dal cookie `stargate_callback`

#### Risposta

**200 OK** - Restituisce HTML pagina di login

La pagina include:
- Form di login
- Titolo personalizzabile (`LOGIN_PAGE_TITLE`)
- Testo piè di pagina personalizzabile (`LOGIN_PAGE_FOOTER_TEXT`)

#### Esempio

```bash
# Accedere alla pagina di login
curl http://auth.example.com/_login?callback=app.example.com
```

### `POST /_login`

Elabora le richieste di login e supporta due modalità di autenticazione:

1. **Autenticazione con password**: verifica la password e crea una sessione
2. **Autenticazione Warden**: verifica un codice challenge Herald oppure un codice TOTP già configurato e crea una sessione

#### Corpo Richiesta

Dati form (`application/x-www-form-urlencoded`):

**Autenticazione con password:**

| Campo | Tipo | Richiesto | Descrizione |
|-------|------|-----------|-------------|
| `auth_method` | String | No | Metodo di autenticazione; `password` è il valore predefinito |
| `password` | String | Sì | Password utente |
| `callback` | String | No | URL callback dopo login riuscito |

**Autenticazione Warden:**

| Campo | Tipo | Richiesto | Descrizione |
|-------|------|-----------|-------------|
| `auth_method` | String | Sì | Metodo di autenticazione; valore `warden` |
| `phone` | String | No | Numero di telefono; è richiesto almeno uno tra `phone` e `mail` |
| `mail` | String | No | Indirizzo email; è richiesto almeno uno tra `mail` e `phone` |
| `challenge_id` | String | Condizionale | Richiesto con `verify_code`; facoltativo quando `use_otp=true` |
| `verify_code` | String | Condizionale | Richiesto per il challenge Herald quando `use_otp` è assente o `false` |
| `use_otp` | Boolean | No | Impostare a `true` per usare un autenticatore TOTP configurato invece di un challenge Herald |
| `otp_code` | String | Condizionale | Richiesto quando `use_otp=true` |
| `callback` | String | No | URL callback dopo login riuscito |

Il gestore accetta anche `phone` + `mail` nella stessa richiesta Warden.
La variante TOTP richiede `HERALD_TOTP_ENABLED=true` e un utente già registrato. In caso contrario, usare la variante `challenge_id` + `verify_code`.

#### Priorità Recupero Callback

L'elaborazione login recupera il callback nel seguente ordine di priorità:

1. **Dal cookie**: Se il dominio differiva durante l'accesso precedente alla pagina di login, il callback è memorizzato nel cookie `stargate_callback`
2. **Dai dati del form**: Il campo `callback` nei dati del form della richiesta POST
3. **Dai parametri query**: Il `callback` nei parametri query dell'URL
4. **Inferenza automatica**: Se nessuno dei precedenti esiste, e il dominio di origine (`X-Forwarded-Host`) differisce dal dominio del servizio di autenticazione, utilizzare il dominio di origine come callback

#### Risposta

**Risposta di Successo (200 OK)**

La risposta varia a seconda che ci sia un callback e il tipo di richiesta:

1. **Con callback**:
   - Reindirizza a `{callback}/_session_exchange?ticket={opaque_ticket}`
   - Codice di stato: `302 Found`

2. **Senza callback**:
   - **Richiesta HTML**: Restituisce una pagina HTML con meta refresh, reindirizzando automaticamente al dominio di origine
   - **Richiesta API**: Restituisce una risposta JSON
     ```json
     {
       "success": true,
       "message": "Login successful"
     }
     ```

**Risposta di Fallimento**

| Codice di Stato | Descrizione | Corpo Risposta |
|----------------|-------------|----------------|
| `400 Bad Request` | Identificatore Warden o campi del metodo scelto mancanti/non validi, oppure TOTP non registrato | Messaggio di errore in formato JSON/XML/testo secondo header Accept |
| `401 Unauthorized` | Password errata, utente Warden assente/inattivo o verifica challenge/TOTP fallita | Messaggio di errore in formato JSON/XML/testo secondo header Accept |
| `502 Bad Gateway` | Lettura dello stato TOTP da Herald non riuscita | Messaggio di errore |
| `503 Service Unavailable` | Servizio di verifica Herald non disponibile | Messaggio di errore |
| `500 Internal Server Error` | Errore server | Messaggio di errore |

#### Esempi

```bash
# Inviare form di login (con callback)
curl -X POST \
     -d "auth_method=password&password=yourpassword&callback=app.example.com" \
     -c cookies.txt \
     http://auth.example.com/_login

# Inviare form di login (senza callback, inferirà automaticamente)
curl -X POST \
     -d "auth_method=password&password=yourpassword" \
     -H "X-Forwarded-Host: app.example.com" \
     -c cookies.txt \
     http://auth.example.com/_login

# Warden + Herald: login con il challenge restituito da /_send_verify_code
curl -X POST \
     -d "auth_method=warden&mail=user@example.com&challenge_id=ch_xxx&verify_code=123456&callback=app.example.com" \
     -c cookies.txt \
     http://auth.example.com/_login

# Warden + TOTP: login con un autenticatore già configurato
curl -X POST \
     -d "auth_method=warden&mail=user@example.com&use_otp=true&otp_code=123456&callback=app.example.com" \
     -c cookies.txt \
     http://auth.example.com/_login
```

## Endpoint Invio Codice di Verifica

### `POST /_send_verify_code`

Richiesta di invio codice di verifica. Questo endpoint è utilizzato nel flusso di autenticazione OTP Warden + Herald.

#### Intestazioni di Richiesta (opzionale)

| Intestazione | Descrizione |
|---------------|-------------|
| `Idempotency-Key` | Opzionale. Se presente, Stargate la inoltra a Herald; Herald restituisce la stessa risposta di challenge per richieste duplicate con la stessa chiave entro la TTL. |

#### Corpo della Richiesta

Dati del form `application/x-www-form-urlencoded` o `multipart/form-data`; i corpi JSON non sono supportati:

| Campo | Tipo | Richiesto | Descrizione |
|-------|------|-----------|-------------|
| `phone` | String | No | Numero di telefono; è richiesto almeno uno tra `phone` e `mail` |
| `mail` | String | No | Indirizzo email; è richiesto almeno uno tra `mail` e `phone` |
| `deliver_via` | String | No | Canale preferito: `sms`, `email` o `dingtalk`. Se omesso, Stargate prova prima una destinazione SMS abilitata e poi l'email. DingTalk non viene mai selezionato implicitamente; il client deve inviare `deliver_via=dingtalk`. |

Il gestore accetta anche `phone` + `mail` nella stessa richiesta di codice di verifica.

#### Flusso di Elaborazione

1. **Stargate → Warden** : Query informazioni utente
   - Verificare se l'utente è nella whitelist
   - Verificare lo stato utente (se attivo)
   - Ottenere email e telefono dell'utente

2. **Stargate → Herald** : Creare challenge e inviare codice di verifica
   - Utilizzare email, telefono o identificativo DingTalk restituito da Warden come destinazione
   - Chiamare API Herald per creare challenge
   - Herald invia il codice tramite il canale SMS, email o DingTalk selezionato

3. **Restituire Risultato** : Restituire challenge_id e informazioni correlate

#### Risposta

**Risposta di Successo (200 OK)**

```json
{
  "success": true,
  "message": "Verification code sent",
  "challenge_id": "ch_xxxxxxxxxxxx",
  "expires_in": 300,
  "next_resend_in": 60
}
```

**Risposta di Errore**

| Codice di Stato | Descrizione | Corpo della Risposta |
|-----------------|-------------|----------------------|
| `400 Bad Request` | Parametri richiesta non validi (manca phone o mail) | Messaggio di errore |
| `401 Unauthorized` | Utente assente da Warden o non attivo | Errore JSON con `success=false` |
| `429 Too Many Requests` | Limite di velocità attivato | Messaggio di errore |
| `503 Service Unavailable` | Client o servizio Herald non disponibile | Errore JSON con `success=false` |
| `500 Internal Server Error` | Altro errore del provider durante l'invio | Errore JSON con `success=false` |

#### Esempi

```bash
# Inviare codice di verifica (usando email)
curl -X POST \
     -d "mail=user@example.com" \
     -H "Content-Type: application/x-www-form-urlencoded" \
     http://auth.example.com/_send_verify_code

# Inviare codice di verifica (usando telefono)
curl -X POST \
     -d "phone=13800138000" \
     -H "Content-Type: application/x-www-form-urlencoded" \
     http://auth.example.com/_send_verify_code

```

#### Note

- Richiede `WARDEN_ENABLED=true` e `HERALD_ENABLED=true`
- L'utente deve essere nella whitelist Warden per inviare codice di verifica
- Herald esegue limitazione della velocità, con limiti di frequenza per lo stesso utente/telefono/email
- Il tempo di scadenza del codice è determinato dalla configurazione Herald (predefinito 300 secondi)
- Il tempo di attesa per il reinvio è determinato dalla configurazione Herald (predefinito 60 secondi)

## Endpoint Logout

### `POST /_logout`

Disconnette l'utente corrente e distrugge la sessione.

#### Risposta

**Risposta di Successo (200 OK)**

```
HTTP/1.1 200 OK
Content-Type: text/plain

Logged out
```

Il cookie di sessione sarà cancellato.

#### Esempio

```bash
curl -X POST -b cookies.txt http://auth.example.com/_logout
```

## Endpoint di Autenticazione Step-up

### `GET /_step_up`

Mostra un modulo di riverifica della password per una sessione autenticata. Il parametro di query facoltativo `callback` deve essere un percorso locale; i valori non sicuri vengono sostituiti con `/`.

### `POST /_step_up`

Riverifica la password e registra l'autenticazione step-up nella sessione per 10 minuti. Campi del modulo: `password` e `callback` locale facoltativo. La route è protetta dal controllo same-origin e dal rate limit.

## Endpoint Scambio Sessione

### `GET /_session_exchange`

Utilizzato per la condivisione della sessione cross-domain. Riscatta un ticket cifrato, di breve durata e legato al dominio di destinazione, verifica la sessione autenticata, imposta il cookie e reindirizza al percorso root.

Questo endpoint è principalmente utilizzato per condividere sessioni di autenticazione tra più domini/sottodomini. Dopo che un utente si connette su un dominio, questo endpoint può essere utilizzato per impostare il cookie di sessione su un altro dominio.

#### Parametri Query

| Parametro | Tipo | Richiesto | Descrizione |
|-----------|------|-----------|-------------|
| `ticket` | String | Sì | Ticket di scambio legato al dominio di destinazione e restituito dopo il login |

#### Risposta

**Risposta di Successo (302 Redirect)**

```
HTTP/1.1 302 Found
Location: /
Set-Cookie: stargate_session_id=<session_id>; Path=/; HttpOnly; SameSite=Lax; Domain=<cookie_domain>; Expires=<expiry>
```

**Risposta di Fallimento**

| Codice di Stato | Descrizione | Corpo Risposta |
|----------------|-------------|----------------|
| `400 Bad Request` | Ticket mancante | Messaggio di errore |
| `401 Unauthorized` | Ticket non valido, scaduto, riutilizzato o destinato a un altro dominio | Messaggio di errore |

#### Dominio Cookie

Se la variabile d'ambiente `COOKIE_DOMAIN` è configurata, il cookie sarà impostato al dominio specificato, permettendo la condivisione cross-sottodominio.

#### Esempio

```bash
# Riscattare il ticket sul dominio callback a cui è legato
curl "https://app.example.com/_session_exchange?ticket=<opaque_ticket>"
```

**Scenario di Utilizzo Tipico:**

1. L'utente si connette a `auth.example.com`
2. Dopo login riuscito, reindirizza a `app.example.com/_session_exchange?ticket=<opaque_ticket>`
3. Il cookie di sessione è impostato al dominio `.example.com` (se `COOKIE_DOMAIN=.example.com` è configurato)
4. Reindirizza a `app.example.com/`
5. L'utente può utilizzare questa sessione su tutti i sottodomini `*.example.com`

## Endpoint TOTP

Questi endpoint sono disponibili quando Herald e Herald TOTP sono abilitati. Tutti richiedono una sessione autenticata. La registrazione deve iniziare e terminare entro 10 minuti dal login che ha creato la sessione.

### `GET /totp/enroll`

Mostra una pagina di conferma senza creare lo stato di registrazione. Richiede una sessione creata da un login riuscito negli ultimi 10 minuti. Un utente non autenticato viene reindirizzato con `302 Found` a `/_login`; una sessione più vecchia riceve `401 Unauthorized`.

### `POST /totp/enroll`

Avvia la registrazione e mostra la pagina di associazione TOTP. Richiede una sessione creata da un login riuscito negli ultimi 10 minuti. Un utente non autenticato viene reindirizzato con `302 Found` a `/_login`; una sessione più vecchia riceve `401 Unauthorized`.

### `POST /totp/enroll/confirm`

Conferma la registrazione con dati del form `application/x-www-form-urlencoded` o `multipart/form-data`; i corpi JSON non sono supportati. Sono richiesti una sessione autenticata creata negli ultimi 10 minuti, `enroll_id` e il `code` TOTP a sei cifre. Un'autenticazione assente o non recente restituisce `401 Unauthorized`; in caso di successo la risposta JSON include i codici di backup mostrati una sola volta.

### `GET /totp/revoke`

Mostra la pagina di conferma per rimuovere l'associazione TOTP. È richiesta una sessione autenticata; un utente non autenticato viene reindirizzato con `302 Found` a `/_login`. A questa pagina non si applica il limite di 10 minuti.

### `POST /totp/revoke`

Rimuove l'associazione TOTP. È richiesta una sessione autenticata e il client deve riautenticarsi con `password` o con un `code` TOTP corrente valido. Una riautenticazione assente o fallita restituisce `401 Unauthorized`.

## Endpoint di Salute

### `GET /healthz`

Sonda di liveness del processo senza interrogare Redis, Warden o Herald. Usarla per i controlli del container e le sonde di liveness Kubernetes.

### `GET /readyz`

Stato di readiness aggregato delle dipendenze Redis, Warden e Herald configurate. Usarlo per bilanciatori di carico e sonde di readiness Kubernetes.

```bash
curl http://auth.example.com/healthz
curl http://auth.example.com/readyz
```

`GET /health` rimane un alias di compatibilità deprecato per `GET /readyz`.

## Endpoint Metriche

### `GET /metrics`

Restituisce metriche Prometheus in formato testo. L'endpoint non richiede autenticazione e deve essere protetto tramite regole di rete o del proxy.

## Endpoint del Livello di Log a Runtime

### `GET /log/level`

Restituisce il livello di log corrente. `PUT /log/level` o `POST /log/level` lo modifica con il corpo JSON `{"level":"debug"}` o il parametro `?level=debug`. Stargate limita questo endpoint a `127.0.0.1`; non deve essere esposto tramite il reverse proxy pubblico.

## Endpoint Root

### `GET /`

Percorso root, visualizza informazioni del servizio.

#### Risposta

**200 OK** - Restituisce pagina informazioni del servizio

#### Esempio

```bash
curl http://auth.example.com/
```

## Formato Risposta Errore

Tutte le risposte di errore API selezionano automaticamente il formato in base all'header `Accept` del client:

### Formato JSON (`Accept: application/json`)

```json
{
  "error": "Error message",
  "code": 401
}
```

### Formato XML (`Accept: application/xml`)

```xml
<errors>
  <error code="401">Error message</error>
</errors>
```

### Formato Testo (Predefinito)

```
Error message
```

I messaggi di errore supportano l'internazionalizzazione, restituendo messaggi in cinese o inglese secondo la variabile d'ambiente `LANGUAGE`.

## Esempi Flusso Autenticazione

### Flusso Autenticazione Applicazione Web

1. L'utente accede a una risorsa protetta (es., `https://app.example.com/dashboard`)
2. Traefik intercetta la richiesta e la inoltra a `https://auth.example.com/_auth`
3. Stargate verifica la sessione nel cookie
4. Se non autenticato, reindirizza a `https://auth.example.com/_login?callback=app.example.com`
5. L'utente inserisce la password e invia
6. Stargate verifica la password, crea una sessione, imposta il cookie
7. Reindirizza a `https://app.example.com/_session_exchange?ticket=<opaque_ticket>`
8. Il cookie di sessione è impostato al dominio `app.example.com`
9. L'utente accede nuovamente alla risorsa protetta, l'autenticazione riesce

### Flusso Autenticazione API

1. Il client API invia una richiesta a una risorsa protetta
2. Traefik intercetta la richiesta e la inoltra a `https://auth.example.com/_auth`
3. Il client API include `Stargate-Password: <password>` nell'header della richiesta
4. Stargate verifica la password
5. Se la verifica riesce, imposta l'header `X-Forwarded-User` e restituisce 200
6. Traefik permette alla richiesta di continuare verso il servizio backend

## Note

1. **Tempo di scadenza sessione**: Predefinito 24 ore, richiede nuovo login dopo scadenza
2. **Sicurezza cookie**: Tutti i cookie sono impostati con i flag `HttpOnly` e `SameSite=Lax`
3. **Verifica password**: Le password sono valori opachi e distinguono maiuscole, minuscole e spazi
4. **Supporto più password**: Più password possono essere configurate, qualsiasi password che passa la verifica è accettabile
5. **Sessioni cross-domain**: La variabile d'ambiente `COOKIE_DOMAIN` deve essere configurata per abilitare la condivisione sessione cross-domain
