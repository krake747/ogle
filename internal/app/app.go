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
	"github.com/ma-tf/ogle/internal/services/scanner"
	svcwatcher "github.com/ma-tf/ogle/internal/services/watcher"
	"github.com/ma-tf/ogle/internal/ui/flows/dashboard2"
	"github.com/ma-tf/ogle/internal/ui/flows/startup"
	"github.com/ma-tf/ogle/internal/ui/theme"
)

// watcherReadyMsg is delivered when a watcher retry succeeds, carrying the
// newly-created Watcher.
type watcherReadyMsg struct{ w svcwatcher.Watcher }

// Model is the root flow orchestrator.
type Model struct {
	ctx       context.Context
	cfg       config.Config
	configDir string
	dir       string
	logger    *slog.Logger
	scanner   scanner.Scanner
	parser    parser.Parser
	theme     *theme.Theme
	zm        *zone.Manager
	w         svcwatcher.Watcher
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
	logger *slog.Logger,
	sc scanner.Scanner,
	p parser.Parser,
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

	w, watcherErr := svcwatcher.New(dir, sc, logger)

	width, height, err := term.GetSize(os.Stdout.Fd())
	if err != nil {
		width, height = 0, 0
	}

	zm := zone.New()

	return Model{
		ctx:       ctx,
		cfg:       cfg,
		configDir: configDir,
		dir:       dir,
		logger:    logger,
		scanner:   sc,
		parser:    p,
		theme:     th,
		zm:        zm,
		w:         w,
		current:   startup.New(cfg, dir, watcherErr, sc, p, th, zm, width, height),
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
		m.w.Next(),
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

	case msgs.FileAvailabilityChanged:
		// Re-subscribe before forwarding so the next snapshot is not missed.
		watchCmd := m.w.Next()

		next, subCmd := m.current.Update(msg)
		m.current = next

		return m, tea.Batch(watchCmd, subCmd)

	case msgs.ProjectLoaded:
		m.current = dashboard2.New(
			m.ctx,
			msg.Project,
			m.theme,
			m.zm,
			m.width,
			m.height,
			m.cfg.PollInterval,
		)

		return m, m.current.Init()

	case msgs.SettingsApplied:
		// m.cfg and m.theme are updated so that the next ProjectLoaded event
		// (triggered by a compose-file change) constructs the dashboard model
		// with the new settings. The live UI update is already handled by
		// Settings.Update returning a new Screen state directly.
		th, err := theme.Load(msg.Theme, m.configDir)
		if err != nil {
			m.logger.Warn("settings: theme load failed, keeping previous", slog.Any("err", err))
		} else {
			m.theme = th
		}

		m.cfg.Theme = msg.Theme
		m.cfg.PollInterval = msg.PollInterval
		m.cfg.LogBufferCap = msg.LogBufferCap

		return m, nil

	case msgs.RetryWatcher:
		return m, retryWatcherCmd(m.dir, m.scanner, m.logger)

	case watcherReadyMsg:
		m.w = msg.w

		return m, tea.Batch(
			m.w.Next(),
			m.w.Snapshot(),
		)

	case profiling.ProfilesDumped:
		if msg.Err != nil {
			m.logger.Error("profiling dump failed", slog.Any("err", msg.Err))
		} else {
			m.logger.Info("profiling dump written",
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

// Close releases the watcher and unblocks any goroutine blocked in w.Next().
// Call after the Bubble Tea program returns.
func (m Model) Close() error {
	if err := m.w.Close(); err != nil {
		return fmt.Errorf("close watcher: %w", err)
	}

	return nil
}

func retryWatcherCmd(
	dir string,
	sc scanner.Scanner,
	logger *slog.Logger,
) tea.Cmd {
	return func() tea.Msg {
		w, err := svcwatcher.New(dir, sc, logger)
		if err != nil {
			// w is a NullWatcher on failure; close it to release the done channel.
			_ = w.Close()

			return msgs.WatcherError{Err: fmt.Errorf("retry failed: %w", err)}
		}

		return watcherReadyMsg{w: w}
	}
}
