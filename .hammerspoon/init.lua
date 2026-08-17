-- Input
require('modules.input.vim_esc')
require('modules.input.tmux_lang')

hs.hotkey.bind({'ctrl', 'option', 'shift'}, 'R', hs.reload)

hs.alert.show("Config loaded")
