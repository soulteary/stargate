# API 문서

이 문서는 Stargate Forward Auth 서비스의 모든 API 엔드포인트를 자세히 설명합니다.

v0.12.0에서 업그레이드할 때는 먼저 [v1.0.0 마이그레이션 가이드](MIGRATION_V1.md)를 확인하십시오.

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
   - 기본적으로 비활성화됩니다. 신뢰할 수 있는 레거시 클라이언트에만 `PASSWORD_HEADER_AUTH_ENABLED=true`를 설정하세요.
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
# 서버 전제 조건: PASSWORD_HEADER_AUTH_ENABLED=true
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

로그인 요청을 처리하며 다음 두 가지 인증 모드를 지원합니다:

1. **비밀번호 인증**: 비밀번호를 검증하고 세션 생성
2. **Warden 인증**: Herald challenge 코드 또는 등록된 TOTP 코드를 검증하고 세션 생성

#### 요청 본문

폼 데이터 (`application/x-www-form-urlencoded`):

**비밀번호 인증:**

| 필드 | 타입 | 필수 | 설명 |
|------|------|------|------|
| `auth_method` | String | 아니오 | 인증 방식이며 기본값은 `password` |
| `password` | String | 예 | 사용자 비밀번호 |
| `callback` | String | 아니오 | 로그인 성공 후 콜백 URL |

**Warden 인증:**

| 필드 | 타입 | 필수 | 설명 |
|------|------|------|------|
| `auth_method` | String | 예 | 인증 방식이며 값은 `warden` |
| `phone` | String | 아니오 | 사용자 전화번호. `phone` 또는 `mail` 중 하나 이상 필요 |
| `mail` | String | 아니오 | 사용자 이메일. `mail` 또는 `phone` 중 하나 이상 필요 |
| `challenge_id` | String | 조건부 | `verify_code`와 함께 사용할 때 필수이며 `use_otp=true`일 때 선택 사항 |
| `verify_code` | String | 조건부 | `use_otp`가 없거나 `false`일 때 Herald challenge 검증에 필수 |
| `use_otp` | Boolean | 아니오 | Herald challenge 대신 등록된 TOTP 인증기를 사용하려면 `true`로 설정 |
| `otp_code` | String | 조건부 | `use_otp=true`일 때 필수 |
| `callback` | String | 아니오 | 로그인 성공 후 콜백 URL |

핸들러는 같은 Warden 요청에 `phone` + `mail`을 함께 보내는 것도 허용합니다.
TOTP 방식에는 `HERALD_TOTP_ENABLED=true`와 이미 등록된 사용자가 필요합니다. 그렇지 않으면 `challenge_id` + `verify_code` 방식을 사용합니다.

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
   - `{callback}/_session_exchange?ticket={opaque_ticket}`로 리디렉션
   - 상태 코드: `302 Found`

2. **콜백 없음**:
   - **HTML 요청**: meta refresh를 포함한 HTML 페이지를 반환하고 원본 도메인으로 자동 리디렉션
   - **API 요청**: JSON 응답 반환
     ```json
     {
       "success": true,
       "message": "Login successful"
     }
     ```

**실패 응답**

| 상태 코드 | 설명 | 응답 본문 |
|-----------|------|-----------|
| `400 Bad Request` | Warden 식별자 또는 선택한 검증 방식의 필드가 잘못되었거나 누락됨, 또는 TOTP 미등록 | Accept 헤더에 따라 JSON/XML/텍스트 형식의 오류 메시지 |
| `401 Unauthorized` | 잘못된 비밀번호, Warden 사용자 없음/비활성 또는 challenge/TOTP 검증 실패 | Accept 헤더에 따라 JSON/XML/텍스트 형식의 오류 메시지 |
| `502 Bad Gateway` | Herald TOTP 상태 조회 실패 | 오류 메시지 |
| `503 Service Unavailable` | Herald 검증 서비스를 사용할 수 없음 | 오류 메시지 |
| `500 Internal Server Error` | 서버 오류 | 오류 메시지 |

#### 예제

