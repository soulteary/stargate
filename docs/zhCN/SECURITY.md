# 安全文档

> 🌐 **Language / 语言**: [English](../enUS/SECURITY.md) | [中文](SECURITY.md) | [Français](../frFR/SECURITY.md) | [Italiano](../itIT/SECURITY.md) | [日本語](../jaJP/SECURITY.md) | [Deutsch](../deDE/SECURITY.md) | [한국어](../koKR/SECURITY.md)

本文档说明 Stargate 的安全特性、安全配置和最佳实践。

## 已实现的安全功能

1. **Forward Auth 保护**: 集中式认证层，用于保护后端服务
2. **密码校验**: 支持 bcrypt 与 plaintext（仅限本地开发）；MD5 和未加盐 SHA-512 会被拒绝
3. **安全会话管理**: 基于 Cookie 的会话，Cookie 域名可配置；会话有效期固定为 24 小时
4. **服务集成安全**: 生产环境中，Herald 要求所选的 HMAC 或 API Key 请求凭据；可选 mTLS 叠加传输保护，不替代请求认证。 Warden 仍使用其独立配置的 API Key/HMAC/mTLS 策略。
5. **会话共享安全**: 安全的跨域会话交换机制
6. **输入验证**: 严格验证所有输入参数
7. **错误处理**: 错误消息和其他响应数据由各端点定义；`DEBUG` 不是通用的不披露保证
8. **安全响应头**: 自动添加安全相关的 HTTP 响应头
9. **HTTPS 强制**: 生产环境应使用 HTTPS
10. **OTP 集成**: 与 Herald 的安全集成，用于 OTP/验证码认证

## 安全最佳实践

### 1. 生产环境配置

**必须配置项**:
- 生产环境的密码认证必须使用 bcrypt
- 配置 `COOKIE_DOMAIN` 以正确管理会话
- 通过反向代理（Traefik、Nginx 等）使用 HTTPS
- 将反向代理的来源 IP 或 CIDR 配置到 `TRUSTED_PROXIES`
- 配置安全的会话 Cookie 设置

**配置示例**:
```bash
export AUTH_HOST=auth.example.com
export PASSWORDS='bcrypt:$2a$10$N9qo8uLOickgx2ZMRZoMyeIjZAgcfl7p92ldGxad68LJZdL17lhWy'
export COOKIE_DOMAIN=.example.com
export TRUSTED_PROXIES=10.20.0.10
```

**说明**：Cookie Secure 默认启用，并可通过 `COOKIE_SECURE` 配置；SameSite 仍固定为 `Lax`。当前未提供 `COOKIE_SAME_SITE`、`SESSION_TTL`，详见 [CONFIG.md](CONFIG.md)。

### 2. 密码安全

**推荐做法**:
- ✅ 生产环境使用 bcrypt 密码哈希
- ✅ 在环境变量中存储密码哈希
- ✅ 为不同环境使用不同密码
- ✅ 定期轮换密码

**不推荐做法**:
- ❌ 在生产环境中使用明文密码
- ❌ 在配置文件中硬编码密码
- ❌ 配置 MD5 或未加盐 SHA-512；Stargate 会拒绝这两种配置
- ❌ 跨环境共享密码

**密码算法比较**:
- `bcrypt`: 已支持，生产环境推荐
- `plaintext`: 仅限本地开发与测试
- `sha512`: 已拒绝；快速、无盐的通用哈希不适合保存密码
- `md5`: 已拒绝；不适合保存密码

### 3. 会话安全

**会话配置**（以当前实现为准，完整配置项见 [CONFIG.md](CONFIG.md)）:
- **Cookie 域名**: 设置 `COOKIE_DOMAIN` 以在子域之间共享会话
- **Secure 标志**: 默认通过 `COOKIE_SECURE=true` 启用；仅本地 HTTP 开发环境可关闭
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
- 配置尽可能精确的 `TRUSTED_PROXIES` 白名单；默认忽略所有转发头
- 确保代理覆盖转发头，而不是追加客户端提供的不可信值
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
export WARDEN_HMAC_KEY_ID=stargate
export WARDEN_HMAC_SECRET=your-secret-key

