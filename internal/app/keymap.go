package app

import (
	"charm.land/bubbles/v2/key"
)

//nolint:gochecknoglobals // package-level key bindings
var watchingQuitKey = key.NewBinding(key.WithKeys("q", "ctrl+c"), key.WithHelp("q", "quit"))

type watchingKeymap struct{}

func (k watchingKeymap) ShortHelp() []key.Binding {
	return []key.Binding{watchingQuitKey}
}

func (k watchingKeymap) FullHelp() [][]key.Binding {
	return [][]key.Binding{{watchingQuitKey}}
}
