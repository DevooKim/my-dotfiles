# Dotfiles

macOS 설정을 애플리케이션별로 관리하는 dotfiles 저장소입니다. 루트의 각
디렉터리는 독립적인 설정 패키지이며, `bin/setup` TUI에서 필요한 항목만
선택해 설치하거나 제거할 수 있습니다.

## 설치 및 실행

간편 설치 스크립트가 추가되면 다음 명령으로 저장소를 설치하고 TUI를
실행할 수 있습니다.

```bash
curl -fsSL \
  https://raw.githubusercontent.com/DevooKim/my-dotfiles/main/install.sh \
  | sh
```

> 현재 저장소에는 `install.sh`가 아직 없습니다. 위 명령은 설치 스크립트가
> 추가된 이후 사용할 수 있습니다.

현재는 저장소를 직접 복제한 뒤 TUI를 실행합니다.

```bash
git clone https://github.com/DevooKim/my-dotfiles.git ~/.dotfiles
~/.dotfiles/bin/setup
```

`bin/setup`은 Apple Silicon과 Intel Mac을 모두 지원하는 universal macOS
실행 파일입니다. 실행할 때 Homebrew, Go, GNU Stow는 필요하지 않습니다.

## TUI 기능

- **Install**: 설치되지 않은 설정을 기본 선택해 링크합니다.
- **Reapply**: 선택한 설정의 링크를 다시 적용합니다. 기본 선택은 없습니다.
- **Update**: `git pull --ff-only`로 저장소를 갱신한 뒤 Reapply를 엽니다.
- **Remove**: 선택한 설정에서 관리하는 링크만 제거합니다. 기본 선택은 없습니다.
- **Doctor**: 패키지 소스, 링크 상태, 실행 명령, Git 사용 여부를 검사합니다.

방향키 또는 `j`/`k`로 이동하고, `Space`로 항목을 선택합니다. `a`는 전체
선택, `n`은 전체 해제, `Enter`는 다음 단계, `q` 또는 `Esc`는 취소입니다.

## 관리하는 설정

| 앱 | 용도 | 주요 설정 경로 |
|---|---|---|
| AeroSpace | 타일링 윈도우 관리자 | `~/.config/aerospace/` |
| Ghostty | 터미널 | `~/.config/ghostty/` |
| Hammerspoon | macOS 자동화 및 단축키 | `~/.hammerspoon/` |
| Herdr | 에이전트 작업공간 | `~/.config/herdr/` |
| Rift | 윈도우 관리자 | `~/.config/rift/` |
| Starship | 셸 프롬프트 | `~/.config/starship.toml` |
| WezTerm | 터미널 | `~/.config/wezterm/` |
| Zed | 코드 에디터 | `~/.config/zed/` |
| Zsh | 셸 설정 | `~/.zshrc`, `~/.config/zsh/` |

`homebrew/Brewfile`은 개발 환경 기록용이며 TUI의 설치·업데이트 대상에서
제외됩니다.

## 간편 설치 스크립트 배포 방식

간편 설치 스크립트를 추가할 경우 저장소의 `install.sh`를
`raw.githubusercontent.com`에서 직접 실행하는 방식을 사용합니다.

설치 스크립트는 `~/.dotfiles`에 저장소를 복제하거나 fast-forward 방식으로
갱신한 뒤 `~/.dotfiles/bin/setup`을 실행해야 합니다. 설정 파일과 Git 이력이
필요하므로 `bin/setup` 바이너리만 단독으로 내려받는 방식은 사용하지
않습니다.

## 개발 및 빌드

TUI 소스는 `tools/dotfiles-tui`에 있습니다. Go와 Apple의 `lipo`를 사용할 수
있는 macOS에서 다음 명령으로 테스트하고 `bin/setup`을 다시 빌드합니다.

```bash
./tools/dotfiles-tui/build
```

빌드 과정은 Go 테스트와 정적 검사를 실행하고 arm64 및 x86_64 실행 파일을
하나의 universal 바이너리로 결합합니다.
