# Documentazione di Sicurezza

> 🌐 **Language / 语言**: [English](../enUS/SECURITY.md) | [中文](../zhCN/SECURITY.md) | [Français](../frFR/SECURITY.md) | [Italiano](SECURITY.md) | [日本語](../jaJP/SECURITY.md) | [Deutsch](../deDE/SECURITY.md) | [한국어](../koKR/SECURITY.md)

Questo documento spiega le funzionalità di sicurezza di Stargate, la configurazione di sicurezza e le migliori pratiche.

> ⚠️ **Nota**: Questa documentazione è in fase di traduzione. Per la versione completa, consulta la [versione inglese](../enUS/SECURITY.md).

## Funzionalità di Sicurezza Implementate

1. **Protezione Forward Auth**: Livello di autenticazione centralizzato per proteggere i servizi backend
2. **Algoritmi di password multipli**: bcrypt e plaintext solo per sviluppo locale; MD5 e SHA-512 senza salt sono rifiutati
3. **Gestione sicura delle sessioni**: Sessioni basate su Cookie con dominio e scadenza configurabili
4. **Sicurezza dell'integrazione del servizio**: Comunicazione sicura con i servizi Warden e Herald utilizzando mTLS o HMAC
5. **Sicurezza della condivisione delle sessioni**: Meccanismo di scambio di sessioni cross-domain sicuro
6. **Validazione degli input**: Validazione rigorosa di tutti i parametri di input
7. **Gestione degli errori**: I messaggi di errore e gli altri dati di risposta dipendono dall'endpoint; `DEBUG` non è una garanzia generale di non divulgazione
8. **Intestazioni di risposta di sicurezza**: Aggiunge automaticamente intestazioni di risposta HTTP relative alla sicurezza
9. **Applicazione HTTPS**: Gli ambienti di produzione devono utilizzare HTTPS
10. **Integrazione OTP**: Integrazione sicura con Herald per l'autenticazione OTP/codice di verifica

## Risposte di errore e codici di verifica di debug

Stargate non espone un'impostazione `MODE`. Le risposte di errore dipendono dall'endpoint.
I dettagli operativi vengono normalmente registrati nei log, ma `DEBUG` non deve essere
considerato una garanzia generale di sanitizzazione delle risposte o di non divulgazione.

Quando Stargate è eseguito con `DEBUG=true`, il servizio Herald configurato è in modalità di test
(`HERALD_TEST_MODE`) e restituisce un `debug_code` non vuoto, Stargate include quel codice di test
come `debug_code` nella risposta riuscita a `POST /_send_verify_code`. La pagina di accesso inclusa
mostra il codice e lo inserisce automaticamente nel campo di verifica.

Questa combinazione divulga al client richiedente un codice di verifica equivalente a una credenziale.
Usarla solo per sviluppo locale o test, mai in produzione. In produzione impostare Stargate
`DEBUG=false` e mantenere disabilitato `HERALD_TEST_MODE` in Herald.

Per maggiori dettagli, consulta la [versione inglese](../enUS/SECURITY.md).

## Configurazione sensibile alla sicurezza

- Usa `COOKIE_SECURE=true` con HTTPS e limita `TRUSTED_PROXIES` agli indirizzi proxy conosciuti.
- Configura `CALLBACK_ALLOWED_HOSTS`; i ticket tra domini richiedono un `SESSION_EXCHANGE_SECRET` di almeno 32 caratteri.
- Per Warden e Herald, ID e segreto HMAC devono essere impostati insieme, così come certificato client mTLS e relativa chiave.
- Consulta [CONFIG.md](CONFIG.md) per l'elenco sincronizzato delle variabili.

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

## Confine di attendibilità del deployment

Forward Auth protegge solo le richieste che attraversano realmente il reverse proxy configurato. Non esporre direttamente le porte dei servizi backend. Limitarne l'accesso alla rete del proxy e rimuovere gli header di identità o credenziali forniti dal client prima dell'inoltro.
