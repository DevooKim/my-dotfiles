# 이전 설정에서 남은 alias를 제거한다.
unalias gco gch 2>/dev/null

# 브랜치 자동완성으로 원하는 브랜치를 checkout한다.
gco() {
  git checkout "$@"
}

# 인자나 클립보드로 가장 잘 맞는 로컬 브랜치를 checkout한다.
gcoo() {
  local query branch

  if (( $# )); then
    query="$*"
  else
    query="$(pbpaste 2>/dev/null)"
  fi

  command git rev-parse --git-dir >/dev/null 2>&1 || {
    print -u2 'gcoo: not inside a Git repository'
    return 1
  }

  (( $+commands[fzf] )) || {
    print -u2 'gcoo: fzf is not installed'
    return 1
  }

  branch="$(
    command git for-each-ref --sort=-authordate --format='%(refname:short)' refs/heads |
      command fzf --tiebreak=begin,length --filter="$query" |
      command head -n 1
  )"

  [[ -n "$branch" ]] || {
    print -u2 "gcoo: no branch matched: $query"
    return 1
  }

  command git checkout "$branch"
}

# gco에 로컬·원격 브랜치 자동완성을 제공한다.
_gco_branches() {
  local -a branches
  branches=("${(@f)$(git for-each-ref --format='%(refname:short)' refs/heads refs/remotes 2>/dev/null)}")
  (( $#branches )) && _describe 'branch' branches
}
compdef _gco_branches gco

# 커밋을 detached HEAD 상태로 checkout한다.
gch() {
  git checkout --detach "$@"
}

# gch에 최근 커밋 200개의 자동완성을 제공한다.
_gch_commits() {
  local -a commits
  commits=("${(@f)$(git log --all --max-count=200 --format='%h:%s' 2>/dev/null)}")
  (( $#commits )) && _describe 'commit' commits
}
compdef _gch_commits gch

typeset -g _prompt_skip_newline=1

# clear와 cls 다음 프롬프트의 빈 줄을 생략한다.
_prompt_spacing_preexec() {
  case "$1" in
    clear|clear\ *|cls|cls\ *) _prompt_skip_newline=1 ;;
  esac
}

# 생략 대상이 아니면 프롬프트 사이에 빈 줄 하나를 추가한다.
_prompt_spacing_precmd() {
  if (( _prompt_skip_newline )); then
    _prompt_skip_newline=0
  else
    print
  fi
}

autoload -Uz add-zsh-hook
add-zsh-hook preexec _prompt_spacing_preexec
add-zsh-hook precmd _prompt_spacing_precmd
