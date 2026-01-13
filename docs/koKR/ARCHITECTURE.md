# Stargate 아키텍처 문서

이 문서는 Stargate 프로젝트의 기술 아키텍처와 설계 결정을 설명합니다.

## 기술 스택

- **언어**: Go 1.25
- **웹 프레임워크**: [Fiber v2.52.10](https://github.com/gofiber/fiber)
- **템플릿 엔진**: [Fiber Template v1.7.5](https://github.com/gofiber/template)
- **세션 관리**: Fiber Session Middleware
- **로깅**: [Logrus v1.9.3](https://github.com/sirupsen/logrus)
- **터미널 출력**: [Pterm v0.12.82](https://github.com/pterm/pterm)
- **테스트 프레임워크**: [Testza v0.5.2](https://github.com/MarvinJWendt/testza)

## 프로젝트 구조

```
codes/src/
├── cmd/stargate/          # 애플리케이션 진입점
│   ├── main.go            # 메인 함수, 설정 초기화 및 서버 시작
│   ├── server.go          # 서버 설정 및 라우트 설정
│   └── constants.go       # 라우트 및 설정 상수
│
├── internal/              # 내부 패키지 (외부에 노출되지 않음)
│   ├── auth/              # 인증 로직
│   │   ├── auth.go        # 인증의 주요 기능
│   │   └── auth_test.go   # 인증 테스트
│   │
│   ├── config/            # 설정 관리
│   │   ├── config.go      # 설정 변수 정의 및 초기화
│   │   ├── validation.go  # 설정 검증 로직
│   │   └── config_test.go # 설정 테스트
│   │
│   ├── handlers/          # HTTP 요청 핸들러
│   │   ├── check.go       # 인증 확인 핸들러
│   │   ├── login.go       # 로그인 핸들러
│   │   ├── logout.go      # 로그아웃 핸들러
│   │   ├── session_share.go # 세션 공유 핸들러
│   │   ├── health.go       # 헬스 체크 핸들러
│   │   ├── index.go        # 루트 경로 핸들러
│   │   ├── utils.go        # 핸들러 유틸리티 함수
│   │   └── handlers_test.go # 핸들러 테스트
│   │
│   ├── i18n/              # 국제화 지원
│   │   └── i18n.go        # 다국어 번역
│   │
│   ├── middleware/        # HTTP 미들웨어
│   │   └── log.go         # 로그 미들웨어
│   │
│   ├── secure/            # 비밀번호 암호화 알고리즘
│   │   ├── interface.go   # 암호화 알고리즘 인터페이스
│   │   ├── plaintext.go    # 평문 비밀번호 (테스트만)
│   │   ├── bcrypt.go       # BCrypt 알고리즘
│   │   ├── md5.go          # MD5 알고리즘
│   │   ├── sha512.go       # SHA512 알고리즘
│   │   └── secure_test.go  # 암호화 알고리즘 테스트
│   │
│   └── web/               # 웹 리소스
│       └── templates/     # HTML 템플릿
│           ├── login.html # 로그인 페이지 템플릿
│           └── assets/   # 정적 리소스
│               └── favicon.ico
```

## 주요 컴포넌트

### 1. 인증 시스템 (`internal/auth`)

인증 시스템은 다음을 담당합니다:
- 비밀번호 검증 (여러 암호화 알고리즘 지원)
- 세션 관리 (생성, 검증, 파괴)
- 인증 상태 검증

**주요 함수:**
- `CheckPassword(password string) bool`: 비밀번호 검증
- `Authenticate(session *session.Session) error`: 세션을 인증됨으로 표시
- `IsAuthenticated(session *session.Session) bool`: 세션이 인증되었는지 확인
- `Unauthenticate(session *session.Session) error`: 세션 파괴

### 2. 설정 시스템 (`internal/config`)

설정 시스템은 다음을 제공합니다:
- 환경 변수 관리
- 설정 검증
- 기본값 지원

**설정 변수:**
- `AUTH_HOST`: 인증 서비스 호스트명 (필수)
- `PASSWORDS`: 비밀번호 설정 (알고리즘:비밀번호 목록) (필수)
- `DEBUG`: 디버그 모드 (기본값: false)
- `LANGUAGE`: 인터페이스 언어 (기본값: en, en/zh/fr/it/ja/de/ko 지원)
- `COOKIE_DOMAIN`: Cookie 도메인 (선택 사항, 크로스 도메인 세션 공유용)
- `LOGIN_PAGE_TITLE`: 로그인 페이지 제목 (기본값: Stargate - Login)
- `LOGIN_PAGE_FOOTER_TEXT`: 로그인 페이지 푸터 텍스트 (기본값: Copyright © 2024 - Stargate)
- `USER_HEADER_NAME`: 인증 성공 후 설정되는 사용자 헤더 이름 (기본값: X-Forwarded-User)
- `PORT`: 서비스 리스닝 포트 (로컬 개발만, 기본값: 80)

### 3. 요청 핸들러 (`internal/handlers`)

핸들러는 HTTP 요청 처리를 담당합니다:

- **CheckRoute**: Traefik Forward Auth 인증 확인
- **LoginRoute/LoginAPI**: 로그인 페이지 및 로그인 처리
- **LogoutRoute**: 로그아웃 처리
- **SessionShareRoute**: 크로스 도메인 세션 공유
- **HealthRoute**: 헬스 체크
- **IndexRoute**: 루트 경로 처리

### 4. 비밀번호 암호화 (`internal/secure`)

여러 비밀번호 암호화 알고리즘 지원:
- `plaintext`: 평문 (테스트만)
- `bcrypt`: BCrypt 해시
- `md5`: MD5 해시
- `sha512`: SHA512 해시

모든 알고리즘은 `HashResolver` 인터페이스를 구현합니다:
```go
type HashResolver interface {
    Check(h string, password string) bool
}
```

## 워크플로우

### 인증 플로우

1. **사용자가 보호된 리소스에 액세스**
   - Traefik이 요청을 가로챔
   - Stargate 엔드포인트 `/_auth`로 전달

2. **Stargate가 인증을 검증**
   - 먼저 `Stargate-Password` 헤더 확인 (API 인증)
   - 헤더 인증이 실패하면 `stargate_session_id` Cookie 확인 (웹 인증)

3. **인증 성공**
   - `X-Forwarded-User` 헤더 (또는 설정된 사용자 헤더 이름)에 "authenticated" 설정
   - 200 OK 반환
   - Traefik이 요청 계속을 허용

4. **인증 실패**
   - HTML 요청: 로그인 페이지로 리디렉션 (`/_login?callback=<originalURL>`)
   - API 요청 (JSON/XML): 401 Unauthorized 반환

### 로그인 플로우

1. **사용자가 로그인 페이지에 액세스**
   - `GET /_login?callback=<url>`
   - 이미 로그인한 경우, 세션 교환 엔드포인트로 리디렉션
   - 도메인이 다른 경우, 콜백을 Cookie (`stargate_callback`)에 저장

2. **로그인 폼 제출**
   - `POST /_login`에 비밀번호
   - 비밀번호 검증
   - 세션 생성 및 Cookie 설정
   - **콜백 가져오기 우선순위**:
     1. Cookie에서 (이전에 설정된 경우)
     2. 폼 데이터에서
     3. 쿼리 매개변에서
     4. 위의 항목이 없고 원본 도메인이 인증 서비스 도메인과 다른 경우, 원본 도메인을 콜백으로 사용

3. **세션 교환**
   - 콜백이 있는 경우, `{callback}/_session_exchange?id=<session_id>`로 리디렉션
   - `GET /_session_exchange?id=<session_id>`
   - 세션 Cookie 설정 (`COOKIE_DOMAIN`이 설정된 경우, 지정된 도메인에 설정)
   - 루트 경로 `/`로 리디렉션

## 보안 고려 사항

### 세션 보안

- Cookie는 XSS 공격을 방지하기 위해 `HttpOnly` 플래그 사용
- Cookie는 CSRF 공격을 방지하기 위해 `SameSite=Lax` 사용
- Cookie 경로는 `/`로 설정되어 전체 도메인에서 사용 가능
- 세션 만료 시간: 24시간 (`config.SessionExpiration`)
- 사용자 정의 Cookie 도메인 지원 (크로스 도메인 시나리오용)
- 세션 ID는 UUID를 사용하여 생성되어 고유성과 보안 보장

### 비밀번호 보안

- 여러 암호화 알고리즘 지원 (bcrypt 또는 sha512 사용 권장)
- 비밀번호 설정은 환경 변수를 통해 전달되며 코드에 저장되지 않음
- 검증 시 비밀번호 정규화 (공백 제거, 대문자로 변환)

### 요청 보안

- 인증 확인 엔드포인트는 두 가지 인증 방법을 지원합니다:
  - 헤더 인증 (`Stargate-Password`): API 요청용
  - Cookie 인증: 웹 요청용
- HTML 요청과 API 요청을 구분하여 적절한 응답 반환

## 확장성

### 새로운 비밀번호 알고리즘 추가

1. `internal/secure/`에 새로운 알고리즘 구현 생성
2. `HashResolver` 인터페이스 구현
3. `config/validation.go`에 알고리즘 등록

### 새로운 언어 추가

1. `internal/i18n/i18n.go`에 언어 상수 추가
2. 번역 매핑 추가
3. 설정에 언어 옵션 추가

### 로그인 페이지 사용자 정의

템플릿 파일 `internal/web/templates/login.html`을 수정합니다.

## 성능 최적화

- Fiber 프레임워크 사용, fasthttp 기반으로 우수한 성능
- 세션은 메모리에 저장되어 빠른 액세스
- 정적 리소스는 Fiber의 정적 파일 서비스를 통해 제공
- 디버그 모드 지원, 프로덕션 환경에서 비활성화 가능

## 배포 아키텍처

### Docker 배포

- 이미지 크기 감소를 위한 다단계 빌드
- 빌드 단계로 `golang:1.25-alpine` 사용
- 보안 위험을 최소화하기 위해 실행 단계로 `scratch` 기본 이미지 사용
- 템플릿 파일을 `src/internal/web/templates`에서 이미지 내 `/app/web/templates`로 복사
- 의존성 다운로드 가속을 위해 중국 미러 소스 (`GOPROXY=https://goproxy.cn`) 사용
- 바이너리 크기 감소를 위해 컴파일 시 `-ldflags "-s -w"` 사용
- 애플리케이션은 자동으로 템플릿 경로를 찾습니다 (로컬 개발용 `./internal/web/templates`, 프로덕션용 `./web/templates` 지원)

### Traefik 통합

- Forward Auth 미들웨어를 통해 통합
- HTTP 및 HTTPS 지원
- 여러 도메인 및 경로 규칙 지원

## 로깅 및 모니터링

- 로깅에는 Logrus 사용
- 디버그 모드 지원 (DEBUG=true)
- 모든 중요한 작업이 로깅됨
- 모니터링을 위한 헬스 체크 엔드포인트 사용 가능

## 테스트

- 단위 테스트는 주요 기능을 다룹니다
- 테스트 파일은 각 패키지의 `*_test.go` 파일에 위치
- 어설션에 `testza` 사용
- 테스트 커버리지에는 다음이 포함됩니다:
  - 인증 로직 (`internal/auth/auth_test.go`)
  - 설정 검증 (`internal/config/config_test.go`)
  - 비밀번호 암호화 알고리즘 (`internal/secure/secure_test.go`)
  - HTTP 핸들러 (`internal/handlers/handlers_test.go`)

## 향후 개선 사항

- [ ] 더 많은 비밀번호 암호화 알고리즘 지원
- [ ] OAuth2/OpenID Connect 지원
- [ ] 다중 사용자 및 역할 관리 지원
- [ ] 관리 인터페이스 추가
- [ ] 외부 세션 스토리지 (Redis 등) 지원
- [ ] Prometheus 메트릭 내보내기 추가
- [ ] 설정 파일 (YAML/JSON) 지원
