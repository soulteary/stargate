# Documentation de Sécurité

> 🌐 **Language / 语言**: [English](../enUS/SECURITY.md) | [中文](../zhCN/SECURITY.md) | [Français](SECURITY.md) | [Italiano](../itIT/SECURITY.md) | [日本語](../jaJP/SECURITY.md) | [Deutsch](../deDE/SECURITY.md) | [한국어](../koKR/SECURITY.md)

Ce document explique les fonctionnalités de sécurité de Stargate, la configuration de sécurité et les meilleures pratiques.

> ⚠️ **Note**: Cette documentation est en cours de traduction. Pour la version complète, consultez la [version anglaise](../enUS/SECURITY.md).

## Fonctionnalités de Sécurité Implémentées

1. **Protection Forward Auth**: Couche d'authentification centralisée pour protéger les services backend
2. **Algorithmes de mot de passe multiples**: bcrypt et plaintext uniquement en développement local ; MD5 et SHA-512 non salé sont refusés
3. **Gestion sécurisée des sessions**: Sessions basées sur Cookie avec domaine et expiration configurables
4. **Sécurité d'intégration de service**: Communication sécurisée avec les services Warden et Herald en utilisant mTLS ou HMAC
5. **Sécurité de partage de session**: Mécanisme d'échange de session cross-domain sécurisé
6. **Validation des entrées**: Validation stricte de tous les paramètres d'entrée
7. **Gestion des erreurs**: Les réponses HTTP utilisent des erreurs limitées ; `DEBUG` ne contrôle que la journalisation
8. **En-têtes de réponse de sécurité**: Ajoute automatiquement les en-têtes de réponse HTTP liés à la sécurité
9. **Application HTTPS**: Les environnements de production doivent utiliser HTTPS
10. **Intégration OTP**: Intégration sécurisée avec Herald pour l'authentification OTP/code de vérification

Pour plus de détails, consultez la [version anglaise](../enUS/SECURITY.md).

## Configuration sensible à la sécurité

- Utilisez `COOKIE_SECURE=true` avec HTTPS et limitez `TRUSTED_PROXIES` aux adresses de proxy connues.
- Configurez `CALLBACK_ALLOWED_HOSTS` ; les tickets inter-domaines exigent un `SESSION_EXCHANGE_SECRET` d'au moins 32 caractères.
- Pour Warden et Herald, l'identifiant et le secret HMAC doivent être fournis ensemble, comme le certificat client mTLS et sa clé.
- Consultez [CONFIG.md](CONFIG.md) pour la liste synchronisée des variables.

## Signalement de Vulnérabilité

Si vous découvrez une vulnérabilité de sécurité, veuillez la signaler via:

1. **GitHub Security Advisory** (Préféré)
   - Allez dans l'onglet [Security](https://github.com/soulteary/stargate/security) du dépôt
   - Cliquez sur "Report a vulnerability"
   - Remplissez le formulaire de conseil de sécurité

2. **Email** (Si GitHub Security Advisory n'est pas disponible)
   - Envoyez un email aux mainteneurs du projet
   - Incluez une description détaillée de la vulnérabilité

**Veuillez ne pas signaler les vulnérabilités de sécurité via les problèmes GitHub publics.**
