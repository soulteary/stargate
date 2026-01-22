# Stargate - Forward Auth Service

[![License](https://img.shields.io/badge/license-Apache%202.0-blue.svg)](LICENSE)
[![Go Version](https://img.shields.io/badge/go-1.25+-blue.svg)](https://golang.org)
[![codecov](https://codecov.io/gh/soulteary/stargate/branch/main/graph/badge.svg)](https://codecov.io/gh/soulteary/stargate)
[![Go Report Card](https://goreportcard.com/badge/github.com/soulteary/stargate)](https://goreportcard.com/report/github.com/soulteary/stargate)

> **🚀 보안 마이크로서비스로의 게이트웨이**

![Stargate](.github/assets/banner.jpg)

Stargate는 전체 인프라의 **단일 인증 지점**이 되도록 설계된 프로덕션 준비가 된 경량 Forward Auth 서비스입니다. Go로 구축되고 성능에 최적화된 Stargate는 Traefik 및 기타 리버스 프록시와 원활하게 통합되어 백엔드 서비스를 보호합니다—**애플리케이션에 인증 코드를 한 줄도 작성할 필요가 없습니다**。

## 🌐 다국어 문서

- [English](README.md) | [中文](README.zhCN.md) | [Français](README.frFR.md) | [Italiano](README.itIT.md) | [日本語](README.jaJP.md) | [Deutsch](README.deDE.md) | [한국어](README.koKR.md)

![미리보기](.github/assets/preview.png)

### 🎯 Stargate를 선택하는 이유?

모든 서비스에서 인증 로직을 구현하는 데 지치셨나요? Stargate는 엣지에서 인증을 중앙화하여 이 문제를 해결하며, 다음을 가능하게 합니다:

- ✅ **단일 인증 레이어로 여러 서비스 보호**
- ✅ **애플리케이션에서 인증 로직을 제거하여 코드 복잡성 감소**
- ✅ **Docker와 간단한 구성으로 몇 분 안에 배포**
- ✅ **최소한의 리소스 사용으로 쉽게 확장**
- ✅ **여러 암호화 알고리즘과 안전한 세션 관리로 보안 유지**

### 💼 사용 사례

Stargate는 다음에 완벽합니다:

- **마이크로서비스 아키텍처**: 애플리케이션 코드를 수정하지 않고 여러 백엔드 서비스 보호
- **다중 도메인 애플리케이션**: 다양한 도메인 및 하위 도메인 간에 인증 세션 공유
- **내부 도구 및 대시보드**: 내부 서비스 및 관리 패널에 빠르게 인증 추가
- **API 게이트웨이 통합**: Traefik, Nginx 또는 기타 리버스 프록시와 통합 인증 레이어로 사용
- **개발 및 테스트**: 개발 환경을 위한 간단한 비밀번호 기반 인증
- **엔터프라이즈 인증**: Warden(사용자 화이트리스트) 및 Herald(OTP/인증 코드)와의 통합을 통한 프로덕션급 인증

## ✨ 기능

### 🔐 엔터프라이즈급 보안

- **여러 비밀번호 암호화 알고리즘**: plaintext(테스트), bcrypt, MD5, SHA512 등에서 선택
- **안전한 세션 관리**: 사용자 정의 가능한 도메인 및 만료 시간을 가진 Cookie 기반 세션
- **유연한 인증**: 비밀번호 기반 및 세션 기반 인증 모두 지원
- **OTP/인증 코드 지원**: Herald 서비스와의 통합으로 SMS/Email 인증 코드 제공
- **사용자 화이트리스트 관리**: Warden 서비스와의 통합으로 사용자 액세스 제어 제공

### 🌐 고급 기능

- **크로스 도메인 세션 공유**: 다양한 도메인/하위 도메인 간에 원활하게 인증 세션 공유
- **다국어 지원**: 영어 및 중국어 인터페이스 내장, 더 많은 언어로 쉽게 확장 가능
- **사용자 정의 가능한 UI**: 사용자 정의 제목 및 바닥글 텍스트로 로그인 페이지 브랜딩

### 🚀 성능 및 신뢰성

- **경량 및 빠름**: Go 및 Fiber 프레임워크로 구축되어 뛰어난 성능 제공
- **최소한의 리소스 사용**: 메모리 사용량이 적어 컨테이너화된 환경에 완벽
- **프로덕션 준비**: 신뢰성을 위해 설계된 실전에서 테스트된 아키텍처

### 📦 개발자 경험

- **Docker 우선**: 즉시 사용 가능한 완전한 Docker 이미지 및 docker-compose 구성
- **Traefik 네이티브**: 제로 구성 Traefik Forward Auth 미들웨어 통합
- **간단한 구성**: 환경 변수 기반 구성, 복잡한 파일 불필요

## 📋 목차

- [빠른 시작](#-빠른-시작)
- [문서](#-문서)
- [기본 구성](#-기본-구성)
- [선택적 서비스 통합](#-선택적-서비스-통합)
- [프로덕션 체크리스트](#-프로덕션-체크리스트)
- [라이선스](#-라이선스)

## 🚀 빠른 시작

**2분 이내**에 Stargate를 실행하세요!

### Docker Compose 사용(권장)

**1단계:** 저장소 복제
```bash
git clone <repository-url>
cd stargate
```

**2단계:** 인증 구성(`docker-compose.yml` 편집)

**옵션 A: 비밀번호 인증(간단)**
```yaml
services:
  stargate:
    environment:
      - AUTH_HOST=auth.example.com
      - PASSWORDS=plaintext:yourpassword1|yourpassword2
```

**옵션 B: Warden + Herald OTP 인증(프로덕션)**
```yaml
services:
  stargate:
    environment:
      - AUTH_HOST=auth.example.com
      - WARDEN_ENABLED=true
      - WARDEN_URL=http://warden:8080
      - WARDEN_API_KEY=your-warden-api-key
      - HERALD_ENABLED=true
      - HERALD_URL=http://herald:8080
      - HERALD_HMAC_SECRET=your-herald-hmac-secret
```

**3단계:** 서비스 시작
```bash
docker-compose up -d
```

**완료!** 인증 서비스가 이제 실행 중입니다. 🎉

### 로컬 개발

로컬 개발의 경우 Go 1.25+가 설치되어 있는지 확인한 다음:

```bash
chmod +x start-local.sh
./start-local.sh
```

로그인 페이지에 액세스: `http://localhost:8080/_login?callback=localhost`

## 📚 문서

Stargate를 최대한 활용할 수 있도록 포괄적인 문서가 제공됩니다:

### 핵심 문서

- 📐 **[아키텍처 문서](docs/koKR/ARCHITECTURE.md)** - 기술 아키텍처 및 설계 결정에 대한 심층 분석
- 🔌 **[API 문서](docs/koKR/API.md)** - 예제가 포함된 완전한 API 엔드포인트 참조
- ⚙️ **[구성 참조](docs/koKR/CONFIG.md)** - 자세한 구성 옵션 및 모범 사례
- 🚀 **[배포 가이드](docs/koKR/DEPLOYMENT.md)** - 프로덕션 배포 전략 및 권장 사항

### 빠른 참조

- **API 엔드포인트**: `GET /_auth`(인증 확인), `GET /_login`(로그인 페이지), `POST /_login`(로그인), `GET /_logout`(로그아웃), `GET /_session_exchange`(크로스 도메인), `GET /health`(상태 확인)
- **배포**: 빠른 시작에는 Docker Compose를 권장합니다. 프로덕션 배포는 [DEPLOYMENT.md](docs/koKR/DEPLOYMENT.md)를 참조하세요.
- **개발**: 개발 관련 문서는 [ARCHITECTURE.md](docs/koKR/ARCHITECTURE.md)를 참조하세요

## ⚙️ 기본 구성

Stargate는 환경 변수를 사용하여 구성을 수행합니다. 다음은 가장 일반적인 설정입니다:

### 필수 구성

- **`AUTH_HOST`**: 인증 서비스의 호스트명(예: `auth.example.com`)
- **`PASSWORDS`**: 비밀번호 구성, 형식: `algorithm:password1|password2|password3`

### 일반적인 구성 예

```bash
# 간단한 비밀번호 인증
AUTH_HOST=auth.example.com
PASSWORDS=plaintext:test123|admin456

# BCrypt 해시 사용
PASSWORDS=bcrypt:$2a$10$N9qo8uLOickgx2ZMRZoMyeIjZAgcfl7p92ldGxad68LJZdL17lhWy

# 크로스 도메인 세션 공유
COOKIE_DOMAIN=.example.com

# 로그인 페이지 사용자 정의
LOGIN_PAGE_TITLE=내 인증 서비스
LANGUAGE=ko  # 또는 'en'
```

**지원되는 비밀번호 알고리즘**: `plaintext`(테스트 전용), `bcrypt`, `md5`, `sha512`

**전체 구성 참조는 [docs/koKR/CONFIG.md](docs/koKR/CONFIG.md)를 참조하세요**

## 🔗 선택적 서비스 통합

Stargate는 완전히 독립적으로 사용할 수 있습니다. 또한 다음 서비스와 선택적으로 통합하여 기능을 확장할 수 있습니다:

### Warden 통합(사용자 화이트리스트)

사용자 화이트리스트 관리 및 사용자 정보를 제공합니다. 활성화되면 Stargate는 Warden에 쿼리하여 사용자가 허용 목록에 있는지 확인합니다.

```bash
WARDEN_ENABLED=true
WARDEN_URL=http://warden:8080
WARDEN_API_KEY=your-api-key
```

### Herald 통합(OTP/인증 코드)

OTP/인증 코드 서비스를 제공합니다. 활성화되면 Stargate는 Herald를 호출하여 인증 코드(SMS/Email)를 생성, 전송 및 확인합니다.

```bash
HERALD_ENABLED=true
HERALD_URL=http://herald:8080
HERALD_HMAC_SECRET=your-hmac-secret  # 프로덕션
# 또는
HERALD_API_KEY=your-api-key  # 개발
```

**참고**: Warden 및 Herald 통합은 선택 사항입니다. Stargate는 비밀번호 인증으로 독립적으로 사용할 수 있으며, 이러한 통합 기능을 선택적으로 활성화할 수 있습니다.

**전체 통합 가이드는 [docs/koKR/ARCHITECTURE.md](docs/koKR/ARCHITECTURE.md)를 참조하세요**

## ⚠️ 프로덕션 체크리스트

프로덕션에 배포하기 전에:

- ✅ 강력한 비밀번호 알고리즘 사용(`bcrypt` 또는 `sha512`, `plaintext` 피하기)
- ✅ Traefik 또는 리버스 프록시를 통해 HTTPS 활성화
- ✅ 하위 도메인 간 적절한 세션 관리를 위해 `COOKIE_DOMAIN` 설정
- ✅ 고급 기능이 필요한 경우 OTP 인증을 위해 Warden + Herald를 선택적으로 통합
- ✅ Stargate ↔ Herald/Warden 통신에 HMAC 서명 또는 mTLS 사용
- ✅ 적절한 로깅 및 모니터링 설정
- ✅ 보안 패치를 위해 Stargate를 최신 버전으로 유지

## 🎯 설계 원칙

Stargate는 독립적으로 사용할 수 있도록 설계되었습니다:

- **독립 사용**: Stargate는 비밀번호 인증 모드를 사용하여 독립적으로 실행할 수 있으며 외부 종속성이 필요하지 않습니다
- **선택적 통합**: Warden(사용자 화이트리스트) 및 Herald(OTP/인증 코드)와 선택적으로 통합할 수 있습니다
- **고성능**: forwardAuth 메인 경로는 세션만 확인하여 빠른 응답을 보장합니다
- **유연성**: 여러 인증 모드를 지원하며 필요에 따라 선택할 수 있습니다

## 📝 라이선스

이 프로젝트는 Apache License 2.0에 따라 라이선스됩니다. 자세한 내용은 [LICENSE](LICENSE) 파일을 참조하세요.

## 🤝 기여

기여를 환영합니다! 다음을 포함합니다:
- 🐛 버그 보고
- 💡 기능 제안
- 📝 문서 개선
- 🔧 코드 기여

Issue를 열거나 Pull Request를 제출해 주세요.
