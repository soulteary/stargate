# Stargate - Forward Auth Service

[![License](https://img.shields.io/badge/license-Apache%202.0-blue.svg)](LICENSE)
[![Go Version](https://img.shields.io/badge/go-1.25+-blue.svg)](https://golang.org)
[![codecov](https://codecov.io/gh/soulteary/stargate/branch/main/graph/badge.svg)](https://codecov.io/gh/soulteary/stargate)

> **🚀 セキュアなマイクロサービスへのゲートウェイ**

Stargate は、本番環境に対応した軽量な Forward Auth サービスで、インフラ全体の**単一認証ポイント**として設計されています。Go で構築され、パフォーマンスに最適化されており、Stargate は Traefik やその他のリバースプロキシとシームレスに統合し、バックエンドサービスを保護します—**アプリケーションに認証コードを一行も書く必要はありません**。

## 🌐 多言語ドキュメント

- [English](README.md) | [中文](README.zhCN.md) | [Français](README.frFR.md) | [Italiano](README.itIT.md) | [日本語](README.jaJP.md) | [Deutsch](README.deDE.md) | [한국어](README.koKR.md)

### 🎯 なぜ Stargate なのか？

すべてのサービスで認証ロジックを実装することに疲れていませんか？Stargate は、エッジで認証を集中化することでこの問題を解決し、以下を可能にします：

- ✅ **単一の認証レイヤーで複数のサービスを保護**
- ✅ **アプリケーションから認証ロジックを削除してコードの複雑さを軽減**
- ✅ **Docker とシンプルな設定で数分でデプロイ**
- ✅ **最小限のリソースフットプリントで簡単にスケール**
- ✅ **複数の暗号化アルゴリズムと安全なセッション管理でセキュリティを維持**

### 💼 使用例

Stargate は以下に最適です：

- **マイクロサービスアーキテクチャ**：アプリケーションコードを変更せずに複数のバックエンドサービスを保護
- **マルチドメインアプリケーション**：異なるドメインとサブドメイン間で認証セッションを共有
- **内部ツールとダッシュボード**：内部サービスと管理パネルに迅速に認証を追加
- **API ゲートウェイ統合**：Traefik、Nginx、またはその他のリバースプロキシと統合認証レイヤーとして使用
- **開発とテスト**：開発環境向けのシンプルなパスワードベースの認証

## 📋 目次

- [機能](#機能)
- [クイックスタート](#クイックスタート)
- [設定](#設定)
- [ドキュメント](#ドキュメント)
- [API ドキュメント](#api-ドキュメント)
- [デプロイメントガイド](#デプロイメントガイド)
- [開発ガイド](#開発ガイド)
- [ライセンス](#ライセンス)

## ✨ 機能

### 🔐 エンタープライズグレードのセキュリティ

- **複数のパスワード暗号化アルゴリズム**：plaintext（テスト用）、bcrypt、MD5、SHA512 などから選択
- **安全なセッション管理**：カスタマイズ可能なドメインと有効期限を持つ Cookie ベースのセッション
- **柔軟な認証**：パスワードベースとセッションベースの両方の認証をサポート

### 🌐 高度な機能

- **クロスドメインセッション共有**：異なるドメイン/サブドメイン間でシームレスに認証セッションを共有
- **多言語サポート**：英語と中国語のインターフェースを内蔵、より多くの言語に簡単に拡張可能
- **カスタマイズ可能な UI**：カスタムタイトルとフッターテキストでログインページをブランディング

### 🚀 パフォーマンスと信頼性

- **軽量で高速**：Go と Fiber フレームワークで構築され、優れたパフォーマンスを実現
- **最小限のリソース使用**：メモリフットプリントが小さく、コンテナ化環境に最適
- **本番環境対応**：信頼性のために設計された実戦でテストされたアーキテクチャ

### 📦 開発者体験

- **Docker ファースト**：すぐに使える完全な Docker イメージと docker-compose 設定
- **Traefik ネイティブ**：ゼロ設定の Traefik Forward Auth ミドルウェア統合
- **シンプルな設定**：環境変数ベースの設定、複雑なファイルは不要

## 🚀 クイックスタート

**2 分以内**に Stargate を起動して実行できます！

### Docker Compose の使用（推奨）

**ステップ 1：** リポジトリをクローン
```bash
git clone <repository-url>
cd forward-auth
```

**ステップ 2：** 認証を設定（`codes/docker-compose.yml` を編集）
```yaml
services:
  stargate:
    environment:
      - AUTH_HOST=auth.example.com
      - PASSWORDS=plaintext:yourpassword1|yourpassword2
```

**ステップ 3：** サービスを起動
```bash
cd codes
docker-compose up -d
```

**これで完了です！** 認証サービスが実行されています。🎉

### ローカル開発

1. Go 1.25 以上がインストールされていることを確認

2. プロジェクトディレクトリに移動：
```bash
cd codes
```

3. ローカル起動スクリプトを実行：
```bash
chmod +x start-local.sh
./start-local.sh
```

4. ログインページにアクセス：
```
http://localhost:8080/_login?callback=localhost
```

## ⚙️ 設定

Stargate は、シンプルな環境変数ベースの設定システムを使用します。複雑な YAML ファイルや設定解析は不要—環境変数を設定するだけで準備完了です。

### 必須設定

| 環境変数 | 説明 | 例 |
|---------|------|------|
| `AUTH_HOST` | 認証サービスのホスト名 | `auth.example.com` |
| `PASSWORDS` | パスワード設定、形式：`algorithm:password1\|password2\|password3` | `plaintext:test123\|admin456` |

### オプション設定

| 環境変数 | 説明 | デフォルト | 例 |
|---------|------|-----------|------|
| `DEBUG` | デバッグモードを有効化 | `false` | `true` |
| `LANGUAGE` | インターフェース言語 | `en` | `ja`（日本語）、`zh`（中国語）、`en`（英語）、`fr`（フランス語）、`it`（イタリア語）、`de`（ドイツ語）、`ko`（韓国語） |
| `LOGIN_PAGE_TITLE` | ログインページのタイトル | `Stargate - Login` | `私の認証サービス` |
| `LOGIN_PAGE_FOOTER_TEXT` | ログインページのフッターテキスト | `Copyright © 2024 - Stargate` | `© 2024 私の会社` |
| `USER_HEADER_NAME` | 認証成功後に設定されるユーザーヘッダー名 | `X-Forwarded-User` | `X-Authenticated-User` |
| `COOKIE_DOMAIN` | Cookie ドメイン（クロスドメインセッション共有用） | 空（設定なし） | `.example.com` |
| `PORT` | サービスリスニングポート（ローカル開発のみ） | `80` | `8080` |

### パスワード設定形式

パスワード設定は次の形式を使用します：
```
algorithm:password1|password2|password3
```

サポートされているアルゴリズム：
- `plaintext`：プレーンテキストパスワード（テストのみ）
- `bcrypt`：BCrypt ハッシュ
- `md5`：MD5 ハッシュ
- `sha512`：SHA512 ハッシュ

例：
```bash
# プレーンテキストパスワード（複数）
PASSWORDS=plaintext:test123|admin456|user789

# BCrypt ハッシュ
PASSWORDS=bcrypt:$2a$10$N9qo8uLOickgx2ZMRZoMyeIjZAgcfl7p92ldGxad68LJZdL17lhWy

# MD5 ハッシュ
PASSWORDS=md5:5e884898da28047151d0e56f8dc6292773603d0d6aabbdd62a11ef721d1542d8
```

**詳細な設定については、[docs/jaJP/CONFIG.md](docs/jaJP/CONFIG.md) を参照してください**

## 📚 ドキュメント

Stargate を最大限に活用するための包括的なドキュメントが利用可能です：

- 📐 **[アーキテクチャドキュメント](docs/jaJP/ARCHITECTURE.md)** - 技術アーキテクチャと設計決定の詳細
- 🔌 **[API ドキュメント](docs/jaJP/API.md)** - 例付きの完全な API エンドポイントリファレンス
- ⚙️ **[設定リファレンス](docs/jaJP/CONFIG.md)** - 詳細な設定オプションとベストプラクティス
- 🚀 **[デプロイメントガイド](docs/jaJP/DEPLOYMENT.md)** - 本番環境デプロイメント戦略と推奨事項

## 📚 API ドキュメント

### 認証チェックエンドポイント

#### `GET /_auth`

Traefik Forward Auth の主要な認証チェックエンドポイント。

**リクエストヘッダー：**
- `Stargate-Password`（オプション）：API リクエスト用のパスワード認証
- `Cookie: stargate_session_id`（オプション）：Web リクエスト用のセッション認証

**レスポンス：**
- `200 OK`：認証成功、`X-Forwarded-User` ヘッダー（または設定されたユーザーヘッダー名）を設定
- `401 Unauthorized`：認証失敗
- `500 Internal Server Error`：サーバーエラー

**注意：**
- HTML リクエストは認証失敗時にログインページにリダイレクト
- API リクエスト（JSON/XML）は認証失敗時に 401 エラーを返す

### ログインエンドポイント

#### `GET /_login`

ログインページを表示します。

**クエリパラメータ：**
- `callback`（オプション）：ログイン成功後のコールバック URL

**レスポンス：**
- ログインページの HTML を返す

#### `POST /_login`

ログインリクエストを処理します。

**フォームデータ：**
- `password`：ユーザーパスワード
- `callback`（オプション）：ログイン成功後のコールバック URL

**コールバック取得の優先順位：**
1. Cookie から（以前に設定されている場合）
2. フォームデータから
3. クエリパラメータから
4. 上記のいずれもなく、発信元ドメインが認証サービスドメインと異なる場合、発信元ドメインをコールバックとして使用

**レスポンス：**
- `200 OK`：ログイン成功
  - コールバックが存在する場合、`{callback}/_session_exchange?id={session_id}` にリダイレクト
  - コールバックがない場合、成功メッセージを返す（リクエストタイプに応じて HTML または JSON 形式）
- `401 Unauthorized`：パスワードが間違っている
- `500 Internal Server Error`：サーバーエラー

### ログアウトエンドポイント

#### `GET /_logout`

現在のユーザーをログアウトし、セッションを破棄します。

**レスポンス：**
- `200 OK`：ログアウト成功、「Logged out」を返す

### セッション交換エンドポイント

#### `GET /_session_exchange`

クロスドメインセッション共有に使用されます。指定されたセッション ID の Cookie を設定してリダイレクトします。

**クエリパラメータ：**
- `id`（必須）：設定するセッション ID

**レスポンス：**
- `302 Redirect`：ルートパスにリダイレクト
- `400 Bad Request`：セッション ID が不足

### ヘルスチェックエンドポイント

#### `GET /health`

サービスヘルスチェックエンドポイント。

**レスポンス：**
- `200 OK`：サービスは正常

### ルートエンドポイント

#### `GET /`

ルートパス、サービス情報を表示します。

**詳細な API ドキュメントについては、[docs/jaJP/API.md](docs/jaJP/API.md) を参照してください**

## 🐳 デプロイメントガイド

### Docker デプロイメント

#### イメージをビルド

```bash
cd codes
docker build -t stargate:latest .
```

#### コンテナを実行

```bash
docker run -d \
  --name stargate \
  -p 80:80 \
  -e AUTH_HOST=auth.example.com \
  -e PASSWORDS=plaintext:yourpassword \
  stargate:latest
```

### Docker Compose デプロイメント

プロジェクトは `docker-compose.yml` のサンプル設定を提供し、Stargate サービスとサンプルの whoami サービスを含みます：

```bash
cd codes
docker-compose up -d
```

### Traefik 統合

`docker-compose.yml` で Traefik ラベルを設定：

```yaml
services:
  stargate:
    image: stargate:latest
    environment:
      - AUTH_HOST=auth.example.com
      - PASSWORDS=plaintext:yourpassword
    networks:
      - traefik
    labels:
      - "traefik.enable=true"
      - "traefik.docker.network=traefik"
      - "traefik.http.routers.auth.entrypoints=http"
      - "traefik.http.routers.auth.rule=Host(`auth.example.com`) || Path(`/_session_exchange`)"
      - "traefik.http.middlewares.stargate.forwardauth.address=http://stargate/_auth"

  your-service:
    image: your-service:latest
    networks:
      - traefik
    labels:
      - "traefik.enable=true"
      - "traefik.docker.network=traefik"
      - "traefik.http.routers.your-service.entrypoints=http"
      - "traefik.http.routers.your-service.rule=Host(`your-service.example.com`)"
      - "traefik.http.routers.your-service.middlewares=stargate"  # Stargate ミドルウェアを使用

networks:
  traefik:
    external: true
```

### 本番環境の推奨事項

1. **HTTPS を使用**：本番環境では、Traefik 経由で HTTPS が設定されていることを確認
2. **強力なパスワードアルゴリズムを使用**：`plaintext` を避け、`bcrypt` または `sha512` の使用を推奨
3. **Cookie ドメインを設定**：複数のサブドメイン間でセッションを共有する必要がある場合、`COOKIE_DOMAIN` を設定
4. **ログ管理**：適切なログローテーションと監視を設定
5. **リソース制限**：コンテナに適切な CPU とメモリ制限を設定

**詳細なデプロイメントガイドについては、[docs/jaJP/DEPLOYMENT.md](docs/jaJP/DEPLOYMENT.md) を参照してください**

## 💻 開発ガイド

### プロジェクト構造

```
codes/
├── src/
│   ├── cmd/
│   │   └── stargate/          # メインプログラムのエントリーポイント
│   │       ├── main.go        # プログラムエントリー
│   │       ├── server.go      # サーバー設定
│   │       └── constants.go  # 定数定義
│   ├── internal/
│   │   ├── auth/              # 認証ロジック
│   │   ├── config/            # 設定管理
│   │   ├── handlers/          # HTTP ハンドラー
│   │   ├── i18n/              # 国際化
│   │   ├── middleware/        # ミドルウェア
│   │   ├── secure/            # パスワード暗号化アルゴリズム
│   │   └── web/               # Web テンプレートと静的リソース
│   ├── go.mod
│   └── go.sum
├── Dockerfile
├── docker-compose.yml
└── start-local.sh
```

### ローカル開発

1. 依存関係をインストール：
```bash
cd codes
go mod download
```

2. テストを実行：
```bash
go test ./...
```

3. 開発サーバーを起動：
```bash
./start-local.sh
```

### 新しいパスワードアルゴリズムの追加

1. `src/internal/secure/` ディレクトリに新しいアルゴリズム実装を作成：
```go
package secure

type NewAlgorithmResolver struct{}

func (r *NewAlgorithmResolver) Check(h string, password string) bool {
    // パスワード検証ロジックを実装
    return false
}
```

2. `src/internal/config/validation.go` でアルゴリズムを登録：
```go
SupportedAlgorithms = map[string]secure.HashResolver{
    // ...
    "newalgorithm": &secure.NewAlgorithmResolver{},
}
```

### 新しい言語サポートの追加

1. `src/internal/i18n/i18n.go` に言語定数を追加：
```go
const (
    LangEN Language = "en"
    LangZH Language = "zh"
    LangJA Language = "ja"  // 新規
)
```

2. 翻訳マッピングを追加：
```go
var translations = map[Language]map[string]string{
    // ...
    LangJA: {
        "error.auth_required": "認証が必要です",
        // ...
    },
}
```

3. `src/internal/config/config.go` に言語オプションを追加：
```go
Language = EnvVariable{
    PossibleValues: []string{"en", "zh", "ja"},  // 新しい言語を追加
}
```

## 📝 ライセンス

このプロジェクトは Apache License 2.0 の下でライセンスされています。詳細については [LICENSE](codes/LICENSE) ファイルを参照してください。

## 🤝 貢献

貢献を歓迎します！以下を含みます：
- 🐛 バグレポート
- 💡 機能の提案
- 📝 ドキュメントの改善
- 🔧 コードの貢献

Issue を開くか、Pull Request を送信してください。すべての貢献が Stargate をより良くします！

---

## ⚠️ 本番環境チェックリスト

本番環境にデプロイする前に、以下のセキュリティのベストプラクティスを完了していることを確認してください：

- ✅ **強力なパスワードを使用**：`plaintext` を避け、パスワードハッシュに `bcrypt` または `sha512` を使用
- ✅ **HTTPS を有効化**：Traefik またはリバースプロキシ経由で HTTPS を設定
- ✅ **Cookie ドメインを設定**：サブドメイン間で適切なセッション管理のために `COOKIE_DOMAIN` を設定
- ✅ **監視とログ**：デプロイメントに適切なログ記録と監視を設定
- ✅ **定期的な更新**：セキュリティパッチのために Stargate を最新バージョンに保つ
