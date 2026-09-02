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
7. **エラー処理**: エラーメッセージやその他のレスポンスデータはエンドポイントごとに異なり、`DEBUG` は一般的な非開示保証ではありません
8. **セキュリティレスポンスヘッダー**: セキュリティ関連の HTTP レスポンスヘッダーを自動的に追加
9. **HTTPS 強制**: 本番環境は HTTPS を使用する必要があります
10. **OTP 統合**: OTP/検証コード認証のための Herald との安全な統合

## エラーレスポンスとデバッグ用検証コード

Stargate に `MODE` 設定はありません。エラーレスポンスはエンドポイントごとに定義されます。
運用上の詳細は通常ログに記録されますが、`DEBUG` をレスポンスの無害化や非開示に関する
一般的な保証として扱わないでください。

Stargate が `DEBUG=true` で動作し、設定先の Herald サービスがテストモード
（`HERALD_TEST_MODE`）で空でない `debug_code` を返すと、Stargate は成功した
`POST /_send_verify_code` レスポンスにテスト用検証コードを `debug_code` として含めます。
同梱のログインページはこのコードを表示し、検証コード欄へ自動入力します。

この組み合わせは、認証資格情報に相当する検証コードをリクエスト元のクライアントへ開示します。
ローカル開発またはテストでのみ使用し、本番環境では絶対に使用しないでください。本番環境では
Stargate を `DEBUG=false` にし、Herald の `HERALD_TEST_MODE` を無効にしてください。

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

## デプロイの信頼境界

Forward Auth が保護するのは、設定済みリバースプロキシを実際に通過するリクエストだけです。バックエンドのポートを直接公開せず、プロキシのネットワークからのみ接続を許可してください。クライアントが指定した ID・資格情報ヘッダーは転送前に削除してください。
