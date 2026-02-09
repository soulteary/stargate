# 安全文档

> 🌐 **Language / 语言**: [English](../enUS/SECURITY.md) | [中文](SECURITY.md) | [Français](../frFR/SECURITY.md) | [Italiano](../itIT/SECURITY.md) | [日本語](../jaJP/SECURITY.md) | [Deutsch](../deDE/SECURITY.md) | [한국어](../koKR/SECURITY.md)

本文档说明 Stargate 的安全特性、安全配置和最佳实践。

## 已实现的安全功能

1. **Forward Auth 保护**: 集中式认证层，用于保护后端服务
2. **多种密码算法**: 支持 bcrypt、SHA512、MD5 和 plaintext（仅开发环境）
3. **安全会话管理**: 基于 Cookie 的会话，可配置域名和过期时间
4. **服务集成安全**: 使用 mTLS 或 HMAC 与 Warden 和 Herald 服务进行安全通信
5. **会话共享安全**: 安全的跨域会话交换机制
6. **输入验证**: 严格验证所有输入参数
7. **错误处理**: 生产模式隐藏详细错误信息
8. **安全响应头**: 自动添加安全相关的 HTTP 响应头
9. **HTTPS 强制**: 生产环境应使用 HTTPS
10. **OTP 集成**: 与 Herald 的安全集成，用于 OTP/验证码认证

## 安全最佳实践

### 1. 生产环境配置

**必须配置项**:
- 必须使用安全算法（bcrypt 或 SHA512）设置强密码
- 设置 `MODE=production` 启用生产模式
- 配置 `COOKIE_DOMAIN` 以正确管理会话
- 通过反向代理（Traefik、Nginx 等）使用 HTTPS
- 配置安全的会话 Cookie 设置

**配置示例**:
```bash
export AUTH_HOST=auth.example.com
export PASSWORDS=bcrypt:$2a$10$N9qo8uLOickgx2ZMRZoMyeIjZAgcfl7p92ldGxad68LJZdL17lhWy
export COOKIE_DOMAIN=.example.com
```

**说明**：当前实现中 Cookie 的 Secure 由 `X-Forwarded-Proto` 等请求头推断，SameSite 固定为 `Lax`；未提供 `COOKIE_SECURE`、`COOKIE_SAME_SITE`、`SESSION_TTL` 等环境变量，详见 [CONFIG.md](CONFIG.md)。

### 2. 密码安全

**推荐做法**:
- ✅ 使用强密码哈希算法（bcrypt 或 SHA512）
- ✅ 在环境变量中存储密码哈希
- ✅ 为不同环境使用不同密码
- ✅ 定期轮换密码

**不推荐做法**:
- ❌ 在生产环境中使用明文密码
- ❌ 在配置文件中硬编码密码
- ❌ 在生产环境中使用弱密码算法（MD5）
- ❌ 跨环境共享密码

**密码算法比较**:
- `bcrypt`: 生产环境推荐（慢速、安全）
- `sha512`: 生产环境良好（快速、安全）
- `md5`: 不推荐用于生产环境（快速、安全性较低）
- `plaintext`: 仅开发环境（无安全性）

### 3. 会话安全

**会话配置**（以当前实现为准，完整配置项见 [CONFIG.md](CONFIG.md)）:
- **Cookie 域名**: 设置 `COOKIE_DOMAIN` 以在子域之间共享会话
- **Secure 标志**: 由反向代理/请求协议（如 `X-Forwarded-Proto: https`）推断，暂无 `COOKIE_SECURE` 环境变量
- **SameSite**: 当前固定为 `Lax`，暂无 `COOKIE_SAME_SITE` 环境变量
- **HttpOnly**: Cookie 自动设置为 HttpOnly 以防止 XSS 攻击
- **过期时间**: 当前为代码内固定 24 小时，暂无 `SESSION_TTL` 环境变量

**配置示例**:
```bash
export COOKIE_DOMAIN=.example.com
```

### 4. 网络安全

**必须配置**:
- 生产环境必须使用 HTTPS
- 配置反向代理（Traefik、Nginx）处理 SSL/TLS
- 使用防火墙规则限制访问
- 定期更新依赖项以修复已知漏洞

**推荐配置**:
- 使用带 Let's Encrypt 的 Traefik 自动获取 SSL 证书
- 如果在反向代理后面，配置 `TRUSTED_PROXY_IPS`
- 使用网络策略限制服务访问
- 监控和记录认证尝试

### 5. 服务集成安全

与 Warden 和 Herald 服务集成时：

**推荐：mTLS**
- 使用双向 TLS 证书获得最高安全性
- 为 Stargate 配置客户端证书
- 验证 Warden 和 Herald 的服务端证书

