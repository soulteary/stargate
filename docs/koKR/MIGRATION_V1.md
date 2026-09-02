# v0.12.0에서 v1.0.0으로 마이그레이션

Stargate v1.0.0은 안정적인 배포 및 HTTP 계약을 정의합니다. 프로덕션 트래픽을 전환하기 전에 스테이징에서 검증하십시오.

## 업그레이드 전

1. 현재 이미지, 환경 변수, 프록시 경로, 콜백 호스트와 Probe를 기록합니다.
2. 롤백을 위해 `v0.12.0` 이미지와 이전 컨테이너를 보존합니다.
3. `/_logout`, `/totp/enroll`, `/_session_exchange` 호출자를 확인합니다.
4. 여러 프로세스가 트래픽을 처리하거나 Rollout 중 이전 프로세스와 새 프로세스가 겹치거나 Session과 소비된 Ticket 상태가 재시작 후에도 유지되어야 하면 Redis를 준비합니다. 하나의 실행 중인 프로세스는 교차 도메인 Callback이 있어도 인메모리 스토리지를 사용할 수 있지만 모든 상태는 프로세스 로컬이며 재시작 시 사라집니다.

## 롤백 설정 보존 및 v1 환경 생성

이전 환경 파일로 v1을 시작하지 마십시오. `stargate.env`를 변경하지 않고 v0.12.0 컨테이너도 보존합니다. 다음 명령은 보호된 롤백 사본과 별도의 v1 파일을 생성하고 기존 출력을 덮어쓰지 않으며 폐기 설정을 v1 파일에서만 제거합니다.

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

v1은 `WARDEN_OTP_ENABLED` 또는 `WARDEN_OTP_SECRET_KEY`가 존재하면 빈 값이나 `false`여도 거부합니다. `stargate-v1.env`에 추가하지 마십시오. TOTP에는 Stargate의 `HERALD_ENABLED=true`, `HERALD_TOTP_ENABLED=true`, `HERALD_URL`과 지원되는 Herald Service Auth를 설정합니다. Herald가 TOTP를 herald-totp로 전달하도록 하고 herald-totp에는 별도의 `HERALD_TOTP_ENCRYPTION_KEY`를 설정하며 이전 Warden Secret을 자동 재사용하지 마십시오. [설정](CONFIG.md)을 참조하십시오.

`--env-file ./stargate-v1.env`로 v1을 검증하고 배포합니다. 롤백 기간이 끝날 때까지 `stargate-v0.12.0.env`, 변경하지 않은 원본 파일 및 이전 컨테이너를 보존하십시오.

## 필수 변경

| 계약 | v0.12.0 | v1.0.0 |
| --- | --- | --- |
| 컨테이너 포트 | `80` | `8080` |
| 런타임 사용자 | root | UID/GID `10001` |
| Liveness | `/health` | `/healthz` |
| Readiness | `/health` | `/readyz` |

Traefik ForwardAuth 대상을 `http://stargate:8080/_auth`로 변경합니다. 여러 복제본에서는 `SESSION_STORAGE_ENABLED=true`를 사용하고 Redis와 Ticket Secret을 공유합니다.

## 보안 및 HTTP 계약

- 교차 도메인 교환은 원본 Session ID `?id=` 대신 `?ticket=`을 사용합니다. `CALLBACK_ALLOWED_HOSTS`와 32자 이상의 `SESSION_EXCHANGE_SECRET`을 설정합니다.
- Forwarded Header는 `TRUSTED_PROXIES`에 등록된 원본에서만 허용됩니다.
- `Stargate-Password`는 기본적으로 비활성화됩니다. 신뢰하는 Proxy에서만 `PASSWORD_HEADER_AUTH_ENABLED=true`를 사용합니다.
- Identity Header에는 Warden, `HEADER_AUTH_ENABLED=true`, 강력한 공유 Secret이 필요합니다.
- `GET /_logout`는 `POST /_logout`로 변경됩니다.
- `GET /totp/enroll`는 확인 페이지만 표시하고 `POST /totp/enroll`가 등록을 시작합니다.
- `GET /metrics`는 변경되지 않습니다.

## 롤아웃 및 검증

```bash
curl --fail http://127.0.0.1:8080/healthz
curl --fail http://127.0.0.1:8080/readyz
curl --fail http://127.0.0.1:8080/metrics
```

로그인, 로그아웃, TOTP, 허용·거부된 콜백과 만료·재사용 Ticket을 확인합니다. v0.12.0과 v1.0.0을 같은 Pool에 혼합하지 마십시오.

## 롤백

v0.12.0 이미지, 포트, Probe를 함께 복원합니다. 진행 중인 v1 Ticket을 무효화하고 다시 로그인하도록 합니다.
