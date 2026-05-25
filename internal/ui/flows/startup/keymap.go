package startup

import (
	"charm.land/bubbles/v2/help"
	"charm.land/bubbles/v2/key"
)

//nolint:gochecknoglobals // package-level key bindings are shared across all Model instances
var (
	keyUp     = key.NewBinding(key.WithKeys("up", "k"), key.WithHelp("↑/k", "up"))
	keyDown   = key.NewBinding(key.WithKeys("down", "j"), key.WithHelp("↓/j", "down"))
	keySelect = key.NewBinding(key.WithKeys("enter"), key.WithHelp("enter", "select"))
	keyQuit   = key.NewBinding(key.WithKeys("q", "ctrl+c"), key.WithHelp("q", "quit"))
)

type startupKeymap struct{}

func (k startupKeymap) ShortHelp() []key.Binding {
	return []key.Binding{keyUp, keyDown, keySelect, keyQuit}
}

func (k startupKeymap) FullHelp() [][]key.Binding {
	return [][]key.Binding{{keyUp, keyDown, keySelect, keyQuit}}
}

var _ help.KeyMap = startupKeymap{}
