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
func New(th *theme.Theme, def domain.ServiceDef, project string, w, h, logBufferCap int) Model {
	s := logs.New(def.Name)

	return Model{
		def:             def,
		inspector:       inspector.New(th, def, w, h),
		logPane:         logpane.New(w, h, logBufferCap, s.Lines()),
		streamer:        s,
		streamerStarted: false,
		theme:           th,
		project:         project,
	}
}

// ServiceName returns the service name.
func (m Model) ServiceName() string {
	return m.def.Name
}

// Init batches the init cmds of all children.
func (m Model) Init() tea.Cmd {
	return tea.Batch(m.logPane.Init(), m.inspector.Init())
}

// Update routes messages to children.
func (m Model) Update(msg tea.Msg) (Model, tea.Cmd) {
	var cmds []tea.Cmd

	switch msg.(type) {
	case msgs.DaemonConnected:
		if !m.streamerStarted {
			m.streamerStarted = true
			containerName := logs.ContainerName(m.project, m.def.Name, m.def.ContainerName)
			m.streamer.Start(context.Background(), containerName)
			cmds = append(cmds, m.streamer.Next())
		}

	case msgs.LogLinesAvailable, msgs.LogStreamError, msgs.LogStreamContainerNotFound:
		cmds = append(cmds, m.streamer.Next())
	}

	var cmd tea.Cmd

	m.inspector, cmd = m.inspector.Update(msg)
	if cmd != nil {
		cmds = append(cmds, cmd)
	}

	var logCmd tea.Cmd

	m.logPane, logCmd = m.logPane.Update(msg)
	if logCmd != nil {
		cmds = append(cmds, logCmd)
	}

	return m, tea.Batch(cmds...)
}

// View returns the rendered content for this host's position in the compositor.
func (m Model) View() tea.View {
	inspView := m.inspector.View().Content
	logView := m.logPane.View().Content

	if logView == "" {
		return tea.NewView(inspView)
	}

	return tea.NewView(inspView + "\n" + logView)
}
