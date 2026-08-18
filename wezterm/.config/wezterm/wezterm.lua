-- Pullin the wezterm API
local wezterm = require("wezterm")

-- This will hold the configuration
local config = wezterm.config_builder()

local act = wezterm.action

-- 주 폰트는 Meslo Bold, 한글은 D2Coding Bold로 fallback (한글도 두껍게)
config.font = wezterm.font_with_fallback({{
    -- family = "MesloLGS Nerd Font Mono",
    family = "Google Sans Code",
    weight = "Medium"
}, {
    family = "D2Coding",
    weight = "Medium"
}})
config.font_size = 16

-- Ghostty(CoreText) 렌더링에 가깝게 맞추기
config.freetype_load_target = "Light"
config.freetype_render_target = "Normal"

-- keep adding configuration options here

config.hide_tab_bar_if_only_one_tab = true
-- config.enable_tab_bar = false
config.window_decorations = "RESIZE"

config.send_composed_key_when_left_alt_is_pressed = false
config.use_ime = true
config.use_dead_keys = false
config.enable_kitty_keyboard = true

config.leader = {
    key = "q",
    mods = "ALT",
    timeout_milliseconds = 1000
}

config.keys = { -- Cmd+Left / Cmd+Right: 줄 맨 앞/뒤로 이동
{
    key = "LeftArrow",
    mods = "CMD",
    action = act.SendKey({
        key = "a",
        mods = "CTRL"
    })
}, {
    key = "RightArrow",
    mods = "CMD",
    action = act.SendKey({
        key = "e",
        mods = "CTRL"
    })
}, -- Cmd+Backspace: 커서 앞쪽 줄 전체 지우기
{
    key = "Backspace",
    mods = "CMD",
    action = act.SendKey({
        key = "u",
        mods = "CTRL"
    })
}, -- Option+Left / Option+Right: 단어 단위 이동
{
    key = "LeftArrow",
    mods = "OPT",
    action = act.SendString("\x1bb")
}, {
    key = "RightArrow",
    mods = "OPT",
    action = act.SendString("\x1bf")
}}

config.colors = {
    -- foreground = "#CBE0F0",
    foreground = "#ffffff",
    background = "#011423"
    -- background = "#011423",
    -- cursor_bg = "#47FF9C",
    -- cursor_border = "#47FF9C",
    -- cursor_fg = "#011423",
    -- split = "#47FF9C",
    -- selection_bg = "#033259",
    -- selection_fg = "#CBE0F0",
    -- ansi = { "#214969", "#E52E2E", "#44FFB1", "#FFE073", "#0FC5ED", "#a277ff", "#24EAF7", "#24EAF7" },
    -- brights = { "#214969", "#E52E2E", "#44FFB1", "#FFE073", "#A277FF", "#a277ff", "#24EAF7", "#24EAF7" },
}

config.window_background_opacity = 0.9
config.macos_window_background_blur = 5

return config
