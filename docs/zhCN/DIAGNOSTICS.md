# 部署诊断

本命令是当前源码新增能力，已发布的 v1.3.5 二进制尚不包含它。请使用包含本改动的构建。
旧二进制可能忽略未知子命令而启动服务，不应运行 `doctor` 来试探是否支持；应先检查发布/构建元数据。

`stargate doctor` 默认只检查配置快照，不启动应用、审计写入器、OTLP、SDK 或 Redis
客户端。不带参数运行 Stargate 仍按原方式启动。

```sh
stargate doctor
stargate doctor --config doctor.json --json
stargate doctor --config doctor.json --health --json
```

未指定文件时，只读取网关的主机、Cookie、回调、会话交换、集成服务 URL/认证/TLS、
TOTP 开关及会话 Redis 开关/地址等已声明环境字段，不自动加载 `.env`。指定文件后仅使用
该 JSON 快照，不与环境合并，不执行 shell。文件格式与完整字段说明见
[英文指南](../enUS/DIAGNOSTICS.md)。`gateway` 为必填对象；`warden`、`herald`、`totp`
为可选对象，内部使用各服务环境变量名，值必须为字符串。

输入可能包含服务凭据，须在本地由维护者提供；Unix 上凭据文件要求 `chmod 600`，
不得提交仓库、上传工单或作为命令行参数粘贴密钥。输入中的占位符不会自动替换为环境值。
输出不包含输入值、哈希、Cookie、文件路径、原始错误或响应正文。

离线检查复用启动时的主机/Cookie 域、回调语法、服务根 URL 和会话落地路径校验，
另检查部分开关、设置 COOKIE_DOMAIN 或 CALLBACK_ALLOWED_HOSTS 时启动所需的至少 32 字节会话交换密钥、Redis 地址语法，以及本地提供的三条凭据配对。
密码格式使用共享启动解析器，但不进行密码认证。它不覆盖完整启动校验、UI 配置（包括 `LOGIN_SUPPORT_URL`）、全部开关组合和服务动作策略。
未提供服务端配置时显示 `unknown`，不会声称认证成功。客户端选择 HMAC 时不会改发 API Key；
mTLS 的实际接受与授权不能仅通过配置对比证明。只有未设置上述跨域选项时，未配置会话交换密钥才不影响该项启动检查。

只有显式使用 `--health` 才访问网络：网关 `/readyz`、Warden `/health`、Herald `/readyz`、
TOTP `/healthz`。网关 `/healthz` 只表示存活；Herald/TOTP 就绪也不证明 TOTP 代理、邮件投递
或指定账号可用。使用顶层 `gateway_url` 或 `--gateway-url` 指定网关健康地址，须匹配 `AUTH_HOST`；
不能从 `COOKIE_SECURE` 推断 HTTP/HTTPS。显式公网 HTTP 与 Secure Cookie 的冲突会报告失败；
localhost 的浏览器例外仍需实际核验。私有网关 CA 可用 `gateway_tls_ca_file` 或
`--gateway-ca-file` 指定。

在线模式校验 HTTPS 证书，禁用重定向及环境代理，不发送 API Key、HMAC 或浏览器 Cookie。
每个请求最多 2 秒、64 KiB，总 context 最多 10 秒；仅接受可识别的 JSON 健康结果。
健康检查可能产生服务访问日志和指标，但不调用业务或会话端点。不应为取得通过而关闭健康接口访问策略。

健康通过不证明 HMAC、Redis 写权限、回调、Cookie 传输、验证码或备用码恢复。
工具不提供主动认证探针或自动修复；认证 GET 也可能写 replay nonce 和审计，因此不能标记为只读。
完整烟测须另用明确授权的合成账号和隔离状态进行。

状态为 `pass`、`fail`、`unknown`、`skipped`；退出码 0 表示本次请求的有限检查通过，
1 表示失败，2 表示参数/输入错误，3 表示信息不足。它们不代替启动校验或真实浏览器登录。
主机/Cookie/回调错误先核对对应配置；凭据错误在本地核对 ID、密钥和认证模式；健康错误核对
网络、路由、TLS、访问策略和 Redis。单个 401 不能断定一定是 key ID 错误，不得直接输出后端原始正文。
