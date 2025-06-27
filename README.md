# Device GO Agent

## 디바이스 장비를 모니터링하고, 애플리케이션을 관리할 수 있는 Go 기반 에이전트
- 타겟: OS가 설치되어 있고, Network가 연결되어 있는 장비
- OS: Window, Linux 계열
- Architecture: Arm, Amd 계열
- 리소스 정보
  - 용량: 78MB
  - CPU 사용량: 2% 이하(12코어 기준)
  - Memory 사용량: 50MB 이하

## 동작
- 에이전트는 디바이스에 설치되어 동작합니다. 에이전트는 바이너리 파일로 빌드되어 설치되며 설치 시, 장비의 고유번호, 클라우드 서버위치 정보가 필요합니다. 
- 에이전트는와 MQTT 프로토콜으로 서버와 통신하며, Onprem 서버와 클라우드 서버에서 운영 가능합니다.
- 에이전트는 프로젝트(Workspace)라는 개념이 존재하며, 프로젝트에 소속해야 애플리케이션 배포 및 관리가 가능합니다.

## 소스 구조
- 소스 코드는 6개로 구성되어 있습니다.
  1. agent-cli: CLI 기능을 제공하는 소스코드입니다.  
  2. agent-control: 디바이스의 애플리케이션 관리를 위한 소스코드입니다.
  3. agent-init: 에이전트를 설치하기 위한 초기화 소스코드입니다.
  4. agent-management: 에이전트 기능을 관리하는 소스코드입니다.
  5. app-health-collector: 디바이스에 설치된 애플리케이션의 데이터 수집 소스코드입니다.
  6. health-collector: 디바이스의 데이터 수집 소스코드입니다.

## Agent-cli
- agent-cli는 디바이스 장비에서 CLI 기능을 제공합니다.
- 제공하는 기능은 다음과 같습니다.
  - 터미널 CLI 기능 (ifconfig, reboot 등등)
  - docker control
  - 애플리케이션 배포, 삭제, 관리
  - 애플리케이션 Config 수정(.json 파일)

## Agent-control
- agent-control은 에이전트의 연결 대상인 서버로부터 명령을 받아 처리합니다.
- 처리하는 기능은 다음과 같습니다.
  - 디바이스 소속 프로젝트 변경
  - 애플리케이션 배포, 삭제, 관리
  - 애플리케이션 Config 수정(.json 파일)
  - 디바이스 재부팅

## Agent-init
- agent-init은 디바이스에 에이전트를 설치 및 초기화합니다.
- 에이전트를 설치하기 위해서 장비의 고유번호와 타겟 서버 정보를 입력해야 합니다.

## Agent-management
- agent-management는 에이전트의 구성 서비스들을 관리합니다.
- 디바이스의 프로젝트가 변경될 떄, 에이전트를 재시작합니다.

## App-health-collector
- app-health-collector는 디바이스에 설치된 애플리케이션의 상태(status, cpu/mem resource 등)을 수집합니다.
- 수집한 데이터는 서버에 전달하며, 서버에서 애플리케이션을 모니터링할 수 있습니다.

## Health-collector
- health-collector는 디바이스의 상태(status, cpu/mem/network/disk/gpu 등)을 수집합니다.
- 수집한 데이터는 서버에 전달하며, 서버에서 애플리케이션을 모니터링할 수 있습니다.