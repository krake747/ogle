// Package dashboard implements the project dashboard flow.
package dashboard

import (
	"context"

	"charm.land/bubbles/v2/key"
	"charm.land/bubbles/v2/spinner"
	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
	zone "github.com/lrstanley/bubblezone/v2"

	"github.com/ma-tf/ogle/internal/domain"
	"github.com/ma-tf/ogle/internal/msgs"
	svcdocker "github.com/ma-tf/ogle/internal/services/docker"
	"github.com/ma-tf/ogle/internal/services/docker/connection"
	"github.com/ma-tf/ogle/internal/ui/components/daemonstatus"
	"github.com/ma-tf/ogle/internal/ui/components/helpbar"
	"github.com/ma-tf/ogle/internal/ui/components/servicelist"
	"github.com/ma-tf/ogle/internal/ui/components/servicepanel"
	"github.com/ma-tf/ogle/internal/ui/theme"
)

// Model is the dashboard flow orchestrator.
type Model struct {
	ctx         context.Context
	project     *domain.Project
	conn        *connection.Machine
	daemon      daemonstatus.Model
	serviceList servicelist.Model
	panel       servicepanel.Model
	helpbar     helpbar.Model
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

	svcList := servicelist.New(project, th, zm, w)

	return Model{
		ctx:         ctx,
		project:     project,
		conn:        conn,
		daemon:      daemonstatus.New(ctx, conn, th),
		serviceList: svcList,
		panel:       servicepanel.New(project, th, w, h),
		helpbar:     helpbar.New(),
		w:           w,
		h:           h,
	}
}

// Init fires the daemon status init and sends the keymap to the helpbar.
func (m Model) Init() tea.Cmd {
	return tea.Batch(
		m.daemon.Init(),
		m.serviceList.Init(),
		m.panel.Init(),
		m.helpbar.Init(),
		func() tea.Msg {
			return msgs.BindingsMsg{
				Keymap: appKeymap{
					list: m.serviceList,
					actions: []key.Binding{
						servicelist.KeyStop,
						servicelist.KeyStart,
						servicelist.KeyRestart,
						servicelist.KeyRebuild,
					},
				},
			}
		},
	)
}

// Update handles dashboard-level messages and forwards daemon messages.
func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmd, panCmd, helpCmd tea.Cmd

	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.w = msg.Width
		m.h = msg.Height

	case msgs.DaemonMsg, spinner.TickMsg:
		m.daemon, cmd = m.daemon.Update(msg)
		m.panel, panCmd = m.panel.Update(msg)
		cmd = tea.Batch(cmd, panCmd)

		return m, cmd

	case msgs.StatePollTick:
		m.panel, panCmd = m.panel.Update(msg)

		return m, tea.Batch(
			svcdocker.Ps(m.ctx, m.project.File, m.project.Name),
			panCmd,
		)

	case tea.KeyPressMsg:
		if key.Matches(msg, keyQuit) {
			return m, tea.Quit
		}

	case msgs.ServiceStop:
		return m, svcdocker.Stop(m.ctx, m.project.File, m.project.Name, msg.ServiceName)

	case msgs.ServiceStart:
		return m, svcdocker.Start(m.ctx, m.project.File, m.project.Name, msg.ServiceName)

	case msgs.ServiceRestart:
		return m, svcdocker.Restart(m.ctx, m.project.File, m.project.Name, msg.ServiceName)

	case msgs.ServiceRebuild:
		return m, svcdocker.Rebuild(m.ctx, m.project.File, m.project.Name, msg.ServiceName)

	case msgs.ServiceActionCompleted:
		m.serviceList, _ = m.serviceList.Update(msg)

		return m, nil
	}

	m.serviceList, cmd = m.serviceList.Update(msg)
	m.panel, panCmd = m.panel.Update(msg)
	m.helpbar, helpCmd = m.helpbar.Update(msg)

	return m, tea.Batch(cmd, panCmd, helpCmd)
}

// View renders the daemon status header, service list + inspector side by side,
// and a help bar at the bottom.
func (m Model) View() tea.View {
	statusContent := m.daemon.View()

	listContent := m.serviceList.View()
	panContent := m.panel.View()

	body := lipgloss.JoinHorizontal(lipgloss.Top, listContent, panContent)

	const helpbarHeight = 2

	bodyHeight := max(m.h-helpbarHeight, 0)
	body = lipgloss.NewStyle().Height(bodyHeight).Render(body)

	return tea.NewView(lipgloss.JoinVertical(lipgloss.Top,
		statusContent,
		body,
		m.helpbar.View(),
	))
}
