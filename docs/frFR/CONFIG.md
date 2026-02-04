# Référence de Configuration

Ce document détaille toutes les options de configuration pour Stargate.

## Table des Matières

- [Méthodes de Configuration](#méthodes-de-configuration)
- [Configuration Requise](#configuration-requise)
- [Configuration Optionnelle](#configuration-optionnelle)
- [Configuration du Mot de Passe](#configuration-du-mot-de-passe)
- [Exemples de Configuration](#exemples-de-configuration)

## Méthodes de Configuration

Stargate est configuré via des variables d'environnement. Tous les éléments de configuration sont définis via des variables d'environnement, aucun fichier de configuration n'est nécessaire.

### Définition des Variables d'Environnement

**Linux/macOS :**

```bash
export AUTH_HOST=auth.example.com
export PASSWORDS=plaintext:yourpassword
```

**Docker :**

```bash
docker run -e AUTH_HOST=auth.example.com -e PASSWORDS=plaintext:yourpassword stargate:latest
```

**Docker Compose :**

```yaml
services:
  stargate:
    environment:
      - AUTH_HOST=auth.example.com
      - PASSWORDS=plaintext:yourpassword
```

## Configuration Requise

Les éléments de configuration suivants sont requis. Le fait de ne pas les définir empêchera le service de démarrer.

### `AUTH_HOST`

Nom d'hôte du service d'authentification.

| Attribut | Valeur |
|----------|--------|
| **Type** | String |
| **Requis** | Oui |
| **Par Défaut** | Aucun |
| **Exemple** | `auth.example.com` |

**Description :**

- Utilisé pour construire les URLs de callback de connexion
- Généralement défini au nom d'hôte du service Stargate
- Supporte le wildcard `*` (non recommandé pour la production)

**Exemple :**

```bash
AUTH_HOST=auth.example.com
```

### `PASSWORDS`

Configuration du mot de passe, spécifiant l'algorithme de chiffrement du mot de passe et la liste des mots de passe.

| Attribut | Valeur |
|----------|--------|
| **Type** | String |
| **Requis** | Oui |
| **Par Défaut** | Aucun |
| **Format** | `algorithm:password1|password2|password3` |

**Description :**

- Format : `algorithm:password1|password2|password3`
- Supporte plusieurs mots de passe, séparés par `|`
- Tout mot de passe qui passe la vérification permet la connexion
- Algorithmes supportés voir la section [Configuration du Mot de Passe](#configuration-du-mot-de-passe)

**Exemples :**

```bash
# Mot de passe en texte brut unique
PASSWORDS=plaintext:test123

# Plusieurs mots de passe en texte brut
PASSWORDS=plaintext:test123|admin456|user789

# Hash BCrypt
PASSWORDS=bcrypt:$2a$10$N9qo8uLOickgx2ZMRZoMyeIjZAgcfl7p92ldGxad68LJZdL17lhWy

# Hash SHA512
PASSWORDS=sha512:5e884898da28047151d0e56f8dc6292773603d0d6aabbdd62a11ef721d1542d8
```

## Configuration Optionnelle

Les éléments de configuration suivants sont optionnels. Les valeurs par défaut sont utilisées s'ils ne sont pas définis.

### `DEBUG`

Activer le mode débogage.

| Attribut | Valeur |
|----------|--------|
| **Type** | Boolean |
| **Requis** | Non |
| **Par Défaut** | `false` |
| **Valeurs Possibles** | `true`, `false` |

**Description :**

- Lorsqu'activé, le niveau de journalisation est défini à `DEBUG`
- Affiche des informations de débogage plus détaillées
- Recommandé de définir à `false` en production

**Exemple :**

```bash
DEBUG=true
```

### `LANGUAGE`

Langue de l'interface.

| Attribut | Valeur |
|----------|--------|
| **Type** | String |
| **Requis** | Non |
| **Par Défaut** | `en` |
| **Valeurs Possibles** | `en` (Anglais), `zh` (Chinois), `fr` (Français), `it` (Italien), `ja` (Japonais), `de` (Allemand), `ko` (Coréen) |

**Description :**

- Affecte la langue des messages d'erreur et du texte de l'interface
- Insensible à la casse (`EN`, `en`, `En` fonctionnent tous)

**Exemple :**

```bash
LANGUAGE=fr
```

### `LOGIN_PAGE_TITLE`

Titre de la page de connexion.

| Attribut | Valeur |
|----------|--------|
| **Type** | String |
| **Requis** | Non |
| **Par Défaut** | `Stargate - Login` |

**Description :**

- Affiché à la position du titre de la page de connexion
- Supporte les balises HTML (non recommandé)

**Exemple :**

```bash
LOGIN_PAGE_TITLE=Mon Service d'Authentification
```

### `LOGIN_PAGE_FOOTER_TEXT`

Texte du pied de page de la page de connexion.

| Attribut | Valeur |
|----------|--------|
| **Type** | String |
| **Requis** | Non |
| **Par Défaut** | `Copyright © 2024 - Stargate` |

**Description :**

- Affiché à la position du pied de page de la page de connexion
- Supporte les balises HTML (non recommandé)

**Exemple :**

```bash
LOGIN_PAGE_FOOTER_TEXT=© 2024 Ma Société
```

### `USER_HEADER_NAME`

Nom de l'en-tête utilisateur défini après authentification réussie.

| Attribut | Valeur |
|----------|--------|
| **Type** | String |
| **Requis** | Non |
| **Par Défaut** | `X-Forwarded-User` |

**Description :**

- Après authentification réussie, Stargate définit cet en-tête dans la réponse
- La valeur de l'en-tête est `authenticated`
- Les services backend peuvent déterminer si un utilisateur est authentifié via cet en-tête
- Doit être une chaîne non vide

**Exemple :**

```bash
USER_HEADER_NAME=X-Authenticated-User
```

### `COOKIE_DOMAIN`

Domaine du cookie, utilisé pour le partage de session cross-domain.

| Attribut | Valeur |
|----------|--------|
| **Type** | String |
| **Requis** | Non |
| **Par Défaut** | Vide (non défini) |

**Description :**

- Si défini, les cookies de session seront définis au domaine spécifié
- Supporte le partage de session cross-sous-domaine
- Format : `.example.com` (notez le point initial)
- Lorsqu'il est défini à vide, les cookies ne sont valides que pour le domaine actuel

**Exemple :**

```bash
# Permettre le partage de session sur tous les sous-domaines *.example.com
COOKIE_DOMAIN=.example.com
```

**Scénario de Partage de Session Cross-Domain :**

Supposons les domaines suivants :
- `auth.example.com` - Service d'authentification
- `app1.example.com` - Application 1
- `app2.example.com` - Application 2

Après avoir défini `COOKIE_DOMAIN=.example.com` :
1. L'utilisateur se connecte à `auth.example.com`
2. Le cookie de session est défini au domaine `.example.com`
3. L'utilisateur peut utiliser la même session sur `app1.example.com` et `app2.example.com`

### `PORT`

Port d'écoute du service (développement local uniquement). Géré par le package config avec les autres options d'environnement.

| Attribut | Valeur |
|----------|--------|
| **Type** | String |
| **Requis** | Non |
| **Par Défaut** | Vide (si vide, le serveur utilise le port par défaut `:80`) |

**Description :**

- Uniquement pour l'environnement de développement local
- Généralement non nécessaire dans les conteneurs Docker (utilise le port par défaut 80)
- Format : numéro de port (par ex., `8080`) ou `:port` (par ex., `:8080`)

**Exemple :**

```bash
PORT=8080
```

## Configuration du Mot de Passe

Stargate supporte plusieurs algorithmes de chiffrement de mot de passe. Format de configuration du mot de passe : `algorithm:password1|password2|password3`

### Algorithmes Supportés

#### `plaintext` - Mot de Passe en Texte Brut

**Description :**

- Stocké en texte brut, aucun chiffrement
- **Environnement de test uniquement**
- Fortement non recommandé pour la production

**Exemple :**

```bash
PASSWORDS=plaintext:test123|admin456
```

#### `bcrypt` - Hash BCrypt

**Description :**

- Utilise l'algorithme BCrypt pour le hachage
- Haute sécurité, recommandé pour la production
- Le mot de passe doit utiliser la valeur de hash BCrypt

**Générer un Hash BCrypt :**

```bash
# Utilisant Go
go run -c 'golang.org/x/crypto/bcrypt' <<< 'password'

# Utilisant des outils en ligne ou d'autres outils
```

**Exemple :**

```bash
PASSWORDS=bcrypt:$2a$10$N9qo8uLOickgx2ZMRZoMyeIjZAgcfl7p92ldGxad68LJZdL17lhWy
```

#### `md5` - Hash MD5

**Description :**

- Utilise l'algorithme MD5 pour le hachage
- Sécurité plus faible, non recommandé pour la production
- Le mot de passe doit utiliser la valeur de hash MD5 (chaîne hexadécimale de 32 caractères)

**Générer un Hash MD5 :**

```bash
# Linux/macOS
echo -n "password" | md5sum

# Ou utiliser des outils en ligne
```

**Exemple :**

```bash
PASSWORDS=md5:5e884898da28047151d0e56f8dc6292773603d0d6aabbdd62a11ef721d1542d8
```

#### `sha512` - Hash SHA512

**Description :**

- Utilise l'algorithme SHA512 pour le hachage
- Haute sécurité, recommandé pour la production
- Le mot de passe doit utiliser la valeur de hash SHA512 (chaîne hexadécimale de 128 caractères)

**Générer un Hash SHA512 :**

```bash
# Linux/macOS
echo -n "password" | shasum -a 512

# Ou utiliser des outils en ligne
```

**Exemple :**

```bash
PASSWORDS=sha512:5e884898da28047151d0e56f8dc6292773603d0d6aabbdd62a11ef721d1542d8
```

### Règles de Vérification du Mot de Passe

1. **Normalisation du Mot de Passe** : Les espaces sont supprimés et convertis en majuscules avant vérification
2. **Support de Plusieurs Mots de Passe** : Plusieurs mots de passe peuvent être configurés, tout mot de passe qui passe la vérification est acceptable
3. **Cohérence de l'Algorithme** : Tous les mots de passe doivent utiliser le même algorithme

## Exemples de Configuration

### Configuration de Base

```bash
# Configuration requise
AUTH_HOST=auth.example.com
PASSWORDS=plaintext:test123

# Configuration optionnelle
DEBUG=false
LANGUAGE=en
```

### Configuration de Production

```bash
# Configuration requise
AUTH_HOST=auth.example.com
PASSWORDS=bcrypt:$2a$10$N9qo8uLOickgx2ZMRZoMyeIjZAgcfl7p92ldGxad68LJZdL17lhWy

# Configuration optionnelle
DEBUG=false
LANGUAGE=fr
LOGIN_PAGE_TITLE=Mon Service d'Authentification
LOGIN_PAGE_FOOTER_TEXT=© 2024 Ma Société
USER_HEADER_NAME=X-Forwarded-User
COOKIE_DOMAIN=.example.com
```

### Configuration Docker Compose

```yaml
services:
  stargate:
    image: stargate:latest
    environment:
      # Configuration requise
      - AUTH_HOST=auth.example.com
      - PASSWORDS=bcrypt:$2a$10$N9qo8uLOickgx2ZMRZoMyeIjZAgcfl7p92ldGxad68LJZdL17lhWy
      
      # Configuration optionnelle
      - DEBUG=false
      - LANGUAGE=fr
      - LOGIN_PAGE_TITLE=Mon Service d'Authentification
      - LOGIN_PAGE_FOOTER_TEXT=© 2024 Ma Société
      - COOKIE_DOMAIN=.example.com
```

### Configuration de Développement Local

```bash
# Configuration requise
AUTH_HOST=localhost
PASSWORDS=plaintext:test123|admin456

# Configuration optionnelle
DEBUG=true
LANGUAGE=fr
PORT=8080
```

## Validation de Configuration

Stargate valide tous les éléments de configuration au démarrage :

1. **Vérification de Configuration Requise** : Si la configuration requise n'est pas définie, le service échouera au démarrage et affichera un message d'erreur
2. **Validation de Format** : Un format de configuration de mot de passe incorrect provoquera un échec au démarrage
3. **Validation d'Algorithme** : Les algorithmes de mot de passe non supportés provoqueront un échec au démarrage
4. **Validation de Valeur** : Certains éléments de configuration ont des restrictions de valeur (par ex., `LANGUAGE`, `DEBUG`)

**Exemples d'Erreurs :**

```bash
# Configuration requise manquante
Error: Configuration error: environment variable 'AUTH_HOST' is required but not set.

# Format de mot de passe incorrect
Error: Configuration error: invalid value for environment variable 'PASSWORDS': 'invalid_format'

# Algorithme non supporté
Error: Configuration error: invalid value for environment variable 'PASSWORDS': 'unknown:password'
```

## Meilleures Pratiques de Configuration

1. **Sécurité de Production** :
   - Utiliser les algorithmes `bcrypt` ou `sha512`, éviter `plaintext`
   - Définir `DEBUG=false`
   - Utiliser des mots de passe forts

2. **Sessions Cross-Domain** :
   - Si vous devez partager des sessions entre sous-domaines, définir `COOKIE_DOMAIN`
   - Format : `.example.com` (notez le point initial)

3. **Support Multilingue** :
   - Définir `LANGUAGE` selon la base d'utilisateurs
   - Supporte `en`, `zh`, `fr`, `it`, `ja`, `de`, `ko`

4. **Interface Personnalisée** :
   - Utiliser `LOGIN_PAGE_TITLE` et `LOGIN_PAGE_FOOTER_TEXT` pour personnaliser la page de connexion

5. **Surveillance et Débogage** :
   - Définir `DEBUG=true` dans l'environnement de développement pour des journaux détaillés
   - Définir `DEBUG=false` dans l'environnement de production pour réduire la sortie des journaux
