# 邮件验证码与验证器完整示例

本示例从当前源码目录构建 Stargate，连接正式发布的 Warden 1.6.2、Herald 1.5.0 和
herald-totp 2.0.3，使用真实 SMTP 投递、持久 Redis、显式 HMAC key ID 及三条独立服务凭据。
不提供读取验证码的测试接口。仅需密码演示时，请使用[根目录快速开始](../../README.zhCN.md#-快速开始)。

## 本地邮件 OTP

需要 Python 3、OpenSSL、Unix 兼容环境中的 Docker Compose v2（Windows 使用 WSL2），以及空闲的 loopback 8088、8025 端口。
浏览器须将两个 `*.test.localhost` 域名解析到本机；若不能自动解析，在 hosts 文件加入：
`127.0.0.1 auth.test.localhost hello.test.localhost`。

```sh
cd examples/full-stack
python3 configure.py --mode mail
docker compose config --quiet
docker compose up -d --build
```

打开 **http://hello.test.localhost:8088**，输入 `demo@example.test` 并发送邮件验证码，
再打开 **http://127.0.0.1:8025** 的 Mailpit 收件箱查看邮件。输入实际投递的验证码后，
确认浏览器返回受保护的演示服务。本地覆盖文件明确设置 Herald 为 development，同时保留 HMAC、strict 投递及
`HERALD_TEST_MODE=false`。邮件经过 Herald 的真实 SMTP 提供方，但由本地 Mailpit
捕获，并非投递到外部邮箱。本例没有配置短信提供方。

认证页为 `http://auth.test.localhost:8088/_login`。访问 **http://auth.test.localhost:8088/_account**，点击登出按钮，发起同源
`POST /_logout`。再访问受保护服务，确认需要重新认证。不要清空 Redis 或删除卷。
此账号页来自源码构建，不在旧版 v1.3.5 镜像中。

初次生成前可指定其他域名和端口：

```sh
python3 configure.py --port 9090 --mailbox-port 9025 \
  --auth-host auth.demo.localhost --app-host app.demo.localhost
```

生成器分别配置仅含 DNS 名的代理规则和包含实际端口的回调地址。访问
`http://app.demo.localhost:9090` 和 `http://127.0.0.1:9025`。
80/443 使用不带端口后缀的规范地址。默认网络冲突时，用 `--subnet` 指定空闲的私有 /24。
共享 Docker 环境必须选择独立 `--project`，不要复用其他部署的项目或网络。

### macOS 共享虚拟机的绑定挂载

非 root 服务必须能够读取指定的公开目录文件。某些共享虚拟机文件系统即使看到
`users.json` 为 0644，也会因私有宿主机父目录（例如 Documents）拒绝访问。应检查实际
容器错误，不能把超时当作权限原因的证明。

不要放宽 `.env`、证书私钥或私有父目录权限。**仅针对合成本地演示**，维护者可以只将
`state/users.json`、`state/routes.yml` 和公开的 `state/provider-ca.pem` 放到 Documents 之外自行创建的临时目录，将此公开
夹具父目录设为 0755、这些公开文件设为 0644，并用明确的 `STATE_DIR` 覆盖指向该目录。
原私有 `.env` 和全部凭据保持不变。不得将真实用户资料、`.env` 或私钥复制到公开夹具目录。
须验证实际挂载文件可读后，才能认定启动成功，并如实记录这是本地文件系统适配，而非生产默认值。
停止所属测试栈后只清理自己创建的夹具目录。生产应使用满足访问控制要求的宿主机或挂载方式。

### 本地 SMTP TLS 与既有配置

即使网页使用本地 HTTP，Herald 也要求经过验证的 SMTP STARTTLS。初始化会生成私有本地 CA
和 SAN 为 `mailpit` 的服务端证书，Mailpit 强制 STARTTLS，仅将公开 CA 信任包通过
`SSL_CERT_FILE` 挂载给 Herald，不挂载 CA 私钥，也不修改宿主机信任库。
`mail`、`totp` 两种组合都需要 OpenSSL；独立的根目录密码演示不需要证书。

较早的本地配置缺少这些文件时，保留原 `.env`、Redis、账号 ID 和全部应用/HMAC/TOTP 密钥。
通过 `certificates.py --output` 在其私有状态目录下新建 `local-smtp` 证书目录，将其中公开
`ca.pem` 复制为 `STATE_DIR/provider-ca.pem`（0644），再向既有私有 `.env` 添加绝对路径
`SMTP_TLS_DIR`、宿主机数字 UID/GID 对应的 `FIXTURE_UID`/`FIXTURE_GID`，以及
`HERALD_CA_FILE="/certs/provider-ca.pem"`。重新创建 Herald、Mailpit 以加载信任。
不要另建应用配置或轮换应用凭据。

本地 SMTP 证书有效期为 365 天。过期、残缺或证书/私钥不匹配时拒绝覆盖。
需要续期时，仅私密备份原 `local-smtp` 证书目录，在原路径显式生成新证书，更新公开 CA
信任包，再重新创建 Herald、Mailpit。账号、Redis、HMAC 和 TOTP 加密密钥保持不变。
这仅是本地测试证书流程，不代表生产 CA 管理方案。

## 添加验证器

新建配置时使用 `python3 configure.py --mode totp`，即可同时启用 TOTP 服务及两层代理开关。
先通过邮件 OTP 登录，再访问 `http://auth.test.localhost:8088/totp/enroll`，点击开始绑定，
扫描二维码或在验证器中手动输入设置密钥，使用动态码确认。离开前妥善保存页面显示的备用码。

登出后测试验证器登录，再用明确的恢复模式测试一个备用码。已使用的备用码在新的登录
challenge 中仍应失败。新版恢复表单来自本源码目录，不在旧版 v1.3.5 镜像中。
没有全局共享 TOTP 密钥；凭据绑定到稳定的 `demo-user-001` 账号 ID。修改邮箱不能顺便创建新 ID。

## 配置与持久化

生成器创建权限为 0600 的 `.env` 及私有 `state/` 目录，不打印秘密，也不覆盖既有配置。
重复运行保留所有密钥及账号 ID，**不会应用新的命令行参数**。发现残缺配置时拒绝继续，留待检查。
不要为了改端口或启用 TOTP 而删除既有配置：先备份，再协调修改非秘密参数，保留全部密钥。
已有**本地**邮件配置启用 TOTP 时设置 `COMPOSE_PROFILES=totp`、`TOTP_ENABLED=true`，其余密钥保持不变。
生产还需要下文的 TLS 扩展和路由文件，仅修改这两个开关并不足够。

生成的用户目录不含密码或验证码，允许非 root Warden 容器读取，宿主机访问由私有父目录限制。
Redis AOF 和 Herald 审计暂存使用具名卷。重启或 `docker compose down` 不删除这些卷，
**需要保留状态的部署不要使用 `down -v`**。保留本源码目录，`COMPOSE_FILE` 使用其绝对路径。Redis 与 `.env` 一同备份；丢失 TOTP 加密密钥后，
已有凭据将无法解密。升级和回滚保留账号 ID、前缀、重放状态及会话绝对截止时间。
不要提交 `.env`，不要将不带 `--quiet` 的 `docker compose config` 输出写入日志，也不要分享含验证码或 Cookie 的浏览器记录。

三条 HMAC 链路分别为 Stargate→Warden（`stargate-directory`）、Stargate→Herald
（`stargate-otp`）、Herald→TOTP（`herald-totp`）。API key 留空，HMAC 失败不回退。
限流、单次消费和严格投递错误处理均保留。目录每 15 秒刷新，快照 60 秒过期；Stargate 来源
最大年龄为 90 秒，时钟容差 1 秒，仍需同步时钟。

代理使用生成的文件配置，不挂载 Docker socket。全部服务标记 `traefik.enable=false`，
避免被宿主机上其他采用 Docker 自动发现的代理接管。本地只映射 loopback 端口；
Redis、Warden、Herald、TOTP 不发布宿主机端口。

## 生产模板

生产覆盖文件是单台可信宿主机的起点，不负责签发证书或配置邮件提供方。需要真实 DNS、
覆盖两个域名的 TLS 证书，以及经过认证的 SMTP。生产模式不启动 Mailpit，也不发布后端端口。
其他内部 HTTP 仅用于私有 Compose 网络。Herald 的生产 TOTP 客户端即使在此网络中也要求
验证 HTTPS；TOTP 生产扩展提供内部 TLS 代理，将私有 CA 仅加入 Herald 容器信任。
跨主机或不可信网络须先为其余服务链路配置 TLS/mTLS。
mTLS 不能替代 Herald/TOTP 的请求认证。

1. 在受保护目录准备 `cert.pem`、`key.pem`，将 `tls.yml.example` 复制为 `tls.yml`。
   私钥不得进入 Git，宿主机权限限制为 0600。TOTP 还需 `totp-cert.pem`（SAN 含
   `DNS:totp-tls`）、`totp-key.pem` 及签发 CA 的 `totp-ca.pem`。CA 私钥离线保管，不挂载。
   TOTP 证书须与私钥配对，证书链须由该 CA 验证。
2. 在 Shell 环境设置 `SMTP_HOST`、`SMTP_PORT`、`SMTP_FROM`、`SMTP_USER`、`SMTP_PASSWORD`
   、`TLS_DIR` 和 `STARGATE_IMAGE`。`STARGATE_IMAGE` 必须是已验证、包含账号与恢复界面的
   网关镜像 `@sha256:...` digest，不能只填写标签。不要在命令行参数或 Shell 历史中写密码。此模板要求认证 SMTP；使用其他
   已批准中继认证方式时，应明确调整模板。
3. 从本示例目录初始化新的输出目录：

   ```sh
   python3 configure.py --environment production --mode totp \
     --auth-host auth.example.com --app-host app.example.com \
     --email operator@example.com --user-id account-001 \
     --output /protected/path/stargate-config
   cd /protected/path/stargate-config
   docker compose config --quiet
   docker compose up -d
   ```

生产覆盖文件不构建 Stargate。请从已审查源码单独构建和验证网关镜像，推送到已批准的镜像
仓库后填写不可变 digest。不要替换为缺少新版账号与恢复界面的旧 v1.3.5 镜像。

生产 TOTP 配置还会选择 `compose.totp-production.yml`。Herald 通过
`https://totp-tls:8443` 调用内部代理，原始 HMAC 签名请求保持不变，代理再转发至私有 TOTP
服务。代理不发布宿主机端口，`SSL_CERT_FILE` 仅作用于 Herald 容器，不修改宿主机信任库，
不跳过 TLS 验证。生产 SMTP 使用私有 CA 时，初次配置提供 `SMTP_CA_FILE`；生成器会将
该公开 CA 与 TOTP 的签发 CA 合并到 `state/provider-ca.pem`。未指定私有 SMTP CA 时，
生产邮件使用镜像原有的公开根证书。启动后必须通过 Herald 执行一次带认证的 TOTP 操作；进程健康或 Compose
渲染通过不能证明这条链路可用。

生成器在写入前校验证书/私钥配对、解析 CA 及检查必填项，不代表已验证证书域名覆盖、SMTP 可投递性、备份质量或网络策略。
这些要求须另行验收。生产 `COOKIE_SECURE=true`，代理通过 443 端口提供 TLS。
打开 `https://app.example.com`，先验证受控邮箱的真实投递，再添加其他账号。
将 `hello` 演示替换为实际应用时，保留 ForwardAuth 中间件。

### 既有提供方信任包迁移

旧配置缺少 `state/provider-ca.pem` 时会明确拒绝；初始化不会静默补文件或替换凭据。
仅使用公开 SMTP 证书且未启用 TOTP 的生产邮件部署，可创建权限 0644 的空信任包，保持
`HERALD_CA_FILE=""`，继续使用原有公开根证书。私有 SMTP 签发方只将公开 CA 证书放入该文件。
启用 TOTP 时还应加入其公开签发 CA，保留已有 SMTP CA，并设置
`HERALD_CA_FILE="/certs/provider-ca.pem"`。信任包中不得包含 CA 私钥。修改后重新创建 Herald，
保留全部应用密钥和 Redis。生成的路径使用 JSON 双引号，字面量美元符以 `$$` 转义；
手动编辑时填写绝对路径，不写 Shell 表达式。

## 为已有生产邮件部署启用 TOTP

不要期待重复运行生成器会修改既有部署，也不要重新生成秘密。先备份既有 `.env` 和 Redis。

1. 在原 `TLS_DIR` 中准备上文要求的 TOTP 证书（SAN 含 `DNS:totp-tls`）、配对的
   `totp-key.pem`（宿主机权限 0600）及签发 CA 的 `totp-ca.pem`。
2. 将本源码目录的 `totp-tls.yml.example` 复制为原 `STATE_DIR/totp-tls.yml`，将此非秘密
   路由文件设为代理可读（0644）。将公开 TOTP 签发 CA 加入 `STATE_DIR/provider-ca.pem`，
   保留既有 SMTP CA，并在私有 `.env` 设置 `HERALD_CA_FILE="/certs/provider-ca.pem"`。不要复制或更改用户目录、加密密钥或 HMAC 凭据。
3. 编辑既有私有 `.env`：设置 `COMPOSE_PROFILES="totp"` 和 `TOTP_ENABLED="true"`；
   在 `COMPOSE_FILE` 的基础文件与生产覆盖文件后，追加本源码目录中
   `compose.totp-production.yml` 的绝对路径，使用当前平台的路径分隔符（Unix 为 `:`）。
   保留原 `TLS_DIR`、`TOTP_ENCRYPTION_KEY`、全部 HMAC secret/key ID、Redis 密码、
   会话交换秘密、前缀和账号 ID。不要另建配置或用新初始化文件覆盖此文件。
4. 执行 `docker compose config --quiet`，重启所选服务，保持证书验证开启，通过 Herald
   验证一次带认证的 TOTP 操作，再完成浏览器绑定。仅启动健康不能证明新的 TLS 与请求认证链路。

此过程同时选择 TLS 代理、CA 挂载及功能开关。回滚或停用 TOTP 时仍保留加密密钥、凭据、
重放状态和原账号 ID；不要清空 Redis 或重新生成配置。

## 停止与排障

使用 `docker compose ps` 和 `docker compose logs --tail=50 SERVICE` 检查启动问题，日志按敏感数据处理。
进程健康不等于能登录。依次检查 DNS/端口、TLS 与 Cookie、准确的回调地址、Warden 稳定 ID/来源有效期、
三条 HMAC key ID 配对、Redis、真实投递。不要通过关闭校验、切换 soft 投递或重置限流来让示例通过。
参阅[部署指南](../../docs/zhCN/DEPLOYMENT.md)。

## 源码身份与正式版本

本地使用 `docker/Dockerfile` 构建当前源码，必须带 `--build`，更新源码后尤其如此。
构建前记录 `git rev-parse HEAD` 和 `git status --short`，构建后用 `docker compose images`
记录镜像 ID。有未提交修改时必须如实记录，不能将源码构建称为正式产物。
三个后端镜像仍固定到正式版本及不可变 digest。

旧版正式 Stargate v1.3.5 不含新版账号与恢复界面，本地流程使用源码构建。在合适的正式版本
发布前，生产模板要求提供自行验证的不可变网关镜像。本例不会发布镜像或自动替用户选择新版本。

### 构建上下文边界

仓库 `.dockerignore` 仅允许 Go 源码、明确的运行时网页资源和两个已打包 Linux 二进制。
Git 凭据、示例配置、`.env*`、审计暂存和私钥/证书文件不进入构建上下文。新增自定义运行时
资源须显式允许并验证，不能删除私有文件排除规则。CI 通过实际 scratch 镜像的 `COPY`
检查合成夹具，不复制真实私有数据。
