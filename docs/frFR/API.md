# Documentation API

Ce document décrit en détail tous les points de terminaison API du service Stargate Forward Auth.

Pour une mise à niveau depuis v0.12.0, consultez d'abord le [guide de migration v1.0.0](MIGRATION_V1.md).

## Table des Matières

- [Point de Terminaison de Vérification d'Authentification](#point-de-terminaison-de-vérification-dauthentification)
- [Point de Terminaison de Connexion](#point-de-terminaison-de-connexion)
- [Point de Terminaison d'Envoi de Code de Vérification](#point-de-terminaison-denvoi-de-code-de-vérification)
- [Point de Terminaison de Déconnexion](#point-de-terminaison-de-déconnexion)
- [Point de Terminaison d'Échange de Session](#point-de-terminaison-déchange-de-session)
- [Point de Terminaison de Vérification de Santé](#points-de-terminaison-de-santé)
- [Point de Terminaison Racine](#point-de-terminaison-racine)

## Point de Terminaison de Vérification d'Authentification

### `GET /_auth`

Le point de terminaison principal de vérification d'authentification pour Traefik Forward Auth. Ce point de terminaison est la fonctionnalité principale de Stargate, utilisé pour vérifier si un utilisateur a été authentifié.

#### Méthodes d'Authentification

Stargate supporte trois méthodes d'authentification, vérifiées dans l'ordre de priorité suivant :

1. **Authentification par En-tête** (requêtes API)
   - En-tête de requête : `Stargate-Password: <password>`
   - Désactivée par défaut. Définissez `PASSWORD_HEADER_AUTH_ENABLED=true` uniquement pour des clients historiques de confiance.
   - Adapté aux requêtes API, scripts d'automatisation, etc.

2. **En-têtes d’identité de confiance**
   - Exige `HEADER_AUTH_ENABLED=true`, Warden et le bon `HEADER_AUTH_SHARED_SECRET` dans `HEADER_AUTH_SECRET_HEADER` (par défaut `X-Stargate-Header-Auth`). `X-User-Phone` / `X-User-Mail` sont vérifiés dans l’annuaire. Le proxy doit remplacer les en-têtes d’identité et de secret fournis par le client ; ne jamais exposer ce secret au navigateur.

3. **Authentification par Cookie** (requêtes Web)
   - Cookie : `stargate_session_id=<session_id>`
   - Adapté aux applications Web accessibles via navigateurs

#### En-têtes de Requête

| En-tête | Type | Requis | Description |
|---------|------|--------|-------------|
| `Stargate-Password` | String | Non | Authentification par mot de passe pour les requêtes API |
| `X-User-Phone`, `X-User-Mail` | String | — | Identité de confiance facultative ; exige l’en-tête secret du proxy configuré |
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
| `429 Too Many Requests` | `RATE_LIMIT_LOGIN_MAX` tentatives `Stargate-Password` échouées depuis ce client dans la fenêtre en cours. Envoyé avec un en-tête `Retry-After` (secondes) pour toute requête qui porte l'en-tête, même avec le bon mot de passe, jusqu'à la fin de la fenêtre. Uniquement avec `PASSWORD_HEADER_AUTH_ENABLED=true` | Message d'erreur |
| `503 Service Unavailable` | Autorisation de l’annuaire temporairement indisponible | — |
| `500 Internal Server Error` | Erreur serveur | Message d'erreur |

#### Gestion du Type de Requête

- **Requêtes HTML** : Redirection vers `/_login?callback=<originalURL>` en cas d'échec d'authentification
- **Requêtes API** (JSON/XML) : Retour d'une réponse d'erreur 401 en cas d'échec d'authentification
- **Requêtes avec `Stargate-Password` au-delà du quota d'échecs** : Retour d'une réponse 429 avec `Retry-After`, pour les requêtes HTML comme pour les requêtes API ; il n'y a pas de redirection

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

1. **Depuis les données du formulaire** : Le champ `callback` dans les données du formulaire de la requête POST. La page de connexion le remplit avec le callback pour lequel elle a été ouverte
2. **Depuis le cookie** : Si le formulaire ne contient pas de `callback`, le cookie `stargate_callback` enregistré lorsque la page de connexion a été ouverte pour un domaine différent
3. **Depuis les paramètres de requête** : Le `callback` dans les paramètres de requête de l'URL
4. **Inférence automatique** : Si aucun des éléments ci-dessus n'existe, et que le domaine d'origine (`X-Forwarded-Host`) diffère du domaine du service d'authentification, utiliser le domaine d'origine comme callback

Quelle que soit la source de la valeur, elle doit passer la même validation de l'hôte du callback ; un champ `callback` non autorisé reçoit `400 Bad Request` et n'est pas remplacé par le cookie. Le cookie `stargate_callback` est supprimé après une connexion réussie, même lorsque la valeur du formulaire a été utilisée.

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
| `409 Conflict` | Verrou du vérificateur occupé ; aucune nouvelle tentative automatique du code | — |
| `429 Too Many Requests` | Limite de requêtes dépassée | — |
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

<!-- api-contract: send-verify-code-request-body -->
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

### `POST /_send_reauth_code`

Envoie un code à usage unique au contact enregistré dans Warden pour l'utilisateur de la session, pour la ré-authentification avant le step-up et avant l'ajout ou la suppression de TOTP. Nécessite le cookie de session ; le champ facultatif `deliver_via` ne choisit que le canal, la destination vient toujours de Warden. La réponse a le même format que `POST /_send_verify_code`. Le code est lié à la session qui l'a demandé et ne peut pas servir à se connecter.

## Point de Terminaison d'Authentification Renforcée

### `GET /_step_up`

Affiche un formulaire de revérification du mot de passe pour une session authentifiée. Le paramètre de requête facultatif `callback` doit être un chemin local ; les valeurs non sûres sont remplacées par `/`.

### `POST /_step_up`

Revérifie le mot de passe et enregistre l'authentification renforcée dans la session pendant 10 minutes. Champs du formulaire : `password` et `callback` local facultatif. La route est protégée par un contrôle same-origin et une limitation de débit.

## Point de Terminaison d'Échange de Session

### `GET /_session_exchange`

Démarre ou termine l’échange inter-domaines lié au navigateur. Sans `ticket` ou sans cookie nonce, la destination crée un cookie HttpOnly limité à cet hôte et redirige vers la connexion avec un `exchange_state` scellé (`302 Found`). Le ticket n’est pas nécessaire pour démarrer.

Le ticket retourné doit correspondre au cookie nonce de ce navigateur, à la destination, au schéma et à `SESSION_EXCHANGE_REDIRECT_PATH`. Il reste valable au maximum deux minutes et n’est utilisable qu’une fois ; plusieurs répliques exigent Redis partagé. Les anciens tickets non liés sont rejetés.

Le succès définit `stargate_session_id`, efface le cookie nonce et redirige vers le chemin local configuré. `COOKIE_DOMAIN` ne s’applique qu’aux destinations de ce domaine ; les autres destinations autorisées reçoivent un cookie propre à l’hôte. Un ticket invalide, expiré, consommé ou non concordant donne `401` ; une destination invalide ou HTTP non sécurisé avec des cookies sécurisés donne `400`.

Commencer dans le même navigateur : `https://app.example.com/_session_exchange?start=1`. Un lien contenant uniquement le ticket ou `curl` sans le cookie du navigateur ne peut pas importer la session. Voir les [conditions de migration](../enUS/DEPLOYMENT.md#browser-bound-session-exchange).

## Points de Terminaison TOTP

Ces points de terminaison sont disponibles lorsque Herald et Herald TOTP sont activés. Ils exigent tous une session authentifiée. L'inscription doit commencer et se terminer dans les 10 minutes suivant la connexion qui a créé la session.

### `GET /totp/enroll`

Affiche une page de confirmation sans créer d'état d'inscription. Nécessite une session créée par une connexion réussie au cours des 10 dernières minutes. Un utilisateur non authentifié est redirigé avec `302 Found` vers `/_login` ; une session plus ancienne reçoit `401 Unauthorized`.

### `POST /totp/enroll`

Démarre l'inscription et affiche la page d'association TOTP. Nécessite une session créée par une connexion réussie au cours des 10 dernières minutes. Un utilisateur non authentifié est redirigé avec `302 Found` vers `/_login` ; une session plus ancienne reçoit `401 Unauthorized`.

Les erreurs du statut TOTP ou du démarrage d’inscription en amont donnent `502 Bad Gateway`.

### `POST /totp/enroll/confirm`

Confirme l'inscription. Une session authentifiée créée au cours des 10 dernières minutes est obligatoire.

<!-- api-contract: totp-enroll-confirm-request-body -->
#### Corps de Requête

| Type de média du corps | Pris en charge |
|------------------------|----------------|
| `application/x-www-form-urlencoded` | ✅ |
| `multipart/form-data` | ✅ |
| `application/json` | ❌ |

Le formulaire doit contenir `enroll_id` et le `code` TOTP à six chiffres.

#### Réponse

Une authentification absente ou trop ancienne renvoie `401 Unauthorized` ; en cas de succès, la réponse JSON inclut les codes de secours affichés une seule fois.

Le quota de confirmation donne `429 Too Many Requests` avec `rate_limited` ; l’inscription en attente est conservée. Une réponse invalide du fournisseur donne `502`. Ne pas réessayer un code automatiquement.

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

Les vérifications des dépendances s'exécutent au plus une fois toutes les 5 secondes. Les deux routes répondent à partir de ce résultat jusqu'à son expiration : une dépendance qui tombe en panne ou se rétablit est donc signalée avec un retard pouvant atteindre 5 secondes, et les requêtes qui arrivent pendant l'exécution des vérifications attendent leur résultat au lieu de lancer les leurs.

## Point de Terminaison des Métriques

### `GET /metrics`

Renvoie les métriques Prometheus au format texte. Ce point de terminaison requiert `Authorization: Bearer <METRICS_API_KEY>` et reste désactivé tant que `METRICS_API_KEY` n'est pas configuré : sans `METRICS_API_KEY` il répond `404 Not Found`, et `401 Unauthorized` lorsque le jeton est absent ou incorrect. Restreignez aussi l'accès par des règles réseau ou de proxy.

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
5. **Sessions cross-domain** : Les destinations autorisées utilisent l’échange de session lié au navigateur décrit ci-dessus. `COOKIE_DOMAIN` est facultatif et partage les cookies uniquement dans ce domaine DNS ; les autres domaines autorisés reçoivent des cookies limités à l’hôte


### `GET /_account`

Page de gestion de session authentifiée avec liens TOTP et formulaire `POST /_logout` de même origine. Sans authentification, redirection vers `/_login` ; la fraîcheur de l’annuaire reste vérifiée. Ce GET ne démarre pas de configuration. Le point racine reste inchangé.
