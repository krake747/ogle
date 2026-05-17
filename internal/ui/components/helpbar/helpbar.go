// Package helpbar provides a self-contained help bar component that renders a
// keymap received via BindingsMsg.
package helpbar

import (
	"charm.land/bubbles/v2/help"
	tea "charm.land/bubbletea/v2"

	"github.com/ma-tf/ogle/internal/msgs"
)

// Model is a value-type sub-model that renders a help bar.
type Model struct {
	help   help.Model
	keymap help.KeyMap
}

// New returns a Model.
func New() Model {
	return Model{
		help:   help.New(),
		keymap: nil,
	}
}

// Init satisfies tea.Model.
func (m Model) Init() tea.Cmd {
	return nil
}

// Update satisfies tea.Model.
func (m Model) Update(msg tea.Msg) (Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.help.SetWidth(msg.Width)
	case msgs.BindingsMsg:
		m.keymap = msg.Keymap
	}

	return m, nil
}

// View renders the help bar with the current keymap.
func (m Model) View() tea.View {
	if m.keymap == nil {
		return tea.NewView("")
	}

	return tea.NewView(m.help.View(m.keymap))
}
