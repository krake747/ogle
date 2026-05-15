package servicelist2

import (
	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"

	"github.com/ma-tf/ogle/internal/domain"
	"github.com/ma-tf/ogle/internal/msgs"
	"github.com/ma-tf/ogle/internal/ui/theme"
)

// serviceItem is a per-service list item that renders a coloured state icon,
// service name, and optional action label.
type serviceItem struct {
	def         domain.ServiceDef
	runtime     *domain.ServiceRuntimeData
	inFlight    bool
	actionLabel string
	actionError string
	theme       *theme.Theme
}

func newServiceItem(def domain.ServiceDef, th *theme.Theme) serviceItem {
	return serviceItem{
		def:         def,
		theme:       th,
		runtime:     nil,
		inFlight:    false,
		actionLabel: "",
		actionError: "",
	}
}

func (m serviceItem) Init() tea.Cmd {
	return nil
}

func (m serviceItem) Update(msg tea.Msg) (serviceItem, tea.Cmd) {
	if sp, ok := msg.(msgs.ServicesPolled); ok && sp.Err == nil && m.def.Name != "" {
		m.runtime = sp.Runtimes[m.def.Name]
	}

	return m, nil
}

func (m serviceItem) View() string {
	icon := "●"
	colour := m.theme.StateMuted

	switch {
	case m.inFlight:
		icon = "◌"
		colour = m.theme.StateTransient
	case m.runtime == nil:

	default:
		switch m.runtime.State {
		case domain.ServiceStateRunning:
			icon = "●"
			colour = m.theme.StateRunning
		case domain.ServiceStateExited, domain.ServiceStateDead:
			icon = "●"
			colour = m.theme.StateExited
		case domain.ServiceStateNotCreated:
			icon = "○"
		case domain.ServiceStatePaused:
			icon = "●"
			colour = m.theme.StatePaused
		case domain.ServiceStateRestarting:
			icon = "●"
			colour = m.theme.StateTransient
		case domain.ServiceStateUnknown:
			icon = "●"
		}
	}

	rendered := lipgloss.NewStyle().Foreground(colour).Render(icon) + " " + m.def.Name

	if m.inFlight && m.actionLabel != "" {
		rendered += "  " + m.actionLabel
	}

	if !m.inFlight && m.actionError != "" {
		rendered += "  " + lipgloss.NewStyle().Foreground(m.theme.ActionError).Render(m.actionError)
	}

	return rendered
}

func (m serviceItem) Title() string {
	return m.View()
}

func (m serviceItem) Description() string {
	if m.runtime == nil {
		return "—"
	}

	return string(m.runtime.State)
}

func (m serviceItem) FilterValue() string {
	return m.def.Name
}
