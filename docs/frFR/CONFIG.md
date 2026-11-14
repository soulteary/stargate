# Référence de Configuration

> **Traduction historique** : ce guide ne reflète pas toutes les modifications actuelles. Consultez la [configuration anglaise actuelle](../enUS/CONFIG.md), notamment les prérequis Redis et les limites de mise à niveau.

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
export PASSWORDS='plaintext:yourpassword'
```

**Docker :**

```bash
docker run -e AUTH_HOST=auth.example.com -e PASSWORDS=plaintext:yourpassword ghcr.io/soulteary/stargate:1.3.8
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
PASSWORDS='plaintext:test123'

# Plusieurs mots de passe en texte brut
PASSWORDS='plaintext:test123|admin456|user789'

# Hash BCrypt
PASSWORDS='bcrypt:$2a$10$N9qo8uLOickgx2ZMRZoMyeIjZAgcfl7p92ldGxad68LJZdL17lhWy'

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
LOGIN_PAGE_TITLE="Mon Service d'Authentification"
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
LOGIN_PAGE_FOOTER_TEXT='© 2024 Ma Société'
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
- La valeur est l'identifiant de l'utilisateur authentifié lorsqu'il est disponible ; l'authentification par mot de passe seule utilise `authenticated`
- Les services backend peuvent déterminer si un utilisateur est authentifié via cet en-tête
- Doit être une chaîne non vide

**Exemple :**

```bash
USER_HEADER_NAME=X-Authenticated-User
```

### `TRUSTED_PROXIES` et `PROXY_HEADER`

`TRUSTED_PROXIES` liste les IP sources ou plages CIDR des proxys inverses dont Stargate accepte les en-têtes transférés. La valeur par défaut est vide ; `PROXY_HEADER` désigne l'en-tête qui porte l'IP du client (par défaut : `X-Forwarded-For`).

Derrière un proxy inverse, définissez `TRUSTED_PROXIES` même si Stargate démarre sans cette valeur. Tant que le proxy n'y figure pas, Stargate ne voit que la requête envoyée par le proxy à `/_auth`, pas celle de l'utilisateur :

