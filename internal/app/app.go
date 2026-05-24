// Package app implements the root flow orchestrator. It owns the watcher
// lifecycle (creation, subscription, retry, reconnect) and drives the top-level
// flow transitions: startup → dashboard on msgs.ProjectLoaded.
package app

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"

	"charm.land/bubbles/v2/key"
	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
	"github.com/charmbracelet/x/term"
	zone "github.com/lrstanley/bubblezone/v2"
	"go.yaml.in/yaml/v3"

	"github.com/ma-tf/ogle/config"
	"github.com/ma-tf/ogle/internal/domain"
	"github.com/ma-tf/ogle/internal/msgs"
	"github.com/ma-tf/ogle/internal/profiling"
	"github.com/ma-tf/ogle/internal/services/docker/connection"
	"github.com/ma-tf/ogle/internal/services/parser"
	"github.com/ma-tf/ogle/internal/services/watcher"
	"github.com/ma-tf/ogle/internal/ui/components/helpbar"
	"github.com/ma-tf/ogle/internal/ui/components/statusbar"
	"github.com/ma-tf/ogle/internal/ui/components/topbar"
	"github.com/ma-tf/ogle/internal/ui/components/watching"
	"github.com/ma-tf/ogle/internal/ui/flows/dashboard"
	"github.com/ma-tf/ogle/internal/ui/flows/startup"
	"github.com/ma-tf/ogle/internal/ui/layout"
	"github.com/ma-tf/ogle/internal/ui/theme"
)

//nolint:gochecknoglobals // package-level key bindings
var (
	keyQuit    = key.NewBinding(key.WithKeys("ctrl+c"))
	keyProfile = key.NewBinding(key.WithKeys("ctrl+p"))
)

type phase int

const (
	phaseStartup phase = iota
	phaseDashboard
	phaseWatching
)

// Model is the root flow orchestrator.
type Model struct {
	ctx         context.Context
	cfg         config.Config
	configPath  string
	projectFile string
	log         *slog.Logger
	theme       *theme.Theme
	zm          *zone.Manager
	watcher     watcher.Watcher

	topbar    topbar.Model
	helpbar   helpbar.Model
	statusbar statusbar.Model
	startup   tea.Model
	dashboard tea.Model
	watching  tea.Model
	phase     phase
	width     int
	height    int
}

// New constructs the app Model. Watcher creation is synchronous; if it
// fails the entire program exits with an error.
func New(
	ctx context.Context,
	cfg config.Config,
	configPath string,
	projectFile string,
	log *slog.Logger,
	th *theme.Theme,
) (Model, func() error, error) {
	width, height, errSize := term.GetSize(os.Stdout.Fd())
	if errSize != nil {
		width, height = 0, 0
	}

	watchDir, err := filepath.Abs(filepath.Dir(projectFile))
	if err != nil {
		return Model{}, nil, fmt.Errorf("resolve watch directory: %w", err)
	}

	wtr, errWatch := watcher.New(watchDir, log, projectFile)
	if errWatch != nil {
		return Model{}, nil, fmt.Errorf("create watcher: %w", errWatch)
	}

	var (
		project *domain.Project
		dash    tea.Model
	)

	currentPhase := phaseStartup
	zm := zone.New()
	pf := ""

	if projectFile != "" {
		var errParse error
		if project, errParse = parser.New(ctx, log).Parse(projectFile); errParse != nil {
			_ = wtr.Close()

			return Model{}, nil, fmt.Errorf("parse project file: %w", errParse)
		}

		currentPhase = phaseDashboard
		pf = filepath.Base(projectFile)

		dash = dashboard.New(
			ctx,
			project,
			log,
			th,
			cfg,
			zm,
			filepath.Dir(configPath),
			width,
			height,
		)
	}

	return Model{
		ctx:         ctx,
		cfg:         cfg,
		configPath:  configPath,
		projectFile: pf,
		log:         log,
		theme:       th,
		zm:          zm,
		watcher:     wtr,
		topbar:      topbar.New(ctx, connection.New(), th),
		helpbar:     helpbar.New(th),
		statusbar:   statusbar.New(th),
		startup:     startup.New(ctx, log, width, height, zm, th),
		dashboard:   dash,
		watching:    nil,
		phase:       currentPhase,
		width:       width,
		height:      height,
	}, wtr.Close, nil
}

// Init fires the initial snapshot and starts the active phase.
func (m Model) Init() tea.Cmd {
	cmds := []tea.Cmd{m.watcher.Snapshot(), m.topbar.Init(), m.helpbar.Init()}

	switch m.phase {
	case phaseDashboard:
		cmds = append(cmds, m.dashboard.Init())
		cmds = append(cmds, func() tea.Msg {
			return msgs.TopbarContext{Phase: "dashboard", File: m.projectFile}
		})
	case phaseStartup:
		cmds = append(cmds, m.startup.Init())
	case phaseWatching:
	}

	return tea.Batch(cmds...)
}

