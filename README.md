# Stargate - Forward Auth Service

[![License](https://img.shields.io/badge/license-Apache%202.0-blue.svg)](LICENSE)
[![Go Version](https://img.shields.io/badge/go-1.25+-blue.svg)](https://golang.org)
[![codecov](https://codecov.io/gh/soulteary/stargate/branch/main/graph/badge.svg)](https://codecov.io/gh/soulteary/stargate)

> **🚀 Your Gateway to Secure Microservices**

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

## 📋 Table of Contents

- [Features](#features)
- [Quick Start](#quick-start)
- [Configuration](#configuration)
- [Documentation](#documentation)
- [API Documentation](#api-documentation)
- [Deployment Guide](#deployment-guide)
- [Development Guide](#development-guide)
- [License](#license)

## ✨ Features

### 🔐 Enterprise-Grade Security
- **Multiple Password Encryption Algorithms**: Choose from plaintext (testing), bcrypt, MD5, SHA512, and more
- **Secure Session Management**: Cookie-based sessions with customizable domain and expiration
- **Flexible Authentication**: Support for both password-based and session-based authentication

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

## 🚀 Quick Start

Get Stargate up and running in **under 2 minutes**!

### Using Docker Compose (Recommended)

**Step 1:** Clone the repository
```bash
git clone <repository-url>
cd forward-auth
```

**Step 2:** Configure your authentication (edit `codes/docker-compose.yml`)
```yaml
services:
  stargate:
    environment:
      - AUTH_HOST=auth.example.com
      - PASSWORDS=plaintext:yourpassword1|yourpassword2
```

**Step 3:** Start the service
```bash
cd codes
docker-compose up -d
```

**That's it!** Your authentication service is now running. 🎉

### Local Development

1. Ensure Go 1.25 or higher is installed

2. Navigate to the project directory:
```bash
cd codes
```

3. Run the local startup script:
```bash
chmod +x start-local.sh
./start-local.sh
```

4. Access the login page:
```
http://localhost:8080/_login?callback=localhost
```

## ⚙️ Configuration

Stargate uses a simple, environment variable-based configuration system. No complex YAML files or config parsing—just set environment variables and you're ready to go.

### Required Configuration

| Environment Variable | Description | Example |
|---------------------|-------------|---------|
| `AUTH_HOST` | Hostname of the authentication service | `auth.example.com` |
| `PASSWORDS` | Password configuration, format: `algorithm:password1\|password2\|password3` | `plaintext:test123\|admin456` |

### Optional Configuration

| Environment Variable | Description | Default | Example |
|---------------------|-------------|---------|---------|
| `DEBUG` | Enable debug mode | `false` | `true` |
| `LANGUAGE` | Interface language | `en` | `zh` (Chinese) or `en` (English) |
| `LOGIN_PAGE_TITLE` | Login page title | `Stargate - Login` | `My Auth Service` |
| `LOGIN_PAGE_FOOTER_TEXT` | Login page footer text | `Copyright © 2024 - Stargate` | `© 2024 My Company` |
| `USER_HEADER_NAME` | User header name set after successful authentication | `X-Forwarded-User` | `X-Authenticated-User` |
| `COOKIE_DOMAIN` | Cookie domain (for cross-domain session sharing) | Empty (not set) | `.example.com` |
| `PORT` | Service listening port (local development only) | `80` | `8080` |

### Password Configuration Format

Password configuration uses the following format:
```
algorithm:password1|password2|password3
```

Supported algorithms:
- `plaintext`: Plain text password (testing only)
- `bcrypt`: BCrypt hash
- `md5`: MD5 hash
- `sha512`: SHA512 hash

Examples:
```bash
# Plain text passwords (multiple)
PASSWORDS=plaintext:test123|admin456|user789

# BCrypt hash
PASSWORDS=bcrypt:$2a$10$N9qo8uLOickgx2ZMRZoMyeIjZAgcfl7p92ldGxad68LJZdL17lhWy

# MD5 hash
PASSWORDS=md5:5e884898da28047151d0e56f8dc6292773603d0d6aabbdd62a11ef721d1542d8
```

**For detailed configuration, see: [docs/enUS/CONFIG.md](docs/enUS/CONFIG.md)**

## 📚 Documentation

Comprehensive documentation is available to help you get the most out of Stargate:

- 📐 **[Architecture Document](docs/enUS/ARCHITECTURE.md)** - Deep dive into technical architecture and design decisions
- 🔌 **[API Document](docs/enUS/API.md)** - Complete API endpoint reference with examples
- ⚙️ **[Configuration Reference](docs/enUS/CONFIG.md)** - Detailed configuration options and best practices
- 🚀 **[Deployment Guide](docs/enUS/DEPLOYMENT.md)** - Production deployment strategies and recommendations

## 📚 API Documentation

### Authentication Check Endpoint

#### `GET /_auth`

The main authentication check endpoint for Traefik Forward Auth.

**Request Headers:**
- `Stargate-Password` (optional): Password authentication for API requests
- `Cookie: stargate_session_id` (optional): Session authentication for Web requests

**Response:**
- `200 OK`: Authentication successful, sets `X-Forwarded-User` header (or configured user header name)
- `401 Unauthorized`: Authentication failed
- `500 Internal Server Error`: Server error

**Notes:**
- HTML requests redirect to login page on authentication failure
- API requests (JSON/XML) return 401 error on authentication failure

### Login Endpoint

#### `GET /_login`

Displays the login page.

**Query Parameters:**
- `callback` (optional): Callback URL after successful login

**Response:**
- Returns login page HTML

#### `POST /_login`

Handles login requests.

**Form Data:**
- `password`: User password
- `callback` (optional): Callback URL after successful login

**Callback Retrieval Priority:**
1. From Cookie (if previously set)
2. From form data
3. From query parameters
4. If none of the above, and the origin domain differs from the authentication service domain, use the origin domain as callback

**Response:**
- `200 OK`: Login successful
  - If callback exists, redirects to `{callback}/_session_exchange?id={session_id}`
  - If no callback, returns success message (HTML or JSON format, depending on request type)
- `401 Unauthorized`: Incorrect password
- `500 Internal Server Error`: Server error

### Logout Endpoint

#### `GET /_logout`

Logs out the current user and destroys the session.

**Response:**
- `200 OK`: Logout successful, returns "Logged out"

### Session Exchange Endpoint

#### `GET /_session_exchange`

Used for cross-domain session sharing. Sets the specified session ID cookie and redirects.

**Query Parameters:**
- `id` (required): Session ID to set

**Response:**
- `302 Redirect`: Redirects to root path
- `400 Bad Request`: Missing session ID

### Health Check Endpoint

#### `GET /health`

Service health check endpoint.

**Response:**
- `200 OK`: Service is healthy

### Root Endpoint

#### `GET /`

Root path, displays service information.

**For detailed API documentation, see: [docs/enUS/API.md](docs/enUS/API.md)**

## 🐳 Deployment Guide

### Docker Deployment

#### Build Image

```bash
cd codes
docker build -t stargate:latest .
```

#### Run Container

```bash
docker run -d \
  --name stargate \
  -p 80:80 \
  -e AUTH_HOST=auth.example.com \
  -e PASSWORDS=plaintext:yourpassword \
  stargate:latest
```

### Docker Compose Deployment

The project provides a `docker-compose.yml` example configuration, including Stargate service and example whoami service:

```bash
cd codes
docker-compose up -d
```

### Traefik Integration

Configure Traefik labels in `docker-compose.yml`:

```yaml
services:
  stargate:
    image: stargate:latest
    environment:
      - AUTH_HOST=auth.example.com
      - PASSWORDS=plaintext:yourpassword
    networks:
      - traefik
    labels:
      - "traefik.enable=true"
      - "traefik.docker.network=traefik"
      - "traefik.http.routers.auth.entrypoints=http"
      - "traefik.http.routers.auth.rule=Host(`auth.example.com`) || Path(`/_session_exchange`)"
      - "traefik.http.middlewares.stargate.forwardauth.address=http://stargate/_auth"

  your-service:
    image: your-service:latest
    networks:
      - traefik
    labels:
      - "traefik.enable=true"
      - "traefik.docker.network=traefik"
      - "traefik.http.routers.your-service.entrypoints=http"
      - "traefik.http.routers.your-service.rule=Host(`your-service.example.com`)"
      - "traefik.http.routers.your-service.middlewares=stargate"  # Use Stargate middleware

networks:
  traefik:
    external: true
```

### Production Recommendations

1. **Use HTTPS**: In production, ensure HTTPS is configured via Traefik
2. **Use Strong Password Algorithms**: Avoid `plaintext`, recommend using `bcrypt` or `sha512`
3. **Set Cookie Domain**: If you need to share sessions across multiple subdomains, set `COOKIE_DOMAIN`
4. **Log Management**: Configure appropriate log rotation and monitoring
5. **Resource Limits**: Set appropriate CPU and memory limits for containers

**For detailed deployment guide, see: [docs/enUS/DEPLOYMENT.md](docs/enUS/DEPLOYMENT.md)**

## 💻 Development Guide

### Project Structure

```
codes/
├── src/
│   ├── cmd/
│   │   └── stargate/          # Main program entry point
│   │       ├── main.go        # Program entry
│   │       ├── server.go      # Server configuration
│   │       └── constants.go  # Constant definitions
│   ├── internal/
│   │   ├── auth/              # Authentication logic
│   │   ├── config/            # Configuration management
│   │   ├── handlers/          # HTTP handlers
│   │   ├── i18n/              # Internationalization
│   │   ├── middleware/        # Middleware
│   │   ├── secure/            # Password encryption algorithms
│   │   └── web/               # Web templates and static resources
│   ├── go.mod
│   └── go.sum
├── Dockerfile
├── docker-compose.yml
└── start-local.sh
```

### Local Development

1. Install dependencies:
```bash
cd codes
go mod download
```

2. Run tests:
```bash
go test ./...
```

3. Start development server:
```bash
./start-local.sh
```

### Adding New Password Algorithms

1. Create a new algorithm implementation in `src/internal/secure/` directory:
```go
package secure

type NewAlgorithmResolver struct{}

func (r *NewAlgorithmResolver) Check(h string, password string) bool {
    // Implement password verification logic
    return false
}
```

2. Register the algorithm in `src/internal/config/validation.go`:
```go
SupportedAlgorithms = map[string]secure.HashResolver{
    // ...
    "newalgorithm": &secure.NewAlgorithmResolver{},
}
```

### Adding New Language Support

1. Add language constant in `src/internal/i18n/i18n.go`:
```go
const (
    LangEN Language = "en"
    LangZH Language = "zh"
    LangFR Language = "fr"  // New
)
```

2. Add translation mapping:
```go
var translations = map[Language]map[string]string{
    // ...
    LangFR: {
        "error.auth_required": "Authentification requise",
        // ...
    },
}
```

3. Add language option in `src/internal/config/config.go`:
```go
Language = EnvVariable{
    PossibleValues: []string{"en", "zh", "fr"},  // Add new language
}
```

## 📝 License

This project is licensed under the Apache License 2.0. See the [LICENSE](codes/LICENSE) file for details.

## 🤝 Contributing

We welcome contributions! Whether it's:
- 🐛 Bug reports
- 💡 Feature suggestions
- 📝 Documentation improvements
- 🔧 Code contributions

Please feel free to open an Issue or submit a Pull Request. Every contribution makes Stargate better!

---

## ⚠️ Production Checklist

Before deploying to production, ensure you've completed these security best practices:

- ✅ **Use Strong Passwords**: Avoid `plaintext`, use `bcrypt` or `sha512` for password hashing
- ✅ **Enable HTTPS**: Configure HTTPS via Traefik or your reverse proxy
- ✅ **Set Cookie Domain**: Configure `COOKIE_DOMAIN` for proper session management across subdomains
- ✅ **Monitor & Log**: Set up appropriate logging and monitoring for your deployment
- ✅ **Regular Updates**: Keep Stargate updated to the latest version for security patches
