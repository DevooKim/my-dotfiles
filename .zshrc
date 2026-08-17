# Paths
typeset -U path PATH
path=(
  "$HOME/.local/bin"
  "$HOME/.cargo/bin"
  "$HOME/.opencode/bin"
  ${path:#$HOME/.antigravity/antigravity/bin}
)

# History
HISTFILE="$HOME/.zsh_history"
HISTSIZE=5000
SAVEHIST=$HISTSIZE
setopt APPEND_HISTORY
setopt SHARE_HISTORY
setopt HIST_IGNORE_SPACE
setopt HIST_SAVE_NO_DUPS
setopt HIST_FIND_NO_DUPS

# Completion
fpath=("$HOME/.docker/completions" $fpath)
autoload -Uz compinit
typeset zcompdump="${ZDOTDIR:-$HOME}/.zcompdump"
typeset -a recent_zcompdump
recent_zcompdump=("$zcompdump"(N.mh-24))
if (( $#recent_zcompdump )); then
  compinit -C -d "$zcompdump"
else
  compinit -d "$zcompdump"
fi
unset zcompdump recent_zcompdump
bindkey -e

[[ -s "$HOME/.bun/_bun" ]] && source "$HOME/.bun/_bun"

# Tools and plugins
if (( $+commands[fnm] )); then
  eval "$(fnm env --use-on-cd --log-level=quiet --shell zsh)"
fi

if (( $+commands[fzf] )); then
  : ${FZF_DEFAULT_OPTS:="--height=40% --layout=reverse --border"}
  export FZF_DEFAULT_OPTS
  source <(fzf --zsh)
fi

[[ -r "$HOME/.local/share/zsh/plugins/fzf-tab/fzf-tab.plugin.zsh" ]] && \
  source "$HOME/.local/share/zsh/plugins/fzf-tab/fzf-tab.plugin.zsh"
zstyle ':fzf-tab:*' use-fzf-default-opts yes

[[ -r "$HOME/.local/share/zsh/plugins/zsh-autosuggestions/zsh-autosuggestions.zsh" ]] && \
  source "$HOME/.local/share/zsh/plugins/zsh-autosuggestions/zsh-autosuggestions.zsh"

if (( $+commands[zoxide] )); then
  eval "$(zoxide init zsh)"
fi

if (( $+commands[direnv] )); then
  eval "$(direnv hook zsh)"
fi

# Aliases
[[ -r "$HOME/.dotfiles/.config/zsh/alias.zsh" ]] && \
  source "$HOME/.dotfiles/.config/zsh/alias.zsh"

if (( $+commands[starship] )); then
  eval "$(starship init zsh)"
fi

# Functions and hooks
[[ -r "$HOME/.dotfiles/.config/zsh/functions.zsh" ]] && \
  source "$HOME/.dotfiles/.config/zsh/functions.zsh"

# Must be loaded after all other ZLE plugins.
[[ -r "$HOME/.local/share/zsh/plugins/zsh-syntax-highlighting/zsh-syntax-highlighting.zsh" ]] && \
  source "$HOME/.local/share/zsh/plugins/zsh-syntax-highlighting/zsh-syntax-highlighting.zsh"
