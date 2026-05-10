// Package project implements the project sub-flow: the state machine active
// after a compose project is successfully loaded. It is only reachable through
// the dashboard orchestrator, reflected by its location under flows/dashboard/.
package project

import (
	tea "charm.land/bubbletea/v2"

	"github.com/ma-tf/ogle/internal/services/parser"
	"github.com/ma-tf/ogle/internal/ui/flows/dashboard/project/states"
)

// Model is the project flow orchestrator.
type Model struct {
	current states.State
}

// New constructs a project Model initialised in the Idle state.
func New(project *parser.Project) Model {
	return Model{current: states.NewIdle(project)}
}

// Init implements tea.Model.
func (m Model) Init() tea.Cmd {
	return m.current.Init()
}

// Update implements tea.Model.
func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	next, cmd := m.current.Update(msg)
	m.current = next

	return m, cmd
}

// View implements tea.Model.
func (m Model) View() tea.View { return tea.NewView(m.current.View()) }