export HERALD_ENABLED=true
export HERALD_URL=https://herald:8082
export HERALD_HMAC_SECRET=your-secret-key
```

`WARDEN_HMAC_KEY_ID` 和 `WARDEN_HMAC_SECRET` 必须同时配置，并与 Warden 的 `WARDEN_HMAC_KEYS` 中的一项一致；只设置其中一项时，Stargate 拒绝启动。`HERALD_HMAC_KEY_ID` 为可选项：Herald 没有有效的默认 Key ID 时需要设置。

如需将签名绑定到某一个 Warden 实例，还应将 `WARDEN_HMAC_AUDIENCE` 设置为该 Warden 实例在 `WARDEN_HMAC_AUDIENCES` 中列出的名称。此时 Stargate 以 v3 形式签名，为该实例签名的请求在共用同一密钥的另一个实例上无效。此功能要求 Warden 1.8.0 或更高版本；更早的 Warden 会拒绝 v3 请求。参见 [CONFIG.md](CONFIG.md#warden_hmac_audience)。

## API 安全

### 认证方式

Stargate 支持两种认证方式：

1. **请求头认证**（API 请求）
   - 请求头: `Stargate-Password: <password>`
   - 默认关闭。仅为受信任的旧客户端设置 `PASSWORD_HEADER_AUTH_ENABLED=true`。
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
export SESSION_STORAGE_ENABLED=true
export SESSION_STORAGE_REDIS_ADDR=redis:6379
export SESSION_STORAGE_REDIS_PASSWORD=your-redis-password
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

## 错误响应和调试验证码

Stargate 不提供 `MODE` 配置。错误响应由各端点分别定义。运行细节通常写入日志，
但不能把 `DEBUG` 当作通用的响应净化机制或不披露保证。

当 Stargate 以 `DEBUG=true` 且 `COOKIE_SECURE=false` 运行，并且所配置的 Herald 服务启用了测试模式
（`HERALD_TEST_MODE`）且返回非空 `debug_code` 时，Stargate 会在成功的
`POST /_send_verify_code` 响应中以 `debug_code` 返回该测试验证码；`POST /_send_reauth_code`
在相同条件下同样会返回。内置登录页会显示该验证码，并自动填入验证码输入框。
`COOKIE_SECURE=true`（默认值，HTTPS 部署使用该值）时，无论 `DEBUG` 如何设置，都不会返回该验证码。

该组合会向请求客户端披露等同认证凭据的验证码，只能用于本地开发或测试，绝不能用于生产。
生产环境应设置 Stargate `DEBUG=false`，并保持 Herald `HERALD_TEST_MODE` 关闭。

## 安全响应头

Stargate 自动添加以下安全相关的 HTTP 响应头：

- `X-Content-Type-Options: nosniff` - 防止 MIME 类型嗅探
- `X-Frame-Options: DENY` - 防止点击劫持
- `X-XSS-Protection: 0` - 关闭已被弃用的旧版浏览器 XSS 过滤器
- `Referrer-Policy: strict-origin-when-cross-origin`
- `Content-Security-Policy: default-src 'none'; script-src 'self'; style-src 'self'; img-src 'self' data:; connect-src 'self'; base-uri 'none'; frame-ancestors 'none'` - Stargate 的页面只从自身的 `/assets` 路径加载脚本和样式，不使用内联脚本或内联样式。`img-src` 允许 `data:`，用于 TOTP 绑定页面的二维码。未设置 `form-action`，因为登录或 Step-up 成功后会重定向到回调主机。
- `Strict-Transport-Security: max-age=31536000` - 仅在 `COOKIE_SECURE=true`（默认值）时发送。`COOKIE_SECURE=false` 的本地 HTTP 部署不会发送该响应头；其值不包含 `includeSubDomains`。

`/_auth` 的响应不会添加 `Content-Security-Policy` 和 `Strict-Transport-Security`。反向代理可能把被拒绝的 ForwardAuth 响应转发给浏览器，浏览器会把它当作受保护应用所在源的响应，而这两个响应头只适用于 Stargate 自身的主机。

## 跨域会话共享

Stargate 支持安全的跨域会话共享：

- **会话交换端点**: `GET /_session_exchange`
- **不透明票据**: AES-GCM 票据不暴露原始会话标识
- **受众校验**: 票据绑定到允许列表中的回调主机
- **防重放**: 票据短时过期且只能兑换一次

**安全注意事项**:
- 仅在受信任的域名之间共享会话
- 使用 HTTPS 进行会话交换
- 监控会话交换尝试

## OTP/验证码安全

使用 Herald 集成进行 OTP 认证时：

- **基于 Challenge**: 使用 challenge-verify 模型
- 生产环境中，Herald 要求所选的 HMAC 或 API Key 请求凭据；可选 mTLS 叠加传输保护，不替代请求认证。
- **限流**: Herald 处理限流
- **审计日志**: Herald 维护审计日志

## 统一响应

默认情况下，`POST /_send_verify_code` 以及通过 `POST /_login` 进行的 Warden 登录，
对 Warden 白名单内和白名单外的标识（手机号或邮箱）给出不同的响应；对白名单内的账号，
还会透露是否有可用的发送渠道、是否已绑定 TOTP。未登录的客户端可以利用这些差异，
探测出哪些手机号和邮箱在白名单内。

`LOGIN_UNIFORM_RESPONSES=true` 会消除响应内容上的这些差异。对同一个请求，无论标识
未知、已停用还是在白名单内，无论白名单内账号处于什么状态，也无论 Herald 返回什么，
这两个端点返回的状态码、响应头和响应体都相同。具体行为见英文版
[CONFIG.md](../enUS/CONFIG.md#login_uniform_responses) 和
[API.md](../enUS/API.md#uniform-responses)（中文版尚未包含对应章节）。

覆盖范围：

- `POST /_send_verify_code` 以及 `auth_method=warden` 的 `POST /_login` 的状态码、
  响应头和响应体，包括 JSON、XML 和纯文本三种格式。
- `challenge_id`：每个通过检查的请求都会得到一个随机句柄，因此它是否出现、
  形式如何，都不能说明标识是否在白名单内。Herald 的 challenge id 只保留在服务端。
- Herald 自身的应答（重发冷却、按用户和按发送目标的限制、用户被锁定、服务故障）
  以及 `Idempotency-Key` 的行为，都不会传递给客户端。
- 调试验证码：该模式下这两个端点不会返回 `debug_code`，即使 `DEBUG=true` 且
  `COOKIE_SECURE=false` 也是如此。
- 从 `phone` 到 `mail` 的回退。默认模式下，同一请求中的 `phone` 不在白名单内时，
  会改用 `mail` 查询。即使响应完全相同，这种回退仍会让某个白名单内邮箱的所有者
  判断出一个手机号是否在白名单内：只有与不在白名单内的手机号一同提交时，
  验证码才会发到该邮箱，用它登录才会成功。在该模式下，提交了 `phone` 的请求只按
  手机号处理，每个请求只向 Warden 查询一个标识。

不在覆盖范围内：

- **响应时间。** 响应时间没有被拉平。为白名单内的标识发送验证码时要调用 Herald
  并等待发送完成，为白名单内的标识登录时要调用 Herald 校验；两者都比针对白名单外
  标识的同类请求耗时更长，单个发送请求就可以通过延迟区分出来。该模式去掉的是
  响应内容中的确定性信号，并不能对测量时间的客户端隐藏白名单。
- **消息本身。** 白名单内的用户会收到验证码。和以前一样，任何知道标识的人都可以
  触发这条消息。
- **Herald 配额。** 针对白名单外标识的请求不会到达 Herald，也不消耗 Herald 的任何
  限额。如果一个客户端知道多个白名单内的标识，并且能收到其中一个标识的验证码，
  它可以先向这些标识发送验证码，把 Herald 的按 IP 限额用满，再为另一个标识请求
  验证码，然后根据自己的下一个验证码是否还能送达做出判断。未设置
  `HERALD_CLIENT_IP_HEADER` 时，Herald 把所有用户的请求都计在 Stargate 的地址上：
  一个客户端就能把所有人的限额用完，而被 Herald 拒绝的发送仍会显示为已发送。
  请设置 `HERALD_CLIENT_IP_HEADER`，并在 Herald 一侧同时设置
  `HERALD_TRUSTED_PROXY_HEADER` 和 `HERALD_TRUSTED_PROXIES`；在该模式下如果这个
  请求头名称为空，Stargate 会在启动时记录一条警告。这样限额就按客户端地址分别计算，
  但对上述客户端而言信号并未消除：只要 Herald 的按 IP 限额能被用满，信号就存在。
  `RATE_LIMIT_WINDOW` 取默认值一分钟时，一个客户端地址在任意一分钟内最多可以通过
  Stargate 请求 20 个验证码：`POST /_send_verify_code` 和 `POST /_send_reauth_code`
  两个端点各为 `RATE_LIMIT_VERIFICATION_MAX` 的两倍。把 Herald 的 `RATE_LIMIT_PER_IP`
  设置为不低于部署实际对应的这个数值，该限额就不会被用满。Herald 的默认值 5 做不到这一点。
- **直接访问 Warden 或 Herald。** 该模式只涉及 Stargate 的响应。

需要事先考虑的后果：

- 页面在什么都没有发送时也显示“已发送”，所有失败的登录都显示同一段文字。
  具体原因只记录在日志和审计日志中。
- 同时提交 `phone` 和 `mail` 的请求只按手机号处理。客户端如果两者都发送，
  而手机号不在白名单内，就收不到验证码，即使邮箱在白名单内也一样。登录页有这两个
  输入框：在该模式下，页面提示只填写其中一个；两个都填写时，页面显示该提示，
  不发送任何请求。
- 重发冷却在任何查询之前按提交的标识计数。任何知道某个标识的人都可以让它一直
  处于冷却窗口内，标识所有者自己的请求在窗口结束前会得到 `429`。对白名单内的标识，
  Herald 的冷却本来就允许这样做。按客户端的限额 `RATE_LIMIT_VERIFICATION_MAX`
  限制了一个客户端能同时占住多少个标识。
- 一个 `challenge_id` 要么绑定到它所在请求创建的 challenge，要么绑定到请求方出示的
  `challenge_id` 所对应的 challenge。因此，当 Herald 拒绝重发时，之前送达的验证码
  只对出示了自己所持句柄的客户端继续有效，登录页就是这样做的。请求如果什么都不出示，
  或者出示的句柄属于其他用户、渠道或发送目标，得到的 `challenge_id` 不绑定任何
  challenge。为某个标识请求验证码，永远不会得到另一个客户端所持 challenge 的句柄。

## 敏感操作的重新认证

Step-up（`/_step_up`）、TOTP 绑定和 TOTP 解绑要求提交属于会话用户本人的证明，仅有有效的会话 Cookie 是不够的：

- 已绑定 TOTP 的 Warden 用户必须提交当前的 TOTP 码。
- 其他 Warden 用户必须提交一次性验证码，该验证码由 Herald 发送到用户在 Warden 中登记的联系方式；请求不能指定发送目标。
- 使用共享密钥 `PASSWORDS` 创建的会话没有用户身份，需要再次输入该密码。只有这种会话才接受共享密码。

因此，登录后不久被窃取的会话 Cookie 无法用来绑定认证器；共享密码也不能替 Warden 用户通过 Step-up 或解绑 TOTP。

证明只对提交它时所针对的操作有效。提交给 `POST /totp/enroll` 的证明只在携带它的那次请求中启动一次绑定，这次绑定可在 5 分钟内确认；通过 Step-up 或解绑 TOTP 都不能让会话绑定认证器。会话中不会保留任何可供后续请求启动绑定的证明。如果证明通过后 Herald 调用失败，则不保留任何内容：用户需要重新获取验证码并再次提交，重发冷却和 Herald 的限制同样适用。因此，无论会话 Cookie 是在其所有者通过 Step-up 之后被窃取，还是在其所有者的一次启动请求因 Herald 失败之后被窃取，窃取者都无法绑定自己的认证器。Step-up 会把自身的通过结果记录 10 分钟：在此期间，该会话可以访问要求 Step-up 的路径，持有其 Cookie 副本的人同样可以。

该机制的局限：

- 在同时启用 Warden 和 `PASSWORDS` 的部署中，任何持有共享密码的人都可以用它登录，并在该会话中通过 Step-up。这样的会话不带用户身份。移除 `PASSWORDS` 即可关闭这条路径；启用 Herald 后，`STEP_UP_ENABLED` 不再要求配置 `PASSWORDS`。
- 用户如果同时丢失了认证器和备用码，就无法通过 Step-up，也无法解绑 TOTP，必须由管理员在 Herald 中移除该因子。
- 重新认证验证码在 Herald 中以 `login` 用途创建。Stargate 通过自身的 challenge 绑定把它与登录验证码区分开，但用户收到的消息内容以及 Herald 的限流规则都与登录验证码相同。

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

## 部署信任边界

Forward Auth 只能保护确实经过已配置反向代理的请求，它本身不会为后端服务提供网络隔离。

- 不要把受保护后端的端口暴露到公网或不可信网络。
- 仅允许反向代理所在网络或安全组访问后端。
- 转发前删除客户端自行提供的身份和凭据请求头，只允许可信认证链路生成的值。
- 仅向预期的运维网络暴露 Stargate 健康检查与指标端点。

### 来源时效与监控迁移

对撤销时效敏感的部署，应在支持快照元数据 v1 的 Warden 上启用 `WARDEN_SOURCE_MAX_AGE`，
并在 Warden 一侧同样强制检查来源新鲜度。每五分钟一次的会话刷新并不能限制底层目录
数据的陈旧程度。新增的来源检查以及独立的 `/metrics` Bearer 凭据见
[配置参考](CONFIG.md#来源新鲜度与监控迁移)。
