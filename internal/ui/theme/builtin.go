package theme

import "charm.land/lipgloss/v2"

// BuiltinNames returns the names of all built-in themes in display order.
func BuiltinNames() []string {
	return []string{"default", "catppuccino_mocha"}
}

// Default returns the default built-in theme using ANSI-256 palette colours.
func Default() *Theme {
	focused := lipgloss.Color("62")
	blurred := lipgloss.Color("240")

	return &Theme{
		BorderFocused: lipgloss.NewStyle().
			Border(lipgloss.NormalBorder()).
			BorderForeground(focused),
		BorderBlurred: lipgloss.NewStyle().
			Border(lipgloss.NormalBorder()).
			BorderForeground(blurred),
		ServiceListTitle: lipgloss.NewStyle().Bold(true).Foreground(blurred),
		HoverBackground:  lipgloss.Color("237"),
		StateRunning:     lipgloss.Color("2"),
		StateExited:      lipgloss.Color("1"),
		StatePaused:      lipgloss.Color("3"),
		StateTransient:   lipgloss.Color("214"),
		StateMuted:       lipgloss.Color("240"),
		ActionError:      lipgloss.Color("1"),
	}
}

