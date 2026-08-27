# v0.12.0 から v1.0.0 への移行

Stargate v1.0.0 はデプロイと HTTP の安定した契約を定義します。本番トラフィックを切り替える前にステージングで確認してください。

## 更新前の準備

1. 現在のイメージ、環境変数、Proxy ルート、Callback ホスト、Probe を記録します。
2. ロールバック用に `v0.12.0` イメージと以前のコンテナを保持します。
3. `/_logout`、`/totp/enroll`、`/_session_exchange` の呼び出し元を確認します。
4. 複数レプリカでは Redis セッションストレージを準備します。

## 必須の変更

| 契約 | v0.12.0 | v1.0.0 |
| --- | --- | --- |
| コンテナポート | `80` | `8080` |
| 実行ユーザー | root | UID/GID `10001` |
| Liveness | `/health` | `/healthz` |
| Readiness | `/health` | `/readyz` |

Traefik ForwardAuth の接続先を `http://stargate:8080/_auth` に変更します。複数レプリカでは `SESSION_STORAGE_ENABLED=true` を有効にし、Redis と Ticket Secret を共有します。

## セキュリティと HTTP 契約

- クロスドメイン交換は生の Session ID `?id=` ではなく `?ticket=` を使用します。`CALLBACK_ALLOWED_HOSTS` と 32 文字以上の `SESSION_EXCHANGE_SECRET` を設定します。
- Forwarded Header は `TRUSTED_PROXIES` に登録した送信元からのみ受け入れます。
- `Stargate-Password` は既定で無効です。信頼済み Proxy でのみ `PASSWORD_HEADER_AUTH_ENABLED=true` を設定します。
- Identity Header には Warden、`HEADER_AUTH_ENABLED=true`、強力な共有 Secret が必要です。
- `GET /_logout` は `POST /_logout` に変わります。
- `GET /totp/enroll` は確認ページのみ表示し、`POST /totp/enroll` が登録を開始します。
- `GET /metrics` は変更されません。

## ロールアウトと確認

```bash
curl --fail http://127.0.0.1:8080/healthz
curl --fail http://127.0.0.1:8080/readyz
curl --fail http://127.0.0.1:8080/metrics
```

Login、Logout、TOTP、許可・拒否される Callback、期限切れ・再利用 Ticket を確認します。v0.12.0 と v1.0.0 を同じ Pool に混在させないでください。

## ロールバック

v0.12.0 のイメージ、ポート、Probe を一緒に復元します。処理中の v1 Ticket を無効化し、再ログインを要求します。
