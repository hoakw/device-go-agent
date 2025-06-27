# Device GO Agent

## 디바이스 장비를 모니터링하고, 애플리케이션을 관리할 수 있는 Go 기반 에이전트
- 타겟: OS가 설치되어 있고, Network가 연결되어 있는 장비
- OS: Window, Linux 계열
- Architecture: Arm, Amd 계열
- 리소스 정보
  - 용량: 78MB
  - CPU 사용량: 2% 이하(12코어 기준)
  - Memory 사용량: 50MB 이하

## 소스 구조
- 소스 코드는 6개로 구성되어 있습니다.
  1. bwc-cli: Device 
  2. bwc-management
  3. device-control
  4. device-init
  5. health-collector
  6. process-checker