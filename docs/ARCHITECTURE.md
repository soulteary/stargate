# Stargate 架构文档

本文档描述了 Stargate 项目的技术架构和设计决策。

## 技术栈

- **语言**: Go 1.25
- **Web 框架**: [Fiber v2.52.10](https://github.com/gofiber/fiber)
- **模板引擎**: [Fiber Template v1.7.5](https://github.com/gofiber/template)
- **会话管理**: Fiber Session Middleware
- **日志**: [Logrus v1.9.3](https://github.com/sirupsen/logrus)
- **终端输出**: [Pterm v0.12.82](https://github.com/pterm/pterm)
- **测试框架**: [Testza v0.5.2](https://github.com/MarvinJWendt/testza)

## 项目结构

```
codes/src/
├── cmd/stargate/          # 应用程序入口点
│   ├── main.go            # 主函数，初始化配置和启动服务器
│   ├── server.go          # 服务器配置和路由设置
│   └── constants.go       # 路由和配置常量
│
├── internal/              # 内部包（不对外暴露）
│   ├── auth/              # 认证逻辑
│   │   ├── auth.go        # 认证核心功能
│   │   └── auth_test.go   # 认证测试
│   │
│   ├── config/            # 配置管理
│   │   ├── config.go      # 配置变量定义和初始化
│   │   ├── validation.go  # 配置验证逻辑
│   │   └── config_test.go # 配置测试
│   │
│   ├── handlers/          # HTTP 请求处理器
│   │   ├── check.go       # 认证检查处理器
│   │   ├── login.go       # 登录处理器
│   │   ├── logout.go      # 登出处理器
│   │   ├── session_share.go # 会话共享处理器
│   │   ├── health.go      # 健康检查处理器
│   │   ├── index.go       # 根路径处理器
│   │   ├── utils.go       # 处理器工具函数
│   │   └── handlers_test.go # 处理器测试
│   │
│   ├── i18n/              # 国际化支持
│   │   └── i18n.go        # 多语言翻译
│   │
│   ├── middleware/        # HTTP 中间件
│   │   └── log.go         # 日志中间件
│   │
│   ├── secure/            # 密码加密算法
│   │   ├── interface.go   # 加密算法接口
│   │   ├── plaintext.go   # 明文密码（仅测试）
│   │   ├── bcrypt.go      # BCrypt 算法
│   │   ├── md5.go         # MD5 算法
│   │   ├── sha512.go      # SHA512 算法
│   │   └── secure_test.go # 加密算法测试
│   │
│   └── web/               # Web 资源
│       └── templates/     # HTML 模板
│           ├── login.html # 登录页面模板
│           └── assets/   # 静态资源
│               └── favicon.ico
```

## 核心组件

### 1. 认证系统 (`internal/auth`)

认证系统负责：
- 密码验证（支持多种加密算法）
- 会话管理（创建、验证、销毁）
- 认证状态检查

**关键函数：**
- `CheckPassword(password string) bool`: 验证密码
- `Authenticate(session *session.Session) error`: 标记会话为已认证
- `IsAuthenticated(session *session.Session) bool`: 检查会话是否已认证
- `Unauthenticate(session *session.Session) error`: 销毁会话

### 2. 配置系统 (`internal/config`)

配置系统提供：
- 环境变量管理
- 配置验证
- 默认值支持

**配置变量：**
- `AUTH_HOST`: 认证主机名（必需）
- `PASSWORDS`: 密码配置（算法:密码列表）（必需）
- `DEBUG`: 调试模式（默认：false）
- `LANGUAGE`: 界面语言（默认：en，支持 en/zh）
- `COOKIE_DOMAIN`: Cookie 域名（可选，用于跨域会话共享）
- `LOGIN_PAGE_TITLE`: 登录页面标题（默认：Stargate - Login）
- `LOGIN_PAGE_FOOTER_TEXT`: 登录页面页脚文本（默认：Copyright © 2024 - Stargate）
- `USER_HEADER_NAME`: 认证成功后设置的用户头名称（默认：X-Forwarded-User）
- `PORT`: 服务监听端口（仅本地开发，默认：80）

### 3. 请求处理器 (`internal/handlers`)

处理器负责处理 HTTP 请求：

- **CheckRoute**: Traefik Forward Auth 认证检查
- **LoginRoute/LoginAPI**: 登录页面和登录处理
- **LogoutRoute**: 登出处理
- **SessionShareRoute**: 跨域会话共享
- **HealthRoute**: 健康检查
- **IndexRoute**: 根路径处理

### 4. 密码加密 (`internal/secure`)

支持多种密码加密算法：
- `plaintext`: 明文（仅用于测试）
- `bcrypt`: BCrypt 哈希
- `md5`: MD5 哈希
- `sha512`: SHA512 哈希

所有算法实现 `HashResolver` 接口：
```go
type HashResolver interface {
    Check(h string, password string) bool
}
```

## 工作流程

### 认证流程

1. **用户访问受保护资源**
   - Traefik 拦截请求
   - 转发到 Stargate `/_auth` 端点

2. **Stargate 检查认证**
   - 优先检查 `Stargate-Password` 头（API 认证）
   - 如果 Header 认证失败，检查 `stargate_session_id` Cookie（Web 认证）

3. **认证成功**
   - 设置 `X-Forwarded-User` 头（或配置的用户头名称），值为 "authenticated"
   - 返回 200 OK
   - Traefik 允许请求继续

4. **认证失败**
   - HTML 请求：重定向到登录页（`/_login?callback=<原始URL>`）
   - API 请求（JSON/XML）：返回 401 Unauthorized

### 登录流程

1. **用户访问登录页**
   - `GET /_login?callback=<url>`
   - 如果已登录，重定向到会话交换端点
   - 如果域名不一致，会将 callback 存储在 Cookie 中（`stargate_callback`）

2. **提交登录表单**
   - `POST /_login` 携带密码
   - 验证密码
   - 创建会话并设置 Cookie
   - **Callback 获取优先级**：
     1. 从 Cookie 中获取（如果之前已设置）
     2. 从表单数据中获取
     3. 从查询参数中获取
     4. 如果以上都没有，且来源域名与认证服务域名不一致，则使用来源域名作为 callback

3. **会话交换**
   - 如果有 callback，重定向到 `{callback}/_session_exchange?id=<session_id>`
   - `GET /_session_exchange?id=<session_id>`
   - 设置会话 Cookie（如果配置了 `COOKIE_DOMAIN`，会设置到指定域名）
   - 重定向到根路径 `/`

## 安全考虑

### 会话安全

- Cookie 使用 `HttpOnly` 标志，防止 XSS 攻击
- Cookie 使用 `SameSite=Lax`，防止 CSRF 攻击
- Cookie 路径设置为 `/`，允许在整个域名下使用
- 会话过期时间：24 小时（`config.SessionExpiration`）
- 支持自定义 Cookie 域名（用于跨域场景）
- 会话 ID 使用 UUID 生成，确保唯一性和安全性

### 密码安全

- 支持多种加密算法（推荐使用 bcrypt 或 sha512）
- 密码配置通过环境变量传递，不存储在代码中
- 密码验证时进行规范化处理（去除空格、转大写）

### 请求安全

- 认证检查端点支持两种认证方式：
  - Header 认证（`Stargate-Password`）：用于 API 请求
  - Cookie 认证：用于 Web 请求
- 区分 HTML 和 API 请求，返回适当的响应

## 扩展性

### 添加新的密码算法

1. 在 `internal/secure/` 创建新的算法实现
2. 实现 `HashResolver` 接口
3. 在 `config/validation.go` 中注册算法

### 添加新的语言

1. 在 `internal/i18n/i18n.go` 中添加语言常量
2. 添加翻译映射
3. 在配置中添加语言选项

### 自定义登录页面

修改 `internal/web/templates/login.html` 模板文件。

## 性能优化

- 使用 Fiber 框架，基于 fasthttp，性能优异
- 会话存储在内存中，访问快速
- 静态资源通过 Fiber 静态文件服务提供
- 支持调试模式，生产环境可关闭

## 部署架构

### Docker 部署

- 多阶段构建，减小镜像体积
- 使用 `golang:1.25-alpine` 作为构建阶段
- 使用 `scratch` 基础镜像作为运行阶段，最小化安全风险
- 模板文件从 `src/internal/web/templates` 复制到镜像中的 `/app/web/templates`
- 使用中国镜像源（`GOPROXY=https://goproxy.cn`）加速依赖下载
- 编译时使用 `-ldflags "-s -w"` 减小二进制体积
- 应用会自动查找模板路径（支持本地开发的 `./internal/web/templates` 和生产环境的 `./web/templates`）

### Traefik 集成

- 通过 Forward Auth 中间件集成
- 支持 HTTP 和 HTTPS
- 支持多域名和路径规则

## 日志和监控

- 使用 Logrus 进行日志记录
- 支持调试模式（DEBUG=true）
- 所有关键操作都有日志记录
- 健康检查端点可用于监控

## 测试

- 单元测试覆盖核心功能
- 测试文件位于各包的 `*_test.go` 文件中
- 使用 `testza` 进行断言
- 测试覆盖的模块：
  - 认证逻辑（`internal/auth/auth_test.go`）
  - 配置验证（`internal/config/config_test.go`）
  - 密码加密算法（`internal/secure/secure_test.go`）
  - HTTP 处理器（`internal/handlers/handlers_test.go`）

## 未来改进方向

- [ ] 支持更多密码加密算法
- [ ] 支持 OAuth2/OpenID Connect
- [ ] 支持多用户和角色管理
- [ ] 添加管理界面
- [ ] 支持 Redis 等外部会话存储
- [ ] 添加 Prometheus 指标导出
- [ ] 支持配置文件（YAML/JSON）
