# Starship 사용자 및 호스트 표시 설계

## 목표

셸 프롬프트에서 현재 디렉토리 앞에 `사용자@호스트명`을 항상 표시한다.

## 변경 범위

- `starship/.config/starship.toml`의 프롬프트 형식 앞부분에 Starship 내장 `username`, `hostname` 모듈을 배치한다.
- 두 모듈은 로컬 셸과 SSH 셸 모두에서 표시한다.
- 출력 형식은 `사용자@호스트명 ~/directory`로 한다.
- 기존 디렉토리, Git 상태, 실행 시간, Node.js 버전, 시각 및 프롬프트 문자 설정은 유지한다.

## 검증

Starship이 변경된 설정을 오류 없이 읽고 프롬프트를 렌더링하는지 확인한다.
