package servicehost

import (
	tea "charm.land/bubbletea/v2"

	"github.com/ma-tf/ogle/internal/domain"
	"github.com/ma-tf/ogle/internal/ui/components/inspector2"
	"github.com/ma-tf/ogle/internal/ui/components/logpane2"
	"github.com/ma-tf/ogle/internal/ui/theme"
)

// Model wraps a per-service inspector and log pane into a single
// compositor-hostable unit.
type Model struct {
	def       domain.ServiceDef
	inspector inspector2.Model
	logPane   logpane2.Model
	theme     *theme.Theme
}

// New constructs a host for the given service.
func New(th *theme.Theme, def domain.ServiceDef, w, h int) Model {
	return Model{
		def:       def,
		inspector: inspector2.New(th, def, w, h),
		logPane:   logpane2.New(),
		theme:     th,
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
	var cmd tea.Cmd

	m.logPane, _ = m.logPane.Update(msg)
	m.inspector, cmd = m.inspector.Update(msg)

	return m, cmd
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
