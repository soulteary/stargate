# Stargate アーキテクチャドキュメント

このドキュメントは、Stargate プロジェクトの技術アーキテクチャと設計決定を説明します。

## 技術スタック

- **言語**: Go 1.25
- **Web フレームワーク**: [Fiber v2.52.10](https://github.com/gofiber/fiber)
- **テンプレートエンジン**: [Fiber Template v1.7.5](https://github.com/gofiber/template)
- **セッション管理**: Fiber Session Middleware
- **ログ**: [Logrus v1.9.3](https://github.com/sirupsen/logrus)
- **ターミナル出力**: [Pterm v0.12.82](https://github.com/pterm/pterm)
- **テストフレームワーク**: [Testza v0.5.2](https://github.com/MarvinJWendt/testza)

## プロジェクト構造

```
codes/src/
├── cmd/stargate/          # アプリケーションエントリーポイント
│   ├── main.go            # メイン関数、設定の初期化とサーバーの起動
│   ├── server.go          # サーバー設定とルート設定
│   └── constants.go       # ルートと設定の定数
│
├── internal/              # 内部パッケージ（外部に公開されない）
│   ├── auth/              # 認証ロジック
│   │   ├── auth.go        # 認証の主要機能
│   │   └── auth_test.go   # 認証テスト
│   │
│   ├── config/            # 設定管理
│   │   ├── config.go      # 設定変数の定義と初期化
│   │   ├── validation.go  # 設定検証ロジック
│   │   └── config_test.go # 設定テスト
│   │
│   ├── handlers/          # HTTP リクエストハンドラー
│   │   ├── check.go       # 認証チェックハンドラー
│   │   ├── login.go       # ログインハンドラー
│   │   ├── logout.go      # ログアウトハンドラー
│   │   ├── session_share.go # セッション共有ハンドラー
│   │   ├── health.go      # ヘルスチェックハンドラー
│   │   ├── index.go       # ルートパスハンドラー
│   │   ├── utils.go       # ハンドラーユーティリティ関数
│   │   └── handlers_test.go # ハンドラーテスト
│   │
│   ├── i18n/              # 国際化サポート
│   │   └── i18n.go        # 多言語翻訳
│   │
│   ├── middleware/        # HTTP ミドルウェア
│   │   └── log.go         # ログミドルウェア
│   │
│   ├── secure/            # パスワード暗号化アルゴリズム
│   │   ├── interface.go   # 暗号化アルゴリズムインターフェース
│   │   ├── plaintext.go   # プレーンテキストパスワード（テストのみ）
│   │   ├── bcrypt.go      # BCrypt アルゴリズム
│   │   ├── md5.go         # MD5 アルゴリズム
│   │   ├── sha512.go      # SHA512 アルゴリズム
│   │   └── secure_test.go # 暗号化アルゴリズムテスト
│   │
│   └── web/               # Web リソース
│       └── templates/     # HTML テンプレート
│           ├── login.html # ログインページテンプレート
│           └── assets/   # 静的リソース
│               └── favicon.ico
```

## 主要コンポーネント

### 1. 認証システム (`internal/auth`)

認証システムは以下を担当します：
- パスワードの検証（複数の暗号化アルゴリズムをサポート）
- セッション管理（作成、検証、破棄）
- 認証状態の検証

**主要関数：**
- `CheckPassword(password string) bool`: パスワードを検証
- `Authenticate(session *session.Session) error`: セッションを認証済みとしてマーク
- `IsAuthenticated(session *session.Session) bool`: セッションが認証されているか確認
- `Unauthenticate(session *session.Session) error`: セッションを破棄

### 2. 設定システム (`internal/config`)

設定システムは以下を提供します：
- 環境変数の管理
- 設定の検証
- デフォルト値のサポート

**設定変数：**
- `AUTH_HOST`: 認証サービスのホスト名（必須）
- `PASSWORDS`: パスワード設定（アルゴリズム:パスワードのリスト）（必須）
- `DEBUG`: デバッグモード（デフォルト: false）
- `LANGUAGE`: インターフェース言語（デフォルト: en、en/zh/fr/it/ja/de/ko をサポート）
- `COOKIE_DOMAIN`: Cookie ドメイン（オプション、クロスドメインセッション共有用）
- `LOGIN_PAGE_TITLE`: ログインページのタイトル（デフォルト: Stargate - Login）
- `LOGIN_PAGE_FOOTER_TEXT`: ログインページのフッターテキスト（デフォルト: Copyright © 2024 - Stargate）
- `USER_HEADER_NAME`: 認証成功後に設定されるユーザーヘッダー名（デフォルト: X-Forwarded-User）
- `PORT`: サービスのリスニングポート（ローカル開発のみ、デフォルト: 80）

### 3. リクエストハンドラー (`internal/handlers`)

ハンドラーは HTTP リクエストの処理を担当します：

- **CheckRoute**: Traefik Forward Auth の認証チェック
- **LoginRoute/LoginAPI**: ログインページとログイン処理
- **LogoutRoute**: ログアウト処理
- **SessionShareRoute**: クロスドメインセッション共有
- **HealthRoute**: ヘルスチェック
- **IndexRoute**: ルートパスの処理

### 4. パスワード暗号化 (`internal/secure`)

複数のパスワード暗号化アルゴリズムをサポート：
- `plaintext`: プレーンテキスト（テストのみ）
- `bcrypt`: BCrypt ハッシュ
- `md5`: MD5 ハッシュ
- `sha512`: SHA512 ハッシュ

すべてのアルゴリズムは `HashResolver` インターフェースを実装します：
```go
type HashResolver interface {
    Check(h string, password string) bool
}
```

## ワークフロー

### 認証フロー

1. **ユーザーが保護されたリソースにアクセス**
   - Traefik がリクエストをインターセプト
   - Stargate エンドポイント `/_auth` に転送

2. **Stargate が認証を検証**
   - まず `Stargate-Password` ヘッダーを確認（API 認証）
   - ヘッダー認証が失敗した場合、`stargate_session_id` Cookie を確認（Web 認証）

3. **認証成功**
   - `X-Forwarded-User` ヘッダー（または設定されたユーザーヘッダー名）に "authenticated" を設定
   - 200 OK を返す
   - Traefik がリクエストの継続を許可

4. **認証失敗**
   - HTML リクエスト: ログインページにリダイレクト (`/_login?callback=<originalURL>`)
   - API リクエスト（JSON/XML）: 401 Unauthorized を返す

### ログインフロー

1. **ユーザーがログインページにアクセス**
   - `GET /_login?callback=<url>`
   - 既にログインしている場合、セッション交換エンドポイントにリダイレクト
   - ドメインが異なる場合、コールバックを Cookie (`stargate_callback`) に保存

2. **ログインフォームの送信**
   - `POST /_login` にパスワード
   - パスワードを検証
   - セッションを作成し、Cookie を設定
   - **コールバック取得の優先順位**：
     1. Cookie から（以前に設定されている場合）
     2. フォームデータから
     3. クエリパラメータから
     4. 上記のいずれもなく、元のドメインが認証サービスドメインと異なる場合、元のドメインをコールバックとして使用

3. **セッション交換**
   - コールバックが存在する場合、`{callback}/_session_exchange?id=<session_id>` にリダイレクト
   - `GET /_session_exchange?id=<session_id>`
   - セッション Cookie を設定（`COOKIE_DOMAIN` が設定されている場合、指定されたドメインに設定）
   - ルートパス `/` にリダイレクト

## セキュリティの考慮事項

### セッションセキュリティ

- Cookie は XSS 攻撃を防ぐために `HttpOnly` フラグを使用
- Cookie は CSRF 攻撃を防ぐために `SameSite=Lax` を使用
- Cookie のパスは `/` に設定され、ドメイン全体で使用可能
- セッションの有効期限: 24 時間 (`config.SessionExpiration`)
- カスタム Cookie ドメインをサポート（クロスドメインシナリオ用）
- セッション ID は UUID を使用して生成され、一意性とセキュリティを保証

### パスワードセキュリティ

- 複数の暗号化アルゴリズムをサポート（bcrypt または sha512 の使用を推奨）
- パスワード設定は環境変数経由で渡され、コードに保存されない
- 検証時にパスワードを正規化（スペースを削除し、大文字に変換）

### リクエストセキュリティ

- 認証チェックエンドポイントは 2 つの認証方法をサポート：
  - ヘッダー認証 (`Stargate-Password`): API リクエスト用
  - Cookie 認証: Web リクエスト用
- HTML リクエストと API リクエストを区別し、適切な応答を返す

## 拡張性

### 新しいパスワードアルゴリズムの追加

1. `internal/secure/` に新しいアルゴリズム実装を作成
2. `HashResolver` インターフェースを実装
3. `config/validation.go` にアルゴリズムを登録

### 新しい言語の追加

1. `internal/i18n/i18n.go` に言語定数を追加
2. 翻訳マッピングを追加
3. 設定に言語オプションを追加

### ログインページのカスタマイズ

テンプレートファイル `internal/web/templates/login.html` を変更します。

## パフォーマンスの最適化

- Fiber フレームワークを使用、fasthttp ベースで優れたパフォーマンス
- セッションはメモリに保存され、高速アクセス
- 静的リソースは Fiber の静的ファイルサービス経由で提供
- デバッグモードをサポート、本番環境で無効化可能

## デプロイメントアーキテクチャ

### Docker デプロイメント

- イメージサイズを削減するためのマルチステージビルド
- ビルドステージとして `golang:1.25-alpine` を使用
- セキュリティリスクを最小化するため、実行ステージとして `scratch` ベースイメージを使用
- テンプレートファイルを `src/internal/web/templates` からイメージ内の `/app/web/templates` にコピー
- 依存関係のダウンロードを加速するため、中国ミラーソース (`GOPROXY=https://goproxy.cn`) を使用
- バイナリサイズを削減するため、コンパイル時に `-ldflags "-s -w"` を使用
- アプリケーションは自動的にテンプレートパスを見つける（ローカル開発用に `./internal/web/templates`、本番環境用に `./web/templates` をサポート）

### Traefik 統合

- Forward Auth ミドルウェア経由で統合
- HTTP と HTTPS をサポート
- 複数のドメインとパスルールをサポート

## ログと監視

- ログには Logrus を使用
- デバッグモードをサポート（DEBUG=true）
- すべての重要な操作がログに記録される
- 監視用のヘルスチェックエンドポイントが利用可能

## テスト

- ユニットテストは主要機能をカバー
- テストファイルは各パッケージの `*_test.go` ファイルに配置
- アサーションに `testza` を使用
- テストカバレッジには以下が含まれます：
  - 認証ロジック (`internal/auth/auth_test.go`)
  - 設定検証 (`internal/config/config_test.go`)
  - パスワード暗号化アルゴリズム (`internal/secure/secure_test.go`)
  - HTTP ハンドラー (`internal/handlers/handlers_test.go`)

## 今後の改善

- [ ] より多くのパスワード暗号化アルゴリズムをサポート
- [ ] OAuth2/OpenID Connect をサポート
- [ ] マルチユーザーとロール管理をサポート
- [ ] 管理インターフェースを追加
- [ ] 外部セッションストレージ（Redis など）をサポート
- [ ] Prometheus メトリクスのエクスポートを追加
- [ ] 設定ファイル（YAML/JSON）をサポート
