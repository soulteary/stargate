# Stargate - Forward Auth Service

[![License](https://img.shields.io/badge/license-Apache%202.0-blue.svg)](LICENSE)
[![Go Version](https://img.shields.io/badge/go-1.25+-blue.svg)](https://golang.org)
[![codecov](https://codecov.io/gh/soulteary/stargate/branch/main/graph/badge.svg)](https://codecov.io/gh/soulteary/stargate)
[![Go Report Card](https://goreportcard.com/badge/github.com/soulteary/stargate)](https://goreportcard.com/report/github.com/soulteary/stargate)

> **🚀 微服务安全的统一入口**

![Stargate](.github/assets/banner.jpg)

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
- **企业级认证**：与 Warden（用户白名单）和 Herald（OTP/验证码）集成，提供生产级认证能力

## ✨ 功能特性

### 🔐 企业级安全
- **多种密码加密算法**：支持 plaintext（测试用）、bcrypt、MD5、SHA512 等多种加密算法
- **安全会话管理**：基于 Cookie 的会话管理，支持自定义域名和过期时间
- **灵活的认证方式**：同时支持基于密码和基于会话的认证
- **OTP/验证码支持**：与 Herald 服务集成，支持短信/邮件验证码
- **用户白名单管理**：与 Warden 服务集成，提供用户访问控制

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

## 📋 目录

- [快速开始](#-快速开始)
- [文档导航](#-文档导航)
- [基本配置](#-基本配置)
- [可选服务集成](#-可选服务集成)
- [生产环境检查清单](#-生产环境检查清单)
- [许可证](#-许可证)

## 🚀 快速开始

**2 分钟内**让 Stargate 运行起来！

### 使用 Docker Compose（推荐）

**步骤 1：** 克隆项目
```bash
git clone <repository-url>
cd stargate
```

**步骤 2：** 配置认证信息（编辑 `docker-compose.yml`）

**选项 A：密码认证（简单）**
```yaml
services:
  stargate:
    environment:
      - AUTH_HOST=auth.example.com
      - PASSWORDS=plaintext:yourpassword1|yourpassword2
```

**选项 B：Warden + Herald OTP 认证（生产环境）**
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

**步骤 3：** 启动服务
```bash
docker-compose up -d
```

**完成！** 您的认证服务现在已经在运行了。🎉

### 本地开发

本地开发需要 Go 1.25 或更高版本，然后：

```bash
chmod +x start-local.sh
./start-local.sh
```

访问登录页面：`http://localhost:8080/_login?callback=localhost`

## 📚 文档导航

我们提供了全面的文档，帮助您充分利用 Stargate：

### 核心文档

- 📐 **[架构文档](docs/zhCN/ARCHITECTURE.md)** - 深入了解技术架构和设计决策
- 🔌 **[API 文档](docs/zhCN/API.md)** - 完整的 API 端点参考和示例
- ⚙️ **[配置参考](docs/zhCN/CONFIG.md)** - 详细的配置选项和最佳实践
- 🚀 **[部署指南](docs/zhCN/DEPLOYMENT.md)** - 生产环境部署策略和建议

### 快速参考

- **API 端点**：`GET /_auth`（认证检查）、`GET /_login`（登录页面）、`POST /_login`（登录）、`GET /_logout`（登出）、`GET /_session_exchange`（跨域）、`GET /health`（健康检查）
- **部署**：推荐使用 Docker Compose 快速开始。生产环境部署请参阅 [DEPLOYMENT.md](docs/zhCN/DEPLOYMENT.md)
- **开发**：开发相关文档请参阅 [ARCHITECTURE.md](docs/zhCN/ARCHITECTURE.md)

## ⚙️ 基本配置

Stargate 使用环境变量进行配置。以下是最常用的配置项：

### 必需配置

- **`AUTH_HOST`**：认证服务的主机名（例如：`auth.example.com`）
- **`PASSWORDS`**：密码配置，格式为 `算法:密码1|密码2|密码3`

### 常用配置示例

```bash
# 简单密码认证
AUTH_HOST=auth.example.com
PASSWORDS=plaintext:test123|admin456

# 使用 BCrypt 哈希
PASSWORDS=bcrypt:$2a$10$N9qo8uLOickgx2ZMRZoMyeIjZAgcfl7p92ldGxad68LJZdL17lhWy

# 跨域会话共享
COOKIE_DOMAIN=.example.com

# 自定义登录页面
LOGIN_PAGE_TITLE=我的认证服务
LANGUAGE=zh  # 或 'en'
```

**支持的密码算法：** `plaintext`（仅测试用）、`bcrypt`、`md5`、`sha512`

**完整配置参考请参阅：[docs/zhCN/CONFIG.md](docs/zhCN/CONFIG.md)**

## 🔗 可选服务集成

Stargate 可以完全独立使用，也可以选择性地与以下服务集成：

### Warden 集成（用户白名单）

提供用户白名单管理和用户信息。启用后，Stargate 会查询 Warden 以验证用户是否在允许列表中。

```bash
WARDEN_ENABLED=true
WARDEN_URL=http://warden:8080
WARDEN_API_KEY=your-api-key
```

### Herald 集成（OTP/验证码）

提供 OTP/验证码服务。启用后，Stargate 会调用 Herald 创建、发送和验证验证码（短信/邮件）。

```bash
HERALD_ENABLED=true
HERALD_URL=http://herald:8080
HERALD_HMAC_SECRET=your-hmac-secret  # 生产环境
# 或
HERALD_API_KEY=your-api-key  # 开发环境
```

**注意**：这两个集成都是可选的。Stargate 可以独立使用密码认证。

**完整集成指南请参阅：[docs/zhCN/ARCHITECTURE.md](docs/zhCN/ARCHITECTURE.md)**

## ⚠️ 生产环境检查清单

在部署到生产环境之前：

- ✅ 使用强密码算法（`bcrypt` 或 `sha512`，避免使用 `plaintext`）
- ✅ 通过 Traefik 或您的反向代理启用 HTTPS
- ✅ 设置 `COOKIE_DOMAIN` 以在子域名间正确管理会话
- ✅ 如需更高级功能，可选择集成 Warden + Herald 进行 OTP 认证
- ✅ Stargate ↔ Herald/Warden 通信使用 HMAC 签名或 mTLS
- ✅ 设置适当的日志记录和监控
- ✅ 保持 Stargate 更新到最新版本

## 🎯 设计原则

Stargate 设计为可以完全独立使用：

- **独立使用**：可以独立运行，使用密码认证模式，无需任何外部依赖
- **可选集成**：可以选择性地集成 Warden（用户白名单）和 Herald（OTP/验证码）服务
- **高性能**：forwardAuth 主链路只校验 session，确保快速响应
- **灵活性**：支持多种认证模式，可根据需求选择

## 📝 许可证

本项目采用 Apache License 2.0 许可证。详情请参阅 [LICENSE](LICENSE) 文件。

## 🤝 贡献

我们欢迎各种形式的贡献！无论是错误报告、功能建议、文档改进还是代码贡献，请随时提交 Issue 或 Pull Request。
