# 설정 참조

> **이전 번역**: 이 문서에는 현재 변경 사항이 모두 반영되어 있지 않습니다. Redis 요구 사항과 업그레이드 제한은 [최신 영어 설정 문서](../enUS/CONFIG.md)를 확인하세요.

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
docker run -e AUTH_HOST=auth.example.com -e PASSWORDS=plaintext:yourpassword ghcr.io/soulteary/stargate:1.3.8
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

### `TRUSTED_PROXIES` 및 `PROXY_HEADER`

`TRUSTED_PROXIES`에는 Stargate가 전달 헤더를 신뢰할 리버스 프록시의 출발지 IP 또는 CIDR을 나열합니다. 기본값은 비어 있습니다. `PROXY_HEADER`는 클라이언트 IP를 담는 헤더 이름입니다(기본값: `X-Forwarded-For`).

`TRUSTED_PROXIES` 없이도 Stargate는 시작되지만, 리버스 프록시 뒤에서는 반드시 설정하세요. 프록시가 목록에 없는 동안 Stargate는 프록시가 `/_auth`로 보낸 요청만 볼 수 있고 사용자가 보낸 요청은 볼 수 없습니다:

- 로그인 리디렉션에 `callback`이 없고 `http://`로 만들어지므로, 사용자는 로그인 후 애플리케이션으로 돌아가지 못하고 인증 호스트에 머무릅니다.
- 프록시가 HTTPS를 종료하는 경우 사용자는 아예 로그인할 수 없습니다. 브라우저는 `Origin: https://...`를 보내고 Stargate는 이를 프록시로부터 받은 `http://` 요청과 비교하므로, `POST /_login`, `/_logout`, `/_step_up`, `/_send_verify_code`, `/_send_reauth_code`, `/totp/*`는 `403`과 `cross-site request rejected`로 응답됩니다.
- step-up 경로 규칙을 평가할 수 없습니다(`STEP_UP_ENABLED=true`가 `TRUSTED_PROXIES`를 요구하는 이유입니다).
- 모든 클라이언트가 프록시 주소로 취급되어 하나의 요청 속도 제한 할당량과 감사 기록의 주소 하나를 공유합니다.

Stargate는 `TRUSTED_PROXIES`가 비어 있으면 시작 시 경고를 기록하고, 목록에 없는 피어의 전달 헤더를 처음 무시할 때 한 번 더 경고를 기록합니다. 또한 무시된 `X-Forwarded-Proto` 및 `X-Forwarded-Host`와 `Origin`이 일치하는 요청을 교차 사이트 요청으로 거부할 때에도 경고를 한 번만 기록합니다. `403` 응답 자체에는 원인이 표시되지 않습니다.

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

### `SESSION_EXCHANGE_REDIRECT_PATH`

검증된 세션 교환 후 콜백과 같은 출처에서 이동할 경로입니다. 기본값은 `/`입니다. 앱의 자체 로그인을 완료하려면 `/auth/login/example/` 등을 설정합니다. 관리자 설정의 절대 경로만 허용하며 쿼리, 프래그먼트, 퍼센트 인코딩 및 제어 문자는 거부합니다. 콜백 허용 목록과 재사용 방지 검증은 변경되지 않습니다.

### `PORT`

네이티브 실행 및 컨테이너 배포의 서비스 리스닝 포트. config 패키지에서 다른 환경 변수와 함께 로드 및 검증됩니다.

| 속성 | 값 |
|------|-----|
| **타입** | String |
| **필수** | 아니오 |
| **기본값** | 네이티브 바이너리는 비어 있음 (`:80`); 공식 컨테이너 이미지는 `PORT=8080` 설정 |

**설명:**

