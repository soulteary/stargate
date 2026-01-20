# Stargate - Forward Auth Service

[![License](https://img.shields.io/badge/license-Apache%202.0-blue.svg)](LICENSE)
[![Go Version](https://img.shields.io/badge/go-1.25+-blue.svg)](https://golang.org)
[![codecov](https://codecov.io/gh/soulteary/stargate/branch/main/graph/badge.svg)](https://codecov.io/gh/soulteary/stargate)
[![Go Report Card](https://goreportcard.com/badge/github.com/soulteary/stargate)](https://goreportcard.com/report/github.com/soulteary/stargate)

> **🚀 Votre Passerelle vers des Microservices Sécurisés**

Stargate est un service d'authentification avant prêt pour la production, léger, conçu pour être le **point d'authentification unique** de toute votre infrastructure. Construit avec Go et optimisé pour les performances, Stargate s'intègre parfaitement avec Traefik et d'autres proxies inverses pour protéger vos services backend—**sans écrire une seule ligne de code d'authentification dans vos applications**.

## 🌐 Documentation Multilingue

- [English](README.md) | [中文](README.zhCN.md) | [Français](README.frFR.md) | [Italiano](README.itIT.md) | [日本語](README.jaJP.md) | [Deutsch](README.deDE.md) | [한국어](README.koKR.md)

![Aperçu](.github/assets/preview.png)

### 🎯 Pourquoi Stargate ?

Fatigué d'implémenter la logique d'authentification dans chaque service ? Stargate résout ce problème en centralisant l'authentification au niveau du bord, vous permettant de :

- ✅ **Protéger plusieurs services** avec une seule couche d'authentification
- ✅ **Réduire la complexité du code** en supprimant la logique d'authentification de vos applications
- ✅ **Déployer en quelques minutes** avec Docker et une configuration simple
- ✅ **Évoluer sans effort** avec une empreinte de ressources minimale
- ✅ **Maintenir la sécurité** avec plusieurs algorithmes de chiffrement et une gestion de session sécurisée

### 💼 Cas d'Utilisation

Stargate est parfait pour :

- **Architecture de Microservices** : Protéger plusieurs services backend sans modifier le code de l'application
- **Applications Multi-Domaines** : Partager les sessions d'authentification entre différents domaines et sous-domaines
- **Outils Internes et Tableaux de Bord** : Ajouter rapidement l'authentification aux services internes et aux panneaux d'administration
- **Intégration de Passerelle API** : Utiliser avec Traefik, Nginx ou d'autres proxies inverses comme couche d'authentification unifiée
- **Développement et Tests** : Authentification simple basée sur un mot de passe pour les environnements de développement

## 📋 Table des Matières

- [Fonctionnalités](#fonctionnalités)
- [Démarrage Rapide](#démarrage-rapide)
- [Configuration](#configuration)
- [Documentation](#documentation)
- [Documentation API](#documentation-api)
- [Guide de Déploiement](#guide-de-déploiement)
- [Guide de Développement](#guide-de-développement)
- [Licence](#licence)

## ✨ Fonctionnalités

### 🔐 Sécurité de Niveau Entreprise

- **Plusieurs Algorithmes de Chiffrement de Mot de Passe** : Choisissez parmi plaintext (test), bcrypt, MD5, SHA512, et plus encore
- **Gestion de Session Sécurisée** : Sessions basées sur Cookie avec domaine et expiration personnalisables
- **Authentification Flexible** : Support pour l'authentification basée sur mot de passe et basée sur session

### 🌐 Capacités Avancées

- **Partage de Session Cross-Domain** : Partager de manière transparente les sessions d'authentification entre différents domaines/sous-domaines
- **Support Multilingue** : Interfaces intégrées en anglais et chinois, facilement extensibles pour plus de langues
- **Interface Personnalisable** : Personnalisez votre page de connexion avec des titres et textes de pied de page personnalisés

### 🚀 Performance et Fiabilité

- **Léger et Rapide** : Construit sur Go et le framework Fiber pour des performances exceptionnelles
- **Utilisation Minimale des Ressources** : Faible empreinte mémoire, parfait pour les environnements conteneurisés
- **Prêt pour la Production** : Architecture testée en conditions réelles conçue pour la fiabilité

### 📦 Expérience Développeur

- **Docker en Priorité** : Image Docker complète et configuration docker-compose prêtes à l'emploi
- **Traefik Natif** : Intégration du middleware Traefik Forward Auth sans configuration
- **Configuration Simple** : Configuration basée sur des variables d'environnement, aucun fichier complexe nécessaire

## 🚀 Démarrage Rapide

Mettez Stargate en route en **moins de 2 minutes** !

### Utilisation de Docker Compose (Recommandé)

**Étape 1 :** Cloner le dépôt
```bash
git clone <repository-url>
cd forward-auth
```

**Étape 2 :** Configurer votre authentification (modifier `codes/docker-compose.yml`)
```yaml
services:
  stargate:
    environment:
      - AUTH_HOST=auth.example.com
      - PASSWORDS=plaintext:yourpassword1|yourpassword2
```

**Étape 3 :** Démarrer le service
```bash
cd codes
docker-compose up -d
```

**C'est tout !** Votre service d'authentification est maintenant en cours d'exécution. 🎉

### Développement Local

1. Assurez-vous que Go 1.25 ou supérieur est installé

2. Naviguez vers le répertoire du projet :
```bash
cd codes
```

3. Exécutez le script de démarrage local :
```bash
chmod +x start-local.sh
./start-local.sh
```

4. Accédez à la page de connexion :
```
http://localhost:8080/_login?callback=localhost
```

## ⚙️ Configuration

Stargate utilise un système de configuration simple basé sur des variables d'environnement. Pas de fichiers YAML complexes ou d'analyse de configuration—il suffit de définir des variables d'environnement et vous êtes prêt.

### Configuration Requise

| Variable d'Environnement | Description | Exemple |
|-------------------------|-------------|---------|
| `AUTH_HOST` | Nom d'hôte du service d'authentification | `auth.example.com` |
| `PASSWORDS` | Configuration du mot de passe, format : `algorithm:password1\|password2\|password3` | `plaintext:test123\|admin456` |

### Configuration Optionnelle

| Variable d'Environnement | Description | Par Défaut | Exemple |
|-------------------------|-------------|------------|---------|
| `DEBUG` | Activer le mode débogage | `false` | `true` |
| `LANGUAGE` | Langue de l'interface | `en` | `fr` (Français), `zh` (Chinois), `en` (Anglais), `it` (Italien), `ja` (Japonais), `de` (Allemand), `ko` (Coréen) |
| `LOGIN_PAGE_TITLE` | Titre de la page de connexion | `Stargate - Login` | `Mon Service d'Authentification` |
| `LOGIN_PAGE_FOOTER_TEXT` | Texte du pied de page de connexion | `Copyright © 2024 - Stargate` | `© 2024 Ma Société` |
| `USER_HEADER_NAME` | Nom de l'en-tête utilisateur défini après authentification réussie | `X-Forwarded-User` | `X-Authenticated-User` |
| `COOKIE_DOMAIN` | Domaine du cookie (pour le partage de session cross-domain) | Vide (non défini) | `.example.com` |
| `PORT` | Port d'écoute du service (développement local uniquement) | `80` | `8080` |

### Format de Configuration du Mot de Passe

La configuration du mot de passe utilise le format suivant :
```
algorithm:password1|password2|password3
```

Algorithmes supportés :
- `plaintext` : Mot de passe en texte brut (test uniquement)
- `bcrypt` : Hash BCrypt
- `md5` : Hash MD5
- `sha512` : Hash SHA512

Exemples :
```bash
# Mots de passe en texte brut (plusieurs)
PASSWORDS=plaintext:test123|admin456|user789

# Hash BCrypt
PASSWORDS=bcrypt:$2a$10$N9qo8uLOickgx2ZMRZoMyeIjZAgcfl7p92ldGxad68LJZdL17lhWy

# Hash MD5
PASSWORDS=md5:5e884898da28047151d0e56f8dc6292773603d0d6aabbdd62a11ef721d1542d8
```

**Pour une configuration détaillée, voir : [docs/frFR/CONFIG.md](docs/frFR/CONFIG.md)**

## 📚 Documentation

Une documentation complète est disponible pour vous aider à tirer le meilleur parti de Stargate :

- 📐 **[Document d'Architecture](docs/frFR/ARCHITECTURE.md)** - Plongée approfondie dans l'architecture technique et les décisions de conception
- 🔌 **[Document API](docs/frFR/API.md)** - Référence complète des points de terminaison API avec exemples
- ⚙️ **[Référence de Configuration](docs/frFR/CONFIG.md)** - Options de configuration détaillées et meilleures pratiques
- 🚀 **[Guide de Déploiement](docs/frFR/DEPLOYMENT.md)** - Stratégies de déploiement en production et recommandations

## 📚 Documentation API

### Point de Terminaison de Vérification d'Authentification

#### `GET /_auth`

Le point de terminaison principal de vérification d'authentification pour Traefik Forward Auth.

**En-têtes de Requête :**
- `Stargate-Password` (optionnel) : Authentification par mot de passe pour les requêtes API
- `Cookie: stargate_session_id` (optionnel) : Authentification par session pour les requêtes Web

**Réponse :**
- `200 OK` : Authentification réussie, définit l'en-tête `X-Forwarded-User` (ou le nom d'en-tête utilisateur configuré)
- `401 Unauthorized` : Authentification échouée
- `500 Internal Server Error` : Erreur serveur

**Notes :**
- Les requêtes HTML redirigent vers la page de connexion en cas d'échec d'authentification
- Les requêtes API (JSON/XML) retournent une erreur 401 en cas d'échec d'authentification

### Point de Terminaison de Connexion

#### `GET /_login`

Affiche la page de connexion.

**Paramètres de Requête :**
- `callback` (optionnel) : URL de callback après connexion réussie

**Réponse :**
- Retourne le HTML de la page de connexion

#### `POST /_login`

Traite les requêtes de connexion.

**Données du Formulaire :**
- `password` : Mot de passe utilisateur
- `callback` (optionnel) : URL de callback après connexion réussie

**Priorité de Récupération du Callback :**
1. Depuis le Cookie (si précédemment défini)
2. Depuis les données du formulaire
3. Depuis les paramètres de requête
4. Si aucun des éléments ci-dessus, et le domaine d'origine diffère du domaine du service d'authentification, utiliser le domaine d'origine comme callback

**Réponse :**
- `200 OK` : Connexion réussie
  - Si le callback existe, redirige vers `{callback}/_session_exchange?id={session_id}`
  - Si aucun callback, retourne un message de succès (format HTML ou JSON, selon le type de requête)
- `401 Unauthorized` : Mot de passe incorrect
- `500 Internal Server Error` : Erreur serveur

### Point de Terminaison de Déconnexion

#### `GET /_logout`

Déconnecte l'utilisateur actuel et détruit la session.

**Réponse :**
- `200 OK` : Déconnexion réussie, retourne "Logged out"

### Point de Terminaison d'Échange de Session

#### `GET /_session_exchange`

Utilisé pour le partage de session cross-domain. Définit le cookie d'ID de session spécifié et redirige.

**Paramètres de Requête :**
- `id` (requis) : ID de session à définir

**Réponse :**
- `302 Redirect` : Redirige vers le chemin racine
- `400 Bad Request` : ID de session manquant

### Point de Terminaison de Vérification de Santé

#### `GET /health`

Point de terminaison de vérification de santé du service.

**Réponse :**
- `200 OK` : Le service est en bonne santé

### Point de Terminaison Racine

#### `GET /`

Chemin racine, affiche les informations du service.

**Pour la documentation API détaillée, voir : [docs/frFR/API.md](docs/frFR/API.md)**

## 🐳 Guide de Déploiement

### Déploiement Docker

#### Construire l'Image

```bash
cd codes
docker build -t stargate:latest .
```

#### Exécuter le Conteneur

```bash
docker run -d \
  --name stargate \
  -p 80:80 \
  -e AUTH_HOST=auth.example.com \
  -e PASSWORDS=plaintext:yourpassword \
  stargate:latest
```

### Déploiement Docker Compose

Le projet fournit un exemple de configuration `docker-compose.yml`, incluant le service Stargate et un service exemple whoami :

```bash
cd codes
docker-compose up -d
```

### Intégration Traefik

Configurer les labels Traefik dans `docker-compose.yml` :

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
      - "traefik.http.routers.your-service.middlewares=stargate"  # Utiliser le middleware Stargate

networks:
  traefik:
    external: true
```

### Recommandations de Production

1. **Utiliser HTTPS** : En production, assurez-vous que HTTPS est configuré via Traefik
2. **Utiliser des Algorithmes de Mot de Passe Forts** : Évitez `plaintext`, recommandez d'utiliser `bcrypt` ou `sha512`
3. **Définir le Domaine du Cookie** : Si vous devez partager des sessions entre plusieurs sous-domaines, définissez `COOKIE_DOMAIN`
4. **Gestion des Journaux** : Configurez une rotation de journaux et une surveillance appropriées
5. **Limites de Ressources** : Définissez des limites CPU et mémoire appropriées pour les conteneurs

**Pour le guide de déploiement détaillé, voir : [docs/frFR/DEPLOYMENT.md](docs/frFR/DEPLOYMENT.md)**

## 💻 Guide de Développement

### Structure du Projet

```
codes/
├── src/
│   ├── cmd/
│   │   └── stargate/          # Point d'entrée principal du programme
│   │       ├── main.go        # Point d'entrée du programme
│   │       ├── server.go      # Configuration du serveur
│   │       └── constants.go  # Définitions de constantes
│   ├── internal/
│   │   ├── auth/              # Logique d'authentification
│   │   ├── config/            # Gestion de la configuration
│   │   ├── handlers/          # Gestionnaires HTTP
│   │   ├── i18n/              # Internationalisation
│   │   ├── middleware/        # Middleware
│   │   ├── secure/            # Algorithmes de chiffrement de mot de passe
│   │   └── web/               # Templates Web et ressources statiques
│   ├── go.mod
│   └── go.sum
├── Dockerfile
├── docker-compose.yml
└── start-local.sh
```

### Développement Local

1. Installer les dépendances :
```bash
cd codes
go mod download
```

2. Exécuter les tests :
```bash
go test ./...
```

3. Démarrer le serveur de développement :
```bash
./start-local.sh
```

### Ajout de Nouveaux Algorithmes de Mot de Passe

1. Créer une nouvelle implémentation d'algorithme dans le répertoire `src/internal/secure/` :
```go
package secure

type NewAlgorithmResolver struct{}

func (r *NewAlgorithmResolver) Check(h string, password string) bool {
    // Implémenter la logique de vérification du mot de passe
    return false
}
```

2. Enregistrer l'algorithme dans `src/internal/config/validation.go` :
```go
SupportedAlgorithms = map[string]secure.HashResolver{
    // ...
    "newalgorithm": &secure.NewAlgorithmResolver{},
}
```

### Ajout de Support de Nouvelle Langue

1. Ajouter la constante de langue dans `src/internal/i18n/i18n.go` :
```go
const (
    LangEN Language = "en"
    LangZH Language = "zh"
    LangFR Language = "fr"  // Nouveau
)
```

2. Ajouter le mapping de traduction :
```go
var translations = map[Language]map[string]string{
    // ...
    LangFR: {
        "error.auth_required": "Authentification requise",
        // ...
    },
}
```

3. Ajouter l'option de langue dans `src/internal/config/config.go` :
```go
Language = EnvVariable{
    PossibleValues: []string{"en", "zh", "fr"},  // Ajouter la nouvelle langue
}
```

## 📝 Licence

Ce projet est sous licence Apache License 2.0. Voir le fichier [LICENSE](codes/LICENSE) pour plus de détails.

## 🤝 Contribution

Nous accueillons les contributions ! Que ce soit :
- 🐛 Rapports de bugs
- 💡 Suggestions de fonctionnalités
- 📝 Améliorations de la documentation
- 🔧 Contributions de code

N'hésitez pas à ouvrir un Issue ou à soumettre une Pull Request. Chaque contribution rend Stargate meilleur !

---

## ⚠️ Liste de Vérification de Production

Avant de déployer en production, assurez-vous d'avoir complété ces meilleures pratiques de sécurité :

- ✅ **Utiliser des Mots de Passe Forts** : Évitez `plaintext`, utilisez `bcrypt` ou `sha512` pour le hachage des mots de passe
- ✅ **Activer HTTPS** : Configurez HTTPS via Traefik ou votre proxy inverse
- ✅ **Définir le Domaine du Cookie** : Configurez `COOKIE_DOMAIN` pour une gestion de session appropriée entre sous-domaines
- ✅ **Surveiller et Journaliser** : Configurez une journalisation et une surveillance appropriées pour votre déploiement
- ✅ **Mises à Jour Régulières** : Gardez Stargate à jour vers la dernière version pour les correctifs de sécurité