**替代方案：HMAC 签名**
- 使用 HMAC-SHA256 签名进行安全通信
- 安全配置共享密钥
- 使用时间戳验证以防止重放攻击

**配置示例（HMAC）**:
```bash
export WARDEN_ENABLED=true
export WARDEN_URL=https://warden:8080
export WARDEN_HMAC_SECRET=your-secret-key

export HERALD_ENABLED=true
export HERALD_URL=https://herald:8082
export HERALD_HMAC_SECRET=your-secret-key
```

## API 安全

### 认证方式

Stargate 支持两种认证方式：

1. **请求头认证**（API 请求）
   - 请求头: `Stargate-Password: <password>`
   - 适用于 API 请求、自动化脚本
   - 密码根据配置的密码哈希进行验证

2. **Cookie 认证**（Web 请求）
   - Cookie: `stargate_session_id=<session_id>`
   - 适用于通过浏览器访问的 Web 应用程序
   - 会话根据存储的会话数据进行验证

### Forward Auth 端点

主要认证端点 `GET /_auth`：

- **成功（200 OK）**: 设置 `X-Forwarded-User` 请求头并返回 200
- **失败（401 Unauthorized）**: 重定向到登录页面（HTML）或返回 401（API）

### 限流

考虑在反向代理层面实现限流：
- 限制每个 IP 的登录尝试次数
- 限制每个 IP 的 forward auth 请求次数
- 使用 Traefik 中间件或 Nginx 限流

## 数据安全

### 会话存储

Stargate 支持多种会话存储后端：

1. **Redis**（生产环境推荐）
   - 分布式会话存储
   - 支持跨实例会话共享
   - 配置密码保护

2. **内存**（仅开发环境）
   - 简单，无外部依赖
   - 不适合生产环境（重启后丢失）

**Redis 配置**:
```bash
export REDIS_ENABLED=true
export REDIS_ADDR=redis:6379
export REDIS_PASSWORD=your-redis-password
```

### 敏感信息管理

**推荐做法**:
- ✅ 使用环境变量存储密码和密钥
- ✅ 使用密码文件存储敏感配置
- ✅ 永远不要记录密码或会话令牌
- ✅ 在生产环境中使用安全密钥管理服务

**不推荐做法**:
- ❌ 在配置文件中硬编码密码
- ❌ 通过命令行参数传递密码
- ❌ 将敏感信息提交到版本控制
- ❌ 记录敏感用户数据

## 错误处理

### 生产模式

在生产模式下（`MODE=production` 或 `MODE=prod`）：

- 隐藏详细错误信息，防止信息泄露
- 返回通用错误消息
- 详细错误信息仅记录在日志中
- 认证失败时重定向到登录页面

### 开发模式

在开发模式下：

- 显示详细错误信息以便调试
- 包含堆栈跟踪信息
- 更详细的日志记录

## 安全响应头

Stargate 自动添加以下安全相关的 HTTP 响应头：

- `X-Content-Type-Options: nosniff` - 防止 MIME 类型嗅探
- `X-Frame-Options: DENY` - 防止点击劫持
- `X-XSS-Protection: 1; mode=block` - XSS 保护

## 跨域会话共享

Stargate 支持安全的跨域会话共享：

- **会话交换端点**: `GET /_session_exchange`
- **安全令牌**: 使用安全令牌进行会话交换
- **域名验证**: 在共享会话之前验证目标域名
- **过期时间**: 交换令牌在短 TTL 后过期

**安全注意事项**:
- 仅在受信任的域名之间共享会话
- 使用 HTTPS 进行会话交换
- 监控会话交换尝试

## OTP/验证码安全

使用 Herald 集成进行 OTP 认证时：

- **基于 Challenge**: 使用 challenge-verify 模型
- **安全通信**: 使用 mTLS 或 HMAC 与 Herald 通信
- **限流**: Herald 处理限流
- **审计日志**: Herald 维护审计日志

## 漏洞报告

如果发现安全漏洞，请通过以下方式报告：

1. **GitHub Security Advisory**（推荐）
   - 访问仓库的 [Security 标签页](https://github.com/soulteary/stargate/security)
   - 点击 "Report a vulnerability"
   - 填写安全咨询表单

2. **邮件**（如果 GitHub Security Advisory 不可用）
   - 发送邮件给项目维护者
   - 包含漏洞的详细描述

**请不要通过公开的 GitHub Issues 报告安全漏洞。**

## 相关文档

- [API 文档](API.md) - 了解 API 安全特性
- [架构文档](ARCHITECTURE.md) - 了解安全架构
- [配置参考](CONFIG.md) - 了解安全相关的配置选项
- [部署指南](DEPLOYMENT.md) - 了解生产环境部署安全建议
