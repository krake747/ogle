package helpbar

import "charm.land/bubbles/v2/key"

// dashboardKeys defines the bindings owned by this component.
type dashboardKeys struct {
	Quit key.Binding
}

func (k dashboardKeys) ShortHelp() []key.Binding {
	return []key.Binding{k.Quit}
}

func (k dashboardKeys) FullHelp() [][]key.Binding {
	return [][]key.Binding{k.ShortHelp()}
}
