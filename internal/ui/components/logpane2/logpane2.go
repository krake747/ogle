package logpane2

import tea "charm.land/bubbletea/v2"

// Model is a stub log pane component.
type Model struct{}

// New returns a Model.
func New() Model { return Model{} }

// Init satisfies tea.Model.
func (m Model) Init() tea.Cmd { return nil }

// Update satisfies tea.Model.
func (m Model) Update(tea.Msg) (Model, tea.Cmd) { return m, nil }

// View returns the log pane content.
func (m Model) View() string { return "" }
