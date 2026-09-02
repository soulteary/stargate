# 部署指南

本文档提供 Stargate Forward Auth Service 的详细部署指南。

从 v0.12.0 升级时，请先阅读 [v1.0.0 迁移指南](MIGRATION_V1.md)，再切换生产流量。

## 目录

- [部署方式](#部署方式)
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

## 服务依赖

Stargate 可以与以下可选服务集成：

### Warden 服务

**功能：** 用户白名单管理和用户信息提供

**部署要求：**
- 需要数据库（PostgreSQL/MySQL/SQLite）
- 提供 HTTP API 接口
- 支持 API Key 认证

**配置：**
```bash
WARDEN_ENABLED=true
WARDEN_URL=http://warden:8080
WARDEN_API_KEY=your-api-key
```

### Herald 服务

**功能：** OTP/验证码发送和验证

**部署要求：**
- 需要 Redis（存储 challenge 和限流状态）
- 提供 HTTP API 接口
- 支持 HMAC 签名或 mTLS 认证（生产环境推荐）

**配置：**
```bash
HERALD_ENABLED=true
HERALD_URL=http://herald:8080
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

- **基础镜像**：`golang:1.27.0-alpine3.24`（构建阶段）
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
docker compose config
docker compose up -d
```

### 接入已有的外部 Traefik 网络

仅当 Traefik 已运行在共享的外部 Docker 网络时，才使用下面这套独立示例。Compose 网络、每个 `traefik.docker.network` label 和 `TRUSTED_PROXIES` 都由实际网络名称和 inspect 得到的 CIDR 驱动。

```yaml
services:
  stargate:
    image: ghcr.io/soulteary/stargate:v1.0.0
    environment:
      - AUTH_HOST=auth.test.localhost
      - PASSWORDS=plaintext:test1234|test1337
      - CALLBACK_ALLOWED_HOSTS=whoami.test.localhost
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

  whoami:
    image: traefik/whoami
    networks:
      - traefik
    labels:
      - traefik.enable=true
      - traefik.docker.network=${TRAEFIK_NETWORK_NAME:-traefik}
      - traefik.http.routers.whoami.entrypoints=http
      - traefik.http.routers.whoami.rule=Host(`whoami.test.localhost`)
      - "traefik.http.routers.whoami.middlewares=stargate"

networks:
  traefik:
    name: ${TRAEFIK_NETWORK_NAME:-traefik}
    external: true
```

### 完整配置（Warden + Herald OTP 认证）

生产环境推荐使用 Warden + Herald 的完整配置：

```yaml
services:
  # Stargate 认证服务
  stargate:
    image: ghcr.io/soulteary/stargate:v1.0.0
    environment:
      - AUTH_HOST=auth.example.com
      - WARDEN_ENABLED=true
      - WARDEN_URL=http://warden:8080
      - WARDEN_API_KEY=your-warden-api-key
      - WARDEN_CACHE_TTL=300
      - HERALD_ENABLED=true
      - HERALD_URL=http://herald:8080
      - HERALD_HMAC_SECRET=your-herald-hmac-secret
      - DEBUG=false
      - LANGUAGE=zh
      - COOKIE_DOMAIN=.example.com
      - SESSION_EXCHANGE_SECRET=replace-with-at-least-32-random-characters
      - TRUSTED_PROXIES=${TRAEFIK_NETWORK_CIDR:?set TRAEFIK_NETWORK_CIDR from docker network inspect}
    networks:
      - traefik
      - internal
    depends_on:
      - warden
      - herald
    labels:
      - traefik.enable=true
      - traefik.docker.network=${TRAEFIK_NETWORK_NAME:-traefik}
      - traefik.http.routers.auth.entrypoints=http,https
      - traefik.http.routers.auth.rule=Host(`auth.example.com`) || Path(`/_session_exchange`)
      - traefik.http.middlewares.stargate.forwardauth.address=http://stargate:8080/_auth
      - "traefik.http.middlewares.stargate.forwardauth.authResponseHeaders=X-Forwarded-User,X-Auth-User,X-Auth-Email,X-Auth-Name,X-Auth-Scopes,X-Auth-Role,X-Auth-AMR"

  # Warden 用户白名单服务
  warden:
    image: warden:latest
    environment:
      - WARDEN_DB_URL=postgres://warden:password@postgres:5432/warden
      - WARDEN_API_KEY=your-warden-api-key
    networks:
      - internal
    depends_on:
      - postgres

  # Herald OTP/验证码服务
  herald:
    image: herald:latest
    environment:
      - HERALD_REDIS_URL=redis://redis:6379/0
      - HERALD_HMAC_SECRET=your-herald-hmac-secret
      - HERALD_EMAIL_API_URL=http://email-service:8080/v1/send
      - HERALD_SMS_API_URL=http://sms-service:8080/v1/send
    networks:
      - internal
    depends_on:
      - redis

  # PostgreSQL 数据库（Warden 使用）
  postgres:
    image: postgres:15-alpine
    environment:
      - POSTGRES_DB=warden
      - POSTGRES_USER=warden
      - POSTGRES_PASSWORD=password
    volumes:
      - postgres_data:/var/lib/postgresql/data
    networks:
      - internal

  # Redis（Herald 使用）
  redis:
    image: redis:7-alpine
    volumes:
      - redis_data:/data
    networks:
      - internal

  # 示例受保护服务
  your-app:
    image: your-app:latest
    networks:
      - traefik
    labels:
      - traefik.enable=true
      - traefik.docker.network=${TRAEFIK_NETWORK_NAME:-traefik}
      - traefik.http.routers.your-app.entrypoints=http,https
      - traefik.http.routers.your-app.rule=Host(`app.example.com`)
      - traefik.http.routers.your-app.middlewares=stargate

networks:
  traefik:
    name: ${TRAEFIK_NETWORK_NAME:-traefik}
    external: true
  internal:
    internal: true  # 内部网络，不暴露到外部

volumes:
  postgres_data:
  redis_data:
```

**说明：**

- `internal` 网络用于服务间通信，不暴露到外部
- `traefik` 网络用于与 Traefik 和外部服务通信
- 所有服务间通信都在内网进行，提高安全性
- 生产环境建议使用环境变量文件（`.env`）管理敏感配置

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

docker compose config
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
    image: ghcr.io/soulteary/stargate:v1.0.0
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
    image: ghcr.io/soulteary/stargate:v1.0.0
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
curl http://warden:8080/health

# 检查用户是否在白名单中
curl -H "Authorization: Bearer $WARDEN_API_KEY" \
     http://warden:8080/v1/users?mail=user@example.com
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
curl http://herald:8080/healthz

# 检查 Redis 连接
docker exec herald redis-cli ping

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
docker exec stargate wget -q -O - http://warden:8080/health
docker exec stargate wget -q -O - http://herald:8080/healthz

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
