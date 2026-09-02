# Migration de v0.12.0 vers v1.0.0

Stargate v1.0.0 fixe les premiers contrats stables de déploiement et d'API HTTP. Validez la migration en préproduction avant de déplacer le trafic.

## Avant la mise à niveau

1. Relever l'image, les variables d'environnement, les routes proxy, les hôtes de callback et les sondes.
2. Conserver l'image `v0.12.0` et le conteneur précédent pour le retour en arrière.
3. Identifier les clients de `/_logout`, `/totp/enroll` et `/_session_exchange`.
4. Préparer Redis si plusieurs processus servent le trafic, si anciens et nouveaux processus se chevauchent pendant le déploiement, ou si les sessions et tickets consommés doivent survivre à un redémarrage. Un seul processus actif peut utiliser la mémoire même avec des callbacks inter-domaines, mais tout état est local au processus et perdu au redémarrage.

## Conserver la configuration de retour arrière et créer l'environnement v1

Ne démarrez pas v1 avec l'ancien fichier d'environnement. Conservez `stargate.env` inchangé ainsi que le conteneur v0.12.0. Ces commandes créent une copie de retour arrière protégée et un fichier v1 distinct, refusent d'écraser toute sortie existante et retirent les anciens paramètres uniquement du fichier v1 :

```bash
set -eu
old_env=./stargate.env
rollback_env=./stargate-v0.12.0.env
v1_env=./stargate-v1.env
old_container=${STARGATE_OLD_CONTAINER:-stargate}

test ! -e "$rollback_env"
test ! -e "$v1_env"
umask 077

# Si le fichier manque, exporter de manière protégée l'environnement du conteneur existant.
if [ ! -e "$old_env" ]; then
  export_tmp=$(mktemp "${old_env}.tmp.XXXXXX")
  trap 'rm -f "$export_tmp"' 0 1 2 15
  docker inspect --format '{{range .Config.Env}}{{println .}}{{end}}' "$old_container" > "$export_tmp"
  ln "$export_tmp" "$old_env"
  rm -f "$export_tmp"
  trap - 0 1 2 15
fi

test -f "$old_env"
chmod 600 "$old_env"
(set -C; cat "$old_env" > "$rollback_env")
(set -C; awk '
  /^[[:space:]]*WARDEN_OTP_(ENABLED|SECRET_KEY)[[:space:]]*(=|$)/ { next }
  /^[[:space:]]*PORT[[:space:]]*(=|$)/ { print "PORT=8080"; next }
  { print }
' "$old_env" > "$v1_env")
```

Le TOTP via Herald exige que Stargate résolve les utilisateurs authentifiés par Warden ; configurez donc aussi `WARDEN_ENABLED=true` et `WARDEN_URL`.

v1 refuse `WARDEN_OTP_ENABLED` et `WARDEN_OTP_SECRET_KEY` dès qu'une variable est présente, même vide ou à `false`. Ne les ajoutez pas à `stargate-v1.env`. Pour TOTP, configurez Stargate avec `HERALD_ENABLED=true`, `HERALD_TOTP_ENABLED=true`, `HERALD_URL` et une authentification de service Herald prise en charge. Configurez Herald pour relayer TOTP vers herald-totp et fournissez à herald-totp son propre `HERALD_TOTP_ENCRYPTION_KEY` ; ne réutilisez pas automatiquement l'ancien secret Warden. Voir la [configuration](CONFIG.md).

Validez et déployez v1 avec `--env-file ./stargate-v1.env`. Conservez `stargate-v0.12.0.env`, le fichier original inchangé et l'ancien conteneur jusqu'à la fin de la fenêtre de retour arrière.

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
