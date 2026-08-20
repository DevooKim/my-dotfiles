# Navigation
alias ..='cd ..'
alias ...='cd ../..'
alias ll='ls -lah'
alias la='ls -A'
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
alias ggpull='git pull origin "$(git branch --show-current)"'
alias ggpush='git push origin "$(git branch --show-current)"'
alias cpbr='git rev-parse --abbrev-ref HEAD | pbcopy'

# Tools
(( $+functions[z] )) && alias j=z
alias p=pnpm
alias zshconfig='vim ~/.zshrc'
