// Package project implements the project sub-flow: the state machine active
// after a compose project is successfully loaded. It is only reachable through
// the dashboard orchestrator, reflected by its location under flows/dashboard/.
package project

import (
	"context"
	"time"

	tea "charm.land/bubbletea/v2"

	"github.com/ma-tf/ogle/internal/domain"
	"github.com/ma-tf/ogle/internal/ui/flows/dashboard/project/states"
	"github.com/ma-tf/ogle/internal/ui/theme"
)

// Model is the project flow orchestrator.
type Model struct {
	ctx     context.Context
	current states.State
}

// New constructs a project Model initialised in the Dashboard state.
func New(
	ctx context.Context,
	project *domain.Project,
	th *theme.Theme,
	themeName string,
	poll time.Duration,
	logBufCap int,
	w, h int,
) Model {
	m := Model{ctx: ctx, current: states.NewDashboard(ctx, project, th, themeName, poll, logBufCap)}
	m.current.SetSize(w, h)

	return m
}

// Init implements tea.Model.
func (m Model) Init() tea.Cmd {
	return m.current.Init()
}

// Update implements tea.Model.
func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	// WindowSizeMsg is intercepted here to keep state dimensions current via
	// SetSize. The message is still forwarded to Update so states can return
	// commands or trigger transitions in response to resize.
	if sz, ok := msg.(tea.WindowSizeMsg); ok {
		m.current.SetSize(sz.Width, sz.Height)
	}

	next, cmd := m.current.Update(msg)
	m.current = next

	return m, cmd
}

// View implements tea.Model.
func (m Model) View() tea.View { return tea.NewView(m.current.View()) }
