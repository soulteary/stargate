# Stargate - Forward Auth Service

[![License](https://img.shields.io/badge/license-Apache%202.0-blue.svg)](LICENSE)
[![Go Version](https://img.shields.io/badge/go-1.25+-blue.svg)](https://golang.org)
[![codecov](https://codecov.io/gh/soulteary/stargate/branch/main/graph/badge.svg)](https://codecov.io/gh/soulteary/stargate)

> **🚀 Il Tuo Gateway verso Microservizi Sicuri**

Stargate è un servizio di autenticazione forward pronto per la produzione, leggero, progettato per essere il **punto di autenticazione unico** per tutta la tua infrastruttura. Costruito con Go e ottimizzato per le prestazioni, Stargate si integra perfettamente con Traefik e altri proxy inversi per proteggere i tuoi servizi backend—**senza scrivere una sola riga di codice di autenticazione nelle tue applicazioni**.

## 🌐 Documentazione Multilingue

- [English](README.md) | [中文](README.zhCN.md) | [Français](README.frFR.md) | [Italiano](README.itIT.md) | [日本語](README.jaJP.md) | [Deutsch](README.deDE.md) | [한국어](README.koKR.md)

![Anteprima](.github/assets/preview.png)

### 🎯 Perché Stargate?

Stanco di implementare la logica di autenticazione in ogni servizio? Stargate risolve questo problema centralizzando l'autenticazione al bordo, permettendoti di:

- ✅ **Proteggere più servizi** con un unico strato di autenticazione
- ✅ **Ridurre la complessità del codice** rimuovendo la logica di autenticazione dalle tue applicazioni
- ✅ **Distribuire in pochi minuti** con Docker e una configurazione semplice
- ✅ **Scalare senza sforzo** con un'impronta di risorse minima
- ✅ **Mantenere la sicurezza** con più algoritmi di crittografia e gestione sicura delle sessioni

### 💼 Casi d'Uso

Stargate è perfetto per:

- **Architettura di Microservizi**: Proteggere più servizi backend senza modificare il codice dell'applicazione
- **Applicazioni Multi-Dominio**: Condividere sessioni di autenticazione tra diversi domini e sottodomini
- **Strumenti Interni e Dashboard**: Aggiungere rapidamente l'autenticazione a servizi interni e pannelli di amministrazione
- **Integrazione Gateway API**: Utilizzare con Traefik, Nginx o altri proxy inversi come strato di autenticazione unificato
- **Sviluppo e Test**: Autenticazione semplice basata su password per ambienti di sviluppo

## 📋 Indice

- [Funzionalità](#funzionalità)
- [Avvio Rapido](#avvio-rapido)
- [Configurazione](#configurazione)
- [Documentazione](#documentazione)
- [Documentazione API](#documentazione-api)
- [Guida al Deployment](#guida-al-deployment)
- [Guida allo Sviluppo](#guida-allo-sviluppo)
- [Licenza](#licenza)

## ✨ Funzionalità

### 🔐 Sicurezza di Livello Aziendale

- **Più Algoritmi di Crittografia Password**: Scegli tra plaintext (test), bcrypt, MD5, SHA512 e altro ancora
- **Gestione Sicura delle Sessioni**: Sessioni basate su Cookie con dominio e scadenza personalizzabili
- **Autenticazione Flessibile**: Supporto per autenticazione basata su password e basata su sessione

### 🌐 Capacità Avanzate

- **Condivisione Sessioni Cross-Domain**: Condividere perfettamente le sessioni di autenticazione tra diversi domini/sottodomini
- **Supporto Multilingue**: Interfacce integrate in inglese e cinese, facilmente estendibili per più lingue
- **Interfaccia Personalizzabile**: Personalizza la tua pagina di login con titoli e testi di piè di pagina personalizzati

### 🚀 Prestazioni e Affidabilità

- **Leggero e Veloce**: Costruito su Go e il framework Fiber per prestazioni eccezionali
- **Utilizzo Minimo delle Risorse**: Impronta di memoria ridotta, perfetto per ambienti containerizzati
- **Pronto per la Produzione**: Architettura testata in battaglia progettata per l'affidabilità

### 📦 Esperienza Sviluppatore

- **Docker First**: Immagine Docker completa e configurazione docker-compose pronte all'uso
- **Traefik Nativo**: Integrazione middleware Traefik Forward Auth a zero configurazione
- **Configurazione Semplice**: Configurazione basata su variabili d'ambiente, nessun file complesso necessario

## 🚀 Avvio Rapido

Metti Stargate in funzione in **meno di 2 minuti**!

### Utilizzo di Docker Compose (Consigliato)

**Passo 1:** Clona il repository
```bash
git clone <repository-url>
cd forward-auth
```

**Passo 2:** Configura la tua autenticazione (modifica `codes/docker-compose.yml`)
```yaml
services:
  stargate:
    environment:
      - AUTH_HOST=auth.example.com
      - PASSWORDS=plaintext:yourpassword1|yourpassword2
```

**Passo 3:** Avvia il servizio
```bash
cd codes
docker-compose up -d
```

**Ecco fatto!** Il tuo servizio di autenticazione è ora in esecuzione. 🎉

### Sviluppo Locale

1. Assicurati che Go 1.25 o superiore sia installato

2. Naviga nella directory del progetto:
```bash
cd codes
```

3. Esegui lo script di avvio locale:
```bash
chmod +x start-local.sh
./start-local.sh
```

4. Accedi alla pagina di login:
```
http://localhost:8080/_login?callback=localhost
```

## ⚙️ Configurazione

Stargate utilizza un sistema di configurazione semplice basato su variabili d'ambiente. Nessun file YAML complesso o parsing di configurazione—basta impostare variabili d'ambiente e sei pronto.

### Configurazione Richiesta

| Variabile d'Ambiente | Descrizione | Esempio |
|---------------------|-------------|---------|
| `AUTH_HOST` | Nome host del servizio di autenticazione | `auth.example.com` |
| `PASSWORDS` | Configurazione password, formato: `algorithm:password1\|password2\|password3` | `plaintext:test123\|admin456` |

### Configurazione Opzionale

| Variabile d'Ambiente | Descrizione | Predefinito | Esempio |
|---------------------|-------------|-------------|---------|
| `DEBUG` | Abilita modalità debug | `false` | `true` |
| `LANGUAGE` | Lingua dell'interfaccia | `en` | `it` (Italiano), `zh` (Cinese), `en` (Inglese), `fr` (Francese), `ja` (Giapponese), `de` (Tedesco), `ko` (Coreano) |
| `LOGIN_PAGE_TITLE` | Titolo della pagina di login | `Stargate - Login` | `Il Mio Servizio di Autenticazione` |
| `LOGIN_PAGE_FOOTER_TEXT` | Testo del piè di pagina della pagina di login | `Copyright © 2024 - Stargate` | `© 2024 La Mia Azienda` |
| `USER_HEADER_NAME` | Nome dell'intestazione utente impostato dopo autenticazione riuscita | `X-Forwarded-User` | `X-Authenticated-User` |
| `COOKIE_DOMAIN` | Dominio del cookie (per condivisione sessioni cross-domain) | Vuoto (non impostato) | `.example.com` |
| `PORT` | Porta di ascolto del servizio (solo sviluppo locale) | `80` | `8080` |

### Formato Configurazione Password

La configurazione della password utilizza il seguente formato:
```
algorithm:password1|password2|password3
```

Algoritmi supportati:
- `plaintext`: Password in testo normale (solo test)
- `bcrypt`: Hash BCrypt
- `md5`: Hash MD5
- `sha512`: Hash SHA512

Esempi:
```bash
# Password in testo normale (multiple)
PASSWORDS=plaintext:test123|admin456|user789

# Hash BCrypt
PASSWORDS=bcrypt:$2a$10$N9qo8uLOickgx2ZMRZoMyeIjZAgcfl7p92ldGxad68LJZdL17lhWy

# Hash MD5
PASSWORDS=md5:5e884898da28047151d0e56f8dc6292773603d0d6aabbdd62a11ef721d1542d8
```

**Per la configurazione dettagliata, vedere: [docs/itIT/CONFIG.md](docs/itIT/CONFIG.md)**

## 📚 Documentazione

È disponibile una documentazione completa per aiutarti a sfruttare al meglio Stargate:

- 📐 **[Documento Architettura](docs/itIT/ARCHITECTURE.md)** - Approfondimento sull'architettura tecnica e decisioni di progettazione
- 🔌 **[Documento API](docs/itIT/API.md)** - Riferimento completo degli endpoint API con esempi
- ⚙️ **[Riferimento Configurazione](docs/itIT/CONFIG.md)** - Opzioni di configurazione dettagliate e best practice
- 🚀 **[Guida al Deployment](docs/itIT/DEPLOYMENT.md)** - Strategie di deployment in produzione e raccomandazioni

## 📚 Documentazione API

### Endpoint Verifica Autenticazione

#### `GET /_auth`

L'endpoint principale di verifica autenticazione per Traefik Forward Auth.

**Intestazioni Richiesta:**
- `Stargate-Password` (opzionale): Autenticazione password per richieste API
- `Cookie: stargate_session_id` (opzionale): Autenticazione sessione per richieste Web

**Risposta:**
- `200 OK`: Autenticazione riuscita, imposta l'intestazione `X-Forwarded-User` (o il nome intestazione utente configurato)
- `401 Unauthorized`: Autenticazione fallita
- `500 Internal Server Error`: Errore server

**Note:**
- Le richieste HTML reindirizzano alla pagina di login in caso di fallimento autenticazione
- Le richieste API (JSON/XML) restituiscono errore 401 in caso di fallimento autenticazione

### Endpoint Login

#### `GET /_login`

Visualizza la pagina di login.

**Parametri Query:**
- `callback` (opzionale): URL callback dopo login riuscito

**Risposta:**
- Restituisce HTML pagina di login

#### `POST /_login`

Gestisce le richieste di login.

**Dati Form:**
- `password`: Password utente
- `callback` (opzionale): URL callback dopo login riuscito

**Priorità Recupero Callback:**
1. Dal Cookie (se precedentemente impostato)
2. Dai dati del form
3. Dai parametri query
4. Se nessuno dei precedenti, e il dominio di origine differisce dal dominio del servizio di autenticazione, utilizzare il dominio di origine come callback

**Risposta:**
- `200 OK`: Login riuscito
  - Se esiste callback, reindirizza a `{callback}/_session_exchange?id={session_id}`
  - Se nessun callback, restituisce messaggio di successo (formato HTML o JSON, a seconda del tipo di richiesta)
- `401 Unauthorized`: Password errata
- `500 Internal Server Error`: Errore server

### Endpoint Logout

#### `GET /_logout`

Disconnette l'utente corrente e distrugge la sessione.

**Risposta:**
- `200 OK`: Logout riuscito, restituisce "Logged out"

### Endpoint Scambio Sessione

#### `GET /_session_exchange`

Utilizzato per condivisione sessioni cross-domain. Imposta il cookie ID sessione specificato e reindirizza.

**Parametri Query:**
- `id` (richiesto): ID sessione da impostare

**Risposta:**
- `302 Redirect`: Reindirizza al percorso root
- `400 Bad Request`: ID sessione mancante

### Endpoint Verifica Salute

#### `GET /health`

Endpoint verifica salute del servizio.

**Risposta:**
- `200 OK`: Il servizio è sano

### Endpoint Root

#### `GET /`

Percorso root, visualizza informazioni del servizio.

**Per la documentazione API dettagliata, vedere: [docs/itIT/API.md](docs/itIT/API.md)**

## 🐳 Guida al Deployment

### Deployment Docker

#### Costruisci Immagine

```bash
cd codes
docker build -t stargate:latest .
```

#### Esegui Container

```bash
docker run -d \
  --name stargate \
  -p 80:80 \
  -e AUTH_HOST=auth.example.com \
  -e PASSWORDS=plaintext:yourpassword \
  stargate:latest
```

### Deployment Docker Compose

Il progetto fornisce un esempio di configurazione `docker-compose.yml`, inclusi il servizio Stargate e un servizio esempio whoami:

```bash
cd codes
docker-compose up -d
```

### Integrazione Traefik

Configura le etichette Traefik in `docker-compose.yml`:

```yaml
services:
  stargate:
    image: stargate:latest
    environment:
      - AUTH_HOST=auth.example.com
      - PASSWORDS=plaintext:yourpassword
    networks:
      - traefik
    labels:
      - "traefik.enable=true"
      - "traefik.docker.network=traefik"
      - "traefik.http.routers.auth.entrypoints=http"
      - "traefik.http.routers.auth.rule=Host(`auth.example.com`) || Path(`/_session_exchange`)"
      - "traefik.http.middlewares.stargate.forwardauth.address=http://stargate/_auth"

  your-service:
    image: your-service:latest
    networks:
      - traefik
    labels:
      - "traefik.enable=true"
      - "traefik.docker.network=traefik"
      - "traefik.http.routers.your-service.entrypoints=http"
      - "traefik.http.routers.your-service.rule=Host(`your-service.example.com`)"
      - "traefik.http.routers.your-service.middlewares=stargate"  # Usa middleware Stargate

networks:
  traefik:
    external: true
```

### Raccomandazioni Produzione

1. **Usa HTTPS**: In produzione, assicurati che HTTPS sia configurato via Traefik
2. **Usa Algoritmi Password Forti**: Evita `plaintext`, raccomanda l'uso di `bcrypt` o `sha512`
3. **Imposta Dominio Cookie**: Se devi condividere sessioni tra più sottodomini, imposta `COOKIE_DOMAIN`
4. **Gestione Log**: Configura rotazione log e monitoraggio appropriati
5. **Limiti Risorse**: Imposta limiti CPU e memoria appropriati per i container

**Per la guida al deployment dettagliata, vedere: [docs/itIT/DEPLOYMENT.md](docs/itIT/DEPLOYMENT.md)**

## 💻 Guida allo Sviluppo

### Struttura Progetto

```
codes/
├── src/
│   ├── cmd/
│   │   └── stargate/          # Punto di ingresso principale del programma
│   │       ├── main.go        # Punto di ingresso del programma
│   │       ├── server.go      # Configurazione server
│   │       └── constants.go  # Definizioni costanti
│   ├── internal/
│   │   ├── auth/              # Logica autenticazione
│   │   ├── config/            # Gestione configurazione
│   │   ├── handlers/          # Gestori HTTP
│   │   ├── i18n/              # Internazionalizzazione
│   │   ├── middleware/        # Middleware
│   │   ├── secure/            # Algoritmi crittografia password
│   │   └── web/               # Template Web e risorse statiche
│   ├── go.mod
│   └── go.sum
├── Dockerfile
├── docker-compose.yml
└── start-local.sh
```

### Sviluppo Locale

1. Installa dipendenze:
```bash
cd codes
go mod download
```

2. Esegui test:
```bash
go test ./...
```

3. Avvia server di sviluppo:
```bash
./start-local.sh
```

### Aggiunta Nuovi Algoritmi Password

1. Crea una nuova implementazione algoritmo nella directory `src/internal/secure/`:
```go
package secure

type NewAlgorithmResolver struct{}

func (r *NewAlgorithmResolver) Check(h string, password string) bool {
    // Implementa logica verifica password
    return false
}
```

2. Registra l'algoritmo in `src/internal/config/validation.go`:
```go
SupportedAlgorithms = map[string]secure.HashResolver{
    // ...
    "newalgorithm": &secure.NewAlgorithmResolver{},
}
```

### Aggiunta Supporto Nuova Lingua

1. Aggiungi costante lingua in `src/internal/i18n/i18n.go`:
```go
const (
    LangEN Language = "en"
    LangZH Language = "zh"
    LangIT Language = "it"  // Nuovo
)
```

2. Aggiungi mapping traduzione:
```go
var translations = map[Language]map[string]string{
    // ...
    LangIT: {
        "error.auth_required": "Autenticazione richiesta",
        // ...
    },
}
```

3. Aggiungi opzione lingua in `src/internal/config/config.go`:
```go
Language = EnvVariable{
    PossibleValues: []string{"en", "zh", "it"},  // Aggiungi nuova lingua
}
```

## 📝 Licenza

Questo progetto è concesso in licenza sotto Apache License 2.0. Vedi il file [LICENSE](codes/LICENSE) per i dettagli.

## 🤝 Contribuire

Accogliamo i contributi! Che siano:
- 🐛 Segnalazioni di bug
- 💡 Suggerimenti di funzionalità
- 📝 Miglioramenti alla documentazione
- 🔧 Contributi di codice

Sentiti libero di aprire un Issue o inviare una Pull Request. Ogni contributo rende Stargate migliore!

---

## ⚠️ Checklist Produzione

Prima di distribuire in produzione, assicurati di aver completato queste best practice di sicurezza:

- ✅ **Usa Password Forti**: Evita `plaintext`, usa `bcrypt` o `sha512` per l'hashing delle password
- ✅ **Abilita HTTPS**: Configura HTTPS via Traefik o il tuo proxy inverso
- ✅ **Imposta Dominio Cookie**: Configura `COOKIE_DOMAIN` per una gestione sessione appropriata tra sottodomini
- ✅ **Monitora e Registra**: Configura registrazione e monitoraggio appropriati per il tuo deployment
- ✅ **Aggiornamenti Regolari**: Mantieni Stargate aggiornato all'ultima versione per patch di sicurezza
