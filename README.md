# Stargate - Forward Auth Service

[![License](https://img.shields.io/badge/license-Apache%202.0-blue.svg)](LICENSE)
[![Go Version](https://img.shields.io/badge/go-1.25+-blue.svg)](https://golang.org)
[![codecov](https://codecov.io/gh/soulteary/stargate/branch/main/graph/badge.svg)](https://codecov.io/gh/soulteary/stargate)
[![Go Report Card](https://goreportcard.com/badge/github.com/soulteary/stargate)](https://goreportcard.com/report/github.com/soulteary/stargate)

> **🚀 Your Gateway to Secure Microservices**

![Stargate](.github/assets/banner.jpg)

Stargate is a production-ready, lightweight Forward Auth Service designed to be the **single point of authentication** for your entire infrastructure. Built with Go and optimized for performance, Stargate seamlessly integrates with Traefik and other reverse proxies to protect your backend services—**without writing a single line of auth code in your applications**.

## 🌐 Multi-language Documentation

- [English](README.md) | [中文](README.zhCN.md) | [Français](README.frFR.md) | [Italiano](README.itIT.md) | [日本語](README.jaJP.md) | [Deutsch](README.deDE.md) | [한국어](README.koKR.md)

![Preview](.github/assets/preview.png)

### 🎯 Why Stargate?

Tired of implementing authentication logic in every service? Stargate solves this by centralizing authentication at the edge, allowing you to:

- ✅ **Protect multiple services** with a single authentication layer
- ✅ **Reduce code complexity** by removing auth logic from your applications
- ✅ **Deploy in minutes** with Docker and simple configuration
- ✅ **Scale effortlessly** with minimal resource footprint
- ✅ **Maintain security** with multiple encryption algorithms and secure session management

### 💼 Use Cases

Stargate is perfect for:

- **Microservices Architecture**: Protect multiple backend services without modifying application code
- **Multi-Domain Applications**: Share authentication sessions across different domains and subdomains
- **Internal Tools & Dashboards**: Quickly add authentication to internal services and admin panels
- **API Gateway Integration**: Use with Traefik, Nginx, or other reverse proxies as a unified auth layer
- **Development & Testing**: Simple password-based auth for development environments
- **Enterprise Authentication**: Integration with Warden (user whitelist) and Herald (OTP/verification codes) for production-grade authentication

## ✨ Features

### 🔐 Enterprise-Grade Security
- **Multiple Password Encryption Algorithms**: Choose from plaintext (testing), bcrypt, MD5, SHA512, and more
- **Secure Session Management**: Cookie-based sessions with customizable domain and expiration
- **Flexible Authentication**: Support for both password-based and session-based authentication
- **OTP/Verification Code Support**: Integration with Herald service for SMS/Email verification codes
- **User Whitelist Management**: Integration with Warden service for user access control

### 🌐 Advanced Capabilities
- **Cross-Domain Session Sharing**: Seamlessly share authentication sessions across different domains/subdomains
- **Multi-Language Support**: Built-in English and Chinese interfaces, easily extensible for more languages
- **Customizable UI**: Brand your login page with custom titles and footer text

### 🚀 Performance & Reliability
- **Lightweight & Fast**: Built on Go and Fiber framework for exceptional performance
- **Minimal Resource Usage**: Low memory footprint, perfect for containerized environments
- **Production Ready**: Battle-tested architecture designed for reliability

### 📦 Developer Experience
- **Docker First**: Complete Docker image and docker-compose configuration out of the box
- **Traefik Native**: Zero-configuration Traefik Forward Auth middleware integration
- **Simple Configuration**: Environment variable-based configuration, no complex files needed

## 📋 Table of Contents

- [Quick Start](#-quick-start)
- [Documentation](#-documentation)
- [Basic Configuration](#-basic-configuration)
- [Optional Service Integration](#-optional-service-integration)
- [Production Checklist](#-production-checklist)
- [License](#-license)

## 🚀 Quick Start

Get Stargate up and running in **under 2 minutes**!

### Using Docker Compose (Recommended)

**Step 1:** Clone the repository
```bash
git clone <repository-url>
cd stargate
```

**Step 2:** Configure your authentication (edit `docker-compose.yml`)

**Option A: Password Authentication (Simple)**
```yaml
services:
  stargate:
    environment:
      - AUTH_HOST=auth.example.com
      - PASSWORDS=plaintext:yourpassword1|yourpassword2
```

**Option B: Warden + Herald OTP Authentication (Production)**
```yaml
services:
  stargate:
    environment:
      - AUTH_HOST=auth.example.com
      - WARDEN_ENABLED=true
      - WARDEN_URL=http://warden:8080
      - WARDEN_API_KEY=your-warden-api-key
      - HERALD_ENABLED=true
      - HERALD_URL=http://herald:8080
      - HERALD_HMAC_SECRET=your-herald-hmac-secret
```

**Step 3:** Start the service
```bash
docker-compose up -d
```

**That's it!** Your authentication service is now running. 🎉

### Local Development

For local development, ensure Go 1.25+ is installed, then:

```bash
chmod +x start-local.sh
./start-local.sh
```

Access the login page at `http://localhost:8080/_login?callback=localhost`

## 📚 Documentation

Comprehensive documentation is available to help you get the most out of Stargate:

### Core Documents

- 📐 **[Architecture Document](docs/enUS/ARCHITECTURE.md)** - Deep dive into technical architecture and design decisions
- 🔌 **[API Document](docs/enUS/API.md)** - Complete API endpoint reference with examples
- ⚙️ **[Configuration Reference](docs/enUS/CONFIG.md)** - Detailed configuration options and best practices
- 🚀 **[Deployment Guide](docs/enUS/DEPLOYMENT.md)** - Production deployment strategies and recommendations

### Quick Reference

- **API Endpoints**: `GET /_auth` (auth check), `GET /_login` (login page), `POST /_login` (login), `GET /_logout` (logout), `GET /_session_exchange` (cross-domain), `GET /health` (health check)
- **Deployment**: Docker Compose recommended for quick start. See [DEPLOYMENT.md](docs/enUS/DEPLOYMENT.md) for production deployment.
- **Development**: For development-related documentation, see [ARCHITECTURE.md](docs/enUS/ARCHITECTURE.md)

## ⚙️ Basic Configuration

Stargate uses environment variables for configuration. Here are the most common settings:

### Required Configuration

- **`AUTH_HOST`**: Hostname of the authentication service (e.g., `auth.example.com`)
- **`PASSWORDS`**: Password configuration in format `algorithm:password1|password2|password3`

### Common Configuration Examples

```bash
# Simple password authentication
AUTH_HOST=auth.example.com
PASSWORDS=plaintext:test123|admin456

# Using BCrypt hash
PASSWORDS=bcrypt:$2a$10$N9qo8uLOickgx2ZMRZoMyeIjZAgcfl7p92ldGxad68LJZdL17lhWy

# Cross-domain session sharing
COOKIE_DOMAIN=.example.com

# Customize login page
LOGIN_PAGE_TITLE=My Auth Service
LANGUAGE=zh  # or 'en'
```

**Supported password algorithms:** `plaintext` (testing only), `bcrypt`, `md5`, `sha512`

**For complete configuration reference, see: [docs/enUS/CONFIG.md](docs/enUS/CONFIG.md)**

## 🔗 Optional Service Integration

Stargate can be used completely independently, or optionally integrate with the following services:

### Warden Integration (User Whitelist)

Provides user whitelist management and user information. When enabled, Stargate queries Warden to verify if a user is in the allowed list.

```bash
WARDEN_ENABLED=true
WARDEN_URL=http://warden:8080
WARDEN_API_KEY=your-api-key
```

### Herald Integration (OTP/Verification Codes)

Provides OTP/verification code services. When enabled, Stargate calls Herald to create, send, and verify verification codes (SMS/Email).

```bash
HERALD_ENABLED=true
HERALD_URL=http://herald:8080
HERALD_HMAC_SECRET=your-hmac-secret  # Production
# or
HERALD_API_KEY=your-api-key  # Development
```

**Note:** Both integrations are optional. Stargate can be used independently with password authentication.

**For complete integration guide, see: [docs/enUS/ARCHITECTURE.md](docs/enUS/ARCHITECTURE.md)**

## ⚠️ Production Checklist

Before deploying to production:

- ✅ Use strong password algorithms (`bcrypt` or `sha512`, avoid `plaintext`)
- ✅ Enable HTTPS via Traefik or your reverse proxy
- ✅ Set `COOKIE_DOMAIN` for proper session management across subdomains
- ✅ For advanced features, optionally integrate Warden + Herald for OTP authentication
- ✅ Use HMAC signatures or mTLS for Stargate ↔ Herald/Warden communication
- ✅ Set up appropriate logging and monitoring
- ✅ Keep Stargate updated to the latest version

## 🎯 Design Principles

Stargate is designed to be used independently:

- **Standalone Usage**: Can run independently using password authentication mode without any external dependencies
- **Optional Integration**: Can optionally integrate with Warden (user whitelist) and Herald (OTP/verification codes) services
- **High Performance**: forwardAuth main path only verifies sessions, ensuring fast response
- **Flexibility**: Supports multiple authentication modes, choose according to your needs

## 📝 License

This project is licensed under the Apache License 2.0. See the [LICENSE](LICENSE) file for details.

## 🤝 Contributing

We welcome contributions! Whether it's bug reports, feature suggestions, documentation improvements, or code contributions, please feel free to open an Issue or submit a Pull Request.
