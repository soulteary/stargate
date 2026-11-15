# Index de la Documentation

> **Traduction abrégée** : les guides en français sont des traductions abrégées et ne reflètent pas toutes les modifications actuelles. La [documentation anglaise](../enUS/README.md) est complète et fait foi.

Bienvenue dans la documentation du service Stargate Forward Auth.

## 🌐 Documentation Multilingue

- [English](../enUS/README.md) | [中文](../zhCN/README.md) | [Français](README.md) | [Italiano](../itIT/README.md) | [日本語](../jaJP/README.md) | [Deutsch](../deDE/README.md) | [한국어](../koKR/README.md)

## 📚 Liste des Documents

### Documents Principaux

- **[README.md](../../README.frFR.md)** - Vue d'ensemble du projet et guide de démarrage rapide
- **[ARCHITECTURE.md](ARCHITECTURE.md)** - Architecture technique et décisions de conception

### Documents Détaillés

- **[API.md](API.md)** - Documentation complète des points de terminaison API
  - Points de terminaison de vérification d'authentification
  - Points de terminaison de connexion et de déconnexion
  - Points de terminaison d'échange de session
  - Points de terminaison de vérification de santé
  - Formats de réponse d'erreur
  - Exemples de flux d'authentification

- **[CONFIG.md](CONFIG.md)** - Référence de configuration
  - Méthodes de configuration
  - Éléments de configuration requis
  - Éléments de configuration optionnels
  - Détails de configuration du mot de passe
  - Exemples de configuration
  - Meilleures pratiques de configuration

- **[DEPLOYMENT.md](DEPLOYMENT.md)** - Guide de déploiement
  - Déploiement Docker
  - Déploiement Docker Compose
  - Intégration Traefik
  - Déploiement en production
  - Surveillance et maintenance
  - Dépannage

## 🚀 Navigation Rapide

### Pour Commencer

1. Lisez [README.frFR.md](../../README.frFR.md) pour comprendre le projet
2. Consultez la section [Démarrage Rapide](../../README.frFR.md#-démarrage-rapide)
3. Référez-vous à [Configuration](../../README.frFR.md#%EF%B8%8F-configuration-de-base) pour configurer le service

### Développeurs

1. Lisez [ARCHITECTURE.md](ARCHITECTURE.md) pour comprendre l'architecture
2. Consultez [API.md](API.md) pour comprendre les interfaces API
3. Référez-vous au [Guide de Développement](CONTRIBUTING.md) pour le développement

### Opérations

1. Lisez [DEPLOYMENT.md](DEPLOYMENT.md) pour comprendre les méthodes de déploiement
2. Consultez [CONFIG.md](CONFIG.md) pour comprendre les options de configuration
3. Référez-vous au [Dépannage](DEPLOYMENT.md#dépannage) pour résoudre les problèmes

## 📖 Structure des Documents

Traductions abrégées dans ce répertoire :

- [API.md](API.md)
- [ARCHITECTURE.md](ARCHITECTURE.md)
- [CONFIG.md](CONFIG.md)
- [CONTRIBUTING.md](CONTRIBUTING.md)
- [DEPLOYMENT.md](DEPLOYMENT.md)
- [MIGRATION_V1.md](MIGRATION_V1.md)
- [SECURITY.md](SECURITY.md)

Disponible uniquement en anglais :

- [DIAGNOSTICS.md](../enUS/DIAGNOSTICS.md) - Deployment diagnostics
- [AUTHENTICATION_UI.md](../enUS/AUTHENTICATION_UI.md) - Authentication pages
- [DINGTALK_USERID_ONLY.md](../enUS/DINGTALK_USERID_ONLY.md) - Opt-in userid-only work-IM login
- [PERFORMANCE.md](../enUS/PERFORMANCE.md) - Redis rate-limit measurements
- [VERIFIED_EMAIL.md](../enUS/VERIFIED_EMAIL.md) - Verified email assertions

## 🔍 Recherche par Sujet

### Configuration

- Configuration des variables d'environnement : [CONFIG.md](CONFIG.md)
- Configuration du mot de passe : [CONFIG.md#configuration-du-mot-de-passe](CONFIG.md#configuration-du-mot-de-passe)
- Exemples de configuration : [CONFIG.md#exemples-de-configuration](CONFIG.md#exemples-de-configuration)

### API

- Liste des points de terminaison API : [API.md](API.md)
- Flux d'authentification : [API.md#exemples-de-flux-dauthentification](API.md#exemples-de-flux-dauthentification)
- Gestion des erreurs : [API.md#format-de-réponse-derreur](API.md#format-de-réponse-derreur)

### Déploiement

- Déploiement Docker : [DEPLOYMENT.md#déploiement-docker](DEPLOYMENT.md#déploiement-docker)
- Intégration Traefik : [DEPLOYMENT.md#intégration-traefik](DEPLOYMENT.md#intégration-traefik)
- Environnement de production : [DEPLOYMENT.md#déploiement-en-production](DEPLOYMENT.md#déploiement-en-production)

### Architecture

- Pile technologique : [ARCHITECTURE.md#pile-technologique](ARCHITECTURE.md#pile-technologique)
- Structure du projet : [ARCHITECTURE.md#structure-du-projet](ARCHITECTURE.md#structure-du-projet)
- Composants principaux : [ARCHITECTURE.md#composants-principaux](ARCHITECTURE.md#composants-principaux)

## 💡 Recommandations d'Utilisation

1. **Utilisateurs pour la première fois** : Commencez par [README.frFR.md](../../README.frFR.md) et suivez le guide de démarrage rapide
2. **Configurer le service** : Référez-vous à [CONFIG.md](CONFIG.md) pour comprendre toutes les options de configuration
3. **Intégrer Traefik** : Consultez la section d'intégration Traefik dans [DEPLOYMENT.md](DEPLOYMENT.md)
4. **Développer des extensions** : Lisez [ARCHITECTURE.md](ARCHITECTURE.md) pour comprendre la conception de l'architecture
5. **Dépannage** : Consultez [DEPLOYMENT.md#dépannage](DEPLOYMENT.md#dépannage)

## 📝 Mises à Jour des Documents

La documentation est mise à jour en continu au fur et à mesure de l'évolution du projet. Si vous trouvez des erreurs ou avez besoin d'ajouts, veuillez soumettre un Issue ou une Pull Request.

## 🤝 Contribution

Les améliorations de la documentation sont les bienvenues :

1. Trouvez des erreurs ou des domaines à améliorer
2. Soumettez un Issue décrivant le problème
3. Ou soumettez directement une Pull Request
