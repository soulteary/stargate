# Documentazione API

Questo documento descrive in dettaglio tutti gli endpoint API del servizio Stargate Forward Auth.

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

Elabora le richieste di login, verifica la password e crea una sessione.

#### Corpo Richiesta

Dati form (`application/x-www-form-urlencoded`):

| Campo | Tipo | Richiesto | Descrizione |
|-------|------|-----------|-------------|
| `password` | String | Sì | Password utente |
| `callback` | String | No | URL callback dopo login riuscito |

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
   - Reindirizza a `{callback}/_session_exchange?id={session_id}`
   - Codice di stato: `302 Found`

2. **Senza callback**:
   - **Richiesta HTML**: Restituisce una pagina HTML con meta refresh, reindirizzando automaticamente al dominio di origine
   - **Richiesta API**: Restituisce una risposta JSON
     ```json
     {
       "success": true,
       "message": "Login successful",
       "session_id": "<session_id>"
     }
     ```

**Risposta di Fallimento**

| Codice di Stato | Descrizione | Corpo Risposta |
|----------------|-------------|----------------|
| `401 Unauthorized` | Password errata | Messaggio di errore in formato JSON/XML/testo secondo header Accept |
| `500 Internal Server Error` | Errore server | Messaggio di errore |

#### Esempi

```bash
# Inviare form di login (con callback)
curl -X POST \
     -d "password=yourpassword&callback=app.example.com" \
     -c cookies.txt \
     http://auth.example.com/_login

# Inviare form di login (senza callback, inferirà automaticamente)
curl -X POST \
     -d "password=yourpassword" \
     -H "X-Forwarded-Host: app.example.com" \
     -c cookies.txt \
     http://auth.example.com/_login
```

## Endpoint Logout

### `GET /_logout`

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
curl -b cookies.txt http://auth.example.com/_logout
```

## Endpoint Scambio Sessione

### `GET /_session_exchange`

Utilizzato per condivisione sessione cross-domain. Imposta il cookie ID sessione specificato e reindirizza al percorso root.

Questo endpoint è principalmente utilizzato per condividere sessioni di autenticazione tra più domini/sottodomini. Dopo che un utente si connette su un dominio, questo endpoint può essere utilizzato per impostare il cookie di sessione su un altro dominio.

#### Parametri Query

| Parametro | Tipo | Richiesto | Descrizione |
|-----------|------|-----------|-------------|
| `id` | String | Sì | ID sessione da impostare |

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
| `400 Bad Request` | ID sessione mancante | Messaggio di errore |

#### Dominio Cookie

Se la variabile d'ambiente `COOKIE_DOMAIN` è configurata, il cookie sarà impostato al dominio specificato, permettendo la condivisione cross-sottodominio.

#### Esempio

```bash
# Impostare cookie sessione (per scenari cross-domain)
curl "http://auth.example.com/_session_exchange?id=<session_id>"
```

**Scenario di Utilizzo Tipico:**

1. L'utente si connette a `auth.example.com`
2. Dopo login riuscito, reindirizza a `app.example.com/_session_exchange?id=<session_id>`
3. Il cookie di sessione è impostato al dominio `.example.com` (se `COOKIE_DOMAIN=.example.com` è configurato)
4. Reindirizza a `app.example.com/`
5. L'utente può utilizzare questa sessione su tutti i sottodomini `*.example.com`

## Endpoint Verifica Salute

### `GET /health`

Endpoint verifica salute del servizio. Utilizzato per monitorare lo stato del servizio.

#### Risposta

**Risposta di Successo (200 OK)**

```
HTTP/1.1 200 OK
```

#### Esempio

```bash
curl http://auth.example.com/health
```

**Utilizzi Tipici:**

- Verifiche salute Docker
- Sonde liveness Kubernetes
- Verifiche salute bilanciatore di carico

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
7. Reindirizza a `https://app.example.com/_session_exchange?id=<session_id>`
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
3. **Verifica password**: Le password sono normalizzate prima della verifica (rimuovere spazi, convertire in maiuscolo)
4. **Supporto più password**: Più password possono essere configurate, qualsiasi password che passa la verifica è accettabile
5. **Sessioni cross-domain**: La variabile d'ambiente `COOKIE_DOMAIN` deve essere configurata per abilitare la condivisione sessione cross-domain
