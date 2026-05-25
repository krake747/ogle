package servicehost

import (
	"context"
	"time"

	tea "charm.land/bubbletea/v2"

	"github.com/ma-tf/ogle/internal/domain"
	"github.com/ma-tf/ogle/internal/msgs"
	"github.com/ma-tf/ogle/internal/services/docker/logs"
	"github.com/ma-tf/ogle/internal/ui/components/logpane"
	"github.com/ma-tf/ogle/internal/ui/theme"
)

// logStreamRetryDelay is the delay before retrying the log stream after an
// error (container not found or read error).
const logStreamRetryDelay = 2 * time.Second

// Model wraps a per-service log pane and streamer into a compositor-hostable unit.
type Model struct {
	def             domain.ServiceDef
	logPane         logpane.Model
	streamer        logs.Streamer
	streamerStarted bool
	theme           *theme.Theme
	project         string
	selected        bool
}

// New constructs a host for the given service.
func New(
	th *theme.Theme,
	def domain.ServiceDef,
	project string,
	w, h, logBufferCap int,
	streamer logs.Streamer,
) Model {
	return Model{
		def:             def,
		logPane:         logpane.New(th, w, h, logBufferCap, streamer.Lines()),
		streamer:        streamer,
		streamerStarted: false,
		theme:           th,
		project:         project,
		selected:        false,
	}
}

// Init batches the init cmds of all children.
func (m Model) Init() tea.Cmd {
	return m.logPane.Init()
}

// Update routes messages to children.
func (m Model) Update(msg tea.Msg) (Model, tea.Cmd) {
	var cmds []tea.Cmd

	switch msg := msg.(type) {
	case msgs.ServiceSelected:
		m.selected = (msg.ServiceName == m.def.Name)

		return m, nil

	case tea.KeyPressMsg, tea.MouseWheelMsg:
		if !m.selected {
			return m, nil
		}

	case msgs.DaemonConnected:
		if !m.streamerStarted {
			m.streamerStarted = true
			containerName := logs.ContainerName(m.project, m.def.Name, m.def.ContainerName)
			m.streamer.Start(context.Background(), containerName)
			cmds = append(cmds, m.streamer.Next())
		}

	case msgs.LogLinesAvailable:
		cmds = append(cmds, m.streamer.Next())

	case msgs.LogStreamError, msgs.LogStreamContainerNotFound:
		m.streamer.Close()
		m.streamerStarted = false

		cmds = append(cmds, tea.Tick(logStreamRetryDelay, func(_ time.Time) tea.Msg {
			return msgs.LogStreamRetryTick{}
		}))

	case msgs.LogStreamRetryTick:
		if !m.streamerStarted {
			m.streamerStarted = true
			containerName := logs.ContainerName(m.project, m.def.Name, m.def.ContainerName)
			m.streamer.Start(context.Background(), containerName)
			cmds = append(cmds, m.streamer.Next())
		}

	case theme.Changed:
		m.theme = msg.Theme
	}

	var logCmd tea.Cmd

	m.logPane, logCmd = m.logPane.Update(msg)
	if logCmd != nil {
		cmds = append(cmds, logCmd)
	}

	return m, tea.Batch(cmds...)
}

// View returns the log pane for the selected service, or an empty view.
func (m Model) View() tea.View {
	if !m.selected {
		return tea.NewView("")
	}

	return m.logPane.View()
}
