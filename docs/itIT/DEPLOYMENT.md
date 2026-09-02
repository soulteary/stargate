# Guida al Deployment

Questo documento fornisce una guida dettagliata al deployment per il servizio Stargate Forward Auth.

Per aggiornare da v0.12.0, leggere prima la [guida alla migrazione v1.0.0](MIGRATION_V1.md).

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

Dalla root del progetto:

```bash
docker build -f docker/Dockerfile -t stargate:latest .
```

#### Parametri di Build

- **Immagine Base**: `golang:1.27.0-alpine3.24` (stage di build)
- **Immagine di Esecuzione**: `alpine:3.24` (certificati CA e BusyBox `wget` per HTTPS e health check)
- **Directory di Lavoro**: `/app`
- **Porta Esposta**: `8080`

### Eseguire il Container

#### Esecuzione Base

```bash
docker run -d \
  --name stargate \
  -p 8080:8080 \
  -e AUTH_HOST=auth.example.com \
  -e 'PASSWORDS=plaintext:yourpassword' \
  stargate:latest
```

#### Esecuzione con Configurazione Completa

```bash
docker run -d \
  --name stargate \
  -p 8080:8080 \
  -e AUTH_HOST=auth.example.com \
  -e 'PASSWORDS=bcrypt:$2a$10$N9qo8uLOickgx2ZMRZoMyeIjZAgcfl7p92ldGxad68LJZdL17lhWy' \
  -e DEBUG=false \
  -e LANGUAGE=it \
  -e 'LOGIN_PAGE_TITLE=Il Mio Servizio di Autenticazione' \
  -e 'LOGIN_PAGE_FOOTER_TEXT=© 2024 La Mia Azienda' \
  -e COOKIE_DOMAIN=.example.com \
  --restart unless-stopped \
  stargate:latest
```

#### Descrizione Parametri

- `-d`: Eseguire in background
- `--name stargate`: Nome del container
- `-p 8080:8080`: Mapping porta (porta host:porta container)
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

```

## Deployment Docker Compose

### Configurazione Base

Il file `docker-compose.yml` nella radice del repository è lo stack incluso: avvia Traefik, Stargate e il servizio demo protetto sulla rete `stargate-traefik` gestita da Compose (`172.30.0.0/24`). Non richiede una rete esterna preesistente.

```bash
docker compose config
docker compose up -d
```

### Usare una rete Traefik esterna esistente

Usare l'esempio separato seguente solo quando Traefik è già in esecuzione su una rete Docker esterna condivisa. Il nome reale della rete e il CIDR ottenuto con inspect controllano la rete Compose, ogni label `traefik.docker.network` e `TRUSTED_PROXIES`.

```yaml
services:
  stargate:
    image: ghcr.io/soulteary/stargate:v1.0.0
    environment:
      - AUTH_HOST=auth.test.localhost
      - PASSWORDS=plaintext:test1234|test1337
      - CALLBACK_ALLOWED_HOSTS=whoami.test.localhost
      - SESSION_EXCHANGE_SECRET=local-development-session-secret-change-me
      - TRUSTED_PROXIES=${TRAEFIK_NETWORK_CIDR:?set TRAEFIK_NETWORK_CIDR from docker network inspect}
      - COOKIE_SECURE=false # Local HTTP only; omit for HTTPS.
    networks:
      - traefik
    labels:
      - traefik.enable=true
      - traefik.docker.network=${TRAEFIK_NETWORK_NAME:-traefik}
      - traefik.http.routers.auth.entrypoints=http
      - traefik.http.routers.auth.rule=Host(`auth.test.localhost`) || Path(`/_session_exchange`)
      - traefik.http.middlewares.stargate.forwardauth.address=http://stargate:8080/_auth
      - "traefik.http.middlewares.stargate.forwardauth.authResponseHeaders=X-Forwarded-User,X-Auth-User,X-Auth-Email,X-Auth-Name,X-Auth-Scopes,X-Auth-Role,X-Auth-AMR"

  whoami:
    image: traefik/whoami
    networks:
      - traefik
    labels:
      - traefik.enable=true
      - traefik.docker.network=${TRAEFIK_NETWORK_NAME:-traefik}
      - traefik.http.routers.whoami.entrypoints=http
      - traefik.http.routers.whoami.rule=Host(`whoami.test.localhost`)
      - "traefik.http.routers.whoami.middlewares=stargate"

networks:
  traefik:
    name: ${TRAEFIK_NETWORK_NAME:-traefik}
    external: true
```

### Controllare o creare la rete esterna

Se la rete esiste, i comandi la ispezionano e riutilizzano tutti i CIDR assegnati; non eliminare o ricreare una rete di produzione esistente. Se non esiste, la creano con la subnet documentata.

```bash
export TRAEFIK_NETWORK_NAME="${TRAEFIK_NETWORK_NAME:-traefik}"

