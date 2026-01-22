# 보안 문서

> 🌐 **Language / 语言**: [English](../enUS/SECURITY.md) | [中文](../zhCN/SECURITY.md) | [Français](../frFR/SECURITY.md) | [Italiano](../itIT/SECURITY.md) | [日本語](../jaJP/SECURITY.md) | [Deutsch](../deDE/SECURITY.md) | [한국어](SECURITY.md)

이 문서는 Stargate의 보안 기능, 보안 구성 및 모범 사례를 설명합니다.

> ⚠️ **참고**: 이 문서는 현재 번역 중입니다. 전체 버전은 [영어 버전](../enUS/SECURITY.md)을 참조하세요.

## 구현된 보안 기능

1. **Forward Auth 보호**: 백엔드 서비스를 보호하기 위한 중앙 집중식 인증 레이어
2. **다중 비밀번호 알고리즘**: bcrypt, SHA512, MD5 및 평문(개발 전용) 지원
3. **안전한 세션 관리**: 도메인 및 만료 시간이 구성 가능한 Cookie 기반 세션
4. **서비스 통합 보안**: mTLS 또는 HMAC을 사용한 Warden 및 Herald 서비스와의 안전한 통신
5. **세션 공유 보안**: 안전한 크로스 도메인 세션 교환 메커니즘
6. **입력 검증**: 모든 입력 매개변수의 엄격한 검증
7. **오류 처리**: 프로덕션 모드에서 상세한 오류 정보 숨김
8. **보안 응답 헤더**: 보안 관련 HTTP 응답 헤더를 자동으로 추가
9. **HTTPS 강제**: 프로덕션 환경은 HTTPS를 사용해야 합니다
10. **OTP 통합**: OTP/검증 코드 인증을 위한 Herald와의 안전한 통합

자세한 내용은 [영어 버전](../enUS/SECURITY.md)을 참조하세요.

## 취약점 보고

보안 취약점을 발견한 경우 다음을 통해 보고하세요:

1. **GitHub Security Advisory** (권장)
   - 저장소의 [Security 탭](https://github.com/soulteary/stargate/security)으로 이동
   - "Report a vulnerability" 클릭
   - 보안 자문 양식 작성

2. **이메일** (GitHub Security Advisory를 사용할 수 없는 경우)
   - 프로젝트 유지 관리자에게 이메일 보내기
   - 취약점에 대한 자세한 설명 포함

**공개 GitHub Issues를 통해 보안 취약점을 보고하지 마세요.**
