package helpbar

import (
	"charm.land/bubbles/v2/help"
	"charm.land/bubbles/v2/key"
)

// combinedKeyMap merges dashboard, list, and action bindings into one KeyMap.
type combinedKeyMap struct {
	dash dashboardKeys
	list help.KeyMap
}

func (c combinedKeyMap) ShortHelp() []key.Binding {
	var bindings []key.Binding

	if c.list != nil {
		bindings = append(bindings, c.list.ShortHelp()...)
	}

	bindings = append(bindings, c.dash.ShortHelp()...)

	return bindings
}

func (c combinedKeyMap) FullHelp() [][]key.Binding {
	return [][]key.Binding{c.dash.ShortHelp()}
}
