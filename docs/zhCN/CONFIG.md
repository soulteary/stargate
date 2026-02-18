# 配置参考

本文档详细说明 Stargate 的所有配置选项。

## 目录

- [配置方式](#配置方式)
- [配置项速查表](#配置项速查表)
- [必需配置](#必需配置)
- [可选配置](#可选配置)
- [密码配置](#密码配置)
- [配置示例](#配置示例)

## 配置方式

Stargate 通过环境变量进行配置。所有配置项都通过环境变量设置，无需配置文件。

### 设置环境变量

**Linux/macOS：**

```bash
export AUTH_HOST=auth.example.com
export PASSWORDS=plaintext:yourpassword
```

**Docker：**

```bash
docker run -e AUTH_HOST=auth.example.com -e PASSWORDS=plaintext:yourpassword stargate:latest
```

**Docker Compose：**

```yaml
services:
  stargate:
    environment:
      - AUTH_HOST=auth.example.com
      - PASSWORDS=plaintext:yourpassword
```

## 配置项速查表

以下为代码中实际存在的环境变量一览（必填以“是”标注；未标注为否且无默认的表示按需设置）。

| 环境变量 | 类型/可选值 | 默认值 | 必需 |
|----------|-------------|--------|------|
| `AUTH_HOST` | String | — | 是 |
| `PASSWORDS` | 见密码配置 | — | 未启用 Warden 时为是 |
| `DEBUG` | true/false | false | 否 |
| `LOGIN_PAGE_TITLE` | String | Stargate - Login | 否 |
| `LOGIN_PAGE_FOOTER_TEXT` | String | Copyright © 2024 - Stargate | 否 |
| `USER_HEADER_NAME` | String | X-Forwarded-User | 否 |
| `COOKIE_DOMAIN` | String | 空 | 否 |
| `LANGUAGE` | en, zh, fr, it, ja, de, ko | en | 否 |
| `PORT` | String | 空（:80） | 否 |
| `WARDEN_ENABLED` | true/false | false | 否 |
| `WARDEN_URL` | String | 空 | 否 |
| `WARDEN_API_KEY` | String | 空 | 否 |
| `WARDEN_CACHE_TTL` | String | 300 | 否 |
| `WARDEN_OTP_ENABLED` | true/false | false | 否 |
| `WARDEN_OTP_SECRET_KEY` | String | 空 | 否 |
| `HERALD_ENABLED` | true/false | false | 否 |
| `HERALD_URL` | String | 空 | 否 |
| `HERALD_API_KEY` | String | 空 | 否 |
| `HERALD_HMAC_SECRET` | String | 空 | 否 |
| `HERALD_TLS_CA_CERT_FILE` | 路径 | 空 | 否 |
| `HERALD_TLS_CLIENT_CERT_FILE` | 路径 | 空 | 否 |
| `HERALD_TLS_CLIENT_KEY_FILE` | 路径 | 空 | 否 |
| `HERALD_TLS_SERVER_NAME` | String | 空 | 否 |
| `HERALD_TOTP_ENABLED` | true/false | false | 否 |
| `HERALD_TOTP_BASE_URL` | String | 空 | 否 |
| `HERALD_TOTP_API_KEY` | String | 空 | 否 |
| `HERALD_TOTP_HMAC_SECRET` | String | 空 | 否 |
| `LOGIN_SMS_ENABLED` | true/false | true | 否 |
| `LOGIN_EMAIL_ENABLED` | true/false | true | 否 |
| `SESSION_STORAGE_ENABLED` | true/false | false | 否 |
| `SESSION_STORAGE_REDIS_ADDR` | String | localhost:6379 | 否 |
| `SESSION_STORAGE_REDIS_PASSWORD` | String | 空 | 否 |
| `SESSION_STORAGE_REDIS_DB` | String | 0 | 否 |
| `SESSION_STORAGE_REDIS_KEY_PREFIX` | String | stargate:session: | 否 |
| `AUDIT_LOG_ENABLED` | true/false | true | 否 |
| `AUDIT_LOG_FORMAT` | json/text | json | 否 |
| `STEP_UP_ENABLED` | true/false | false | 否 |
| `STEP_UP_PATHS` | 逗号分隔路径 | 空 | 否 |
| `OTLP_ENABLED` | true/false | false | 否 |
| `OTLP_ENDPOINT` | String | 空 | 否 |
| `AUTH_REFRESH_ENABLED` | true/false | false | 否 |
| `AUTH_REFRESH_INTERVAL` | duration | 5m | 否 |

## 必需配置

以下配置项是必需的，未设置会导致服务启动失败。

### `AUTH_HOST`

认证服务的主机名。

| 属性 | 值 |
|------|-----|
| **类型** | String |
| **必需** | 是 |
| **默认值** | 无 |
| **示例** | `auth.example.com` |

**说明：**

- 用于构建登录回调 URL
- 通常设置为 Stargate 服务的主机名
- 支持通配符 `*`（不推荐在生产环境使用）

**示例：**

```bash
AUTH_HOST=auth.example.com
```

### `PASSWORDS`

密码配置，指定密码加密算法和密码列表。

| 属性 | 值 |
|------|-----|
| **类型** | String |
| **必需** | 是 |
| **默认值** | 无 |
| **格式** | `算法:密码1|密码2|密码3` |

**说明：**

- 格式：`算法:密码1|密码2|密码3`
- 支持多个密码，用 `|` 分隔
- 任一密码验证通过即可登录
- 支持的算法见 [密码配置](#密码配置) 章节

**示例：**

```bash
# 单个明文密码
PASSWORDS=plaintext:test123

# 多个明文密码
PASSWORDS=plaintext:test123|admin456|user789

# BCrypt 哈希
PASSWORDS=bcrypt:$2a$10$N9qo8uLOickgx2ZMRZoMyeIjZAgcfl7p92ldGxad68LJZdL17lhWy

# SHA512 哈希
PASSWORDS=sha512:5e884898da28047151d0e56f8dc6292773603d0d6aabbdd62a11ef721d1542d8
```

## 可选配置

以下配置项是可选的，未设置时使用默认值。

### `DEBUG`

启用调试模式。

| 属性 | 值 |
|------|-----|
| **类型** | Boolean |
| **必需** | 否 |
| **默认值** | `false` |
| **可选值** | `true`, `false` |

**说明：**

- 启用后，日志级别设置为 `DEBUG`
- 会输出更详细的调试信息
- 生产环境建议设置为 `false`

**示例：**

```bash
DEBUG=true
```

### `LANGUAGE`

界面语言。

| 属性 | 值 |
|------|-----|
| **类型** | String |
| **必需** | 否 |
| **默认值** | `en` |
| **可选值** | `en`（英文）, `zh`（中文）, `fr`（法语）, `it`（意大利语）, `ja`（日语）, `de`（德语）, `ko`（韩语） |

**说明：**

- 影响错误消息和界面文本的语言
- 支持大小写不敏感（`EN`、`en`、`En` 都可以）

**示例：**

```bash
LANGUAGE=zh
```

### `LOGIN_PAGE_TITLE`

登录页面标题。

| 属性 | 值 |
|------|-----|
| **类型** | String |
| **必需** | 否 |
| **默认值** | `Stargate - Login` |

**说明：**

- 显示在登录页面的标题位置
- 支持 HTML 标签（不推荐）

**示例：**

```bash
LOGIN_PAGE_TITLE=我的认证服务
```

### `LOGIN_PAGE_FOOTER_TEXT`

登录页面页脚文本。

| 属性 | 值 |
|------|-----|
| **类型** | String |
| **必需** | 否 |
| **默认值** | `Copyright © 2024 - Stargate` |

**说明：**

- 显示在登录页面的页脚位置
- 支持 HTML 标签（不推荐）

**示例：**

```bash
LOGIN_PAGE_FOOTER_TEXT=© 2024 我的公司
```

### `USER_HEADER_NAME`

认证成功后设置的用户头名称。

| 属性 | 值 |
|------|-----|
| **类型** | String |
| **必需** | 否 |
| **默认值** | `X-Forwarded-User` |

**说明：**

- 认证成功后，Stargate 会在响应头中设置此头
- 头值为 `authenticated`
- 后端服务可以通过此头判断用户是否已认证
- 必须是非空字符串

**示例：**

```bash
USER_HEADER_NAME=X-Authenticated-User
```

### `COOKIE_DOMAIN`

Cookie 域名，用于跨域会话共享。

| 属性 | 值 |
|------|-----|
| **类型** | String |
| **必需** | 否 |
| **默认值** | 空（不设置） |

**说明：**

- 如果设置，会话 Cookie 会设置到指定域名
- 支持跨子域名共享会话
- 格式：`.example.com`（注意前面的点）
- 设置为空时，Cookie 只在当前域名有效

**示例：**

```bash
# 允许在所有 *.example.com 子域名共享会话
COOKIE_DOMAIN=.example.com
```

**跨域会话共享场景：**

假设有以下域名：
- `auth.example.com` - 认证服务
- `app1.example.com` - 应用 1
- `app2.example.com` - 应用 2

设置 `COOKIE_DOMAIN=.example.com` 后：
1. 用户在 `auth.example.com` 登录
2. 会话 Cookie 被设置到 `.example.com` 域名
3. 用户可以在 `app1.example.com` 和 `app2.example.com` 下使用同一会话

**Cookie 与会话行为说明（当前实现）**：
- **会话过期时间**：由代码内常量固定为 24 小时，暂无 `SESSION_TTL` 等环境变量可配置。
- **Cookie Secure**：根据请求协议（如 `X-Forwarded-Proto: https`）自动设置，暂无 `COOKIE_SECURE` 环境变量。
- **Cookie SameSite**：固定为 `Lax`，暂无 `COOKIE_SAME_SITE` 环境变量。

### `PORT`

服务监听端口（仅用于本地开发）。由 config 包统一管理，与其它配置项一起通过环境变量加载与校验。

| 属性 | 值 |
|------|-----|
| **类型** | String |
| **必需** | 否 |
| **默认值** | 空（空时服务使用默认端口 `:80`） |

**说明：**

- 仅用于本地开发环境
- Docker 容器中通常不需要设置（使用默认 80 端口）
- 格式：端口号（如 `8080`）或 `:端口号`（如 `:8080`）

**示例：**

```bash
PORT=8080
```

### Warden 集成（可选）

Warden 是用户白名单/授权用户信息服务。**这是可选的**，如果不需要用户白名单功能，可以不启用。以下配置选项用于与 Warden 集成。

#### `WARDEN_ENABLED`

启用 Warden 集成以使用用户白名单认证功能。

| 属性 | 值 |
|------|-----|
| **类型** | Boolean |
| **必需** | 否 |
| **默认值** | `false` |
| **可选值** | `true`, `false` |

**说明：**

- 启用后，Stargate 将使用 Warden 服务进行用户白名单验证
- 需要设置 `WARDEN_URL`
- 这是可选功能，Stargate 可以独立使用密码认证

**示例：**

```bash
WARDEN_ENABLED=true
```

#### `WARDEN_URL`

Warden 服务的基础 URL。

| 属性 | 值 |
|------|-----|
| **类型** | String |
| **必需** | 否（如果 `WARDEN_ENABLED=true` 则为必需） |
| **默认值** | 空 |
| **示例** | `http://warden:8080` 或 `https://warden.example.com` |

**说明：**

- Warden 服务的完整基础 URL（不带尾部斜杠）
- 如果 `WARDEN_ENABLED=true` 则必须设置
- 用于查询用户信息和白名单验证

**示例：**

```bash
WARDEN_URL=http://warden:8080
```

#### `WARDEN_API_KEY`

用于与 Warden 服务进行 API Key 认证。

| 属性 | 值 |
|------|-----|
| **类型** | String |
| **必需** | 否 |
| **默认值** | 空 |

**说明：**

- 使用 API Key 的简单认证方式
- 适用于开发和生产环境
- 如果未设置，Warden 客户端可能无法正确认证

**示例：**

```bash
WARDEN_API_KEY=your-warden-api-key-here
```

#### `WARDEN_CACHE_TTL`

Warden 用户信息缓存的 TTL（生存时间）。

| 属性 | 值 |
|------|-----|
| **类型** | Integer |
| **必需** | 否 |
| **默认值** | `300`（5 分钟） |
| **单位** | 秒 |

**说明：**

- 用户信息在本地缓存的时间
- 减少对 Warden 服务的请求频率
- 提高认证性能

**示例：**

```bash
WARDEN_CACHE_TTL=300
```

#### `WARDEN_OTP_ENABLED`

启用 Warden 内置 OTP 校验（与 Herald OTP 不同，为旧版/内置能力）。

| 属性 | 值 |
|------|-----|
| **类型** | Boolean |
| **必需** | 否 |
| **默认值** | `false` |
| **可选值** | `true`, `false` |

**说明：**

- 为 `true` 时使用 Warden 侧 OTP 校验（需配合 `WARDEN_OTP_SECRET_KEY`）
- 与 Herald 验证码服务（`HERALD_ENABLED`）为两套独立能力，按需选用其一或组合

**示例：**

```bash
WARDEN_OTP_ENABLED=true
```

#### `WARDEN_OTP_SECRET_KEY`

Warden OTP 校验所用密钥（仅在 `WARDEN_OTP_ENABLED=true` 时使用）。

| 属性 | 值 |
|------|-----|
| **类型** | String |
| **必需** | 否 |
| **默认值** | 空 |

**示例：**

```bash
WARDEN_OTP_SECRET_KEY=your-warden-otp-secret
```

### Herald 集成（可选）

Herald 是 OTP/验证码服务。**这是可选的**，如果不需要验证码功能，可以不启用。以下配置选项用于与 Herald 集成。

#### `HERALD_ENABLED`

启用 Herald 集成以使用 OTP/验证码功能。

| 属性 | 值 |
|------|-----|
| **类型** | Boolean |
| **必需** | 否 |
| **默认值** | `false` |
| **可选值** | `true`, `false` |

**说明：**

- 启用后，Stargate 将使用 Herald 服务发送和验证 OTP 验证码
- 需要设置 `HERALD_URL`
- 这是可选功能，Stargate 可以独立使用密码认证

**示例：**

```bash
HERALD_ENABLED=true
```

#### `LOGIN_SMS_ENABLED`

在启用 Herald 时是否允许通过短信验证码登录。

| 属性 | 值 |
|------|-----|
| **类型** | Boolean |
| **必需** | 否 |
| **默认值** | `true` |
| **可选值** | `true`, `false` |

**说明：**

- 为 `false` 时，不提供短信通道，请求短信验证码会被拒绝（或在邮箱启用时回退到邮箱）
- 仅当 `HERALD_ENABLED=true` 且使用 Warden 登录时生效

**示例：**

```bash
LOGIN_SMS_ENABLED=false
```

#### `LOGIN_EMAIL_ENABLED`

在启用 Herald 时是否允许通过邮箱验证码登录。

| 属性 | 值 |
|------|-----|
| **类型** | Boolean |
| **必需** | 否 |
| **默认值** | `true` |
| **可选值** | `true`, `false` |

**说明：**

- 为 `false` 时，不提供邮箱通道，请求邮箱验证码会被拒绝（或在短信启用时回退到短信）
- 仅当 `HERALD_ENABLED=true` 且使用 Warden 登录时生效

**示例：**

```bash
LOGIN_EMAIL_ENABLED=false
```

#### `HERALD_URL`

Herald 服务的基础 URL。

| 属性 | 值 |
|------|-----|
| **类型** | String |
| **必需** | 否（如果 `HERALD_ENABLED=true` 则为必需） |
| **默认值** | 空 |
| **示例** | `http://herald:8080` 或 `https://herald.example.com` |

**说明：**

- Herald 服务的完整基础 URL（不带尾部斜杠）
- 如果 `HERALD_ENABLED=true` 则必须设置
- 用于创建 challenge 和验证验证码

**示例：**

```bash
HERALD_URL=http://herald:8080
```

#### `HERALD_API_KEY`

用于与 Herald 服务进行 API Key 认证（简单认证方式）。

| 属性 | 值 |
|------|-----|
| **类型** | String |
| **必需** | 否 |
| **默认值** | 空 |

**说明：**

- 使用 API Key 的简单认证方式
- 适用于开发环境
- 如果同时设置了 `HERALD_API_KEY` 和 `HERALD_HMAC_SECRET`，HMAC 优先
- 如果未设置 `HERALD_HMAC_SECRET` 则必须设置

**示例：**

```bash
HERALD_API_KEY=your-api-key-here
```

#### `HERALD_HMAC_SECRET`

用于与 Herald 服务进行 HMAC 签名认证的密钥（生产环境推荐）。

| 属性 | 值 |
|------|-----|
| **类型** | String |
| **必需** | 否 |
| **默认值** | 空 |

**说明：**

- 基于 HMAC-SHA256 签名的认证方式（比 API Key 更安全）
- 推荐用于生产环境
- 如果同时设置了 `HERALD_API_KEY` 和 `HERALD_HMAC_SECRET`，HMAC 优先
- 为服务间通信提供更好的安全性
- 必须与 Herald 服务中配置的 HMAC 密钥匹配

**认证方式：**

Stargate 支持两种与 Herald 的认证方式：

1. **API Key**（简单，适用于开发）：
   - 仅设置 `HERALD_API_KEY`
   - 安全性较低但配置简单
   - 适用于开发和测试环境

2. **HMAC 签名**（安全，适用于生产）：
   - 设置 `HERALD_HMAC_SECRET`
   - 更安全，使用 HMAC-SHA256 签名
   - 推荐用于生产环境
   - 提供基于时间戳的请求签名

**示例：**

```bash
# 生产环境：使用 HMAC
HERALD_HMAC_SECRET=your-hmac-secret-key-here

# 开发环境：使用 API Key
HERALD_API_KEY=your-api-key-here
```

**注意：** 如果既未设置 `HERALD_API_KEY` 也未设置 `HERALD_HMAC_SECRET`，Herald 客户端可能无法正确认证，请求可能失败。

#### Herald mTLS（可选）

与 Herald 通信时使用 TLS 客户端证书认证时可配置以下项。与 API Key / HMAC 可同时存在，由客户端实现决定优先使用哪种认证。

#### `HERALD_TLS_CA_CERT_FILE`

Herald 服务 CA 证书文件路径，用于验证服务端证书。

| 属性 | 值 |
|------|-----|
| **类型** | String（文件路径） |
| **必需** | 否 |
| **默认值** | 空 |

#### `HERALD_TLS_CLIENT_CERT_FILE`

客户端证书文件路径（mTLS）。

| 属性 | 值 |
|------|-----|
| **类型** | String（文件路径） |
| **必需** | 否 |
| **默认值** | 空 |

#### `HERALD_TLS_CLIENT_KEY_FILE`

客户端私钥文件路径（mTLS）。

| 属性 | 值 |
|------|-----|
| **类型** | String（文件路径） |
| **必需** | 否 |
| **默认值** | 空 |

#### `HERALD_TLS_SERVER_NAME`

TLS 握手时使用的 Server Name Indication（SNI），用于证书校验。

| 属性 | 值 |
|------|-----|
| **类型** | String |
| **必需** | 否 |
| **默认值** | 空 |

#### Herald TOTP（可选，每用户 2FA）

与 herald-totp 服务集成，用于 TOTP 动态码绑定与校验（Authenticator 通道）。与 Herald OTP（短信/邮件验证码）为不同通道。

#### `HERALD_TOTP_ENABLED`

启用 Herald TOTP 集成（绑定/校验 TOTP 动态码）。

| 属性 | 值 |
|------|-----|
| **类型** | Boolean |
| **必需** | 否 |
| **默认值** | `false` |
| **可选值** | `true`, `false` |

#### `HERALD_TOTP_BASE_URL`

herald-totp 服务的基础 URL。

| 属性 | 值 |
|------|-----|
| **类型** | String |
| **必需** | 否（若 `HERALD_TOTP_ENABLED=true` 则应设置） |
| **默认值** | 空 |

**示例：** `http://herald-totp:8080`

#### `HERALD_TOTP_API_KEY`

与 herald-totp 通信的 API Key（简单认证）。

| 属性 | 值 |
|------|-----|
| **类型** | String |
| **必需** | 否 |
| **默认值** | 空 |

#### `HERALD_TOTP_HMAC_SECRET`

与 herald-totp 通信的 HMAC 密钥（生产环境推荐）。

| 属性 | 值 |
|------|-----|
| **类型** | String |
| **必需** | 否 |
| **默认值** | 空 |

#### `HERALD_TOTP_*` 与 stargate-suite `keys-step` 对照

在 `stargate-suite` 的 Web UI 中，`keys-step` 会集中展示跨服务密钥项。与 Stargate 的 TOTP 集成相关项建议按下述对应关系配置：

- `HERALD_TOTP_ENABLED=true`：启用 Stargate -> herald-totp 通道
- `HERALD_TOTP_BASE_URL`：填写 herald-totp 服务地址（如 `http://herald-totp:8084`）
- `HERALD_TOTP_API_KEY`：对应 herald-totp 服务的 API Key（开发可用）
- `HERALD_TOTP_HMAC_SECRET`：对应 herald-totp 服务 HMAC 密钥（生产推荐）

建议：

- 开发环境可只配 `HERALD_TOTP_API_KEY`。
- 生产环境优先 `HERALD_TOTP_HMAC_SECRET`，并结合 mTLS/网络策略限制调用源。

### 会话存储（Redis，可选）

启用后会话将存储在 Redis，便于多实例共享与持久化；未启用时使用内存或 Cookie 存储。

#### `SESSION_STORAGE_ENABLED`

启用 Redis 会话存储。

| 属性 | 值 |
|------|-----|
| **类型** | Boolean |
| **必需** | 否 |
| **默认值** | `false` |
| **可选值** | `true`, `false` |

#### `SESSION_STORAGE_REDIS_ADDR`

Redis 地址。

| 属性 | 值 |
|------|-----|
| **类型** | String |
| **必需** | 否 |
| **默认值** | `localhost:6379` |

#### `SESSION_STORAGE_REDIS_PASSWORD`

Redis 密码（无密码时留空）。

| 属性 | 值 |
|------|-----|
| **类型** | String |
| **必需** | 否 |
| **默认值** | 空 |

#### `SESSION_STORAGE_REDIS_DB`

Redis 数据库编号。

| 属性 | 值 |
|------|-----|
| **类型** | Integer（0–15 等） |
| **必需** | 否 |
| **默认值** | `0` |

#### `SESSION_STORAGE_REDIS_KEY_PREFIX`

会话键前缀，用于区分多套 Stargate 或其它应用。

| 属性 | 值 |
|------|-----|
| **类型** | String |
| **必需** | 否 |
| **默认值** | `stargate:session:` |

#### 会话 Redis 配置建议（多场景）

- **单实例开发**：可不启用 Redis（`SESSION_STORAGE_ENABLED=false`）。
- **多实例/滚动升级**：启用 Redis 会话（`SESSION_STORAGE_ENABLED=true`），确保不同实例共享会话。
- **推荐最小配置**：
  - `SESSION_STORAGE_ENABLED=true`
  - `SESSION_STORAGE_REDIS_ADDR=<redis-host>:6379`
  - `SESSION_STORAGE_REDIS_DB=0`
  - `SESSION_STORAGE_REDIS_KEY_PREFIX=stargate:session:`
- **生产建议**：
  - Redis 开启访问控制（密码/ACL/网络隔离）
  - 为 Stargate 使用独立 DB 或前缀，避免与业务缓存混用
  - 结合监控关注连接数、命中率、延迟和内存水位

### 审计日志（可选）

#### `AUDIT_LOG_ENABLED`

是否启用审计日志。

| 属性 | 值 |
|------|-----|
| **类型** | Boolean |
| **必需** | 否 |
| **默认值** | `true` |
| **可选值** | `true`, `false` |

#### `AUDIT_LOG_FORMAT`

审计日志输出格式。

| 属性 | 值 |
|------|-----|
| **类型** | String |
| **必需** | 否 |
| **默认值** | `json` |
| **可选值** | `json`, `text` |

### Step-up 认证（可选）

对部分路径要求二次认证（如再次输入密码或 OTP）时启用。

#### `STEP_UP_ENABLED`

是否启用 Step-up 认证。

| 属性 | 值 |
|------|-----|
| **类型** | Boolean |
| **必需** | 否 |
| **默认值** | `false` |
| **可选值** | `true`, `false` |

#### `STEP_UP_PATHS`

需要 Step-up 的路径，多个路径用英文逗号分隔；支持路径前缀匹配。

| 属性 | 值 |
|------|-----|
| **类型** | String |
| **必需** | 否 |
| **默认值** | 空 |

**示例：** `/admin,/api/sensitive`

### OpenTelemetry（可选）

#### `OTLP_ENABLED`

是否开启 OTLP 遥测导出。

| 属性 | 值 |
|------|-----|
| **类型** | Boolean |
| **必需** | 否 |
| **默认值** | `false` |
| **可选值** | `true`, `false` |

#### `OTLP_ENDPOINT`

OTLP 采集端地址（如 Jaeger/OTLP Collector）。

| 属性 | 值 |
|------|-----|
| **类型** | String |
| **必需** | 否 |
| **默认值** | 空 |

**示例：** `http://jaeger:4318/v1/traces`

### 授权信息刷新（可选）

#### `AUTH_REFRESH_ENABLED`

是否在会话有效期内周期性从 Warden 刷新用户/授权信息并更新会话。

| 属性 | 值 |
|------|-----|
| **类型** | Boolean |
| **必需** | 否 |
| **默认值** | `false` |
| **可选值** | `true`, `false` |

#### `AUTH_REFRESH_INTERVAL`

刷新间隔，Go duration 格式（如 `5m`、`1h`）。

| 属性 | 值 |
|------|-----|
| **类型** | String（duration） |
| **必需** | 否 |
| **默认值** | `5m` |

**示例：** `AUTH_REFRESH_INTERVAL=10m`

## 密码配置

Stargate 支持多种密码加密算法。密码配置格式为：`算法:密码1|密码2|密码3`

### 支持的算法

#### `plaintext` - 明文密码

**说明：**

- 明文存储，不进行加密
- **仅用于测试环境**
- 生产环境强烈不推荐使用

**示例：**

```bash
PASSWORDS=plaintext:test123|admin456
```

#### `bcrypt` - BCrypt 哈希

**说明：**

- 使用 BCrypt 算法进行哈希
- 安全性高，推荐用于生产环境
- 密码必须使用 BCrypt 哈希值

**生成 BCrypt 哈希：**

```bash
# 使用 Go 生成
go run -c 'golang.org/x/crypto/bcrypt' <<< 'password'

# 使用在线工具或其他工具生成
```

**示例：**

```bash
PASSWORDS=bcrypt:$2a$10$N9qo8uLOickgx2ZMRZoMyeIjZAgcfl7p92ldGxad68LJZdL17lhWy
```

#### `md5` - MD5 哈希

**说明：**

- 使用 MD5 算法进行哈希
- 安全性较低，不推荐用于生产环境
- 密码必须使用 MD5 哈希值（32 位十六进制字符串）

**生成 MD5 哈希：**

```bash
# Linux/macOS
echo -n "password" | md5sum

# 或使用在线工具
```

**示例：**

```bash
PASSWORDS=md5:5e884898da28047151d0e56f8dc6292773603d0d6aabbdd62a11ef721d1542d8
```

#### `sha512` - SHA512 哈希

**说明：**

- 使用 SHA512 算法进行哈希
- 安全性较高，推荐用于生产环境
- 密码必须使用 SHA512 哈希值（128 位十六进制字符串）

**生成 SHA512 哈希：**

```bash
# Linux/macOS
echo -n "password" | shasum -a 512

# 或使用在线工具
```

**示例：**

```bash
PASSWORDS=sha512:5e884898da28047151d0e56f8dc6292773603d0d6aabbdd62a11ef721d1542d8
```

### 密码验证规则

1. **密码规范化**：验证前会去除空格并转换为大写
2. **多密码支持**：可以配置多个密码，任一密码验证通过即可
3. **算法一致性**：所有密码必须使用相同的算法

## 配置示例

### 仅 Stargate + Redis（密码认证 + 会话存 Redis）

```bash
# 必需配置
AUTH_HOST=auth.example.com
PASSWORDS=plaintext:test123

# 会话存储到 Redis（多实例或持久化会话）
SESSION_STORAGE_ENABLED=true
SESSION_STORAGE_REDIS_ADDR=redis:6379
SESSION_STORAGE_REDIS_PASSWORD=
SESSION_STORAGE_REDIS_DB=0
SESSION_STORAGE_REDIS_KEY_PREFIX=stargate:session:

# 可选
DEBUG=false
LANGUAGE=zh
```

适用于：单点密码登录，但希望会话集中在 Redis、多实例共享或重启不丢会话的场景。

### 基础配置（密码认证）

```bash
# 必需配置
AUTH_HOST=auth.example.com
PASSWORDS=plaintext:test123

# 可选配置
DEBUG=false
LANGUAGE=en
```

### 生产环境配置（密码认证）

```bash
# 必需配置
AUTH_HOST=auth.example.com
PASSWORDS=bcrypt:$2a$10$N9qo8uLOickgx2ZMRZoMyeIjZAgcfl7p92ldGxad68LJZdL17lhWy

# 可选配置
DEBUG=false
LANGUAGE=zh
LOGIN_PAGE_TITLE=我的认证服务
LOGIN_PAGE_FOOTER_TEXT=© 2024 我的公司
USER_HEADER_NAME=X-Forwarded-User
COOKIE_DOMAIN=.example.com
```

### Warden + Herald OTP 认证配置（生产环境推荐）

```bash
# 必需配置
AUTH_HOST=auth.example.com

# Warden 配置
WARDEN_ENABLED=true
WARDEN_URL=http://warden:8080
WARDEN_API_KEY=your-warden-api-key
WARDEN_CACHE_TTL=300

# Herald 配置
HERALD_ENABLED=true
HERALD_URL=http://herald:8080
HERALD_HMAC_SECRET=your-herald-hmac-secret

# 可选配置
DEBUG=false
LANGUAGE=zh
LOGIN_PAGE_TITLE=我的认证服务
LOGIN_PAGE_FOOTER_TEXT=© 2024 我的公司
USER_HEADER_NAME=X-Forwarded-User
COOKIE_DOMAIN=.example.com
```

**说明：**

- 此配置展示了如何同时使用 Warden 和 Herald 集成
- Warden 提供用户白名单验证
- Herald 提供 OTP/验证码发送和验证
- 生产环境推荐使用 HMAC 签名（`HERALD_HMAC_SECRET`）而非 API Key
- **注意**：这是可选配置，Stargate 可以独立使用密码认证，无需这些服务

### Docker Compose 配置

**密码认证模式：**

```yaml
services:
  stargate:
    image: stargate:latest
    environment:
      # 必需配置
      - AUTH_HOST=auth.example.com
      - PASSWORDS=bcrypt:$2a$10$N9qo8uLOickgx2ZMRZoMyeIjZAgcfl7p92ldGxad68LJZdL17lhWy
      
      # 可选配置
      - DEBUG=false
      - LANGUAGE=zh
      - LOGIN_PAGE_TITLE=我的认证服务
      - LOGIN_PAGE_FOOTER_TEXT=© 2024 我的公司
      - COOKIE_DOMAIN=.example.com
```

**Warden + Herald OTP 认证模式：**

```yaml
services:
  stargate:
    image: stargate:latest
    environment:
      # 必需配置
      - AUTH_HOST=auth.example.com
      
      # Warden 配置
      - WARDEN_ENABLED=true
      - WARDEN_URL=http://warden:8080
      - WARDEN_API_KEY=your-warden-api-key
      - WARDEN_CACHE_TTL=300
      
      # Herald 配置
      - HERALD_ENABLED=true
      - HERALD_URL=http://herald:8080
      - HERALD_HMAC_SECRET=your-herald-hmac-secret
      
      # 可选配置
      - DEBUG=false
      - LANGUAGE=zh
      - LOGIN_PAGE_TITLE=我的认证服务
      - LOGIN_PAGE_FOOTER_TEXT=© 2024 我的公司
      - COOKIE_DOMAIN=.example.com
    depends_on:
      - warden
      - herald

  warden:
    image: warden:latest
    environment:
      - WARDEN_DB_URL=postgres://user:pass@db:5432/warden
    # ... 其他配置

  herald:
    image: herald:latest
    environment:
      - HERALD_REDIS_URL=redis://redis:6379/0
    # ... 其他配置
```

### 本地开发配置

```bash
# 必需配置
AUTH_HOST=localhost
PASSWORDS=plaintext:test123|admin456

# 可选配置
DEBUG=true
LANGUAGE=zh
PORT=8080
```

## 配置验证

Stargate 在启动时会验证所有配置项：

1. **必需配置检查**：如果必需配置未设置，服务会启动失败并显示错误信息
2. **格式验证**：密码配置格式不正确时会启动失败
3. **算法验证**：不支持的密码算法会导致启动失败
4. **值验证**：某些配置项有可选值限制（如 `LANGUAGE`、`DEBUG`）

**错误示例：**

```bash
# 缺少必需配置
Error: Configuration error: environment variable 'AUTH_HOST' is required but not set.

# 密码格式错误
Error: Configuration error: invalid value for environment variable 'PASSWORDS': 'invalid_format'

# 不支持的算法
Error: Configuration error: invalid value for environment variable 'PASSWORDS': 'unknown:password'
```

## 配置依赖关系

### 配置项依赖

- **Warden 集成**：
  - `WARDEN_ENABLED=true` 时，必须设置 `WARDEN_URL`
  - `WARDEN_API_KEY` 建议设置（用于服务认证）
  - 若启用 `WARDEN_OTP_ENABLED=true`，需设置 `WARDEN_OTP_SECRET_KEY`

- **Herald 集成（OTP 短信/邮件）**：
  - `HERALD_ENABLED=true` 时，必须设置 `HERALD_URL`
  - 必须设置 `HERALD_API_KEY` 或 `HERALD_HMAC_SECRET` 之一（推荐生产环境使用 HMAC）
  - 可选：mTLS 通过 `HERALD_TLS_*` 配置；与 API Key/HMAC 可并存

- **Herald TOTP 集成（每用户 2FA）**：
  - `HERALD_TOTP_ENABLED=true` 时，建议设置 `HERALD_TOTP_BASE_URL`
  - 需设置 `HERALD_TOTP_API_KEY` 或 `HERALD_TOTP_HMAC_SECRET` 之一

- **会话存储**：
  - `SESSION_STORAGE_ENABLED=true` 时，需保证 Redis 可访问（默认 `SESSION_STORAGE_REDIS_ADDR=localhost:6379`）

- **Step-up 认证**：
  - `STEP_UP_ENABLED=true` 时，可通过 `STEP_UP_PATHS` 指定需二次认证的路径

- **授权刷新**：
  - `AUTH_REFRESH_ENABLED=true` 时，需启用 Warden（`WARDEN_ENABLED=true`），并可通过 `AUTH_REFRESH_INTERVAL` 调整刷新间隔

- **Warden + Herald 组合使用**（可选）：
  - 当需要 OTP 认证时，可以选择同时启用 Warden 和 Herald
  - Warden 提供用户白名单验证和用户信息
  - Herald 提供验证码发送和验证
  - **注意**：这些集成都是可选的，Stargate 可以独立使用

### 认证模式选择

1. **密码认证模式**（简单，适合开发/测试）：
   - 只需设置 `PASSWORDS`
   - 不需要 Warden 和 Herald

2. **Warden + Herald OTP 认证模式**（可选，适合需要高级功能的场景）：
   - 需要启用 `WARDEN_ENABLED=true` 和 `HERALD_ENABLED=true`
   - 提供用户白名单管理和 OTP 验证码功能
   - 更安全，支持限流和审计
   - **注意**：这是可选功能，Stargate 可以独立使用密码认证

## 配置最佳实践

1. **生产环境安全**：
   - 使用 `bcrypt` 或 `sha512` 算法，避免使用 `plaintext`
   - 设置 `DEBUG=false`
   - 使用强密码（密码认证模式）
   - 或使用 Warden + Herald OTP 认证（推荐）

2. **服务间安全**（如果启用了可选服务集成）：
   - 生产环境使用 `HERALD_HMAC_SECRET` 而非 `HERALD_API_KEY`
   - 确保 Warden 和 Herald 服务可访问
   - 配置适当的网络策略和防火墙规则

3. **跨域会话**：
   - 如果需要跨子域名共享会话，设置 `COOKIE_DOMAIN`
   - 格式：`.example.com`（注意前面的点）

4. **多语言支持**：
   - 根据用户群体设置 `LANGUAGE`
   - 支持 `en`、`zh`、`fr`、`it`、`ja`、`de`、`ko`

5. **自定义界面**：
   - 使用 `LOGIN_PAGE_TITLE` 和 `LOGIN_PAGE_FOOTER_TEXT` 自定义登录页面

6. **监控和调试**：
   - 开发环境设置 `DEBUG=true` 获取详细日志
   - 生产环境设置 `DEBUG=false` 减少日志输出

7. **性能优化**：
   - 设置 `WARDEN_CACHE_TTL` 以减少对 Warden 的请求
   - 根据实际需求调整缓存时间
