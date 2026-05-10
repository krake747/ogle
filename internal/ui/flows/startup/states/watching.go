package states

import (
	tea "charm.land/bubbletea/v2"

	"github.com/ma-tf/ogle/internal/msgs"
	"github.com/ma-tf/ogle/internal/services/parser"
	"github.com/ma-tf/ogle/internal/services/scanner"
	"github.com/ma-tf/ogle/internal/ui/views/watching"
)

// Watching is the state rendered when no valid compose files are present. The
// watching view is active and the user waits for a file to appear.
type Watching struct {
	model   watching.Model
	handler fileHandler
}

// NewWatching constructs a Watching state for the given directory.
func NewWatching(
	dir string,
	sc scanner.Scanner,
	p parser.Parser,
) tea.Model {
	return Watching{
		model:   watching.New(dir),
		handler: fileHandler{dir: dir, scanner: sc, parser: p},
	}
}

// NewWatchingWithError constructs a Watching state with an error displayed.
func NewWatchingWithError(
	dir string,
	err error,
	sc scanner.Scanner,
	p parser.Parser,
) tea.Model {
	return Watching{
		model:   watching.New(dir).SetError(err),
		handler: fileHandler{dir: dir, scanner: sc, parser: p},
	}
}

// withNotice returns a copy of w with a notice set on the underlying view.
func (w Watching) withNotice(notice string) Watching {
	return Watching{model: w.model.SetNotice(notice), handler: w.handler}
}

// withParsing returns a copy of w with the parsing indicator set.
func (w Watching) withParsing(v bool) Watching {
	return Watching{model: w.model.SetParsing(v), handler: w.handler}
}

// Init returns nil — the watcher subscription is managed by the dashboard
// orchestrator, not the startup flow.
func (w Watching) Init() tea.Cmd { return nil }

// Update implements tea.Model.
func (w Watching) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	if fac, ok := msg.(msgs.FileAvailabilityChanged); ok {
		valid := validateFiles(fac.Files, w.handler.parser)

		return w.handler.handle(valid, w)
	}

	updated, cmd := w.model.Update(msg)

	return Watching{model: updated, handler: w.handler}, cmd
}

// View implements tea.Model.
func (w Watching) View() tea.View { return tea.NewView(w.model.View()) }
