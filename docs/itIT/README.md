# Indice della Documentazione

Benvenuto nella documentazione del servizio Stargate Forward Auth.

## 🌐 Documentazione Multilingue

- [English](../enUS/README.md) | [中文](../zhCN/README.md) | [Français](../frFR/README.md) | [Italiano](README.md) | [日本語](../jaJP/README.md) | [Deutsch](../deDE/README.md) | [한국어](../koKR/README.md)

## 📚 Elenco Documenti

### Documenti Principali

- **[README.md](../../README.itIT.md)** - Panoramica del progetto e guida rapida
- **[ARCHITECTURE.md](ARCHITECTURE.md)** - Architettura tecnica e decisioni di progettazione

### Documenti Dettagliati

- **[PERFORMANCE.md](../enUS/PERFORMANCE.md)** - Redis rate-limit measurements (English)

- **[API.md](API.md)** - Documentazione completa degli endpoint API
  - Endpoint di verifica dell'autenticazione
  - Endpoint di login e logout
  - Endpoint di scambio sessione
  - Endpoint di controllo dello stato
  - Formati di risposta di errore
  - Esempi di flusso di autenticazione

- **[CONFIG.md](CONFIG.md)** - Riferimento di configurazione
  - Metodi di configurazione
  - Elementi di configurazione richiesti
  - Elementi di configurazione opzionali
  - Dettagli di configurazione della password
  - Esempi di configurazione
  - Best practice di configurazione

- **[DEPLOYMENT.md](DEPLOYMENT.md)** - Guida al deployment
  - Deployment Docker
  - Deployment Docker Compose
  - Integrazione Traefik
  - Deployment in produzione
  - Monitoraggio e manutenzione
  - Risoluzione dei problemi

## 🚀 Navigazione Rapida

### Per Iniziare

1. Leggi [README.itIT.md](../../README.itIT.md) per comprendere il progetto
2. Controlla la sezione [Quick Start](../../README.itIT.md#-avvio-rapido)
3. Fai riferimento a [Configurazione](../../README.itIT.md#-configurazione-di-base) per configurare il servizio

### Sviluppatori

1. Leggi [ARCHITECTURE.md](ARCHITECTURE.md) per comprendere l'architettura
2. Controlla [API.md](API.md) per comprendere le interfacce API
3. Fai riferimento alla [Guida allo Sviluppo](CONTRIBUTING.md) per lo sviluppo

### Operazioni

1. Leggi [DEPLOYMENT.md](DEPLOYMENT.md) per comprendere i metodi di deployment
2. Controlla [CONFIG.md](CONFIG.md) per comprendere le opzioni di configurazione
3. Fai riferimento alla [Risoluzione dei Problemi](DEPLOYMENT.md#risoluzione-dei-problemi) per risolvere i problemi

## 📖 Struttura dei Documenti

```
stargate/
├── README.md              # Documento principale del progetto (Inglese)
├── README.zhCN.md         # Documento principale del progetto (Cinese)
├── README.frFR.md         # Documento principale del progetto (Francese)
├── README.itIT.md         # Documento principale del progetto (Italiano)
├── README.jaJP.md         # Documento principale del progetto (Giapponese)
├── README.deDE.md         # Documento principale del progetto (Tedesco)
├── README.koKR.md         # Documento principale del progetto (Coreano)
├── docs/
│   ├── enUS/
│   │   ├── README.md       # Indice della documentazione (Inglese)
│   │   ├── ARCHITECTURE.md # Documento di architettura (Inglese)
│   │   ├── API.md          # Documento API (Inglese)
│   │   ├── CONFIG.md       # Riferimento di configurazione (Inglese)
│   │   └── DEPLOYMENT.md   # Guida al deployment (Inglese)
│   ├── zhCN/
│   │   ├── README.md       # Indice della documentazione (Cinese)
│   │   ├── ARCHITECTURE.md # Documento di architettura (Cinese)
│   │   ├── API.md          # Documento API (Cinese)
│   │   ├── CONFIG.md       # Riferimento di configurazione (Cinese)
│   │   └── DEPLOYMENT.md   # Guida al deployment (Cinese)
│   └── itIT/
│       ├── README.md       # Indice della documentazione (Italiano, questo file)
│       ├── ARCHITECTURE.md # Documento di architettura (Italiano)
│       ├── API.md          # Documento API (Italiano)
│       ├── CONFIG.md       # Riferimento di configurazione (Italiano)
│       └── DEPLOYMENT.md   # Guida al deployment (Italiano)
└── ...
```

## 🔍 Trova per Argomento

### Configurazione

- Configurazione delle variabili d'ambiente : [CONFIG.md](CONFIG.md)
- Configurazione della password : [CONFIG.md#configurazione-password](CONFIG.md#configurazione-password)
- Esempi di configurazione : [CONFIG.md#esempi-di-configurazione](CONFIG.md#esempi-di-configurazione)

### API

- Elenco degli endpoint API : [API.md](API.md)
- Flusso di autenticazione : [API.md#esempi-flusso-autenticazione](API.md#esempi-flusso-autenticazione)
- Gestione degli errori : [API.md#formato-risposta-errore](API.md#formato-risposta-errore)

### Deployment

- Deployment Docker : [DEPLOYMENT.md#deployment-docker](DEPLOYMENT.md#deployment-docker)
- Integrazione Traefik : [DEPLOYMENT.md#integrazione-traefik](DEPLOYMENT.md#integrazione-traefik)
- Ambiente di produzione : [DEPLOYMENT.md#deployment-in-produzione](DEPLOYMENT.md#deployment-in-produzione)

### Architettura

- Stack tecnologico : [ARCHITECTURE.md#stack-tecnologico](ARCHITECTURE.md#stack-tecnologico)
- Struttura del progetto : [ARCHITECTURE.md#struttura-del-progetto](ARCHITECTURE.md#struttura-del-progetto)
- Componenti principali : [ARCHITECTURE.md#componenti-principali](ARCHITECTURE.md#componenti-principali)

## 💡 Raccomandazioni d'Uso

1. **Utenti per la prima volta** : Inizia con [README.itIT.md](../../README.itIT.md) e segui la guida rapida
2. **Configurare il servizio** : Fai riferimento a [CONFIG.md](CONFIG.md) per comprendere tutte le opzioni di configurazione
3. **Integrare Traefik** : Controlla la sezione di integrazione Traefik in [DEPLOYMENT.md](DEPLOYMENT.md)
4. **Sviluppare estensioni** : Leggi [ARCHITECTURE.md](ARCHITECTURE.md) per comprendere la progettazione dell'architettura
5. **Risoluzione dei problemi** : Controlla [DEPLOYMENT.md#risoluzione-dei-problemi](DEPLOYMENT.md#risoluzione-dei-problemi)

## 📝 Aggiornamenti dei Documenti

La documentazione viene aggiornata continuamente man mano che il progetto evolve. Se trovi errori o hai bisogno di aggiunte, invia un Issue o una Pull Request.

## 🤝 Contribuire

Sono benvenuti i miglioramenti alla documentazione :

1. Trova errori o aree che necessitano di miglioramento
2. Invia un Issue descrivendo il problema
3. O invia direttamente una Pull Request
