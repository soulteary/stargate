# Riferimento di Configurazione

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
export PASSWORDS=plaintext:yourpassword
```

**Docker:**

```bash
docker run -e AUTH_HOST=auth.example.com -e PASSWORDS=plaintext:yourpassword stargate:latest
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
PASSWORDS=plaintext:test123

# Più password in testo normale
PASSWORDS=plaintext:test123|admin456|user789

# Hash BCrypt
PASSWORDS=bcrypt:$2a$10$N9qo8uLOickgx2ZMRZoMyeIjZAgcfl7p92ldGxad68LJZdL17lhWy

# Hash SHA512
PASSWORDS=sha512:5e884898da28047151d0e56f8dc6292773603d0d6aabbdd62a11ef721d1542d8
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
LOGIN_PAGE_TITLE=Il Mio Servizio di Autenticazione
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
LOGIN_PAGE_FOOTER_TEXT=© 2024 La Mia Azienda
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
- Il valore dell'intestazione è `authenticated`
- I servizi backend possono determinare se un utente è autenticato tramite questa intestazione
- Deve essere una stringa non vuota

**Esempio:**

```bash
USER_HEADER_NAME=X-Authenticated-User
```

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

### `PORT`

Porta di ascolto del servizio (solo sviluppo locale).

| Attributo | Valore |
|-----------|--------|
| **Tipo** | String |
| **Richiesto** | No |
| **Predefinito** | `80` |

**Descrizione:**

- Solo per ambiente di sviluppo locale
- Generalmente non necessario nei container Docker (utilizza porta predefinita 80)
- Formato: numero porta (es., `8080`) o `:port` (es., `:8080`)

**Esempio:**

```bash
PORT=8080
```

## Configurazione Password

Stargate supporta più algoritmi di crittografia password. Formato configurazione password: `algorithm:password1|password2|password3`

### Algoritmi Supportati

#### `plaintext` - Password in Testo Normale

**Descrizione:**

- Memorizzato in testo normale, nessuna crittografia
- **Solo ambiente di test**
- Fortemente non raccomandato per produzione

**Esempio:**

```bash
PASSWORDS=plaintext:test123|admin456
```

#### `bcrypt` - Hash BCrypt

**Descrizione:**

- Utilizza algoritmo BCrypt per l'hashing
- Alta sicurezza, raccomandato per produzione
- La password deve utilizzare valore hash BCrypt

**Generare Hash BCrypt:**

```bash
# Utilizzando Go
go run -c 'golang.org/x/crypto/bcrypt' <<< 'password'

# Utilizzando strumenti online o altri strumenti
```

**Esempio:**

```bash
PASSWORDS=bcrypt:$2a$10$N9qo8uLOickgx2ZMRZoMyeIjZAgcfl7p92ldGxad68LJZdL17lhWy
```

#### `md5` - Hash MD5

**Descrizione:**

- Utilizza algoritmo MD5 per l'hashing
- Sicurezza inferiore, non raccomandato per produzione
- La password deve utilizzare valore hash MD5 (stringa esadecimale di 32 caratteri)

**Generare Hash MD5:**

```bash
# Linux/macOS
echo -n "password" | md5sum

# O utilizzare strumenti online
```

**Esempio:**

```bash
PASSWORDS=md5:5e884898da28047151d0e56f8dc6292773603d0d6aabbdd62a11ef721d1542d8
```

#### `sha512` - Hash SHA512

**Descrizione:**

- Utilizza algoritmo SHA512 per l'hashing
- Alta sicurezza, raccomandato per produzione
- La password deve utilizzare valore hash SHA512 (stringa esadecimale di 128 caratteri)

**Generare Hash SHA512:**

```bash
# Linux/macOS
echo -n "password" | shasum -a 512

# O utilizzare strumenti online
```

**Esempio:**

```bash
PASSWORDS=sha512:5e884898da28047151d0e56f8dc6292773603d0d6aabbdd62a11ef721d1542d8
```

### Regole Verifica Password

1. **Normalizzazione Password**: Gli spazi sono rimossi e convertiti in maiuscolo prima della verifica
2. **Supporto Più Password**: Più password possono essere configurate, qualsiasi password che passa la verifica è accettabile
3. **Coerenza Algoritmo**: Tutte le password devono utilizzare lo stesso algoritmo

## Esempi di Configurazione

### Configurazione Base

```bash
# Configurazione richiesta
AUTH_HOST=auth.example.com
PASSWORDS=plaintext:test123

# Configurazione opzionale
DEBUG=false
LANGUAGE=en
```

### Configurazione Produzione

```bash
# Configurazione richiesta
AUTH_HOST=auth.example.com
PASSWORDS=bcrypt:$2a$10$N9qo8uLOickgx2ZMRZoMyeIjZAgcfl7p92ldGxad68LJZdL17lhWy

# Configurazione opzionale
DEBUG=false
LANGUAGE=it
LOGIN_PAGE_TITLE=Il Mio Servizio di Autenticazione
LOGIN_PAGE_FOOTER_TEXT=© 2024 La Mia Azienda
USER_HEADER_NAME=X-Forwarded-User
COOKIE_DOMAIN=.example.com
```

### Configurazione Docker Compose

```yaml
services:
  stargate:
    image: stargate:latest
    environment:
      # Configurazione richiesta
      - AUTH_HOST=auth.example.com
      - PASSWORDS=bcrypt:$2a$10$N9qo8uLOickgx2ZMRZoMyeIjZAgcfl7p92ldGxad68LJZdL17lhWy
      
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
PASSWORDS=plaintext:test123|admin456

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

**Esempi di Errori:**

```bash
# Configurazione richiesta mancante
Error: Configuration error: environment variable 'AUTH_HOST' is required but not set.

# Formato password errato
Error: Configuration error: invalid value for environment variable 'PASSWORDS': 'invalid_format'

# Algoritmo non supportato
Error: Configuration error: invalid value for environment variable 'PASSWORDS': 'unknown:password'
```

## Best Practice di Configurazione

1. **Sicurezza Produzione**:
   - Utilizzare algoritmi `bcrypt` o `sha512`, evitare `plaintext`
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
