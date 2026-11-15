# ドキュメントインデックス

> **部分翻訳**: 日本語のガイドは一部のみを翻訳したもので、現在の変更がすべて反映されているわけではありません。[英語版のドキュメント](../enUS/README.md)が完全な正式版です。

Stargate Forward Auth Service のドキュメントへようこそ。

## 🌐 多言語ドキュメント

- [English](../enUS/README.md) | [中文](../zhCN/README.md) | [Français](../frFR/README.md) | [Italiano](../itIT/README.md) | [日本語](README.md) | [Deutsch](../deDE/README.md) | [한국어](../koKR/README.md)

## 📚 ドキュメント一覧

### コアドキュメント

- **[README.md](../../README.jaJP.md)** - プロジェクト概要とクイックスタートガイド
- **[ARCHITECTURE.md](ARCHITECTURE.md)** - 技術アーキテクチャと設計決定

### 詳細ドキュメント

- **[API.md](API.md)** - 完全な API エンドポイントドキュメント
  - 認証チェックエンドポイント
  - ログインとログアウトエンドポイント
  - セッション交換エンドポイント
  - ヘルスチェックエンドポイント
  - エラーレスポンス形式
  - 認証フロー例

- **[CONFIG.md](CONFIG.md)** - 設定リファレンス
  - 設定方法
  - 必須設定項目
  - オプション設定項目
  - パスワード設定の詳細
  - 設定例
  - 設定のベストプラクティス

- **[DEPLOYMENT.md](DEPLOYMENT.md)** - デプロイメントガイド
  - Docker デプロイメント
  - Docker Compose デプロイメント
  - Traefik 統合
  - 本番環境デプロイメント
  - 監視とメンテナンス
  - トラブルシューティング

## 🚀 クイックナビゲーション

### はじめに

1. [README.jaJP.md](../../README.jaJP.md) を読んでプロジェクトを理解する
2. [クイックスタート](../../README.jaJP.md#-クイックスタート) セクションを確認する
3. [設定](../../README.jaJP.md#%EF%B8%8F-基本設定) を参照してサービスを設定する

### 開発者

1. [ARCHITECTURE.md](ARCHITECTURE.md) を読んでアーキテクチャを理解する
2. [API.md](API.md) を確認して API インターフェースを理解する
3. [開発ガイド](CONTRIBUTING.md) を参照して開発する

### 運用

1. [DEPLOYMENT.md](DEPLOYMENT.md) を読んでデプロイメント方法を理解する
2. [CONFIG.md](CONFIG.md) を確認して設定オプションを理解する
3. [トラブルシューティング](DEPLOYMENT.md#トラブルシューティング) を参照して問題を解決する

## 📖 ドキュメント構造

このディレクトリにある部分翻訳:

- [API.md](API.md)
- [ARCHITECTURE.md](ARCHITECTURE.md)
- [CONFIG.md](CONFIG.md)
- [CONTRIBUTING.md](CONTRIBUTING.md)
- [DEPLOYMENT.md](DEPLOYMENT.md)
- [MIGRATION_V1.md](MIGRATION_V1.md)
- [SECURITY.md](SECURITY.md)

英語版のみ:

- [DIAGNOSTICS.md](../enUS/DIAGNOSTICS.md) - Deployment diagnostics
- [AUTHENTICATION_UI.md](../enUS/AUTHENTICATION_UI.md) - Authentication pages
- [DINGTALK_USERID_ONLY.md](../enUS/DINGTALK_USERID_ONLY.md) - Opt-in userid-only work-IM login
- [PERFORMANCE.md](../enUS/PERFORMANCE.md) - Redis rate-limit measurements
- [VERIFIED_EMAIL.md](../enUS/VERIFIED_EMAIL.md) - Verified email assertions

## 🔍 トピック別検索

### 設定関連

- 環境変数設定：[CONFIG.md](CONFIG.md)
- パスワード設定：[CONFIG.md#パスワード設定](CONFIG.md#パスワード設定)
- 設定例：[CONFIG.md#設定例](CONFIG.md#設定例)

### API 関連

- API エンドポイント一覧：[API.md](API.md)
- 認証フロー：[API.md#認証フローの例](API.md#認証フローの例)
- エラー処理：[API.md#エラーレスポンス形式](API.md#エラーレスポンス形式)

### デプロイメント関連

- Docker デプロイメント：[DEPLOYMENT.md#docker-デプロイメント](DEPLOYMENT.md#docker-デプロイメント)
- Traefik 統合：[DEPLOYMENT.md#traefik-統合](DEPLOYMENT.md#traefik-統合)
- 本番環境：[DEPLOYMENT.md#本番環境デプロイメント](DEPLOYMENT.md#本番環境デプロイメント)

### アーキテクチャ関連

- 技術スタック：[ARCHITECTURE.md#技術スタック](ARCHITECTURE.md#技術スタック)
- プロジェクト構造：[ARCHITECTURE.md#プロジェクト構造](ARCHITECTURE.md#プロジェクト構造)
- コアコンポーネント：[ARCHITECTURE.md#主要コンポーネント](ARCHITECTURE.md#主要コンポーネント)

## 💡 使用推奨事項

1. **初回ユーザー**：[README.jaJP.md](../../README.jaJP.md) から始めて、クイックスタートガイドに従う
2. **サービス設定**：[CONFIG.md](CONFIG.md) を参照してすべての設定オプションを理解する
3. **Traefik 統合**：[DEPLOYMENT.md](DEPLOYMENT.md) の Traefik 統合セクションを確認する
4. **拡張機能の開発**：[ARCHITECTURE.md](ARCHITECTURE.md) を読んでアーキテクチャ設計を理解する
5. **トラブルシューティング**：[DEPLOYMENT.md#トラブルシューティング](DEPLOYMENT.md#トラブルシューティング) を確認する

## 📝 ドキュメント更新

ドキュメントはプロジェクトの進化に伴って継続的に更新されます。エラーを見つけたり、追加が必要な場合は、Issue または Pull Request を送信してください。

## 🤝 貢献

ドキュメントの改善を歓迎します：

1. エラーや改善が必要な領域を見つける
2. 問題を説明する Issue を送信する
3. または直接 Pull Request を送信する
