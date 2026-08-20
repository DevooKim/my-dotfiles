# Dotfiles

macOS 설정을 애플리케이션별로 관리하는 dotfiles 저장소입니다. 루트의 각 디렉터리가 독립적인 설정 패키지이며 `bin/setup` TUI에서 필요한 것만 골라 설치하거나 제거합니다.

## 설치

```bash
curl -fsSL \
  https://raw.githubusercontent.com/DevooKim/my-dotfiles/main/scripts/install.sh \
  | sh
```

직접 복제해서 실행할 수도 있습니다.

```bash
git clone https://github.com/DevooKim/my-dotfiles.git ~/.dotfiles
~/.dotfiles/bin/setup
```

`bin/setup`은 Apple Silicon과 Intel을 모두 지원하는 universal 바이너리입니다. 실행할 때 Homebrew, Go, GNU Stow가 없어도 됩니다.

## TUI 기능

- **Install** — 아직 설치되지 않은 설정을 링크합니다.
- **Reapply** — 선택한 설정의 링크를 다시 적용합니다.
- **Update** — `git pull --ff-only`로 저장소를 갱신한 뒤 Reapply를 엽니다.
- **Remove** — 선택한 설정이 관리하는 링크만 제거합니다.
- **Doctor** — 패키지 소스, 링크 상태, 실행 명령, Git 사용 여부를 검사합니다.

## 관리하는 설정

| 앱 | 용도 | 설정 경로 |
|---|---|---|
| [AeroSpace](https://github.com/nikitabobko/AeroSpace) | 타일링 윈도우 관리자 | [`aerospace/`](aerospace/.config/aerospace) |
| [Ghostty](https://github.com/ghostty-org/ghostty) | 터미널 | [`ghostty/`](ghostty/.config/ghostty) |
| [Hammerspoon](https://github.com/Hammerspoon/hammerspoon) | macOS 자동화 및 단축키 | [`hammerspoon/`](hammerspoon/.hammerspoon) |
| [Herdr](https://herdr.dev) | 에이전트 작업공간 | [`herdr/`](herdr/.config/herdr) |
| [Rift](https://github.com/acsandmann/rift) | 윈도우 관리자 | [`rift/`](rift/.config/rift) |
| [Starship](https://github.com/starship/starship) | 셸 프롬프트 | [`starship/`](starship/.config/starship.toml) |
| [WezTerm](https://github.com/wezterm/wezterm) | 터미널 | [`wezterm/`](wezterm/.config/wezterm) |
| [Zed](https://github.com/zed-industries/zed) | 코드 에디터 | [`zed/`](zed/.config/zed) |
| [Zsh](https://www.zsh.org) | 셸 설정 | [`zsh/`](zsh) |

`homebrew/Brewfile`은 개발 환경 기록용이라 TUI가 다루지 않습니다.

## 빌드

TUI 소스는 [`tools/dotfiles-tui`](tools/dotfiles-tui)에 있습니다. Go와 `lipo`가 있는 macOS에서 다음 명령으로 테스트하고 `bin/setup`을 다시 빌드합니다.

```bash
./tools/dotfiles-tui/build
```

Go 테스트와 정적 검사를 실행한 뒤 arm64와 x86_64 실행 파일을 universal 바이너리로 합칩니다.
