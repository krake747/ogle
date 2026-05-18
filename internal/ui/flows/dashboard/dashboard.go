// Package dashboard implements the project dashboard flow.
package dashboard

import (
	"context"
	"log/slog"
	"slices"

	"charm.land/bubbles/v2/key"
	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
	zone "github.com/lrstanley/bubblezone/v2"

	"github.com/ma-tf/ogle/internal/domain"
	"github.com/ma-tf/ogle/internal/msgs"
	svcdocker "github.com/ma-tf/ogle/internal/services/docker"
	"github.com/ma-tf/ogle/internal/services/docker/connection"
	"github.com/ma-tf/ogle/internal/services/parser"
	"github.com/ma-tf/ogle/internal/ui/components/daemonstatus"
	"github.com/ma-tf/ogle/internal/ui/components/helpbar"
	"github.com/ma-tf/ogle/internal/ui/components/servicelist"
	"github.com/ma-tf/ogle/internal/ui/components/servicepanel"
	"github.com/ma-tf/ogle/internal/ui/components/settings2"
	"github.com/ma-tf/ogle/internal/ui/theme"
)

// Model is the dashboard flow orchestrator.
type Model struct {
	ctx     context.Context
	log     *slog.Logger
	parser  parser.Parser
	project *domain.Project
	th      *theme.Theme
	zm      *zone.Manager
	conn    *connection.Machine

	daemon          daemonstatus.Model
	serviceList     servicelist.Model
	panel           servicepanel.Model
	helpbar         helpbar.Model
	settings2       settings2.Model
	showingSettings bool
	w, h            int
}

// New returns a Model in the Connecting state.
func New(
	ctx context.Context,
	project *domain.Project,
	log *slog.Logger,
	th *theme.Theme,
	zm *zone.Manager,
	w, h int,
) tea.Model {
	conn := connection.New()

	return Model{
		ctx:             ctx,
		log:             log,
		parser:          parser.New(ctx, log),
		project:         project,
		th:              th,
		zm:              zm,
		conn:            conn,
		daemon:          daemonstatus.New(ctx, conn, th),
		serviceList:     servicelist.New(project, th, zm, w),
		panel:           servicepanel.New(project, th, w, h),
		helpbar:         helpbar.New(),
		settings2:       settings2.New(th, w, h),
		showingSettings: false,
		w:               w,
		h:               h,
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
						servicelist.KeyToggleService,
						servicelist.KeyRestart,
						servicelist.KeyRebuild,
						keyToggleWrap,
					},
				},
			}
		},
	)
}

// Update handles dashboard-level messages and forwards daemon messages.
func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var daemonCmd, svcListCmd, panCmd, helpCmd, settings2Cmd tea.Cmd

	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.w = msg.Width
		m.h = msg.Height

	case msgs.StatePollTick:
		m.panel, panCmd = m.panel.Update(msg)

		return m, tea.Batch(
			svcdocker.Ps(m.ctx, m.project.File, m.project.Name),
			panCmd,
		)

	case tea.KeyPressMsg:
		if !m.showingSettings && key.Matches(msg, keyQuit) {
			return m, tea.Quit
		}

		if !m.showingSettings && key.Matches(msg, keySettings) {
			m.showingSettings = true

			return m, nil
		}

		if key.Matches(msg, keyToggleWrap) && !m.showingSettings {
			return m, func() tea.Msg {
				return msgs.ToggleLogWrap{}
			}
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

	case msgs.FileAvailabilityChanged:
		if !slices.Contains(msg.Files, m.project.File) {
			return m, func() tea.Msg {
				return msgs.FileRemoved{File: m.project.File}
			}
		}

		p, err := m.parser.Parse(m.project.File)
		if err != nil {
			m.log.WarnContext(
				m.ctx,
				"dashboard: re-parse failed, keeping current state",
				slog.Any("err", err),
			)

			return m, nil
		}

		newDash := New(m.ctx, p, m.log, m.th, m.zm, m.w, m.h)

		return newDash, newDash.Init()

	case msgs.SettingsVisibilityChanged:
		m.showingSettings = msg.Visible

		return m, nil
	}

	m.daemon, daemonCmd = m.daemon.Update(msg)
	m.serviceList, svcListCmd = m.serviceList.Update(msg)
	m.panel, panCmd = m.panel.Update(msg)
	m.helpbar, helpCmd = m.helpbar.Update(msg)

	if m.showingSettings {
		m.settings2, settings2Cmd = m.settings2.Update(msg)
	}

	return m, tea.Batch(
		daemonCmd,
		svcListCmd,
		panCmd,
		helpCmd,
		settings2Cmd,
	)
}

// View renders the daemon status header, service list + inspector side by side,
// and a help bar at the bottom. When settings2 is visible it renders as an
// overlay on top of the normal dashboard.
func (m Model) View() tea.View {
	statusContent := m.daemon.View().Content

	listContent := m.serviceList.View().Content
	panContent := m.panel.View().Content

	body := lipgloss.JoinHorizontal(lipgloss.Top, listContent, panContent)

	const helpbarHeight = 2

	bodyHeight := max(m.h-helpbarHeight, 0)
	body = lipgloss.NewStyle().Height(bodyHeight).Render(body)

	content := lipgloss.JoinVertical(lipgloss.Top,
		statusContent,
		body,
		m.helpbar.View().Content,
	)

	if m.showingSettings {
		overContent := m.settings2.View().Content
		overW := lipgloss.Width(overContent)
		overH := lipgloss.Height(overContent)
		overX := max((m.w-overW)/2, 0) //nolint:mnd // halving to centre overlay
		overY := max((m.h-overH)/2, 0) //nolint:mnd // halving to centre overlay

		return tea.NewView(lipgloss.NewCompositor(
			lipgloss.NewLayer(content).X(0).Y(0).Z(0),
			lipgloss.NewLayer(overContent).X(overX).Y(overY).Z(1),
		).Render())
	}

	return tea.NewView(content)
}
