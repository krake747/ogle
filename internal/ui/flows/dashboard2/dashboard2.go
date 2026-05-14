// Package dashboard2 implements the project dashboard flow.
// Stub implementation — displays "dashboard2" as a placeholder.
package dashboard2

import tea "charm.land/bubbletea/v2"

// Model is a minimal tea.Model stub.
type Model struct{}

// New returns a new Model.
func New() tea.Model {
	return Model{}
}

// Init implements tea.Model.
func (m Model) Init() tea.Cmd {
	return nil
}

// Update implements tea.Model.
func (m Model) Update(_ tea.Msg) (tea.Model, tea.Cmd) {
	return m, nil
}

// View implements tea.Model.
func (m Model) View() tea.View {
	return tea.NewView("dashboard2")
}