- 개발 및 운영 환경에서 설정 가능
- 공식 컨테이너의 기본값은 `8080`입니다. 상태 확인이 URL을 구성하므로 재정의할 때는 `PORT=8080`처럼 숫자만 사용합니다.
- 네이티브 바이너리 형식: 포트 번호 (예: `8080`) 또는 `:port` (예: `:8080`)

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
| `SESSION_EXCHANGE_REDIRECT_PATH` | 경로 | `/` | 아니요 |
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
| `WARDEN_CACHE_TTL` | 양의 정수(초) | 300 | 호환성을 위해서만 허용됨. 효과 없음 |
| `WARDEN_SOURCE_MAX_AGE` | 기간 | 0s | Warden 원본 데이터의 최대 경과 시간; 0s는 검사 비활성화 |
| `WARDEN_SOURCE_CLOCK_SKEW` | 기간 | 0s | 허용 시계 오차; 최대 경과 시간보다 작아야 함 |
| `METRICS_API_KEY` | 비밀 값 | 비어 있음 | /metrics 전용 Bearer 키; 32바이트 이상 |
| `HERALD_ENABLED` | true/false | false | Herald 연동 |
| `HERALD_URL` | URL | 비어 있음 | Herald 엔드포인트 |
| `HERALD_API_KEY` | 비밀값 | 비어 있음 | API 키 인증 |
| `HERALD_HMAC_KEY_ID` | 문자열 | 비어 있음 | HMAC 키 ID |
| `HERALD_HMAC_SECRET` | 비밀값 | 비어 있음 | HMAC 서명 비밀 |
| `HERALD_CLIENT_IP_HEADER` | 헤더 이름 | 비어 있음 | Stargate가 클라이언트 주소를 Herald에 전달하는 헤더. Herald의 `HERALD_TRUSTED_PROXY_HEADER`와 같아야 하며 Stargate가 `HERALD_TRUSTED_PROXIES`에 포함되어야 합니다 |
| `HERALD_TLS_CA_CERT_FILE` | 경로 | 비어 있음 | 사용자 지정 CA |
| `HERALD_TLS_CLIENT_CERT_FILE` | 경로 | 비어 있음 | mTLS 클라이언트 인증서 |
| `HERALD_TLS_CLIENT_KEY_FILE` | 경로 | 비어 있음 | mTLS 클라이언트 키 |
| `HERALD_TLS_SERVER_NAME` | 문자열 | 비어 있음 | 예상 TLS 서버 이름 |
| `HERALD_TOTP_ENABLED` | true/false | false | TOTP 로그인 및 관리 |
| `LOGIN_SMS_ENABLED` | true/false | true | SMS 로그인 채널 |
| `LOGIN_EMAIL_ENABLED` | true/false | true | 이메일 로그인 채널 |
| `LOGIN_DINGTALK_REQUIRE_USERID` | true/false | false | DingTalk에 명시적인 Warden userid 요구 |
| `LOGIN_UNIFORM_RESPONSES` | true/false | false | 식별자가 Warden에 등록되어 있는지와 관계없이 `/_send_verify_code`와 Warden 로그인의 응답을 동일하게 함. 응답 시간은 맞추지 않음 |
| `LOGIN_UNIFORM_RESEND_COOLDOWN` | 기간 | `60s` | `LOGIN_UNIFORM_RESPONSES=true`일 때 식별자별 재전송 대기 시간. `1s` 이상이며 Herald의 `RESEND_COOLDOWN` 이상이어야 함 |
| `SESSION_STORAGE_ENABLED` | true/false | false | Redis 세션 저장소 |
| `SESSION_STORAGE_REDIS_ADDR` | host:port | `localhost:6379` | Redis 주소 |
| `SESSION_STORAGE_REDIS_PASSWORD` | 비밀값 | 비어 있음 | Redis 비밀번호 |
| `SESSION_STORAGE_REDIS_DB` | 정수 | 0 | Redis DB |
| `SESSION_STORAGE_REDIS_KEY_PREFIX` | 문자열 | `stargate:session:` | Redis 키 접두사 |
| `SESSION_STORAGE_REDIS_USERNAME` | 문자열 | 비어 있음 | Redis ACL 사용자. `SESSION_STORAGE_REDIS_PASSWORD` 필요 |
| `SESSION_STORAGE_REDIS_TLS_ENABLED` | true/false | false | Redis 연결에 TLS를 사용하고 서버 인증서를 검증 |
| `SESSION_STORAGE_REDIS_TLS_CA_CERT_FILE` | 경로 | 비어 있음 | Redis용 사용자 지정 CA. `SESSION_STORAGE_REDIS_TLS_ENABLED=true` 필요 |
| `SESSION_STORAGE_REDIS_TLS_SERVER_NAME` | 문자열 | 비어 있음 | Redis의 예상 TLS 서버 이름. `SESSION_STORAGE_REDIS_TLS_ENABLED=true` 필요 |
| `AUDIT_LOG_ENABLED` | true/false | true | 감사 로그 |
| `AUDIT_LOG_FORMAT` | json/text | json | 감사 로그 형식 |
| `STEP_UP_ENABLED` | true/false | false | 추가 인증 |
| `STEP_UP_PATHS` | 경로 목록 | 비어 있음 | 보호할 업무 경로 |
| `OTLP_ENABLED` | true/false | false | OpenTelemetry 내보내기 |
| `OTLP_ENDPOINT` | URL | 비어 있음 | OTLP 엔드포인트 |
| `AUTH_REFRESH_ENABLED` | true/false | true | 권한 정보 주기적 갱신 |
| `AUTH_REFRESH_INTERVAL` | 기간 | `5m` | 갱신 간격 |
| `REQUEST_CONTEXT_TIMEOUT` | 기간 | `10s` | 업스트림 호출 기한. 클라이언트 연결 종료 시 즉시 취소는 보장하지 않음 |
| `RATE_LIMIT_LOGIN_MAX` | 정수 | 10 | 클라이언트별·윈도우별 로그인 시도 수. 0이면 비활성화 |
| `RATE_LIMIT_VERIFICATION_MAX` | 정수 | 5 | 클라이언트별·윈도우별 인증 코드 요청 수. 0이면 비활성화 |
| `RATE_LIMIT_WINDOW` | 기간 | `1m` | 레이트 리밋 고정 윈도우 |

