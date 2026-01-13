# 배포 가이드

이 문서는 Stargate Forward Auth 서비스의 자세한 배포 가이드를 제공합니다.

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

## Docker 배포

### 이미지 빌드

#### 소스에서 빌드

```bash
cd codes
docker build -t stargate:latest .
```

#### 빌드 매개변수

- **베이스 이미지**: `golang:1.25-alpine` (빌드 단계)
- **실행 이미지**: `scratch` (최소 이미지)
- **작업 디렉토리**: `/app`
- **공개 포트**: `80`

### 컨테이너 실행

#### 기본 실행

```bash
docker run -d \
  --name stargate \
  -p 80:80 \
  -e AUTH_HOST=auth.example.com \
  -e PASSWORDS=plaintext:yourpassword \
  stargate:latest
```

#### 완전한 설정으로 실행

```bash
docker run -d \
  --name stargate \
  -p 80:80 \
  -e AUTH_HOST=auth.example.com \
  -e PASSWORDS=bcrypt:$2a$10$N9qo8uLOickgx2ZMRZoMyeIjZAgcfl7p92ldGxad68LJZdL17lhWy \
  -e DEBUG=false \
  -e LANGUAGE=ko \
  -e LOGIN_PAGE_TITLE=내 인증 서비스 \
  -e LOGIN_PAGE_FOOTER_TEXT=© 2024 내 회사 \
  -e COOKIE_DOMAIN=.example.com \
  --restart unless-stopped \
  stargate:latest
```

#### 매개변수 설명

- `-d`: 백그라운드에서 실행
- `--name stargate`: 컨테이너 이름
- `-p 80:80`: 포트 매핑 (호스트 포트:컨테이너 포트)
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

# 중지하고 삭제
docker rm -f stargate
```

## Docker Compose 배포

### 기본 설정

프로젝트는 `docker-compose.yml` 샘플 파일을 제공합니다:

```yaml
services:
  stargate:
    image: stargate
    environment:
      - AUTH_HOST=auth.test.localhost
      - PASSWORDS=plaintext:test1234|test1337
    networks:
      - traefik
    labels:
      - traefik.enable=true
      - traefik.docker.network=proxy
      - traefik.http.routers.auth.entrypoints=http
      - traefik.http.routers.auth.rule=Host(`auth.test.localhost`) || Path(`/_session_exchange`)
      - traefik.http.middlewares.stargate.forwardauth.address=http://stargate/_auth

  whoami:
    image: traefik/whoami
    networks:
      - traefik
    labels:
      - traefik.enable=true
      - traefik.docker.network=proxy
      - traefik.http.routers.whoami.entrypoints=http
      - traefik.http.routers.whoami.rule=Host(`whoami.test.localhost`)
      - "traefik.http.routers.whoami.middlewares=stargate"

networks:
  traefik:
    external: true
```

### 서비스 시작

```bash
cd codes
docker-compose up -d
```

### 서비스 중지

```bash
docker-compose down
```

### 로그 표시

```bash
# 모든 서비스의 로그 표시
docker-compose logs -f

# 특정 서비스의 로그 표시
docker-compose logs -f stargate
```

### 커스텀 설정

`docker-compose.yml`을 편집하고 환경 변수를 변경합니다:

```yaml
services:
  stargate:
    image: stargate
    environment:
      - AUTH_HOST=auth.example.com
      - PASSWORDS=bcrypt:$2a$10$...
      - DEBUG=false
      - LANGUAGE=ko
      - COOKIE_DOMAIN=.example.com
