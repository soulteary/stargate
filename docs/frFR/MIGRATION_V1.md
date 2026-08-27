# Migration de v0.12.0 vers v1.0.0

Stargate v1.0.0 fixe les premiers contrats stables de déploiement et d'API HTTP. Validez la migration en préproduction avant de déplacer le trafic.

## Avant la mise à niveau

1. Relever l'image, les variables d'environnement, les routes proxy, les hôtes de callback et les sondes.
2. Conserver l'image `v0.12.0` et le conteneur précédent pour le retour en arrière.
3. Identifier les clients de `/_logout`, `/totp/enroll` et `/_session_exchange`.
4. Préparer le stockage de session Redis lorsqu'il existe plusieurs répliques.

## Modifications obligatoires

| Contrat | v0.12.0 | v1.0.0 |
| --- | --- | --- |
| Port du conteneur | `80` | `8080` |
| Utilisateur d'exécution | root | UID/GID `10001` |
| Liveness | `/health` | `/healthz` |
| Readiness | `/health` | `/readyz` |

La cible Traefik ForwardAuth devient `http://stargate:8080/_auth`. Avec plusieurs répliques, activez `SESSION_STORAGE_ENABLED=true` et partagez Redis ainsi que les secrets de ticket.

## Contrats de sécurité et HTTP

- L'échange inter-domaines utilise `?ticket=` au lieu de l'identifiant de session brut `?id=`. Configurez `CALLBACK_ALLOWED_HOSTS` et un `SESSION_EXCHANGE_SECRET` d'au moins 32 caractères.
- Les en-têtes transférés ne sont acceptés que depuis `TRUSTED_PROXIES`.
- `Stargate-Password` est désactivé par défaut ; activez `PASSWORD_HEADER_AUTH_ENABLED=true` uniquement pour un proxy de confiance.
- Les en-têtes d'identité nécessitent Warden, `HEADER_AUTH_ENABLED=true` et un secret partagé fort.
- `GET /_logout` devient `POST /_logout`.
- `GET /totp/enroll` affiche une confirmation sûre ; `POST /totp/enroll` démarre l'inscription.
- `GET /metrics` reste inchangé.

## Déploiement et vérification

```bash
curl --fail http://127.0.0.1:8080/healthz
curl --fail http://127.0.0.1:8080/readyz
curl --fail http://127.0.0.1:8080/metrics
```

Testez la connexion, la déconnexion, TOTP, les callbacks autorisés et refusés, ainsi que les tickets expirés ou réutilisés. Ne mélangez pas v0.12.0 et v1.0.0 dans un même pool.

## Retour en arrière

Restaurez ensemble l'image, le port et les sondes v0.12.0. Invalidez les tickets v1 en cours et imposez une nouvelle connexion.
