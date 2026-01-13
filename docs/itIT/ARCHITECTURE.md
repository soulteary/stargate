# Documento di Architettura Stargate

Questo documento descrive l'architettura tecnica e le decisioni di progettazione del progetto Stargate.

## Stack Tecnologico

- **Linguaggio**: Go 1.25
- **Framework Web**: [Fiber v2.52.10](https://github.com/gofiber/fiber)
- **Motore Template**: [Fiber Template v1.7.5](https://github.com/gofiber/template)
- **Gestione Sessioni**: Middleware Session Fiber
- **Registrazione**: [Logrus v1.9.3](https://github.com/sirupsen/logrus)
- **Output Terminale**: [Pterm v0.12.82](https://github.com/pterm/pterm)
- **Framework di Test**: [Testza v0.5.2](https://github.com/MarvinJWendt/testza)

## Struttura del Progetto

```
codes/src/
├── cmd/stargate/          # Punto di ingresso dell'applicazione
│   ├── main.go            # Funzione principale, inizializza la configurazione e avvia il server
│   ├── server.go          # Configurazione del server e configurazione delle route
│   └── constants.go       # Costanti di route e configurazione
│
├── internal/              # Package interni (non esposti esternamente)
│   ├── auth/              # Logica di autenticazione
│   │   ├── auth.go        # Funzionalità principale di autenticazione
│   │   └── auth_test.go   # Test di autenticazione
│   │
│   ├── config/            # Gestione della configurazione
│   │   ├── config.go      # Definizioni e inizializzazione delle variabili di configurazione
│   │   ├── validation.go  # Logica di validazione della configurazione
│   │   └── config_test.go # Test di configurazione
│   │
│   ├── handlers/          # Gestori di richieste HTTP
│   │   ├── check.go       # Gestore di verifica autenticazione
│   │   ├── login.go       # Gestore di login
│   │   ├── logout.go      # Gestore di logout
│   │   ├── session_share.go # Gestore di condivisione sessione
│   │   ├── health.go       # Gestore di verifica salute
│   │   ├── index.go        # Gestore del percorso root
│   │   ├── utils.go        # Funzioni di utilità dei gestori
│   │   └── handlers_test.go # Test dei gestori
│   │
│   ├── i18n/              # Supporto internazionalizzazione
│   │   └── i18n.go        # Traduzioni multilingue
│   │
│   ├── middleware/        # Middleware HTTP
│   │   └── log.go         # Middleware di registrazione
│   │
│   ├── secure/            # Algoritmi di crittografia password
│   │   ├── interface.go   # Interfaccia algoritmo di crittografia
│   │   ├── plaintext.go   # Password in testo normale (solo test)
│   │   ├── bcrypt.go      # Algoritmo BCrypt
│   │   ├── md5.go         # Algoritmo MD5
│   │   ├── sha512.go      # Algoritmo SHA512
│   │   └── secure_test.go # Test algoritmi di crittografia
│   │
│   └── web/               # Risorse Web
│       └── templates/     # Template HTML
│           ├── login.html # Template pagina di login
│           └── assets/   # Risorse statiche
│               └── favicon.ico
```

## Componenti Principali

### 1. Sistema di Autenticazione (`internal/auth`)

Il sistema di autenticazione è responsabile di:
- Verifica password (supporta più algoritmi di crittografia)
- Gestione sessioni (creare, verificare, distruggere)
- Verifica stato autenticazione

**Funzioni Chiave:**
- `CheckPassword(password string) bool`: Verifica la password
- `Authenticate(session *session.Session) error`: Segna la sessione come autenticata
- `IsAuthenticated(session *session.Session) bool`: Verifica se la sessione è autenticata
- `Unauthenticate(session *session.Session) error`: Distrugge la sessione

### 2. Sistema di Configurazione (`internal/config`)

Il sistema di configurazione fornisce:
- Gestione variabili d'ambiente
- Validazione configurazione
- Supporto valori predefiniti

**Variabili di Configurazione:**
- `AUTH_HOST`: Nome host autenticazione (richiesto)
- `PASSWORDS`: Configurazione password (lista algoritmo:password) (richiesto)
- `DEBUG`: Modalità debug (predefinito: false)
- `LANGUAGE`: Lingua interfaccia (predefinito: en, supporta en/zh/fr/it/ja/de/ko)
- `COOKIE_DOMAIN`: Dominio cookie (opzionale, per condivisione sessione cross-domain)
- `LOGIN_PAGE_TITLE`: Titolo pagina login (predefinito: Stargate - Login)
- `LOGIN_PAGE_FOOTER_TEXT`: Testo piè di pagina login (predefinito: Copyright © 2024 - Stargate)
- `USER_HEADER_NAME`: Nome header utente impostato dopo autenticazione riuscita (predefinito: X-Forwarded-User)
- `PORT`: Porta di ascolto servizio (solo sviluppo locale, predefinito: 80)

### 3. Gestori di Richieste (`internal/handlers`)

I gestori sono responsabili dell'elaborazione delle richieste HTTP:

- **CheckRoute**: Verifica autenticazione Traefik Forward Auth
- **LoginRoute/LoginAPI**: Pagina login ed elaborazione login
- **LogoutRoute**: Elaborazione logout
- **SessionShareRoute**: Condivisione sessione cross-domain
- **HealthRoute**: Verifica salute
- **IndexRoute**: Elaborazione percorso root

### 4. Crittografia Password (`internal/secure`)

Supporta più algoritmi di crittografia password:
- `plaintext`: Testo normale (solo test)
- `bcrypt`: Hash BCrypt
- `md5`: Hash MD5
- `sha512`: Hash SHA512

Tutti gli algoritmi implementano l'interfaccia `HashResolver`:
```go
type HashResolver interface {
    Check(h string, password string) bool
}
```

## Flusso di Lavoro

### Flusso di Autenticazione

1. **L'utente accede a una risorsa protetta**
   - Traefik intercetta la richiesta
   - Inoltra all'endpoint Stargate `/_auth`

2. **Stargate verifica l'autenticazione**
   - Verifica prima l'header `Stargate-Password` (autenticazione API)
   - Se l'autenticazione header fallisce, verifica il cookie `stargate_session_id` (autenticazione Web)

3. **Autenticazione riuscita**
   - Imposta l'header `X-Forwarded-User` (o nome header utente configurato) con valore "authenticated"
   - Restituisce 200 OK
   - Traefik permette alla richiesta di continuare

4. **Autenticazione fallita**
   - Richieste HTML: Reindirizza alla pagina di login (`/_login?callback=<originalURL>`)
   - Richieste API (JSON/XML): Restituisce 401 Unauthorized

### Flusso di Login

1. **L'utente accede alla pagina di login**
   - `GET /_login?callback=<url>`
   - Se già connesso, reindirizza all'endpoint scambio sessione
   - Se il dominio differisce, memorizza il callback nel cookie (`stargate_callback`)

2. **Invio form di login**
   - `POST /_login` con password
   - Verifica la password
   - Crea una sessione e imposta il cookie
   - **Priorità recupero callback**:
     1. Dal cookie (se precedentemente impostato)
     2. Dai dati del form
     3. Dai parametri query
     4. Se nessuno dei precedenti, e il dominio di origine differisce dal dominio del servizio di autenticazione, utilizzare il dominio di origine come callback

3. **Scambio sessione**
   - Se il callback esiste, reindirizza a `{callback}/_session_exchange?id=<session_id>`
   - `GET /_session_exchange?id=<session_id>`
   - Imposta il cookie sessione (se `COOKIE_DOMAIN` è configurato, imposta al dominio specificato)
   - Reindirizza al percorso root `/`

## Considerazioni di Sicurezza

### Sicurezza Sessione

- I cookie utilizzano il flag `HttpOnly` per prevenire attacchi XSS
- I cookie utilizzano `SameSite=Lax` per prevenire attacchi CSRF
- Il percorso del cookie è impostato a `/`, permettendo l'uso su tutto il dominio
- Tempo di scadenza sessione: 24 ore (`config.SessionExpiration`)
- Supporta dominio cookie personalizzato (per scenari cross-domain)
- Gli ID sessione sono generati utilizzando UUID per garantire unicità e sicurezza

### Sicurezza Password

- Supporta più algoritmi di crittografia (raccomandato utilizzare bcrypt o sha512)
- Configurazione password trasmessa via variabili d'ambiente, non memorizzata nel codice
- Normalizzazione password durante la verifica (rimuovere spazi, convertire in maiuscolo)

### Sicurezza Richieste

- L'endpoint di verifica autenticazione supporta due metodi di autenticazione:
  - Autenticazione header (`Stargate-Password`): Per richieste API
  - Autenticazione cookie: Per richieste Web
- Distingue tra richieste HTML e API, restituisce risposte appropriate

## Estensibilità

### Aggiunta Nuovi Algoritmi Password

1. Creare nuova implementazione algoritmo in `internal/secure/`
2. Implementare l'interfaccia `HashResolver`
3. Registrare l'algoritmo in `config/validation.go`

### Aggiunta Nuove Lingue

1. Aggiungere costante lingua in `internal/i18n/i18n.go`
2. Aggiungere mapping traduzione
3. Aggiungere opzione lingua nella configurazione

### Personalizzazione Pagina Login

Modificare il file template `internal/web/templates/login.html`.

## Ottimizzazione Prestazioni

- Utilizza il framework Fiber, basato su fasthttp, prestazioni eccellenti
- Sessioni memorizzate in memoria per accesso rapido
- Risorse statiche servite via servizio file statici Fiber
- Supporta modalità debug, può essere disabilitata in produzione

## Architettura Deployment

### Deployment Docker

- Build multi-stage per ridurre dimensione immagine
- Utilizza `golang:1.25-alpine` come stage di build
- Utilizza immagine base `scratch` come stage di esecuzione per minimizzare rischi sicurezza
- File template copiati da `src/internal/web/templates` a `/app/web/templates` nell'immagine
- Utilizza sorgente mirror cinese (`GOPROXY=https://goproxy.cn`) per accelerare download dipendenze
- Utilizza `-ldflags "-s -w"` durante compilazione per ridurre dimensione binario
- L'applicazione trova automaticamente i percorsi template (supporta `./internal/web/templates` per sviluppo locale e `./web/templates` per produzione)

### Integrazione Traefik

- Integrato via middleware Forward Auth
- Supporta HTTP e HTTPS
- Supporta più domini e regole percorso

## Registrazione e Monitoraggio

- Utilizza Logrus per la registrazione
- Supporta modalità debug (DEBUG=true)
- Tutte le operazioni critiche sono registrate
- Endpoint verifica salute disponibile per monitoraggio

## Test

- I test unitari coprono la funzionalità principale
- File di test situati nei file `*_test.go` di ogni package
- Utilizza `testza` per le asserzioni
- Copertura test include:
  - Logica autenticazione (`internal/auth/auth_test.go`)
  - Validazione configurazione (`internal/config/config_test.go`)
  - Algoritmi crittografia password (`internal/secure/secure_test.go`)
  - Gestori HTTP (`internal/handlers/handlers_test.go`)

## Miglioramenti Futuri

- [ ] Supportare più algoritmi di crittografia password
- [ ] Supportare OAuth2/OpenID Connect
- [ ] Supportare gestione multi-utente e ruoli
- [ ] Aggiungere interfaccia amministrazione
- [ ] Supportare storage sessione esterno (Redis, ecc.)
- [ ] Aggiungere esportazione metriche Prometheus
- [ ] Supportare file di configurazione (YAML/JSON)
