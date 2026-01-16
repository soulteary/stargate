# OIDC 认证

Stargate 支持 OpenID Connect (OIDC) 认证，可与企业 SSO 提供商集成。

## 配置

通过设置以下环境变量启用 OIDC：

| 变量 | 必需 | 默认值 | 描述 |
|------|------|--------|------|
| `OIDC_ENABLED` | 是 | `false` | 设置为 `true` 启用 OIDC |
| `OIDC_ISSUER_URL` | 是* | - | OIDC 提供商的 Issuer URL |
| `OIDC_CLIENT_ID` | 是* | - | 从 OIDC 提供商获取的客户端 ID |
| `OIDC_CLIENT_SECRET` | 是* | - | 从 OIDC 提供商获取的客户端密钥 |
| `OIDC_REDIRECT_URI` | 否 | `https://{AUTH_HOST}/_oidc/callback` | 应用的回调 URL |
| `OIDC_PROVIDER_NAME` | 否 | `OIDC` | 登录按钮显示的名称 |

*当 `OIDC_ENABLED=true` 时必需

## docker-compose.yml 示例

```yaml
services:
  stargate:
    image: stargate:latest
    environment:
      - AUTH_HOST=auth.example.com
      - OIDC_ENABLED=true
      - OIDC_ISSUER_URL=https://keycloak.example.com/realms/myrealm
      - OIDC_CLIENT_ID=stargate-client
      - OIDC_CLIENT_SECRET=your-secret
      - OIDC_PROVIDER_NAME=公司账号
```

## 支持的提供商

任何符合 OIDC 标准的提供商都可以使用，包括：
- Keycloak
- Azure AD / Entra ID
- Auth0
- Okta
- Google Workspace
- 自建 OIDC 服务器

## 注意事项

- 启用 OIDC 后，密码认证会自动禁用
- 仅请求 `openid` 和 `email` 范围
- 用户 ID（sub 声明）和邮箱存储在会话中
- `X-Forwarded-User` 请求头包含用户 ID 或邮箱
- 如果需要使用 `http` 或自定义回调地址，请显式设置 `OIDC_REDIRECT_URI`
