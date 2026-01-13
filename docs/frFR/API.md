# Documentation API

Ce document décrit en détail tous les points de terminaison API du service Stargate Forward Auth.

## Table des Matières

- [Point de Terminaison de Vérification d'Authentification](#point-de-terminaison-de-vérification-dauthentification)
- [Point de Terminaison de Connexion](#point-de-terminaison-de-connexion)
- [Point de Terminaison de Déconnexion](#point-de-terminaison-de-déconnexion)
- [Point de Terminaison d'Échange de Session](#point-de-terminaison-déchange-de-session)
- [Point de Terminaison de Vérification de Santé](#point-de-terminaison-de-vérification-de-santé)
- [Point de Terminaison Racine](#point-de-terminaison-racine)

## Point de Terminaison de Vérification d'Authentification

### `GET /_auth`

Le point de terminaison principal de vérification d'authentification pour Traefik Forward Auth. Ce point de terminaison est la fonctionnalité principale de Stargate, utilisé pour vérifier si un utilisateur a été authentifié.

#### Méthodes d'Authentification

Stargate supporte deux méthodes d'authentification, vérifiées dans l'ordre de priorité suivant :

1. **Authentification par En-tête** (requêtes API)
   - En-tête de requête : `Stargate-Password: <password>`
   - Adapté aux requêtes API, scripts d'automatisation, etc.

2. **Authentification par Cookie** (requêtes Web)
   - Cookie : `stargate_session_id=<session_id>`
   - Adapté aux applications Web accessibles via navigateurs

#### En-têtes de Requête

| En-tête | Type | Requis | Description |
|---------|------|--------|-------------|
| `Stargate-Password` | String | Non | Authentification par mot de passe pour les requêtes API |
| `Cookie` | String | Non | Cookie de session contenant `stargate_session_id` |
| `Accept` | String | Non | Utilisé pour déterminer le type de requête (HTML/API) |

#### Réponse

**Réponse de Succès (200 OK)**

Lorsque l'authentification réussit, Stargate définit l'en-tête d'information utilisateur et retourne un code de statut 200 :

```
HTTP/1.1 200 OK
X-Forwarded-User: authenticated
```

Le nom de l'en-tête utilisateur peut être configuré via la variable d'environnement `USER_HEADER_NAME` (par défaut : `X-Forwarded-User`).

**Réponse d'Échec**

| Code de Statut | Description | Corps de Réponse |
|----------------|-------------|-------------------|
| `401 Unauthorized` | Authentification échouée | Message d'erreur (format JSON pour les requêtes API) ou redirection vers la page de connexion (requêtes HTML) |
| `500 Internal Server Error` | Erreur serveur | Message d'erreur |

#### Gestion du Type de Requête

- **Requêtes HTML** : Redirection vers `/_login?callback=<originalURL>` en cas d'échec d'authentification
- **Requêtes API** (JSON/XML) : Retour d'une réponse d'erreur 401 en cas d'échec d'authentification

#### Exemples

**Utilisation de l'Authentification par En-tête (Requête API)**

```bash
curl -H "Stargate-Password: yourpassword" \
     http://auth.example.com/_auth
```

**Utilisation de l'Authentification par Cookie (Requête Web)**

```bash
curl -H "Cookie: stargate_session_id=<session_id>" \
     http://auth.example.com/_auth
```

## Point de Terminaison de Connexion

### `GET /_login`

Affiche la page de connexion.

#### Paramètres de Requête

| Paramètre | Type | Requis | Description |
|-----------|------|--------|-------------|
| `callback` | String | Non | URL de callback après connexion réussie (généralement le domaine de la requête originale) |

#### Comportement

- Si l'utilisateur est déjà connecté, redirige automatiquement vers le point de terminaison d'échange de session
- Si l'utilisateur n'est pas connecté, affiche la page de connexion
- Si l'URL contient un paramètre `callback` et que le domaine diffère, le callback est stocké dans le cookie `stargate_callback` (expire dans 10 minutes)

#### Priorité de Récupération du Callback

1. **Depuis les paramètres de requête** : Le paramètre `callback` dans l'URL (priorité la plus élevée)
2. **Depuis le cookie** : Si non présent dans les paramètres de requête, récupérer depuis le cookie `stargate_callback`

#### Réponse

**200 OK** - Retourne le HTML de la page de connexion

La page inclut :
- Formulaire de connexion
- Titre personnalisable (`LOGIN_PAGE_TITLE`)
- Texte de pied de page personnalisable (`LOGIN_PAGE_FOOTER_TEXT`)

#### Exemple

```bash
# Accéder à la page de connexion
curl http://auth.example.com/_login?callback=app.example.com
```

### `POST /_login`

Traite les requêtes de connexion, vérifie le mot de passe et crée une session.

#### Corps de Requête

Données de formulaire (`application/x-www-form-urlencoded`) :

| Champ | Type | Requis | Description |
|-------|------|--------|-------------|
| `password` | String | Oui | Mot de passe utilisateur |
| `callback` | String | Non | URL de callback après connexion réussie |

#### Priorité de Récupération du Callback

Le traitement de connexion récupère le callback dans l'ordre de priorité suivant :

1. **Depuis le cookie** : Si le domaine différait lors de l'accès précédent à la page de connexion, le callback est stocké dans le cookie `stargate_callback`
2. **Depuis les données du formulaire** : Le champ `callback` dans les données du formulaire de la requête POST
3. **Depuis les paramètres de requête** : Le `callback` dans les paramètres de requête de l'URL
4. **Inférence automatique** : Si aucun des éléments ci-dessus n'existe, et que le domaine d'origine (`X-Forwarded-Host`) diffère du domaine du service d'authentification, utiliser le domaine d'origine comme callback

#### Réponse

**Réponse de Succès (200 OK)**

La réponse varie selon qu'il y a un callback et le type de requête :

1. **Avec callback** :
   - Redirige vers `{callback}/_session_exchange?id={session_id}`
   - Code de statut : `302 Found`

2. **Sans callback** :
   - **Requête HTML** : Retourne une page HTML avec meta refresh, redirigeant automatiquement vers le domaine d'origine
   - **Requête API** : Retourne une réponse JSON
     ```json
     {
       "success": true,
       "message": "Login successful",
       "session_id": "<session_id>"
     }
     ```

**Réponse d'Échec**

| Code de Statut | Description | Corps de Réponse |
|----------------|-------------|-------------------|
| `401 Unauthorized` | Mot de passe incorrect | Message d'erreur au format JSON/XML/texte selon l'en-tête Accept |
| `500 Internal Server Error` | Erreur serveur | Message d'erreur |

#### Exemples

```bash
# Soumettre le formulaire de connexion (avec callback)
curl -X POST \
     -d "password=yourpassword&callback=app.example.com" \
     -c cookies.txt \
     http://auth.example.com/_login

# Soumettre le formulaire de connexion (sans callback, inférera automatiquement)
curl -X POST \
     -d "password=yourpassword" \
     -H "X-Forwarded-Host: app.example.com" \
     -c cookies.txt \
     http://auth.example.com/_login
```

## Point de Terminaison de Déconnexion

### `GET /_logout`

Déconnecte l'utilisateur actuel et détruit la session.

#### Réponse

**Réponse de Succès (200 OK)**

```
HTTP/1.1 200 OK
Content-Type: text/plain

Logged out
```

Le cookie de session sera effacé.

#### Exemple

```bash
curl -b cookies.txt http://auth.example.com/_logout
```

## Point de Terminaison d'Échange de Session

### `GET /_session_exchange`

Utilisé pour le partage de session cross-domain. Définit le cookie d'ID de session spécifié et redirige vers le chemin racine.

Ce point de terminaison est principalement utilisé pour partager les sessions d'authentification entre plusieurs domaines/sous-domaines. Après qu'un utilisateur se connecte sur un domaine, ce point de terminaison peut être utilisé pour définir le cookie de session sur un autre domaine.

#### Paramètres de Requête

| Paramètre | Type | Requis | Description |
|-----------|------|--------|-------------|
| `id` | String | Oui | ID de session à définir |

#### Réponse

**Réponse de Succès (302 Redirect)**

```
HTTP/1.1 302 Found
Location: /
Set-Cookie: stargate_session_id=<session_id>; Path=/; HttpOnly; SameSite=Lax; Domain=<cookie_domain>; Expires=<expiry>
```

**Réponse d'Échec**

| Code de Statut | Description | Corps de Réponse |
|----------------|-------------|-------------------|
| `400 Bad Request` | ID de session manquant | Message d'erreur |

#### Domaine du Cookie

Si la variable d'environnement `COOKIE_DOMAIN` est configurée, le cookie sera défini au domaine spécifié, permettant le partage cross-sous-domaine.

#### Exemple

```bash
# Définir le cookie de session (pour les scénarios cross-domain)
curl "http://auth.example.com/_session_exchange?id=<session_id>"
```

**Scénario d'Utilisation Typique :**

1. L'utilisateur se connecte à `auth.example.com`
2. Après connexion réussie, redirige vers `app.example.com/_session_exchange?id=<session_id>`
3. Le cookie de session est défini au domaine `.example.com` (si `COOKIE_DOMAIN=.example.com` est configuré)
4. Redirige vers `app.example.com/`
5. L'utilisateur peut utiliser cette session sur tous les sous-domaines `*.example.com`

## Point de Terminaison de Vérification de Santé

### `GET /health`

Point de terminaison de vérification de santé du service. Utilisé pour surveiller le statut du service.

#### Réponse

**Réponse de Succès (200 OK)**

```
HTTP/1.1 200 OK
```

#### Exemple

```bash
curl http://auth.example.com/health
```

**Utilisations Typiques :**

- Vérifications de santé Docker
- Sondes de liveness Kubernetes
- Vérifications de santé de répartiteur de charge

## Point de Terminaison Racine

### `GET /`

Chemin racine, affiche les informations du service.

#### Réponse

**200 OK** - Retourne la page d'informations du service

#### Exemple

```bash
curl http://auth.example.com/
```

## Format de Réponse d'Erreur

Toutes les réponses d'erreur API sélectionnent automatiquement le format en fonction de l'en-tête `Accept` du client :

### Format JSON (`Accept: application/json`)

```json
{
  "error": "Error message",
  "code": 401
}
```

### Format XML (`Accept: application/xml`)

```xml
<errors>
  <error code="401">Error message</error>
</errors>
```

### Format Texte (Par Défaut)

```
Error message
```

Les messages d'erreur supportent l'internationalisation, retournant des messages en chinois ou en anglais selon la variable d'environnement `LANGUAGE`.

## Exemples de Flux d'Authentification

### Flux d'Authentification d'Application Web

1. L'utilisateur accède à une ressource protégée (par ex., `https://app.example.com/dashboard`)
2. Traefik intercepte la requête et la transmet à `https://auth.example.com/_auth`
3. Stargate vérifie la session dans le cookie
4. Si non authentifié, redirige vers `https://auth.example.com/_login?callback=app.example.com`
5. L'utilisateur entre le mot de passe et soumet
6. Stargate vérifie le mot de passe, crée une session, définit le cookie
7. Redirige vers `https://app.example.com/_session_exchange?id=<session_id>`
8. Le cookie de session est défini au domaine `app.example.com`
9. L'utilisateur accède à nouveau à la ressource protégée, l'authentification réussit

### Flux d'Authentification API

1. Le client API envoie une requête à une ressource protégée
2. Traefik intercepte la requête et la transmet à `https://auth.example.com/_auth`
3. Le client API inclut `Stargate-Password: <password>` dans l'en-tête de requête
4. Stargate vérifie le mot de passe
5. Si la vérification réussit, définit l'en-tête `X-Forwarded-User` et retourne 200
6. Traefik permet à la requête de continuer vers le service backend

## Notes

1. **Temps d'expiration de session** : Par défaut 24 heures, nécessite une reconnexion après expiration
2. **Sécurité des cookies** : Tous les cookies sont définis avec les flags `HttpOnly` et `SameSite=Lax`
3. **Vérification du mot de passe** : Les mots de passe sont normalisés avant vérification (supprimer les espaces, convertir en majuscules)
4. **Support de plusieurs mots de passe** : Plusieurs mots de passe peuvent être configurés, tout mot de passe qui passe la vérification est acceptable
5. **Sessions cross-domain** : La variable d'environnement `COOKIE_DOMAIN` doit être configurée pour activer le partage de session cross-domain
