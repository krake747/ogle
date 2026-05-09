// Package dashboard implements the root flow orchestrator. It owns the watcher
// lifecycle (creation, subscription, retry, reconnect) and drives the top-level
// flow transitions: startup → project on msgs.ProjectLoaded.
//
// dashboard.Model implements flows.State so that app.go can remain a minimal
// tea.Model adapter shim with no domain knowledge.
package dashboard

import (
	"fmt"
	"log/slog"
	"os"
	"path/filepath"

	tea "charm.land/bubbletea/v2"

	"github.com/ma-tf/ogle/config"
	"github.com/ma-tf/ogle/internal/compose"
	"github.com/ma-tf/ogle/internal/msgs"
	"github.com/ma-tf/ogle/internal/ui/flows"
	"github.com/ma-tf/ogle/internal/ui/flows/dashboard/project"
	"github.com/ma-tf/ogle/internal/ui/flows/startup"
	"github.com/ma-tf/ogle/internal/watcher"
)

// watcherReadyMsg is dashboard-internal. It is delivered when a watcher retry
// succeeds, carrying the newly-created Watcher.
type watcherReadyMsg struct{ w *watcher.Watcher }

// Model is the root flow orchestrator. It implements flows.State.
type Model struct {
	cfg     config.Config
	dir     string
	logger  *slog.Logger
	w       *watcher.Watcher // nil if initialisation failed
	current flows.State
}

// New constructs the dashboard Model. Watcher creation is synchronous; a
// failure is surfaced to the startup flow as a WatcherError so the watching
// view enters its error state with a retry keybinding.
func New(cfg config.Config, logger *slog.Logger) Model {
	dir := watchDir(cfg)
	w, watcherErr := watcher.New(dir, logger)

	return Model{
		cfg:     cfg,
		dir:     dir,
		logger:  logger,
		w:       w, // nil on failure
		current: startup.New(cfg, dir, watcherErr),
	}
}

// Init fires the first batch of commands:
//   - current.Init() kicks off either a scan or an immediate parse (-f case).
//   - If the watcher is healthy, w.Next() begins the subscription loop.
func (m Model) Init() tea.Cmd {
	cmds := []tea.Cmd{m.current.Init()}

	if m.w != nil {
		cmds = append(cmds, m.w.Next())
	}

	return tea.Batch(cmds...)
}

// Update drives the root state machine.
//
// Intercepted centrally:
//   - tea.KeyPressMsg "ctrl+c" → quit
//   - msgs.FileAvailabilityChanged → re-subscribe watcher, forward to current
//   - msgs.ProjectLoaded → transition current to project.New
//   - msgs.RetryWatcher → fire retryWatcherCmd
//   - watcherReadyMsg → store new watcher, resume subscription + scan
//
// Everything else (msgs.WatcherError, all input, window events) is forwarded
// to the active state. WatcherError routing: startup.Model handles it during
// the startup phase; project.Model handles it during the project phase. Each
// flow owns its own error semantics.
func (m Model) Update(msg tea.Msg) (flows.State, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyPressMsg:
		if msg.String() == "ctrl+c" {
			return m, tea.Quit
		}

	case msgs.FileAvailabilityChanged:
		// Re-subscribe before forwarding so the next snapshot is not missed.
		var watchCmd tea.Cmd
		if m.w != nil {
			watchCmd = m.w.Next()
		}

		next, subCmd := m.current.Update(msg)
		m.current = next

		return m, tea.Batch(watchCmd, subCmd)

	case msgs.ProjectLoaded:
		m.current = project.New(msg.Project)

		return m, m.current.Init()

	case msgs.RetryWatcher:
		return m, retryWatcherCmd(m.dir, m.logger)

	case watcherReadyMsg:
		m.w = msg.w

		return m, tea.Batch(
			m.w.Next(),
			scanAndNotify(m.w.Dir()),
		)
	}

	// Forward all remaining messages to the active state.
	next, cmd := m.current.Update(msg)
	m.current = next

	return m, cmd
}

// View delegates rendering to the active state.
func (m Model) View() string { return m.current.View() }

// ---- helpers ---------------------------------------------------------------

// watchDir returns the directory that should be monitored.
func watchDir(cfg config.Config) string {
	if cfg.ProjectFile != "" {
		return filepath.Dir(cfg.ProjectFile)
	}

	dir, err := os.Getwd()
	if err != nil {
		// Getwd failing means something is deeply wrong with the process
		// environment. Fall back to "." so the program can still start and
		// surface a useful error via the watcher.
		return "."
	}

	return dir
}

// retryWatcherCmd attempts to create a new Watcher for dir. It returns either
// a watcherReadyMsg (success) or a msgs.WatcherError (failure).
func retryWatcherCmd(dir string, logger *slog.Logger) tea.Cmd {
	return func() tea.Msg {
		w, err := watcher.New(dir, logger)
		if err != nil {
			return msgs.WatcherError{Err: fmt.Errorf("retry failed: %w", err)}
		}

		return watcherReadyMsg{w: w}
	}
}

// scanAndNotify runs a fresh ScanAll sweep and delivers the result as a
// msgs.FileAvailabilityChanged. Used after a successful watcher retry so the
// startup flow can evaluate the current state of the directory.
func scanAndNotify(dir string) tea.Cmd {
	return func() tea.Msg {
		return msgs.FileAvailabilityChanged{
			Files: compose.ScanAll(dir),
		}
	}
}
