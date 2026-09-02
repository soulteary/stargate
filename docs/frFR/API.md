# Documentation API

Ce document décrit en détail tous les points de terminaison API du service Stargate Forward Auth.

Pour une mise à niveau depuis v0.12.0, consultez d'abord le [guide de migration v1.0.0](MIGRATION_V1.md).

## Table des Matières

- [Point de Terminaison de Vérification d'Authentification](#point-de-terminaison-de-vérification-dauthentification)
- [Point de Terminaison de Connexion](#point-de-terminaison-de-connexion)
- [Point de Terminaison d'Envoi de Code de Vérification](#point-de-terminaison-denvoi-de-code-de-vérification)
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
   - Désactivée par défaut. Définissez `PASSWORD_HEADER_AUTH_ENABLED=true` uniquement pour des clients historiques de confiance.
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
# Prérequis côté serveur : PASSWORD_HEADER_AUTH_ENABLED=true
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

Traite les requêtes de connexion et prend en charge deux modes d'authentification :

1. **Authentification par mot de passe** : vérifie le mot de passe et crée une session
2. **Authentification Warden** : vérifie soit un code de challenge Herald, soit un code TOTP déjà configuré, puis crée une session

#### Corps de Requête

Données de formulaire (`application/x-www-form-urlencoded`) :

**Authentification par mot de passe :**

| Champ | Type | Requis | Description |
|-------|------|--------|-------------|
| `auth_method` | String | Non | Méthode d'authentification ; `password` est la valeur par défaut |
| `password` | String | Oui | Mot de passe utilisateur |
| `callback` | String | Non | URL de callback après connexion réussie |

**Authentification Warden :**

| Champ | Type | Requis | Description |
|-------|------|--------|-------------|
| `auth_method` | String | Oui | Méthode d'authentification ; valeur `warden` |
| `phone` | String | Non | Numéro de téléphone ; au moins l'un de `phone` ou `mail` est requis |
| `mail` | String | Non | Adresse e-mail ; au moins l'un de `mail` ou `phone` est requis |
| `challenge_id` | String | Conditionnel | Requis avec `verify_code` ; facultatif lorsque `use_otp=true` |
| `verify_code` | String | Conditionnel | Requis pour le challenge Herald lorsque `use_otp` est absent ou vaut `false` |
| `use_otp` | Boolean | Non | Définir à `true` pour utiliser un authentificateur TOTP configuré au lieu d'un challenge Herald |
| `otp_code` | String | Conditionnel | Requis lorsque `use_otp=true` |
| `callback` | String | Non | URL de callback après connexion réussie |

Le gestionnaire accepte également `phone` + `mail` dans la même requête Warden.
La variante TOTP nécessite `HERALD_TOTP_ENABLED=true` et un utilisateur déjà inscrit. Sinon, utilisez la variante `challenge_id` + `verify_code`.

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
   - Redirige vers `{callback}/_session_exchange?ticket={opaque_ticket}`
   - Code de statut : `302 Found`

2. **Sans callback** :
   - **Requête HTML** : Retourne une page HTML avec meta refresh, redirigeant automatiquement vers le domaine d'origine
   - **Requête API** : Retourne une réponse JSON
     ```json
     {
       "success": true,
       "message": "Login successful"
     }
     ```

**Réponse d'Échec**

| Code de Statut | Description | Corps de Réponse |
|----------------|-------------|-------------------|
| `400 Bad Request` | Identifiant Warden ou champs de la méthode choisie invalides/manquants, ou TOTP non inscrit | Message d'erreur au format JSON/XML/texte selon l'en-tête Accept |
| `401 Unauthorized` | Mot de passe incorrect, utilisateur Warden absent/inactif ou échec du challenge/TOTP | Message d'erreur au format JSON/XML/texte selon l'en-tête Accept |
| `502 Bad Gateway` | Échec de la consultation du statut TOTP auprès de Herald | Message d'erreur |
| `503 Service Unavailable` | Service de vérification Herald indisponible | Message d'erreur |
| `500 Internal Server Error` | Erreur serveur | Message d'erreur |

#### Exemples

```bash
# Soumettre le formulaire de connexion (avec callback)
curl -X POST \
     -d "auth_method=password&password=yourpassword&callback=app.example.com" \
     -c cookies.txt \
     http://auth.example.com/_login

# Soumettre le formulaire de connexion (sans callback, inférera automatiquement)
curl -X POST \
     -d "auth_method=password&password=yourpassword" \
     -H "X-Forwarded-Host: app.example.com" \
     -c cookies.txt \
     http://auth.example.com/_login

# Warden + Herald : connexion avec le challenge renvoyé par /_send_verify_code
curl -X POST \
     -d "auth_method=warden&mail=user@example.com&challenge_id=ch_xxx&verify_code=123456&callback=app.example.com" \
     -c cookies.txt \
     http://auth.example.com/_login

# Warden + TOTP : connexion avec un authentificateur déjà configuré
curl -X POST \
     -d "auth_method=warden&mail=user@example.com&use_otp=true&otp_code=123456&callback=app.example.com" \
     -c cookies.txt \
     http://auth.example.com/_login
```

## Point de Terminaison d'Envoi de Code de Vérification

### `POST /_send_verify_code`

Requête d'envoi de code de vérification. Ce point de terminaison est utilisé dans le flux d'authentification OTP Warden + Herald.

#### En-têtes de Requête (optionnel)

| En-tête | Description |
|---------|-------------|
| `Idempotency-Key` | Optionnel. Si présent, Stargate le transmet à Herald ; Herald renvoie la même réponse de défi pour les requêtes en double avec la même clé dans la TTL. |

#### Corps de Requête

Types de média pris en charge pour le corps de requête :

| Type de média du corps | Pris en charge |
|------------------------|----------------|
| `application/x-www-form-urlencoded` | ✅ |
| `multipart/form-data` | ✅ |
| `application/json` | ❌ |

| Champ | Type | Requis | Description |
|-------|------|--------|-------------|
| `phone` | String | Non | Numéro de téléphone ; au moins l'un de `phone` ou `mail` est requis |
| `mail` | String | Non | Adresse e-mail ; au moins l'un de `mail` ou `phone` est requis |
| `deliver_via` | String | Non | Canal préféré : `sms`, `email` ou `dingtalk`. Sans valeur, Stargate essaie d'abord une destination SMS activée, puis l'e-mail. DingTalk n'est jamais choisi implicitement ; le client doit envoyer `deliver_via=dingtalk`. |

Le gestionnaire accepte également `phone` + `mail` dans la même requête d'envoi de code.

#### Flux de Traitement

1. **Stargate → Warden** : Requête d'informations utilisateur
   - Vérifier si l'utilisateur est dans la liste blanche
   - Vérifier le statut utilisateur (si actif)
   - Obtenir l'email et le téléphone de l'utilisateur

2. **Stargate → Herald** : Créer un challenge et envoyer un code de vérification
   - Utiliser l'e-mail, le téléphone ou l'identifiant DingTalk renvoyé par Warden comme destination
   - Appeler l'API Herald pour créer un challenge
   - Herald envoie le code via le canal SMS, e-mail ou DingTalk sélectionné

3. **Retourner le Résultat** : Retourner challenge_id et informations associées

#### Réponse

**Réponse de Succès (200 OK)**

```json
{
  "success": true,
  "message": "Verification code sent",
  "challenge_id": "ch_xxxxxxxxxxxx",
  "expires_in": 300,
  "next_resend_in": 60
}
```

**Réponse d'Échec**

| Code de Statut | Description | Corps de Réponse |
|----------------|-------------|-------------------|
| `400 Bad Request` | Paramètres de requête invalides (phone ou mail manquant) | Message d'erreur |
| `401 Unauthorized` | Utilisateur absent de Warden ou inactif | Erreur JSON avec `success=false` |
| `429 Too Many Requests` | Limite de débit déclenchée | Message d'erreur |
| `503 Service Unavailable` | Client ou service Herald indisponible | Erreur JSON avec `success=false` |
| `500 Internal Server Error` | Autre erreur du fournisseur lors de l'envoi | Erreur JSON avec `success=false` |

#### Exemples

```bash
# Envoyer un code de vérification (en utilisant l'email)
curl -X POST \
     -d "mail=user@example.com" \
     -H "Content-Type: application/x-www-form-urlencoded" \
     http://auth.example.com/_send_verify_code

# Envoyer un code de vérification (en utilisant le téléphone)
curl -X POST \
     -d "phone=13800138000" \
     -H "Content-Type: application/x-www-form-urlencoded" \
     http://auth.example.com/_send_verify_code

```

#### Notes

- Nécessite `WARDEN_ENABLED=true` et `HERALD_ENABLED=true`
- L'utilisateur doit être dans la liste blanche Warden pour envoyer un code de vérification
- Herald effectue une limitation de débit, avec des limites de fréquence pour le même utilisateur/téléphone/email
- Le temps d'expiration du code est déterminé par la configuration Herald (par défaut 300 secondes)
- Le délai de renvoi est déterminé par la configuration Herald (par défaut 60 secondes)

## Point de Terminaison de Déconnexion

### `POST /_logout`

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
curl -X POST -b cookies.txt http://auth.example.com/_logout
```

## Point de Terminaison d'Authentification Renforcée

### `GET /_step_up`

Affiche un formulaire de revérification du mot de passe pour une session authentifiée. Le paramètre de requête facultatif `callback` doit être un chemin local ; les valeurs non sûres sont remplacées par `/`.

### `POST /_step_up`

Revérifie le mot de passe et enregistre l'authentification renforcée dans la session pendant 10 minutes. Champs du formulaire : `password` et `callback` local facultatif. La route est protégée par un contrôle same-origin et une limitation de débit.

## Point de Terminaison d'Échange de Session

### `GET /_session_exchange`

Utilisé pour le partage de session cross-domain. Échange un ticket chiffré, de courte durée et lié au domaine cible, vérifie la session authentifiée, définit le cookie puis redirige vers le chemin racine.

Ce point de terminaison est principalement utilisé pour partager les sessions d'authentification entre plusieurs domaines/sous-domaines. Après qu'un utilisateur se connecte sur un domaine, ce point de terminaison peut être utilisé pour définir le cookie de session sur un autre domaine.

#### Paramètres de Requête

| Paramètre | Type | Requis | Description |
|-----------|------|--------|-------------|
| `ticket` | String | Oui | Ticket d'échange lié au domaine cible et retourné après la connexion |

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
| `400 Bad Request` | Ticket manquant | Message d'erreur |
| `401 Unauthorized` | Ticket invalide, expiré, rejoué ou destiné à un autre domaine | Message d'erreur |

#### Domaine du Cookie

Si la variable d'environnement `COOKIE_DOMAIN` est configurée, le cookie sera défini au domaine spécifié, permettant le partage cross-sous-domaine.

#### Exemple

```bash
# Échanger le ticket sur le domaine de callback auquel il est lié
curl "https://app.example.com/_session_exchange?ticket=<opaque_ticket>"
```

**Scénario d'Utilisation Typique :**

1. L'utilisateur se connecte à `auth.example.com`
2. Après connexion réussie, redirige vers `app.example.com/_session_exchange?ticket=<opaque_ticket>`
3. Le cookie de session est défini au domaine `.example.com` (si `COOKIE_DOMAIN=.example.com` est configuré)
4. Redirige vers `app.example.com/`
5. L'utilisateur peut utiliser cette session sur tous les sous-domaines `*.example.com`

## Points de Terminaison TOTP

Ces points de terminaison sont disponibles lorsque Herald et Herald TOTP sont activés. Ils exigent tous une session authentifiée. L'inscription doit commencer et se terminer dans les 10 minutes suivant la connexion qui a créé la session.

### `GET /totp/enroll`

Affiche une page de confirmation sans créer d'état d'inscription. Nécessite une session créée par une connexion réussie au cours des 10 dernières minutes. Un utilisateur non authentifié est redirigé avec `302 Found` vers `/_login` ; une session plus ancienne reçoit `401 Unauthorized`.

### `POST /totp/enroll`

Démarre l'inscription et affiche la page d'association TOTP. Nécessite une session créée par une connexion réussie au cours des 10 dernières minutes. Un utilisateur non authentifié est redirigé avec `302 Found` vers `/_login` ; une session plus ancienne reçoit `401 Unauthorized`.

### `POST /totp/enroll/confirm`

Confirme l'inscription avec des données de formulaire `application/x-www-form-urlencoded` ou `multipart/form-data` ; les corps JSON ne sont pas pris en charge. Une session authentifiée créée au cours des 10 dernières minutes, `enroll_id` et le `code` TOTP à six chiffres sont obligatoires. Une authentification absente ou trop ancienne renvoie `401 Unauthorized` ; en cas de succès, la réponse JSON inclut les codes de secours affichés une seule fois.

### `GET /totp/revoke`

Affiche la page de confirmation de suppression de l'association TOTP. Une session authentifiée est requise ; un utilisateur non authentifié est redirigé avec `302 Found` vers `/_login`. La limite de 10 minutes ne s'applique pas à cette page.

### `POST /totp/revoke`

Supprime l'association TOTP. Une session authentifiée est requise et le client doit se réauthentifier avec `password` ou un `code` TOTP actuel valide. Une réauthentification absente ou échouée renvoie `401 Unauthorized`.

## Points de Terminaison de Santé

### `GET /healthz`

Sonde de liveness du processus sans interroger Redis, Warden ou Herald. Utilisez-la pour les vérifications de conteneur et les sondes de liveness Kubernetes.

### `GET /readyz`

État de disponibilité agrégé des dépendances Redis, Warden et Herald configurées. Utilisez-le pour les répartiteurs de charge et les sondes de readiness Kubernetes.

```bash
curl http://auth.example.com/healthz
curl http://auth.example.com/readyz
```

`GET /health` reste un alias de compatibilité obsolète de `GET /readyz`.

## Point de Terminaison des Métriques

### `GET /metrics`

Renvoie les métriques Prometheus au format texte. Ce point de terminaison ne requiert pas d'authentification et doit être protégé par des règles réseau ou de proxy.

## Point de Terminaison du Niveau de Journalisation

### `GET /log/level`

Renvoie le niveau de journalisation actuel. `PUT /log/level` ou `POST /log/level` le modifie avec le corps JSON `{"level":"debug"}` ou le paramètre `?level=debug`. Stargate limite ce point de terminaison à `127.0.0.1` ; il ne doit pas être exposé par le reverse proxy public.

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
7. Redirige vers `https://app.example.com/_session_exchange?ticket=<opaque_ticket>`
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
3. **Vérification du mot de passe** : Les mots de passe sont des valeurs opaques sensibles à la casse ; les espaces et la casse sont conservés
4. **Support de plusieurs mots de passe** : Plusieurs mots de passe peuvent être configurés, tout mot de passe qui passe la vérification est acceptable
5. **Sessions cross-domain** : La variable d'environnement `COOKIE_DOMAIN` doit être configurée pour activer le partage de session cross-domain
