# 문서 인덱스

> **부분 번역**: 한국어 문서는 일부만 번역한 것으로, 현재 변경 사항이 모두 반영되어 있지 않습니다. [영어 문서](../enUS/README.md)가 완전한 기준 문서입니다.

Stargate Forward Auth Service 문서에 오신 것을 환영합니다.

## 🌐 다국어 문서

- [English](../enUS/README.md) | [中文](../zhCN/README.md) | [Français](../frFR/README.md) | [Italiano](../itIT/README.md) | [日本語](../jaJP/README.md) | [Deutsch](../deDE/README.md) | [한국어](README.md)

## 📚 문서 목록

### 핵심 문서

- **[README.md](../../README.koKR.md)** - 프로젝트 개요 및 빠른 시작 가이드
- **[ARCHITECTURE.md](ARCHITECTURE.md)** - 기술 아키텍처 및 설계 결정

### 상세 문서

- **[API.md](API.md)** - 완전한 API 엔드포인트 문서
  - 인증 확인 엔드포인트
  - 로그인 및 로그아웃 엔드포인트
  - 세션 교환 엔드포인트
  - 상태 확인 엔드포인트
  - 오류 응답 형식
  - 인증 흐름 예제

- **[CONFIG.md](CONFIG.md)** - 구성 참조
  - 구성 방법
  - 필수 구성 항목
  - 선택적 구성 항목
  - 비밀번호 구성 세부 정보
  - 구성 예제
  - 구성 모범 사례

- **[DEPLOYMENT.md](DEPLOYMENT.md)** - 배포 가이드
  - Docker 배포
  - Docker Compose 배포
  - Traefik 통합
  - 프로덕션 배포
  - 모니터링 및 유지보수
  - 문제 해결

## 🚀 빠른 탐색

### 시작하기

1. [README.koKR.md](../../README.koKR.md)를 읽어 프로젝트를 이해하세요
2. [빠른 시작](../../README.koKR.md#-빠른-시작) 섹션을 확인하세요
3. [구성](../../README.koKR.md#%EF%B8%8F-기본-구성)을 참조하여 서비스를 구성하세요

### 개발자

1. [ARCHITECTURE.md](ARCHITECTURE.md)를 읽어 아키텍처를 이해하세요
2. [API.md](API.md)를 확인하여 API 인터페이스를 이해하세요
3. [개발 가이드](CONTRIBUTING.md)를 참조하여 개발하세요

### 운영

1. [DEPLOYMENT.md](DEPLOYMENT.md)를 읽어 배포 방법을 이해하세요
2. [CONFIG.md](CONFIG.md)를 확인하여 구성 옵션을 이해하세요
3. [문제 해결](DEPLOYMENT.md#문제-해결)을 참조하여 문제를 해결하세요

## 📖 문서 구조

이 디렉터리의 부분 번역:

- [API.md](API.md)
- [ARCHITECTURE.md](ARCHITECTURE.md)
- [CONFIG.md](CONFIG.md)
- [CONTRIBUTING.md](CONTRIBUTING.md)
- [DEPLOYMENT.md](DEPLOYMENT.md)
- [MIGRATION_V1.md](MIGRATION_V1.md)
- [SECURITY.md](SECURITY.md)

영어로만 제공:

- [DIAGNOSTICS.md](../enUS/DIAGNOSTICS.md) - Deployment diagnostics
- [AUTHENTICATION_UI.md](../enUS/AUTHENTICATION_UI.md) - Authentication pages
- [DINGTALK_USERID_ONLY.md](../enUS/DINGTALK_USERID_ONLY.md) - Opt-in userid-only work-IM login
- [PERFORMANCE.md](../enUS/PERFORMANCE.md) - Redis rate-limit measurements
- [VERIFIED_EMAIL.md](../enUS/VERIFIED_EMAIL.md) - Verified email assertions

## 🔍 주제별 검색

### 구성 관련

- 환경 변수 구성: [CONFIG.md](CONFIG.md)
- 비밀번호 구성: [CONFIG.md#비밀번호-설정](CONFIG.md#비밀번호-설정)
- 구성 예제: [CONFIG.md#설정-예제](CONFIG.md#설정-예제)

### API 관련

- API 엔드포인트 목록: [API.md](API.md)
- 인증 흐름: [API.md#인증-플로우-예제](API.md#인증-플로우-예제)
- 오류 처리: [API.md#오류-응답-형식](API.md#오류-응답-형식)

### 배포 관련

- Docker 배포: [DEPLOYMENT.md#docker-배포](DEPLOYMENT.md#docker-배포)
- Traefik 통합: [DEPLOYMENT.md#traefik-통합](DEPLOYMENT.md#traefik-통합)
- 프로덕션 환경: [DEPLOYMENT.md#프로덕션-환경-배포](DEPLOYMENT.md#프로덕션-환경-배포)

### 아키텍처 관련

- 기술 스택: [ARCHITECTURE.md#기술-스택](ARCHITECTURE.md#기술-스택)
- 프로젝트 구조: [ARCHITECTURE.md#프로젝트-구조](ARCHITECTURE.md#프로젝트-구조)
- 핵심 구성 요소: [ARCHITECTURE.md#주요-컴포넌트](ARCHITECTURE.md#주요-컴포넌트)

## 💡 사용 권장 사항

1. **처음 사용하는 사용자**: [README.koKR.md](../../README.koKR.md)로 시작하여 빠른 시작 가이드를 따르세요
2. **서비스 구성**: [CONFIG.md](CONFIG.md)를 참조하여 모든 구성 옵션을 이해하세요
3. **Traefik 통합**: [DEPLOYMENT.md](DEPLOYMENT.md)의 Traefik 통합 섹션을 확인하세요
4. **확장 기능 개발**: [ARCHITECTURE.md](ARCHITECTURE.md)를 읽어 아키텍처 설계를 이해하세요
5. **문제 해결**: [DEPLOYMENT.md#문제-해결](DEPLOYMENT.md#문제-해결)을 확인하세요

## 📝 문서 업데이트

문서는 프로젝트가 발전함에 따라 지속적으로 업데이트됩니다. 오류를 발견하거나 추가가 필요한 경우 Issue 또는 Pull Request를 제출해 주세요.

## 🤝 기여

문서 개선을 환영합니다:

1. 오류나 개선이 필요한 영역을 찾으세요
2. 문제를 설명하는 Issue를 제출하세요
3. 또는 직접 Pull Request를 제출하세요
