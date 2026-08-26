# 从 v0.12.0 迁移到 v1.0.0

Stargate v1.0.0 确立了首个稳定的部署与 HTTP 契约，同时调整了多项安全默认值。生产切流前，请先在预发布环境完成升级验证。

## 升级前准备

1. 记录当前镜像、环境变量、反向代理路由、回调域名和健康检查配置。
2. 备份部署配置，并保留 v0.12.0 镜像以便回滚。
3. 找出所有调用 `/_logout`、`/totp/enroll` 和跨域会话交换路由的客户端。
4. 如果运行多个 Stargate 副本，请在升级前准备 Redis 会话存储。

## 必须调整的部署配置

### 官方容器端口与运行用户

| 契约 | v0.12.0 | v1.0.0 |
| --- | --- | --- |
| 官方容器端口 | `80` | `8080` |
| 运行用户 | root | UID/GID `10001` |
| 存活探针 | `/health` | `/healthz` |
| 依赖就绪探针 | `/health` | `/readyz` |

反向代理目标和容器健康检查需要同时更新。最小 Compose 服务示例：

```yaml
services:
  stargate:
    image: ghcr.io/soulteary/stargate:v1.0.0
    ports:
      - "8080:8080"
    read_only: true
    tmpfs:
      - /tmp
    healthcheck:
      test: ["CMD", "wget", "--quiet", "--tries=1", "--spider", "http://127.0.0.1:8080/healthz"]
```

使用 Traefik ForwardAuth 时，将目标改为 `http://stargate:8080/_auth`。

### 跨域会话交换

v1.0.0 使用加密、短时、单次有效且绑定目标域名的 `?ticket=` 替代直接传递会话 ID 的 `?id=`。

- 在 `CALLBACK_ALLOWED_HOSTS` 中列出所有允许的回调域名。
- 所有副本必须使用相同的 `SESSION_EXCHANGE_SECRET`；启用跨域回调时，其长度至少为 32 个字符。
- HTTPS 部署保持 `COOKIE_SECURE=true`。
- 下游应用不要自行拼接交换 URL，应直接跟随 Stargate 返回的重定向。
- 当票据或会话需要被多个副本接受时，使用 Redis 会话存储。

旧的 `?id=` 交换链接与 v1.0.0 不兼容。

### 代理信任边界

只有直接上游地址匹配 `TRUSTED_PROXIES` 时，Stargate 才接受转发请求头。该配置应仅包含直接连接 Stargate 的反向代理 IP 或 CIDR。`PROXY_HEADER` 用于选择客户端 IP 请求头，默认值为 `X-Forwarded-For`。

不要将任意客户端网段加入 `TRUSTED_PROXIES`。

### 认证模式

- 登录表单仍通过 `POST /_login` 提交。
- `Stargate-Password` 请求头认证默认关闭。仅在可信代理集成中设置 `PASSWORD_HEADER_AUTH_ENABLED=true`，并确保代理先删除客户端传入的同名请求头。
- 可信身份请求头需要同时启用 `HEADER_AUTH_ENABLED=true`、配置匹配的 `HEADER_AUTH_SHARED_SECRET` 并接入 Warden。代理必须先删除客户端传入的身份和密钥请求头，再写入可信值。
- Warden 或 Herald 客户端配置无效时，服务现在会在启动阶段直接失败，而不是以不完整状态运行。
- 已移除旧版 Warden 全局 OTP 回退逻辑，请使用 Herald 支持的 TOTP 注册与校验。

## HTTP 契约变化

| 旧契约 | v1.0.0 契约 | 迁移动作 |
| --- | --- | --- |
| `GET /_logout` | `POST /_logout` | 将链接或客户端改为提交 POST 请求。 |
| `GET /totp/enroll` | `POST /totp/enroll` | 使用 POST 发起注册。 |
| 所有探针使用 `GET /health` | `GET /healthz` 与 `GET /readyz` | 区分进程存活和依赖就绪。 |
| 无指标端点 | `GET /metrics` | 将 Prometheus 兼容采集目标指向此端点。 |
| 会话交换 `?id=` | 会话交换 `?ticket=` | 跟随 Stargate 重定向，不再暴露会话 ID。 |

`GET /health` 仅作为已弃用的就绪检查兼容别名保留，新部署不应继续使用。浏览器中改变认证状态的请求会受到同源检查和端点级限流保护。

## 配置行为变化

- `AUTH_REFRESH_ENABLED` 默认值改为 `true`。只有能够接受 Warden 撤权延迟时才应关闭。
- `COOKIE_SECURE` 默认值为 `true`；本地 HTTP 测试需要显式关闭。
- 启动校验会拒绝格式错误的 URL、端口、时间间隔、Redis DB、过短的交换密钥，以及不完整的 TLS 证书/私钥和 HMAC Key ID/Secret 组合。
- 只有在 `CALLBACK_ALLOWED_HOSTS` 中显式允许，才能使用跨域回调目标。
- 启用 Warden 或 Herald 的 HMAC/TLS 后必须提供完整配置；Herald HMAC 认证还需要 `HERALD_HMAC_KEY_ID`。

切流前，使用生产环境变量启动 v1.0.0 镜像，并逐项修复所有启动校验错误。

## 发布步骤

1. 使用新端口和探针将 v1.0.0 部署到独立实例池。
2. 多副本部署先启用并验证共享 Redis 会话存储，再接收用户流量。
3. 验证部署实际启用的密码、Warden、请求头和 TOTP 认证流程。
4. 验证同域以及所有允许域名的跨域回调。
5. 仅在 `/healthz` 和 `/readyz` 均成功后切换流量。
6. 迁移期间将 v0.12.0 与 v1.0.0 保持在不同实例池，不要混用两种会话交换契约。

## 验证清单

```bash
curl --fail http://127.0.0.1:8080/healthz
curl --fail http://127.0.0.1:8080/readyz
curl --fail http://127.0.0.1:8080/metrics
```

- 容器以 UID/GID `10001` 运行，且不依赖可写根文件系统。
- 反向代理可以访问 `http://stargate:8080/_auth`。
- 登录成功，登出使用 `POST /_logout`。
- 配置 Herald 后，TOTP 注册、确认、验证和撤销均正常。
- 未允许的回调域名，以及重复使用或过期的交换票据会被拒绝。
- 客户端自行传入密码或身份请求头不能绕过可信代理。
- 依赖故障时 `/healthz` 仍表示存活，而 `/readyz` 返回未就绪。

## 回滚

同时恢复 v0.12.0 镜像、代理端口和旧探针。v0.12.0 无法消费 v1.0.0 的交换票据，因此回滚后应作废尚未完成的跨域交换，并要求用户重新登录。不要假设活动会话可以在混合版本实例池之间迁移。
