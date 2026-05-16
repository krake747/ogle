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
	tileHeight = 0
)

// Model manages a set of per-service hosts and the state polling lifecycle.
type Model struct {
	hosts         []servicehost.Model
	theme         *theme.Theme
	pollerStarted bool
	selectedName  string
}

// New constructs a Model with one host per project service.
func New(project *domain.Project, th *theme.Theme, w, h int) Model {
	hosts := make([]servicehost.Model, len(project.Services))
	for i, svc := range project.Services {
		hosts[i] = servicehost.New(th, svc, project.Name, w, h)
	}

	return Model{
		hosts:         hosts,
		theme:         th,
		pollerStarted: false,
		selectedName:  "",
	}
}

// Init starts all hosts' flush ticks.
func (m Model) Init() tea.Cmd {
	cmds := make([]tea.Cmd, len(m.hosts))
	for i := range m.hosts {
		cmds[i] = m.hosts[i].Init()
	}

	return tea.Batch(cmds...)
}

// Update handles poll lifecycle messages and forwards everything else to hosts.
func (m Model) Update(msg tea.Msg) (Model, tea.Cmd) {
	cmds := make([]tea.Cmd, 0, len(m.hosts)+1)

	switch msg := msg.(type) {
	case msgs.ServiceSelected:
		m.selectedName = msg.ServiceName

	case msgs.DaemonConnected:
		if !m.pollerStarted {
			m.pollerStarted = true
			cmds = append(cmds, m.pollStateCmd())
		}

	case msgs.StatePollTick:
		cmds = append(cmds, m.pollStateCmd())
	}

	for i := range m.hosts {
		var cmd tea.Cmd

		m.hosts[i], cmd = m.hosts[i].Update(msg)
		cmds = append(cmds, cmd)
	}

	return m, tea.Batch(cmds...)
}

// View renders all hosts as compositor layers with the selected host at top.
func (m Model) View() string {
	if len(m.hosts) == 0 {
		return ""
	}

	topIdx := -1

	for i, h := range m.hosts {
		if h.ServiceName() == m.selectedName {
			topIdx = i

			break
		}
	}

	lyrs := make([]*lipgloss.Layer, 0, len(m.hosts))
	y := 0

	if topIdx >= 0 {
		lyrs = append(lyrs, lipgloss.NewLayer(m.hosts[topIdx].View()).X(0).Y(0).Z(1))
		y += tileHeight
	}

	for i, h := range m.hosts {
		if i == topIdx {
			continue
		}

		lyrs = append(lyrs, lipgloss.NewLayer(h.View()).X(0).Y(y).Z(0))
		y += tileHeight
	}

	return lipgloss.NewCompositor(lyrs...).Render()
}

func (m Model) pollStateCmd() tea.Cmd {
	return tea.Tick(time.Second, func(_ time.Time) tea.Msg {
		return msgs.StatePollTick{}
	})
}
