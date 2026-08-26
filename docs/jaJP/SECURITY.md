# セキュリティドキュメント

> 🌐 **Language / 语言**: [English](../enUS/SECURITY.md) | [中文](../zhCN/SECURITY.md) | [Français](../frFR/SECURITY.md) | [Italiano](../itIT/SECURITY.md) | [日本語](SECURITY.md) | [Deutsch](../deDE/SECURITY.md) | [한국어](../koKR/SECURITY.md)

このドキュメントは、Stargate のセキュリティ機能、セキュリティ設定、ベストプラクティスについて説明します。

> ⚠️ **注意**: このドキュメントは翻訳中です。完全なバージョンについては、[英語版](../enUS/SECURITY.md)を参照してください。

## 実装されたセキュリティ機能

1. **Forward Auth 保護**: バックエンドサービスを保護するための集中認証レイヤー
2. **複数のパスワードアルゴリズム**: bcrypt とローカル開発用 plaintext をサポートし、MD5 とソルトなし SHA-512 は拒否
3. **安全なセッション管理**: ドメインと有効期限が設定可能な Cookie ベースのセッション
4. **サービス統合セキュリティ**: mTLS または HMAC を使用した Warden および Herald サービスとの安全な通信
5. **セッション共有セキュリティ**: 安全なクロスドメインセッション交換メカニズム
6. **入力検証**: すべての入力パラメータの厳密な検証
7. **エラー処理**: HTTP 応答は限定されたエラーを返し、`DEBUG` はログ出力だけを制御
8. **セキュリティレスポンスヘッダー**: セキュリティ関連の HTTP レスポンスヘッダーを自動的に追加
9. **HTTPS 強制**: 本番環境は HTTPS を使用する必要があります
10. **OTP 統合**: OTP/検証コード認証のための Herald との安全な統合

詳細については、[英語版](../enUS/SECURITY.md)を参照してください。

## セキュリティ関連の設定

- HTTPS では `COOKIE_SECURE=true` を使用し、`TRUSTED_PROXIES` を既知のプロキシアドレスに限定します。
- `CALLBACK_ALLOWED_HOSTS` を設定し、クロスドメインチケットには 32 文字以上の `SESSION_EXCHANGE_SECRET` を使用します。
- Warden と Herald では、HMAC のキー ID とシークレット、mTLS のクライアント証明書と鍵をそれぞれ組で設定します。
- 同期済みの変数一覧は [CONFIG.md](CONFIG.md) を参照してください。

## 脆弱性の報告

セキュリティの脆弱性を発見した場合は、以下を通じて報告してください：

1. **GitHub Security Advisory**（推奨）
   - リポジトリの [Security タブ](https://github.com/soulteary/stargate/security) に移動
   - "Report a vulnerability" をクリック
   - セキュリティアドバイザリフォームに記入

2. **メール**（GitHub Security Advisory が利用できない場合）
   - プロジェクトのメンテナーにメールを送信
   - 脆弱性の詳細な説明を含める

**公開の GitHub Issues を通じてセキュリティの脆弱性を報告しないでください。**
