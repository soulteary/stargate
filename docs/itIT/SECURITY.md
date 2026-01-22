# Documentazione di Sicurezza

> 🌐 **Language / 语言**: [English](../enUS/SECURITY.md) | [中文](../zhCN/SECURITY.md) | [Français](../frFR/SECURITY.md) | [Italiano](SECURITY.md) | [日本語](../jaJP/SECURITY.md) | [Deutsch](../deDE/SECURITY.md) | [한국어](../koKR/SECURITY.md)

Questo documento spiega le funzionalità di sicurezza di Stargate, la configurazione di sicurezza e le migliori pratiche.

> ⚠️ **Nota**: Questa documentazione è in fase di traduzione. Per la versione completa, consulta la [versione inglese](../enUS/SECURITY.md).

## Funzionalità di Sicurezza Implementate

1. **Protezione Forward Auth**: Livello di autenticazione centralizzato per proteggere i servizi backend
2. **Algoritmi di password multipli**: Supporto per bcrypt, SHA512, MD5 e plaintext (solo sviluppo)
3. **Gestione sicura delle sessioni**: Sessioni basate su Cookie con dominio e scadenza configurabili
4. **Sicurezza dell'integrazione del servizio**: Comunicazione sicura con i servizi Warden e Herald utilizzando mTLS o HMAC
5. **Sicurezza della condivisione delle sessioni**: Meccanismo di scambio di sessioni cross-domain sicuro
6. **Validazione degli input**: Validazione rigorosa di tutti i parametri di input
7. **Gestione degli errori**: La modalità produzione nasconde informazioni dettagliate sugli errori
8. **Intestazioni di risposta di sicurezza**: Aggiunge automaticamente intestazioni di risposta HTTP relative alla sicurezza
9. **Applicazione HTTPS**: Gli ambienti di produzione devono utilizzare HTTPS
10. **Integrazione OTP**: Integrazione sicura con Herald per l'autenticazione OTP/codice di verifica

Per maggiori dettagli, consulta la [versione inglese](../enUS/SECURITY.md).

## Segnalazione di Vulnerabilità

Se scopri una vulnerabilità di sicurezza, segnalala tramite:

1. **GitHub Security Advisory** (Preferito)
   - Vai alla scheda [Security](https://github.com/soulteary/stargate/security) nel repository
   - Clicca su "Report a vulnerability"
   - Compila il modulo di consulenza sulla sicurezza

2. **Email** (Se GitHub Security Advisory non è disponibile)
   - Invia un'email ai maintainer del progetto
   - Includi una descrizione dettagliata della vulnerabilità

**Si prega di non segnalare vulnerabilità di sicurezza tramite problemi GitHub pubblici.**
