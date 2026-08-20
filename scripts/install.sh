#!/bin/sh

set -eu

repository=${DOTFILES_REPOSITORY:-https://github.com/DevooKim/my-dotfiles.git}
target=${DOTFILES_DIR:-"${HOME:?HOME is required}/.dotfiles"}
git_command=${DOTFILES_GIT:-git}

if [ "$(uname -s)" != "Darwin" ]; then
  printf 'dotfiles installer: macOS만 지원합니다.\n' >&2
  exit 1
fi

if [ -L "$target" ]; then
  printf 'dotfiles installer: 대상 경로가 심볼릭 링크입니다: %s\n' "$target" >&2
  exit 1
fi

if [ -e "$target" ]; then
  if [ ! -d "$target/.git" ]; then
    printf 'dotfiles installer: 기존 경로가 Git 저장소가 아닙니다: %s\n' "$target" >&2
    exit 1
  fi
else
  if ! command -v "$git_command" >/dev/null 2>&1; then
    printf 'dotfiles installer: git을 찾을 수 없습니다.\n' >&2
    exit 1
  fi
  printf 'dotfiles 저장소를 %s에 복제합니다.\n' "$target"
  "$git_command" clone -- "$repository" "$target"
fi

setup="$target/bin/setup"
if [ ! -x "$setup" ]; then
  printf 'dotfiles installer: 실행 파일을 찾을 수 없습니다: %s\n' "$setup" >&2
  exit 1
fi

exec "$setup"
