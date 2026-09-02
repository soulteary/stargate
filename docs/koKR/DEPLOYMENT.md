# 배포 가이드

이 문서는 Stargate Forward Auth 서비스의 자세한 배포 가이드를 제공합니다.

v0.12.0에서 업그레이드할 때는 먼저 [v1.0.0 마이그레이션 가이드](MIGRATION_V1.md)를 확인하십시오.

## 목차

- [배포 방법](#배포-방법)
- [Docker 배포](#docker-배포)
- [Docker Compose 배포](#docker-compose-배포)
- [Traefik 통합](#traefik-통합)
- [프로덕션 환경 배포](#프로덕션-환경-배포)
- [모니터링 및 유지보수](#모니터링-및-유지보수)
- [문제 해결](#문제-해결)

## 배포 방법

Stargate는 다음 배포 방법을 지원합니다:

1. **Docker 컨테이너** (권장) - 가장 간단하고 일반적
2. **Docker Compose** - 로컬 개발 및 테스트에 적합
3. **Kubernetes** - 대규모 프로덕션 환경에 적합
4. **바이너리 직접 실행** - 특수 시나리오에 적합

이 문서에서는 주로 Docker와 Docker Compose 배포 방법을 소개합니다.

## 서비스 종속성

Stargate는 다음 선택적 서비스와 통합할 수 있습니다:

### Warden 서비스

**기능:** 사용자 화이트리스트 관리 및 사용자 정보 제공

**배포 요구 사항:**
- 데이터베이스 필요 (PostgreSQL/MySQL/SQLite)
- HTTP API 인터페이스 제공
- API 키 인증 지원

**설정:**
```bash
WARDEN_ENABLED=true
WARDEN_URL=http://warden:8080
WARDEN_API_KEY=your-api-key
```

### Herald 서비스

**기능:** OTP/인증 코드 전송 및 검증

**배포 요구 사항:**
- Redis 필요 (챌린지 및 속도 제한 상태 저장)
- HTTP API 인터페이스 제공
- HMAC 서명 또는 mTLS 인증 지원 (프로덕션 환경에서 권장)

**설정:**
```bash
HERALD_ENABLED=true
HERALD_URL=http://herald:8080
HERALD_HMAC_SECRET=your-hmac-secret  # 프로덕션 환경에서 권장
```

### 서비스 간 통신 보안

**프로덕션 환경 요구 사항:**

1. **HMAC 서명 인증** (권장):
   - Stargate ↔ Herald는 HMAC-SHA256 서명 사용
   - `HERALD_HMAC_SECRET` 설정
   - 타임스탬프 검증 포함 (재생 공격 방지)

2. **mTLS 인증** (선택 사항, 더 안전):
   - TLS 클라이언트 인증서 설정
   - `HERALD_TLS_CLIENT_CERT_FILE` 및 `HERALD_TLS_CLIENT_KEY_FILE` 설정
   - CA 인증서 검증 설정

3. **네트워크 격리:**
   - 서비스 간 통신은 내부 네트워크에서 이루어져야 합니다
   - 방화벽 규칙을 사용하여 액세스 제한
   - 서비스를 공용 네트워크에 노출하지 않도록 합니다

## Docker 배포

### 이미지 빌드

#### 소스에서 빌드

프로젝트 루트에서:

```bash
docker build -f docker/Dockerfile -t stargate:latest .
```

#### 빌드 매개변수

- **베이스 이미지**: `golang:1.27.0-alpine3.24` (빌드 단계)
- **실행 이미지**: `alpine:3.24` (CA 인증서와 HTTPS·헬스 체크용 BusyBox `wget` 포함)
- **작업 디렉토리**: `/app`
- **공개 포트**: `8080`

### 컨테이너 실행

#### 기본 실행

```bash
docker run -d \
  --name stargate \
  -p 8080:8080 \
  -e AUTH_HOST=auth.example.com \
  -e 'PASSWORDS=plaintext:yourpassword' \
  stargate:latest
```

#### 완전한 설정으로 실행

```bash
docker run -d \
  --name stargate \
  -p 8080:8080 \
  -e AUTH_HOST=auth.example.com \
  -e 'PASSWORDS=bcrypt:$2a$10$N9qo8uLOickgx2ZMRZoMyeIjZAgcfl7p92ldGxad68LJZdL17lhWy' \
  -e DEBUG=false \
  -e LANGUAGE=ko \
  -e 'LOGIN_PAGE_TITLE=내 인증 서비스' \
  -e 'LOGIN_PAGE_FOOTER_TEXT=© 2024 내 회사' \
  -e COOKIE_DOMAIN=.example.com \
  --restart unless-stopped \
  stargate:latest
```

#### 매개변수 설명

- `-d`: 백그라운드에서 실행
- `--name stargate`: 컨테이너 이름
- `-p 8080:8080`: 포트 매핑 (호스트 포트:컨테이너 포트)
- `-e`: 환경 변수
- `--restart unless-stopped`: 자동 재시작 정책

### 로그 표시

```bash
# 실시간으로 로그 표시
docker logs -f stargate

# 마지막 100줄의 로그 표시
docker logs --tail 100 stargate
```

### 중지 및 삭제

```bash
# 컨테이너 중지
docker stop stargate

# 컨테이너 삭제
docker rm stargate

```

## Docker Compose 배포

### 기본 설정

저장소 루트의 `docker-compose.yml`은 기본 제공 스택입니다. Compose가 관리하는 `stargate-traefik` 네트워크(`172.30.0.0/24`)에서 Traefik, Stargate 및 보호된 데모 서비스를 시작하므로 기존 외부 네트워크가 필요하지 않습니다.

```bash
docker compose config
docker compose up -d
```

### 기존 외부 Traefik 네트워크 사용

Traefik이 공유 외부 Docker 네트워크에서 이미 실행 중인 경우에만 아래의 별도 예제를 사용하십시오. 실제 네트워크 이름과 inspect로 확인한 CIDR이 Compose 네트워크, 모든 `traefik.docker.network` 레이블 및 `TRUSTED_PROXIES`에 반영됩니다.

```yaml
services:
  stargate:
    image: ghcr.io/soulteary/stargate:v1.0.0
    environment:
      - AUTH_HOST=auth.test.localhost
      - PASSWORDS=plaintext:test1234|test1337
      - CALLBACK_ALLOWED_HOSTS=whoami.test.localhost
      - SESSION_EXCHANGE_SECRET=local-development-session-secret-change-me
      - TRUSTED_PROXIES=${TRAEFIK_NETWORK_CIDR:?set TRAEFIK_NETWORK_CIDR from docker network inspect}
      - COOKIE_SECURE=false # Local HTTP only; omit for HTTPS.
    networks:
      - traefik
    labels:
      - traefik.enable=true
      - traefik.docker.network=${TRAEFIK_NETWORK_NAME:-traefik}
      - traefik.http.routers.auth.entrypoints=http
      - traefik.http.routers.auth.rule=Host(`auth.test.localhost`) || Path(`/_session_exchange`)
      - traefik.http.middlewares.stargate.forwardauth.address=http://stargate:8080/_auth
      - "traefik.http.middlewares.stargate.forwardauth.authResponseHeaders=X-Forwarded-User,X-Auth-User,X-Auth-Email,X-Auth-Name,X-Auth-Scopes,X-Auth-Role,X-Auth-AMR"

  whoami:
    image: traefik/whoami
    networks:
      - traefik
    labels:
      - traefik.enable=true
      - traefik.docker.network=${TRAEFIK_NETWORK_NAME:-traefik}
      - traefik.http.routers.whoami.entrypoints=http
      - traefik.http.routers.whoami.rule=Host(`whoami.test.localhost`)
      - "traefik.http.routers.whoami.middlewares=stargate"

networks:
  traefik:
    name: ${TRAEFIK_NETWORK_NAME:-traefik}
    external: true
```

### 외부 네트워크 검사 또는 생성

네트워크가 있으면 명령이 할당된 모든 CIDR을 검사하고 재사용합니다. 기존 프로덕션 네트워크를 삭제하거나 다시 만들지 마십시오. 네트워크가 없는 경우에만 문서화된 서브넷으로 생성합니다.

```bash
export TRAEFIK_NETWORK_NAME="${TRAEFIK_NETWORK_NAME:-traefik}"

if docker network inspect "$TRAEFIK_NETWORK_NAME" >/dev/null 2>&1; then
  export TRAEFIK_NETWORK_CIDR="$(
    docker network inspect "$TRAEFIK_NETWORK_NAME" \
      --format '{{range .IPAM.Config}}{{println .Subnet}}{{end}}' |
      awk 'NF { cidr = cidr separator $0; separator = "," } END { print cidr }'
  )"
  if [ -z "$TRAEFIK_NETWORK_CIDR" ]; then
    echo "No subnet found for Docker network $TRAEFIK_NETWORK_NAME" >&2
    exit 1
  fi
else
  docker network create \
    --driver bridge \
    --subnet 172.30.0.0/24 \
    "$TRAEFIK_NETWORK_NAME"
  export TRAEFIK_NETWORK_CIDR=172.30.0.0/24
fi

docker compose config
docker compose up -d
```

### 서비스 중지

```bash
docker compose down
```

### 로그 표시

```bash
# 모든 서비스의 로그 표시
docker compose logs -f

# 특정 서비스의 로그 표시
docker compose logs -f stargate
```

### 커스텀 설정

`docker-compose.yml`을 편집하고 환경 변수를 변경합니다:

```yaml
services:
  stargate:
    image: ghcr.io/soulteary/stargate:v1.0.0
    environment:
      - AUTH_HOST=auth.example.com
      - PASSWORDS=bcrypt:$$2a$$10$$...
      - TRUSTED_PROXIES=${TRAEFIK_NETWORK_CIDR:?set TRAEFIK_NETWORK_CIDR from docker network inspect}
      - DEBUG=false
      - LANGUAGE=ko
      - COOKIE_DOMAIN=.example.com
      - SESSION_EXCHANGE_SECRET=replace-with-at-least-32-random-characters
```

## Traefik 통합

### 기본 설정

Stargate는 Traefik과 통합하도록 설계되어 있으며, Forward Auth 미들웨어를 통해 인증을 제공합니다.

#### 1. Stargate 서비스 설정

`docker-compose.yml`에서 Stargate를 설정합니다:

```yaml
services:
  stargate:
    image: ghcr.io/soulteary/stargate:v1.0.0
    environment:
      - AUTH_HOST=auth.example.com
      - PASSWORDS=bcrypt:$$2a$$10$$...
      - TRUSTED_PROXIES=${TRAEFIK_NETWORK_CIDR:?set TRAEFIK_NETWORK_CIDR from docker network inspect}
    networks:
      - traefik
    labels:
      - "traefik.enable=true"
      - "traefik.docker.network=${TRAEFIK_NETWORK_NAME:-traefik}"
      - "traefik.http.routers.auth.entrypoints=http,https"
      - "traefik.http.routers.auth.rule=Host(`auth.example.com`) || Path(`/_session_exchange`)"
      - "traefik.http.middlewares.stargate.forwardauth.address=http://stargate:8080/_auth"
      - "traefik.http.middlewares.stargate.forwardauth.authResponseHeaders=X-Forwarded-User,X-Auth-User,X-Auth-Email,X-Auth-Name,X-Auth-Scopes,X-Auth-Role,X-Auth-AMR"
```

#### 2. 보호된 서비스 설정

인증이 필요한 서비스에 Stargate 미들웨어를 적용합니다:

```yaml
services:
  your-app:
    image: your-app:latest
    networks:
      - traefik
    labels:
      - "traefik.enable=true"
      - "traefik.docker.network=${TRAEFIK_NETWORK_NAME:-traefik}"
      - "traefik.http.routers.your-app.entrypoints=http,https"
      - "traefik.http.routers.your-app.rule=Host(`app.example.com`)"
      - "traefik.http.routers.your-app.middlewares=stargate"  # 인증 미들웨어 적용
```

### HTTPS 설정

#### Let's Encrypt 사용

```yaml
services:
  stargate:
    labels:
      - "traefik.http.routers.auth.entrypoints=https"
      - "traefik.http.routers.auth.tls.certresolver=letsencrypt"
      - "traefik.http.routers.auth.tls=true"
```

#### 커스텀 인증서 사용

```yaml
services:
  stargate:
    labels:
      - "traefik.http.routers.auth.entrypoints=https"
      - "traefik.http.routers.auth.tls=true"
      - "traefik.http.routers.auth.tls.certfile=/path/to/cert.pem"
      - "traefik.http.routers.auth.tls.keyfile=/path/to/key.pem"
```

### 크로스 도메인 세션 공유

서브도메인 간 세션을 공유해야 하는 경우:

1. 환경 변수 `COOKIE_DOMAIN`을 설정합니다:

```yaml
services:
  stargate:
    environment:
      - COOKIE_DOMAIN=.example.com
      - SESSION_EXCHANGE_SECRET=replace-with-at-least-32-random-characters
```

2. 관련된 모든 도메인이 Traefik을 통해 Stargate로 라우팅되는지 확인

3. 로그인 흐름:
   - 사용자가 `auth.example.com`에 로그인
   - `app.example.com/_session_exchange?ticket=<opaque_ticket>`로 리디렉션
   - 세션 Cookie가 `.example.com` 도메인에 설정됨
   - 모든 서브도메인 `*.example.com`에서 이 세션을 사용할 수 있습니다

## 프로덕션 환경 배포

### 보안 권장 사항

#### 1. 강력한 비밀번호 알고리즘 사용

**권장하지 않음:**

```bash
PASSWORDS='plaintext:yourpassword'
```

**권장:**

```bash
PASSWORDS='bcrypt:$2a$10$N9qo8uLOickgx2ZMRZoMyeIjZAgcfl7p92ldGxad68LJZdL17lhWy'
```

#### 2. HTTPS 활성화

- Traefik을 통해 HTTPS 설정
- 자동 Let's Encrypt 인증서 사용
- HTTPS 리디렉션 강제

#### 3. 디버그 모드 비활성화

```bash
DEBUG=false
```

#### 4. 리소스 제한 설정

```yaml
services:
  stargate:
    deploy:
      resources:
        limits:
          cpus: '0.5'
          memory: 128M
        reservations:
          cpus: '0.25'
          memory: 64M
```

#### 5. 헬스 체크 사용

```yaml
services:
  stargate:
    healthcheck:
      test: ["CMD", "wget", "-q", "-O", "-", "http://127.0.0.1:8080/healthz"]
      interval: 30s
      timeout: 10s
      retries: 3
      start_period: 40s
```

### 고가용성 배포

#### 1. 다중 인스턴스 배포

```yaml
services:
  stargate:
    deploy:
      replicas: 3
```

**세션 스토리지 적용 범위:**

- **단일 프로세스 / 단일 복제본(교차 도메인 Callback 포함):** Ticket 발급, 교환 및 이후 Session 사용이 모두 같은 실행 중인 Stargate 프로세스에 도달하면 인메모리 스토리지(`SESSION_STORAGE_ENABLED=false`)를 사용할 수 있습니다. Session과 소비된 Ticket 해시는 해당 프로세스에만 존재합니다. 재시작하면 두 상태가 사라지고 활성 Session이 무효화되며 Ticket의 일회성 상태도 유지되지 않습니다.
- **여러 프로세스 또는 복제본:** Session이나 교환 Ticket이 서로 다른 프로세스에서 처리될 수 있으면 Redis가 필수입니다. 모든 복제본에서 `SESSION_STORAGE_ENABLED=true`, 동일한 `SESSION_STORAGE_REDIS_*` 네임스페이스와 동일한 `SESSION_EXCHANGE_SECRET`을 사용합니다. Sticky Session은 라우팅 최적화일 뿐 공유 상태나 프로세스 간 재사용 방지를 대신하지 않습니다.
- **롤링 업그레이드:** 이전 프로세스와 새 프로세스가 겹치므로 무중단 교체에는 Redis가 필수입니다. Redis가 없으면 중지 후 시작하고 Session 무효화와 재로그인을 허용해야 합니다. v0.12.0과 v1.0.0을 같은 Serving Pool에 혼합하지 마십시오.
- **재시작 후 상태 유지:** Session 또는 소비된 Ticket의 재사용 방지 상태가 프로세스 교체나 재시작 후에도 유지되어야 하면 Redis가 필수입니다.

#### 2. 로드 밸런싱

Traefik 앞에 로드 밸런서를 추가:

```yaml
services:
  traefik:
    labels:
      - "traefik.http.services.stargate.loadbalancer.server.port=8080"
```

### 모니터링 설정

#### 1. 로그 수집

```yaml
services:
  stargate:
    logging:
      driver: "json-file"
      options:
        max-size: "10m"
        max-file: "3"
```

#### 2. 헬스 체크 엔드포인트

프로세스 liveness에는 `/healthz`, 의존성 readiness에는 `/readyz`를 사용합니다. `/health`는 더 이상 권장되지 않는 readiness 호환 별칭입니다.

```bash
# 헬스 체크 스크립트
#!/bin/bash
if curl -f http://localhost:8080/healthz > /dev/null 2>&1; then
  exit 0
else
  exit 1
fi
```

#### 3. Prometheus 통합

Stargate는 `GET /metrics`에서 Prometheus 메트릭을 노출합니다. Prometheus 서버에서 이 엔드포인트를 스크래핑하도록 설정하세요. 메트릭 엔드포인트는 기본적으로 요청 로그 및 트레이싱에서 제외됩니다.

## 모니터링 및 유지보수

### 로그 관리

#### 로그 표시

```bash
# Docker
docker logs -f stargate

# Docker Compose
docker compose logs -f stargate
```

#### 로그 레벨

- `DEBUG=true`: 자세한 디버깅 정보
- `DEBUG=false`: 중요한 정보만

#### 로그 로테이션

Docker 로그 드라이버 설정:

```yaml
services:
  stargate:
    logging:
      driver: "json-file"
      options:
        max-size: "10m"
        max-file: "3"
```

### 성능 모니터링

#### 리소스 사용량

```bash
# 컨테이너의 리소스 사용량 표시
docker stats stargate
```

#### 응답 시간

헬스 체크 엔드포인트를 사용하여 응답 시간 모니터링:

```bash
time curl http://auth.example.com/healthz
```

### 정기 유지보수

1. **이미지 업데이트**: 정기적으로 최신 이미지를 다운로드
2. **로그 확인**: 정기적으로 오류 로그 확인
3. **리소스 모니터링**: CPU 및 메모리 사용량 모니터링
4. **설정 백업**: 환경 변수 설정 백업

## 문제 해결

### 일반적인 문제

#### 1. 서비스가 시작되지 않음

**문제:** 컨테이너가 시작 직후 종료됨

**문제 해결 단계:**

```bash
# 컨테이너의 로그 표시
docker logs stargate

# 설정 확인
docker inspect stargate | grep -A 20 Env
```

**일반적인 원인:**

- 필수 설정 부족 (`AUTH_HOST`, `PASSWORDS`)
- 비밀번호 설정 형식이 올바르지 않음
- 포트가 사용 중

#### 2. 인증 실패

**문제:** 사용자가 로그인할 수 없음

**문제 해결 단계:**

1. 비밀번호 설정이 올바른지 확인
2. 비밀번호 알고리즘이 일치하는지 확인
3. 서비스 로그 표시: `docker logs stargate`

**일반적인 원인:**

- 비밀번호 설정이 잘못됨
- 비밀번호 알고리즘 불일치 (예: bcrypt가 설정되어 있지만 평문 비밀번호가 사용됨)
- Cookie 도메인 설정이 잘못됨

#### 3. 크로스 도메인 세션이 작동하지 않음

**문제:** 서브도메인 간 세션을 공유할 수 없음

**문제 해결 단계:**

1. `COOKIE_DOMAIN` 설정 확인
2. Cookie 도메인 형식이 올바른지 확인 (`.example.com`)
3. 브라우저의 Cookie 설정 확인

**해결 방법:**

```bash
# COOKIE_DOMAIN이 설정되어 있는지 확인
COOKIE_DOMAIN=.example.com
```

#### 4. Traefik 통합 문제

**문제:** Traefik이 인증 요청을 올바르게 전달할 수 없음

**문제 해결 단계:**

1. Traefik 레이블 설정 확인
2. 네트워크 설정이 올바른지 확인
3. Forward Auth 미들웨어의 주소 확인

**해결 방법:**

```yaml
# 미들웨어의 주소가 올바른지 확인
- "traefik.http.middlewares.stargate.forwardauth.address=http://stargate:8080/_auth"
- "traefik.http.middlewares.stargate.forwardauth.authResponseHeaders=X-Forwarded-User,X-Auth-User,X-Auth-Email,X-Auth-Name,X-Auth-Scopes,X-Auth-Role,X-Auth-AMR"
```

### 디버깅 팁

#### 1. 디버그 모드 활성화

```bash
DEBUG=true
```

#### 2. 네트워크 연결 확인

```bash
# 컨테이너 내에서 테스트
docker exec stargate wget -q -O - http://127.0.0.1:8080/healthz
```

#### 3. Traefik 로그 표시

```bash
docker logs traefik
```

#### 4. API 엔드포인트 테스트

```bash
# 헬스 체크 테스트
curl http://auth.example.com/healthz

# 인증 테스트 (헤더 사용)
# 서버 전제 조건: PASSWORD_HEADER_AUTH_ENABLED=true
curl -H "Stargate-Password: yourpassword" http://auth.example.com/_auth

# 인증 테스트 (Cookie 사용)
curl -H "Cookie: stargate_session_id=<session_id>" http://auth.example.com/_auth
```

### 도움 받기

문제가 발생한 경우:

1. 로그 표시: `docker logs stargate`
2. 설정 확인: 모든 환경 변수가 올바른지 확인
3. 문서 참조: [API 문서](API.md), [설정 참조](CONFIG.md)
4. Issue 제출: 프로젝트 저장소에 문제 보고서 제출

## 업데이트 가이드

### 업데이트 절차

1. **롤백용과 v1용 환경 파일을 별도로 생성**: 기존 `stargate.env`는 변경하지 않습니다. 다음 명령은 기존 대상 파일을 덮어쓰지 않으며 폐기된 Warden OTP 설정을 새 v1 파일에서만 제거합니다.

```bash
set -eu
old_env=./stargate.env
rollback_env=./stargate-v0.12.0.env
v1_env=./stargate-v1.env

test -f "$old_env"
test ! -e "$rollback_env"
test ! -e "$v1_env"
chmod 600 "$old_env"
umask 077
(set -C; cat "$old_env" > "$rollback_env")
(set -C; awk '!/^[[:space:]]*WARDEN_OTP_(ENABLED|SECRET_KEY)[[:space:]]*(=|$)/' "$old_env" > "$v1_env")
```

Herald 기반 TOTP에서는 Stargate가 Warden을 통해 인증된 사용자를 확인해야 하므로 `WARDEN_ENABLED=true`와 `WARDEN_URL`도 설정하십시오.

v1은 `WARDEN_OTP_ENABLED` 또는 `WARDEN_OTP_SECRET_KEY`가 존재하기만 해도 빈 값이나 `false` 여부와 관계없이 시작을 거부합니다. 다시 추가하지 마십시오. TOTP가 필요하면 `stargate-v1.env`에 `HERALD_ENABLED=true`, `HERALD_TOTP_ENABLED=true`, `HERALD_URL`과 지원되는 Herald Service Auth 방식 하나를 설정합니다. Herald의 TOTP Proxy와 herald-totp의 독립적인 `HERALD_TOTP_ENCRYPTION_KEY`도 설정하고 이전 Warden Secret을 자동 재사용하지 마십시오. [설정](CONFIG.md)을 참조하십시오.

2. **이전 컨테이너와 원래 환경을 롤백용으로 보존:**

```bash
docker stop stargate
docker rename stargate stargate-previous
```

3. **새 이미지 다운로드:**

```bash
docker pull ghcr.io/soulteary/stargate:v1.0.0
```

4. **새 컨테이너 시작:**

```bash
docker run -d \
  --name stargate \
  --env-file ./stargate-v1.env \
  -p 8080:8080 \
  --restart unless-stopped \
  ghcr.io/soulteary/stargate:v1.0.0
```

5. **서비스 확인:**

```bash
curl --fail http://127.0.0.1:8080/healthz
curl --fail http://127.0.0.1:8080/readyz
```

### 롤백

업데이트 후 문제가 발생한 경우:

```bash
# 새 컨테이너를 제거하고 변경되지 않은 이전 컨테이너 복원.
# stargate-v0.12.0.env를 롤백 설정 기록으로 보존.
docker rm -f stargate
docker rename stargate-previous stargate
docker start stargate
```
