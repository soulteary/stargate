# v0.12.0 から v1.0.0 への移行

Stargate v1.0.0 はデプロイと HTTP の安定した契約を定義します。本番トラフィックを切り替える前にステージングで確認してください。

## 更新前の準備

1. 現在のイメージ、環境変数、Proxy ルート、Callback ホスト、Probe を記録します。
2. ロールバック用に `v0.12.0` イメージと以前のコンテナを保持します。
3. `/_logout`、`/totp/enroll`、`/_session_exchange` の呼び出し元を確認します。
4. 複数プロセスが Traffic を処理する場合、Rollout 中に新旧プロセスが重なる場合、または Session と消費済み Ticket の状態を再起動後も保持する場合は Redis を準備します。1 つの稼働中プロセスなら Cross-Domain Callback があってもインメモリストレージを使えますが、すべての状態はプロセスローカルで再起動時に失われます。

## ロールバック設定を保持して v1 環境を作成

古い環境ファイルで v1 を起動しないでください。`stargate.env` を変更せず、v0.12.0 Container も保持します。次のコマンドは保護されたロールバックコピーと独立した v1 ファイルを作成し、既存の出力を上書きせず、廃止設定を v1 ファイルからだけ除去します。

```bash
set -eu
old_env=./stargate.env
rollback_env=./stargate-v0.12.0.env
v1_env=./stargate-v1.env

test -f "$old_env"
test ! -e "$rollback_env"
test ! -e "$v1_env"
chmod 600 "$old_env"
umask 077
(set -C; cat "$old_env" > "$rollback_env")
(set -C; awk '!/^[[:space:]]*WARDEN_OTP_(ENABLED|SECRET_KEY)[[:space:]]*=/' "$old_env" > "$v1_env")
```

Herald ベースの TOTP では、Stargate が Warden 経由で認証済みユーザーを解決する必要があるため、`WARDEN_ENABLED=true` と `WARDEN_URL` も設定してください。

v1 は `WARDEN_OTP_ENABLED` または `WARDEN_OTP_SECRET_KEY` が存在するだけで、空や `false` でも拒否します。`stargate-v1.env` に追加しないでください。TOTP には Stargate の `HERALD_ENABLED=true`、`HERALD_TOTP_ENABLED=true`、`HERALD_URL` と対応する Herald Service Auth を設定します。Herald から herald-totp への TOTP Proxy と、herald-totp 独自の `HERALD_TOTP_ENCRYPTION_KEY` も設定し、旧 Warden Secret を自動再利用しないでください。[設定](CONFIG.md)を参照してください。

`--env-file ./stargate-v1.env` で v1 を検証・起動します。ロールバック期間が終わるまで `stargate-v0.12.0.env`、未変更の元ファイル、旧 Container を保持してください。

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
