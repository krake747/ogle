// Package app implements the root flow orchestrator. It owns the watcher
// lifecycle (creation, subscription, retry, reconnect) and drives the top-level
// flow transitions: startup → dashboard on msgs.ProjectLoaded.
package app

import (
	"context"
	"log/slog"
	"os"
	"path/filepath"

	tea "charm.land/bubbletea/v2"
	"github.com/charmbracelet/x/term"
	zone "github.com/lrstanley/bubblezone/v2"

	"github.com/ma-tf/ogle/config"
	"github.com/ma-tf/ogle/internal/msgs"
	"github.com/ma-tf/ogle/internal/profiling"
	"github.com/ma-tf/ogle/internal/ui/flows/dashboard"
	"github.com/ma-tf/ogle/internal/ui/flows/startup"
	"github.com/ma-tf/ogle/internal/ui/theme"
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
	current   tea.Model
	width     int
	height    int
}

// New constructs the app Model. Watcher creation is synchronous; a
// failure is surfaced to the startup flow as a WatcherError so the watching
// view enters its error state with a retry keybinding.
func New(
	ctx context.Context,
	cfg config.Config,
	configDir string,
	log *slog.Logger,
	th *theme.Theme,
) Model {
	var dir string
	if cfg.ProjectFile != "" {
		dir = filepath.Dir(cfg.ProjectFile)
	} else {
		var err error
		if dir, err = os.Getwd(); err != nil {
			dir = "."
		}
	}

	width, height, err := term.GetSize(os.Stdout.Fd())
	if err != nil {
		width, height = 0, 0
	}

	zm := zone.New()

	s, err := startup.New(ctx, log, dir, width, height)
	if err != nil {
		log.WarnContext(
			ctx,
			"startup: watcher creation failed, continuing with degraded features",
			slog.Any("error", err),
		)
	}

	return Model{
		ctx:       ctx,
		cfg:       cfg,
		configDir: configDir,
		dir:       dir,
		log:       log,
		theme:     th,
		zm:        zm,
		current:   s,
		width:     width,
		height:    height,
	}
}

// Init fires the first batch of commands:
//   - current.Init() kicks off either a scan or an immediate parse (-f case).
//   - w.Next() begins the watcher subscription loop.
func (m Model) Init() tea.Cmd {
	return tea.Batch(
		m.current.Init(),
	)
}

// Update drives the root state machine. msgs.WatcherError is forwarded
// unhandled to the active state — startup.Model and dashboard.Model each own
// their own error semantics.
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
		m.current = dashboard.New(m.ctx, msg.Project, m.theme, m.zm, m.width, m.height)

		return m, m.current.Init()

	case msgs.SettingsApplied:
		// m.cfg and m.theme are updated so that the next ProjectLoaded event
		// (triggered by a compose-file change) constructs the dashboard model
		// with the new settings. The live UI update is already handled by
		// Settings.Update returning a new Screen state directly.
		th, err := theme.Load(msg.Theme, m.configDir)
		if err != nil {
			m.log.Warn("settings: theme load failed, keeping previous", slog.Any("err", err))
		} else {
			m.theme = th
		}

		m.cfg.Theme = msg.Theme
		m.cfg.LogBufferCap = msg.LogBufferCap

		return m, nil

	case profiling.ProfilesDumped:
		if msg.Err != nil {
			m.log.Error("profiling dump failed", slog.Any("err", msg.Err))
		} else {
			m.log.Info("profiling dump written",
				slog.String("goroutine", msg.GoroutinePath),
				slog.String("heap", msg.HeapPath),
			)
		}

		return m, nil
	}

	next, cmd := m.current.Update(msg)
	m.current = next

	return m, cmd
}

// View delegates rendering to the active state.
func (m Model) View() tea.View {
	v := m.current.View()
	v.Content = m.zm.Scan(v.Content)
	v.AltScreen = true
	v.MouseMode = tea.MouseModeCellMotion

	return v
}