- la redirection vers la connexion ne porte pas de `callback` et est construite avec `http://` ; après s'être connectés, les utilisateurs restent donc sur l'hôte d'authentification au lieu de revenir à l'application ;
- lorsque le proxy termine HTTPS, les utilisateurs ne peuvent pas se connecter du tout : le navigateur envoie `Origin: https://...`, Stargate le compare à la requête `http://` reçue du proxy, et `POST /_login`, `/_logout`, `/_step_up`, `/_send_verify_code`, `/_send_reauth_code` et `/totp/*` reçoivent `403` avec `cross-site request rejected` ;
- les règles de chemin step-up ne peuvent pas être évaluées (c'est pourquoi `STEP_UP_ENABLED=true` exige `TRUSTED_PROXIES`) ;
- chaque client est attribué à l'adresse du proxy et partage un seul quota de limitation de débit et une seule adresse dans les enregistrements d'audit.

Stargate journalise un avertissement au démarrage lorsque `TRUSTED_PROXIES` est vide, puis un autre la première fois qu'il ignore les en-têtes transférés d'un pair qui n'y figure pas. Il journalise aussi un avertissement, une seule fois, lorsqu'il rejette comme inter-site une requête dont l'`Origin` correspond aux en-têtes `X-Forwarded-Proto` et `X-Forwarded-Host` ignorés ; la réponse `403` elle-même n'indique pas la cause.

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

### `SESSION_EXCHANGE_REDIRECT_PATH`

Chemin de destination sur la même origine après un échange de session validé. Valeur par défaut : `/`. Utilisez par exemple `/auth/login/example/` pour terminer la connexion native. Seuls les chemins absolus configurés par l’administrateur, sans requête, fragment, encodage pour cent ou caractères de contrôle, sont acceptés. La liste des hôtes autorisés et la protection anti-rejeu restent inchangées.

### `PORT`

Port d'écoute du service pour les déploiements natifs et en conteneur. Géré par le package config avec les autres options d'environnement.

| Attribut | Valeur |
|----------|--------|
| **Type** | String |
| **Requis** | Non |
| **Par Défaut** | Vide pour le binaire natif (`:80`) ; les images officielles définissent `PORT=8080` |

**Description :**

- Configurable en développement et en production
- Les conteneurs officiels utilisent `8080` par défaut. Utiliser un nombre seul, comme `PORT=8080`, car le contrôle de santé construit une URL.
- Format du binaire natif : numéro de port (par ex., `8080`) ou `:port` (par ex., `:8080`)

**Exemple :**

```bash
PORT=8080
```

## Configuration de sécurité et des intégrations

Ce tableau est synchronisé avec les variables de sécurité réellement enregistrées dans `internal/config`. Une valeur optionnelle vide désactive l'intégration correspondante.

| Variable | Valeurs | Défaut | Rôle |
|----------|---------|--------|------|
| `TRUSTED_PROXIES` | liste IP/CIDR | vide | Sources autorisées à fournir les en-têtes Forwarded |
| `PROXY_HEADER` | nom d'en-tête | `X-Forwarded-For` | En-tête contenant l'IP source |
| `COOKIE_SECURE` | true/false | true | Attribut Secure des cookies de session |
| `CALLBACK_ALLOWED_HOSTS` | liste d'hôtes | vide | Destinations de redirection autorisées |
| `SESSION_EXCHANGE_SECRET` | secret, 32 caractères min. | vide | Signe les tickets inter-domaines à courte durée |
| `SESSION_EXCHANGE_REDIRECT_PATH` | Chemin | `/` | Non |
| `PASSWORD_HEADER_AUTH_ENABLED` | true/false | false | Active l'en-tête de mot de passe historique de confiance |
| `PASSWORD_HEADER_REQUIRE_TRUSTED_PROXY` | true/false | false | Refuse les requêtes `Stargate-Password` qui n'arrivent pas par un proxy de `TRUSTED_PROXIES` fournissant l'adresse du client |
| `LOG_LEVEL` | debug/info/warn/error | info | Niveau de journalisation initial |
| `HEADER_AUTH_ENABLED` | true/false | false | Authentification par en-têtes de confiance |
| `HEADER_AUTH_SHARED_SECRET` | secret, 32 caractères min. | vide | Secret partagé de l'authentification par en-tête |
| `HEADER_AUTH_SECRET_HEADER` | nom d'en-tête | `X-Stargate-Header-Auth` | Transporte le secret partagé |
| `WARDEN_ENABLED` | true/false | false | Intégration Warden |
| `WARDEN_URL` | URL | vide | Point de terminaison Warden |
| `WARDEN_API_KEY` | secret | vide | Authentification par clé API |
| `WARDEN_HMAC_KEY_ID` | chaîne | vide | Identifiant de clé HMAC |
| `WARDEN_HMAC_SECRET` | secret | vide | Secret de signature HMAC |
| `WARDEN_TLS_CA_CERT_FILE` | chemin | vide | Autorité de certification personnalisée |
| `WARDEN_TLS_CLIENT_CERT_FILE` | chemin | vide | Certificat client mTLS |
| `WARDEN_TLS_CLIENT_KEY_FILE` | chemin | vide | Clé client mTLS |
| `WARDEN_TLS_SERVER_NAME` | chaîne | vide | Nom de serveur TLS attendu |
| `WARDEN_CACHE_TTL` | entier positif (secondes) | 300 | Accepté uniquement par compatibilité ; sans effet |
| `WARDEN_SOURCE_MAX_AGE` | durée | 0s | Âge maximal de la source Warden ; 0s désactive le contrôle |
| `WARDEN_SOURCE_CLOCK_SKEW` | durée | 0s | Tolérance d’horloge, inférieure à l’âge maximal |
| `METRICS_API_KEY` | secret | vide | Clé Bearer distincte pour /metrics, au moins 32 octets |
| `HERALD_ENABLED` | true/false | false | Intégration Herald |
| `HERALD_URL` | URL | vide | Point de terminaison Herald |
| `HERALD_API_KEY` | secret | vide | Authentification par clé API |
| `HERALD_HMAC_KEY_ID` | chaîne | vide | Identifiant de clé HMAC |
| `HERALD_HMAC_SECRET` | secret | vide | Secret de signature HMAC |
| `HERALD_CLIENT_IP_HEADER` | nom d'en-tête | vide | En-tête dans lequel Stargate envoie l'adresse du client à Herald ; doit correspondre à `HERALD_TRUSTED_PROXY_HEADER` de Herald, et Stargate doit figurer dans `HERALD_TRUSTED_PROXIES` |
| `HERALD_TLS_CA_CERT_FILE` | chemin | vide | Autorité de certification personnalisée |
| `HERALD_TLS_CLIENT_CERT_FILE` | chemin | vide | Certificat client mTLS |
| `HERALD_TLS_CLIENT_KEY_FILE` | chemin | vide | Clé client mTLS |
| `HERALD_TLS_SERVER_NAME` | chaîne | vide | Nom de serveur TLS attendu |
| `HERALD_TOTP_ENABLED` | true/false | false | Connexion et gestion TOTP |
| `LOGIN_SMS_ENABLED` | true/false | true | Canal de connexion SMS |
| `LOGIN_EMAIL_ENABLED` | true/false | true | Canal de connexion e-mail |
| `LOGIN_DINGTALK_REQUIRE_USERID` | true/false | false | Exiger un userid Warden explicite pour DingTalk |
| `LOGIN_UNIFORM_RESPONSES` | true/false | false | Réponses identiques de `/_send_verify_code` et de la connexion Warden, que l'identifiant figure ou non dans Warden ; le temps de réponse n'est pas égalisé |
| `LOGIN_UNIFORM_RESEND_COOLDOWN` | durée | `60s` | Délai avant renvoi par identifiant avec `LOGIN_UNIFORM_RESPONSES=true` ; au moins `1s` et pas inférieur au `RESEND_COOLDOWN` de Herald |
| `SESSION_STORAGE_ENABLED` | true/false | false | Stockage de session Redis |
| `SESSION_STORAGE_REDIS_ADDR` | hôte:port | `localhost:6379` | Adresse Redis |
| `SESSION_STORAGE_REDIS_PASSWORD` | secret | vide | Mot de passe Redis |
| `SESSION_STORAGE_REDIS_DB` | entier | 0 | Base Redis |
| `SESSION_STORAGE_REDIS_KEY_PREFIX` | chaîne | `stargate:session:` | Préfixe des clés Redis |
| `SESSION_STORAGE_REDIS_USERNAME` | chaîne | vide | Utilisateur ACL Redis ; nécessite `SESSION_STORAGE_REDIS_PASSWORD` |
| `SESSION_STORAGE_REDIS_TLS_ENABLED` | true/false | false | TLS pour la connexion Redis, avec vérification du certificat du serveur |
| `SESSION_STORAGE_REDIS_TLS_CA_CERT_FILE` | chemin | vide | Autorité de certification personnalisée pour Redis ; nécessite `SESSION_STORAGE_REDIS_TLS_ENABLED=true` |
| `SESSION_STORAGE_REDIS_TLS_SERVER_NAME` | chaîne | vide | Nom de serveur TLS attendu pour Redis ; nécessite `SESSION_STORAGE_REDIS_TLS_ENABLED=true` |
| `AUDIT_LOG_ENABLED` | true/false | true | Journal d'audit |
| `AUDIT_LOG_FORMAT` | json/text | json | Format du journal d'audit |
| `STEP_UP_ENABLED` | true/false | false | Authentification renforcée |
| `STEP_UP_PATHS` | liste de chemins | vide | Chemins métier protégés |
| `OTLP_ENABLED` | true/false | false | Export OpenTelemetry |
| `OTLP_ENDPOINT` | URL | vide | Point de terminaison OTLP |
| `AUTH_REFRESH_ENABLED` | true/false | true | Actualisation périodique des autorisations |
| `AUTH_REFRESH_INTERVAL` | durée | `5m` | Intervalle d'actualisation |
| `REQUEST_CONTEXT_TIMEOUT` | durée | `10s` | Échéance des appels amont ; aucune garantie d'annulation à la déconnexion du client |
| `RATE_LIMIT_LOGIN_MAX` | entier | 10 | Tentatives de connexion par fenêtre et par client ; 0 désactive |
| `RATE_LIMIT_VERIFICATION_MAX` | entier | 5 | Demandes de vérification par fenêtre et par client ; 0 désactive |
| `RATE_LIMIT_WINDOW` | durée | `1m` | Fenêtre fixe de limitation de débit |

`LOGIN_DINGTALK_REQUIRE_USERID`: Valeur par défaut : `false` (mode de compatibilité). Avec `true`, seule la valeur `dingtalk_userid` gérée côté serveur par Warden est admise : aucun repli vers le numéro de téléphone, et les correspondances absentes ou invalides sont refusées. La même règle s’applique à la demande du code et à la vérification d’une liaison existante. Cette option ne s’active pas automatiquement et ne change ni les sessions existantes ni les limites de révocation du cache. Voir le [profil userid en anglais](../enUS/DINGTALK_USERID_ONLY.md).

`PORT` s’applique au développement et à la production. Le binaire natif utilise `:80` sans valeur ; les images officielles définissent `PORT=8080`. Dans un conteneur, utiliser un nombre seul, par exemple `PORT=8080` : le contrôle de santé construit une URL, donc la forme native `:8080` ne convient pas.

Warden exige l’identifiant et le secret HMAC ensemble. Pour Herald, fournir un identifiant explicite si le serveur utilise `HERALD_HMAC_KEYS` ; un seul `HMAC_SECRET` conserve le défaut implicite. Le certificat et la clé client mTLS doivent former une paire complète pour les deux services.

Si `WARDEN_URL` ou `HERALD_URL` est une URL `http://` et qu'une clé API ou un secret HMAC est configuré, Stargate journalise un avertissement au démarrage : la clé API ainsi que toutes les données de requête et de réponse transitent alors en clair sur le réseau. Utilisez `https://`, sauf si la connexion à Warden ou à Herald reste sur un réseau privé, par exemple un réseau interne à Compose.

`WARDEN_CACHE_TTL` est accepté uniquement par compatibilité et n'a aucun effet sur le comportement de Stargate. La valeur est transmise au client Warden comme durée de vie de son cache de la liste des utilisateurs, mais Stargate ne demande jamais cette liste : la connexion, l'envoi d'un code, l'authentification par en-tête de confiance et le rafraîchissement de l'autorisation recherchent chacun un seul utilisateur par numéro de téléphone ou adresse e-mail, et ces recherches ne sont pas mises en cache. Chacune est une requête à Warden. Le délai au bout duquel un utilisateur désactivé ou supprimé perd une session existante, ainsi que la fréquence à laquelle une session provoque une requête à Warden, dépendent de `AUTH_REFRESH_INTERVAL` ; un intervalle plus long réduit les requêtes à Warden mais retarde aussi l'effet d'une désactivation. Avec `HEADER_AUTH_ENABLED=true`, chaque requête transférée qui porte un en-tête d'identité de confiance provoque une recherche Warden. La valeur doit toujours être un entier positif, sinon Stargate ne démarre pas.

Les paramètres `SESSION_STORAGE_REDIS_*` n'ont d'effet qu'avec `SESSION_STORAGE_ENABLED=true`. Si l'un d'eux a une valeur différente de sa valeur par défaut alors que `SESSION_STORAGE_ENABLED` vaut `false`, Stargate démarre, conserve les sessions et les compteurs de limitation de débit en mémoire et journalise au démarrage un avertissement qui liste les variables ignorées.

L'utilisateur indiqué par `SESSION_STORAGE_REDIS_USERNAME` a besoin des droits de lecture, d'écriture et de script (`EVAL`/`EVALSHA`) sur les clés sous `SESSION_STORAGE_REDIS_KEY_PREFIX`. Sans `SESSION_STORAGE_REDIS_TLS_ENABLED=true`, les sessions, les contextes de vérification et les compteurs de limitation de débit transitent en clair vers Redis ; ne l'utilisez ainsi que sur un réseau privé. Avec TLS (1.2 ou ultérieur), le certificat est vérifié par rapport aux racines du système et à l'hôte de `SESSION_STORAGE_REDIS_ADDR`, sauf indication contraire de `SESSION_STORAGE_REDIS_TLS_CA_CERT_FILE` (fichier PEM de certificats d'autorité supplémentaires) et de `SESSION_STORAGE_REDIS_TLS_SERVER_NAME`. La vérification ne peut pas être désactivée ; les certificats client ne sont pas pris en charge.

## Configuration du Mot de Passe

Stargate accepte `bcrypt` en production et `plaintext` uniquement pour les tests locaux. MD5 et SHA-512 non salé sont refusés lors de la validation. Les mots de passe respectent la casse et conservent les espaces. Les mots de passe de démonstration publiés dans l'exemple Compose de ce dépôt (`test1234`, `test1337`) sont refusés au démarrage lorsque `COOKIE_SECURE` vaut `true` ; ils ne sont acceptés que pour la démonstration HTTP locale.

```bash
PASSWORDS='bcrypt:<bcrypt-hash>'
# Tests locaux uniquement :
PASSWORDS='plaintext:test123'
```

## Exemples de Configuration

### Configuration de Base

```bash
# Configuration requise
AUTH_HOST=auth.example.com
PASSWORDS='plaintext:test123'

# Configuration optionnelle
DEBUG=false
LANGUAGE=en
```

### Configuration de Production

```bash
# Configuration requise
AUTH_HOST=auth.example.com
PASSWORDS='bcrypt:$2a$10$N9qo8uLOickgx2ZMRZoMyeIjZAgcfl7p92ldGxad68LJZdL17lhWy'

# Configuration optionnelle
DEBUG=false
LANGUAGE=fr
LOGIN_PAGE_TITLE="Mon Service d'Authentification"
LOGIN_PAGE_FOOTER_TEXT='© 2024 Ma Société'
USER_HEADER_NAME=X-Forwarded-User
COOKIE_DOMAIN=.example.com
```

### Configuration Docker Compose

```yaml
services:
  stargate:
    image: ghcr.io/soulteary/stargate:1.3.8
    environment:
      # Configuration requise
      - AUTH_HOST=auth.example.com
      - PASSWORDS=bcrypt:$$2a$$10$$N9qo8uLOickgx2ZMRZoMyeIjZAgcfl7p92ldGxad68LJZdL17lhWy
      
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
PASSWORDS='plaintext:test123|admin456'

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
5. **Validation de Règle** : Certains éléments de configuration dépendent d'autres éléments ou ont une longueur minimale ou un format imposé (par ex., `HERALD_ENABLED` exige `WARDEN_ENABLED=true`)

Le message nomme la variable et indique lequel de ces cas s'applique. Une règle est rapportée comme motif ; elle n'est pas présentée comme la valeur configurée, et les valeurs configurées des secrets ne sont jamais affichées.

**Exemples d'Erreurs :**

```bash
# Configuration requise manquante
Error: Configuration error: environment variable 'AUTH_HOST' is required but not set.

# Format de mot de passe incorrect
Error: Configuration error: invalid value for environment variable 'PASSWORDS': 'invalid_format'

# Algorithme non supporté
Error: Configuration error: invalid value for environment variable 'PASSWORDS': 'unknown:password'

# Valeur hors de l'ensemble accepté
Error: Configuration error: invalid value for environment variable 'DEBUG': 'maybe'.
  Accepted values: [true false]

# Règle qui implique un autre paramètre
Error: Configuration error: environment variable 'HERALD_ENABLED': requires WARDEN_ENABLED=true to resolve authenticated users.
```

## Meilleures Pratiques de Configuration

1. **Sécurité de Production** :
   - Utiliser `bcrypt` en production ; réserver `plaintext` aux tests locaux
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

## Authentification historique par en-tête de mot de passe

L'authentification `Stargate-Password` est désactivée par défaut. Activez-la uniquement pour un client historique de confiance avec `PASSWORD_HEADER_AUTH_ENABLED=true`. Utilisez HTTPS, appliquez une limitation de débit et supprimez cet en-tête dans le proxy avant de transmettre la requête au service backend.

Les tentatives échouées avec cet en-tête sur `/_auth` sont comptées par adresse de client dans le quota `RATE_LIMIT_LOGIN_MAX`. Seules les valeurs incorrectes restent comptées. Une fois le quota épuisé, toute requête de cette adresse qui porte l'en-tête reçoit `429` avec `Retry-After` avant la comparaison du mot de passe, jusqu'à la fin de la fenêtre, y compris avec le bon mot de passe. Une requête est comptée pendant la comparaison de son mot de passe et décomptée lorsque le mot de passe est correct ; des valeurs incorrectes envoyées en même temps ne peuvent donc pas dépasser ensemble le quota. Pour la même raison, un client qui a plus de requêtes avec l'en-tête en cours que le quota restant reçoit `429` pour l'excédent et peut les répéter. Les clients qui partagent une adresse partagent ce quota :

- les clients situés derrière un même NAT ou proxy sortant forment une seule adresse ; si l'un d'eux envoie des valeurs incorrectes, il bloque les autres pour le reste de la fenêtre et peut recommencer à chaque fenêtre ;
- lorsque `TRUSTED_PROXIES` est vide, ou derrière un proxy ou un CDN qui n'y figure pas, chaque requête est attribuée au proxy ; cela concerne donc tous les clients du déploiement qui utilisent l'en-tête. Stargate journalise un avertissement au démarrage pour `PASSWORD_HEADER_AUTH_ENABLED=true` avec un `TRUSTED_PROXIES` vide.

Définissez `TRUSTED_PROXIES` pour que le quota s'applique par client. Les sessions par cookie ne sont pas concernées par ce quota.

## Fraîcheur de la source et supervision

`WARDEN_SOURCE_MAX_AGE=0s` reste la valeur par défaut pour la compatibilité.
Une valeur positive exige les métadonnées Warden v1 et
`AUTH_REFRESH_ENABLED=true`. `WARDEN_SOURCE_CLOCK_SKEW` est soustrait à la durée
valide. Une session ne prolonge pas une source expirée ; une erreur de source
refuse la requête avec HTTP 503.

Sans `METRICS_API_KEY`, `/metrics` renvoie HTTP 404. Avant la mise à jour,
configurer une clé aléatoire distincte d’au moins 32 octets et le scraper avec
`Authorization: Bearer <secret>`. Ne pas réutiliser une clé de service ou un mot
de passe de connexion. Entre hôtes, utiliser HTTPS avec vérification du certificat ;
HTTP suppose un réseau local isolé et de confiance.
Les règles détaillées et la migration sont décrites dans la
[référence anglaise](../enUS/CONFIG.md#source-freshness-and-scrape-credentials).


### `LOGIN_SUPPORT_URL`

Lien HTTPS facultatif ou une seule adresse `mailto:help@example.com`. Vide, le lien est masqué. Identifiants, caractères de contrôle et en-têtes mailto sont refusés.
