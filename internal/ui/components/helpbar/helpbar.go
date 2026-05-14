// Package helpbar provides a self-contained help bar component that owns its
// key bindings and delegates rendering to a bubbles help.Model.
package helpbar

import (
	"charm.land/bubbles/v2/help"
	"charm.land/bubbles/v2/key"
	tea "charm.land/bubbletea/v2"
)

// Model is a value-type sub-model that renders a help bar. It owns the
// dashboard-level key bindings and accepts external keymaps for composition.
type Model struct {
	help help.Model
	keys dashboardKeys
	list help.KeyMap
}

// New returns a Model with default dashboard key bindings.
func New() Model {
	h := help.New()

	return Model{
		help: h,
		keys: dashboardKeys{
			Quit: key.NewBinding(key.WithKeys("q", "ctrl+c"), key.WithHelp("q", "quit")),
		},
		list: nil,
	}
}

// SetWidth returns a copy with the help bar width set.
func (m Model) SetWidth(w int) Model {
	m.help.SetWidth(w)

	return m
}

// WithListKeys returns a copy that includes the given keymap in the help bar.
// Typically the list model from the parent dashboard.
func (m Model) WithListKeys(l help.KeyMap) Model {
	m.list = l

	return m
}

// Quit returns the quit key binding for use in key dispatch.
func (m Model) Quit() key.Binding {
	return m.keys.Quit
}

// Init satisfies tea.Model.
func (m Model) Init() tea.Cmd { return nil }

// Update satisfies tea.Model.
func (m Model) Update(_ tea.Msg) (Model, tea.Cmd) { return m, nil }

// View renders the help bar with all composed bindings.
func (m Model) View() string {
	km := combinedKeyMap{dash: m.keys, list: m.list}

	return m.help.View(km)
}
