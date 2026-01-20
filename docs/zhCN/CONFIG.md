# 配置参考

本文档详细说明 Stargate 的所有配置选项。

## 目录

- [配置方式](#配置方式)
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
| **可选值** | `en`（英文）, `zh`（中文） |

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

### `PORT`

服务监听端口（仅用于本地开发）。

| 属性 | 值 |
|------|-----|
| **类型** | String |
| **必需** | 否 |
| **默认值** | `80` |

**说明：**

- 仅用于本地开发环境
- Docker 容器中通常不需要设置（使用默认 80 端口）
- 格式：端口号（如 `8080`）或 `:端口号`（如 `:8080`）

**示例：**

```bash
PORT=8080
```

### Herald 集成

Herald 是 OTP/验证码服务。以下配置选项用于与 Herald 集成。

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
- 建议在使用 Warden 认证的生产环境中启用

**示例：**

```bash
HERALD_ENABLED=true
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

### 基础配置

```bash
# 必需配置
AUTH_HOST=auth.example.com
PASSWORDS=plaintext:test123

# 可选配置
DEBUG=false
LANGUAGE=en
```

### 生产环境配置

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

### Docker Compose 配置

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

## 配置最佳实践

1. **生产环境安全**：
   - 使用 `bcrypt` 或 `sha512` 算法，避免使用 `plaintext`
   - 设置 `DEBUG=false`
   - 使用强密码

2. **跨域会话**：
   - 如果需要跨子域名共享会话，设置 `COOKIE_DOMAIN`
   - 格式：`.example.com`（注意前面的点）

3. **多语言支持**：
   - 根据用户群体设置 `LANGUAGE`
   - 支持 `en` 和 `zh`

4. **自定义界面**：
   - 使用 `LOGIN_PAGE_TITLE` 和 `LOGIN_PAGE_FOOTER_TEXT` 自定义登录页面

5. **监控和调试**：
   - 开发环境设置 `DEBUG=true` 获取详细日志
   - 生产环境设置 `DEBUG=false` 减少日志输出