`LOGIN_DINGTALK_REQUIRE_USERID`: 기본값은 `false`(호환 모드)입니다. `true`이면 Warden이 서버에서 관리하는 `dingtalk_userid`만 허용하며 전화번호로 대체하지 않습니다. 매핑이 없거나 유효하지 않으면 거부합니다. 코드 요청과 기존 바인딩 검증에 같은 규칙이 적용됩니다. 자동으로 활성화되지 않으며 기존 세션이나 캐시의 폐기 경계를 변경하지 않습니다. 자세한 내용은 [영문 userid 프로필](../enUS/DINGTALK_USERID_ONLY.md)을 참조하세요.

`PORT`는 개발과 프로덕션 모두에 적용됩니다. 환경 값이 없는 네이티브 바이너리는 `:80`, 공식 이미지는 `PORT=8080`을 사용합니다. 컨테이너에서는 `PORT=8080`처럼 숫자만 지정하세요. 상태 검사가 URL을 구성하므로 네이티브의 `:8080` 형식은 적합하지 않습니다.

Warden의 HMAC Key ID와 Secret은 함께 설정합니다. Herald 서버가 `HERALD_HMAC_KEYS`를 사용하면 명시적 ID가 필요하지만 단일 `HMAC_SECRET`은 암시적 기본값을 지원합니다. 두 서비스 모두 mTLS 클라이언트 인증서와 키를 완전한 쌍으로 설정해야 합니다.

`WARDEN_URL` 또는 `HERALD_URL`이 `http://` URL이고 API 키 또는 HMAC Secret이 설정되어 있으면 Stargate는 시작 시 경고를 기록합니다. 이 경우 API 키와 모든 요청 및 응답 데이터가 암호화되지 않은 채 네트워크를 지나갑니다. Warden 또는 Herald와의 연결이 Compose 내부 네트워크 같은 사설 네트워크 안에만 머무르는 경우가 아니라면 `https://`를 사용하세요.

