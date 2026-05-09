// Package app is the minimal tea.Model adapter shim. Its only job is to
// bridge flows.State → tea.Model so that the Bubble Tea runtime can drive the
// flow tree. No domain logic lives here.
package app

import (
	"context"
	"log/slog"

	tea "charm.land/bubbletea/v2"

	"github.com/ma-tf/ogle/config"
	"github.com/ma-tf/ogle/internal/ui/flows"
	"github.com/ma-tf/ogle/internal/ui/flows/dashboard"
)

// model wraps a flows.State so it satisfies tea.Model. A type cannot satisfy
// both flows.State and tea.Model simultaneously in Go (same method name,
// conflicting Update signatures), which is why this adapter exists.
type model struct{ current flows.State }

func (m model) Init() tea.Cmd { return m.current.Init() }

func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	next, cmd := m.current.Update(msg)

	return model{current: next}, cmd
}

func (m model) View() tea.View { return tea.NewView(m.current.View()) }

// Setup creates and runs a new Bubble Tea program. ctx is passed via
// tea.WithContext so that external cancellation propagates into the TUI
// runtime. All watcher lifecycle and flow orchestration is owned by
// dashboard.New.
func Setup(ctx context.Context, cfg config.Config, logger *slog.Logger) *tea.Program {
	return tea.NewProgram(
		model{current: dashboard.New(cfg, logger)},
		tea.WithContext(ctx),
	)
}
