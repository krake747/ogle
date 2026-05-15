// Package servicepanel manages a set of per-service inspector hosts and their
// polling lifecycle. It renders all hosts as a vertical stack in the right-pane
// content area.
package servicepanel

import (
	"time"

	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"

	"github.com/ma-tf/ogle/internal/domain"
	"github.com/ma-tf/ogle/internal/msgs"
	"github.com/ma-tf/ogle/internal/ui/components/inspector2"
	"github.com/ma-tf/ogle/internal/ui/theme"
)

// Model manages a set of per-service hosts and the state polling lifecycle.
type Model struct {
	hosts         []inspector2.Model
	theme         *theme.Theme
	pollerStarted bool
}

// New constructs a Model with one host per project service.
func New(project *domain.Project, th *theme.Theme, w, h int) Model {
	hosts := make([]inspector2.Model, len(project.Services))
	for i, svc := range project.Services {
		hosts[i] = inspector2.New(th, svc, w, h)
	}

	return Model{
		hosts:         hosts,
		theme:         th,
		pollerStarted: false,
	}
}

// Init satisfies tea.Model.
func (m Model) Init() tea.Cmd { return nil }

// Update handles poll lifecycle messages and forwards everything else to hosts.
func (m Model) Update(msg tea.Msg) (Model, tea.Cmd) {
	switch msg.(type) {
	case msgs.DaemonConnected:
		if !m.pollerStarted {
			m.pollerStarted = true

			return m, m.pollStateCmd()
		}

	case msgs.StatePollTick:
		return m, m.pollStateCmd()
	}

	for i := range m.hosts {
		m.hosts[i], _ = m.hosts[i].Update(msg)
	}

	return m, nil
}

// View renders all hosts as a vertical stack.
func (m Model) View() string {
	if len(m.hosts) == 0 {
		return ""
	}

	var rows []string

	for _, h := range m.hosts {
		rows = append(rows, h.View())
	}

	return lipgloss.JoinVertical(lipgloss.Top, rows...)
}

func (m Model) pollStateCmd() tea.Cmd {
	return tea.Tick(time.Second, func(_ time.Time) tea.Msg {
		return msgs.StatePollTick{}
	})
}
