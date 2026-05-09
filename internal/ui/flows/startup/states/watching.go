package states

import (
	tea "charm.land/bubbletea/v2"

	"github.com/ma-tf/ogle/internal/msgs"
	"github.com/ma-tf/ogle/internal/ui/views/watching"
)

// Watching is the state rendered when no valid compose files are present. The
// watching view is active and the user waits for a file to appear.
type Watching struct {
	model   watching.Model
	handler fileHandler
}

// NewWatching constructs a Watching state for the given directory.
func NewWatching(dir string) tea.Model {
	return Watching{model: watching.New(dir), handler: fileHandler{dir: dir}}
}

// NewWatchingWithError constructs a Watching state with an error displayed.
func NewWatchingWithError(dir string, err error) tea.Model {
	return Watching{
		model:   watching.New(dir).SetError(err),
		handler: fileHandler{dir: dir},
	}
}

// withNotice returns a copy of w with a notice set on the underlying view.
func (w Watching) withNotice(notice string) Watching {
	return Watching{model: w.model.SetNotice(notice), handler: w.handler}
}

// Init returns nil — the watcher subscription is managed by the dashboard
// orchestrator, not the startup flow.
func (w Watching) Init() tea.Cmd { return nil }

// Update implements tea.Model.
func (w Watching) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	if fac, ok := msg.(msgs.FileAvailabilityChanged); ok {
		valid := validateFiles(fac.Files)
		return w.handler.handle(valid, w)
	}

	updated, cmd := w.model.Update(msg)

	return Watching{model: updated, handler: w.handler}, cmd
}

// View implements tea.Model.
func (w Watching) View() tea.View { return tea.NewView(w.model.View()) }
