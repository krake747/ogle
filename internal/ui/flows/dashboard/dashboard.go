// Package dashboard implements the root flow orchestrator. It owns the watcher
// lifecycle (creation, subscription, retry, reconnect) and drives the top-level
// flow transitions: startup → project on msgs.ProjectLoaded.
package dashboard

import (
	"fmt"
	"log/slog"
	"os"
	"path/filepath"

	tea "charm.land/bubbletea/v2"
	"github.com/charmbracelet/x/term"

	"github.com/ma-tf/ogle/config"
	"github.com/ma-tf/ogle/internal/msgs"
	"github.com/ma-tf/ogle/internal/services/parser"
	"github.com/ma-tf/ogle/internal/services/scanner"
	svcwatcher "github.com/ma-tf/ogle/internal/services/watcher"
	"github.com/ma-tf/ogle/internal/ui/flows/dashboard/project"
	"github.com/ma-tf/ogle/internal/ui/flows/startup"
)

// watcherReadyMsg is delivered when a watcher retry succeeds, carrying the
// newly-created Watcher.
type watcherReadyMsg struct{ w svcwatcher.Watcher }

// Model is the root flow orchestrator.
type Model struct {
	cfg     config.Config
	dir     string
	logger  *slog.Logger
	scanner scanner.Scanner
	parser  parser.Parser
	w       svcwatcher.Watcher
	current tea.Model
}

// New constructs the dashboard Model. Watcher creation is synchronous; a
// failure is surfaced to the startup flow as a WatcherError so the watching
// view enters its error state with a retry keybinding.
func New(cfg config.Config, logger *slog.Logger, sc scanner.Scanner, p parser.Parser) Model {
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

	return Model{
		cfg:     cfg,
		dir:     dir,
		logger:  logger,
		scanner: sc,
		parser:  p,
		w:       w,
		current: startup.New(cfg, dir, watcherErr, sc, p, width, height),
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
// unhandled to the active state — startup.Model and project.Model each own
// their own error semantics.
func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyPressMsg:
		if msg.String() == "ctrl+c" {
			return m, tea.Quit
		}

	case msgs.FileAvailabilityChanged:
		// Re-subscribe before forwarding so the next snapshot is not missed.
		watchCmd := m.w.Next()

		next, subCmd := m.current.Update(msg)
		m.current = next

		return m, tea.Batch(watchCmd, subCmd)

	case msgs.ProjectLoaded:
		m.current = project.New(msg.Project)

		return m, m.current.Init()

	case msgs.RetryWatcher:
		return m, retryWatcherCmd(m.dir, m.scanner, m.logger)

	case watcherReadyMsg:
		m.w = msg.w

		return m, tea.Batch(
			m.w.Next(),
			m.w.Snapshot(),
		)
	}

	next, cmd := m.current.Update(msg)
	m.current = next

	return m, cmd
}

// View delegates rendering to the active state.
func (m Model) View() tea.View {
	v := m.current.View()
	v.AltScreen = true

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
