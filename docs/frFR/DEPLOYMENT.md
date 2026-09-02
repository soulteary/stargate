# Guide de Déploiement

Ce document fournit un guide de déploiement détaillé pour le service Stargate Forward Auth.

Pour une mise à niveau depuis v0.12.0, consultez d'abord le [guide de migration v1.0.0](MIGRATION_V1.md).

## Table des Matières

- [Méthodes de Déploiement](#méthodes-de-déploiement)
- [Déploiement Docker](#déploiement-docker)
- [Déploiement Docker Compose](#déploiement-docker-compose)
- [Intégration Traefik](#intégration-traefik)
- [Déploiement en Production](#déploiement-en-production)
- [Surveillance et Maintenance](#surveillance-et-maintenance)
- [Dépannage](#dépannage)

## Méthodes de Déploiement

Stargate supporte les méthodes de déploiement suivantes :

1. **Conteneur Docker** (Recommandé) - Le plus simple et le plus courant
2. **Docker Compose** - Adapté au développement local et aux tests
3. **Kubernetes** - Adapté aux environnements de production à grande échelle
4. **Exécution Binaire Directe** - Adapté aux scénarios spéciaux

Ce document présente principalement les méthodes de déploiement Docker et Docker Compose.

## Dépendances de Service

Stargate peut s'intégrer avec les services optionnels suivants :

### Service Warden

**Fonction :** Gestion de la liste blanche d'utilisateurs et fourniture d'informations utilisateur

**Exigences de Déploiement :**
- Nécessite une base de données (PostgreSQL/MySQL/SQLite)
- Fournit une interface API HTTP
- Supporte l'authentification par clé API

**Configuration :**
```bash
WARDEN_ENABLED=true
WARDEN_URL=http://warden:8080
WARDEN_API_KEY=your-api-key
```

### Service Herald

**Fonction :** Envoi et vérification OTP/code de vérification

**Exigences de Déploiement :**
- Nécessite Redis (stocke les challenges et l'état de limitation de débit)
- Fournit une interface API HTTP
- Supporte l'authentification par signature HMAC ou mTLS (recommandé pour la production)

**Configuration :**
```bash
HERALD_ENABLED=true
HERALD_URL=http://herald:8080
HERALD_HMAC_SECRET=your-hmac-secret  # Recommandé pour la production
```

### Sécurité de Communication Inter-Services

**Exigences pour l'Environnement de Production :**

1. **Authentification par Signature HMAC** (Recommandé) :
   - Stargate ↔ Herald utilise la signature HMAC-SHA256
   - Configurer `HERALD_HMAC_SECRET`
   - Inclut la vérification de l'horodatage (prévent les attaques de rejeu)

2. **Authentification mTLS** (Optionnel, plus sécurisé) :
   - Configurer le certificat client TLS
   - Définir `HERALD_TLS_CLIENT_CERT_FILE` et `HERALD_TLS_CLIENT_KEY_FILE`
   - Configurer la vérification du certificat CA

3. **Isolation Réseau :**
   - La communication inter-services doit être sur le réseau interne
   - Utiliser des règles de pare-feu pour restreindre l'accès
   - Éviter d'exposer les services au réseau public

## Déploiement Docker

### Construire l'Image

#### Construire depuis la Source

À la racine du projet :

```bash
docker build -f docker/Dockerfile -t stargate:latest .
```

#### Paramètres de Build

- **Image de Base** : `golang:1.27.0-alpine3.24` (étape de build)
- **Image d'Exécution** : `alpine:3.24` (certificats CA et BusyBox `wget` pour HTTPS et les health checks)
- **Répertoire de Travail** : `/app`
- **Port Exposé** : `8080`

### Exécuter le Conteneur

#### Exécution de Base

```bash
docker run -d \
  --name stargate \
  -p 8080:8080 \
  -e AUTH_HOST=auth.example.com \
  -e 'PASSWORDS=plaintext:yourpassword' \
  stargate:latest
```

#### Exécution avec Configuration Complète

```bash
docker run -d \
  --name stargate \
  -p 8080:8080 \
  -e AUTH_HOST=auth.example.com \
  -e 'PASSWORDS=bcrypt:$2a$10$N9qo8uLOickgx2ZMRZoMyeIjZAgcfl7p92ldGxad68LJZdL17lhWy' \
  -e DEBUG=false \
  -e LANGUAGE=fr \
  -e "LOGIN_PAGE_TITLE=Mon Service d'Authentification" \
  -e 'LOGIN_PAGE_FOOTER_TEXT=© 2024 Ma Société' \
  -e COOKIE_DOMAIN=.example.com \
  --restart unless-stopped \
  stargate:latest
```

#### Description des Paramètres

- `-d` : Exécuter en arrière-plan
- `--name stargate` : Nom du conteneur
- `-p 8080:8080` : Mappage de port (port hôte:port conteneur)
- `-e` : Variable d'environnement
- `--restart unless-stopped` : Politique de redémarrage automatique

### Afficher les Journaux

```bash
# Afficher les journaux en temps réel
docker logs -f stargate

# Afficher les 100 dernières lignes des journaux
docker logs --tail 100 stargate
```

### Arrêter et Supprimer

```bash
# Arrêter le conteneur
docker stop stargate

# Supprimer le conteneur
docker rm stargate

```

## Déploiement Docker Compose

### Configuration de Base

Le fichier `docker-compose.yml` à la racine du dépôt constitue la pile intégrée : il démarre Traefik, Stargate et le service de démonstration protégé sur le réseau `stargate-traefik` géré par Compose (`172.30.0.0/24`). Aucun réseau externe préexistant n'est nécessaire.

```bash
docker compose config
docker compose up -d
```

### Utiliser un réseau Traefik externe existant

Utilisez l'exemple distinct ci-dessous uniquement si Traefik fonctionne déjà sur un réseau Docker externe partagé. Le nom réel du réseau et le CIDR obtenu par inspection alimentent le réseau Compose, chaque label `traefik.docker.network` et `TRUSTED_PROXIES`.

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

### Inspecter ou créer le réseau externe

Si le réseau existe, les commandes l'inspectent et réutilisent tous ses CIDR attribués ; ne supprimez ni ne recréez un réseau de production existant. S'il n'existe pas, elles le créent avec le sous-réseau documenté.

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

### Arrêter les Services

```bash
docker compose down
```

### Afficher les Journaux

```bash
# Afficher tous les journaux des services
docker compose logs -f

# Afficher les journaux d'un service spécifique
docker compose logs -f stargate
```

### Configuration Personnalisée

Modifier `docker-compose.yml` et modifier les variables d'environnement :

```yaml
services:
  stargate:
    image: ghcr.io/soulteary/stargate:v1.0.0
    environment:
      - AUTH_HOST=auth.example.com
      - PASSWORDS=bcrypt:$$2a$$10$$...
      - TRUSTED_PROXIES=${TRAEFIK_NETWORK_CIDR:?set TRAEFIK_NETWORK_CIDR from docker network inspect}
      - DEBUG=false
      - LANGUAGE=fr
      - COOKIE_DOMAIN=.example.com
      - SESSION_EXCHANGE_SECRET=replace-with-at-least-32-random-characters
```

## Intégration Traefik

### Configuration de Base

Stargate est conçu pour s'intégrer avec Traefik, fournissant l'authentification via le middleware Forward Auth.

#### 1. Configurer le Service Stargate

Configurer Stargate dans `docker-compose.yml` :

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

#### 2. Configurer les Services Protégés

Appliquer le middleware Stargate aux services nécessitant une authentification :

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
      - "traefik.http.routers.your-app.middlewares=stargate"  # Appliquer le middleware d'authentification
```

### Configuration HTTPS

#### Utilisation de Let's Encrypt

```yaml
services:
  stargate:
    labels:
      - "traefik.http.routers.auth.entrypoints=https"
      - "traefik.http.routers.auth.tls.certresolver=letsencrypt"
      - "traefik.http.routers.auth.tls=true"
```

#### Utilisation de Certificats Personnalisés

```yaml
services:
  stargate:
    labels:
      - "traefik.http.routers.auth.entrypoints=https"
      - "traefik.http.routers.auth.tls=true"
      - "traefik.http.routers.auth.tls.certfile=/path/to/cert.pem"
      - "traefik.http.routers.auth.tls.keyfile=/path/to/key.pem"
```

### Partage de Session Cross-Domain

Si vous devez partager des sessions entre sous-domaines :

1. Définir la variable d'environnement `COOKIE_DOMAIN` :

```yaml
services:
  stargate:
    environment:
      - COOKIE_DOMAIN=.example.com
      - SESSION_EXCHANGE_SECRET=replace-with-at-least-32-random-characters
```

2. S'assurer que tous les domaines associés sont routés vers Stargate via Traefik

3. Flux de connexion :
   - L'utilisateur se connecte à `auth.example.com`
   - Redirige vers `app.example.com/_session_exchange?ticket=<opaque_ticket>`
   - Le cookie de session est défini au domaine `.example.com`
   - Tous les sous-domaines `*.example.com` peuvent utiliser cette session

## Déploiement en Production

### Recommandations de Sécurité

#### 1. Utiliser des Algorithmes de Mot de Passe Fort

**Non Recommandé :**

```bash
PASSWORDS='plaintext:yourpassword'
```

**Recommandé :**

```bash
PASSWORDS='bcrypt:$2a$10$N9qo8uLOickgx2ZMRZoMyeIjZAgcfl7p92ldGxad68LJZdL17lhWy'
```

#### 2. Activer HTTPS

- Configurer HTTPS via Traefik
- Utiliser les certificats automatiques Let's Encrypt
- Forcer la redirection HTTPS

#### 3. Désactiver le Mode Débogage

```bash
DEBUG=false
```

#### 4. Définir les Limites de Ressources

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

#### 5. Utiliser les Vérifications de Santé

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

### Déploiement à Haute Disponibilité

#### 1. Déploiement Multi-Instance

```yaml
services:
  stargate:
    deploy:
      replicas: 3
```

**Périmètre du stockage de session :**

- **Un processus / une réplique, y compris avec des callbacks inter-domaines :** le stockage en mémoire (`SESSION_STORAGE_ENABLED=false`) est pris en charge si l'émission, l'échange du ticket et l'utilisation ultérieure de la session atteignent le même processus Stargate actif. Les sessions et les empreintes de tickets consommés n'existent que dans ce processus. Un redémarrage perd les deux états, invalide les sessions actives et ne conserve pas l'usage unique des tickets.
- **Plusieurs processus ou répliques :** Redis est obligatoire dès qu'une session ou un ticket peut être traité par des processus différents. Utilisez `SESSION_STORAGE_ENABLED=true`, le même espace `SESSION_STORAGE_REDIS_*` et le même `SESSION_EXCHANGE_SECRET` sur toutes les répliques. Les Sticky Sessions ne sont qu'une optimisation de routage et ne remplacent ni l'état partagé ni la protection anti-rejeu inter-processus.
- **Mise à niveau progressive :** Redis est obligatoire pour un remplacement sans interruption, car anciens et nouveaux processus se chevauchent. Sans Redis, arrêtez puis démarrez le service et acceptez l'invalidation des sessions et une nouvelle connexion. Ne mélangez jamais v0.12.0 et v1.0.0 dans le même pool.
- **État persistant après redémarrage :** Redis est obligatoire si les sessions ou l'état anti-rejeu des tickets consommés doivent survivre au remplacement ou au redémarrage d'un processus.

#### 2. Répartition de Charge

Ajouter un répartiteur de charge avant Traefik :

```yaml
services:
  traefik:
    labels:
      - "traefik.http.services.stargate.loadbalancer.server.port=8080"
```

### Configuration de Surveillance

#### 1. Collecte de Journaux

```yaml
services:
  stargate:
    logging:
      driver: "json-file"
      options:
        max-size: "10m"
        max-file: "3"
```

#### 2. Point de Terminaison de Vérification de Santé

Utilisez `/healthz` pour la liveness du processus et `/readyz` pour la disponibilité des dépendances. `/health` n'est plus qu'un alias de disponibilité obsolète.

```bash
# Script de vérification de santé
#!/bin/bash
if curl -f http://localhost:8080/healthz > /dev/null 2>&1; then
  exit 0
else
  exit 1
fi
```

#### 3. Intégration Prometheus

Stargate expose les métriques Prometheus sur `GET /metrics`. Configurez Prometheus pour scraper cet endpoint. L'endpoint métriques est exclu par défaut de la journalisation et du tracing des requêtes.

## Surveillance et Maintenance

### Gestion des Journaux

#### Afficher les Journaux

```bash
# Docker
docker logs -f stargate

# Docker Compose
docker compose logs -f stargate
```

#### Niveaux de Journalisation

- `DEBUG=true` : Informations de débogage détaillées
- `DEBUG=false` : Uniquement les informations critiques

#### Rotation des Journaux

Configurer le pilote de journal Docker :

```yaml
services:
  stargate:
    logging:
      driver: "json-file"
      options:
        max-size: "10m"
        max-file: "3"
```

### Surveillance des Performances

#### Utilisation des Ressources

```bash
# Afficher l'utilisation des ressources du conteneur
docker stats stargate
```

#### Temps de Réponse

Surveiller le temps de réponse en utilisant le point de terminaison de vérification de santé :

```bash
time curl http://auth.example.com/healthz
```

### Maintenance Régulière

1. **Mettre à Jour les Images** : Télécharger régulièrement les dernières images
2. **Vérifier les Journaux** : Vérifier régulièrement les journaux d'erreur
3. **Surveiller les Ressources** : Surveiller l'utilisation du CPU et de la mémoire
4. **Sauvegarder la Configuration** : Sauvegarder la configuration des variables d'environnement

## Dépannage

### Problèmes Courants

#### 1. Le Service Ne Démarre Pas

**Problème :** Le conteneur se ferme immédiatement après le démarrage

**Étapes de Dépannage :**

```bash
# Afficher les journaux du conteneur
docker logs stargate

# Vérifier la configuration
docker inspect stargate | grep -A 20 Env
```

**Causes Courantes :**

- Configuration requise manquante (`AUTH_HOST`, `PASSWORDS`)
- Format de configuration de mot de passe incorrect
- Port occupé

#### 2. L'Authentification Échoue

**Problème :** Les utilisateurs ne peuvent pas se connecter

**Étapes de Dépannage :**

1. Vérifier si la configuration du mot de passe est correcte
2. Vérifier si l'algorithme du mot de passe correspond
3. Afficher les journaux du service : `docker logs stargate`

**Causes Courantes :**

- Configuration de mot de passe incorrecte
- Incompatibilité d'algorithme de mot de passe (par ex., bcrypt configuré mais mot de passe en texte brut utilisé)
- Configuration incorrecte du domaine de cookie

#### 3. Les Sessions Cross-Domain Ne Fonctionnent Pas

**Problème :** Impossible de partager des sessions entre sous-domaines

**Étapes de Dépannage :**

1. Vérifier la configuration `COOKIE_DOMAIN`
2. Confirmer que le format du domaine de cookie est correct (`.example.com`)
3. Vérifier les paramètres de cookie du navigateur

**Solution :**

```bash
# S'assurer que COOKIE_DOMAIN est défini
COOKIE_DOMAIN=.example.com
```

#### 4. Problèmes d'Intégration Traefik

**Problème :** Traefik ne peut pas transmettre correctement les requêtes d'authentification

**Étapes de Dépannage :**

1. Vérifier la configuration des labels Traefik
2. Confirmer que la configuration réseau est correcte
3. Vérifier l'adresse du middleware Forward Auth

**Solution :**

```yaml
# S'assurer que l'adresse du middleware est correcte
- "traefik.http.middlewares.stargate.forwardauth.address=http://stargate:8080/_auth"
- "traefik.http.middlewares.stargate.forwardauth.authResponseHeaders=X-Forwarded-User,X-Auth-User,X-Auth-Email,X-Auth-Name,X-Auth-Scopes,X-Auth-Role,X-Auth-AMR"
```

### Conseils de Débogage

#### 1. Activer le Mode Débogage

```bash
DEBUG=true
```

#### 2. Vérifier la Connexion Réseau

```bash
# Tester depuis l'intérieur du conteneur
docker exec stargate wget -q -O - http://127.0.0.1:8080/healthz
```

#### 3. Afficher les Journaux Traefik

```bash
docker logs traefik
```

#### 4. Tester les Points de Terminaison API

```bash
# Tester la vérification de santé
curl http://auth.example.com/healthz

# Tester l'authentification (utilisant l'En-tête)
# Prérequis côté serveur : PASSWORD_HEADER_AUTH_ENABLED=true
curl -H "Stargate-Password: yourpassword" http://auth.example.com/_auth

# Tester l'authentification (utilisant le Cookie)
curl -H "Cookie: stargate_session_id=<session_id>" http://auth.example.com/_auth
```

### Obtenir de l'Aide

Si vous rencontrez des problèmes :

1. Afficher les journaux : `docker logs stargate`
2. Vérifier la configuration : Confirmer que toutes les variables d'environnement sont correctes
3. Consulter la documentation : [Documentation API](API.md), [Référence de Configuration](CONFIG.md)
4. Soumettre un Issue : Soumettre un rapport de problème dans le dépôt du projet

## Guide de Mise à Jour

### Étapes de Mise à Jour

1. **Créer des fichiers d'environnement distincts pour le retour arrière et v1** : conserver `stargate.env` sans modification. Les commandes suivantes refusent d'écraser les fichiers cibles et retirent les anciens paramètres Warden OTP uniquement du nouveau fichier v1.

```bash
set -eu
old_env=./stargate.env
rollback_env=./stargate-v0.12.0.env
v1_env=./stargate-v1.env
old_container=${STARGATE_OLD_CONTAINER:-stargate}

test ! -e "$rollback_env"
test ! -e "$v1_env"
umask 077

# Si le fichier manque, exporter de manière protégée l'environnement du conteneur existant.
if [ ! -e "$old_env" ]; then
  export_tmp=$(mktemp "${old_env}.tmp.XXXXXX")
  trap 'rm -f "$export_tmp"' 0 1 2 15
  docker inspect --format '{{range .Config.Env}}{{println .}}{{end}}' "$old_container" > "$export_tmp"
  ln "$export_tmp" "$old_env"
  rm -f "$export_tmp"
  trap - 0 1 2 15
fi

test -f "$old_env"
chmod 600 "$old_env"
(set -C; cat "$old_env" > "$rollback_env")
(set -C; awk '
  /^[[:space:]]*WARDEN_OTP_(ENABLED|SECRET_KEY)[[:space:]]*(=|$)/ { next }
  /^[[:space:]]*PORT[[:space:]]*(=|$)/ { print "PORT=8080"; next }
  { print }
' "$old_env" > "$v1_env")
```

Le TOTP via Herald exige que Stargate résolve les utilisateurs authentifiés par Warden ; configurez donc aussi `WARDEN_ENABLED=true` et `WARDEN_URL`.

v1 refuse `WARDEN_OTP_ENABLED` et `WARDEN_OTP_SECRET_KEY` dès que l'un de ces noms est présent, même vide ou à `false`. Ne les réintroduisez pas. Pour TOTP, configurez dans `stargate-v1.env` `HERALD_ENABLED=true`, `HERALD_TOTP_ENABLED=true`, `HERALD_URL` et une méthode d'authentification de service Herald prise en charge. Configurez aussi le proxy TOTP de Herald et un `HERALD_TOTP_ENCRYPTION_KEY` indépendant dans herald-totp ; ne réutilisez pas automatiquement l'ancien secret Warden. Voir la [configuration](CONFIG.md).

2. **Conserver l'ancien conteneur et son environnement d'origine pour le retour en arrière :**

```bash
docker stop stargate
docker rename stargate stargate-previous
```

3. **Télécharger la Nouvelle Image :**

```bash
docker pull ghcr.io/soulteary/stargate:v1.0.0
```

4. **Démarrer le Nouveau Conteneur :**

```bash
docker run -d \
  --name stargate \
  --env-file ./stargate-v1.env \
  -p 8080:8080 \
  --restart unless-stopped \
  ghcr.io/soulteary/stargate:v1.0.0
```

5. **Vérifier le Service :**

```bash
curl --fail http://127.0.0.1:8080/healthz
curl --fail http://127.0.0.1:8080/readyz
```

### Retour en Arrière

Si des problèmes surviennent après la mise à jour :

```bash
# Supprimer le nouveau conteneur et restaurer l'ancien conteneur intact.
# Conserver stargate-v0.12.0.env comme référence de configuration de retour arrière.
docker rm -f stargate
docker rename stargate-previous stargate
docker start stargate
```
