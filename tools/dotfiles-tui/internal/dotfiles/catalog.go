package dotfiles

// Package describes one independently selectable dotfiles package.
type Package struct {
	Name        string
	Description string
	Command     string
	References  []string
}

// Catalog returns the packages in their stable UI order.
func Catalog() []Package {
	return []Package{
		{Name: "aerospace", Description: "AeroSpace window manager", Command: "aerospace"},
		{Name: "ghostty", Description: "Ghostty terminal", Command: "ghostty"},
		{
			Name:        "hammerspoon",
			Description: "Hammerspoon automation",
			Command:     "hs",
			References:  []string{".hammerspoon/modules/input/tmux_lang.lua"},
		},
		{Name: "herdr", Description: "Herdr agent workspace", Command: "herdr"},
		{Name: "rift", Description: "Rift window manager", Command: "rift"},
		{Name: "starship", Description: "Starship shell prompt", Command: "starship"},
		{Name: "wezterm", Description: "WezTerm terminal", Command: "wezterm"},
		{Name: "zed", Description: "Zed editor", Command: "zed"},
		{Name: "zsh", Description: "Zsh shell", Command: "zsh"},
	}
}