```bash
# 로그인 폼 제출 (콜백 있음)
curl -X POST \
     -d "auth_method=password&password=yourpassword&callback=app.example.com" \
     -c cookies.txt \
     http://auth.example.com/_login

# 로그인 폼 제출 (콜백 없음, 자동 추론)
curl -X POST \
     -d "auth_method=password&password=yourpassword" \
     -H "X-Forwarded-Host: app.example.com" \
     -c cookies.txt \
     http://auth.example.com/_login

# Warden + Herald: /_send_verify_code가 반환한 challenge로 로그인
curl -X POST \
     -d "auth_method=warden&mail=user@example.com&challenge_id=ch_xxx&verify_code=123456&callback=app.example.com" \
     -c cookies.txt \
     http://auth.example.com/_login

# Warden + TOTP: 등록된 인증기로 로그인
curl -X POST \
     -d "auth_method=warden&mail=user@example.com&use_otp=true&otp_code=123456&callback=app.example.com" \
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

요청 본문 미디어 타입 지원 여부:

| 요청 본문 미디어 타입 | 지원 |
|----------------------|------|
| `application/x-www-form-urlencoded` | ✅ |
| `multipart/form-data` | ✅ |
| `application/json` | ❌ |

| 필드 | 유형 | 필수 | 설명 |
|------|------|------|------|
| `phone` | String | 아니오 | 사용자 전화번호. `phone` 또는 `mail` 중 하나 이상 필요 |
| `mail` | String | 아니오 | 사용자 이메일. `mail` 또는 `phone` 중 하나 이상 필요 |
| `deliver_via` | String | 아니오 | 선호 채널: `sms`, `email`, `dingtalk`. 생략하면 활성화된 SMS 대상을 먼저 시도한 뒤 이메일을 시도합니다. DingTalk는 암묵적으로 선택되지 않으므로 호출자는 `deliver_via=dingtalk`를 명시해야 합니다. |

핸들러는 같은 인증 코드 요청에 `phone` + `mail`을 함께 보내는 것도 허용합니다.

#### 처리 흐름

1. **Stargate → Warden** : 사용자 정보 쿼리
   - 사용자가 화이트리스트에 있는지 확인
   - 사용자 상태 확인 (활성 상태인지)
   - 사용자의 이메일과 전화 가져오기

2. **Stargate → Herald** : 챌린지 생성 및 인증 코드 전송
   - Warden에서 반환된 이메일, 전화번호 또는 DingTalk 식별자를 대상으로 사용
   - Herald API를 호출하여 챌린지 생성
   - Herald가 선택된 SMS, 이메일 또는 DingTalk 채널로 인증 코드 전송

3. **결과 반환** : challenge_id 및 관련 정보 반환

#### 응답

**성공 응답 (200 OK)**

```json
{
  "success": true,
  "message": "Verification code sent",
  "challenge_id": "ch_xxxxxxxxxxxx",
  "expires_in": 300,
  "next_resend_in": 60
}
```

**실패 응답**

| 상태 코드 | 설명 | 응답 본문 |
|----------|------|----------|
| `400 Bad Request` | 잘못된 요청 매개변수 (phone 또는 mail 누락) | 오류 메시지 |
| `401 Unauthorized` | Warden에 사용자가 없거나 활성 상태가 아님 | `success=false` JSON 오류 |
| `429 Too Many Requests` | 속도 제한 트리거됨 | 오류 메시지 |
| `503 Service Unavailable` | Herald 클라이언트 또는 서비스를 사용할 수 없음 | `success=false` JSON 오류 |
| `500 Internal Server Error` | 그 밖의 공급자 전송 오류 | `success=false` JSON 오류 |

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

```

#### 참고 사항

- `WARDEN_ENABLED=true` 및 `HERALD_ENABLED=true` 필요
- 인증 코드를 전송하려면 사용자가 Warden 화이트리스트에 있어야 합니다
- Herald는 속도 제한을 수행하며, 동일한 사용자/전화/이메일에 대해 빈도 제한이 있습니다
- 코드 만료 시간은 Herald 설정에 의해 결정됩니다 (기본값: 300초)
- 재전송 쿨다운 시간은 Herald 설정에 의해 결정됩니다 (기본값: 60초)

## 로그아웃 엔드포인트