`WARDEN_CACHE_TTL`은 호환성을 위해서만 허용되며 Stargate의 동작에 영향을 주지 않습니다. 이 값은 사용자 목록 캐시의 수명으로 Warden 클라이언트에 전달되지만, Stargate는 사용자 목록을 요청하지 않습니다. 로그인, 코드 전송, 신뢰할 수 있는 헤더 인증, 권한 갱신은 모두 전화번호 또는 메일 주소로 사용자 한 명을 조회하며, 이 조회는 캐시되지 않습니다. 조회할 때마다 Warden에 요청이 발생합니다. 비활성화되거나 삭제된 사용자가 기존 세션을 잃기까지의 시간과 하나의 세션이 Warden 요청을 발생시키는 빈도는 `AUTH_REFRESH_INTERVAL`로 정해집니다. 간격을 늘리면 Warden 요청은 줄지만 사용자 비활성화가 반영되는 시점도 늦어집니다. `HEADER_AUTH_ENABLED=true`인 경우 신뢰할 수 있는 ID 헤더를 포함한 전달 요청마다 Warden 조회가 한 번 발생합니다. 값은 여전히 양의 정수여야 하며, 그렇지 않으면 Stargate가 시작되지 않습니다.

`SESSION_STORAGE_REDIS_*` 설정은 `SESSION_STORAGE_ENABLED=true`일 때만 적용됩니다. `SESSION_STORAGE_ENABLED`가 `false`인 상태에서 그중 하나라도 기본값이 아닌 값으로 설정되어 있으면, Stargate는 시작되고 세션과 요청 속도 제한 카운터를 메모리에 보관하며, 무시된 변수를 나열한 경고를 시작 시 기록합니다.

`SESSION_STORAGE_REDIS_USERNAME`의 사용자에게는 `SESSION_STORAGE_REDIS_KEY_PREFIX` 아래 키에 대한 읽기, 쓰기, 스크립트(`EVAL`/`EVALSHA`) 권한이 필요합니다. `SESSION_STORAGE_REDIS_TLS_ENABLED=true`가 아니면 세션, 인증 컨텍스트, 요청 속도 제한 카운터가 암호화되지 않은 채 Redis로 전송되므로 사설 네트워크에서만 사용하세요. TLS(1.2 이상)에서는 `SESSION_STORAGE_REDIS_TLS_CA_CERT_FILE`(추가 CA 인증서가 담긴 PEM 파일)과 `SESSION_STORAGE_REDIS_TLS_SERVER_NAME`으로 변경하지 않는 한, 인증서를 시스템 루트 인증서와 `SESSION_STORAGE_REDIS_ADDR`의 호스트에 대해 검증합니다. 검증은 끌 수 없으며 클라이언트 인증서는 지원하지 않습니다.

## 비밀번호 설정

운영 환경에서는 `bcrypt`를 사용하며 `plaintext`는 로컬 테스트에서만 사용할 수 있습니다. MD5와 salt 없는 SHA-512는 설정 검증에서 거부됩니다. 비밀번호는 대소문자와 공백을 그대로 구분합니다. 이 저장소의 Compose 예제에 공개된 데모 비밀번호(`test1234`, `test1337`)는 `COOKIE_SECURE`가 `true`이면 시작 시 거부되며, 로컬 HTTP 데모에서만 허용됩니다.

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
    image: ghcr.io/soulteary/stargate:1.3.8
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
5. **규칙 검증**: 일부 설정 항목은 다른 설정 항목에 의존하거나 최소 길이 또는 형식 요건이 있습니다 (예: `HERALD_ENABLED`는 `WARDEN_ENABLED=true` 필요)

메시지는 변수 이름을 표시하고 위의 경우 중 어느 것에 해당하는지 알려 줍니다. 규칙은 이유로 보고되며 설정된 값처럼 표시되지 않습니다. 비밀값의 설정 값은 절대 출력되지 않습니다.

**오류 예제:**

