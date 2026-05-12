package states

import (
	"errors"
	"path/filepath"

	tea "charm.land/bubbletea/v2"

	"github.com/ma-tf/ogle/internal/services/parser"
	"github.com/ma-tf/ogle/internal/services/scanner"
	"github.com/ma-tf/ogle/internal/ui/theme"
	"github.com/ma-tf/ogle/internal/ui/views/fileselect"
	"github.com/ma-tf/ogle/internal/ui/views/watching"
)

// fileHandler is the single source of truth for startup state transitions.
// It encodes the 0/1/2+ dispatch: how many valid compose files are present
// determines which state the startup flow enters next.
type fileHandler struct {
	dir     string
	scanner scanner.Scanner
	parser  parser.Parser
	theme   *theme.Theme
	width   int
	height  int
}

// handle dispatches on the count of valid files and returns the next state.
func (fh fileHandler) handle(valid []string, current tea.Model) (tea.Model, tea.Cmd) {
	switch len(valid) {
	case 0:
		return fh.newWatching(), nil
	case 1:
		parse := ParseCmd(valid[0], fh.parser)

		return Parsing{
			path:    valid[0],
			parse:   parse,
			display: fh.visibleState(current),
		}, parse
	default:
		return Selecting{model: fileselect.New(valid, fh.theme, fh.width, fh.height), handler: fh}, nil
	}
}

// newWatching constructs a Watching state. If a file exists on disk but
// cannot be parsed, a notice is set on the watching view.
func (fh fileHandler) newWatching() tea.Model {
	m := watching.New(fh.dir, fh.width, fh.height)
	for _, name := range fh.scanner.KnownFilenames() {
		path := filepath.Join(fh.dir, name)
		if err := fh.parser.Validate(path); err != nil {
			if !errors.Is(err, parser.ErrReadComposeFile) {
				m = m.SetNotice(name + " exists but could not be parsed")

				break
			}
		}
	}

	return Watching{model: m, handler: fh}
}

// visibleState returns the state to use as Parsing.display. Unwraps a nested
// Parsing to its display; passes Watching and Selecting through unchanged;
// falls back to a blank Watching for Scanning (sub-millisecond, blank screen).
func (fh fileHandler) visibleState(current tea.Model) tea.Model {
	switch c := current.(type) {
	case Parsing:
		return c.display
	case Watching, Selecting:
		return current
	default:
		return Watching{model: watching.New(fh.dir, fh.width, fh.height), handler: fh}
	}
}
