// Package watching provides the disconnected state shown when the project
// file disappears at runtime. This is a stub — the return path is not yet
// implemented.
package watching

import tea "charm.land/bubbletea/v2"

// Model is the disconnected watching state.
type Model struct {
	File string
}

// New returns a Model watching for file to reappear.
func New(file string) Model {
	return Model{
		File: file,
	}
}

// Init implements tea.Model.
func (m Model) Init() tea.Cmd {
	return nil
}

// Update implements tea.Model. Currently a no-op stub.
func (m Model) Update(tea.Msg) (tea.Model, tea.Cmd) {
	return m, nil
}

// View implements tea.Model.
func (m Model) View() tea.View {
	return tea.NewView("compose file unavailable — waiting...")
}
