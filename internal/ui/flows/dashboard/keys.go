package dashboard

import (
	"charm.land/bubbles/v2/key"
	"charm.land/bubbles/v2/list"
)

type dashboardKeyMap struct {
	Quit          key.Binding
	Zoom          key.Binding
	ToggleLabels  key.Binding
	Settings      key.Binding
	ActionStop    key.Binding
	ActionStart   key.Binding
	ActionRestart key.Binding
	ActionRebuild key.Binding
	ScrollUp      key.Binding
	ScrollDown    key.Binding
}

func (k dashboardKeyMap) ShortHelp() []key.Binding {
	return []key.Binding{k.Quit, k.Zoom, k.ToggleLabels}
}

func (k dashboardKeyMap) FullHelp() [][]key.Binding {
	return [][]key.Binding{{k.Quit, k.Zoom, k.ToggleLabels}}
}

// combinedKeyMap merges the dashboard-level bindings with the service list
// bindings for the help bar.
type combinedKeyMap struct {
	dashboard      dashboardKeyMap
	list           list.KeyMap
	actionBindings []key.Binding
}

// shortHelpBaseCount is the number of fixed bindings in combinedKeyMap.ShortHelp.
const shortHelpBaseCount = 8

func (c combinedKeyMap) ShortHelp() []key.Binding {
	bindings := make([]key.Binding, 0, shortHelpBaseCount+len(c.actionBindings))
	bindings = append(bindings,
		c.list.CursorUp,
		c.list.CursorDown,
		c.list.Filter,
		c.list.ClearFilter,
		c.dashboard.Zoom,
		c.dashboard.ToggleLabels,
		c.dashboard.Settings,
		c.dashboard.Quit,
	)

	return append(bindings, c.actionBindings...)
}

func (c combinedKeyMap) FullHelp() [][]key.Binding {
	return [][]key.Binding{
		{c.list.CursorUp, c.list.CursorDown, c.list.NextPage, c.list.PrevPage},
		{
			c.list.Filter,
			c.list.ClearFilter,
			c.list.AcceptWhileFiltering,
			c.list.CancelWhileFiltering,
		},
		{c.dashboard.Zoom, c.dashboard.ToggleLabels, c.dashboard.Quit},
	}
}

//nolint:gochecknoglobals // list of key bindings should be global and immutable
var defaultDashboardKeys = dashboardKeyMap{
	Quit: key.NewBinding(
		key.WithKeys("q"),
		key.WithHelp("q", "quit"),
	),
	Zoom: key.NewBinding(
		key.WithKeys("z"),
		key.WithHelp("z", "fullscreen"),
	),
	ToggleLabels: key.NewBinding(
		key.WithKeys("l"),
		key.WithHelp("l", "labels"),
	),
	Settings: key.NewBinding(
		key.WithKeys(","),
		key.WithHelp(",", "settings"),
	),
	ActionStop: key.NewBinding(
		key.WithKeys("s"),
		key.WithHelp("s", "stop"),
	),
	ActionStart: key.NewBinding(
		key.WithKeys("s"),
		key.WithHelp("s", "start"),
	),
	ActionRestart: key.NewBinding(
		key.WithKeys("r"),
		key.WithHelp("r", "restart"),
	),
	ActionRebuild: key.NewBinding(
		key.WithKeys("b"),
		key.WithHelp("b", "rebuild"),
	),
	ScrollUp: key.NewBinding(
		key.WithKeys("pgup"),
		key.WithHelp("pgup", "scroll up"),
	),
	ScrollDown: key.NewBinding(
		key.WithKeys("pgdn"),
		key.WithHelp("pgdn", "scroll down"),
	),
}

type keyBinding struct {
	binding key.Binding
	handle  func()
}
