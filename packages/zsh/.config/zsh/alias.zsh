# Navigation
alias ..='cd ..'
alias ...='cd ../..'
alias ls='lsd'
alias ll='lsd -lah'
alias la='lsd -A'
alias reload='source ~/.zshrc'
alias cls='clear'

# Git
alias g='git'
alias gst='git status --short --branch'
alias gd='git diff'
alias gl='git log --oneline --decorate --graph -15'
alias ga='git add'
alias gc='git commit'
alias gp='git push'
alias gpl='git pull --ff-only'
function git_main_branch() {
  command git rev-parse --git-dir &>/dev/null || return

  local remote ref
  for ref in refs/{heads,remotes/{origin,upstream}}/{main,master}; do
    if command git show-ref --verify --quiet "$ref"; then
      echo "${ref:t}"
      return 0
    fi
  done

  for remote in origin upstream; do
    ref=$(command git rev-parse --abbrev-ref "$remote/HEAD" 2>/dev/null)
    if [[ "$ref" == "$remote"/* ]]; then
      echo "${ref#"$remote/"}"
      return 0
    fi
  done

  echo master
  return 1
}
alias gcm='git checkout "$(git_main_branch)"'
alias gcd='git checkout develop'
alias ggpull='git pull origin "$(git branch --show-current)"'
alias ggpush='git push origin "$(git branch --show-current)"'
alias cpbr='git rev-parse --abbrev-ref HEAD | pbcopy'

# Tools
(( $+functions[z] )) && alias j=z
alias p=pnpm
alias zshconfig='vim ~/.zshrc'
