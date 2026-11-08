# 邮件验证码与验证器完整示例

本示例连接正式发布的 Stargate 1.3.5、Warden 1.6.2、Herald 1.4.4 和
herald-totp 2.0.3，使用真实 SMTP 投递、持久 Redis、显式 HMAC key ID 及三条独立服务凭据。
不构建应用、不提供读取验证码的测试接口。仅需密码演示时，请使用[根目录快速开始](../../README.zhCN.md#-快速开始)。

## 本地邮件 OTP

需要 Python 3、Docker Compose v2，以及空闲的 loopback 8088、8025 端口。
浏览器须将两个 `*.test.localhost` 域名解析到本机；若不能自动解析，在 hosts 文件加入：
`127.0.0.1 auth.test.localhost hello.test.localhost`。

```sh
cd examples/full-stack
python3 configure.py --mode mail
docker compose config --quiet
docker compose up -d
```

打开 **http://hello.test.localhost:8088**，输入 `demo@example.test` 并发送邮件验证码，
再打开 **http://127.0.0.1:8025** 的 Mailpit 收件箱查看邮件。输入实际投递的验证码后，
确认浏览器返回受保护的演示服务。邮件经过 Herald 的真实 SMTP 提供方，但由本地 Mailpit
捕获，并非投递到外部邮箱。本例没有配置短信提供方。

认证页为 `http://auth.test.localhost:8088/_login`。登出需要在认证域发起同源
`POST /_logout`，不能用 GET 链接代替。使用浏览器登出操作或同源表单后，再访问受保护服务，
确认需要重新认证。验收时不要清空 Redis 或删除卷。

初次生成前可指定其他域名和端口：

```sh
python3 configure.py --port 9090 --mailbox-port 9025 \
  --auth-host auth.demo.localhost --app-host app.demo.localhost
```

生成器分别配置仅含 DNS 名的代理规则和包含实际端口的回调地址。访问
`http://app.demo.localhost:9090` 和 `http://127.0.0.1:9025`。
80/443 使用不带端口后缀的规范地址。默认网络冲突时，用 `--subnet` 指定空闲的私有 /24。
共享 Docker 环境必须选择独立 `--project`，不要复用其他部署的项目或网络。

## 添加验证器

新建配置时使用 `python3 configure.py --mode totp`，即可同时启用 TOTP 服务及两层代理开关。
先通过邮件 OTP 登录，再访问 `http://auth.test.localhost:8088/totp/enroll`，点击开始绑定，
扫描二维码或在验证器中手动输入设置密钥，使用动态码确认。离开前妥善保存页面显示的备用码。

登出后测试验证器登录，再测试一个备用码。已使用的备用码在新的登录 challenge 中仍应失败。
没有全局共享 TOTP 密钥；凭据绑定到稳定的 `demo-user-001` 账号 ID。修改邮箱不能顺便创建新 ID。

## 配置与持久化

生成器创建权限为 0600 的 `.env` 及私有 `state/` 目录，不打印秘密，也不覆盖既有配置。
重复运行保留所有密钥及账号 ID，**不会应用新的命令行参数**。发现残缺配置时拒绝继续，留待检查。
不要为了改端口或启用 TOTP 而删除既有配置：先备份，再协调修改非秘密参数，保留全部密钥。
已有邮件配置启用 TOTP 时设置 `COMPOSE_PROFILES=totp`、`TOTP_ENABLED=true`，其余密钥保持不变。

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
内部 HTTP 仅用于私有 Compose 网络；跨主机或不可信网络须先配置服务及 SDK 的 TLS/mTLS。
mTLS 不能替代 Herald/TOTP 的请求认证。

1. 在受保护目录准备 `cert.pem`、`key.pem`，将 `tls.yml.example` 复制为 `tls.yml`。
   私钥不得进入 Git。
2. 在 Shell 环境设置 `SMTP_HOST`、`SMTP_PORT`、`SMTP_FROM`、`SMTP_USER`、`SMTP_PASSWORD`
   和 `TLS_DIR`。不要在命令行参数或 Shell 历史中写密码。此模板要求认证 SMTP；使用其他
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

生成器在写入前校验证书/私钥配对及必填项，不代表已验证证书域名覆盖、SMTP 可投递性、备份质量或网络策略。
这些要求须另行验收。生产 `COOKIE_SECURE=true`，代理通过 443 端口提供 TLS。
打开 `https://app.example.com`，先验证受控邮箱的真实投递，再添加其他账号。
将 `hello` 演示替换为实际应用时，保留 ForwardAuth 中间件。

## 停止与排障

使用 `docker compose ps` 和 `docker compose logs --tail=50 SERVICE` 检查启动问题，日志按敏感数据处理。
进程健康不等于能登录。依次检查 DNS/端口、TLS 与 Cookie、准确的回调地址、Warden 稳定 ID/来源有效期、
三条 HMAC key ID 配对、Redis、真实投递。不要通过关闭校验、切换 soft 投递或重置限流来让示例通过。
参阅[部署指南](../../docs/zhCN/DEPLOYMENT.md)。
