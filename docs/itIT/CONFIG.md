# Riferimento di Configurazione

> **Traduzione storica**: questa guida non include tutte le modifiche attuali. Consultare la [configurazione inglese aggiornata](../enUS/CONFIG.md) per i requisiti Redis e i limiti di aggiornamento.

Questo documento dettaglia tutte le opzioni di configurazione per Stargate.

## Indice

- [Metodi di Configurazione](#metodi-di-configurazione)
- [Configurazione Richiesta](#configurazione-richiesta)
- [Configurazione Opzionale](#configurazione-opzionale)
- [Configurazione Password](#configurazione-password)
- [Esempi di Configurazione](#esempi-di-configurazione)

## Metodi di Configurazione

Stargate è configurato tramite variabili d'ambiente. Tutti gli elementi di configurazione sono impostati tramite variabili d'ambiente, nessun file di configurazione è necessario.

### Impostazione Variabili d'Ambiente

**Linux/macOS:**

```bash
export AUTH_HOST=auth.example.com
export PASSWORDS='plaintext:yourpassword'
```

**Docker:**

```bash
docker run -e AUTH_HOST=auth.example.com -e PASSWORDS=plaintext:yourpassword ghcr.io/soulteary/stargate:1.3.8
```

**Docker Compose:**

```yaml
services:
  stargate:
    environment:
      - AUTH_HOST=auth.example.com
      - PASSWORDS=plaintext:yourpassword
```

## Configurazione Richiesta

I seguenti elementi di configurazione sono richiesti. Il mancato impostarli impedirà al servizio di avviarsi.

### `AUTH_HOST`

Nome host del servizio di autenticazione.

| Attributo | Valore |
|-----------|--------|
| **Tipo** | String |
| **Richiesto** | Sì |
| **Predefinito** | Nessuno |
| **Esempio** | `auth.example.com` |

**Descrizione:**

- Utilizzato per costruire gli URL callback di login
- Generalmente impostato al nome host del servizio Stargate
- Supporta wildcard `*` (non raccomandato per produzione)

**Esempio:**

```bash
AUTH_HOST=auth.example.com
```

### `PASSWORDS`

Configurazione password, specificando l'algoritmo di crittografia password e la lista password.

| Attributo | Valore |
|-----------|--------|
| **Tipo** | String |
| **Richiesto** | Sì |
| **Predefinito** | Nessuno |
| **Formato** | `algorithm:password1|password2|password3` |

**Descrizione:**

- Formato: `algorithm:password1|password2|password3`
- Supporta più password, separate da `|`
- Qualsiasi password che passa la verifica consente il login
- Algoritmi supportati vedere sezione [Configurazione Password](#configurazione-password)

**Esempi:**

```bash
# Password in testo normale unica
PASSWORDS='plaintext:test123'

# Più password in testo normale
PASSWORDS='plaintext:test123|admin456|user789'

# Hash BCrypt
PASSWORDS='bcrypt:$2a$10$N9qo8uLOickgx2ZMRZoMyeIjZAgcfl7p92ldGxad68LJZdL17lhWy'

```

## Configurazione Opzionale

I seguenti elementi di configurazione sono opzionali. I valori predefiniti sono utilizzati se non impostati.

### `DEBUG`

Abilita modalità debug.

| Attributo | Valore |
|-----------|--------|
| **Tipo** | Boolean |
| **Richiesto** | No |
| **Predefinito** | `false` |
| **Valori Possibili** | `true`, `false` |

**Descrizione:**

- Quando abilitato, il livello di registrazione è impostato a `DEBUG`
- Mostra informazioni di debug più dettagliate
- Raccomandato impostare a `false` in produzione

**Esempio:**

```bash
DEBUG=true
```

### `LANGUAGE`

Lingua dell'interfaccia.

| Attributo | Valore |
|-----------|--------|
| **Tipo** | String |
| **Richiesto** | No |
| **Predefinito** | `en` |
| **Valori Possibili** | `en` (Inglese), `zh` (Cinese), `fr` (Francese), `it` (Italiano), `ja` (Giapponese), `de` (Tedesco), `ko` (Coreano) |

**Descrizione:**

- Influenza la lingua dei messaggi di errore e del testo dell'interfaccia
- Insensibile alle maiuscole/minuscole (`EN`, `en`, `En` funzionano tutti)

**Esempio:**

```bash
LANGUAGE=it
```

### `LOGIN_PAGE_TITLE`

Titolo della pagina di login.

| Attributo | Valore |
|-----------|--------|
| **Tipo** | String |
| **Richiesto** | No |
| **Predefinito** | `Stargate - Login` |

**Descrizione:**

- Visualizzato alla posizione del titolo della pagina di login
- Supporta tag HTML (non raccomandato)

**Esempio:**

```bash
LOGIN_PAGE_TITLE='Il Mio Servizio di Autenticazione'
```

### `LOGIN_PAGE_FOOTER_TEXT`

Testo del piè di pagina della pagina di login.

| Attributo | Valore |
|-----------|--------|
| **Tipo** | String |
| **Richiesto** | No |
| **Predefinito** | `Copyright © 2024 - Stargate` |

**Descrizione:**

- Visualizzato alla posizione del piè di pagina della pagina di login
- Supporta tag HTML (non raccomandato)

**Esempio:**

```bash
LOGIN_PAGE_FOOTER_TEXT='© 2024 La Mia Azienda'
```

### `USER_HEADER_NAME`

Nome dell'intestazione utente impostato dopo autenticazione riuscita.

| Attributo | Valore |
|-----------|--------|
| **Tipo** | String |
| **Richiesto** | No |
| **Predefinito** | `X-Forwarded-User` |

**Descrizione:**

- Dopo autenticazione riuscita, Stargate imposta questa intestazione nella risposta
- Il valore è l'ID dell'utente autenticato quando disponibile; l'autenticazione con sola password usa `authenticated`
- I servizi backend possono determinare se un utente è autenticato tramite questa intestazione
- Deve essere una stringa non vuota

**Esempio:**

```bash
USER_HEADER_NAME=X-Authenticated-User
```

### `TRUSTED_PROXIES` e `PROXY_HEADER`

`TRUSTED_PROXIES` elenca gli IP sorgente o gli intervalli CIDR dei reverse proxy di cui Stargate accetta gli header inoltrati. Il valore predefinito è vuoto; `PROXY_HEADER` indica l'header che contiene l'IP del client (predefinito: `X-Forwarded-For`).

Dietro un reverse proxy, impostare `TRUSTED_PROXIES` anche se Stargate si avvia senza. Finché il proxy non è elencato, Stargate vede solo la richiesta che il proxy ha inviato a `/_auth`, non quella dell'utente:

- il reindirizzamento al login non contiene `callback` ed è costruito con `http://`, quindi dopo l'accesso gli utenti restano sull'host di autenticazione invece di tornare all'applicazione;
- quando il proxy termina HTTPS, gli utenti non possono accedere affatto: il browser invia `Origin: https://...`, Stargate lo confronta con la richiesta `http://` ricevuta dal proxy, e a `POST /_login`, `/_logout`, `/_step_up`, `/_send_verify_code`, `/_send_reauth_code` e `/totp/*` si risponde con `403` e `cross-site request rejected`;
- le regole di percorso step-up non possono essere valutate (per questo `STEP_UP_ENABLED=true` richiede `TRUSTED_PROXIES`);
- ogni client viene attribuito all'indirizzo del proxy e condivide un'unica quota di limitazione della frequenza e un unico indirizzo nei record di audit.

Stargate registra un avviso all'avvio quando `TRUSTED_PROXIES` è vuoto, e un altro la prima volta che ignora gli header inoltrati da un peer non elencato. Registra inoltre un avviso, una sola volta, quando rifiuta come cross-site una richiesta il cui `Origin` corrisponde agli header `X-Forwarded-Proto` e `X-Forwarded-Host` ignorati; la risposta `403` in sé non indica la causa.

### `COOKIE_DOMAIN`

Dominio del cookie, utilizzato per condivisione sessione cross-domain.

| Attributo | Valore |
|-----------|--------|
| **Tipo** | String |
| **Richiesto** | No |
| **Predefinito** | Vuoto (non impostato) |

**Descrizione:**

- Se impostato, i cookie di sessione saranno impostati al dominio specificato
- Supporta condivisione sessione cross-sottodominio
- Formato: `.example.com` (notare il punto iniziale)
- Quando impostato a vuoto, i cookie sono validi solo per il dominio corrente

**Esempio:**

```bash
# Permettere condivisione sessione su tutti i sottodomini *.example.com
COOKIE_DOMAIN=.example.com
```

**Scenario Condivisione Sessione Cross-Domain:**

Supponiamo i seguenti domini:
- `auth.example.com` - Servizio di autenticazione
- `app1.example.com` - Applicazione 1
- `app2.example.com` - Applicazione 2

Dopo aver impostato `COOKIE_DOMAIN=.example.com`:
1. L'utente si connette a `auth.example.com`
2. Il cookie di sessione è impostato al dominio `.example.com`
3. L'utente può utilizzare la stessa sessione su `app1.example.com` e `app2.example.com`

### `SESSION_EXCHANGE_REDIRECT_PATH`

Percorso nella stessa origine dopo uno scambio di sessione verificato. Predefinito: `/`. Per completare la sessione nativa si può usare `/auth/login/example/`. Sono ammessi solo percorsi assoluti configurati dall’amministratore, senza query, frammenti, codifica percentuale o caratteri di controllo. La lista degli host consentiti e la protezione dal riutilizzo restano invariate.

### `PORT`

Porta di ascolto del servizio per distribuzioni native e in container. Gestita dal package config insieme alle altre opzioni da variabili d'ambiente.

| Attributo | Valore |
|-----------|--------|
| **Tipo** | String |
| **Richiesto** | No |
| **Predefinito** | Vuoto per il binario nativo (`:80`); le immagini ufficiali impostano `PORT=8080` |

**Descrizione:**

- Configurabile in sviluppo e in produzione
- I container ufficiali usano `8080` per impostazione predefinita. Usare un numero senza prefisso, come `PORT=8080`, perché il controllo di integrità costruisce un URL.
- Formato del binario nativo: numero porta (es., `8080`) o `:port` (es., `:8080`)

**Esempio:**

```bash
PORT=8080
```

## Configurazione di sicurezza e integrazioni

La tabella è sincronizzata con le variabili di sicurezza realmente registrate in `internal/config`. Un valore opzionale vuoto disabilita la relativa integrazione.

| Variabile | Valori | Predefinito | Scopo |
|-----------|--------|-------------|-------|
| `TRUSTED_PROXIES` | elenco IP/CIDR | vuoto | Sorgenti autorizzate a fornire header Forwarded |
| `PROXY_HEADER` | nome header | `X-Forwarded-For` | Header contenente l'IP sorgente |
| `COOKIE_SECURE` | true/false | true | Attributo Secure dei cookie di sessione |
| `CALLBACK_ALLOWED_HOSTS` | elenco host | vuoto | Destinazioni di redirect consentite |
| `SESSION_EXCHANGE_SECRET` | segreto, minimo 32 caratteri | vuoto | Firma ticket cross-domain di breve durata |
| `SESSION_EXCHANGE_REDIRECT_PATH` | Percorso | `/` | No |
| `PASSWORD_HEADER_AUTH_ENABLED` | true/false | false | Abilita l'header password legacy attendibile |
| `LOG_LEVEL` | debug/info/warn/error | info | Livello di log iniziale |
| `HEADER_AUTH_ENABLED` | true/false | false | Autenticazione tramite header attendibili |
| `HEADER_AUTH_SHARED_SECRET` | segreto, almeno 32 caratteri | vuoto | Segreto condiviso per header auth |
| `HEADER_AUTH_SECRET_HEADER` | nome header | `X-Stargate-Header-Auth` | Trasporta il segreto condiviso |
| `WARDEN_ENABLED` | true/false | false | Integrazione Warden |
| `WARDEN_URL` | URL | vuoto | Endpoint Warden |
| `WARDEN_API_KEY` | segreto | vuoto | Autenticazione API key |
| `WARDEN_HMAC_KEY_ID` | stringa | vuoto | Identificatore chiave HMAC |
| `WARDEN_HMAC_SECRET` | segreto | vuoto | Segreto firma HMAC |
| `WARDEN_TLS_CA_CERT_FILE` | percorso | vuoto | CA personalizzata |
| `WARDEN_TLS_CLIENT_CERT_FILE` | percorso | vuoto | Certificato client mTLS |
| `WARDEN_TLS_CLIENT_KEY_FILE` | percorso | vuoto | Chiave client mTLS |
| `WARDEN_TLS_SERVER_NAME` | stringa | vuoto | Nome server TLS atteso |
| `WARDEN_CACHE_TTL` | intero positivo (secondi) | 300 | Accettato solo per compatibilità; senza effetto |
| `WARDEN_SOURCE_MAX_AGE` | durata | 0s | Età massima dei dati Warden; 0s disabilita il controllo |
| `WARDEN_SOURCE_CLOCK_SKEW` | durata | 0s | Tolleranza dell’orologio, inferiore all’età massima |
| `METRICS_API_KEY` | segreto | vuoto | Chiave Bearer separata per /metrics, almeno 32 byte |
| `HERALD_ENABLED` | true/false | false | Integrazione Herald |
| `HERALD_URL` | URL | vuoto | Endpoint Herald |
| `HERALD_API_KEY` | segreto | vuoto | Autenticazione API key |
| `HERALD_HMAC_KEY_ID` | stringa | vuoto | Identificatore chiave HMAC |
| `HERALD_HMAC_SECRET` | segreto | vuoto | Segreto firma HMAC |
| `HERALD_CLIENT_IP_HEADER` | nome header | vuoto | Header in cui Stargate invia a Herald l'indirizzo del client; deve corrispondere a `HERALD_TRUSTED_PROXY_HEADER` di Herald, e Stargate deve essere in `HERALD_TRUSTED_PROXIES` |
| `HERALD_TLS_CA_CERT_FILE` | percorso | vuoto | CA personalizzata |
| `HERALD_TLS_CLIENT_CERT_FILE` | percorso | vuoto | Certificato client mTLS |
| `HERALD_TLS_CLIENT_KEY_FILE` | percorso | vuoto | Chiave client mTLS |
| `HERALD_TLS_SERVER_NAME` | stringa | vuoto | Nome server TLS atteso |
| `HERALD_TOTP_ENABLED` | true/false | false | Login e gestione TOTP |
| `LOGIN_SMS_ENABLED` | true/false | true | Canale login SMS |
| `LOGIN_EMAIL_ENABLED` | true/false | true | Canale login e-mail |
| `LOGIN_DINGTALK_REQUIRE_USERID` | true/false | false | Richiedere un userid Warden esplicito per DingTalk |
| `LOGIN_UNIFORM_RESPONSES` | true/false | false | Risposte identiche di `/_send_verify_code` e dell'accesso Warden, che l'identificatore sia o no in Warden; il tempo di risposta non viene uniformato |
| `LOGIN_UNIFORM_RESEND_COOLDOWN` | durata | `60s` | Attesa prima di un nuovo invio per identificatore con `LOGIN_UNIFORM_RESPONSES=true`; almeno `1s` e non inferiore al `RESEND_COOLDOWN` di Herald |
| `SESSION_STORAGE_ENABLED` | true/false | false | Storage sessioni Redis |
| `SESSION_STORAGE_REDIS_ADDR` | host:porta | `localhost:6379` | Indirizzo Redis |
| `SESSION_STORAGE_REDIS_PASSWORD` | segreto | vuoto | Password Redis |
| `SESSION_STORAGE_REDIS_DB` | intero | 0 | Database Redis |
| `SESSION_STORAGE_REDIS_KEY_PREFIX` | stringa | `stargate:session:` | Prefisso chiavi Redis |
| `SESSION_STORAGE_REDIS_USERNAME` | stringa | vuoto | Utente ACL Redis; richiede `SESSION_STORAGE_REDIS_PASSWORD` |
| `SESSION_STORAGE_REDIS_TLS_ENABLED` | true/false | false | TLS per la connessione Redis, con verifica del certificato del server |
| `SESSION_STORAGE_REDIS_TLS_CA_CERT_FILE` | percorso | vuoto | CA personalizzata per Redis; richiede `SESSION_STORAGE_REDIS_TLS_ENABLED=true` |
| `SESSION_STORAGE_REDIS_TLS_SERVER_NAME` | stringa | vuoto | Nome server TLS atteso per Redis; richiede `SESSION_STORAGE_REDIS_TLS_ENABLED=true` |
| `AUDIT_LOG_ENABLED` | true/false | true | Registro audit |
| `AUDIT_LOG_FORMAT` | json/text | json | Formato audit |
| `STEP_UP_ENABLED` | true/false | false | Autenticazione aggiuntiva |
| `STEP_UP_PATHS` | elenco percorsi | vuoto | Percorsi applicativi protetti |
| `OTLP_ENABLED` | true/false | false | Esportazione OpenTelemetry |
| `OTLP_ENDPOINT` | URL | vuoto | Endpoint OTLP |
| `AUTH_REFRESH_ENABLED` | true/false | true | Aggiornamento periodico autorizzazioni |
| `AUTH_REFRESH_INTERVAL` | durata | `5m` | Intervallo aggiornamento |
| `REQUEST_CONTEXT_TIMEOUT` | durata | `10s` | Scadenza delle chiamate upstream; nessuna garanzia di annullamento alla disconnessione del client |
| `RATE_LIMIT_LOGIN_MAX` | intero | 10 | Tentativi di accesso per finestra e per client; 0 disabilita |
| `RATE_LIMIT_VERIFICATION_MAX` | intero | 5 | Richieste di verifica per finestra e per client; 0 disabilita |
| `RATE_LIMIT_WINDOW` | durata | `1m` | Finestra fissa di limitazione |

`LOGIN_DINGTALK_REQUIRE_USERID`: Valore predefinito: `false` (modalità compatibile). Con `true` è ammesso solo il valore `dingtalk_userid` gestito da Warden sul server: nessun ripiego sul numero di telefono; associazioni mancanti o non valide vengono rifiutate. La stessa regola vale per la richiesta del codice e per la verifica di un’associazione esistente. L’opzione non si attiva automaticamente e non modifica le sessioni esistenti né i limiti di revoca della cache. Vedere il [profilo userid in inglese](../enUS/DINGTALK_USERID_ONLY.md).

`PORT` vale per sviluppo e produzione. Il binario nativo usa `:80` senza un valore; le immagini ufficiali impostano `PORT=8080`. Nei container usare solo un numero, ad esempio `PORT=8080`: il controllo di salute costruisce un URL, quindi la forma nativa `:8080` non è adatta.

Warden richiede ID e segreto HMAC insieme. Per Herald, impostare un ID esplicito se il server usa `HERALD_HMAC_KEYS`; un singolo `HMAC_SECRET` mantiene il valore implicito. Certificato e chiave client mTLS devono formare una coppia completa per entrambi i servizi.

Se `WARDEN_URL` o `HERALD_URL` è un URL `http://` ed è configurata una chiave API o un segreto HMAC, Stargate registra un avviso all'avvio: la chiave API e tutti i dati di richiesta e risposta attraversano allora la rete non cifrati. Usare `https://`, a meno che la connessione a Warden o a Herald resti su una rete privata, ad esempio una rete interna di Compose.

`WARDEN_CACHE_TTL` è accettato solo per compatibilità e non ha alcun effetto sul comportamento di Stargate. Il valore viene passato al client Warden come durata della sua cache dell'elenco utenti, ma Stargate non richiede mai quell'elenco: il login, l'invio di un codice, l'autenticazione tramite header attendibile e l'aggiornamento dell'autorizzazione cercano ciascuno un solo utente per numero di telefono o indirizzo e-mail, e queste ricerche non vengono memorizzate nella cache. Ognuna è una richiesta a Warden. Dopo quanto tempo un utente disabilitato o rimosso perde una sessione esistente, e con quale frequenza una sessione provoca una richiesta a Warden, dipende da `AUTH_REFRESH_INTERVAL`; un intervallo più lungo riduce le richieste a Warden ma ritarda anche l'effetto della disabilitazione. Con `HEADER_AUTH_ENABLED=true`, ogni richiesta inoltrata che contiene un header di identità attendibile provoca una ricerca su Warden. Il valore deve comunque essere un intero positivo, altrimenti Stargate non si avvia.

Le impostazioni `SESSION_STORAGE_REDIS_*` hanno effetto solo con `SESSION_STORAGE_ENABLED=true`. Se una di esse ha un valore diverso da quello predefinito mentre `SESSION_STORAGE_ENABLED` è `false`, Stargate si avvia, mantiene sessioni e contatori di limitazione della frequenza in memoria e registra all'avvio un avviso che elenca le variabili ignorate.

L'utente indicato in `SESSION_STORAGE_REDIS_USERNAME` necessita dei permessi di lettura, scrittura e script (`EVAL`/`EVALSHA`) sulle chiavi sotto `SESSION_STORAGE_REDIS_KEY_PREFIX`. Senza `SESSION_STORAGE_REDIS_TLS_ENABLED=true`, sessioni, contesti di verifica e contatori di limitazione della frequenza viaggiano verso Redis non cifrati; usarlo così solo su una rete privata. Con TLS (1.2 o successivo) il certificato viene verificato rispetto alle radici di sistema e all'host in `SESSION_STORAGE_REDIS_ADDR`, salvo diversa indicazione di `SESSION_STORAGE_REDIS_TLS_CA_CERT_FILE` (file PEM con certificati CA aggiuntivi) e `SESSION_STORAGE_REDIS_TLS_SERVER_NAME`. La verifica non può essere disattivata; i certificati client non sono supportati.

## Configurazione Password

Stargate accetta `bcrypt` in produzione e `plaintext` solo per test locali. MD5 e SHA-512 senza salt vengono rifiutati durante la validazione. Le password distinguono maiuscole e minuscole e conservano gli spazi. Le password dimostrative pubblicate nell'esempio Compose di questo repository (`test1234`, `test1337`) vengono rifiutate all'avvio quando `COOKIE_SECURE` è `true`; sono accettate solo per la demo HTTP locale.

```bash
PASSWORDS='bcrypt:<bcrypt-hash>'
# Solo test locali:
PASSWORDS='plaintext:test123'
```

## Esempi di Configurazione

### Configurazione Base

```bash
# Configurazione richiesta
AUTH_HOST=auth.example.com
PASSWORDS='plaintext:test123'

# Configurazione opzionale
DEBUG=false
LANGUAGE=en
```

### Configurazione Produzione

```bash
# Configurazione richiesta
AUTH_HOST=auth.example.com
PASSWORDS='bcrypt:$2a$10$N9qo8uLOickgx2ZMRZoMyeIjZAgcfl7p92ldGxad68LJZdL17lhWy'

# Configurazione opzionale
DEBUG=false
LANGUAGE=it
LOGIN_PAGE_TITLE='Il Mio Servizio di Autenticazione'
LOGIN_PAGE_FOOTER_TEXT='© 2024 La Mia Azienda'
USER_HEADER_NAME=X-Forwarded-User
COOKIE_DOMAIN=.example.com
```

### Configurazione Docker Compose

```yaml
services:
  stargate:
    image: ghcr.io/soulteary/stargate:1.3.8
    environment:
      # Configurazione richiesta
      - AUTH_HOST=auth.example.com
      - PASSWORDS=bcrypt:$$2a$$10$$N9qo8uLOickgx2ZMRZoMyeIjZAgcfl7p92ldGxad68LJZdL17lhWy
      
      # Configurazione opzionale
      - DEBUG=false
      - LANGUAGE=it
      - LOGIN_PAGE_TITLE=Il Mio Servizio di Autenticazione
      - LOGIN_PAGE_FOOTER_TEXT=© 2024 La Mia Azienda
      - COOKIE_DOMAIN=.example.com
```

### Configurazione Sviluppo Locale

```bash
# Configurazione richiesta
AUTH_HOST=localhost
PASSWORDS='plaintext:test123|admin456'

# Configurazione opzionale
DEBUG=true
LANGUAGE=it
PORT=8080
```

## Validazione Configurazione

Stargate valida tutti gli elementi di configurazione all'avvio:

1. **Verifica Configurazione Richiesta**: Se la configurazione richiesta non è impostata, il servizio fallirà all'avvio e mostrerà un messaggio di errore
2. **Validazione Formato**: Un formato di configurazione password errato causerà un fallimento all'avvio
3. **Validazione Algoritmo**: Algoritmi password non supportati causeranno un fallimento all'avvio
4. **Validazione Valore**: Alcuni elementi di configurazione hanno restrizioni di valore (es., `LANGUAGE`, `DEBUG`)
5. **Validazione Regola**: Alcuni elementi di configurazione dipendono da altri o hanno una lunghezza minima o un formato obbligatorio (es., `HERALD_ENABLED` richiede `WARDEN_ENABLED=true`)

Il messaggio indica la variabile e specifica quale di questi casi si applica. Una regola viene riportata come motivo; non viene mostrata come valore configurato, e i valori configurati dei segreti non vengono mai stampati.

**Esempi di Errori:**

```bash
# Configurazione richiesta mancante
Error: Configuration error: environment variable 'AUTH_HOST' is required but not set.

# Formato password errato
Error: Configuration error: invalid value for environment variable 'PASSWORDS': 'invalid_format'

# Algoritmo non supportato
Error: Configuration error: invalid value for environment variable 'PASSWORDS': 'unknown:password'

# Valore fuori dall'insieme accettato
Error: Configuration error: invalid value for environment variable 'DEBUG': 'maybe'.
  Accepted values: [true false]

# Regola che coinvolge un'altra impostazione
Error: Configuration error: environment variable 'HERALD_ENABLED': requires WARDEN_ENABLED=true to resolve authenticated users.
```

## Best Practice di Configurazione

1. **Sicurezza Produzione**:
   - Utilizzare `bcrypt` in produzione; limitare `plaintext` ai test locali
   - Impostare `DEBUG=false`
   - Utilizzare password forti

2. **Sessioni Cross-Domain**:
   - Se è necessario condividere sessioni tra sottodomini, impostare `COOKIE_DOMAIN`
   - Formato: `.example.com` (notare il punto iniziale)

3. **Supporto Multilingue**:
   - Impostare `LANGUAGE` secondo la base utenti
   - Supporta `en`, `zh`, `fr`, `it`, `ja`, `de`, `ko`

4. **Interfaccia Personalizzata**:
   - Utilizzare `LOGIN_PAGE_TITLE` e `LOGIN_PAGE_FOOTER_TEXT` per personalizzare la pagina di login

5. **Monitoraggio e Debug**:
   - Impostare `DEBUG=true` nell'ambiente di sviluppo per log dettagliati
   - Impostare `DEBUG=false` nell'ambiente di produzione per ridurre l'output dei log

## Autenticazione legacy tramite header password

L'autenticazione `Stargate-Password` è disabilitata per impostazione predefinita. Abilitarla solo per client legacy attendibili con `PASSWORD_HEADER_AUTH_ENABLED=true`. Usare HTTPS, applicare limiti di frequenza e rimuovere l'header nel reverse proxy prima di inoltrare la richiesta al backend.

I tentativi falliti con questo header su `/_auth` vengono conteggiati per indirizzo del client nella quota `RATE_LIMIT_LOGIN_MAX`. Restano conteggiati solo i valori errati. Esaurita la quota, a ogni richiesta da quell'indirizzo che contiene l'header si risponde con `429` e `Retry-After` prima del confronto della password, fino al termine della finestra, anche con la password corretta. Una richiesta viene conteggiata mentre la sua password viene confrontata e scalata quando la password è corretta; valori errati inviati contemporaneamente non possono quindi superare insieme la quota. Per lo stesso motivo, un client che ha in corso più richieste con l'header di quante la quota ne consenta ancora riceve `429` per quelle in eccesso e può ripeterle. I client che condividono un indirizzo condividono questa quota:

- i client dietro lo stesso NAT o proxy in uscita sono un unico indirizzo; se uno di essi invia valori errati, blocca gli altri per il resto della finestra e può ripeterlo a ogni finestra;
- con `TRUSTED_PROXIES` vuoto, o dietro un proxy o una CDN non elencati, ogni richiesta viene attribuita al proxy, quindi ciò vale per tutti i client del deployment che usano l'header. Stargate registra un avviso all'avvio per `PASSWORD_HEADER_AUTH_ENABLED=true` con `TRUSTED_PROXIES` vuoto.

Impostare `TRUSTED_PROXIES` affinché la quota si applichi per client. Le sessioni basate su cookie non sono interessate da questa quota.

## Validità della fonte e monitoraggio

`WARDEN_SOURCE_MAX_AGE=0s` resta il valore predefinito per compatibilità.
Un valore positivo richiede i metadati Warden v1 e
`AUTH_REFRESH_ENABLED=true`. `WARDEN_SOURCE_CLOCK_SKEW` viene sottratto alla
validità. Le sessioni non estendono una fonte scaduta; gli errori della fonte
impediscono l’accesso con HTTP 503.

Senza `METRICS_API_KEY`, `/metrics` restituisce HTTP 404. Prima dell’aggiornamento,
configurare una chiave casuale separata di almeno 32 byte e il relativo scraper con
`Authorization: Bearer <secret>`. Non riutilizzare credenziali di servizio o di
accesso. Tra host usare HTTPS con verifica del certificato; HTTP richiede una
rete locale isolata e attendibile.
Per le regole complete e la migrazione consultare la
[guida inglese](../enUS/CONFIG.md#source-freshness-and-scrape-credentials).


### `LOGIN_SUPPORT_URL`

Link HTTPS facoltativo o una sola casella `mailto:help@example.com`. Vuoto nasconde il link. Credenziali, caratteri di controllo e intestazioni mailto non sono consentiti.
