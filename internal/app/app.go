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

	tea "charm.land/bubbletea/v2"
	"github.com/charmbracelet/x/term"
	zone "github.com/lrstanley/bubblezone/v2"

	"github.com/ma-tf/ogle/config"
	"github.com/ma-tf/ogle/internal/msgs"
	"github.com/ma-tf/ogle/internal/profiling"
	"github.com/ma-tf/ogle/internal/services/parser"
	"github.com/ma-tf/ogle/internal/services/watcher"
	"github.com/ma-tf/ogle/internal/ui/components/watching"
	"github.com/ma-tf/ogle/internal/ui/flows/dashboard"
	"github.com/ma-tf/ogle/internal/ui/flows/startup"
	"github.com/ma-tf/ogle/internal/ui/theme"
)

type phase int

const (
	phaseStartup phase = iota
	phaseDashboard
	phaseWatching
)

// Model is the root flow orchestrator.
type Model struct {
	ctx       context.Context
	cfg       config.Config
	configDir string
	dir       string
	log       *slog.Logger
	theme     *theme.Theme
	zm        *zone.Manager
	watcher   watcher.Watcher
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
	configDir string,
	projectFile string,
	log *slog.Logger,
	th *theme.Theme,
) (Model, func() error, error) {
	width, height, err := term.GetSize(os.Stdout.Fd())
	if err != nil {
		width, height = 0, 0
	}

	zm := zone.New()

	if projectFile != "" {
		dir := filepath.Dir(projectFile)

		var wtr watcher.Watcher

		wtr, err = watcher.New(dir, log, filepath.Base(projectFile))
		if err != nil {
			return Model{}, nil, fmt.Errorf("create watcher: %w", err)
		}

		p, parseErr := parser.New(ctx, log).Parse(projectFile)
		if parseErr != nil {
			_ = wtr.Close()

			return Model{}, nil, fmt.Errorf("parse project file: %w", parseErr)
		}

		dash := dashboard.New(ctx, p, log, th, zm, width, height)

		return Model{
			ctx:       ctx,
			cfg:       cfg,
			configDir: configDir,
			dir:       dir,
			log:       log,
			theme:     th,
			zm:        zm,
			watcher:   wtr,
			startup:   nil,
			dashboard: dash,
			watching:  nil,
			phase:     phaseDashboard,
			width:     width,
			height:    height,
		}, wtr.Close, nil
	}

	var dir string

	if dir, err = os.Getwd(); err != nil {
		dir = "."
	}

	var wtr watcher.Watcher

	wtr, err = watcher.New(dir, log)
	if err != nil {
		return Model{}, nil, fmt.Errorf("create watcher: %w", err)
	}

	return Model{
		ctx:       ctx,
		cfg:       cfg,
		configDir: configDir,
		dir:       dir,
		log:       log,
		theme:     th,
		zm:        zm,
		watcher:   wtr,
		startup:   startup.New(ctx, log, width, height),
		dashboard: nil,
		watching:  nil,
		phase:     phaseStartup,
		width:     width,
		height:    height,
	}, wtr.Close, nil
}

// Init fires the initial snapshot and starts the active phase.
func (m Model) Init() tea.Cmd {
	switch m.phase {
	case phaseDashboard:
		return tea.Batch(m.watcher.Snapshot(), m.dashboard.Init())
	case phaseStartup:
		return tea.Batch(m.watcher.Snapshot(), m.startup.Init())
	case phaseWatching:
		return tea.Batch(m.watcher.Snapshot())
	}

	return nil
}

// Update drives the root state machine. Messages are either handled by app
// directly or dispatched to the active phase model.
func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height

	case tea.KeyPressMsg:
		switch msg.String() {
		case "ctrl+c":
			return m, tea.Quit
		case "ctrl+p":
			return m, profiling.DumpCmd()
		}

	case msgs.ProjectLoaded:
		m.dashboard = dashboard.New(m.ctx, msg.Project, m.log, m.theme, m.zm, m.width, m.height)
		m.phase = phaseDashboard

		return m, m.dashboard.Init()

	case msgs.SettingsApplied:
		th, err := theme.Load(msg.Theme, m.configDir)
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

		return m, nil

	case profiling.ProfilesDumped:
		if msg.Err != nil {
			m.log.ErrorContext(
				m.ctx,
				"profiling dump failed",
				slog.Any("err", msg.Err),
			)
		} else {
			m.log.InfoContext(
				m.ctx,
				"profiling dump written",
				slog.String("goroutine", msg.GoroutinePath),
				slog.String("heap", msg.HeapPath),
			)
		}

		return m, nil

	case msgs.FileAvailabilityChanged:
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

	case msgs.FileRemoved:
		m.watching = watching.New(m.ctx, m.log, msg.File, m.width, m.height)
		m.phase = phaseWatching

		return m, nil
	}

	var cmd tea.Cmd

	switch m.phase {
	case phaseStartup:
		m.startup, cmd = m.startup.Update(msg)
	case phaseDashboard:
		m.dashboard, cmd = m.dashboard.Update(msg)
	case phaseWatching:
		m.watching, cmd = m.watching.Update(msg)
	}

	return m, cmd
}

// View delegates rendering to the active phase model.
func (m Model) View() tea.View {
	var v tea.View

	switch m.phase {
	case phaseStartup:
		v = m.startup.View()
	case phaseDashboard:
		v = m.dashboard.View()
	case phaseWatching:
		v = m.watching.View()
	}

	v.Content = m.zm.Scan(v.Content)
	v.AltScreen = true
	v.MouseMode = tea.MouseModeCellMotion

	return v
}
