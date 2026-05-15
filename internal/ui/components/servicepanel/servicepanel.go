// Package servicepanel manages a set of per-service hosts and their polling
// lifecycle. It renders all hosts as compositor layers stacked vertically.
package servicepanel

import (
	"time"

	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"

	"github.com/ma-tf/ogle/internal/domain"
	"github.com/ma-tf/ogle/internal/msgs"
	"github.com/ma-tf/ogle/internal/ui/components/servicehost"
	"github.com/ma-tf/ogle/internal/ui/theme"
)

const (
	tileHeight = 8
)

// Model manages a set of per-service hosts and the state polling lifecycle.
type Model struct {
	hosts         []servicehost.Model
	theme         *theme.Theme
	pollerStarted bool
}

// New constructs a Model with one host per project service.
func New(project *domain.Project, th *theme.Theme, w, h int) Model {
	hosts := make([]servicehost.Model, len(project.Services))
	for i, svc := range project.Services {
		hosts[i] = servicehost.New(th, svc, w, h, nil)
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

// View renders all hosts as compositor layers stacked vertically.
func (m Model) View() string {
	if len(m.hosts) == 0 {
		return ""
	}

	lyrs := make([]*lipgloss.Layer, len(m.hosts))

	for i, h := range m.hosts {
		lyrs[i] = lipgloss.NewLayer(h.View()).X(0).Y(i * tileHeight).Z(0)
	}

	return lipgloss.NewCompositor(lyrs...).Render()
}

func (m Model) pollStateCmd() tea.Cmd {
	return tea.Tick(time.Second, func(_ time.Time) tea.Msg {
		return msgs.StatePollTick{}
	})
}
