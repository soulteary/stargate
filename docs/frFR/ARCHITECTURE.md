# Document d'Architecture Stargate

Ce document décrit l'architecture technique et les décisions de conception du projet Stargate.

## Pile Technologique

- **Langage** : Go 1.25
- **Framework Web** : [Fiber v2.52.10](https://github.com/gofiber/fiber)
- **Moteur de Template** : [Fiber Template v1.7.5](https://github.com/gofiber/template)
- **Gestion de Session** : Middleware de Session Fiber
- **Journalisation** : [Logrus v1.9.3](https://github.com/sirupsen/logrus)
- **Sortie Terminal** : [Pterm v0.12.82](https://github.com/pterm/pterm)
- **Framework de Test** : [Testza v0.5.2](https://github.com/MarvinJWendt/testza)

## Structure du Projet

```
codes/src/
├── cmd/stargate/          # Point d'entrée de l'application
│   ├── main.go            # Fonction principale, initialise la configuration et démarre le serveur
│   ├── server.go          # Configuration du serveur et configuration des routes
│   └── constants.go       # Constantes de route et de configuration
│
├── internal/              # Packages internes (non exposés en externe)
│   ├── auth/              # Logique d'authentification
│   │   ├── auth.go        # Fonctionnalité principale d'authentification
│   │   └── auth_test.go   # Tests d'authentification
│   │
│   ├── config/            # Gestion de la configuration
│   │   ├── config.go      # Définitions et initialisation des variables de configuration
│   │   ├── validation.go  # Logique de validation de la configuration
│   │   └── config_test.go # Tests de configuration
│   │
│   ├── handlers/          # Gestionnaires de requêtes HTTP
│   │   ├── check.go       # Gestionnaire de vérification d'authentification
│   │   ├── login.go       # Gestionnaire de connexion
│   │   ├── logout.go      # Gestionnaire de déconnexion
│   │   ├── session_share.go # Gestionnaire de partage de session
│   │   ├── health.go      # Gestionnaire de vérification de santé
│   │   ├── index.go       # Gestionnaire du chemin racine
│   │   ├── utils.go       # Fonctions utilitaires des gestionnaires
│   │   └── handlers_test.go # Tests des gestionnaires
│   │
│   ├── i18n/              # Support d'internationalisation
│   │   └── i18n.go        # Traductions multilingues
│   │
│   ├── middleware/        # Middleware HTTP
│   │   └── log.go         # Middleware de journalisation
│   │
│   ├── secure/            # Algorithmes de chiffrement de mot de passe
│   │   ├── interface.go   # Interface d'algorithme de chiffrement
│   │   ├── plaintext.go   # Mot de passe en texte brut (test uniquement)
│   │   ├── bcrypt.go      # Algorithme BCrypt
│   │   ├── md5.go         # Algorithme MD5
│   │   ├── sha512.go      # Algorithme SHA512
│   │   └── secure_test.go # Tests d'algorithmes de chiffrement
│   │
│   └── web/               # Ressources Web
│       └── templates/     # Templates HTML
│           ├── login.html # Template de page de connexion
│           └── assets/   # Ressources statiques
│               └── favicon.ico
```

## Composants Principaux

### 1. Système d'Authentification (`internal/auth`)

Le système d'authentification est responsable de :
- Vérification du mot de passe (supporte plusieurs algorithmes de chiffrement)
- Gestion de session (créer, vérifier, détruire)
- Vérification du statut d'authentification

**Fonctions Clés :**
- `CheckPassword(password string) bool` : Vérifie le mot de passe
- `Authenticate(session *session.Session) error` : Marque la session comme authentifiée
- `IsAuthenticated(session *session.Session) bool` : Vérifie si la session est authentifiée
- `Unauthenticate(session *session.Session) error` : Détruit la session

### 2. Système de Configuration (`internal/config`)

Le système de configuration fournit :
- Gestion des variables d'environnement
- Validation de la configuration
- Support des valeurs par défaut

**Variables de Configuration :**
- `AUTH_HOST` : Nom d'hôte d'authentification (requis)
- `PASSWORDS` : Configuration du mot de passe (liste algorithme:mot de passe) (requis)
- `DEBUG` : Mode débogage (par défaut : false)
- `LANGUAGE` : Langue de l'interface (par défaut : en, supporte en/zh/fr/it/ja/de/ko)
- `COOKIE_DOMAIN` : Domaine du cookie (optionnel, pour le partage de session cross-domain)
- `LOGIN_PAGE_TITLE` : Titre de la page de connexion (par défaut : Stargate - Login)
- `LOGIN_PAGE_FOOTER_TEXT` : Texte du pied de page de connexion (par défaut : Copyright © 2024 - Stargate)
- `USER_HEADER_NAME` : Nom de l'en-tête utilisateur défini après authentification réussie (par défaut : X-Forwarded-User)
- `PORT` : Port d'écoute du service (développement local uniquement, par défaut : 80)

### 3. Gestionnaires de Requêtes (`internal/handlers`)

Les gestionnaires sont responsables du traitement des requêtes HTTP :

- **CheckRoute** : Vérification d'authentification Traefik Forward Auth
- **LoginRoute/LoginAPI** : Page de connexion et traitement de connexion
- **LogoutRoute** : Traitement de déconnexion
- **SessionShareRoute** : Partage de session cross-domain
- **HealthRoute** : Vérification de santé
- **IndexRoute** : Traitement du chemin racine

### 4. Chiffrement de Mot de Passe (`internal/secure`)

Supporte plusieurs algorithmes de chiffrement de mot de passe :
- `plaintext` : Texte brut (test uniquement)
- `bcrypt` : Hash BCrypt
- `md5` : Hash MD5
- `sha512` : Hash SHA512

Tous les algorithmes implémentent l'interface `HashResolver` :
```go
type HashResolver interface {
    Check(h string, password string) bool
}
```

## Flux de Travail

### Flux d'Authentification

1. **L'utilisateur accède à une ressource protégée**
   - Traefik intercepte la requête
   - Transmet à l'endpoint Stargate `/_auth`

2. **Stargate vérifie l'authentification**
   - Vérifie d'abord l'en-tête `Stargate-Password` (authentification API)
   - Si l'authentification par en-tête échoue, vérifie le cookie `stargate_session_id` (authentification Web)

3. **Authentification réussie**
   - Définit l'en-tête `X-Forwarded-User` (ou le nom d'en-tête utilisateur configuré) avec la valeur "authenticated"
   - Retourne 200 OK
   - Traefik permet à la requête de continuer

4. **Authentification échouée**
   - Requêtes HTML : Redirige vers la page de connexion (`/_login?callback=<originalURL>`)
   - Requêtes API (JSON/XML) : Retourne 401 Unauthorized

### Flux de Connexion

1. **L'utilisateur accède à la page de connexion**
   - `GET /_login?callback=<url>`
   - Si déjà connecté, redirige vers l'endpoint d'échange de session
   - Si le domaine diffère, stocke le callback dans le cookie (`stargate_callback`)

2. **Soumission du formulaire de connexion**
   - `POST /_login` avec mot de passe
   - Vérifie le mot de passe
   - Crée une session et définit le cookie
   - **Priorité de récupération du callback** :
     1. Depuis le cookie (si précédemment défini)
     2. Depuis les données du formulaire
     3. Depuis les paramètres de requête
     4. Si aucun des éléments ci-dessus, et le domaine d'origine diffère du domaine du service d'authentification, utiliser le domaine d'origine comme callback

3. **Échange de session**
   - Si le callback existe, redirige vers `{callback}/_session_exchange?id=<session_id>`
   - `GET /_session_exchange?id=<session_id>`
   - Définit le cookie de session (si `COOKIE_DOMAIN` est configuré, définit au domaine spécifié)
   - Redirige vers le chemin racine `/`

## Considérations de Sécurité

### Sécurité de Session

- Les cookies utilisent le flag `HttpOnly` pour prévenir les attaques XSS
- Les cookies utilisent `SameSite=Lax` pour prévenir les attaques CSRF
- Le chemin du cookie est défini à `/`, permettant l'utilisation sur tout le domaine
- Temps d'expiration de session : 24 heures (`config.SessionExpiration`)
- Supporte le domaine de cookie personnalisé (pour les scénarios cross-domain)
- Les IDs de session sont générés en utilisant UUID pour assurer l'unicité et la sécurité

### Sécurité du Mot de Passe

- Supporte plusieurs algorithmes de chiffrement (recommandé d'utiliser bcrypt ou sha512)
- Configuration du mot de passe transmise via variables d'environnement, non stockée dans le code
- Normalisation du mot de passe lors de la vérification (supprimer les espaces, convertir en majuscules)

### Sécurité des Requêtes

- L'endpoint de vérification d'authentification supporte deux méthodes d'authentification :
  - Authentification par en-tête (`Stargate-Password`) : Pour les requêtes API
  - Authentification par cookie : Pour les requêtes Web
- Distingue entre les requêtes HTML et API, retourne des réponses appropriées

## Extensibilité

### Ajout de Nouveaux Algorithmes de Mot de Passe

1. Créer une nouvelle implémentation d'algorithme dans `internal/secure/`
2. Implémenter l'interface `HashResolver`
3. Enregistrer l'algorithme dans `config/validation.go`

### Ajout de Nouvelles Langues

1. Ajouter la constante de langue dans `internal/i18n/i18n.go`
2. Ajouter les mappings de traduction
3. Ajouter l'option de langue dans la configuration

### Personnalisation de la Page de Connexion

Modifier le fichier template `internal/web/templates/login.html`.

## Optimisation des Performances

- Utilise le framework Fiber, basé sur fasthttp, excellentes performances
- Sessions stockées en mémoire pour un accès rapide
- Ressources statiques servies via le service de fichiers statiques Fiber
- Supporte le mode débogage, peut être désactivé en production

## Architecture de Déploiement

### Déploiement Docker

- Build multi-étapes pour réduire la taille de l'image
- Utilise `golang:1.25-alpine` comme étape de build
- Utilise l'image de base `scratch` comme étape d'exécution pour minimiser les risques de sécurité
- Fichiers template copiés de `src/internal/web/templates` vers `/app/web/templates` dans l'image
- Utilise la source miroir chinoise (`GOPROXY=https://goproxy.cn`) pour accélérer les téléchargements de dépendances
- Utilise `-ldflags "-s -w"` lors de la compilation pour réduire la taille du binaire
- L'application trouve automatiquement les chemins de template (supporte `./internal/web/templates` pour le développement local et `./web/templates` pour la production)

### Intégration Traefik

- Intégré via le middleware Forward Auth
- Supporte HTTP et HTTPS
- Supporte plusieurs domaines et règles de chemin

## Journalisation et Surveillance

- Utilise Logrus pour la journalisation
- Supporte le mode débogage (DEBUG=true)
- Toutes les opérations critiques sont journalisées
- Endpoint de vérification de santé disponible pour la surveillance

## Tests

- Les tests unitaires couvrent la fonctionnalité principale
- Fichiers de test situés dans les fichiers `*_test.go` de chaque package
- Utilise `testza` pour les assertions
- Couverture de test inclut :
  - Logique d'authentification (`internal/auth/auth_test.go`)
  - Validation de configuration (`internal/config/config_test.go`)
  - Algorithmes de chiffrement de mot de passe (`internal/secure/secure_test.go`)
  - Gestionnaires HTTP (`internal/handlers/handlers_test.go`)

## Améliorations Futures

- [ ] Supporter plus d'algorithmes de chiffrement de mot de passe
- [ ] Supporter OAuth2/OpenID Connect
- [ ] Supporter la gestion multi-utilisateur et de rôles
- [ ] Ajouter une interface d'administration
- [ ] Supporter le stockage de session externe (Redis, etc.)
- [ ] Ajouter l'export de métriques Prometheus
- [ ] Supporter les fichiers de configuration (YAML/JSON)