```

## Traefik 통합

### 기본 설정

Stargate는 Traefik과 통합하도록 설계되어 있으며, Forward Auth 미들웨어를 통해 인증을 제공합니다.

#### 1. Stargate 서비스 설정

`docker-compose.yml`에서 Stargate를 설정합니다:

```yaml
services:
  stargate:
    image: stargate:latest
    environment:
      - AUTH_HOST=auth.example.com
      - PASSWORDS=bcrypt:$2a$10$...
    networks:
      - traefik
    labels:
      - "traefik.enable=true"
      - "traefik.docker.network=traefik"
      - "traefik.http.routers.auth.entrypoints=http,https"
      - "traefik.http.routers.auth.rule=Host(`auth.example.com`) || Path(`/_session_exchange`)"
      - "traefik.http.middlewares.stargate.forwardauth.address=http://stargate/_auth"
      - "traefik.http.middlewares.stargate.forwardauth.authResponseHeaders=X-Forwarded-User"
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
      - "traefik.docker.network=traefik"
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
```

2. 관련된 모든 도메인이 Traefik을 통해 Stargate로 라우팅되는지 확인

3. 로그인 흐름:
   - 사용자가 `auth.example.com`에 로그인
   - `app.example.com/_session_exchange?id=<session_id>`로 리디렉션
   - 세션 Cookie가 `.example.com` 도메인에 설정됨
   - 모든 서브도메인 `*.example.com`에서 이 세션을 사용할 수 있습니다

## 프로덕션 환경 배포

### 보안 권장 사항

#### 1. 강력한 비밀번호 알고리즘 사용

**권장하지 않음:**

```bash
PASSWORDS=plaintext:yourpassword
```

**권장:**

```bash
PASSWORDS=bcrypt:$2a$10$N9qo8uLOickgx2ZMRZoMyeIjZAgcfl7p92ldGxad68LJZdL17lhWy
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
      test: ["CMD", "wget", "--quiet", "--tries=1", "--spider", "http://localhost/health"]
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

**주의:** Stargate는 메모리 내 세션 저장소를 사용하므로, 인스턴스 간 세션이 공유되지 않습니다. 다중 인스턴스 배포가 필요한 경우, 다음을 권장합니다:

- 로드 밸런서의 세션 영속성 (Sticky Session) 사용
- 또는 외부 세션 저장소 (Redis) 지원을 기다림

#### 2. 로드 밸런싱

Traefik 앞에 로드 밸런서를 추가:

```yaml
services:
  traefik:
    labels:
      - "traefik.http.services.stargate.loadbalancer.server.port=80"
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

모니터링에 `/health` 엔드포인트 사용:

```bash
# 헬스 체크 스크립트
#!/bin/bash
if curl -f http://localhost/health > /dev/null 2>&1; then
  exit 0
else
  exit 1
fi
```

#### 3. Prometheus 통합

(구현 예정) 향후 버전에서는 Prometheus 메트릭 내보내기를 지원합니다.

## 모니터링 및 유지보수

### 로그 관리

#### 로그 표시

```bash
# Docker
docker logs -f stargate

# Docker Compose
docker-compose logs -f stargate
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
time curl http://auth.example.com/health
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
- "traefik.http.middlewares.stargate.forwardauth.address=http://stargate/_auth"
```

### 디버깅 팁

#### 1. 디버그 모드 활성화

```bash
DEBUG=true
```

#### 2. 네트워크 연결 확인

```bash
# 컨테이너 내에서 테스트
docker exec stargate wget -O- http://localhost/health
```

#### 3. Traefik 로그 표시

```bash
docker logs traefik
```

#### 4. API 엔드포인트 테스트

```bash
# 헬스 체크 테스트
curl http://auth.example.com/health

# 인증 테스트 (헤더 사용)
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

1. **설정 백업**: 현재 환경 변수 설정 백업

2. **서비스 중지:**

```bash
docker stop stargate
```

3. **새 이미지 다운로드:**

```bash
docker pull stargate:latest
```

4. **새 컨테이너 시작:**

```bash
docker run -d \
  --name stargate \
  ...(저장된 설정 사용)
  stargate:latest
```

5. **서비스 확인:**

```bash
curl http://auth.example.com/health
```

### 롤백

업데이트 후 문제가 발생한 경우:

```bash
# 새 컨테이너 중지
docker stop stargate

# 이전 이미지로 시작
docker run -d \
  --name stargate \
  ...(저장된 설정 사용)
  stargate:<old-version>
```
