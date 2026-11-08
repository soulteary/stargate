# 部署指南

升级至 v1.3.5 前，请阅读[认证优先级升级与回滚](#认证优先级升级与回滚v135)。有效显式凭据不再依赖无关旧 Cookie 的目录刷新。

升级至 v1.3.4 前，请阅读[依赖补丁升级与回滚](#依赖补丁升级与回滚v134)。本次更新调整脱敏行为和依赖版本，保留既有会话与验证码状态。


当前部署示例使用 v1.3.5。替换 v1.3 之前的部署前，请阅读[可撤销会话迁移](../enUS/DEPLOYMENT.md#revocable-session-migration)和[浏览器绑定会话交换](../enUS/DEPLOYMENT.md#browser-bound-session-exchange)要求。须协调全部副本，保留所需 Redis 状态，并按迁移说明安排重新登录。

本文档提供 Stargate Forward Auth Service 的详细部署指南。

v1.3.3 升级前请阅读[Herald 凭据选择](#herald-凭据选择-v133)，尤其是同时配置 API key 与 HMAC 的客户端。先为全部现有调用方配置并验证显式 key ID，再升级 Herald v1.4.3，最后升级 Stargate v1.3.3。此补丁保留 Redis 记录和会话截止时间。

升级 v1.3.2 前先部署 herald-totp v2.0.2，再部署 Herald v1.4.2，最后切换网关。本文末尾说明 OTP 验证的 503/409 错误语义及回滚行为；保留 v1.3.1 已有记录和会话截止时间。

升级 v1.3.1 前先部署 herald-totp v2.0.1，再部署 Herald v1.4.1，最后按本文的登录验证码绑定升级与回滚说明切换网关。保留 v1.3.0 已有的可撤销会话、撤销标记、期限配置及命名空间；本补丁不迁移会话格式，也不延长现有截止时间。

升级 v1.3.0 前请阅读本文末尾的会话迁移说明及[发行说明](../../CHANGELOG.md)。现有会话需要重新登录，后端服务升级与 SSO 副本切换需要协调进行。

从 v0.12.0 升级时，请先阅读 [v1.0.0 迁移指南](MIGRATION_V1.md)，再切换生产流量。

## 目录

- [部署方式](#部署方式)
- [直接运行二进制](#直接运行二进制)
- [Docker 部署](#docker-部署)
- [Docker Compose 部署](#docker-compose-部署)
- [Traefik 集成](#traefik-集成)
- [生产环境部署](#生产环境部署)
- [监控和维护](#监控和维护)
- [故障排查](#故障排查)

## 部署方式

Stargate 支持以下部署方式：

1. **Docker 容器**（推荐）- 最简单、最常用
2. **Docker Compose** - 适合本地开发和测试
3. **Kubernetes** - 适合大规模生产环境
4. **直接运行二进制** - 适合特殊场景

本文档主要介绍 Docker 和 Docker Compose 部署方式。

## 直接运行二进制

发布二进制的构建目标为 Linux、macOS、Windows 的 amd64/arm64；容器镜像支持 Linux
amd64/arm64。二进制不内嵌 HTML 模板和静态资源，需要从对应源码标签安装完整的
`src/internal/web/templates` 目录，包括其中的 `assets` 子目录。

模板和资源路径相对于进程工作目录，查找顺序为 `./internal/web/templates`、
`./web/templates`、`./src/internal/web/templates`。可从源码根目录运行，或在安装
工作目录提供匹配版本的 `web/templates`。发布镜像已在 `/app/web/templates` 中包含
这些文件。`/healthz` 成功不能证明模板可用，还应检查未登录页面，例如执行
`curl -fsS -o /dev/null http://127.0.0.1:8080/_login`，不输出响应正文。

## 服务依赖

Stargate 可以与以下可选服务集成：

### Warden 服务

**功能：** 用户白名单管理和用户信息提供

**部署要求：**
- 使用本地 JSON 数据文件和/或远程 API；Warden 本身不要求 SQL 数据库。
- 提供 HTTP API 接口
- 支持 API Key 认证

**配置：**
```bash
WARDEN_ENABLED=true
WARDEN_URL=http://warden:8081
WARDEN_API_KEY=your-api-key
```

### Herald 服务

**功能：** OTP/验证码发送和验证

**部署要求：**
- 需要 Redis（存储 challenge 和限流状态）
- 提供 HTTP API 接口
- 生产环境中，Herald 要求所选的 HMAC 或 API Key 请求凭据；可选 mTLS 叠加传输保护，不替代请求认证。

**配置：**
```bash
HERALD_ENABLED=true
HERALD_URL=http://herald:8082
HERALD_HMAC_SECRET=your-hmac-secret  # 生产环境推荐
```

### 服务间通信安全

**生产环境要求：**

1. **HMAC 签名认证**（推荐）：
   - Stargate ↔ Herald 使用 HMAC-SHA256 签名
   - 配置 `HERALD_HMAC_SECRET`
   - 包含时间戳校验（防止重放攻击）

2. **mTLS 认证**（可选，更安全）：
   - 配置 TLS 客户端证书
   - 设置 `HERALD_TLS_CLIENT_CERT_FILE` 和 `HERALD_TLS_CLIENT_KEY_FILE`
   - 配置 CA 证书验证

3. **网络隔离**：
   - 服务间通信应在内网进行
   - 使用防火墙规则限制访问
   - 避免将服务暴露到公网

## Docker 部署

### 构建镜像

#### 从源码构建

在项目根目录下执行：

```bash
docker build -f docker/Dockerfile -t stargate:latest .
```

#### 构建参数

- **基础镜像**：`golang:1.27.3-alpine3.24`（构建阶段）
- **运行镜像**：`alpine:3.24`（运行阶段，包含 CA 证书和用于 HTTPS/健康检查的 BusyBox `wget`）
- **工作目录**：`/app`
- **暴露端口**：`8080`

### 运行容器

#### 基础运行

```bash
docker run -d \
  --name stargate \
  -p 8080:8080 \
  -e AUTH_HOST=auth.example.com \
  -e 'PASSWORDS=plaintext:yourpassword' \
  stargate:latest
```

#### 完整配置运行

```bash
docker run -d \
  --name stargate \
  -p 8080:8080 \
  -e AUTH_HOST=auth.example.com \
  -e 'PASSWORDS=bcrypt:$2a$10$N9qo8uLOickgx2ZMRZoMyeIjZAgcfl7p92ldGxad68LJZdL17lhWy' \
  -e DEBUG=false \
  -e LANGUAGE=zh \
  -e 'LOGIN_PAGE_TITLE=我的认证服务' \
  -e 'LOGIN_PAGE_FOOTER_TEXT=© 2024 我的公司' \
  -e COOKIE_DOMAIN=.example.com \
  --restart unless-stopped \
  stargate:latest
```

#### 参数说明

- `-d`：后台运行
- `--name stargate`：容器名称
- `-p 8080:8080`：端口映射（主机端口:容器端口）
- `-e`：环境变量
- `--restart unless-stopped`：自动重启策略

### 查看日志

```bash
# 查看实时日志
docker logs -f stargate

# 查看最近 100 行日志
docker logs --tail 100 stargate
```

### 停止和删除

```bash
# 停止容器
docker stop stargate

# 删除容器
docker rm stargate

```

## Docker Compose 部署

### 基础配置

仓库根目录的 `docker-compose.yml` 是自带完整栈：它在 Compose 管理的 `stargate-traefik` 网络（`172.30.0.0/24`）中启动 Traefik、Stargate 和受保护的演示服务，不依赖预先存在的外部网络。

```bash
docker compose config >/dev/null
docker compose up -d
```

校验命令会丢弃标准输出，因为 Compose 解析后的配置可能包含凭据。既有终端或
CI 输出不会自动改写；若其中实际包含凭据，应限制访问，按保留策略处理日志及
导出副本，并轮换已暴露的凭据。

### 接入已有的外部 Traefik 网络

仅当 Traefik 已运行在共享的外部 Docker 网络时，才使用下面这套独立示例。Compose 网络、每个 `traefik.docker.network` label 和 `TRUSTED_PROXIES` 都由实际网络名称和 inspect 得到的 CIDR 驱动。

```yaml
services:
  stargate:
    image: ghcr.io/soulteary/stargate:1.3.5
    environment:
      - AUTH_HOST=auth.test.localhost
      - PASSWORDS=plaintext:test1234|test1337
      - CALLBACK_ALLOWED_HOSTS=hello.test.localhost
      - SESSION_EXCHANGE_SECRET=local-development-session-secret-change-me
      - TRUSTED_PROXIES=${TRAEFIK_NETWORK_CIDR:?set TRAEFIK_NETWORK_CIDR from docker network inspect}
      - COOKIE_SECURE=false # 仅本地 HTTP；HTTPS 部署请省略。
    networks:
      - traefik
    labels:
      - traefik.enable=true
      - traefik.docker.network=${TRAEFIK_NETWORK_NAME:-traefik}
      - traefik.http.routers.auth.entrypoints=http
      - traefik.http.routers.auth.rule=Host(`auth.test.localhost`) || Path(`/_session_exchange`)
      - traefik.http.middlewares.stargate.forwardauth.address=http://stargate:8080/_auth
      - "traefik.http.middlewares.stargate.forwardauth.authResponseHeaders=X-Forwarded-User,X-Auth-User,X-Auth-Email,X-Auth-Name,X-Auth-Scopes,X-Auth-Role,X-Auth-AMR"

  hello:
    image: ghcr.io/soulteary/hello:2.3.0
    command: ["-listen", ":8080"]
    networks:
      - traefik
    labels:
      - traefik.enable=true
      - traefik.docker.network=${TRAEFIK_NETWORK_NAME:-traefik}
      - traefik.http.routers.hello.entrypoints=http
      - traefik.http.routers.hello.rule=Host(`hello.test.localhost`)
      - "traefik.http.routers.hello.middlewares=stargate"
      - traefik.http.services.hello.loadbalancer.server.port=8080

networks:
  traefik:
    name: ${TRAEFIK_NETWORK_NAME:-traefik}
    external: true
```

### 完整配置（Warden + Herald OTP 认证）

下面是对接独立部署 Warden 和 Herald 的 Stargate 端配置：

这是 API-key 模式下的 Stargate 端配置。Warden 和 Herald 应分别按照其
[Warden 部署指南](https://github.com/soulteary/warden/blob/v1.4.4/docs/enUS/DEPLOYMENT.md)
和 [Herald 部署指南](https://github.com/soulteary/herald/blob/v1.3.13/docs/enUS/DEPLOYMENT.md)
部署，并落实各自的数据和持久化要求。提供容器内可达的 URL 和匹配的凭据；
上游默认端口分别为 Warden `8081`、Herald `8082`。需显式配置共享网络或可路由
地址。使用 Herald HMAC 模式时，改用匹配的 `HERALD_HMAC_SECRET` 和
`HERALD_HMAC_KEY_ID` 配置。Warden 用户 ID 必须稳定，并保留部署所需的会话和
重放状态持久化。本次文档修正不会迁移或删除已有服务数据和数据卷。

```yaml
services:
  stargate:
    image: ghcr.io/soulteary/stargate:1.3.5
    environment:
      - AUTH_HOST=auth.example.com
      - WARDEN_ENABLED=true
      - WARDEN_URL=${WARDEN_URL:?set the reachable Warden URL}
      - WARDEN_API_KEY=${WARDEN_API_KEY:?set the matching Warden API key}
      - HERALD_ENABLED=true
      - HERALD_URL=${HERALD_URL:?set the reachable Herald URL}
      - HERALD_API_KEY=${HERALD_API_KEY:?set the matching Herald API key}
      - CALLBACK_ALLOWED_HOSTS=app.example.com
      - COOKIE_DOMAIN=.example.com
      - SESSION_EXCHANGE_SECRET=${SESSION_EXCHANGE_SECRET:?set at least 32 random characters}
```

### 检查或创建外部网络

如果网络已经存在，下面的命令会检查并复用它分配的全部 CIDR；不要删除或重建已有生产网络。如果网络不存在，命令才会使用文档指定的子网创建它。

```bash
export TRAEFIK_NETWORK_NAME="${TRAEFIK_NETWORK_NAME:-traefik}"

if docker network inspect "$TRAEFIK_NETWORK_NAME" >/dev/null 2>&1; then
  export TRAEFIK_NETWORK_CIDR="$(
    docker network inspect "$TRAEFIK_NETWORK_NAME" \
      --format '{{range .IPAM.Config}}{{println .Subnet}}{{end}}' |
      awk 'NF { cidr = cidr separator $0; separator = "," } END { print cidr }'
  )"
  if [ -z "$TRAEFIK_NETWORK_CIDR" ]; then
    echo "No subnet found for Docker network $TRAEFIK_NETWORK_NAME" >&2
    exit 1
  fi
else
  docker network create \
    --driver bridge \
    --subnet 172.30.0.0/24 \
    "$TRAEFIK_NETWORK_NAME"
  export TRAEFIK_NETWORK_CIDR=172.30.0.0/24
fi

docker compose config >/dev/null
docker compose up -d
```

### 停止服务

```bash
docker compose down
```

### 查看日志

```bash
# 查看所有服务日志
docker compose logs -f

# 查看特定服务日志
docker compose logs -f stargate
```

### 自定义配置

编辑 `docker-compose.yml`，修改环境变量：

```yaml
services:
  stargate:
    image: ghcr.io/soulteary/stargate:1.3.5
    environment:
      - AUTH_HOST=auth.example.com
      - PASSWORDS=bcrypt:$$2a$$10$$...
      - TRUSTED_PROXIES=${TRAEFIK_NETWORK_CIDR:?set TRAEFIK_NETWORK_CIDR from docker network inspect}
      - DEBUG=false
      - LANGUAGE=zh
      - COOKIE_DOMAIN=.example.com
      - SESSION_EXCHANGE_SECRET=replace-with-at-least-32-random-characters
```

## Traefik 集成

### 基本配置

Stargate 设计用于与 Traefik 集成，通过 Forward Auth 中间件提供认证。

#### 1. 配置 Stargate 服务

在 `docker-compose.yml` 中配置 Stargate：

```yaml
services:
  stargate:
    image: ghcr.io/soulteary/stargate:1.3.5
    environment:
      - AUTH_HOST=auth.example.com
      - PASSWORDS=bcrypt:$$2a$$10$$...
      - TRUSTED_PROXIES=${TRAEFIK_NETWORK_CIDR:?set TRAEFIK_NETWORK_CIDR from docker network inspect}
    networks:
      - traefik
    labels:
      - "traefik.enable=true"
      - "traefik.docker.network=${TRAEFIK_NETWORK_NAME:-traefik}"
      - "traefik.http.routers.auth.entrypoints=http,https"
      - "traefik.http.routers.auth.rule=Host(`auth.example.com`) || Path(`/_session_exchange`)"
      - "traefik.http.middlewares.stargate.forwardauth.address=http://stargate:8080/_auth"
      - "traefik.http.middlewares.stargate.forwardauth.authResponseHeaders=X-Forwarded-User,X-Auth-User,X-Auth-Email,X-Auth-Name,X-Auth-Scopes,X-Auth-Role,X-Auth-AMR"
```

#### 2. 配置受保护的服务

在需要认证的服务上应用 Stargate 中间件：

```yaml
services:
  your-app:
    image: your-app:latest
    networks:
      - traefik
    labels:
      - "traefik.enable=true"
      - "traefik.docker.network=${TRAEFIK_NETWORK_NAME:-traefik}"
      - "traefik.http.routers.your-app.entrypoints=http,https"
      - "traefik.http.routers.your-app.rule=Host(`app.example.com`)"
      - "traefik.http.routers.your-app.middlewares=stargate"  # 应用认证中间件
```

### HTTPS 配置

#### 使用 Let's Encrypt

```yaml
services:
  stargate:
    labels:
      - "traefik.http.routers.auth.entrypoints=https"
      - "traefik.http.routers.auth.tls.certresolver=letsencrypt"
      - "traefik.http.routers.auth.tls=true"
```

#### 使用自定义证书

```yaml
services:
  stargate:
    labels:
      - "traefik.http.routers.auth.entrypoints=https"
      - "traefik.http.routers.auth.tls=true"
      - "traefik.http.routers.auth.tls.certfile=/path/to/cert.pem"
      - "traefik.http.routers.auth.tls.keyfile=/path/to/key.pem"
```

### 跨域会话共享

如果需要跨子域名共享会话：

1. 设置 `COOKIE_DOMAIN` 环境变量：

```yaml
services:
  stargate:
    environment:
      - COOKIE_DOMAIN=.example.com
      - SESSION_EXCHANGE_SECRET=replace-with-at-least-32-random-characters
```

2. 确保所有相关域名都通过 Traefik 路由到 Stargate

3. 登录流程：
   - 用户在 `auth.example.com` 登录
   - 重定向到 `app.example.com/_session_exchange?ticket=<opaque_ticket>`
   - 会话 Cookie 被设置到 `.example.com` 域名
   - 所有 `*.example.com` 子域名都可以使用该会话

## 生产环境部署

### 安全建议

#### 1. 使用强密码算法

**不推荐：**

```bash
PASSWORDS='plaintext:yourpassword'
```

**推荐：**

```bash
PASSWORDS='bcrypt:$2a$10$N9qo8uLOickgx2ZMRZoMyeIjZAgcfl7p92ldGxad68LJZdL17lhWy'
```

#### 2. 启用 HTTPS

- 使用 Traefik 配置 HTTPS
- 使用 Let's Encrypt 自动证书
- 强制 HTTPS 重定向

#### 3. 关闭调试模式

```bash
DEBUG=false
```

#### 4. 设置资源限制

```yaml
services:
  stargate:
    deploy:
      resources:
        limits:
          cpus: '0.5'
          memory: 128M
        reservations:
          cpus: '0.25'
          memory: 64M
```

#### 5. 使用健康检查

```yaml
services:
  stargate:
    healthcheck:
      test: ["CMD", "wget", "-q", "-O", "-", "http://127.0.0.1:8080/healthz"]
      interval: 30s
      timeout: 10s
      retries: 3
      start_period: 40s
```

### 高可用部署

#### 1. 多实例部署

```yaml
services:
  stargate:
    deploy:
      replicas: 3
```

**会话存储适用范围：**

- **单进程、单副本（包括跨域回调）：** 只要 Ticket 签发、兑换及后续 Session 使用始终由同一个存活的 Stargate 进程处理，就可以使用进程内存储（`SESSION_STORAGE_ENABLED=false`）。Session 记录与已消费 Ticket 哈希只存在于该进程；进程重启后两者都会丢失，活动 Session 失效，也无法跨重启保留 Ticket 单次消费状态。
- **多进程或多副本：** 只要 Session 或交换 Ticket 可能由不同进程处理，就必须使用 Redis。设置 `SESSION_STORAGE_ENABLED=true`，所有副本使用相同的 `SESSION_STORAGE_REDIS_*` 命名空间和 `SESSION_EXCHANGE_SECRET`。Sticky Session 只能优化路由，不能替代共享状态或跨进程重放防护。
- **跨副本限流：** 限流计数通过同一个 Redis 连接共享。未设置 `SESSION_STORAGE_ENABLED=true` 时，每个副本各自执行配额，N 个副本的实际放行量最多是 `RATE_LIMIT_LOGIN_MAX` 与 `RATE_LIMIT_VERIFICATION_MAX` 的 N 倍。配额按客户端地址统计，因此必须在 `TRUSTED_PROXIES` 中列出反向代理，否则所有客户端都会归属到代理地址并共用一份配额。
- **滚动升级：** 新旧进程会重叠运行；要无中断滚动替换，必须使用 Redis。未使用 Redis 时，应停旧再启新，并接受 Session 失效和用户重新登录。不要在同一服务实例池中混用 v0.12.0 与 v1.0.0。
- **跨进程重启保留状态：** 只要要求 Session 或已消费 Ticket 的重放状态在进程替换或重启后继续有效，就必须使用 Redis。

#### 2. 负载均衡

在 Traefik 前添加负载均衡器：

```yaml
services:
  traefik:
    labels:
      - "traefik.http.services.stargate.loadbalancer.server.port=8080"
```

### 监控配置

#### 1. 日志收集

```yaml
services:
  stargate:
    logging:
      driver: "json-file"
      options:
        max-size: "10m"
        max-file: "3"
```

#### 2. 健康检查端点

使用 `/healthz` 检查进程存活。需要在 Redis、Warden 和 Herald 依赖就绪后才接收流量时，另用 `/readyz`；旧的 `/health` 端点保留为弃用的就绪检查别名。

```bash
# 健康检查脚本
#!/bin/bash
if curl -f http://localhost:8080/healthz > /dev/null 2>&1; then
  exit 0
else
  exit 1
fi
```

#### 3. Prometheus 集成

Stargate 通过 `GET /metrics` 暴露 Prometheus 指标。在 Prometheus 服务中配置抓取该端点即可。该端点默认已从请求日志与追踪中排除。

## 监控和维护

### 日志管理

#### 查看日志

```bash
# Docker
docker logs -f stargate

# Docker Compose
docker compose logs -f stargate
```

#### 日志级别

- `DEBUG=true`：详细调试信息
- `DEBUG=false`：仅关键信息

#### 日志轮转

配置 Docker 日志驱动：

```yaml
services:
  stargate:
    logging:
      driver: "json-file"
      options:
        max-size: "10m"
        max-file: "3"
```

### 性能监控

#### 资源使用

```bash
# 查看容器资源使用
docker stats stargate
```

#### 响应时间

使用健康检查端点监控响应时间：

```bash
time curl http://auth.example.com/healthz
```

### 定期维护

1. **更新镜像**：定期拉取最新镜像
2. **检查日志**：定期检查错误日志
3. **监控资源**：监控 CPU 和内存使用
4. **备份配置**：备份环境变量配置

## 故障排查

### 常见问题

#### 1. 服务无法启动

**问题：** 容器启动后立即退出

**排查步骤：**

```bash
# 查看容器日志
docker logs stargate

# 检查配置
docker inspect stargate | grep -A 20 Env
```

**常见原因：**

- 缺少必需配置（`AUTH_HOST`、`PASSWORDS`）
- 密码配置格式错误
- 端口被占用

#### 2. 认证失败

**问题：** 用户无法登录

**排查步骤：**

1. 检查密码配置是否正确
2. 检查密码算法是否匹配
3. 查看服务日志：`docker logs stargate`

**常见原因：**

- 密码配置错误
- 密码算法不匹配（如配置了 bcrypt 但使用了明文密码）
- Cookie 域名配置错误

#### 3. 跨域会话不工作

**问题：** 跨子域名无法共享会话

**排查步骤：**

1. 检查 `COOKIE_DOMAIN` 配置
2. 确认 Cookie 域名格式正确（`.example.com`）
3. 检查浏览器 Cookie 设置

**解决方案：**

```bash
# 确保设置了 COOKIE_DOMAIN
COOKIE_DOMAIN=.example.com
```

#### 4. Traefik 集成问题

**问题：** Traefik 无法正确转发认证请求

**排查步骤：**

1. 检查 Traefik 标签配置
2. 确认网络配置正确
3. 检查 Forward Auth 中间件地址

**解决方案：**

```yaml
# 确保中间件地址正确
- "traefik.http.middlewares.stargate.forwardauth.address=http://stargate:8080/_auth"
- "traefik.http.middlewares.stargate.forwardauth.authResponseHeaders=X-Forwarded-User,X-Auth-User,X-Auth-Email,X-Auth-Name,X-Auth-Scopes,X-Auth-Role,X-Auth-AMR"
```

#### 5. Warden 服务问题

**问题：** 用户无法登录，提示用户不在白名单中

**排查步骤：**

1. 检查 Warden 服务是否正常运行：`docker logs warden`
2. 检查 `WARDEN_URL` 配置是否正确
3. 检查 `WARDEN_API_KEY` 是否正确
4. 确认用户在 Warden 数据库中已配置

**解决方案：**

```bash
# 检查 Warden 健康状态
curl http://warden:8081/health

# 查询单个用户；仅检查 HTTP 状态，不输出个人信息
curl -fsS -o /dev/null -H "Authorization: Bearer $WARDEN_API_KEY" \
     "http://warden:8081/user?mail=user@example.com"
```

#### 6. Herald 服务问题

**问题：** 收不到验证码或验证码错误

**排查步骤：**

1. 检查 Herald 服务是否正常运行：`docker logs herald`
2. 检查 `HERALD_URL` 配置是否正确
3. 检查 `HERALD_HMAC_SECRET` 或 `HERALD_API_KEY` 是否正确
4. 检查 Redis 连接是否正常
5. 检查 Herald 日志中的错误信息

**常见错误：**

- **401 Unauthorized**：HMAC 签名或 API Key 错误
- **429 Too Many Requests**：触发限流，需要等待
- **Connection Failed**：Herald 服务不可用或网络问题

**解决方案：**

```bash
# 检查 Herald 健康状态
curl http://herald:8082/healthz

# 在实际 Redis 容器内检查连接；认证和 TLS 参数需按部署配置补充
docker exec "${HERALD_REDIS_CONTAINER:?set the Redis container name}" redis-cli ping

# 查看 Herald 日志
docker logs herald | grep -i error
```

#### 7. 服务间通信问题

**问题：** Stargate 无法连接到 Warden 或 Herald

**排查步骤：**

1. 检查服务是否在同一网络中
2. 检查服务名称解析（DNS）
3. 检查防火墙规则
4. 检查服务健康状态

**解决方案：**

```bash
# 从 Stargate 容器内测试连接
docker exec stargate wget -q -O - http://warden:8081/health
docker exec stargate wget -q -O - http://herald:8082/healthz

# 检查网络配置
docker network inspect <network_name>
```

### 调试技巧

#### 1. 启用调试模式

```bash
DEBUG=true
```

#### 2. 检查网络连接

```bash
# 从容器内测试
docker exec stargate wget -q -O - http://127.0.0.1:8080/healthz
```

#### 3. 查看 Traefik 日志

```bash
docker logs traefik
```

#### 4. 测试 API 端点

```bash
# 测试健康检查
curl http://auth.example.com/healthz

# 测试认证（使用 Header）
# 服务端启动前必须设置：PASSWORD_HEADER_AUTH_ENABLED=true
curl -H "Stargate-Password: yourpassword" http://auth.example.com/_auth

# 测试认证（使用 Cookie）
curl -H "Cookie: stargate_session_id=<session_id>" http://auth.example.com/_auth
```

### 获取帮助

如果遇到问题：

1. 查看日志：`docker logs stargate`
2. 检查配置：确认所有环境变量正确
3. 查看文档：[API 文档](API.md)、[配置参考](CONFIG.md)
4. 提交 Issue：在项目仓库提交问题报告

## 升级指南

### 升级步骤

1. **分别生成回滚与 v1 环境文件**：保持现有 `stargate.env` 不变。下面的命令在任一目标文件已存在时会拒绝覆盖，并且只从新的 v1 文件中删除废弃的 Warden OTP 配置。

```bash
set -eu
old_env=./stargate.env
rollback_env=./stargate-v0.12.0.env
v1_env=./stargate-v1.env
old_container=${STARGATE_OLD_CONTAINER:-stargate}

test ! -e "$rollback_env"
test ! -e "$v1_env"
umask 077

# 文件不存在时，安全导出现有容器的环境。
if [ ! -e "$old_env" ]; then
  export_tmp=$(mktemp "${old_env}.tmp.XXXXXX")
  trap 'rm -f "$export_tmp"' 0 1 2 15
  docker inspect --format '{{range .Config.Env}}{{println .}}{{end}}' "$old_container" > "$export_tmp"
  ln "$export_tmp" "$old_env"
  rm -f "$export_tmp"
  trap - 0 1 2 15
fi

test -f "$old_env"
chmod 600 "$old_env"
(set -C; cat "$old_env" > "$rollback_env")
(set -C; awk '
  /^[[:space:]]*WARDEN_OTP_(ENABLED|SECRET_KEY)[[:space:]]*(=|$)/ { next }
  /^[[:space:]]*PORT[[:space:]]*(=|$)/ { print "PORT=8080"; next }
  { print }
' "$old_env" > "$v1_env")
```

Herald TOTP 要求 Stargate 通过 Warden 解析已认证用户，因此还必须配置 `WARDEN_ENABLED=true` 和 `WARDEN_URL`。

只要环境中存在 `WARDEN_OTP_ENABLED` 或 `WARDEN_OTP_SECRET_KEY`（即使为空或为 `false`），v1 都会拒绝启动，请勿把它们加回新文件。需要 TOTP 时，在 `stargate-v1.env` 中为 Stargate 配置 `HERALD_ENABLED=true`、`HERALD_TOTP_ENABLED=true`、`HERALD_URL` 及一种受支持的 Herald 服务鉴权方式；同时在 Herald 配置 TOTP 代理，并为 herald-totp 单独配置 `HERALD_TOTP_ENCRYPTION_KEY`，不要自动复用旧 Warden 密钥。参见[配置说明](CONFIG.md)。

2. **保留旧容器及其原始环境用于回滚**：

```bash
docker stop stargate
docker rename stargate stargate-previous
```

3. **拉取新镜像**：

```bash
docker pull ghcr.io/soulteary/stargate:v1.0.0
```

4. **启动新容器**：

```bash
docker run -d \
  --name stargate \
  --env-file ./stargate-v1.env \
  -p 8080:8080 \
  --restart unless-stopped \
  ghcr.io/soulteary/stargate:v1.0.0
```

5. **验证服务**：

```bash
curl --fail http://127.0.0.1:8080/healthz
curl --fail http://127.0.0.1:8080/readyz
```

### 回滚

如果升级后出现问题：

```bash
# 删除新容器并恢复未修改的旧容器。
# 保留 stargate-v0.12.0.env 作为回滚配置记录。
docker rm -f stargate
docker rename stargate-previous stargate
docker start stargate
```

## 可撤销会话迁移

新版本在规范化后的 `SESSION_STORAGE_REDIS_KEY_PREFIX` 下追加 `revocable:v1:`
保存会话。旧命名空间的会话及其未完成的 SSO 票据需要重新登录，不能直接复制到
新空间。所有网关实例都应升级，旧实例不得向新命名空间写入；旧记录自然过期即可。

新会话 ID 带有 24 小时的绝对截止时间。注销通过原子操作保留撤销标记到该时间，
在途请求不能再把已注销 ID 保存成已认证会话，存储 TTL 也不能延长 ID 截止时间。
这是在原空闲超时之外增加的绝对有效期。保持 Redis 与应用时钟同步，并为专用会话
key 空间允许 `EVAL`、`TIME`、`GET`、`SET`、`DEL`。

TOTP 注册、确认、撤销和 step-up 操作会在执行前重新查询 Warden，即使关闭了后台
授权刷新也会执行。目录或来源故障返回 503，不调用因子操作；确认停用、删除或身份
被重新分配时返回 401。纯密码会话继续使用配置的本地凭据。

限流按注册的方法和路由分桶，大小写与尾斜线变体共享额度，不同注册端点仍相互独立。

### 浏览器绑定的跨域会话交换

跨域登录先访问目标站点的 `/_session_exchange`，由目标站点生成有效期十分钟、仅限当前主机的 HttpOnly、SameSite=Lax 随机 Cookie。加密请求绑定目标来源、配置的 `SESSION_EXCHANGE_REDIRECT_PATH` 和随机值摘要；认证站点签发有效期最多两分钟的一次性票据，目标站点必须先核对自己的 Cookie 才能导入会话。直接在认证站点登录并指定回调时，会自动完成这一步。每个允许的目标站点仍需代理 `/_session_exchange` 路径。

生产环境必须使用 HTTPS 和 `COOKIE_SECURE=true`。随机 Cookie 使用 `__Host-` 前缀且不设置 Domain，目标站点之间不能共享。HTTP 开发环境在 `COOKIE_SECURE=false` 时使用无前缀 Cookie，仅适用于受信任的本地开发网络。参与交换的副本须配置相同的 `SESSION_EXCHANGE_SECRET`，多副本部署须使用共享 Redis 防重放存储。

旧版票据不再接受。请统一升级所有副本后重新完成尚未结束的登录，滚动升级期间不要混用新旧交换处理器。客户端不能指定最终返回地址或随机值，最终地址始终是配置的同源路径。同一浏览器并行登录同一目标站点可能覆盖待处理的随机 Cookie，此时需重新开始其中一次登录。

### Herald 凭据优先级

`HERALD_HMAC_SECRET` 非空时，即使同时配置 `HERALD_API_KEY`，Stargate 也只向 Herald SDK 提供 HMAC 凭据。HMAC 认证失败后不会回退到 API key。Herald 服务端使用 `REQUEST_AUTH_MODE=api_key` 时，应留空 `HERALD_HMAC_SECRET` 并配置 `HERALD_API_KEY`。升级同时配置两类凭据的客户端前，需确认服务端接受对应 HMAC 凭据，或明确选择仅 API key 模式。

### Warden 凭据优先级

同时配置 `WARDEN_API_KEY` 和 `WARDEN_HMAC_KEY_ID` / `WARDEN_HMAC_SECRET` 时，新版 SDK 只发送 HMAC 凭据，不会改用 API key 重试认证。使用该组合升级前，请先在 Warden 服务端配置匹配的 HMAC 凭据。若服务端只启用 API key，请移除客户端的两个 HMAC 配置并保留 `WARDEN_API_KEY`。API-key-only 和 mTLS 仍受支持，服务端认证策略没有改变。


## 登录验证码绑定升级与回滚

OTP 登录必须携带网关保存的可信 challenge 上下文，并匹配 Warden 当前主体、
发送渠道与目录联系方式。网关使用 `/v2/otp/verifications`，声明预期主体、
`login` 用途和渠道，成功后再核对返回的 challenge ID 及这些字段。不回退到
无上下文约束的旧验证接口；切换前先确认 Herald 支持该绑定路由。

所有网关副本必须共享已有的 challenge-context Redis 命名空间。设置
`SESSION_STORAGE_ENABLED=true`，为每个副本配置同一 Redis 数据库和
`SESSION_STORAGE_REDIS_KEY_PREFIX` 的会话 Redis 连接。只有该 Redis client
初始化后，服务才安装 Redis 上下文存储，否则仍使用进程内存。内存上下文仅
适用于单进程，重启或其他副本无法验证自己未保存的绑定。旧 pending 记录的
格式无需转换，但必须保留原绑定且目录联系方式未变；缺少绑定或联系方式
变化时需要重新发起登录。邮箱本地部分和 plus tag 保持精确身份，域名大小写
及支持的手机号格式可规范化，但不将不同联系方式视为等同。

若 Herald 已发送验证码而网关保存绑定失败，创建返回
`503/binding_unavailable`，不返回 challenge ID。恢复后使用原幂等键重试，
不要改用新键绕过不确定的发送状态。已存绑定冲突同样拒绝请求。

替换所有网关前暂停新发送、排空登录请求。旧网关仍允许旧验证路径，混合部署
会保留旧行为；回滚也会重新引入该风险，不应靠回滚接受用途不符的验证码。
升级和回滚都保留 provider 幂等、nonce 与审计状态，并遵守后端各自的数据
迁移规则。

TOTP 注册确认限流经 Herald 和 Stargate 保留为 429 及固定错误字段，网关保留
注册绑定供稍后重试。升级 SDK 不等于部署新的 TOTP 服务；subject 预算与注册
开始共用，应覆盖两个步骤和错误尝试。每小时仅一次的配额无法在同一窗口内
完成注册。session-kit 的时钟修复只放宽未来期限合理性上界，不延长绝对期限、
不允许撤销后写回。继续同步时钟；回滚旧 adapter 可能拒绝时钟略快副本签发的
会话，应保留可撤销命名空间及期限检查，不把记录复制到旧命名空间。

## OTP 验证错误处理（v1.3.2）

OTP 登录将 Herald 服务端错误保留为 503，将明确的
`409/lock_contention` 保留为 409。无效凭据和 challenge 绑定不匹配仍被拒绝。
本次修正的是已有的错误分类缺口，不改变证明校验。现有 SDK
不暴露上游重试头，网关不会编造 `Retry-After` 或自动重试验证码验证。

从 v1.3.1 升级时保留 Redis 数据集、命名空间、待验证绑定、会话和撤销标记。
不转换记录、不要求重新登录，也不延长原有截止时间。混合副本可以读取同一记录，
但旧副本仍可能返回 401；客户端须兼容滚动切换期间的状态差异。

只有确认失败发生在消费前时，恢复后才可以沿用原 challenge 重试。失败响应本身
不保证验证码未被消费，应保留状态并核对不确定结果，不盲目重新发送或重启流程。
按 TOTP 和 Herald 各自指南升级，并保留后端审计、防重放和幂等记录。

回滚网关会恢复旧的 401 分类；保留相同记录与期限，不要顺带独立回滚后端服务。
从 v1.3.1 之前版本升级时，仍须执行对应的会话与绑定迁移要求。

## Herald 凭据选择 (v1.3.3)

`HERALD_HMAC_SECRET` 非空时，Stargate 选择 HMAC，签名请求不携带
`HERALD_API_KEY`。升级使用 key map 的 Herald 验证端前，先为全部现有 Stargate 调用方设置
并验证服务端当前 key ID 对应的 `HERALD_HMAC_KEY_ID`。再升级 Herald v1.4.3，
最后替换 Stargate v1.3.3 二进制。
服务端使用 `REQUEST_AUTH_MODE=api_key` 时，客户端应留空 HMAC secret 并保留
API key。HMAC 请求失败后不会换用 API key 重试。先部署并验证 Herald v1.4.3，
再升级 Stargate v1.3.3；更新 SDK 依赖不会更新独立部署的 Herald 服务。

保留共享 Redis 数据、待验证 OTP 绑定、重放标记以及已有会话命名空间和截止时间。
从 v1.3.2 升级无需记录格式迁移、强制重新认证或会话续期。网关不会自动重试
OTP 验证，503 及特定锁争用 409 的含义与 v1.3.2 一致。

回滚前核对认证模式和有效密钥。旧网关可能重新发送同时配置的两类凭据；回滚
不会恢复已撤销的密钥，也不会删除配置中的 secret。旧 Herald 服务会恢复原有
key-map 选择规则，且可能不支持新的代理 key-ID 配置。应遵循其回滚说明，保留
状态，并在切换流量前验证相同的登录与 TOTP 流程。从更旧版本升级时仍须遵循
v1.3.0/v1.3.1 的迁移要求。

## 依赖补丁升级与回滚（v1.3.4）

先部署已验证的 Warden v1.6.2 和 Herald v1.4.4 服务，并按 Herald 文档完成
herald-totp v2.0.3 的升级，再升级 Stargate v1.3.4。更新网关 SDK 模块不会部署这些
独立服务。既有 HMAC key ID、凭据优先级及不回退 API Key 的行为不变；较旧部署仍需
遵循 v1.3.3 的分阶段 key ID 配置要求。

secure-kit/v2 v2.1.4 按 Unicode 码点处理日志脱敏，并安全处理超大脱敏边界。
非 ASCII 脱敏输出可能变化，既有 ASCII 输出保持不变。依赖精确脱敏字符串的日志消费端
应确认兼容性。此次更新不改变登录证明或会话记录格式。

混合版本升级期间，保留 Redis 记录、待验证 challenge 绑定、重放与注销标记、会话命名空间
和绝对截止时间。本补丁不会续期会话或自动重试验证码。既有 503 和锁竞争 409 语义不变。

回滚至 v1.3.3 前，确认当前凭据和提供方版本。已验证的旧二进制或镜像使用其编译时的依赖与脱敏行为。源码或 SDK 回滚应恢复升级前应用的
`go.mod` 和 `go.sum`，检查 `go list -m all` 的实际版本后重新构建；仅降级父模块可能因 Go
最小版本选择而保留较新的直接依赖或传递依赖下限。回滚不会重写既有日志、恢复已退役
密钥或撤销已提交的验证码消费。保留共享状态，对不确定的提供方结果
进行核对，不应独立降级提供方服务或删除重放记录。

## 认证优先级升级与回滚（v1.3.5）

从 v1.3.4 升级至 v1.3.5 时，替换网关二进制或镜像，保留原有配置与服务版本。
本次更新不改变依赖、Redis 结构、会话命名空间或绝对截止时间。升级和回滚期间，
保留待验证 challenge 绑定、撤销与重放记录，不要清空 Redis。

有效密码头或可信身份头的认证不再受无关旧 Warden 会话刷新影响。实际选中会话认证时，
仍刷新目录授权并检查来源有效期，目录不可用时拒绝放行。错误密码头仍可按既有行为回退到
已认证会话，该会话仍须通过上述检查。这与服务 HMAC 认证不同：HMAC 请求不会改用 API key
重试。无状态凭据不能借用会话的 step-up 验证标记。本次更新不增加自动验证码重试或会话续期。

回滚至已验证的 v1.3.4 二进制或镜像保留既有状态，但恢复旧认证顺序：旧 Cookie 的目录刷新
可能再次阻断其他有效显式凭据。源码构建应检出 v1.3.4 源码
（`19d2997ff0bb6c61ff8a302cf4b99287c8b27008`）后重新构建。本次依赖未变，仅恢复
`go.mod` 与 `go.sum` 不能撤销认证源码修改。重新构建前仍须核对实际选中的模块版本。上文 v1.3.4 依赖与更早版本的迁移要求仍然适用。
