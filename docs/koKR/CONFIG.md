# 설정 참조

이 문서는 Stargate의 모든 설정 옵션을 자세히 설명합니다.

## 목차

- [설정 방법](#설정-방법)
- [필수 설정](#필수-설정)
- [선택적 설정](#선택적-설정)
- [비밀번호 설정](#비밀번호-설정)
- [설정 예제](#설정-예제)

## 설정 방법

Stargate는 환경 변수를 통해 설정됩니다. 모든 설정 항목은 환경 변수로 정의되며, 설정 파일은 필요하지 않습니다.

### 환경 변수 설정

**Linux/macOS:**

```bash
export AUTH_HOST=auth.example.com
export PASSWORDS='plaintext:yourpassword'
```

**Docker:**

```bash
docker run -e AUTH_HOST=auth.example.com -e PASSWORDS=plaintext:yourpassword ghcr.io/soulteary/stargate:v1.0.0
```

**Docker Compose:**

```yaml
services:
  stargate:
    environment:
      - AUTH_HOST=auth.example.com
      - PASSWORDS=plaintext:yourpassword
```

## 필수 설정

다음 설정 항목은 필수입니다. 이를 설정하지 않으면 서비스가 시작되지 않습니다.

### `AUTH_HOST`

인증 서비스의 호스트 이름.

| 속성 | 값 |
|------|-----|
| **타입** | String |
| **필수** | 예 |
| **기본값** | 없음 |
| **예제** | `auth.example.com` |

**설명:**

- 로그인 콜백 URL 구성에 사용
- 일반적으로 Stargate 서비스의 호스트 이름으로 설정
- 와일드카드 `*` 지원 (프로덕션 환경에서는 권장하지 않음)

**예제:**

```bash
AUTH_HOST=auth.example.com
```

### `PASSWORDS`

비밀번호 설정. 비밀번호 암호화 알고리즘과 비밀번호 목록을 지정합니다.

| 속성 | 값 |
|------|-----|
| **타입** | String |
| **필수** | 예 |
| **기본값** | 없음 |
| **형식** | `algorithm:password1|password2|password3` |

**설명:**

- 형식: `algorithm:password1|password2|password3`
- 여러 비밀번호 지원, `|`로 구분
- 검증을 통과한 비밀번호는 모두 로그인 허용
- 지원되는 알고리즘은 [비밀번호 설정](#비밀번호-설정) 섹션을 참조

**예제:**

```bash
# 단일 평문 비밀번호
PASSWORDS='plaintext:test123'

# 여러 평문 비밀번호
PASSWORDS='plaintext:test123|admin456|user789'

# BCrypt 해시
PASSWORDS='bcrypt:$2a$10$N9qo8uLOickgx2ZMRZoMyeIjZAgcfl7p92ldGxad68LJZdL17lhWy'

```

## 선택적 설정

다음 설정 항목은 선택적입니다. 설정되지 않은 경우 기본값이 사용됩니다.

### `DEBUG`

디버그 모드를 활성화합니다.

| 속성 | 값 |
|------|-----|
| **타입** | Boolean |
| **필수** | 아니오 |
| **기본값** | `false` |
| **가능한 값** | `true`, `false` |

**설명:**

- 활성화하면 로그 레벨이 `DEBUG`로 설정됩니다
- 더 자세한 디버깅 정보 표시
- 프로덕션 환경에서는 `false`로 설정하는 것을 권장

**예제:**

```bash
DEBUG=true
```

### `LANGUAGE`

인터페이스 언어.

| 속성 | 값 |
|------|-----|
| **타입** | String |
| **필수** | 아니오 |
| **기본값** | `en` |
| **가능한 값** | `en`(영어), `zh`(중국어), `fr`(프랑스어), `it`(이탈리아어), `ja`(일본어), `de`(독일어), `ko`(한국어) |

**설명:**

- 오류 메시지와 인터페이스 텍스트의 언어에 영향
- 대소문자 구분 없음 (`EN`, `en`, `En` 모두 작동)

**예제:**

```bash
LANGUAGE=ko
```

### `LOGIN_PAGE_TITLE`

로그인 페이지의 제목.

| 속성 | 값 |
|------|-----|
| **타입** | String |
| **필수** | 아니오 |
| **기본값** | `Stargate - Login` |

**설명:**

- 로그인 페이지의 제목 위치에 표시
- HTML 태그 지원 (권장하지 않음)

**예제:**

```bash
LOGIN_PAGE_TITLE='내 인증 서비스'
```

### `LOGIN_PAGE_FOOTER_TEXT`

로그인 페이지의 푸터 텍스트.

| 속성 | 값 |
|------|-----|
| **타입** | String |
| **필수** | 아니오 |
| **기본값** | `Copyright © 2024 - Stargate` |

**설명:**

- 로그인 페이지의 푸터 위치에 표시
- HTML 태그 지원 (권장하지 않음)

**예제:**

```bash
LOGIN_PAGE_FOOTER_TEXT='© 2024 내 회사'
```

### `USER_HEADER_NAME`

인증 성공 후 설정되는 사용자 헤더 이름.

| 속성 | 값 |
|------|-----|
| **타입** | String |
| **필수** | 아니오 |
| **기본값** | `X-Forwarded-User` |

**설명:**

- 인증 성공 후, Stargate는 응답에서 이 헤더를 설정합니다
- 사용 가능한 경우 인증된 사용자 ID가 값이며, 비밀번호 전용 인증에서는 `authenticated`를 사용합니다
- 백엔드 서비스는 이 헤더를 통해 사용자가 인증되었는지 확인할 수 있습니다
- 비어 있지 않은 문자열이어야 합니다

**예제:**

```bash
USER_HEADER_NAME=X-Authenticated-User
```

### `COOKIE_DOMAIN`

Cookie 도메인. 크로스 도메인 세션 공유에 사용됩니다.

| 속성 | 값 |
|------|-----|
| **타입** | String |
| **필수** | 아니오 |
| **기본값** | 비어 있음 (설정되지 않음) |

**설명:**

- 설정된 경우, 세션 Cookie는 지정된 도메인에 설정됩니다
- 크로스 서브도메인 세션 공유 지원
- 형식: `.example.com` (앞의 점에 주의)
- 비어 있게 설정된 경우, Cookie는 현재 도메인에서만 유효

**예제:**

```bash
# 모든 서브도메인 *.example.com에서 세션 공유 허용
COOKIE_DOMAIN=.example.com
```

**크로스 도메인 세션 공유 시나리오:**

다음 도메인을 가정:
- `auth.example.com` - 인증 서비스
- `app1.example.com` - 애플리케이션 1
- `app2.example.com` - 애플리케이션 2

`COOKIE_DOMAIN=.example.com` 설정 후:
1. 사용자가 `auth.example.com`에 로그인
2. 세션 Cookie가 `.example.com` 도메인에 설정됨
3. 사용자는 `app1.example.com`과 `app2.example.com`에서 동일한 세션을 사용할 수 있습니다

### `PORT`

서비스의 리스닝 포트 (로컬 개발 전용). config 패키지에서 다른 환경 변수와 함께 로드 및 검증됩니다.

| 속성 | 값 |
|------|-----|
| **타입** | String |
| **필수** | 아니오 |
| **기본값** | 비어 있음 (비어 있으면 서버는 기본 포트 `:80` 사용) |

**설명:**

- 로컬 개발 환경 전용
- Docker 컨테이너에서는 일반적으로 불필요 (기본 포트 80 사용)
- 형식: 포트 번호 (예: `8080`) 또는 `:port` (예: `:8080`)

**예제:**

```bash
PORT=8080
```

## 보안 및 연동 설정

다음 표는 `internal/config`에 실제로 등록된 보안 관련 변수와 동기화되어 있습니다. 선택 값이 비어 있으면 해당 연동은 비활성화됩니다.

| 변수 | 값 | 기본값 | 용도 |
|------|----|--------|------|
| `TRUSTED_PROXIES` | IP/CIDR 목록 | 비어 있음 | Forwarded 헤더를 신뢰할 출처 |
| `PROXY_HEADER` | 헤더 이름 | `X-Forwarded-For` | 원본 IP 헤더 |
| `COOKIE_SECURE` | true/false | true | 세션 쿠키 Secure 속성 |
| `CALLBACK_ALLOWED_HOSTS` | 호스트 목록 | 비어 있음 | 허용된 리디렉션 대상 |
| `SESSION_EXCHANGE_SECRET` | 32자 이상 비밀값 | 비어 있음 | 단기 교차 도메인 티켓 서명 |
| `PASSWORD_HEADER_AUTH_ENABLED` | true/false | false | 신뢰된 레거시 비밀번호 헤더 활성화 |
| `LOG_LEVEL` | debug/info/warn/error | info | 시작 로그 레벨 |
| `HEADER_AUTH_ENABLED` | true/false | false | 신뢰 헤더 인증 |
| `HEADER_AUTH_SHARED_SECRET` | 최소 32자의 비밀값 | 비어 있음 | 헤더 인증 공유 비밀 |
| `HEADER_AUTH_SECRET_HEADER` | 헤더 이름 | `X-Stargate-Header-Auth` | 공유 비밀 전달 헤더 |
| `WARDEN_ENABLED` | true/false | false | Warden 연동 |
| `WARDEN_URL` | URL | 비어 있음 | Warden 엔드포인트 |
| `WARDEN_API_KEY` | 비밀값 | 비어 있음 | API 키 인증 |
| `WARDEN_HMAC_KEY_ID` | 문자열 | 비어 있음 | HMAC 키 ID |
| `WARDEN_HMAC_SECRET` | 비밀값 | 비어 있음 | HMAC 서명 비밀 |
| `WARDEN_TLS_CA_CERT_FILE` | 경로 | 비어 있음 | 사용자 지정 CA |
| `WARDEN_TLS_CLIENT_CERT_FILE` | 경로 | 비어 있음 | mTLS 클라이언트 인증서 |
| `WARDEN_TLS_CLIENT_KEY_FILE` | 경로 | 비어 있음 | mTLS 클라이언트 키 |
| `WARDEN_TLS_SERVER_NAME` | 문자열 | 비어 있음 | 예상 TLS 서버 이름 |
| `WARDEN_CACHE_TTL` | 초 | 300 | Warden 캐시 시간 |
| `HERALD_ENABLED` | true/false | false | Herald 연동 |
| `HERALD_URL` | URL | 비어 있음 | Herald 엔드포인트 |
| `HERALD_API_KEY` | 비밀값 | 비어 있음 | API 키 인증 |
| `HERALD_HMAC_KEY_ID` | 문자열 | 비어 있음 | HMAC 키 ID |
| `HERALD_HMAC_SECRET` | 비밀값 | 비어 있음 | HMAC 서명 비밀 |
| `HERALD_TLS_CA_CERT_FILE` | 경로 | 비어 있음 | 사용자 지정 CA |
| `HERALD_TLS_CLIENT_CERT_FILE` | 경로 | 비어 있음 | mTLS 클라이언트 인증서 |
| `HERALD_TLS_CLIENT_KEY_FILE` | 경로 | 비어 있음 | mTLS 클라이언트 키 |
| `HERALD_TLS_SERVER_NAME` | 문자열 | 비어 있음 | 예상 TLS 서버 이름 |
| `HERALD_TOTP_ENABLED` | true/false | false | TOTP 로그인 및 관리 |
| `LOGIN_SMS_ENABLED` | true/false | true | SMS 로그인 채널 |
| `LOGIN_EMAIL_ENABLED` | true/false | true | 이메일 로그인 채널 |
| `SESSION_STORAGE_ENABLED` | true/false | false | Redis 세션 저장소 |
| `SESSION_STORAGE_REDIS_ADDR` | host:port | `localhost:6379` | Redis 주소 |
| `SESSION_STORAGE_REDIS_PASSWORD` | 비밀값 | 비어 있음 | Redis 비밀번호 |
| `SESSION_STORAGE_REDIS_DB` | 정수 | 0 | Redis DB |
| `SESSION_STORAGE_REDIS_KEY_PREFIX` | 문자열 | `stargate:session:` | Redis 키 접두사 |
| `AUDIT_LOG_ENABLED` | true/false | true | 감사 로그 |
| `AUDIT_LOG_FORMAT` | json/text | json | 감사 로그 형식 |
| `STEP_UP_ENABLED` | true/false | false | 추가 인증 |
| `STEP_UP_PATHS` | 경로 목록 | 비어 있음 | 보호할 업무 경로 |
| `OTLP_ENABLED` | true/false | false | OpenTelemetry 내보내기 |
| `OTLP_ENDPOINT` | URL | 비어 있음 | OTLP 엔드포인트 |
| `AUTH_REFRESH_ENABLED` | true/false | true | 권한 정보 주기적 갱신 |
| `AUTH_REFRESH_INTERVAL` | 기간 | `5m` | 갱신 간격 |
| `REQUEST_CONTEXT_TIMEOUT` | 기간 | `10s` | 업스트림 호출 기한. 클라이언트 연결 종료 시 즉시 취소는 보장하지 않음 |

HMAC Key ID와 Secret은 함께 설정해야 합니다. mTLS 클라이언트 인증서와 키도 완전한 쌍이어야 하며, 불완전한 설정은 시작 실패로 처리됩니다.

## 비밀번호 설정

운영 환경에서는 `bcrypt`를 사용하며 `plaintext`는 로컬 테스트에서만 사용할 수 있습니다. MD5와 salt 없는 SHA-512는 설정 검증에서 거부됩니다. 비밀번호는 대소문자와 공백을 그대로 구분합니다.

```bash
PASSWORDS='bcrypt:<bcrypt-hash>'
# 로컬 테스트 전용:
PASSWORDS='plaintext:test123'
```

## 설정 예제

### 기본 설정

```bash
# 필수 설정
AUTH_HOST=auth.example.com
PASSWORDS='plaintext:test123'

# 선택적 설정
DEBUG=false
LANGUAGE=en
```

### 프로덕션 환경 설정

```bash
# 필수 설정
AUTH_HOST=auth.example.com
PASSWORDS='bcrypt:$2a$10$N9qo8uLOickgx2ZMRZoMyeIjZAgcfl7p92ldGxad68LJZdL17lhWy'

# 선택적 설정
DEBUG=false
LANGUAGE=ko
LOGIN_PAGE_TITLE='내 인증 서비스'
LOGIN_PAGE_FOOTER_TEXT='© 2024 내 회사'
USER_HEADER_NAME=X-Forwarded-User
COOKIE_DOMAIN=.example.com
```

### Docker Compose 설정

```yaml
services:
  stargate:
    image: ghcr.io/soulteary/stargate:v1.0.0
    environment:
      # 필수 설정
      - AUTH_HOST=auth.example.com
      - PASSWORDS=bcrypt:$$2a$$10$$N9qo8uLOickgx2ZMRZoMyeIjZAgcfl7p92ldGxad68LJZdL17lhWy
      
      # 선택적 설정
      - DEBUG=false
      - LANGUAGE=ko
      - LOGIN_PAGE_TITLE=내 인증 서비스
      - LOGIN_PAGE_FOOTER_TEXT=© 2024 내 회사
      - COOKIE_DOMAIN=.example.com
```

### 로컬 개발 설정

```bash
# 필수 설정
AUTH_HOST=localhost
PASSWORDS='plaintext:test123|admin456'

# 선택적 설정
DEBUG=true
LANGUAGE=ko
PORT=8080
```

## 설정 검증

Stargate는 시작 시 모든 설정 항목을 검증합니다:

1. **필수 설정 확인**: 필수 설정이 설정되지 않은 경우, 서비스는 시작에 실패하고 오류 메시지를 표시합니다
2. **형식 검증**: 비밀번호 설정 형식이 올바르지 않은 경우, 시작에 실패합니다
3. **알고리즘 검증**: 지원되지 않는 비밀번호 알고리즘은 시작에 실패합니다
4. **값 검증**: 일부 설정 항목에는 값 제한이 있습니다 (예: `LANGUAGE`, `DEBUG`)

**오류 예제:**

```bash
# 필수 설정 부족
Error: Configuration error: environment variable 'AUTH_HOST' is required but not set.

# 비밀번호 형식이 올바르지 않음
Error: Configuration error: invalid value for environment variable 'PASSWORDS': 'invalid_format'

# 지원되지 않는 알고리즘
Error: Configuration error: invalid value for environment variable 'PASSWORDS': 'unknown:password'
```

## 설정 모범 사례

1. **프로덕션 환경 보안**:
   - 운영 환경에서는 `bcrypt`를 사용하고 `plaintext`는 로컬 테스트로 제한하세요
   - `DEBUG=false`로 설정
   - 강력한 비밀번호 사용

2. **크로스 도메인 세션**:
   - 서브도메인 간 세션 공유가 필요한 경우, `COOKIE_DOMAIN`을 설정하세요
   - 형식: `.example.com` (앞의 점에 주의)

3. **다국어 지원**:
   - 사용자 기반에 따라 `LANGUAGE`를 설정하세요
   - `en`, `zh`, `fr`, `it`, `ja`, `de`, `ko` 지원

4. **커스텀 인터페이스**:
   - `LOGIN_PAGE_TITLE`과 `LOGIN_PAGE_FOOTER_TEXT`를 사용하여 로그인 페이지를 커스터마이즈하세요

5. **모니터링 및 디버깅**:
   - 개발 환경에서는 `DEBUG=true`를 설정하여 자세한 로그를 얻으세요
   - 프로덕션 환경에서는 `DEBUG=false`를 설정하여 로그 출력을 줄이세요

## 레거시 비밀번호 헤더 인증

`Stargate-Password` 인증은 기본적으로 비활성화됩니다. 신뢰할 수 있는 레거시 클라이언트에만 `PASSWORD_HEADER_AUTH_ENABLED=true`로 활성화하세요. HTTPS와 요청 속도 제한을 사용하고, 백엔드로 전달하기 전에 리버스 프록시에서 이 헤더를 제거하세요.
