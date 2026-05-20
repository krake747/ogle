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

	"github.com/ma-tf/ogle/config"
	"github.com/ma-tf/ogle/internal/domain"
	"github.com/ma-tf/ogle/internal/msgs"
	svcdocker "github.com/ma-tf/ogle/internal/services/docker"
	"github.com/ma-tf/ogle/internal/services/parser"
	"github.com/ma-tf/ogle/internal/ui/components/carousel"
	"github.com/ma-tf/ogle/internal/ui/components/servicelist"
	"github.com/ma-tf/ogle/internal/ui/components/servicepanel"
	"github.com/ma-tf/ogle/internal/ui/components/settings"
	"github.com/ma-tf/ogle/internal/ui/theme"
)

// frameHeight is the number of terminal lines consumed by the app-level chrome
// (topbar + helpbar) that each phase must subtract from its allocated height.
const frameHeight = 3

// Model is the dashboard flow orchestrator.
type Model struct {
	ctx       context.Context
	log       *slog.Logger
	parser    parser.Parser
	project   *domain.Project
	th        *theme.Theme
	zm        *zone.Manager
	configDir string

	serviceList     servicelist.Model
	carousel        carousel.Model
	panel           servicepanel.Model
	settings        settings.Model
	showingSettings bool
	cfg             config.Config
	w, h            int
}

// New returns a Model.
func New(
	ctx context.Context,
	project *domain.Project,
	log *slog.Logger,
	th *theme.Theme,
	cfg config.Config,
	zm *zone.Manager,
	configDir string,
	w, h int,
) tea.Model {
	return Model{
		ctx:             ctx,
		log:             log,
		parser:          parser.New(ctx, log),
		project:         project,
		th:              th,
		zm:              zm,
		configDir:       configDir,
		serviceList:     servicelist.New(project, th, zm, w),
		carousel:        carousel.New(project, w, h, th),
		panel:           servicepanel.New(project, th, w, h, cfg.LogBufferCap),
		settings:        settings.New(th, cfg, w, h),
		showingSettings: false,
		cfg:             cfg,
		w:               w,
		h:               h,
	}
}

// Init fires sub-model initialisation and sends the keymap to the helpbar.
func (m Model) Init() tea.Cmd {
	return tea.Batch(
		m.serviceList.Init(),
		m.panel.Init(),
		func() tea.Msg {
			return msgs.BindingsMsg{
				Keymap: appKeymap{
					list: m.serviceList,
					actions: []key.Binding{
						servicelist.KeyPrev,
						servicelist.KeyNext,
						servicelist.KeyToggleService,
						servicelist.KeyRestart,
						servicelist.KeyRebuild,
						keyScrollUp,
						keyScrollDown,
						keyScrollLeft,
						keyScrollRight,
						keyToggleWrap,
					},
				},
			}
		},
	)
}

