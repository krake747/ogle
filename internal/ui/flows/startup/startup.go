// Package startup implements the startup flow: the orchestration state machine
// that runs from app launch until a compose project is successfully loaded.
package startup

import (
	"errors"
	"path/filepath"

	tea "charm.land/bubbletea/v2"

	"github.com/ma-tf/ogle/config"
	"github.com/ma-tf/ogle/internal/compose"
	"github.com/ma-tf/ogle/internal/msgs"
	"github.com/ma-tf/ogle/internal/ui/flows/startup/states"
	"github.com/ma-tf/ogle/internal/ui/views/fileselect"
	"github.com/ma-tf/ogle/internal/ui/views/watching"
)

// Model is the startup flow orchestrator.
type Model struct {
	dir     string
	current tea.Model
}

// New constructs a startup Model, selecting the initial state from cfg and watcherErr.
func New(cfg config.Config, dir string, watcherErr error) Model {
	hf := makeHandleFiles(dir)

	var current tea.Model

	switch {
	case watcherErr != nil:
		current = states.Watching{
			Model:       watching.New(dir).SetError(watcherErr),
			HandleFiles: hf,
		}
	case cfg.ProjectFile != "":
		cw := states.Watching{Model: watching.New(dir), HandleFiles: hf}
		parse := states.ParseCmd(cfg.ProjectFile)
		current = states.Parsing{Path: cfg.ProjectFile, Parse: parse, Display: cw}
	default:
		current = states.Scanning{Scan: states.ScanCmd(dir), HandleFiles: hf}
	}

	return Model{dir: dir, current: current}
}

// Init implements tea.Model.
func (m Model) Init() tea.Cmd { return m.current.Init() }

// Update delegates to the current state. msgs.WatcherError is intercepted here
// because the transition is identical for all states and requires rebuilding
// the handleFiles closure with the same dir.
func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	if we, ok := msg.(msgs.WatcherError); ok {
		hf := makeHandleFiles(m.dir)
		m.current = states.Watching{
			Model:       watching.New(m.dir).SetError(we.Err),
			HandleFiles: hf,
		}

		return m, nil
	}

	next, cmd := m.current.Update(msg)
	m.current = next

	return m, cmd
}

// View implements tea.Model.
func (m Model) View() tea.View { return m.current.View() }

// makeHandleFiles produces the central 0/1/2+ dispatch function for a given
// directory. It is a pure function of dir: calling it twice with the same dir
// produces behaviourally identical closures.
//
// The var-hf pattern allows newWatching and the returned closure to reference
// each other safely; both inner functions are only ever called after hf is
// assigned.
func makeHandleFiles(dir string) func([]string, tea.Model) (tea.Model, tea.Cmd) {
	var hf func([]string, tea.Model) (tea.Model, tea.Cmd)

	newWatching := func() tea.Model {
		m := watching.New(dir)

		for _, name := range compose.KnownFilenames() {
			path := filepath.Join(dir, name)
			if err := compose.Validate(path); err != nil {
				if !errors.Is(err, compose.ErrReadComposeFile) {
					m = m.SetNotice(name + " exists but could not be parsed")

					break
				}
			}
		}

		return states.Watching{Model: m, HandleFiles: hf}
	}

	// visibleState returns the state to store as Parsing.Display, unwrapping
	// nested Parsing and falling back to a cold Watching for Scanning.
	visibleState := func(current tea.Model) tea.Model {
		switch c := current.(type) {
		case states.Parsing:
			return c.Display
		case states.Watching, states.Selecting:
			return current
		default:
			return states.Watching{Model: watching.New(dir), HandleFiles: hf}
		}
	}

	hf = func(valid []string, current tea.Model) (tea.Model, tea.Cmd) {
		switch len(valid) {
		case 0:
			return newWatching(), nil
		case 1:
			parse := states.ParseCmd(valid[0])

			return states.Parsing{
				Path:    valid[0],
				Parse:   parse,
				Display: visibleState(current),
			}, parse
		default:
			return states.Selecting{
				Model:       fileselect.New(valid),
				HandleFiles: hf,
			}, nil
		}
	}

	return hf
}
