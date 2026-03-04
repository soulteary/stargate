# Stargate - Forward Auth Service

[![License](https://img.shields.io/badge/license-Apache%202.0-blue.svg)](LICENSE)
[![Go Version](https://img.shields.io/badge/go-1.26+-blue.svg)](https://golang.org)
[![codecov](https://codecov.io/gh/soulteary/stargate/branch/main/graph/badge.svg)](https://codecov.io/gh/soulteary/stargate)
[![Go Report Card](https://goreportcard.com/badge/github.com/soulteary/stargate)](https://goreportcard.com/report/github.com/soulteary/stargate)

> **🚀 Votre Passerelle vers des Microservices Sécurisés**

![Stargate](.github/assets/banner.jpg)

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
- **Authentification Entreprise** : Intégration avec Warden (liste blanche d'utilisateurs) et Herald (OTP/codes de vérification) pour une authentification de niveau production

## ✨ Fonctionnalités

### 🔐 Sécurité de Niveau Entreprise

- **Plusieurs Algorithmes de Chiffrement de Mot de Passe** : Choisissez parmi plaintext (test), bcrypt, MD5, SHA512, et plus encore
- **Gestion de Session Sécurisée** : Sessions basées sur Cookie avec domaine et expiration personnalisables
- **Authentification Flexible** : Support pour l'authentification basée sur mot de passe et basée sur session
- **Support OTP/Code de Vérification** : Intégration avec le service Herald pour les codes de vérification SMS/Email
- **Gestion de Liste Blanche d'Utilisateurs** : Intégration avec le service Warden pour le contrôle d'accès utilisateur

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

## 📋 Table des Matières

- [Démarrage Rapide](#-démarrage-rapide)
- [Documentation](#-documentation)
- [Configuration de Base](#-configuration-de-base)
- [Intégration de Services Optionnels](#-intégration-de-services-optionnels)
- [Liste de Vérification de Production](#-liste-de-vérification-de-production)
- [Licence](#-licence)

## 🚀 Démarrage Rapide

Mettez Stargate en route en **moins de 2 minutes** !

### Utilisation de Docker Compose (Recommandé)

**Étape 1 :** Cloner le dépôt
```bash
git clone <repository-url>
cd stargate
```

**Étape 2 :** Configurer votre authentification (modifier `docker-compose.yml`)

**Option A : Authentification par Mot de Passe (Simple)**
```yaml
services:
  stargate:
    environment:
      - AUTH_HOST=auth.example.com
      - PASSWORDS=plaintext:yourpassword1|yourpassword2
```

**Option B : Authentification OTP Warden + Herald (Production)**
```yaml
services:
  stargate:
    environment:
      - AUTH_HOST=auth.example.com
      - WARDEN_ENABLED=true
      - WARDEN_URL=http://warden:8080
      - WARDEN_API_KEY=your-warden-api-key
      - HERALD_ENABLED=true
      - HERALD_URL=http://herald:8080
      - HERALD_HMAC_SECRET=your-herald-hmac-secret
```

**Étape 3 :** Démarrer le service
```bash
docker-compose up -d
```

**C'est tout !** Votre service d'authentification est maintenant en cours d'exécution. 🎉

### Développement Local

Pour le développement local, assurez-vous que Go 1.26+ est installé, puis :

```bash
chmod +x start-local.sh
./start-local.sh
```

Accédez à la page de connexion à `http://localhost:8080/_login?callback=localhost`

## 📚 Documentation

Une documentation complète est disponible pour vous aider à tirer le meilleur parti de Stargate :

### Documents Principaux

- 📐 **[Document d'Architecture](docs/frFR/ARCHITECTURE.md)** - Plongée approfondie dans l'architecture technique et les décisions de conception
- 🔌 **[Document API](docs/frFR/API.md)** - Référence complète des points de terminaison API avec exemples
- ⚙️ **[Référence de Configuration](docs/frFR/CONFIG.md)** - Options de configuration détaillées et meilleures pratiques
- 🚀 **[Guide de Déploiement](docs/frFR/DEPLOYMENT.md)** - Stratégies de déploiement en production et recommandations

### Référence Rapide

- **Points de Terminaison API** : `GET /_auth` (vérification d’auth), `GET /_login` (page de connexion), `POST /_login` (connexion), `POST /_send_verify_code` (envoyer OTP), `GET /_logout` (déconnexion), `GET /_session_exchange` (cross-domain), `GET /totp/enroll`, `POST /totp/enroll/confirm`, `GET /totp/revoke`, `POST /totp/revoke` (TOTP si Herald activé), `GET /health` (santé), `GET /metrics` (Prometheus)
- **Déploiement** : Docker Compose recommandé pour un démarrage rapide. Voir [DEPLOYMENT.md](docs/frFR/DEPLOYMENT.md) pour le déploiement en production.
- **Développement** : Pour la documentation liée au développement, voir [ARCHITECTURE.md](docs/frFR/ARCHITECTURE.md)

## ⚙️ Configuration de Base

Stargate utilise des variables d'environnement pour la configuration. Voici les paramètres les plus courants :

### Configuration Requise

- **`AUTH_HOST`** : Nom d'hôte du service d'authentification (par exemple, `auth.example.com`)
- **`PASSWORDS`** : Configuration du mot de passe, format : `algorithm:password1|password2|password3`

### Exemples de Configuration Courants

```bash
# Authentification par mot de passe simple
AUTH_HOST=auth.example.com
PASSWORDS=plaintext:test123|admin456

# Utilisation du hash BCrypt
PASSWORDS=bcrypt:$2a$10$N9qo8uLOickgx2ZMRZoMyeIjZAgcfl7p92ldGxad68LJZdL17lhWy

# Partage de session cross-domain
COOKIE_DOMAIN=.example.com

# Personnaliser la page de connexion
LOGIN_PAGE_TITLE=Mon Service d'Authentification
LANGUAGE=fr  # ou 'en'
```

**Algorithmes de mot de passe supportés :** `plaintext` (test uniquement), `bcrypt`, `md5`, `sha512`

**Pour la référence de configuration complète, voir : [docs/frFR/CONFIG.md](docs/frFR/CONFIG.md)**

## 🔗 Intégration de Services Optionnels

Stargate peut être utilisé complètement de manière indépendante, ou peut optionnellement s'intégrer avec les services suivants :

### Intégration Warden (Liste Blanche d'Utilisateurs)

Fournit la gestion de liste blanche d'utilisateurs et les informations utilisateur. Lorsqu'il est activé, Stargate interroge Warden pour vérifier si un utilisateur est dans la liste autorisée.

```bash
WARDEN_ENABLED=true
WARDEN_URL=http://warden:8080
WARDEN_API_KEY=your-api-key
```

### Intégration Herald (OTP/Codes de Vérification)

Fournit les services OTP/code de vérification. Lorsqu'il est activé, Stargate appelle Herald pour créer, envoyer et vérifier les codes de vérification (SMS/Email).

```bash
HERALD_ENABLED=true
HERALD_URL=http://herald:8080
HERALD_HMAC_SECRET=your-hmac-secret  # Production
# ou
HERALD_API_KEY=your-api-key  # Développement
```

**Note** : Les deux intégrations sont optionnelles. Stargate peut être utilisé indépendamment avec l'authentification par mot de passe.

**Guide d'intégration complet, voir : [docs/frFR/ARCHITECTURE.md](docs/frFR/ARCHITECTURE.md)**

## ⚠️ Liste de Vérification de Production

Avant de déployer en production :

- ✅ Utiliser des algorithmes de mot de passe forts (`bcrypt` ou `sha512`, éviter `plaintext`)
- ✅ Activer HTTPS via Traefik ou votre proxy inverse
- ✅ Définir `COOKIE_DOMAIN` pour une gestion de session appropriée entre sous-domaines
- ✅ Pour des fonctionnalités avancées, intégrer optionnellement Warden + Herald pour l'authentification OTP
- ✅ Utiliser des signatures HMAC ou mTLS pour la communication Stargate ↔ Herald/Warden
- ✅ Configurer une journalisation et une surveillance appropriées
- ✅ Maintenir Stargate à jour vers la dernière version

## 🎯 Principes de Conception

Stargate est conçu pour être utilisé de manière indépendante :

- **Utilisation Autonome** : Stargate peut fonctionner indépendamment en utilisant le mode d'authentification par mot de passe, sans aucune dépendance externe
- **Intégration Optionnelle** : Peut optionnellement s'intégrer avec Warden (liste blanche d'utilisateurs) et Herald (OTP/codes de vérification)
- **Haute Performance** : Le chemin principal forwardAuth ne vérifie que la session, garantissant une réponse rapide
- **Flexibilité** : Supporte plusieurs modes d'authentification, choisissez selon vos besoins

## 📝 Licence

Ce projet est sous licence Apache License 2.0. Voir le fichier [LICENSE](LICENSE) pour plus de détails.

## 🤝 Contribution

Nous accueillons les contributions ! Que ce soit :
- 🐛 Rapports de bugs
- 💡 Suggestions de fonctionnalités
- 📝 Améliorations de la documentation
- 🔧 Contributions de code

N'hésitez pas à ouvrir un Issue ou à soumettre une Pull Request.
