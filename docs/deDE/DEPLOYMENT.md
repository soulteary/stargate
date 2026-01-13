# Bereitstellungsanleitung

Dieses Dokument bietet eine detaillierte Bereitstellungsanleitung für den Stargate Forward Auth-Dienst.

## Inhaltsverzeichnis

- [Bereitstellungsmethoden](#bereitstellungsmethoden)
- [Docker-Bereitstellung](#docker-bereitstellung)
- [Docker Compose-Bereitstellung](#docker-compose-bereitstellung)
- [Traefik-Integration](#traefik-integration)
- [Produktionsbereitstellung](#produktionsbereitstellung)
- [Überwachung und Wartung](#überwachung-und-wartung)
- [Fehlerbehebung](#fehlerbehebung)

## Bereitstellungsmethoden

Stargate unterstützt die folgenden Bereitstellungsmethoden:

1. **Docker-Container** (Empfohlen) - Die einfachste und häufigste Methode
2. **Docker Compose** - Geeignet für lokale Entwicklung und Tests
3. **Kubernetes** - Geeignet für großskalige Produktionsumgebungen
4. **Direkte Binär-Ausführung** - Geeignet für spezielle Szenarien

Dieses Dokument behandelt hauptsächlich Docker- und Docker Compose-Bereitstellungsmethoden.

## Docker-Bereitstellung

### Image erstellen

#### Aus Quellcode erstellen

```bash
cd codes
docker build -t stargate:latest .
```

#### Build-Parameter

- **Basis-Image**: `golang:1.25-alpine` (Build-Stufe)
- **Ausführungs-Image**: `scratch` (minimales Image)
- **Arbeitsverzeichnis**: `/app`
- **Exponierter Port**: `80`

### Container ausführen

#### Grundlegende Ausführung

```bash
docker run -d \
  --name stargate \
  -p 80:80 \
  -e AUTH_HOST=auth.example.com \
  -e PASSWORDS=plaintext:yourpassword \
  stargate:latest
```

#### Ausführung mit vollständiger Konfiguration

```bash
docker run -d \
  --name stargate \
  -p 80:80 \
  -e AUTH_HOST=auth.example.com \
  -e PASSWORDS=bcrypt:$2a$10$N9qo8uLOickgx2ZMRZoMyeIjZAgcfl7p92ldGxad68LJZdL17lhWy \
  -e DEBUG=false \
  -e LANGUAGE=de \
  -e LOGIN_PAGE_TITLE=Mein Authentifizierungsdienst \
  -e LOGIN_PAGE_FOOTER_TEXT=© 2024 Meine Firma \
  -e COOKIE_DOMAIN=.example.com \
  --restart unless-stopped \
  stargate:latest
```

#### Parameterbeschreibung

- `-d`: Im Hintergrund ausführen
- `--name stargate`: Containername
- `-p 80:80`: Port-Mapping (Host-Port:Container-Port)
- `-e`: Umgebungsvariable
- `--restart unless-stopped`: Automatische Neustart-Richtlinie

### Protokolle anzeigen

```bash
# Protokolle in Echtzeit anzeigen
docker logs -f stargate

# Die letzten 100 Zeilen der Protokolle anzeigen
docker logs --tail 100 stargate
```

### Stoppen und Entfernen

```bash
# Container stoppen
docker stop stargate

# Container entfernen
docker rm stargate

# Stoppen und entfernen
docker rm -f stargate
```

## Docker Compose-Bereitstellung

### Grundlegende Konfiguration

Das Projekt stellt eine Beispiel-Datei `docker-compose.yml` bereit:

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

### Dienste starten

```bash
cd codes
docker-compose up -d
```

### Dienste stoppen

```bash
docker-compose down
```

### Protokolle anzeigen

```bash
# Alle Dienstprotokolle anzeigen
docker-compose logs -f

# Protokolle eines bestimmten Dienstes anzeigen
docker-compose logs -f stargate
```

### Angepasste Konfiguration

Bearbeiten Sie `docker-compose.yml` und ändern Sie die Umgebungsvariablen:

```yaml
services:
  stargate:
    image: stargate
    environment:
      - AUTH_HOST=auth.example.com
      - PASSWORDS=bcrypt:$2a$10$...
      - DEBUG=false
      - LANGUAGE=de
      - COOKIE_DOMAIN=.example.com
```

## Traefik-Integration

### Grundlegende Konfiguration

Stargate ist für die Integration mit Traefik konzipiert und bietet Authentifizierung über das Forward Auth Middleware.

#### 1. Stargate-Dienst konfigurieren

Konfigurieren Sie Stargate in `docker-compose.yml`:

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

#### 2. Geschützte Dienste konfigurieren

Wenden Sie das Stargate Middleware auf Dienste an, die Authentifizierung benötigen:

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
      - "traefik.http.routers.your-app.middlewares=stargate"  # Authentifizierungs-Middleware anwenden
```

### HTTPS-Konfiguration

#### Verwendung von Let's Encrypt

```yaml
services:
  stargate:
    labels:
      - "traefik.http.routers.auth.entrypoints=https"
      - "traefik.http.routers.auth.tls.certresolver=letsencrypt"
      - "traefik.http.routers.auth.tls=true"
```

#### Verwendung benutzerdefinierter Zertifikate

```yaml
services:
  stargate:
    labels:
      - "traefik.http.routers.auth.entrypoints=https"
      - "traefik.http.routers.auth.tls=true"
      - "traefik.http.routers.auth.tls.certfile=/path/to/cert.pem"
      - "traefik.http.routers.auth.tls.keyfile=/path/to/key.pem"
```

### Cross-Domain-Sitzungsfreigabe

Wenn Sie Sitzungen zwischen Subdomains teilen müssen:

1. Setzen Sie die Umgebungsvariable `COOKIE_DOMAIN`:

```yaml
services:
  stargate:
    environment:
      - COOKIE_DOMAIN=.example.com
```

2. Stellen Sie sicher, dass alle zugehörigen Domains über Traefik zu Stargate geroutet werden

3. Anmeldeablauf:
   - Der Benutzer meldet sich bei `auth.example.com` an
   - Weiterleitung zu `app.example.com/_session_exchange?id=<session_id>`
   - Das Sitzungs-Cookie wird auf die Domain `.example.com` gesetzt
   - Alle Subdomains `*.example.com` können diese Sitzung verwenden

## Produktionsbereitstellung

### Sicherheitsempfehlungen

#### 1. Starke Passwort-Algorithmen verwenden

**Nicht empfohlen:**

```bash
PASSWORDS=plaintext:yourpassword
```

**Empfohlen:**

```bash
PASSWORDS=bcrypt:$2a$10$N9qo8uLOickgx2ZMRZoMyeIjZAgcfl7p92ldGxad68LJZdL17lhWy
```

#### 2. HTTPS aktivieren

- HTTPS über Traefik konfigurieren
- Automatische Let's Encrypt-Zertifikate verwenden
- HTTPS-Weiterleitung erzwingen

#### 3. Debug-Modus deaktivieren

```bash
DEBUG=false
```

#### 4. Ressourcengrenzen festlegen

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

#### 5. Gesundheitsprüfungen verwenden

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

### Hochverfügbarkeits-Bereitstellung

#### 1. Multi-Instance-Bereitstellung

```yaml
services:
  stargate:
    deploy:
      replicas: 3
```

**Hinweis:** Stargate verwendet In-Memory-Sitzungsspeicher, Sitzungen werden nicht zwischen Instanzen geteilt. Wenn eine Multi-Instance-Bereitstellung erforderlich ist, wird empfohlen:

- Sitzungspersistenz des Load Balancers (Sticky Session) verwenden
- Oder auf Unterstützung für externen Sitzungsspeicher (Redis) warten

#### 2. Lastverteilung

Hinzufügen eines Load Balancers vor Traefik:

```yaml
services:
  traefik:
    labels:
      - "traefik.http.services.stargate.loadbalancer.server.port=80"
```

### Überwachungskonfiguration

#### 1. Protokollsammlung

```yaml
services:
  stargate:
    logging:
      driver: "json-file"
      options:
        max-size: "10m"
        max-file: "3"
```

#### 2. Gesundheitsprüfungs-Endpunkt

Verwenden Sie den Endpunkt `/health` für die Überwachung:

```bash
# Gesundheitsprüfungs-Skript
#!/bin/bash
if curl -f http://localhost/health > /dev/null 2>&1; then
  exit 0
else
  exit 1
fi
```

#### 3. Prometheus-Integration

(In Implementierung) Zukünftige Versionen werden den Export von Prometheus-Metriken unterstützen.

## Überwachung und Wartung

### Protokollverwaltung

#### Protokolle anzeigen

```bash
# Docker
docker logs -f stargate

# Docker Compose
docker-compose logs -f stargate
```

#### Protokollierungsebenen

- `DEBUG=true`: Detaillierte Debug-Informationen
- `DEBUG=false`: Nur kritische Informationen

#### Protokollrotation

Docker-Protokolltreiber konfigurieren:

```yaml
services:
  stargate:
    logging:
      driver: "json-file"
      options:
        max-size: "10m"
        max-file: "3"
```

### Leistungsüberwachung

#### Ressourcennutzung

```bash
# Ressourcennutzung des Containers anzeigen
docker stats stargate
```

#### Antwortzeit

Überwachen Sie die Antwortzeit mit dem Gesundheitsprüfungs-Endpunkt:

```bash
time curl http://auth.example.com/health
```

### Regelmäßige Wartung

1. **Images aktualisieren**: Regelmäßig die neuesten Images herunterladen
2. **Protokolle überprüfen**: Regelmäßig Fehlerprotokolle überprüfen
3. **Ressourcen überwachen**: CPU- und Speicherverwendung überwachen
4. **Konfiguration sichern**: Konfiguration der Umgebungsvariablen sichern

## Fehlerbehebung

### Häufige Probleme

#### 1. Dienst startet nicht

**Problem:** Container beendet sich sofort nach dem Start

**Fehlerbehebungs-Schritte:**

```bash
# Container-Protokolle anzeigen
docker logs stargate

# Konfiguration überprüfen
docker inspect stargate | grep -A 20 Env
```

**Häufige Ursachen:**

- Fehlende erforderliche Konfiguration (`AUTH_HOST`, `PASSWORDS`)
- Falsches Passwort-Konfigurationsformat
- Port belegt

#### 2. Authentifizierung schlägt fehl

**Problem:** Benutzer können sich nicht anmelden

**Fehlerbehebungs-Schritte:**

1. Überprüfen, ob die Passwort-Konfiguration korrekt ist
2. Überprüfen, ob der Passwort-Algorithmus übereinstimmt
3. Dienstprotokolle anzeigen: `docker logs stargate`

**Häufige Ursachen:**

- Falsche Passwort-Konfiguration
- Passwort-Algorithmus-Inkompatibilität (z. B. bcrypt konfiguriert, aber Klartext-Passwort verwendet)
- Falsche Cookie-Domain-Konfiguration

#### 3. Cross-Domain-Sitzungen funktionieren nicht

**Problem:** Sitzungen können nicht zwischen Subdomains geteilt werden

**Fehlerbehebungs-Schritte:**

1. Überprüfen Sie die `COOKIE_DOMAIN`-Konfiguration
2. Bestätigen Sie, dass das Cookie-Domain-Format korrekt ist (`.example.com`)
3. Überprüfen Sie die Cookie-Einstellungen des Browsers

**Lösung:**

```bash
# Sicherstellen, dass COOKIE_DOMAIN definiert ist
COOKIE_DOMAIN=.example.com
```

#### 4. Traefik-Integrationsprobleme

**Problem:** Traefik kann Authentifizierungsanfragen nicht korrekt weiterleiten

**Fehlerbehebungs-Schritte:**

1. Überprüfen Sie die Traefik-Label-Konfiguration
2. Bestätigen Sie, dass die Netzwerkkonfiguration korrekt ist
3. Überprüfen Sie die Forward Auth Middleware-Adresse

**Lösung:**

```yaml
# Sicherstellen, dass die Middleware-Adresse korrekt ist
- "traefik.http.middlewares.stargate.forwardauth.address=http://stargate/_auth"
```

### Debug-Tipps

#### 1. Debug-Modus aktivieren

```bash
DEBUG=true
```

#### 2. Netzwerkverbindung überprüfen

```bash
# Von innerhalb des Containers testen
docker exec stargate wget -O- http://localhost/health
```

#### 3. Traefik-Protokolle anzeigen

```bash
docker logs traefik
```

#### 4. API-Endpunkte testen

```bash
# Gesundheitsprüfung testen
curl http://auth.example.com/health

# Authentifizierung testen (mit Header)
curl -H "Stargate-Password: yourpassword" http://auth.example.com/_auth

# Authentifizierung testen (mit Cookie)
curl -H "Cookie: stargate_session_id=<session_id>" http://auth.example.com/_auth
```

### Hilfe erhalten

Wenn Sie Probleme haben:

1. Protokolle anzeigen: `docker logs stargate`
2. Konfiguration überprüfen: Bestätigen, dass alle Umgebungsvariablen korrekt sind
3. Dokumentation konsultieren: [API-Dokumentation](API.md), [Konfigurationsreferenz](CONFIG.md)
4. Issue einreichen: Einen Problembericht im Projekt-Repository einreichen

## Aktualisierungsanleitung

### Aktualisierungsschritte

1. **Konfiguration sichern**: Aktuelle Konfiguration der Umgebungsvariablen sichern

2. **Dienst stoppen:**

```bash
docker stop stargate
```

3. **Neues Image herunterladen:**

```bash
docker pull stargate:latest
```

4. **Neuen Container starten:**

```bash
docker run -d \
  --name stargate \
  ...(gespeicherte Konfiguration verwenden)
  stargate:latest
```

5. **Dienst überprüfen:**

```bash
curl http://auth.example.com/health
```

### Zurücksetzen

Wenn nach der Aktualisierung Probleme auftreten:

```bash
# Neuen Container stoppen
docker stop stargate

# Mit altem Image starten
docker run -d \
  --name stargate \
  ...(gespeicherte Konfiguration verwenden)
  stargate:<old-version>
```
