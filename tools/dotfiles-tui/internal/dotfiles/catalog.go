package dotfiles

import "path/filepath"

// packagesDirectory holds every selectable package inside the repository.
const packagesDirectory = "packages"

// Package describes one independently selectable dotfiles package.
type Package struct {
	Name        string
	Description string
	Command     string
}

// PackageRoot returns the repository path that holds the named package.
func PackageRoot(repo, name string) string {
	return filepath.Join(repo, packagesDirectory, name)
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
		},
		{Name: "herdr", Description: "Herdr agent workspace", Command: "herdr"},
		{Name: "rift", Description: "Rift window manager", Command: "rift"},
		{Name: "starship", Description: "Starship shell prompt", Command: "starship"},
		{Name: "wezterm", Description: "WezTerm terminal", Command: "wezterm"},
		{Name: "zed", Description: "Zed editor", Command: "zed"},
		{Name: "zsh", Description: "Zsh shell", Command: "zsh"},
	}
}
