# 보안 문서

> 🌐 **Language / 语言**: [English](../enUS/SECURITY.md) | [中文](../zhCN/SECURITY.md) | [Français](../frFR/SECURITY.md) | [Italiano](../itIT/SECURITY.md) | [日本語](../jaJP/SECURITY.md) | [Deutsch](../deDE/SECURITY.md) | [한국어](SECURITY.md)

이 문서는 Stargate의 보안 기능, 보안 구성 및 모범 사례를 설명합니다.

> ⚠️ **참고**: 이 문서는 현재 번역 중입니다. 전체 버전은 [영어 버전](../enUS/SECURITY.md)을 참조하세요.

## 구현된 보안 기능

1. **Forward Auth 보호**: 백엔드 서비스를 보호하기 위한 중앙 집중식 인증 레이어
2. **다중 비밀번호 알고리즘**: bcrypt와 로컬 개발용 plaintext를 지원하며 MD5와 salt 없는 SHA-512는 거부
3. **안전한 세션 관리**: 도메인 및 만료 시간이 구성 가능한 Cookie 기반 세션
4. **서비스 통합 보안**: mTLS 또는 HMAC을 사용한 Warden 및 Herald 서비스와의 안전한 통신
5. **세션 공유 보안**: 안전한 크로스 도메인 세션 교환 메커니즘
6. **입력 검증**: 모든 입력 매개변수의 엄격한 검증
7. **오류 처리**: 오류 메시지와 기타 응답 데이터는 엔드포인트마다 다르며 `DEBUG`는 일반적인 비공개 보장이 아님
8. **보안 응답 헤더**: 보안 관련 HTTP 응답 헤더를 자동으로 추가
9. **HTTPS 강제**: 프로덕션 환경은 HTTPS를 사용해야 합니다
10. **OTP 통합**: OTP/검증 코드 인증을 위한 Herald와의 안전한 통합

## 오류 응답 및 디버그 검증 코드

Stargate에는 `MODE` 설정이 없습니다. 오류 응답은 엔드포인트별로 정의됩니다.
운영 세부 정보는 일반적으로 로그에 기록되지만, `DEBUG`를 응답 정제나 비공개에 대한
일반적인 보장으로 간주해서는 안 됩니다.

Stargate가 `DEBUG=true`로 실행되고 구성된 Herald 서비스가 테스트 모드
(`HERALD_TEST_MODE`)에서 비어 있지 않은 `debug_code`를 반환하면, Stargate는 성공한
`POST /_send_verify_code` 응답에 테스트 검증 코드를 `debug_code`로 포함합니다.
기본 로그인 페이지는 이 코드를 표시하고 검증 코드 입력란에 자동으로 채웁니다.

이 조합은 인증 자격 증명에 해당하는 검증 코드를 요청 클라이언트에 노출합니다.
로컬 개발 또는 테스트에서만 사용하고 프로덕션에서는 절대 사용하지 마세요. 프로덕션에서는
Stargate를 `DEBUG=false`로 설정하고 Herald의 `HERALD_TEST_MODE`를 비활성화하세요.

자세한 내용은 [영어 버전](../enUS/SECURITY.md)을 참조하세요.

## 보안 관련 구성

- HTTPS에서는 `COOKIE_SECURE=true`를 사용하고 `TRUSTED_PROXIES`를 알려진 프록시 주소로 제한합니다.
- `CALLBACK_ALLOWED_HOSTS`를 구성하고, 도메인 간 티켓에는 32자 이상의 `SESSION_EXCHANGE_SECRET`를 사용합니다.
- Warden과 Herald에서는 HMAC 키 ID와 비밀, mTLS 클라이언트 인증서와 키를 각각 함께 설정해야 합니다.
- 동기화된 전체 변수 목록은 [CONFIG.md](CONFIG.md)를 참조하세요.

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

## 배포 신뢰 경계

Forward Auth는 구성된 리버스 프록시를 실제로 통과하는 요청만 보호합니다. 백엔드 포트를 직접 공개하지 말고 프록시 네트워크에서만 접근하도록 제한하세요. 클라이언트가 제공한 ID 또는 자격 증명 헤더는 전달 전에 제거하세요.
