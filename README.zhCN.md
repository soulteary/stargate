# Stargate - Forward Auth Service

[![License](https://img.shields.io/badge/license-Apache%202.0-blue.svg)](LICENSE)
[![Go Version](https://img.shields.io/badge/go-1.25+-blue.svg)](https://golang.org)
[![codecov](https://codecov.io/gh/soulteary/stargate/branch/main/graph/badge.svg)](https://codecov.io/gh/soulteary/stargate)

> **🚀 微服务安全的统一入口**

Stargate 是一个生产就绪的轻量级前向认证服务，旨在成为您整个基础设施的**单一认证入口**。基于 Go 构建并针对性能优化，Stargate 可与 Traefik 等反向代理无缝集成，保护您的后端服务——**无需在应用程序中编写任何认证代码**。

## 🌐 多语言文档 / Multi-language Documentation

- [English](README.md) | [中文](README.zhCN.md) | [Français](README.frFR.md) | [Italiano](README.itIT.md) | [日本語](README.jaJP.md) | [Deutsch](README.deDE.md) | [한국어](README.koKR.md)

![预览图](.github/assets/preview.png)

### 🎯 为什么选择 Stargate？

厌倦了在每个服务中重复实现认证逻辑？Stargate 通过在边缘集中处理认证来解决这个问题，让您能够：

- ✅ **统一保护多个服务**，只需一个认证层
- ✅ **降低代码复杂度**，从应用中移除认证逻辑
- ✅ **快速部署**，使用 Docker 和简单配置，几分钟即可完成
- ✅ **轻松扩展**，资源占用极小
- ✅ **保障安全**，支持多种加密算法和安全会话管理

### 💼 使用场景

Stargate 非常适合以下场景：

- **微服务架构**：保护多个后端服务，无需修改应用代码
- **多域名应用**：在不同域名和子域名之间共享认证会话
- **内部工具与仪表板**：快速为内部服务和管理面板添加认证
- **API 网关集成**：与 Traefik、Nginx 或其他反向代理配合，作为统一的认证层
- **开发与测试**：为开发环境提供简单的基于密码的认证

## 📋 目录

- [功能特性](#功能特性)
- [快速开始](#快速开始)
- [配置说明](#配置说明)
- [文档导航](#文档导航)
- [API 文档](#api-文档)
- [部署指南](#部署指南)
- [开发指南](#开发指南)
- [许可证](#许可证)

## ✨ 功能特性

### 🔐 企业级安全
- **多种密码加密算法**：支持 plaintext（测试用）、bcrypt、MD5、SHA512 等多种加密算法
- **安全会话管理**：基于 Cookie 的会话管理，支持自定义域名和过期时间
- **灵活的认证方式**：同时支持基于密码和基于会话的认证

### 🌐 高级能力
- **跨域会话共享**：在不同域名/子域名之间无缝共享认证会话
- **多语言支持**：内置中英文界面，可轻松扩展支持更多语言
- **可定制界面**：使用自定义标题和页脚文本打造专属登录页面

### 🚀 性能与可靠性
- **轻量且快速**：基于 Go 和 Fiber 框架，性能卓越
- **资源占用极低**：内存占用小，完美适配容器化环境
- **生产就绪**：经过实战验证的架构设计，确保可靠性

### 📦 开发体验
- **Docker 优先**：开箱即用的完整 Docker 镜像和 docker-compose 配置
- **Traefik 原生支持**：零配置的 Traefik Forward Auth 中间件集成
- **简单配置**：基于环境变量的配置方式，无需复杂的配置文件

## 🚀 快速开始

**2 分钟内**让 Stargate 运行起来！

### 使用 Docker Compose（推荐）

**步骤 1：** 克隆项目
```bash
git clone <repository-url>
cd forward-auth
```

**步骤 2：** 配置认证信息（编辑 `codes/docker-compose.yml`）
```yaml
services:
  stargate:
    environment:
      - AUTH_HOST=auth.example.com
      - PASSWORDS=plaintext:yourpassword1|yourpassword2
```

**步骤 3：** 启动服务
```bash
cd codes
docker-compose up -d
```

**完成！** 您的认证服务现在已经在运行了。🎉

### 本地开发

1. 确保已安装 Go 1.25 或更高版本

2. 进入项目目录：
```bash
cd codes
```

3. 运行本地启动脚本：
```bash
chmod +x start-local.sh
./start-local.sh
```

4. 访问登录页面：
```
http://localhost:8080/_login?callback=localhost
```

## ⚙️ 配置说明

Stargate 采用简单直观的环境变量配置系统。无需复杂的 YAML 文件或配置解析——只需设置环境变量，即可开始使用。

### 必需配置

| 环境变量 | 说明 | 示例 |
|---------|------|------|
| `AUTH_HOST` | 认证服务的主机名 | `auth.example.com` |
| `PASSWORDS` | 密码配置，格式：`算法:密码1\|密码2\|密码3` | `plaintext:test123\|admin456` |

### 可选配置

| 环境变量 | 说明 | 默认值 | 示例 |
|---------|------|--------|------|
| `DEBUG` | 启用调试模式 | `false` | `true` |
| `LANGUAGE` | 界面语言 | `en` | `zh`（中文）或 `en`（英文） |
| `LOGIN_PAGE_TITLE` | 登录页面标题 | `Stargate - Login` | `我的认证服务` |
| `LOGIN_PAGE_FOOTER_TEXT` | 登录页面页脚文本 | `Copyright © 2024 - Stargate` | `© 2024 我的公司` |
| `USER_HEADER_NAME` | 认证成功后设置的用户头名称 | `X-Forwarded-User` | `X-Authenticated-User` |
| `COOKIE_DOMAIN` | Cookie 域名（用于跨域会话共享） | 空（不设置） | `.example.com` |
| `PORT` | 服务监听端口（仅本地开发） | `80` | `8080` |

### 密码配置格式

密码配置使用以下格式：
```
算法:密码1|密码2|密码3
```

支持的算法：
- `plaintext`：明文密码（仅用于测试）
- `bcrypt`：BCrypt 哈希
- `md5`：MD5 哈希
- `sha512`：SHA512 哈希

示例：
```bash
# 明文密码（多个）
PASSWORDS=plaintext:test123|admin456|user789

# BCrypt 哈希
PASSWORDS=bcrypt:$2a$10$N9qo8uLOickgx2ZMRZoMyeIjZAgcfl7p92ldGxad68LJZdL17lhWy

# MD5 哈希
PASSWORDS=md5:5e884898da28047151d0e56f8dc6292773603d0d6aabbdd62a11ef721d1542d8
```

**详细配置说明请参阅：[docs/zhCN/CONFIG.md](docs/zhCN/CONFIG.md)**

## 📚 文档导航

我们提供了全面的文档，帮助您充分利用 Stargate：

- 📐 **[架构文档](docs/zhCN/ARCHITECTURE.md)** - 深入了解技术架构和设计决策
- 🔌 **[API 文档](docs/zhCN/API.md)** - 完整的 API 端点参考和示例
- ⚙️ **[配置参考](docs/zhCN/CONFIG.md)** - 详细的配置选项和最佳实践
- 🚀 **[部署指南](docs/zhCN/DEPLOYMENT.md)** - 生产环境部署策略和建议

## 📚 API 文档

### 认证检查端点

#### `GET /_auth`

Traefik Forward Auth 的主要认证检查端点。

**请求头：**
- `Stargate-Password`（可选）：用于 API 请求的密码认证
- `Cookie: stargate_session_id`（可选）：用于 Web 请求的会话认证

**响应：**
- `200 OK`：认证成功，设置 `X-Forwarded-User` 头（或配置的用户头名称）
- `401 Unauthorized`：认证失败
- `500 Internal Server Error`：服务器错误

**说明：**
- HTML 请求认证失败时会重定向到登录页面
- API 请求（JSON/XML）认证失败时返回 401 错误

### 登录端点

#### `GET /_login`

显示登录页面。

**查询参数：**
- `callback`（可选）：登录成功后的回调 URL

**响应：**
- 返回登录页面 HTML

#### `POST /_login`

处理登录请求。

**表单数据：**
- `password`：用户密码
- `callback`（可选）：登录成功后的回调 URL

**Callback 获取优先级：**
1. 从 Cookie 中获取（如果之前已设置）
2. 从表单数据中获取
3. 从查询参数中获取
4. 如果以上都没有，且来源域名与认证服务域名不一致，则使用来源域名作为 callback

**响应：**
- `200 OK`：登录成功
  - 如果有 callback，重定向到 `{callback}/_session_exchange?id={session_id}`
  - 如果没有 callback，返回成功消息（HTML 或 JSON 格式，根据请求类型）
- `401 Unauthorized`：密码错误
- `500 Internal Server Error`：服务器错误

### 登出端点

#### `GET /_logout`

登出当前用户，销毁会话。

**响应：**
- `200 OK`：登出成功，返回 "Logged out"

### 会话交换端点

#### `GET /_session_exchange`

用于跨域会话共享。设置指定会话 ID 的 Cookie 并重定向。

**查询参数：**
- `id`（必需）：要设置的会话 ID

**响应：**
- `302 Redirect`：重定向到根路径
- `400 Bad Request`：缺少会话 ID

### 健康检查端点

#### `GET /health`

服务健康检查端点。

**响应：**
- `200 OK`：服务正常

### 根端点

#### `GET /`

根路径，显示服务信息。

**详细 API 文档请参阅：[docs/zhCN/API.md](docs/zhCN/API.md)**

## 🐳 部署指南

### Docker 部署

#### 构建镜像

```bash
cd codes
docker build -t stargate:latest .
```

#### 运行容器

```bash
docker run -d \
  --name stargate \
  -p 80:80 \
  -e AUTH_HOST=auth.example.com \
  -e PASSWORDS=plaintext:yourpassword \
  stargate:latest
```

### Docker Compose 部署

项目提供了 `docker-compose.yml` 示例配置，包含 Stargate 服务和示例的 whoami 服务：

```bash
cd codes
docker-compose up -d
```

### Traefik 集成

在 `docker-compose.yml` 中配置 Traefik 标签：

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
      - "traefik.http.routers.your-service.middlewares=stargate"  # 使用 Stargate 中间件

networks:
  traefik:
    external: true
```

### 生产环境建议

1. **使用 HTTPS**：在生产环境中，确保通过 Traefik 配置 HTTPS
2. **使用强密码算法**：避免使用 `plaintext`，推荐使用 `bcrypt` 或 `sha512`
3. **设置 Cookie 域名**：如果需要在多个子域名间共享会话，设置 `COOKIE_DOMAIN`
4. **日志管理**：配置适当的日志轮转和监控
5. **资源限制**：为容器设置适当的 CPU 和内存限制

**详细部署指南请参阅：[docs/zhCN/DEPLOYMENT.md](docs/zhCN/DEPLOYMENT.md)**

## 💻 开发指南

### 项目结构

```
codes/
├── src/
│   ├── cmd/
│   │   └── stargate/          # 主程序入口
│   │       ├── main.go        # 程序入口
│   │       ├── server.go      # 服务器配置
│   │       └── constants.go   # 常量定义
│   ├── internal/
│   │   ├── auth/              # 认证逻辑
│   │   ├── config/            # 配置管理
│   │   ├── handlers/          # HTTP 处理器
│   │   ├── i18n/              # 国际化
│   │   ├── middleware/        # 中间件
│   │   ├── secure/            # 密码加密算法
│   │   └── web/               # Web 模板和静态资源
│   ├── go.mod
│   └── go.sum
├── Dockerfile
├── docker-compose.yml
└── start-local.sh
```

### 本地开发

1. 安装依赖：
```bash
cd codes
go mod download
```

2. 运行测试：
```bash
go test ./...
```

3. 启动开发服务器：
```bash
./start-local.sh
```

### 添加新的密码算法

1. 在 `src/internal/secure/` 目录下创建新的算法实现：
```go
package secure

type NewAlgorithmResolver struct{}

func (r *NewAlgorithmResolver) Check(h string, password string) bool {
    // 实现密码验证逻辑
    return false
}
```

2. 在 `src/internal/config/validation.go` 中注册算法：
```go
SupportedAlgorithms = map[string]secure.HashResolver{
    // ...
    "newalgorithm": &secure.NewAlgorithmResolver{},
}
```

### 添加新的语言支持

1. 在 `src/internal/i18n/i18n.go` 中添加语言常量：
```go
const (
    LangEN Language = "en"
    LangZH Language = "zh"
    LangFR Language = "fr"  // 新增
)
```

2. 添加翻译映射：
```go
var translations = map[Language]map[string]string{
    // ...
    LangFR: {
        "error.auth_required": "Authentification requise",
        // ...
    },
}
```

3. 在 `src/internal/config/config.go` 中添加语言选项：
```go
Language = EnvVariable{
    PossibleValues: []string{"en", "zh", "fr"},  // 添加新语言
}
```

## 📝 许可证

本项目采用 Apache License 2.0 许可证。详情请参阅 [LICENSE](codes/LICENSE) 文件。

## 🤝 贡献

我们欢迎各种形式的贡献！无论是：
- 🐛 错误报告
- 💡 功能建议
- 📝 文档改进
- 🔧 代码贡献

请随时提交 Issue 或 Pull Request。每一个贡献都会让 Stargate 变得更好！

---

## ⚠️ 生产环境检查清单

在部署到生产环境之前，请确保您已完成以下安全最佳实践：

- ✅ **使用强密码**：避免使用 `plaintext`，使用 `bcrypt` 或 `sha512` 进行密码哈希
- ✅ **启用 HTTPS**：通过 Traefik 或您的反向代理配置 HTTPS
- ✅ **设置 Cookie 域名**：配置 `COOKIE_DOMAIN` 以在子域名间正确管理会话
- ✅ **监控与日志**：为您的部署设置适当的日志记录和监控
- ✅ **定期更新**：保持 Stargate 更新到最新版本，以获取安全补丁
