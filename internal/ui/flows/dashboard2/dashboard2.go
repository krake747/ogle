// Package dashboard2 implements the project dashboard flow.
package dashboard2

import (
	"context"
	"time"

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
	"github.com/ma-tf/ogle/internal/ui/components/inspector2"
	"github.com/ma-tf/ogle/internal/ui/components/servicelist2"
	"github.com/ma-tf/ogle/internal/ui/theme"
)

const (
	helpbarHeight = 2
	statusHeight  = 1
	listRatio     = 30 // percent
	listMaxWidth  = 80
	pctDivisor    = 100
)

// statePollMsg is delivered by a tick to trigger a docker compose ps poll.
type statePollMsg struct{}

// Model is the dashboard flow orchestrator.
type Model struct {
	ctx           context.Context
	project       *domain.Project
	conn          *connection.Machine
	daemon        daemonstatus.Model
	serviceList   servicelist2.Model
	inspector     inspector2.Model
	helpbar       helpbar.Model
	w, h          int
	pollInterval  time.Duration
	pollerStarted bool
}

// New returns a Model in the Connecting state.
func New(
	ctx context.Context,
	project *domain.Project,
	th *theme.Theme,
	zm *zone.Manager,
	w, h int,
	pollInterval time.Duration,
) tea.Model {
	conn := connection.New()

	contentH := max(h-statusHeight-helpbarHeight, 0)
	listW := listWidth(w)
	svcList := servicelist2.New(project, th, zm, listW, contentH)

	return Model{
		ctx:           ctx,
		project:       project,
		conn:          conn,
		daemon:        daemonstatus.New(ctx, conn, th),
		serviceList:   svcList,
		inspector:     inspector2.New(th).SetBounds(w-listW, contentH),
		helpbar:       helpbar.New().WithListKeys(svcList),
		w:             w,
		h:             h,
		pollInterval:  pollInterval,
		pollerStarted: false,
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
		m.helpbar = m.helpbar.SetWidth(m.w)

		contentH := max(m.h-statusHeight-helpbarHeight, 0)
		listW := listWidth(m.w)
		m.serviceList = m.serviceList.SetBounds(0, 0, listW, contentH)
		m.inspector = m.inspector.SetBounds(m.w-listW, contentH)

		return m, nil

	case msgs.DaemonMsg, spinner.TickMsg:
		m.daemon, cmd = m.daemon.Update(msg)
		if _, isConn := msg.(msgs.DaemonConnected); isConn && !m.pollerStarted {
			m.pollerStarted = true
			cmd = tea.Batch(cmd, m.pollStateCmd())
		}

		return m, cmd

	case statePollMsg:
		return m, tea.Batch(
			svcdocker.Ps(m.ctx, m.project.File, m.project.Name),
			m.pollStateCmd(),
		)

	case tea.KeyPressMsg:
		if m.serviceList.IsFiltering() {
			break
		}

		if key.Matches(msg, m.helpbar.Quit()) {
			return m, tea.Quit
		}
	}

	var inspCmd tea.Cmd

	m.serviceList, cmd = m.serviceList.Update(msg)
	m.inspector, inspCmd = m.inspector.Update(msg)

	return m, tea.Batch(cmd, inspCmd)
}

// View renders the daemon status header, service list + inspector side by side,
// and a help bar at the bottom.
func (m Model) View() tea.View {
	statusContent := m.daemon.View().Content

	contentH := max(m.h-statusHeight-helpbarHeight, 0)

	listContent := m.serviceList.View().Content
	inspContent := lipgloss.NewStyle().Height(contentH).Render(m.inspector.View())

	body := lipgloss.JoinHorizontal(lipgloss.Top, listContent, inspContent)

	return tea.NewView(statusContent + "\n" + body + "\n" + m.helpbar.View())
}

func listWidth(totalW int) int {
	w := min(totalW*listRatio/pctDivisor, listMaxWidth)

	return w
}

func (m Model) pollStateCmd() tea.Cmd {
	return tea.Tick(m.pollInterval, func(_ time.Time) tea.Msg {
		return statePollMsg{}
	})
}
