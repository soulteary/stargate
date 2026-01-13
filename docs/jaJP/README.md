# ドキュメントインデックス

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
2. [クイックスタート](../../README.jaJP.md#クイックスタート) セクションを確認する
3. [設定](../../README.jaJP.md#設定) を参照してサービスを設定する

### 開発者

1. [ARCHITECTURE.md](ARCHITECTURE.md) を読んでアーキテクチャを理解する
2. [API.md](API.md) を確認して API インターフェースを理解する
3. [開発ガイド](../../README.jaJP.md#開発ガイド) を参照して開発する

### 運用

1. [DEPLOYMENT.md](DEPLOYMENT.md) を読んでデプロイメント方法を理解する
2. [CONFIG.md](CONFIG.md) を確認して設定オプションを理解する
3. [トラブルシューティング](DEPLOYMENT.md#トラブルシューティング) を参照して問題を解決する

## 📖 ドキュメント構造

```
codes/
├── README.md              # プロジェクトメインドキュメント（英語）
├── README.zhCN.md         # プロジェクトメインドキュメント（中国語）
├── README.frFR.md         # プロジェクトメインドキュメント（フランス語）
├── README.itIT.md         # プロジェクトメインドキュメント（イタリア語）
├── README.jaJP.md         # プロジェクトメインドキュメント（日本語）
├── README.deDE.md         # プロジェクトメインドキュメント（ドイツ語）
├── README.koKR.md         # プロジェクトメインドキュメント（韓国語）
├── docs/
│   ├── enUS/
│   │   ├── README.md       # ドキュメントインデックス（英語）
│   │   ├── ARCHITECTURE.md # アーキテクチャドキュメント（英語）
│   │   ├── API.md          # API ドキュメント（英語）
│   │   ├── CONFIG.md       # 設定リファレンス（英語）
│   │   └── DEPLOYMENT.md   # デプロイメントガイド（英語）
│   ├── zhCN/
│   │   ├── README.md       # ドキュメントインデックス（中国語）
│   │   ├── ARCHITECTURE.md # アーキテクチャドキュメント（中国語）
│   │   ├── API.md          # API ドキュメント（中国語）
│   │   ├── CONFIG.md       # 設定リファレンス（中国語）
│   │   └── DEPLOYMENT.md   # デプロイメントガイド（中国語）
│   └── jaJP/
│       ├── README.md       # ドキュメントインデックス（日本語、このファイル）
│       ├── ARCHITECTURE.md # アーキテクチャドキュメント（日本語）
│       ├── API.md          # API ドキュメント（日本語）
│       ├── CONFIG.md       # 設定リファレンス（日本語）
│       └── DEPLOYMENT.md   # デプロイメントガイド（日本語）
└── ...
```

## 🔍 トピック別検索

### 設定関連

- 環境変数設定：[CONFIG.md](CONFIG.md)
- パスワード設定：[CONFIG.md#パスワード設定](CONFIG.md#パスワード設定)
- 設定例：[CONFIG.md#設定例](CONFIG.md#設定例)

### API 関連

- API エンドポイント一覧：[API.md](API.md)
- 認証フロー：[API.md#認証フロー例](API.md#認証フロー例)
- エラー処理：[API.md#エラーレスポンス形式](API.md#エラーレスポンス形式)

### デプロイメント関連

- Docker デプロイメント：[DEPLOYMENT.md#docker-デプロイメント](DEPLOYMENT.md#docker-デプロイメント)
- Traefik 統合：[DEPLOYMENT.md#traefik-統合](DEPLOYMENT.md#traefik-統合)
- 本番環境：[DEPLOYMENT.md#本番環境デプロイメント](DEPLOYMENT.md#本番環境デプロイメント)

### アーキテクチャ関連

- 技術スタック：[ARCHITECTURE.md#技術スタック](ARCHITECTURE.md#技術スタック)
- プロジェクト構造：[ARCHITECTURE.md#プロジェクト構造](ARCHITECTURE.md#プロジェクト構造)
- コアコンポーネント：[ARCHITECTURE.md#コアコンポーネント](ARCHITECTURE.md#コアコンポーネント)

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
