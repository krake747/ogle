package app

import (
	"context"

	tea "charm.land/bubbletea/v2"
	"github.com/ma-tf/ogle/config"
)

type screen int

const (
	// screenStartup is the initial state: the startup flow manages file
	// discovery and hands off to screenDashboard once a project is loaded.
	screenStartup screen = iota

	// screenDashboard is the main monitoring view shown after a project is
	// successfully loaded.
	// screenDashboard
)

// model is the root Bubble Tea model. It owns the active screen and delegates
// Init/Update/View to the appropriate child model.
type model struct {
	cfg    config.Config
	screen screen
	// startupModel ui/flows/startup.Model — wired in when that package exists.
	// dashboardModel ui/views/dashboard.Model — wired in when that package exists.
}

func newModel(cfg config.Config) model {
	return model{
		cfg:    cfg,
		screen: screenStartup,
	}
}

func (m model) Init() tea.Cmd {
	return nil
}

func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyPressMsg:
		switch msg.String() {
		case "ctrl+c", "q":
			return m, tea.Quit
		}
	}

	return m, nil
}

func (m model) View() tea.View {
	// Placeholder rendered until ui/flows/startup and ui/views/dashboard are
	// wired into this model.
	return tea.NewView("ogle\n\nPress q to quit.\n")
}

// Setup creates and runs a new Bubble Tea program. ctx is passed to the
// program via tea.WithContext so that external cancellation (e.g. a signal)
// propagates cleanly into the TUI runtime.
func Setup(ctx context.Context, cfg config.Config) *tea.Program {
	return tea.NewProgram(newModel(cfg), tea.WithContext(ctx))
}
