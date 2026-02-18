# デプロイメントガイド

このドキュメントは、Stargate Forward Auth サービスの詳細なデプロイメントガイドを提供します。

## 目次

- [デプロイメント方法](#デプロイメント方法)
- [Docker デプロイメント](#docker-デプロイメント)
- [Docker Compose デプロイメント](#docker-compose-デプロイメント)
- [Traefik 統合](#traefik-統合)
- [本番環境デプロイメント](#本番環境デプロイメント)
- [監視とメンテナンス](#監視とメンテナンス)
- [トラブルシューティング](#トラブルシューティング)

## デプロイメント方法

Stargate は以下のデプロイメント方法をサポートします：

1. **Docker コンテナ**（推奨）- 最もシンプルで一般的
2. **Docker Compose** - ローカル開発とテストに適しています
3. **Kubernetes** - 大規模な本番環境に適しています
4. **バイナリ直接実行** - 特別なシナリオに適しています

このドキュメントでは、主に Docker と Docker Compose のデプロイメント方法を紹介します。

## サービス依存関係

Stargateは以下のオプションサービスと統合できます：

### Wardenサービス

**機能:** ユーザーホワイトリスト管理とユーザー情報の提供

**デプロイメント要件:**
- データベースが必要（PostgreSQL/MySQL/SQLite）
- HTTP APIインターフェースを提供
- APIキー認証をサポート

**設定:**
```bash
WARDEN_ENABLED=true
WARDEN_URL=http://warden:8080
WARDEN_API_KEY=your-api-key
```

### Heraldサービス

**機能:** OTP/検証コードの送信と検証

**デプロイメント要件:**
- Redisが必要（チャレンジとレート制限状態を保存）
- HTTP APIインターフェースを提供
- HMAC署名またはmTLS認証をサポート（本番環境で推奨）

**設定:**
```bash
HERALD_ENABLED=true
HERALD_URL=http://herald:8080
HERALD_HMAC_SECRET=your-hmac-secret  # 本番環境で推奨
```

### サービス間通信のセキュリティ

**本番環境の要件:**

1. **HMAC署名認証**（推奨）:
   - Stargate ↔ HeraldはHMAC-SHA256署名を使用
   - `HERALD_HMAC_SECRET`を設定
   - タイムスタンプ検証を含む（リプレイ攻撃を防止）

2. **mTLS認証**（オプション、より安全）:
   - TLSクライアント証明書を設定
   - `HERALD_TLS_CLIENT_CERT_FILE`と`HERALD_TLS_CLIENT_KEY_FILE`を設定
   - CA証明書の検証を設定

3. **ネットワーク分離:**
   - サービス間通信は内部ネットワーク上で行う必要があります
   - ファイアウォールルールを使用してアクセスを制限
   - サービスをパブリックネットワークに公開しないようにする

## Docker デプロイメント

### イメージのビルド

#### ソースからビルド

```bash
cd codes
docker build -t stargate:latest .
```

#### ビルドパラメータ

- **ベースイメージ**: `golang:1.26-alpine`（ビルドステージ）
- **実行イメージ**: `scratch`（最小イメージ）
- **作業ディレクトリ**: `/app`
- **公開ポート**: `80`

### コンテナの実行

#### 基本実行

```bash
docker run -d \
  --name stargate \
  -p 80:80 \
  -e AUTH_HOST=auth.example.com \
  -e PASSWORDS=plaintext:yourpassword \
  stargate:latest
```

#### 完全設定での実行

```bash
docker run -d \
  --name stargate \
  -p 80:80 \
  -e AUTH_HOST=auth.example.com \
  -e PASSWORDS=bcrypt:$2a$10$N9qo8uLOickgx2ZMRZoMyeIjZAgcfl7p92ldGxad68LJZdL17lhWy \
  -e DEBUG=false \
  -e LANGUAGE=ja \
  -e LOGIN_PAGE_TITLE=私の認証サービス \
  -e LOGIN_PAGE_FOOTER_TEXT=© 2024 私の会社 \
  -e COOKIE_DOMAIN=.example.com \
  --restart unless-stopped \
  stargate:latest
```

#### パラメータの説明

- `-d`: バックグラウンドで実行
- `--name stargate`: コンテナ名
- `-p 80:80`: ポートマッピング（ホストポート:コンテナポート）
- `-e`: 環境変数
- `--restart unless-stopped`: 自動再起動ポリシー

### ログの表示

```bash
# リアルタイムでログを表示
docker logs -f stargate

# 最後の 100 行のログを表示
docker logs --tail 100 stargate
```

### 停止と削除

```bash
# コンテナを停止
docker stop stargate

# コンテナを削除
docker rm stargate

# 停止して削除
docker rm -f stargate
```

## Docker Compose デプロイメント

### 基本設定

プロジェクトは `docker-compose.yml` のサンプルファイルを提供します：

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

### サービスの起動

```bash
cd codes
docker-compose up -d
```

### サービスの停止

```bash
docker-compose down
```

### ログの表示

```bash
# すべてのサービスのログを表示
docker-compose logs -f

# 特定のサービスのログを表示
docker-compose logs -f stargate
```

### カスタム設定

`docker-compose.yml` を編集し、環境変数を変更します：

```yaml
services:
  stargate:
    image: stargate
    environment:
      - AUTH_HOST=auth.example.com
      - PASSWORDS=bcrypt:$2a$10$...
      - DEBUG=false
      - LANGUAGE=ja
      - COOKIE_DOMAIN=.example.com
```

## Traefik 統合

### 基本設定

Stargate は Traefik と統合するように設計されており、Forward Auth ミドルウェア経由で認証を提供します。

#### 1. Stargate サービスの設定

`docker-compose.yml` で Stargate を設定します：

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

#### 2. 保護されたサービスの設定

認証が必要なサービスに Stargate ミドルウェアを適用します：

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
      - "traefik.http.routers.your-app.middlewares=stargate"  # 認証ミドルウェアを適用
```

### HTTPS 設定

#### Let's Encrypt の使用

```yaml
services:
  stargate:
    labels:
      - "traefik.http.routers.auth.entrypoints=https"
      - "traefik.http.routers.auth.tls.certresolver=letsencrypt"
      - "traefik.http.routers.auth.tls=true"
```

#### カスタム証明書の使用

```yaml
services:
  stargate:
    labels:
      - "traefik.http.routers.auth.entrypoints=https"
      - "traefik.http.routers.auth.tls=true"
      - "traefik.http.routers.auth.tls.certfile=/path/to/cert.pem"
      - "traefik.http.routers.auth.tls.keyfile=/path/to/key.pem"
```

### クロスドメインセッション共有

サブドメイン間でセッションを共有する必要がある場合：

1. 環境変数 `COOKIE_DOMAIN` を設定します：

```yaml
services:
  stargate:
    environment:
      - COOKIE_DOMAIN=.example.com
```

2. 関連するすべてのドメインが Traefik 経由で Stargate にルーティングされていることを確認

3. ログインフロー：
   - ユーザーが `auth.example.com` にログイン
   - `app.example.com/_session_exchange?id=<session_id>` にリダイレクト
   - セッション Cookie が `.example.com` ドメインに設定される
   - すべてのサブドメイン `*.example.com` でこのセッションを使用できます

## 本番環境デプロイメント

### セキュリティの推奨事項

#### 1. 強力なパスワードアルゴリズムを使用

**推奨されません：**

```bash
PASSWORDS=plaintext:yourpassword
```

**推奨：**

```bash
PASSWORDS=bcrypt:$2a$10$N9qo8uLOickgx2ZMRZoMyeIjZAgcfl7p92ldGxad68LJZdL17lhWy
```

#### 2. HTTPS を有効化

- Traefik 経由で HTTPS を設定
- 自動 Let's Encrypt 証明書を使用
- HTTPS リダイレクトを強制

#### 3. デバッグモードを無効化

```bash
DEBUG=false
```

#### 4. リソース制限を設定

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

#### 5. ヘルスチェックを使用

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

### 高可用性デプロイメント

#### 1. マルチインスタンスデプロイメント

```yaml
services:
  stargate:
    deploy:
      replicas: 3
```

**注意:** Stargate はメモリ内セッションストレージを使用するため、インスタンス間でセッションは共有されません。マルチインスタンスデプロイメントが必要な場合、以下を推奨します：

- ロードバランサーのセッション永続化（Sticky Session）を使用
- または外部セッションストレージ（Redis）のサポートを待つ

#### 2. ロードバランシング

Traefik の前にロードバランサーを追加：

```yaml
services:
  traefik:
    labels:
      - "traefik.http.services.stargate.loadbalancer.server.port=80"
```

### 監視設定

#### 1. ログ収集

```yaml
services:
  stargate:
    logging:
      driver: "json-file"
      options:
        max-size: "10m"
        max-file: "3"
```

#### 2. ヘルスチェックエンドポイント

監視に `/health` エンドポイントを使用：

```bash
# ヘルスチェックスクリプト
#!/bin/bash
if curl -f http://localhost/health > /dev/null 2>&1; then
  exit 0
else
  exit 1
fi
```

#### 3. Prometheus 統合

（実装予定）将来のバージョンでは Prometheus メトリクスのエクスポートをサポートします。

## 監視とメンテナンス

### ログ管理

#### ログの表示

```bash
# Docker
docker logs -f stargate

# Docker Compose
docker-compose logs -f stargate
```

#### ログレベル

- `DEBUG=true`: 詳細なデバッグ情報
- `DEBUG=false`: 重要な情報のみ

#### ログローテーション

Docker ログドライバーを設定：

```yaml
services:
  stargate:
    logging:
      driver: "json-file"
      options:
        max-size: "10m"
        max-file: "3"
```

### パフォーマンス監視

#### リソース使用量

```bash
# コンテナのリソース使用量を表示
docker stats stargate
```

#### 応答時間

ヘルスチェックエンドポイントを使用して応答時間を監視：

```bash
time curl http://auth.example.com/health
```

### 定期的なメンテナンス

1. **イメージの更新**: 定期的に最新のイメージをダウンロード
2. **ログの確認**: 定期的にエラーログを確認
3. **リソースの監視**: CPU とメモリの使用量を監視
4. **設定のバックアップ**: 環境変数の設定をバックアップ

## トラブルシューティング

### 一般的な問題

#### 1. サービスが起動しない

**問題:** コンテナが起動直後に終了する

**トラブルシューティング手順:**

```bash
# コンテナのログを表示
docker logs stargate

# 設定を確認
docker inspect stargate | grep -A 20 Env
```

**一般的な原因:**

- 必須設定が不足（`AUTH_HOST`、`PASSWORDS`）
- パスワード設定形式が正しくない
- ポートが使用中

#### 2. 認証が失敗する

**問題:** ユーザーがログインできない

**トラブルシューティング手順:**

1. パスワード設定が正しいか確認
2. パスワードアルゴリズムが一致しているか確認
3. サービスのログを表示: `docker logs stargate`

**一般的な原因:**

- パスワード設定が間違っている
- パスワードアルゴリズムの不一致（例: bcrypt が設定されているがプレーンテキストパスワードが使用されている）
- Cookie ドメインの設定が間違っている

#### 3. クロスドメインセッションが機能しない

**問題:** サブドメイン間でセッションを共有できない

**トラブルシューティング手順:**

1. `COOKIE_DOMAIN` 設定を確認
2. Cookie ドメインの形式が正しいことを確認（`.example.com`）
3. ブラウザの Cookie 設定を確認

**解決策:**

```bash
# COOKIE_DOMAIN が設定されていることを確認
COOKIE_DOMAIN=.example.com
```

#### 4. Traefik 統合の問題

**問題:** Traefik が認証リクエストを正しく転送できない

**トラブルシューティング手順:**

1. Traefik ラベルの設定を確認
2. ネットワーク設定が正しいことを確認
3. Forward Auth ミドルウェアのアドレスを確認

**解決策:**

```yaml
# ミドルウェアのアドレスが正しいことを確認
- "traefik.http.middlewares.stargate.forwardauth.address=http://stargate/_auth"
```

### デバッグのヒント

#### 1. デバッグモードを有効化

```bash
DEBUG=true
```

#### 2. ネットワーク接続を確認

```bash
# コンテナ内からテスト
docker exec stargate wget -O- http://localhost/health
```

#### 3. Traefik ログを表示

```bash
docker logs traefik
```

#### 4. API エンドポイントをテスト

```bash
# ヘルスチェックをテスト
curl http://auth.example.com/health

# 認証をテスト（ヘッダーを使用）
curl -H "Stargate-Password: yourpassword" http://auth.example.com/_auth

# 認証をテスト（Cookie を使用）
curl -H "Cookie: stargate_session_id=<session_id>" http://auth.example.com/_auth
```

### ヘルプの取得

問題が発生した場合：

1. ログを表示: `docker logs stargate`
2. 設定を確認: すべての環境変数が正しいことを確認
3. ドキュメントを参照: [API ドキュメント](API.md)、[設定リファレンス](CONFIG.md)
4. Issue を提出: プロジェクトリポジトリに問題レポートを提出

## 更新ガイド

### 更新手順

1. **設定のバックアップ**: 現在の環境変数設定をバックアップ

2. **サービスの停止:**

```bash
docker stop stargate
```

3. **新しいイメージのダウンロード:**

```bash
docker pull stargate:latest
```

4. **新しいコンテナの起動:**

```bash
docker run -d \
  --name stargate \
  ...(保存された設定を使用)
  stargate:latest
```

5. **サービスの確認:**

```bash
curl http://auth.example.com/health
```

### ロールバック

更新後に問題が発生した場合：

```bash
# 新しいコンテナを停止
docker stop stargate

# 古いイメージで起動
docker run -d \
  --name stargate \
  ...(保存された設定を使用)
  stargate:<old-version>
```