```bash
# 필수 설정 부족
Error: Configuration error: environment variable 'AUTH_HOST' is required but not set.

# 비밀번호 형식이 올바르지 않음
Error: Configuration error: invalid value for environment variable 'PASSWORDS': 'invalid_format'

# 지원되지 않는 알고리즘
Error: Configuration error: invalid value for environment variable 'PASSWORDS': 'unknown:password'

# 허용되는 값 집합을 벗어난 값
Error: Configuration error: invalid value for environment variable 'DEBUG': 'maybe'.
  Accepted values: [true false]

# 다른 설정과 관련된 규칙
Error: Configuration error: environment variable 'HERALD_ENABLED': requires WARDEN_ENABLED=true to resolve authenticated users.
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

`/_auth`에서 이 헤더로 실패한 시도는 클라이언트 주소별로 `RATE_LIMIT_LOGIN_MAX` 할당량에 집계됩니다. 잘못된 값만 집계에 남습니다. 할당량을 다 쓰면 윈도우가 끝날 때까지 해당 주소에서 이 헤더를 포함해 보낸 모든 요청은 비밀번호를 비교하기 전에 `429`와 `Retry-After`로 응답되며, 비밀번호가 올바른 경우도 마찬가지입니다. 요청은 비밀번호를 비교하는 동안 집계되고 비밀번호가 올바르면 집계에서 빠집니다. 따라서 동시에 보낸 잘못된 값들이 합쳐서 할당량을 넘을 수 없습니다. 같은 이유로, 남은 할당량보다 많은 헤더 포함 요청을 동시에 보낸 클라이언트는 초과분에 `429`를 받지만 다시 보낼 수 있습니다. 같은 주소를 공유하는 클라이언트는 이 할당량을 공유합니다:

- 같은 NAT 또는 아웃바운드 프록시 뒤의 클라이언트는 하나의 주소입니다. 그중 하나가 잘못된 값을 보내면 윈도우의 남은 시간 동안 다른 클라이언트도 차단되며, 이는 윈도우마다 반복될 수 있습니다.
- `TRUSTED_PROXIES`가 비어 있거나 거기에 나열되지 않은 프록시 또는 CDN 뒤에 있으면 모든 요청이 프록시의 것으로 취급되므로, 배포에서 이 헤더를 사용하는 모든 클라이언트가 영향을 받습니다. `PASSWORD_HEADER_AUTH_ENABLED=true`이고 `TRUSTED_PROXIES`가 비어 있으면 Stargate는 시작 시 경고를 기록합니다.

할당량이 클라이언트별로 적용되도록 `TRUSTED_PROXIES`를 설정하세요. Cookie 세션은 이 할당량의 영향을 받지 않습니다.

## 원본 데이터 유효성 및 모니터링

호환성을 위해 `WARDEN_SOURCE_MAX_AGE`의 기본값은 `0s`입니다.
양수를 설정하려면 Warden 메타데이터 v1과 `AUTH_REFRESH_ENABLED=true`가
필요합니다. `WARDEN_SOURCE_CLOCK_SKEW`는 유효 기간에서 차감됩니다.
세션이 원본 데이터의 유효 기간을 연장하지 않으며, 원본 장애가 발생하면
HTTP 503으로 접근을 거부합니다.

`METRICS_API_KEY`가 없으면 `/metrics`는 HTTP 404를 반환합니다.
업데이트 전에 32바이트 이상의 별도 무작위 키를 설정하고 모니터링 클라이언트가
`Authorization: Bearer <secret>`를 전송하도록 구성하세요. 서비스 키나 로그인
비밀번호를 재사용하지 마세요. 호스트 간 통신에는 인증서를 검증하는 HTTPS를
사용하세요. HTTP는 신뢰할 수 있는 격리된 로컬 네트워크를 전제로 합니다.
전체 규칙과 이전 절차는
[영문 설정 문서](../enUS/CONFIG.md#source-freshness-and-scrape-credentials)를 참조하세요.


### `LOGIN_SUPPORT_URL`

선택적인 HTTPS 지원 링크 또는 단일 `mailto:help@example.com` 주소입니다. 비어 있으면 링크를 숨깁니다. 인증 정보, 제어 문자, mailto 헤더는 허용되지 않습니다.
