# 部署指南

本文档提供 Stargate Forward Auth Service 的详细部署指南。

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

## Docker 部署

### 构建镜像

#### 从源码构建

```bash
cd codes
docker build -t stargate:latest .
```

#### 构建参数

- **基础镜像**：`golang:1.20.0-alpine3.17`（构建阶段）
- **运行镜像**：`scratch`（最小化镜像）
- **工作目录**：`/app`
- **暴露端口**：`80`

### 运行容器

#### 基础运行

```bash
docker run -d \
  --name stargate \
  -p 80:80 \
  -e AUTH_HOST=auth.example.com \
  -e PASSWORDS=plaintext:yourpassword \
  stargate:latest
```

#### 完整配置运行

```bash
docker run -d \
  --name stargate \
  -p 80:80 \
  -e AUTH_HOST=auth.example.com \
  -e PASSWORDS=bcrypt:$2a$10$N9qo8uLOickgx2ZMRZoMyeIjZAgcfl7p92ldGxad68LJZdL17lhWy \
  -e DEBUG=false \
  -e LANGUAGE=zh \
  -e LOGIN_PAGE_TITLE=我的认证服务 \
  -e LOGIN_PAGE_FOOTER_TEXT=© 2024 我的公司 \
  -e COOKIE_DOMAIN=.example.com \
  --restart unless-stopped \
  stargate:latest
```

#### 参数说明

- `-d`：后台运行
- `--name stargate`：容器名称
- `-p 80:80`：端口映射（主机端口:容器端口）
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

# 停止并删除
docker rm -f stargate
```

## Docker Compose 部署

### 基础配置

项目提供了 `docker-compose.yml` 示例文件：

```yaml
services:
  stargate:
    image: stargate
    environment:
      - AUTH_HOST=auth.test.localhost
      - PASSWORDS=plaintext:test1234|test1337
    networks:
      - traefik
    labels:
      - traefik.enable=true
      - traefik.docker.network=proxy
      - traefik.http.routers.auth.entrypoints=http
      - traefik.http.routers.auth.rule=Host(`auth.test.localhost`) || Path(`/_session_exchange`)
      - traefik.http.middlewares.stargate.forwardauth.address=http://stargate/_auth

  whoami:
    image: traefik/whoami
    networks:
      - traefik
    labels:
      - traefik.enable=true
      - traefik.docker.network=proxy
      - traefik.http.routers.whoami.entrypoints=http
      - traefik.http.routers.whoami.rule=Host(`whoami.test.localhost`)
      - "traefik.http.routers.whoami.middlewares=stargate"

networks:
  traefik:
    external: true
```

### 启动服务

```bash
cd codes
docker-compose up -d
```

### 停止服务

```bash
docker-compose down
```

### 查看日志

```bash
# 查看所有服务日志
docker-compose logs -f

# 查看特定服务日志
docker-compose logs -f stargate
```

### 自定义配置

编辑 `docker-compose.yml`，修改环境变量：

```yaml
services:
  stargate:
    image: stargate
    environment:
      - AUTH_HOST=auth.example.com
      - PASSWORDS=bcrypt:$2a$10$...
      - DEBUG=false
      - LANGUAGE=zh
      - COOKIE_DOMAIN=.example.com
```

## Traefik 集成

### 基本配置

Stargate 设计用于与 Traefik 集成，通过 Forward Auth 中间件提供认证。

#### 1. 配置 Stargate 服务

在 `docker-compose.yml` 中配置 Stargate：

```yaml
services:
  stargate:
    image: stargate:latest
    environment:
      - AUTH_HOST=auth.example.com
      - PASSWORDS=bcrypt:$2a$10$...
    networks:
      - traefik
    labels:
      - "traefik.enable=true"
      - "traefik.docker.network=traefik"
      - "traefik.http.routers.auth.entrypoints=http,https"
      - "traefik.http.routers.auth.rule=Host(`auth.example.com`) || Path(`/_session_exchange`)"
      - "traefik.http.middlewares.stargate.forwardauth.address=http://stargate/_auth"
      - "traefik.http.middlewares.stargate.forwardauth.authResponseHeaders=X-Forwarded-User"
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
      - "traefik.docker.network=traefik"
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
```

2. 确保所有相关域名都通过 Traefik 路由到 Stargate

3. 登录流程：
   - 用户在 `auth.example.com` 登录
   - 重定向到 `app.example.com/_session_exchange?id=<session_id>`
   - 会话 Cookie 被设置到 `.example.com` 域名
   - 所有 `*.example.com` 子域名都可以使用该会话

## 生产环境部署

### 安全建议

#### 1. 使用强密码算法

**不推荐：**

```bash
PASSWORDS=plaintext:yourpassword
```

**推荐：**

```bash
PASSWORDS=bcrypt:$2a$10$N9qo8uLOickgx2ZMRZoMyeIjZAgcfl7p92ldGxad68LJZdL17lhWy
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
      test: ["CMD", "wget", "--quiet", "--tries=1", "--spider", "http://localhost/health"]
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

**注意：** Stargate 使用内存会话存储，多实例间不会共享会话。如果需要多实例部署，建议：

- 使用负载均衡器的会话保持（Sticky Session）
- 或等待支持外部会话存储（Redis）功能

#### 2. 负载均衡

在 Traefik 前添加负载均衡器：

```yaml
services:
  traefik:
    labels:
      - "traefik.http.services.stargate.loadbalancer.server.port=80"
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

使用 `/health` 端点进行监控：

```bash
# 健康检查脚本
#!/bin/bash
if curl -f http://localhost/health > /dev/null 2>&1; then
  exit 0
else
  exit 1
fi
```

#### 3. Prometheus 集成

（待实现）未来版本将支持 Prometheus 指标导出。

## 监控和维护

### 日志管理

#### 查看日志

```bash
# Docker
docker logs -f stargate

# Docker Compose
docker-compose logs -f stargate
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
time curl http://auth.example.com/health
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
- "traefik.http.middlewares.stargate.forwardauth.address=http://stargate/_auth"
```

### 调试技巧

#### 1. 启用调试模式

```bash
DEBUG=true
```

#### 2. 检查网络连接

```bash
# 从容器内测试
docker exec stargate wget -O- http://localhost/health
```

#### 3. 查看 Traefik 日志

```bash
docker logs traefik
```

#### 4. 测试 API 端点

```bash
# 测试健康检查
curl http://auth.example.com/health

# 测试认证（使用 Header）
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

1. **备份配置**：保存当前环境变量配置

2. **停止服务**：

```bash
docker stop stargate
```

3. **拉取新镜像**：

```bash
docker pull stargate:latest
```

4. **启动新容器**：

```bash
docker run -d \
  --name stargate \
  ...（使用备份的配置）
  stargate:latest
```

5. **验证服务**：

```bash
curl http://auth.example.com/health
```

### 回滚

如果升级后出现问题：

```bash
# 停止新容器
docker stop stargate

# 使用旧镜像启动
docker run -d \
  --name stargate \
  ...（使用备份的配置）
  stargate:<old-version>
```
