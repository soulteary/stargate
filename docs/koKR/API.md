# API 문서

이 문서는 Stargate Forward Auth 서비스의 모든 API 엔드포인트를 자세히 설명합니다.

## 목차

- [인증 확인 엔드포인트](#인증-확인-엔드포인트)
- [로그인 엔드포인트](#로그인-엔드포인트)
- [로그아웃 엔드포인트](#로그아웃-엔드포인트)
- [세션 교환 엔드포인트](#세션-교환-엔드포인트)
- [헬스 체크 엔드포인트](#헬스-체크-엔드포인트)
- [루트 엔드포인트](#루트-엔드포인트)

## 인증 확인 엔드포인트

### `GET /_auth`

Traefik Forward Auth의 주요 인증 확인 엔드포인트. 이 엔드포인트는 Stargate의 주요 기능으로, 사용자가 인증되었는지 확인하는 데 사용됩니다.

#### 인증 방법

Stargate는 다음 우선순위로 확인되는 두 가지 인증 방법을 지원합니다:

1. **헤더 인증** (API 요청)
   - 요청 헤더: `Stargate-Password: <password>`
   - API 요청, 자동화 스크립트 등에 적합합니다

2. **Cookie 인증** (웹 요청)
   - Cookie: `stargate_session_id=<session_id>`
   - 브라우저를 통해 액세스하는 웹 애플리케이션에 적합합니다

#### 요청 헤더

| 헤더 | 타입 | 필수 | 설명 |
|------|------|------|------|
| `Stargate-Password` | String | 아니오 | API 요청용 비밀번호 인증 |
| `Cookie` | String | 아니오 | `stargate_session_id`를 포함하는 세션 Cookie |
| `Accept` | String | 아니오 | 요청 타입 (HTML/API)을 결정하는 데 사용 |

#### 응답

**성공 응답 (200 OK)**

인증이 성공하면 Stargate는 사용자 정보 헤더를 설정하고 상태 코드 200을 반환합니다:

```
HTTP/1.1 200 OK
X-Forwarded-User: authenticated
```

사용자 헤더 이름은 환경 변수 `USER_HEADER_NAME`으로 설정할 수 있습니다 (기본값: `X-Forwarded-User`).

**실패 응답**

| 상태 코드 | 설명 | 응답 본문 |
|-----------|------|-----------|
| `401 Unauthorized` | 인증 실패 | 오류 메시지 (API 요청의 경우 JSON 형식) 또는 로그인 페이지로 리디렉션 (HTML 요청) |
| `500 Internal Server Error` | 서버 오류 | 오류 메시지 |

#### 요청 타입 처리

- **HTML 요청**: 인증 실패 시 `/_login?callback=<originalURL>`로 리디렉션
- **API 요청** (JSON/XML): 인증 실패 시 401 오류 응답 반환

#### 예제

**헤더 인증 사용 (API 요청)**

```bash
curl -H "Stargate-Password: yourpassword" \
     http://auth.example.com/_auth
```

**Cookie 인증 사용 (웹 요청)**

```bash
curl -H "Cookie: stargate_session_id=<session_id>" \
     http://auth.example.com/_auth
```

## 로그인 엔드포인트

### `GET /_login`

로그인 페이지를 표시합니다.

#### 쿼리 매개변

| 매개변수 | 타입 | 필수 | 설명 |
|----------|------|------|------|
| `callback` | String | 아니오 | 로그인 성공 후 콜백 URL (일반적으로 원본 요청의 도메인) |

#### 동작

- 사용자가 이미 로그인한 경우, 세션 교환 엔드포인트로 자동 리디렉션
- 사용자가 로그인하지 않은 경우, 로그인 페이지 표시
- URL에 `callback` 매개변수가 포함되고 도메인이 다른 경우, 콜백은 Cookie `stargate_callback`에 저장됩니다 (10분 후 만료)

#### 콜백 가져오기 우선순위

1. **쿼리 매개변에서**: URL 내 `callback` 매개변수 (최고 우선순위)
2. **Cookie에서**: 쿼리 매개변에 없는 경우, Cookie `stargate_callback`에서 가져옴

#### 응답

**200 OK** - 로그인 페이지의 HTML 반환

페이지에는 다음이 포함됩니다:
- 로그인 폼
- 사용자 정의 가능한 제목 (`LOGIN_PAGE_TITLE`)
- 사용자 정의 가능한 푸터 텍스트 (`LOGIN_PAGE_FOOTER_TEXT`)

#### 예제

```bash
# 로그인 페이지에 액세스
curl http://auth.example.com/_login?callback=app.example.com
```

### `POST /_login`

로그인 요청을 처리하고 비밀번호를 검증하여 세션을 생성합니다.

#### 요청 본문

폼 데이터 (`application/x-www-form-urlencoded`):

| 필드 | 타입 | 필수 | 설명 |
|------|------|------|------|
| `password` | String | 예 | 사용자 비밀번호 |
| `callback` | String | 아니오 | 로그인 성공 후 콜백 URL |

#### 콜백 가져오기 우선순위

로그인 처리 시 다음 우선순위로 콜백을 가져옵니다:

1. **Cookie에서**: 이전에 로그인 페이지에 액세스할 때 도메인이 달랐던 경우, 콜백은 Cookie `stargate_callback`에 저장됩니다
2. **폼 데이터에서**: POST 요청의 폼 데이터 내 `callback` 필드
3. **쿼리 매개변에서**: URL의 쿼리 매개변 내 `callback`
4. **자동 추론**: 위 항목이 없고 원본 도메인 (`X-Forwarded-Host`)이 인증 서비스 도메인과 다른 경우, 원본 도메인을 콜백으로 사용

#### 응답

**성공 응답 (200 OK)**

응답은 콜백 존재 여부와 요청 타입에 따라 다릅니다:

1. **콜백 있음**:
   - `{callback}/_session_exchange?id={session_id}`로 리디렉션
   - 상태 코드: `302 Found`

2. **콜백 없음**:
   - **HTML 요청**: meta refresh를 포함한 HTML 페이지를 반환하고 원본 도메인으로 자동 리디렉션
   - **API 요청**: JSON 응답 반환
     ```json
     {
       "success": true,
       "message": "Login successful",
       "session_id": "<session_id>"
     }
     ```

**실패 응답**

| 상태 코드 | 설명 | 응답 본문 |
|-----------|------|-----------|
| `401 Unauthorized` | 잘못된 비밀번호 | Accept 헤더에 따라 JSON/XML/텍스트 형식의 오류 메시지 |
| `500 Internal Server Error` | 서버 오류 | 오류 메시지 |

#### 예제

```bash
# 로그인 폼 제출 (콜백 있음)
curl -X POST \
     -d "password=yourpassword&callback=app.example.com" \
     -c cookies.txt \
     http://auth.example.com/_login

# 로그인 폼 제출 (콜백 없음, 자동 추론)
curl -X POST \
     -d "password=yourpassword" \
     -H "X-Forwarded-Host: app.example.com" \
     -c cookies.txt \
     http://auth.example.com/_login
```

## 인증 코드 전송 엔드포인트

### `POST /_send_verify_code`

인증 코드 전송 요청. 이 엔드포인트는 Warden + Herald OTP 인증 흐름에서 사용됩니다.

#### 요청 헤더 (선택)

| 헤더 | 설명 |
|------|------|
| `Idempotency-Key` | 선택. 있으면 Stargate가 Herald로 전달하며, Herald는 TTL 내 동일 key 중복 요청에 같은 challenge 응답을 반환한다. |

#### 요청 본문

폼 데이터 (`application/x-www-form-urlencoded`) 또는 JSON (`application/json`) :

| 필드 | 유형 | 필수 | 설명 |
|------|------|------|------|
| `phone` | String | 아니오 | 사용자 전화번호 (`phone` 또는 `mail` 중 하나) |
| `mail` | String | 아니오 | 사용자 이메일 (`phone` 또는 `mail` 중 하나) |

#### 처리 흐름

1. **Stargate → Warden** : 사용자 정보 쿼리
   - 사용자가 화이트리스트에 있는지 확인
   - 사용자 상태 확인 (활성 상태인지)
   - 사용자의 이메일과 전화 가져오기

2. **Stargate → Herald** : 챌린지 생성 및 인증 코드 전송
   - Warden에서 반환된 이메일/전화를 대상으로 사용
   - Herald API를 호출하여 챌린지 생성
   - Herald가 인증 코드 전송 (SMS 또는 이메일)

3. **결과 반환** : challenge_id 및 관련 정보 반환

#### 응답

**성공 응답 (200 OK)**

```json
{
  "success": true,
  "challenge_id": "ch_xxxxxxxxxxxx",
  "expires_in": 300,
  "next_resend_in": 60,
  "channel": "email",
  "destination": "u***@example.com"
}
```

**실패 응답**

| 상태 코드 | 설명 | 응답 본문 |
|----------|------|----------|
| `400 Bad Request` | 잘못된 요청 매개변수 (phone 또는 mail 누락) | 오류 메시지 |
| `404 Not Found` | 사용자가 Warden 화이트리스트에 없음 | 오류 메시지 |
| `429 Too Many Requests` | 속도 제한 트리거됨 | 오류 메시지 |
| `500 Internal Server Error` | 서버 오류 또는 Herald 서비스 사용 불가 | 오류 메시지 |

#### 예제

```bash
# 인증 코드 전송 (이메일 사용)
curl -X POST \
     -d "mail=user@example.com" \
     -H "Content-Type: application/x-www-form-urlencoded" \
     http://auth.example.com/_send_verify_code

# 인증 코드 전송 (전화 사용)
curl -X POST \
     -d "phone=13800138000" \
     -H "Content-Type: application/x-www-form-urlencoded" \
     http://auth.example.com/_send_verify_code

# JSON 형식 사용
curl -X POST \
     -H "Content-Type: application/json" \
     -d '{"mail":"user@example.com"}' \
     http://auth.example.com/_send_verify_code
```

#### 참고 사항

- `WARDEN_ENABLED=true` 및 `HERALD_ENABLED=true` 필요
- 인증 코드를 전송하려면 사용자가 Warden 화이트리스트에 있어야 합니다
- Herald는 속도 제한을 수행하며, 동일한 사용자/전화/이메일에 대해 빈도 제한이 있습니다
- 코드 만료 시간은 Herald 설정에 의해 결정됩니다 (기본값: 300초)
- 재전송 쿨다운 시간은 Herald 설정에 의해 결정됩니다 (기본값: 60초)

## 로그아웃 엔드포인트

### `GET /_logout`

현재 사용자를 로그아웃하고 세션을 파괴합니다.

#### 응답

**성공 응답 (200 OK)**

```
HTTP/1.1 200 OK
Content-Type: text/plain

Logged out
```

세션 Cookie가 삭제됩니다.

#### 예제

```bash
curl -b cookies.txt http://auth.example.com/_logout
```

## 세션 교환 엔드포인트

### `GET /_session_exchange`

크로스 도메인 세션 공유에 사용됩니다. 지정된 세션 ID의 Cookie를 설정하고 루트 경로로 리디렉션합니다.

이 엔드포인트는 주로 여러 도메인/하위 도메인 간에 인증 세션을 공유하는 데 사용됩니다. 사용자가 한 도메인에 로그인한 후, 이 엔드포인트를 사용하여 다른 도메인에 세션 Cookie를 설정할 수 있습니다.

#### 쿼리 매개변

| 매개변수 | 타입 | 필수 | 설명 |
|----------|------|------|------|
| `id` | String | 예 | 설정할 세션 ID |

#### 응답

**성공 응답 (302 Redirect)**

```
HTTP/1.1 302 Found
Location: /
Set-Cookie: stargate_session_id=<session_id>; Path=/; HttpOnly; SameSite=Lax; Domain=<cookie_domain>; Expires=<expiry>
```

**실패 응답**

| 상태 코드 | 설명 | 응답 본문 |
|-----------|------|-----------|
| `400 Bad Request` | 세션 ID 누락 | 오류 메시지 |

#### Cookie 도메인

환경 변수 `COOKIE_DOMAIN`이 설정된 경우, Cookie는 지정된 도메인에 설정되어 크로스 하위 도메인 공유가 가능합니다.

#### 예제

```bash
# 세션 Cookie 설정 (크로스 도메인 시나리오용)
curl "http://auth.example.com/_session_exchange?id=<session_id>"
```

**일반적인 사용 시나리오:**

1. 사용자가 `auth.example.com`에 로그인
2. 로그인 성공 후, `app.example.com/_session_exchange?id=<session_id>`로 리디렉션
3. 세션 Cookie가 `.example.com` 도메인에 설정됩니다 (`COOKIE_DOMAIN=.example.com`이 설정된 경우)
4. `app.example.com/`로 리디렉션
5. 사용자는 모든 하위 도메인 `*.example.com`에서 이 세션을 사용할 수 있습니다

## 헬스 체크 엔드포인트

### `GET /health`

서비스의 헬스 체크 엔드포인트. 서비스 상태를 모니터링하는 데 사용됩니다.

#### 응답

**성공 응답 (200 OK)**

```
HTTP/1.1 200 OK
```

#### 예제

```bash
curl http://auth.example.com/health
```

**일반적인 사용 예:**

- Docker 헬스 체크
- Kubernetes liveness 프로브
- 로드 밸런서 헬스 체크

## 루트 엔드포인트

### `GET /`

루트 경로, 서비스 정보를 표시합니다.

#### 응답

**200 OK** - 서비스 정보 페이지 반환

#### 예제

```bash
curl http://auth.example.com/
```

## 오류 응답 형식

모든 API 오류 응답은 클라이언트의 `Accept` 헤더에 따라 자동으로 형식을 선택합니다:

### JSON 형식 (`Accept: application/json`)

```json
{
  "error": "Error message",
  "code": 401
}
```

### XML 형식 (`Accept: application/xml`)

```xml
<errors>
  <error code="401">Error message</error>
</errors>
```

### 텍스트 형식 (기본값)

```
Error message
```

오류 메시지는 국제화를 지원하며, 환경 변수 `LANGUAGE`에 따라 중국어 또는 영어 메시지를 반환합니다.

## 인증 플로우 예제

### 웹 애플리케이션 인증 플로우

1. 사용자가 보호된 리소스에 액세스 (예: `https://app.example.com/dashboard`)
2. Traefik이 요청을 가로채고 `https://auth.example.com/_auth`로 전달
3. Stargate가 Cookie 내 세션을 확인
4. 인증되지 않은 경우, `https://auth.example.com/_login?callback=app.example.com`로 리디렉션
5. 사용자가 비밀번호를 입력하고 제출
6. Stargate가 비밀번호를 검증하고 세션을 생성하여 Cookie 설정
7. `https://app.example.com/_session_exchange?id=<session_id>`로 리디렉션
8. 세션 Cookie가 `app.example.com` 도메인에 설정됩니다
9. 사용자가 다시 보호된 리소스에 액세스하고 인증이 성공합니다

### API 인증 플로우

1. API 클라이언트가 보호된 리소스에 요청 전송
2. Traefik이 요청을 가로채고 `https://auth.example.com/_auth`로 전달
3. API 클라이언트가 요청 헤더에 `Stargate-Password: <password>` 포함
4. Stargate가 비밀번호를 검증
5. 검증이 성공하면 `X-Forwarded-User` 헤더를 설정하고 200 반환
6. Traefik이 요청을 백엔드 서비스로 계속 진행하도록 허용

## 참고 사항

1. **세션 만료 시간**: 기본적으로 24시간, 만료 후 재로그인 필요
2. **Cookie 보안**: 모든 Cookie는 `HttpOnly` 및 `SameSite=Lax` 플래그로 설정됩니다
3. **비밀번호 검증**: 비밀번호는 검증 전에 정규화됩니다 (공백 제거, 대문자로 변환)
4. **여러 비밀번호 지원**: 여러 비밀번호를 설정할 수 있으며, 검증을 통과한 비밀번호는 모두 허용됩니다
5. **크로스 도메인 세션**: 크로스 도메인 세션 공유를 활성화하려면 환경 변수 `COOKIE_DOMAIN`을 설정해야 합니다