// Update drives the root state machine. Messages are either handled by app
// directly or dispatched to the active phase model.
func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var topbarCmd, helpbarCmd, statusbarCmd tea.Cmd

	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height

	case tea.KeyPressMsg:
		switch {
		case key.Matches(msg, keyQuit):
			return m, tea.Quit
		case key.Matches(msg, keyProfile):
			return m, profiling.DumpCmd()
		}

	case msgs.ProjectLoaded:
		m.dashboard = dashboard.New(
			m.ctx,
			msg.Project,
			m.log,
			m.theme,
			m.cfg,
			m.zm,
			filepath.Dir(m.configPath),
			m.width,
			m.height,
		)
		m.phase = phaseDashboard

		return m, tea.Batch(
			m.dashboard.Init(),
			func() tea.Msg {
				return msgs.TopbarContext{Phase: "dashboard", File: filepath.Base(msg.Project.File)}
			},
		)

	case msgs.SettingsApplied:
		return m.handleSettingsApplied(msg)

	case theme.Changed:
		m.theme = msg.Theme

	case profiling.ProfilesDumped:
		if msg.Err != nil {
			m.log.ErrorContext(m.ctx,
				"profiling dump failed",
				slog.Any("err", msg.Err),
			)
		} else {
			m.log.InfoContext(m.ctx,
				"profiling dump written",
				slog.String("goroutine", msg.GoroutinePath),
				slog.String("heap", msg.HeapPath),
			)
		}

		return m, nil

	case msgs.FileAvailabilityChanged:
		return m.handleFileAvailabilityChanged(msg)

	case msgs.FileRemoved:
		m.watching = watching.New(m.ctx, m.log, msg.File, m.width, m.height, m.theme)
		m.phase = phaseWatching

		return m, tea.Batch(
			func() tea.Msg { return msgs.TopbarContext{Phase: "watching", File: ""} },
			func() tea.Msg { return msgs.BindingsMsg{Keymap: watchingKeymap{}} },
		)

	case msgs.DisplayError,
		msgs.DisplayStatus,
		msgs.ClearStatusMsg:
		m.statusbar, statusbarCmd = m.statusbar.Update(msg)

		return m, statusbarCmd
	}

	m.topbar, topbarCmd = m.topbar.Update(msg)
	m.helpbar, helpbarCmd = m.helpbar.Update(msg)
	m.statusbar, statusbarCmd = m.statusbar.Update(msg)

	var cmd tea.Cmd

	switch m.phase {
	case phaseStartup:
		m.startup, cmd = m.startup.Update(msg)
	case phaseDashboard:
		m.dashboard, cmd = m.dashboard.Update(msg)
	case phaseWatching:
		m.watching, cmd = m.watching.Update(msg)
	}

	return m, tea.Batch(cmd, topbarCmd, helpbarCmd, statusbarCmd)
}

func (m Model) handleSettingsApplied(msg msgs.SettingsApplied) (tea.Model, tea.Cmd) {
	th, err := theme.Load(msg.Theme, filepath.Dir(m.configPath))
	if err != nil {
		m.log.WarnContext(
			m.ctx,
			"settings: theme load failed, keeping previous",
			slog.Any("err", err),
		)
	} else {
		m.theme = th
	}

	m.cfg.Theme = msg.Theme
	m.cfg.LogBufferCap = msg.LogBufferCap

	data, marshalErr := yaml.Marshal(&m.cfg)
	if marshalErr != nil {
		m.log.WarnContext(
			m.ctx,
			"settings: failed to marshal config",
			slog.Any("err", marshalErr),
		)
	} else if writeErr := os.WriteFile(m.configPath, data, 0o600); writeErr != nil {
		m.log.WarnContext(
			m.ctx,
			"settings: failed to write config file",
			slog.Any("err", writeErr),
		)
	}

	return m, func() tea.Msg { return theme.Changed{Theme: m.theme} }
}

func (m Model) handleFileAvailabilityChanged(msg msgs.FileAvailabilityChanged) (Model, tea.Cmd) {
	var cmd tea.Cmd

	switch m.phase {
	case phaseStartup:
		m.startup, cmd = m.startup.Update(msg)
	case phaseDashboard:
		m.dashboard, cmd = m.dashboard.Update(msg)
	case phaseWatching:
		m.watching, cmd = m.watching.Update(msg)
	}

	return m, tea.Batch(cmd, m.watcher.Next())
}

// View composes the top bar, active phase body, status bar, and help bar into a unified frame.
func (m Model) View() tea.View {
	var body tea.View

	switch m.phase {
	case phaseStartup:
		body = m.startup.View()
	case phaseDashboard:
		body = m.dashboard.View()
	case phaseWatching:
		body = m.watching.View()
	}

	bodyH := max(0, m.height-layout.FrameHeight)

	parts := []string{
		m.topbar.View().Content,
		lipgloss.NewStyle().
			Width(m.width).
			Height(bodyH).
			Background(m.theme.BodyBackground).
			Render(body.Content),
	}

	statusView := m.statusbar.View()
	if statusView.Content == "" {
		parts = append(parts, m.helpbar.View().Content)
	} else {
		parts = append(parts, statusView.Content)
	}

	frame := lipgloss.JoinVertical(lipgloss.Top, parts...)

	v := tea.NewView(frame)
	v.Content = m.zm.Scan(v.Content)
	v.AltScreen = true
	v.MouseMode = tea.MouseModeAllMotion

	return v
}
