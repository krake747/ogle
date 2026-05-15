// Package dashboard implements the dashboard sub-flow: the state machine active
// after a compose project is successfully loaded. It is only reachable through
// the app orchestrator.
package dashboard

import (
	"context"

	tea "charm.land/bubbletea/v2"
	zone "github.com/lrstanley/bubblezone/v2"

	"github.com/ma-tf/ogle/internal/domain"
	logs "github.com/ma-tf/ogle/internal/services/docker/logs"
	"github.com/ma-tf/ogle/internal/ui/theme"
)

// Model is the dashboard flow orchestrator.
type Model struct {
	ctx     context.Context
	current State
	w, h    int
}

// New constructs a dashboard Model initialised in the Screen state.
func New(
	ctx context.Context,
	project *domain.Project,
	th *theme.Theme,
	themeName string,
	logBufCap int,
	zm *zone.Manager,
	w, h int,
) Model {
	m := Model{
		ctx:     ctx,
		current: NewScreen(ctx, project, th, themeName, logBufCap, logs.New(), zm),
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
