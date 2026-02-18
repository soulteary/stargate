# Stargate - Forward Auth Service

[![License](https://img.shields.io/badge/license-Apache%202.0-blue.svg)](LICENSE)
[![Go Version](https://img.shields.io/badge/go-1.26+-blue.svg)](https://golang.org)
[![codecov](https://codecov.io/gh/soulteary/stargate/branch/main/graph/badge.svg)](https://codecov.io/gh/soulteary/stargate)
[![Go Report Card](https://goreportcard.com/badge/github.com/soulteary/stargate)](https://goreportcard.com/report/github.com/soulteary/stargate)

> **🚀 セキュアなマイクロサービスへのゲートウェイ**

![Stargate](.github/assets/banner.jpg)

Stargate は、本番環境に対応した軽量な Forward Auth サービスで、インフラ全体の**単一認証ポイント**として設計されています。Go で構築され、パフォーマンスに最適化されており、Stargate は Traefik やその他のリバースプロキシとシームレスに統合し、バックエンドサービスを保護します—**アプリケーションに認証コードを一行も書く必要はありません**。

## 🌐 多言語ドキュメント

- [English](README.md) | [中文](README.zhCN.md) | [Français](README.frFR.md) | [Italiano](README.itIT.md) | [日本語](README.jaJP.md) | [Deutsch](README.deDE.md) | [한국어](README.koKR.md)

![プレビュー](.github/assets/preview.png)

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
- **エンタープライズ認証**：Warden（ユーザーホワイトリスト）と Herald（OTP/検証コード）との統合による本番グレードの認証

## ✨ 機能

### 🔐 エンタープライズグレードのセキュリティ

- **複数のパスワード暗号化アルゴリズム**：plaintext（テスト用）、bcrypt、MD5、SHA512 などから選択
- **安全なセッション管理**：カスタマイズ可能なドメインと有効期限を持つ Cookie ベースのセッション
- **柔軟な認証**：パスワードベースとセッションベースの両方の認証をサポート
- **OTP/検証コードサポート**：Heraldサービスとの統合によるSMS/Email検証コード
- **ユーザーホワイトリスト管理**：Wardenサービスとの統合によるユーザーアクセス制御

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

## 📋 目次

- [クイックスタート](#-クイックスタート)
- [ドキュメント](#-ドキュメント)
- [基本設定](#-基本設定)
- [オプションサービス統合](#-オプションサービス統合)
- [本番環境チェックリスト](#-本番環境チェックリスト)
- [ライセンス](#-ライセンス)

## 🚀 クイックスタート

**2 分以内**に Stargate を起動して実行できます！

### Docker Compose の使用（推奨）

**ステップ 1：** リポジトリをクローン
```bash
git clone <repository-url>
cd stargate
```

**ステップ 2：** 認証を設定（`docker-compose.yml` を編集）

**オプション A: パスワード認証（シンプル）**
```yaml
services:
  stargate:
    environment:
      - AUTH_HOST=auth.example.com
      - PASSWORDS=plaintext:yourpassword1|yourpassword2
```

**オプション B: Warden + Herald OTP認証（本番環境）**
```yaml
services:
  stargate:
    environment:
      - AUTH_HOST=auth.example.com
      - WARDEN_ENABLED=true
      - WARDEN_URL=http://warden:8080
      - WARDEN_API_KEY=your-warden-api-key
      - HERALD_ENABLED=true
      - HERALD_URL=http://herald:8080
      - HERALD_HMAC_SECRET=your-herald-hmac-secret
```

**ステップ 3：** サービスを起動
```bash
docker-compose up -d
```

**これで完了です！** 認証サービスが実行されています。🎉

### ローカル開発

ローカル開発では、Go 1.26+ がインストールされていることを確認し、次を実行します：

```bash
chmod +x start-local.sh
./start-local.sh
```

ログインページにアクセス：`http://localhost:8080/_login?callback=localhost`

## 📚 ドキュメント

Stargate を最大限に活用するための包括的なドキュメントが利用可能です：

### コアドキュメント

- 📐 **[アーキテクチャドキュメント](docs/jaJP/ARCHITECTURE.md)** - 技術アーキテクチャと設計決定の詳細
- 🔌 **[API ドキュメント](docs/jaJP/API.md)** - 例付きの完全な API エンドポイントリファレンス
- ⚙️ **[設定リファレンス](docs/jaJP/CONFIG.md)** - 詳細な設定オプションとベストプラクティス
- 🚀 **[デプロイメントガイド](docs/jaJP/DEPLOYMENT.md)** - 本番環境デプロイメント戦略と推奨事項

### クイックリファレンス

- **API エンドポイント**：`GET /_auth`（認証チェック）、`GET /_login`（ログインページ）、`POST /_login`（ログイン）、`GET /_logout`（ログアウト）、`GET /_session_exchange`（クロスドメイン）、`GET /health`（ヘルスチェック）
- **デプロイメント**：クイックスタートには Docker Compose を推奨。本番環境デプロイメントについては [DEPLOYMENT.md](docs/jaJP/DEPLOYMENT.md) を参照。
- **開発**：開発関連のドキュメントについては [ARCHITECTURE.md](docs/jaJP/ARCHITECTURE.md) を参照

## ⚙️ 基本設定

Stargate は環境変数を使用して設定します。以下は最も一般的な設定です：

### 必須設定

- **`AUTH_HOST`**：認証サービスのホスト名（例：`auth.example.com`）
- **`PASSWORDS`**：パスワード設定、形式：`algorithm:password1|password2|password3`

### 一般的な設定例

```bash
# シンプルなパスワード認証
AUTH_HOST=auth.example.com
PASSWORDS=plaintext:test123|admin456

# BCrypt ハッシュを使用
PASSWORDS=bcrypt:$2a$10$N9qo8uLOickgx2ZMRZoMyeIjZAgcfl7p92ldGxad68LJZdL17lhWy

# クロスドメインセッション共有
COOKIE_DOMAIN=.example.com

# ログインページをカスタマイズ
LOGIN_PAGE_TITLE=私の認証サービス
LANGUAGE=ja  # または 'en'
```

**サポートされているパスワードアルゴリズム：** `plaintext`（テストのみ）、`bcrypt`、`md5`、`sha512`

**完全な設定リファレンスについては、[docs/jaJP/CONFIG.md](docs/jaJP/CONFIG.md) を参照してください**

## 🔗 オプションサービス統合

Stargateは完全に独立して使用できます。また、以下のサービスとオプションで統合して機能を拡張することもできます：

### Warden統合（ユーザーホワイトリスト）

ユーザーホワイトリスト管理とユーザー情報を提供します。有効にすると、StargateはWardenに問い合わせて、ユーザーが許可リストに含まれているかどうかを確認します。

```bash
WARDEN_ENABLED=true
WARDEN_URL=http://warden:8080
WARDEN_API_KEY=your-api-key
```

### Herald統合（OTP/検証コード）

OTP/検証コードサービスを提供します。有効にすると、StargateはHeraldを呼び出して検証コード（SMS/Email）を作成、送信、検証します。

```bash
HERALD_ENABLED=true
HERALD_URL=http://herald:8080
HERALD_HMAC_SECRET=your-hmac-secret  # 本番環境
# または
HERALD_API_KEY=your-api-key  # 開発環境
```

**注意**：WardenとHeraldの統合はオプションです。Stargateはパスワード認証で独立して使用でき、これらの統合機能をオプションで有効にすることもできます。

**完全な統合ガイドについては、[docs/jaJP/ARCHITECTURE.md](docs/jaJP/ARCHITECTURE.md) を参照してください**

## ⚠️ 本番環境チェックリスト

本番環境にデプロイする前に：

- ✅ 強力なパスワードアルゴリズムを使用（`bcrypt` または `sha512`、`plaintext` を避ける）
- ✅ Traefik またはリバースプロキシ経由で HTTPS を有効化
- ✅ サブドメイン間で適切なセッション管理のために `COOKIE_DOMAIN` を設定
- ✅ 高度な機能が必要な場合、OTP認証のために Warden + Herald をオプションで統合
- ✅ Stargate ↔ Herald/Warden 通信に HMAC 署名または mTLS を使用
- ✅ 適切なログ記録と監視を設定
- ✅ セキュリティパッチのために Stargate を最新バージョンに保つ

## 🎯 設計原則

Stargateは独立して使用できるように設計されています：

- **独立使用**：Stargateはパスワード認証モードを使用して独立して実行でき、外部依存関係は不要です
- **オプション統合**：Warden（ユーザーホワイトリスト）とHerald（OTP/検証コード）とオプションで統合できます
- **高性能**：forwardAuthメインパスはセッションのみを検証し、高速な応答を保証します
- **柔軟性**：複数の認証モードをサポートし、ニーズに応じて選択できます

## 📝 ライセンス

このプロジェクトは Apache License 2.0 の下でライセンスされています。詳細については [LICENSE](LICENSE) ファイルを参照してください。

## 🤝 貢献

貢献を歓迎します！以下を含みます：
- 🐛 バグレポート
- 💡 機能の提案
- 📝 ドキュメントの改善
- 🔧 コードの貢献

Issue を開くか、Pull Request を送信してください。
