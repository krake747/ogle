// Package startup implements the startup flow: the orchestration state machine
// that runs from app launch until a compose project is successfully loaded.
package startup

import (
	tea "charm.land/bubbletea/v2"

	"github.com/ma-tf/ogle/config"
	"github.com/ma-tf/ogle/internal/msgs"
	"github.com/ma-tf/ogle/internal/services/parser"
	"github.com/ma-tf/ogle/internal/services/scanner"
	"github.com/ma-tf/ogle/internal/ui/flows/startup/states"
)

// Model is the startup flow orchestrator.
type Model struct {
	dir     string
	scanner scanner.Service
	parser  parser.Service
	current tea.Model
}

// New constructs a startup Model, selecting the initial state from cfg and watcherErr.
func New(cfg config.Config, dir string, watcherErr error, scannerSvc scanner.Service, parserSvc parser.Service) Model {
	var current tea.Model

	switch {
	case watcherErr != nil:
		current = states.NewWatchingWithError(dir, watcherErr, scannerSvc, parserSvc)
	case cfg.ProjectFile != "":
		current = states.NewParsing(cfg.ProjectFile, states.NewWatching(dir, scannerSvc, parserSvc), parserSvc)
	default:
		current = states.NewScanning(dir, scannerSvc, parserSvc)
	}

	return Model{dir: dir, scanner: scannerSvc, parser: parserSvc, current: current}
}

// Init implements tea.Model.
func (m Model) Init() tea.Cmd { return m.current.Init() }

// Update delegates to the current state. msgs.WatcherError is intercepted here
// because the transition is identical for all states and requires no knowledge
// of the current state's internals.
func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	if we, ok := msg.(msgs.WatcherError); ok {
		m.current = states.NewWatchingWithError(m.dir, we.Err, m.scanner, m.parser)

		return m, nil
	}

	next, cmd := m.current.Update(msg)
	m.current = next

	return m, cmd
}

// View implements tea.Model.
func (m Model) View() tea.View { return m.current.View() }
