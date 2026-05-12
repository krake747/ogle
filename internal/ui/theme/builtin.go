package theme

import "charm.land/lipgloss/v2"

// Default returns the default built-in theme using ANSI-256 palette colours.
func Default() *Theme {
	focused := lipgloss.Color("62")
	blurred := lipgloss.Color("240")

	return &Theme{
		BorderFocused:    lipgloss.NewStyle().Border(lipgloss.NormalBorder()).BorderForeground(focused),
		BorderBlurred:    lipgloss.NewStyle().Border(lipgloss.NormalBorder()).BorderForeground(blurred),
		ServiceListTitle: lipgloss.NewStyle().Bold(true).Foreground(blurred),
		HoverBackground:  lipgloss.Color("237"),
	}
}

// CatppuccinoMocha returns a theme based on the Catppuccin Mocha palette.
func CatppuccinoMocha() *Theme {
	mauve := lipgloss.Color("#cba6f7")
	overlay0 := lipgloss.Color("#6c7086")
	surface0 := lipgloss.Color("#313244")

	return &Theme{
		BorderFocused:    lipgloss.NewStyle().Border(lipgloss.NormalBorder()).BorderForeground(mauve),
		BorderBlurred:    lipgloss.NewStyle().Border(lipgloss.NormalBorder()).BorderForeground(overlay0),
		ServiceListTitle: lipgloss.NewStyle().Bold(true).Foreground(overlay0),
		HoverBackground:  surface0,
	}
}
