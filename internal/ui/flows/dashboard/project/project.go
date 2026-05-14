// Package project implements the project sub-flow: the state machine active
// after a compose project is successfully loaded. It is only reachable through
// the dashboard orchestrator, reflected by its location under flows/dashboard/.
package project

import (
	"context"
	"time"

	tea "charm.land/bubbletea/v2"
	zone "github.com/lrstanley/bubblezone/v2"

	"github.com/ma-tf/ogle/internal/domain"
	logs "github.com/ma-tf/ogle/internal/services/docker/logs"
	"github.com/ma-tf/ogle/internal/ui/flows/dashboard/project/states"
	"github.com/ma-tf/ogle/internal/ui/theme"
)

// Model is the project flow orchestrator.
type Model struct {
	ctx     context.Context
	current states.State
	w, h    int
}

// New constructs a project Model initialised in the Dashboard state.
func New(
	ctx context.Context,
	project *domain.Project,
	th *theme.Theme,
	themeName string,
	poll time.Duration,
	logBufCap int,
	zm *zone.Manager,
	w, h int,
) Model {
	m := Model{
		ctx:     ctx,
		current: states.NewDashboard(ctx, project, th, themeName, poll, logBufCap, logs.New(), zm),
		w:       w,
		h:       h,
	}
	m.current.SetSize(w, h)

	return m
}

// Init implements tea.Model.
func (m Model) Init() tea.Cmd {
	return m.current.Init()
}

// Update implements tea.Model.
func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	if sz, ok := msg.(tea.WindowSizeMsg); ok {
		m.w = sz.Width
		m.h = sz.Height
	}

	next, cmd := m.current.Update(msg)
	next.SetSize(m.w, m.h)
	m.current = next

	return m, cmd
}

// View implements tea.Model.
func (m Model) View() tea.View { return tea.NewView(m.current.View()) }
