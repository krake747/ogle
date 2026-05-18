package dashboard

import (
	"charm.land/bubbles/v2/help"
	"charm.land/bubbles/v2/key"
)

//nolint:gochecknoglobals // package-level key bindings are shared across all Model instances
var (
	keyQuit       = key.NewBinding(key.WithKeys("q", "ctrl+c"), key.WithHelp("q", "quit"))
	keySettings   = key.NewBinding(key.WithKeys(","), key.WithHelp(",", "settings"))
	keyToggleWrap = key.NewBinding(key.WithKeys("w"), key.WithHelp("w", "toggle wrap"))
)

const extraBindings = 2 // extra short-help entries beyond list and actions

// appKeymap merges the service list, action, and quit bindings into one KeyMap.
type appKeymap struct {
	list    help.KeyMap
	actions []key.Binding
}

func (k appKeymap) ShortHelp() []key.Binding {
	out := make([]key.Binding, 0, len(k.list.ShortHelp())+len(k.actions)+extraBindings)
	out = append(out, k.list.ShortHelp()...)
	out = append(out, k.actions...)
	out = append(out, keySettings, keyQuit)

	return out
}

func (k appKeymap) FullHelp() [][]key.Binding {
	return k.list.FullHelp()
}
