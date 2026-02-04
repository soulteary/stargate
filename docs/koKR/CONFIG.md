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
PASSWORDS=plaintext:test123

# 여러 평문 비밀번호
PASSWORDS=plaintext:test123|admin456|user789

# BCrypt 해시
PASSWORDS=bcrypt:$2a$10$N9qo8uLOickgx2ZMRZoMyeIjZAgcfl7p92ldGxad68LJZdL17lhWy

# SHA512 해시
PASSWORDS=sha512:5e884898da28047151d0e56f8dc6292773603d0d6aabbdd62a11ef721d1542d8
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
LOGIN_PAGE_TITLE=내 인증 서비스
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
LOGIN_PAGE_FOOTER_TEXT=© 2024 내 회사
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
- 헤더의 값은 `authenticated`
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

## 비밀번호 설정

Stargate는 여러 비밀번호 암호화 알고리즘을 지원합니다. 비밀번호 설정 형식: `algorithm:password1|password2|password3`

### 지원되는 알고리즘

#### `plaintext` - 평문 비밀번호

**설명:**

- 평문으로 저장, 암호화 없음
- **테스트 환경 전용**
- 프로덕션 환경에서는 강력히 권장하지 않음

**예제:**

```bash
PASSWORDS=plaintext:test123|admin456
```

#### `bcrypt` - BCrypt 해시

**설명:**

- BCrypt 알고리즘을 사용하여 해시화
- 높은 보안성, 프로덕션 환경에서 권장
- 비밀번호는 BCrypt 해시 값을 사용해야 합니다

**BCrypt 해시 생성:**

```bash
# Go 사용
go run -c 'golang.org/x/crypto/bcrypt' <<< 'password'

# 온라인 도구 또는 기타 도구 사용
```

**예제:**

```bash
PASSWORDS=bcrypt:$2a$10$N9qo8uLOickgx2ZMRZoMyeIjZAgcfl7p92ldGxad68LJZdL17lhWy
```

#### `md5` - MD5 해시

**설명:**

- MD5 알고리즘을 사용하여 해시화
- 낮은 보안성, 프로덕션 환경에서는 권장하지 않음
- 비밀번호는 MD5 해시 값 (32자리 16진수 문자열)을 사용해야 합니다

**MD5 해시 생성:**

```bash
# Linux/macOS
echo -n "password" | md5sum

# 또는 온라인 도구 사용
```

**예제:**

```bash
PASSWORDS=md5:5e884898da28047151d0e56f8dc6292773603d0d6aabbdd62a11ef721d1542d8
```

#### `sha512` - SHA512 해시

**설명:**

- SHA512 알고리즘을 사용하여 해시화
- 높은 보안성, 프로덕션 환경에서 권장
- 비밀번호는 SHA512 해시 값 (128자리 16진수 문자열)을 사용해야 합니다

**SHA512 해시 생성:**

```bash
# Linux/macOS
echo -n "password" | shasum -a 512

# 또는 온라인 도구 사용
```

**예제:**

```bash
PASSWORDS=sha512:5e884898da28047151d0e56f8dc6292773603d0d6aabbdd62a11ef721d1542d8
```

### 비밀번호 검증 규칙

1. **비밀번호 정규화**: 검증 전에 공백을 제거하고 대문자로 변환
2. **여러 비밀번호 지원**: 여러 비밀번호를 설정할 수 있으며, 검증을 통과한 비밀번호는 모두 허용됩니다
3. **알고리즘 일관성**: 모든 비밀번호는 동일한 알고리즘을 사용해야 합니다

## 설정 예제

### 기본 설정

```bash
# 필수 설정
AUTH_HOST=auth.example.com
PASSWORDS=plaintext:test123

# 선택적 설정
DEBUG=false
LANGUAGE=en
```

### 프로덕션 환경 설정

```bash
# 필수 설정
AUTH_HOST=auth.example.com
PASSWORDS=bcrypt:$2a$10$N9qo8uLOickgx2ZMRZoMyeIjZAgcfl7p92ldGxad68LJZdL17lhWy

# 선택적 설정
DEBUG=false
LANGUAGE=ko
LOGIN_PAGE_TITLE=내 인증 서비스
LOGIN_PAGE_FOOTER_TEXT=© 2024 내 회사
USER_HEADER_NAME=X-Forwarded-User
COOKIE_DOMAIN=.example.com
```

### Docker Compose 설정

```yaml
services:
  stargate:
    image: stargate:latest
    environment:
      # 필수 설정
      - AUTH_HOST=auth.example.com
      - PASSWORDS=bcrypt:$2a$10$N9qo8uLOickgx2ZMRZoMyeIjZAgcfl7p92ldGxad68LJZdL17lhWy
      
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
PASSWORDS=plaintext:test123|admin456

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
   - `bcrypt` 또는 `sha512` 알고리즘을 사용하고, `plaintext`를 피하세요
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
