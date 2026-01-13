# API ドキュメント

このドキュメントは、Stargate Forward Auth サービスのすべての API エンドポイントを詳しく説明します。

## 目次

- [認証チェックエンドポイント](#認証チェックエンドポイント)
- [ログインエンドポイント](#ログインエンドポイント)
- [ログアウトエンドポイント](#ログアウトエンドポイント)
- [セッション交換エンドポイント](#セッション交換エンドポイント)
- [ヘルスチェックエンドポイント](#ヘルスチェックエンドポイント)
- [ルートエンドポイント](#ルートエンドポイント)

## 認証チェックエンドポイント

### `GET /_auth`

Traefik Forward Auth の主要な認証チェックエンドポイント。このエンドポイントは Stargate の主要機能で、ユーザーが認証されているかどうかを確認するために使用されます。

#### 認証方法

Stargate は、次の優先順位で確認される 2 つの認証方法をサポートします：

1. **ヘッダー認証**（API リクエスト）
   - リクエストヘッダー: `Stargate-Password: <password>`
   - API リクエスト、自動化スクリプトなどに適しています

2. **Cookie 認証**（Web リクエスト）
   - Cookie: `stargate_session_id=<session_id>`
   - ブラウザ経由でアクセスする Web アプリケーションに適しています

#### リクエストヘッダー

| ヘッダー | 型 | 必須 | 説明 |
|---------|-----|------|------|
| `Stargate-Password` | String | いいえ | API リクエスト用のパスワード認証 |
| `Cookie` | String | いいえ | `stargate_session_id` を含むセッション Cookie |
| `Accept` | String | いいえ | リクエストタイプ（HTML/API）を決定するために使用 |

#### レスポンス

**成功レスポンス（200 OK）**

認証が成功すると、Stargate はユーザー情報ヘッダーを設定し、ステータスコード 200 を返します：

```
HTTP/1.1 200 OK
X-Forwarded-User: authenticated
```

ユーザーヘッダー名は、環境変数 `USER_HEADER_NAME` で設定できます（デフォルト: `X-Forwarded-User`）。

**失敗レスポンス**

| ステータスコード | 説明 | レスポンス本文 |
|----------------|------|----------------|
| `401 Unauthorized` | 認証失敗 | エラーメッセージ（API リクエストの場合は JSON 形式）またはログインページへのリダイレクト（HTML リクエスト） |
| `500 Internal Server Error` | サーバーエラー | エラーメッセージ |

#### リクエストタイプの処理

- **HTML リクエスト**: 認証失敗時に `/_login?callback=<originalURL>` にリダイレクト
- **API リクエスト**（JSON/XML）: 認証失敗時に 401 エラーレスポンスを返す

#### 例

**ヘッダー認証の使用（API リクエスト）**

```bash
curl -H "Stargate-Password: yourpassword" \
     http://auth.example.com/_auth
```

**Cookie 認証の使用（Web リクエスト）**

```bash
curl -H "Cookie: stargate_session_id=<session_id>" \
     http://auth.example.com/_auth
```

## ログインエンドポイント

### `GET /_login`

ログインページを表示します。

#### クエリパラメータ

| パラメータ | 型 | 必須 | 説明 |
|-----------|-----|------|------|
| `callback` | String | いいえ | ログイン成功後のコールバック URL（通常は元のリクエストのドメイン） |

#### 動作

- ユーザーが既にログインしている場合、セッション交換エンドポイントに自動的にリダイレクト
- ユーザーがログインしていない場合、ログインページを表示
- URL に `callback` パラメータが含まれ、ドメインが異なる場合、コールバックは Cookie `stargate_callback` に保存されます（10 分で期限切れ）

#### コールバック取得の優先順位

1. **クエリパラメータから**: URL 内の `callback` パラメータ（最高優先度）
2. **Cookie から**: クエリパラメータに存在しない場合、Cookie `stargate_callback` から取得

#### レスポンス

**200 OK** - ログインページの HTML を返す

ページには以下が含まれます：
- ログインフォーム
- カスタマイズ可能なタイトル（`LOGIN_PAGE_TITLE`）
- カスタマイズ可能なフッターテキスト（`LOGIN_PAGE_FOOTER_TEXT`）

#### 例

```bash
# ログインページにアクセス
curl http://auth.example.com/_login?callback=app.example.com
```

### `POST /_login`

ログインリクエストを処理し、パスワードを検証してセッションを作成します。

#### リクエスト本文

フォームデータ（`application/x-www-form-urlencoded`）：

| フィールド | 型 | 必須 | 説明 |
|-----------|-----|------|------|
| `password` | String | はい | ユーザーパスワード |
| `callback` | String | いいえ | ログイン成功後のコールバック URL |

#### コールバック取得の優先順位

ログイン処理は、次の優先順位でコールバックを取得します：

1. **Cookie から**: 以前にログインページにアクセスした際にドメインが異なっていた場合、コールバックは Cookie `stargate_callback` に保存されています
2. **フォームデータから**: POST リクエストのフォームデータ内の `callback` フィールド
3. **クエリパラメータから**: URL のクエリパラメータ内の `callback`
4. **自動推論**: 上記のいずれも存在せず、元のドメイン（`X-Forwarded-Host`）が認証サービスドメインと異なる場合、元のドメインをコールバックとして使用

#### レスポンス

**成功レスポンス（200 OK）**

レスポンスは、コールバックの有無とリクエストタイプによって異なります：

1. **コールバックあり**:
   - `{callback}/_session_exchange?id={session_id}` にリダイレクト
   - ステータスコード: `302 Found`

2. **コールバックなし**:
   - **HTML リクエスト**: meta refresh を含む HTML ページを返し、元のドメインに自動的にリダイレクト
   - **API リクエスト**: JSON レスポンスを返す
     ```json
     {
       "success": true,
       "message": "Login successful",
       "session_id": "<session_id>"
     }
     ```

**失敗レスポンス**

| ステータスコード | 説明 | レスポンス本文 |
|----------------|------|----------------|
| `401 Unauthorized` | パスワードが間違っている | Accept ヘッダーに応じて JSON/XML/テキスト形式のエラーメッセージ |
| `500 Internal Server Error` | サーバーエラー | エラーメッセージ |

#### 例

```bash
# ログインフォームを送信（コールバックあり）
curl -X POST \
     -d "password=yourpassword&callback=app.example.com" \
     -c cookies.txt \
     http://auth.example.com/_login

# ログインフォームを送信（コールバックなし、自動推論）
curl -X POST \
     -d "password=yourpassword" \
     -H "X-Forwarded-Host: app.example.com" \
     -c cookies.txt \
     http://auth.example.com/_login
```

## ログアウトエンドポイント

### `GET /_logout`

現在のユーザーをログアウトし、セッションを破棄します。

#### レスポンス

**成功レスポンス（200 OK）**

```
HTTP/1.1 200 OK
Content-Type: text/plain

Logged out
```

セッション Cookie は削除されます。

#### 例

```bash
curl -b cookies.txt http://auth.example.com/_logout
```

## セッション交換エンドポイント

### `GET /_session_exchange`

クロスドメインセッション共有に使用されます。指定されたセッション ID の Cookie を設定し、ルートパスにリダイレクトします。

このエンドポイントは、主に複数のドメイン/サブドメイン間で認証セッションを共有するために使用されます。ユーザーが 1 つのドメインでログインした後、このエンドポイントを使用して別のドメインにセッション Cookie を設定できます。

#### クエリパラメータ

| パラメータ | 型 | 必須 | 説明 |
|-----------|-----|------|------|
| `id` | String | はい | 設定するセッション ID |

#### レスポンス

**成功レスポンス（302 Redirect）**

```
HTTP/1.1 302 Found
Location: /
Set-Cookie: stargate_session_id=<session_id>; Path=/; HttpOnly; SameSite=Lax; Domain=<cookie_domain>; Expires=<expiry>
```

**失敗レスポンス**

| ステータスコード | 説明 | レスポンス本文 |
|----------------|------|----------------|
| `400 Bad Request` | セッション ID が不足 | エラーメッセージ |

#### Cookie ドメイン

環境変数 `COOKIE_DOMAIN` が設定されている場合、Cookie は指定されたドメインに設定され、クロスサブドメイン共有が可能になります。

#### 例

```bash
# セッション Cookie を設定（クロスドメインシナリオ用）
curl "http://auth.example.com/_session_exchange?id=<session_id>"
```

**典型的な使用シナリオ：**

1. ユーザーが `auth.example.com` にログイン
2. ログイン成功後、`app.example.com/_session_exchange?id=<session_id>` にリダイレクト
3. セッション Cookie が `.example.com` ドメインに設定される（`COOKIE_DOMAIN=.example.com` が設定されている場合）
4. `app.example.com/` にリダイレクト
5. ユーザーはすべてのサブドメイン `*.example.com` でこのセッションを使用できます

## ヘルスチェックエンドポイント

### `GET /health`

サービスのヘルスチェックエンドポイント。サービスの状態を監視するために使用されます。

#### レスポンス

**成功レスポンス（200 OK）**

```
HTTP/1.1 200 OK
```

#### 例

```bash
curl http://auth.example.com/health
```

**典型的な使用例：**

- Docker ヘルスチェック
- Kubernetes の liveness プローブ
- ロードバランサーのヘルスチェック

## ルートエンドポイント

### `GET /`

ルートパス、サービス情報を表示します。

#### レスポンス

**200 OK** - サービス情報ページを返す

#### 例

```bash
curl http://auth.example.com/
```

## エラーレスポンス形式

すべての API エラーレスポンスは、クライアントの `Accept` ヘッダーに応じて自動的に形式を選択します：

### JSON 形式（`Accept: application/json`）

```json
{
  "error": "Error message",
  "code": 401
}
```

### XML 形式（`Accept: application/xml`）

```xml
<errors>
  <error code="401">Error message</error>
</errors>
```

### テキスト形式（デフォルト）

```
Error message
```

エラーメッセージは国際化をサポートし、環境変数 `LANGUAGE` に応じて中国語または英語のメッセージを返します。

## 認証フローの例

### Web アプリケーションの認証フロー

1. ユーザーが保護されたリソースにアクセス（例: `https://app.example.com/dashboard`）
2. Traefik がリクエストをインターセプトし、`https://auth.example.com/_auth` に転送
3. Stargate が Cookie 内のセッションを確認
4. 認証されていない場合、`https://auth.example.com/_login?callback=app.example.com` にリダイレクト
5. ユーザーがパスワードを入力して送信
6. Stargate がパスワードを検証し、セッションを作成して Cookie を設定
7. `https://app.example.com/_session_exchange?id=<session_id>` にリダイレクト
8. セッション Cookie が `app.example.com` ドメインに設定される
9. ユーザーが再度保護されたリソースにアクセスし、認証が成功

### API 認証フロー

1. API クライアントが保護されたリソースにリクエストを送信
2. Traefik がリクエストをインターセプトし、`https://auth.example.com/_auth` に転送
3. API クライアントがリクエストヘッダーに `Stargate-Password: <password>` を含める
4. Stargate がパスワードを検証
5. 検証が成功した場合、`X-Forwarded-User` ヘッダーを設定して 200 を返す
6. Traefik がリクエストをバックエンドサービスに継続することを許可

## 注意事項

1. **セッションの有効期限**: デフォルトで 24 時間、期限切れ後は再ログインが必要
2. **Cookie のセキュリティ**: すべての Cookie は `HttpOnly` と `SameSite=Lax` フラグで設定されます
3. **パスワードの検証**: パスワードは検証前に正規化されます（スペースを削除し、大文字に変換）
4. **複数パスワードのサポート**: 複数のパスワードを設定でき、検証を通過したパスワードはすべて受け入れられます
5. **クロスドメインセッション**: クロスドメインセッション共有を有効にするには、環境変数 `COOKIE_DOMAIN` を設定する必要があります