if docker network inspect "$TRAEFIK_NETWORK_NAME" >/dev/null 2>&1; then
  export TRAEFIK_NETWORK_CIDR="$(
    docker network inspect "$TRAEFIK_NETWORK_NAME" \
      --format '{{range .IPAM.Config}}{{println .Subnet}}{{end}}' |
      awk 'NF { cidr = cidr separator $0; separator = "," } END { print cidr }'
  )"
  if [ -z "$TRAEFIK_NETWORK_CIDR" ]; then
    echo "No subnet found for Docker network $TRAEFIK_NETWORK_NAME" >&2
    exit 1
  fi
else
  docker network create \
    --driver bridge \
    --subnet 172.30.0.0/24 \
    "$TRAEFIK_NETWORK_NAME"
  export TRAEFIK_NETWORK_CIDR=172.30.0.0/24
fi

docker compose config
docker compose up -d
```

### Arrestare i Servizi

```bash
docker compose down
```

### Visualizzare i Log

```bash
# Visualizzare tutti i log dei servizi
docker compose logs -f

# Visualizzare i log di un servizio specifico
docker compose logs -f stargate
```

### Configurazione Personalizzata

Modificare `docker-compose.yml` e modificare le variabili d'ambiente:

```yaml
services:
  stargate:
    image: ghcr.io/soulteary/stargate:v1.0.0
    environment:
      - AUTH_HOST=auth.example.com
      - PASSWORDS=bcrypt:$$2a$$10$$...
      - TRUSTED_PROXIES=${TRAEFIK_NETWORK_CIDR:?set TRAEFIK_NETWORK_CIDR from docker network inspect}
      - DEBUG=false
      - LANGUAGE=it
      - COOKIE_DOMAIN=.example.com
      - SESSION_EXCHANGE_SECRET=replace-with-at-least-32-random-characters
```

## Integrazione Traefik

### Configurazione Base

Stargate è progettato per integrarsi con Traefik, fornendo autenticazione tramite middleware Forward Auth.

#### 1. Configurare il Servizio Stargate

Configurare Stargate in `docker-compose.yml`:

```yaml
services:
  stargate:
    image: ghcr.io/soulteary/stargate:v1.0.0
    environment:
      - AUTH_HOST=auth.example.com
      - PASSWORDS=bcrypt:$$2a$$10$$...
      - TRUSTED_PROXIES=${TRAEFIK_NETWORK_CIDR:?set TRAEFIK_NETWORK_CIDR from docker network inspect}
    networks:
      - traefik
    labels:
      - "traefik.enable=true"
      - "traefik.docker.network=${TRAEFIK_NETWORK_NAME:-traefik}"
      - "traefik.http.routers.auth.entrypoints=http,https"
      - "traefik.http.routers.auth.rule=Host(`auth.example.com`) || Path(`/_session_exchange`)"
      - "traefik.http.middlewares.stargate.forwardauth.address=http://stargate:8080/_auth"
      - "traefik.http.middlewares.stargate.forwardauth.authResponseHeaders=X-Forwarded-User,X-Auth-User,X-Auth-Email,X-Auth-Name,X-Auth-Scopes,X-Auth-Role,X-Auth-AMR"
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
      - "traefik.docker.network=${TRAEFIK_NETWORK_NAME:-traefik}"
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
      - SESSION_EXCHANGE_SECRET=replace-with-at-least-32-random-characters
```

2. Assicurarsi che tutti i domini associati siano instradati verso Stargate tramite Traefik

3. Flusso di login:
   - L'utente si connette a `auth.example.com`
   - Reindirizza a `app.example.com/_session_exchange?ticket=<opaque_ticket>`
   - Il cookie di sessione è impostato al dominio `.example.com`
   - Tutti i sottodomini `*.example.com` possono utilizzare questa sessione

## Deployment in Produzione

### Raccomandazioni di Sicurezza

#### 1. Utilizzare Algoritmi Password Forti

**Non Consigliato:**

```bash
PASSWORDS='plaintext:yourpassword'
```

**Consigliato:**

```bash
PASSWORDS='bcrypt:$2a$10$N9qo8uLOickgx2ZMRZoMyeIjZAgcfl7p92ldGxad68LJZdL17lhWy'
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
      test: ["CMD", "wget", "-q", "-O", "-", "http://127.0.0.1:8080/healthz"]
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

**Ambito dello storage delle sessioni:**

