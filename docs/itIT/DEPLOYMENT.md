# Guida al Deployment

Questo documento fornisce una guida dettagliata al deployment per il servizio Stargate Forward Auth.

## Indice

- [Metodi di Deployment](#metodi-di-deployment)
- [Deployment Docker](#deployment-docker)
- [Deployment Docker Compose](#deployment-docker-compose)
- [Integrazione Traefik](#integrazione-traefik)
- [Deployment in Produzione](#deployment-in-produzione)
- [Monitoraggio e Manutenzione](#monitoraggio-e-manutenzione)
- [Risoluzione dei Problemi](#risoluzione-dei-problemi)

## Metodi di Deployment

Stargate supporta i seguenti metodi di deployment:

1. **Container Docker** (Consigliato) - Il più semplice e comune
2. **Docker Compose** - Adatto per sviluppo locale e test
3. **Kubernetes** - Adatto per ambienti di produzione su larga scala
4. **Esecuzione Binario Diretta** - Adatto per scenari speciali

Questo documento presenta principalmente i metodi di deployment Docker e Docker Compose.

## Dipendenze del Servizio

Stargate può integrarsi con i seguenti servizi opzionali:

### Servizio Warden

**Funzione:** Gestione whitelist utenti e fornitura informazioni utente

**Requisiti di Deployment:**
- Richiede database (PostgreSQL/MySQL/SQLite)
- Fornisce interfaccia API HTTP
- Supporta autenticazione API Key

**Configurazione:**
```bash
WARDEN_ENABLED=true
WARDEN_URL=http://warden:8080
WARDEN_API_KEY=your-api-key
```

### Servizio Herald

**Funzione:** Invio e verifica OTP/codice di verifica

**Requisiti di Deployment:**
- Richiede Redis (memorizza challenge e stato limite velocità)
- Fornisce interfaccia API HTTP
- Supporta autenticazione firma HMAC o mTLS (raccomandato per produzione)

**Configurazione:**
```bash
HERALD_ENABLED=true
HERALD_URL=http://herald:8080
HERALD_HMAC_SECRET=your-hmac-secret  # Raccomandato per produzione
```

### Sicurezza Comunicazione Inter-Servizio

**Requisiti per Ambiente di Produzione:**

1. **Autenticazione Firma HMAC** (Raccomandato):
   - Stargate ↔ Herald utilizza firma HMAC-SHA256
   - Configurare `HERALD_HMAC_SECRET`
   - Include verifica timestamp (previene attacchi replay)

2. **Autenticazione mTLS** (Opzionale, più sicuro):
   - Configurare certificato client TLS
   - Impostare `HERALD_TLS_CLIENT_CERT_FILE` e `HERALD_TLS_CLIENT_KEY_FILE`
   - Configurare verifica certificato CA

3. **Isolamento Rete:**
   - La comunicazione inter-servizio dovrebbe essere su rete interna
   - Utilizzare regole firewall per limitare l'accesso
   - Evitare di esporre servizi alla rete pubblica

## Deployment Docker

### Costruire l'Immagine

#### Costruire dalla Sorgente

```bash
cd codes
docker build -t stargate:latest .
```

#### Parametri di Build

- **Immagine Base**: `golang:1.25-alpine` (stage di build)
- **Immagine di Esecuzione**: `scratch` (immagine minima)
- **Directory di Lavoro**: `/app`
- **Porta Esposta**: `80`

### Eseguire il Container

#### Esecuzione Base

```bash
docker run -d \
  --name stargate \
  -p 80:80 \
  -e AUTH_HOST=auth.example.com \
  -e PASSWORDS=plaintext:yourpassword \
  stargate:latest
```

#### Esecuzione con Configurazione Completa

```bash
docker run -d \
  --name stargate \
  -p 80:80 \
  -e AUTH_HOST=auth.example.com \
  -e PASSWORDS=bcrypt:$2a$10$N9qo8uLOickgx2ZMRZoMyeIjZAgcfl7p92ldGxad68LJZdL17lhWy \
  -e DEBUG=false \
  -e LANGUAGE=it \
  -e LOGIN_PAGE_TITLE=Il Mio Servizio di Autenticazione \
  -e LOGIN_PAGE_FOOTER_TEXT=© 2024 La Mia Azienda \
  -e COOKIE_DOMAIN=.example.com \
  --restart unless-stopped \
  stargate:latest
```

#### Descrizione Parametri

- `-d`: Eseguire in background
- `--name stargate`: Nome del container
- `-p 80:80`: Mapping porta (porta host:porta container)
- `-e`: Variabile d'ambiente
- `--restart unless-stopped`: Politica di riavvio automatico

### Visualizzare i Log

```bash
# Visualizzare i log in tempo reale
docker logs -f stargate

# Visualizzare le ultime 100 righe dei log
docker logs --tail 100 stargate
```

### Arrestare e Rimuovere

```bash
# Arrestare il container
docker stop stargate

# Rimuovere il container
docker rm stargate

# Arrestare e rimuovere
docker rm -f stargate
```

## Deployment Docker Compose

### Configurazione Base

Il progetto fornisce un file di esempio `docker-compose.yml`:

```yaml
services:
  stargate:
    image: stargate
    environment:
      - AUTH_HOST=auth.test.localhost
      - PASSWORDS=plaintext:test1234|test1337
    networks:
      - traefik
    labels:
      - traefik.enable=true
      - traefik.docker.network=proxy
      - traefik.http.routers.auth.entrypoints=http
      - traefik.http.routers.auth.rule=Host(`auth.test.localhost`) || Path(`/_session_exchange`)
      - traefik.http.middlewares.stargate.forwardauth.address=http://stargate/_auth

  whoami:
    image: traefik/whoami
    networks:
      - traefik
    labels:
      - traefik.enable=true
      - traefik.docker.network=proxy
      - traefik.http.routers.whoami.entrypoints=http
      - traefik.http.routers.whoami.rule=Host(`whoami.test.localhost`)
      - "traefik.http.routers.whoami.middlewares=stargate"

networks:
  traefik:
    external: true
```

### Avviare i Servizi

```bash
cd codes
docker-compose up -d
```

### Arrestare i Servizi

```bash
docker-compose down
```

### Visualizzare i Log

```bash
# Visualizzare tutti i log dei servizi
docker-compose logs -f

# Visualizzare i log di un servizio specifico
docker-compose logs -f stargate
```

### Configurazione Personalizzata

Modificare `docker-compose.yml` e modificare le variabili d'ambiente:

```yaml
services:
  stargate:
    image: stargate
    environment:
      - AUTH_HOST=auth.example.com
      - PASSWORDS=bcrypt:$2a$10$...
      - DEBUG=false
      - LANGUAGE=it
      - COOKIE_DOMAIN=.example.com
```

## Integrazione Traefik

### Configurazione Base

Stargate è progettato per integrarsi con Traefik, fornendo autenticazione tramite middleware Forward Auth.

#### 1. Configurare il Servizio Stargate

Configurare Stargate in `docker-compose.yml`:

```yaml
services:
  stargate:
    image: stargate:latest
    environment:
      - AUTH_HOST=auth.example.com
      - PASSWORDS=bcrypt:$2a$10$...
    networks:
      - traefik
    labels:
      - "traefik.enable=true"
      - "traefik.docker.network=traefik"
      - "traefik.http.routers.auth.entrypoints=http,https"
      - "traefik.http.routers.auth.rule=Host(`auth.example.com`) || Path(`/_session_exchange`)"
      - "traefik.http.middlewares.stargate.forwardauth.address=http://stargate/_auth"
      - "traefik.http.middlewares.stargate.forwardauth.authResponseHeaders=X-Forwarded-User"
```

#### 2. Configurare i Servizi Protetti

Applicare il middleware Stargate ai servizi che richiedono autenticazione:

```yaml
services:
  your-app:
    image: your-app:latest
    networks:
      - traefik
    labels:
      - "traefik.enable=true"
      - "traefik.docker.network=traefik"
      - "traefik.http.routers.your-app.entrypoints=http,https"
      - "traefik.http.routers.your-app.rule=Host(`app.example.com`)"
      - "traefik.http.routers.your-app.middlewares=stargate"  # Applicare middleware autenticazione
```

### Configurazione HTTPS

#### Utilizzo Let's Encrypt

```yaml
services:
  stargate:
    labels:
      - "traefik.http.routers.auth.entrypoints=https"
      - "traefik.http.routers.auth.tls.certresolver=letsencrypt"
      - "traefik.http.routers.auth.tls=true"
```

#### Utilizzo Certificati Personalizzati

```yaml
services:
  stargate:
    labels:
      - "traefik.http.routers.auth.entrypoints=https"
      - "traefik.http.routers.auth.tls=true"
      - "traefik.http.routers.auth.tls.certfile=/path/to/cert.pem"
      - "traefik.http.routers.auth.tls.keyfile=/path/to/key.pem"
```

### Condivisione Sessione Cross-Domain

Se è necessario condividere sessioni tra sottodomini:

1. Impostare la variabile d'ambiente `COOKIE_DOMAIN`:

```yaml
services:
  stargate:
    environment:
      - COOKIE_DOMAIN=.example.com
```

2. Assicurarsi che tutti i domini associati siano instradati verso Stargate tramite Traefik

3. Flusso di login:
   - L'utente si connette a `auth.example.com`
   - Reindirizza a `app.example.com/_session_exchange?id=<session_id>`
   - Il cookie di sessione è impostato al dominio `.example.com`
   - Tutti i sottodomini `*.example.com` possono utilizzare questa sessione

## Deployment in Produzione

### Raccomandazioni di Sicurezza

#### 1. Utilizzare Algoritmi Password Forti

**Non Consigliato:**

```bash
PASSWORDS=plaintext:yourpassword
```

**Consigliato:**

```bash
PASSWORDS=bcrypt:$2a$10$N9qo8uLOickgx2ZMRZoMyeIjZAgcfl7p92ldGxad68LJZdL17lhWy
```

#### 2. Abilitare HTTPS

- Configurare HTTPS tramite Traefik
- Utilizzare certificati automatici Let's Encrypt
- Forzare reindirizzamento HTTPS

#### 3. Disabilitare Modalità Debug

```bash
DEBUG=false
```

#### 4. Impostare Limiti Risorse

```yaml
services:
  stargate:
    deploy:
      resources:
        limits:
          cpus: '0.5'
          memory: 128M
        reservations:
          cpus: '0.25'
          memory: 64M
```

#### 5. Utilizzare Verifiche Salute

```yaml
services:
  stargate:
    healthcheck:
      test: ["CMD", "wget", "--quiet", "--tries=1", "--spider", "http://localhost/health"]
      interval: 30s
      timeout: 10s
      retries: 3
      start_period: 40s
```

### Deployment Alta Disponibilità

#### 1. Deployment Multi-Istanza

```yaml
services:
  stargate:
    deploy:
      replicas: 3
```

**Nota:** Stargate utilizza storage sessione in memoria, le sessioni non sono condivise tra le istanze. Se è necessario un deployment multi-istanza, si raccomanda di:

- Utilizzare persistenza sessione del bilanciatore di carico (Sticky Session)
- O attendere supporto storage sessione esterno (Redis)

#### 2. Bilanciamento Carico

Aggiungere un bilanciatore di carico prima di Traefik:

```yaml
services:
  traefik:
    labels:
      - "traefik.http.services.stargate.loadbalancer.server.port=80"
```

### Configurazione Monitoraggio

#### 1. Raccolta Log

```yaml
services:
  stargate:
    logging:
      driver: "json-file"
      options:
        max-size: "10m"
        max-file: "3"
```

#### 2. Endpoint Verifica Salute

Utilizzare l'endpoint `/health` per il monitoraggio:

```bash
# Script verifica salute
#!/bin/bash
if curl -f http://localhost/health > /dev/null 2>&1; then
  exit 0
else
  exit 1
fi
```

#### 3. Integrazione Prometheus

(Da implementare) Le versioni future supporteranno esportazione metriche Prometheus.

## Monitoraggio e Manutenzione

### Gestione Log

#### Visualizzare Log

```bash
# Docker
docker logs -f stargate

# Docker Compose
docker-compose logs -f stargate
```

#### Livelli di Registrazione

- `DEBUG=true`: Informazioni di debug dettagliate
- `DEBUG=false`: Solo informazioni critiche

#### Rotazione Log

Configurare il driver di log Docker:

```yaml
services:
  stargate:
    logging:
      driver: "json-file"
      options:
        max-size: "10m"
        max-file: "3"
```

### Monitoraggio Prestazioni

#### Utilizzo Risorse

```bash
# Visualizzare utilizzo risorse del container
docker stats stargate
```

#### Tempo di Risposta

Monitorare il tempo di risposta utilizzando l'endpoint verifica salute:

```bash
time curl http://auth.example.com/health
```

### Manutenzione Regolare

1. **Aggiornare Immagini**: Scaricare regolarmente le ultime immagini
2. **Verificare Log**: Verificare regolarmente i log di errore
3. **Monitorare Risorse**: Monitorare utilizzo CPU e memoria
4. **Backup Configurazione**: Eseguire backup della configurazione delle variabili d'ambiente

## Risoluzione dei Problemi

### Problemi Comuni

#### 1. Il Servizio Non Si Avvia

**Problema:** Il container si chiude immediatamente dopo l'avvio

**Passi di Risoluzione:**

```bash
# Visualizzare log del container
docker logs stargate

# Verificare configurazione
docker inspect stargate | grep -A 20 Env
```

**Cause Comuni:**

- Configurazione richiesta mancante (`AUTH_HOST`, `PASSWORDS`)
- Formato configurazione password errato
- Porta occupata

#### 2. L'Autenticazione Fallisce

**Problema:** Gli utenti non possono connettersi

**Passi di Risoluzione:**

1. Verificare se la configurazione password è corretta
2. Verificare se l'algoritmo password corrisponde
3. Visualizzare log del servizio: `docker logs stargate`

**Cause Comuni:**

- Configurazione password errata
- Incompatibilità algoritmo password (es., bcrypt configurato ma password in testo normale utilizzata)
- Configurazione dominio cookie errata

#### 3. Sessioni Cross-Domain Non Funzionano

**Problema:** Impossibile condividere sessioni tra sottodomini

**Passi di Risoluzione:**

1. Verificare configurazione `COOKIE_DOMAIN`
2. Confermare che il formato dominio cookie è corretto (`.example.com`)
3. Verificare impostazioni cookie del browser

**Soluzione:**

```bash
# Assicurarsi che COOKIE_DOMAIN sia impostato
COOKIE_DOMAIN=.example.com
```

#### 4. Problemi Integrazione Traefik

**Problema:** Traefik non può inoltrare correttamente le richieste di autenticazione

**Passi di Risoluzione:**

1. Verificare configurazione label Traefik
2. Confermare che la configurazione di rete è corretta
3. Verificare indirizzo middleware Forward Auth

**Soluzione:**

```yaml
# Assicurarsi che l'indirizzo middleware sia corretto
- "traefik.http.middlewares.stargate.forwardauth.address=http://stargate/_auth"
```

### Suggerimenti Debug

#### 1. Abilitare Modalità Debug

```bash
DEBUG=true
```

#### 2. Verificare Connessione di Rete

```bash
# Testare dall'interno del container
docker exec stargate wget -O- http://localhost/health
```

#### 3. Visualizzare Log Traefik

```bash
docker logs traefik
```

#### 4. Testare Endpoint API

```bash
# Testare verifica salute
curl http://auth.example.com/health

# Testare autenticazione (utilizzando Header)
curl -H "Stargate-Password: yourpassword" http://auth.example.com/_auth

# Testare autenticazione (utilizzando Cookie)
curl -H "Cookie: stargate_session_id=<session_id>" http://auth.example.com/_auth
```

### Ottenere Aiuto

Se si incontrano problemi:

1. Visualizzare log: `docker logs stargate`
2. Verificare configurazione: Confermare che tutte le variabili d'ambiente sono corrette
3. Consultare documentazione: [Documentazione API](API.md), [Riferimento Configurazione](CONFIG.md)
4. Inviare Issue: Inviare un rapporto problema nel repository del progetto

## Guida Aggiornamento

### Passi di Aggiornamento

1. **Backup Configurazione**: Eseguire backup della configurazione attuale delle variabili d'ambiente

2. **Arrestare il Servizio:**

```bash
docker stop stargate
```

3. **Scaricare Nuova Immagine:**

```bash
docker pull stargate:latest
```

4. **Avviare Nuovo Container:**

```bash
docker run -d \
  --name stargate \
  ...(utilizzare configurazione salvata)
  stargate:latest
```

5. **Verificare il Servizio:**

```bash
curl http://auth.example.com/health
```

### Rollback

Se si verificano problemi dopo l'aggiornamento:

```bash
# Arrestare nuovo container
docker stop stargate

# Avviare con vecchia immagine
docker run -d \
  --name stargate \
  ...(utilizzare configurazione salvata)
  stargate:<old-version>
```
