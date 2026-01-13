# API 文档

本文档详细描述了 Stargate Forward Auth Service 的所有 API 端点。

## 目录

- [认证检查端点](#认证检查端点)
- [登录端点](#登录端点)
- [登出端点](#登出端点)
- [会话交换端点](#会话交换端点)
- [健康检查端点](#健康检查端点)
- [根端点](#根端点)

## 认证检查端点

### `GET /_auth`

Traefik Forward Auth 的主要认证检查端点。此端点是 Stargate 的核心功能，用于验证用户是否已通过身份验证。

#### 认证方式

Stargate 支持两种认证方式，按以下优先级检查：

1. **Header 认证**（API 请求）
   - 请求头：`Stargate-Password: <password>`
   - 适用于 API 请求、自动化脚本等场景

2. **Cookie 认证**（Web 请求）
   - Cookie：`stargate_session_id=<session_id>`
   - 适用于浏览器访问的 Web 应用

#### 请求头

| 请求头 | 类型 | 必需 | 说明 |
|--------|------|------|------|
| `Stargate-Password` | String | 否 | 用于 API 请求的密码认证 |
| `Cookie` | String | 否 | 包含 `stargate_session_id` 的会话 Cookie |
| `Accept` | String | 否 | 用于判断请求类型（HTML/API） |

#### 响应

**成功响应（200 OK）**

认证成功时，Stargate 会设置用户信息头并返回 200 状态码：

```
HTTP/1.1 200 OK
X-Forwarded-User: authenticated
```

用户头名称可通过 `USER_HEADER_NAME` 环境变量配置（默认：`X-Forwarded-User`）。

**失败响应**

| 状态码 | 说明 | 响应体 |
|--------|------|--------|
| `401 Unauthorized` | 认证失败 | 错误消息（JSON 格式，API 请求）或重定向到登录页（HTML 请求） |
| `500 Internal Server Error` | 服务器错误 | 错误消息 |

#### 请求类型处理

- **HTML 请求**：认证失败时重定向到 `/_login?callback=<原始URL>`
- **API 请求**（JSON/XML）：认证失败时返回 401 错误响应

#### 示例

**使用 Header 认证（API 请求）**

```bash
curl -H "Stargate-Password: yourpassword" \
     http://auth.example.com/_auth
```

**使用 Cookie 认证（Web 请求）**

```bash
curl -H "Cookie: stargate_session_id=<session_id>" \
     http://auth.example.com/_auth
```

## 登录端点

### `GET /_login`

显示登录页面。

#### 查询参数

| 参数 | 类型 | 必需 | 说明 |
|------|------|------|------|
| `callback` | String | 否 | 登录成功后的回调 URL（通常是原始请求的域名） |

#### 行为

- 如果用户已登录，自动重定向到会话交换端点
- 如果用户未登录，显示登录页面
- 如果 URL 中有 `callback` 参数且域名不一致，会将 callback 存储在 `stargate_callback` Cookie 中（10 分钟过期）

#### Callback 获取优先级

1. **从查询参数获取**：URL 中的 `callback` 参数（优先级最高）
2. **从 Cookie 获取**：如果查询参数中没有，则从 `stargate_callback` Cookie 中获取

#### 响应

**200 OK** - 返回登录页面 HTML

页面包含：
- 登录表单
- 可自定义的标题（`LOGIN_PAGE_TITLE`）
- 可自定义的页脚文本（`LOGIN_PAGE_FOOTER_TEXT`）

#### 示例

```bash
# 访问登录页面
curl http://auth.example.com/_login?callback=app.example.com
```

### `POST /_login`

处理登录请求，验证密码并创建会话。

#### 请求体

表单数据（`application/x-www-form-urlencoded`）：

| 字段 | 类型 | 必需 | 说明 |
|------|------|------|------|
| `password` | String | 是 | 用户密码 |
| `callback` | String | 否 | 登录成功后的回调 URL |

#### Callback 获取优先级

登录处理会按以下优先级获取 callback：

1. **从 Cookie 获取**：如果之前访问登录页时域名不一致，callback 会存储在 `stargate_callback` Cookie 中
2. **从表单数据获取**：POST 请求的表单数据中的 `callback` 字段
3. **从查询参数获取**：URL 查询参数中的 `callback`
4. **自动推断**：如果以上都没有，且来源域名（`X-Forwarded-Host`）与认证服务域名不一致，则使用来源域名作为 callback

#### 响应

**成功响应（200 OK）**

根据是否有 callback 和请求类型，响应会有所不同：

1. **有 callback 时**：
   - 重定向到 `{callback}/_session_exchange?id={session_id}`
   - 状态码：`302 Found`

2. **无 callback 时**：
   - **HTML 请求**：返回包含 meta refresh 的 HTML 页面，自动跳转到来源域名
   - **API 请求**：返回 JSON 响应
     ```json
     {
       "success": true,
       "message": "Login successful",
       "session_id": "<session_id>"
     }
     ```

**失败响应**

| 状态码 | 说明 | 响应体 |
|--------|------|--------|
| `401 Unauthorized` | 密码错误 | 根据 Accept 头返回 JSON/XML/文本格式的错误消息 |
| `500 Internal Server Error` | 服务器错误 | 错误消息 |

#### 示例

```bash
# 提交登录表单（带 callback）
curl -X POST \
     -d "password=yourpassword&callback=app.example.com" \
     -c cookies.txt \
     http://auth.example.com/_login

# 提交登录表单（不带 callback，会自动推断）
curl -X POST \
     -d "password=yourpassword" \
     -H "X-Forwarded-Host: app.example.com" \
     -c cookies.txt \
     http://auth.example.com/_login
```

## 登出端点

### `GET /_logout`

登出当前用户，销毁会话。

#### 响应

**成功响应（200 OK）**

```
HTTP/1.1 200 OK
Content-Type: text/plain

Logged out
```

会话 Cookie 会被清除。

#### 示例

```bash
curl -b cookies.txt http://auth.example.com/_logout
```

## 会话交换端点

### `GET /_session_exchange`

用于跨域会话共享。设置指定会话 ID 的 Cookie 并重定向到根路径。

此端点主要用于在多个域名/子域名之间共享认证会话。当用户在一个域名登录后，可以通过此端点将会话 Cookie 设置到另一个域名。

#### 查询参数

| 参数 | 类型 | 必需 | 说明 |
|------|------|------|------|
| `id` | String | 是 | 要设置的会话 ID |

#### 响应

**成功响应（302 Redirect）**

```
HTTP/1.1 302 Found
Location: /
Set-Cookie: stargate_session_id=<session_id>; Path=/; HttpOnly; SameSite=Lax; Domain=<cookie_domain>; Expires=<expiry>
```

**失败响应**

| 状态码 | 说明 | 响应体 |
|--------|------|--------|
| `400 Bad Request` | 缺少会话 ID | 错误消息 |

#### Cookie 域名

如果配置了 `COOKIE_DOMAIN` 环境变量，Cookie 会设置到指定的域名，从而实现跨子域名共享。

#### 示例

```bash
# 设置会话 Cookie（用于跨域场景）
curl "http://auth.example.com/_session_exchange?id=<session_id>"
```

**典型使用场景：**

1. 用户在 `auth.example.com` 登录
2. 登录成功后，重定向到 `app.example.com/_session_exchange?id=<session_id>`
3. 会话 Cookie 被设置到 `.example.com` 域名（如果配置了 `COOKIE_DOMAIN=.example.com`）
4. 重定向到 `app.example.com/`
5. 用户可以在所有 `*.example.com` 子域名下使用该会话

## 健康检查端点

### `GET /health`

服务健康检查端点。用于监控服务状态。

#### 响应

**成功响应（200 OK）**

```
HTTP/1.1 200 OK
```

#### 示例

```bash
curl http://auth.example.com/health
```

**典型用途：**

- Docker 健康检查
- Kubernetes 存活探针
- 负载均衡器健康检查

## 根端点

### `GET /`

根路径，显示服务信息。

#### 响应

**200 OK** - 返回服务信息页面

#### 示例

```bash
curl http://auth.example.com/
```

## 错误响应格式

所有 API 错误响应根据客户端的 `Accept` 头自动选择格式：

### JSON 格式（`Accept: application/json`）

```json
{
  "error": "错误消息",
  "code": 401
}
```

### XML 格式（`Accept: application/xml`）

```xml
<errors>
  <error code="401">错误消息</error>
</errors>
```

### 文本格式（默认）

```
错误消息
```

错误消息支持国际化，根据 `LANGUAGE` 环境变量返回中文或英文消息。

## 认证流程示例

### Web 应用认证流程

1. 用户访问受保护资源（如 `https://app.example.com/dashboard`）
2. Traefik 拦截请求，转发到 `https://auth.example.com/_auth`
3. Stargate 检查 Cookie 中的会话
4. 如果未认证，重定向到 `https://auth.example.com/_login?callback=app.example.com`
5. 用户输入密码并提交
6. Stargate 验证密码，创建会话，设置 Cookie
7. 重定向到 `https://app.example.com/_session_exchange?id=<session_id>`
8. 会话 Cookie 被设置到 `app.example.com` 域名
9. 用户再次访问受保护资源，认证成功

### API 认证流程

1. API 客户端发送请求到受保护资源
2. Traefik 拦截请求，转发到 `https://auth.example.com/_auth`
3. API 客户端在请求头中包含 `Stargate-Password: <password>`
4. Stargate 验证密码
5. 如果验证成功，设置 `X-Forwarded-User` 头并返回 200
6. Traefik 允许请求继续到后端服务

## 注意事项

1. **会话过期时间**：默认 24 小时，过期后需要重新登录
2. **Cookie 安全**：所有 Cookie 都设置了 `HttpOnly` 和 `SameSite=Lax` 标志
3. **密码验证**：密码在验证前会进行规范化处理（去除空格、转大写）
4. **多密码支持**：可以配置多个密码，任一密码验证通过即可
5. **跨域会话**：需要配置 `COOKIE_DOMAIN` 环境变量才能实现跨域会话共享
