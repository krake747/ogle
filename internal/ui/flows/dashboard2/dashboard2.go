// Package dashboard2 implements the project dashboard flow.
package dashboard2

import (
	"context"

	"charm.land/bubbles/v2/spinner"
	tea "charm.land/bubbletea/v2"
	zone "github.com/lrstanley/bubblezone/v2"

	"github.com/ma-tf/ogle/internal/domain"
	"github.com/ma-tf/ogle/internal/msgs"
	"github.com/ma-tf/ogle/internal/services/docker/connection"
	"github.com/ma-tf/ogle/internal/ui/components/daemonstatus"
	"github.com/ma-tf/ogle/internal/ui/components/servicelist"
	"github.com/ma-tf/ogle/internal/ui/theme"
)

// Model is the dashboard flow orchestrator.
type Model struct {
	conn        *connection.Machine
	daemon      daemonstatus.Model
	serviceList servicelist.Model
	selected    string
	w, h        int
}

// New returns a Model in the Connecting state.
func New(
	ctx context.Context,
	project *domain.Project,
	th *theme.Theme,
	zm *zone.Manager,
	w, h int,
) tea.Model {
	conn := connection.New()

	return Model{
		conn:        conn,
		daemon:      daemonstatus.New(ctx, conn, th),
		serviceList: servicelist.New(project, th, zm, 0, 0),
		selected:    "",
		w:           w,
		h:           h,
	}
}

// Init delegates to the daemon status sub-model.
func (m Model) Init() tea.Cmd {
	return m.daemon.Init()
}

// Update handles dashboard-level messages and forwards daemon messages.
func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmd tea.Cmd

	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.w = msg.Width
		m.h = msg.Height

		listH := max(m.h-1, 0)
		m.serviceList = m.serviceList.SetBounds(0, 0, m.w, listH)

	case msgs.DaemonMsg, spinner.TickMsg:
		m.daemon, cmd = m.daemon.Update(msg)

		return m, cmd

	case msgs.ServiceSelected:
		m.selected = msg.Service.Name

		return m, nil
	}

	m.serviceList, cmd = m.serviceList.Update(msg)

	return m, cmd
}

// View renders the daemon status header above the service list.
func (m Model) View() tea.View {
	statusContent := m.daemon.View().Content
	listContent := m.serviceList.View()

	return tea.NewView(statusContent + "\n" + listContent)
}