### `POST /_logout`

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
curl -X POST -b cookies.txt http://auth.example.com/_logout
```

## Step-up 인증 엔드포인트

### `GET /_step_up`

인증된 세션에 비밀번호 재확인 양식을 표시합니다. 선택적 `callback` 쿼리 매개변수는 로컬 경로여야 하며 안전하지 않은 값은 `/`로 대체됩니다.

### `POST /_step_up`

비밀번호를 다시 확인하고 성공한 Step-up 인증을 세션에 10분 동안 기록합니다. 양식 필드는 `password`와 선택적 로컬 `callback`입니다. 동일 출처 검사와 속도 제한으로 보호됩니다.

## 세션 교환 엔드포인트

### `GET /_session_exchange`

크로스 도메인 세션 공유에 사용됩니다. 수명이 짧고 대상 도메인에 바인딩된 암호화 티켓을 교환하고 인증된 세션을 확인한 뒤 Cookie를 설정하고 루트 경로로 리디렉션합니다.

이 엔드포인트는 주로 여러 도메인/하위 도메인 간에 인증 세션을 공유하는 데 사용됩니다. 사용자가 한 도메인에 로그인한 후, 이 엔드포인트를 사용하여 다른 도메인에 세션 Cookie를 설정할 수 있습니다.

#### 쿼리 매개변

| 매개변수 | 타입 | 필수 | 설명 |
|----------|------|------|------|
| `ticket` | String | 예 | 로그인 후 반환되며 대상 도메인에 바인딩된 교환 티켓 |

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
| `400 Bad Request` | 티켓 누락 | 오류 메시지 |
| `401 Unauthorized` | 티켓이 유효하지 않거나 만료, 재사용 또는 다른 도메인용임 | 오류 메시지 |

#### Cookie 도메인

환경 변수 `COOKIE_DOMAIN`이 설정된 경우, Cookie는 지정된 도메인에 설정되어 크로스 하위 도메인 공유가 가능합니다.

#### 예제

```bash
# 티켓이 바인딩된 콜백 도메인에서 교환
curl "https://app.example.com/_session_exchange?ticket=<opaque_ticket>"
```

**일반적인 사용 시나리오:**

1. 사용자가 `auth.example.com`에 로그인
2. 로그인 성공 후, `app.example.com/_session_exchange?ticket=<opaque_ticket>`로 리디렉션
3. 세션 Cookie가 `.example.com` 도메인에 설정됩니다 (`COOKIE_DOMAIN=.example.com`이 설정된 경우)
4. `app.example.com/`로 리디렉션
5. 사용자는 모든 하위 도메인 `*.example.com`에서 이 세션을 사용할 수 있습니다

## TOTP 엔드포인트

Herald와 Herald TOTP가 활성화된 경우 사용할 수 있습니다. 모든 엔드포인트에는 인증된 세션이 필요합니다. 등록은 해당 세션을 만든 로그인 후 10분 이내에 시작하고 완료해야 합니다.

### `GET /totp/enroll`

등록 상태를 만들지 않고 확인 페이지를 표시합니다. 최근 10분 이내의 로그인으로 생성된 세션이 필요합니다. 인증되지 않은 사용자는 `302 Found`로 `/_login`에 리디렉션되고 오래된 세션에는 `401 Unauthorized`가 반환됩니다.

### `POST /totp/enroll`

등록을 시작하고 TOTP 연결 페이지를 표시합니다. 최근 10분 이내의 로그인으로 생성된 세션이 필요합니다. 인증되지 않은 사용자는 `302 Found`로 `/_login`에 리디렉션되고 오래된 세션에는 `401 Unauthorized`가 반환됩니다.

### `POST /totp/enroll/confirm`

`application/x-www-form-urlencoded` 또는 `multipart/form-data` 폼 데이터로 등록을 확인하며 JSON 요청 본문은 지원하지 않습니다. 최근 10분 이내의 로그인으로 생성된 인증 세션, `enroll_id`, 6자리 TOTP `code`가 필요합니다. 인증이 없거나 오래된 경우 `401 Unauthorized`를 반환하며 성공 JSON에는 한 번만 표시되는 백업 코드가 포함됩니다.

### `GET /totp/revoke`

TOTP 연결 해제 확인 페이지를 표시합니다. 인증된 세션이 필요하며 인증되지 않은 사용자는 `302 Found`로 `/_login`에 리디렉션됩니다. 이 페이지에는 10분 제한이 적용되지 않습니다.

### `POST /totp/revoke`

TOTP 연결을 해제합니다. 인증된 세션과 함께 `password` 또는 유효한 현재 TOTP `code`로 재인증해야 합니다. 재인증이 없거나 실패하면 `401 Unauthorized`를 반환합니다.

## 헬스 체크 엔드포인트

### `GET /healthz`

Redis, Warden, Herald를 조회하지 않는 프로세스 liveness 엔드포인트입니다. 컨테이너 헬스 체크와 Kubernetes liveness 프로브에 사용합니다.

### `GET /readyz`

설정된 Redis, Warden, Herald 의존성의 집계 readiness 엔드포인트입니다. 로드 밸런서와 Kubernetes readiness 프로브에 사용합니다.

```bash
curl http://auth.example.com/healthz
curl http://auth.example.com/readyz
```

`GET /health`는 `GET /readyz`의 더 이상 권장되지 않는 호환 별칭으로 유지됩니다.

## 메트릭 엔드포인트

### `GET /metrics`

Prometheus 메트릭을 텍스트 형식으로 반환합니다. 인증이 필요하지 않으므로 네트워크 또는 프록시 규칙으로 보호해야 합니다.

## 런타임 로그 레벨 엔드포인트

### `GET /log/level`

현재 로그 레벨을 반환합니다. `PUT /log/level` 또는 `POST /log/level`에 JSON 본문 `{"level":"debug"}`나 쿼리 매개변수 `?level=debug`를 전달해 변경합니다. Stargate는 이 엔드포인트를 `127.0.0.1`로 제한하므로 공개 리버스 프록시를 통해 노출하면 안 됩니다.

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
7. `https://app.example.com/_session_exchange?ticket=<opaque_ticket>`로 리디렉션
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
3. **비밀번호 검증**: 비밀번호는 대소문자와 공백을 그대로 구분하는 불투명한 값으로 검증됩니다
4. **여러 비밀번호 지원**: 여러 비밀번호를 설정할 수 있으며, 검증을 통과한 비밀번호는 모두 허용됩니다
5. **크로스 도메인 세션**: 크로스 도메인 세션 공유를 활성화하려면 환경 변수 `COOKIE_DOMAIN`을 설정해야 합니다
