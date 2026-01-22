# 문서 인덱스

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
2. [빠른 시작](../../README.koKR.md#빠른-시작) 섹션을 확인하세요
3. [구성](../../README.koKR.md#구성)을 참조하여 서비스를 구성하세요

### 개발자

1. [ARCHITECTURE.md](ARCHITECTURE.md)를 읽어 아키텍처를 이해하세요
2. [API.md](API.md)를 확인하여 API 인터페이스를 이해하세요
3. [개발 가이드](../../README.koKR.md#개발-가이드)를 참조하여 개발하세요

### 운영

1. [DEPLOYMENT.md](DEPLOYMENT.md)를 읽어 배포 방법을 이해하세요
2. [CONFIG.md](CONFIG.md)를 확인하여 구성 옵션을 이해하세요
3. [문제 해결](DEPLOYMENT.md#문제-해결)을 참조하여 문제를 해결하세요

## 📖 문서 구조

```
stargate/
├── README.md              # 프로젝트 메인 문서 (영어)
├── README.zhCN.md         # 프로젝트 메인 문서 (중국어)
├── README.frFR.md         # 프로젝트 메인 문서 (프랑스어)
├── README.itIT.md         # 프로젝트 메인 문서 (이탈리아어)
├── README.jaJP.md         # 프로젝트 메인 문서 (일본어)
├── README.deDE.md         # 프로젝트 메인 문서 (독일어)
├── README.koKR.md         # 프로젝트 메인 문서 (한국어)
├── docs/
│   ├── enUS/
│   │   ├── README.md       # 문서 인덱스 (영어)
│   │   ├── ARCHITECTURE.md # 아키텍처 문서 (영어)
│   │   ├── API.md          # API 문서 (영어)
│   │   ├── CONFIG.md       # 구성 참조 (영어)
│   │   └── DEPLOYMENT.md   # 배포 가이드 (영어)
│   ├── zhCN/
│   │   ├── README.md       # 문서 인덱스 (중국어)
│   │   ├── ARCHITECTURE.md # 아키텍처 문서 (중국어)
│   │   ├── API.md          # API 문서 (중국어)
│   │   ├── CONFIG.md       # 구성 참조 (중국어)
│   │   └── DEPLOYMENT.md   # 배포 가이드 (중국어)
│   └── koKR/
│       ├── README.md       # 문서 인덱스 (한국어, 이 파일)
│       ├── ARCHITECTURE.md # 아키텍처 문서 (한국어)
│       ├── API.md          # API 문서 (한국어)
│       ├── CONFIG.md       # 구성 참조 (한국어)
│       └── DEPLOYMENT.md   # 배포 가이드 (한국어)
└── ...
```

## 🔍 주제별 검색

### 구성 관련

- 환경 변수 구성: [CONFIG.md](CONFIG.md)
- 비밀번호 구성: [CONFIG.md#비밀번호-구성](CONFIG.md#비밀번호-구성)
- 구성 예제: [CONFIG.md#구성-예제](CONFIG.md#구성-예제)

### API 관련

- API 엔드포인트 목록: [API.md](API.md)
- 인증 흐름: [API.md#인증-흐름-예제](API.md#인증-흐름-예제)
- 오류 처리: [API.md#오류-응답-형식](API.md#오류-응답-형식)

### 배포 관련

- Docker 배포: [DEPLOYMENT.md#docker-배포](DEPLOYMENT.md#docker-배포)
- Traefik 통합: [DEPLOYMENT.md#traefik-통합](DEPLOYMENT.md#traefik-통합)
- 프로덕션 환경: [DEPLOYMENT.md#프로덕션-배포](DEPLOYMENT.md#프로덕션-배포)

### 아키텍처 관련

- 기술 스택: [ARCHITECTURE.md#기술-스택](ARCHITECTURE.md#기술-스택)
- 프로젝트 구조: [ARCHITECTURE.md#프로젝트-구조](ARCHITECTURE.md#프로젝트-구조)
- 핵심 구성 요소: [ARCHITECTURE.md#핵심-구성-요소](ARCHITECTURE.md#핵심-구성-요소)

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
