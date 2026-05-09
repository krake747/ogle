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
	HandleFiles func([]string, tea.Model) (tea.Model, tea.Cmd)
}

// Init returns nil — the watcher subscription is managed by the dashboard
// orchestrator, not the startup flow.
func (w Watching) Init() tea.Cmd {
	return nil
}

// Update implements tea.Model.
func (w Watching) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	if fac, ok := msg.(msgs.FileAvailabilityChanged); ok {
		valid := validateFiles(fac.Files)

		return w.HandleFiles(valid, w)
	}

	updated, cmd := w.Model.Update(msg)

	return Watching{Model: updated, HandleFiles: w.HandleFiles}, cmd
}

// View implements tea.Model.
func (w Watching) View() tea.View { return tea.NewView(w.Model.View()) }
