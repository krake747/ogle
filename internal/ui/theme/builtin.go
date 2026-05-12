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
		URLHover:         lipgloss.NewStyle().Underline(true),
		StateRunning:     lipgloss.Color("2"),
		StateExited:      lipgloss.Color("1"),
		StatePaused:      lipgloss.Color("3"),
		StateTransient:   lipgloss.Color("214"),
		StateMuted:       lipgloss.Color("240"),
		ActionError:      lipgloss.Color("1"),
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
		URLHover:         lipgloss.NewStyle().Underline(true),
		StateRunning:     lipgloss.Color("#a6e3a1"),
		StateExited:      lipgloss.Color("#f38ba8"),
		StatePaused:      lipgloss.Color("#f9e2af"),
		StateTransient:   lipgloss.Color("#fab387"),
		StateMuted:       lipgloss.Color("#6c7086"),
		ActionError:      lipgloss.Color("#f38ba8"),
	}
}
