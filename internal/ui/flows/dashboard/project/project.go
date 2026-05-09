// Package project implements the project sub-flow: the state machine active
// after a compose project is successfully loaded. It is only reachable through
// the dashboard orchestrator, reflected by its location under flows/dashboard/.
package project

import (
	tea "charm.land/bubbletea/v2"

	"github.com/ma-tf/ogle/internal/compose"
	"github.com/ma-tf/ogle/internal/ui/flows"
	"github.com/ma-tf/ogle/internal/ui/flows/dashboard/project/states"
)

// Model is the project flow orchestrator. It implements flows.State.
type Model struct {
	current states.State
}

// New constructs a project Model initialised in the Idle state.
func New(project *compose.Project) Model {
	return Model{current: states.NewIdle(project)}
}

// Init fires the first command for the current state.
func (m Model) Init() tea.Cmd { return m.current.Init() }

// Update delegates to the current state.
func (m Model) Update(msg tea.Msg) (flows.State, tea.Cmd) {
	next, cmd := m.current.Update(msg)
	m.current = next

	return m, cmd
}

// View delegates rendering to the current state.
func (m Model) View() string { return m.current.View() }