// Update handles dashboard-level messages.
func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var svcListCmd, carouselCmd, panCmd, settingsCmd tea.Cmd

	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.w = msg.Width
		m.h = msg.Height - frameHeight

	case msgs.StatePollTick:
		m.panel, panCmd = m.panel.Update(msg)

		return m, tea.Batch(
			svcdocker.Ps(m.ctx, m.project.File, m.project.Name),
			panCmd,
		)

	case tea.MouseClickMsg, tea.MouseMotionMsg, tea.MouseReleaseMsg, tea.MouseWheelMsg:
		if m.showingSettings {
			return m, nil
		}

	case tea.KeyPressMsg:
		if m.showingSettings {
			m.settings, settingsCmd = m.settings.Update(msg)

			return m, settingsCmd
		}

		switch {
		case key.Matches(msg, keyQuit):
			return m, tea.Quit

		case key.Matches(msg, keySettings):
			m.showingSettings = true

			return m, nil

		case key.Matches(msg, keyToggleWrap):
			return m, func() tea.Msg { return msgs.ToggleLogWrap{} }

		case key.Matches(msg, keyScrollUp), key.Matches(msg, keyScrollDown),
			key.Matches(msg, keyScrollLeft), key.Matches(msg, keyScrollRight):
			m.panel, panCmd = m.panel.Update(msg)

			return m, panCmd
		}

	case msgs.ServiceStop,
		msgs.ServiceStart,
		msgs.ServiceRestart,
		msgs.ServiceRebuild,
		msgs.ServiceActionCompleted:
		return m.handleServiceAction(msg)

	case msgs.FileAvailabilityChanged:
		return m.handleFileAvailabilityChanged(msg.Files)

	case msgs.SettingsApplied:
		if th, err := theme.Load(msg.Theme, m.configDir); err == nil {
			m.th = th
		}

		m.cfg.Theme = msg.Theme
		m.cfg.LogBufferCap = msg.LogBufferCap

		return m, nil

	case msgs.ThemeChanged:
		m.th = msg.Theme

	case msgs.SettingsVisibilityChanged:
		m.showingSettings = msg.Visible

		return m, nil
	}

	m.serviceList, svcListCmd = m.serviceList.Update(msg)
	m.carousel, carouselCmd = m.carousel.Update(msg)
	m.panel, panCmd = m.panel.Update(msg)

	if m.showingSettings {
		m.settings, settingsCmd = m.settings.Update(msg)
	}

	return m, tea.Batch(svcListCmd, carouselCmd, panCmd, settingsCmd)
}

func (m Model) handleServiceAction(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
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

		if msg.Err != nil {
			return m, func() tea.Msg {
				return msgs.DisplayError{Err: msg.Err.Error()}
			}
		}

		return m, func() tea.Msg {
			return msgs.ClearStatusMsg{}
		}
	}

	return m, nil
}

func (m Model) handleFileAvailabilityChanged(files []string) (tea.Model, tea.Cmd) {
	if !slices.Contains(files, m.project.File) {
		return m, func() tea.Msg {
			return msgs.FileRemoved{File: m.project.File}
		}
	}

	p, err := m.parser.Parse(m.project.File)
	if err != nil {
		m.log.WarnContext(m.ctx,
			"dashboard: re-parse failed, keeping current state",
			slog.Any("err", err),
		)

		return m, nil
	}

	newDash := New(m.ctx, p, m.log, m.th, m.cfg, m.zm, m.configDir, m.w, m.h)

	return newDash, newDash.Init()
}

// View renders the service list and inspector side by side. When settings is
// visible it renders as an overlay on top of the normal dashboard.
func (m Model) View() tea.View {
	listContent := lipgloss.JoinVertical(lipgloss.Top,
		m.serviceList.View().Content,
		m.carousel.View().Content,
	)
	listH := lipgloss.Height(listContent)
	listW := lipgloss.Width(listContent)

	if listH < m.h {
		filler := lipgloss.NewStyle().
			Width(listW).
			Height(m.h - listH).
			Background(m.th.ServiceListBackground).
			Render("")
		listContent = lipgloss.JoinVertical(lipgloss.Top, listContent, filler)
	}

	panContent := m.panel.View().Content

	body := lipgloss.JoinHorizontal(lipgloss.Top, listContent, panContent)

	if m.showingSettings {
		overContent := m.settings.View().Content
		overW := lipgloss.Width(overContent)
		overH := lipgloss.Height(overContent)
		overX := max((m.w-overW)/2, 0) //nolint:mnd // halving to centre overlay
		overY := max((m.h-overH)/2, 0) //nolint:mnd // halving to centre overlay

		return tea.NewView(lipgloss.NewCompositor(
			lipgloss.NewLayer(body).X(0).Y(0).Z(0),
			lipgloss.NewLayer(overContent).X(overX).Y(overY).Z(1),
		).Render())
	}

	return tea.NewView(body)
}
