# 設定リファレンス

このドキュメントは、Stargate のすべての設定オプションを詳しく説明します。

## 目次

- [設定方法](#設定方法)
- [必須設定](#必須設定)
- [オプション設定](#オプション設定)
- [パスワード設定](#パスワード設定)
- [設定例](#設定例)

## 設定方法

Stargate は環境変数経由で設定されます。すべての設定項目は環境変数で定義され、設定ファイルは必要ありません。

### 環境変数の設定

**Linux/macOS:**

```bash
export AUTH_HOST=auth.example.com
export PASSWORDS=plaintext:yourpassword
```

**Docker:**

```bash
docker run -e AUTH_HOST=auth.example.com -e PASSWORDS=plaintext:yourpassword stargate:latest
```

**Docker Compose:**

```yaml
services:
  stargate:
    environment:
      - AUTH_HOST=auth.example.com
      - PASSWORDS=plaintext:yourpassword
```

## 必須設定

以下の設定項目は必須です。これらを設定しないと、サービスが起動しません。

### `AUTH_HOST`

認証サービスのホスト名。

| 属性 | 値 |
|------|-----|
| **型** | String |
| **必須** | はい |
| **デフォルト** | なし |
| **例** | `auth.example.com` |

**説明:**

- ログインコールバック URL の構築に使用
- 通常、Stargate サービスのホスト名に設定
- ワイルドカード `*` をサポート（本番環境では推奨されません）

**例:**

```bash
AUTH_HOST=auth.example.com
```

### `PASSWORDS`

パスワード設定。パスワード暗号化アルゴリズムとパスワードリストを指定します。

| 属性 | 値 |
|------|-----|
| **型** | String |
| **必須** | はい |
| **デフォルト** | なし |
| **形式** | `algorithm:password1|password2|password3` |

**説明:**

- 形式: `algorithm:password1|password2|password3`
- 複数のパスワードをサポート、`|` で区切る
- 検証を通過したパスワードはすべてログインを許可
- サポートされているアルゴリズムは [パスワード設定](#パスワード設定) セクションを参照

**例:**

```bash
# 単一のプレーンテキストパスワード
PASSWORDS=plaintext:test123

# 複数のプレーンテキストパスワード
PASSWORDS=plaintext:test123|admin456|user789

# BCrypt ハッシュ
PASSWORDS=bcrypt:$2a$10$N9qo8uLOickgx2ZMRZoMyeIjZAgcfl7p92ldGxad68LJZdL17lhWy

# SHA512 ハッシュ
PASSWORDS=sha512:5e884898da28047151d0e56f8dc6292773603d0d6aabbdd62a11ef721d1542d8
```

## オプション設定

以下の設定項目はオプションです。設定されていない場合、デフォルト値が使用されます。

### `DEBUG`

デバッグモードを有効化します。

| 属性 | 値 |
|------|-----|
| **型** | Boolean |
| **必須** | いいえ |
| **デフォルト** | `false` |
| **可能な値** | `true`, `false` |

**説明:**

- 有効にすると、ログレベルが `DEBUG` に設定されます
- より詳細なデバッグ情報を表示
- 本番環境では `false` に設定することを推奨

**例:**

```bash
DEBUG=true
```

### `LANGUAGE`

インターフェース言語。

| 属性 | 値 |
|------|-----|
| **型** | String |
| **必須** | いいえ |
| **デフォルト** | `en` |
| **可能な値** | `en`（英語）、`zh`（中国語）、`fr`（フランス語）、`it`（イタリア語）、`ja`（日本語）、`de`（ドイツ語）、`ko`（韓国語） |

**説明:**

- エラーメッセージとインターフェーステキストの言語に影響
- 大文字小文字を区別しない（`EN`、`en`、`En` はすべて機能）

**例:**

```bash
LANGUAGE=ja
```

### `LOGIN_PAGE_TITLE`

ログインページのタイトル。

| 属性 | 値 |
|------|-----|
| **型** | String |
| **必須** | いいえ |
| **デフォルト** | `Stargate - Login` |

**説明:**

- ログインページのタイトル位置に表示
- HTML タグをサポート（推奨されません）

**例:**

```bash
LOGIN_PAGE_TITLE=私の認証サービス
```

### `LOGIN_PAGE_FOOTER_TEXT`

ログインページのフッターテキスト。

| 属性 | 値 |
|------|-----|
| **型** | String |
| **必須** | いいえ |
| **デフォルト** | `Copyright © 2024 - Stargate` |

**説明:**

- ログインページのフッター位置に表示
- HTML タグをサポート（推奨されません）

**例:**

```bash
LOGIN_PAGE_FOOTER_TEXT=© 2024 私の会社
```

### `USER_HEADER_NAME`

認証成功後に設定されるユーザーヘッダー名。

| 属性 | 値 |
|------|-----|
| **型** | String |
| **必須** | いいえ |
| **デフォルト** | `X-Forwarded-User` |

**説明:**

- 認証成功後、Stargate はレスポンスでこのヘッダーを設定します
- ヘッダーの値は `authenticated`
- バックエンドサービスはこのヘッダーを介してユーザーが認証されているかどうかを判断できます
- 空でない文字列である必要があります

**例:**

```bash
USER_HEADER_NAME=X-Authenticated-User
```

### `COOKIE_DOMAIN`

Cookie ドメイン。クロスドメインセッション共有に使用されます。

| 属性 | 値 |
|------|-----|
| **型** | String |
| **必須** | いいえ |
| **デフォルト** | 空（設定されていない） |

**説明:**

- 設定されている場合、セッション Cookie は指定されたドメインに設定されます
- クロスサブドメインセッション共有をサポート
- 形式: `.example.com`（最初のドットに注意）
- 空に設定されている場合、Cookie は現在のドメインでのみ有効

**例:**

```bash
# すべてのサブドメイン *.example.com でセッション共有を許可
COOKIE_DOMAIN=.example.com
```

**クロスドメインセッション共有のシナリオ:**

以下のドメインを想定：
- `auth.example.com` - 認証サービス
- `app1.example.com` - アプリケーション 1
- `app2.example.com` - アプリケーション 2

`COOKIE_DOMAIN=.example.com` を設定した後：
1. ユーザーが `auth.example.com` にログイン
2. セッション Cookie が `.example.com` ドメインに設定される
3. ユーザーは `app1.example.com` と `app2.example.com` で同じセッションを使用できます

### `PORT`

サービスのリスニングポート（ローカル開発のみ）。

| 属性 | 値 |
|------|-----|
| **型** | String |
| **必須** | いいえ |
| **デフォルト** | `80` |

**説明:**

- ローカル開発環境専用
- Docker コンテナでは通常不要（デフォルトポート 80 を使用）
- 形式: ポート番号（例: `8080`）または `:port`（例: `:8080`）

**例:**

```bash
PORT=8080
```

## パスワード設定

Stargate は複数のパスワード暗号化アルゴリズムをサポートします。パスワード設定形式: `algorithm:password1|password2|password3`

### サポートされているアルゴリズム

#### `plaintext` - プレーンテキストパスワード

**説明:**

- プレーンテキストで保存、暗号化なし
- **テスト環境のみ**
- 本番環境では強く推奨されません

**例:**

```bash
PASSWORDS=plaintext:test123|admin456
```

#### `bcrypt` - BCrypt ハッシュ

**説明:**

- BCrypt アルゴリズムを使用してハッシュ化
- 高セキュリティ、本番環境で推奨
- パスワードは BCrypt ハッシュ値を使用する必要があります

**BCrypt ハッシュの生成:**

```bash
# Go を使用
go run -c 'golang.org/x/crypto/bcrypt' <<< 'password'

# オンラインツールまたはその他のツールを使用
```

**例:**

```bash
PASSWORDS=bcrypt:$2a$10$N9qo8uLOickgx2ZMRZoMyeIjZAgcfl7p92ldGxad68LJZdL17lhWy
```

#### `md5` - MD5 ハッシュ

**説明:**

- MD5 アルゴリズムを使用してハッシュ化
- セキュリティが低い、本番環境では推奨されません
- パスワードは MD5 ハッシュ値（32 文字の 16 進文字列）を使用する必要があります

**MD5 ハッシュの生成:**

```bash
# Linux/macOS
echo -n "password" | md5sum

# またはオンラインツールを使用
```

**例:**

```bash
PASSWORDS=md5:5e884898da28047151d0e56f8dc6292773603d0d6aabbdd62a11ef721d1542d8
```

#### `sha512` - SHA512 ハッシュ

**説明:**

- SHA512 アルゴリズムを使用してハッシュ化
- 高セキュリティ、本番環境で推奨
- パスワードは SHA512 ハッシュ値（128 文字の 16 進文字列）を使用する必要があります

**SHA512 ハッシュの生成:**

```bash
# Linux/macOS
echo -n "password" | shasum -a 512

# またはオンラインツールを使用
```

**例:**

```bash
PASSWORDS=sha512:5e884898da28047151d0e56f8dc6292773603d0d6aabbdd62a11ef721d1542d8
```

### パスワード検証のルール

1. **パスワードの正規化**: 検証前にスペースを削除し、大文字に変換
2. **複数パスワードのサポート**: 複数のパスワードを設定でき、検証を通過したパスワードはすべて受け入れられます
3. **アルゴリズムの一貫性**: すべてのパスワードは同じアルゴリズムを使用する必要があります

## 設定例

### 基本設定

```bash
# 必須設定
AUTH_HOST=auth.example.com
PASSWORDS=plaintext:test123

# オプション設定
DEBUG=false
LANGUAGE=en
```

### 本番環境設定

```bash
# 必須設定
AUTH_HOST=auth.example.com
PASSWORDS=bcrypt:$2a$10$N9qo8uLOickgx2ZMRZoMyeIjZAgcfl7p92ldGxad68LJZdL17lhWy

# オプション設定
DEBUG=false
LANGUAGE=ja
LOGIN_PAGE_TITLE=私の認証サービス
LOGIN_PAGE_FOOTER_TEXT=© 2024 私の会社
USER_HEADER_NAME=X-Forwarded-User
COOKIE_DOMAIN=.example.com
```

### Docker Compose 設定

```yaml
services:
  stargate:
    image: stargate:latest
    environment:
      # 必須設定
      - AUTH_HOST=auth.example.com
      - PASSWORDS=bcrypt:$2a$10$N9qo8uLOickgx2ZMRZoMyeIjZAgcfl7p92ldGxad68LJZdL17lhWy
      
      # オプション設定
      - DEBUG=false
      - LANGUAGE=ja
      - LOGIN_PAGE_TITLE=私の認証サービス
      - LOGIN_PAGE_FOOTER_TEXT=© 2024 私の会社
      - COOKIE_DOMAIN=.example.com
```

### ローカル開発設定

```bash
# 必須設定
AUTH_HOST=localhost
PASSWORDS=plaintext:test123|admin456

# オプション設定
DEBUG=true
LANGUAGE=ja
PORT=8080
```

## 設定の検証

Stargate は起動時にすべての設定項目を検証します：

1. **必須設定の確認**: 必須設定が設定されていない場合、サービスは起動に失敗し、エラーメッセージを表示します
2. **形式の検証**: パスワード設定形式が正しくない場合、起動に失敗します
3. **アルゴリズムの検証**: サポートされていないパスワードアルゴリズムは起動に失敗します
4. **値の検証**: 一部の設定項目には値の制限があります（例: `LANGUAGE`、`DEBUG`）

**エラーの例:**

```bash
# 必須設定が不足
Error: Configuration error: environment variable 'AUTH_HOST' is required but not set.

# パスワード形式が正しくない
Error: Configuration error: invalid value for environment variable 'PASSWORDS': 'invalid_format'

# サポートされていないアルゴリズム
Error: Configuration error: invalid value for environment variable 'PASSWORDS': 'unknown:password'
```

## 設定のベストプラクティス

1. **本番環境のセキュリティ**:
   - `bcrypt` または `sha512` アルゴリズムを使用し、`plaintext` を避ける
   - `DEBUG=false` に設定
   - 強力なパスワードを使用

2. **クロスドメインセッション**:
   - サブドメイン間でセッションを共有する必要がある場合、`COOKIE_DOMAIN` を設定
   - 形式: `.example.com`（最初のドットに注意）

3. **多言語サポート**:
   - ユーザーベースに応じて `LANGUAGE` を設定
   - `en`、`zh`、`fr`、`it`、`ja`、`de`、`ko` をサポート

4. **カスタムインターフェース**:
   - `LOGIN_PAGE_TITLE` と `LOGIN_PAGE_FOOTER_TEXT` を使用してログインページをカスタマイズ

5. **監視とデバッグ**:
   - 開発環境では `DEBUG=true` を設定して詳細なログを取得
   - 本番環境では `DEBUG=false` を設定してログ出力を削減
