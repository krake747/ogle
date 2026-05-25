package startup

import (
	"charm.land/bubbles/v2/help"
	"charm.land/bubbles/v2/key"
)

type startupKeymap struct{}

func (k startupKeymap) ShortHelp() []key.Binding {
	return []key.Binding{
		key.NewBinding(key.WithKeys("up", "k"), key.WithHelp("↑/k", "up")),
		key.NewBinding(key.WithKeys("down", "j"), key.WithHelp("↓/j", "down")),
		key.NewBinding(key.WithKeys("enter"), key.WithHelp("enter", "select")),
		key.NewBinding(key.WithKeys("q", "ctrl+c"), key.WithHelp("q", "quit")),
	}
}

func (k startupKeymap) FullHelp() [][]key.Binding {
	return [][]key.Binding{
		{
			key.NewBinding(key.WithKeys("up", "k"), key.WithHelp("↑/k", "up")),
			key.NewBinding(key.WithKeys("down", "j"), key.WithHelp("↓/j", "down")),
			key.NewBinding(key.WithKeys("enter"), key.WithHelp("enter", "select")),
			key.NewBinding(key.WithKeys("q", "ctrl+c"), key.WithHelp("q", "quit")),
		},
	}
}

var _ help.KeyMap = startupKeymap{}
