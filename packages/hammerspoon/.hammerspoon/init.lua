-- Input
require('modules.input.vim_esc')

hs.hotkey.bind({'ctrl', 'option', 'shift'}, 'R', hs.reload)

hs.alert.show("Config loaded")
