# Documentation Index

Welcome to the Stargate Forward Auth Service documentation.

## 🌐 Multi-language Documentation

- [English](README.md) | [中文](../zhCN/README.md) | [Français](../frFR/README.md) | [Italiano](../itIT/README.md) | [日本語](../jaJP/README.md) | [Deutsch](../deDE/README.md) | [한국어](../koKR/README.md)

## 📚 Document List

### Core Documents

- **[README.md](../../README.md)** - Project overview and quick start guide
- **[ARCHITECTURE.md](ARCHITECTURE.md)** - Technical architecture and design decisions

### Detailed Documents

- **[API.md](API.md)** - Complete API endpoint documentation
  - Authentication check endpoints
  - Login and logout endpoints
  - Session exchange endpoints
  - Health check endpoints
  - Error response formats
  - Authentication flow examples

- **[CONFIG.md](CONFIG.md)** - Configuration reference
  - Configuration methods
  - Required configuration items
  - Optional configuration items
  - Password configuration details
  - Configuration examples
  - Configuration best practices

- **[DEPLOYMENT.md](DEPLOYMENT.md)** - Deployment guide
  - Docker deployment
  - Docker Compose deployment
  - Traefik integration
  - Production deployment
  - Monitoring and maintenance
  - Troubleshooting

## 🚀 Quick Navigation

### Getting Started

1. Read [README.md](../../README.md) to understand the project
2. Check the [Quick Start](../../README.md#quick-start) section
3. Refer to [Configuration](../../README.md#configuration) to configure the service

### Developers

1. Read [ARCHITECTURE.md](ARCHITECTURE.md) to understand the architecture
2. Check [API.md](API.md) to understand the API interfaces
3. Refer to [Development Guide](../../README.md#development-guide) for development

### Operations

1. Read [DEPLOYMENT.md](DEPLOYMENT.md) to understand deployment methods
2. Check [CONFIG.md](CONFIG.md) to understand configuration options
3. Refer to [Troubleshooting](DEPLOYMENT.md#troubleshooting) to solve problems

## 📖 Document Structure

```
codes/
├── README.md              # Main project document (English)
├── README.zhCN.md         # Main project document (Chinese)
├── docs/
│   ├── enUS/
│   │   ├── README.md       # Documentation index (English, this file)
│   │   ├── ARCHITECTURE.md # Architecture document (English)
│   │   ├── API.md          # API document (English)
│   │   ├── CONFIG.md       # Configuration reference (English)
│   │   └── DEPLOYMENT.md   # Deployment guide (English)
│   └── zhCN/
│       ├── README.md       # Documentation index (Chinese)
│       ├── ARCHITECTURE.md # Architecture document (Chinese)
│       ├── API.md          # API document (Chinese)
│       ├── CONFIG.md       # Configuration reference (Chinese)
│       └── DEPLOYMENT.md   # Deployment guide (Chinese)
└── ...
```

## 🔍 Find by Topic

### Configuration Related

- Environment variable configuration: [CONFIG.md](CONFIG.md)
- Password configuration: [CONFIG.md#password-configuration](CONFIG.md#password-configuration)
- Configuration examples: [CONFIG.md#configuration-examples](CONFIG.md#configuration-examples)

### API Related

- API endpoint list: [API.md](API.md)
- Authentication flow: [API.md#authentication-flow-examples](API.md#authentication-flow-examples)
- Error handling: [API.md#error-response-format](API.md#error-response-format)

### Deployment Related

- Docker deployment: [DEPLOYMENT.md#docker-deployment](DEPLOYMENT.md#docker-deployment)
- Traefik integration: [DEPLOYMENT.md#traefik-integration](DEPLOYMENT.md#traefik-integration)
- Production environment: [DEPLOYMENT.md#production-deployment](DEPLOYMENT.md#production-deployment)

### Architecture Related

- Technology stack: [ARCHITECTURE.md#technology-stack](ARCHITECTURE.md#technology-stack)
- Project structure: [ARCHITECTURE.md#project-structure](ARCHITECTURE.md#project-structure)
- Core components: [ARCHITECTURE.md#core-components](ARCHITECTURE.md#core-components)

## 💡 Usage Recommendations

1. **First-time users**: Start with [README.md](../../README.md) and follow the quick start guide
2. **Configure service**: Refer to [CONFIG.md](CONFIG.md) to understand all configuration options
3. **Integrate Traefik**: Check the Traefik integration section in [DEPLOYMENT.md](DEPLOYMENT.md)
4. **Develop extensions**: Read [ARCHITECTURE.md](ARCHITECTURE.md) to understand the architecture design
5. **Troubleshooting**: Check [DEPLOYMENT.md#troubleshooting](DEPLOYMENT.md#troubleshooting)

## 📝 Document Updates

Documentation is continuously updated as the project evolves. If you find errors or need additions, please submit an Issue or Pull Request.

## 🤝 Contributing

Documentation improvements are welcome:

1. Find errors or areas that need improvement
2. Submit an Issue describing the problem
3. Or directly submit a Pull Request
