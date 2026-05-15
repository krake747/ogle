// Package servicehost wraps a per-service inspector and log stream into a
// single compositor-hostable unit.
package servicehost

import (
	tea "charm.land/bubbletea/v2"

	"github.com/ma-tf/ogle/internal/domain"
	"github.com/ma-tf/ogle/internal/msgs"
	"github.com/ma-tf/ogle/internal/ui/components/inspector2"
	"github.com/ma-tf/ogle/internal/ui/components/logpane"
	"github.com/ma-tf/ogle/internal/ui/theme"
)

// Model wraps a per-service inspector and log stream.
type Model struct {
	def       domain.ServiceDef
	inspector inspector2.Model
	logPane   *logpane.LogPane
	theme     *theme.Theme
}

// New constructs a host for the given service. logPane may be nil; when set,
// the host wires log output into the inspector display.
func New(th *theme.Theme, def domain.ServiceDef, w, h int, logPane *logpane.LogPane) Model {
	return Model{
		def:       def,
		inspector: inspector2.New(th, def, w, h),
		logPane:   logPane,
		theme:     th,
	}
}

// ServiceName returns the service name.
func (m Model) ServiceName() string { return m.def.Name }

// Init satisfies tea.Model.
func (m Model) Init() tea.Cmd { return nil }

// Update routes messages to the correct child.
func (m Model) Update(msg tea.Msg) (Model, tea.Cmd) {
	if m.logPane != nil {
		if _, ok := msg.(msgs.LogLine); ok {
			m.logPane, _ = m.logPane.Update(msg)
		}

		if _, ok := msg.(msgs.LogStreamError); ok {
			m.logPane, _ = m.logPane.Update(msg)
		}

		if _, ok := msg.(msgs.LogStreamContainerNotFound); ok {
			m.logPane, _ = m.logPane.Update(msg)
		}
	}

	var cmd tea.Cmd

	m.inspector, cmd = m.inspector.Update(msg)

	return m, cmd
}

// View returns the rendered content for this host's position in the compositor.
func (m Model) View() string {
	return m.inspector.View()
}

// Close stops the log stream if one is active.
func (m Model) Close() {
	if m.logPane != nil {
		m.logPane.Close()
	}
}
