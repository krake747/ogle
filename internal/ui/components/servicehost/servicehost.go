package servicehost

import (
	"context"

	tea "charm.land/bubbletea/v2"

	"github.com/ma-tf/ogle/internal/domain"
	"github.com/ma-tf/ogle/internal/msgs"
	"github.com/ma-tf/ogle/internal/services/docker/logs"
	"github.com/ma-tf/ogle/internal/ui/components/inspector"
	"github.com/ma-tf/ogle/internal/ui/components/logpane"
	"github.com/ma-tf/ogle/internal/ui/theme"
)

// Model wraps a per-service inspector and log pane into a single
// compositor-hostable unit.
type Model struct {
	def             domain.ServiceDef
	inspector       inspector.Model
	logPane         logpane.Model
	streamer        logs.Streamer
	streamerStarted bool
	theme           *theme.Theme
	project         string
}

// New constructs a host for the given service.
func New(th *theme.Theme, def domain.ServiceDef, project string, w, h int) Model {
	return Model{
		def:             def,
		inspector:       inspector.New(th, def, w, h),
		logPane:         logpane.New(w, h),
		streamer:        logs.New(def.Name),
		streamerStarted: false,
		theme:           th,
		project:         project,
	}
}

// ServiceName returns the service name.
func (m Model) ServiceName() string {
	return m.def.Name
}

// Init satisfies tea.Model.
func (m Model) Init() tea.Cmd {
	return nil
}

// Update routes messages to children.
func (m Model) Update(msg tea.Msg) (Model, tea.Cmd) {
	var cmds []tea.Cmd

	switch msg := msg.(type) {
	case msgs.DaemonConnected:
		if !m.streamerStarted {
			m.streamerStarted = true
			containerName := logs.ContainerName(m.project, m.def.Name, m.def.ContainerName)
			m.streamer.Start(context.Background(), containerName)
			cmds = append(cmds, m.streamer.Next())
		}

	case msgs.LogLine:
		cmds = append(cmds, m.streamer.Next())
		if msg.ServiceName == m.def.Name {
			var logCmd tea.Cmd

			m.logPane, logCmd = m.logPane.Update(msg)
			if logCmd != nil {
				cmds = append(cmds, logCmd)
			}
		}

	case msgs.LogStreamError, msgs.LogStreamContainerNotFound:
		cmds = append(cmds, m.streamer.Next())
	}

	var cmd tea.Cmd

	m.inspector, cmd = m.inspector.Update(msg)
	if cmd != nil {
		cmds = append(cmds, cmd)
	}

	return m, tea.Batch(cmds...)
}

// View returns the rendered content for this host's position in the compositor.
func (m Model) View() string {
	inspView := m.inspector.View()
	logView := m.logPane.View()

	if logView == "" {
		return inspView
	}

	return inspView + "\n" + logView
}
