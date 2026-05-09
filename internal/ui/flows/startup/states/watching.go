package states

import (
	tea "charm.land/bubbletea/v2"

	"github.com/ma-tf/ogle/internal/msgs"
	"github.com/ma-tf/ogle/internal/ui/views/watching"
)

// Watching is the state rendered when no valid compose files are present. The
// watching view is active and the user waits for a file to appear.
type Watching struct {
	Model       watching.Model
	HandleFiles func([]string, State) (State, tea.Cmd)
}

// Init has no startup command; the watcher subscription is managed by the
// dashboard orchestrator, not the startup flow.
func (w Watching) Init() tea.Cmd {
	return nil
}

// Update reacts to file availability changes and forwards all other messages
// to the watching sub-model.
func (w Watching) Update(msg tea.Msg) (State, tea.Cmd) {
	if fac, ok := msg.(msgs.FileAvailabilityChanged); ok {
		valid := validateFiles(fac.Files)

		return w.HandleFiles(valid, w)
	}

	updated, cmd := w.Model.Update(msg)

	return Watching{Model: updated, HandleFiles: w.HandleFiles}, cmd
}

// View renders the watching screen.
func (w Watching) View() string {
	return w.Model.View()
}
