package dashboard2

import (
	"charm.land/bubbles/v2/help"
	"charm.land/bubbles/v2/key"
)

//nolint:gochecknoglobals // package-level key bindings are shared across all Model instances
var (
	keyQuit    = key.NewBinding(key.WithKeys("q", "ctrl+c"), key.WithHelp("q", "quit"))
	keyStop    = key.NewBinding(key.WithKeys("s"), key.WithHelp("s", "stop"))
	keyStart   = key.NewBinding(key.WithKeys("s"), key.WithHelp("s", "start"))
	keyRestart = key.NewBinding(key.WithKeys("r"), key.WithHelp("r", "restart"))
	keyRebuild = key.NewBinding(key.WithKeys("b"), key.WithHelp("b", "rebuild"))
)

// appKeymap merges the service list, action, and quit bindings into one KeyMap.
type appKeymap struct {
	list    help.KeyMap
	actions []key.Binding
}

func (k appKeymap) ShortHelp() []key.Binding {
	out := make([]key.Binding, 0, len(k.list.ShortHelp())+len(k.actions)+1)
	out = append(out, k.list.ShortHelp()...)
	out = append(out, k.actions...)
	out = append(out, keyQuit)

	return out
}

func (k appKeymap) FullHelp() [][]key.Binding {
	return k.list.FullHelp()
}
