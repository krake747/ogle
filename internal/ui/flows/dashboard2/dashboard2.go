// Package dashboard2 implements the project dashboard flow.
package dashboard2

import (
	"context"

	"charm.land/bubbles/v2/spinner"
	tea "charm.land/bubbletea/v2"

	"github.com/ma-tf/ogle/internal/msgs"
	"github.com/ma-tf/ogle/internal/services/docker/connection"
	"github.com/ma-tf/ogle/internal/ui/components/daemonstatus"
	"github.com/ma-tf/ogle/internal/ui/theme"
)

// Model is the dashboard flow orchestrator.
type Model struct {
	conn   *connection.Machine
	daemon daemonstatus.Model
	w, h   int
}

// New returns a Model in the Connecting state.
func New(ctx context.Context, th *theme.Theme) tea.Model {
	conn := connection.New()

	return Model{
		conn:   conn,
		daemon: daemonstatus.New(ctx, conn, th),
		w:      0,
		h:      0,
	}
}

// Init delegates to the daemon status sub-model.
func (m Model) Init() tea.Cmd {
	return m.daemon.Init()
}

// Update handles dashboard-level messages and forwards daemon messages.
func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.w = msg.Width
		m.h = msg.Height

	case msgs.DaemonMsg, spinner.TickMsg:
		var cmd tea.Cmd

		m.daemon, cmd = m.daemon.Update(msg)

		return m, cmd
	}

	return m, nil
}

// View renders daemon connection status.
func (m Model) View() tea.View {
	return m.daemon.View()
}
