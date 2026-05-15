// Package dashboard2 implements the project dashboard flow.
package dashboard2

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
	"github.com/ma-tf/ogle/internal/ui/components/servicelist2"
	"github.com/ma-tf/ogle/internal/ui/components/servicepanel"
	"github.com/ma-tf/ogle/internal/ui/theme"
)

const (
	helpbarHeight = 2
	statusHeight  = 1
)

// Model is the dashboard flow orchestrator.
type Model struct {
	ctx         context.Context
	project     *domain.Project
	conn        *connection.Machine
	daemon      daemonstatus.Model
	serviceList servicelist2.Model
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

	contentH := max(h-statusHeight-helpbarHeight, 0)
	listW := servicelist2.ListWidth(w)
	svcList := servicelist2.New(project, th, zm, listW, contentH)

	return Model{
		ctx:         ctx,
		project:     project,
		conn:        conn,
		daemon:      daemonstatus.New(ctx, conn, th),
		serviceList: svcList,
		panel:       servicepanel.New(project, th, w-listW, contentH),
		helpbar:     helpbar.New(),
		w:           w,
		h:           h,
	}
}

// Init fires the daemon status init and sends the keymap to the helpbar.
func (m Model) Init() tea.Cmd {
	return tea.Batch(
		m.daemon.Init(),
		func() tea.Msg {
			return msgs.BindingsMsg{
				Keymap: appKeymap{
					list:    m.serviceList,
					actions: []key.Binding{keyStop, keyStart, keyRestart, keyRebuild},
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
		if m.serviceList.IsFiltering() {
			break
		}

		if key.Matches(msg, keyQuit) {
			return m, tea.Quit
		}

	case msgs.ServiceActionCompleted:
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

	contentH := max(m.h-statusHeight-helpbarHeight, 0)

	listContent := m.serviceList.View()
	panContent := lipgloss.NewStyle().Height(contentH).Render(m.panel.View())

	body := lipgloss.JoinHorizontal(lipgloss.Top, listContent, panContent)

	return tea.NewView(statusContent + "\n" + body + "\n" + m.helpbar.View())
}
