# Guide de Déploiement

Ce document fournit un guide de déploiement détaillé pour le service Stargate Forward Auth.

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

```bash
cd codes
docker build -t stargate:latest .
```

#### Paramètres de Build

- **Image de Base** : `golang:1.25-alpine` (étape de build)
- **Image d'Exécution** : `scratch` (image minimale)
- **Répertoire de Travail** : `/app`
- **Port Exposé** : `80`

### Exécuter le Conteneur

#### Exécution de Base

```bash
docker run -d \
  --name stargate \
  -p 80:80 \
  -e AUTH_HOST=auth.example.com \
  -e PASSWORDS=plaintext:yourpassword \
  stargate:latest
```

#### Exécution avec Configuration Complète

```bash
docker run -d \
  --name stargate \
  -p 80:80 \
  -e AUTH_HOST=auth.example.com \
  -e PASSWORDS=bcrypt:$2a$10$N9qo8uLOickgx2ZMRZoMyeIjZAgcfl7p92ldGxad68LJZdL17lhWy \
  -e DEBUG=false \
  -e LANGUAGE=fr \
  -e LOGIN_PAGE_TITLE=Mon Service d'Authentification \
  -e LOGIN_PAGE_FOOTER_TEXT=© 2024 Ma Société \
  -e COOKIE_DOMAIN=.example.com \
  --restart unless-stopped \
  stargate:latest
```

#### Description des Paramètres

- `-d` : Exécuter en arrière-plan
- `--name stargate` : Nom du conteneur
- `-p 80:80` : Mappage de port (port hôte:port conteneur)
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

# Arrêter et supprimer
docker rm -f stargate
```

## Déploiement Docker Compose

### Configuration de Base

Le projet fournit un fichier d'exemple `docker-compose.yml` :

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

### Démarrer les Services

```bash
cd codes
docker-compose up -d
```

### Arrêter les Services

```bash
docker-compose down
```

### Afficher les Journaux

```bash
# Afficher tous les journaux des services
docker-compose logs -f

# Afficher les journaux d'un service spécifique
docker-compose logs -f stargate
```

### Configuration Personnalisée

Modifier `docker-compose.yml` et modifier les variables d'environnement :

```yaml
services:
  stargate:
    image: stargate
    environment:
      - AUTH_HOST=auth.example.com
      - PASSWORDS=bcrypt:$2a$10$...
      - DEBUG=false
      - LANGUAGE=fr
      - COOKIE_DOMAIN=.example.com
```

## Intégration Traefik

### Configuration de Base

Stargate est conçu pour s'intégrer avec Traefik, fournissant l'authentification via le middleware Forward Auth.

#### 1. Configurer le Service Stargate

Configurer Stargate dans `docker-compose.yml` :

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
      - "traefik.docker.network=traefik"
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
```

2. S'assurer que tous les domaines associés sont routés vers Stargate via Traefik

3. Flux de connexion :
   - L'utilisateur se connecte à `auth.example.com`
   - Redirige vers `app.example.com/_session_exchange?id=<session_id>`
   - Le cookie de session est défini au domaine `.example.com`
   - Tous les sous-domaines `*.example.com` peuvent utiliser cette session

## Déploiement en Production

### Recommandations de Sécurité

#### 1. Utiliser des Algorithmes de Mot de Passe Fort

**Non Recommandé :**

```bash
PASSWORDS=plaintext:yourpassword
```

**Recommandé :**

```bash
PASSWORDS=bcrypt:$2a$10$N9qo8uLOickgx2ZMRZoMyeIjZAgcfl7p92ldGxad68LJZdL17lhWy
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
      test: ["CMD", "wget", "--quiet", "--tries=1", "--spider", "http://localhost/health"]
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

**Note :** Stargate utilise le stockage de session en mémoire, les sessions ne sont pas partagées entre les instances. Si un déploiement multi-instance est nécessaire, il est recommandé de :

- Utiliser la persistance de session du répartiteur de charge (Sticky Session)
- Ou attendre le support du stockage de session externe (Redis)

#### 2. Répartition de Charge

Ajouter un répartiteur de charge avant Traefik :

```yaml
services:
  traefik:
    labels:
      - "traefik.http.services.stargate.loadbalancer.server.port=80"
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

Utiliser le point de terminaison `/health` pour la surveillance :

```bash
# Script de vérification de santé
#!/bin/bash
if curl -f http://localhost/health > /dev/null 2>&1; then
  exit 0
else
  exit 1
fi
```

#### 3. Intégration Prometheus

(À implémenter) Les versions futures supporteront l'export de métriques Prometheus.

## Surveillance et Maintenance

### Gestion des Journaux

#### Afficher les Journaux

```bash
# Docker
docker logs -f stargate

# Docker Compose
docker-compose logs -f stargate
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
time curl http://auth.example.com/health
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
- "traefik.http.middlewares.stargate.forwardauth.address=http://stargate/_auth"
```

### Conseils de Débogage

#### 1. Activer le Mode Débogage

```bash
DEBUG=true
```

#### 2. Vérifier la Connexion Réseau

```bash
# Tester depuis l'intérieur du conteneur
docker exec stargate wget -O- http://localhost/health
```

#### 3. Afficher les Journaux Traefik

```bash
docker logs traefik
```

#### 4. Tester les Points de Terminaison API

```bash
# Tester la vérification de santé
curl http://auth.example.com/health

# Tester l'authentification (utilisant l'En-tête)
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

1. **Sauvegarder la Configuration** : Sauvegarder la configuration actuelle des variables d'environnement

2. **Arrêter le Service :**

```bash
docker stop stargate
```

3. **Télécharger la Nouvelle Image :**

```bash
docker pull stargate:latest
```

4. **Démarrer le Nouveau Conteneur :**

```bash
docker run -d \
  --name stargate \
  ...(utiliser la configuration sauvegardée)
  stargate:latest
```

5. **Vérifier le Service :**

```bash
curl http://auth.example.com/health
```

### Retour en Arrière

Si des problèmes surviennent après la mise à jour :

```bash
# Arrêter le nouveau conteneur
docker stop stargate

# Démarrer avec l'ancienne image
docker run -d \
  --name stargate \
  ...(utiliser la configuration sauvegardée)
  stargate:<old-version>
```
