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
	"github.com/ma-tf/ogle/internal/ui/theme"
)

// Model is the startup flow orchestrator.
type Model struct {
	dir     string
	width   int
	height  int
	scanner scanner.Scanner
	parser  parser.Parser
	theme   *theme.Theme
	current tea.Model
}

// New constructs a startup Model, selecting the initial state from cfg and watcherErr.
func New(
	cfg config.Config,
	dir string,
	watcherErr error,
	sc scanner.Scanner,
	p parser.Parser,
	th *theme.Theme,
	width,
	height int,
) Model {
	var current tea.Model

	switch {
	case watcherErr != nil:
		current = states.NewWatchingWithError(dir, watcherErr, sc, p, th, width, height)
	case cfg.ProjectFile != "":
		current = states.NewParsing(cfg.ProjectFile, states.NewWatching(dir, sc, p, th, width, height), p)
	default:
		current = states.NewScanning(dir, sc, p, th, width, height)
	}

	return Model{
		dir:     dir,
		width:   width,
		height:  height,
		scanner: sc,
		parser:  p,
		theme:   th,
		current: current,
	}
}

// Init implements tea.Model.
func (m Model) Init() tea.Cmd { return m.current.Init() }

// Update delegates to the current state. msgs.WatcherError is intercepted here
// because the transition is identical for all states and requires no knowledge
// of the current state's internals.
func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	if sz, ok := msg.(tea.WindowSizeMsg); ok {
		m.width = sz.Width
		m.height = sz.Height
	}

	if we, ok := msg.(msgs.WatcherError); ok {
		m.current = states.NewWatchingWithError(
			m.dir,
			we.Err,
			m.scanner,
			m.parser,
			m.theme,
			m.width,
			m.height,
		)

		return m, nil
	}

	next, cmd := m.current.Update(msg)
	m.current = next

	return m, cmd
}

// View implements tea.Model.
func (m Model) View() tea.View { return m.current.View() }