- **Un processo / una replica, inclusi callback cross-domain:** lo storage in memoria (`SESSION_STORAGE_ENABLED=false`) è supportato quando emissione e riscatto del ticket e il successivo uso della sessione raggiungono lo stesso processo Stargate attivo. Sessioni e hash dei ticket consumati esistono solo in quel processo. Un riavvio perde entrambi gli stati, invalida le sessioni attive e non conserva lo stato monouso dei ticket.
- **Più processi o repliche:** Redis è obbligatorio quando una sessione o un ticket può essere gestito da processi diversi. Usare `SESSION_STORAGE_ENABLED=true`, lo stesso namespace `SESSION_STORAGE_REDIS_*` e lo stesso `SESSION_EXCHANGE_SECRET` su tutte le repliche. Le Sticky Session sono solo un'ottimizzazione del routing e non sostituiscono lo stato condiviso o la protezione anti-replay tra processi.
- **Aggiornamento progressivo:** Redis è obbligatorio per la sostituzione senza interruzioni perché vecchi e nuovi processi si sovrappongono. Senza Redis, arrestare e poi avviare il servizio, accettando invalidazione delle sessioni e nuovo login. Non mescolare v0.12.0 e v1.0.0 nello stesso pool.
- **Stato oltre il riavvio:** Redis è obbligatorio se sessioni o stato anti-replay dei ticket consumati devono sopravvivere alla sostituzione o al riavvio del processo.

#### 2. Bilanciamento Carico

Aggiungere un bilanciatore di carico prima di Traefik:

```yaml
services:
  traefik:
    labels:
      - "traefik.http.services.stargate.loadbalancer.server.port=8080"
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

Usare `/healthz` per la liveness del processo e `/readyz` per la readiness delle dipendenze. `/health` è solo un alias di readiness deprecato.

```bash
# Script verifica salute
#!/bin/bash
if curl -f http://localhost:8080/healthz > /dev/null 2>&1; then
  exit 0
else
  exit 1
fi
```

#### 3. Integrazione Prometheus

Stargate espone le metriche Prometheus su `GET /metrics`. Configurare Prometheus per lo scrape di questo endpoint. L'endpoint metriche è escluso per default da log e tracing delle richieste.

## Monitoraggio e Manutenzione

### Gestione Log

#### Visualizzare Log

```bash
# Docker
docker logs -f stargate

# Docker Compose
docker compose logs -f stargate
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
time curl http://auth.example.com/healthz
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
- "traefik.http.middlewares.stargate.forwardauth.address=http://stargate:8080/_auth"
- "traefik.http.middlewares.stargate.forwardauth.authResponseHeaders=X-Forwarded-User,X-Auth-User,X-Auth-Email,X-Auth-Name,X-Auth-Scopes,X-Auth-Role,X-Auth-AMR"
```

### Suggerimenti Debug

#### 1. Abilitare Modalità Debug

```bash
DEBUG=true
```

#### 2. Verificare Connessione di Rete

```bash
# Testare dall'interno del container
docker exec stargate wget -q -O - http://127.0.0.1:8080/healthz
```

#### 3. Visualizzare Log Traefik

```bash
docker logs traefik
```

#### 4. Testare Endpoint API

```bash
# Testare verifica salute
curl http://auth.example.com/healthz

# Testare autenticazione (utilizzando Header)
# Prerequisito del server: PASSWORD_HEADER_AUTH_ENABLED=true
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

1. **Creare file di ambiente separati per rollback e v1**: lasciare invariato `stargate.env`. I comandi seguenti rifiutano di sovrascrivere i file di destinazione e rimuovono le impostazioni Warden OTP ritirate solo dal nuovo file v1.

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

v1 rifiuta `WARDEN_OTP_ENABLED` e `WARDEN_OTP_SECRET_KEY` se uno dei nomi è presente, anche vuoto o `false`. Non reinserirli. Per TOTP configurare in `stargate-v1.env` `HERALD_ENABLED=true`, `HERALD_TOTP_ENABLED=true`, `HERALD_URL` e un metodo di autenticazione del servizio Herald supportato. Configurare anche il proxy TOTP di Herald e un `HERALD_TOTP_ENCRYPTION_KEY` indipendente in herald-totp; non riutilizzare automaticamente il vecchio secret Warden. Vedere [Configurazione](CONFIG.md).

2. **Conservare il container precedente e il suo ambiente originale per il rollback:**

```bash
docker stop stargate
docker rename stargate stargate-previous
```

3. **Scaricare Nuova Immagine:**

```bash
docker pull ghcr.io/soulteary/stargate:v1.0.0
```

4. **Avviare Nuovo Container:**

```bash
docker run -d \
  --name stargate \
  --env-file ./stargate-v1.env \
  -p 8080:8080 \
  --restart unless-stopped \
  ghcr.io/soulteary/stargate:v1.0.0
```

5. **Verificare il Servizio:**

```bash
curl --fail http://127.0.0.1:8080/healthz
curl --fail http://127.0.0.1:8080/readyz
```

### Rollback

Se si verificano problemi dopo l'aggiornamento:

```bash
# Rimuovere il nuovo container e ripristinare quello precedente invariato.
# Conservare stargate-v0.12.0.env come riferimento di configurazione per il rollback.
docker rm -f stargate
docker rename stargate-previous stargate
docker start stargate
```
